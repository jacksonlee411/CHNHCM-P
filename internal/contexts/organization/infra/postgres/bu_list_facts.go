package postgres

import (
	"context"

	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
)

// BU list/selection and Department labels share only immutable BU members.
// Every request pins its own exact BU heads and applies current grants later.
func (u *unit) buListFacts(ctx context.Context, wanted map[string]string, day string) (map[string]d.BusinessUnit, error) {
	ids := []string{}
	for id := range wanted {
		ids = append(ids, id)
	}
	query := `SELECT bu_id::text,head_source_id::text FROM organization.business_unit WHERE instance_id=$1`
	args := []any{u.instance}
	if wanted != nil {
		query += ` AND bu_id=ANY($2::text[]::uuid[])`
		args = append(args, ids)
	}
	rows, err := u.tx.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	heads := map[string]string{}
	for rows.Next() {
		var id, source string
		if err = rows.Scan(&id, &source); err != nil {
			rows.Close()
			return nil, err
		}
		heads[id] = source
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	c := u.listCache
	owns := false
	defer func() {
		if owns {
			c.mu.Lock()
			close(c.loading)
			c.loading = nil
			c.mu.Unlock()
		}
	}()
	for {
		out := make(map[string]d.BusinessUnit, len(heads))
		missing := []map[string]any{}
		c.mu.RLock()
		generation := c.generation
		for id, source := range heads {
			v := c.bus[id].value
			if v.Source == source && buCovers(v, day) {
				out[id] = v
			} else {
				missing = append(missing, map[string]any{"bu_id": id, "source_id": source, "dates": []string{day}})
			}
		}
		wait := c.loading
		c.mu.RUnlock()
		if len(missing) == 0 {
			return out, nil
		}
		if wait != nil && !owns {
			select {
			case <-wait:
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		c.mu.Lock()
		if !owns && (c.loading != nil || c.generation != generation) {
			c.mu.Unlock()
			continue
		}
		if !owns {
			c.loading = make(chan struct{})
			owns = true
		}
		c.mu.Unlock()
		loaded, err := (&departmentUnit{unit: u}).loadBUListFacts(ctx, missing)
		if err != nil {
			return nil, err
		}
		prepared := make(map[string]cachedBU, len(loaded.BusinessUnits))
		for _, v := range loaded.BusinessUnits {
			old := c.bus[v.ID]
			merged := mergeBU(old.value, v)
			prepared[v.ID] = cachedBU{merged, buFactBytes(merged)}
		}
		c.mu.Lock()
		if c.bus == nil {
			c.bus = map[string]cachedBU{}
		}
		for id, v := range prepared {
			c.bytes += v.bytes - c.bus[id].bytes
			c.bus[id] = v
		}
		c.generation++
		if c.bytes > listCacheBytes {
			c.bus = nil
			c.bytes = 0
		}
		c.mu.Unlock()
		// Current supported shapes fit the shared bound. A very large response
		// remains correct even if it cannot be retained by the cache.
		if c.bus == nil {
			for _, v := range loaded.BusinessUnits {
				out[v.ID] = v
			}
			if len(out) != len(heads) {
				return nil, resultUnavailable()
			}
			return out, nil
		}
	}
}

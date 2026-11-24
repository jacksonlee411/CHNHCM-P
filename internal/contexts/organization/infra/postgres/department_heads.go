package postgres

import (
	"context"
	"slices"
)

type departmentHead struct {
	id, source, relation string
	bus, parents         []string
}

// departmentHeads selects the same conservative graph as Discover: descendants
// first, then ancestors (including proposed parents). Adding ancestors must never
// restart reverse traversal into siblings. IDs refer to all dated relationships.
func (u *unit) departmentHeads(ctx context.Context, id string, bus, parents []string, all, reverse, allBUs bool) (map[string]string, map[string]string, map[string]string, error) {
	ds, bs, relations := map[string]string{}, map[string]string{}, map[string]string{}
	heads := map[string]departmentHead{}
	read := func(predicate string, ids []string) ([]string, error) {
		args := []any{u.instance}
		if ids != nil {
			args = append(args, ids)
		}
		// Read each relation by its owner key. A regular join can repeatedly
		// scan the whole relation table for a small frontier at real scale.
		rows, e := u.tx.Query(ctx, `SELECT d.department_id::text,d.head_source_id::text,
 (SELECT h.head_source_id::text FROM organization.department_hierarchy_relation h WHERE h.instance_id=d.instance_id AND h.enterprise_id=d.enterprise_id AND h.department_id=d.department_id),
 d.management_bus::text[],d.parents::text[] FROM organization.department d WHERE d.instance_id=$1 AND `+predicate, args...)
		if e != nil {
			return nil, e
		}
		defer rows.Close()
		added := []string{}
		for rows.Next() {
			var h departmentHead
			if e = rows.Scan(&h.id, &h.source, &h.relation, &h.bus, &h.parents); e != nil {
				return nil, e
			}
			if _, seen := heads[h.id]; !seen {
				heads[h.id] = h
				added = append(added, h.id)
			}
		}
		return added, rows.Err()
	}
	var frontier []string
	var e error
	if all {
		frontier, e = read("true", nil)
	} else if id == "" {
		frontier, e = read("d.management_bus && $2::text[]::uuid[]", bus)
	} else {
		frontier, e = read("d.department_id=ANY($2::text[]::uuid[])", []string{id})
	}
	if e != nil {
		return nil, nil, nil, e
	}
	if reverse && !all {
		for len(frontier) > 0 {
			if len(frontier) > 128 {
				// Many GIN overlap probes repeatedly scan the same index/pages.
				// Read only identity/parent pairs once for a broad frontier, then
				// load heads and complete facts solely for reachable identities.
				ids, err := u.departmentDescendantIDs(ctx, frontier)
				if err != nil {
					return nil, nil, nil, err
				}
				wanted := ids[:0]
				for _, id := range ids {
					if _, seen := heads[id]; !seen {
						wanted = append(wanted, id)
					}
				}
				for start := 0; start < len(wanted); start += 512 {
					if _, err = read("d.department_id=ANY($2::text[]::uuid[])", wanted[start:min(start+512, len(wanted))]); err != nil {
						return nil, nil, nil, err
					}
				}
				break
			}
			next := []string{}
			// Larger overlap arrays can turn a GIN lookup into a repeated full
			// department scan. Keep the traversal itself and all dated edges.
			for start := 0; start < len(frontier); start += 128 {
				end := min(start+128, len(frontier))
				v, err := read("d.parents && $2::text[]::uuid[]", frontier[start:end])
				if err != nil {
					return nil, nil, nil, err
				}
				next = append(next, v...)
			}
			frontier = next
		}
	}
	if !all {
		// Every already loaded head contributes its parents below. Starting the
		// ancestor walk from those heads again needlessly reloads a broad subtree.
		// Keep only missing identities; this still includes former/future parents
		// and proposed parents, and never expands ancestors back into siblings.
		missing := map[string]bool{}
		addParent := func(id string) {
			if _, present := heads[id]; id != "" && !present {
				missing[id] = true
			}
		}
		for _, parent := range parents {
			addParent(parent)
		}
		for _, h := range heads {
			for _, parent := range h.parents {
				addParent(parent)
			}
		}
		frontier = make([]string, 0, len(missing))
		for parent := range missing {
			frontier = append(frontier, parent)
		}
		slices.Sort(frontier)
		// Keep one recursive read for a narrow ancestor frontier: a deep chain
		// must not become one network round trip per level. The broad case below
		// avoids the large-seed recursive plan without penalizing ordinary leaves.
		if len(frontier) > 0 && len(frontier) <= 128 {
			_, err := read(`d.department_id IN (
 WITH RECURSIVE ancestors(id) AS (
  SELECT unnest($2::text[]::uuid[])
  UNION
  SELECT unnest(p.parents) FROM ancestors a JOIN organization.department p
   ON p.instance_id=$1 AND p.department_id=a.id
 ) SELECT id FROM ancestors)`, frontier)
			if err != nil {
				return nil, nil, nil, err
			}
			frontier = nil
		}
		// Load each missing ancestor once by owner key. A recursive join over a
		// broad seed set can repeatedly scan the whole instance; bounded identity
		// reads keep both cold and warm plans local. Only newly read heads expand
		// upwards, so cycles terminate and ancestors never introduce siblings.
		for len(frontier) > 0 {
			added := []string{}
			for start := 0; start < len(frontier); start += 512 {
				v, err := read("d.department_id=ANY($2::text[]::uuid[])", frontier[start:min(start+512, len(frontier))])
				if err != nil {
					return nil, nil, nil, err
				}
				added = append(added, v...)
			}
			clear(missing)
			for _, id := range added {
				for _, parent := range heads[id].parents {
					addParent(parent)
				}
			}
			frontier = frontier[:0]
			for parent := range missing {
				frontier = append(frontier, parent)
			}
			slices.Sort(frontier)
		}
	}
	for _, h := range heads {
		ds[h.id] = h.source
		relations[h.id] = h.relation
		bus = append(bus, h.bus...)
	}
	slices.Sort(bus)
	bus = slices.Compact(bus)
	predicate := "bu_id=ANY($2::text[]::uuid[])"
	if all || allBUs {
		predicate = "true"
	}
	args := []any{u.instance}
	if !all && !allBUs {
		args = append(args, bus)
	}
	rows, e := u.tx.Query(ctx, "SELECT bu_id::text,head_source_id::text FROM organization.business_unit WHERE instance_id=$1 AND "+predicate, args...)
	if e != nil {
		return nil, nil, nil, e
	}
	defer rows.Close()
	for rows.Next() {
		var id, head string
		if e = rows.Scan(&id, &head); e != nil {
			return nil, nil, nil, e
		}
		bs[id] = head
	}
	return ds, bs, relations, rows.Err()
}

// departmentDescendantIDs discovers storage identities only. Domain Discover
// still determines and rechecks the lock closure from complete dated facts.
// This adjacency map is local to this read, never a cached mutable tree.
func (u *unit) departmentDescendantIDs(ctx context.Context, roots []string) ([]string, error) {
	rows, err := u.tx.Query(ctx, "SELECT department_id::text,parents::text[] FROM organization.department WHERE instance_id=$1", u.instance)
	if err != nil {
		return nil, err
	}
	children := map[string][]string{}
	for rows.Next() {
		var id string
		var parents []string
		if err = rows.Scan(&id, &parents); err != nil {
			rows.Close()
			return nil, err
		}
		for _, parent := range parents {
			children[parent] = append(children[parent], id)
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	queue := []string{}
	for _, id := range roots {
		if !seen[id] {
			seen[id] = true
			queue = append(queue, id)
		}
	}
	for index := 0; index < len(queue); index++ {
		if err = ctx.Err(); err != nil {
			return nil, err
		}
		for _, id := range children[queue[index]] {
			if !seen[id] {
				seen[id] = true
				queue = append(queue, id)
			}
		}
	}
	return queue, nil
}

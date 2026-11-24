package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	"chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/compact"
)

type segmentRow struct {
	id, enterprise, code, first, source, start, end string
	rule, dict, shape                               int32
	ids                                             []string
	seal                                            []byte
}

func (u *departmentUnit) readListSegments(ctx context.Context, ds, bs []map[string]any) (out d.Snapshot, err error) {
	for _, group := range []struct {
		kind  history.Kind
		wants []map[string]any
		key   string
	}{{history.Department, ds, "department_id"}, {history.BusinessUnit, bs, "bu_id"}} {
		if len(group.wants) == 0 {
			continue
		}
		table, key, prefix := "business_unit", "bu_id", "bu"
		if group.kind == history.Department {
			table, key, prefix = "department", "department_id", "department"
		}
		input := []map[string]string{}
		wanted := map[string]map[string]any{}
		for _, w := range group.wants {
			id := w[group.key].(string)
			wanted[id] = w
			for _, date := range w["dates"].([]string) {
				input = append(input, map[string]string{"id": id, "date": date})
			}
		}
		body, e := json.Marshal(input)
		if e != nil {
			return out, e
		}
		// This is a range lookup over Go-generated boundaries. No status, date
		// choice, lifecycle or authorization policy is implemented by SQL.
		query := fmt.Sprintf(`SELECT b.%s::text,b.enterprise_id::text,b.code,b.first_record_id::text,b.head_source_id::text,coalesce(lower(s.span)::text,''),coalesce(upper(s.span)::text,''),s.rule,s.dictionary_id,s.shape_id,s.ids::text[],s.envelope
 FROM jsonb_to_recordset($2::jsonb) w(id uuid,date date)
 JOIN organization.%s b ON b.instance_id=$1 AND b.%s=w.id
 JOIN LATERAL (SELECT s.* FROM organization.%s_read_segment s
 WHERE s.instance_id=b.instance_id AND s.object_id=b.%s AND s.source_id=b.head_source_id
 AND coalesce(lower(s.span),'-infinity'::date)<=w.date AND s.span @> w.date
 ORDER BY coalesce(lower(s.span),'-infinity'::date) DESC LIMIT 1) s ON true
`, key, table, key, prefix, key)
		done := foundation.BeginStage(ctx, "segments.sql_scan")
		rows, e := u.tx.Query(ctx, query, u.instance, body)
		if e != nil {
			return out, e
		}
		projections := []segmentRow{}
		dicts, shapes := map[string][]int32{}, map[string][]int32{}
		for rows.Next() {
			var r segmentRow
			if e = rows.Scan(&r.id, &r.enterprise, &r.code, &r.first, &r.source, &r.start, &r.end, &r.rule, &r.dict, &r.shape, &r.ids, &r.seal); e != nil {
				rows.Close()
				return out, e
			}
			projections = append(projections, r)
			dicts[r.enterprise] = append(dicts[r.enterprise], r.dict)
			shapes[r.enterprise] = append(shapes[r.enterprise], r.shape)
		}
		e = rows.Err()
		rows.Close()
		done()
		if e != nil {
			return out, e
		}
		if u.catalog == nil {
			u.catalog = &history.Catalog{}
		}
		for ent, ids := range dicts {
			if e = u.catalog.Prepare(ctx, u.tx, group.kind, u.instance, ent, ids, shapes[ent]); e != nil {
				return out, e
			}
		}
		found := map[string]int{}
		deps := map[string]d.Timeline{}
		bus := map[string]d.BusinessUnit{}
		decodeDone := foundation.BeginStage(ctx, "segments.decode_validate")
		for _, r := range projections {
			if e = ctx.Err(); e != nil {
				return out, e
			}
			w := wanted[r.id]
			source := w["source_id"].(string)
			// If a later statement observes a newer head, the original pinned
			// source has one explicit historical path. A missing current index
			// never falls back to decoding the full current source.
			if source != r.source {
				continue
			}
			if r.rule != history.ReadRule {
				return out, compact.ErrCorrupt
			}
			m, e := u.catalog.Get(group.kind, u.instance, r.enterprise, r.dict, r.shape)
			if e != nil {
				return out, e
			}
			seal, e := history.ReadSegmentDigest(group.kind, u.instance, r.enterprise, r.id, source, r.start, r.end, r.ids, m.DictionaryDigest, m.ShapeDigest)
			if e != nil || !bytes.Equal(seal, r.seal) {
				return out, compact.ErrCorrupt
			}
			m.Values.IDs = r.ids
			if group.kind == history.Department {
				v, e := history.DecodeDepartmentRead(ctx, m.Values, m.Data)
				if e != nil {
					return out, e
				}
				t := v.Domain()
				if t.ID != r.id || t.Enterprise != r.enterprise || t.Source != source || t.First != r.first || len(t.Records) == 0 || t.Records[0].ID != t.First || t.RelationSource != w["relation_source_id"].(string) {
					return out, compact.ErrCorrupt
				}
				t.Code = r.code
				deps[t.ID] = mergeDepartment(deps[t.ID], t)
			} else {
				v, e := history.DecodeBURead(ctx, m.Values, m.Data)
				if e != nil {
					return out, e
				}
				t := v.Timeline
				if t.ID != r.id || t.Source != source || t.First != r.first || len(t.Records) == 0 || t.Records[0].ID != t.First {
					return out, compact.ErrCorrupt
				}
				t.Code = r.code
				bus[t.ID] = mergeBU(bus[t.ID], t)
			}
			found[r.id]++
		}
		decodeDone()
		for id, w := range wanted {
			if found[id] != len(w["dates"].([]string)) {
				head, e := u.head(ctx, group.kind, id)
				if e != nil {
					return out, e
				}
				if head.Source == w["source_id"].(string) {
					return out, compact.ErrCorrupt
				}
				if group.kind == history.Department {
					v, e := u.readDepartment(ctx, id, w["source_id"].(string), false)
					if e != nil {
						return out, e
					}
					if v.State.Timeline.RelationSource != w["relation_source_id"].(string) {
						return out, compact.ErrCorrupt
					}
					deps[id] = v.State.Timeline
				} else {
					v, e := u.readBU(ctx, id, w["source_id"].(string))
					if e != nil {
						return out, e
					}
					bus[id] = departmentBU(v.State.Timeline)
				}
			}
		}
		for _, v := range deps {
			out.Departments = append(out.Departments, v)
		}
		for _, v := range bus {
			out.BusinessUnits = append(out.BusinessUnits, v)
		}
	}
	return out, nil
}

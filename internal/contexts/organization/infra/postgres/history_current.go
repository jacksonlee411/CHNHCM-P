package postgres

import (
	"bytes"
	"context"
	"fmt"

	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history/model"
	"chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/compact"
	"github.com/jackc/pgx/v5"
)

type currentFacts struct {
	BUs         map[string]model.BUState
	Departments map[string]model.DepartmentState
}
type projectionRow struct {
	head        ownerHead
	dict, shape int32
	ids         []string
	envelope    []byte
}

func (u *unit) currentFacts(ctx context.Context, kind history.Kind, wanted map[string]string) (currentFacts, error) {
	defer foundation.BeginStage(ctx, "facts.current")()
	out := currentFacts{map[string]model.BUState{}, map[string]model.DepartmentState{}}
	table, key, prefix := "business_unit", "bu_id", "bu"
	if kind == history.Department {
		table, key, prefix = "department", "department_id", "department"
	}
	ids := []string{}
	for id := range wanted {
		ids = append(ids, id)
	}
	query := fmt.Sprintf(`SELECT b.%s::text,b.enterprise_id::text,b.code,b.first_record_id::text,b.head_source_id::text,b.head_no,b.head_chunk,h.dictionary_id,b.current_shape,organization.project_ids(h.ids,b.current_refs)::text[],b.current_envelope
 FROM organization.%s b JOIN organization.%s_history h ON (h.instance_id,h.object_id,h.chunk_no)=(b.instance_id,b.%s,b.head_chunk)
 WHERE b.instance_id=$1 AND ($2 OR b.%s=ANY($3::text[]::uuid[])) ORDER BY b.%s`, key, table, prefix, key, key, key)
	sqlDone := foundation.BeginStage(ctx, "facts.sql_scan")
	// The complete-instance and explicit-identity cases must not share a generic
	// plan: its unknown boolean/array can turn a narrow locked read into a scan
	// of every current history chunk. Preserve all facts and the reread under
	// locks, while planning the query for this request's actual closure.
	rows, e := u.tx.Query(ctx, query, pgx.QueryExecModeCacheDescribe, u.instance, wanted == nil, ids)
	if e != nil {
		return out, e
	}
	projections := []projectionRow{}
	dicts, shapes := map[string][]int32{}, map[string][]int32{}
	for rows.Next() {
		var r projectionRow
		if e = rows.Scan(&r.head.ID, &r.head.Enterprise, &r.head.Code, &r.head.First, &r.head.Source, &r.head.Number, &r.head.Chunk, &r.dict, &r.shape, &r.ids, &r.envelope); e != nil {
			rows.Close()
			return out, e
		}
		projections = append(projections, r)
		dicts[r.head.Enterprise] = append(dicts[r.head.Enterprise], r.dict)
		shapes[r.head.Enterprise] = append(shapes[r.head.Enterprise], r.shape)
	}
	e = rows.Err()
	rows.Close()
	sqlDone()
	if e != nil {
		return out, e
	}
	if u.catalog == nil {
		u.catalog = &history.Catalog{}
	}
	catalogDone := foundation.BeginStage(ctx, "facts.catalog")
	for enterprise, d := range dicts {
		if e = u.catalog.Prepare(ctx, u.tx, kind, u.instance, enterprise, d, shapes[enterprise]); e != nil {
			return out, e
		}
	}
	catalogDone()
	defer foundation.BeginStage(ctx, "facts.decode_validate")()
	for _, r := range projections {
		if e = ctx.Err(); e != nil {
			return out, e
		}
		requested := wanted[r.head.ID]
		if requested != "" && requested != r.head.Source {
			if kind == history.BusinessUnit {
				v, e := u.readBU(ctx, r.head.ID, requested)
				if e != nil {
					return out, e
				}
				out.BUs[r.head.ID] = v.State
			} else {
				v, e := u.readDepartment(ctx, r.head.ID, requested, false)
				if e != nil {
					return out, e
				}
				out.Departments[r.head.ID] = v.State
			}
			continue
		}
		m, e := u.catalog.Get(kind, u.instance, r.head.Enterprise, r.dict, r.shape)
		if e != nil {
			return out, e
		}
		seal, e := history.ProjectionDigest(kind, u.instance, r.head.Enterprise, r.head.ID, r.head.Source, r.ids, m.DictionaryDigest, m.ShapeDigest)
		if e != nil || !bytes.Equal(seal, r.envelope) {
			return out, compact.ErrCorrupt
		}
		m.Values.IDs = r.ids
		if kind == history.BusinessUnit {
			v, e := history.DecodeBUState(ctx, m.Values, m.Data)
			if e != nil {
				return out, e
			}
			if v.Timeline.BUID != r.head.ID || v.Timeline.Source != r.head.Source || v.Timeline.First != r.head.First || v.Timeline.SourceNo != r.head.Number {
				return out, compact.ErrCorrupt
			}
			if e = validateBUState(v); e != nil {
				return out, e
			}
			v.Timeline.Code = r.head.Code
			out.BUs[r.head.ID] = v
		} else {
			v, e := history.DecodeDepartmentState(ctx, m.Values, m.Data)
			if e != nil {
				return out, e
			}
			if v.Timeline.ID != r.head.ID || v.Timeline.Enterprise != r.head.Enterprise || v.Timeline.Source != r.head.Source || v.Timeline.First != r.head.First || v.Timeline.SourceNo != r.head.Number {
				return out, compact.ErrCorrupt
			}
			if e = validateDepartmentState(v); e != nil {
				return out, e
			}
			v.Timeline.Code = r.head.Code
			out.Departments[r.head.ID] = v
		}
	}
	return out, nil
}

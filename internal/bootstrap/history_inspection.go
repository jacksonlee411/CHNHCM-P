package bootstrap

import (
	"context"
	"errors"
	"fmt"

	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history/model"
	"chnhcm.local/chnhcm/internal/platform/compact"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// HistoryInspection is a privileged, read-only composition-root tool used by
// verification and offline export. It is not exposed through the business API.
type HistoryInspection struct {
	BUs         []BUHistoryFact
	Departments []DepartmentHistoryFact
}
type BUHistoryFact struct {
	Instance, Enterprise, Object string
	State                        model.BUState
	Operation                    model.BUOperation
}
type DepartmentHistoryFact struct {
	Instance, Enterprise, Object string
	State                        model.DepartmentState
	Operation                    model.DepartmentOperation
	Evidence                     model.DepartmentEvidence
}

func InspectHistory(ctx context.Context, pool *pgxpool.Pool, instance string) (HistoryInspection, error) {
	out := HistoryInspection{BUs: []BUHistoryFact{}, Departments: []DepartmentHistoryFact{}}
	err := VisitHistory(ctx, pool, instance, func(b *BUHistoryFact, d *DepartmentHistoryFact) error {
		if b != nil {
			out.BUs = append(out.BUs, *b)
		}
		if d != nil {
			out.Departments = append(out.Departments, *d)
		}
		return nil
	})
	return out, err
}

// VisitHistory bounds live decoded memory to one source. It uses one read-only
// snapshot and never reconstructs identities from fixture numbering.
func VisitHistory(ctx context.Context, pool *pgxpool.Pool, instance string, visit func(*BUHistoryFact, *DepartmentHistoryFact) error) error {
	tx, e := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	cat := &history.Catalog{}
	for _, kind := range []history.Kind{history.BusinessUnit, history.Department} {
		prefix := "bu"
		if kind == history.Department {
			prefix = "department"
		}
		table, key := "business_unit", "bu_id"
		if kind == history.Department {
			table, key = "department", "department_id"
		}
		rows, e := tx.Query(ctx, fmt.Sprintf(`SELECT h.instance_id::text,h.object_id::text,h.chunk_no,o.code FROM organization.%s_history h JOIN organization.%s o ON (o.instance_id,o.%s)=(h.instance_id,h.object_id) WHERE ($1='' OR h.instance_id=NULLIF($1,'')::uuid) ORDER BY h.instance_id,h.object_id,h.chunk_no`, prefix, table, key), instance)
		if e != nil {
			return e
		}
		type directory struct {
			instance, object, code string
			number                 int32
		}
		dirs := []directory{}
		for rows.Next() {
			var r directory
			if e = rows.Scan(&r.instance, &r.object, &r.number, &r.code); e != nil {
				rows.Close()
				return e
			}
			dirs = append(dirs, r)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		for _, r := range dirs {
			if e = ctx.Err(); e != nil {
				return e
			}
			cat.Enter()
			// Release between bounded directories; decoded facts are owned by the callback.
			c, e := history.LoadChunk(ctx, tx, kind, r.instance, r.object, r.number)
			if e != nil {
				return e
			}
			roots := append(append(append([]int32{}, c.States...), c.Operations...), c.Evidence...)
			if e = cat.Prepare(ctx, tx, kind, r.instance, c.Enterprise, []int32{c.Dictionary}, roots); e != nil {
				return e
			}
			for i := range c.Sources {
				a, e := cat.Get(kind, r.instance, c.Enterprise, c.Dictionary, c.States[i])
				if e != nil {
					return e
				}
				b, e := cat.Get(kind, r.instance, c.Enterprise, c.Dictionary, c.Operations[i])
				if e != nil {
					return e
				}
				if kind == history.BusinessUnit {
					state, e := history.DecodeBUState(ctx, c.Values, a.Data)
					if e != nil {
						return e
					}
					op, e := history.DecodeBUOperation(ctx, c.Values, b.Data)
					if e != nil {
						return e
					}
					if op.Source.ID != c.ID(c.Sources[i]) || op.Source.Submission != c.ID(c.Submissions[i]) || op.Source.Subject != c.ID(c.Subjects[i]) || !compact.IsID(op.EventID) || op.Terminal.ID != op.Source.Submission || state.Timeline.BUID != r.object || state.Timeline.Source != op.Source.ID {
						return compact.ErrCorrupt
					}
					state.Timeline.Code = r.code
					if e = visit(&BUHistoryFact{r.instance, c.Enterprise, r.object, state, op}, nil); e != nil {
						return e
					}
				} else {
					state, e := history.DecodeDepartmentState(ctx, c.Values, a.Data)
					if e != nil {
						return e
					}
					op, e := history.DecodeDepartmentOperation(ctx, c.Values, b.Data)
					if e != nil {
						return e
					}
					v, e := cat.Get(kind, r.instance, c.Enterprise, c.Dictionary, c.Evidence[i])
					if e != nil {
						return e
					}
					evidence, e := history.DecodeDepartmentEvidence(ctx, c.Values, v.Data)
					if e != nil {
						return e
					}
					if op.Source.ID != c.ID(c.Sources[i]) || op.Source.Submission != c.ID(c.Submissions[i]) || op.Source.Subject != c.ID(c.Subjects[i]) || !compact.IsID(op.EventID) || op.Terminal.ID != op.Source.Submission || state.Timeline.ID != r.object || state.Timeline.Source != op.Source.ID {
						return compact.ErrCorrupt
					}
					state.Timeline.Code = r.code
					if e = visit(nil, &DepartmentHistoryFact{r.instance, c.Enterprise, r.object, state, op, evidence}); e != nil {
						return e
					}
				}
			}
			cat.Leave()
		}
	}
	if e = tx.Commit(ctx); e != nil && !errors.Is(e, pgx.ErrTxClosed) {
		return e
	}
	return nil
}

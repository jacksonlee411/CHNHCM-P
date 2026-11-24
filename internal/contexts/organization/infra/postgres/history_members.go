package postgres

import (
	"context"
	"fmt"

	"chnhcm.local/chnhcm/internal/contexts/organization/domain"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
)

type buMemberRow struct {
	RecordID, RevisionID, EffectiveDate, State, Name                string
	NameEn                                                          *string
	OperationType, OperationReason, TypeLabel, ReasonLabel, EndDate string
}

func buMembers(rows []domain.Record) []buMemberRow {
	out := make([]buMemberRow, len(rows))
	for i, r := range rows {
		out[i] = buMemberRow{r.ID, r.Revision, r.Date, r.State, r.Data.Name, r.Data.NameEn, r.Type, r.Reason, r.TypeLabel, r.ReasonLabel, r.End}
	}
	return out
}
func (u *unit) eachOperation(ctx context.Context, kind history.Kind, id string, through int64, fn func(history.Chunk, int, []byte) error) error {
	prefix := "bu"
	if kind == history.Department {
		prefix = "department"
	}
	rows, e := u.tx.Query(ctx, fmt.Sprintf(`SELECT chunk_no FROM organization.%s_history WHERE instance_id=$1 AND object_id=$2 AND first_no<=$3 ORDER BY chunk_no`, prefix), u.instance, id, through)
	if e != nil {
		return e
	}
	numbers := []int32{}
	for rows.Next() {
		var n int32
		if e = rows.Scan(&n); e != nil {
			rows.Close()
			return e
		}
		numbers = append(numbers, n)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	if u.catalog == nil {
		u.catalog = &history.Catalog{}
	}
	for _, n := range numbers {
		c, e := history.LoadChunk(ctx, u.tx, kind, u.instance, id, n)
		if e != nil {
			return e
		}
		if e = u.catalog.Prepare(ctx, u.tx, kind, u.instance, c.Enterprise, []int32{c.Dictionary}, c.Operations); e != nil {
			return e
		}
		for i, op := range c.Operations {
			if c.First+int64(i) > through {
				break
			}
			m, e := u.catalog.Get(kind, u.instance, c.Enterprise, c.Dictionary, op)
			if e != nil {
				return e
			}
			if e = fn(c, i, m.Data); e != nil {
				return e
			}
		}
	}
	return nil
}

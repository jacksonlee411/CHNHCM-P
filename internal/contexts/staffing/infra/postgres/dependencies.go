package postgres

import (
	"context"
	"errors"

	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	"github.com/jackc/pgx/v5"
)

// This is only a tentative guard plan, never business validation. Both sets
// come from one statement snapshot. LockObject still rereads and validates the
// entire sealed source under the consumer guard, rolling back and replanning
// if that authoritative source adds a dependency. Missing/corrupt seals cannot
// be accepted through this cheaper pre-lock read.
func (u *unit) dependencyIDs(ctx context.Context, k d.Kind, id string) (families, functions, grades []string, err error) {
	ent, err := u.Enterprise(ctx, false)
	if err != nil {
		return nil, nil, nil, err
	}
	err = u.tx.QueryRow(ctx, `SELECT
 ARRAY(SELECT DISTINCT r->'data'->>'jobFamilyId' FROM jsonb_array_elements(h.head->'records') r WHERE COALESCE(r->'data'->>'jobFamilyId','')<>''),
 ARRAY(SELECT DISTINCT r->'data'->>'jobFunctionId' FROM jsonb_array_elements(h.head->'records') r WHERE COALESCE(r->'data'->>'jobFunctionId','')<>''),
 ARRAY(SELECT DISTINCT m.grade_id::text FROM `+table(k)+`_grade_member m WHERE m.instance_id=h.instance_id AND m.object_id=h.id AND m.source_id=h.source_id)
 FROM `+table(k)+` h WHERE h.instance_id=$1 AND h.enterprise_id=$2 AND h.id=$3`, u.store.Instance, ent.ID, id).Scan(&families, &functions, &grades)
	if errors.Is(err, pgx.ErrNoRows) {
		err = nil
	}
	return
}

// Member qualification, adoption and current names need all dated Grade facts,
// but not operation summaries or descriptions. This is the same Go-produced
// projection used by candidates, with record/revision identity for adoption.
// Accurate historical display continues to read the immutable original source.
func (u *unit) gradeHeads(ctx context.Context, enterprise string, ids []string) ([]d.Timeline, error) {
	rows, err := u.tx.Query(ctx, `SELECT h.id::text,h.code,h.source_id::text,r.record_id::text,r.fact->>'revision',r.start_date::text,COALESCE(r.end_date::text,''),r.fact->>'type',r.fact->'data'->>'name'
 FROM staffing.grade h JOIN staffing.grade_read r ON (r.instance_id,r.object_id,r.source_id)=(h.instance_id,h.id,h.source_id)
 WHERE h.instance_id=$1 AND h.enterprise_id=$2 AND h.id=ANY($3::uuid[]) ORDER BY h.id,r.start_date`, u.store.Instance, enterprise, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []d.Timeline{}
	for rows.Next() {
		var id, code, source, name string
		var r d.Record
		if err = rows.Scan(&id, &code, &source, &r.ID, &r.Revision, &r.Date, &r.End, &r.Type, &name); err != nil {
			return nil, err
		}
		if len(out) == 0 || out[len(out)-1].ID != id {
			out = append(out, d.Timeline{Kind: d.Grade, Enterprise: enterprise, ID: id, Code: code, Source: source})
		}
		r.Data = map[string]string{"name": name}
		out[len(out)-1].Records = append(out[len(out)-1].Records, r)
	}
	return out, rows.Err()
}

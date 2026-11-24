package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"slices"

	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
)

func (u *unit) LockQuotaDepartments(ctx context.Context, departments []string) error {
	ids := slices.Clone(departments)
	slices.Sort(ids)
	for _, id := range slices.Compact(ids) {
		if err := u.guardLock(ctx, u.tx, false, "staffing.department.quota", u.store.Instance, id); err != nil {
			return err
		}
	}
	return nil
}

func (u *unit) DepartmentMembers(ctx context.Context, id string) ([]d.AllocatedPosition, error) {
	ent, err := u.Enterprise(ctx, false)
	if err != nil {
		return nil, err
	}
	rows, err := u.tx.Query(ctx, `SELECT p.id,p.source_id,r.start_date::text,COALESCE(r.end_date::text,''),r.active,r.fact->'data'->>'departmentId',s.snapshot
FROM staffing.position p JOIN staffing.position_read r ON (p.instance_id,p.id,p.source_id)=(r.instance_id,r.object_id,r.source_id)
LEFT JOIN staffing.quota q ON (q.instance_id,q.enterprise_id,q.target_id)=(p.instance_id,p.enterprise_id,p.id) AND q.target_kind='position'
LEFT JOIN staffing.quota_source s ON (s.instance_id,s.object_id,s.source_id)=(q.instance_id,q.id,q.source_id)
WHERE p.instance_id=$1 AND p.enterprise_id=$2 AND EXISTS (SELECT 1 FROM staffing.position_read member
 WHERE (member.instance_id,member.object_id,member.source_id)=(p.instance_id,p.id,p.source_id) AND member.fact->'data'->>'departmentId'=$3)
ORDER BY p.id,r.start_date`, u.store.Instance, ent.ID, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []d.AllocatedPosition
	for rows.Next() {
		var position, source string
		var period d.QuotaPeriod
		var snapshot []byte
		if err = rows.Scan(&position, &source, &period.Date, &period.End, &period.Active, &period.DepartmentID, &snapshot); err != nil {
			return nil, err
		}
		if len(out) == 0 || out[len(out)-1].Position.ID != position {
			member := d.AllocatedPosition{Position: d.QuotaPeriods{ID: position, Source: source, Kind: d.Position}}
			if len(snapshot) > 0 {
				if err = json.Unmarshal(snapshot, &member.Quota); err != nil {
					return nil, err
				}
			}
			out = append(out, member)
		}
		p := &out[len(out)-1].Position
		p.Records = append(p.Records, period)
	}
	return out, rows.Err()
}

func (s *Store) DepartmentQuotaStart(ctx context.Context, enterprise, id string) (string, error) {
	tx, ok := dbtx.From(ctx)
	if !ok {
		return "", errors.New("department quota requires current transaction")
	}
	var first string
	err := tx.QueryRow(ctx, `SELECT COALESCE(min(r.start_date)::text,'') FROM staffing.quota q JOIN staffing.quota_record r ON (r.instance_id,r.object_id)=(q.instance_id,q.id) WHERE q.instance_id=$1 AND q.enterprise_id=$2 AND q.target_kind='department' AND q.target_id=$3`, s.Instance, enterprise, id).Scan(&first)
	return first, err
}

func (u *unit) DepartmentQuotaIDs(ctx context.Context) ([]string, error) {
	e, err := u.Enterprise(ctx, false)
	if err != nil {
		return nil, err
	}
	rows, err := u.tx.Query(ctx, `SELECT target_id::text FROM staffing.quota WHERE instance_id=$1 AND enterprise_id=$2 AND target_kind='department' ORDER BY target_id`, u.store.Instance, e.ID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

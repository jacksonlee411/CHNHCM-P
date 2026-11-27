package postgres

import (
	"context"
	"errors"
	"time"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	department "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
	"github.com/jackc/pgx/v5"
)

// ManagementTargets returns only the identification fields requested by X03's delegation directory.
func (s *Store) ManagementTargets(ctx context.Context, ids []string) ([]access.Target, error) {
	tx, ok := dbtx.From(ctx)
	if !ok {
		return nil, errors.New("management directory requires a transaction")
	}
	day := time.Now().In(time.FixedZone("CST", 8*60*60)).Format(time.DateOnly)
	rows, e := tx.Query(ctx, `SELECT 'enterprise',enterprise_id::text,code,name FROM organization.enterprise WHERE instance_id=$1 AND ($2::uuid[] IS NULL OR enterprise_id=ANY($2))
 UNION ALL SELECT 'department',h.department_id::text,h.code,d.q_name FROM organization.department h JOIN organization.department_read_segment d
 ON (h.instance_id,h.department_id,h.head_source_id)=(d.instance_id,d.object_id,d.source_id) WHERE h.instance_id=$1 AND d.span @> $3::date AND ($2::uuid[] IS NULL OR h.department_id=ANY($2))`, s.Instance, ids, day)
	if e != nil {
		return nil, e
	}
	out, e := pgx.CollectRows(rows, func(r pgx.CollectableRow) (access.Target, error) {
		var t access.Target
		err := r.Scan(&t.Kind, &t.ID, &t.Code, &t.Name)
		return t, err
	})
	if e != nil {
		return nil, e
	}
	var wanted map[string]string
	if ids != nil {
		wanted = map[string]string{}
		for _, id := range ids {
			wanted[id] = ""
		}
	}
	s.historyCatalog.Enter()
	defer s.historyCatalog.Leave()
	u := unit{tx: tx, instance: s.Instance, listCache: s.departmentReadCache(), catalog: &s.historyCatalog}
	bus, e := u.buListFacts(ctx, wanted, day)
	if e != nil {
		return nil, e
	}
	for _, bu := range bus {
		if record := department.BUAt(bu.Records, day); record != nil {
			out = append(out, access.Target{Kind: "bu", ID: bu.ID, Code: bu.Code, Name: record.Name})
		}
	}
	return out, nil
}

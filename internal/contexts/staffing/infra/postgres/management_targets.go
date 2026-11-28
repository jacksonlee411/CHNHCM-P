package postgres

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"time"
)

func (s *Store) ManagementTargets(ctx context.Context, ids []string) ([]access.Target, error) {
	tx, ok := dbtx.From(ctx)
	if !ok {
		return nil, errors.New("management directory requires a transaction")
	}
	day := time.Now().In(time.FixedZone("CST", 8*60*60)).Format(time.DateOnly)
	rows, e := tx.Query(ctx, `SELECT 'position',p.id::text,p.code,r.fact->'data'->>'name'
 FROM staffing.position p JOIN staffing.position_read r ON (p.instance_id,p.id,p.source_id)=(r.instance_id,r.object_id,r.source_id)
 WHERE p.instance_id=$1 AND ($2::uuid[] IS NULL OR p.id=ANY($2)) AND r.valid_period @> $3::date`, s.Instance, ids, day)
	if e != nil {
		return nil, e
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (access.Target, error) {
		var t access.Target
		err := r.Scan(&t.Kind, &t.ID, &t.Code, &t.Name)
		return t, err
	})
}

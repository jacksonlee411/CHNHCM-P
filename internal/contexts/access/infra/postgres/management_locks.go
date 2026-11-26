package postgres

import (
	"context"
	"errors"
	"slices"

	"chnhcm.local/chnhcm/internal/contexts/access/infra/postgres/generated"
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"github.com/jackc/pgx/v5"
)

// LockManagement fixes participants and definitions before a management use
// case checks delegation in this same transaction. The retired row alone
// needs an exclusive version lock. All callers order keys the same way.
func LockManagement(ctx context.Context, tx pgx.Tx, instance, actor, target string, binding access.VersionBinding, retireKind, retireID string) error {
	subjects := []string{}
	if actor != "" {
		subjects = append(subjects, actor)
	}
	if target != "" {
		subjects = append(subjects, target)
	}
	slices.Sort(subjects)
	for _, subject := range slices.Compact(subjects) {
		if _, err := generated.New(tx).LockPolicyExclusive(ctx, generated.LockPolicyExclusiveParams{InstanceID: instance, PrincipalID: subject}); err != nil {
			return err
		}
	}
	rows, err := tx.Query(ctx, `WITH keys(kind,id) AS (
 SELECT ref.kind,ref.id FROM access.grant_assignment a JOIN access.grant_revision b USING(instance_id,grant_id,revision)
 CROSS JOIN LATERAL (VALUES ('fields',b.fields_id),('role',b.role_id),('scope',b.scope_id)) ref(kind,id)
 WHERE a.instance_id=$1 AND a.principal_id=$2 AND b.operation<>'revoke'
 UNION SELECT * FROM (VALUES ('fields',$3::uuid),('role',$4::uuid),('scope',$5::uuid),($6::text,$7::uuid)) input
 ) SELECT kind,id::text FROM keys WHERE id IS NOT NULL ORDER BY kind COLLATE "C",id`,
		instance, nullable(actor), nullable(binding.Fields), nullable(binding.Role), nullable(binding.Scope), retireKind, nullable(retireID))
	if err != nil {
		return err
	}
	type key struct{ kind, id string }
	keys, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (key, error) { var k key; err := r.Scan(&k.kind, &k.id); return k, err })
	if err != nil {
		return err
	}
	batch := &pgx.Batch{}
	for _, k := range keys {
		query := `SELECT version_id FROM access.policy_version WHERE instance_id=$1 AND kind=$2 AND version_id=$3 FOR SHARE`
		if k.kind == retireKind && k.id == retireID {
			query = `SELECT version_id FROM access.policy_version WHERE instance_id=$1 AND kind=$2 AND version_id=$3 FOR UPDATE`
		}
		batch.Queue(query, instance, k.kind, k.id)
	}
	result := tx.SendBatch(ctx, batch)
	defer result.Close()
	for range keys {
		tag, err := result.Exec()
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return errors.New("referenced policy version missing")
		}
	}
	return result.Close()
}

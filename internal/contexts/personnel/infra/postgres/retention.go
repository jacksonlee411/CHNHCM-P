package postgres

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DisposeDue runs with the maintenance connection. Holds and expiry decisions
// are serialized with the deletion of the private payload and its audit entry.
func DisposeDue(ctx context.Context, pool *pgxpool.Pool) error {
	tx, e := pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, "lock table personnel.hold in share mode"); e != nil {
		return e
	}
	// Serialize with saves, including copying an unedited item to a new source.
	if _, e = tx.Exec(ctx, "select pg_advisory_xact_lock(hashtextextended(instance_id::text||':personnel-writes',0)) from (select distinct instance_id from personnel.person order by instance_id) customers"); e != nil {
		return e
	}
	for _, rule := range []struct{ category, relation, expires, item string }{
		{"addresses", "profile_item_payload", "v.evidence_until", "v.item_id::text"}, {"family", "profile_item_payload", "v.evidence_until", "v.item_id::text"}, {"identity", "identity_payload", "p.evidence_until", "''"}, {"evidence", "evidence_payload", "p.material_evidence_until", "''"}, {"contacts", "contact_payload", "v.evidence_until", "v.item_id::text"}} {
		categoryExpr := "$1::text"
		if rule.category == "contacts" {
			categoryExpr = "'contact.'||v.kind"
		}
		section := ""
		if rule.relation == "profile_item_payload" {
			section = " AND v.section=$1"
		}
		sql := `with removed as (
 delete from personnel.` + rule.relation + ` v using personnel.person p
 where v.instance_id=p.instance_id and v.person_id=p.person_id and ` + rule.expires + `<=statement_timestamp()` + section + `
 and not exists(select from personnel.hold h where (h.instance_id,h.person_id,h.version)=(v.instance_id,v.person_id,v.version)
 and h.category=$1 and h.until_at>statement_timestamp())
 returning v.instance_id,v.person_id,v.version,` + rule.item + ` as item,` + categoryExpr + ` as category)
 insert into personnel.disposition(instance_id,person_id,version,category,item) select instance_id,person_id,version,category,item from removed on conflict do nothing`
		if _, e = tx.Exec(ctx, sql, rule.category); e != nil {
			return e
		}
	}
	return tx.Commit(ctx)
}

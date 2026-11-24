//go:build integration

package bootstrap_test

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"encoding/json"
	"testing"
)

// This rolled-back SQL lower bound deliberately excludes the Go seal design,
// full candidate validation, durability and API latency. It cannot be adopted
// as a production binding or counted as a successful business save.
func prototype133SharedBinding(t *testing.T, f *fixture, id string) {
	tx, err := f.admin.Begin(f.ctx)
	must(t, err)
	defer tx.Rollback(f.ctx)
	_, err = tx.Exec(f.ctx, `CREATE TABLE organization.r133_binding_prototype(instance_id uuid NOT NULL, object_id uuid NOT NULL, source_id uuid NOT NULL, PRIMARY KEY(instance_id,object_id))`)
	must(t, err)
	_, err = tx.Exec(f.ctx, `INSERT INTO organization.r133_binding_prototype SELECT instance_id,bu_id,head_source_id FROM organization.business_unit WHERE bu_id=$1`, id)
	must(t, err)
	var count int
	must(t, tx.QueryRow(f.ctx, `SELECT count(*) FROM organization.bu_read_segment WHERE object_id=$1`, id).Scan(&count))
	if count < 10000 {
		t.Fatal("prototype requires full long current timeline", count)
	}
	rows := []map[string]any{}
	measure := func(kind, query string, args ...any) {
		_, e := tx.Exec(f.ctx, "SAVEPOINT measurement")
		must(t, e)
		var plan json.RawMessage
		must(t, tx.QueryRow(f.ctx, "EXPLAIN (ANALYZE,BUFFERS,WAL,FORMAT JSON) "+query, args...).Scan(&plan))
		rows = append(rows, map[string]any{"kind": kind, "plan": plan})
		_, e = tx.Exec(f.ctx, "ROLLBACK TO SAVEPOINT measurement")
		must(t, e)
	}
	for n := 0; n < 5; n++ {
		source := foundation.NewID()
		measure("current-all-bindings", `UPDATE organization.bu_read_segment SET source_id=$2::uuid,envelope=sha256(convert_to($2::text,'UTF8')) WHERE object_id=$1`, id, source)
		measure("shared-one-binding", `UPDATE organization.r133_binding_prototype SET source_id=$2::uuid WHERE object_id=$1`, id, source)
		measure("shared-one-changed-segment", `UPDATE organization.bu_read_segment SET source_id=$2::uuid,envelope=sha256(convert_to($2::text,'UTF8')) WHERE object_id=$1 AND slot=5000`, id, source)
		measure("current-read", `SELECT r.* FROM organization.bu_read_segment r WHERE object_id=$1 ORDER BY slot`, id)
		measure("shared-join-read", `SELECT r.*,b.source_id AS bound_source FROM organization.bu_read_segment r JOIN organization.r133_binding_prototype b USING(instance_id,object_id) WHERE object_id=$1 ORDER BY slot`, id)
	}
	var bytes int64
	must(t, tx.QueryRow(f.ctx, `SELECT pg_total_relation_size('organization.r133_binding_prototype')`).Scan(&bytes))
	raw, e := json.Marshal(map[string]any{"rows": count, "rounds": 5, "bindingTableBytes": bytes, "rolledBack": true, "productionAdopted": false, "capacityAcceptance": false, "samples": rows})
	must(t, e)
	must(t, tx.Rollback(f.ctx))
	t.Logf("R133_BINDING_PROTOTYPE %s", raw)
}

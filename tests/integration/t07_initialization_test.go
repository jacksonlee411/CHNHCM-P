//go:build integration

package integration_test

import (
	"reflect"
	"testing"

	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/foundation"
)

func TestT07InitializationAtomicityAndNoGrantExpansion(t *testing.T) {
	f := fresh(t)
	original := bootstrap.InitInput{Instance: f.instance, EnterpriseCode: "LOCAL", EnterpriseName: "本地验证企业", Start: "2020-01-01", Login: "admin", Password: f.password}
	before := businessSnapshot(t, f)
	for _, kind := range []string{"different_code", "different_start"} {
		v := original
		if kind == "different_code" {
			v.EnterpriseCode = "DIFFERENT"
		} else {
			v.Start = "2019-01-01"
		}
		if e := bootstrap.Initialize(f.ctx, f.admin, v); e == nil {
			t.Fatal(kind, "accepted replacement")
		}
	}
	if !reflect.DeepEqual(before, businessSnapshot(t, f)) {
		t.Fatal("reinitialization replaced business evidence")
	}
	failed := original
	failed.Instance = foundation.NewID()
	failed.Password = "short"
	if e := bootstrap.Initialize(f.ctx, f.admin, failed); e == nil {
		t.Fatal("bad account initialized")
	}
	for _, table := range []string{"organization.enterprise", "access.principal", "access.policy_guard", "access.subject_grant", "access.credential"} {
		if f.scalar("SELECT count(*) FROM "+table+" WHERE instance_id=$1", failed.Instance) != 0 {
			t.Fatal("half-initialized state", table)
		}
	}
	b := f.login("admin", f.password)
	mutateLifeGrants(f, b.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action IN ('bu.deactivation.redate','bu.timeline.revise')`, b.subject)
	must(t, bootstrap.Initialize(f.ctx, f.admin, original))
	if f.scalar(`SELECT count(*) FROM access.subject_grant WHERE principal_id=$1 AND action IN ('bu.deactivation.redate','bu.timeline.revise')`, b.subject) != 0 {
		t.Fatal("idempotent initialization expanded authority")
	}
	// The current DDL makes finite/ambiguous enterprise scopes unrepresentable:
	// valid range has an infinite upper bound; one enterprise per instance.
	for _, sql := range []string{
		`UPDATE organization.enterprise SET valid_range=daterange('2020-01-01','2026-01-01','[)') WHERE instance_id=$1`,
		`INSERT INTO organization.enterprise SELECT instance_id,gen_random_uuid(),code||'2',name,valid_range,initialization_digest FROM organization.enterprise WHERE instance_id=$1`,
	} {
		tx, e := f.admin.Begin(f.ctx)
		must(t, e)
		_, e = tx.Exec(f.ctx, sql, f.instance)
		if e == nil {
			t.Fatal("invalid enterprise scope represented")
		}
		must(t, tx.Rollback(f.ctx))
	}
}

func TestT07RetiredReasonRetainsExactStoredSnapshot(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	id := b.call("POST", "/api/v1/business-units", createBody("T07_OLD_REASON", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	b.call("POST", "/api/v1/business-units/"+id+"/records", addBody(detail(b, id), "2026-06-01", map[string]any{"name": "乙"}), foundation.NewID(), 201)
	d := detail(b, id)
	r := recordOn(t, d, "2026-06-01")
	// Privileged fixture representing a formerly valid reason that is absent from
	// the current Go directory. No runtime history update permission is introduced.
	f.editBURevision(r["revision"].(string), func(v *bootstrap.IntegrationBURevision) {
		v.Reason = "RETIRED_REASON"
		v.TypeLabel = "旧类型名"
		v.ReasonLabel = "旧原因名"
	})
	old := detail(b, id)
	r = recordOn(t, old, "2026-06-01")
	saved := b.call("POST", "/api/v1/business-units/"+id+"/records/"+r["recordId"].(string)+"/correct", correctBody(old, r, map[string]any{"name": "新"}), foundation.NewID(), 200)
	current := recordOn(t, detail(b, id), "2026-06-01")
	for _, field := range []string{"operationType", "operationReason", "typeLabel", "reasonLabel"} {
		if current[field] != r[field] {
			t.Fatal("historical classification refreshed", field, current, r)
		}
	}
	historical := b.call("GET", "/api/v1/business-units/"+id+"?sourceVersion="+old["headSourceVersion"].(string), nil, "", 200)
	if !reflect.DeepEqual(historical["timeline"], old["timeline"]) {
		t.Fatal("old source changed")
	}
	if saved["sourceVersion"] == old["headSourceVersion"] {
		t.Fatal("correction did not save")
	}
	bad := addBody(detail(b, id), "2026-08-01", map[string]any{"name": "丙"})
	bad["operationReason"] = "RETIRED_REASON"
	b.call("POST", "/api/v1/business-units/"+id+"/records", bad, foundation.NewID(), 400)
}

//go:build integration

package integration_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"chnhcm.local/chnhcm/db/migrations"
	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/foundation"
)

func assertCaptureRetired(t *testing.T, f *fixture) {
	t.Helper()
	if f.scalar("SELECT count(*) FROM pg_replication_slots WHERE slot_name='chnhcm_m02_commits'") != 0 ||
		f.scalar("SELECT count(*) FROM pg_publication WHERE pubname='chnhcm_m02_commits'") != 0 ||
		f.scalar("SELECT count(*) FROM pg_roles WHERE rolname IN ('chnhcm_capture','chnhcm_capture_login')") != 0 ||
		f.scalar("SELECT count(*) FROM information_schema.tables WHERE table_schema='organization' AND table_name IN ('capture_progress','capture_marker','commit_evidence')") != 0 {
		t.Fatal("retired capture facilities remain")
	}
	must(t, f.runtime.Ready(f.ctx))
}

func businessSnapshot(t *testing.T, f *fixture) map[string]string {
	t.Helper()
	return logicalBusinessSnapshot(f)
}

// 025 retires the legacy upgrade lane. Current initialization preserves current-model history.
func TestBUCurrentInitializationPreservesHistory(t *testing.T) {
	f := fresh(t)
	a := f.login("admin", f.password)
	saved := a.call("POST", "/api/v1/business-units", createBody("CURRENT_INIT", "2026-01-01"), foundation.NewID(), 201)
	before := businessSnapshot(t, f)
	must(t, bootstrap.Migrate(f.ctx, f.admin, migrations.Files))
	must(t, bootstrap.Initialize(f.ctx, f.admin, bootstrap.InitInput{Instance: f.instance, EnterpriseCode: "LOCAL", EnterpriseName: "本地验证企业", Start: "2020-01-01", Login: "admin", Password: f.password}))
	after := businessSnapshot(t, f)
	for table, original := range before {
		if after[table] != original {
			t.Fatalf("initialization changed %s", table)
		}
	}
	a.call("GET", "/api/v1/business-units/"+saved["buId"].(string)+"?sourceVersion="+saved["sourceVersion"].(string), nil, "", 200)
}

// Only the recorded-at Clock is controlled; revisions and business history are produced by the real CORRECT command.
func appendHistoryFixture(t *testing.T, f *fixture, client *browser, id, previous, name, recorded string) string {
	t.Helper()
	restore := f.recordedAt(recorded)
	defer restore()

	d := detail(client, id)
	target := selected(d)
	if d["headSourceVersion"] != previous {
		t.Fatal("fixture lost previous source")
	}
	result := client.call("POST", "/api/v1/business-units/"+id+"/records/"+target["recordId"].(string)+"/correct", correctBody(d, target, map[string]any{"name": name}), foundation.NewID(), 200)
	return result["sourceVersion"].(string)
}

func TestBUExactHistoryIgnoresAuditClockOrder(t *testing.T) {
	f := fresh(t)
	a := f.login("admin", f.password)
	assertCaptureRetired(t, f)
	var level string
	must(t, f.admin.QueryRow(f.ctx, "SHOW wal_level").Scan(&level))
	if level != "replica" {
		t.Fatal("fresh install unexpectedly requires logical WAL")
	}
	today := a.call("GET", "/api/v1/organization/context", nil, "", 200)["currentDate"].(string)
	saved := a.call("POST", "/api/v1/business-units", createBody("VERSIONS", today), foundation.NewID(), 201)
	id, first := saved["buId"].(string), saved["sourceVersion"].(string)
	second := appendHistoryFixture(t, f, a, id, first, "第二版", "2026-01-02T00:00:00Z")
	third := appendHistoryFixture(t, f, a, id, second, "第三版", "2026-01-02T00:00:00Z")
	fourth := appendHistoryFixture(t, f, a, id, third, "第四版", "2026-01-01T00:00:00Z")
	for i, v := range []struct{ source, name string }{{first, "未来业务单元"}, {second, "第二版"}, {third, "第三版"}, {fourth, "第四版"}} {
		d := a.call("GET", "/api/v1/business-units/"+id+"?sourceVersion="+v.source, nil, "", 200)
		if d["viewFact"].(map[string]any)["name"] != v.name || d["resolvedSourceVersion"] != v.source || d["readOnly"] != true {
			t.Fatalf("source %d was reinterpreted", i)
		}
	}
	current := a.call("GET", "/api/v1/business-units/"+id, nil, "", 200)
	if current["resolvedSourceVersion"] != fourth {
		t.Fatal("audit clock selected the head")
	}
	// Logs must use source order, and each action must come from its source rather than a CREATE constant.
	logs := a.call("GET", "/api/v1/business-units/"+id+"/changes", nil, "", 200)
	items := logs["items"].([]any)
	if len(items) != 4 || items[0].(map[string]any)["sourceVersion"] != fourth || items[0].(map[string]any)["action"] != "CORRECT" {
		t.Fatal("log order or action is not source-owned")
	}
	must(t, bootstrap.ChangeAccess(f.ctx, f.admin, bootstrap.AccessChange{Instance: f.instance, Login: "admin", Operation: "revoke", Action: "bu.history.read"}))
	a.call("GET", "/api/v1/business-units/"+id+"?sourceVersion="+first, nil, "", 403)
	a.call("GET", "/api/v1/business-units/"+id, nil, "", 200)
	raw, _ := json.Marshal(items)
	if strings.Contains(string(raw), "committedAt") {
		t.Fatal("retired commit timestamp leaked")
	}
	t.Log(fmt.Sprintf("four sources readable by identity despite equal/backward audit instants; current=%s", fourth))
}

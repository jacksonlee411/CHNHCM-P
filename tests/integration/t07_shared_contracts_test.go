//go:build integration

package integration_test

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/foundation"
)

func TestT07NoChangeOriginalSourceAfterLaterAcceptedWrite(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	id := b.call("POST", "/api/v1/business-units", createBody("T07_NO_CHANGE_HISTORY", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	d := detail(b, id)
	r := recordOn(t, d, "2026-01-01")
	u := correctBody(d, r, map[string]any{"name": r["name"]})
	u["changes"] = map[string]any{"name": r["name"]}
	input := u
	key := foundation.NewID()
	path := "/api/v1/business-units/" + id
	saved := b.call("POST", path+"/records/"+r["recordId"].(string)+"/correct", input, key, 200)
	if saved["outcome"] != "no_change" || saved["observedSourceVersion"] != d["headSourceVersion"] || saved["evaluatedAt"] == nil || saved["evaluatedBusinessDate"] == nil || saved["recordedAt"] != nil {
		t.Fatal(saved)
	}
	b.call("POST", path+"/records/"+r["recordId"].(string)+"/correct", correctBody(d, r, map[string]any{"name": "后来已变"}), foundation.NewID(), 200)
	for _, got := range []map[string]any{b.call("POST", path+"/records/"+r["recordId"].(string)+"/correct", input, key, 200), b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)} {
		if got["outcome"] != "no_change" || got["observedSourceVersion"] != saved["observedSourceVersion"] || !reflect.DeepEqual(got["items"], saved["items"]) {
			t.Fatal("original no-change reinterpreted using later head", got)
		}
	}
	// Early business rejection and explicit closure have no handling-day sample;
	// their later reads must not invent one from the current server clock.
	invalid := createBody("T07_EARLY_TIME", "2026-01-01")
	invalid["name"] = ""
	rejectedKey := foundation.NewID()
	b.call("POST", "/api/v1/business-units", invalid, rejectedKey, 400)
	closedKey := foundation.NewID()
	for _, identity := range []string{rejectedKey, closedKey} {
		out := b.call("POST", "/api/v1/organization/submissions/"+identity+"/resolve", nil, "", 200)
		if out["evaluatedAt"] != nil || out["evaluatedBusinessDate"] != nil || out["recordedAt"] != nil {
			t.Fatal("invented sample for early terminal", out)
		}
		terminal, err := bootstrap.InspectBUResult(f.ctx, f.admin, f.instance, b.subject, identity)
		must(t, err)
		if terminal.EvaluatedAt != nil || terminal.Result.EvaluatedBusinessDate != nil {
			t.Fatal("stored invented sample")
		}
	}
	sampled := 0
	for _, fact := range f.history().BUs {
		if fact.Object == id && fact.Operation.Terminal.Outcome == "accepted" && fact.Operation.Terminal.EvaluatedAt != nil && fact.Operation.Terminal.Result.EvaluatedBusinessDate != nil {
			sampled++
		}
	}
	if sampled != 2 {
		t.Fatal("CREATE or maintenance lacks original sample")
	}
	if f.scalar("SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.bu_history WHERE object_id=$1", id) != 2 {
		t.Fatal("replayed no-change added a source")
	}
}

func TestT07WriteGrantsCannotComposeAcrossObjectsAndMissingGuardFailsClosed(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	reader := f.login("reader", f.readPassword)
	ids := []string{}
	for i := 0; i < 2; i++ {
		ids = append(ids, b.call("POST", "/api/v1/business-units", createBody(fmt.Sprintf("T07_GRANT_%d", i), "2026-01-01"), foundation.NewID(), 201)["buId"].(string))
	}
	// The correct action on BU1 lacks fields. The same action with fields on BU2
	// cannot fill that gap; a complete grant must match this very object.
	for i, id := range ids {
		fields := []string{}
		if i == 1 {
			fields = []string{"name"}
		}
		mutateLifeGrants(f, reader.subject, `INSERT INTO access.subject_grant(instance_id,grant_id,principal_id,scope_kind,scope_id,action,read_fields,write_fields) VALUES($1,$2,$3,'bu',$4,'bu.record.correct',ARRAY[]::text[],$5)`, f.instance, foundation.NewID(), reader.subject, id, fields)
	}
	d := detail(reader, ids[0])
	r := recordOn(t, d, "2026-01-01")
	path := "/api/v1/business-units/" + ids[0] + "/records/" + r["recordId"].(string) + "/correct"
	in := correctBody(d, r, map[string]any{"name": "不可拼接"})
	reader.call("POST", path, in, foundation.NewID(), 403)
	// Missing policy authority is unavailable/denied, never an implicit grant.
	f.exec(`DELETE FROM access.policy_guard WHERE instance_id=$1 AND principal_id=$2`, f.instance, reader.subject)
	s, out, e := reader.response("POST", path, in, foundation.NewID())
	must(t, e)
	if s < 400 || out["outcome"] == "accepted" {
		t.Fatal("missing authority allowed write", s, out)
	}
	if recordOn(t, detail(b, ids[0]), "2026-01-01")["name"] != "未来业务单元" {
		t.Fatal("denied write changed facts")
	}
	// CSRF is required for a business write and logout, not only for resolve.
	old := b.csrf
	b.csrf = ""
	for _, result := range []map[string]any{b.call("POST", path, in, foundation.NewID(), 403), b.call("DELETE", "/api/v1/session", nil, "", 403)} {
		if result["code"] != "AUTH_REQUEST_VALIDATION_FAILED" {
			t.Fatal("request validation must not be business denial", result)
		}
	}
	b.csrf = old
	b.call("GET", "/api/v1/session", nil, "", 200)
}

func TestT07UnsupportedBusinessEntrypointsRemainClosed(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	id := b.call("POST", "/api/v1/business-units", createBody("T07_UNSUPPORTED", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	path := "/api/v1/business-units/" + id
	d := detail(b, id)
	before := businessSnapshot(t, f)
	for _, endpoint := range []string{path + "/void", path + "/restore", path + "/records/" + d["timeline"].([]any)[0].(map[string]any)["recordId"].(string) + "/restore", "/api/v1/business-units/import", "/api/v1/business-units/scripts"} {
		req, e := http.NewRequest("POST", f.server.URL+endpoint, strings.NewReader("{}"))
		must(t, e)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", f.server.URL)
		req.Header.Set("X-CSRF-Token", b.csrf)
		req.Header.Set("Idempotency-Key", foundation.NewID())
		res, e := b.client.Do(req)
		must(t, e)
		res.Body.Close()
		status := res.StatusCode
		if status != 404 && status != 405 {
			t.Fatal("unopened route accepted", endpoint, status)
		}
	}
	for _, action := range []string{"VOID", "RESTORE", "IMPORT", "SCRIPT"} {
		in := map[string]any{"expectedSourceVersion": d["headSourceVersion"], "items": []any{map[string]any{"itemId": foundation.NewID(), "action": action}}}
		if status, _, _ := b.response("POST", path+"/timeline-revisions", in, foundation.NewID()); status != 404 {
			t.Fatal("retired general route open", status)
		}
	}
	after := businessSnapshot(t, f)
	for table, value := range before {
		if table != "terminals" && after[table] != value {
			t.Fatal("unsupported operation changed facts", table)
		}
	}
	old := b.call("GET", path+"?sourceVersion="+d["headSourceVersion"].(string), nil, "", 200)
	if !reflect.DeepEqual(d["timeline"], old["timeline"]) {
		t.Fatal("unsupported request changed original source")
	}
}

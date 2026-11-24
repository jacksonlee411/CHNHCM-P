//go:build integration

package integration_test

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func pairInsertBody(d map[string]any, anchor map[string]any, stop, resume string) map[string]any {
	return map[string]any{"expectedSourceVersion": d["headSourceVersion"], "anchorRecordId": anchor["recordId"], "deactivationDate": stop, "deactivationReason": "BUSINESS_CLOSURE", "reactivationDate": resume, "reactivationReason": "BUSINESS_RESUMPTION"}
}
func pairDeleteBody(d map[string]any, stop, resume map[string]any) map[string]any {
	return map[string]any{"expectedSourceVersion": d["headSourceVersion"], "deactivationRecordId": stop["recordId"], "deactivationRevision": stop["revision"], "reactivationRecordId": resume["recordId"], "reactivationRevision": resume["revision"]}
}
func TestLifecyclePairInsertDeleteSourcesAndRecovery(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	id := b.call("POST", "/api/v1/business-units", createBody("PAIR_JOURNEY", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	path := "/api/v1/business-units/" + id
	b.call("POST", path+"/records", addBody(detail(b, id), "2026-06-01", map[string]any{"name": "乙", "nameEn": "Beta"}), foundation.NewID(), 201)
	b.call("POST", path+"/records", addBody(detail(b, id), "2026-10-01", map[string]any{"name": "丙"}), foundation.NewID(), 201)
	before := detail(b, id)
	first := recordOn(t, before, "2026-01-01")
	selected := b.call("GET", path+"?selectedRecordId="+first["recordId"].(string), nil, "", 200)
	window := selected["pairInsertion"].(map[string]any)
	if window["defaultStopDate"] != "2026-01-02" || window["defaultResumeDate"] != "2026-01-03" || window["maxStopDate"] != "2026-05-30" || selected["capabilities"].(map[string]any)["jointEdit"] != nil {
		t.Fatal(selected)
	}
	input := pairInsertBody(before, first, "2026-03-01", "2026-04-01")
	key := foundation.NewID()
	saved := b.call("POST", path+"/lifecycle-pairs", input, key, 200)
	if saved["outcome"] != "accepted" || len(saved["items"].([]any)) != 2 {
		t.Fatal(saved)
	}
	after := detail(b, id)
	stop, resume := recordOn(t, after, "2026-03-01"), recordOn(t, after, "2026-04-01")
	if stop["state"] != "inactive" || resume["state"] != "active" || resume["name"] != first["name"] || recordOn(t, after, "2026-06-01")["name"] != "乙" {
		t.Fatal(after)
	}
	if !reflect.DeepEqual(before["timeline"], b.call("GET", path+"?sourceVersion="+before["headSourceVersion"].(string), nil, "", 200)["timeline"]) {
		t.Fatal("old source changed")
	}
	for _, r := range []map[string]any{b.call("POST", path+"/lifecycle-pairs", input, key, 200), b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)} {
		if r["sourceVersion"] != saved["sourceVersion"] || !reflect.DeepEqual(r["items"], saved["items"]) {
			t.Fatal("pair replay changed", r)
		}
	}
	for _, r := range []map[string]any{stop, resume} {
		s := b.call("GET", path+"?selectedRecordId="+r["recordId"].(string), nil, "", 200)["pairDeletion"].(map[string]any)
		if s["deactivation"].(map[string]any)["recordId"] != stop["recordId"] || s["reactivation"].(map[string]any)["recordId"] != resume["recordId"] {
			t.Fatal(s)
		}
	}
	deleteKey := foundation.NewID()
	deleteInput := pairDeleteBody(after, stop, resume)
	deleted := b.call("POST", path+"/lifecycle-pairs/delete", deleteInput, deleteKey, 200)
	if len(detail(b, id)["timeline"].([]any)) != 3 {
		t.Fatal("pair not deleted")
	}
	if !reflect.DeepEqual(b.call("POST", "/api/v1/organization/submissions/"+deleteKey+"/resolve", nil, "", 200)["items"], deleted["items"]) {
		t.Fatal("deleted original unavailable")
	}
	b.call("POST", path+"/lifecycle-pairs/delete", deleteInput, deleteKey, 200)
	logs := b.call("GET", path+"/changes", nil, "", 200)["items"].([]any)
	if len(logs[0].(map[string]any)["items"].([]any)) != 3 {
		t.Fatal(logs)
	}
	if !reflect.DeepEqual(after["timeline"], b.call("GET", path+"?sourceVersion="+after["headSourceVersion"].(string), nil, "", 200)["timeline"]) {
		t.Fatal("deleted source rewritten")
	}
}
func TestLifecyclePairStrictScopeAndAtomicRefusal(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	id := b.call("POST", "/api/v1/business-units", createBody("PAIR_REJECT", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	path := "/api/v1/business-units/" + id
	b.call("POST", path+"/records", addBody(detail(b, id), "2026-06-01", map[string]any{"name": "乙"}), foundation.NewID(), 201)
	d := detail(b, id)
	a := recordOn(t, d, "2026-01-01")
	for _, dates := range [][2]string{{"2026-01-01", "2026-02-01"}, {"2026-03-01", "2026-03-01"}, {"2026-04-01", "2026-03-01"}, {"2026-03-01", "2026-06-01"}, {"2026-03-01", "2026-07-01"}} {
		before := businessSnapshot(t, f)
		r := pairInsertBody(d, a, dates[0], dates[1])
		status := 400
		if dates[0] == "2026-01-01" || dates[1] == "2026-06-01" {
			status = 409
		}
		key := foundation.NewID()
		rejected := b.call("POST", path+"/lifecycle-pairs", r, key, status)
		replay := b.call("POST", path+"/lifecycle-pairs", r, key, status)
		resolved := b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
		if replay["code"] != rejected["code"] || resolved["outcome"] != "rejected" {
			t.Fatal("rejection recovery", replay, resolved)
		}
		if detail(b, id)["headSourceVersion"] != d["headSourceVersion"] {
			t.Fatal("invalid pair changed head")
		}
		after := businessSnapshot(t, f)
		for table, snapshot := range before {
			if table != "terminals" && after[table] != snapshot {
				t.Fatal("partial fact", table)
			}
		}
	}
	for _, field := range []string{"items", "changes", "classification", "operationType", "name", "action"} {
		v := pairInsertBody(d, a, "2026-03-01", "2026-04-01")
		v[field] = map[string]any{}
		b.call("POST", path+"/lifecycle-pairs", v, foundation.NewID(), 400)
	}
	if status, _, _ := b.response("POST", path+"/timeline-revisions", map[string]any{"expectedSourceVersion": d["headSourceVersion"], "items": []any{}}, foundation.NewID()); status != 404 {
		t.Fatal("generic write route remains", status)
	}
	b.call("POST", path+"/preview", map[string]any{"action": "REVISE_TIMELINE", "expectedSourceVersion": d["headSourceVersion"], "items": []any{}}, "", 400)
	in := pairInsertBody(d, a, "2026-03-01", "2026-04-01")
	b.call("POST", path+"/lifecycle-pairs", in, foundation.NewID(), 200)
	b.call("POST", path+"/lifecycle-pairs", in, foundation.NewID(), 409)
	current := detail(b, id)
	stop, resume := recordOn(t, current, "2026-03-01"), recordOn(t, current, "2026-04-01")
	wrong := pairDeleteBody(current, stop, recordOn(t, current, "2026-06-01"))
	wrongKey := foundation.NewID()
	b.call("POST", path+"/lifecycle-pairs/delete", wrong, wrongKey, 409)
	b.call("POST", path+"/lifecycle-pairs/delete", wrong, wrongKey, 409)
	wrongResult := b.call("POST", "/api/v1/organization/submissions/"+wrongKey+"/resolve", nil, "", 200)
	if wrongResult["outcome"] != "rejected" {
		t.Fatal(wrongResult)
	}
	badRevision := pairDeleteBody(current, stop, resume)
	badRevision["reactivationRevision"] = foundation.NewID()
	b.call("POST", path+"/lifecycle-pairs/delete", badRevision, foundation.NewID(), 409)
	if detail(b, id)["headSourceVersion"] != current["headSourceVersion"] {
		t.Fatal("bad delete changed source")
	}
}
func TestLifecyclePairProjectedDeletionAndBothPermissions(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	reader := f.login("reader", f.readPassword)
	id := b.call("POST", "/api/v1/business-units", createBody("PAIR_RIGHTS", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	path := "/api/v1/business-units/" + id
	d := detail(b, id)
	in := pairInsertBody(d, recordOn(t, d, "2026-01-01"), "2026-03-01", "2026-04-01")
	reader.call("POST", path+"/lifecycle-pairs", in, foundation.NewID(), 403)
	mutateLifeGrants(f, reader.subject, `INSERT INTO access.subject_grant SELECT instance_id,gen_random_uuid(),$1,scope_kind,scope_id,action,read_fields,write_fields FROM access.subject_grant WHERE principal_id=$2 AND action='bu.deactivate'`, reader.subject, b.subject)
	reader.call("POST", path+"/lifecycle-pairs", in, foundation.NewID(), 403)
	mutateLifeGrants(f, reader.subject, `INSERT INTO access.subject_grant SELECT instance_id,gen_random_uuid(),$1,scope_kind,scope_id,action,read_fields,write_fields FROM access.subject_grant WHERE principal_id=$2 AND action='bu.reactivate'`, reader.subject, b.subject)
	reader.call("POST", path+"/lifecycle-pairs", in, foundation.NewID(), 200)
	d = detail(b, id)
	r := recordOn(t, d, "2026-04-01")
	b.call("POST", path+"/records/"+r["recordId"].(string)+"/correct", correctBody(d, r, map[string]any{"name": "恢复资料乙", "nameEn": "Hidden"}), foundation.NewID(), 200)
	lifeSave(b, id, "2026-07-01", "DEACTIVATE", nil)
	lifeSave(b, id, "2026-08-01", "REACTIVATE", map[string]any{"name": "丙"})
	d = detail(b, id)
	s, r := recordOn(t, d, "2026-03-01"), recordOn(t, d, "2026-04-01")
	deleteInput := pairDeleteBody(d, s, r)
	stored := f.physicalOrganizationSnapshot()
	prepared := b.call("GET", path+"?selectedRecordId="+s["recordId"].(string), nil, "", 200)["pairDeletion"].(map[string]any)
	changes := prepared["nameChanges"].([]any)
	if len(changes) != 1 || changes[0].(map[string]any)["start"] != "2026-04-01" || changes[0].(map[string]any)["endExclusive"] != "2026-08-01" || changes[0].(map[string]any)["endInclusive"] != "2026-07-31" {
		t.Fatal(prepared)
	}
	reader.call("POST", path+"/lifecycle-pairs/delete", deleteInput, foundation.NewID(), 403)
	mutateLifeGrants(f, reader.subject, `INSERT INTO access.subject_grant SELECT instance_id,gen_random_uuid(),$1,scope_kind,scope_id,action,read_fields,write_fields FROM access.subject_grant WHERE principal_id=$2 AND action IN ('bu.deactivation.delete','bu.reactivation.delete')`, reader.subject, b.subject)
	mutateLifeGrants(f, reader.subject, `UPDATE access.subject_grant SET read_fields=ARRAY['code','effectiveDate','operationType'] WHERE principal_id=$1 AND action IN ('bu.read','bu.history.read')`, reader.subject)
	safe := reader.call("GET", path+"?selectedRecordId="+s["recordId"].(string), nil, "", 200)
	raw, _ := json.Marshal(safe)
	if strings.Contains(string(raw), "Hidden") || strings.Contains(string(raw), "恢复资料乙") || len(safe["pairDeletion"].(map[string]any)["nameChanges"].([]any)) != 0 {
		t.Fatal("hidden changes leaked", safe)
	}
	if !reflect.DeepEqual(stored, f.physicalOrganizationSnapshot()) {
		t.Fatal("projecting the deletion changed stored history or receipts")
	}
	saved := reader.call("POST", path+"/lifecycle-pairs/delete", deleteInput, foundation.NewID(), 200)
	after := detail(b, id)
	if recordOn(t, after, "2026-07-01")["name"] != recordOn(t, d, "2026-01-01")["name"] || recordOn(t, after, "2026-08-01")["name"] != "丙" {
		t.Fatal(after)
	}
	mutateLifeGrants(f, reader.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action NOT IN ('bu.read','bu.history.read')`, reader.subject)
	reader.call("POST", "/api/v1/organization/submissions/"+saved["submissionId"].(string)+"/resolve", nil, "", 200)
}

func TestLifecyclePairResumeSnapshotAndDigestBinding(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	id := b.call("POST", "/api/v1/business-units", createBody("PAIR_SNAPSHOT", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	path := "/api/v1/business-units/" + id
	d := detail(b, id)
	first := recordOn(t, d, "2026-01-01")
	in := pairInsertBody(d, first, "2026-03-01", "2026-04-01")
	key := foundation.NewID()
	b.call("POST", path+"/lifecycle-pairs", in, key, 200)
	for _, field := range []string{"expectedSourceVersion", "anchorRecordId", "deactivationDate", "deactivationReason", "reactivationDate", "reactivationReason"} {
		changed := pairInsertBody(d, first, "2026-03-01", "2026-04-01")
		switch field {
		case "expectedSourceVersion", "anchorRecordId":
			changed[field] = foundation.NewID()
		case "deactivationDate":
			changed[field] = "2026-03-02"
		case "deactivationReason":
			changed[field] = "ORG_RESTRUCTURING"
		case "reactivationDate":
			changed[field] = "2026-04-02"
		case "reactivationReason":
			changed[field] = "BUSINESS_RESUMPTION "
		}
		want := 409
		if field == "reactivationReason" {
			want = 400
		}
		b.call("POST", path+"/lifecycle-pairs", changed, key, want)
	}
	d = detail(b, id)
	first = recordOn(t, d, "2026-01-01")
	resume := recordOn(t, d, "2026-04-01")
	b.call("POST", path+"/records/"+first["recordId"].(string)+"/correct", correctBody(d, first, map[string]any{"name": "前期资料更正"}), foundation.NewID(), 200)
	after := detail(b, id)
	if recordOn(t, after, "2026-03-01")["name"] != "前期资料更正" || !reflect.DeepEqual(recordOn(t, after, "2026-04-01"), resume) {
		t.Fatal("resume was not independent", after)
	}
}

//go:build integration

package integration_test

import (
	"chnhcm.local/chnhcm/internal/bootstrap"
	"fmt"
	"reflect"
	"testing"

	"chnhcm.local/chnhcm/internal/foundation"
)

func TestPositionedAdditionStrictGapSameDataAndOriginalRecovery(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	body := createBody("POSITIONED_GAP", "2026-01-01")
	body["name"], body["nameEn"] = "甲", "Alpha"
	id := b.call("POST", "/api/v1/business-units", body, foundation.NewID(), 201)["buId"].(string)
	path := "/api/v1/business-units/" + id
	b.call("POST", path+"/records", addBody(detail(b, id), "2026-06-01", map[string]any{"name": "乙", "nameEn": "Beta"}), foundation.NewID(), 201)
	b.call("POST", path+"/records", addBody(detail(b, id), "2026-10-01", map[string]any{"name": "丙", "nameEn": "Gamma"}), foundation.NewID(), 201)
	d := detail(b, id)
	anchor := recordOn(t, d, "2026-06-01")
	selected := b.call("GET", path+"?selectedRecordId="+anchor["recordId"].(string), nil, "", 200)
	c := selected["additionContext"].(map[string]any)
	if c["mode"] != "INSERT" || c["minEffectiveDate"] != "2026-06-02" || c["maxEffectiveDate"] != "2026-09-30" || c["defaultEffectiveDate"] != "2026-06-02" || c["defaults"].(map[string]any)["name"] != "乙" {
		t.Fatal(c)
	}
	for _, date := range []string{"2026-01-01", "2026-10-01", "2026-11-01", "2026-05-01"} {
		input := addBody(d, date, map[string]any{"name": "不应保存"})
		input["anchorRecordId"] = anchor["recordId"]
		status := 400
		if date == "2026-01-01" || date == "2026-10-01" {
			status = 409
		}
		b.call("POST", path+"/records", input, foundation.NewID(), status)
	}
	if detail(b, id)["headSourceVersion"] != d["headSourceVersion"] {
		t.Fatal("out-of-gap write changed source")
	}
	unchanged := addBody(d, "2026-06-02", map[string]any{})
	key := foundation.NewID()
	no := b.call("POST", path+"/records", unchanged, key, 200)
	if no["outcome"] != "no_change" || no["selectedRecordId"] != anchor["recordId"] || detail(b, id)["headSourceVersion"] != d["headSourceVersion"] {
		t.Fatal("initial defaults made an empty record", no)
	}
	acceptedBody := addBody(d, "2026-06-02", map[string]any{"name": "乙2", "nameEn": nil})
	acceptedKey := foundation.NewID()
	saved := b.call("POST", path+"/records", acceptedBody, acceptedKey, 201)
	after := detail(b, id)
	if len(after["timeline"].([]any)) != 4 || recordOn(t, after, "2026-10-01")["name"] != "丙" {
		t.Fatal(after)
	}
	if !reflect.DeepEqual(b.call("GET", path+"?sourceVersion="+d["headSourceVersion"].(string), nil, "", 200)["timeline"], d["timeline"]) {
		t.Fatal("old source rewritten")
	}
	for _, r := range []map[string]any{b.call("POST", path+"/records", unchanged, key, 200), b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)} {
		if r["observedSourceVersion"] != d["headSourceVersion"] || r["selectedRecordId"] != anchor["recordId"] {
			t.Fatal("no-change recovery rebound position", r)
		}
	}
	other := map[string]any{}
	for k, v := range acceptedBody {
		other[k] = v
	}
	other["anchorRecordId"] = recordOn(t, d, "2026-01-01")["recordId"]
	b.call("POST", path+"/records", other, acceptedKey, 409)
	replay := b.call("POST", path+"/records", acceptedBody, acceptedKey, 201)
	if replay["sourceVersion"] != saved["sourceVersion"] || replay["targetRecordId"] != saved["targetRecordId"] {
		t.Fatal(replay)
	}
	missing := addBody(after, "2026-07-01", map[string]any{"name": "缺位置"})
	delete(missing, "anchorRecordId")
	b.call("POST", path+"/records", missing, foundation.NewID(), 400)
	bad := addBody(after, "2026-07-01", map[string]any{"name": "错位置"})
	bad["anchorRecordId"] = foundation.NewID()
	rejectedKey := foundation.NewID()
	fault := b.call("POST", path+"/records", bad, rejectedKey, 400)
	if fault["field"] != "anchorRecordId" {
		t.Fatal(fault)
	}
	recovered := b.call("POST", "/api/v1/organization/submissions/"+rejectedKey+"/resolve", nil, "", 200)
	if recovered["outcome"] != "rejected" {
		t.Fatal(recovered)
	}
	terminal, e := bootstrap.InspectBUResult(f.ctx, f.admin, f.instance, b.subject, key)
	must(t, e)
	if terminal.Result.SelectedRecordID == nil || *terminal.Result.SelectedRecordID != anchor["recordId"] {
		t.Fatal("original no-change anchor missing from business result")
	}

	deleteRecord(b, id, recordOn(t, detail(b, id), "2026-06-01"))
	deletedReplay := b.call("POST", path+"/records", acceptedBody, acceptedKey, 201)
	if deletedReplay["sourceVersion"] != saved["sourceVersion"] || deletedReplay["targetRecordId"] != saved["targetRecordId"] {
		t.Fatal("deleted anchor broke accepted result recovery", deletedReplay)
	}
	deletedNoChange := b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
	if deletedNoChange["selectedRecordId"] != anchor["recordId"] || deletedNoChange["observedSourceVersion"] != d["headSourceVersion"] {
		t.Fatal("deleted anchor rebound original no-change", deletedNoChange)
	}

}

func TestPositionedAdditionRealBoundaryDatesAndScope(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	for i, tc := range []struct{ left, right, date string }{{"2026-01-01", "2026-10-01", "2026-01-02"}, {"2026-01-01", "2026-10-01", "2026-09-30"}, {"2028-02-28", "2028-03-01", "2028-02-29"}, {"2026-06-01", "2026-06-03", "2026-06-02"}, {"2026-01-01", "", "2026-02-01"}, {"9999-12-30", "", "9999-12-31"}} {
		id := b.call("POST", "/api/v1/business-units", createBody(fmt.Sprintf("POSITION_EDGE_%d", i), tc.left), foundation.NewID(), 201)["buId"].(string)
		path := "/api/v1/business-units/" + id
		if tc.right != "" {
			b.call("POST", path+"/records", addBody(detail(b, id), tc.right, map[string]any{"name": "后继"}), foundation.NewID(), 201)
		}
		before := detail(b, id)
		in := addBody(before, tc.date, map[string]any{"name": "边界资料"})
		b.call("POST", path+"/records", in, foundation.NewID(), 201)
		after := detail(b, id)
		if recordOn(t, after, tc.date)["name"] != "边界资料" {
			t.Fatal(after)
		}
		if !reflect.DeepEqual(b.call("GET", path+"?sourceVersion="+before["headSourceVersion"].(string), nil, "", 200)["timeline"], before["timeline"]) {
			t.Fatal("boundary write changed old source")
		}
	}
	id := b.call("POST", "/api/v1/business-units", createBody("POSITION_SCOPE", "2026-06-01"), foundation.NewID(), 201)["buId"].(string)
	path := "/api/v1/business-units/" + id
	b.call("POST", path+"/records", addBody(detail(b, id), "2026-06-02", map[string]any{"name": "紧邻"}), foundation.NewID(), 201)
	before := b.call("GET", path+"?selectedRecordId="+recordOn(t, detail(b, id), "2026-06-01")["recordId"].(string), nil, "", 200)
	c := before["additionContext"].(map[string]any)
	if c["available"] != false || c["defaults"] != nil || c["minEffectiveDate"] != nil || c["defaultEffectiveDate"] != nil {
		t.Fatal("unavailable gap leaks invalid form context", c)
	}
	history := b.call("GET", path+"?sourceVersion="+before["headSourceVersion"].(string), nil, "", 200)
	if history["additionContext"] != nil {
		t.Fatal("historical source offered a writable context")
	}
	foreign := b.call("POST", "/api/v1/business-units", createBody("POSITION_FOREIGN", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	in := addBody(before, "2026-06-03", map[string]any{"name": "错位"})
	in["anchorRecordId"] = selected(detail(b, foreign))["recordId"]
	if r := b.call("POST", path+"/records", in, foundation.NewID(), 400); r["field"] != "anchorRecordId" {
		t.Fatal(r)
	}
}

func TestPositionedAdditionPartialReadDefaultsAndFinalAuthorization(t *testing.T) {
	f := fresh(t)
	a := f.login("admin", f.password)
	r := f.login("reader", f.readPassword)
	body := createBody("POSITION_READ", "2026-01-01")
	body["nameEn"] = "Secret inherited English"
	id := a.call("POST", "/api/v1/business-units", body, foundation.NewID(), 201)["buId"].(string)
	path := "/api/v1/business-units/" + id
	mutateLifeGrants(f, r.subject, `INSERT INTO access.subject_grant VALUES($1,$2,$3,'bu',$4,'bu.record.add',ARRAY[]::text[],ARRAY['name','effectiveDate','operationType','operationReason'])`, f.instance, foundation.NewID(), r.subject, id)
	mutateLifeGrants(f, r.subject, `UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'nameEn') WHERE principal_id=$1 AND action IN ('bu.read','bu.history.read')`, r.subject)
	d := detail(r, id)
	c := d["additionContext"].(map[string]any)
	if c["defaults"].(map[string]any)["nameEn"] != nil {
		t.Fatal("unread English leaked", c)
	}
	in := addBody(d, "2026-02-01", map[string]any{"name": "中文变更"})
	aBefore := detail(a, id)
	b := r.call("POST", path+"/records", in, foundation.NewID(), 201)
	if b["targetRecordId"] == nil || recordOn(t, detail(a, id), "2026-02-01")["nameEn"] != "Secret inherited English" {
		t.Fatal("hidden field lost on inheritance")
	}
	mutateLifeGrants(f, r.subject, `UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'effectiveDate') WHERE principal_id=$1 AND action IN ('bu.read','bu.history.read')`, r.subject)
	if detail(r, id)["additionContext"] != nil {
		t.Fatal("unread dates leaked position context")
	}
	d = detail(a, id)
	in = addBody(d, "2026-03-01", map[string]any{"name": "无权修改", "nameEn": "disallowed"})
	r.call("POST", path+"/records", in, foundation.NewID(), 403)
	if detail(a, id)["headSourceVersion"] != d["headSourceVersion"] {
		t.Fatal("denied fields changed source")
	}
	if !reflect.DeepEqual(aBefore["timeline"], a.call("GET", path+"?sourceVersion="+aBefore["headSourceVersion"].(string), nil, "", 200)["timeline"]) {
		t.Fatal("old English snapshot changed")
	}
}

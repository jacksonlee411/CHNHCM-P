//go:build integration

package integration_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"chnhcm.local/chnhcm/internal/foundation"
)

func TestUnifiedCorrectionAtomicHistoryAndRecovery(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	input := createBody("UNIFIED_COMBO", "2025-12-01")
	input["name"] = "初始资料"
	input["nameEn"] = "Alpha"
	id := b.call("POST", "/api/v1/business-units", input, foundation.NewID(), 201)["buId"].(string)
	base := "/api/v1/business-units/" + id
	b.call("POST", base+"/records", addBody(detail(b, id), "2026-01-01", map[string]any{"name": "甲", "nameEn": "Alpha"}), foundation.NewID(), 201)
	b.call("POST", base+"/records", addBody(detail(b, id), "2026-06-01", map[string]any{"name": "乙", "nameEn": "Beta"}), foundation.NewID(), 201)
	before := detail(b, id)
	first := before["timeline"].([]any)[1].(map[string]any)
	source := before["headSourceVersion"].(string)
	path := base + "/records/" + first["recordId"].(string) + "/correct"
	body := correctBody(before, first, map[string]any{"name": "甲2", "nameEn": nil})
	body["newEffectiveDate"] = "2026-02-01"
	body["classification"] = map[string]any{"operationType": "CHANGE", "operationReason": "OTHER_DATA_ADJUSTMENT"}
	previewBody := map[string]any{}
	for k, v := range body {
		previewBody[k] = v
	}
	previewBody["action"], previewBody["recordId"] = "CORRECT", first["recordId"]
	preview := b.call("POST", base+"/preview", previewBody, "", 200)
	if preview["beforeRecord"] != nil || preview["afterRecord"].(map[string]any)["effectiveDate"] != "2026-02-01" || preview["newNoFactRange"] != nil || preview["previousCurrentStatus"] != "active" {
		t.Fatal("combined assistance lost date impact", preview)
	}
	key := foundation.NewID()
	result := b.call("POST", path, body, key, 200)
	after := detail(b, id)
	rows := after["timeline"].([]any)
	got := rows[1].(map[string]any)
	if len(rows) != 3 || got["recordId"] != first["recordId"] || got["effectiveDate"] != "2026-02-01" || got["name"] != "甲2" || got["nameEn"] != nil || got["operationReason"] != "OTHER_DATA_ADJUSTMENT" {
		t.Fatal(got)
	}
	if rows[2].(map[string]any)["nameEn"] != "Beta" {
		t.Fatal("successor overwritten")
	}
	if result["previousEffectiveDate"] != "2026-01-01" || result["selectedRecordId"] != first["recordId"] {
		t.Fatal(result)
	}
	if f.scalar("SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.bu_history") != 4 || f.eventCount() != 4 {
		t.Fatal("correction was not one source/event")
	}
	old := b.getOnDay("2026-01-01", base+"?sourceVersion="+source)
	if old["viewFact"].(map[string]any)["name"] != "甲" {
		t.Fatal("old source changed")
	}
	for _, reply := range []map[string]any{b.call("POST", path, body, key, 200), b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)} {
		if reply["sourceVersion"] != result["sourceVersion"] || reply["previousEffectiveDate"] != "2026-01-01" {
			t.Fatal("recovery lost original", reply)
		}
	}
	logs := b.call("GET", base+"/changes", nil, "", 200)["items"].([]any)
	entry := logs[0].(map[string]any)
	if entry["action"] != "CORRECT" || directChange(t, entry)["beforeContext"].(map[string]any)["effectiveDate"] != "2026-01-01" || directChange(t, entry)["afterContext"].(map[string]any)["operationReason"] != "OTHER_DATA_ADJUSTMENT" {
		t.Fatal(entry)
	}
	current := rows[1].(map[string]any)
	same := correctBody(after, current, map[string]any{"name": "甲2"})
	same["newEffectiveDate"] = "2026-02-01"
	same["classification"] = map[string]any{"operationType": "CHANGE", "operationReason": "OTHER_DATA_ADJUSTMENT"}
	if b.call("POST", path, same, foundation.NewID(), 200)["outcome"] != "no_change" || detail(b, id)["headSourceVersion"] != after["headSourceVersion"] {
		t.Fatal("same input created a revision")
	}
	// Later edits must not replace the original combined submission's dates or revision.
	later := correctBody(after, current, map[string]any{"name": "甲3"})
	later["newEffectiveDate"] = "2026-03-01"
	later["classification"] = map[string]any{"operationType": "CHANGE", "operationReason": "NAME_ADJUSTMENT"}
	b.call("POST", path, later, foundation.NewID(), 200)
	for _, reply := range []map[string]any{b.call("POST", path, body, key, 200), b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)} {
		if reply["sourceVersion"] != result["sourceVersion"] || reply["targetRevision"] != result["targetRevision"] || reply["previousEffectiveDate"] != "2026-01-01" || reply["effectiveDate"] != "2026-02-01" {
			t.Fatal("later edits replaced original combined result", reply)
		}
	}
}

func TestUnifiedCorrectionRejectsWholeInvalidCombination(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	id := b.call("POST", "/api/v1/business-units", createBody("UNIFIED_REJECT", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	base := "/api/v1/business-units/" + id
	for _, date := range []string{"2026-06-01", "2026-10-01"} {
		b.call("POST", base+"/records", addBody(detail(b, id), date, map[string]any{"name": date}), foundation.NewID(), 201)
	}
	d := detail(b, id)
	target := d["timeline"].([]any)[1].(map[string]any)
	path := base + "/records/" + target["recordId"].(string) + "/correct"
	count := f.revisionCount(false)
	for _, date := range []string{"2026-01-01", "2026-10-01", "2026-11-01"} {
		body := correctBody(d, target, map[string]any{"name": "必须回滚"})
		body["newEffectiveDate"] = date
		status := 400
		if date == "2026-01-01" || date == "2026-10-01" {
			status = 409
		}
		r := b.call("POST", path, body, foundation.NewID(), status)
		if r["field"] != "newEffectiveDate" {
			t.Fatal(r)
		}
	}
	body := map[string]any{"expectedSourceVersion": d["headSourceVersion"], "expectedRecordRevision": target["revision"], "newEffectiveDate": "2026-07-01", "classification": map[string]any{"operationType": "DEACTIVATE", "operationReason": "BUSINESS_CLOSURE"}}
	key := foundation.NewID()
	r := b.call("POST", path, body, key, 409)
	if r["code"] != "M02_TIMELINE_INVALID" {
		t.Fatal(r)
	}
	resolved := b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
	if resolved["outcome"] != "rejected" || resolved["failureDetails"] == nil {
		t.Fatal("original rejection lost", resolved)
	}
	if detail(b, id)["headSourceVersion"] != d["headSourceVersion"] || f.revisionCount(false) != count {
		t.Fatal("partial correction persisted")
	}
	for _, bad := range []map[string]any{{"classification": nil}, {"classification": map[string]any{"operationType": "CREATE", "operationReason": "NEW_ORGANIZATION", "extra": true}}, {"changes": map[string]any{}}, {"newEffectiveDate": ""}} {
		bad["expectedSourceVersion"] = d["headSourceVersion"]
		bad["expectedRecordRevision"] = target["revision"]
		b.call("POST", path, bad, foundation.NewID(), 400)
	}
}

func TestUnifiedCorrectionFacetPermissionsWithoutJointEntry(t *testing.T) {
	for _, facet := range []string{"name", "date", "classification"} {
		t.Run(facet, func(t *testing.T) {
			f := fresh(t)
			b := f.login("admin", f.password)
			id := b.call("POST", "/api/v1/business-units", createBody("UNIFIED_PERMISSION_"+facet, "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
			b.call("POST", "/api/v1/business-units/"+id+"/records", addBody(detail(b, id), "2026-06-01", map[string]any{"name": "更改资料"}), foundation.NewID(), 201)
			mutateLifeGrants(f, b.subject, "DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='bu.timeline.revise'", b.subject)
			mutateLifeGrants(f, b.subject, "UPDATE access.subject_grant SET write_fields='{}' WHERE principal_id=$1 AND action LIKE 'bu.%'", b.subject)
			action, fs := "bu.record.correct", `{"name","nameEn"}`
			if facet == "date" {
				action, fs = "bu.record.redate", `{"effectiveDate"}`
			}
			if facet == "classification" {
				action, fs = "bu.record.reclassify", `{"operationType","operationReason"}`
			}
			mutateLifeGrants(f, b.subject, "UPDATE access.subject_grant SET write_fields=$2::text[] WHERE principal_id=$1 AND action=$3", b.subject, fs, action)
			d := detail(b, id)
			target := selected(d)
			caps := d["capabilities"].(map[string]any)
			if caps["correct"] != true || caps["jointEdit"] != nil {
				t.Fatal(caps)
			}
			path := "/api/v1/business-units/" + id + "/records/" + target["recordId"].(string) + "/correct"
			inputs := map[string]map[string]any{"name": {"changes": map[string]any{"name": "更正后"}}, "date": {"newEffectiveDate": "2026-02-01"}, "classification": {"classification": map[string]any{"operationType": "CHANGE", "operationReason": "OTHER_DATA_ADJUSTMENT"}}}
			for key, body := range inputs {
				if key == facet {
					continue
				}
				body["expectedSourceVersion"] = d["headSourceVersion"]
				body["expectedRecordRevision"] = target["revision"]
				b.call("POST", path, body, foundation.NewID(), 403)
			}
			body := inputs[facet]
			body["expectedSourceVersion"] = d["headSourceVersion"]
			body["expectedRecordRevision"] = target["revision"]
			result := b.call("POST", path, body, foundation.NewID(), 200)
			if result["outcome"] != "accepted" {
				raw, _ := json.Marshal(result)
				t.Fatal(fmt.Sprint(string(raw)))
			}
		})
	}
}

func TestUnifiedCorrectionRequiresBothCategoriesForDateAndClassification(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	id := b.call("POST", "/api/v1/business-units", createBody("UNIFIED_CATEGORY_RIGHTS", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	base := "/api/v1/business-units/" + id
	b.call("POST", base+"/records", addBody(detail(b, id), "2026-06-01", map[string]any{"name": "乙"}), foundation.NewID(), 201)
	d := detail(b, id)
	r := recordOn(t, d, "2026-06-01")
	endpoint := base + "/records/" + r["recordId"].(string) + "/correct"
	body := map[string]any{"expectedSourceVersion": d["headSourceVersion"], "expectedRecordRevision": r["revision"], "newEffectiveDate": "2026-07-01", "classification": map[string]any{"operationType": "DEACTIVATE", "operationReason": "BUSINESS_CLOSURE"}}
	for _, action := range []string{"bu.record.redate", "bu.deactivation.redate", "bu.record.reclassify", "bu.deactivation.reclassify"} {
		fields := []string{"effectiveDate"}
		if action == "bu.record.reclassify" || action == "bu.deactivation.reclassify" {
			fields = []string{"operationType", "operationReason"}
		}
		mutateLifeGrants(f, b.subject, "UPDATE access.subject_grant SET write_fields='{}' WHERE principal_id=$1 AND action=$2", b.subject, action)
		b.call("POST", endpoint, body, foundation.NewID(), 403)
		if detail(b, id)["headSourceVersion"] != d["headSourceVersion"] {
			t.Fatal("denial changed source")
		}
		mutateLifeGrants(f, b.subject, "UPDATE access.subject_grant SET write_fields=$3 WHERE principal_id=$1 AND action=$2", b.subject, action, fields)
	}
	b.call("POST", endpoint, body, foundation.NewID(), 200)
	if recordOn(t, detail(b, id), "2026-07-01")["operationType"] != "DEACTIVATE" {
		t.Fatal("combined category correction not saved")
	}
}

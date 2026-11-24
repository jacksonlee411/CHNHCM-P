//go:build integration

package integration_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"chnhcm.local/chnhcm/internal/foundation"
)

func TestUnifiedCorrectionCompleteFacetMatrixAndDeclaredSameValues(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	mutateLifeGrants(f, b.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='bu.timeline.revise'`, b.subject)
	for mask := 1; mask < 8; mask++ {
		t.Run(fmt.Sprintf("facets_%d", mask), func(t *testing.T) {
			id := b.call("POST", "/api/v1/business-units", createBody(fmt.Sprintf("CORRECTION_MATRIX_%d", mask), "2025-12-01"), foundation.NewID(), 201)["buId"].(string)
			base := "/api/v1/business-units/" + id
			b.call("POST", base+"/records", addBody(detail(b, id), "2026-01-01", map[string]any{"name": "甲资料"}), foundation.NewID(), 201)
			b.call("POST", base+"/records", addBody(detail(b, id), "2026-06-01", map[string]any{"name": "独立后继"}), foundation.NewID(), 201)
			d := detail(b, id)
			first := recordOn(t, d, "2026-01-01")
			body := map[string]any{"expectedSourceVersion": d["headSourceVersion"], "expectedRecordRevision": first["revision"]}
			name, date, reason := first["name"], "2026-01-01", "NAME_ADJUSTMENT"
			if mask&1 != 0 {
				body["changes"] = map[string]any{"name": "更正名称"}
				name = "更正名称"
			}
			if mask&2 != 0 {
				body["newEffectiveDate"] = "2026-02-01"
				date = "2026-02-01"
			}
			if mask&4 != 0 {
				body["classification"] = map[string]any{"operationType": "CHANGE", "operationReason": "OTHER_DATA_ADJUSTMENT"}
				reason = "OTHER_DATA_ADJUSTMENT"
			}
			sources, events := f.scalar("SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.bu_history"), f.eventCount()
			saved := b.call("POST", base+"/records/"+first["recordId"].(string)+"/correct", body, foundation.NewID(), 200)
			after := detail(b, id)
			r := recordOn(t, after, date)
			if r["name"] != name || r["operationReason"] != reason || r["recordId"] != first["recordId"] || saved["outcome"] != "accepted" {
				t.Fatal(r, saved)
			}
			if f.scalar("SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.bu_history") != sources+1 || f.eventCount() != events+1 {
				t.Fatal("combination saved more than one source/event")
			}
			if recordOn(t, after, "2026-06-01")["name"] != "独立后继" || !reflect.DeepEqual(b.call("GET", base+"?sourceVersion="+d["headSourceVersion"].(string), nil, "", 200)["timeline"], d["timeline"]) {
				t.Fatal("successor or old source changed")
			}
		})
	}
	// Accept the literal inclusive edges through the actual command, not only the range helper.
	edgeID := b.call("POST", "/api/v1/business-units", createBody("CORRECTION_REAL_EDGES", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	edgeBase := "/api/v1/business-units/" + edgeID
	b.call("POST", edgeBase+"/records", addBody(detail(b, edgeID), "2026-06-01", map[string]any{"name": "中段"}), foundation.NewID(), 201)
	b.call("POST", edgeBase+"/records", addBody(detail(b, edgeID), "2026-10-01", map[string]any{"name": "后继"}), foundation.NewID(), 201)
	previous := "2026-06-01"
	for _, date := range []string{"2026-01-02", "2026-09-30"} {
		d := detail(b, edgeID)
		r := recordOn(t, d, previous)
		body := map[string]any{"expectedSourceVersion": d["headSourceVersion"], "expectedRecordRevision": r["revision"], "newEffectiveDate": date}
		saved := b.call("POST", edgeBase+"/records/"+r["recordId"].(string)+"/correct", body, foundation.NewID(), 200)
		if saved["effectiveDate"] != date || recordOn(t, detail(b, edgeID), date)["recordId"] != r["recordId"] {
			t.Fatal(saved)
		}
		previous = date
	}
	for _, pair := range [][2]string{{"2026-01-01", "2020-01-01"}, {"2026-10-01", "2050-01-01"}} {
		d := detail(b, edgeID)
		r := recordOn(t, d, pair[0])
		b.call("POST", edgeBase+"/records/"+r["recordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": d["headSourceVersion"], "expectedRecordRevision": r["revision"], "newEffectiveDate": pair[1]}, foundation.NewID(), 200)
		if recordOn(t, detail(b, edgeID), pair[1])["recordId"] != r["recordId"] {
			t.Fatal("enterprise edge/open end changed identity")
		}
	}
	id := b.call("POST", "/api/v1/business-units", createBody("CORRECTION_DECLARED", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	base := "/api/v1/business-units/" + id
	d := detail(b, id)
	r := selected(d)
	path := base + "/records/" + r["recordId"].(string) + "/correct"
	for _, action := range []string{"bu.first.redate", "bu.first.reclassify", "bu.record.correct"} {
		body := correctBody(d, r, map[string]any{"name": "合法名称草稿"})
		fields := []string{"effectiveDate"}
		if action == "bu.first.redate" {
			body["newEffectiveDate"] = "2026-01-01"
		}
		if action == "bu.first.reclassify" {
			body["classification"] = map[string]any{"operationType": "CREATE", "operationReason": "NEW_ORGANIZATION"}
			fields = []string{"operationType", "operationReason"}
		}
		if action == "bu.record.correct" {
			body["changes"] = map[string]any{"name": r["name"]}
			body["newEffectiveDate"] = "2026-02-01"
			fields = []string{"name", "nameEn"}
		}
		mutateLifeGrants(f, b.subject, `UPDATE access.subject_grant SET write_fields='{}' WHERE principal_id=$1 AND action=$2`, b.subject, action)
		b.call("POST", path, body, foundation.NewID(), 403)
		if detail(b, id)["headSourceVersion"] != d["headSourceVersion"] {
			t.Fatal("same-value declaration bypassed authorization")
		}
		mutateLifeGrants(f, b.subject, `UPDATE access.subject_grant SET write_fields=$3 WHERE principal_id=$1 AND action=$2`, b.subject, action, fields)
	}
	b.call("POST", path, correctBody(d, r, map[string]any{"name": "只声明名称"}), foundation.NewID(), 200)
}

func TestUnifiedCorrectionStrictNewModifierProtocol(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	id := b.call("POST", "/api/v1/business-units", createBody("CORRECTION_STRICT_NEW", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	base := "/api/v1/business-units/" + id
	d := detail(b, id)
	r := selected(d)
	path := base + "/records/" + r["recordId"].(string) + "/correct"
	for _, bad := range []map[string]any{
		{}, {"changes": nil}, {"newEffectiveDate": nil}, {"newEffectiveDate": ""}, {"newEffectiveDate": "2026-02-29"}, {"classification": nil},
		{"classification": map[string]any{"operationType": "CREATE"}}, {"classification": map[string]any{"operationReason": "NEW_ORGANIZATION"}},
		{"classification": map[string]any{"operationType": "CREATE", "operationReason": "NEW_ORGANIZATION", "reason": "extra"}},
		{"anchorRecordId": r["recordId"], "changes": map[string]any{"name": "不应保存"}},
	} {
		bad["expectedSourceVersion"], bad["expectedRecordRevision"] = d["headSourceVersion"], r["revision"]
		b.call("POST", path, bad, foundation.NewID(), 400)
	}
	duplicate := json.RawMessage(fmt.Sprintf(`{"expectedSourceVersion":%q,"expectedRecordRevision":%q,"classification":{"operationType":"CREATE","operationReason":"NEW_ORGANIZATION","operationReason":"NEW_ORGANIZATION"}}`, d["headSourceVersion"], r["revision"]))
	b.call("POST", path, duplicate, foundation.NewID(), 400)
	if detail(b, id)["headSourceVersion"] != d["headSourceVersion"] || f.scalar("SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.bu_history") != 1 || f.revisionCount(false) != 1 || f.eventCount() != 1 {
		t.Fatal("malformed modifier partially wrote facts")
	}
}

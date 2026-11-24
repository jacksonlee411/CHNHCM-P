//go:build integration

package integration_test

import (
	"fmt"
	"net/url"
	"strings"
	"testing"

	"chnhcm.local/chnhcm/internal/foundation"
)

func TestBUNamePolicyAcrossCreateAddCorrect(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	body := createBody("NAME_PROFILE", "2026-01-01")
	body["name"], body["nameEn"] = "　𠮷祥研发（一期） ", "Socie\u0301te\u0301 R&D"
	created := b.call("POST", "/api/v1/business-units", body, foundation.NewID(), 201)
	id := created["buId"].(string)
	path := "/api/v1/business-units/" + id
	d := detail(b, id)
	if selected(d)["name"] != "𠮷祥研发（一期）" || selected(d)["nameEn"] != "Société R&D" || d["nameGuidance"] == "" {
		t.Fatal("normalized create", d)
	}
	before := businessSnapshot(t, f)
	for i, value := range []string{"华东😀", "中国🇨🇳", "第1️⃣部", "开发👩‍💻", "👍🏽研发", "甲\n", "\t甲", "甲\u200b乙", "甲\u202e乙", "甲\ue000乙", "---"} {
		for _, field := range []string{"name", "nameEn"} {
			badCreate := createBody(fmt.Sprintf("INVALID_%d_%s", i, field), "2026-01-01")
			badCreate[field] = value
			key := foundation.NewID()
			b.call("POST", "/api/v1/business-units", badCreate, key, 400)
			b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
			changes := map[string]any{field: value}
			b.call("POST", path+"/records", addBody(d, "2026-03-01", changes), foundation.NewID(), 400)
			b.call("POST", path+"/records/"+selected(d)["recordId"].(string)+"/correct", correctBody(d, selected(d), changes), foundation.NewID(), 400)
			b.call("POST", path+"/preview", map[string]any{"action": "ADD", "anchorRecordId": d["timeline"].([]any)[0].(map[string]any)["recordId"], "expectedSourceVersion": d["headSourceVersion"], "effectiveDate": "2026-03-01", "operationType": "CHANGE", "operationReason": "NAME_ADJUSTMENT", "changes": changes}, "", 400)
		}
	}
	for table, got := range businessSnapshot(t, f) {
		if table != "terminals" && before[table] != got {
			t.Fatal("invalid names wrote business facts", table)
		}
	}
	if f.scalar("SELECT count(*) FROM organization.bu_terminal WHERE outcome='rejected'") != 66 {
		t.Fatal("invalid saves must retain their rejection outcomes")
	}
	// Raw input exceeds 200 codepoints but NFC is exactly 200. No outer
	// protocol cap may disagree with the domain's normalized length.
	valid := strings.Repeat("e\u0301", 200)
	input := addBody(d, "2026-03-01", map[string]any{"name": valid})
	key := foundation.NewID()
	added := b.call("POST", path+"/records", input, key, 201)
	d = detail(b, id)
	if selected(d)["name"] != strings.Repeat("é", 200) || selected(d)["nameEn"] != "Société R&D" {
		t.Fatal("NFC add or inheritance", d)
	}
	recovered := b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
	if recovered["sourceVersion"] != added["sourceVersion"] {
		t.Fatal("result source changed")
	}
	noChange := b.call("POST", path+"/records/"+selected(d)["recordId"].(string)+"/correct", correctBody(d, selected(d), map[string]any{"name": valid}), foundation.NewID(), 200)
	if noChange["outcome"] != "no_change" {
		t.Fatal("canonical equivalent created fact", noChange)
	}
	correct := b.call("POST", path+"/records/"+selected(d)["recordId"].(string)+"/correct", correctBody(d, selected(d), map[string]any{"name": "نامه\u200cای / क्\u200dष", "nameEn": nil}), foundation.NewID(), 200)
	if correct["outcome"] != "accepted" {
		t.Fatal(correct)
	}
	old := b.getOnDay("2026-03-01", path+"?sourceVersion="+added["sourceVersion"].(string))
	if selected(old)["name"] != strings.Repeat("é", 200) || selected(old)["nameEn"] != "Société R&D" {
		t.Fatal("old source changed", old)
	}
	search := b.call("GET", "/api/v1/business-units?q="+url.QueryEscape("نامه\u200cای"), nil, "", 200)
	if search["count"] != float64(1) {
		t.Fatal("allowed joining text not searchable", search)
	}
}

//go:build integration

package integration_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"chnhcm.local/chnhcm/internal/foundation"
)

func TestLifecyclePreviewAndCommitInheritanceImpact(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	id := b.call("POST", "/api/v1/business-units", createBody("INHERITANCE_IMPACT", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	path := "/api/v1/business-units/" + id
	lifeSave(b, id, "2026-03-01", "DEACTIVATE", nil)
	resumed := lifeSave(b, id, "2026-04-01", "REACTIVATE", map[string]any{"name": "乙"})
	stopped := lifeSave(b, id, "2026-05-01", "DEACTIVATE", nil)
	lifeSave(b, id, "2026-06-01", "REACTIVATE", map[string]any{"name": "丙"})
	d := b.getOnDay("2026-05-01", path+"?selectedRecordId="+resumed["selectedRecordId"].(string))
	r := selected(d)
	s := d["viewFact"].(map[string]any)
	oldSource := d["resolvedSourceVersion"].(string)
	old := b.getOnDay("2026-05-01", path+"?sourceVersion="+oldSource)
	body := correctBody(d, r, map[string]any{"name": "乙2"})
	body["action"], body["recordId"] = "CORRECT", r["recordId"]
	preview := b.call("POST", path+"/preview", body, "", 200)
	delete(body, "action")
	delete(body, "recordId")
	saved := b.call("POST", path+"/records/"+r["recordId"].(string)+"/correct", body, foundation.NewID(), 200)
	p := preview["impact"].(map[string]any)
	wantStructure := []any{map[string]any{"recordId": stopped["selectedRecordId"], "beforeRevisionId": s["revision"], "afterRevisionId": nil, "facets": []any{"INHERITANCE"}}}
	if !reflect.DeepEqual(p["structuralChanges"], wantStructure) || !reflect.DeepEqual(p["affectedRecordIds"], []any{r["recordId"], s["recordId"]}) || !reflect.DeepEqual(p["dataAffectedRanges"], []any{map[string]any{"start": "2026-04-01", "endExclusive": "2026-06-01"}}) {
		t.Fatal("preview must explain the exact inherited change", p)
	}
	committed := saved["impact"].(map[string]any)
	event, err := json.Marshal(f.buSource(saved["sourceVersion"].(string)).Operation.Event().Impact)
	must(t, err)
	var eventImpact map[string]any
	must(t, json.Unmarshal(event, &eventImpact))
	if !reflect.DeepEqual(eventImpact, committed) {
		t.Fatal("outbox and result disagree", eventImpact, committed)
	}
	current := b.getOnDay("2026-05-01", path)["viewFact"].(map[string]any)
	structure := committed["structuralChanges"].([]any)[0].(map[string]any)
	if structure["afterRevisionId"] != current["revision"] || structure["afterRevisionId"] == s["revision"] || current["name"] != "乙2" {
		t.Fatal("saved inheritance revision incorrect", structure, current)
	}
	// Only final allocated revision IDs differ; all impact semantics are identical.
	structure["afterRevisionId"] = nil
	if !reflect.DeepEqual(p, committed) {
		t.Fatal("preview/commit mismatch", p, committed)
	}
	if b.getOnDay("2026-06-01", path)["viewFact"].(map[string]any)["name"] != "丙" {
		t.Fatal("correction propagated into independent reactivation")
	}
	oldAgain := b.getOnDay("2026-05-01", path+"?sourceVersion="+oldSource)
	if !reflect.DeepEqual(old["timeline"], oldAgain["timeline"]) {
		t.Fatal("original source changed")
	}
	var inherited string
	for _, r := range f.buSource(saved["sourceVersion"].(string)).State.Timeline.Records {
		if r.Revision == current["revision"] {
			inherited = r.InheritedRevision
		}
	}
	if inherited != saved["targetRevision"] {
		t.Fatal("stop does not reference the new predecessor revision", inherited, saved["targetRevision"])
	}
}

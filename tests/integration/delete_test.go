//go:build integration

package integration_test

import (
	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/foundation"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgconn"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func deleteBody(d, r map[string]any) map[string]any {
	return map[string]any{"expectedSourceVersion": d["headSourceVersion"], "expectedRecordRevision": r["revision"]}
}
func recordOn(t *testing.T, d map[string]any, date string) map[string]any {
	t.Helper()
	for _, raw := range d["timeline"].([]any) {
		r := raw.(map[string]any)
		if r["effectiveDate"] == date {
			return r
		}
	}
	t.Fatal("record missing", date)
	return nil
}
func deleteRecord(b *browser, id string, r map[string]any) map[string]any {
	return b.call("POST", "/api/v1/business-units/"+id+"/records/"+r["recordId"].(string)+"/delete", deleteBody(detail(b, id), r), foundation.NewID(), 200)
}

func TestDeleteJourneyExactOriginalResultAndSameDayNewIdentity(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	body := createBody("DELETE_JOURNEY", "2026-01-01")
	body["name"] = "甲"
	body["nameEn"] = "Alpha"
	id := b.call("POST", "/api/v1/business-units", body, foundation.NewID(), 201)["buId"].(string)
	path := "/api/v1/business-units/" + id
	b.call("POST", path+"/records", addBody(detail(b, id), "2026-06-01", map[string]any{"name": "乙", "nameEn": "Beta"}), foundation.NewID(), 201)
	b.call("POST", path+"/records", addBody(detail(b, id), "2026-10-01", map[string]any{"name": "丙", "nameEn": "Gamma"}), foundation.NewID(), 201)
	d := detail(b, id)
	target := recordOn(t, d, "2026-06-01")
	first := recordOn(t, d, "2026-01-01")
	last := recordOn(t, d, "2026-10-01")
	oldSource := d["headSourceVersion"].(string)
	input := deleteBody(d, target)
	key := foundation.NewID()
	endpoint := path + "/records/" + target["recordId"].(string) + "/delete"
	previewInput := deleteBody(d, target)
	previewInput["action"] = "DELETE"
	previewInput["recordId"] = target["recordId"]
	preview := b.call("POST", path+"/preview", previewInput, "", 200)
	if preview["afterRecord"].(map[string]any)["name"] != "甲" {
		t.Fatal("wrong target-date fact", preview)
	}
	saved := b.call("POST", endpoint, input, key, 200)
	if saved["targetRecordId"] != target["recordId"] || saved["selectedRecordId"] != first["recordId"] || saved["targetRevision"] == target["revision"] || saved["effectiveDate"] != "2026-06-01" {
		t.Fatal("deletion result is not its direct tombstone", saved)
	}
	current := detail(b, id)
	if len(current["timeline"].([]any)) != 2 || recordOn(t, current, "2026-01-01")["endExclusive"] != "2026-10-01" || !reflect.DeepEqual(recordOn(t, current, "2026-10-01"), last) {
		t.Fatal("membership/independent successor", current)
	}
	for _, at := range []string{"2026-06-01", "2026-09-30"} {
		v := b.getOnDay(at, path)
		if v["viewFact"].(map[string]any)["name"] != "甲" {
			t.Fatal(v)
		}
	}
	old := b.getOnDay("2026-06-01", path+"?sourceVersion="+oldSource)
	if !reflect.DeepEqual(old["timeline"], d["timeline"]) {
		t.Fatal("old source changed")
	}
	original := b.getOnDay("2026-11-01", path+"?sourceVersion="+saved["sourceVersion"].(string)+"&selectedRecordId="+target["recordId"].(string))
	tombstone := original["selectedRecord"].(map[string]any)
	if tombstone["deleted"] != true || tombstone["revision"] != saved["targetRevision"] || tombstone["name"] != "乙" || tombstone["endExclusive"] != "2026-10-01" || original["viewFact"].(map[string]any)["name"] != "丙" {
		t.Fatal("recovery substituted current fact", original)
	}
	logs := b.call("GET", path+"/changes", nil, "", 200)["items"].([]any)
	log := logs[0].(map[string]any)
	if len(logs) != 4 || log["action"] != "DELETE" || directChange(t, log)["afterRef"] != nil || directChange(t, log)["beforeRef"].(map[string]any)["revision"] != target["revision"] {
		t.Fatal("deletion log missing original", logs)
	}
	if f.revisionCount(true) != 1 || containsBURevision(f.buSource(saved["sourceVersion"].(string)), saved["targetRevision"].(string)) {
		t.Fatal("deletion leaked into membership")
	}
	event, err := json.Marshal(f.buSource(saved["sourceVersion"].(string)).Operation.Event().Impact)
	must(t, err)
	var eventImpact map[string]any
	must(t, json.Unmarshal(event, &eventImpact))
	if !reflect.DeepEqual(eventImpact, saved["impact"]) {
		t.Fatal("event/result impact mismatch")
	}
	structures := eventImpact["structuralChanges"].([]any)
	found := false
	for _, raw := range structures {
		s := raw.(map[string]any)
		if s["recordId"] == target["recordId"] {
			found = true
			if s["afterRevisionId"] != saved["targetRevision"] || !reflect.DeepEqual(s["facets"], []any{"DELETE"}) {
				t.Fatal(s)
			}
		}
	}
	if !found {
		t.Fatal("missing DELETE impact")
	}
	added := b.call("POST", path+"/records", addBody(detail(b, id), "2026-06-01", map[string]any{"name": "新"}), foundation.NewID(), 201)
	if added["selectedRecordId"] == target["recordId"] {
		t.Fatal("deleted identity revived")
	}
	replay := b.call("POST", endpoint, input, key, 200)
	resolved := b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
	for _, v := range []map[string]any{replay, resolved} {
		if v["targetRevision"] != saved["targetRevision"] || v["selectedRecordId"] != saved["selectedRecordId"] || v["sourceVersion"] != saved["sourceVersion"] || v["currentChanged"] != true {
			t.Fatal("original terminal changed", v)
		}
	}
	b.call("POST", endpoint, deleteBody(detail(b, id), target), foundation.NewID(), 409)
	if f.recordCount("") != 4 || f.scalar("SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.bu_history") != 5 {
		t.Fatal("duplicate deletion created facts")
	}
	// Runtime cannot rewrite or erase the source content holding the tombstone.
	for _, query := range []string{"UPDATE organization.bu_shape SET parts=parts", "DELETE FROM organization.bu_shape"} {
		_, err := f.runtime.Pool.Exec(f.ctx, query)
		var pg *pgconn.PgError
		if !errors.As(err, &pg) || pg.Code != "42501" {
			t.Fatalf("runtime evidence privilege: %v", err)
		}
	}
	// A correctly hashed source whose live member is its deletion revision still
	// fails the typed closure validator; hashes alone do not prove membership.
	originalFact := f.buSource(saved["sourceVersion"].(string))
	f.editBU(saved["sourceVersion"].(string), func(v *bootstrap.BUHistoryFact) {
		for _, deleted := range v.Operation.Revisions {
			if deleted.Deleted {
				v.State.Timeline.Records = append(v.State.Timeline.Records, deleted.Record)
			}
		}
	})
	b.call("GET", path+"?sourceVersion="+saved["sourceVersion"].(string), nil, "", 503)
	f.editBU(saved["sourceVersion"].(string), func(v *bootstrap.BUHistoryFact) { *v = originalFact })

}

func TestDeleteDedicatedCategoryRightsAndCurrentReadRecovery(t *testing.T) {
	f := fresh(t)
	admin := f.login("admin", f.password)
	reader := f.login("reader", f.readPassword)
	for i, kind := range []string{"CHANGE", "DEACTIVATE", "REACTIVATE"} {
		body := createBody("DELETE_RIGHT_"+kind, "2026-01-01")
		body["name"] = "甲"
		body["nameEn"] = "Alpha"
		id := admin.call("POST", "/api/v1/business-units", body, foundation.NewID(), 201)["buId"].(string)
		path := "/api/v1/business-units/" + id
		date := "2026-04-01"
		if kind == "CHANGE" {
			admin.call("POST", path+"/records", addBody(detail(admin, id), date, map[string]any{"name": "乙"}), foundation.NewID(), 201)
		} else {
			lifeSave(admin, id, date, "DEACTIVATE", nil)
			if kind == "REACTIVATE" {
				date = "2026-05-01"
				lifeSave(admin, id, date, "REACTIVATE", nil)
			}
		}
		d := detail(reader, id)
		r := recordOn(t, d, date)
		endpoint := path + "/records/" + r["recordId"].(string) + "/delete"
		in := deleteBody(d, r)
		if d["capabilities"].(map[string]any)["delete"] != false {
			t.Fatal("reader gained deletion")
		}
		reader.call("POST", endpoint, in, foundation.NewID(), 403)
		action := []string{"bu.record.delete", "bu.deactivation.delete", "bu.reactivation.delete"}[i]
		mutateLifeGrants(f, reader.subject, `INSERT INTO access.subject_grant VALUES($1,$2,$3,'bu',$4,$5,ARRAY[]::text[],ARRAY[]::text[])`, f.instance, foundation.NewID(), reader.subject, id, action)
		d = detail(reader, id)
		if d["capabilities"].(map[string]any)["delete"] != true || d["readOnly"] != false {
			t.Fatal("dedicated deletion rejected or ordinary rights required", d)
		}
		in = deleteBody(d, r)
		previewInput := deleteBody(d, r)
		previewInput["action"] = "DELETE"
		previewInput["recordId"] = r["recordId"]
		reader.call("POST", path+"/preview", previewInput, "", 200)
		key := foundation.NewID()
		saved := reader.call("POST", endpoint, in, key, 200)
		var declared []byte
		terminal, err := bootstrap.InspectBUResult(f.ctx, f.admin, f.instance, reader.subject, key)
		must(t, err)
		declared, err = json.Marshal(terminal.Fields)
		must(t, err)
		if terminal.Action != "DELETE" || string(declared) != "[]" {
			t.Fatal("wrong original category or declared fields", terminal.Action, string(declared))
		}
		mutateLifeGrants(f, reader.subject, `DELETE FROM access.subject_grant WHERE instance_id=$1 AND principal_id=$2 AND action=$3`, f.instance, reader.subject, action)
		for _, v := range []map[string]any{reader.call("POST", endpoint, in, key, 200), reader.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)} {
			if v["targetRevision"] != saved["targetRevision"] || v["sourceVersion"] != saved["sourceVersion"] {
				t.Fatal("write revocation blocked original", v)
			}
		}
		reader.call("POST", endpoint, deleteBody(detail(reader, id), r), foundation.NewID(), 403)
		if i == 2 {
			mutateLifeGrants(f, reader.subject, `UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'nameEn') WHERE instance_id=$1 AND principal_id=$2 AND action IN ('bu.read','bu.history.read')`, f.instance, reader.subject)
			cropped := reader.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
			if cropped["impact"] != nil {
				t.Fatal("hidden field impact exposed", cropped)
			}
			original := reader.call("GET", path+"?sourceVersion="+saved["sourceVersion"].(string)+"&selectedRecordId="+r["recordId"].(string), nil, "", 200)
			if original["selectedRecord"].(map[string]any)["nameEn"] != nil || original["selectedRecord"].(map[string]any)["deleted"] != true {
				t.Fatal("tombstone bypassed current field rights", original)
			}
			mutateLifeGrants(f, reader.subject, `DELETE FROM access.subject_grant WHERE instance_id=$1 AND principal_id=$2 AND action='bu.history.read'`, f.instance, reader.subject)
			reader.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 403)
			reader.call("GET", path+"?sourceVersion="+saved["sourceVersion"].(string)+"&selectedRecordId="+r["recordId"].(string), nil, "", 403)
			mutateLifeGrants(f, reader.subject, `DELETE FROM access.subject_grant WHERE instance_id=$1 AND principal_id=$2 AND action='bu.read'`, f.instance, reader.subject)
			reader.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 404)
		}
	}
}

func TestDeleteSameValuePredecessorExactInheritanceAndPureStructure(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	body := createBody("DELETE_SAME", "2026-01-01")
	body["name"] = "甲"
	body["nameEn"] = "Beta"
	id := b.call("POST", "/api/v1/business-units", body, foundation.NewID(), 201)["buId"].(string)
	path := "/api/v1/business-units/" + id
	b.call("POST", path+"/records", addBody(detail(b, id), "2026-03-01", map[string]any{"name": "乙"}), foundation.NewID(), 201)
	lifeSave(b, id, "2026-05-01", "DEACTIVATE", nil)
	lifeSave(b, id, "2026-07-01", "REACTIVATE", nil)
	d := detail(b, id)
	first := recordOn(t, d, "2026-01-01")
	b.call("POST", path+"/records/"+first["recordId"].(string)+"/correct", correctBody(d, first, map[string]any{"name": "乙"}), foundation.NewID(), 200)
	d = detail(b, id)
	oldSource := d["headSourceVersion"].(string)
	target := recordOn(t, d, "2026-03-01")
	stop := recordOn(t, d, "2026-05-01")
	resume := recordOn(t, d, "2026-07-01")
	p := deleteBody(d, target)
	p["action"] = "DELETE"
	p["recordId"] = target["recordId"]
	preview := b.call("POST", path+"/preview", p, "", 200)
	saved := deleteRecord(b, id, target)
	impact := saved["impact"].(map[string]any)
	if len(impact["dataAffectedRanges"].([]any)) != 0 || len(impact["structuralChanges"].([]any)) != 3 || saved["outcome"] != "accepted" {
		t.Fatal("pure structure was lost", saved)
	}
	current := detail(b, id)
	newFirst := recordOn(t, current, "2026-01-01")
	newStop := recordOn(t, current, "2026-05-01")
	if newStop["revision"] == stop["revision"] || !reflect.DeepEqual(recordOn(t, current, "2026-07-01"), resume) {
		t.Fatal("derived or independent snapshot", current)
	}
	var inherited string
	for _, r := range f.buSource(saved["sourceVersion"].(string)).State.Timeline.Records {
		if r.Revision == newStop["revision"] {
			inherited = r.InheritedRevision
		}
	}
	if inherited != newFirst["revision"] {
		t.Fatal("not exact current predecessor")
	}
	pi := preview["impact"].(map[string]any)
	for _, raw := range impact["structuralChanges"].([]any) {
		raw.(map[string]any)["afterRevisionId"] = nil
	}
	if !reflect.DeepEqual(pi, impact) {
		t.Fatal("preview/persist differs", pi, impact)
	}
	old := b.call("GET", path+"?sourceVersion="+oldSource, nil, "", 200)
	if !reflect.DeepEqual(old["timeline"], d["timeline"]) {
		t.Fatal("original source changed")
	}
	if f.revisionCount(true) != 1 {
		t.Fatal("tombstone missing")
	}
}

func TestDeleteStrictInputsDependenciesAndConcurrentSave(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	id := b.call("POST", "/api/v1/business-units", createBody("DELETE_STRICT", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	path := "/api/v1/business-units/" + id
	only := detail(b, id)
	initial := recordOn(t, only, "2026-01-01")
	protected := b.call("POST", path+"/records/"+initial["recordId"].(string)+"/delete", deleteBody(only, initial), foundation.NewID(), 409)
	if protected["code"] != "M02_FIRST_RECORD_PROTECTED" {
		t.Fatal("sole first record lost protection", protected)
	}
	b.call("POST", path+"/records", addBody(detail(b, id), "2026-06-01", map[string]any{"name": "乙"}), foundation.NewID(), 201)
	b.call("POST", path+"/records", addBody(detail(b, id), "2026-10-01", map[string]any{"name": "丙"}), foundation.NewID(), 201)
	d := detail(b, id)
	target := recordOn(t, d, "2026-06-01")
	in := deleteBody(d, target)
	endpoint := path + "/records/" + target["recordId"].(string) + "/delete"
	other := b.call("POST", "/api/v1/business-units", createBody("DELETE_OTHER", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	otherAdded := b.call("POST", "/api/v1/business-units/"+other+"/records", addBody(detail(b, other), "2026-06-01", map[string]any{"name": "另一个BU"}), foundation.NewID(), 201)
	crossBU := map[string]any{"expectedSourceVersion": d["headSourceVersion"], "expectedRecordRevision": otherAdded["targetRevision"]}
	b.call("POST", path+"/records/"+otherAdded["targetRecordId"].(string)+"/delete", crossBU, foundation.NewID(), 409)
	before := businessSnapshot(t, f)
	for _, field := range []string{"changes", "effectiveDate", "newEffectiveDate", "operationType", "operationReason", "name", "nameEn", "note", "state", "recordId"} {
		body := deleteBody(d, target)
		body[field] = ""
		b.call("POST", endpoint, body, foundation.NewID(), 400)
		preview := deleteBody(d, target)
		preview["action"] = "DELETE"
		preview["recordId"] = target["recordId"]
		preview[field] = ""
		b.call("POST", path+"/preview", preview, "", 400)
	}
	for _, raw := range []string{
		fmt.Sprintf(`{"expectedSourceVersion":%q}`, d["headSourceVersion"]),
		fmt.Sprintf(`{"expectedSourceVersion":%q,"expectedRecordRevision":null}`, d["headSourceVersion"]),
		fmt.Sprintf(`{"expectedSourceVersion":%q,"expectedRecordRevision":%q,"expectedRecordRevision":%q}`, d["headSourceVersion"], target["revision"], target["revision"]),
	} {
		req, e := http.NewRequest("POST", f.server.URL+endpoint, strings.NewReader(raw))
		must(t, e)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", f.server.URL)
		req.Header.Set("X-CSRF-Token", b.csrf)
		req.Header.Set("Idempotency-Key", foundation.NewID())
		response, e := b.client.Do(req)
		must(t, e)
		response.Body.Close()
		if response.StatusCode != 400 {
			t.Fatal("strict JSON accepted", response.StatusCode)
		}
	}
	first := recordOn(t, d, "2026-01-01")
	refused := b.call("POST", path+"/records/"+first["recordId"].(string)+"/delete", deleteBody(d, first), foundation.NewID(), 409)
	if refused["code"] != "M02_FIRST_RECORD_PROTECTED" {
		t.Fatal(refused)
	}
	for _, field := range []string{"expectedSourceVersion", "expectedRecordRevision"} {
		body := deleteBody(d, target)
		body[field] = foundation.NewID()
		b.call("POST", endpoint, body, foundation.NewID(), 409)
	}
	b.call("POST", path+"/records/"+foundation.NewID()+"/delete", in, foundation.NewID(), 409)
	for table, value := range businessSnapshot(t, f) {
		if table != "terminals" && before[table] != value {
			t.Fatal("refusal left business writes", table)
		}
	}
	type answer struct {
		key    string
		status int
		out    map[string]any
		err    error
	}
	start := make(chan struct{})
	answers := make(chan answer, 2)
	for i := 0; i < 2; i++ {
		key := foundation.NewID()
		go func() {
			<-start
			status, out, e := b.response("POST", endpoint, in, key)
			answers <- answer{key, status, out, e}
		}()
	}
	close(start)
	accepted := 0
	for i := 0; i < 2; i++ {
		a := <-answers
		must(t, a.err)
		switch a.status {
		case 200:
			accepted++
			b.call("POST", endpoint, in, a.key, 200)
		case 409:
			if a.out["code"] != "M02_SOURCE_CONFLICT" {
				t.Fatal(a.out)
			}
		default:
			t.Fatal(a.status, a.out)
		}
	}
	if accepted != 1 || f.revisionCount(true) != 1 || f.scalar("SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.bu_history") != 6 {
		t.Fatal("concurrent deletion not unique")
	}
	// A legitimate S/R chain is guarded after removing a single member.
	cycle := b.call("POST", "/api/v1/business-units", createBody("DELETE_DEPENDENCY", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	cp := "/api/v1/business-units/" + cycle
	lifeSave(b, cycle, "2026-04-01", "DEACTIVATE", nil)
	lifeSave(b, cycle, "2026-05-01", "REACTIVATE", nil)
	b.call("POST", cp+"/records", addBody(detail(b, cycle), "2026-06-01", map[string]any{"name": "后继"}), foundation.NewID(), 201)
	lifeSave(b, cycle, "2026-08-01", "DEACTIVATE", nil)
	for _, date := range []string{"2026-04-01", "2026-05-01"} {
		cd := detail(b, cycle)
		r := recordOn(t, cd, date)
		request := deleteBody(cd, r)
		key := foundation.NewID()
		out := b.call("POST", cp+"/records/"+r["recordId"].(string)+"/delete", request, key, 409)
		if out["code"] != "M02_TIMELINE_INVALID" {
			t.Fatal(out)
		}
		if out["field"] != "" || len(out["failureDetails"].(map[string]any)["issues"].([]any)[0].(map[string]any)["locations"].([]any)) != 0 {
			t.Fatalf("deletion cannot focus an editable business field: %v", out)
		}
		request["action"] = "DELETE"
		request["recordId"] = r["recordId"]
		b.call("POST", cp+"/preview", request, "", 409)
		recovered := b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
		if recovered["outcome"] != "rejected" || !reflect.DeepEqual(recovered["failureDetails"], out["failureDetails"]) {
			t.Fatal("original rejection replaced", recovered)
		}
	}
	// Ordinary delete permission never substitutes for a lifecycle-specific action.
	mutateLifeGrants(f, b.subject, `DELETE FROM access.subject_grant WHERE instance_id=$1 AND principal_id=$2 AND action IN ('bu.deactivation.delete','bu.reactivation.delete')`, f.instance, b.subject)
	cd := detail(b, cycle)
	r := recordOn(t, cd, "2026-08-01")
	v := b.call("GET", cp+"?selectedRecordId="+r["recordId"].(string), nil, "", 200)
	if v["capabilities"].(map[string]any)["delete"] != false {
		t.Fatal("ordinary right exposed lifecycle delete")
	}
	b.call("POST", cp+"/records/"+r["recordId"].(string)+"/delete", deleteBody(cd, r), foundation.NewID(), 403)
}

func containsBURevision(v bootstrap.BUHistoryFact, revision string) bool {
	for _, r := range v.State.Timeline.Records {
		if r.Revision == revision {
			return true
		}
	}
	return false
}

//go:build integration

package integration_test

import (
	"chnhcm.local/chnhcm/internal/bootstrap"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"chnhcm.local/chnhcm/internal/foundation"
)

func redateBody(d, r map[string]any, date string) map[string]any {
	return map[string]any{"expectedSourceVersion": d["headSourceVersion"], "expectedRecordRevision": r["revision"], "newEffectiveDate": date}
}
func redatePath(id string, r map[string]any) string {
	return "/api/v1/business-units/" + id + "/records/" + r["recordId"].(string) + "/redate"
}
func TestRedateJourneyOriginalSourcesAndExactResults(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	body := createBody("REDATE_JOURNEY", "2026-01-01")
	body["name"] = "甲"
	body["nameEn"] = "Alpha"
	id := b.call("POST", "/api/v1/business-units", body, foundation.NewID(), 201)["buId"].(string)
	path := "/api/v1/business-units/" + id
	b.call("POST", path+"/records", addBody(detail(b, id), "2026-06-01", map[string]any{"name": "乙", "nameEn": "Beta"}), foundation.NewID(), 201)
	b.call("POST", path+"/records", addBody(detail(b, id), "2026-10-01", map[string]any{"name": "丙", "nameEn": "Gamma"}), foundation.NewID(), 201)
	d := detail(b, id)
	r := recordOn(t, d, "2026-06-01")
	old := d["headSourceVersion"].(string)
	input := redateBody(d, r, "2026-09-01")
	key := foundation.NewID()
	p := redateBody(d, r, "2026-09-01")
	p["action"] = "REDATE"
	p["recordId"] = r["recordId"]
	preview := b.call("POST", path+"/preview", p, "", 200)
	if preview["beforeRecord"] != nil || preview["afterRecord"].(map[string]any)["effectiveDate"] != "2026-09-01" || preview["currentStatus"] != "active" {
		t.Fatal("wrong before snapshot or target predecessor", preview)
	}
	saved := b.call("POST", redatePath(id, r), input, key, 200)
	if saved["targetRecordId"] != r["recordId"] || saved["selectedRecordId"] != r["recordId"] || saved["targetRevision"] == r["revision"] || saved["effectiveDate"] != "2026-09-01" || saved["previousEffectiveDate"] != "2026-06-01" {
		t.Fatal(saved)
	}
	for _, tc := range []struct{ date, name, en string }{{"2026-06-15", "甲", "Alpha"}, {"2026-10-15", "丙", "Gamma"}, {"2026-09-01", "乙", "Beta"}} {
		v := b.getOnDay(tc.date, path)["viewFact"].(map[string]any)
		if v["name"] != tc.name || v["nameEn"] != tc.en {
			t.Fatal(tc, v)
		}
	}
	if b.getOnDay("2026-06-15", path+"?sourceVersion="+old)["viewFact"].(map[string]any)["name"] != "乙" {
		t.Fatal("old source rewritten")
	}
	current := detail(b, id)
	moved := recordOn(t, current, "2026-09-01")
	if moved["operationType"] != r["operationType"] || moved["operationReason"] != r["operationReason"] || moved["name"] != "乙" || moved["nameEn"] != "Beta" {
		t.Fatal(moved)
	}
	// Compare preview and persisted impact without using either as the oracle for
	// expected snapshots. Only assigned revision UUIDs may differ.
	pi := preview["impact"].(map[string]any)
	si := saved["impact"].(map[string]any)
	for _, v := range []map[string]any{pi, si} {
		for _, raw := range v["structuralChanges"].([]any) {
			delete(raw.(map[string]any), "afterRevisionId")
		}
	}
	if !reflect.DeepEqual(pi, si) {
		t.Fatal("preview/save impact mismatch", pi, si)
	}
	logs := b.call("GET", path+"/changes", nil, "", 200)["items"].([]any)
	entry := logs[0].(map[string]any)
	if entry["action"] != "REDATE" || directChange(t, entry)["beforeContext"].(map[string]any)["effectiveDate"] != "2026-06-01" || directChange(t, entry)["afterContext"].(map[string]any)["effectiveDate"] != "2026-09-01" {
		t.Fatal(entry)
	}
	b.call("POST", redatePath(id, moved), redateBody(current, moved, "2026-09-15"), foundation.NewID(), 200)
	// Crossing a neighbour is outside ordinary correction; no generic fallback.
	crossBase := detail(b, id)
	cross := recordOn(t, crossBase, "2026-09-15")
	b.call("POST", redatePath(id, cross), redateBody(crossBase, cross, "2026-12-01"), foundation.NewID(), 400)
	if detail(b, id)["headSourceVersion"] != crossBase["headSourceVersion"] {
		t.Fatal("cross-position move saved")
	}

	for _, v := range []map[string]any{b.call("POST", redatePath(id, r), input, key, 200), b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)} {
		if v["targetRevision"] != saved["targetRevision"] || v["sourceVersion"] != saved["sourceVersion"] || v["effectiveDate"] != "2026-09-01" || v["previousEffectiveDate"] != "2026-06-01" {
			t.Fatal("restored current instead of original", v)
		}
	}
	if f.recordCount(id) != 3 {
		t.Fatal("redate allocated new identities")
	}
}

func TestRedateCategoryRightsMatrixNoChangeAndReadRecovery(t *testing.T) {
	f := fresh(t)
	admin := f.login("admin", f.password)
	reader := f.login("reader", f.readPassword)
	actions := []string{"bu.first.redate", "bu.record.redate", "bu.deactivation.redate", "bu.reactivation.redate"}
	for i, kind := range []string{"CREATE", "CHANGE", "DEACTIVATE", "REACTIVATE"} {
		id := admin.call("POST", "/api/v1/business-units", createBody("REDATE_RIGHT_"+kind, "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
		date := "2026-01-01"
		if kind == "CHANGE" {
			date = "2026-04-01"
			admin.call("POST", "/api/v1/business-units/"+id+"/records", addBody(detail(admin, id), date, map[string]any{"name": "乙"}), foundation.NewID(), 201)
		}
		if kind == "DEACTIVATE" || kind == "REACTIVATE" {
			date = "2026-04-01"
			lifeSave(admin, id, date, "DEACTIVATE", nil)
			if kind == "REACTIVATE" {
				date = "2026-05-01"
				lifeSave(admin, id, date, "REACTIVATE", map[string]any{"name": "恢复", "nameEn": "Restored"})
			}
		}
		d := detail(reader, id)
		r := recordOn(t, d, date)
		before := businessSnapshot(t, f)
		for j, action := range actions {
			mutateLifeGrants(f, reader.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action LIKE '%.redate'`, reader.subject)
			mutateLifeGrants(f, reader.subject, `INSERT INTO access.subject_grant VALUES($1,$2,$3,'bu',$4,$5,ARRAY[]::text[],ARRAY['effectiveDate'])`, f.instance, foundation.NewID(), reader.subject, id, action)
			status := 403
			if i == j {
				status = 200
			}
			v := reader.call("POST", redatePath(id, r), redateBody(d, r, date), foundation.NewID(), status)
			if status == 200 && v["outcome"] != "no_change" {
				t.Fatal("same date must be no_change", v)
			}
		}
		for table, value := range businessSnapshot(t, f) {
			if table != "terminals" && before[table] != value {
				t.Fatal("no_change or denial wrote business fact", table)
			}
		}
		mutateLifeGrants(f, reader.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action LIKE '%.redate'`, reader.subject)
		mutateLifeGrants(f, reader.subject, `INSERT INTO access.subject_grant VALUES($1,$2,$3,'bu',$4,$5,ARRAY[]::text[],ARRAY['name'])`, f.instance, foundation.NewID(), reader.subject, id, actions[i])
		reader.call("POST", redatePath(id, r), redateBody(d, r, date), foundation.NewID(), 403)
		mutateLifeGrants(f, reader.subject, `UPDATE access.subject_grant SET write_fields=ARRAY['effectiveDate'] WHERE principal_id=$1 AND action=$2`, reader.subject, actions[i])
		input := redateBody(d, r, "2026-06-01")
		key := foundation.NewID()
		saved := reader.call("POST", redatePath(id, r), input, key, 200)
		if saved["previousEffectiveDate"] != date || saved["effectiveDate"] != "2026-06-01" {
			t.Fatal("missing original/new dates", saved)
		}
		var fields []byte
		terminal, err := bootstrap.InspectBUResult(f.ctx, f.admin, f.instance, reader.subject, key)
		must(t, err)
		fields, err = json.Marshal(terminal.Fields)
		must(t, err)
		if terminal.Action != "REDATE" || string(fields) != `["effectiveDate"]` {
			t.Fatal("wrong authority/declared field", terminal.Action, string(fields))
		}
		mutateLifeGrants(f, reader.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action LIKE '%.redate'`, reader.subject)
		for _, v := range []map[string]any{reader.call("POST", redatePath(id, r), input, key, 200), reader.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)} {
			if v["targetRevision"] != saved["targetRevision"] || v["previousEffectiveDate"] != date {
				t.Fatal("withdrawn write right blocked original result", v)
			}
		}
		if i == 3 {
			mutateLifeGrants(f, reader.subject, `UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'effectiveDate') WHERE principal_id=$1 AND action IN ('bu.read','bu.history.read')`, reader.subject)
			cropped := reader.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
			if cropped["effectiveDate"] != nil || cropped["previousEffectiveDate"] != nil || cropped["impact"] != nil {
				t.Fatal(cropped)
			}
			retryCropped := reader.call("POST", redatePath(id, r), input, key, 200)
			if retryCropped["previousEffectiveDate"] != nil || retryCropped["effectiveDate"] != nil {
				t.Fatal("retry exposed revoked dates", retryCropped)
			}
			mutateLifeGrants(f, reader.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='bu.history.read'`, reader.subject)
			reader.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 403)
		}
	}
}

func TestRedateStrictInputsConcurrencyAndInheritance(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	id := b.call("POST", "/api/v1/business-units", createBody("REDATE_STRICT", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	path := "/api/v1/business-units/" + id
	b.call("POST", path+"/records", addBody(detail(b, id), "2026-03-01", map[string]any{"name": "乙"}), foundation.NewID(), 201)
	lifeSave(b, id, "2026-05-01", "DEACTIVATE", nil)
	d := detail(b, id)
	r := recordOn(t, d, "2026-03-01")
	input := redateBody(d, r, "2026-04-01")
	endpoint := redatePath(id, r)
	other := b.call("POST", "/api/v1/business-units", createBody("REDATE_OTHER", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	b.call("POST", "/api/v1/business-units/"+other+"/records", addBody(detail(b, other), "2026-03-01", map[string]any{"name": "其他记录"}), foundation.NewID(), 201)
	foreign := recordOn(t, detail(b, other), "2026-03-01")
	b.call("POST", redatePath(id, foreign), redateBody(d, foreign, "2026-04-01"), foundation.NewID(), 409)
	deleteRecord(b, other, foreign)
	b.call("POST", redatePath(other, foreign), redateBody(detail(b, other), foreign, "2026-04-01"), foundation.NewID(), 409)
	for _, field := range []string{"oldEffectiveDate", "effectiveDate", "changes", "operationType", "operationReason", "name", "state", "note", "recordId"} {
		in := redateBody(d, r, "2026-04-01")
		in[field] = ""
		b.call("POST", endpoint, in, foundation.NewID(), 400)
		in["action"] = "REDATE"
		in["recordId"] = r["recordId"]
		if field != "recordId" {
			b.call("POST", path+"/preview", in, "", 400)
		}
	}
	raw, _ := json.Marshal(input)
	for _, bad := range []string{`{}`, strings.Replace(string(raw), `"newEffectiveDate":"2026-04-01"`, `"newEffectiveDate":null`, 1), fmt.Sprintf(`{"expectedSourceVersion":%q,"expectedRecordRevision":%q,"newEffectiveDate":"2026-04-01","newEffectiveDate":"2026-04-01"}`, d["headSourceVersion"], r["revision"])} {
		req, e := http.NewRequest("POST", f.server.URL+endpoint, strings.NewReader(bad))
		must(t, e)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", f.server.URL)
		req.Header.Set("X-CSRF-Token", b.csrf)
		req.Header.Set("Idempotency-Key", foundation.NewID())
		res, e := b.client.Do(req)
		must(t, e)
		res.Body.Close()
		if res.StatusCode != 400 {
			t.Fatal("strict JSON", res.StatusCode)
		}
	}
	for _, date := range []string{"2025-12-31", "2026-01-01", "2026-05-01", "2026-06-01"} {
		key := foundation.NewID()
		status := 400
		if date == "2026-01-01" || date == "2026-05-01" {
			status = 409
		}
		v := b.call("POST", endpoint, redateBody(d, r, date), key, status)
		if v["field"] != "newEffectiveDate" {
			t.Fatal("unsafe error location", v)
		}
		recovered := b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
		if recovered["outcome"] != "rejected" {
			t.Fatal("redate diagnostic cannot be decoded", recovered)
		}
	}
	for _, field := range []string{"expectedSourceVersion", "expectedRecordRevision"} {
		in := redateBody(d, r, r["effectiveDate"].(string))
		in[field] = foundation.NewID()
		b.call("POST", endpoint, in, foundation.NewID(), 409)
	}
	type answer struct {
		status int
		err    error
	}
	answers := make(chan answer, 2)
	for range 2 {
		go func() {
			s, _, err := b.response("POST", endpoint, input, foundation.NewID())
			answers <- answer{s, err}
		}()
	}
	counts := map[int]int{}
	for range 2 {
		a := <-answers
		must(t, a.err)
		counts[a.status]++
	}
	if counts[200] != 1 || counts[409] != 1 {
		t.Fatal(counts)
	}
	current := detail(b, id)
	moved := recordOn(t, current, "2026-04-01")
	if moved["recordId"] != r["recordId"] {
		t.Fatal("new identity")
	}
	staleRevision := b.call("POST", endpoint, redateBody(current, r, "2026-04-01"), foundation.NewID(), 409)
	if staleRevision["code"] != "M02_REVISION_CONFLICT" {
		t.Fatal("current source accepted an old target revision", staleRevision)
	}
	f.assertBUInheritance()

	// Equal predecessor values still produce structural revisions, including
	// the exact newly allocated revision inherited by the later stop.
	first := recordOn(t, current, "2026-01-01")
	b.call("POST", path+"/records/"+first["recordId"].(string)+"/correct", correctBody(current, first, map[string]any{"name": "乙"}), foundation.NewID(), 200)
	current = detail(b, id)
	moved = recordOn(t, current, "2026-04-01")
	structural := b.call("POST", redatePath(id, moved), redateBody(current, moved, "2026-04-15"), foundation.NewID(), 200)
	if len(structural["impact"].(map[string]any)["dataAffectedRanges"].([]any)) != 0 {
		t.Fatal("equal values are not a data impact", structural)
	}
	stops := 0
	for _, r := range f.buSource(structural["sourceVersion"].(string)).State.Timeline.Records {
		if r.Type == "DEACTIVATE" && r.InheritedRevision == structural["targetRevision"] {
			stops++
		}
	}
	if stops != 1 {
		t.Fatal("stop did not inherit exact new target revision")
	}
}

func TestRedateMissingClassificationFailsClosed(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	for _, kind := range []string{"CREATE", "CHANGE", "DEACTIVATE", "REACTIVATE"} {
		id := b.call("POST", "/api/v1/business-units", createBody("REDATE_MISSING_"+kind, "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
		date := "2026-01-01"
		if kind == "CHANGE" {
			date = "2026-06-01"
			b.call("POST", "/api/v1/business-units/"+id+"/records", addBody(detail(b, id), date, map[string]any{"name": "乙"}), foundation.NewID(), 201)
		}
		if kind == "DEACTIVATE" || kind == "REACTIVATE" {
			date = "2026-04-01"
			lifeSave(b, id, date, "DEACTIVATE", nil)
			if kind == "REACTIVATE" {
				date = "2026-06-01"
				lifeSave(b, id, date, kind, nil)
			}
		}
		d := detail(b, id)
		r := recordOn(t, d, date)
		for field, key := range map[string]string{"operation_reason": "operationReason", "type_label": "typeLabel", "reason_label": "reasonLabel"} {
			before := businessSnapshot(t, f)
			// Deliberate corruption in this disposable owner fixture only. The
			// application role cannot rewrite these revisions.
			f.editBURevision(r["revision"].(string), func(v *bootstrap.IntegrationBURevision) {
				switch field {
				case "operation_reason":
					v.Reason = ""
				case "type_label":
					v.TypeLabel = ""
				case "reason_label":
					v.ReasonLabel = ""
				}
			})
			input := redateBody(d, r, "2026-07-01")
			submission := foundation.NewID()
			out := b.call("POST", redatePath(id, r), input, submission, 503)
			if out["code"] != "M02_CLASSIFICATION_UNAVAILABLE" || out["outcome"] != nil {
				t.Fatal("incomplete evidence escaped", out)
			}
			input["action"] = "REDATE"
			input["recordId"] = r["recordId"]
			p := b.call("POST", "/api/v1/business-units/"+id+"/preview", input, "", 503)
			if p["code"] != "M02_CLASSIFICATION_UNAVAILABLE" {
				t.Fatal(p)
			}
			if f.scalar("SELECT (SELECT count(*) FROM organization.bu_history h CROSS JOIN LATERAL unnest(h.submission_refs) r WHERE h.ids[r]=$1::uuid)+(SELECT count(*) FROM organization.bu_terminal WHERE submission_id=$1)", submission) != 0 {
				t.Fatal("technical failure invented a terminal")
			}
			f.editBURevision(r["revision"].(string), func(v *bootstrap.IntegrationBURevision) {
				switch field {
				case "operation_reason":
					v.Reason = r[key].(string)
				case "type_label":
					v.TypeLabel = r[key].(string)
				case "reason_label":
					v.ReasonLabel = r[key].(string)
				}
			})
			if !reflect.DeepEqual(before, businessSnapshot(t, f)) {
				t.Fatal("failure changed business history")
			}
		}
	}
}

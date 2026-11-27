//go:build integration

package bootstrap_test

import (
	"reflect"
	"testing"

	legalapp "chnhcm.local/chnhcm/internal/contexts/organization/application/legalentity"
	locapp "chnhcm.local/chnhcm/internal/contexts/organization/application/location"
	"chnhcm.local/chnhcm/internal/contexts/organization/domain/dated"
	"chnhcm.local/chnhcm/internal/foundation"
)

// Read committed state through the fixture's independent PostgreSQL connection.
// A resolve on an unknown submission is itself a write and must come afterwards.
func checkSubmission140(t *testing.T, f *fixture, table, code, sid string, head, writes int, terminal string) {
	t.Helper()
	object := "organization." + table
	n := f.scalar("SELECT coalesce(sum(head_no),0) FROM "+object+" WHERE code=$1", code)
	w := f.scalar("SELECT count(*) FROM "+object+"_history h CROSS JOIN unnest(h.submission_refs) r WHERE h.ids[r]=$1", sid)
	var outcomes string
	must(t, f.admin.QueryRow(f.ctx, "SELECT coalesce(string_agg(outcome,','),'') FROM "+object+"_terminal WHERE submission_id=$1", sid).Scan(&outcomes))
	if n != head || w != writes || outcomes != terminal {
		t.Fatalf("%s %s: head=%d writes=%d terminals=%q, want %d/%d/%q", table, code, n, w, outcomes, head, writes, terminal)
	}
}

func TestHistory140DeleteAndRecover(t *testing.T) {
	h := legalSetup206(t)
	loc := location204{t, h.f, h.b, h.countries}
	reader := h.f.login("reader", h.f.readPassword)
	sharedID := foundation.NewID()
	receipts := map[string]map[string]any{}
	cases := []struct {
		action, table, path, objectKey, code string
		newObject                            func() map[string]any
		add                                  func(map[string]any, string, string, map[string]any, int) map[string]any
	}{
		{"location", "location", "/api/v1/locations", "locationId", "T04-LOC", func() map[string]any { return loc.create("T04-LOC", "MO", "", "") }, loc.add},
		{"legalentity", "legal_entity", legal206Path, "legalEntityId", "T04-LE", func() map[string]any { return h.create("T04-LE", "MO", "branch") }, h.add},
	}
	for _, tc := range cases {
		t.Run(tc.action, func(t *testing.T) {
			root := h.f.t
			h.f.t = t
			defer func() { h.f.t = root }()
			v := tc.newObject()
			id := v["id"].(string)
			base := tc.path + "/" + id
			read := func(query string) map[string]any { return h.b.call("GET", base+query, nil, "", 200) }
			for _, date := range []string{"2000-01-01", "2010-01-01", "2020-01-01"} {
				tc.add(v, "ADD", date, map[string]any{"name": "独立资料 " + date}, 201)
				v = read("")
			}
			tc.add(v, "DEACTIVATE", "2021-01-01", nil, 201)
			tc.add(read(""), "REACTIVATE", "2022-01-01", nil, 201)
			v = read("")
			row := func(v map[string]any, i int) map[string]any { return v["timeline"].([]any)[i].(map[string]any) }
			fourth, stop, resume := row(v, 3), row(v, 4), row(v, 5)
			records := base + "/records/"
			h.b.call("POST", records+fourth["recordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": v["headSourceVersion"], "expectedRecordRevision": fourth["revision"], "changes": map[string]any{"name": "更正后的前序"}}, foundation.NewID(), 200)
			v = read("")
			if row(v, 3)["revision"] == fourth["revision"] || row(v, 4)["revision"] == stop["revision"] || row(v, 4)["name"] != "更正后的前序" || !reflect.DeepEqual(resume, row(v, 5)) {
				t.Fatal("correction must update the predecessor and its dependent stop only")
			}
			first, second, removed := row(v, 0), row(v, 1), row(v, 2)
			target := removed["recordId"].(string)
			selectTarget := "?selectedRecordId=" + target
			before := read(selectTarget + "&sourceVersion=" + v["headSourceVersion"].(string))["selectedRecord"]
			del := records + target + "/delete"
			body := map[string]any{"expectedSourceVersion": v["headSourceVersion"], "expectedRecordRevision": removed["revision"]}
			denied := foundation.NewID()
			reader.call("POST", del, body, denied, 403)
			checkSubmission140(t, h.f, tc.table, tc.code, denied, 7, 0, "")
			noChange := foundation.NewID()
			same := map[string]any{"expectedSourceVersion": v["headSourceVersion"], "expectedRecordRevision": removed["revision"], "changes": map[string]any{"name": removed["name"]}}
			if h.b.call("POST", records+target+"/correct", same, noChange, 200)["outcome"] != "no_change" {
				t.Fatal("expected no change")
			}
			checkSubmission140(t, h.f, tc.table, tc.code, noChange, 7, 0, "no_change")
			conflict := foundation.NewID()
			h.b.call("POST", records+first["recordId"].(string)+"/delete", map[string]any{"expectedSourceVersion": v["headSourceVersion"], "expectedRecordRevision": first["revision"]}, conflict, 409)
			checkSubmission140(t, h.f, tc.table, tc.code, conflict, 7, 0, "rejected")

			got := h.b.call("POST", del, body, sharedID, 200)
			checkSubmission140(t, h.f, tc.table, tc.code, sharedID, 8, 1, "")
			resolve := tc.path + "/submissions/" + sharedID + "/resolve"
			replay := h.b.call("POST", del, body, sharedID, 200)
			if replay["replayed"] != true {
				t.Fatal("expected original submission replay")
			}
			for _, receipt := range []map[string]any{got, replay, h.b.call("POST", resolve, nil, "", 200)} {
				if receipt[tc.objectKey] != id || receipt["submissionId"] != sharedID || receipt["sourceVersion"] != got["sourceVersion"] || receipt["targetRecordId"] != target || receipt["targetRevision"] != removed["revision"] || receipt["effectiveDate"] != "2010-01-01" || receipt["selectedRecordId"] != second["recordId"] {
					t.Fatal("incorrect delete receipt", receipt)
				}
			}
			after := read("?selectedRecordId=" + second["recordId"].(string))
			if len(after["timeline"].([]any)) != 5 || legalRecord206(after)["recordId"] != second["recordId"] || row(after, 1)["endExclusive"] != "2020-01-01" || row(after, 1)["revision"] == second["revision"] {
				t.Fatal("incorrect surviving interval", row(after, 1))
			}
			for _, i := range []int{0, 3, 4, 5} {
				j := i
				if i > 2 {
					j--
				}
				if !reflect.DeepEqual(row(v, i), row(after, j)) {
					t.Fatalf("unaffected row %d changed", i)
				}
			}
			old := read(selectTarget + "&sourceVersion=" + v["headSourceVersion"].(string))["selectedRecord"]
			frozen := read(selectTarget + "&sourceVersion=" + got["sourceVersion"].(string))["selectedRecord"].(map[string]any)
			if !reflect.DeepEqual(old, before) || frozen["deleted"] != true || frozen["recordId"] != target || frozen["revision"] != removed["revision"] || frozen["name"] != removed["name"] || frozen["effectiveDate"] != removed["effectiveDate"] {
				t.Fatal("deleted original or previous source changed")
			}
			checkHistory140(t, h.f, tc.table, id, got["sourceVersion"].(string), sharedID)

			body["expectedRecordRevision"] = second["revision"]
			if h.b.call("POST", del, body, sharedID, 409)["code"] != "M02_REQUEST_MISMATCH" {
				t.Fatal("original number accepted a different payload")
			}
			checkSubmission140(t, h.f, tc.table, tc.code, sharedID, 8, 1, "")
			// Remove the newly selected second record as a separate submission.
			// Resolving the first deletion must still identify the original selection and source.
			currentSecond := row(after, 1)
			later := h.b.call("POST", records+second["recordId"].(string)+"/delete", map[string]any{
				"expectedSourceVersion": got["sourceVersion"], "expectedRecordRevision": currentSecond["revision"],
			}, foundation.NewID(), 200)
			if later["selectedRecordId"] != first["recordId"] || later["sourceVersion"] == got["sourceVersion"] {
				t.Fatal("invalid subsequent deletion", later)
			}
			receipts[tc.action] = got
		})
	}
	// Both objects have now committed the same number. Resolve both again under changing grants.
	if t.Failed() {
		return
	}
	for _, tc := range cases {
		got := receipts[tc.action]
		resolve := tc.path + "/submissions/" + sharedID + "/resolve"
		if original := h.b.call("POST", resolve, nil, "", 200); original[tc.objectKey] != got[tc.objectKey] || original["sourceVersion"] != got["sourceVersion"] || original["selectedRecordId"] != got["selectedRecordId"] || original["currentChanged"] != true {
			t.Fatal("same number resolved to another object")
		}
		h.policy("UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'effectiveDate') WHERE principal_id=$1 AND action=$2", h.b.subject, tc.action+".history.read")
		if trimmed := h.b.call("POST", resolve, nil, "", 200); trimmed["effectiveDate"] != nil || trimmed["targetRevision"] != got["targetRevision"] {
			t.Fatal("receipt field permissions", trimmed)
		}
		h.policy("DELETE FROM access.subject_grant WHERE principal_id=$1 AND action=$2", h.b.subject, tc.action+".history.read")
		h.b.call("POST", resolve, nil, "", 403)
	}
}

func checkInheritance140[T any](t *testing.T, rows []dated.Record[T]) {
	t.Helper()
	predecessor, stop := rows[2], rows[3]
	if stop.Type != "DEACTIVATE" || stop.InheritedRecord != predecessor.ID || stop.InheritedRevision != predecessor.Revision || !reflect.DeepEqual(stop.Data, predecessor.Data) {
		t.Fatal("stored stop does not inherit its exact predecessor")
	}
}

func checkHistory140(t *testing.T, f *fixture, table, id, head, submission string) {
	t.Helper()
	c := geoCredentials203(t, f, "admin", f.password)
	events, sources, submissions := map[string]bool{}, map[string]bool{}, map[string]bool{}
	want, number := head, int64(8)
	// The history port returns newest first. Every source has one distinct object event.
	operation := func(n int64, source, previous, event, sid string) {
		t.Helper()
		if n != number || source != want || !foundation.ValidID(event) || events[event] || sources[source] || submissions[sid] || (n == 8 && sid != submission) {
			t.Fatalf("incorrect source chain or duplicate event at %d", n)
		}
		events[event], sources[source], submissions[sid] = true, true, true
		number--
		want = previous
	}
	if table == "location" {
		must(t, f.runtime.Location.(locapp.Service).Store.WithinLocation(f.ctx, c, func(u locapp.Unit) error {
			state, err := u.Load(f.ctx, id, head)
			if err != nil {
				return err
			}
			checkInheritance140(t, state.Timeline.Records)
			ops, err := u.History(f.ctx, id, head)
			for _, op := range ops {
				operation(op.Number, op.ID, op.Previous, op.EventID, op.Submission)
			}
			return err
		}))
	} else {
		must(t, f.runtime.LegalEntity.(legalapp.Service).Store.WithinLegalEntity(f.ctx, c, func(u legalapp.Unit) error {
			state, err := u.Load(f.ctx, id, head)
			if err != nil {
				return err
			}
			checkInheritance140(t, state.Timeline.Records)
			ops, err := u.History(f.ctx, id, head)
			for _, op := range ops {
				operation(op.Number, op.ID, op.Previous, op.EventID, op.Submission)
			}
			return err
		}))
	}
	if number != 0 || want != "" {
		t.Fatal("incomplete source chain")
	}
}

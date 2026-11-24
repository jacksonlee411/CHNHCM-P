//go:build integration

package bootstrap_test

import (
	app "chnhcm.local/chnhcm/internal/contexts/staffing/application"
	store "chnhcm.local/chnhcm/internal/contexts/staffing/infra/postgres"
	"chnhcm.local/chnhcm/internal/foundation"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestStaffing310MemberPermissionsAndCorruption(t *testing.T) {
	h := staffingSetup307(t)
	f, b := h.f, h.b
	family := h.create("jobfamily", "PF310", "")
	a := h.create("grade", "PA310", "")
	ids := []string{a["id"].(string)}
	for _, kind := range []string{"job", "gradeladder"} {
		t.Run(kind, func(t *testing.T) {
			v := create310(h, kind, "P310", ids, family["id"].(string))
			id := v["id"].(string)
			source := v["headSourceVersion"].(string)
			// Scalar maintenance inherits exact adoption without acquiring history rights.
			f.exec("DELETE FROM access.subject_grant WHERE action='grade.history.read' AND principal_id=$1", b.subject)
			h.correct(kind, v, map[string]any{"name": "继承旧采用"}, nil, 200)
			v = h.refresh(kind, v)
			exact := b.call("GET", base307(kind)+"/"+id+"?sourceVersion="+source, nil, "", 200)
			if rec307(exact)["gradeMembers"] != nil || rec307(exact)["gradeIds"] != nil || rec307(exact)["gradesRestricted"] != true {
				t.Fatal("history source leaked")
			}
			h.correct(kind, v, map[string]any{"gradeIds": []string{}}, nil, kindStatus310(kind))
			f.exec("INSERT INTO access.subject_grant SELECT instance_id,gen_random_uuid(),principal_id,scope_kind,scope_id,'grade.history.read',read_fields,write_fields FROM access.subject_grant WHERE action='grade.read' AND principal_id=$1", b.subject)
			// Missing collection write right cannot be bypassed by implicit deletion.
			f.exec("UPDATE access.subject_grant SET write_fields=array_remove(write_fields,'gradeIds') WHERE action=$2 AND principal_id=$1", b.subject, kind+".record.correct")
			h.correct(kind, v, map[string]any{"gradeIds": ids}, nil, 403)
			h.correct(kind, v, map[string]any{"description": "资料"}, nil, map[bool]int{true: 400, false: 200}[kind == "job"])
			f.exec("UPDATE access.subject_grant SET write_fields=array_append(write_fields,'gradeIds') WHERE action=$2 AND principal_id=$1", b.subject, kind+".record.correct")
			v = h.refresh(kind, v)
			f.exec("UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'gradeIds') WHERE action LIKE $2 AND principal_id=$1", b.subject, kind+".%")
			hidden := h.refresh(kind, v)
			encoded, _ := json.Marshal(hidden)
			if strings.Contains(string(encoded), ids[0]) || strings.Contains(string(encoded), "gradeMembers") || strings.Contains(string(encoded), "gradesRestricted") {
				t.Fatal("collection field leaked")
			}
			h.correct(kind, hidden, map[string]any{"gradeIds": ids}, nil, 403)
			f.exec("UPDATE access.subject_grant SET read_fields=array_append(read_fields,'gradeIds') WHERE action LIKE $2 AND principal_id=$1", b.subject, kind+".%")
			// Grade source rights are separate from consumer field rights.
			f.exec("UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'name') WHERE action='grade.read' AND principal_id=$1", b.subject)
			hidden = h.refresh(kind, v)
			encoded, _ = json.Marshal(hidden)
			if strings.Contains(string(encoded), ids[0]) || strings.Contains(string(encoded), "position") {
				t.Fatal("hidden source identity/order leaked")
			}
			if rec307(hidden)["gradesRestricted"] != true {
				t.Fatal("missing whole-set restriction")
			}
			h.correct(kind, hidden, map[string]any{"gradeIds": ids}, nil, 403)
			h.correct(kind, hidden, map[string]any{"name": "仍获权的资料"}, nil, 200)
			f.exec("UPDATE access.subject_grant SET read_fields=array_append(read_fields,'name') WHERE action='grade.read' AND principal_id=$1", b.subject)
			// Damaged or missing seals never read as an empty collection.
			table := "job"
			if kind == "gradeladder" {
				table = "grade_ladder"
			}
			v = h.refresh(kind, v)
			head := v["headSourceVersion"].(string)
			f.exec("ALTER TABLE staffing." + table + "_grade_manifest DISABLE TRIGGER immutable_manifest")
			f.exec("UPDATE staffing."+table+"_grade_manifest SET digest=decode(repeat('00',32),'hex') WHERE source_id=$1", head)
			b.call("GET", base307(kind)+"/"+id, nil, "", 503)
			f.exec("ALTER TABLE staffing." + table + "_grade_manifest ENABLE TRIGGER immutable_manifest")
			// A damaged current head also fails the authorization/context read for an old source.
			original := b.call("GET", base307(kind)+"/"+id+"?sourceVersion="+source, nil, "", 503)
			if original["gradeIds"] != nil {
				t.Fatal("partial result returned")
			}
		})
	}
}
func kindStatus310(k string) int {
	if k == "gradeladder" {
		return 400
	}
	return 403
}

func TestStaffing310LadderExportAndLog(t *testing.T) {
	h := staffingSetup307(t)
	f, b := h.f, h.b
	clock := f.runtime.Staffing.(app.Service).Store.(*store.Store)
	beforeMidnight := time.Date(2035, 1, 31, 15, 59, 59, 0, time.UTC)
	clock.Clock = func() time.Time { return beforeMidnight }
	a := h.create("grade", "EA310", "")
	bb := h.create("grade", "EB310", "")
	h.correct("grade", a, map[string]any{"name": "原名,含逗号"}, nil, 200)
	a = h.refresh("grade", a)
	ids := []string{a["id"].(string), bb["id"].(string)}
	l := create310(h, "gradeladder", "EXPORT310", ids, "")
	before := l["headSourceVersion"].(string)
	h.correct("gradeladder", l, map[string]any{"gradeIds": []string{ids[1], ids[0]}}, nil, 200)
	l = h.refresh("gradeladder", l)
	log := b.call("GET", base307("gradeladder")+"/"+l["id"].(string)+"/changes", nil, "", 200)
	encoded, _ := json.Marshal(log)
	if !strings.Contains(string(encoded), `"field":"gradeIds"`) || !strings.Contains(string(encoded), `"kind":"members"`) {
		t.Fatal("missing typed member difference")
	}
	in := map[string]any{"draftId": foundation.NewID(), "query": map[string]any{"q": "EXPORT310"}}
	draft := b.call("POST", base307("gradeladder")+"/selection/start", in, "", 200)
	in["expectedRevision"] = draft["revision"]
	rows := csv307(t, b, base307("gradeladder")+"/export", in, 200)
	if len(rows) != 2 || len(rows[0]) != 7 || rows[0][6] != "职级成员" {
		t.Fatal("wrong CSV shape")
	}
	var members [][2]string
	must(t, json.Unmarshal([]byte(rows[1][6]), &members))
	if !reflect.DeepEqual(members, [][2]string{{"EB310", "资料 EB310"}, {"EA310", "原名,含逗号"}}) {
		t.Fatalf("wrong CSV order/value: %v", members)
	}
	// One second crosses the Beijing business day without expiring the 30-minute
	// draft. Both reuse and export must reject the captured prior-day selection.
	clock.Clock = func() time.Time { return beforeMidnight.Add(time.Second) }
	if b.call("POST", base307("gradeladder")+"/selection/resolve", in, "", 409)["code"] != "M03_SELECTION_UNAVAILABLE" {
		t.Fatal("Ladder selection crossed the business day")
	}
	csv307(t, b, base307("gradeladder")+"/export", in, 409)
	clock.Clock = func() time.Time { return beforeMidnight }
	f.exec("UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'code') WHERE action='grade.read' AND principal_id=$1", b.subject)
	csv307(t, b, base307("gradeladder")+"/export", in, 403)
	old := b.call("GET", base307("gradeladder")+"/"+l["id"].(string)+"?sourceVersion="+before, nil, "", 200)
	if rec307(old)["gradeMembers"].([]any)[0].(map[string]any)["gradeId"] != ids[0] {
		t.Fatal("old order overwritten")
	}
}

func TestStaffing310SelectedPermissionAndMissingSeals(t *testing.T) {
	for _, kind := range []string{"job", "gradeladder"} {
		t.Run(kind+"-changed-member-cannot-escape-reverse-index", func(t *testing.T) {
			h := staffingSetup307(t)
			family := h.create("jobfamily", "DAMAGED310-F", "")
			a := h.create("grade", "DAMAGED310-A", "")
			b := h.create("grade", "DAMAGED310-B", "")
			first := create310(h, kind, "DAMAGED310-1", []string{a["id"].(string)}, family["id"].(string))
			last := create310(h, kind, "DAMAGED310-2", []string{a["id"].(string)}, family["id"].(string))
			if first["id"].(string) > last["id"].(string) {
				first, last = last, first
			}
			table := kind
			if kind == "gradeladder" {
				table = "grade_ladder"
			}
			// Both manifests originally seal the same exact members. The first valid
			// inventory entry must not let a later damaged entry reuse its digest.
			var identical bool
			must(t, h.f.admin.QueryRow(h.f.ctx, "SELECT count(DISTINCT digest)=1 FROM staffing."+table+"_grade_manifest WHERE object_id IN ($1,$2)", first["id"], last["id"]).Scan(&identical))
			if !identical {
				t.Fatal("fixture must have identical original membership seals")
			}
			br := rec307(b)
			h.f.exec("ALTER TABLE staffing." + table + "_grade_member DISABLE TRIGGER immutable_member")
			h.f.exec("UPDATE staffing."+table+"_grade_member SET grade_id=$2,adopted_source=$3,adopted_record=$4,adopted_revision=$5 WHERE object_id=$1", last["id"], b["id"], b["headSourceVersion"], br["recordId"], br["revision"])
			h.f.exec("ALTER TABLE staffing." + table + "_grade_member ENABLE TRIGGER immutable_member")
			h.b.call("GET", base307(kind)+"/"+last["id"].(string), nil, "", 503)
			h.add("grade", a, "DEACTIVATE", "2030-01-01", nil, 503)
		})
	}
	h := staffingSetup307(t)
	h.f.runtime.Staffing.(app.Service).Store.(*store.Store).Clock = func() time.Time { return time.Date(2026, 11, 17, 4, 0, 0, 0, time.UTC) }
	f, b := h.f, h.b
	family := h.create("jobfamily", "SCOPED310-F", "")
	a := h.create("grade", "SCOPED310-G", "")
	aid := a["id"].(string)
	for _, k := range []string{"job", "gradeladder"} {
		v := create310(h, k, "SCOPED310", []string{aid}, family["id"].(string))
		// Independent past collection does not require extra Grade history permission
		// when the selected current-day collection is itself fully readable.
		h.add(k, v, "ADD", "2026-11-17", map[string]any{"name": "今日独立记录"}, 200)
		v = h.refresh(k, v)
		f.exec("DELETE FROM access.subject_grant WHERE action='grade.history.read' AND principal_id=$1", b.subject)
		result := h.correct(k, v, map[string]any{"gradeIds": []string{aid}}, nil, 200)
		if result["outcome"] != "no_change" {
			t.Fatal("same selected set changed source")
		}
		r := rec307(v)
		q := map[string]any{"action": "CORRECT", "objectId": v["id"], "recordId": r["recordId"], "expectedSourceVersion": v["headSourceVersion"], "expectedRecordRevision": r["revision"], "changes": map[string]any{"gradeIds": []string{aid}}}
		b.call("POST", base307(k)+"/grade-reference-context", q, "", 200)
		q["objectId"] = "bad-id"
		b.call("POST", base307(k)+"/grade-reference-context", q, "", 400)
		f.exec("INSERT INTO access.subject_grant SELECT instance_id,gen_random_uuid(),principal_id,scope_kind,scope_id,'grade.history.read',read_fields,write_fields FROM access.subject_grant WHERE action='grade.read' AND principal_id=$1", b.subject)
	}
	// Empty old/new Jobs still need an explicit seal; absence is never inferred empty.
	empty := create310(h, "job", "MISSING310", []string{}, family["id"].(string))
	f.exec("ALTER TABLE staffing.job_grade_manifest DISABLE TRIGGER immutable_manifest")
	f.exec("DELETE FROM staffing.job_grade_manifest WHERE object_id=$1", empty["id"])
	f.exec("ALTER TABLE staffing.job_grade_manifest ENABLE TRIGGER immutable_manifest")
	b.call("GET", base307("job")+"/"+empty["id"].(string), nil, "", 503)
	// Reverse qualification must not trust a missing consumer in the member index.
	h.add("grade", a, "DEACTIVATE", "2030-01-01", nil, 503)
}

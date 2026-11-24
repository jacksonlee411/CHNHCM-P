//go:build integration

package bootstrap_test

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func create310(h *staffing307, k, code string, ids []string, family string) map[string]any {
	h.t.Helper()
	reason := "NEW_GRADE_LADDER"
	if k == "job" {
		reason = "NEW_JOB"
	}
	body := map[string]any{"code": code, "name": "资料 " + code, "effectiveDate": "2000-01-01", "operationType": "CREATE", "operationReason": reason, "gradeIds": ids}
	if k == "job" {
		body["jobFunctionId"] = h.b.functionForJobs("1900-01-01")
		body["jobFamilyId"] = family
	}
	r := h.b.call("POST", base307(k), body, foundation.NewID(), 201)
	return h.detail(k, r["objectId"].(string), "")
}
func TestStaffing310CollectionsAndSeals(t *testing.T) {
	h := staffingSetup307(t)
	f := h.create("jobfamily", "F310", "")
	a := h.create("grade", "A310", "")
	b := h.create("grade", "B310", "")
	aid, bid := a["id"].(string), b["id"].(string)
	for _, k := range []string{"job", "gradeladder"} {
		t.Run(k, func(t *testing.T) {
			v := create310(h, k, "C310", []string{aid, bid}, f["id"].(string))
			source := v["headSourceVersion"].(string)
			for _, ids := range []any{nil, []string{aid, aid}, []string{"bad"}} {
				h.correct(k, v, map[string]any{"gradeIds": ids}, nil, 400)
			}
			if k == "gradeladder" {
				h.correct(k, v, map[string]any{"gradeIds": []string{}}, nil, 400)
			}
			old := h.b.call("GET", base307(k)+"/"+v["id"].(string)+"?sourceVersion="+source, nil, "", 200)
			reordered := h.correct(k, v, map[string]any{"gradeIds": []string{bid, aid}}, nil, 200)
			want := "accepted"
			if k == "job" {
				want = "no_change"
			}
			if reordered["outcome"] != want {
				t.Fatalf("order: %v", reordered)
			}
			v = h.refresh(k, v)
			h.correct(k, v, map[string]any{"name": "新名称"}, nil, 200)
			v = h.refresh(k, v)
			if len(rec307(v)["gradeIds"].([]any)) != 2 {
				t.Fatal("omission lost collection")
			}
			if k == "job" {
				h.correct(k, v, map[string]any{"gradeIds": []string{}}, nil, 200)
				v = h.refresh(k, v)
				if len(rec307(v)["gradeIds"].([]any)) != 0 {
					t.Fatal("clear not explicit empty")
				}
			}
			after := h.b.call("GET", base307(k)+"/"+v["id"].(string)+"?sourceVersion="+source, nil, "", 200)
			if !reflect.DeepEqual(rec307(old)["gradeMembers"], rec307(after)["gradeMembers"]) {
				t.Fatal("old evidence changed")
			}
			table := "job"
			if k == "gradeladder" {
				table = "grade_ladder"
			}
			if h.f.scalar("SELECT count(*) FROM staffing."+table+"_grade_manifest") < 1 {
				t.Fatal("missing manifests")
			}
			if _, e := h.f.admin.Exec(h.f.ctx, "DELETE FROM staffing."+table+"_grade_member"); e == nil {
				t.Fatal("mutable member")
			}
		})
	}
}
func TestStaffing310WholePeriodReverseAndEvidence(t *testing.T) {
	h := staffingSetup307(t)
	f := h.create("jobfamily", "RF310", "")
	a := h.create("grade", "RA310", "")
	b := h.create("grade", "RB310", "")
	aid, bid := a["id"].(string), b["id"].(string)
	j := create310(h, "job", "RJ310", []string{aid, bid}, f["id"].(string))
	l := create310(h, "gradeladder", "RL310", []string{aid, bid}, "")
	source := l["headSourceVersion"].(string)
	old := h.b.call("GET", base307("gradeladder")+"/"+l["id"].(string)+"?sourceVersion="+source, nil, "", 200)
	h.add("grade", b, "DEACTIVATE", "2030-01-01", nil, 409)
	h.correct("grade", a, map[string]any{"name": "当前新名"}, nil, 200)
	now := h.refresh("gradeladder", l)
	if rec307(now)["gradeMembers"].([]any)[0].(map[string]any)["name"] != "当前新名" {
		t.Fatal("current label not refreshed")
	}
	exact := h.b.call("GET", base307("gradeladder")+"/"+l["id"].(string)+"?sourceVersion="+source, nil, "", 200)
	if !reflect.DeepEqual(rec307(old)["gradeMembers"], rec307(exact)["gradeMembers"]) {
		t.Fatal("source rename rewrote evidence")
	}
	h.add("job", j, "ADD", "2030-01-01", map[string]any{"gradeIds": []string{aid}}, 200)
	h.add("grade", b, "DEACTIVATE", "2030-01-01", nil, 409)
	h.add("gradeladder", l, "ADD", "2030-01-01", map[string]any{"gradeIds": []string{aid}}, 200)
	h.add("grade", b, "DEACTIVATE", "2030-01-01", nil, 200)
	// Removing the future narrowed consumer record would extend B beyond its end.
	j = h.refresh("job", j)
	timeline := j["timeline"].([]any)
	future := timeline[len(timeline)-1].(map[string]any)
	h.b.call("POST", base307("job")+"/"+j["id"].(string)+"/records/"+future["recordId"].(string)+"/delete", map[string]any{"expectedSourceVersion": j["headSourceVersion"], "expectedRecordRevision": future["revision"]}, foundation.NewID(), 409)
	// An open new Ladder cannot reference a Grade with scheduled permanent end.
	h.b.call("POST", base307("gradeladder"), map[string]any{"code": "OPEN310", "name": "开放尾段", "effectiveDate": "2000-01-01", "operationType": "CREATE", "operationReason": "NEW_GRADE_LADDER", "gradeIds": []string{bid}}, foundation.NewID(), 409)
	// The source's future resume is required by both consumer kinds. Grade
	// redating, deletion fallback and a larger fixed inactive period must all
	// reject, even when the actor cannot read the consumers.
	x, y := h.create("grade", "REVERSE-X310", ""), h.create("grade", "REVERSE-Y310", "")
	xid, yid := x["id"].(string), y["id"].(string)
	for _, kind := range []string{"job", "gradeladder"} {
		consumer := create310(h, kind, "FUTURE310", []string{yid}, f["id"].(string))
		h.add(kind, consumer, "ADD", "2040-01-01", map[string]any{"gradeIds": []string{xid}}, 200)
	}
	path := base307("grade") + "/" + xid
	h.b.call("POST", path+"/lifecycle-pairs/insert", map[string]any{"expectedSourceVersion": x["headSourceVersion"], "anchorRecordId": rec307(x)["recordId"], "deactivationDate": "2030-01-01", "reactivationDate": "2040-01-01", "deactivationReason": "GRADE_CLOSURE", "reactivationReason": "GRADE_RESUMPTION"}, foundation.NewID(), 200)
	x = h.refresh("grade", x)
	records := x["timeline"].([]any)
	stop, start := records[1].(map[string]any), records[2].(map[string]any)
	h.f.exec("UPDATE access.subject_grant SET read_fields=ARRAY[]::text[] WHERE principal_id=$1 AND action IN ('job.read','gradeladder.read')", h.b.subject)
	count := h.f.scalar("SELECT count(*) FROM staffing.grade_source")
	h.correct("grade", y, nil, map[string]any{"newEffectiveDate": "2001-01-01"}, 409)
	h.b.call("POST", path+"/records/"+start["recordId"].(string)+"/redate", map[string]any{"expectedSourceVersion": x["headSourceVersion"], "expectedRecordRevision": start["revision"], "newEffectiveDate": "2041-01-01"}, foundation.NewID(), 409)
	h.b.call("POST", path+"/records/"+start["recordId"].(string)+"/delete", map[string]any{"expectedSourceVersion": x["headSourceVersion"], "expectedRecordRevision": start["revision"]}, foundation.NewID(), 409)
	h.b.call("POST", path+"/lifecycle-pairs/maintain", map[string]any{"expectedSourceVersion": x["headSourceVersion"], "deactivationRecordId": stop["recordId"], "deactivationRevision": stop["revision"], "reactivationRecordId": start["recordId"], "reactivationRevision": start["revision"], "deactivationDate": "2030-01-01", "reactivationDate": "2041-01-01", "deactivationReason": "GRADE_CLOSURE", "reactivationReason": "GRADE_RESUMPTION"}, foundation.NewID(), 409)
	if h.f.scalar("SELECT count(*) FROM staffing.grade_source") != count || h.refresh("grade", x)["headSourceVersion"] != x["headSourceVersion"] {
		t.Fatal("rejected reverse qualification wrote facts")
	}
}
func TestStaffing310CandidateAndDigest(t *testing.T) {
	h := staffingSetup307(t)
	f := h.create("jobfamily", "CF310", "")
	a := h.create("grade", "CA310", "")
	b := h.create("grade", "CB310", "")
	aid, bid := a["id"].(string), b["id"].(string)
	h.add("grade", b, "DEACTIVATE", "2030-01-01", nil, 200)
	q := map[string]any{"action": "CREATE", "operationType": "CREATE", "effectiveDate": "2000-01-01", "gradeIds": []string{}, "referencePageSize": 50}
	v := h.b.call("POST", base307("gradeladder")+"/grade-reference-context", q, "", 200)
	if v["totalCount"] != float64(1) || v["items"].([]any)[0].(map[string]any)["id"] != aid {
		t.Fatalf("open candidate: %v", v)
	}
	q["referenceQ"] = strings.Repeat("级", 2001)
	h.b.call("POST", base307("gradeladder")+"/grade-reference-context", q, "", 400)
	q["referenceQ"] = strings.Repeat("级", 2000)
	h.b.call("POST", base307("gradeladder")+"/grade-reference-context", q, "", 200)
	delete(q, "referenceQ")
	q["gradeIds"] = []string{bid}
	v = h.b.call("POST", base307("gradeladder")+"/grade-reference-context", q, "", 200)
	if v["selectedUnavailable"] != true || len(v["selected"].([]any)) != 1 {
		t.Fatal("invalid selected coverage not explained")
	}
	// Moving the independent successor must not advertise candidates while
	// its unchanged predecessor would extend beyond B's scheduled end.
	ladder := create310(h, "gradeladder", "FINITE310", []string{aid}, "")
	h.add("gradeladder", ladder, "ADD", "2030-01-01", map[string]any{"gradeIds": []string{aid}, "name": "独立后继"}, 200)
	ladder = h.refresh("gradeladder", ladder)
	h.correct("gradeladder", ladder, map[string]any{"gradeIds": []string{bid}}, nil, 200)
	ladder = h.refresh("gradeladder", ladder)
	future := ladder["timeline"].([]any)[1].(map[string]any)
	moved := map[string]any{"action": "CORRECT", "objectId": ladder["id"], "recordId": future["recordId"], "expectedSourceVersion": ladder["headSourceVersion"], "expectedRecordRevision": future["revision"], "newEffectiveDate": "2031-01-01", "changes": map[string]any{"gradeIds": []string{aid}}}
	h.b.call("POST", base307("gradeladder")+"/grade-reference-context", moved, "", 409)
	moved["newEffectiveDate"] = "2030-01-01"
	h.b.call("POST", base307("gradeladder")+"/grade-reference-context", moved, "", 200)
	moved["changes"] = map[string]any{"gradeIds": []string{}}
	draft := h.b.call("POST", base307("gradeladder")+"/grade-reference-context", moved, "", 200)
	if len(draft["selected"].([]any)) != 0 || draft["totalCount"] == float64(0) {
		t.Fatal("empty correction draft must still return selectable candidates")
	}
	q["gradeIds"] = nil
	h.b.call("POST", base307("gradeladder")+"/grade-reference-context", q, "", 400)
	// Same logical Job set uses the same canonical submission digest.
	c := h.create("grade", "CC310", "")
	cid := c["id"].(string)
	body := map[string]any{"code": "CJ310", "jobFunctionId": h.b.functionForJobs("1900-01-01"), "name": "集合", "effectiveDate": "2000-01-01", "operationType": "CREATE", "operationReason": "NEW_JOB", "jobFamilyId": f["id"], "gradeIds": []string{aid, cid}}
	key := foundation.NewID()
	created := h.b.call("POST", base307("job"), body, key, 201)
	body["gradeIds"] = []string{cid, aid}
	replay := h.b.call("POST", base307("job"), body, key, 201)
	if created["sourceVersion"] != replay["sourceVersion"] {
		t.Fatal("replayed new source")
	}
	t.Run("hidden-code-candidate-order", func(t *testing.T) {
		// Names deliberately contain no code. Expected order comes from the
		// public identities, independently of the candidate query's ordering.
		codeOrder := []string{}
		for i := 0; i < 23; i++ {
			v := h.b.call("POST", base307("grade"), map[string]any{"code": fmt.Sprintf("HIDDEN-ORD-%02d", i), "name": "候选排序样本", "effectiveDate": "2000-01-01", "operationType": "CREATE", "operationReason": "NEW_GRADE"}, foundation.NewID(), 201)
			codeOrder = append(codeOrder, v["objectId"].(string))
		}
		idOrder := slices.Clone(codeOrder)
		slices.Sort(idOrder)
		if slices.Equal(codeOrder, idOrder) {
			t.Fatal("fixture did not distinguish code order from identity order")
		}
		for _, hidden := range []bool{true, false} {
			operation := "array_remove"
			want := idOrder
			if !hidden {
				operation, want = "array_append", codeOrder
			}
			h.f.exec("UPDATE access.subject_grant SET read_fields="+operation+"(read_fields,'code') WHERE action IN ('grade.read','grade.history.read') AND principal_id=$1", h.b.subject)
			for _, kind := range []string{"job", "gradeladder"} {
				query := map[string]any{"action": "CREATE", "operationType": "CREATE", "effectiveDate": "2000-01-01", "gradeIds": []string{}, "referencePageSize": 20, "referenceQ": "候选排序样本"}
				if kind == "job" {
					query["jobFamilyId"] = f["id"]
				}
				actual := []string{}
				for page := 1; page <= 2; page++ {
					query["referencePage"] = page
					v := h.b.call("POST", base307(kind)+"/grade-reference-context", query, "", 200)
					if v["totalCount"] != float64(23) {
						t.Fatal("candidate count differs")
					}
					for _, value := range v["items"].([]any) {
						row := value.(map[string]any)
						if (row["code"] == nil) != hidden {
							t.Fatal("candidate code field permission differs")
						}
						actual = append(actual, row["id"].(string))
					}
				}
				if !slices.Equal(actual, want) {
					t.Fatalf("%s hidden=%t: candidate order does not follow readable fields", kind, hidden)
				}
				query["referencePage"], query["referenceQ"] = 1, "HIDDEN-ORD-"
				v := h.b.call("POST", base307(kind)+"/grade-reference-context", query, "", 200)
				count := float64(23)
				if hidden {
					count = 0
				}
				if v["totalCount"] != count {
					t.Fatal("candidate search used an unreadable code")
				}
			}
		}
	})
}

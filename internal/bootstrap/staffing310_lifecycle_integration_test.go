//go:build integration

package bootstrap_test

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"reflect"
	"testing"
)

func TestStaffing310LifecycleCoverageAndAdoption(t *testing.T) {
	h := staffingSetup307(t)
	family := h.create("jobfamily", "DATE310-F", "")
	for _, k := range []string{"job", "gradeladder"} {
		t.Run(k, func(t *testing.T) {
			a, b := h.create("grade", "DATE-A-"+k, ""), h.create("grade", "DATE-B-"+k, "")
			aid, bid := a["id"].(string), b["id"].(string)
			v := create310(h, k, "DATE310", []string{aid, bid}, family["id"].(string))
			id := v["id"].(string)
			base := base307(k) + "/" + id
			source := v["headSourceVersion"].(string)
			first := rec307(v)["recordId"].(string)
			exact := func(v map[string]any) []any {
				out := h.b.call("GET", base+"?sourceVersion="+v["headSourceVersion"].(string)+"&selectedRecordId="+rec307(v)["recordId"].(string), nil, "", 200)
				return rec307(out)["gradeMembers"].([]any)
			}
			original := exact(v)
			stopReason, startReason := "JOB_CLOSURE", "JOB_RESUMPTION"
			if k == "gradeladder" {
				stopReason, startReason = "GRADE_LADDER_CLOSURE", "GRADE_LADDER_RESUMPTION"
			}
			h.correct("grade", a, map[string]any{"name": "恢复时新采用"}, nil, 200)
			a = h.refresh("grade", a)
			body := map[string]any{"expectedSourceVersion": v["headSourceVersion"], "anchorRecordId": first, "deactivationDate": "2005-01-01", "reactivationDate": "2010-01-01", "deactivationReason": stopReason, "reactivationReason": startReason, "changes": map[string]any{"gradeIds": []string{aid}}}
			h.b.call("POST", base+"/lifecycle-pairs/insert", body, foundation.NewID(), 200)
			v = h.refresh(k, v)
			restored := exact(v)
			if len(restored) != 1 || restored[0].(map[string]any)["name"] != "恢复时新采用" || restored[0].(map[string]any)["adopted"].(map[string]any)["businessAt"] != "2010-01-01" {
				t.Fatal("restore did not adopt complete final membership")
			}
			h.add("grade", b, "DEACTIVATE", "2005-01-01", nil, 200)
			b = h.refresh("grade", b)
			pairBody := func(v map[string]any) map[string]any {
				pair := v["pairContext"].(map[string]any)
				stop, start := pair["deactivation"].(map[string]any), pair["reactivation"].(map[string]any)
				return map[string]any{"expectedSourceVersion": v["headSourceVersion"], "deactivationRecordId": stop["recordId"], "deactivationRevision": stop["revision"], "reactivationRecordId": start["recordId"], "reactivationRevision": start["revision"]}
			}
			body = pairBody(v)
			body["deactivationDate"] = "2006-01-01"
			body["reactivationDate"] = "2010-01-01"
			body["deactivationReason"] = stopReason
			body["reactivationReason"] = startReason
			h.b.call("POST", base+"/lifecycle-pairs/maintain", body, foundation.NewID(), 409)
			if h.refresh(k, v)["headSourceVersion"] != v["headSourceVersion"] {
				t.Fatal("rejected extension wrote source")
			}
			body["deactivationDate"] = "2004-01-01"
			h.b.call("POST", base+"/lifecycle-pairs/maintain", body, foundation.NewID(), 200)
			v = h.refresh(k, v)
			if !reflect.DeepEqual(restored, exact(v)) {
				t.Fatal("stop-only change rewrote restored adoption")
			}
			h.b.call("POST", base+"/lifecycle-pairs/delete", pairBody(v), foundation.NewID(), 409)
			// Missing collection authority rejects the implicit membership restored by deletion.
			h.f.exec("UPDATE access.subject_grant SET write_fields=array_remove(write_fields,'gradeIds') WHERE principal_id=$1 AND action=$2", h.b.subject, k+".lifecycle.pair")
			h.b.call("POST", base+"/lifecycle-pairs/delete", pairBody(v), foundation.NewID(), 403)
			h.f.exec("UPDATE access.subject_grant SET write_fields=array_append(write_fields,'gradeIds') WHERE principal_id=$1 AND action=$2", h.b.subject, k+".lifecycle.pair")
			// Reopening the source is safe; deleting the consumer pair now inherits old evidence.
			h.b.call("POST", base307("grade")+"/"+bid+"/records/"+rec307(b)["recordId"].(string)+"/delete", map[string]any{"expectedSourceVersion": b["headSourceVersion"], "expectedRecordRevision": rec307(b)["revision"]}, foundation.NewID(), 200)
			h.b.call("POST", base+"/lifecycle-pairs/delete", pairBody(v), foundation.NewID(), 200)
			v = h.refresh(k, v)
			if !reflect.DeepEqual(original, exact(v)) {
				t.Fatal("delete fallback rewrote adoption")
			}
			// A first-date move re-adopts every member but must stay inside source coverage.
			h.correct(k, v, nil, map[string]any{"newEffectiveDate": "1999-12-31"}, 409)
			h.correct(k, v, nil, map[string]any{"newEffectiveDate": "2001-01-01"}, 200)
			v = h.refresh(k, v)
			for _, m := range exact(v) {
				if m.(map[string]any)["adopted"].(map[string]any)["businessAt"] != "2001-01-01" {
					t.Fatal("redate retained old adoption")
				}
			}
			old := h.b.call("GET", base+"?sourceVersion="+source, nil, "", 200)
			if !reflect.DeepEqual(original, rec307(old)["gradeMembers"]) {
				t.Fatal("old source drifted")
			}
			// Single stop inherits the original set; an unavailable member cannot silently disappear on restore.
			h.add(k, v, "DEACTIVATE", "2015-01-01", nil, 200)
			v = h.refresh(k, v)
			b = h.refresh("grade", b)
			h.add("grade", b, "DEACTIVATE", "2015-01-01", nil, 200)
			h.add(k, v, "REACTIVATE", "2020-01-01", nil, 409)
			h.add(k, v, "REACTIVATE", "2020-01-01", map[string]any{"gradeIds": []string{aid}}, 200)
			v = h.refresh(k, v)
			if len(rec307(v)["gradeIds"].([]any)) != 1 {
				t.Fatal("restore failed explicit replacement")
			}
		})
	}
}

func TestStaffing310RedateMemberAuthority(t *testing.T) {
	h := staffingSetup307(t)
	family := h.create("jobfamily", "MOVE310-F", "")
	a, b := h.create("grade", "MOVE310-A", ""), h.create("grade", "MOVE310-B", "")
	aid, bid := a["id"].(string), b["id"].(string)
	for _, kind := range []string{"job", "gradeladder"} {
		v := create310(h, kind, "MOVE310", []string{aid, bid}, family["id"].(string))
		added := h.add(kind, v, "ADD", "2026-03-01", map[string]any{"gradeIds": []string{aid}}, 200)
		second := h.detail(kind, v["id"].(string), added["selectedRecordId"].(string))
		third := h.add(kind, second, "ADD", "2026-06-01", map[string]any{"name": "同成员后继"}, 200)
		second = h.detail(kind, v["id"].(string), added["selectedRecordId"].(string))
		h.f.exec("UPDATE access.subject_grant SET write_fields=array_remove(write_fields,'gradeIds') WHERE principal_id=$1 AND action=$2", h.b.subject, kind+".record.correct")
		for _, date := range []string{"2026-02-01", "2026-04-01"} {
			fail := h.correct(kind, second, nil, map[string]any{"newEffectiveDate": date}, 403)
			if fail["code"] != "M03_FORBIDDEN" || h.refresh(kind, v)["headSourceVersion"] != second["headSourceVersion"] {
				t.Fatalf("%s %s: lost authority or unexpected source change: %v", kind, date, fail)
			}
		}
		equal := h.detail(kind, v["id"].(string), third["selectedRecordId"].(string))
		h.correct(kind, equal, nil, map[string]any{"newEffectiveDate": "2026-05-01"}, 200)
	}
}

func TestStaffing310GradeGuardsWithoutReadProjection(t *testing.T) {
	h := staffingSetup307(t)
	family := h.create("jobfamily", "HEAD310-F", "")
	for _, kind := range []string{"job", "gradeladder"} {
		g := h.create("grade", "HEAD310-"+kind, "")
		id := g["id"].(string)
		v := create310(h, kind, "HEAD310", []string{id}, family["id"].(string))
		table := "job_read"
		if kind == "gradeladder" {
			table = "grade_ladder_read"
		}
		// Drop only this disposable consumer's rebuildable projection.
		h.f.exec("DELETE FROM staffing."+table+" WHERE object_id=$1", v["id"])
		probe := map[string]any{"action": "DEACTIVATE", "expectedSourceVersion": g["headSourceVersion"], "anchorRecordId": rec307(g)["recordId"], "effectiveDate": "2030-01-01", "operationType": "DEACTIVATE", "operationReason": "GRADE_CLOSURE"}
		preview := h.b.call("POST", base307("grade")+"/"+id+"/preview", probe, "", 409)
		write := h.add("grade", g, "DEACTIVATE", "2030-01-01", nil, 409)
		if preview["code"] != "M03_REFERENCE_CONFLICT" || write["code"] != preview["code"] || h.refresh("grade", g)["headSourceVersion"] != g["headSourceVersion"] {
			t.Fatalf("%s head failed to protect its active reference", kind)
		}
	}
}

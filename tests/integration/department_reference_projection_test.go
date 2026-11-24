//go:build integration

package integration_test

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"testing"
)

func TestDepartmentSelectedReferenceIsIndependentOfEligibilityAndSearch(t *testing.T) {
	s := newDepartmentContract(t)
	s.create("P_OTHER", "可读但不符合当前BU的上级", "BU_B", "", "2026-01-01", s.b, 201)
	query := map[string]any{"context": map[string]any{"mode": "create", "effectiveDate": "2026-01-01", "managementBuId": s.aliases["BU_A"]}, "selectedId": s.aliases["P_OTHER"], "pageSize": 20}
	read := func() map[string]any {
		return s.b.call("POST", "/api/v1/departments/references/parents", query, "", 200)
	}
	assertSelection := func(out map[string]any) {
		t.Helper()
		ref := out["selectedReference"].(map[string]any)
		if ref["id"] != s.aliases["P_OTHER"] || ref["name"] != "可读但不符合当前BU的上级" {
			t.Fatal("readable current selection lost its label", out)
		}
		if _, exists := ref["eligibility"]; exists {
			t.Fatal("selected label must not authorize a candidate", ref)
		}
		for _, item := range out["items"].([]any) {
			if item.(map[string]any)["id"] == s.aliases["P_OTHER"] {
				t.Fatal("cross-BU parent became eligible", out)
			}
		}
	}
	assertSelection(read())
	query["q"] = "不匹配任何候选"
	assertSelection(read())
	// Removing reference name authority must not be confused with an empty page.
	mutateLifeGrants(s.f, s.b.subject, `UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'name') WHERE principal_id=$1 AND action='department.reference.read'`, s.b.subject)
	ref := read()["selectedReference"].(map[string]any)
	if ref["code"] != "P_OTHER" || ref["name"] != nil {
		t.Fatal("partial reference projection", ref)
	}
	mutateLifeGrants(s.f, s.b.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='department.reference.read'`, s.b.subject)
	ref = read()["selectedReference"].(map[string]any)
	if len(ref) != 1 || ref["id"] != s.aliases["P_OTHER"] {
		t.Fatal("revoked selected reference leaked", ref)
	}
	unknown := foundation.NewID()
	query["selectedId"] = unknown
	ref = read()["selectedReference"].(map[string]any)
	if len(ref) != 1 || ref["id"] != unknown {
		t.Fatal("unknown and unreadable references must have the same opaque shape", ref)
	}
}

func TestDepartmentSelectedBUReferenceUsesCurrentAuthorityAndStrictInput(t *testing.T) {
	s := newDepartmentContract(t)
	query := map[string]any{"context": map[string]any{"mode": "filter"}, "selectedId": s.aliases["BU_A"], "q": "不匹配任何候选", "page": 1, "pageSize": 20}
	read := func(status int) map[string]any {
		return s.actor.call("POST", "/api/v1/departments/references/business-units", query, "", status)
	}
	before := read(200)
	if before["count"] != float64(0) || before["selectedReference"].(map[string]any)["code"] != "BU_A" {
		t.Fatal("selection depends on the search result", before)
	}
	mutateLifeGrants(s.f, s.actor.subject, `UPDATE access.subject_grant SET read_fields=ARRAY[]::text[] WHERE principal_id=$1 AND action='bu.reference.read'`, s.actor.subject)
	ref := read(200)["selectedReference"].(map[string]any)
	if ref["id"] != s.aliases["BU_A"] || ref["name"] != nil || ref["code"] != nil {
		t.Fatal("current minimal reference rights ignored", ref)
	}
	for _, bad := range []any{"", nil, "not-a-uuid", 1} {
		query["selectedId"] = bad
		read(400)
	}
	delete(query, "selectedId")
	if _, exists := read(200)["selectedReference"]; exists {
		t.Fatal("absent selection fabricated a reference")
	}
	query["selectedId"] = s.aliases["BU_A"]
	mutateLifeGrants(s.f, s.actor.subject, `UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'managementBuId') WHERE principal_id=$1 AND action='department.read'`, s.actor.subject)
	if denied := read(403); denied["selectedReference"] != nil {
		t.Fatal("selection bypassed the entry field boundary", denied)
	}
}

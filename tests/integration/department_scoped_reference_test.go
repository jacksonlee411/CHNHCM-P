//go:build integration

package integration_test

import "testing"

func TestDepartmentScopedReadRetainsInactiveHistoricalReference(t *testing.T) {
	s := newDepartmentContract(t)
	s.create("OLD", "原上级", "BU_A", "", "2026-01-01", s.b, 201)
	s.create("NEW", "更正后上级", "BU_A", "", "2026-01-01", s.b, 201)
	s.create("C", "待停用部门", "BU_A", "OLD", "2026-01-01", s.b, 201)
	s.append("C", "STOP", "2026-03-01", "DEACTIVATE", nil, s.b, 201)
	s.edit("C", "C_FIRST", "correct", map[string]any{"changes": map[string]any{"parentDepartmentId": s.aliases["NEW"]}}, s.b, 200)
	// The inactive row keeps its exact historical member; the current path follows
	// corrected facts on the last active day. Both referenced identities are real.
	d := s.b.call("GET", "/api/v1/departments/"+s.aliases["C"]+"?selectedRecordId="+s.records["STOP"], nil, "", 200)
	r := d["selectedRecord"].(map[string]any)
	var parent string
	for _, fact := range s.f.departmentHead(s.aliases["C"]).State.Timeline.Records {
		if fact.ID == s.records["STOP"] {
			parent = fact.Data.Member.Parent
		}
	}
	want := map[string]string{s.aliases["OLD"]: "原上级", s.aliases["NEW"]: "更正后上级"}[parent]
	ref := r["parentDepartment"].(map[string]any)
	if want == "" || ref["name"] != want {
		t.Fatalf("scoped snapshot dropped an authorized stored reference: stored=%s response=%v", parent, ref)
	}
}

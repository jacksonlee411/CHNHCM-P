//go:build integration

package integration_test

import (
	"strings"
	"testing"
)

func TestDepartmentR5FilterReferenceReadBoundary(t *testing.T) {
	s := newDepartmentContract(t)
	s.create("C", "筛选部门", "BU_A", "", "2026-01-01", s.b, 201)
	query := map[string]any{"context": map[string]any{"mode": "filter"}, "page": 1, "pageSize": 20}
	read := func() map[string]any {
		return s.actor.call("POST", "/api/v1/departments/references/business-units", query, "", 200)
	}
	// The default reader has no department create/maintenance rights.
	out := read()
	if out["count"] != float64(2) {
		t.Fatal("read-only filter must expose both authorized BU references", out)
	}
	for _, item := range out["items"].([]any) {
		if item.(map[string]any)["eligibility"] != "readable" {
			t.Fatal("filter must not claim maintenance eligibility")
		}
	}
	query["q"] = s.aliases["BU_A"]
	if got := read(); got["count"] != float64(1) || got["items"].([]any)[0].(map[string]any)["id"] != s.aliases["BU_A"] {
		t.Fatal("stable ID label resolution failed", got)
	}
	// No date or action can be smuggled into a read-filter context.
	query["context"] = map[string]any{"mode": "filter", "effectiveDate": "2027-01-01"}
	s.actor.call("POST", "/api/v1/departments/references/business-units", query, "", 400)
	query["context"] = map[string]any{"mode": "filter"}
	s.actor.call("POST", "/api/v1/departments/references/parents", query, "", 400)
	mutateLifeGrants(s.f, s.actor.subject, `UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'name') WHERE principal_id=$1 AND action='bu.reference.read'`, s.actor.subject)
	got := read()["items"].([]any)[0].(map[string]any)
	if _, ok := got["name"]; ok {
		t.Fatal("revoked reference name leaked")
	}
	mutateLifeGrants(s.f, s.actor.subject, `UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'managementBuId') WHERE principal_id=$1 AND action='department.read'`, s.actor.subject)
	s.actor.call("POST", "/api/v1/departments/references/business-units", query, "", 403)
}

func TestDepartmentR5PathTextPeriodAndReferenceRights(t *testing.T) {
	s := newDepartmentContract(t)
	s.create("P", "上级旧名", "BU_A", "", "2026-01-01", s.b, 201)
	s.create("C", "子部门", "BU_A", "P", "2026-01-01", s.b, 201)
	s.append("P", "P_JULY", "2026-07-01", "CHANGE", map[string]any{"name": "上级新名"}, s.b, 201)
	detail := s.actor.departmentCurrent(s.aliases["C"])
	path := detail["path"].(map[string]any)
	segments := path["pathSegments"].([]any)
	if len(segments) != 2 || segments[0].(map[string]any)["endInclusive"] != "2026-06-30" || segments[1].(map[string]any)["endInclusive"] != nil {
		t.Fatal("Go path display periods", path)
	}
	if detail["nameGuidance"] == "" {
		t.Fatal("maintenance needs server-owned name guidance")
	}
	mutateLifeGrants(s.f, s.actor.subject, `UPDATE access.subject_grant SET scope_kind='department',scope_id=$2 WHERE principal_id=$1 AND action='department.read'`, s.actor.subject, s.aliases["C"])
	scoped := s.actor.departmentCurrent(s.aliases["C"])["path"].(map[string]any)["pathSegments"].([]any)
	for _, seg := range scoped {
		item := seg.(map[string]any)
		if _, exists := item["nodes"]; exists {
			t.Fatal("navigation-only path nodes remain", item)
		}
		if !strings.Contains(item["path"].(string), "上级") {
			t.Fatal("reference name incorrectly required detail permission", item)
		}
	}

	mutateLifeGrants(s.f, s.actor.subject, `UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'name') WHERE principal_id=$1 AND action='department.reference.read'`, s.actor.subject)
	restricted := s.actor.departmentCurrent(s.aliases["C"])["path"].(map[string]any)
	if restricted["pathMode"] != "restricted" {
		t.Fatal("missing ancestor name must restrict whole path", restricted)
	}
	for _, key := range []string{"pathSegments", "pathBusinessAt", "departmentPath"} {
		if _, ok := restricted[key]; ok {
			t.Fatal("restricted path leaked structure", key)
		}
	}
}

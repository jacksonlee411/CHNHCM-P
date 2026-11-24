//go:build integration

package integration_test

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"testing"
)

func TestDepartmentNameValidationLocationAndRevokedRecovery(t *testing.T) {
	s := newDepartmentContract(t)
	s.create("C", "原部门名称", "BU_A", "", "2026-01-01", s.b, 201)
	before := s.b.departmentCurrent(s.aliases["C"])
	r := before["selectedRecord"].(map[string]any)
	location := func(value map[string]any, expected string) {
		t.Helper()
		issues := value["failureDetails"].(map[string]any)["issues"].([]any)
		locations := issues[0].(map[string]any)["locations"].([]any)
		if len(locations) != 1 || locations[0].(map[string]any)["fieldPath"] != expected {
			t.Fatal("name validation must locate the readable field", value)
		}
	}
	var nameSubmission string
	for _, field := range []string{"name", "shortName"} {
		key := foundation.NewID()
		body := map[string]any{"expectedSourceVersion": before["headSourceVersion"], "expectedRecordRevision": r["revision"], "changes": map[string]any{field: "😀"}}
		out := s.b.call("POST", "/api/v1/departments/"+s.aliases["C"]+"/records/"+r["recordId"].(string)+"/correct", body, key, 400)
		location(out, field)
		location(s.b.call("POST", "/api/v1/departments/submissions/"+key+"/resolve", nil, "", 200), field)
		if field == "name" {
			nameSubmission = key
		}
	}
	created := departmentBody("INVALID_NAME", "😀", s.aliases["BU_A"], "2026-01-01", nil)
	location(s.b.call("POST", "/api/v1/departments", created, foundation.NewID(), 400), "name")
	mutateLifeGrants(s.f, s.b.subject, `UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'name') WHERE principal_id=$1 AND action IN ('department.read','department.history.read')`, s.b.subject)
	resolved := s.b.call("POST", "/api/v1/departments/submissions/"+nameSubmission+"/resolve", nil, "", 200)
	for _, issue := range resolved["failureDetails"].(map[string]any)["issues"].([]any) {
		if len(issue.(map[string]any)["locations"].([]any)) != 0 {
			t.Fatal("revoked field location leaked on recovery", resolved)
		}
	}
	if after := s.b.departmentCurrent(s.aliases["C"]); after["headSourceVersion"] != before["headSourceVersion"] {
		t.Fatal("invalid name advanced business source", after)
	}
}

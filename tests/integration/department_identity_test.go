//go:build integration

package integration_test

import "testing"

func TestDepartmentV09HumanWithoutDisplayName(t *testing.T) {
	s := newDepartmentContract(t)
	s.f.exec(`UPDATE access.principal SET display_name=NULL WHERE principal_id=$1`, s.b.subject)
	result := s.create("EMPTY_NAME", "空姓名操作者", "BU_A", "", "2026-01-01", s.b, 201)
	current := s.current("EMPTY_NAME")
	op := summaryOf(t, current)["operator"].(map[string]any)
	if op["kind"] != "human" || op["displayName"] != nil || op["login"] != "admin" || op["identityEvidence"] != "captured" {
		t.Fatal("human identity snapshot incomplete", op)
	}
	r := selected(current)
	for _, field := range []string{"shortName", "endExclusive", "endInclusive"} {
		value, present := r[field]
		if !present || value != nil {
			t.Fatal("authorized empty value must be explicit null", field, r)
		}
	}
	s.f.exec(`UPDATE access.principal SET display_name='后来补全' WHERE principal_id=$1`, s.b.subject)
	exact := s.exact("EMPTY_NAME", result["sourceVersion"].(string))
	if summaryOf(t, exact)["operator"].(map[string]any)["displayName"] != nil {
		t.Fatal("historical identity filled from current principal")
	}
}

//go:build integration

package integration_test

import "testing"

func TestDepartmentReadProjectionCurrentScopeWithoutPersistentDecisions(t *testing.T) {
	s := newDepartmentContract(t)
	s.create("P", "获权上级", "BU_A", "", "2026-01-01", s.b, 201)
	s.create("C", "获权子部门", "BU_A", "P", "2026-01-01", s.b, 201)
	s.create("H", "范围外部门", "BU_B", "", "2026-01-01", s.b, 201)
	allDepartmentRights(s)
	mutateLifeGrants(s.f, s.actor.subject, `UPDATE access.subject_grant SET scope_kind='bu',scope_id=$2 WHERE principal_id=$1 AND action='department.read'`, s.actor.subject, s.aliases["BU_A"])
	securityEvents := s.f.scalar("SELECT count(*) FROM access.security_event")
	first := s.actor.call("GET", "/api/v1/departments", nil, "", 200)
	if first["count"] != float64(2) {
		t.Fatal("batch reads crossed BU scope", first)
	}
	for _, v := range first["items"].([]any) {
		row := v.(map[string]any)
		if row["name"] == nil || row["id"] == s.aliases["H"] {
			t.Fatal("wrong authorized projection", row)
		}
	}
	if s.f.scalar("SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='access' AND c.relname='evidence_content'") != 0 {
		t.Fatal("authorization content table remains")
	}
	draft := departmentDraft(s.actor, map[string]any{})
	sources := s.f.scalar("SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.department_history")
	mutateLifeGrants(s.f, s.actor.subject, `UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'name') WHERE principal_id=$1 AND action='department.read'`, s.actor.subject)
	second := s.actor.call("GET", "/api/v1/departments", nil, "", 200)
	resolved := departmentResolve(s.actor, draftBody(map[string]any{}, draft), 200)
	if resolved["draftId"] != draft["draftId"] {
		t.Fatal("field revocation replaced fixed identities")
	}
	for _, v := range second["items"].([]any) {
		if _, ok := v.(map[string]any)["name"]; ok {
			t.Fatal("cached field survived revocation")
		}
	}
	mutateLifeGrants(s.f, s.actor.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='department.reference.read'`, s.actor.subject)
	third := s.actor.call("GET", "/api/v1/departments", nil, "", 200)
	for _, v := range third["items"].([]any) {
		if v.(map[string]any)["path"].(map[string]any)["pathMode"] != "restricted" {
			t.Fatal("reference revocation exposed path")
		}
	}
	mutateLifeGrants(s.f, s.actor.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='department.read'`, s.actor.subject)
	s.actor.call("GET", "/api/v1/departments", nil, "", 403)
	if s.f.scalar("SELECT count(*) FROM access.security_event") != securityEvents {
		t.Fatal("ordinary reads or denials persisted authorization events")
	}
	if s.f.scalar("SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.department_history") != sources {
		t.Fatal("read changed immutable business facts")
	}
}

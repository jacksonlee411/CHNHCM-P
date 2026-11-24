//go:build integration

package integration_test

import (
	"chnhcm.local/chnhcm/internal/bootstrap"
	"testing"
)

func TestDepartmentPreview133ScopedFactsAndReverseQualification(t *testing.T) {
	s := newDepartmentContract(t)
	s.create("PREVIEW_ROOT", "预览根", "BU_A", "", "2026-01-01", s.b, 201)
	s.create("PREVIEW_CHILD", "未来下级", "BU_A", "PREVIEW_ROOT", "2035-01-01", s.b, 201)
	unrelated := s.create("UNRELATED", "不相关部门", "BU_B", "", "2026-01-01", s.b, 201)
	detail, record := s.target("PREVIEW_ROOT", "PREVIEW_ROOT_FIRST")
	// Corrupt only an unrelated projection. The old whole-enterprise preview
	// would decode it. Required descendants and proposed parents remain loaded.
	s.f.editDepartment(unrelated["sourceVersion"].(string), func(v *bootstrap.DepartmentHistoryFact) { v.State.Summaries = nil })
	before := s.f.scalar(`SELECT (SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.department_history)+(SELECT count(*) FROM organization.department_terminal)`)
	path := "/api/v1/departments/" + s.aliases["PREVIEW_ROOT"] + "/preview"
	body := map[string]any{"action": "CORRECT", "expectedSourceVersion": detail["headSourceVersion"], "recordId": record["recordId"], "expectedRecordRevision": record["revision"], "changes": map[string]any{"name": "预览新名"}}
	v := s.b.call("POST", path, body, "", 200)
	if v["valid"] != true || v["noChange"] != false {
		t.Fatal("scoped preview failed")
	}
	for _, key := range []string{"capabilities", "additionContext", "lifecycleContext"} {
		if _, ok := v[key]; ok {
			t.Fatal("unused capability result still computed", key)
		}
	}
	body["changes"] = map[string]any{"name": "预览根"}
	v = s.b.call("POST", path, body, "", 200)
	if v["noChange"] != true {
		t.Fatal("unchanged valid input no longer recognized")
	}
	allDepartmentRights(s)
	s.actor.call("POST", path, body, "", 200)
	mutateLifeGrants(s.f, s.actor.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='department.record.correct'`, s.actor.subject)
	s.actor.call("POST", path, body, "", 403) // unchanged input still needs current write authority
	body["changes"] = map[string]any{"managementBuId": s.aliases["BU_B"]}
	s.b.call("POST", path, body, "", 409) // future child still needs the original BU
	body["changes"] = map[string]any{"parentDepartmentId": s.aliases["PREVIEW_CHILD"]}
	s.b.call("POST", path, body, "", 409) // proposed parent is a future descendant
	body["changes"] = map[string]any{"name": "预览根"}
	body["newEffectiveDate"] = "2036-01-01"
	s.b.call("POST", path, body, "", 409) // same BU/parent identities, but lost coverage for the future child
	delete(body, "newEffectiveDate")
	body["changes"] = map[string]any{"parentDepartmentId": "invalid"}
	s.b.call("POST", path, body, "", 400)
	child := s.current("PREVIEW_CHILD")
	s.f.editDepartment(child["headSourceVersion"].(string), func(v *bootstrap.DepartmentHistoryFact) { v.State.Summaries = nil })
	body["changes"] = map[string]any{"name": "名称预览"}
	s.b.call("POST", path, body, "", 200) // no dated relation change: child facts are not consumed
	add := map[string]any{"action": "ADD", "expectedSourceVersion": detail["headSourceVersion"], "anchorRecordId": record["recordId"], "effectiveDate": "2055-01-01", "operationType": "CHANGE", "operationReason": "NAME_ADJUSTMENT", "changes": map[string]any{"name": "未来名称"}}
	s.b.call("POST", path, add, "", 200) // the real automatic new-record assistance consumer
	body["changes"] = map[string]any{"managementBuId": s.aliases["BU_B"]}
	s.b.call("POST", path, body, "", 503) // changed relation must still consume the now-damaged child
	after := s.f.scalar(`SELECT (SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.department_history)+(SELECT count(*) FROM organization.department_terminal)`)
	if before != after {
		t.Fatal("preview created a source or terminal")
	}
	body["changes"] = map[string]any{"name": "预览根"}
	s.f.editDepartment(detail["headSourceVersion"].(string), func(v *bootstrap.DepartmentHistoryFact) { v.State.Summaries = nil })
	s.b.call("POST", path, body, "", 503) // a required damaged fact cannot be ignored
}

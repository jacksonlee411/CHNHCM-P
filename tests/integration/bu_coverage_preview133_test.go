//go:build integration

package integration_test

import (
	"encoding/json"
	"strings"
	"testing"

	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/foundation"
)

func coverageStopPreview(s *depFixture, alias, date string) (string, map[string]any) {
	path := "/api/v1/business-units/" + s.aliases[alias]
	detail := s.b.call("GET", path, nil, "", 200)
	rows := detail["timeline"].([]any)
	last := rows[len(rows)-1].(map[string]any)
	return path + "/preview", map[string]any{"action": "DEACTIVATE", "expectedSourceVersion": detail["headSourceVersion"], "anchorRecordId": last["recordId"], "effectiveDate": date, "operationType": "DEACTIVATE", "operationReason": "ORG_RESTRUCTURING"}
}

func TestBUCoveragePreview133FutureReferenceAndRequiredIntegrity(t *testing.T) {
	s := newDepartmentContract(t)
	child := s.create("FUTURE_REFERENCE", "未来部门", "BU_A", "", "2035-01-01", s.b, 201)
	unrelated := s.create("UNRELATED_REFERENCE", "无关部门", "BU_B", "", "2026-01-01", s.b, 201)
	s.f.editDepartment(unrelated["sourceVersion"].(string), func(v *bootstrap.DepartmentHistoryFact) { v.State.Summaries = nil })
	path, body := coverageStopPreview(s, "BU_A", "2036-01-01")
	count := func() int {
		return s.f.scalar(`SELECT (SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.bu_history)+(SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.department_history)+(SELECT count(*) FROM organization.bu_terminal)+(SELECT count(*) FROM organization.department_terminal)`)
	}
	before := count()
	v := s.b.call("POST", path, body, "", 409)
	if v["code"] != "M02_DEPARTMENT_REFERENCE_CONFLICT" {
		t.Fatal(v)
	}
	raw, _ := json.Marshal(v)
	if strings.Contains(string(raw), s.aliases["FUTURE_REFERENCE"]) || strings.Contains(string(raw), "未来部门") {
		t.Fatal("private relation identity disclosed")
	}
	s.actor.call("POST", path, body, "", 403)
	mutateLifeGrants(s.f, s.actor.subject, `INSERT INTO access.subject_grant VALUES($1,$2,$3,'bu',$4,'bu.deactivate',ARRAY[]::text[],ARRAY['effectiveDate','operationType','operationReason'])`, s.f.instance, foundation.NewID(), s.actor.subject, s.aliases["BU_A"])
	mutateLifeGrants(s.f, s.actor.subject, `DELETE FROM access.subject_grant WHERE instance_id=$1 AND principal_id=$2 AND action LIKE 'department.%'`, s.f.instance, s.actor.subject)
	v = s.actor.call("POST", path, body, "", 409)
	raw, _ = json.Marshal(v)
	if v["code"] != "M02_DEPARTMENT_REFERENCE_CONFLICT" || strings.Contains(string(raw), s.aliases["FUTURE_REFERENCE"]) || strings.Contains(string(raw), "未来部门") {
		t.Fatal("hidden dependency must give only a generic conflict", v)
	}
	mutateLifeGrants(s.f, s.actor.subject, `DELETE FROM access.subject_grant WHERE instance_id=$1 AND principal_id=$2 AND action='bu.deactivate'`, s.f.instance, s.actor.subject)
	s.actor.call("POST", path, body, "", 403)
	body["expectedSourceVersion"] = foundation.NewID()
	if v = s.b.call("POST", path, body, "", 409); v["code"] != "M02_SOURCE_CONFLICT" {
		t.Fatal(v)
	}
	path, body = coverageStopPreview(s, "BU_A", "2036-01-01")
	s.f.editDepartment(child["sourceVersion"].(string), func(v *bootstrap.DepartmentHistoryFact) { v.State.Summaries = nil })
	s.b.call("POST", path, body, "", 503)
	if before != count() {
		t.Fatal("preview changed business sources or terminals")
	}
}

func TestBUCoveragePreview133NewReferenceBeforeSave(t *testing.T) {
	s := newDepartmentContract(t)
	path, body := coverageStopPreview(s, "BU_A", "2035-01-01")
	if v := s.b.call("POST", path, body, "", 200); v["outcome"] != "accepted" {
		t.Fatal(v)
	}
	// The preview supplies no reusable permission. A newly committed reference
	// must still be checked under the ordinary write guards.
	s.create("AFTER_PREVIEW", "预览后新增的部门", "BU_A", "", "2036-01-01", s.b, 201)
	delete(body, "action")
	v := s.b.call("POST", strings.TrimSuffix(path, "/preview")+"/deactivate", body, foundation.NewID(), 409)
	if v["code"] != "M02_DEPARTMENT_REFERENCE_CONFLICT" {
		t.Fatal(v)
	}
	d := s.b.call("GET", strings.TrimSuffix(path, "/preview"), nil, "", 200)
	if d["headSourceVersion"] != body["expectedSourceVersion"] {
		t.Fatal("rejected save changed BU source")
	}
}

func TestBUCoveragePreview133FiniteReferencesAndExpansion(t *testing.T) {
	s := newDepartmentContract(t)
	s.create("FINITE_REFERENCE", "有截止日的部门", "BU_A", "", "2026-01-01", s.b, 201)
	s.append("FINITE_REFERENCE", "FINITE_STOP", "2035-01-01", "DEACTIVATE", nil, s.b, 201)
	path, body := coverageStopPreview(s, "BU_A", "2035-01-01")
	if v := s.b.call("POST", path, body, "", 200); v["outcome"] != "accepted" {
		t.Fatal(v)
	}
	body["effectiveDate"] = "2034-12-31"
	if v := s.b.call("POST", path, body, "", 409); v["code"] != "M02_DEPARTMENT_REFERENCE_CONFLICT" {
		t.Fatal(v)
	}
	s.stopBU("BU_A", "2035-01-01", 201)
	path, body = coverageStopPreview(s, "BU_A", "2036-01-01")
	body["action"], body["operationType"], body["operationReason"] = "REACTIVATE", "REACTIVATE", "BUSINESS_RESUMPTION"
	if v := s.b.call("POST", path, body, "", 200); v["outcome"] != "accepted" {
		t.Fatal(v)
	}
}

func TestBUCoveragePreview133FirstDateAndCrossingWindow(t *testing.T) {
	s := newDepartmentContract(t)
	s.create("CROSSING_REFERENCE", "跨入减少期间的部门", "BU_A", "", "2026-01-01", s.b, 201)
	path := "/api/v1/business-units/" + s.aliases["BU_A"]
	detail := s.b.call("GET", path, nil, "", 200)
	first := detail["timeline"].([]any)[0].(map[string]any)
	body := map[string]any{"action": "REDATE", "expectedSourceVersion": detail["headSourceVersion"], "recordId": first["recordId"], "expectedRecordRevision": first["revision"], "newEffectiveDate": "2027-01-01"}
	if v := s.b.call("POST", path+"/preview", body, "", 409); v["code"] != "M02_DEPARTMENT_REFERENCE_CONFLICT" {
		t.Fatal(v)
	}
	// Moving the first date within a period with no references is legal.
	path = "/api/v1/business-units/" + s.aliases["BU_B"]
	detail = s.b.call("GET", path, nil, "", 200)
	first = detail["timeline"].([]any)[0].(map[string]any)
	body["expectedSourceVersion"], body["recordId"], body["expectedRecordRevision"] = detail["headSourceVersion"], first["recordId"], first["revision"]
	if v := s.b.call("POST", path+"/preview", body, "", 200); v["outcome"] != "accepted" {
		t.Fatal(v)
	}
}

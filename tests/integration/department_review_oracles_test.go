//go:build integration

package integration_test

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"slices"
	"testing"
)

func TestDepartmentV21ReferenceClassificationOracles(t *testing.T) {
	s := newDepartmentContract(t)
	s.day = "2026-11-05"
	for _, p := range []struct{ alias, bu, date string }{{"P_JUNE", "BU_A", "2026-06-01"}, {"P_JULY", "BU_A", "2026-07-01"}, {"P_SHORT", "BU_A", "2026-06-01"}, {"P_B", "BU_B", "2026-06-01"}} {
		s.create(p.alias, p.alias, p.bu, "", p.date, s.b, 201)
	}
	s.append("P_SHORT", "P_SHORT_STOP", "2026-10-01", "DEACTIVATE", nil, s.b, 201)
	s.create("C", "部门", "BU_A", "", "2026-01-01", s.b, 201)
	s.append("C", "C_STOP", "2026-06-01", "DEACTIVATE", nil, s.b, 201)
	detail, r := s.target("C", "C_STOP")
	base := map[string]any{"mode": "maintain", "departmentId": s.aliases["C"], "expectedSourceVersion": detail["headSourceVersion"], "action": "CORRECT", "recordId": r["recordId"], "expectedRecordRevision": r["revision"]}
	raw, e := os.ReadFile("../../docs/verification/data/116-r4-review-fixes/review-case-oracles.json")
	must(t, e)
	var oracle struct {
		Cases []map[string]any `json:"cases"`
	}
	must(t, json.Unmarshal(raw, &oracle))
	var variants []any
	for _, c := range oracle.Cases {
		if c["id"] == "R4-V21" {
			variants = c["variants"].([]any)
		}
	}
	query := func(delta map[string]any, status int) map[string]any {
		ctx := map[string]any{}
		for k, v := range base {
			ctx[k] = v
		}
		for k, v := range delta {
			ctx[k] = v
		}
		return s.b.call("POST", "/api/v1/departments/references/parents", map[string]any{"context": ctx, "pageSize": 100}, "", status)
	}
	named := map[string][]string{}
	for _, value := range variants {
		variant := value.(map[string]any)
		name := variant["id"].(string)
		t.Run(name, func(t *testing.T) {
			s.f.t = t
			if name == "invalid_presence" {
				for _, bad := range variant["inputs"].([]any) {
					v := query(bad.(map[string]any), 400)
					if v["code"] != "M02_INVALID_INPUT" {
						t.Fatal("presence treated as different command", v)
					}
				}
				return
			}
			delta, _ := variant["input_delta"].(map[string]any)
			if changes, ok := delta["changes"].(map[string]any); ok {
				if alias, ok := changes["managementBuId"].(string); ok {
					changes["managementBuId"] = s.aliases[alias]
				}
			}
			result := query(delta, 200)
			ids := []string{}
			for _, v := range result["items"].([]any) {
				candidateID := v.(map[string]any)["id"].(string)
				if slices.Contains([]string{s.aliases["P_JUNE"], s.aliases["P_JULY"], s.aliases["P_SHORT"], s.aliases["P_B"]}, candidateID) {
					ids = append(ids, candidateID)
				}
			}
			named[name] = ids
			if name == "inherit" || name == "explicit_same" {
				if result["applicable"] != false || len(ids) != 0 {
					t.Fatal("inactive projection exposed selectable parents")
				}
				return
			}
			if name == "reason_not_selected" {
				if !reflect.DeepEqual(ids, named["change_june"]) {
					t.Fatal("unselected reason changed relation eligibility")
				}
				s.edit("C", "C_STOP", "correct", map[string]any{"classification": map[string]any{"operationType": "CHANGE"}}, s.b, 400)
				return
			}
			expected, ok := variant["expected"].(map[string]any)
			if ok {
				for _, key := range []string{"eligible_parents", "ineligible_parents"} {
					for _, alias := range expected[key].([]any) {
						got := slices.Contains(ids, s.aliases[alias.(string)])
						if got != (key == "eligible_parents") {
							t.Fatalf("%s candidate %s eligibility %v", name, alias, got)
						}
					}
				}
				// Compare each declared candidate with an independent full save from the same before state.
				for _, key := range []string{"eligible_parents", "ineligible_parents"} {
					for i, alias := range expected[key].([]any) {
						target := fmt.Sprintf("S_%s_%s_%d", name, key, i)
						s.create(target, "部门", "BU_A", "", "2026-01-01", s.b, 201)
						stopKey := target + "_STOP"
						s.append(target, stopKey, "2026-06-01", "DEACTIVATE", nil, s.b, 201)
						extra := map[string]any{}
						for k, v := range delta {
							extra[k] = v
						}
						changes := map[string]any{"parentDepartmentId": s.aliases[alias.(string)]}
						if ch, ok := delta["changes"].(map[string]any); ok {
							for k, v := range ch {
								changes[k] = v
							}
						}
						extra["changes"] = changes
						status := 200
						if key == "ineligible_parents" {
							status = 409
						}
						s.edit(target, stopKey, "correct", extra, s.b, status)
					}
				}
			}
		})
	}
	s.f.t = t
}
func TestDepartmentV22V23AuthorizationBeforeDiagnostics(t *testing.T) {
	s := newDepartmentContract(t)
	s.create("C", "部门", "BU_A", "", "2026-01-01", s.b, 201)
	s.create("P", "候选上级", "BU_B", "", "2026-01-01", s.b, 201)
	s.profile("maintain_A")
	beforeTerm := s.f.scalar("SELECT (SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.department_history)+(SELECT count(*) FROM organization.department_terminal)")
	var first map[string]any
	for i, target := range []string{foundation.NewID(), s.aliases["BU_B"], s.aliases["BU_B"]} {
		if i == 2 {
			s.append("P", "P_STOP", "2026-06-01", "DEACTIVATE", nil, s.b, 201)
			s.stopBU("BU_B", "2026-06-01", 201)
			beforeTerm = s.f.scalar("SELECT (SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.department_history)+(SELECT count(*) FROM organization.department_terminal)")
		}
		v := s.edit("C", "C_FIRST", "correct", map[string]any{"changes": map[string]any{"managementBuId": target, "parentDepartmentId": s.aliases["P"]}}, s.actor, 403)
		if first == nil {
			first = v
		} else if !reflect.DeepEqual(first, v) {
			t.Fatal("unauthorized target state changed error shape", first, v)
		}
		if s.f.scalar("SELECT (SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.department_history)+(SELECT count(*) FROM organization.department_terminal)") != beforeTerm {
			t.Fatal("authorization denial stored business terminal")
		}
	}
	s.profile("maintain_AB")
	mutateLifeGrants(s.f, s.actor.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='department.reference.read'`, s.actor.subject)
	first = nil
	for _, parent := range []string{foundation.NewID(), s.aliases["P"]} {
		v := s.edit("C", "C_FIRST", "correct", map[string]any{"changes": map[string]any{"managementBuId": s.aliases["BU_B"], "parentDepartmentId": parent}}, s.actor, 404)
		if first == nil {
			first = v
		} else if !reflect.DeepEqual(first, v) {
			t.Fatal("hidden parent existence exposed", first, v)
		}
	}
	// Both correct and stale expectations are hidden before version diagnostics.
	detail, r := s.target("C", "C_FIRST")
	s.profile("read_B")
	for _, source := range []string{detail["headSourceVersion"].(string), foundation.NewID()} {
		s.actor.call("POST", "/api/v1/departments/"+s.aliases["C"]+"/records/"+r["recordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": source, "expectedRecordRevision": r["revision"], "changes": map[string]any{"name": "更名"}}, foundation.NewID(), 404)
	}
	s.profile("read_A")
	for _, source := range []string{detail["headSourceVersion"].(string), foundation.NewID()} {
		s.actor.call("POST", "/api/v1/departments/"+s.aliases["C"]+"/records/"+r["recordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": source, "expectedRecordRevision": r["revision"], "changes": map[string]any{"name": "更名"}}, foundation.NewID(), 403)
	}
	s.profile("maintain_AB")
	for _, field := range []string{"expectedSourceVersion", "expectedRecordRevision"} {
		body := map[string]any{"expectedSourceVersion": detail["headSourceVersion"], "expectedRecordRevision": r["revision"], "changes": map[string]any{"name": "更名"}}
		body[field] = foundation.NewID()
		v := s.actor.call("POST", "/api/v1/departments/"+s.aliases["C"]+"/records/"+r["recordId"].(string)+"/correct", body, foundation.NewID(), 409)
		expected := "M02_SOURCE_CONFLICT"
		if field == "expectedRecordRevision" {
			expected = "M02_REVISION_CONFLICT"
		}
		if v["code"] != expected {
			t.Fatalf("actual conflict lost: %v", v)
		}
	}
}
func TestDepartmentV24ImplicitRollbackBeforeQualification(t *testing.T) {
	for _, stopBU := range []bool{false, true} {
		t.Run(fmt.Sprintf("old_B_stopped_%v", stopBU), func(t *testing.T) {
			s := newDepartmentContract(t)
			s.create("C", "部门", "BU_B", "", "2026-01-01", s.b, 201)
			s.append("C", "C_MOVE", "2026-07-01", "CHANGE", map[string]any{"managementBuId": s.aliases["BU_A"]}, s.b, 201)
			if stopBU {
				s.stopBU("BU_B", "2026-08-01", 201)
			}
			s.profile("maintain_A")
			for _, action := range []string{"delete", "redate"} {
				detail, r := s.target("C", "C_MOVE")
				extra := map[string]any{}
				if action == "redate" {
					extra["newEffectiveDate"] = "2026-12-01"
				}
				before := s.businessDigest()
				count := s.f.scalar("SELECT (SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.department_history)+(SELECT count(*) FROM organization.department_terminal)")
				s.edit("C", "C_MOVE", action, extra, s.actor, 403)
				s.assertNoBusinessChange(before)
				if s.f.scalar("SELECT (SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.department_history)+(SELECT count(*) FROM organization.department_terminal)") != count {
					t.Fatal("implicit permission denial made terminal")
				}
				preview := map[string]any{"action": map[string]string{"delete": "DELETE", "redate": "REDATE"}[action], "expectedSourceVersion": detail["headSourceVersion"], "recordId": r["recordId"], "expectedRecordRevision": r["revision"]}
				for k, v := range extra {
					preview[k] = v
				}
				s.actor.call("POST", "/api/v1/departments/"+s.aliases["C"]+"/preview", preview, "", 403)
			}
		})
	}
}

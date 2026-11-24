//go:build integration

package integration_test

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"fmt"
	"testing"
)

func TestDepartmentParent131WindowsAndInvariantFailure(t *testing.T) {
	s := newDepartmentContract(t)
	s.create("P", "窗口前生效上级", "BU_A", "", "2026-01-01", s.b, 201)
	s.append("P", "P_STOP", "2099-01-01", "DEACTIVATE", nil, s.b, 201)
	s.create("C", "候选目标", "BU_A", "", "2026-01-01", s.b, 201)
	s.append("C", "C_STOP", "2027-01-01", "DEACTIVATE", nil, s.b, 201)
	detail, r := s.target("C", "C_FIRST")
	query := map[string]any{"context": map[string]any{"mode": "maintain", "departmentId": s.aliases["C"], "expectedSourceVersion": detail["headSourceVersion"], "action": "CORRECT", "recordId": r["recordId"], "expectedRecordRevision": r["revision"]}, "pageSize": 20}
	read := func(status int) map[string]any {
		return s.b.call("POST", "/api/v1/departments/references/parents", query, "", status)
	}
	v := read(200)
	if len(v["items"].([]any)) != 1 || v["items"].([]any)[0].(map[string]any)["id"] != s.aliases["P"] {
		t.Fatal("closed window incorrectly includes distant stop", v)
	}
	s.edit("C", "C_FIRST", "correct", map[string]any{"changes": map[string]any{"parentDepartmentId": s.aliases["P"]}}, s.b, 200)
	// An open create window must see the registered 2099 stop.
	query["context"] = map[string]any{"mode": "create", "effectiveDate": "2026-01-01", "managementBuId": s.aliases["BU_A"]}
	v = read(200)
	if len(v["items"].([]any)) != 0 {
		t.Fatal("open window dropped future stops", v)
	}
	// An unchanged child makes the common candidate invalid independently of the chosen parent.
	s.create("T", "共用部分目标", "BU_A", "", "2026-01-01", s.b, 201)
	s.create("CHILD", "反向下级", "BU_A", "T", "2026-01-01", s.b, 201)
	detail, r = s.target("T", "T_FIRST")
	query["context"] = map[string]any{"mode": "maintain", "departmentId": s.aliases["T"], "expectedSourceVersion": detail["headSourceVersion"], "action": "CORRECT", "recordId": r["recordId"], "expectedRecordRevision": r["revision"], "changes": map[string]any{"managementBuId": s.aliases["BU_B"]}}
	v = read(409)
	if v["count"] != nil || v["items"] != nil {
		t.Fatal("invalid invariant reported as successful empty candidates", v)
	}
}

func TestDepartmentParent131CurrentContextAndCandidateSave(t *testing.T) {
	for _, change := range []string{"stop", "transfer", "revoke", "future_cycle"} {
		t.Run(change, func(t *testing.T) {
			s := newDepartmentContract(t)
			s.create("P", "可选上级", "BU_A", "", "2026-01-01", s.b, 201)
			s.create("C", "候选目标", "BU_A", "", "2026-01-01", s.b, 201)
			before, r := s.target("C", "C_FIRST")
			query := map[string]any{"context": map[string]any{"mode": "maintain", "departmentId": s.aliases["C"], "expectedSourceVersion": before["headSourceVersion"], "action": "CORRECT", "recordId": r["recordId"], "expectedRecordRevision": r["revision"]}, "pageSize": 20}
			got := s.b.call("POST", "/api/v1/departments/references/parents", query, "", 200)
			if len(got["items"].([]any)) != 1 {
				t.Fatal("nonempty candidate prerequisite", got)
			}
			status := 409
			switch change {
			case "stop":
				s.append("P", "P_STOP", "2099-01-01", "DEACTIVATE", nil, s.b, 201)
			case "transfer":
				s.append("P", "P_TRANSFER", "2027-01-01", "CHANGE", map[string]any{"managementBuId": s.aliases["BU_B"]}, s.b, 201)
			case "revoke":
				mutateLifeGrants(s.f, s.b.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='department.reference.read'`, s.b.subject)
				status = 404
			case "future_cycle":
				s.append("P", "P_FUTURE", "2099-01-01", "CHANGE", map[string]any{"parentDepartmentId": s.aliases["C"]}, s.b, 201)
			}
			sources := s.f.scalar(`SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.department_history`)
			s.edit("C", "C_FIRST", "correct", map[string]any{"changes": map[string]any{"parentDepartmentId": s.aliases["P"]}}, s.b, status)
			after, _ := s.target("C", "C_FIRST")
			if after["headSourceVersion"] != before["headSourceVersion"] || s.f.scalar(`SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.department_history`) != sources {
				t.Fatal("rejected candidate save left business source")
			}
		})
	}
	s := newDepartmentContract(t)
	query := map[string]any{"context": map[string]any{"mode": "create", "effectiveDate": "2026-01-01", "managementBuId": foundation.NewID()}, "pageSize": 20}
	v := s.b.call("POST", "/api/v1/departments/references/parents", query, "", 404)
	if v["code"] != "M02_OBJECT_NOT_FOUND" {
		t.Fatal("unknown BU misclassified", v)
	}
}

func TestDepartmentParent131CandidateSaveBothOrders(t *testing.T) {
	for _, change := range []string{"stop", "transfer"} {
		for _, first := range []int{0, 1} {
			t.Run(fmt.Sprintf("%s_first_%d", change, first), func(t *testing.T) {
				s := newDepartmentContract(t)
				s.create("P", "候选上级", "BU_A", "", "2026-01-01", s.b, 201)
				s.create("C", "原独立叶子", "BU_A", "", "2026-01-01", s.b, 201)
				child, cr := s.target("C", "C_FIRST")
				parent, pr := s.target("P", "P_FIRST")
				query := map[string]any{"context": map[string]any{"mode": "maintain", "departmentId": s.aliases["C"], "expectedSourceVersion": child["headSourceVersion"], "action": "CORRECT", "recordId": cr["recordId"], "expectedRecordRevision": cr["revision"]}, "pageSize": 20}
				candidates := s.b.call("POST", "/api/v1/departments/references/parents", query, "", 200)
				if len(candidates["items"].([]any)) != 1 {
					t.Fatal("candidate prerequisite", candidates)
				}
				paths := []string{"/api/v1/departments/" + s.aliases["C"] + "/records/" + cr["recordId"].(string) + "/correct", "/api/v1/departments/" + s.aliases["P"] + "/deactivate"}
				bodies := []map[string]any{{"expectedSourceVersion": child["headSourceVersion"], "expectedRecordRevision": cr["revision"], "changes": map[string]any{"parentDepartmentId": s.aliases["P"]}}, {"expectedSourceVersion": parent["headSourceVersion"], "anchorRecordId": pr["recordId"], "effectiveDate": "2027-01-01", "operationType": "DEACTIVATE", "operationReason": "ORG_RESTRUCTURING"}}
				if change == "transfer" {
					paths[1] = "/api/v1/departments/" + s.aliases["P"] + "/records"
					bodies[1]["operationType"] = "CHANGE"
					bodies[1]["operationReason"] = "ORGANIZATION_REASSIGNMENT"
					bodies[1]["changes"] = map[string]any{"managementBuId": s.aliases["BU_B"]}
				}
				before := s.f.scalar(`SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.department_history`)
				tx, e := s.f.admin.Begin(s.f.ctx)
				must(t, e)
				defer tx.Rollback(s.f.ctx)
				_, e = tx.Exec(s.f.ctx, `SELECT department_id FROM organization.department WHERE instance_id=$1 AND department_id=$2 FOR UPDATE`, s.f.instance, s.aliases["P"])
				must(t, e)
				clients := []*browser{s.b, s.f.login("admin", s.f.password)}
				channels := []chan depAnswer{make(chan depAnswer, 1), make(chan depAnswer, 1)}
				start := func(index int) {
					go func() {
						status, value, e := clients[index].response("POST", paths[index], bodies[index], foundation.NewID())
						channels[index] <- depAnswer{status, value, e}
					}()
				}
				start(first)
				waitDepartmentLocks(t, s.f, 1)
				start(1 - first)
				waitDepartmentLocks(t, s.f, 2)
				must(t, tx.Rollback(s.f.ctx))
				replies := []depAnswer{<-channels[0], <-channels[1]}
				wins := 0
				for _, r := range replies {
					must(t, r.err)
					if r.status == 200 || r.status == 201 {
						wins++
					} else if r.status != 409 {
						t.Fatal("unexpected race failure", r)
					}
				}
				if wins != 1 || s.f.scalar(`SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.department_history`) != before+1 {
					t.Fatal("incompatible saves both accepted or residual business source", replies)
				}
				t.Logf("actual queued order %d then %d, child=%d parent=%d", first, 1-first, replies[0].status, replies[1].status)
			})
		}
	}
}

func TestDepartmentParent131ExtremeFirstDates(t *testing.T) {
	f := fresh(t)
	g := t07Instance(t, f, "0001-01-01")
	b := g.login("admin", g.password)
	bu := b.call("POST", "/api/v1/business-units", createBody("P131_EXTREME_BU", "0001-01-01"), foundation.NewID(), 201)["buId"].(string)
	for i, day := range []string{"0001-01-01", "9999-12-31"} {
		saved := b.call("POST", "/api/v1/departments", departmentBody(fmt.Sprintf("P131_EXTREME_%d", i), "真实首条", bu, day, nil), foundation.NewID(), 201)
		query := map[string]any{"context": map[string]any{"mode": "maintain", "departmentId": saved["departmentId"], "expectedSourceVersion": saved["sourceVersion"], "action": "CORRECT", "recordId": saved["targetRecordId"], "expectedRecordRevision": saved["targetRevision"]}, "pageSize": 20}
		out := b.call("POST", "/api/v1/departments/references/parents", query, "", 200)
		if len(out["items"].([]any)) != i {
			t.Fatal("extreme first date shifted or excluded", out)
		}
	}
}

func TestDepartmentParent131VisibleSearchAndReadOnlyMaintain(t *testing.T) {
	s := newDepartmentContract(t)
	s.create("P", "ÉCOLE_%候选", "BU_A", "", "2026-01-01", s.b, 201)
	s.create("C", "目标", "BU_A", "", "2026-01-01", s.b, 201)
	before, r := s.target("C", "C_FIRST")
	query := map[string]any{"context": map[string]any{"mode": "maintain", "departmentId": s.aliases["C"], "expectedSourceVersion": before["headSourceVersion"], "action": "CORRECT", "recordId": r["recordId"], "expectedRecordRevision": r["revision"]}, "q": "école_%", "pageSize": 20}
	for _, actor := range []*browser{s.b, s.actor} {
		v := actor.call("POST", "/api/v1/departments/references/parents", query, "", 200)
		if len(v["items"].([]any)) != 1 {
			t.Fatal("current visible Unicode/literal label missed", v)
		}
	}
	mutateLifeGrants(s.f, s.actor.subject, `UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'name') WHERE principal_id=$1 AND action='department.reference.read'`, s.actor.subject)
	v := s.actor.call("POST", "/api/v1/departments/references/parents", query, "", 200)
	if len(v["items"].([]any)) != 0 {
		t.Fatal("hidden label influenced candidate search", v)
	}
	// Unsearched ordering must not restore hidden names when every code is
	// readable, nor leak codes after switching to name-only reference rights.
	delete(query, "q")
	v = s.actor.call("POST", "/api/v1/departments/references/parents", query, "", 200)
	items := v["items"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["id"] != s.aliases["P"] || items[0].(map[string]any)["name"] != nil || items[0].(map[string]any)["code"] == nil {
		t.Fatal("unsearched code-only reference exposed a hidden name", v)
	}
	mutateLifeGrants(s.f, s.actor.subject, `UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'code') || ARRAY['name']::text[] WHERE principal_id=$1 AND action='department.reference.read'`, s.actor.subject)
	v = s.actor.call("POST", "/api/v1/departments/references/parents", query, "", 200)
	items = v["items"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["id"] != s.aliases["P"] || items[0].(map[string]any)["code"] != nil || items[0].(map[string]any)["name"] != "ÉCOLE_%候选" {
		t.Fatal("unsearched name-only reference exposed a hidden code", v)
	}
	query["context"] = map[string]any{"mode": "create", "effectiveDate": "2026-01-01", "managementBuId": s.aliases["BU_A"]}
	delete(query, "q")
	v = s.actor.call("POST", "/api/v1/departments/references/parents", query, "", 200)
	if len(v["items"].([]any)) != 0 {
		t.Fatal("read-only account obtained create choices", v)
	}
}

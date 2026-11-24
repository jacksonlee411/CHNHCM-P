//go:build integration

package integration_test

import (
	"reflect"
	"strings"
	"testing"

	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/foundation"
)

func TestDepartment133RenameKeepsHistoryAndRequiredQualification(t *testing.T) {
	s := newDepartmentContract(t)
	root := s.create("ROOT133", "上级原名", "BU_A", "", "2026-01-01", s.b, 201)
	child := s.create("CHILD133", "下级", "BU_A", "ROOT133", "2026-01-01", s.b, 201)
	s.create("FUTURE133", "未来下级", "BU_A", "ROOT133", "2035-01-01", s.b, 201)
	id, record := s.aliases["ROOT133"], s.records["ROOT133_FIRST"]
	before := s.current("ROOT133")
	oldRelation := s.f.departmentHead(id).State.Timeline.RelationSource
	result := s.b.departmentCorrect(id, record, map[string]any{"changes": map[string]any{"name": "上级现名"}}, 200)
	if result["outcome"] != "accepted" || s.f.departmentHead(id).State.Timeline.RelationSource != oldRelation {
		t.Fatal("name correction changed relation identity")
	}
	old := s.b.call("GET", "/api/v1/departments/"+id+"?sourceVersion="+root["sourceVersion"].(string), nil, "", 200)
	if !reflect.DeepEqual(old["timeline"], before["timeline"]) {
		t.Fatal("original source changed")
	}
	current := s.current("CHILD133")
	frozen := s.b.call("GET", "/api/v1/departments/"+s.aliases["CHILD133"]+"?sourceVersion="+child["sourceVersion"].(string), nil, "", 200)
	if current["headSourceVersion"] != child["sourceVersion"] || !strings.Contains(current["path"].(map[string]any)["departmentPath"].(string), "上级现名") || !strings.Contains(frozen["path"].(map[string]any)["departmentPath"].(string), "上级原名") {
		t.Fatal("current and frozen child paths were not kept distinct")
	}
	for _, change := range []map[string]any{
		{"newEffectiveDate": "2036-01-01", "changes": map[string]any{"name": "改日新名"}},
		{"changes": map[string]any{"managementBuId": s.aliases["BU_B"]}},
		{"changes": map[string]any{"parentDepartmentId": s.aliases["CHILD133"]}},
	} {
		s.b.departmentCorrect(id, record, change, 409)
	}
	s.append("ROOT133", "STOP133", "2027-01-01", "DEACTIVATE", nil, s.b, 409)
	allDepartmentRights(s)
	mutateLifeGrants(s.f, s.actor.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='department.record.correct'`, s.actor.subject)
	s.actor.departmentCorrect(id, record, map[string]any{"changes": map[string]any{"name": "上级现名"}}, 403)
	// Ordinary rename must not consume a descendant. Changing effective coverage
	// must still consume it; required original facts must never be skipped.
	s.f.editDepartment(child["sourceVersion"].(string), func(v *bootstrap.DepartmentHistoryFact) { v.State.Summaries = nil })
	s.b.departmentCorrect(id, record, map[string]any{"changes": map[string]any{"name": "上级再次更名"}}, 200)
	s.b.departmentCorrect(id, record, map[string]any{"changes": map[string]any{"managementBuId": s.aliases["BU_B"]}}, 503)
	latest := s.current("ROOT133")
	s.f.editDepartment(latest["headSourceVersion"].(string), func(v *bootstrap.DepartmentHistoryFact) { v.State.Summaries = nil })
	s.b.call("POST", "/api/v1/departments/"+id+"/records/"+record+"/correct", map[string]any{"expectedSourceVersion": latest["headSourceVersion"], "expectedRecordRevision": latest["timeline"].([]any)[0].(map[string]any)["revision"], "changes": map[string]any{"name": "不能忽略原件"}}, foundation.NewID(), 503)
}

func TestDepartment133RenameAndIncomingChildBothOrders(t *testing.T) {
	for _, first := range []int{0, 1} {
		t.Run([]string{"rename_first", "child_first"}[first], func(t *testing.T) {
			s := newDepartmentContract(t)
			s.create("ROOT133", "上级原名", "BU_A", "", "2026-01-01", s.b, 201)
			s.create("EXISTING133", "已有下级", "BU_A", "ROOT133", "2026-01-01", s.b, 201)
			detail, record := s.target("ROOT133", "ROOT133_FIRST")
			id := s.aliases["ROOT133"]
			tx, e := s.f.admin.Begin(s.f.ctx)
			must(t, e)
			defer tx.Rollback(s.f.ctx)
			_, e = tx.Exec(s.f.ctx, `SELECT FROM organization.department WHERE department_id=$1 FOR UPDATE`, id)
			must(t, e)
			clients := []*browser{s.b, s.f.login("admin", s.f.password)}
			answers := []chan depAnswer{make(chan depAnswer, 1), make(chan depAnswer, 1)}
			start := func(index int) {
				go func() {
					path := "/api/v1/departments"
					body := departmentBody("NEW133", "新下级", s.aliases["BU_A"], "2026-01-01", id)
					if index == 0 {
						path += "/" + id + "/records/" + record["recordId"].(string) + "/correct"
						body = map[string]any{"expectedSourceVersion": detail["headSourceVersion"], "expectedRecordRevision": record["revision"], "changes": map[string]any{"name": "上级现名"}}
					}
					status, value, err := clients[index].response("POST", path, body, foundation.NewID())
					answers[index] <- depAnswer{status, value, err}
				}()
			}
			start(first)
			waitDepartmentLocks(t, s.f, 1)
			start(1 - first)
			waitDepartmentLocks(t, s.f, 2)
			must(t, tx.Rollback(s.f.ctx))
			rename, created := <-answers[0], <-answers[1]
			must(t, rename.err)
			must(t, created.err)
			if rename.status != 200 || created.status != 201 {
				t.Fatalf("rename/child failed: %d %v / %d %v", rename.status, rename.body, created.status, created.body)
			}
			childID := created.body["departmentId"].(string)
			current := s.b.departmentCurrent(childID)
			old := s.b.call("GET", "/api/v1/departments/"+childID+"?sourceVersion="+created.body["sourceVersion"].(string), nil, "", 200)
			frozenName := "上级现名"
			if first == 1 {
				frozenName = "上级原名"
			}
			if !strings.Contains(current["path"].(map[string]any)["departmentPath"].(string), "上级现名") || !strings.Contains(old["path"].(map[string]any)["departmentPath"].(string), frozenName) {
				t.Fatal("incoming edge did not serialize its frozen path with rename")
			}
		})
	}
}

func TestDepartment133QueuedRenameRechecksSource(t *testing.T) {
	s := newDepartmentContract(t)
	s.create("ROOT133", "上级原名", "BU_A", "", "2026-01-01", s.b, 201)
	s.create("CHILD133", "下级", "BU_A", "ROOT133", "2026-01-01", s.b, 201)
	detail, record := s.target("ROOT133", "ROOT133_FIRST")
	id := s.aliases["ROOT133"]
	tx, e := s.f.admin.Begin(s.f.ctx)
	must(t, e)
	defer tx.Rollback(s.f.ctx)
	_, e = tx.Exec(s.f.ctx, `SELECT FROM organization.department WHERE department_id=$1 FOR UPDATE`, id)
	must(t, e)
	answers := make(chan depAnswer, 2)
	for index, client := range []*browser{s.b, s.f.login("admin", s.f.password)} {
		name := []string{"更名甲", "更名乙"}[index]
		go func() {
			status, value, err := client.response("POST", "/api/v1/departments/"+id+"/records/"+record["recordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": detail["headSourceVersion"], "expectedRecordRevision": record["revision"], "changes": map[string]any{"name": name}}, foundation.NewID())
			answers <- depAnswer{status, value, err}
		}()
	}
	waitDepartmentLocks(t, s.f, 2)
	must(t, tx.Rollback(s.f.ctx))
	a, b := <-answers, <-answers
	must(t, a.err)
	must(t, b.err)
	if a.status == 409 {
		a, b = b, a
	}
	if a.status != 200 || b.status != 409 || b.body["code"] != "M02_SOURCE_CONFLICT" {
		t.Fatalf("stale source accepted: %d %v / %d %v", a.status, a.body, b.status, b.body)
	}
}

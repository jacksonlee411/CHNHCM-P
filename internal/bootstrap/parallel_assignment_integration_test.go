//go:build integration

package bootstrap_test

import (
	f "chnhcm.local/chnhcm/internal/foundation"
	"maps"
	"testing"
)

func TestPersonOwnsConcurrentAssignmentNumbersAcrossRelationships(t *testing.T) {
	type M = map[string]any
	x := setupAssignment(t)
	b := x.h.b
	id := x.hire["assignment"].(string)
	base := "/api/v1/assignments/" + id
	source := b.call("GET", base+"?businessAt=2027-04-03", nil, "", 200)["sourceVersion"]
	change := M{"from": "2027-04-03", "expectedSourceVersion": source, "job": x.job, "department": x.dep, "notice": "内部评审任命", "parallel": false}
	reg := func(dept string) M {
		return b.call("POST", "/api/v1/assignments/query", M{"businessAt": "2027-04-06", "department": dept, "person": x.person, "page": 1, "pageSize": 20}, "", 200)
	}
	policy := b.call("GET", "/api/v1/assignment-policies", nil, "", 200)
	b.call("POST", "/api/v1/assignment-policies", M{"expectedVersion": policy["version"], "row": M{"id": f.NewID(), "category": "Employee", "purpose": "secondary", "mode": "JobOnly", "from": "2027-04-01"}}, f.NewID(), 200)
	key := f.NewID()
	write := base + "/additional-assignments"
	one := b.call("POST", write, change, key, 200)
	if one["recordNumber"] != 1.0 || one["employment"] != x.hire["employment"] {
		t.Fatal("同关系的第一份兼岗", one)
	}
	retry := b.call("POST", write, change, key, 200)
	if retry["action"] != one["action"] || retry["recordNumber"] != 1.0 {
		t.Fatal("重送变号", retry)
	}
	dept := b.call("POST", "/api/v1/departments", departmentCreateBody("D45", x.bu), f.NewID(), 201)["departmentId"].(string)
	other := x.h.create("LAW45", "CN", "legal_person")["id"].(string)
	b.call("POST", "/api/v1/legal-entities/"+other+"/purposes", M{"expectedVersion": "", "row": M{"id": f.NewID(), "role": "employer", "kind": "declaration", "basis": "独立聘任资格", "from": "2027-04-01"}}, f.NewID(), 200)
	second := maps.Clone(change)
	second["parallel"] = true
	second["employer"] = other
	second["department"] = dept
	second["notice"] = "培训单位独立聘任"
	two := b.call("POST", write, second, f.NewID(), 200)
	if two["recordNumber"] != 2.0 || two["employment"] == one["employment"] {
		t.Fatal("独立关系仍按人编号", two)
	}
	for _, r := range []struct {
		result  M
		primary bool
	}{{one, false}, {two, true}} {
		v := b.call("GET", "/api/v1/assignments/"+r.result["assignment"].(string)+"?businessAt=2027-04-06", nil, "", 200)
		if v["relationshipPrimary"] != r.primary || v["primaryJob"] != false {
			t.Fatal("人层主岗和关系主任职混淆", v)
		}
	}
	if a := reg(x.dep); a["total"] != 2.0 || a["people"] != 1.0 {
		t.Fatal(a)
	}
	if a := reg(dept); a["total"] != 1.0 || a["people"] != 1.0 {
		t.Fatal(a)
	}
	x.h.f.exec("ALTER TABLE staffing.employment_outbox ADD CONSTRAINT test_no_parallel_event CHECK(false) NOT VALID")
	b.call("POST", write, change, f.NewID(), 503)
	x.h.f.exec("ALTER TABLE staffing.employment_outbox DROP CONSTRAINT test_no_parallel_event")
	type result struct {
		status int
		body   M
		err    error
	}
	ch := make(chan result, 2)
	for _, notice := range []string{"其他兼岗甲", "其他兼岗乙"} {
		in := maps.Clone(change)
		in["notice"] = notice
		go func() { status, body, err := b.response("POST", write, in, f.NewID()); ch <- result{status, body, err} }()
	}
	numbers := map[float64]bool{}
	for range 2 {
		r := <-ch
		must(t, r.err)
		if r.status != 200 {
			t.Fatalf("并发办理 %d %v", r.status, r.body)
		}
		numbers[r.body["recordNumber"].(float64)] = true
	}
	if !numbers[3] || !numbers[4] {
		t.Fatal("回滚或并发破坏永久编号", numbers)
	}
	b.call("POST", "/api/v1/employment-actions/"+x.hire["action"].(string)+"/cancel", M{"revision": x.hire["revision"], "reason": "检查在任主岗覆盖"}, f.NewID(), 409)
	x.h.policy("UPDATE access.employment_grant SET department_id=$2 WHERE principal_id=$1 AND action IN ('read','history','directory','register.read')", b.subject, dept)
	if a := reg(""); a["total"] != 1.0 || a["people"] != 1.0 || a["rows"].([]any)[0].(M)["recordNumber"] != 2.0 {
		t.Fatal("仅见兼岗", a)
	}
	b.call("GET", base+"?businessAt=2027-04-06", nil, "", 403)
	view := b.call("GET", "/api/v1/persons/"+x.person+"/employment?businessAt=2027-04-06", nil, "", 200)
	if len(view["terms"].([]any)) != 1 || view["assignments"] != 1.0 {
		t.Fatal("仅见一份兼岗的人员上下文", view)
	}
}

//go:build integration

package bootstrap_test

import (
	f "chnhcm.local/chnhcm/internal/foundation"
	"maps"
	"os"
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
	type result struct {
		status int
		body   M
		err    error
	}
	ch := make(chan result, 2)
	for _, submission := range []string{key, f.NewID()} {
		go func() {
			status, body, err := b.response("POST", write, change, submission)
			ch <- result{status, body, err}
		}()
	}
	var one M
	for range 2 {
		r := <-ch
		must(t, r.err)
		if r.status != 200 || (one != nil && r.body["action"] != one["action"]) {
			t.Fatal("同案同时提交", r)
		}
		one = r.body
	}
	if one["recordNumber"] != 1.0 || one["employment"] != x.hire["employment"] {
		t.Fatal("同关系的第一份兼岗", one)
	}
	retry := b.call("POST", write, change, key, 200)
	if retry["action"] != one["action"] || retry["recordNumber"] != 1.0 {
		t.Fatal("重送变号", retry)
	}
	peerPath := "/api/v1/assignments/" + one["assignment"].(string)
	fromPeer := maps.Clone(change)
	fromPeer["expectedSourceVersion"] = b.call("GET", peerPath+"?businessAt=2027-04-03", nil, "", 200)["sourceVersion"]
	if replay := b.call("POST", peerPath+"/additional-assignments", fromPeer, f.NewID(), 200); replay["action"] != one["action"] {
		t.Fatal("从同关系另一任职应找到原件", replay)
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
	// The ordinary directory also includes people without a readable term and
	// selects each record number independently, even before the earliest hire.
	bare := b.call("POST", "/api/v1/persons", M{"submissionId": f.NewID(), "name": "尚无任职的检索样本"}, "", 200)["id"]
	directory := func(filters M) M {
		query := M{"page": 1, "pageSize": 20, "sort": "number"}
		maps.Copy(query, filters)
		return b.call("POST", "/api/v1/personnel/directory/query", query, "", 200)
	}
	current := directory(M{"anchor": bare})
	if current["assignments"] != 3.0 || current["people"] != 2.0 || current["total"] != 4.0 {
		t.Fatal("名录遗漏人员或未来任职", current)
	}
	if first := current["rows"].([]any)[0].(M); first["assignment"].(M)["recordNumber"] != 0.0 || first["id"] != x.hire["assignment"] {
		t.Fatal("主岗锚点", first)
	}
	bareRow := current["rows"].([]any)[3].(M)
	if bareRow["id"] != bare || bareRow["assignment"] != nil || current["navigation"].(M)["previous"].(M)["assignment"] != two["assignment"] {
		t.Fatal("人员行及上一个任职", current)
	}
	secondary := directory(M{"primary": "secondary", "department": x.dep})
	if secondary["total"] != 1.0 || secondary["rows"].([]any)[0].(M)["id"] != one["assignment"] {
		t.Fatal("按部门及主兼岗过滤", secondary)
	}
	b.call("POST", "/api/v1/personnel/directory/query", M{"page": 1, "pageSize": 20, "businessAt": "2027-04-01"}, "", 400)
	if os.Getenv("CHNHCM_PERSONNEL_CAPACITY") == "1" {
		measureMixedDirectory(t, x, x.hire["assignment"].(string), one["assignment"].(string), bare.(string))
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
	same := maps.Clone(change)
	same["notice"] = "换写通知仍是同一任职"
	if got := b.call("POST", write, same, f.NewID(), 200); got["action"] != one["action"] || got["revision"] != one["revision"] {
		t.Fatal(got)
	}
	newWork := maps.Clone(change)
	newWork["from"] = "2027-04-04"
	b.call("POST", write, newWork, f.NewID(), 503)
	x.h.f.exec("ALTER TABLE staffing.employment_outbox DROP CONSTRAINT test_no_parallel_event")
	for _, start := range []string{"2027-04-04", "2027-04-05"} {
		in := maps.Clone(change)
		in["from"] = start
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

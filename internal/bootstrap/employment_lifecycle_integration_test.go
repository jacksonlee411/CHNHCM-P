//go:build integration

package bootstrap_test

import (
	f "chnhcm.local/chnhcm/internal/foundation"
	"testing"
)

func TestActualManagersConcurrentAndExit(t *testing.T) {
	type M = map[string]any
	x := setupAssignment(t)
	b := x.h.b
	person := b.call("POST", "/api/v1/persons", M{"submissionId": f.NewID(), "name": "实际经理林文", "primaryDocument": M{"kind": "passport", "number": "SYN-MANAGER48"}}, "", 200)["id"].(string)
	other := b.call("POST", "/api/v1/hires", M{"person": person, "employer": x.entity, "department": x.dep, "job": x.job, "arrival": "2027-04-01", "notice": "经理人选的独立首任通知"}, f.NewID(), 200)
	a, c := x.hire["assignment"].(string), other["assignment"].(string)
	base := func(id string) string { return "/api/v1/assignments/" + id }
	view := func(id, day string) M { return b.call("GET", base(id)+"?businessAt="+day, nil, "", 200) }
	type result struct {
		child, parent string
		code          int
		err           error
	}
	results := make(chan result, 2)
	for _, edge := range [][2]string{{a, c}, {c, a}} {
		cmd := M{"businessDate": "2027-05-01", "expectedSourceVersion": view(edge[0], "2027-05-01")["sourceVersion"], "purpose": "direct", "manager": edge[1]}
		go func() {
			code, _, err := b.response("POST", base(edge[0])+"/managers", cmd, f.NewID())
			results <- result{edge[0], edge[1], code, err}
		}()
	}
	counts := map[int]int{}
	var child, parent string
	for range 2 {
		r := <-results
		must(t, r.err)
		counts[r.code]++
		if r.code == 200 {
			child, parent = r.child, r.parent
		}
	}
	if counts[200] != 1 || counts[409] != 1 {
		t.Fatal("同用途并发互设", counts)
	}
	old := view(child, "2027-08-31")
	if facts := old["managers"].([]any); len(facts) != 1 || facts[0].(M)["manager"] != parent {
		t.Fatal(old)
	}
	// The reverse functional relationship is independent from the direct purpose.
	b.call("POST", base(parent)+"/managers", M{"businessDate": "2027-06-01", "expectedSourceVersion": view(parent, "2027-06-01")["sourceVersion"], "purpose": "functional", "manager": child}, f.NewID(), 200)
	p := view(parent, "2027-08-31")
	key := f.NewID()
	cmd := M{"date": "2027-08-31", "meaning": "last-active", "notice": "八月最后一日在职", "expectedSourceVersion": p["sourceVersion"]}
	x.h.f.exec("ALTER TABLE staffing.employment_exit ADD CONSTRAINT test_no_exit CHECK(false) NOT VALID")
	b.call("POST", base(parent)+"/leave", cmd, f.NewID(), 503)
	x.h.f.exec("ALTER TABLE staffing.employment_exit DROP CONSTRAINT test_no_exit")
	if view(child, "2027-09-01")["sourceVersion"] != old["sourceVersion"] {
		t.Fatal("汇报处置没有随离职失败回滚")
	}
	saved := b.call("POST", base(parent)+"/leave", cmd, key, 200)
	if b.call("POST", base(parent)+"/leave", cmd, key, 200)["action"] != saved["action"] {
		t.Fatal("离职重送")
	}
	if v := view(parent, "2027-09-01"); v["state"] != nil || v["until"] != "2027-09-01" {
		t.Fatal("离职边界", v)
	}
	if v := view(child, "2027-09-01"); v["state"] == nil || len(v["managers"].([]any)) != 0 {
		t.Fatal("经理退出", v)
	}
	v := b.call("GET", base(child)+"?businessAt=2027-09-01&sourceVersion="+old["sourceVersion"].(string), nil, "", 200)
	if len(v["managers"].([]any)) != 1 {
		t.Fatal("误改退出之前的汇报来源")
	}
	original := b.call("GET", "/api/v1/employment-actions/"+saved["action"].(string), nil, "", 200)
	if original["exit"].(M)["firstInactive"] != "2027-09-01" {
		t.Fatal("离职原通知边界", original)
	}
}

func TestPartialExitAndRehireKeepPermanentRecordNumbers(t *testing.T) {
	type M = map[string]any
	x := setupAssignment(t)
	b := x.h.b
	id := x.hire["assignment"].(string)
	path := func(id, suffix string) string { return "/api/v1/assignments/" + id + suffix }
	head := func(id, day string) any {
		return b.call("GET", path(id, "?businessAt="+day), nil, "", 200)["sourceVersion"]
	}
	pol := b.call("GET", "/api/v1/assignment-policies", nil, "", 200)
	b.call("POST", "/api/v1/assignment-policies", M{"expectedVersion": pol["version"], "row": M{"id": f.NewID(), "category": "Employee", "purpose": "secondary", "mode": "JobOnly", "from": "2027-04-01"}}, f.NewID(), 200)
	add := func(anchor, day string, parallel bool, employer string) M {
		cmd := M{"from": day, "expectedSourceVersion": head(anchor, day), "job": x.job, "department": x.dep, "notice": "与原任职分别聘任", "parallel": parallel}
		if parallel {
			cmd["employer"] = employer
		}
		key := f.NewID()
		saved := b.call("POST", path(anchor, "/additional-assignments"), cmd, key, 200)
		if b.call("POST", path(anchor, "/additional-assignments"), cmd, key, 200)["action"] != saved["action"] {
			t.Fatal("兼岗重送", saved)
		}
		return saved
	}
	one := add(id, "2027-05-01", false, "")
	other := x.h.create("LAW48", "CN", "legal_person")["id"].(string)
	b.call("POST", "/api/v1/legal-entities/"+other+"/purposes", M{"expectedVersion": "", "row": M{"id": f.NewID(), "role": "employer", "kind": "declaration", "basis": "独立用工主体", "from": "2027-04-01"}}, f.NewID(), 200)
	two := add(id, "2027-05-01", true, other)
	list := func(day string, expected float64) {
		q := b.call("POST", "/api/v1/assignments/query", M{"businessAt": day, "person": x.person, "page": 1, "pageSize": 20}, "", 200)
		if q["total"] != expected {
			t.Fatal(day, expected, q)
		}
	}
	exit := func(term, command, date, meaning string) M {
		return b.call("POST", path(term, command), M{"expectedSourceVersion": head(term, date), "date": date, "meaning": meaning, "notice": "原通知记载退出日期"}, f.NewID(), 200)
	}
	list("2027-05-01", 3)
	exit(two["assignment"].(string), "/leave", "2027-07-01", "first-inactive")
	list("2027-07-01", 2)
	exit(one["assignment"].(string), "/end", "2027-07-15", "last-active")
	list("2027-07-15", 2)
	list("2027-07-16", 1)
	left := exit(id, "/leave", "2027-08-31", "last-active")
	list("2027-08-31", 1)
	list("2027-09-01", 0)
	summary := b.call("GET", "/api/v1/persons/"+x.person+"/employment?businessAt=2027-09-01", nil, "", 200)
	ref := M{"action": left["action"], "revision": left["revision"]}
	if summary["rehireFrom"].(M)["action"] != left["action"] {
		t.Fatal("档案未定位原离职", summary)
	}
	cmd := M{"person": x.person, "employer": x.entity, "department": x.dep, "job": x.job, "arrival": "2027-10-01", "notice": "十月再聘通知"}
	b.call("POST", "/api/v1/hires", cmd, f.NewID(), 409)
	cmd["rehireOf"] = ref
	key := f.NewID()
	saved := b.call("POST", "/api/v1/hires", cmd, key, 200)
	if saved["employment"] == x.hire["employment"] || saved["assignment"] == id || saved["recordNumber"] != 0.0 {
		t.Fatal("再聘未沿用本人0号另建任期", saved)
	}
	if b.call("POST", "/api/v1/hires", cmd, key, 200)["action"] != saved["action"] {
		t.Fatal("再聘重送")
	}
	list("2027-09-30", 0)
	list("2027-10-01", 1)
	if three := add(saved["assignment"].(string), "2027-10-02", false, ""); three["recordNumber"] != 3.0 {
		t.Fatal("结束后号码回收", three)
	}
	original := b.call("GET", "/api/v1/employment-actions/"+saved["action"].(string), nil, "", 200)
	if original["rehireOf"].(M)["revision"] != left["revision"] {
		t.Fatal("再聘的原离职来源", original)
	}
	b.call("GET", "/api/v1/employment-actions/"+left["action"].(string), nil, "", 200)
}

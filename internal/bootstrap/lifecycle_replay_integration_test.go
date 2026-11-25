//go:build integration

package bootstrap_test

import (
	"testing"

	f "chnhcm.local/chnhcm/internal/foundation"
)

func TestManagerSuccessionBeforeAfterAndOnExitBoundary(t *testing.T) {
	type M = map[string]any
	x := lifecycleSample{setupAssignment(t)}
	a := x.hirePerson("原经理", "")["assignment"].(string)
	b := x.hirePerson("接任经理", "")["assignment"].(string)
	early := x.hire["assignment"].(string)
	late := x.hirePerson("迟录接任", "")["assignment"].(string)
	sameDay := x.hirePerson("同日接任", "")["assignment"].(string)
	alone := x.hirePerson("原汇报结束", "")["assignment"].(string)
	set := func(child, day, parent, purpose string, code int) {
		x.post(child, "managers", day, M{"businessDate": day, "manager": parent, "purpose": purpose}, code)
	}
	check := func(id, day, source, want string) {
		t.Helper()
		got := ""
		for _, v := range x.view(id, day, source)["managers"].([]any) {
			if v.(M)["purpose"] == "direct" {
				got = v.(M)["manager"].(string)
			}
		}
		if got != want {
			t.Fatalf("%s 在 %s 来源 %s 的直接经理=%s，预期 %s", id, day, source, got, want)
		}
	}
	for _, child := range []string{early, late, sameDay, alone} {
		set(child, "2027-05-01", a, "direct", 200)
	}
	set(late, "2027-05-01", a, "functional", 200)
	set(late, "2027-06-01", "", "functional", 200)
	before := x.view(late, "2027-09-01", "")["sourceVersion"].(string)
	set(early, "2027-07-01", b, "direct", 200)
	x.exit(a, "leave", "2027-09-01", "first-inactive", 200)
	afterExit := x.view(late, "2027-09-01", "")["sourceVersion"].(string)
	set(late, "2027-07-01", b, "direct", 200)
	set(sameDay, "2027-09-01", b, "direct", 200)
	set(sameDay, "2027-09-01", early, "direct", 409)
	for _, day := range []string{"2027-08-31", "2027-09-01"} {
		check(early, day, "", b)
		check(late, day, "", b)
	}
	check(late, "2027-09-01", before, a)
	check(late, "2027-09-01", afterExit, "")
	check(sameDay, "2027-08-31", "", a)
	check(sameDay, "2027-09-01", "", b)
	check(alone, "2027-09-01", "", "")
	if len(x.view(late, "2027-09-01", "")["managers"].([]any)) != 1 {
		t.Fatal("其他用途的显式结束失效")
	}
}

func TestExitDepartmentReplayRetainsBoundariesAndIndependentRelationFacts(t *testing.T) {
	type M = map[string]any
	x := lifecycleSample{setupAssignment(t)}
	b := x.h.b
	x.policy("secondary")
	x.policy("transfer")
	d := departmentCreateBody("D509", x.bu)
	d["name"] = "产品部"
	product := b.call("POST", "/api/v1/departments", d, f.NewID(), 201)["departmentId"].(string)
	main := x.hire["assignment"].(string)
	add := func(day string) string {
		return x.post(main, "additional-assignments", day, M{"from": day, "job": x.job, "department": x.dep, "parallel": false, "notice": "独立安排 " + day}, 200)["assignment"].(string)
	}
	one, two, three := add("2027-04-02"), add("2027-04-03"), add("2027-04-04")
	move := func(id, day string, insert bool, code int) M {
		return x.post(id, "transfers", day, M{"businessDate": day, "job": x.job, "department": product, "notice": "正式调入产品部", "insert": insert}, code)
	}
	move(one, "2027-05-01", false, 200)
	move(two, "2027-04-25", false, 200)
	x.post(main, "trial-start", "2027-04-05", M{"businessDate": "2027-04-05", "notice": "实际试用开始"}, 200)
	end := x.exit(one, "end", "2027-06-01", "first-inactive", 200)
	original := func(row M, rev string) M {
		return b.call("GET", "/api/v1/employment-actions/"+row["action"].(string)+"?adoptedRef="+rev, nil, "", 200)
	}
	reg := func(kind, dept string) M {
		return b.call("GET", "/api/v1/employment-actions?kind="+kind+"&department="+dept, nil, "", 200)
	}
	if reg("EndAssignment", product)["total"] != 1. || reg("EndAssignment", x.dep)["total"] != 0. || original(end, "")["departmentName"] != "产品部" {
		t.Fatal("首个不在职日发生部门", reg("EndAssignment", ""), original(end, ""))
	}
	x.post(two, "confirm-employment", "2027-07-01", M{"businessDate": "2027-07-01", "notice": "本关系七月转正"}, 200)
	x.exit(main, "leave", "2027-05-31", "last-active", 409)
	endLast := x.exit(two, "end", "2027-05-31", "last-active", 200)
	if original(endLast, "")["departmentName"] != "产品部" || reg("EndAssignment", product)["total"] != 2. || reg("EndAssignment", x.dep)["total"] != 0. {
		t.Fatal("两种通知日期的结束部门不一致")
	}

	if x.view(two, "2027-06-01", "")["state"] != nil || reg("ConfirmEmployment", "")["total"] != 1. {
		t.Fatal("关系转正误阻独立退出")
	}
	left := x.exit(main, "leave", "2027-08-31", "last-active", 200)
	oldHead := x.view(main, "2027-08-31", "")["sourceVersion"].(string)
	peerHead := x.view(three, "2027-09-01", "")["sourceVersion"]
	db := x.h.f
	db.exec("ALTER TABLE staffing.employment_outbox ADD CONSTRAINT fail_exit_rebase CHECK(action_id<>'" + left["action"].(string) + "'::uuid) NOT VALID")
	move(main, "2027-07-15", true, 503)
	db.exec("ALTER TABLE staffing.employment_outbox DROP CONSTRAINT fail_exit_rebase")
	if x.view(main, "2027-08-31", "")["sourceVersion"] != oldHead || x.view(three, "2027-08-31", "")["sourceVersion"] != peerHead {
		t.Fatal("退出原件末步失败未整笔回滚")
	}
	move(main, "2027-07-15", true, 200)
	now := original(left, "")
	current := now["action"].(M)
	if current["revision"] == left["revision"] || now["departmentName"] != "产品部" || original(left, left["revision"].(string))["action"].(M)["department"] != x.dep || reg("Leave", product)["total"] != 1. || reg("Leave", x.dep)["total"] != 0. {
		t.Fatal("退出当前或历史修订", now, original(left, left["revision"].(string)))
	}
	for _, id := range []string{main, three} {
		if v := x.view(id, "2027-09-01", ""); v["until"] != "2027-09-01" || v["state"] != nil {
			t.Fatal("新修订后任期越出边界", v)
		}
	}
	if v := x.view(main, "2027-08-31", oldHead); v["state"].(M)["departmentId"] != x.dep {
		t.Fatal("旧来源部门改变", v)
	}
	rehire := M{"arrival": "2027-10-01", "employer": x.entity, "person": x.person, "department": product, "job": x.job, "notice": "新的录用通知", "rehireOf": M{"action": left["action"], "revision": left["revision"]}}
	b.call("POST", "/api/v1/hires", rehire, f.NewID(), 409)
	rehire["rehireOf"] = M{"action": left["action"], "revision": current["revision"]}
	again := b.call("POST", "/api/v1/hires", rehire, f.NewID(), 200)
	if again["recordNumber"] != 0. || original(again, "")["rehireOf"].(M)["revision"] != current["revision"] {
		t.Fatal("新再聘引用了过时退出修订")
	}
	x.h.policy("UPDATE access.employment_grant SET fields=array_remove(fields,'department') WHERE principal_id=$1 AND action='register.read'", b.subject)
	if reg("EndAssignment", product)["total"] != 0. || reg("EndAssignment", x.dep)["total"] != 0. || reg("EndAssignment", "")["total"] != 2. {
		t.Fatal("具名结束部门越过字段权限")
	}
}

func TestFutureHireCancellationAfterWithdrawnPromotion(t *testing.T) {
	type M = map[string]any
	x := lifecycleSample{setupAssignment(t)}
	b := x.h.b
	staff := staffing307{t, x.h.f, b}
	p5, p6 := staff.create("grade", "P5", "")["id"].(string), staff.create("grade", "P6", "")["id"].(string)
	staff.correct("job", staff.detail("job", x.job, ""), M{"gradeIds": []string{p5, p6}}, nil, 200)
	hire := x.hirePerson("拟入职员工", p5)
	id := hire["assignment"].(string)
	day := "2027-05-01"
	p := x.post(id, "promotions", day, M{"businessDate": day, "grade": p6, "notice": "晋升任命"}, 200)
	cancel := func(row M, body M, code int) {
		b.call("POST", "/api/v1/employment-actions/"+row["action"].(string)+"/cancel", body, f.NewID(), code)
	}
	withdraw := M{"revision": hire["revision"], "reason": "未开始的聘任撤回"}
	cancel(hire, withdraw, 409)
	cancel(p, M{"expectedSourceVersion": x.view(id, day, "")["sourceVersion"], "revision": p["revision"], "reason": "误录晋升"}, 200)
	cancel(hire, withdraw, 200)
	if x.view(id, "2027-04-01", "")["state"] != nil {
		t.Fatal("未来首聘未撤销")
	}
	lookup := "/api/v1/employment-actions/" + p["action"].(string)
	if b.call("GET", lookup, nil, "", 200)["action"].(M)["operation"] != "Withdraw" || b.call("GET", lookup+"?adoptedRef="+p["revision"].(string), nil, "", 200)["promotion"].(M)["after"].(M)["id"] != p6 {
		t.Fatal("原晋升或撤销审计丢失")
	}
}

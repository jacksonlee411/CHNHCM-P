//go:build integration

package bootstrap_test

import (
	"encoding/json"
	"net/http"
	"testing"

	f "chnhcm.local/chnhcm/internal/foundation"
)

type orderInput = map[string]any

type orderFixture struct {
	assignmentFixture
	main string
}

func newOrderFixture(t *testing.T, purposes ...string) orderFixture {
	t.Helper()
	x := setupAssignment(t)
	b := x.h.b
	for _, purpose := range purposes {
		p := b.call("GET", "/api/v1/assignment-policies", nil, "", 200)
		b.call("POST", "/api/v1/assignment-policies", orderInput{"expectedVersion": p["version"], "row": orderInput{
			"id": f.NewID(), "category": "Employee", "purpose": purpose, "mode": "JobOnly", "from": "2027-04-01",
		}}, f.NewID(), 200)
	}
	return orderFixture{x, x.hire["assignment"].(string)}
}

func (x orderFixture) view(id, day string) orderInput {
	return x.h.b.call("GET", "/api/v1/assignments/"+id+"?businessAt="+day, nil, "", 200)
}

func (x orderFixture) write(id, path, day string, body orderInput, status int) orderInput {
	body["expectedSourceVersion"] = x.view(id, day)["sourceVersion"]
	body["businessDate"] = day
	return x.h.b.call("POST", "/api/v1/assignments/"+id+"/"+path, body, f.NewID(), status)
}

func (x orderFixture) department(code string) string {
	body := departmentCreateBody(code, x.bu)
	body["name"] = "部门" + code
	return x.h.b.call("POST", "/api/v1/departments", body, f.NewID(), 201)["departmentId"].(string)
}

func (x orderFixture) secondary(department string) string {
	return x.h.b.call("POST", "/api/v1/assignments/"+x.main+"/additional-assignments", orderInput{
		"from": "2027-04-02", "parallel": false, "department": department, "job": x.job, "notice": "培训兼任通知",
		"expectedSourceVersion": x.view(x.main, "2027-04-02")["sourceVersion"],
	}, f.NewID(), 200)["assignment"].(string)
}

func (x orderFixture) grades(t *testing.T) (string, string) {
	t.Helper()
	s := staffing307{t, x.h.f, x.h.b}
	p5 := s.create("grade", "P5ORDER", "")["id"].(string)
	p6 := s.create("grade", "P6ORDER", "")["id"].(string)
	s.correct("job", s.detail("job", x.job, ""), orderInput{"gradeIds": []string{p5, p6}}, nil, 200)
	x.write(x.main, "grade-records", "2027-04-01", orderInput{"grade": p5, "notice": "首日实际任职级"}, 200)
	return p5, p6
}

func (x orderFixture) totals() [3]int {
	return [3]int{x.h.f.scalar("SELECT count(*) FROM staffing.employment_action_revision"),
		x.h.f.scalar("SELECT count(*) FROM staffing.assignment_source"), x.h.f.scalar("SELECT count(*) FROM staffing.employment_outbox")}
}

func (x orderFixture) original(receipt orderInput, exact bool) orderInput {
	path := "/api/v1/employment-actions/" + receipt["action"].(string)
	if exact {
		path += "?adoptedRef=" + receipt["revision"].(string)
	}
	return x.h.b.call("GET", path, nil, "", 200)
}

func (x orderFixture) predecessors(t *testing.T, id, day string) []orderInput {
	t.Helper()
	res, err := x.h.b.client.Get(x.h.f.server.URL + "/api/v1/assignments/" + id + "/probation-predecessors?businessAt=" + day)
	must(t, err)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("转正前序候选 HTTP %d", res.StatusCode)
	}
	var rows []orderInput
	must(t, json.NewDecoder(res.Body).Decode(&rows))
	return rows
}

func TestAssignmentAddAndInsertUseTheirBusinessPosition(t *testing.T) {
	x := newOrderFixture(t)
	const june, july, aug = "2027-06-01", "2027-07-01", "2027-08-01"
	location := func(day string, idx int, insert bool, code int) orderInput {
		return x.write(x.main, "location-changes", day, orderInput{"location": x.locations[idx], "insert": insert}, code)
	}
	location(july, 1, false, 200)
	before := x.totals()
	location(june, 2, false, 409)
	location(aug, 3, true, 409)
	if x.totals() != before {
		t.Fatal("错误的历史位置产生了业务写入")
	}
	for _, day := range []string{june, july, aug} {
		v := x.view(x.main, day)
		if v["canInsertRecord"] != (day < july) || v["canAddRecord"] != (day >= july) {
			t.Fatalf("%s 的办理位置不正确 %#v %#v", day, v["canAddRecord"], v["canInsertRecord"])
		}
	}
	early := location(june, 2, true, 200)
	late := location(aug, 3, false, 200)
	if x.original(early, true)["action"].(orderInput)["operation"] != "Insert" || x.original(late, true)["action"].(orderInput)["operation"] != "Add" {
		t.Fatal("历史补登与末端新增分类错误")
	}
	x.h.b.call("POST", "/api/v1/assignments/"+x.main+"/actions/"+late["action"].(string)+"/delete", orderInput{
		"expectedSourceVersion": x.view(x.main, aug)["sourceVersion"], "expectedRevision": late["revision"],
	}, f.NewID(), 200)
	if x.view(x.main, "2027-07-15")["canAddRecord"] != true {
		t.Fatal("已删除的办理仍被当作有效末日")
	}
}

func TestTransferCorrectionRecordsOnlyChangedOrder(t *testing.T) {
	x := newOrderFixture(t, "transfer", "secondary")
	const day = "2027-07-01"
	secondary := x.secondary(x.department("TRAINORDER"))
	dest := x.department("PRODORDER")
	x.write(x.main, "trial-start", "2027-04-01", orderInput{"notice": "实际试用日"}, 200)
	move := x.write(x.main, "transfers", day, orderInput{"department": dest, "job": x.job, "notice": "调入产品"}, 200)
	key := f.NewID()
	first := x.h.b.call("POST", "/api/v1/assignments/"+x.main+"/confirm-employment", orderInput{
		"expectedSourceVersion": x.view(x.main, day)["sourceVersion"], "businessDate": day, "notice": "当日转正",
	}, key, 200)
	before := x.original(first, true)["probation"].(orderInput)
	if before["department"] != nil || before["attribution"] != "same-day-order-undetermined" {
		t.Fatal("未载先后时的归属", before)
	}
	correct := "transfers/" + move["action"].(string) + "/correct"
	reference := func(v orderInput) orderInput { return orderInput{"action": v["action"], "revision": v["revision"]} }
	current := x.original(move, false)["action"].(orderInput)
	input := orderInput{"expectedRevision": current["revision"], "job": x.job, "department": dest, "notice": "调入产品", "previous": reference(first)}
	updated := x.write(x.main, correct, day, input, 200)
	adjusted := x.original(first, false)
	latest := adjusted["action"].(orderInput)["revision"]
	if latest == first["revision"] || adjusted["probation"].(orderInput)["department"] != x.dep || x.original(updated, true)["previous"].(orderInput)["revision"] != latest {
		t.Fatal("仅更正调动前序后，转正未取调出部门或调动仍引用旧版", updated, adjusted)
	}
	for _, id := range []string{x.main, secondary} {
		found := false
		for _, item := range x.view(id, day)["actions"].([]any) {
			action := item.(orderInput)
			if action["id"] == first["action"] {
				found = action["revision"] == latest
			}
		}
		if !found {
			t.Fatal("相关任职未采用同一转正修订", id)
		}
	}
	counts := x.totals()
	input["expectedRevision"] = updated["revision"]
	input["previous"] = orderInput{"action": first["action"], "revision": latest}
	x.write(x.main, correct, day, input, 409)
	resolved := x.h.b.call("POST", "/api/v1/employment-actions/submissions/"+key+"/resolve", nil, "", 200)
	if x.totals() != counts || resolved["revision"] != first["revision"] || x.original(first, true)["probation"].(orderInput)["department"] != nil {
		t.Fatal("原样调动更正、原回执或旧转正原件不符", resolved)
	}
}

func TestUnchangedPromotionRetainsPredecessorAndSuccessor(t *testing.T) {
	x := newOrderFixture(t, "transfer")
	const day = "2027-07-01"
	_, p6 := x.grades(t)
	x.write(x.main, "trial-start", "2027-04-01", orderInput{"notice": "实际试用日"}, 200)
	confirm := x.write(x.main, "confirm-employment", day, orderInput{"notice": "最先转正"}, 200)
	ref := func(r orderInput) orderInput { return orderInput{"action": r["action"], "revision": r["revision"]} }
	promote := x.write(x.main, "promotions", day, orderInput{"grade": p6, "notice": "职级任命", "previous": ref(confirm)}, 200)
	move := x.write(x.main, "transfers", day, orderInput{"job": x.job, "department": x.department("P6ORDER"), "notice": "独立调令", "previous": ref(promote)}, 200)
	current := x.original(promote, false)
	path := "promotions/" + promote["action"].(string) + "/correct"
	input := orderInput{"grade": p6, "expectedRevision": current["action"].(orderInput)["revision"], "notice": "职级任命", "previous": current["previous"]}
	before := x.totals()
	x.write(x.main, path, day, input, 409)
	input["precedes"] = ref(move)
	input["department"], input["job"] = x.dep, x.job
	x.write(x.main, path, day, input, 409)
	if x.totals() != before {
		t.Fatal("原样更正增加修订、任职来源或出站", before, x.totals())
	}
	input["reason"] = "任命原因核正"
	edited := x.write(x.main, path, day, input, 200)
	if x.original(edited, true)["promotion"].(orderInput)["reason"] != "任命原因核正" ||
		x.original(promote, true)["promotion"].(orderInput)["reason"] == "任命原因核正" || x.original(move, false)["previous"].(orderInput)["revision"] != edited["revision"] {
		t.Fatal("真实更正、后项引用或旧原因不符")
	}
}

func TestConfirmationUsesPrimaryAssignmentDecisionAndItsAdoptedDepartment(t *testing.T) {
	x := newOrderFixture(t, "transfer", "secondary")
	const day = "2027-07-01"
	_, p6 := x.grades(t)
	x.write(x.main, "trial-start", "2027-04-01", orderInput{"notice": "试用开始"}, 200)
	secondary := x.secondary(x.department("TRAINPRE"))
	dest, other := x.department("PRODPRE"), x.department("QUALPRE")
	otherMove := x.write(secondary, "transfers", day, orderInput{"department": other, "job": x.job, "notice": "兼任质量职务"}, 200)
	promotion := x.write(x.main, "promotions", day, orderInput{"grade": p6, "job": x.job, "department": dest, "notice": "晋升及调入产品的单一任命"}, 200)
	options := x.predecessors(t, secondary, day)
	if len(options) != 1 || options[0]["id"] != promotion["action"] || options[0]["revision"] != promotion["revision"] || options[0]["kind"] != "Promotion" {
		t.Fatal("兼岗入口没有返回主任职的确切任命", options)
	}
	input := orderInput{"notice": "任命之后转正", "afterAction": otherMove["action"], "afterRevision": otherMove["revision"]}
	x.write(secondary, "confirm-employment", day, input, 409)
	input["afterAction"], input["afterRevision"] = promotion["action"], promotion["revision"]
	x.h.policy("UPDATE access.employment_grant SET department_id=$2 WHERE principal_id=$1 AND action='history'", x.h.b.subject, x.dep)
	if options := x.predecessors(t, secondary, day); len(options) != 0 {
		t.Fatal("失去目的端历史权后仍出现任命", options)
	}
	counts := x.totals()
	x.write(secondary, "confirm-employment", day, input, 403)
	if x.totals() != counts {
		t.Fatal("失权提交产生了业务事实")
	}
	x.h.policy("UPDATE access.employment_grant SET department_id=NULL WHERE principal_id=$1 AND action='history'", x.h.b.subject)
	decision := x.write(secondary, "confirm-employment", day, input, 200)
	ev := x.original(decision, true)["probation"].(orderInput)
	orig := x.original(promotion, true)
	adopted := orig["transfer"].(orderInput)["after"].(orderInput)["segments"].([]any)[0].(orderInput)
	if ev["department"] != dest || ev["departmentName"] != "部门PRODPRE" || ev["afterRevision"] != promotion["revision"] || ev["departmentRevision"] != adopted["departmentRevision"] || ev["departmentSource"] != adopted["departmentSource"] || orig["promotion"].(orderInput)["department"] != x.dep {
		t.Fatal("转正未采用任命的确切调入部门，或晋升原件发生部门被改写", ev, orig)
	}
	x.h.policy("UPDATE access.employment_grant SET department_id=$2 WHERE principal_id=$1 AND action='history'", x.h.b.subject, other)
	clipped := x.original(decision, true)["probation"].(orderInput)
	if clipped["afterAction"] != nil || clipped["afterRevision"] != nil {
		t.Fatal("仅获兼岗历史权时披露了主任职的晋升原件", clipped)
	}
}

func TestConfirmationRetainsDocumentedSameDepartmentTransfer(t *testing.T) {
	x := newOrderFixture(t, "transfer", "secondary")
	const day = "2027-07-01"
	s := staffing307{t, x.h.f, x.h.b}
	family := s.create("jobfamily", "ORDERF", "")["id"].(string)
	job := s.create("job", "ORDERSAME", family)["id"].(string)
	employer := x.h.create("ORDERSUB", "CN", "legal_person")["id"].(string)
	x.h.b.call("POST", "/api/v1/legal-entities/"+employer+"/purposes", orderInput{"expectedVersion": "", "row": orderInput{
		"id": f.NewID(), "role": "employer", "kind": "declaration", "basis": "独立任用资格", "from": "2027-04-01",
	}}, f.NewID(), 200)
	parallel := x.h.b.call("POST", "/api/v1/assignments/"+x.main+"/additional-assignments", orderInput{
		"from": "2027-04-02", "parallel": true, "employer": employer, "job": x.job, "department": x.dep, "notice": "独立聘任",
		"expectedSourceVersion": x.view(x.main, "2027-04-02")["sourceVersion"],
	}, f.NewID(), 200)
	id := parallel["assignment"].(string)
	if parallel["recordNumber"] != 1. || parallel["employment"] == x.hire["employment"] || x.view(id, day)["relationshipPrimary"] != true {
		t.Fatal("独立关系的主任职应为正号", parallel)
	}
	x.write(id, "trial-start", "2027-04-02", orderInput{"notice": "独立关系的试用开始"}, 200)
	x.write(x.main, "transfers", day, orderInput{"job": job, "department": x.dep, "notice": "原关系同日调任"}, 200)
	move := x.write(id, "transfers", day, orderInput{"job": job, "department": x.dep, "notice": "并行关系同部门正式调任"}, 200)
	candidates := x.predecessors(t, id, day)
	if len(candidates) != 1 || candidates[0]["id"] != move["action"] {
		t.Fatal("同部门的真实调令没有出现在候选", candidates)
	}
	confirmation := x.write(id, "confirm-employment", day, orderInput{"notice": "同部门调任后转正", "afterAction": move["action"], "afterRevision": move["revision"]}, 200)
	ev := x.original(confirmation, true)["probation"].(orderInput)
	if ev["afterRevision"] != move["revision"] || ev["department"] != x.dep {
		t.Fatal("同部门调整丢失确切前序或改变了归属", ev)
	}
}

func TestGradeOnlyPredecessorPreservesUnknownTransferOrder(t *testing.T) {
	x := newOrderFixture(t, "transfer")
	const day = "2027-07-01"
	_, p6 := x.grades(t)
	x.write(x.main, "trial-start", "2027-04-01", orderInput{"notice": "实际试用"}, 200)
	grade := x.write(x.main, "promotions", day, orderInput{"grade": p6, "notice": "只升个人职级"}, 200)
	other := x.department("UNKNOWNMOVE")
	x.write(x.main, "transfers", day, orderInput{"job": x.job, "department": other, "notice": "另一张未载与晋升先后的调令"}, 200)
	revision := x.original(grade, false)["action"].(orderInput)["revision"]
	confirmation := x.write(x.main, "confirm-employment", day, orderInput{"notice": "通知只载职级晋升后转正", "afterAction": grade["action"], "afterRevision": revision}, 200)
	ev := x.original(confirmation, true)["probation"].(orderInput)
	if ev["afterRevision"] != revision || ev["department"] != nil || ev["attribution"] != "same-day-order-undetermined" {
		t.Fatal("无调入的晋升不能确定其与另一张调令的实际先后", ev)
	}
}

func TestUnchangedStandalonePromotionAndWithdrawnPredecessors(t *testing.T) {
	x := newOrderFixture(t)
	const day = "2027-08-01"
	p5, p6 := x.grades(t)
	s := staffing307{t, x.h.f, x.h.b}
	p7 := s.create("grade", "P7ORDER", "")["id"].(string)
	s.correct("job", s.detail("job", x.job, ""), orderInput{"gradeIds": []string{p5, p6, p7}}, nil, 200)
	x.write(x.main, "trial-start", "2027-04-02", orderInput{"notice": "实际试用"}, 200)
	x.write(x.main, "promotions", "2027-07-01", orderInput{"grade": p6, "notice": "七月 P6 任命"}, 200)
	initial := x.write(x.main, "promotions", day, orderInput{"grade": p7, "notice": "八月 P7 任命"}, 200)
	path := "promotions/" + initial["action"].(string)
	input := orderInput{"grade": p7, "notice": "八月 P7 任命", "expectedRevision": initial["revision"]}
	before := x.totals()
	if refused := x.write(x.main, path+"/correct", day, input, 409); refused["code"] != "PROMOTION_UNCHANGED" || x.totals() != before {
		t.Fatal("无前序的 P7 原样更正仍产生业务写入", refused)
	}
	input["reason"] = "八月任命原因补正"
	edited := x.write(x.main, path+"/correct", day, input, 200)
	confirm := orderInput{"afterAction": initial["action"], "afterRevision": initial["revision"], "notice": "任命后转正"}
	before = x.totals()
	if refused := x.write(x.main, "confirm-employment", day, confirm, 409); refused["code"] != "PROBATION_ORDER" || x.totals() != before {
		t.Fatal("已被更正的晋升修订可用于新转正", refused)
	}
	withdrawn := x.h.b.call("POST", "/api/v1/employment-actions/"+initial["action"].(string)+"/cancel", orderInput{
		"expectedSourceVersion": x.view(x.main, day)["sourceVersion"], "revision": edited["revision"], "reason": "取消八月任命",
	}, f.NewID(), 200)
	if options := x.predecessors(t, x.main, day); len(options) != 0 {
		t.Fatal("已撤销的晋升仍可选作前序", options)
	}
	before = x.totals()
	for _, rev := range []any{edited["revision"], withdrawn["revision"]} {
		confirm["afterRevision"] = rev
		if refused := x.write(x.main, "confirm-employment", day, confirm, 409); refused["code"] != "PROBATION_ORDER" || x.totals() != before {
			t.Fatal("已撤销晋升的确切旧/新修订被用于转正", refused)
		}
	}
}

func TestTransferCorrectionAndExitShareExpectedSource(t *testing.T) {
	x := newOrderFixture(t, "transfer")
	const june, last = "2027-06-01", "2027-08-31"
	old, next := x.department("RACEOLD"), x.department("RACENEW")
	move := x.write(x.main, "transfers", june, orderInput{"department": old, "job": x.job, "notice": "六月调任"}, 200)
	base := "/api/v1/assignments/" + x.main
	source := x.view(x.main, last)["sourceVersion"]
	path := base + "/transfers/" + move["action"].(string) + "/correct"
	inputs := map[string]orderInput{
		path:            {"businessDate": june, "expectedRevision": move["revision"], "department": next, "job": x.job, "notice": "核正调入部门", "expectedSourceVersion": source},
		base + "/leave": {"date": last, "meaning": "last-active", "notice": "期满离职", "expectedSourceVersion": source},
	}
	type outcome struct {
		path string
		code int
		body orderInput
		err  error
	}
	results := make(chan outcome, 2)
	before := x.totals()
	for endpoint, input := range inputs {
		go func() {
			code, body, err := x.h.b.response("POST", endpoint, input, f.NewID())
			results <- outcome{endpoint, code, body, err}
		}()
	}
	var accepted, rejected outcome
	for range 2 {
		r := <-results
		must(t, r.err)
		if r.code == 200 {
			accepted = r
		} else if r.code == 409 && r.body["code"] == "ASSIGNMENT_CHANGED" {
			rejected = r
		} else {
			t.Fatal("并发提交的终态错误", r)
		}
	}
	if accepted.path == "" || rejected.path == "" || x.totals() != [3]int{before[0] + 1, before[1] + 1, before[2] + 1} {
		t.Fatal("同一任职来源只能提交一笔，无效一方没有业务增量", accepted, rejected, before, x.totals())
	}
	retry := inputs[rejected.path]
	retry["expectedSourceVersion"] = x.view(x.main, last)["sourceVersion"]
	resubmitted := x.h.b.call("POST", rejected.path, retry, f.NewID(), 200)
	leave := accepted.body
	if rejected.path == base+"/leave" {
		leave = resubmitted
	}
	if x.original(leave, false)["action"].(orderInput)["department"] != next || x.original(move, true)["transfer"].(orderInput)["after"].(orderInput)["department"] != old {
		t.Fatal("重新核对来源后，离职或原调令的部门不符", leave)
	}
}

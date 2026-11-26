//go:build integration

package bootstrap_test

import (
	app "chnhcm.local/chnhcm/internal/contexts/staffing/application"
	pg "chnhcm.local/chnhcm/internal/contexts/staffing/infra/postgres"
	"slices"
	"testing"
	"time"

	f "chnhcm.local/chnhcm/internal/foundation"
)

func TestConfirmationSurvivesDeletingPrecedingTransfer(t *testing.T) {
	type M = map[string]any
	x := setupAssignment(t)
	b := x.h.b
	root := "/api/v1/assignments/" + x.hire["assignment"].(string)
	day := "2027-07-01"
	view := func(at string) M {
		return b.call("GET", root+"?businessAt="+at, nil, "", 200)
	}
	post := func(tail, at string, body M) M {
		body["expectedSourceVersion"] = view(at)["sourceVersion"]
		return b.call("POST", root+tail, body, f.NewID(), 200)
	}
	dep := departmentCreateBody("R500", x.bu)
	dep["name"] = "复核产品部"
	prod := b.call("POST", "/api/v1/departments", dep, f.NewID(), 201)["departmentId"].(string)
	policy := b.call("GET", "/api/v1/assignment-policies", nil, "", 200)
	b.call("POST", "/api/v1/assignment-policies", M{"expectedVersion": policy["version"], "row": M{"id": f.NewID(), "category": "Employee", "purpose": "transfer", "mode": "JobOnly", "from": "2027-04-01"}}, f.NewID(), 200)
	post("/trial-start", "2027-04-05", M{"businessDate": "2027-04-05", "notice": "实际试用开始"})
	move := post("/transfers", day, M{"businessDate": day, "department": prod, "job": x.job, "notice": "通知载明先调动"})
	conf := post("/confirm-employment", day, M{"businessDate": day, "notice": "先调动，后转正", "afterAction": move["action"], "afterRevision": move["revision"]})
	original := func() M {
		return b.call("GET", "/api/v1/employment-actions/"+conf["action"].(string), nil, "", 200)
	}
	before := original()["probation"].(M)
	if before["department"] != prod || before["afterRevision"] != move["revision"] {
		t.Fatal("试件前置", before)
	}
	var kinds []string
	for _, entry := range view(day)["actions"].([]any) {
		v := entry.(M)
		if v["businessDate"] == day {
			kinds = append(kinds, v["kind"].(string))
		}
	}
	t.Logf("通知已采用调动修订；该日实际显示顺序 %v", kinds)
	reg := func(kind, dep string) M {
		return b.call("GET", "/api/v1/employment-actions?kind="+kind+"&department="+dep, nil, "", 200)
	}
	if got := reg("ConfirmEmployment", prod)["total"]; got != 1.0 {
		t.Fatal(got)
	}
	oldSource := view(day)["sourceVersion"].(string)
	b.call("POST", root+"/actions/"+move["action"].(string)+"/delete", M{"expectedRevision": move["revision"], "expectedSourceVersion": oldSource}, f.NewID(), 200)
	active := view(day)["state"].(M)["departmentId"]
	current := original()
	old := b.call("GET", "/api/v1/employment-actions/"+conf["action"].(string)+"?adoptedRef="+conf["revision"].(string), nil, "", 200)
	if active != x.dep || current["probation"].(M)["department"] != x.dep || current["probation"].(M)["afterRevision"] != nil || current["action"].(M)["revision"] == conf["revision"] ||
		old["probation"].(M)["department"] != prod || old["probation"].(M)["afterRevision"] != move["revision"] || old["action"].(M)["revision"] != conf["revision"] || old["action"].(M)["sourceVersion"] != oldSource {
		t.Fatal("当前与原采用修订", current, old)
	}
	leaveDay := "2027-08-31"
	post("/leave", leaveDay, M{"date": leaveDay, "meaning": "last-active", "notice": "该岗离职通知"})
	register := b.call("GET", "/api/v1/employment-actions?kind=Leave", nil, "", 200)
	leave := register["rows"].([]any)[0].(M)
	t.Logf("离职总数=%v，列表发生部门=%v，所在部门筛选=%v", register["total"], leave["departmentName"], reg("Leave", active.(string))["total"])
	if len(kinds) != 2 || kinds[0] != "Transfer" || kinds[1] != "ConfirmEmployment" {
		t.Error("转正明引调动，展示顺序未承接该前序")
	}
	if reg("ConfirmEmployment", prod)["total"] != 0.0 || reg("ConfirmEmployment", x.dep)["total"] != 1.0 {
		t.Error("删除调动后产生失去事实依据的转正部门")
	}
	if leave["department"] != active || leave["departmentName"] == "" || reg("Leave", prod)["total"] != 0.0 || reg("Leave", active.(string))["total"] != 1.0 {
		t.Error("已知离职部门的筛选漏笔")
	}
}

func TestConfirmationReplaysEarlierTransfer(t *testing.T) {
	type M = map[string]any
	x := setupAssignment(t)
	b := x.h.b
	root := "/api/v1/assignments/" + x.hire["assignment"].(string)
	source := func(day string) any { return b.call("GET", root+"?businessAt="+day, nil, "", 200)["sourceVersion"] }
	post := func(path, day string, cmd M) M {
		cmd["expectedSourceVersion"] = source(day)
		return b.call("POST", root+path, cmd, f.NewID(), 200)
	}
	policy := b.call("GET", "/api/v1/assignment-policies", nil, "", 200)
	b.call("POST", "/api/v1/assignment-policies", M{"expectedVersion": policy["version"], "row": M{"id": f.NewID(), "category": "Employee", "purpose": "transfer", "mode": "JobOnly", "from": "2027-04-01"}}, f.NewID(), 200)
	prod := b.call("POST", "/api/v1/departments", departmentCreateBody("RV500", x.bu), f.NewID(), 201)["departmentId"]
	post("/trial-start", "2027-04-01", M{"businessDate": "2027-04-01", "notice": "实际试用开始"})
	key := f.NewID()
	day := "2027-07-01"
	notice := "原通知未记载发生部门"
	cmd := M{"businessDate": day, "notice": notice, "expectedSourceVersion": source(day)}
	first := b.call("POST", root+"/confirm-employment", cmd, key, 200)
	original := func(rev string) M {
		return b.call("GET", "/api/v1/employment-actions/"+first["action"].(string)+"?adoptedRef="+rev, nil, "", 200)
	}
	from := source(day)
	move := M{"expectedSourceVersion": from, "businessDate": "2027-06-01", "department": prod, "job": x.job, "insert": true, "notice": "补录六月调动"}
	x.h.f.exec("ALTER TABLE staffing.employment_outbox ADD CONSTRAINT rollback_rebase CHECK(false) NOT VALID")
	b.call("POST", root+"/transfers", move, f.NewID(), 503)
	if source(day) != from || original("")["action"].(M)["revision"] != first["revision"] {
		t.Fatal("出站失败仍有部分发布")
	}
	x.h.f.exec("ALTER TABLE staffing.employment_outbox DROP CONSTRAINT rollback_rebase")
	b.call("POST", root+"/transfers", move, f.NewID(), 200)
	current := original("")
	prev := original(first["revision"].(string))
	if source(day) == from || current["probation"].(M)["department"] != prod || current["action"].(M)["revision"] == first["revision"] || prev["probation"].(M)["department"] != x.dep {
		t.Fatal("转正归属及旧来源", current, prev)
	}
	now := source(day)
	count := func() int { return x.h.f.scalar("SELECT count(*) FROM staffing.employment_outbox") }
	events := count()
	newKey := f.NewID()
	repeat := M{"businessDate": day, "notice": notice, "expectedSourceVersion": now}
	again := b.call("POST", root+"/confirm-employment", repeat, newKey, 200)
	if again["action"] != first["action"] || again["revision"] != current["action"].(M)["revision"] || b.call("POST", root+"/confirm-employment", cmd, key, 200)["revision"] != first["revision"] ||
		b.call("POST", "/api/v1/employment-actions/submissions/"+newKey+"/resolve", nil, "", 200)["revision"] != again["revision"] || source(day) != now || count() != events {
		t.Fatal("新旧提交号未固定各自修订", first, again)
	}
	conflict := M{"businessDate": day, "notice": "另一份通知", "expectedSourceVersion": now}
	b.call("POST", root+"/confirm-employment", conflict, f.NewID(), 409)
	b.call("POST", root+"/confirm-employment", conflict, newKey, 409)

}

func TestConfirmationRepeatFromOtherAssignment(t *testing.T) {
	type M = map[string]any
	x := setupAssignment(t)
	b := x.h.b
	root := x.hire["assignment"].(string)
	day := "2027-07-01"
	get := func(id string) M { return b.call("GET", "/api/v1/assignments/"+id+"?businessAt="+day, nil, "", 200) }
	post := func(id, path string, input M, code int) M {
		input["expectedSourceVersion"] = get(id)["sourceVersion"]
		return b.call("POST", "/api/v1/assignments/"+id+"/"+path, input, f.NewID(), code)
	}
	p := b.call("GET", "/api/v1/assignment-policies", nil, "", 200)
	b.call("POST", "/api/v1/assignment-policies", M{"expectedVersion": p["version"], "row": M{"id": f.NewID(), "category": "Employee", "purpose": "secondary", "mode": "JobOnly", "from": "2027-04-01"}}, f.NewID(), 200)
	training := b.call("POST", "/api/v1/departments", departmentCreateBody("TRAINF5", x.bu), f.NewID(), 201)["departmentId"].(string)
	secondary := post(root, "additional-assignments", M{"from": "2027-05-01", "parallel": false, "department": training, "job": x.job, "notice": "同一关系兼任"}, 200)["assignment"].(string)
	post(root, "trial-start", M{"businessDate": "2027-04-08", "notice": "实际开始试用"}, 200)
	decision := func() M {
		return M{"businessDate": day, "notice": "同一份合成转正通知，日期和所属关系相同"}
	}
	first := post(root, "confirm-employment", decision(), 200)
	repeat := decision()
	repeat["expectedSourceVersion"] = get(secondary)["sourceVersion"]
	before := x.h.f.scalar("SELECT count(*) FROM staffing.assignment_source")
	events := x.h.f.scalar("SELECT count(*) FROM staffing.employment_outbox")
	result := b.call("POST", "/api/v1/assignments/"+secondary+"/confirm-employment", repeat, f.NewID(), 200)
	reg := b.call("GET", "/api/v1/employment-actions?kind=ConfirmEmployment", nil, "", 200)
	if result["assignment"] != secondary || result["action"] != first["action"] || result["revision"] != first["revision"] || reg["total"] != 1.0 || get(secondary)["sourceVersion"] != repeat["expectedSourceVersion"] ||
		x.h.f.scalar("SELECT count(*) FROM staffing.assignment_source") != before || x.h.f.scalar("SELECT count(*) FROM staffing.employment_outbox") != events {
		t.Fatal("跨任职同单恢复", result, reg)
	}
	checkOriginalTerm := func(want string) {
		t.Helper()
		path := "/api/v1/employment-actions/" + first["action"].(string) + "?adoptedRef=" + first["revision"].(string)
		ref := b.call("GET", path, nil, "", 200)["action"].(M)
		view := b.call("GET", "/api/v1/assignments/"+ref["assignment"].(string)+"?businessAt="+ref["businessDate"].(string)+"&sourceVersion="+ref["sourceVersion"].(string), nil, "", 200)
		for _, entry := range view["actions"].([]any) {
			row := entry.(M)
			if view["assignment"] == want && view["sourceVersion"] == ref["sourceVersion"] && row["id"] == first["action"] && row["revision"] == first["revision"] {
				return
			}
		}
		t.Fatal("所选关系修订的获权任职和来源没有配对", ref, view)
	}
	checkOriginalTerm(root)
	x.h.policy("UPDATE access.employment_grant SET department_id=$2 WHERE principal_id=$1 AND action IN ('read','history')", b.subject, training)
	checkOriginalTerm(secondary)
	key := f.NewID()
	result = b.call("POST", "/api/v1/assignments/"+secondary+"/confirm-employment", repeat, key, 200)
	resolved := b.call("POST", "/api/v1/employment-actions/submissions/"+key+"/resolve", nil, "", 200)
	original := b.call("GET", "/api/v1/employment-actions/"+first["action"].(string), nil, "", 200)
	if result["assignment"] != secondary || resolved["assignment"] != secondary || resolved["revision"] != first["revision"] || original["probation"].(M)["department"] != nil {
		t.Fatal("培训兼岗的关系回执与字段披露", result, resolved, original)
	}
	x.h.policy("UPDATE access.employment_grant SET department_id=$2 WHERE principal_id=$1 AND action='probation.confirm'", b.subject, training)
	b.call("POST", "/api/v1/assignments/"+secondary+"/confirm-employment", repeat, f.NewID(), 403)
	if x.h.f.scalar("SELECT count(*) FROM staffing.assignment_source") != before || x.h.f.scalar("SELECT count(*) FROM staffing.employment_outbox") != events {
		t.Fatal("换号核对或失权重办生成了新业务事实")
	}
}

func TestConfirmationInsertedBeforeExistingPromotion(t *testing.T) {
	type M = map[string]any
	x := setupAssignment(t)
	b := x.h.b
	s := staffing307{t, x.h.f, b}
	p5 := s.create("grade", "R500P5", "")["id"].(string)
	p6 := s.create("grade", "R500P6", "")["id"].(string)
	s.correct("job", s.detail("job", x.job, ""), M{"gradeIds": []string{p5, p6}}, nil, 200)
	day := "2027-07-01"
	root := "/api/v1/assignments/" + x.hire["assignment"].(string)
	src := func(at string) any { return b.call("GET", root+"?businessAt="+at, nil, "", 200)["sourceVersion"] }
	post := func(path, at string, in M) M {
		in["expectedSourceVersion"] = src(at)
		in["businessDate"] = at
		return b.call("POST", root+"/"+path, in, f.NewID(), 200)
	}
	pol := b.call("GET", "/api/v1/assignment-policies", nil, "", 200)
	b.call("POST", "/api/v1/assignment-policies", M{"expectedVersion": pol["version"], "row": M{"id": f.NewID(), "purpose": "transfer", "category": "Employee", "mode": "JobOnly", "from": "2027-04-01"}}, f.NewID(), 200)
	target := b.call("POST", "/api/v1/departments", departmentCreateBody("RP500", x.bu), f.NewID(), 201)["departmentId"]
	post("trial-start", "2027-04-02", M{"notice": "实际开始"})
	post("grade-records", "2027-04-01", M{"grade": p5, "notice": "个人任用"})
	move := post("transfers", day, M{"department": target, "job": x.job, "notice": "本日最后调动"})
	promo := post("promotions", day, M{"grade": p6, "notice": "本日第二件晋升", "precedes": M{"action": move["action"], "revision": move["revision"]}})
	confirm := post("confirm-employment", day, M{"notice": "原通知载明同日先转正，再晋升，再调动；未具名部门", "precedes": M{"action": promo["action"], "revision": promo["revision"]}})
	c := b.call("GET", "/api/v1/employment-actions/"+confirm["action"].(string), nil, "", 200)["probation"].(M)
	p := b.call("GET", "/api/v1/employment-actions/"+promo["action"].(string), nil, "", 200)
	m := b.call("GET", "/api/v1/employment-actions/"+move["action"].(string), nil, "", 200)
	v := b.call("GET", root+"?businessAt="+day, nil, "", 200)
	var order []string
	for _, entry := range v["actions"].([]any) {
		row := entry.(M)
		if row["businessDate"] == day {
			order = append(order, row["kind"].(string))
		}
	}
	if !slices.Equal(order, []string{"ConfirmEmployment", "Promotion", "Transfer"}) || v["state"].(M)["departmentId"] != target || v["state"].(M)["grade"].(M)["id"] != p6 || v["probation"].(M)["status"] != "confirmed" ||
		c["department"] != x.dep || p["promotion"].(M)["department"] != x.dep || p["previous"].(M)["revision"] != confirm["revision"] || m["previous"].(M)["revision"] != p["action"].(M)["revision"] || p["action"].(M)["revision"] == promo["revision"] {
		t.Fatal("反序通知、准确修订与日状态", order, c, p, m)
	}
	for _, prior := range []M{promo, move} {
		old := b.call("GET", "/api/v1/employment-actions/"+prior["action"].(string)+"?adoptedRef="+prior["revision"].(string), nil, "", 200)
		if old["action"].(M)["revision"] != prior["revision"] || old["previous"] != nil {
			t.Fatal("补录前原件已改变", old)
		}
	}
}

func TestConfirmationFromSecondaryTracksPrimaryAndHonorsNotice(t *testing.T) {
	type M = map[string]any
	x := setupAssignment(t)
	b := x.h.b
	main := x.hire["assignment"].(string)
	day := "2027-07-01"
	view := func(id string) M { return b.call("GET", "/api/v1/assignments/"+id+"?businessAt="+day, nil, "", 200) }
	write := func(id, kind string, body M, status int) M {
		body["expectedSourceVersion"] = view(id)["sourceVersion"]
		return b.call("POST", "/api/v1/assignments/"+id+"/"+kind, body, f.NewID(), status)
	}
	for _, purpose := range []string{"transfer", "secondary"} {
		p := b.call("GET", "/api/v1/assignment-policies", nil, "", 200)
		b.call("POST", "/api/v1/assignment-policies", M{"expectedVersion": p["version"], "row": M{"id": f.NewID(), "purpose": purpose, "category": "Employee", "mode": "JobOnly", "from": "2027-04-01"}}, f.NewID(), 200)
	}
	makeDept := func(code string) string {
		return b.call("POST", "/api/v1/departments", departmentCreateBody(code, x.bu), f.NewID(), 201)["departmentId"].(string)
	}
	secondary := write(main, "additional-assignments", M{"from": "2027-04-02", "parallel": false, "job": x.job, "department": makeDept("TRAIN508"), "notice": "兼任培训"}, 200)["assignment"].(string)
	write(main, "trial-start", M{"businessDate": "2027-04-01", "notice": "试用通知"}, 200)
	dest := makeDept("PROD508")
	move := write(main, "transfers", M{"businessDate": day, "department": dest, "job": x.job, "notice": "先调动"}, 200)
	conf := write(secondary, "confirm-employment", M{"businessDate": day, "department": dest, "afterAction": move["action"], "afterRevision": move["revision"], "notice": "通知同时记载发生部门和先后"}, 200)
	original := func() M {
		return b.call("GET", "/api/v1/employment-actions/"+conf["action"].(string), nil, "", 200)["probation"].(M)
	}
	if ev := original(); ev["department"] != dest || ev["attribution"] != "notice" || ev["afterRevision"] != move["revision"] {
		t.Fatal("通知两项事实", ev)
	}
	path := "actions/" + move["action"].(string) + "/delete"
	headMain, headSecondary := view(main)["sourceVersion"], view(secondary)["sourceVersion"]
	write(main, path, M{"expectedRevision": move["revision"]}, 409)
	if view(main)["sourceVersion"] != headMain || view(secondary)["sourceVersion"] != headSecondary {
		t.Fatal("通知冲突未整体回滚")
	}
	correct := write(main, "transfers/"+move["action"].(string)+"/correct", M{"businessDate": day, "department": dest, "job": x.job, "notice": "同一调动补全依据", "expectedRevision": move["revision"]}, 200)
	if original()["afterRevision"] != correct["revision"] {
		t.Fatal("兼岗办理仍引用旧调动修订")
	}
	var adopted any
	for i, term := range []string{main, secondary} {
		for _, v := range view(term)["actions"].([]any) {
			row := v.(M)
			if row["id"] == conf["action"] {
				if i == 0 {
					adopted = row["revision"]
				} else if row["revision"] != adopted {
					t.Fatal("两个任职未共同采用新转正修订")
				}
			}
		}
	}
	if adopted == nil || adopted == conf["revision"] {
		t.Fatal("未追加关系转正修订")
	}
	later := write(main, "additional-assignments", M{"from": "2027-07-02", "parallel": false, "job": x.job, "department": makeDept("AFTER508"), "notice": "已转正后新增兼任"}, 200)["assignment"].(string)
	latestTerm := b.call("GET", "/api/v1/assignments/"+later+"?businessAt=2027-07-02", nil, "", 200)
	inherited := false
	for _, v := range latestTerm["actions"].([]any) {
		row := v.(M)
		if row["id"] == conf["action"] {
			inherited = row["revision"] == adopted
		}
	}
	if !inherited {
		t.Fatal("新任职没有采用当前的确切转正修订")
	}
	x.h.policy("DELETE FROM access.employment_grant WHERE principal_id=$1 AND action='probation.confirm'", b.subject)
	latest := view(main)["sourceVersion"]
	write(main, "transfers/"+move["action"].(string)+"/correct", M{"businessDate": day, "department": dest, "job": x.job, "notice": "无转正重算权的更正", "expectedRevision": correct["revision"]}, 403)
	if view(main)["sourceVersion"] != latest || original()["afterRevision"] != correct["revision"] {
		t.Fatal("撤权后仍改变了关系来源")
	}
}

func TestLeaveRegisterUsesLastActiveDepartmentAndFieldGrant(t *testing.T) {
	type M = map[string]any
	x := setupAssignment(t)
	b := x.h.b
	root := "/api/v1/assignments/" + x.hire["assignment"].(string)
	view := func() M { return b.call("GET", root+"?businessAt=2027-08-31", nil, "", 200) }
	p := b.call("GET", "/api/v1/assignment-policies", nil, "", 200)
	b.call("POST", "/api/v1/assignment-policies", M{"expectedVersion": p["version"], "row": M{"id": f.NewID(), "purpose": "transfer", "category": "Employee", "mode": "JobOnly", "from": "2027-04-01"}}, f.NewID(), 200)
	dep := departmentCreateBody("EXIT508", x.bu)
	dep["name"] = "离职时部门"
	created := b.call("POST", "/api/v1/departments", dep, f.NewID(), 201)
	id := created["departmentId"].(string)
	b.call("POST", root+"/transfers", M{"expectedSourceVersion": view()["sourceVersion"], "businessDate": "2027-08-01", "job": x.job, "department": id, "notice": "离职前已调动"}, f.NewID(), 200)
	exit := b.call("POST", root+"/leave", M{"expectedSourceVersion": view()["sourceVersion"], "date": "2027-09-01", "meaning": "first-inactive", "notice": "首个不在职日"}, f.NewID(), 200)
	b.call("POST", "/api/v1/departments/"+id+"/records/"+created["targetRecordId"].(string)+"/correct", M{"expectedSourceVersion": created["sourceVersion"], "expectedRecordRevision": created["targetRevision"], "changes": M{"name": "后续核正名称"}}, f.NewID(), 200)
	list := func(department string) M {
		return b.call("GET", "/api/v1/employment-actions?kind=Leave&department="+department, nil, "", 200)
	}
	row := list(id)["rows"].([]any)[0].(M)
	original := b.call("GET", "/api/v1/employment-actions/"+exit["action"].(string), nil, "", 200)
	if list("")["total"] != 1.0 || list(x.dep)["total"] != 0.0 || row["id"] != exit["action"] || row["departmentName"] != "离职时部门" || original["departmentName"] != row["departmentName"] {
		t.Fatal(row, original)
	}
	x.h.policy("UPDATE access.employment_grant SET fields=array_remove(fields,'department') WHERE principal_id=$1 AND action='register.read'", b.subject)
	redacted := list("")
	if redacted["total"] != 1.0 || redacted["rows"].([]any)[0].(M)["department"] != nil || len(redacted["departments"].([]any)) != 0 || list(id)["total"] != 0.0 {
		t.Fatal("发生部门越过字段授权", redacted)
	}
}

func TestAssignmentDefaultDayUsesActualTerm(t *testing.T) {
	type M = map[string]any
	x := setupAssignment(t)
	b := x.h.b
	root := "/api/v1/assignments/" + x.hire["assignment"].(string)
	clock := x.h.f.runtime.Staffing.(app.Service).Store.(*pg.Store)
	clock.Clock = func() time.Time { return time.Date(2027, 3, 1, 12, 0, 0, 0, f.Shanghai) }
	get := func(q string) M { return b.call("GET", root+q, nil, "", 200) }
	future := get("")
	if future["futureTerm"] != true || future["businessAt"] != "2027-04-01" || future["state"].(M)["departmentId"] != x.dep {
		t.Fatal("未来唯一任期", future)
	}
	if explicit := get("?businessAt=2027-03-01"); explicit["state"] != nil || explicit["businessAt"] != "2027-03-01" {
		t.Fatal("明确选日", explicit)
	}
	clock.Clock = func() time.Time { return time.Date(2027, 7, 15, 0, 0, 0, 0, f.Shanghai) }
	active := get("")
	if active["businessAt"] != "2027-07-15" || active["futureTerm"] == true {
		t.Fatal("在职默认日期", active)
	}
	b.call("POST", root+"/leave", M{"date": "2027-08-31", "meaning": "last-active", "notice": "任期结束", "expectedSourceVersion": active["sourceVersion"]}, f.NewID(), 200)
	clock.Clock = func() time.Time { return time.Date(2027, 9, 5, 0, 0, 0, 0, f.Shanghai) }
	ended := get("")
	if ended["lastActive"] != "2027-08-31" || ended["businessAt"] != "2027-08-31" || get("?businessAt=2027-09-01")["state"] != nil {
		t.Fatal("离职间隔回看", ended)
	}
}

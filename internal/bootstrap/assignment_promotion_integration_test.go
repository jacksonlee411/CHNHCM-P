//go:build integration

package bootstrap_test

import (
	"testing"

	f "chnhcm.local/chnhcm/internal/foundation"
)

func TestAssignmentPromotionUsesActualGradeAndOneDailySource(t *testing.T) {
	type M = map[string]any
	for _, mode := range []string{"forward", "reverse", "pair", "separate", "single", "independent"} {
		t.Run(mode, func(t *testing.T) {
			x := setupAssignment(t)
			b := x.h.b
			staff := staffing307{t, x.h.f, b}
			p5 := staff.create("grade", "P5", "")["id"].(string)
			p6 := staff.create("grade", "P6", "")["id"].(string)
			if p5 == p6 {
				t.Fatal("职级目录")
			}
			required := mode == "pair" || mode == "separate" || mode == "single"
			oldSet, newSet := []string{p5, p6}, []string{p5, p6}
			if required {
				oldSet, newSet = []string{p5}, []string{p6}
			}
			staff.correct("job", staff.detail("job", x.job, ""), M{"gradeIds": oldSet}, nil, 200)
			family := staff.create("jobfamily", "PF47", "")["id"].(string)
			targetJob := create310(&staff, "job", "PJ47", newSet, family)["id"].(string)
			depInput := departmentCreateBody("P47", x.bu)
			depInput["name"] = "产品部"
			dep := b.call("POST", "/api/v1/departments", depInput, f.NewID(), 201)["departmentId"].(string)
			pol := b.call("GET", "/api/v1/assignment-policies", nil, "", 200)
			b.call("POST", "/api/v1/assignment-policies", M{"expectedVersion": pol["version"], "row": M{"id": f.NewID(), "category": "Employee", "purpose": "transfer", "mode": "JobOnly", "from": "2027-04-01"}}, f.NewID(), 200)
			assignment := x.hire["assignment"].(string)
			base := "/api/v1/assignments/" + assignment
			day := "2027-07-01"
			view := func(day, source string) M {
				url := base + "?businessAt=" + day
				if source != "" {
					url += "&sourceVersion=" + source
				}
				return b.call("GET", url, nil, "", 200)
			}
			write := func(path, day string, in M, status int) M {
				in["businessDate"] = day
				in["expectedSourceVersion"] = view(day, "")["sourceVersion"]
				return b.call("POST", base+"/"+path, in, f.NewID(), status)
			}
			ref := func(r M) M { return M{"action": r["action"], "revision": r["revision"]} }
			before := view(day, "")
			if before["state"].(M)["grade"] != nil {
				t.Fatal("旧任期被推定职级", before)
			}
			write("grade-records", "2027-04-01", M{"grade": p5, "notice": "原任期个人 P5 任用通知"}, 200)
			write("trial-start", "2027-04-01", M{"notice": "三个月试用安排"}, 200)
			confirm := write("confirm-employment", day, M{"notice": "周宁转正确认"}, 200)
			originalSource := view(day, "")["sourceVersion"].(string)
			check := func(source, department, grade string) {
				t.Helper()
				v := view(day, source)
				state := v["state"].(M)
				if state["departmentId"] != department || state["grade"].(M)["id"] != grade || v["probation"].(M)["status"] != "confirmed" {
					t.Fatal("当日资料不符", v)
				}
			}
			check(originalSource, x.dep, p5)
			transfer := M{"job": targetJob, "department": dep, "notice": "独立调动通知"}
			promote := M{"grade": p6, "notice": "职级任用通知", "previous": ref(confirm)}
			var saved, moved M
			switch mode {
			case "independent":
				saved = write("promotions", day, promote, 200)
				moved = write("transfers", day, transfer, 200)
				check("", dep, p6)
				original := b.call("GET", "/api/v1/employment-actions/"+saved["action"].(string), nil, "", 200)
				if ev := original["promotion"].(M); ev["department"] != "" || ev["departmentName"] != "未确定" {
					t.Fatal("未载先后时推定发生部门", ev)
				}
				for _, department := range []string{x.dep, dep} {
					if b.call("GET", "/api/v1/employment-actions?kind=Promotion&department="+department, nil, "", 200)["total"] != 0.0 {
						t.Fatal("归属不明的晋升计入具名部门")
					}
				}
				return
			case "forward":
				saved = write("promotions", day, promote, 200)
				check("", x.dep, p6)
				transfer["previous"] = ref(saved)
				moved = write("transfers", day, transfer, 200)
			case "reverse":
				moved = write("transfers", day, transfer, 200)
				promote["precedes"] = ref(moved)
				saved = write("promotions", day, promote, 200)
			case "single":
				promote["job"], promote["department"] = targetJob, dep
				saved = write("promotions", day, promote, 200)
			case "pair", "separate":
				write("promotions", day, promote, 409)
				write("transfers", day, transfer, 409)
				check("", x.dep, p5)
				promote["job"], promote["department"] = targetJob, dep
				if mode == "separate" {
					promote["transferNotice"] = "另一份独立调动决定"
				}
				promote["businessDate"], promote["expectedSourceVersion"] = day, originalSource
				x.h.f.exec("ALTER TABLE staffing.promotion_transfer ADD CONSTRAINT test_after_second_fact CHECK(false) NOT VALID")
				b.call("POST", base+"/promotion-transfers", promote, f.NewID(), 503)
				x.h.f.exec("ALTER TABLE staffing.promotion_transfer DROP CONSTRAINT test_after_second_fact")
				if view(day, "")["sourceVersion"] != originalSource {
					t.Fatal("第二项写入失败仍发布来源")
				}
				key := f.NewID()
				saved = b.call("POST", base+"/promotion-transfers", promote, key, 200)
				for _, result := range []M{b.call("POST", base+"/promotion-transfers", promote, key, 200), b.call("POST", "/api/v1/employment-actions/submissions/"+key+"/resolve", nil, "", 200)} {
					if result["action"] != saved["action"] || result["related"].([]any)[0].(M)["action"] != saved["related"].([]any)[0].(M)["action"] {
						t.Fatal("两笔原回执不一致", result)
					}
				}
				moved = saved["related"].([]any)[0].(M)
			}
			check("", dep, p6)
			snapshot := view(day, "")["sourceVersion"].(string)
			sourceAction := "/api/v1/employment-actions/" + saved["action"].(string)
			original := b.call("GET", sourceAction, nil, "", 200)
			ev := original["promotion"].(M)
			if ev["before"].(M)["id"] != p5 || ev["after"].(M)["id"] != p6 || ev["department"] != x.dep || original["previous"].(M)["action"] != confirm["action"] {
				t.Fatal("晋升发生部门或承接事实错误", original)
			}
			if moved != nil {
				m := b.call("GET", "/api/v1/employment-actions/"+moved["action"].(string), nil, "", 200)
				if mode == "pair" || mode == "separate" {
					shared := m["noticeRef"].(M)["revision"] == original["noticeRef"].(M)["revision"]
					if shared != (mode == "pair") {
						t.Fatal("共用及独立通知采用错误", m, original)
					}
				}
				if m["previous"].(M)["revision"] != saved["revision"] {
					t.Fatal("调动未承接准确晋升修订", m)
				}
			}
			register := func(kind, department string) M {
				return b.call("GET", "/api/v1/employment-actions?kind="+kind+"&department="+department+"&businessFrom="+day+"&businessTo="+day, nil, "", 200)
			}
			if register("Promotion", x.dep)["total"] != 1.0 || register("Promotion", dep)["total"] != 0.0 {
				t.Fatal("晋升发生部门筛选")
			}
			if mode == "single" && register("Transfer", dep)["total"] != 0.0 {
				t.Fatal("一份晋升决定被拆成调动")
			}
			revised := write("promotions/"+saved["action"].(string)+"/correct", day, M{"grade": p6, "reason": "专项晋升", "expectedRevision": saved["revision"]}, 200)
			if revised["action"] != saved["action"] {
				t.Fatal("更正改变了业务身份")
			}
			check("", dep, p6)
			end := view(day, "")["sourceVersion"]
			code := 200
			if mode == "pair" || mode == "separate" {
				code = 409
			}
			cancel := M{"expectedSourceVersion": end, "revision": revised["revision"], "reason": "原通知仅载调动，晋升系误录"}
			if mode == "reverse" {
				other := b.call("GET", "/api/v1/employment-actions/"+moved["action"].(string), nil, "", 200)["action"].(M)
				delete(cancel, "reason")
				cancel["basis"] = M{"action": other["id"], "revision": other["revision"]}
			}
			b.call("POST", sourceAction+"/cancel", cancel, f.NewID(), code)
			if code == 200 {
				current := b.call("GET", sourceAction, nil, "", 200)
				if current["promotion"].(M)["priorSource"] != end {
					t.Fatal("撤销前来源丢失", current)
				}
			}
			if code == 409 {
				if view(day, "")["sourceVersion"] != end {
					t.Fatal("单独撤销产生非法岗级")
				}
				check("", dep, p6)
			} else if mode == "single" {
				check("", x.dep, p5)
			} else {
				check("", dep, p5)
			}
			check(snapshot, dep, p6)
		})
	}
}

func TestZhouNingAndChenHeFourTransactions(t *testing.T) {
	type M = map[string]any
	const entry = "2026-09-02"
	const day = "2026-12-02"
	x := setupDatedAssignment(t, entry, "周宁")
	b := x.h.b
	staff := staffing307{t, x.h.f, b}
	p5 := staff.create("grade", "P5", "")["id"].(string)
	p6 := staff.create("grade", "P6", "")["id"].(string)
	staff.correct("job", staff.detail("job", x.job, ""), M{"gradeIds": []string{p5, p6}}, nil, 200)
	for _, purpose := range []string{"secondary", "transfer"} {
		head := b.call("GET", "/api/v1/assignment-policies", nil, "", 200)
		b.call("POST", "/api/v1/assignment-policies", M{"expectedVersion": head["version"], "row": M{"id": f.NewID(), "category": "Employee", "purpose": purpose, "mode": "JobOnly", "from": entry}}, f.NewID(), 200)
	}
	dept := func(code, name string) string {
		body := departmentCreateBody(code, x.bu)
		body["name"] = name
		return b.call("POST", "/api/v1/departments", body, f.NewID(), 201)["departmentId"].(string)
	}
	product, training := dept("P47", "产品部"), dept("T47", "培训部")
	root := x.hire["assignment"].(string)
	view := func(id, date, src string) M {
		url := "/api/v1/assignments/" + id + "?businessAt=" + date
		if src != "" {
			url += "&sourceVersion=" + src
		}
		return b.call("GET", url, nil, "", 200)
	}
	command := func(id, path, date string, in M) M {
		in["expectedSourceVersion"] = view(id, date, "")["sourceVersion"]
		return b.call("POST", "/api/v1/assignments/"+id+"/"+path, in, f.NewID(), 200)
	}
	command(root, "grade-records", entry, M{"businessDate": entry, "grade": p5, "notice": "九月任用原件"})
	command(root, "trial-start", entry, M{"businessDate": entry, "notice": "九月试用安排"})
	command(root, "additional-assignments", entry, M{"from": entry, "parallel": false, "department": x.dep, "job": x.job, "notice": "评审兼任"})
	second := command(root, "additional-assignments", entry, M{"from": entry, "parallel": false, "department": training, "job": x.job, "notice": "培训兼任"})["assignment"].(string)
	// The disposable database clock for registration is controlled independently
	// of the effective date. All four facts are written through their real commands.
	x.h.f.exec("ALTER TABLE staffing.employment_action ALTER COLUMN recorded_at SET DEFAULT ('2026-12-02 09:00:00+08'::timestamptz)")
	confirm := command(second, "confirm-employment", day, M{"businessDate": day, "notice": "试用期满转正"})
	reference := func(a M) M { return M{"action": a["action"], "revision": a["revision"]} }
	promo := command(root, "promotions", day, M{"businessDate": day, "grade": p6, "notice": "晋升通知", "previous": reference(confirm)})
	move := command(root, "transfers", day, M{"businessDate": day, "department": product, "job": x.job, "notice": "调动通知", "previous": reference(promo)})
	old := view(root, day, "")
	state := old["state"].(M)
	if state["grade"].(M)["id"] != p6 || state["departmentId"] != product || old["probation"].(M)["status"] != "confirmed" {
		t.Fatal(old)
	}
	var list []any
	for _, v := range old["actions"].([]any) {
		a := v.(M)
		if a["businessDate"] == day {
			list = append(list, a["id"])
		}
	}
	if len(list) != 3 || list[0] != confirm["action"] || list[1] != promo["action"] || list[2] != move["action"] {
		t.Fatal("时间轴业务先后", list)
	}
	chen := b.call("POST", "/api/v1/persons", M{"submissionId": f.NewID(), "name": "陈禾", "primaryDocument": M{"kind": "passport", "number": "SYN-CHEN47"}}, "", 200)["id"].(string)
	hire := b.call("POST", "/api/v1/hires", M{"person": chen, "arrival": entry, "department": product, "job": x.job, "employer": x.entity, "grade": p5, "notice": "原任通知明载 P5"}, f.NewID(), 200)
	chenID := hire["assignment"].(string)
	if view(chenID, entry, "")["state"].(M)["grade"].(M)["id"] != p5 {
		t.Fatal("原任期没有沿用首聘职级")
	}
	x.h.f.exec("ALTER TABLE staffing.employment_action ALTER COLUMN recorded_at SET DEFAULT ('2026-12-03 09:00:00+08'::timestamptz)")
	change := command(chenID, "location-changes", "2026-12-04", M{"businessDate": "2026-12-04", "location": x.locations[1]})
	count := func(query string, want int) {
		t.Helper()
		v := b.call("GET", "/api/v1/employment-actions?"+query, nil, "", 200)
		if int(v["total"].(float64)) != want {
			t.Fatal(query, want, v)
		}
	}
	count("businessFrom="+day+"&businessTo=2026-12-04", 4)
	count("businessFrom="+day+"&businessTo="+day, 3)
	count("businessFrom=2026-12-03&businessTo=2026-12-03", 0)
	count("kind=DataChange&recordedFrom=2026-12-03&recordedTo=2026-12-03", 1)
	count("kind=Promotion&department="+x.dep, 1)
	count("kind=Promotion&department="+product, 0)
	dir := b.call("POST", "/api/v1/assignments/query", M{"person": x.person, "businessAt": day, "page": 1, "pageSize": 20}, "", 200)
	if dir["total"] != 3.0 || dir["people"] != 1.0 {
		t.Fatal("三份任职未按人去重", dir)
	}
	receipt := b.call("GET", "/api/v1/employment-actions/"+change["action"].(string), nil, "", 200)
	if receipt["action"].(M)["businessDate"] != "2026-12-04" {
		t.Fatal(receipt)
	}
	x.h.policy("UPDATE access.employment_grant SET department_id=$2 WHERE principal_id=$1 AND action='register.read'", b.subject, training)
	count("businessFrom="+day+"&businessTo=2026-12-04", 1)
	count("kind=Promotion&department="+x.dep, 0)
}

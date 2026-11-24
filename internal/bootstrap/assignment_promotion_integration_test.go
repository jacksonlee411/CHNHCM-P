//go:build integration

package bootstrap_test

import (
	"testing"

	f "chnhcm.local/chnhcm/internal/foundation"
)

func TestAssignmentPromotionUsesActualGradeAndOneDailySource(t *testing.T) {
	type M = map[string]any
	for _, mode := range []string{"forward", "reverse", "pair", "single"} {
		t.Run(mode, func(t *testing.T) {
			x := setupAssignment(t)
			b := x.h.b
			staff := staffing307{t, x.h.f, b}
			p5 := staff.create("grade", "P5", "")["id"].(string)
			p6 := staff.create("grade", "P6", "")["id"].(string)
			if p5 == p6 {
				t.Fatal("职级目录")
			}
			required := mode == "pair" || mode == "single"
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
			case "pair":
				write("promotions", day, promote, 409)
				write("transfers", day, transfer, 409)
				check("", x.dep, p5)
				promote["job"], promote["department"] = targetJob, dep
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
			if required && mode == "pair" {
				code = 409
			}
			b.call("POST", sourceAction+"/cancel", M{"expectedSourceVersion": end, "revision": revised["revision"], "reason": "原通知仅载调动，晋升系误录"}, f.NewID(), code)
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

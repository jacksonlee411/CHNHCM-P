//go:build integration

package bootstrap_test

import (
	f "chnhcm.local/chnhcm/internal/foundation"
	"testing"
)

func TestRelationshipProbationSharesOneRealDecision(t *testing.T) {
	type M = map[string]any
	for _, mode := range []string{"primary", "notice", "ambiguous"} {
		t.Run(mode, func(t *testing.T) {
			x := setupAssignment(t)
			b := x.h.b
			root := x.hire["assignment"].(string)
			view := func(id, day string) M {
				return b.call("GET", "/api/v1/assignments/"+id+"?businessAt="+day, nil, "", 200)
			}
			at := "2027-07-01"
			write := func(id, kind, day string, extra M, key string, status int) M {
				input := M{"businessDate": day, "notice": "关系通知 " + mode, "expectedSourceVersion": view(id, day)["sourceVersion"]}
				for k, v := range extra {
					input[k] = v
				}
				return b.call("POST", "/api/v1/assignments/"+id+"/"+kind, input, key, status)
			}
			setPolicy := func(purpose string) {
				policy := b.call("GET", "/api/v1/assignment-policies", nil, "", 200)
				b.call("POST", "/api/v1/assignment-policies", M{"expectedVersion": policy["version"], "row": M{"id": f.NewID(), "category": "Employee", "purpose": purpose, "mode": "JobOnly", "from": "2027-04-01"}}, f.NewID(), 200)
			}
			setPolicy("secondary")
			deptBody := departmentCreateBody("PR46", x.bu)
			deptBody["name"] = "培训授课部"
			training := b.call("POST", "/api/v1/departments", deptBody, f.NewID(), 201)["departmentId"].(string)
			extra := b.call("POST", "/api/v1/assignments/"+root+"/additional-assignments", M{"from": "2027-04-02", "parallel": false, "expectedSourceVersion": view(root, at)["sourceVersion"], "department": training, "job": x.job, "notice": "培训兼任"}, f.NewID(), 200)
			secondary := extra["assignment"].(string)
			write(secondary, "confirm-employment", at, nil, f.NewID(), 409)
			write(root, "trial-start", "2027-04-05", nil, f.NewID(), 200)
			write(secondary, "trial-start", "2027-04-08", nil, f.NewID(), 409)
			term := view(secondary, at)
			if term["probation"].(M)["start"] != "2027-04-05" || term["probation"].(M)["status"] != "probation" {
				t.Fatal(term)
			}
			stored := term["sourceVersion"].(string)
			if mode == "ambiguous" {
				setPolicy("transfer")
				write(root, "transfers", at, M{"job": x.job, "department": training}, f.NewID(), 200)
				x.h.policy("UPDATE access.employment_grant SET department_id=$2 WHERE principal_id=$1 AND action='probation.confirm'", b.subject, training)
				write(secondary, "confirm-employment", at, nil, f.NewID(), 409)
				x.h.policy("UPDATE access.employment_grant SET department_id=NULL WHERE principal_id=$1 AND action='probation.confirm'", b.subject)
			}
			options := M{}
			if mode == "notice" {
				options["department"] = training
			}
			key := f.NewID()
			prior := view(secondary, at)["sourceVersion"]
			if mode == "primary" {
				originalMain := view(root, at)["sourceVersion"]
				x.h.f.exec("ALTER TABLE staffing.employment_outbox ADD CONSTRAINT test_no_relation_event CHECK(false) NOT VALID")
				write(secondary, "confirm-employment", at, options, f.NewID(), 503)
				x.h.f.exec("ALTER TABLE staffing.employment_outbox DROP CONSTRAINT test_no_relation_event")
				if view(root, at)["sourceVersion"] != originalMain || view(secondary, at)["sourceVersion"] != prior {
					t.Fatal("跨任职发布未整笔回滚")
				}
			}

			confirm := write(secondary, "confirm-employment", at, options, key, 200)
			replay := M{"expectedSourceVersion": prior}
			for k, v := range options {
				replay[k] = v
			}
			retry := write(secondary, "confirm-employment", at, replay, key, 200)
			if retry["action"] != confirm["action"] {
				t.Fatal("重送重复办理", retry)
			}
			write(root, "confirm-employment", at, nil, f.NewID(), 409)
			for _, id := range []string{root, secondary} {
				seen := 0
				v := view(id, at)
				if v["probation"].(M)["status"] != "confirmed" {
					t.Fatal(v)
				}
				for _, v := range v["actions"].([]any) {
					if v.(M)["id"] == confirm["action"] {
						seen++
					}
				}
				if seen != 1 {
					t.Fatal("各任职未看到同一笔关系原件", id, seen)
				}
			}
			old := b.call("GET", "/api/v1/assignments/"+secondary+"?businessAt="+at+"&sourceVersion="+stored, nil, "", 200)
			if old["probation"].(M)["status"] != "probation" {
				t.Fatal("旧来源失真", old)
			}
			reg := func(query string) M {
				return b.call("GET", "/api/v1/employment-actions?kind=ConfirmEmployment"+query, nil, "", 200)
			}
			row := reg("")
			if row["total"] != 1.0 {
				t.Fatal("重复计数", row)
			}
			r := row["rows"].([]any)[0].(M)
			if r["id"] != confirm["action"] {
				t.Fatal(r)
			}
			original := b.call("GET", "/api/v1/employment-actions/"+confirm["action"].(string), nil, "", 200)
			ev := original["probation"].(M)
			switch mode {
			case "primary":
				if r["department"] != x.dep || ev["attribution"] != "relationship-primary" {
					t.Fatal(original, r)
				}
			case "notice":
				if r["department"] != training || ev["attribution"] != "notice" {
					t.Fatal(original, r)
				}
			case "ambiguous":
				if r["department"] != nil || ev["attribution"] != "same-day-order-undetermined" || reg("&department=" + training)["total"] != 0.0 {
					t.Fatal(original, r)
				}
			}
			x.h.policy("UPDATE access.employment_grant SET department_id=$2 WHERE principal_id=$1 AND action IN ('read','history','register.read')", b.subject, training)
			if mode == "primary" {
				restricted := reg("")
				if restricted["total"] != 1.0 || restricted["rows"].([]any)[0].(M)["department"] != nil {
					t.Fatal("可见转正摘要误计或泄露主任职", restricted)
				}
				safe := b.call("GET", "/api/v1/employment-actions/"+confirm["action"].(string), nil, "", 200)
				if safe["probation"].(M)["department"] != nil {
					t.Fatal(safe)
				}
			}
		})
	}
}

func TestFirstHireKeepsActualTrialStartAndConcurrentConfirmation(t *testing.T) {
	type M = map[string]any
	x := setupAssignment(t)
	b := x.h.b
	p := b.call("POST", "/api/v1/persons", M{"submissionId": f.NewID(), "name": "带试用入职", "primaryDocument": M{"kind": "passport", "number": "TEST-PROBATION46"}}, "", 200)["id"]
	input := M{"person": p, "employer": x.entity, "department": x.dep, "job": x.job, "arrival": "2027-04-01", "trialStart": "2027-04-08", "notice": "到岗及实际试用日期分别记载"}
	hired := b.call("POST", "/api/v1/hires", input, f.NewID(), 200)
	base := "/api/v1/assignments/" + hired["assignment"].(string)
	state := func(day string) M { return b.call("GET", base+"?businessAt="+day, nil, "", 200) }
	if state("2027-04-07")["probation"].(M)["status"] != "unrecorded" || state("2027-04-08")["probation"].(M)["status"] != "probation" {
		t.Fatal("错误沿用到岗日或丢失试用开始日")
	}
	payload := M{"businessDate": "2027-07-01", "notice": "试用转正决定", "expectedSourceVersion": state("2027-07-01")["sourceVersion"]}
	type result struct {
		status int
		err    error
	}
	done := make(chan result, 2)
	for range 2 {
		go func() {
			status, _, e := b.response("POST", base+"/confirm-employment", payload, f.NewID())
			done <- result{status, e}
		}()
	}
	counts := map[int]int{}
	for range 2 {
		v := <-done
		must(t, v.err)
		counts[v.status]++
	}
	if counts[200] != 1 || counts[409] != 1 {
		t.Fatal(counts)
	}
	orig := b.call("GET", "/api/v1/employment-actions/"+hired["action"].(string), nil, "", 200)
	if orig["trialStart"] != "2027-04-08" {
		t.Fatal("入职原件丢失真实试用日期", orig)
	}
}

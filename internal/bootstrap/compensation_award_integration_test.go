//go:build integration

package bootstrap_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"chnhcm.local/chnhcm/internal/foundation"
	"testing"
)

func TestCompensationAssignmentBasePayThroughHTTP(t *testing.T) {
	const entry = "2026-11-01"
	const first = "2026-11-30"
	const dec = "2026-12-01"
	const jan = "2027-01-01"
	x := setupDatedAssignment(t, entry, "个人月薪联验")
	b, f := x.h.b, x.h.f
	staff := staffing307{t, f, b}
	grade := staff.create("grade", "P5", "")["id"].(string)
	staff.correct("job", staff.detail("job", x.job, ""), planBody{"gradeIds": []string{grade}}, nil, 200)
	id := x.hire["assignment"].(string)
	path := "/api/v1/compensation/assignments/" + id + "/base-pay"
	get := func(day string) planBody { return b.call("GET", path+"/context?businessAt="+day, nil, "", 200) }
	if a := b.call("GET", path+"/access", nil, "", 200); a["available"] != false {
		t.Fatal(a)
	}
	b.call("GET", path, nil, "", 404)
	f.exec("INSERT INTO access.base_pay_grant(instance_id,principal_id,assignment_id,action) SELECT $1,$2,$3,unnest(ARRAY['read','history','first','adjust','correct','withdraw','use'])", f.instance, b.subject, id)
	if got := get(first); len(got["candidates"].([]any)) != 0 {
		t.Fatal(got)
	}
	root := "/api/v1/assignments/" + id
	source := b.call("GET", root+"?businessAt="+entry, nil, "", 200)["sourceVersion"]
	b.call("POST", root+"/grade-records", planBody{"businessDate": entry, "expectedSourceVersion": source, "grade": grade, "notice": "合成 P5 任用原件"}, foundation.NewID(), 200)
	rate := b.call("POST", rates210, planBody{"gradeId": grade, "currency": "CNY", "frequency": "MONTH", "unit": "PER_MONTH", "action": "CREATE", "reason": "建立标准", "effectiveDate": entry, "changes": planBody{"form": "RANGE", "minimum": "10000", "maximum": "15000", "guidance": "12500"}}, foundation.NewID(), 200)
	decRate := b.call("POST", resultPath210(rates210, rate), command210("ADD", rate, dec, planBody{"minimum": "11000", "maximum": "16000", "guidance": "13500"}), foundation.NewID(), 200)
	allowance := fixed210("ALLOW", "岗位津贴", "300", "MONTH")
	basic := sourceRow210(grade, rate, "MONTH")
	input := newPlan210("PRO-M", planBody{"mode": "ENTERPRISE"}, basic, allowance)
	input["effectiveDate"] = entry
	plan := b.call("POST", plans210, input, foundation.NewID(), 200)
	allowance["amount"] = "350"
	b.call("POST", resultPath210(plans210, plan), command210("ADD", plan, jan, planBody{"components": []planBody{basic, allowance}}), foundation.NewID(), 200)
	draft := func(action, day, amount string) planBody {
		v := get(day)
		items := v["candidates"].([]any)
		if len(items) != 1 {
			t.Fatalf("%s candidates: %v", day, v)
		}
		basis := items[0].(planBody)
		return planBody{"action": action, "effectiveDate": day, "expectedVersion": v["version"], "amount": amount, "planId": basis["planId"], "componentId": basis["componentId"], "basisToken": basis["token"], "reason": "个人薪酬决定"}
	}
	for _, amount := range []string{"15500", "", "12000.001"} {
		b.call("POST", path, draft("FIRST", first, amount), foundation.NewID(), 422)
	}
	b.call("POST", path, draft("FIRST", dec, "16250"), foundation.NewID(), 422)
	if n := f.scalar("SELECT count(*) FROM compensation.award_revision WHERE assignment_id=$1", id); n != 0 {
		t.Fatalf("invalid submissions created %d revisions", n)
	}
	one := draft("FIRST", first, "12100")
	key := foundation.NewID()
	// Two authenticated sessions race their exact same timeline expectation.
	other := f.login("admin", f.password)
	type result struct {
		key    string
		status int
		value  planBody
		err    error
	}
	results := make(chan result, 2)
	start := make(chan struct{})
	for _, actor := range []*browser{b, other} {
		go func(actor *browser) {
			<-start
			k := foundation.NewID()
			status, value, e := actor.response("POST", path, one, k)
			results <- result{k, status, value, e}
		}(actor)
	}
	close(start)
	var original planBody
	codes := map[int]int{}
	for range 2 {
		r := <-results
		must(t, r.err)
		codes[r.status]++
		if r.status == 200 {
			key = r.key
			original = r.value["award"].(planBody)
		}
	}
	if codes[200] != 1 || codes[409] != 1 {
		t.Fatalf("concurrent first %v", codes)
	}
	mismatch := draft("ADJUST", dec, "14500")
	b.call("POST", path, mismatch, key, 409)
	b.call("POST", path, one, key, 200)
	b.call("POST", path, one, foundation.NewID(), 409)
	two := draft("CORRECT", first, "12000")
	two["awardId"], two["expectedRevision"], two["note"] = original["awardId"], original["revision"], "与原通知核对"
	corrected := b.call("POST", path, two, foundation.NewID(), 200)["award"].(planBody)
	b.call("POST", path, draft("ADJUST", jan, "13000"), foundation.NewID(), 200)
	for _, sample := range []struct{ day, sum, max string }{{first, "12000.00", "15000.00"}, {dec, "12000.00", "16000.00"}, {jan, "13000.00", "16000.00"}} {
		v := get(sample.day)
		if v["current"].(planBody)["amount"] != sample.sum || v["policy"].(planBody)["maximum"] != sample.max {
			t.Fatal(v)
		}
	}
	origPath := "/api/v1/compensation/awards/" + original["awardId"].(string) + "/revisions/" + original["revision"].(string)
	if p := b.call("GET", origPath, nil, "", 200); p["amount"] != "12100.00" || p["basis"].(planBody)["maximum"] != "15000.00" {
		t.Fatal(p)
	}
	if got := b.call("GET", "/api/v1/compensation/award-receipts/"+key, nil, "", 200); got["award"].(planBody)["revision"] != original["revision"] {
		t.Fatal(got)
	}
	inserted := b.call("POST", path, draft("ADJUST", dec, "15500"), foundation.NewID(), 200)["award"].(planBody)
	del := planBody{"action": "WITHDRAW", "expectedVersion": get(dec)["version"], "effectiveDate": dec, "awardId": inserted["awardId"], "expectedRevision": inserted["revision"], "note": "该笔误建"}
	b.call("POST", path, del, foundation.NewID(), 200)
	if v := get(dec); v["current"].(planBody)["revision"] != corrected["revision"] {
		t.Fatal(v)
	}
	// A recover-before-send race must fence the late packet. The confirmed original can be retried once.
	retryInput := draft("ADJUST", "2026-12-15", "12800")
	retryID := foundation.NewID()
	receiptRoot := "/api/v1/compensation/award-receipts/" + retryID
	b.call("GET", receiptRoot, nil, "", 404)
	resolution := b.call("POST", receiptRoot+"/resolve", planBody{"assignmentId": id, "command": retryInput}, "", 200)
	if resolution["outcome"] != "not_applied" {
		t.Fatal(resolution)
	}
	b.call("POST", path, retryInput, retryID, 409)
	resend := func(body planBody, want int) {
		raw, e := json.Marshal(body)
		must(t, e)
		r, e := http.NewRequest("POST", f.server.URL+path, bytes.NewReader(raw))
		must(t, e)
		r.Header.Set("Origin", f.server.URL)
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-CSRF-Token", b.csrf)
		r.Header.Set("Idempotency-Key", retryID)
		r.Header.Set("X-Award-Confirmation", resolution["confirmation"].(string))
		reply, e := b.client.Do(r)
		must(t, e)
		defer reply.Body.Close()
		if reply.StatusCode != want {
			t.Fatalf("confirmed retry %d want %d", reply.StatusCode, want)
		}
	}
	different := draft("ADJUST", "2026-12-15", "13500")
	resend(different, 409)
	resend(retryInput, 200)
	resend(retryInput, 200)
	saved := b.call("POST", receiptRoot+"/resolve", planBody{}, "", 200)
	if saved["award"].(planBody)["amount"] != "12800.00" {
		t.Fatal(saved)
	}
	// Validate selection against the chosen plan only. A second suitable plan does not invalidate an old draft.
	prior := draft("ADJUST", "2026-12-20", "16000")
	another := newPlan210("DUAL-M", planBody{"mode": "ENTERPRISE"}, basic)
	another["effectiveDate"] = entry
	b.call("POST", plans210, another, foundation.NewID(), 200)
	onlyAllowance := newPlan210("ONLY-A", planBody{"mode": "ENTERPRISE"}, allowance)
	onlyAllowance["effectiveDate"] = entry
	b.call("POST", plans210, onlyAllowance, foundation.NewID(), 200)
	if n := len(get(dec)["candidates"].([]any)); n != 2 {
		t.Fatalf("eligible plans %d", n)
	}
	b.call("POST", path, prior, foundation.NewID(), 200)
	unselected := planBody{"action": "ADJUST", "effectiveDate": "2026-12-24", "amount": "11000", "reason": "决定", "expectedVersion": get(dec)["version"]}
	b.call("POST", path, unselected, foundation.NewID(), 422)
	basis := get(dec)["candidates"].([]any)[0].(planBody)
	unselected["planId"], unselected["componentId"] = basis["planId"], basis["componentId"]
	for _, item := range get("2026-12-24")["candidates"].([]any) {
		v := item.(planBody)
		if v["planId"] == basis["planId"] {
			unselected["basisToken"] = v["token"]
		}
	}
	b.call("POST", path, unselected, foundation.NewID(), 200)
	// Measure actual API reads with retained revisions and eligible candidates.
	startAt := time.Now()
	for range 10 {
		get(dec)
	}
	t.Logf("capacity: one scoped term, three active plans, two candidates, %d active periods, ten reads, one session %s", f.scalar("SELECT count(*) FROM compensation.award_period WHERE assignment_id=$1", id), time.Since(startAt))
	stale := get("2026-12-29")["candidates"].([]any)[0].(planBody)
	change := command210("CORRECT", decRate, "", planBody{"maximum": "15800"})
	raceDay := "2026-12-28"
	raceBasis := get(raceDay)["candidates"].([]any)[0].(planBody)
	raceInput := planBody{"action": "ADJUST", "effectiveDate": raceDay, "expectedVersion": get(dec)["version"], "amount": "15900", "reason": "核对并发制度变更", "planId": raceBasis["planId"], "componentId": raceBasis["componentId"], "basisToken": raceBasis["token"]}
	gate := make(chan struct{})
	sourceResult, awardResult := make(chan result, 1), make(chan result, 1)
	go func() {
		<-gate
		s, v, e := other.response("POST", resultPath210(rates210, rate), change, foundation.NewID())
		sourceResult <- result{status: s, value: v, err: e}
	}()
	go func() {
		<-gate
		s, v, e := b.response("POST", path, raceInput, foundation.NewID())
		awardResult <- result{status: s, value: v, err: e}
	}()
	close(gate)
	sourceChange, raced := <-sourceResult, <-awardResult
	must(t, sourceChange.err)
	must(t, raced.err)
	if sourceChange.status != 200 || raced.status != 409 && raced.status != 200 {
		t.Fatal(sourceChange, raced)
	}
	if raced.status == 200 && raced.value["award"].(planBody)["basis"].(planBody)["maximum"] != "16000.00" {
		t.Fatal("保存时采用了无效的新带", raced.value)
	}

	b.call("POST", path, planBody{"action": "ADJUST", "effectiveDate": "2026-12-29", "expectedVersion": get(dec)["version"], "amount": "14500", "reason": "新决定", "planId": stale["planId"], "componentId": stale["componentId"], "basisToken": stale["token"]}, foundation.NewID(), 409)
	if previous := b.call("GET", receiptRoot, nil, "", 200)["award"].(planBody); previous["basis"].(planBody)["maximum"] != "16000.00" {
		t.Fatal(previous)
	}
	// A second appointment and a rehire are new real terms, including when the record number is reused.
	head := func(term, day string) any {
		return b.call("GET", "/api/v1/assignments/"+term+"?businessAt="+day, nil, "", 200)["sourceVersion"]
	}
	policy := b.call("GET", "/api/v1/assignment-policies", nil, "", 200)
	b.call("POST", "/api/v1/assignment-policies", planBody{"expectedVersion": policy["version"], "row": planBody{"id": foundation.NewID(), "category": "Employee", "purpose": "secondary", "mode": "JobOnly", "from": entry}}, foundation.NewID(), 200)
	secondary := b.call("POST", root+"/additional-assignments", planBody{"expectedSourceVersion": head(id, first), "from": first, "parallel": false, "job": x.job, "department": x.dep, "notice": "另一份任职通知"}, foundation.NewID(), 200)["assignment"].(string)
	secondaryPath := "/api/v1/compensation/assignments/" + secondary + "/base-pay"
	reader := f.login("reader", f.readPassword)
	f.exec("INSERT INTO access.employment_grant(instance_id,principal_id,action,person_id,fields) VALUES($1,$2,'read',$3,ARRAY['arrangement','basis'])", f.instance, reader.subject, x.person)
	f.exec("INSERT INTO access.base_pay_grant(instance_id,principal_id,assignment_id,action) VALUES($1,$2,$3,'read')", f.instance, reader.subject, secondary)
	if v := reader.call("GET", secondaryPath+"?businessAt="+dec, nil, "", 200); v["current"] != nil || len(v["actions"].([]any)) != 0 {
		t.Fatal(v)
	}
	reader.call("GET", path+"?businessAt="+dec, nil, "", 404)
	reader.call("GET", origPath, nil, "", 404)
	reader.call("GET", "/api/v1/compensation/award-receipts/"+key, nil, "", 404)
	secondaryRoot := "/api/v1/assignments/" + secondary
	b.call("POST", secondaryRoot+"/grade-records", planBody{"businessDate": first, "expectedSourceVersion": head(secondary, first), "grade": grade, "notice": "独立兼岗职级决定"}, foundation.NewID(), 200)
	f.exec("INSERT INTO access.base_pay_grant(instance_id,principal_id,assignment_id,action) SELECT $1,$2,$3,unnest(ARRAY['read','history','first','adjust','withdraw','use'])", f.instance, b.subject, secondary)
	secondaryView := func(day string) planBody {
		return b.call("GET", secondaryPath+"/context?businessAt="+day, nil, "", 200)
	}
	secDraft := func(action, day, amount string) planBody {
		v := secondaryView(day)
		for _, item := range v["candidates"].([]any) {
			q := item.(planBody)
			if q["planId"] == plan["objectId"] {
				return planBody{"action": action, "effectiveDate": day, "expectedVersion": v["version"], "amount": amount, "reason": "兼岗决定", "planId": q["planId"], "componentId": q["componentId"], "basisToken": q["token"]}
			}
		}
		t.Fatal("未取得兼岗计划", v)
		return nil
	}
	future := b.call("POST", secondaryPath, secDraft("FIRST", jan, "13000"), foundation.NewID(), 200)["award"].(planBody)
	earlierInput := secDraft("ADJUST", first, "10500")
	earlierKey := foundation.NewID()
	f.exec("ALTER TABLE compensation.award_outbox ADD CONSTRAINT test_atomic_award CHECK(false) NOT VALID")
	b.call("POST", secondaryPath, earlierInput, earlierKey, 503)
	if v := secondaryView(dec); v["current"] != nil {
		t.Fatal("回滚未清除新段", v)
	}
	b.call("GET", "/api/v1/compensation/award-receipts/"+earlierKey, nil, "", 404)
	f.exec("ALTER TABLE compensation.award_outbox DROP CONSTRAINT test_atomic_award")
	earlier := b.call("POST", secondaryPath, earlierInput, earlierKey, 200)["award"].(planBody)
	if v := secondaryView(dec); v["current"].(planBody)["amount"] != "10500.00" || v["policy"].(planBody)["minimum"] != "11000.00" {
		t.Fatal(v)
	}
	b.call("POST", secondaryPath, planBody{"action": "WITHDRAW", "effectiveDate": first, "expectedVersion": secondaryView(first)["version"], "awardId": earlier["awardId"], "expectedRevision": earlier["revision"], "note": "原决定撤销"}, foundation.NewID(), 200)
	if secondaryView(dec)["current"] != nil || secondaryView(jan)["current"].(planBody)["revision"] != future["revision"] {
		t.Fatal("撤去早于首笔的补录篡改了独立后继")
	}
	b.call("POST", "/api/v1/assignments/"+secondary+"/end", planBody{"expectedSourceVersion": head(secondary, jan), "date": jan, "meaning": "last-active", "notice": "兼任届满"}, foundation.NewID(), 200)
	left := b.call("POST", root+"/leave", planBody{"expectedSourceVersion": head(id, "2027-02-01"), "date": "2027-02-01", "meaning": "last-active", "notice": "聘期届满"}, foundation.NewID(), 200)
	if v := get("2027-02-02"); v["active"] != false || v["current"] != nil {
		t.Fatal(v)
	}
	rehire := b.call("POST", "/api/v1/hires", planBody{"person": x.person, "employer": x.entity, "department": x.dep, "job": x.job, "arrival": "2027-03-01", "notice": "再聘通知", "rehireOf": planBody{"action": left["action"], "revision": left["revision"]}}, foundation.NewID(), 200)
	newTerm := rehire["assignment"].(string)
	if newTerm == id || rehire["recordNumber"] != float64(0) {
		t.Fatal(rehire)
	}
	f.exec("INSERT INTO access.base_pay_grant(instance_id,principal_id,assignment_id,action) VALUES($1,$2,$3,'read')", f.instance, b.subject, newTerm)
	if v := b.call("GET", "/api/v1/compensation/assignments/"+newTerm+"/base-pay?businessAt=2027-03-01", nil, "", 200); v["active"] != true || v["current"] != nil {
		t.Fatal(v)
	}
	f.exec("DELETE FROM access.subject_grant WHERE principal_id=$1 AND action IN ('compplan.history.read','graderate.history.read')", b.subject)
	if v := b.call("GET", origPath, nil, "", 200); v["basis"] != nil || v["amount"] != "12100.00" {
		t.Fatal(v)
	}
	f.exec("DELETE FROM access.base_pay_grant WHERE principal_id=$1 AND assignment_id=$2 AND action='read'", b.subject, id)
	b.call("GET", path, nil, "", 404)
	b.call("GET", origPath, nil, "", 404)
	b.call("GET", "/api/v1/compensation/award-receipts/"+key, nil, "", 404)
}

//go:build integration

package bootstrap_test

import (
	"testing"

	app "chnhcm.local/chnhcm/internal/contexts/staffing/application"
	staff "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

func TestStaffing314FunctionRaces(t *testing.T) {
	h := staffingSetup307(t)
	f := h.f
	family := h.create("jobfamily", "HR", "")["id"].(string)
	svc := f.runtime.Staffing.(app.Service)
	creds := geoCredentials203(t, f, "admin", f.password)
	type answer struct {
		result staff.Result
		err    error
	}
	run := func(s app.Service, cmd staff.Command) <-chan answer {
		ch := make(chan answer, 1)
		go func() { r, e := s.Execute(f.ctx, creds, cmd); ch <- answer{r, e} }()
		return ch
	}
	gate := func() *gate307 {
		return &gate307{Store: svc.Store, locked: make(chan struct{}), release: make(chan struct{})}
	}
	for _, first := range []string{"job", "function"} {
		t.Run(first+"_first", func(t *testing.T) {
			function := h.create("jobfunction", "RACE-"+first, "")
			id := function["id"].(string)
			job := h.command307(family, "JOB-"+first)
			job.Input.JobFunctionID = &id
			job.Input.EffectiveDate = "2027-05-01"
			stop := staff.Command{Kind: staff.JobFunction, ObjectID: id, SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Input: staff.Input{Action: "DEACTIVATE", ExpectedSourceVersion: function["headSourceVersion"].(string), AnchorRecordID: rec307(function)["recordId"].(string), EffectiveDate: "2027-05-01", OperationType: "DEACTIVATE", OperationReason: "FUNCTION_CLOSURE"}}
			win, lose := job, stop
			table := "job_function"
			if first == "function" {
				win, lose = stop, job
				table = "job"
			}
			g := gate()
			a := run(app.Service{Store: g}, win)
			wait307(t, f, g.locked)
			b := run(svc, lose)
			blocked307(t, f, g.release)
			close(g.release)
			x, y := <-a, <-b
			must(t, x.err)
			must(t, y.err)
			if x.result.Outcome != "accepted" || y.result.Outcome != "rejected" || y.result.Failure.Code != "M03_REFERENCE_CONFLICT" {
				t.Fatalf("竞争终态 %+v / %+v", x.result, y.result)
			}
			if f.scalar("SELECT count(*) FROM staffing."+table+"_source WHERE submission_id=$1", lose.SubmissionID) != 0 || f.scalar("SELECT count(*) FROM staffing."+table+"_outbox WHERE event->>'submissionId'=$1", lose.SubmissionID) != 0 {
				t.Fatal("拒绝方留下业务写入")
			}
		})
	}
	a := h.create("jobfunction", "SWAP-A", "")["id"].(string)
	b := h.create("jobfunction", "SWAP-B", "")["id"].(string)
	var results [2]<-chan answer
	var gates [2]*gate307
	var jobs [2]staff.Result
	for i, id := range []string{a, b} {
		job := h.command307(family, []string{"JOB-SWAP-A", "JOB-SWAP-B"}[i])
		job.Input.JobFunctionID = &id
		x := <-run(svc, job)
		must(t, x.err)
		if x.result.Outcome != "accepted" {
			t.Fatalf("换类样本 %+v", x.result)
		}
		jobs[i] = x.result
	}
	for i, target := range []string{b, a} {
		j := jobs[i]
		cmd := staff.Command{Kind: staff.Job, ObjectID: j.ObjectID, SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Input: staff.Input{Action: "CORRECT", ExpectedSourceVersion: j.SourceVersion, RecordID: j.SelectedRecordID, ExpectedRecordRevision: j.SelectedRecordRevision, Changes: map[string]*string{"jobFunctionId": &target}}}
		gates[i] = gate()
		results[i] = run(app.Service{Store: gates[i]}, cmd)
	}
	for _, g := range gates {
		wait307(t, f, g.locked)
	}
	for _, g := range gates {
		close(g.release)
	}
	for i, ch := range results {
		x := <-ch
		must(t, x.err)
		if x.result.Outcome != "accepted" {
			t.Fatalf("对调失败 %+v", x.result)
		}
		if f.scalar(`SELECT count(*) FROM staffing.job j JOIN staffing.job_function fn
            ON fn.instance_id=j.instance_id AND fn.enterprise_id=j.enterprise_id AND fn.id=$2
            WHERE j.id=$1 AND j.head#>>'{records,0,data,jobFunctionId}'=fn.id::text
            AND j.head#>'{records,0,functionEvidence}'=jsonb_build_object('functionId',fn.id,'businessAt','2000-01-01',
                'sourceVersion',fn.source_id,'recordId',fn.head#>>'{records,0,id}','revision',fn.head#>>'{records,0,revision}')
            AND j.head#>'{records,0,adoption}'=(SELECT snapshot#>'{records,0,adoption}' FROM staffing.job_source
                WHERE instance_id=j.instance_id AND object_id=j.id AND source_id=$3)`, jobs[i].ObjectID, []string{b, a}[i], jobs[i].SourceVersion) != 1 {
			t.Fatal("对调后的分类或原系列采用不符")
		}
	}
	// A caller without field authority must get 403 even when that exact
	// class is exclusively locked by another PostgreSQL connection.
	view := h.detail("job", jobs[0].ObjectID, "")
	f.exec("UPDATE access.subject_grant SET write_fields=ARRAY['responsibilities'] WHERE principal_id=$1 AND action='job.record.correct'", h.b.subject)
	tx, err := f.admin.Begin(f.ctx)
	must(t, err)
	defer tx.Rollback(f.ctx)
	_, err = tx.Exec(f.ctx, "SELECT pg_advisory_xact_lock($1)", key307("staffing.jobfunction", f.instance, b))
	must(t, err)
	h.correct("job", view, map[string]any{"jobFunctionId": b}, nil, 403)
	must(t, tx.Rollback(f.ctx))
	// The future reference must also survive loss of the disposable list index.
	if f.scalar(`SELECT count(*) FROM staffing.job_read r JOIN staffing.job j ON j.id=r.object_id WHERE j.code='JOB-JOB' AND r.active`) != 1 {
		t.Fatal("缺少未来采用样本")
	}
	f.exec(`DELETE FROM staffing.job_read WHERE object_id IN (SELECT id FROM staffing.job WHERE code='JOB-JOB')`)
	f.exec("DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='job.read'", h.b.subject)
	hidden := h.b.call("GET", "/api/v1/job-functions?q=RACE-job", nil, "", 200)["items"].([]any)[0].(map[string]any)
	v := h.detail("jobfunction", hidden["id"].(string), "")
	h.add("jobfunction", v, "DEACTIVATE", "2027-05-01", nil, 409)
}

func TestStaffing314IndependentJobCandidates(t *testing.T) {
	h := staffingSetup307(t)
	family := h.create("jobfamily", "CAND-FAMILY", "")
	function := h.b.call("POST", base307("jobfunction"), map[string]any{
		"code": "CAND-FUNCTION", "name": "期间职类", "effectiveDate": "2022-01-01",
		"operationType": "CREATE", "operationReason": "NEW_JOB_FUNCTION",
	}, foundation.NewID(), 201)
	const functions = "/api/v1/jobs/function-reference-context"
	const families = "/api/v1/jobs/reference-context"
	check := func(path string, input map[string]any, id any, date string) {
		t.Helper()
		result := h.b.call("POST", path, input, "", 200)
		items := result["items"].([]any)
		if id == nil {
			if len(items) != 0 {
				t.Fatalf("expected no eligible candidate, got %v", items)
			}
			return
		}
		if len(items) != 1 || items[0].(map[string]any)["id"] != id || items[0].(map[string]any)["date"] != date {
			t.Fatalf("candidate for %s on %s: %v", path, date, items)
		}
	}
	input := map[string]any{"action": "CREATE", "effectiveDate": "2021-01-01"}
	check(functions, input, nil, "")
	check(families, input, family["id"], "2021-01-01")
	input["effectiveDate"] = "2022-01-01"
	check(functions, input, function["objectId"], "2022-01-01")

	job := h.create("job", "CAND-JOB", family["id"].(string))
	pair := map[string]any{
		"action": "INSERT_LIFECYCLE_PAIR", "objectId": job["id"], "expectedSourceVersion": job["headSourceVersion"],
		"anchorRecordId": rec307(job)["recordId"], "deactivationDate": "2020-01-01", "reactivationDate": "2022-01-01",
		"referenceQ": "期间职类",
	}
	check(functions, pair, function["objectId"], "2022-01-01")
	pair["reactivationDate"] = "2021-01-01"
	check(functions, pair, nil, "")

	h.f.exec("DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='jobfamily.read'", h.b.subject)
	h.b.call("POST", families, input, "", 403)
	input["referenceQ"] = "期间职类"
	check(functions, input, function["objectId"], "2022-01-01")
	h.f.exec("UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'jobFunctionId') WHERE principal_id=$1 AND action='job.read'", h.b.subject)
	h.b.call("POST", functions, input, "", 403)
}

func TestStaffing314ClassificationQueryAndSelection(t *testing.T) {
	h := staffingSetup307(t)
	family := h.create("jobfamily", "QA-HR", "")["id"].(string)
	job := h.create("job", "QA-JOB", family)
	fn := rec307(job)["jobFunctionId"].(string)
	query := map[string]any{"jobFamilyId": family, "jobFunctionId": fn}
	const jobs = "/api/v1/jobs"
	input := map[string]any{"draftId": foundation.NewID(), "query": query}
	captured := h.b.call("POST", jobs+"/selection/start", input, "", 200)
	input["expectedRevision"] = captured["revision"]
	if captured["count"] != float64(1) || captured["members"].([]any)[0] != job["id"] {
		t.Fatal("交集成员有差异")
	}
	rows := csv307(t, h.b, jobs+"/export", input, 200)
	if len(rows) != 2 || rows[0][3] != "职务系列" || rows[0][8] != "职类" || rows[1][8] != "专业类" {
		t.Fatal("交集导出列或名称有差异")
	}
	query["jobFunctionMissing"] = true
	h.b.call("POST", jobs+"/selection/start", map[string]any{"draftId": foundation.NewID(), "query": query}, "", 400)
	h.b.call("GET", jobs+"?jobFunctionId="+fn+"&jobFunctionMissing=true", nil, "", 400)
	h.b.call("GET", jobs+"?jobFunctionId=", nil, "", 400)
	h.b.call("GET", jobs+"?jobFunctionMissing=invalid", nil, "", 400)
	delete(query, "jobFunctionId")
	h.b.call("POST", jobs+"/selection/resolve", input, "", 409)
	query["jobFunctionId"] = fn
	delete(query, "jobFunctionMissing")
	candidates := map[string]any{"action": "CREATE", "effectiveDate": "2000-01-01", "jobFunctionId": fn, "referenceQ": "其他搜索"}
	selected := h.b.call("POST", jobs+"/function-reference-context", candidates, "", 200)
	if selected["totalCount"] != float64(0) || selected["selectedName"] != "专业类" || selected["selectedAvailable"] != true {
		t.Fatal("搜索候选误清除仍合格的选择")
	}
	h.f.exec("DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='jobfunction.history.read'", h.b.subject)
	selected = h.b.call("POST", jobs+"/function-reference-context", candidates, "", 200)
	if selected["selectedAvailable"] != false || selected["selectedName"] != nil {
		t.Fatal("撤历史权后仍返回候选标签")
	}
	h.f.exec("DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='jobfunction.read'", h.b.subject)
	h.b.call("GET", jobs+"?jobFunctionId="+fn, nil, "", 403)
	h.b.call("GET", jobs+"/"+job["id"].(string)+"/navigation?jobFunctionId="+fn, nil, "", 403)
	h.b.call("POST", jobs+"/selection/resolve", input, "", 403)
	csv307(t, h.b, jobs+"/export", input, 403)
	if total := h.b.call("GET", jobs+"?jobFunctionMissing=true&jobFamilyId="+family, nil, "", 200)["totalCount"]; total != float64(0) {
		t.Fatal("已登记受限职务被当作未登记")
	}
}

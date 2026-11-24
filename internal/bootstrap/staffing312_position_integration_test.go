//go:build integration

package bootstrap_test

import (
	"context"
	"encoding/json"
	"maps"
	"strings"
	"testing"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	app "chnhcm.local/chnhcm/internal/contexts/staffing/application"
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	store "chnhcm.local/chnhcm/internal/contexts/staffing/infra/postgres"
	staff "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

func TestStaffing312Position(t *testing.T) {
	h := staffingSetup307(t)
	reader := h.f.login("reader", h.f.readPassword)
	must(t, h.f.runtime.InitializeCountryCatalog(h.f.ctx, access.LoginInput{Login: "admin", Password: h.f.password}))
	locations := location204{t, h.f, h.b, map[string]string{"MO": geoCountry203(t, h.f, "MO")}}
	location := locations.create("LOC312", "MO", "", "2026-01-01")
	bu := h.b.call("POST", "/api/v1/business-units", createBody("BU312", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	department := h.b.call("POST", "/api/v1/departments", departmentCreateBody("DEP312", bu), foundation.NewID(), 201)
	dep := department["departmentId"].(string)
	family := h.create("jobfamily", "F312", "")
	grade := h.create("grade", "G5", "")["id"].(string)
	job := create310(h, "job", "J312", []string{grade}, family["id"].(string))
	changes := map[string]any{"gradeIds": []string{}}
	const path = "/api/v1/positions"
	body := map[string]any{"code": "P312", "name": "空缺岗位", "effectiveDate": "2027-01-01", "operationType": "CREATE", "operationReason": "NEW_POSITION", "departmentId": dep, "locationId": location["id"], "jobId": job["id"], "gradeId": grade}
	candidate := func(field string, body map[string]any) []any {
		probe := maps.Clone(body)
		probe["action"] = "CREATE"
		return h.b.call("POST", path+"/reference-context?field="+field, probe, "", 200)["items"].([]any)
	}
	requireDate := func(reply map[string]any, date string) {
		t.Helper()
		if !strings.Contains(reply["message"].(string), date) {
			t.Fatalf("期望冲突日 %s，实际 %v", date, reply)
		}
	}
	for _, field := range []string{"departmentId", "locationId", "jobId", "gradeId"} {
		if len(candidate(field, body)) != 1 {
			t.Fatal("missing eligible", field)
		}
	}
	t.Run("同名部门分页与编码脱敏", func(t *testing.T) {
		second := h.b.call("POST", "/api/v1/departments", departmentCreateBody("DEP312B", bu), foundation.NewID(), 201)["departmentId"].(string)
		probe := maps.Clone(body)
		probe["action"] = "CREATE"
		probe["referencePageSize"] = 1
		probe["referenceQ"] = "研发部门"
		list := func() map[string]any {
			return h.b.call("POST", path+"/reference-context?field=departmentId", probe, "", 200)
		}
		first := list()
		probe["referencePage"] = 2
		next := list()
		one, two := first["items"].([]any)[0].(map[string]any), next["items"].([]any)[0].(map[string]any)
		if first["hasMore"] != true || next["hasMore"] != false || one["code"] == two["code"] || one["id"] == two["id"] {
			t.Fatal("同名部门无法翻页区分", first, next)
		}
		probe["referencePage"] = 1
		probe["referenceQ"] = "DEP312B"
		if list()["items"].([]any)[0].(map[string]any)["id"] != second {
			t.Fatal("编码搜索错误")
		}
		h.f.exec("UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'code') WHERE principal_id=$1 AND action='department.history.read'", h.b.subject)
		if len(list()["items"].([]any)) != 0 {
			t.Fatal("隐藏编码可被搜索")
		}
		probe["referenceQ"] = "研发部门"
		if list()["items"].([]any)[0].(map[string]any)["code"] != nil {
			t.Fatal("候选显示了隐藏编码")
		}
		h.f.exec("UPDATE access.subject_grant SET read_fields=array_append(read_fields,'code') WHERE principal_id=$1 AND action='department.history.read'", h.b.subject)
	})
	reader.call("POST", path, body, foundation.NewID(), 403)
	for _, field := range []string{"departmentId", "locationId", "jobId"} {
		empty := maps.Clone(body)
		empty[field] = nil
		h.b.call("POST", path, empty, foundation.NewID(), 400)
	}
	change := h.add("job", job, "ADD", "2027-04-01", changes, 200)
	if len(candidate("gradeId", body)) != 0 || len(candidate("jobId", body)) != 0 {
		t.Fatal("未来移除G5后候选仍可选")
	}
	key := foundation.NewID()
	requireDate(h.b.call("POST", path, body, key, 409), "2027-04-01")
	for _, table := range []string{"position", "position_source", "position_revision", "position_outbox"} {
		if h.f.scalar("SELECT count(*) FROM staffing."+table) != 0 {
			t.Fatal("rejected position wrote business fact", table)
		}
	}
	if recovered := h.b.call("POST", path+"/submissions/"+key+"/resolve", map[string]any{}, "", 200); recovered["outcome"] != "rejected" {
		t.Fatal("rejected receipt lost", recovered)
	}
	ungraded := maps.Clone(body)
	delete(ungraded, "gradeId")
	ungraded["code"] = "UNGRADED312"
	if jobs := candidate("jobId", ungraded); len(jobs) != 1 || jobs[0].(map[string]any)["id"] != job["id"] {
		t.Fatal("不指定职级时职务应仍可选", jobs)
	}
	withoutGrade := h.b.call("POST", path, ungraded, foundation.NewID(), 201)
	withoutGradeDetail := h.detail("position", withoutGrade["objectId"].(string), "")
	if rec307(withoutGradeDetail)["gradeId"] != "" {
		t.Fatal("应可建立无职级的真实岗位")
	}
	h.add("position", withoutGradeDetail, "DEACTIVATE", "2027-02-01", nil, 200)
	latest := h.detail("job", job["id"].(string), change["selectedRecordId"].(string))
	h.correct("job", latest, map[string]any{"gradeIds": []string{grade}}, nil, 200)
	latest = h.detail("job", job["id"].(string), change["selectedRecordId"].(string))
	h.f.exec("DELETE FROM staffing.grade_read WHERE object_id=$1", grade)
	key = foundation.NewID()
	saved := h.b.call("POST", path, body, key, 201)
	id := saved["objectId"].(string)
	if h.b.call("POST", path, body, key, 201)["objectId"] != id {
		t.Fatal("重复建岗")
	}
	position := h.detail("position", id, "")
	if rec307(position)["departmentName"] != "研发部门" {
		t.Fatal(rec307(position))
	}
	h.f.exec("DELETE FROM staffing.position_read WHERE object_id=$1", id)
	requireDate(h.correct("job", latest, changes, nil, 409), "2027-04-01")
	must(t, h.f.runtime.Staffing.(app.Service).Store.(*store.Store).RebuildReads(h.f.ctx))
	if h.f.scalar("SELECT count(*) FROM staffing.position_read WHERE object_id=$1", id) != 1 || h.f.scalar("SELECT count(*) FROM staffing.position_source WHERE object_id=$1", id) != 1 {
		t.Fatal("投影重建改变岗位事实或未恢复索引")
	}
	requireDate(locations.add(location, "DEACTIVATE", "2027-03-01", nil, 409), "2027-03-01")
	requireDate(h.b.call("POST", "/api/v1/departments/"+dep+"/deactivate", map[string]any{"expectedSourceVersion": department["sourceVersion"], "anchorRecordId": department["targetRecordId"], "effectiveDate": "2027-03-01", "operationType": "DEACTIVATE", "operationReason": "ORG_RESTRUCTURING"}, foundation.NewID(), 409), "2027-03-01")
	requireDate(h.add("grade", h.detail("grade", grade, ""), "DEACTIVATE", "2027-03-01", nil, 409), "2027-03-01")
	requireDate(h.add("job", latest, "DEACTIVATE", "2027-05-01", nil, 409), "2027-05-01")
	cleared := h.add("position", position, "ADD", "2027-04-01", map[string]any{"gradeId": nil}, 200)
	h.correct("job", latest, changes, nil, 200)
	after := h.detail("position", id, cleared["selectedRecordId"].(string))
	if rec307(after)["gradeId"] != "" {
		t.Fatal("职级没有清空")
	}
	before := h.detail("position", id, saved["selectedRecordId"].(string))
	if rec307(before)["gradeId"] != grade || rec307(before)["endInclusive"] != "2027-03-31" {
		t.Fatal("清空职级不应改写03-31的有效岗位", rec307(before))
	}
	finite := map[string]any{"objectId": id, "recordId": rec307(before)["recordId"], "expectedSourceVersion": before["headSourceVersion"], "expectedRecordRevision": rec307(before)["revision"], "action": "CORRECT", "changes": map[string]any{"gradeId": grade}}
	available := h.b.call("POST", path+"/reference-context?field=gradeId", finite, "", 200)
	grades := available["items"].([]any)
	if available["endExclusive"] != "2027-04-01" || len(grades) != 1 || grades[0].(map[string]any)["id"] != grade {
		t.Fatal("恰在04-01结束的岗位段应可使用G5", available)
	}
	redate := map[string]any{"objectId": id, "recordId": rec307(after)["recordId"], "expectedSourceVersion": after["headSourceVersion"], "expectedRecordRevision": rec307(after)["revision"], "newEffectiveDate": "2027-04-02", "action": "CORRECT"}
	requireDate(h.b.call("POST", path+"/reference-context?field=gradeId", redate, "", 409), "2027-04-01")
	requireDate(h.correct("position", after, nil, map[string]any{"newEffectiveDate": "2027-04-02"}, 409), "2027-04-01")
	locations.correct(location, map[string]any{"name": "更名后的工作地点"}, nil, 200)
	location = locations.refresh(location)
	if rec307(h.detail("position", id, ""))["locationName"] != "更名后的工作地点" {
		t.Fatal("现行岗位未按查询日期读到工作地点更名")
	}
	h.correct("position", after, map[string]any{"name": "更正岗位名称"}, nil, 200)
	old := reader.call("GET", path+"/"+id+"?sourceVersion="+saved["sourceVersion"].(string), nil, "", 200)
	if rec307(old)["gradeId"] != grade || rec307(old)["name"] != "空缺岗位" || rec307(old)["locationName"] != "地点 LOC312" || rec307(old)["positionAdoptions"] == nil {
		t.Fatal("旧采用错误", rec307(old))
	}
	h.f.exec("DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='location.history.read'", reader.subject)
	h.f.exec("UPDATE access.policy_guard SET policy_no=policy_no+1 WHERE principal_id=$1", reader.subject)
	redacted := reader.call("GET", path+"/"+id+"?sourceVersion="+saved["sourceVersion"].(string), nil, "", 200)
	raw, err := json.Marshal(redacted)
	must(t, err)
	if strings.Contains(string(raw), location["id"].(string)) || strings.Contains(string(raw), "地点 LOC312") || rec307(redacted)["locationName"] != "受限" {
		t.Fatal("撤权后仍可见地点或其原采用")
	}
	raceBody := maps.Clone(body)
	raceBody["locationId"] = locations.create("RACE-LOC312", "MO", "", "2026-01-01")["id"]
	t.Run("两会话同时建岗与移除职级", func(t *testing.T) { positionRace312(t, h, raceBody, family["id"].(string), grade) })
	after = h.detail("position", id, cleared["selectedRecordId"].(string))
	h.f.exec("UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'name') WHERE principal_id=$1 AND action='location.history.read'", h.b.subject)
	h.f.exec("UPDATE access.policy_guard SET policy_no=policy_no+1 WHERE principal_id=$1", h.b.subject)
	h.add("position", after, "DEACTIVATE", "2027-05-01", nil, 403)
	h.f.exec("UPDATE access.subject_grant SET read_fields=array_append(read_fields,'name') WHERE principal_id=$1 AND action='location.history.read'", h.b.subject)
	h.f.exec("UPDATE access.policy_guard SET policy_no=policy_no+1 WHERE principal_id=$1", h.b.subject)
	h.add("position", after, "DEACTIVATE", "2027-05-01", nil, 200)
	locations.add(location, "DEACTIVATE", "2027-05-01", nil, 200)
	simpleJob := h.create("job", "SINGLE312", family["id"].(string))
	ungraded["code"], ungraded["jobId"], ungraded["locationId"] = "SINGLE-P312", simpleJob["id"], raceBody["locationId"]
	h.b.call("POST", path, ungraded, foundation.NewID(), 201)
	requireDate(h.add("job", simpleJob, "DEACTIVATE", "2027-03-01", nil, 409), "2027-03-01")
}

type positionGate312 struct {
	app.Store
	ready, release chan struct{}
}
type positionWrite312 struct {
	app.Unit
	gate positionGate312
}

func (gate positionGate312) Within(ctx context.Context, cred foundation.Credentials, write bool, fn func(app.Unit) error) error {
	return gate.Store.Within(ctx, cred, write, func(unit app.Unit) error { return fn(positionWrite312{unit, gate}) })
}
func (unit positionWrite312) Save(ctx context.Context, plan d.Plan, cmd staff.Command, actor access.OperatorSnapshot) error {
	close(unit.gate.ready)
	select {
	case <-unit.gate.release:
		return unit.Unit.Save(ctx, plan, cmd, actor)
	case <-ctx.Done():
		return ctx.Err()
	}
}

func positionRace312(t *testing.T, h *staffing307, body map[string]any, family, grade string) {
	service := h.f.runtime.Staffing.(app.Service)
	firstAdmin := geoCredentials203(t, h.f, "admin", h.f.password)
	secondAdmin := geoCredentials203(t, h.f, "admin", h.f.password)
	for _, order := range []string{"position", "job"} {
		t.Run(order, func(t *testing.T) {
			job := create310(h, "job", "RACE-J-"+order, []string{grade}, family)
			jobID := job["id"].(string)
			empty := []string{}
			newPosition := staff.Command{Kind: staff.Position, SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Input: staff.Input{Action: "CREATE", Code: "RACE-P-" + order, Name: name307("并发岗位"), DepartmentID: name307(body["departmentId"].(string)), JobID: &jobID, LocationID: name307(body["locationId"].(string)), GradeID: &grade, EffectiveDate: "2027-01-01", OperationType: "CREATE", OperationReason: "NEW_POSITION"}}
			removeGrade := staff.Command{Kind: staff.Job, ObjectID: jobID, SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Input: staff.Input{Action: "ADD", ExpectedSourceVersion: job["headSourceVersion"].(string), AnchorRecordID: rec307(job)["recordId"].(string), EffectiveDate: "2027-04-01", OperationType: "CHANGE", OperationReason: "GRADE_SCOPE_ADJUSTMENT", ChangedGradeIDs: &empty}}
			first, second := newPosition, removeGrade
			if order == "job" {
				first, second = second, first
			}
			gate := positionGate312{service.Store, make(chan struct{}), make(chan struct{})}
			type answer struct {
				result staff.Result
				err    error
			}
			one, two := make(chan answer, 1), make(chan answer, 1)
			go func() {
				result, err := (app.Service{Store: gate}).Execute(h.f.ctx, firstAdmin, first)
				one <- answer{result, err}
			}()
			wait307(t, h.f, gate.ready)
			go func() { result, err := service.Execute(h.f.ctx, secondAdmin, second); two <- answer{result, err} }()
			blocked307(t, h.f, gate.release)
			close(gate.release)
			winner, loser := <-one, <-two
			must(t, winner.err)
			must(t, loser.err)
			if winner.result.Outcome != "accepted" || loser.result.Outcome != "rejected" || loser.result.Failure == nil || loser.result.Failure.Code != "M03_REFERENCE_CONFLICT" || !strings.Contains(loser.result.Failure.Message, "2027-04-01") {
				t.Fatalf("并发结果不符合期间保护: %+v / %+v", winner.result, loser.result)
			}
			positions, jobSources := 0, 2
			if order == "position" {
				positions, jobSources = 1, 1
			}
			if h.f.scalar("SELECT count(*) FROM staffing.position WHERE lower(code)=lower($1)", newPosition.Input.Code) != positions || h.f.scalar("SELECT count(*) FROM staffing.job_outbox WHERE object_id=$1", jobID) != jobSources {
				t.Fatal("失败写入留下事实，或胜方没有发布")
			}
		})
	}
}

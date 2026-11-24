//go:build integration

package bootstrap_test

import (
	f "chnhcm.local/chnhcm/internal/foundation"
	"maps"
	"testing"
)

func TestExemptPositionFirstHireAndReplenishment(t *testing.T) {
	type M = map[string]any
	x := setupAssignment(t)
	b := x.h.b
	policy := b.call("GET", "/api/v1/assignment-policies", nil, "", 200)
	row := policy["rows"].([]any)[0].(M)
	row["mode"] = "PositionExempt"
	b.call("POST", "/api/v1/assignment-policies", M{"expectedVersion": policy["version"], "row": row}, f.NewID(), 200)
	pos := b.call("POST", "/api/v1/positions", M{"code": "P404", "name": "研发计划岗", "effectiveDate": "2027-04-01", "operationType": "CREATE", "operationReason": "NEW_POSITION", "departmentId": x.dep, "jobId": x.job, "locationId": x.locations[0]}, f.NewID(), 201)["objectId"].(string)
	path := "/api/v1/positions/" + pos + "/replenishment"
	status := func(day string) M { return b.call("GET", path+"?businessAt="+day, nil, "", 200) }
	start := status("2027-04-01")
	if start["covered"] != true || start["exempt"] != 0.0 || start["canChange"] != true {
		t.Fatal("新岗位接管与使用", start)
	}
	person := b.call("POST", "/api/v1/persons", M{"submissionId": f.NewID(), "name": "免占编周平", "primaryDocument": M{"kind": "passport", "number": "TEST-P404-1"}}, "", 200)["id"].(string)
	decision := f.NewID()
	edition := b.call("POST", "/api/v1/appointment-decisions", M{"source": decision, "roster": false, "number": "按岗免占编2027", "rows": []M{{"person": person, "position": M{"id": pos, "name": "原通知研发计划岗"}}}}, f.NewID(), 200)
	input := M{"person": person, "employer": x.entity, "position": pos, "arrival": "2027-04-01", "decision": M{"source": decision, "version": edition["version"]}}
	preview := b.call("POST", "/api/v1/hires/preview", input, "", 200)["adoption"].(M)
	segment := preview["segments"].([]any)[0].(M)
	if segment["position"] != pos || segment["departmentId"] != x.dep || segment["jobId"] != x.job || segment["seats"] != 0.0 || segment["positionRevision"] == nil || segment["coverageSource"] == nil || segment["replenishmentSource"] == nil {
		t.Fatal("实际采用", segment)
	}
	input["expected"] = preview
	pause := M{"date": "2027-05-01", "paused": true, "reason": "五月暂停新增任职", "expected": start["version"]}
	key := f.NewID()
	v := b.call("POST", path, pause, key, 200)
	if v["paused"] != true || b.call("POST", path, pause, key, 200)["version"] != v["version"] {
		t.Fatal("补员回执")
	}
	if b.call("POST", "/api/v1/hires", input, f.NewID(), 409)["code"] != "POSITION_PAUSED" {
		t.Fatal("跨期暂停没有整笔拒绝")
	}
	if x.h.f.scalar("SELECT count(*) FROM staffing.employment WHERE person_id=$1", person) != 0 {
		t.Fatal("失败仍建了关系")
	}
	b.call("POST", path, M{"date": "2027-06-01", "paused": false, "reason": "六月恢复", "expected": v["version"]}, f.NewID(), 200)
	delete(input, "expected")
	input["arrival"] = "2027-06-01"
	bad := maps.Clone(input)
	bad["department"] = f.NewID()
	b.call("POST", "/api/v1/hires", bad, f.NewID(), 409)
	key = f.NewID()
	x.h.f.exec("ALTER TABLE staffing.employment_outbox ADD CONSTRAINT fail_position_hire CHECK(false) NOT VALID")
	b.call("POST", "/api/v1/hires", input, f.NewID(), 503)
	x.h.f.exec("ALTER TABLE staffing.employment_outbox DROP CONSTRAINT fail_position_hire")
	if status("2027-06-01")["exempt"] != 0.0 {
		t.Fatal("事件失败留下岗位使用")
	}
	saved := b.call("POST", "/api/v1/hires", input, key, 200)
	if b.call("POST", "/api/v1/hires", input, key, 200)["action"] != saved["action"] || status("2027-06-01")["exempt"] != 1.0 || status("2027-05-31")["exempt"] != 0.0 {
		t.Fatal("免占编使用或重送")
	}
	info := b.call("GET", "/api/v1/assignments/"+saved["assignment"].(string)+"?businessAt=2027-06-01", nil, "", 200)
	state := info["state"].(M)
	if state["position"] != pos || state["positionName"] != "研发计划岗" || state["seats"] != 0.0 {
		t.Fatal(state)
	}
	original := "/api/v1/employment-actions/" + saved["action"].(string)
	staff := staffing307{t, x.h.f, b}
	staff.add("position", staff.detail("position", pos, ""), "DEACTIVATE", "2027-07-01", nil, 409)
	read := b.call("GET", original, nil, "", 200)
	if read["appointment"].(M)["position"].(M)["name"] != "原通知研发计划岗" {
		t.Fatal("原单称谓被改写")
	}
	b.call("POST", original+"/cancel", M{"reason": "未来安排撤回", "revision": saved["revision"]}, f.NewID(), 200)
	if status("2027-06-01")["exempt"] != 0.0 || b.call("POST", "/api/v1/hires", input, key, 200)["action"] != saved["action"] {
		t.Fatal("撤销与恢复")
	}
	if b.call("GET", original, nil, "", 200)["sources"].(M)["segments"].([]any)[0].(M)["position"] != pos {
		t.Fatal("撤销改写旧采用")
	}
}

func TestPositionHireAcrossJobModeAndFutureDepartment(t *testing.T) {
	type M = map[string]any
	x := setupAssignment(t)
	b := x.h.b
	staff := staffing307{t, x.h.f, b}
	path := "/api/v1/assignment-policies"
	policy := b.call("GET", path, nil, "", 200)
	first := policy["rows"].([]any)[0].(M)
	b.call("POST", path, M{"expectedVersion": policy["version"], "insertAfter": first["id"], "row": M{"id": f.NewID(), "category": "Employee", "purpose": "hire", "mode": "PositionExempt", "from": "2027-05-01"}}, f.NewID(), 200)
	dep := b.call("POST", "/api/v1/departments", departmentCreateBody("DP49B", x.bu), f.NewID(), 201)["departmentId"].(string)
	pos := b.call("POST", "/api/v1/positions", M{"code": "P49A", "name": "分段接管岗", "effectiveDate": "2027-04-01", "operationType": "CREATE", "operationReason": "NEW_POSITION", "departmentId": x.dep, "jobId": x.job, "locationId": x.locations[0]}, f.NewID(), 201)["objectId"].(string)
	staff.add("position", staff.detail("position", pos, ""), "ADD", "2027-06-01", M{"departmentId": dep}, 200)
	person := b.call("POST", "/api/v1/persons", M{"submissionId": f.NewID(), "name": "分段周青", "primaryDocument": M{"kind": "passport", "number": "P-49-MIX"}}, "", 200)["id"].(string)
	decision := f.NewID()
	pub := M{"source": decision, "roster": false, "number": "跨模式原件", "rows": []M{{"person": person, "position": M{"id": pos, "name": "原定岗位"}, "department": M{"id": x.dep, "name": "原定部门"}}}}
	v := b.call("POST", "/api/v1/appointment-decisions", pub, f.NewID(), 200)
	cmd := M{"person": person, "employer": x.entity, "department": x.dep, "job": x.job, "position": pos, "arrival": "2027-04-01", "decision": M{"source": decision, "version": v["version"]}}
	if b.call("POST", "/api/v1/hires", cmd, f.NewID(), 409)["code"] != "APPOINTMENT_CHANGED" {
		t.Fatal("原通知部门未与全期核对")
	}
	pub["expected"] = v["version"]
	pub["rows"] = []M{{"person": person, "position": M{"id": pos, "name": "原定岗位"}}}
	v = b.call("POST", "/api/v1/appointment-decisions", pub, f.NewID(), 200)
	cmd["decision"] = M{"source": decision, "version": v["version"]}
	preview := b.call("POST", "/api/v1/hires/preview", cmd, "", 200)["adoption"].(M)
	rows := preview["segments"].([]any)
	if len(rows) != 3 || rows[0].(M)["position"] != "" || rows[1].(M)["position"] != pos || rows[2].(M)["departmentId"] != dep {
		t.Fatal(rows)
	}
	cmd["expected"] = preview
	saved := b.call("POST", "/api/v1/hires", cmd, f.NewID(), 200)
	id := saved["assignment"].(string)
	state := func(day string) M {
		return b.call("GET", "/api/v1/assignments/"+id+"?businessAt="+day, nil, "", 200)["state"].(M)
	}
	if state("2027-04-30")["position"] != nil || state("2027-05-01")["position"] != pos || state("2027-06-01")["departmentId"] != dep {
		t.Fatal("各段实际岗位或部门")
	}
	staff.add("position", staff.detail("position", pos, ""), "ADD", "2027-07-01", M{"departmentId": x.dep}, 409)
	x.h.policy("UPDATE access.employment_grant SET department_id=$2 WHERE principal_id=$1 AND action IN ('hire','read','history')", b.subject, x.dep)
	b.call("GET", "/api/v1/assignments/"+id+"?businessAt=2027-06-01", nil, "", 403)
	original := b.call("GET", "/api/v1/employment-actions/"+saved["action"].(string), nil, "", 200)
	if len(original["sources"].(M)["segments"].([]any)) != 2 {
		t.Fatal("原件披露了未获权的未来部门")
	}
}

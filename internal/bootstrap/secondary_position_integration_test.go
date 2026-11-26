//go:build integration

package bootstrap_test

import (
	"maps"
	"testing"

	f "chnhcm.local/chnhcm/internal/foundation"
)

func TestSecondaryPositionUsesOnlyItsOwnDatedSeat(t *testing.T) {
	type M = map[string]any
	x := setupAssignment(t)
	b, db := x.h.b, x.h.f
	const start = "2027-04-01"
	rule := func(purpose, mode string) {
		rules := b.call("GET", "/api/v1/assignment-policies", nil, "", 200)
		row := M{"category": "Employee", "purpose": purpose, "mode": mode, "id": f.NewID(), "from": start}
		for _, item := range rules["rows"].([]any) {
			v := item.(M)
			if v["purpose"] == purpose {
				row["id"], row["revision"] = v["id"], v["revision"]
			}
		}
		b.call("POST", "/api/v1/assignment-policies", M{"expectedVersion": rules["version"], "row": row}, f.NewID(), 200)
	}
	position := func(code string) string {
		id := b.call("POST", "/api/v1/positions", M{"code": code, "name": "兼岗" + code, "effectiveDate": start, "operationType": "CREATE", "operationReason": "NEW_POSITION", "departmentId": x.dep, "jobId": x.job, "locationId": x.locations[0]}, f.NewID(), 201)["objectId"].(string)
		b.call("POST", "/api/v1/staffing-quotas", M{"positionId": id, "effectiveDate": start, "quota": 3, "reason": "核三编以区分岗位重复与额度不足"}, f.NewID(), 200)
		return id
	}
	rule("hire", "PositionCounted")
	for _, mode := range []string{"PositionExempt", "PositionCounted"} {
		rule("secondary", mode)
		a, z := position("A"+mode), position("B"+mode)
		person := b.call("POST", "/api/v1/persons", M{"submissionId": f.NewID(), "name": "同人两岗" + mode, "primaryDocument": M{"kind": "passport", "number": f.NewID()}}, "", 200)["id"]
		hire := b.call("POST", "/api/v1/hires", M{"person": person, "employer": x.entity, "arrival": start, "position": a, "notice": "主岗任命"}, f.NewID(), 200)
		base := "/api/v1/assignments/" + hire["assignment"].(string)
		version := b.call("GET", base+"?businessAt="+start, nil, "", 200)["sourceVersion"]
		derived := b.call("POST", base+"/additional-arrangement", M{"from": start, "position": z, "parallel": false}, "", 200)
		if derived["positionRequired"] != true || derived["jobRequired"] != false || derived["department"].(M)["id"] != x.dep {
			t.Fatal(derived)
		}
		key := f.NewID()
		cmd := M{"parallel": false, "from": start, "lastDay": "2027-04-30", "position": z, "expectedSourceVersion": version, "notice": "正式兼任另一岗位"}
		wrong := maps.Clone(cmd)
		wrong["department"] = f.NewID()
		checkTargets := func() {
			t.Helper()
			if r := b.call("POST", base+"/additional-assignments", wrong, f.NewID(), 409); r["code"] != "POSITION_TARGET_CHANGED" {
				t.Fatal(r)
			}
		}
		checkTargets()
		db.exec("ALTER TABLE staffing.employment_outbox ADD CONSTRAINT fail_secondary_position CHECK(false) NOT VALID")
		b.call("POST", base+"/additional-assignments", cmd, key, 503)
		db.exec("ALTER TABLE staffing.employment_outbox DROP CONSTRAINT fail_secondary_position")
		added := b.call("POST", base+"/additional-assignments", cmd, key, 200)
		if b.call("POST", base+"/additional-assignments", cmd, key, 200)["assignment"] != added["assignment"] {
			t.Fatal("重送产生第二个兼岗")
		}
		if same := b.call("POST", base+"/additional-assignments", cmd, f.NewID(), 200); same["action"] != added["action"] || same["revision"] != added["revision"] {
			t.Fatal("同一岗位安排未定位原件", same)
		}
		checkTargets()
		confirmed := maps.Clone(cmd)
		confirmed["department"], confirmed["job"] = x.dep, x.job
		if got := b.call("POST", base+"/additional-assignments", confirmed, f.NewID(), 200); got["action"] != added["action"] {
			t.Fatal(got)
		}
		overlap := maps.Clone(cmd)
		overlap["from"] = "2027-04-02"
		if conflict := b.call("POST", base+"/additional-assignments", overlap, f.NewID(), 409); conflict["code"] != "ASSIGNMENT_OVERLAP" {
			t.Fatal(conflict)
		}
		if b.call("GET", "/api/v1/assignments/"+added["assignment"].(string)+"?businessAt="+start, nil, "", 200)["recordNumber"] != 1.0 {
			t.Fatal("兼岗号或失败回滚错误")
		}
		rows := b.call("POST", "/api/v1/assignments/query", M{"person": person, "businessAt": start, "page": 1, "pageSize": 20, "sort": "number"}, "", 200)
		if rows["people"] != 1.0 || rows["total"] != 2.0 {
			t.Fatal("兼岗目录未按人去重", rows)
		}
		names := map[any]bool{}
		for _, v := range rows["rows"].([]any) {
			names[v.(M)["positionName"]] = true
		}
		if !names["兼岗A"+mode] || !names["兼岗B"+mode] {
			t.Fatal("员工列表未显示实际主兼岗位", rows)
		}
		count := func(id, day string) float64 {
			return b.call("GET", "/api/v1/positions/"+id+"/replenishment?businessAt="+day, nil, "", 200)["occupied"].(float64)
		}
		n := 0.0
		if mode == "PositionCounted" {
			n = 1
		}
		if count(a, start) != 1 || count(z, start) != n || count(z, "2027-05-01") != 0 {
			t.Fatal(mode, "主岗、兼岗与期限用量有误")
		}
	}
}

func TestSecondaryReplayChecksOriginalPositionReadFields(t *testing.T) {
	type M = map[string]any
	x := setupAssignment(t)
	b, db := x.h.b, x.h.f
	const start = "2027-04-01"
	const policyPath = "/api/v1/assignment-policies"
	version := func() any { return b.call("GET", policyPath, nil, "", 200)["version"] }
	jobRule := f.NewID()
	b.call("POST", policyPath, M{"expectedVersion": version(), "row": M{"id": jobRule, "category": "Employee", "purpose": "secondary", "mode": "JobOnly", "from": start}}, f.NewID(), 200)
	b.call("POST", policyPath, M{"expectedVersion": version(), "insertAfter": jobRule, "row": M{"id": f.NewID(), "category": "Employee", "purpose": "secondary", "mode": "PositionCounted", "from": "2027-04-10"}}, f.NewID(), 200)
	position := b.call("POST", "/api/v1/positions", M{"code": "SECONDARY509", "name": "兼岗一编", "effectiveDate": start, "operationType": "CREATE", "operationReason": "NEW_POSITION", "departmentId": x.dep, "jobId": x.job, "locationId": x.locations[0]}, f.NewID(), 201)["objectId"].(string)
	b.call("POST", "/api/v1/staffing-quotas", M{"positionId": position, "effectiveDate": start, "quota": 1, "reason": "原兼岗占用最后一编"}, f.NewID(), 200)
	base := "/api/v1/assignments/" + x.hire["assignment"].(string)
	write := base + "/additional-assignments"
	head := b.call("GET", base+"?businessAt="+start, nil, "", 200)["sourceVersion"]
	type issued struct {
		cmd, receipt M
		key          string
		usesPosition bool
	}
	var originals []issued
	for _, example := range []struct {
		from, last                string
		jobPeriod, positionPeriod bool
	}{{start, "2027-04-03", true, false}, {"2027-04-05", "2027-04-12", true, true}, {"2027-04-15", "2027-04-20", false, true}} {
		input := M{"parallel": false, "from": example.from, "lastDay": example.last, "department": x.dep, "job": x.job, "expectedSourceVersion": head, "notice": "按生效期间兼任"}
		if example.positionPeriod {
			input["position"] = position
		}
		key := f.NewID()
		receipt := b.call("POST", write, input, key, 200)
		original := b.call("GET", "/api/v1/employment-actions/"+receipt["action"].(string), nil, "", 200)
		jobSegment, positionSegment := false, false
		for _, row := range original["secondary"].(M)["segments"].([]any) {
			if row.(M)["position"] == position {
				positionSegment = true
			} else {
				jobSegment = true
			}
		}
		if jobSegment != example.jobPeriod || positionSegment != example.positionPeriod {
			t.Fatal("初办采用与试件不符", original)
		}
		originals = append(originals, issued{input, receipt, key, example.positionPeriod})
	}
	for _, day := range []string{"2027-04-10", "2027-04-15"} {
		view := b.call("GET", "/api/v1/positions/"+position+"/replenishment?businessAt="+day, nil, "", 200)
		if view["occupied"] != 1. {
			t.Fatal("原兼岗没有占满岗位", view)
		}
	}
	check := func(row issued, key string) {
		t.Helper()
		got := b.call("POST", write, row.cmd, key, 200)
		if got["action"] != row.receipt["action"] || got["revision"] != row.receipt["revision"] || got["recordNumber"] != row.receipt["recordNumber"] {
			t.Fatal("查回原件不准确", got)
		}
	}
	before := db.scalar("SELECT count(*) FROM staffing.employment_outbox")
	db.exec("ALTER TABLE staffing.employment_outbox ADD CONSTRAINT reject_secondary_replay_event CHECK(false) NOT VALID")
	for _, grant := range []struct{ action, field string }{{"position.read", "departmentId"}, {"position.history.read", "status"}} {
		x.h.policy("UPDATE access.subject_grant SET read_fields=array_remove(read_fields,$3) WHERE principal_id=$1 AND action=$2", b.subject, grant.action, grant.field)
		for _, row := range originals {
			check(row, row.key)
			if !row.usesPosition {
				check(row, f.NewID())
				continue
			}
			if denied := b.call("POST", write, row.cmd, f.NewID(), 403); denied["code"] != "ACCESS_DENIED" || denied["action"] != nil {
				t.Fatal("失去岗位字段后仍披露原件", denied)
			}
		}
		newPeriod := maps.Clone(originals[2].cmd)
		newPeriod["from"], newPeriod["lastDay"] = "2027-04-24", "2027-04-25"
		if denied := b.call("POST", write, newPeriod, f.NewID(), 403); denied["code"] != "ACCESS_DENIED" {
			t.Fatal(denied)
		}
		x.h.policy("UPDATE access.subject_grant SET read_fields=array_append(read_fields,$3) WHERE principal_id=$1 AND action=$2", b.subject, grant.action, grant.field)
		for _, row := range originals {
			check(row, f.NewID())
		}
	}
	if db.scalar("SELECT count(*) FROM staffing.employment_outbox") != before {
		t.Fatal("权限核对或查回原件产生了第二笔人事事件")
	}
}

//go:build integration

package bootstrap_test

import (
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
		db.exec("ALTER TABLE staffing.employment_outbox ADD CONSTRAINT fail_secondary_position CHECK(false) NOT VALID")
		b.call("POST", base+"/additional-assignments", cmd, key, 503)
		db.exec("ALTER TABLE staffing.employment_outbox DROP CONSTRAINT fail_secondary_position")
		added := b.call("POST", base+"/additional-assignments", cmd, key, 200)
		if b.call("POST", base+"/additional-assignments", cmd, key, 200)["assignment"] != added["assignment"] {
			t.Fatal("重送产生第二个兼岗")
		}
		if conflict := b.call("POST", base+"/additional-assignments", cmd, f.NewID(), 409); conflict["code"] != "ASSIGNMENT_OVERLAP" {
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

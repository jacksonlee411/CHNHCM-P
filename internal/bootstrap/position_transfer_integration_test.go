//go:build integration

package bootstrap_test

import (
	f "chnhcm.local/chnhcm/internal/foundation"
	"testing"
)

func TestExemptPositionTransferAndRestore(t *testing.T) {
	type M = map[string]any
	x := setupAssignment(t)
	b := x.h.b
	policy := b.call("GET", "/api/v1/assignment-policies", nil, "", 200)
	b.call("POST", "/api/v1/assignment-policies", M{"expectedVersion": policy["version"], "row": M{"id": f.NewID(), "category": "Employee", "purpose": "transfer", "from": "2027-04-01", "mode": "PositionExempt"}}, f.NewID(), 200)
	makePosition := func(code, dept string) string {
		return b.call("POST", "/api/v1/positions", M{"code": code, "name": "调入岗" + code, "effectiveDate": "2027-04-01", "operationType": "CREATE", "operationReason": "NEW_POSITION", "departmentId": dept, "jobId": x.job, "locationId": x.locations[0]}, f.NewID(), 201)["objectId"].(string)
	}
	dest := b.call("POST", "/api/v1/departments", departmentCreateBody("D49Y", x.bu), f.NewID(), 201)["departmentId"].(string)
	first, second, third := makePosition("MOVEA", x.dep), makePosition("MOVEB", x.dep), makePosition("MOVEC", dest)
	base := "/api/v1/assignments/" + x.hire["assignment"].(string)
	view := func(day string) M { return b.call("GET", base+"?businessAt="+day, nil, "", 200) }
	sub := func(day, id string) M {
		return M{"expectedSourceVersion": view(day)["sourceVersion"], "businessDate": day, "position": id, "notice": "按岗位调入"}
	}
	qty := func(id, day string) any {
		return b.call("GET", "/api/v1/positions/"+id+"/replenishment?businessAt="+day, nil, "", 200)["exempt"]
	}
	day := "2027-06-01"
	mode := b.call("POST", base+"/transfer-arrangement", M{"businessDate": day, "position": first}, "", 200)
	if mode["positionRequired"] != true || mode["jobRequired"] != false || mode["department"].(M)["id"] != x.dep {
		t.Fatal(mode)
	}
	opts := rows400(t, b, base+"/transfer-options", M{"businessDate": day, "kind": "position", "search": "MOVE", "page": 1})
	if len(opts) != 3 {
		t.Fatal(opts)
	}
	record := sub(day, first)
	key := f.NewID()
	got := b.call("POST", base+"/transfers", record, key, 200)
	if b.call("POST", base+"/transfers", record, key, 200)["action"] != got["action"] || view("2027-05-31")["state"].(M)["position"] != nil || qty(first, day) != 1.0 || view(day)["state"].(M)["seats"] != 0.0 {
		t.Fatal("调入与重送")
	}
	july := "2027-07-01"
	move := b.call("POST", base+"/transfers", sub(july, second), f.NewID(), 200)
	old := view(july)
	if old["state"].(M)["position"] != second || qty(first, july) != 0.0 || qty(second, july) != 1.0 {
		t.Fatal("同部门同职务两岗位未交接")
	}
	origPath := "/api/v1/employment-actions/" + move["action"].(string)
	origin := b.call("GET", origPath, nil, "", 200)["transfer"].(M)
	if origin["beforePosition"] != first || origin["after"].(M)["position"] != second {
		t.Fatal(origin)
	}
	repl := "/api/v1/positions/" + first + "/replenishment"
	r := b.call("GET", repl+"?businessAt="+july, nil, "", 200)
	b.call("POST", repl, M{"date": july, "paused": true, "reason": "七月暂停新入", "expected": r["version"]}, f.NewID(), 200)
	reply := b.call("POST", base+"/actions/"+move["action"].(string)+"/delete", M{"expectedSourceVersion": old["sourceVersion"], "expectedRevision": move["revision"]}, f.NewID(), 409)
	if reply["code"] != "POSITION_PAUSED" || view(july)["sourceVersion"] != old["sourceVersion"] {
		t.Fatal(reply)
	}
	fix := sub(july, third)
	fix["expectedRevision"] = move["revision"]
	b.call("POST", base+"/transfers/"+move["action"].(string)+"/correct", fix, f.NewID(), 200)
	now := view(july)["state"].(M)
	if now["position"] != third || now["departmentId"] != dest || qty(second, july) != 0.0 || qty(third, july) != 1.0 {
		t.Fatal(now)
	}
	prior := b.call("GET", base+"?businessAt="+july+"&sourceVersion="+old["sourceVersion"].(string), nil, "", 200)
	original := b.call("GET", origPath+"?adoptedRef="+move["revision"].(string), nil, "", 200)
	if prior["state"].(M)["position"] != second || original["transfer"].(M)["after"].(M)["position"] != second {
		t.Fatal("更正覆盖历史来源")
	}
	x.h.policy("UPDATE access.employment_grant SET department_id=$2 WHERE principal_id=$1 AND action IN ('history','read','transfer')", b.subject, dest)
	disclosed := b.call("GET", origPath, nil, "", 200)["transfer"].(M)
	if disclosed["beforePosition"] != nil || disclosed["beforePositionName"] != nil || disclosed["after"].(M)["position"] != third {
		t.Fatal(disclosed)
	}
}

//go:build integration

package bootstrap_test

import (
	f "chnhcm.local/chnhcm/internal/foundation"
	"strings"
	"testing"
)

func TestInternalSuccessionUsesDedicatedThenRegularSeat(t *testing.T) {
	type M = map[string]any
	x := setupDatedAssignment(t, "2027-03-01", "初聘样本")
	b, db := x.h.b, x.h.f
	entry, day, after := "2027-03-01", "2027-04-01", "2027-04-11"
	for _, purpose := range []string{"hire", "transfer", "recruit"} {
		h := b.call("GET", "/api/v1/assignment-policies", nil, "", 200)
		row := M{"id": f.NewID(), "purpose": purpose, "category": "Employee", "mode": "PositionCounted", "from": entry}
		for _, v := range h["rows"].([]any) {
			if v.(M)["purpose"] == purpose {
				row["id"] = v.(M)["id"]
				row["revision"] = v.(M)["revision"]
			}
		}
		b.call("POST", "/api/v1/assignment-policies", M{"expectedVersion": h["version"], "row": row}, f.NewID(), 200)
	}
	pos := func(code string) string {
		p := b.call("POST", "/api/v1/positions", M{"code": code, "name": "接任" + code, "effectiveDate": entry, "operationType": "CREATE", "operationReason": "NEW_POSITION", "departmentId": x.dep, "jobId": x.job, "locationId": x.locations[0]}, f.NewID(), 201)["objectId"].(string)
		b.call("POST", "/api/v1/staffing-quotas", M{"positionId": p, "effectiveDate": entry, "quota": 1, "reason": "常规一编"}, f.NewID(), 200)
		return p
	}
	person := func(name string) any {
		return b.call("POST", "/api/v1/persons", M{"submissionId": f.NewID(), "name": name, "primaryDocument": M{"kind": "passport", "number": f.NewID()}}, "", 200)["id"]
	}
	hire := func(name, p, end string) string {
		cmd := M{"person": person(name), "employer": x.entity, "arrival": entry, "position": p, "notice": "原安排"}
		if end != "" {
			cmd["lastDay"] = end
		}
		return b.call("POST", "/api/v1/hires", cmd, f.NewID(), 200)["assignment"].(string)
	}
	a, z := pos("SA52"), pos("SB52")
	incoming, incumbent := hire("小林", a, ""), hire("小周", z, "2027-04-10")
	view := func(id string) M { return b.call("GET", "/api/v1/assignments/"+id+"?businessAt="+day, nil, "", 200) }
	usage := func(p, d string) M {
		return b.call("GET", "/api/v1/positions/"+p+"/replenishment?businessAt="+d, nil, "", 200)
	}
	base := "/api/v1/assignments/" + incoming
	choices := rows400(t, b, base+"/succession-options", M{"position": z, "businessDate": day, "page": 1})
	if len(choices) != 1 || choices[0]["assignment"] != incumbent || choices[0]["until"] != after {
		t.Fatal(choices)
	}
	cmd := M{"expectedSourceVersion": view(incoming)["sourceVersion"], "position": z, "businessDate": day, "notice": "小林内部接替小周", "incumbent": M{"assignment": incumbent, "sourceVersion": view(incumbent)["sourceVersion"]}}
	offer := b.call("POST", "/api/v1/recruiting/offers", M{"operation": "issue", "businessDate": day, "basis": "五月已承诺的预留", "terms": M{"number": "S52", "person": person("另一人"), "employer": x.entity, "arrival": after, "reserveFrom": after, "position": z}}, f.NewID(), 200)
	refused := b.call("POST", base+"/transfers", cmd, f.NewID(), 409)
	if !strings.Contains(refused["message"].(string), after) || usage(a, day)["occupied"] != 1. || db.scalar("SELECT count(*) FROM staffing.position_succession") != 0 {
		t.Fatal(refused)
	}
	b.call("POST", "/api/v1/recruiting/offers/"+offer["id"].(string)+"/changes", M{"operation": "release", "expected": offer["version"], "businessDate": after, "basis": "原候选人已释放"}, f.NewID(), 200)
	x.h.policy("UPDATE access.employment_grant SET person_id=$2 WHERE principal_id=$1 AND action='position.handover'", b.subject, view(incoming)["person"])
	b.call("POST", base+"/transfers", cmd, f.NewID(), 403)
	x.h.policy("UPDATE access.employment_grant SET person_id=NULL WHERE principal_id=$1 AND action='position.handover'", b.subject)
	key := f.NewID()
	db.exec("ALTER TABLE staffing.employment_outbox ADD CONSTRAINT fail_succession CHECK(false) NOT VALID")
	b.call("POST", base+"/transfers", cmd, key, 503)
	db.exec("ALTER TABLE staffing.employment_outbox DROP CONSTRAINT fail_succession")
	if usage(a, day)["occupied"] != 1. || db.scalar("SELECT count(*) FROM staffing.position_succession") != 0 {
		t.Fatal("失败留下交接额度")
	}
	result := b.call("POST", base+"/transfers", cmd, key, 200)
	if again := b.call("POST", base+"/transfers", cmd, key, 200); result["action"] != again["action"] {
		t.Fatal(result, again)
	}
	for _, date := range []string{day, "2027-04-10", after} {
		u := usage(z, date)
		special := 1.
		if date == after {
			special = 0
		}
		if u["occupied"] != 1. || u["handoverOccupied"] != special || u["totalOccupied"] != 1+special || u["reserved"] != 0. || u["excess"] != 0. {
			t.Fatal(date, u)
		}
	}
	if usage(a, day)["occupied"] != 0. {
		t.Fatal("原岗未释放")
	}
	original := b.call("GET", "/api/v1/employment-actions/"+result["action"].(string), nil, "", 200)
	succ := original["transfer"].(M)["after"].(M)["succession"].(M)
	if succ["name"] != "小周" || succ["assignment"] != incumbent || succ["until"] != after || succ["sourceVersion"] != view(incumbent)["sourceVersion"] {
		t.Fatal(succ)
	}
}

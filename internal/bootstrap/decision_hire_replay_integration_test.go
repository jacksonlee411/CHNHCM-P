//go:build integration

package bootstrap_test

import (
	"maps"
	"reflect"
	"testing"

	f "chnhcm.local/chnhcm/internal/foundation"
)

func TestFixedTermDecisionRecoversOriginalCountedHire(t *testing.T) {
	type M = map[string]any
	x := lifecycleSample{setupAssignment(t)}
	b, db := x.h.b, x.h.f
	const arrival = "2027-05-01"
	for _, purpose := range []string{"hire", "recruit"} {
		policies := b.call("GET", "/api/v1/assignment-policies", nil, "", 200)
		rule := M{"id": f.NewID(), "purpose": purpose, "category": "Employee", "mode": "PositionCounted", "from": "2027-04-01"}
		for _, item := range policies["rows"].([]any) {
			row := item.(M)
			if row["purpose"] == purpose {
				rule["id"], rule["revision"] = row["id"], row["revision"]
			}
		}
		b.call("POST", "/api/v1/assignment-policies", M{"expectedVersion": policies["version"], "row": rule}, f.NewID(), 200)
	}
	position := b.call("POST", "/api/v1/positions", M{"code": "TERM509", "name": "定期岗位", "effectiveDate": arrival, "operationType": "CREATE", "operationReason": "NEW_POSITION", "departmentId": x.dep, "jobId": x.job, "locationId": x.locations[0]}, f.NewID(), 201)["objectId"]
	b.call("POST", "/api/v1/staffing-quotas", M{"positionId": position, "effectiveDate": arrival, "quota": 1, "reason": "只核一编"}, f.NewID(), 200)
	person := b.call("POST", "/api/v1/persons", M{"submissionId": f.NewID(), "name": "定期任命", "primaryDocument": M{"kind": "passport", "number": f.NewID()}}, "", 200)["id"]
	decision := func(number, day string) M {
		source := f.NewID()
		version := b.call("POST", "/api/v1/appointment-decisions", M{"source": source, "number": number, "roster": false, "rows": []M{{"person": person, "arrival": day, "position": M{"id": position, "name": "定期岗位"}}}}, f.NewID(), 200)["version"]
		return M{"source": source, "version": version}
	}
	cmd := M{"person": person, "employer": x.entity, "position": position, "arrival": arrival, "lastDay": "2027-06-30", "decision": decision("首聘原决定509", arrival)}
	const hires = "/api/v1/hires"
	cmd["expected"] = b.call("POST", hires+"/preview", cmd, "", 200)["adoption"]
	key := f.NewID()
	first := b.call("POST", hires, cmd, key, 200)
	checkOriginal := func(input M, submission string, original M) {
		t.Helper()
		replay := b.call("POST", hires, input, submission, 200)
		if replay["action"] != original["action"] || replay["revision"] != original["revision"] || replay["assignment"] != original["assignment"] {
			t.Fatal("未查回当时的办理、修订及任职", replay)
		}
	}
	outbox := db.scalar("SELECT count(*) FROM staffing.employment_outbox")
	db.exec("ALTER TABLE staffing.employment_outbox ADD CONSTRAINT reject_duplicate_hire CHECK(false) NOT VALID")
	checkOriginal(cmd, key, first)
	checkOriginal(cmd, f.NewID(), first)
	if preview := b.call("POST", hires+"/preview", cmd, "", 200); !reflect.DeepEqual(preview["adoption"], cmd["expected"]) {
		t.Fatal("满编后重读应取得初办时的采用", preview)
	}
	conflict := maps.Clone(cmd)
	conflict["lastDay"] = "2027-06-29"
	if r := b.call("POST", hires, conflict, f.NewID(), 409); r["code"] != "APPOINTMENT_CHANGED" {
		t.Fatal(r)
	}
	x.h.policy("UPDATE access.employment_grant SET department_id=$2 WHERE principal_id=$1 AND action='read'", b.subject, f.NewID())
	b.call("POST", hires, cmd, f.NewID(), 403)
	if r := b.call("POST", hires, cmd, key, 200); r["outcome"] != "accepted" || r["action"] != nil {
		t.Fatal("撤权后原票仍披露了任职", r)
	}
	x.h.policy("UPDATE access.employment_grant SET department_id=NULL WHERE principal_id=$1 AND action='read'", b.subject)
	x.h.policy("UPDATE access.employment_grant SET department_id=$2 WHERE principal_id=$1 AND action='history'", b.subject, f.NewID())
	b.call("POST", hires+"/preview", cmd, "", 403)
	checkOriginal(cmd, f.NewID(), first)
	x.h.policy("UPDATE access.employment_grant SET department_id=NULL WHERE principal_id=$1 AND action='history'", b.subject)
	db.exec("ALTER TABLE staffing.employment_outbox DROP CONSTRAINT reject_duplicate_hire")
	next := M{"person": person, "employer": x.entity, "position": position, "arrival": "2027-09-01", "notice": "定期届满后的新通知"}
	if r := b.call("POST", hires, next, f.NewID(), 409); r["code"] != "HIRE_EXISTING" {
		t.Fatal(r)
	}
	if db.scalar("SELECT count(*) FROM staffing.employment_outbox") != outbox {
		t.Fatal("查回或拒绝产生了人事事件")
	}
	left := x.exit(first["assignment"].(string), "leave", "2027-06-30", "last-active", 200)
	delete(next, "notice")
	next["decision"] = decision("再聘原决定509", "2027-09-01")
	next["rehireOf"] = M{"action": left["action"], "revision": left["revision"]}
	next["expected"] = b.call("POST", hires+"/preview", next, "", 200)["adoption"]
	key = f.NewID()
	back := b.call("POST", hires, next, key, 200)
	if back["assignment"] == first["assignment"] || back["recordNumber"] != 0. {
		t.Fatal(back)
	}
	db.exec("ALTER TABLE staffing.employment_outbox ADD CONSTRAINT reject_repeated_rehire CHECK(false) NOT VALID")
	checkOriginal(next, key, back)
	checkOriginal(next, f.NewID(), back)
}

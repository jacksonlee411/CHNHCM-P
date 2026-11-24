//go:build integration

package bootstrap_test

import (
	"strings"
	"testing"
	"time"

	app "chnhcm.local/chnhcm/internal/contexts/staffing/application"
	pg "chnhcm.local/chnhcm/internal/contexts/staffing/infra/postgres"
	f "chnhcm.local/chnhcm/internal/foundation"
)

func TestActualDutyPreservesFullPositionAndThreeDayExcess(t *testing.T) {
	type M = map[string]any
	x := setupDatedAssignment(t, "2027-03-01", "第四名员工")
	b, db := x.h.b, x.h.f
	const entry, first, last, after = "2027-03-01", "2027-04-01", "2027-04-03", "2027-04-04"
	db.runtime.Staffing.(app.Service).Store.(*pg.Store).Clock = func() time.Time { return time.Date(2027, 4, 4, 12, 0, 0, 0, f.Shanghai) }
	for _, purpose := range []string{"hire", "transfer", "recruit"} {
		rules := b.call("GET", "/api/v1/assignment-policies", nil, "", 200)
		row := M{"id": f.NewID(), "purpose": purpose, "category": "Employee", "mode": "PositionCounted", "from": entry}
		for _, v := range rules["rows"].([]any) {
			if v.(M)["purpose"] == purpose {
				row["id"], row["revision"] = v.(M)["id"], v.(M)["revision"]
			}
		}
		b.call("POST", "/api/v1/assignment-policies", M{"expectedVersion": rules["version"], "row": row}, f.NewID(), 200)
	}
	pos := b.call("POST", "/api/v1/positions", M{"code": "STORE53", "name": "门店值班岗", "effectiveDate": entry, "operationType": "CREATE", "operationReason": "NEW_POSITION", "departmentId": x.dep, "jobId": x.job, "locationId": x.locations[0]}, f.NewID(), 201)["objectId"].(string)
	b.call("POST", "/api/v1/staffing-quotas", M{"positionId": pos, "effectiveDate": entry, "quota": 3, "reason": "门店值班三人"}, f.NewID(), 200)
	person := func(name string) any {
		return b.call("POST", "/api/v1/persons", M{"submissionId": f.NewID(), "name": name, "primaryDocument": M{"kind": "passport", "number": f.NewID()}}, "", 200)["id"]
	}
	for i, name := range []string{"甲", "乙", "丙"} {
		cmd := M{"person": person(name), "employer": x.entity, "position": pos, "arrival": entry, "notice": "门店在岗安排"}
		if i == 2 {
			cmd["lastDay"] = "2027-04-08"
		}
		b.call("POST", "/api/v1/hires", cmd, f.NewID(), 200)
	}
	offer := b.call("POST", "/api/v1/recruiting/offers", M{"operation": "issue", "basis": "下一任录用", "businessDate": entry, "terms": M{"number": "O53", "person": person("后来人"), "employer": x.entity, "arrival": "2027-04-09", "reserveFrom": "2027-04-09", "position": pos}}, f.NewID(), 200)
	usage := func(day string) M {
		return b.call("GET", "/api/v1/positions/"+pos+"/replenishment?businessAt="+day, nil, "", 200)
	}
	base := "/api/v1/assignments/" + x.hire["assignment"].(string)
	view := func(day string) M { return b.call("GET", base+"?businessAt="+day, nil, "", 200) }
	source := view(first)["sourceVersion"]
	ordinary := M{"expectedSourceVersion": source, "position": pos, "businessDate": "2027-04-05", "notice": "拟调入"}
	refused := b.call("POST", base+"/transfers", ordinary, f.NewID(), 409)
	if !strings.Contains(refused["message"].(string), "缺 1") || !strings.Contains(refused["message"].(string), "门店值班岗") {
		t.Fatal(refused)
	}
	dispatch := func(number, day string) M {
		sid := f.NewID()
		v := b.call("POST", "/api/v1/appointment-decisions", M{"source": sid, "number": number, "roster": false, "rows": []M{{"person": x.person, "position": M{"id": pos, "name": "门店值班岗"}, "arrival": day}}}, f.NewID(), 200)
		return M{"source": sid, "version": v["version"]}
	}
	planned := dispatch("未来派岗通知", "2027-04-05")
	paper := M{"dispatch": planned, "issuer": "门店考勤", "number": "K53", "from": first, "lastDay": last, "observedThrough": after}
	b.call("POST", base+"/duty-observations", paper, f.NewID(), 409)
	paper["dispatch"] = dispatch("门店已执行派令", first)
	avail := b.call("GET", base+"/duty-sources", nil, "", 200)
	if len(avail["decisions"].([]any)) != 2 {
		t.Fatal(avail)
	}
	pause := usage(first)
	b.call("POST", "/api/v1/positions/"+pos+"/replenishment", M{"date": first, "expected": pause["version"], "paused": true, "reason": "原政策暂停补员"}, f.NewID(), 200)
	x.h.policy("UPDATE access.employment_grant SET person_id=$2 WHERE principal_id=$1 AND action='duty.correct'", b.subject, f.NewID())
	b.call("POST", base+"/actual-duty", M{"expectedSourceVersion": source, "paper": paper}, f.NewID(), 403)
	if db.scalar("SELECT count(*) FROM staffing.duty_observation") != 0 {
		t.Fatal("越权写入出勤证据")
	}
	x.h.policy("UPDATE access.employment_grant SET person_id=NULL WHERE principal_id=$1 AND action='duty.correct'", b.subject)
	pkey := f.NewID()
	proof := b.call("POST", base+"/duty-observations", paper, pkey, 200)
	if again := b.call("POST", base+"/duty-observations", paper, pkey, 200); again["id"] != proof["id"] {
		t.Fatal(again)
	}
	ref := M{"id": proof["id"], "version": proof["version"]}
	cmd := M{"expectedSourceVersion": source, "observation": ref}
	key := f.NewID()
	db.exec("ALTER TABLE staffing.employment_outbox ADD CONSTRAINT fail_duty CHECK(false) NOT VALID")
	b.call("POST", base+"/actual-duty", cmd, key, 503)
	db.exec("ALTER TABLE staffing.employment_outbox DROP CONSTRAINT fail_duty")
	if usage(first)["occupied"] != 3. || db.scalar("SELECT count(*) FROM staffing.duty_application") != 0 {
		t.Fatal("末步失败残留履职")
	}
	saved := b.call("POST", base+"/actual-duty", cmd, key, 200)
	if again := b.call("POST", base+"/actual-duty", cmd, key, 200); again["action"] != saved["action"] {
		t.Fatal(again)
	}
	for _, day := range []string{first, "2027-04-02", last, after} {
		n := 4.
		if day == after {
			n = 3
		}
		u := usage(day)
		if u["occupied"] != n || u["excess"] != n-3 || u["quota"] != 3. || u["paused"] != true {
			t.Fatal(day, u)
		}
	}
	if usage("2027-04-09")["reserved"] != 1. || offer["version"] == "" {
		t.Fatal("后续另一人预留改变")
	}
	if view(after)["state"].(M)["position"] == pos || view(last)["state"].(M)["position"] != pos {
		t.Fatal("未恢复实际任期原岗")
	}
	action := saved["action"].(string)
	original := b.call("GET", "/api/v1/employment-actions/"+action, nil, "", 200)
	fact := original["transfer"].(M)["after"].(M)["duty"].(M)
	if fact["number"] != "K53" || fact["until"] != after || fact["dispatchNumber"] != "门店已执行派令" {
		t.Fatal(original)
	}
	correction := M{"expectedSourceVersion": view(first)["sourceVersion"], "action": M{"action": action, "revision": saved["revision"]}}
	paper["previous"], paper["from"] = ref, "2027-04-02"
	correction["paper"] = paper
	revised := b.call("POST", base+"/actual-duty", correction, f.NewID(), 200)
	if revised["action"] != action || usage(first)["occupied"] != 3. || usage(last)["occupied"] != 4. {
		t.Fatal("更正漏记来源或占编")
	}
	old := b.call("GET", "/api/v1/employment-actions/"+action+"?adoptedRef="+saved["revision"].(string), nil, "", 200)
	if old["transfer"].(M)["after"].(M)["duty"].(M)["from"] != first {
		t.Fatal(old)
	}
}

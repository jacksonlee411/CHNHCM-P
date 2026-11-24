//go:build integration

package bootstrap_test

import (
	f "chnhcm.local/chnhcm/internal/foundation"
	"reflect"
	"testing"
)

func TestFullPositionsExchangeTogether(t *testing.T) {
	type M = map[string]any
	x := setupAssignment(t)
	b, db := x.h.b, x.h.f
	day, change := "2027-04-01", "2027-05-01"
	for _, purpose := range []string{"hire", "transfer", "recruit"} {
		rules := b.call("GET", "/api/v1/assignment-policies", nil, "", 200)
		row := M{"id": f.NewID(), "purpose": purpose, "category": "Employee", "mode": "PositionCounted", "from": day}
		for _, v := range rules["rows"].([]any) {
			if v.(M)["purpose"] == purpose {
				row["id"] = v.(M)["id"]
				row["revision"] = v.(M)["revision"]
			}
		}
		b.call("POST", "/api/v1/assignment-policies", M{"expectedVersion": rules["version"], "row": row}, f.NewID(), 200)
	}
	holder := func(code string) (string, string) {
		p := b.call("POST", "/api/v1/positions", M{"code": code, "name": "原岗" + code, "effectiveDate": day, "operationType": "CREATE", "operationReason": "NEW_POSITION", "departmentId": x.dep, "jobId": x.job, "locationId": x.locations[0]}, f.NewID(), 201)["objectId"].(string)
		b.call("POST", "/api/v1/staffing-quotas", M{"positionId": p, "effectiveDate": day, "quota": 1, "reason": "各核一编"}, f.NewID(), 200)
		person := b.call("POST", "/api/v1/persons", M{"submissionId": f.NewID(), "name": "原任" + code, "primaryDocument": M{"kind": "passport", "number": "G51-" + code}}, "", 200)["id"]
		h := b.call("POST", "/api/v1/hires", M{"person": person, "employer": x.entity, "position": p, "arrival": day, "notice": "原任通知" + code}, f.NewID(), 200)
		return p, h["assignment"].(string)
	}
	a, first := holder("EX51A")
	z, second := holder("EX51B")
	path := func(id string) string { return "/api/v1/assignments/" + id }
	view := func(id, day string) M { return b.call("GET", path(id)+"?businessAt="+day, nil, "", 200) }
	place := func(id, day string) any { return view(id, day)["state"].(M)["position"] }
	cmd := func(id, peer string) M {
		return M{"expectedSourceVersion": view(id, change)["sourceVersion"], "peerSourceVersion": view(peer, change)["sourceVersion"], "peerAssignment": peer}
	}
	add := cmd(first, second)
	add["businessDate"] = change
	add["notice"] = "甲乙对调通知"
	add["operation"] = "Add"
	dest := path(first) + "/position-exchanges"
	ref := b.call("POST", "/api/v1/persons/queries", M{"keyword": "原任EX51B"}, "", 200)["ref"]
	options := rows400(t, b, path(first)+"/exchange-options", M{"ref": ref, "businessDate": change, "page": 1})
	if len(options) != 1 || options[0]["assignment"] != second || options[0]["targetName"] != "原岗EX51A" {
		t.Fatal(options)
	}
	b.call("POST", path(first)+"/transfers", M{"expectedSourceVersion": add["expectedSourceVersion"], "position": z, "businessDate": change, "notice": "单独调入"}, f.NewID(), 409)
	// A paused incoming position blocks the entire group in either input order.
	repl := "/api/v1/positions/" + z + "/replenishment"
	pause := func(date string, state bool) {
		current := b.call("GET", repl+"?businessAt="+date, nil, "", 200)
		b.call("POST", repl, M{"expected": current["version"], "date": date, "paused": state, "reason": "按日核实能否接任"}, f.NewID(), 200)
	}
	pause("2027-04-15", true)
	if r := b.call("POST", dest, add, f.NewID(), 409); r["code"] != "POSITION_PAUSED" {
		t.Fatal(r)
	}
	pause(change, false)
	key := f.NewID()
	db.exec("ALTER TABLE staffing.transfer_exchange_revision ADD CONSTRAINT fail_pair_revision CHECK(false) NOT VALID")
	b.call("POST", dest, add, key, 503)
	if place(first, change) != a || place(second, change) != z || db.scalar("SELECT count(*) FROM staffing.transfer_exchange") != 0 || db.scalar("SELECT count(*) FROM staffing.hire_terminal WHERE submission_id=$1", key) != 0 {
		t.Fatal("末步失败未整组回滚")
	}
	db.exec("ALTER TABLE staffing.transfer_exchange_revision DROP CONSTRAINT fail_pair_revision")
	rev := cmd(second, first)
	rev["businessDate"] = change
	rev["notice"] = add["notice"]
	rev["operation"] = "Add"
	replies := make(chan jointReply, 2)
	for url, in := range map[string]M{dest: add, path(second) + "/position-exchanges": rev} {
		go func() {
			status, val, e := b.response("POST", url, in, f.NewID())
			replies <- jointReply{status, val, e}
		}()
	}
	r, s := <-replies, <-replies
	if s.status == 200 {
		r, s = s, r
	}
	if r.err != nil || s.err != nil || r.status != 200 || s.status != 409 {
		t.Fatal(r, s)
	}
	if place(first, change) != z || place(second, change) != a || place(first, "2027-04-30") != a || place(second, "2027-04-30") != z {
		t.Fatal("两边未按日互换")
	}
	for _, p := range []string{a, z} {
		u := b.call("GET", "/api/v1/positions/"+p+"/replenishment?businessAt="+change, nil, "", 200)
		if u["occupied"] != 1. || u["reserved"] != 0. {
			t.Fatal(u)
		}
	}
	if len(r.body["related"].([]any)) != 1 || db.scalar("SELECT count(*) FROM staffing.transfer_exchange_revision") != 1 {
		t.Fatal(r)
	}
	original := b.call("GET", "/api/v1/employment-actions/"+r.body["action"].(string), nil, "", 200)
	link := original["exchange"].(M)
	id := original["action"].(M)["assignment"].(string)
	peer := link["peerAssignment"].(string)
	aRef := M{"action": r.body["action"], "revision": r.body["revision"]}
	date := cmd(id, peer)
	date["action"] = aRef
	date["operation"] = "Redate"
	date["businessDate"] = "2027-06-01"
	b.call("POST", path(id)+"/actions/"+r.body["action"].(string)+"/delete", M{"expectedSourceVersion": date["expectedSourceVersion"], "expectedRevision": r.body["revision"]}, f.NewID(), 409)
	x.h.policy("UPDATE access.employment_grant SET person_id=$2 WHERE principal_id=$1 AND action IN ('history','read','transfer','record.redate')", b.subject, original["action"].(M)["person"])
	b.call("POST", path(id)+"/position-exchanges", date, f.NewID(), 403)
	masked := b.call("GET", "/api/v1/employment-actions/"+r.body["action"].(string), nil, "", 200)["exchange"].(M)
	if masked["peerAction"] != nil || masked["peerAssignment"] != nil || masked["canRedate"] != false {
		t.Fatal("越权披露对调成员", masked)
	}
	x.h.policy("UPDATE access.employment_grant SET person_id=NULL WHERE principal_id=$1 AND action IN ('history','read','transfer','record.redate')", b.subject)
	dateKey := f.NewID()
	changed := b.call("POST", path(id)+"/position-exchanges", date, dateKey, 200)
	if place(first, change) != a || place(second, change) != z || place(first, "2027-06-01") != z || place(second, "2027-06-01") != a {
		t.Fatal("改期没有同时生效")
	}
	del := cmd(id, peer)
	del["action"] = M{"action": changed["action"], "revision": changed["revision"]}
	del["operation"] = "Delete"
	b.call("POST", path(id)+"/position-exchanges", del, f.NewID(), 200)
	if place(first, "2027-06-01") != a || place(second, "2027-06-01") != z {
		t.Fatal("撤销后原岗未恢复")
	}
	replay := b.call("POST", path(id)+"/position-exchanges", date, dateKey, 200)
	if !reflect.DeepEqual(replay, changed) || db.scalar("SELECT count(*) FROM staffing.transfer_exchange_revision") != 3 {
		t.Fatal(replay, changed)
	}
	archived := b.call("GET", "/api/v1/employment-actions/"+r.body["action"].(string)+"?adoptedRef="+r.body["revision"].(string), nil, "", 200)
	if !reflect.DeepEqual(archived["exchange"].(M)["peerAction"], link["peerAction"]) || archived["action"].(M)["businessDate"] != change {
		t.Fatal(archived)
	}
}

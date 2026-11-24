//go:build integration

package bootstrap_test

import (
	"maps"
	"testing"

	f "chnhcm.local/chnhcm/internal/foundation"
)

func TestAcceptedOfferConvertsItsOwnReservationOnce(t *testing.T) {
	type M = map[string]any
	x := setupAssignment(t)
	b, db := x.h.b, x.h.f
	const start, arrival = "2027-04-01", "2027-05-07"
	for _, purpose := range []string{"hire", "recruit"} {
		rules := b.call("GET", "/api/v1/assignment-policies", nil, "", 200)
		row := M{"id": f.NewID(), "category": "Employee", "purpose": purpose, "mode": "PositionCounted", "from": start}
		for _, v := range rules["rows"].([]any) {
			if v.(M)["purpose"] == purpose {
				row["id"], row["revision"] = v.(M)["id"], v.(M)["revision"]
			}
		}
		b.call("POST", "/api/v1/assignment-policies", M{"expectedVersion": rules["version"], "row": row}, f.NewID(), 200)
	}
	position := b.call("POST", "/api/v1/positions", M{"code": "HT55", "name": "核编录用岗位", "effectiveDate": start, "operationType": "CREATE", "operationReason": "NEW_POSITION", "departmentId": x.dep, "jobId": x.job, "locationId": x.locations[0]}, f.NewID(), 201)["objectId"].(string)
	b.call("POST", "/api/v1/staffing-quotas", M{"positionId": position, "effectiveDate": start, "quota": 1, "reason": "核一编"}, f.NewID(), 200)
	person := b.call("POST", "/api/v1/persons", M{"submissionId": f.NewID(), "name": "转入职人选", "primaryDocument": M{"kind": "passport", "number": "T06-055"}}, "", 200)["id"].(string)
	issued := b.call("POST", "/api/v1/recruiting/offers", M{"operation": "issue", "businessDate": start, "basis": "正式录用", "terms": M{"number": "OFFER-55", "person": person, "employer": x.entity, "position": position, "arrival": "2027-05-01", "reserveFrom": start}}, f.NewID(), 200)
	path := "/api/v1/recruiting/offers/" + issued["id"].(string)
	state := func(v M) M {
		return b.call("GET", path+"/hire-handoff?sourceVersion="+v["version"].(string), nil, "", 200)
	}
	if state(issued)["canHire"] != false {
		t.Fatal("未接受的录用开放入职")
	}
	accepted := b.call("POST", path+"/changes", M{"operation": "accept", "businessDate": "2027-04-03", "basis": "本人已接受原通知", "expected": issued["version"]}, f.NewID(), 200)
	old := state(accepted)["binding"].(M)
	changed := b.call("POST", path+"/changes", M{"operation": "reschedule", "businessDate": "2027-04-04", "basis": "双方商定推迟到岗", "expected": accepted["version"], "dates": M{"arrival": arrival, "reserveFrom": start}}, f.NewID(), 200)
	if state(accepted)["canHire"] != false || state(changed)["canHire"] != false || b.call("GET", path+"?sourceVersion="+accepted["version"].(string), nil, "", 200)["arrival"] != "2027-05-01" {
		t.Fatal("改期没有保留原约定或仍可消费旧接受")
	}
	b.call("POST", path+"/hire", old, f.NewID(), 409)
	accepted = b.call("POST", path+"/changes", M{"operation": "accept", "businessDate": "2027-04-05", "basis": "本人接受调整后的到岗日", "expected": changed["version"]}, f.NewID(), 200)
	bind := state(accepted)["binding"].(M)
	if bind["arrival"] != arrival {
		t.Fatal("交接沿用了改期前的日期", bind)
	}
	usage := func(day string, reserved, occupied float64) {
		t.Helper()
		v := b.call("GET", "/api/v1/positions/"+position+"/replenishment?businessAt="+day, nil, "", 200)
		if v["reserved"] != reserved || v["occupied"] != occupied {
			t.Fatalf("%s 预留/实占应为 %.0f/%.0f，实际 %v", day, reserved, occupied, v)
		}
	}
	usage(arrival, 1, 0)
	for k, v := range (M{"person": f.NewID(), "employer": f.NewID(), "arrival": "2027-05-02", "position": f.NewID(), "version": issued["version"]}) {
		wrong := maps.Clone(bind)
		wrong[k] = v
		b.call("POST", path+"/hire", wrong, f.NewID(), 409)
	}
	outsider := b.call("POST", "/api/v1/persons", M{"submissionId": f.NewID(), "name": "另一人", "primaryDocument": M{"kind": "passport", "number": "T06-OTHER"}}, "", 200)["id"]
	b.call("POST", "/api/v1/hires", M{"person": outsider, "employer": x.entity, "arrival": arrival, "notice": "拟占已有录用名额", "position": position}, f.NewID(), 409)
	setPaused := func(paused bool) {
		day := arrival
		if paused {
			day = "2027-04-25"
		}
		control := "/api/v1/positions/" + position + "/replenishment"
		current := b.call("GET", control+"?businessAt="+day, nil, "", 200)
		b.call("POST", control, M{"date": day, "expected": current["version"], "paused": paused, "reason": "核对已接受录用及新预留"}, f.NewID(), 200)
	}
	setPaused(true)
	bind = state(accepted)["binding"].(M)
	db.exec("ALTER TABLE recruiting.hire_handoff ADD CONSTRAINT fail_handoff CHECK(false) NOT VALID")
	key := f.NewID()
	b.call("POST", path+"/hire", bind, key, 503)
	db.exec("ALTER TABLE recruiting.hire_handoff DROP CONSTRAINT fail_handoff")
	usage(arrival, 1, 0)
	if db.scalar("SELECT count(*) FROM staffing.hired_offer") != 0 || b.call("GET", path, nil, "", 200)["version"] != accepted["version"] {
		t.Fatal("失败留下入职采用或录用修订")
	}
	replies := make(chan jointReply, 2)
	for _, k := range []string{key, f.NewID()} {
		go func() {
			status, v, err := b.response("POST", path+"/hire", bind, k)
			replies <- jointReply{status, v, err}
		}()
	}
	one, two := <-replies, <-replies
	if one.err != nil || two.err != nil || one.status != 200 || two.status != 200 || one.body["action"] != two.body["action"] || db.scalar("SELECT count(*) FROM recruiting.hire_handoff") != 1 {
		t.Fatal(one, two)
	}
	result := one.body
	usage("2027-04-30", 1, 0)
	usage(arrival, 0, 1)
	action := "/api/v1/employment-actions/" + result["action"].(string)
	original := func() M { return b.call("GET", action, nil, "", 200)["sources"].(M) }
	if original()["offer"].(M)["version"] != accepted["version"] || state(accepted)["assignment"] != result["assignment"] || b.call("GET", path, nil, "", 200)["status"] != "converted" {
		t.Fatal("双向准确来源不符")
	}
	cancelKey := f.NewID()
	cancelCmd := M{"revision": result["revision"], "reason": "本人提前通知不再到岗"}
	db.exec("ALTER TABLE recruiting.handoff_cancellation ADD CONSTRAINT fail_cancel_handoff CHECK(false) NOT VALID")
	b.call("POST", action+"/cancel", cancelCmd, cancelKey, 503)
	db.exec("ALTER TABLE recruiting.handoff_cancellation DROP CONSTRAINT fail_cancel_handoff")
	usage(arrival, 0, 1)
	ended := b.call("POST", action+"/cancel", cancelCmd, cancelKey, 200)
	current := b.call("GET", path, nil, "", 200)
	if current["status"] != "issued" || state(current)["assignment"] != result["assignment"] {
		t.Fatal(current)
	}
	usage(arrival, 0, 0)
	usage("2027-04-30", 1, 0)
	b.call("POST", path+"/changes", M{"operation": "reserve", "businessDate": "2027-04-15", "basis": "招聘重新取得一份名额", "expected": current["version"]}, f.NewID(), 409)
	setPaused(false)
	b.call("POST", path+"/changes", M{"operation": "reserve", "businessDate": "2027-04-15", "basis": "招聘重新取得一份名额", "expected": current["version"]}, f.NewID(), 200)
	for _, k := range []string{key, f.NewID()} {
		if again := b.call("POST", path+"/hire", bind, k, 200); again["action"] != result["action"] {
			t.Fatal("重送产生新入职", again)
		}
	}
	if b.call("POST", action+"/cancel", cancelCmd, cancelKey, 200)["action"] != ended["action"] {
		t.Fatal("重送产生新撤销")
	}
	usage(arrival, 1, 0)
	x.h.policy("DELETE FROM access.offer_grant WHERE principal_id=$1 AND action='read'", b.subject)
	b.call("GET", path+"/hire-handoff?sourceVersion="+accepted["version"].(string), nil, "", 403)
	if original()["offer"] != nil {
		t.Fatal("撤回录用读取权后仍显示来源编号")
	}
}

func TestOfferHireWithoutReservedSeatsAndIncompleteIdentity(t *testing.T) {
	type M = map[string]any
	x := setupAssignment(t)
	b := x.h.b
	const day = "2027-04-01"
	position := b.call("POST", "/api/v1/positions", M{"code": "HT55FREE", "name": "免占录用岗位", "effectiveDate": day, "operationType": "CREATE", "operationReason": "NEW_POSITION", "departmentId": x.dep, "jobId": x.job, "locationId": x.locations[0]}, f.NewID(), 201)["objectId"].(string)
	for _, mode := range []string{"JobOnly", "PositionExempt"} {
		for _, purpose := range []string{"hire", "recruit"} {
			rules := b.call("GET", "/api/v1/assignment-policies", nil, "", 200)
			row := M{"id": f.NewID(), "category": "Employee", "purpose": purpose, "from": day, "mode": mode}
			for _, v := range rules["rows"].([]any) {
				if v.(M)["purpose"] == purpose {
					row["id"], row["revision"] = v.(M)["id"], v.(M)["revision"]
				}
			}
			b.call("POST", "/api/v1/assignment-policies", M{"expectedVersion": rules["version"], "row": row}, f.NewID(), 200)
		}
		for _, complete := range []bool{false, true} {
			in := M{"submissionId": f.NewID(), "name": "录用人选" + mode}
			if complete {
				in["primaryDocument"] = M{"kind": "passport", "number": f.NewID()}
			}
			person := b.call("POST", "/api/v1/persons", in, "", 200)["id"]
			terms := M{"number": f.NewID(), "person": person, "employer": x.entity, "department": x.dep, "job": x.job, "arrival": day, "reserveFrom": day}
			if mode == "PositionExempt" {
				terms["position"] = position
				delete(terms, "department")
				delete(terms, "job")
			}
			issued := b.call("POST", "/api/v1/recruiting/offers", M{"operation": "issue", "businessDate": day, "basis": "零份额录用通知", "terms": terms}, f.NewID(), 200)
			path := "/api/v1/recruiting/offers/" + issued["id"].(string)
			accepted := b.call("POST", path+"/changes", M{"operation": "accept", "businessDate": day, "basis": "接受原录用", "expected": issued["version"]}, f.NewID(), 200)
			view := b.call("GET", path+"/hire-handoff?sourceVersion="+accepted["version"].(string), nil, "", 200)
			if !complete {
				if view["canHire"] != false || view["binding"] != nil || len(view["missing"].([]any)) == 0 {
					t.Fatal("资料未齐仍可转入职", view)
				}
				cmd := M{"person": person, "employer": x.entity, "version": accepted["version"], "department": x.dep, "job": x.job, "arrival": day}
				if mode == "PositionExempt" {
					cmd["position"] = position
				}
				b.call("POST", path+"/hire", cmd, f.NewID(), 409)
				continue
			}
			result := b.call("POST", path+"/hire", view["binding"], f.NewID(), 200)
			original := b.call("GET", "/api/v1/employment-actions/"+result["action"].(string), nil, "", 200)
			for _, v := range original["sources"].(M)["segments"].([]any) {
				if v.(M)["seats"] != 0.0 {
					t.Fatal(mode, "零份额转入职产生占用", v)
				}
			}
			if mode == "PositionExempt" {
				usage := b.call("GET", "/api/v1/positions/"+position+"/replenishment?businessAt="+day, nil, "", 200)
				if usage["reserved"] != 0.0 || usage["occupied"] != 0.0 {
					t.Fatal("免占编份额", usage)
				}
			}
		}
	}
}

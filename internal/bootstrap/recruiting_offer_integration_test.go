//go:build integration

package bootstrap_test

import (
	f "chnhcm.local/chnhcm/internal/foundation"
	"maps"
	"testing"
	"time"
)

func TestRecruitingOfferReservationAndDecision(t *testing.T) {
	type M = map[string]any
	x := setupAssignment(t)
	b := x.h.b
	p := b.call("GET", "/api/v1/assignment-policies", nil, "", 200)
	b.call("POST", "/api/v1/assignment-policies", M{"expectedVersion": p["version"], "row": M{"id": f.NewID(), "category": "Employee", "purpose": "recruit", "mode": "PositionCounted", "from": "2027-04-01"}}, f.NewID(), 200)
	position := b.call("POST", "/api/v1/positions", M{"code": "R500", "name": "招聘有编岗", "effectiveDate": "2027-04-01", "operationType": "CREATE", "operationReason": "NEW_POSITION", "departmentId": x.dep, "jobId": x.job, "locationId": x.locations[0]}, f.NewID(), 201)["objectId"].(string)
	b.call("POST", "/api/v1/staffing-quotas", M{"positionId": position, "effectiveDate": "2027-04-01", "quota": 1, "reason": "一份正式名额"}, f.NewID(), 200)
	person := b.call("POST", "/api/v1/persons", M{"submissionId": f.NewID(), "name": "录用周平", "primaryDocument": M{"kind": "passport", "number": "REC-50-P"}}, "", 200)["id"].(string)
	terms := M{"number": "录用通知650", "person": person, "employer": x.entity, "position": position, "arrival": "2027-05-01", "reserveFrom": "2027-04-01"}
	doc := M{"operation": "issue", "businessDate": "2027-04-01", "basis": "已签发录用通知", "terms": terms}
	path := "/api/v1/recruiting/offers"
	key := f.NewID()
	db := x.h.f
	usage := func(day string) M {
		return b.call("GET", "/api/v1/positions/"+position+"/replenishment?businessAt="+day, nil, "", 200)
	}
	// Hold publication after the cross-domain employer and assignment checks return.
	issueCheck := replaceAdminPurposeGrant(t, db, b)
	hold, err := db.admin.Begin(db.ctx)
	must(t, err)
	defer hold.Rollback(db.ctx)
	_, err = hold.Exec(db.ctx, "LOCK TABLE recruiting.outbox IN EXCLUSIVE MODE")
	must(t, err)
	expiry := time.Now().Add(1800 * time.Millisecond)
	grant := issueCheck(&expiry)
	result := make(chan int, 1)
	go func() {
		status, _, _ := b.response("POST", path, doc, key)
		result <- status
	}()
	waitForBlockedTransaction(t, db, hold.Conn().PgConn().PID())
	waitForStoredGrantExpiry(t, db, hold, grant)
	must(t, hold.Commit(db.ctx))
	if status := <-result; status != 403 || db.scalar("SELECT count(*) FROM recruiting.offer_submission") != 0 || usage("2027-05-01")["reserved"] != 0.0 {
		t.Fatalf("用途授权在录用发布前到期，返回 %d，要求拒绝且无回执、无名额占用", status)
	}
	issueCheck(nil)
	db.exec("ALTER TABLE recruiting.outbox ADD CONSTRAINT fail_offering CHECK(false) NOT VALID")
	b.call("POST", path, doc, key, 503)
	db.exec("ALTER TABLE recruiting.outbox DROP CONSTRAINT fail_offering")
	if usage("2027-05-01")["reserved"] != 0.0 || db.scalar("SELECT count(*) FROM recruiting.offer_submission") != 0 {
		t.Fatal("录用末步失败后留有名额或回执")
	}
	// With issue scope alone, emitting the notice cannot acquire a counted seat.
	x.h.policy("UPDATE access.offer_grant SET department_id=$2 WHERE principal_id=$1 AND action='reserve'", b.subject, f.NewID())
	b.call("POST", path, doc, key, 403)
	x.h.policy("UPDATE access.offer_grant SET department_id=NULL WHERE principal_id=$1 AND action='reserve'", b.subject)
	concurrencyKey := f.NewID()
	competitor := maps.Clone(doc)
	competingTerms := maps.Clone(terms)
	competingTerms["number"] = "同一末编另一份录用"
	competitor["terms"] = competingTerms
	replies := make(chan jointReply, 2)
	send := func(body M, submission string) {
		status, v, err := b.response("POST", path, body, submission)
		replies <- jointReply{status, v, err}
	}
	go send(doc, key)
	go send(competitor, concurrencyKey)
	one, two := <-replies, <-replies
	if one.status == 409 {
		one, two = two, one
	}
	if one.err != nil || two.err != nil || one.status != 200 || two.status != 409 || usage("2027-05-01")["reserved"] != 1.0 {
		t.Fatal(one, two)
	}
	saved := one.body
	if saved["submissionId"] == concurrencyKey {
		key = concurrencyKey
		doc = competitor
	}
	id := saved["id"].(string)
	detail := path + "/" + id
	original := b.call("GET", detail, nil, "", 200)
	if b.call("POST", path, doc, key, 200)["id"] != id || original["status"] != "issued" || original["personName"] != "录用周平" || original["employerAdoption"].([]any)[0].(M)["declaration"] == nil {
		t.Fatal(original)
	}
	segs := original["segments"].([]any)
	if len(segs) == 0 || segs[0].(M)["seats"] != 1.0 || segs[0].(M)["quotaSource"] == nil || segs[0].(M)["recruitingCoverage"] == nil {
		t.Fatal(segs)
	}
	other := maps.Clone(doc)
	otherTerms := maps.Clone(terms)
	otherTerms["number"] = "第二份"
	other["terms"] = otherTerms
	b.call("POST", path, other, f.NewID(), 409)
	releasedFirst := b.call("POST", detail+"/changes", M{"operation": "release", "businessDate": "2027-04-02", "basis": "先由招聘释放", "expected": saved["version"]}, f.NewID(), 200)
	b.call("POST", detail+"/changes", M{"operation": "accept", "businessDate": "2027-04-03", "basis": "份额缺失时尝试登记接受", "expected": releasedFirst["version"]}, f.NewID(), 409)
	reservedAgain := b.call("POST", detail+"/changes", M{"operation": "reserve", "businessDate": "2027-04-04", "basis": "恢复本份录用预留", "expected": releasedFirst["version"]}, f.NewID(), 200)
	b.call("POST", detail+"/changes", M{"operation": "accept", "businessDate": "2027-04-05", "basis": "本人已书面接受", "expected": reservedAgain["version"]}, f.NewID(), 200)
	accepted := b.call("GET", detail, nil, "", 200)
	if accepted["status"] != "accepted" || len(accepted["revisions"].([]any)) != 4 {
		t.Fatal(accepted)
	}
	released := b.call("POST", detail+"/changes", M{"operation": "release", "businessDate": "2027-04-15", "basis": "原份额释放", "expected": accepted["version"]}, f.NewID(), 200)
	read := b.call("GET", detail, nil, "", 200)
	if read["segments"].([]any)[1].(M)["until"] != "2027-04-15" {
		t.Fatal(read)
	}
	b.call("POST", detail+"/changes", M{"operation": "reserve", "businessDate": "2027-04-20", "basis": "重新取得份额", "expected": released["version"]}, f.NewID(), 200)
	if len(b.call("GET", detail, nil, "", 200)["segments"].([]any)) != 3 {
		t.Fatal("重新预留覆盖了原历史")
	}
	unused := f.NewID()
	resolved := b.call("POST", "/api/v1/recruiting/submissions/"+unused+"/resolve", M{}, "", 200)
	if resolved["outcome"] != "not_applied" {
		t.Fatal(resolved)
	}
	b.call("POST", path, other, unused, 409)
	x.h.policy("DELETE FROM access.offer_grant WHERE principal_id=$1 AND action='read'", b.subject)
	b.call("GET", detail, nil, "", 403)
	b.call("GET", path, nil, "", 403)
	b.call("POST", "/api/v1/recruiting/submissions/"+key+"/resolve", M{}, "", 403)
	b.call("POST", path, doc, key, 403)
}

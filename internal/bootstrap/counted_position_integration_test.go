//go:build integration

package bootstrap_test

import (
	"strings"
	"testing"

	f "chnhcm.local/chnhcm/internal/foundation"
)

func TestCountedPositionAdmissionAndPromotion(t *testing.T) {
	type M = map[string]any
	x := setupAssignment(t)
	b, db := x.h.b, x.h.f
	staff := staffing307{t, db, b}
	p5 := staff.create("grade", "G51P5", "")["id"].(string)
	p6 := staff.create("grade", "G51P6", "")["id"].(string)
	staff.correct("job", staff.detail("job", x.job, ""), M{"gradeIds": []string{p5, p6}}, nil, 200)
	day := "2027-04-01"
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
	pos := func(code, grade string) string {
		in := M{"code": code, "name": "核编" + code, "effectiveDate": day, "operationType": "CREATE", "operationReason": "NEW_POSITION", "departmentId": x.dep, "jobId": x.job, "locationId": x.locations[0]}
		if grade != "" {
			in["gradeId"] = grade
		}
		id := b.call("POST", "/api/v1/positions", in, f.NewID(), 201)["objectId"].(string)
		b.call("POST", "/api/v1/staffing-quotas", M{"positionId": id, "effectiveDate": day, "quota": 1, "reason": "核一编"}, f.NewID(), 200)
		return id
	}
	a, z := pos("P51A", p5), pos("P51B", p6)
	usage := func(id, date string) M {
		return b.call("GET", "/api/v1/positions/"+id+"/replenishment?businessAt="+date, nil, "", 200)
	}
	person := func(number string) string {
		return b.call("POST", "/api/v1/persons", M{"submissionId": f.NewID(), "name": "按岗" + number, "primaryDocument": M{"kind": "passport", "number": "G51-" + number}}, "", 200)["id"].(string)
	}
	hire := M{"person": person("ZHOU"), "employer": x.entity, "arrival": day, "position": a, "grade": p5, "notice": "按岗正式入职"}
	b.call("POST", "/api/v1/hires/preview", hire, "", 200)
	db.exec("ALTER TABLE staffing.employment_outbox ADD CONSTRAINT fail_counted_hire CHECK(false) NOT VALID")
	b.call("POST", "/api/v1/hires", hire, f.NewID(), 503)
	db.exec("ALTER TABLE staffing.employment_outbox DROP CONSTRAINT fail_counted_hire")
	if usage(a, day)["occupied"] != 0.0 {
		t.Fatal("失败留下岗位占用")
	}
	key := f.NewID()
	receipt := b.call("POST", "/api/v1/hires", hire, key, 200)
	if b.call("POST", "/api/v1/hires", hire, key, 200)["assignment"] != receipt["assignment"] || usage(a, day)["occupied"] != 1.0 {
		t.Fatal("末编未准确占用")
	}
	newcomer := person("QIAN")
	waiting := M{"employer": x.entity, "person": newcomer, "position": a, "grade": p5, "arrival": day, "notice": "第二份拟入"}
	short := b.call("POST", "/api/v1/hires", waiting, f.NewID(), 409)
	if short["code"] != "M03_QUOTA_SHORTAGE" || !strings.Contains(short["message"].(string), "缺 1 编") {
		t.Fatal(short)
	}
	base := "/api/v1/assignments/" + receipt["assignment"].(string)
	july := "2027-07-01"
	view := func(day string) M { return b.call("GET", base+"?businessAt="+day, nil, "", 200) }
	before := view(july)["sourceVersion"]
	promote := M{"expectedSourceVersion": before, "businessDate": july, "grade": p6, "notice": "七月晋升与调动共同通知"}
	b.call("POST", base+"/promotions", promote, f.NewID(), 409)
	transfer := M{"expectedSourceVersion": before, "businessDate": july, "position": z, "notice": "调往P6岗"}
	b.call("POST", base+"/transfers", transfer, f.NewID(), 409)
	promote["position"] = z
	result := b.call("POST", base+"/promotion-transfers", promote, f.NewID(), 200)
	got := view(july)
	st := got["state"].(M)
	if len(result["related"].([]any)) != 1 || st["position"] != z || st["grade"].(M)["id"] != p6 || st["seats"] != 1.0 || usage(a, july)["occupied"] != 0.0 || usage(z, july)["occupied"] != 1.0 || usage(a, "2027-06-30")["occupied"] != 1.0 {
		t.Fatal(result, st)
	}
	b.call("POST", "/api/v1/employment-actions/"+result["action"].(string)+"/cancel", M{"expectedSourceVersion": got["sourceVersion"], "revision": result["revision"], "reason": "误录晋升"}, f.NewID(), 409)
	if view(july)["sourceVersion"] != got["sourceVersion"] {
		t.Fatal("拒绝撤销改变任职")
	}
	// After the first holder leaves A, an offer and a different hire race for that dated seat.
	waiting["arrival"] = july
	offer := M{"operation": "issue", "businessDate": july, "basis": "七月预留", "terms": M{"number": "R51", "person": person("LIN"), "employer": x.entity, "position": a, "arrival": july, "reserveFrom": july}}
	replies := make(chan jointReply, 2)
	for path, cmd := range map[string]M{"/api/v1/hires": waiting, "/api/v1/recruiting/offers": offer} {
		go func() { status, v, e := b.response("POST", path, cmd, f.NewID()); replies <- jointReply{status, v, e} }()
	}
	one, two := <-replies, <-replies
	if one.status == 409 {
		one, two = two, one
	}
	count := usage(a, july)
	if one.err != nil || two.err != nil || one.status != 200 || two.status != 409 || count["reserved"].(float64)+count["occupied"].(float64) != 1 {
		t.Fatal(one, two, count)
	}
}

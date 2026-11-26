//go:build integration

package bootstrap_test

import (
	"testing"

	f "chnhcm.local/chnhcm/internal/foundation"
)

func TestExpiredPrimaryRequiresRehireFromLeave(t *testing.T) {
	type M = map[string]any
	x := setupAssignment(t)
	b := x.h.b
	person := b.call("POST", "/api/v1/persons", M{"submissionId": f.NewID(), "name": "固定期再聘", "primaryDocument": M{"kind": "passport", "number": f.NewID()}}, "", 200)["id"].(string)
	cmd := M{"person": person, "employer": x.entity, "department": x.dep, "job": x.job, "arrival": "2027-05-01", "lastDay": "2027-06-30", "notice": "两个月的初次任用"}
	first := b.call("POST", "/api/v1/hires", cmd, f.NewID(), 200)
	overview := b.call("GET", "/api/v1/persons/"+person+"/employment?businessAt=2027-09-01", nil, "", 200)
	if overview["assignments"] != 0. || overview["canHire"] != false || overview["rehireFrom"] != nil {
		t.Fatal(overview)
	}
	cmd["arrival"] = "2027-09-01"
	delete(cmd, "lastDay")
	cmd["notice"] = "返岗通知"
	before := x.h.f.scalar("SELECT count(*) FROM staffing.employment_outbox")
	for _, op := range []string{"/api/v1/hires/preview", "/api/v1/hires"} {
		if resp := b.call("POST", op, cmd, f.NewID(), 409); resp["code"] != "HIRE_EXISTING" {
			t.Fatal(resp)
		}
	}
	if x.h.f.scalar("SELECT count(*) FROM staffing.employment_outbox") != before {
		t.Fatal("拒绝再首聘却写入办理")
	}
	path := "/api/v1/assignments/" + first["assignment"].(string)
	last := b.call("GET", path+"?businessAt=2027-06-30", nil, "", 200)
	left := b.call("POST", path+"/leave", M{"date": "2027-06-30", "meaning": "last-active", "notice": "按定期届满办离职", "expectedSourceVersion": last["sourceVersion"]}, f.NewID(), 200)
	cmd["rehireOf"] = M{"action": left["action"], "revision": left["revision"]}
	back := b.call("POST", "/api/v1/hires", cmd, f.NewID(), 200)
	if back["recordNumber"] != 0. || back["assignment"] == first["assignment"] || back["employment"] == first["employment"] {
		t.Fatal(back)
	}
	if orig := b.call("GET", "/api/v1/employment-actions/"+back["action"].(string), nil, "", 200); orig["action"].(M)["kind"] != "Rehire" {
		t.Fatal(orig)
	}
}

func TestSuccessionChoosesRegularQuotaPerSegment(t *testing.T) {
	type M = map[string]any
	x := setupDatedAssignment(t, "2027-03-01", "职务基础")
	b := x.h.b
	const entry = "2027-03-01"
	const day = "2027-04-01"
	for _, purpose := range []string{"hire", "transfer", "recruit"} {
		pol := b.call("GET", "/api/v1/assignment-policies", nil, "", 200)
		row := M{"id": f.NewID(), "purpose": purpose, "category": "Employee", "mode": "PositionCounted", "from": entry}
		for _, a := range pol["rows"].([]any) {
			if v := a.(M); v["purpose"] == purpose {
				row["id"], row["revision"] = v["id"], v["revision"]
			}
		}
		b.call("POST", "/api/v1/assignment-policies", M{"expectedVersion": pol["version"], "row": row}, f.NewID(), 200)
	}
	position := func(code string) (string, M) {
		p := b.call("POST", "/api/v1/positions", M{"code": code, "name": code, "effectiveDate": entry, "operationType": "CREATE", "operationReason": "NEW_POSITION", "departmentId": x.dep, "jobId": x.job, "locationId": x.locations[0]}, f.NewID(), 201)["objectId"].(string)
		q := b.call("POST", "/api/v1/staffing-quotas", M{"positionId": p, "effectiveDate": entry, "quota": 2, "reason": "两份常规名额"}, f.NewID(), 200)
		return p, q
	}
	person := func(name string) any {
		return b.call("POST", "/api/v1/persons", M{"submissionId": f.NewID(), "name": name, "primaryDocument": M{"kind": "passport", "number": f.NewID()}}, "", 200)["id"]
	}
	hire := func(name, pos, last string) string {
		req := M{"person": person(name), "employer": x.entity, "position": pos, "arrival": entry, "notice": "任用" + name}
		if last != "" {
			req["lastDay"] = last
		}
		return b.call("POST", "/api/v1/hires", req, f.NewID(), 200)["assignment"].(string)
	}
	source, _ := position("SB509")
	target, q := position("SA509")
	for _, r := range []struct {
		day string
		n   int
	}{{"2027-04-05", 1}, {"2027-04-08", 2}} {
		q = b.call("POST", "/api/v1/staffing-quotas/"+q["objectId"].(string)+"/adjust", M{"effectiveDate": r.day, "quota": r.n, "reason": "期间内调整常规定额", "expectedSourceVersion": q["sourceVersion"]}, f.NewID(), 200)
	}
	incumbent := hire("原任", target, "2027-04-10")
	incoming := hire("接任", source, "")
	also := hire("另一接任", source, "")
	path := func(id string) string { return "/api/v1/assignments/" + id }
	cmd := func(id string) M {
		head := func(id string) any { return b.call("GET", path(id)+"?businessAt="+day, nil, "", 200)["sourceVersion"] }
		return M{"businessDate": day, "position": target, "notice": "内部接替", "expectedSourceVersion": head(id), "incumbent": M{"assignment": incumbent, "sourceVersion": head(incumbent)}}
	}
	b.call("POST", path(incoming)+"/transfers", cmd(incoming), f.NewID(), 200)
	for _, c := range []struct {
		day                string
		regular, dedicated float64
	}{{day, 2, 0}, {"2027-04-05", 1, 1}, {"2027-04-08", 2, 0}, {"2027-04-11", 1, 0}} {
		view := b.call("GET", "/api/v1/positions/"+target+"/replenishment?businessAt="+c.day, nil, "", 200)
		if view["occupied"] != c.regular || view["handoverOccupied"] != c.dedicated || view["totalOccupied"] != c.regular+c.dedicated {
			t.Fatal(c, view)
		}
	}
	third := M{"person": person("普通第三人"), "employer": x.entity, "position": target, "arrival": "2027-04-02", "lastDay": "2027-04-03", "notice": "独立任用"}
	if r := b.call("POST", "/api/v1/hires", third, f.NewID(), 409); r["code"] != "M03_QUOTA_SHORTAGE" {
		t.Fatal(r)
	}
	if r := b.call("POST", path(also)+"/transfers", cmd(also), f.NewID(), 409); r["code"] != "SUCCESSION_ASSIGNED" {
		t.Fatal(r)
	}
}

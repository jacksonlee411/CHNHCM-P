//go:build integration

package bootstrap_test

import (
	"chnhcm.local/chnhcm/internal/bootstrap"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"chnhcm.local/chnhcm/internal/foundation"
)

func TestCompensationGradeRateLifecycle(t *testing.T) {
	h := staffingSetup307(t)
	b := h.b
	const base = "/api/v1/compensation/grade-rates"
	type fields = map[string]any
	grade := b.call("POST", "/api/v1/grades", fields{"code": "P5", "name": "专业五级", "effectiveDate": "2027-01-01", "operationType": "CREATE", "operationReason": "NEW_GRADE"}, foundation.NewID(), 201)["objectId"]
	input := fields{"gradeId": grade, "currency": "CNY", "frequency": "MONTH", "unit": "PER_MONTH", "action": "CREATE", "reason": "建立标准", "effectiveDate": "1900-01-01", "changes": fields{"form": "RANGE", "minimum": "10000", "guidance": "12500", "maximum": "15000"}}
	b.call("POST", base, input, foundation.NewID(), 422)
	input["effectiveDate"] = "2027-01-01"
	sid := foundation.NewID()
	first := b.call("POST", base, input, sid, 200)
	if again := b.call("POST", base, input, sid, 200); !reflect.DeepEqual(again, first) {
		t.Fatal("retry changed the original receipt")
	}
	b.call("POST", base, input, foundation.NewID(), 409)
	path := base + "/" + first["objectId"].(string)
	write := func(prior fields, action, date string, changes fields, want int) fields {
		cmd := fields{"action": action, "recordId": prior["recordId"], "expectedRevision": prior["revision"], "expectedVersion": prior["sourceVersion"]}
		if date != "" {
			cmd["effectiveDate"] = date
			cmd["reason"] = "薪酬制度调整"
		}
		if changes != nil {
			cmd["changes"] = changes
		}
		return b.call("POST", path, cmd, foundation.NewID(), want)
	}
	july := write(first, "ADD", "2027-07-01", fields{"minimum": "11000", "guidance": "13500", "maximum": "16500"}, 200)
	write(first, "ADD", "2027-07-01", fields{"maximum": "20000"}, 409)
	for date, want := range map[string]string{"2027-06-30": "15000.00", "2027-07-01": "16500.00"} {
		rec := b.call("GET", path+"?businessAt="+date, nil, "", 200)["record"].(map[string]any)
		if rec["maximum"] != want {
			t.Fatalf("%s: %v", date, rec)
		}
	}
	write(july, "CORRECT", "", fields{"guidance": "17500"}, 422)
	corrected := write(july, "CORRECT", "", fields{"maximum": "16000", "guidance": nil}, 200)
	record := b.call("GET", path+"?businessAt=2027-07-01", nil, "", 200)["record"].(map[string]any)
	if record["maximum"] != "16000.00" || record["guidance"] != nil || corrected["recordId"] != july["recordId"] {
		t.Fatal(record)
	}
	history := b.call("GET", path+"/history?sourceVersion="+july["sourceVersion"].(string), nil, "", 200)["items"].([]any)
	if history[0].(fields)["after"].(fields)["maximum"] != "16500.00" {
		t.Fatal(history)
	}
	removed := write(corrected, "DELETE", "", nil, 200)
	if removed["recordId"] != corrected["recordId"] {
		t.Fatal(removed)
	}
	if got := b.call("GET", path+"?businessAt=2027-07-01", nil, "", 200)["record"].(fields)["maximum"]; got != "15000.00" {
		t.Fatal(got)
	}
	if got := b.call("GET", "/api/v1/compensation/grade-rate-receipts/"+sid, nil, "", 200); !reflect.DeepEqual(got, first) {
		t.Fatal(got)
	}
	list := b.call("GET", base+"?q=P5&currency=CNY", nil, "", 200)
	if list["total"] != float64(1) {
		t.Fatal(list)
	}
	reader := h.f.login("reader", h.f.readPassword)
	reader.call("GET", path, nil, "", 200)
	reader.call("POST", base, input, foundation.NewID(), 403)
	if n := h.f.scalar("SELECT count(*) FROM compensation.grade_rate_version"); n != 4 {
		t.Fatal(n)
	}
}

func TestCompensationEffectiveDateAndCurrentGrants(t *testing.T) {
	h := staffingSetup307(t)
	f, b := h.f, h.b
	type body = map[string]any
	const list = "/api/v1/compensation/grade-rates"
	const receipts = "/api/v1/compensation/grade-rate-receipts/"
	var instant atomic.Int64
	setDay := func(iso string) { v, err := time.Parse(time.RFC3339, iso); must(t, err); instant.Store(v.Unix()) }
	setDay("2027-06-30T23:59:59+08:00")
	defer bootstrap.SetCompensationIntegrationClock(f.runtime, func() time.Time { return time.Unix(instant.Load(), 0) })()
	grade := h.create("grade", "P5A", "")["id"]
	create := body{"action": "CREATE", "gradeId": grade, "currency": "CNY", "frequency": "YEAR", "unit": "PER_YEAR", "reason": "建立标准", "effectiveDate": "2027-01-01", "changes": body{"form": "FIXED", "amount": "180000"}}
	sub := foundation.NewID()
	first := b.call("POST", list, create, sub, 200)
	path := list + "/" + first["objectId"].(string)
	command := func(action string, base, changes body, date string) body {
		out := body{"action": action, "recordId": base["recordId"], "expectedVersion": base["sourceVersion"], "expectedRevision": base["revision"]}
		if date != "" {
			out["effectiveDate"], out["reason"] = date, "薪酬制度调整"
		}
		if changes != nil {
			out["changes"] = changes
		}
		return out
	}
	change := command("ADD", first, body{"amount": "180000.04"}, "2027-07-01")
	// Two clients may send an identical request after an uncertain response.
	key := foundation.NewID()
	type response struct {
		status  int
		receipt body
		err     error
	}
	result := make(chan response, 2)
	for range 2 {
		go func() { status, v, err := b.response("POST", path, change, key); result <- response{status, v, err} }()
	}
	take := func() body {
		r := <-result
		must(t, r.err)
		if r.status != 200 {
			t.Fatalf("write status %d: %v", r.status, r.receipt)
		}
		return r.receipt
	}
	july, same := take(), take()
	if !reflect.DeepEqual(july, same) {
		t.Fatalf("duplicate request returned different receipts")
	}
	for _, day := range []struct{ at, amount string }{{"2027-06-30T23:59:59+08:00", "180000.00"}, {"2027-07-01T00:00:00+08:00", "180000.04"}} {
		setDay(day.at)
		got := b.call("GET", list+"?q=P5A", nil, "", 200)["items"].([]any)
		if len(got) != 1 || got[0].(body)["record"].(body)["amount"] != day.amount {
			t.Fatal(day, got)
		}
	}
	f.exec("UPDATE access.subject_grant SET write_fields=array_remove(write_fields,'amount') WHERE principal_id=$1 AND action='graderate.correct'", b.subject)
	// Two independently incomplete grants cannot jointly authorize replacing
	// the form and clearing the old amount.
	f.exec(`INSERT INTO access.subject_grant(instance_id,grant_id,principal_id,scope_kind,scope_id,action,read_fields,write_fields)
        SELECT instance_id,gen_random_uuid(),principal_id,scope_kind,scope_id,action,read_fields,ARRAY['amount']
        FROM access.subject_grant WHERE principal_id=$1 AND action='graderate.correct'`, b.subject)
	b.call("POST", path, command("CORRECT", july, body{"form": "RANGE", "minimum": "11000", "maximum": "16000"}, ""), foundation.NewID(), 403)
	dormant := b.call("POST", path, command("DEACTIVATE", july, nil, "2027-09-01"), foundation.NewID(), 200)
	b.call("POST", path, command("REACTIVATE", dormant, nil, "2027-10-01"), foundation.NewID(), 200)
	inactive := b.call("GET", path+"?businessAt=2027-09-30", nil, "", 200)["record"].(body)
	active := b.call("GET", path+"?businessAt=2027-10-01", nil, "", 200)["record"].(body)
	if inactive["status"] != "INACTIVE" || active["status"] != "ACTIVE" || active["amount"] != "180000.04" {
		t.Fatal(inactive, active)
	}
	reader := f.login("reader", f.readPassword)
	f.exec("DELETE FROM access.subject_grant WHERE principal_id=$1 AND action LIKE 'graderate.%'", reader.subject)
	if allowed := reader.call("GET", "/api/v1/compensation/access", nil, "", 200)["gradeRates"]; allowed != false {
		t.Fatal(allowed)
	}
	for _, url := range []string{list, path, path + "/history", receipts + sub} {
		reader.call("GET", url, nil, "", 404)
	}
	reader.call("POST", receipts+sub+"/resolve", nil, "", 404)
	reader.call("GET", "/api/v1/grades/"+grade.(string), nil, "", 200)
	closed := foundation.NewID()
	if out := b.call("POST", receipts+closed+"/resolve", nil, "", 200); out["outcome"] != "not_applied" {
		t.Fatal(out)
	}
	b.call("POST", list, create, closed, 409)
	if n := f.scalar("SELECT count(*) FROM compensation.grade_rate_version"); n != 4 {
		t.Fatalf("versions=%d", n)
	}
}

func TestCompensationGradeCoverageAndOrigins(t *testing.T) {
	h := staffingSetup307(t)
	f, b := h.f, h.b
	type m = map[string]any
	const base = "/api/v1/compensation/grade-rates"
	grade := h.create("grade", "T210", "")
	cmd := m{"action": "CREATE", "gradeId": grade["id"], "currency": "CNY", "frequency": "MONTH", "unit": "PER_MONTH", "reason": "建立标准", "effectiveDate": "2027-01-01", "changes": m{"form": "RANGE", "minimum": "0", "guidance": "0", "maximum": "15000"}}
	first := b.call("POST", base, cmd, foundation.NewID(), 200)
	path := base + "/" + first["objectId"].(string)
	change := m{"action": "CORRECT", "recordId": first["recordId"], "expectedVersion": first["sourceVersion"], "expectedRevision": first["revision"], "changes": m{"guidance": "0.00"}}
	if v := b.call("POST", path, change, foundation.NewID(), 200); v["outcome"] != "no_change" {
		t.Fatal(v)
	}
	before := b.call("GET", path, nil, "", 200)["record"].(m)
	if before["guidance"] != "0.00" {
		t.Fatal(before)
	}
	h.correct("grade", grade, m{"name": "专业级别更名"}, nil, 200)
	grade = h.detail("grade", grade["id"].(string), "")
	change["changes"] = m{"guidance": nil}
	saved := b.call("POST", path, change, foundation.NewID(), 200)
	after := b.call("GET", path, nil, "", 200)["record"].(m)
	if before["grade"].(m)["name"] == after["grade"].(m)["name"] || after["grade"].(m)["name"] != "专业级别更名" || after["guidance"] != nil {
		t.Fatal(after)
	}
	source := b.call("GET", path+"/history?sourceVersion="+first["sourceVersion"].(string), nil, "", 200)["items"].([]any)[0].(m)["after"].(m)
	if !reflect.DeepEqual(before["adopted"], source["adopted"]) || source["guidance"] != "0.00" {
		t.Fatal(source)
	}
	// March is inactive in M03 even though the endpoints of the proposed change are active.
	h.add("grade", grade, "DEACTIVATE", "2027-03-01", nil, 200)
	grade = h.detail("grade", grade["id"].(string), "")
	nodes := grade["timeline"].([]any)
	last := nodes[len(nodes)-1].(m)["recordId"].(string)
	stopped := h.detail("grade", grade["id"].(string), last)
	h.add("grade", stopped, "REACTIVATE", "2027-04-01", nil, 200)
	change["expectedVersion"], change["expectedRevision"], change["changes"] = saved["sourceVersion"], saved["revision"], m{"maximum": "16000"}
	refusal := b.call("POST", path, change, foundation.NewID(), 422)
	if !strings.Contains(refusal["message"].(string), "2027-03-01") {
		t.Fatal(refusal)
	}
	original := b.call("GET", path+"/history?sourceVersion="+first["sourceVersion"].(string), nil, "", 200)["items"].([]any)[0].(m)
	if original["after"].(m)["maximum"] != "15000.00" {
		t.Fatal(original)
	}
	f.exec("UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'maximum') WHERE principal_id=$1 AND action='graderate.read'", b.subject)
	limited := b.call("GET", path+"/history?sourceVersion="+first["sourceVersion"].(string), nil, "", 200)["items"].([]any)[0].(m)
	if limited["after"].(m)["maximum"] != nil || limited["after"].(m)["minimum"] != "0.00" {
		t.Fatal(limited)
	}
	b.call("POST", path, change, foundation.NewID(), 403)
}

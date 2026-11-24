//go:build integration

package integration_test

import (
	"fmt"
	"testing"
	"time"

	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/foundation"
)

func TestBU129UnchangedCoverageAvoidsDepartmentHeadsAndFacts(t *testing.T) {
	s := newDepartmentContract(t)
	child := s.create("CHILD", "未来部门", "BU_A", "", "2035-01-01", s.b, 201)
	id := s.aliases["BU_A"]
	detail := s.b.call("GET", "/api/v1/business-units/"+id, nil, "", 200)
	r := detail["timeline"].([]any)[0].(map[string]any)
	body := map[string]any{"expectedSourceVersion": detail["headSourceVersion"], "expectedRecordRevision": r["revision"], "changes": map[string]any{"name": "新名称"}}
	// An unrelated department head is exclusively locked. A name change must
	// not acquire it; the BU and administrative range guards remain in use.
	tx, err := s.f.admin.Begin(s.f.ctx)
	must(t, err)
	defer tx.Rollback(s.f.ctx)
	_, err = tx.Exec(s.f.ctx, `SELECT department_id FROM organization.department WHERE instance_id=$1 AND department_id=$2 FOR UPDATE`, s.f.instance, s.aliases["CHILD"])
	must(t, err)
	s.b.call("POST", "/api/v1/business-units/"+id+"/records/"+r["recordId"].(string)+"/correct", body, foundation.NewID(), 200)
	must(t, tx.Rollback(s.f.ctx))
	// Future coverage still matters even when the child is not effective today.
	s.stopBU("BU_A", "2036-01-01", 409)
	detail = s.b.call("GET", "/api/v1/business-units/"+id, nil, "", 200)
	r = detail["timeline"].([]any)[0].(map[string]any)
	s.f.editDepartment(child["sourceVersion"].(string), func(v *bootstrap.DepartmentHistoryFact) { v.State.Summaries = nil })
	body["expectedSourceVersion"], body["expectedRecordRevision"] = detail["headSourceVersion"], r["revision"]
	body["action"], body["recordId"] = "CORRECT", r["recordId"]
	body["changes"] = map[string]any{"name": "再次更名"}
	s.b.call("POST", "/api/v1/business-units/"+id+"/preview", body, "", 200)
	delete(body, "action")
	delete(body, "recordId")
	s.b.call("POST", "/api/v1/business-units/"+id+"/records/"+r["recordId"].(string)+"/correct", body, foundation.NewID(), 200)
	s.stopBU("BU_A", "2036-01-01", 503) // Changed coverage still consumes required facts.
}

func TestBU129QueuedSameSourceRechecksAfterLock(t *testing.T) {
	s := newDepartmentContract(t)
	id := s.aliases["BU_A"]
	detail := s.b.call("GET", "/api/v1/business-units/"+id, nil, "", 200)
	r := detail["timeline"].([]any)[0].(map[string]any)
	tx, err := s.f.admin.Begin(s.f.ctx)
	must(t, err)
	defer tx.Rollback(s.f.ctx)
	_, err = tx.Exec(s.f.ctx, `SELECT FROM organization.business_unit WHERE bu_id=$1 FOR UPDATE`, id)
	must(t, err)
	type reply struct {
		status int
		body   map[string]any
		err    error
	}
	replies := make(chan reply, 2)
	for n := 0; n < 2; n++ {
		go func(n int) {
			body := map[string]any{"expectedSourceVersion": detail["headSourceVersion"], "expectedRecordRevision": r["revision"], "changes": map[string]any{"name": fmt.Sprintf("并发更名%d", n)}}
			status, out, e := s.b.response("POST", "/api/v1/business-units/"+id+"/records/"+r["recordId"].(string)+"/correct", body, foundation.NewID())
			replies <- reply{status, out, e}
		}(n)
	}
	waiting := 0
	until := time.Now().Add(time.Second)
	for waiting < 2 && time.Now().Before(until) {
		must(t, s.f.admin.QueryRow(s.f.ctx, `SELECT count(*) FROM pg_stat_activity WHERE usename='chnhcm_api' AND wait_event_type='Lock'`).Scan(&waiting))
		if waiting < 2 {
			time.Sleep(5 * time.Millisecond)
		}
	}
	must(t, tx.Rollback(s.f.ctx))
	if waiting < 2 {
		t.Fatal("both requests must have reached the real lock boundary", waiting)
	}
	accepted, conflicts := 0, 0
	for n := 0; n < 2; n++ {
		v := <-replies
		must(t, v.err)
		if v.status == 200 && v.body["outcome"] == "accepted" {
			accepted++
		} else if v.status == 409 && v.body["code"] == "M02_SOURCE_CONFLICT" {
			conflicts++
		} else {
			t.Fatal(v.status, v.body)
		}
	}
	if accepted != 1 || conflicts != 1 {
		t.Fatal(accepted, conflicts)
	}
}

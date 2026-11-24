//go:build integration

package bootstrap_test

import (
	enterpriseapp "chnhcm.local/chnhcm/internal/contexts/organization/application/enterprise"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	app "chnhcm.local/chnhcm/internal/contexts/staffing/application"
	store "chnhcm.local/chnhcm/internal/contexts/staffing/infra/postgres"
	staff "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"fmt"
	"testing"
	"time"
)

func TestStaffing309ObjectLockBudgetAndResolve(t *testing.T) {
	h := staffingSetup307(t)
	f := h.f
	svc := f.runtime.Staffing.(app.Service)
	c := geoCredentials203(t, f, "admin", f.password)
	v := h.create("grade", "GUARD309", "")
	command := func(name string) staff.Command {
		r := rec307(v)
		return staff.Command{Kind: staff.Grade, ObjectID: v["id"].(string), SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Input: staff.Input{Action: "CORRECT", RecordID: r["recordId"].(string), ExpectedSourceVersion: v["headSourceVersion"].(string), ExpectedRecordRevision: r["revision"].(string), Changes: map[string]*string{"name": name307(name)}}}
	}
	cmd := command("预算测试")
	submission, e := f.admin.Begin(f.ctx)
	must(t, e)
	defer submission.Rollback(f.ctx)
	guard, e := f.admin.Begin(f.ctx)
	must(t, e)
	defer guard.Rollback(f.ctx)
	_, e = submission.Exec(f.ctx, "SELECT pg_advisory_xact_lock($1)", key307("staffing.submission", "grade", f.instance, h.b.subject, cmd.SubmissionID))
	must(t, e)
	_, e = guard.Exec(f.ctx, "SELECT pg_advisory_xact_lock($1)", key307("staffing.grade", f.instance, cmd.ObjectID))
	must(t, e)
	done := make(chan error, 1)
	start := time.Now()
	go func() { _, e := svc.Execute(f.ctx, c, cmd); done <- e }()
	blocked307(t, f, make(chan struct{}))
	time.Sleep(1100 * time.Millisecond)
	must(t, submission.Commit(f.ctx))
	e = <-done
	elapsed := time.Since(start)
	if e == nil || elapsed > 2900*time.Millisecond || elapsed < 1700*time.Millisecond {
		t.Fatalf("Grade cumulative lock budget: %s %v", elapsed, e)
	}
	must(t, guard.Commit(f.ctx))
	r, e := svc.Resolve(f.ctx, c, staff.Grade, cmd.SubmissionID)
	must(t, e)
	if r.Outcome != "not_applied" {
		t.Fatal("timed-out Grade wrote a fact")
	}
	gate := &gate307{Store: svc.Store, locked: make(chan struct{}), release: make(chan struct{})}
	ctx, cancel := context.WithCancel(f.ctx)
	cmd = command("取消事务")
	go func() { _, e := (app.Service{Store: gate}).Execute(ctx, c, cmd); done <- e }()
	wait307(t, f, gate.locked)
	cancel()
	if <-done == nil {
		t.Fatal("cancelled Grade write committed")
	}
	r, e = svc.Resolve(f.ctx, c, staff.Grade, cmd.SubmissionID)
	must(t, e)
	if r.Outcome != "not_applied" {
		t.Fatal("cancelled Grade left a fact")
	}
	gate = &gate307{Store: svc.Store, locked: make(chan struct{}), release: make(chan struct{})}
	cmd = command("迟到确认")
	written, resolved := make(chan staff.Result, 1), make(chan staff.Result, 1)
	go func() { r, e := (app.Service{Store: gate}).Execute(f.ctx, c, cmd); written <- r; done <- e }()
	wait307(t, f, gate.locked)
	go func() { r, e := svc.Resolve(f.ctx, c, staff.Grade, cmd.SubmissionID); resolved <- r; done <- e }()
	blocked307(t, f, gate.release)
	close(gate.release)
	must(t, <-done)
	must(t, <-done)
	a, b := <-written, <-resolved
	if a.Outcome != "accepted" || b.Outcome != "accepted" || a.SourceVersion != b.SourceVersion {
		t.Fatal("Grade resolve raced past original submit")
	}
	if f.scalar("SELECT count(*) FROM staffing.grade_source") != 2 {
		t.Fatal("Grade lock/recovery duplicated a source")
	}
}

func TestStaffing309EnterpriseFloorAndPhysicalTransaction(t *testing.T) {
	h := staffingSetup307(t)
	f, b := h.f, h.b
	svc := f.runtime.Staffing.(app.Service)
	db := svc.Store.(*store.Store)
	probe := &floorProbe307{EnterpriseFloor: db.Organization}
	db.Organization = probe
	f.exec(`CREATE TABLE public.grade309_tx_probe (pid bigint,txid text);
 GRANT INSERT ON public.grade309_tx_probe TO chnhcm_app;
 CREATE FUNCTION public.grade309_probe() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN INSERT INTO public.grade309_tx_probe VALUES(pg_backend_pid(),pg_current_xact_id()::text); RETURN NEW; END $$;
 CREATE TRIGGER grade309_probe AFTER INSERT ON staffing.grade_source FOR EACH ROW EXECUTE FUNCTION public.grade309_probe();`)
	v := h.create("grade", "TX-FLOOR", "")
	var xid string
	var pid uint32
	must(t, f.admin.QueryRow(f.ctx, `SELECT pid,txid FROM public.grade309_tx_probe`).Scan(&pid, &xid))
	if xid != probe.txid || pid != probe.pid || probe.pid == 0 {
		t.Fatal("enterprise port did not run in source transaction")
	}
	ent := b.call("GET", "/api/v1/enterprise/context", nil, "", 200)
	eid := ent["id"].(string)
	detail := b.call("GET", "/api/v1/enterprise/"+eid, nil, "", 200)
	r := rec307(detail)
	b.call("POST", "/api/v1/enterprise/"+eid+"/records/"+r["recordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": detail["headSourceVersion"], "expectedRecordRevision": r["revision"], "newEffectiveDate": "2001-01-01"}, foundation.NewID(), 409)
	h.add("grade", v, "DEACTIVATE", "2005-01-01", nil, 200)
	b.call("POST", "/api/v1/enterprise/"+eid+"/records/"+r["recordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": detail["headSourceVersion"], "expectedRecordRevision": r["revision"], "newEffectiveDate": "2001-01-01"}, foundation.NewID(), 409)
	// Enterprise stop is not catalog active coverage; it does not prohibit later catalogs.
	b.call("POST", "/api/v1/enterprise/"+eid+"/deactivate", map[string]any{"expectedSourceVersion": detail["headSourceVersion"], "anchorRecordId": r["recordId"], "effectiveDate": "1990-01-01", "operationType": "DEACTIVATE", "operationReason": "BUSINESS_CLOSURE"}, foundation.NewID(), 201)
	h.create("grade", "AFTER-ENTERPRISE-STOP", "")
}

func TestStaffing309EnterpriseBothLockOrders(t *testing.T) {
	for _, first := range []string{"child", "enterprise"} {
		t.Run(first, func(t *testing.T) {
			h := staffingSetup307(t)
			f, b := h.f, h.b
			ss := f.runtime.Staffing.(app.Service)
			es := f.runtime.Enterprise.(enterpriseapp.Service)
			c := geoCredentials203(t, f, "admin", f.password)
			context := b.call("GET", "/api/v1/enterprise/context", nil, "", 200)
			eid := context["id"].(string)
			ent := b.call("GET", "/api/v1/enterprise/"+eid, nil, "", 200)
			r := rec307(ent)
			parent := org.EnterpriseCommand{Action: "CORRECT", EnterpriseID: eid, RecordID: r["recordId"].(string), SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Correct: org.EnterpriseCorrectInput{ExpectedSourceVersion: ent["headSourceVersion"].(string), ExpectedRecordRevision: r["revision"].(string), NewEffectiveDate: "2001-01-01"}}
			child := staff.Command{Kind: staff.Grade, SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Input: staff.Input{Action: "CREATE", Code: "FLOOR-RACE", Name: name307("企业下界并发职级"), EffectiveDate: "2000-01-01", OperationType: "CREATE", OperationReason: "NEW_GRADE"}}
			release, locked := make(chan struct{}), make(chan struct{})
			childResult := make(chan staff.Result, 1)
			childErr, parentErr := make(chan error, 1), make(chan error, 1)
			runChild := func(svc app.Service) { v, e := svc.Execute(f.ctx, c, child); childResult <- v; childErr <- e }
			runParent := func(svc enterpriseapp.Service) { _, e := svc.Maintain(f.ctx, c, parent); parentErr <- e }
			if first == "child" {
				go runChild(app.Service{Store: &gate307{Store: ss.Store, locked: locked, release: release}})
				wait307(t, f, locked)
				go runParent(es)
			} else {
				go runParent(enterpriseapp.Service{Store: &enterpriseGate307{Store: es.Store, locked: locked, release: release}})
				wait307(t, f, locked)
				go runChild(ss)
			}
			blocked307(t, f, release)
			close(release)
			ce, pe, v := <-childErr, <-parentErr, <-childResult
			must(t, ce)
			if first == "child" {
				if v.Outcome != "accepted" || pe == nil {
					t.Fatal("child-first did not protect earliest date")
				}
			} else {
				must(t, pe)
				if v.Outcome != "rejected" || v.Failure.Code != "M03_ENTERPRISE_DATE_CONFLICT" {
					t.Fatal("parent-first did not refresh floor")
				}
			}
		})
	}
}

func TestStaffing309RollbackAndOriginalResolution(t *testing.T) {
	h := staffingSetup307(t)
	f := h.f

	svc := f.runtime.Staffing.(app.Service)
	c := geoCredentials203(t, f, "admin", f.password)
	for _, after := range []bool{false, true} {
		cmd := staff.Command{Kind: staff.Grade, SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Input: staff.Input{Action: "CREATE", Code: fmt.Sprintf("ROLLBACK-%v", after), Name: name307("职级原子保存"), EffectiveDate: "2000-01-01", OperationType: "CREATE", OperationReason: "NEW_GRADE"}}
		_, e := (app.Service{Store: fault307{svc.Store, after}}).Execute(f.ctx, c, cmd)
		if e == nil {
			t.Fatal("fault not injected")
		}
		if f.scalar(`SELECT count(*) FROM staffing.grade WHERE code=$1`, cmd.Input.Code) != 0 || f.scalar(`SELECT count(*) FROM staffing.grade_terminal WHERE submission_id=$1`, cmd.SubmissionID) != 0 {
			t.Fatal("partial write/terminal committed")
		}
		resolved, e := svc.Resolve(f.ctx, c, staff.Grade, cmd.SubmissionID)
		must(t, e)
		if resolved.Outcome != "not_applied" {
			t.Fatal("rollback resolution wrong")
		}
		r, e := svc.Execute(f.ctx, c, cmd)
		must(t, e)
		if r.Outcome != "not_applied" {
			t.Fatal("closed original identity reopened")
		}
	}
}

// The real transaction commits, but its acknowledgement is lost at the store
// boundary. Resolution must recover facts, rather than infer rollback from error.
type lostCommit309 struct{ app.Store }

func (s lostCommit309) Within(ctx context.Context, c foundation.Credentials, write bool, fn func(app.Unit) error) error {
	if err := s.Store.Within(ctx, c, write, fn); err != nil {
		return err
	}
	return foundation.Fail(503, "M03_DEPENDENCY_UNAVAILABLE", "commit acknowledgement lost")
}

func TestStaffing309UnknownCommitRecoversOriginal(t *testing.T) {
	h := staffingSetup307(t)
	f := h.f
	svc := f.runtime.Staffing.(app.Service)
	c := geoCredentials203(t, f, "admin", f.password)
	cmd := staff.Command{Kind: staff.Grade, SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Input: staff.Input{Action: "CREATE", Code: "COMMIT309", Name: name307("未知提交"), EffectiveDate: "2000-01-01", OperationType: "CREATE", OperationReason: "NEW_GRADE"}}
	if _, err := (app.Service{Store: lostCommit309{svc.Store}}).Execute(f.ctx, c, cmd); err == nil {
		t.Fatal("commit acknowledgement was not lost")
	}
	recovered, err := svc.Resolve(f.ctx, c, staff.Grade, cmd.SubmissionID)
	must(t, err)
	if recovered.Outcome != "accepted" || recovered.SourceVersion == "" {
		t.Fatal("unknown commit discarded its accepted result")
	}
	replayed, err := svc.Execute(f.ctx, c, cmd)
	must(t, err)
	if replayed.SourceVersion != recovered.SourceVersion || f.scalar("SELECT count(*) FROM staffing.grade_source") != 1 || f.scalar("SELECT count(*) FROM staffing.grade_outbox") != 1 {
		t.Fatal("unknown commit replay created a second source")
	}
}

//go:build integration

package bootstrap_test

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	enterpriseapp "chnhcm.local/chnhcm/internal/contexts/organization/application/enterprise"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	app "chnhcm.local/chnhcm/internal/contexts/staffing/application"
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	store "chnhcm.local/chnhcm/internal/contexts/staffing/infra/postgres"
	staff "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

type gate307 struct {
	app.Store
	locked, release chan struct{}
	once            sync.Once
}
type gateUnit307 struct {
	app.Unit
	gate *gate307
}

func (g *gate307) Within(ctx context.Context, c foundation.Credentials, write bool, fn func(app.Unit) error) error {
	return g.Store.Within(ctx, c, write, func(u app.Unit) error {
		if write {
			return fn(gateUnit307{u, g})
		}
		return fn(u)
	})
}
func (u gateUnit307) LockObject(ctx context.Context, k d.Kind, id string, refs, functions, grades []string) (d.Timeline, error) {
	v, e := u.Unit.LockObject(ctx, k, id, refs, functions, grades)
	if e != nil {
		return v, e
	}
	u.gate.once.Do(func() { close(u.gate.locked) })
	select {
	case <-u.gate.release:
	case <-ctx.Done():
		return v, ctx.Err()
	}
	return v, nil
}
func wait307(t *testing.T, f *fixture, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(3 * time.Second):
		t.Fatal("writer did not reach actual guard")
	}
}
func blocked307(t *testing.T, f *fixture, release chan struct{}) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for f.scalar("SELECT count(*) FROM pg_locks WHERE NOT granted AND locktype IN ('advisory','transactionid')") == 0 {
		if time.Now().After(deadline) {
			close(release)
			t.Fatal("writer did not wait on physical lock")
		}
		time.Sleep(5 * time.Millisecond)
	}
}
func name307(s string) *string { return &s }
func (h *staffing307) command307(family, code string) staff.Command {
	return staff.Command{Kind: staff.Job, SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Input: staff.Input{Action: "CREATE", Code: code, Name: name307("并发职务"), JobFunctionID: name307(h.b.functionForJobs("1900-01-01")), JobFamilyID: &family, EffectiveDate: "2000-01-01", OperationType: "CREATE", OperationReason: "NEW_JOB"}}
}
func TestStaffing307FamilyWriteRaces(t *testing.T) {
	h := staffingSetup307(t)
	f, b := h.f, h.b
	svc := f.runtime.Staffing.(app.Service)
	c := geoCredentials203(t, f, "admin", f.password)
	for _, first := range []string{"job", "family"} {
		t.Run(first+"_first", func(t *testing.T) {
			family := h.create("jobfamily", "RACE-"+first, "")
			fid := family["id"].(string)
			gate := &gate307{Store: svc.Store, locked: make(chan struct{}), release: make(chan struct{})}
			cmd := h.command307(fid, "RACE-"+first)
			stop := staff.Command{Kind: staff.Family, ObjectID: fid, SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Input: staff.Input{Action: "DEACTIVATE", ExpectedSourceVersion: family["headSourceVersion"].(string), AnchorRecordID: rec307(family)["recordId"].(string), EffectiveDate: "2030-01-01", OperationType: "DEACTIVATE", OperationReason: "FAMILY_CLOSURE"}}
			firstCmd, secondCmd := cmd, stop
			if first == "family" {
				firstCmd, secondCmd = stop, cmd
			}
			type result struct {
				v staff.Result
				e error
			}
			one, two := make(chan result, 1), make(chan result, 1)
			go func() { v, e := (app.Service{Store: gate}).Execute(f.ctx, c, firstCmd); one <- result{v, e} }()
			wait307(t, f, gate.locked)
			go func() { v, e := svc.Execute(f.ctx, c, secondCmd); two <- result{v, e} }()
			blocked307(t, f, gate.release)
			close(gate.release)
			a, z := <-one, <-two
			must(t, a.e)
			must(t, z.e)
			if a.v.Outcome != "accepted" || z.v.Outcome != "rejected" || z.v.Failure.Code != "M03_REFERENCE_CONFLICT" {
				t.Fatalf("race outcomes %s / %s", a.v.Outcome, z.v.Outcome)
			}
		})
	}
	// Reversed family swaps share sorted family guards and lock different Jobs.
	a, bb := h.create("jobfamily", "SWAP-A", ""), h.create("jobfamily", "SWAP-B", "")
	j1, j2 := h.create("job", "SWAP-J1", a["id"].(string)), h.create("job", "SWAP-J2", bb["id"].(string))
	done := make(chan error, 2)
	for i, v := range []map[string]any{j1, j2} {
		target := bb["id"]
		if i == 1 {
			target = a["id"]
		}
		go func(v map[string]any, target any) {
			r := rec307(v)
			status, out, e := b.response("POST", "/api/v1/jobs/"+v["id"].(string)+"/records/"+r["recordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": v["headSourceVersion"], "expectedRecordRevision": r["revision"], "changes": map[string]any{"jobFamilyId": target}}, foundation.NewID())
			if e == nil && (status != 200 || out["outcome"] != "accepted") {
				e = fmt.Errorf("swap status %d", status)
			}
			done <- e
		}(v, target)
	}
	must(t, <-done)
	must(t, <-done)
}

type floorProbe307 struct {
	org.EnterpriseFloor
	txid string
	pid  uint32
}

func (p *floorProbe307) StaffingEnterprise(ctx context.Context, lock bool) (org.Enterprise, error) {
	tx, ok := dbtx.From(ctx)
	if !ok {
		return org.Enterprise{}, fmt.Errorf("no shared transaction")
	}
	if lock {
		p.pid = tx.Conn().PgConn().PID()
		if e := tx.QueryRow(ctx, "SELECT pg_current_xact_id()::text").Scan(&p.txid); e != nil {
			return org.Enterprise{}, e
		}
	}
	return p.EnterpriseFloor.StaffingEnterprise(ctx, lock)
}
func TestStaffing307EnterpriseFloorAndPhysicalTransaction(t *testing.T) {
	h := staffingSetup307(t)
	f, b := h.f, h.b
	svc := f.runtime.Staffing.(app.Service)
	db := svc.Store.(*store.Store)
	probe := &floorProbe307{EnterpriseFloor: db.Organization}
	db.Organization = probe
	f.exec(`CREATE TABLE public.staff307_tx_probe (pid bigint,txid text);
 GRANT INSERT ON public.staff307_tx_probe TO chnhcm_app;
 CREATE FUNCTION public.staff307_probe() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN INSERT INTO public.staff307_tx_probe VALUES(pg_backend_pid(),pg_current_xact_id()::text); RETURN NEW; END $$;
 CREATE TRIGGER staff307_probe AFTER INSERT ON staffing.job_family_source FOR EACH ROW EXECUTE FUNCTION public.staff307_probe();`)
	v := h.create("jobfamily", "TX-FLOOR", "")
	var xid string
	var pid uint32
	must(t, f.admin.QueryRow(f.ctx, `SELECT pid,txid FROM public.staff307_tx_probe`).Scan(&pid, &xid))
	if xid != probe.txid || pid != probe.pid || probe.pid == 0 {
		t.Fatal("enterprise port did not run in source transaction")
	}
	ent := b.call("GET", "/api/v1/enterprise/context", nil, "", 200)
	eid := ent["id"].(string)
	detail := b.call("GET", "/api/v1/enterprise/"+eid, nil, "", 200)
	r := rec307(detail)
	b.call("POST", "/api/v1/enterprise/"+eid+"/records/"+r["recordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": detail["headSourceVersion"], "expectedRecordRevision": r["revision"], "newEffectiveDate": "2001-01-01"}, foundation.NewID(), 409)
	h.add("jobfamily", v, "DEACTIVATE", "2005-01-01", nil, 200)
	b.call("POST", "/api/v1/enterprise/"+eid+"/records/"+r["recordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": detail["headSourceVersion"], "expectedRecordRevision": r["revision"], "newEffectiveDate": "2001-01-01"}, foundation.NewID(), 409)
	// Enterprise stop is not catalog active coverage; it does not prohibit later catalogs.
	b.call("POST", "/api/v1/enterprise/"+eid+"/deactivate", map[string]any{"expectedSourceVersion": detail["headSourceVersion"], "anchorRecordId": r["recordId"], "effectiveDate": "1990-01-01", "operationType": "DEACTIVATE", "operationReason": "BUSINESS_CLOSURE"}, foundation.NewID(), 201)
	h.create("jobfamily", "AFTER-ENTERPRISE-STOP", "")
}

type fault307 struct {
	app.Store
	after bool
}
type faultUnit307 struct {
	app.Unit
	after bool
}

func (f fault307) Within(ctx context.Context, c foundation.Credentials, write bool, fn func(app.Unit) error) error {
	return f.Store.Within(ctx, c, write, func(u app.Unit) error {
		if write {
			return fn(faultUnit307{u, f.after})
		}
		return fn(u)
	})
}
func (u faultUnit307) Save(ctx context.Context, p d.Plan, c staff.Command, op access.OperatorSnapshot) error {
	if u.after {
		if e := u.Unit.Save(ctx, p, c, op); e != nil {
			return e
		}
	}
	return foundation.Fail(503, "M03_DEPENDENCY_UNAVAILABLE", "task-owned failure")
}
func TestStaffing307RollbackAndOriginalResolution(t *testing.T) {
	h := staffingSetup307(t)
	f := h.f
	v := h.create("jobfamily", "ROLLBACK-FAMILY", "")
	svc := f.runtime.Staffing.(app.Service)
	c := geoCredentials203(t, f, "admin", f.password)
	for _, after := range []bool{false, true} {
		cmd := h.command307(v["id"].(string), fmt.Sprintf("ROLLBACK-%v", after))
		_, e := (app.Service{Store: fault307{svc.Store, after}}).Execute(f.ctx, c, cmd)
		if e == nil {
			t.Fatal("fault not injected")
		}
		if f.scalar(`SELECT count(*) FROM staffing.job WHERE code=$1`, cmd.Input.Code) != 0 || f.scalar(`SELECT count(*) FROM staffing.job_terminal WHERE submission_id=$1`, cmd.SubmissionID) != 0 {
			t.Fatal("partial write/terminal committed")
		}
		resolved, e := svc.Resolve(f.ctx, c, staff.Job, cmd.SubmissionID)
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

type enterpriseGate307 struct {
	enterpriseapp.Store
	locked, release chan struct{}
}
type enterpriseUnitGate307 struct {
	enterpriseapp.Unit
	gate *enterpriseGate307
}

func (g *enterpriseGate307) WithinEnterpriseWrite(ctx context.Context, c access.Credentials, fn func(enterpriseapp.Unit) error) error {
	return g.Store.WithinEnterpriseWrite(ctx, c, func(u enterpriseapp.Unit) error { return fn(enterpriseUnitGate307{u, g}) })
}
func (u enterpriseUnitGate307) Lock(ctx context.Context, id string) error {
	if e := u.Unit.Lock(ctx, id); e != nil {
		return e
	}
	close(u.gate.locked)
	select {
	case <-u.gate.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func TestStaffing307EnterpriseBothLockOrders(t *testing.T) {
	enterpriseRace307(t, "2001-01-01", func(h *staffing307) staff.Command {
		return staff.Command{Kind: staff.Family, SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Input: staff.Input{Action: "CREATE", Code: "FLOOR-RACE", Name: name307("企业下界并发系列"), EffectiveDate: "2000-01-01", OperationType: "CREATE", OperationReason: "NEW_JOB_FAMILY"}}
	})
}
func enterpriseRace307(t *testing.T, newFloor string, command func(*staffing307) staff.Command) {
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
			parent := org.EnterpriseCommand{Action: "CORRECT", EnterpriseID: eid, RecordID: r["recordId"].(string), SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Correct: org.EnterpriseCorrectInput{ExpectedSourceVersion: ent["headSourceVersion"].(string), ExpectedRecordRevision: r["revision"].(string), NewEffectiveDate: newFloor}}
			child := command(h)
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
				var fault *foundation.Fault
				if v.Outcome != "accepted" || !errors.As(pe, &fault) || fault.Code != "M02_REFERENCE_CONFLICT" {
					t.Fatal("child-first did not protect earliest date")
				}
			} else {
				must(t, pe)
				if v.Outcome != "rejected" || v.Failure.Code != "M03_ENTERPRISE_DATE_CONFLICT" {
					t.Fatal("parent-first did not refresh floor")
				}
			}
			actual := b.call("GET", "/api/v1/enterprise/"+eid, nil, "", 200)
			if child.Kind == staff.JobFunction {
				var floor string
				must(t, f.admin.QueryRow(f.ctx, "SELECT COALESCE(min(first_date)::text,'') FROM staffing.job_function WHERE instance_id=$1 AND enterprise_id=$2", f.instance, eid).Scan(&floor))
				date := rec307(actual)["effectiveDate"].(string)
				if floor != "" && floor < date || first == "enterprise" && date != newFloor || first == "child" && date != r["effectiveDate"] {
					t.Fatalf("enterprise %s function %s", date, floor)
				}
			}
		})
	}
}

func TestStaffing307ReassignmentAndReactivationBothLockOrders(t *testing.T) {
	h := staffingSetup307(t)
	svc := h.f.runtime.Staffing.(app.Service)
	c := geoCredentials203(t, h.f, "admin", h.f.password)
	for _, mode := range []string{"reassign", "reactivate"} {
		for _, first := range []string{"job", "family"} {
			t.Run(mode+"_"+first, func(t *testing.T) {
				family := h.create("jobfamily", "TARGET-"+mode+first, "")
				fid := family["id"].(string)
				initial := fid
				if mode == "reassign" {
					initial = h.create("jobfamily", "ORIGINAL-"+first, "")["id"].(string)
				}
				job := h.create("job", "JOB-"+mode+first, initial)
				if mode == "reactivate" {
					h.add("job", job, "DEACTIVATE", "2005-01-01", nil, 200)
					job = h.refresh("job", job)
				}
				r := rec307(job)
				in := staff.Input{Action: "CORRECT", ExpectedSourceVersion: job["headSourceVersion"].(string), RecordID: r["recordId"].(string), ExpectedRecordRevision: r["revision"].(string), Changes: map[string]*string{"jobFamilyId": &fid}}
				if mode == "reactivate" {
					in = staff.Input{Action: "REACTIVATE", ExpectedSourceVersion: job["headSourceVersion"].(string), AnchorRecordID: r["recordId"].(string), EffectiveDate: "2010-01-01", OperationType: "REACTIVATE", OperationReason: "JOB_RESUMPTION"}
				}
				jobCmd := staff.Command{Kind: staff.Job, ObjectID: job["id"].(string), SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Input: in}
				familyCmd := staff.Command{Kind: staff.Family, ObjectID: fid, SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Input: staff.Input{Action: "DEACTIVATE", ExpectedSourceVersion: family["headSourceVersion"].(string), AnchorRecordID: rec307(family)["recordId"].(string), EffectiveDate: "2030-01-01", OperationType: "DEACTIVATE", OperationReason: "FAMILY_CLOSURE"}}
				a, b := jobCmd, familyCmd
				if first == "family" {
					a, b = b, a
				}
				gate := &gate307{Store: svc.Store, locked: make(chan struct{}), release: make(chan struct{})}
				type outcome struct {
					value staff.Result
					err   error
				}
				one, two := make(chan outcome, 1), make(chan outcome, 1)
				go func() { v, e := (app.Service{Store: gate}).Execute(h.f.ctx, c, a); one <- outcome{v, e} }()
				wait307(t, h.f, gate.locked)
				go func() { v, e := svc.Execute(h.f.ctx, c, b); two <- outcome{v, e} }()
				blocked307(t, h.f, gate.release)
				close(gate.release)
				x, y := <-one, <-two
				must(t, x.err)
				must(t, y.err)
				if x.value.Outcome != "accepted" || y.value.Outcome != "rejected" || y.value.Failure.Code != "M03_REFERENCE_CONFLICT" {
					t.Fatalf("invalid ordered outcomes: %s / %s, first=%+v second=%+v", x.value.Outcome, y.value.Outcome, x.value.Failure, y.value.Failure)
				}
			})
		}
	}
}

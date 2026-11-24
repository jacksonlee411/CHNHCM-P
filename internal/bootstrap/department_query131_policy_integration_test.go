//go:build integration && query131

package bootstrap_test

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	accessdb "chnhcm.local/chnhcm/internal/contexts/access/infra/postgres"
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	"chnhcm.local/chnhcm/internal/foundation"
)

func TestDepartmentQuery131PolicyGuardOrdersAndSameSession(t *testing.T) {
	f, admin, q := query131Fixture(t)
	bu := query131BU(t, admin, "POLICY_BU")
	query131Create(t, admin, "POLICY_DEP", "policy department", bu, "2026-01-01")
	reader := f.login("reader", f.readPassword)
	first, g1, in1 := query131Guard(t, f, reader, q)
	defer first.Rollback(context.Background())
	second, g2, in2 := query131Guard(t, f, reader, q)
	defer second.Rollback(context.Background())
	ctx, cancel := context.WithTimeout(f.ctx, 15*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		tx, e := f.admin.Begin(ctx)
		if e != nil {
			done <- e
			return
		}
		defer tx.Rollback(context.Background())
		if _, e = tx.Exec(ctx, `SELECT set_config('application_name','q131-revoke-after-read',true)`); e == nil {
			_, e = tx.Exec(ctx, `SELECT policy_no FROM access.policy_guard WHERE instance_id=$1 AND principal_id=$2 FOR UPDATE`, f.instance, reader.subject)
		}
		if e == nil {
			_, e = tx.Exec(ctx, `DELETE FROM access.subject_grant WHERE instance_id=$1 AND principal_id=$2 AND action='department.read'`, f.instance, reader.subject)
		}
		if e == nil {
			e = tx.Commit(ctx)
		}
		done <- e
	}()
	query131WaitLock(t, f, "q131-revoke-after-read")
	v, _, e := history.CapturePlanned131(f.ctx, first, g1, in1)
	must(t, e)
	if v.Count != 1 {
		t.Fatal(v)
	}
	v, _, e = history.CapturePlanned131(f.ctx, second, g2, in2)
	must(t, e)
	if v.Count != 1 {
		t.Fatal(v)
	}
	must(t, g1.RecordSessionActivity(f.ctx))
	must(t, first.Commit(f.ctx))
	must(t, g2.RecordSessionActivity(f.ctx))
	must(t, second.Commit(f.ctx))
	must(t, <-done)
	_, _, e = query131PlanRead(t, f, reader, q, in1.Query)
	var fault *foundation.Fault
	if !errors.As(e, &fault) || fault.Status != 403 {
		t.Fatal("new guard reused revoked rights", e)
	}
	// Reverse order: a reader must wait behind an already-held exclusive policy
	// guard and compile only the grants visible after that writer commits.
	query131Grant(t, f, reader, "bu", bu, "department.read", []string{"code", "name"})
	gate, e := f.admin.Begin(ctx)
	must(t, e)
	defer gate.Rollback(context.Background())
	_, e = gate.Exec(ctx, `SELECT policy_no FROM access.policy_guard WHERE instance_id=$1 AND principal_id=$2 FOR UPDATE`, f.instance, reader.subject)
	must(t, e)
	_, e = gate.Exec(ctx, `DELETE FROM access.subject_grant WHERE instance_id=$1 AND principal_id=$2 AND action='department.read'`, f.instance, reader.subject)
	must(t, e)
	u, e := url.Parse(f.server.URL)
	must(t, e)
	credentials := access.Credentials{Token: reader.client.Jar.Cookies(u)[0].Value}
	go func() {
		tx, e := f.runtime.Pool.Begin(ctx)
		if e != nil {
			done <- e
			return
		}
		defer tx.Rollback(context.Background())
		_, e = tx.Exec(ctx, `SELECT set_config('application_name','q131-read-after-revoke',true)`)
		if e != nil {
			done <- e
			return
		}
		store := accessdb.Store{Pool: f.runtime.Pool, Instance: f.instance}
		g, e := store.Guard(ctx, tx, credentials)
		if e != nil {
			done <- e
			return
		}
		p, e := g.(access.ReadPlanner).PlanReads(ctx, q.Enterprise, in1.Plan.At, access.DepartmentRead)
		if e != nil {
			done <- e
			return
		}
		in := in1
		in.Plan = p
		_, _, e = history.CapturePlanned131(ctx, tx, g, in)
		done <- e
	}()
	query131WaitLock(t, f, "q131-read-after-revoke")
	must(t, gate.Commit(ctx))
	e = <-done
	if !errors.As(e, &fault) || fault.Status != 403 {
		t.Fatal("read crossed revocation boundary", e)
	}
	t.Log("two live same-session guarded reads completed; both revocation lock orders observed with PostgreSQL barriers; next reads denied")
}

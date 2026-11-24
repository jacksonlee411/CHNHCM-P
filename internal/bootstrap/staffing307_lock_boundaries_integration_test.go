//go:build integration

package bootstrap_test

import (
	app "chnhcm.local/chnhcm/internal/contexts/staffing/application"
	db "chnhcm.local/chnhcm/internal/contexts/staffing/infra/postgres"
	staff "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type replanTrace307 struct {
	once              sync.Once
	reached, release  chan struct{}
	guards, rollbacks atomic.Int32
}

func (x *replanTrace307) TraceQueryStart(ctx context.Context, _ *pgx.Conn, v pgx.TraceQueryStartData) context.Context {
	if strings.Contains(v.SQL, "pg_advisory_xact_lock_shared") {
		count := 1
		if len(v.Args) == 1 {
			if keys, ok := v.Args[0].([]int64); ok {
				count = len(keys)
			}
		}
		x.guards.Add(int32(count))
		x.once.Do(func() {
			close(x.reached)
			select {
			case <-x.release:
			case <-ctx.Done():
			}
		})
	}
	if strings.HasPrefix(v.SQL, "rollback to savepoint") {
		x.rollbacks.Add(1)
	}
	return ctx
}
func (x *replanTrace307) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}
func TestStaffing307ReplanAfterUnlockedDependencyChange(t *testing.T) {
	h := staffingSetup307(t)
	f := h.f
	a, b := h.create("jobfamily", "REPLAN-A", ""), h.create("jobfamily", "REPLAN-B", "")
	j := h.create("job", "REPLAN-J", a["id"].(string))
	original := f.runtime.Staffing.(app.Service)
	store := *original.Store.(*db.Store)
	trace := &replanTrace307{reached: make(chan struct{}), release: make(chan struct{})}
	cfg := store.Pool.Config()
	cfg.ConnConfig.Tracer = trace
	p, e := pgxpool.NewWithConfig(f.ctx, cfg)
	must(t, e)
	defer p.Close()
	store.Pool = p
	c := geoCredentials203(t, f, "admin", f.password)
	cmd := staff.Command{Kind: staff.Job, ObjectID: j["id"].(string), SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Input: staff.Input{Action: "CORRECT", ExpectedSourceVersion: j["headSourceVersion"].(string), RecordID: rec307(j)["recordId"].(string), ExpectedRecordRevision: rec307(j)["revision"].(string), Changes: map[string]*string{"name": name307("旧来源写入")}}}
	done := make(chan staff.Result, 1)
	errs := make(chan error, 1)
	go func() { v, e := (app.Service{Store: &store}).Execute(f.ctx, c, cmd); done <- v; errs <- e }()
	wait307(t, f, trace.reached)
	h.correct("job", j, map[string]any{"jobFamilyId": b["id"]}, nil, 200)
	close(trace.release)
	must(t, <-errs)
	v := <-done
	if v.Outcome != "rejected" || v.Failure.Code != "M03_CONFLICT" || trace.rollbacks.Load() < 1 || trace.guards.Load() < 3 {
		t.Fatalf("replan not established: outcome=%s rollback=%d guards=%d", v.Outcome, trace.rollbacks.Load(), trace.guards.Load())
	}
	if rec307(h.refresh("job", j))["jobFamilyId"] != b["id"] {
		t.Fatal("stale writer replaced committed classification")
	}
}
func key307(parts ...string) int64 {
	b, _ := json.Marshal(parts)
	sum := sha256.Sum256(b)
	return int64(binary.BigEndian.Uint64(sum[:8]))
}
func TestStaffing307CumulativeBudgetCancellationAndResolveRace(t *testing.T) {
	h := staffingSetup307(t)
	f := h.f
	family := h.create("jobfamily", "BUDGET-F", "")
	svc := f.runtime.Staffing.(app.Service)
	c := geoCredentials203(t, f, "admin", f.password)
	submission, e := f.admin.Begin(f.ctx)
	must(t, e)
	defer submission.Rollback(f.ctx)
	guard, e := f.admin.Begin(f.ctx)
	must(t, e)
	defer guard.Rollback(f.ctx)
	cmd := h.command307(family["id"].(string), "BUDGET-J")
	_, e = submission.Exec(f.ctx, "SELECT pg_advisory_xact_lock($1)", key307("staffing.submission", "job", f.instance, h.b.subject, cmd.SubmissionID))
	must(t, e)
	_, e = guard.Exec(f.ctx, "SELECT pg_advisory_xact_lock($1)", key307("staffing.family", f.instance, family["id"].(string)))
	must(t, e)
	done := make(chan error, 1)
	started := time.Now()
	go func() { _, e := svc.Execute(f.ctx, c, cmd); done <- e }()
	release := make(chan struct{})
	blocked307(t, f, release)
	time.Sleep(1100 * time.Millisecond)
	must(t, submission.Commit(f.ctx))
	e = <-done
	elapsed := time.Since(started)
	if e == nil || elapsed > 2900*time.Millisecond || elapsed < 1700*time.Millisecond {
		t.Fatalf("cumulative budget missing: duration=%s err=%v", elapsed, e)
	}
	must(t, guard.Commit(f.ctx))
	v, e := svc.Resolve(f.ctx, c, staff.Job, cmd.SubmissionID)
	must(t, e)
	if v.Outcome != "not_applied" {
		t.Fatal("timed-out write left fact")
	}
	t.Logf("two-lock cumulative budget=%s, original safely closed", elapsed)
	// Cancel after actual submission/enterprise/family/job locks. All are released.
	gate := &gate307{Store: svc.Store, locked: make(chan struct{}), release: make(chan struct{})}
	cancelled := h.command307(family["id"].(string), "CANCEL-J")
	ctx, cancel := context.WithCancel(f.ctx)
	go func() { _, e := (app.Service{Store: gate}).Execute(ctx, c, cancelled); done <- e }()
	wait307(t, f, gate.locked)
	cancel()
	if <-done == nil {
		t.Fatal("cancelled write succeeded")
	}
	v, e = svc.Resolve(f.ctx, c, staff.Job, cancelled.SubmissionID)
	must(t, e)
	if v.Outcome != "not_applied" {
		t.Fatal("cancel leaked source")
	}
	// The original writer owns its submission lock while resolve waits.
	gate = &gate307{Store: svc.Store, locked: make(chan struct{}), release: make(chan struct{})}
	late := h.command307(family["id"].(string), "LATE-J")
	written := make(chan staff.Result, 1)
	go func() { v, e := (app.Service{Store: gate}).Execute(f.ctx, c, late); written <- v; done <- e }()
	wait307(t, f, gate.locked)
	resolved := make(chan staff.Result, 1)
	go func() { v, e := svc.Resolve(f.ctx, c, staff.Job, late.SubmissionID); resolved <- v; done <- e }()
	blocked307(t, f, gate.release)
	close(gate.release)
	must(t, <-done)
	must(t, <-done)
	a, b := <-written, <-resolved
	if a.Outcome != "accepted" || b.Outcome != "accepted" || a.SourceVersion != b.SourceVersion {
		t.Fatal("late writer lost original identity")
	}
	v, e = svc.Execute(f.ctx, c, h.command307(family["id"].(string), "AFTER-CANCEL"))
	must(t, e)
	if v.Outcome != "accepted" {
		t.Fatal("locks or connection not released")
	}
}

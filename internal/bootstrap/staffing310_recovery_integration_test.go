//go:build integration

package bootstrap_test

import (
	app "chnhcm.local/chnhcm/internal/contexts/staffing/application"
	db "chnhcm.local/chnhcm/internal/contexts/staffing/infra/postgres"
	staff "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"testing"
	"time"
)

func TestStaffing310RecoveryAndLockBudget(t *testing.T) {
	h := staffingSetup307(t)
	f := h.f
	svc := f.runtime.Staffing.(app.Service)
	c := geoCredentials203(t, f, "admin", f.password)
	family := h.create("jobfamily", "RECOVERY310-F", "")
	a, b := h.create("grade", "RECOVERY310-A", ""), h.create("grade", "RECOVERY310-B", "")
	sharedSubmission := foundation.NewID()
	for _, k := range []staff.Kind{staff.Job, staff.Ladder} {
		t.Run(string(k), func(t *testing.T) {
			v := create310(h, string(k), "RECOVERY310", []string{a["id"].(string)}, family["id"].(string))
			command := func(name string) staff.Command {
				r := rec307(v)
				return staff.Command{Kind: k, ObjectID: v["id"].(string), SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Input: staff.Input{Action: "CORRECT", RecordID: r["recordId"].(string), ExpectedRecordRevision: r["revision"].(string), ExpectedSourceVersion: v["headSourceVersion"].(string), Changes: map[string]*string{"name": name307(name)}}}
			}
			cmd := command("COMMIT确认丢失")
			ids := []string{a["id"].(string), b["id"].(string)}
			cmd.Input.ChangedGradeIDs = &ids
			cmd.SubmissionID = sharedSubmission
			if _, e := (app.Service{Store: lostCommit309{svc.Store}}).Execute(f.ctx, c, cmd); e == nil {
				t.Fatal("missing commit fault")
			}
			result, e := svc.Resolve(f.ctx, c, k, cmd.SubmissionID)
			must(t, e)
			replay, e := svc.Execute(f.ctx, c, cmd)
			must(t, e)
			if result.Outcome != "accepted" || result.SourceVersion != replay.SourceVersion {
				t.Fatal("commit not recovered by original identity")
			}
			table := map[staff.Kind]string{staff.Job: "job", staff.Ladder: "grade_ladder"}[k]
			if f.scalar("SELECT count(*) FROM staffing."+table+"_source") != 2 || f.scalar("SELECT count(*) FROM staffing."+table+"_grade_member WHERE source_id=$1", result.SourceVersion) != 2 {
				t.Fatal("commit duplicated or lost members")
			}
			other := f.login("reader", f.readPassword)
			isolated := other.call("POST", base307(string(k))+"/submissions/"+sharedSubmission+"/resolve", nil, "", 200)
			if isolated["outcome"] != "not_applied" || isolated["sourceVersion"] != nil || isolated["objectId"] != nil {
				t.Fatal("another account recovered original result")
			}
			v = h.refresh(string(k), v)
			// Submission and Grade waits consume a single two-second budget.
			cmd = command("累计锁预算")
			submission, e := f.admin.Begin(f.ctx)
			must(t, e)
			defer submission.Rollback(f.ctx)
			guard, e := f.admin.Begin(f.ctx)
			must(t, e)
			defer guard.Rollback(f.ctx)
			_, e = submission.Exec(f.ctx, "SELECT pg_advisory_xact_lock($1)", key307("staffing.submission", string(k), f.instance, h.b.subject, cmd.SubmissionID))
			must(t, e)
			_, e = guard.Exec(f.ctx, "SELECT pg_advisory_xact_lock($1)", key307("staffing.grade", f.instance, a["id"].(string)))
			must(t, e)
			done := make(chan error, 2)
			start := time.Now()
			go func() { _, e := svc.Execute(f.ctx, c, cmd); done <- e }()
			blocked307(t, f, make(chan struct{}))
			time.Sleep(1100 * time.Millisecond)
			must(t, submission.Commit(f.ctx))
			e = <-done
			elapsed := time.Since(start)
			if e == nil || elapsed < 1700*time.Millisecond || elapsed > 2900*time.Millisecond {
				t.Fatalf("cumulative budget: %s %v", elapsed, e)
			}
			must(t, guard.Commit(f.ctx))
			result, e = svc.Resolve(f.ctx, c, k, cmd.SubmissionID)
			must(t, e)
			if result.Outcome != "not_applied" {
				t.Fatal("timed out write created fact")
			}
			gate := &gate307{Store: svc.Store, locked: make(chan struct{}), release: make(chan struct{})}
			ctx, cancel := context.WithCancel(f.ctx)
			cmd = command("取消事务")
			go func() { _, e := (app.Service{Store: gate}).Execute(ctx, c, cmd); done <- e }()
			wait307(t, f, gate.locked)
			cancel()
			if <-done == nil {
				t.Fatal("cancel committed")
			}
			result, e = svc.Resolve(f.ctx, c, k, cmd.SubmissionID)
			must(t, e)
			if result.Outcome != "not_applied" {
				t.Fatal("cancel left fact")
			}
			gate = &gate307{Store: svc.Store, locked: make(chan struct{}), release: make(chan struct{})}
			cmd = command("迟到保存")
			written, resolved := make(chan staff.Result, 1), make(chan staff.Result, 1)
			go func() { r, e := (app.Service{Store: gate}).Execute(f.ctx, c, cmd); written <- r; done <- e }()
			wait307(t, f, gate.locked)
			go func() { r, e := svc.Resolve(f.ctx, c, k, cmd.SubmissionID); resolved <- r; done <- e }()
			blocked307(t, f, gate.release)
			close(gate.release)
			must(t, <-done)
			must(t, <-done)
			one, two := <-written, <-resolved
			if one.Outcome != "accepted" || two.SourceVersion != one.SourceVersion {
				t.Fatal("late original submit not serialized")
			}
			if f.scalar("SELECT count(*) FROM staffing."+table+"_source") != 3 {
				t.Fatal("cancel/recovery duplicated source")
			}
			t.Logf("%s cumulative=%s originalCommit=%s lateResolve=true cancellationZeroFacts=true", k, elapsed, result.Outcome)
		})
	}
}

func TestStaffing310GradeReplanAndSwaps(t *testing.T) {
	h := staffingSetup307(t)
	f := h.f
	original := f.runtime.Staffing.(app.Service)
	c := geoCredentials203(t, f, "admin", f.password)
	family := h.create("jobfamily", "REPLAN310-F", "")
	a, b := h.create("grade", "REPLAN310-A", ""), h.create("grade", "REPLAN310-B", "")
	aid, bid := a["id"].(string), b["id"].(string)
	for _, k := range []staff.Kind{staff.Job, staff.Ladder} {
		t.Run(string(k), func(t *testing.T) {
			v := create310(h, string(k), "REPLAN310", []string{aid}, family["id"].(string))
			store := *original.Store.(*db.Store)
			trace := &replanTrace307{reached: make(chan struct{}), release: make(chan struct{})}
			cfg := store.Pool.Config()
			cfg.ConnConfig.Tracer = trace
			p, e := pgxpool.NewWithConfig(f.ctx, cfg)
			must(t, e)
			defer p.Close()
			store.Pool = p
			r := rec307(v)
			cmd := staff.Command{Kind: k, ObjectID: v["id"].(string), SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Input: staff.Input{Action: "CORRECT", ExpectedSourceVersion: v["headSourceVersion"].(string), RecordID: r["recordId"].(string), ExpectedRecordRevision: r["revision"].(string), Changes: map[string]*string{"name": name307("旧头")}}}
			done := make(chan staff.Result, 1)
			errs := make(chan error, 2)
			go func() { r, e := (app.Service{Store: &store}).Execute(f.ctx, c, cmd); done <- r; errs <- e }()
			wait307(t, f, trace.reached)
			h.correct(string(k), v, map[string]any{"gradeIds": []string{bid}}, nil, 200)
			close(trace.release)
			must(t, <-errs)
			out := <-done
			if out.Outcome != "rejected" || out.Failure.Code != "M03_CONFLICT" || trace.rollbacks.Load() < 1 || trace.guards.Load() < 3 {
				t.Fatalf("no safe replan: %+v rollback=%d guards=%d", out, trace.rollbacks.Load(), trace.guards.Load())
			}
			// Opposite member changes on different consumers share sorted Grade guards.
			v = h.refresh(string(k), v)
			other := create310(h, string(k), "SWAP310", []string{aid}, family["id"].(string))
			results := make(chan staff.Result, 2)
			ready := make(chan struct{})
			for i, obj := range []map[string]any{v, other} {
				ids := []string{aid}
				if i == 1 {
					ids = []string{bid}
				}
				r := rec307(obj)
				cmd := staff.Command{Kind: k, ObjectID: obj["id"].(string), SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Input: staff.Input{Action: "CORRECT", ExpectedSourceVersion: obj["headSourceVersion"].(string), RecordID: r["recordId"].(string), ExpectedRecordRevision: r["revision"].(string), ChangedGradeIDs: &ids}}
				go func() { <-ready; r, e := original.Execute(f.ctx, c, cmd); results <- r; errs <- e }()
			}
			close(ready)
			for range 2 {
				must(t, <-errs)
				if (<-results).Outcome != "accepted" {
					t.Fatal("opposite swaps failed")
				}
			}
			t.Log(fmt.Sprintf("%s rollback=%d sharedGuardQueries=%d swaps=2", k, trace.rollbacks.Load(), trace.guards.Load()))
		})
	}
}

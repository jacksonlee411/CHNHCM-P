//go:build integration

package bootstrap_test

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	buapp "chnhcm.local/chnhcm/internal/contexts/organization/application"
	depapp "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	orgdb "chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Record durations and SQLSTATE only, never statements, parameters or credentials.
type departmentBudgetTrace struct {
	mu     sync.Mutex
	events []departmentBudgetEvent
}
type departmentBudgetEvent struct {
	Phase, SQLState string
	Elapsed, Wall   time.Duration
	PID             uint32
}
type departmentBudgetQuery struct {
	start time.Time
	phase string
}
type departmentBudgetTraceKey struct{}

func (x *departmentBudgetTrace) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	phase := "sql"
	if strings.Contains(data.SQL, "pg_advisory_xact_lock") {
		phase = "advisory-lock"
	} else if strings.HasPrefix(data.SQL, "rollback") {
		phase = "rollback"
	} else if strings.HasPrefix(data.SQL, "savepoint") {
		phase = "savepoint"
	}
	return context.WithValue(ctx, departmentBudgetTraceKey{}, departmentBudgetQuery{time.Now(), phase})
}

func (x *departmentBudgetTrace) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	q := ctx.Value(departmentBudgetTraceKey{}).(departmentBudgetQuery)
	end := time.Now()
	e := departmentBudgetEvent{Phase: q.phase, Elapsed: end.Sub(q.start), Wall: end.Round(0).Sub(q.start.Round(0)), PID: conn.PgConn().PID()}
	var pg *pgconn.PgError
	if errors.As(data.Err, &pg) {
		e.SQLState = pg.Code
	}
	if e.Phase != "sql" || e.Elapsed > 50*time.Millisecond {
		x.mu.Lock()
		x.events = append(x.events, e)
		x.mu.Unlock()
	}
}

func (x *departmentBudgetTrace) log(t *testing.T) {
	t.Helper()
	x.mu.Lock()
	defer x.mu.Unlock()
	for _, e := range x.events {
		t.Logf("phase=%s monotonic=%s wall=%s clockDelta=%s sqlstate=%s", e.Phase, e.Elapsed, e.Wall, e.Wall-e.Elapsed, e.SQLState)
	}
}

func departmentBudgetWait(t *testing.T, f *fixture) {
	t.Helper()
	until := time.Now().Add(2 * time.Second)
	for time.Now().Before(until) {
		if f.scalar(`SELECT count(*) FROM pg_stat_activity WHERE usename='chnhcm_api' AND wait_event_type='Lock' AND cardinality(pg_blocking_pids(pid))>0`) > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("request did not reach the real database lock")
}

func TestDepartmentV07GuardBudgetWithoutDatabaseLockTimer(t *testing.T) {
	for _, object := range []string{"department", "bu_reverse"} {
		t.Run(object, func(t *testing.T) {
			f := fresh(t)
			b := f.login("admin", f.password)
			bu := b.call("POST", "/api/v1/business-units", createBody("GUARD_BU", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
			created := b.call("POST", "/api/v1/departments", departmentCreateBody("GUARD_DEP", bu), foundation.NewID(), 201)
			d := b.call("GET", "/api/v1/departments/"+created["departmentId"].(string), nil, "", 200)
			r := d["timeline"].([]any)[0].(map[string]any)
			path := "/api/v1/departments/" + created["departmentId"].(string) + "/records/" + r["recordId"].(string) + "/correct"
			var enterprise string
			must(t, f.admin.QueryRow(f.ctx, "SELECT enterprise_id::text FROM organization.enterprise WHERE instance_id=$1", f.instance).Scan(&enterprise))
			if object == "bu_reverse" {
				d = detail(b, bu)
				r = d["timeline"].([]any)[0].(map[string]any)
				path = "/api/v1/business-units/" + bu + "/records/" + r["recordId"].(string) + "/correct"
			}
			body := map[string]any{"expectedSourceVersion": d["headSourceVersion"], "expectedRecordRevision": r["revision"], "changes": map[string]any{"name": "预算后重试"}}
			// Independent guard encoding, as in V07; the real database owns the wait.
			raw, e := json.Marshal([]string{"organization.department.admin-relations", f.instance, enterprise, bu, "DEPARTMENT_ADMIN"})
			must(t, e)
			sum := sha256.Sum256(raw)
			blocker, e := f.admin.Begin(f.ctx)
			must(t, e)
			defer blocker.Rollback(context.Background())
			_, e = blocker.Exec(f.ctx, "SELECT pg_advisory_xact_lock($1)", int64(binary.BigEndian.Uint64(sum[:8])))
			must(t, e)
			var rowBlocker pgx.Tx
			if object == "department" {
				rowBlocker, e = f.admin.Begin(f.ctx)
				must(t, e)
				defer rowBlocker.Rollback(context.Background())
				_, e = rowBlocker.Exec(f.ctx, "SELECT FROM organization.business_unit WHERE bu_id=$1 FOR UPDATE", bu)
				must(t, e)
			}

			buService := f.runtime.Organization.(buapp.Service)
			depService := f.runtime.Department.(depapp.Service)
			originalBU, originalDep := buService.Store, depService.Store
			store := *originalDep.(*orgdb.Store)
			trace := &departmentBudgetTrace{}
			config := store.Pool.Config().Copy()
			config.ConnConfig.Tracer = trace
			pool, e := pgxpool.NewWithConfig(f.ctx, config)
			must(t, e)
			defer pool.Close()
			store.Pool = pool
			originalGuard := store.Guard
			store.Guard = func(ctx context.Context, tx pgx.Tx, c access.Credentials) (access.Guard, error) {
				g, err := originalGuard(ctx, tx, c)
				if err == nil {
					// Model a late DB timer deterministically without touching host time.
					_, err = tx.Exec(ctx, "SET LOCAL lock_timeout='0'")
				}
				return g, err
			}
			buService.Store, depService.Store = &store, &store
			f.runtime.Organization, f.runtime.Department = buService, depService
			defer func() {
				buService.Store, depService.Store = originalBU, originalDep
				f.runtime.Organization, f.runtime.Department = buService, depService
			}()

			facts := func() map[string]int {
				out := map[string]int{}
				for _, table := range []string{"bu_history", "bu_terminal", "business_unit", "department_history", "department_hierarchy_relation", "department_terminal"} {
					out[table] = f.scalar("SELECT count(*) FROM organization." + table)
				}
				return out
			}
			before := facts()
			id := foundation.NewID()
			started := time.Now()
			done := make(chan independent207Reply, 1)
			go func() {
				status, result, err := b.response("POST", path, body, id)
				done <- independent207Reply{status, result, err}
			}()
			departmentBudgetWait(t, f)
			if rowBlocker != nil {
				// Spend part of the budget on the first real row lock; the later
				// relation guard must receive only the remaining time.
				time.Sleep(1200 * time.Millisecond)
				must(t, rowBlocker.Rollback(f.ctx))
				departmentBudgetWait(t, f)
			}
			var backend int
			must(t, f.admin.QueryRow(f.ctx, `SELECT pid FROM pg_stat_activity WHERE usename='chnhcm_api' AND wait_event_type='Lock'`).Scan(&backend))
			answer := <-done
			elapsed := time.Since(started)
			trace.log(t)
			t.Logf("object=%s request=%s status=%d code=%v", object, elapsed, answer.status, answer.body["code"])
			must(t, answer.err)
			if elapsed > 4*time.Second || answer.status != 503 || answer.body["outcome"] != nil {
				t.Fatal("object guard budget depended on database wall-clock timer", elapsed, answer.status, answer.body)
			}
			if rowBlocker != nil && elapsed > 3*time.Second {
				t.Fatal("relation guard renewed the budget after the BU row wait", elapsed)
			}
			if !reflect.DeepEqual(before, facts()) {
				t.Fatal("guard failure left business facts or a terminal")
			}
			until := time.Now().Add(2 * time.Second)
			for f.scalar(`SELECT count(*) FROM pg_locks WHERE pid=$1 AND locktype IN ('advisory','transactionid','tuple')`, backend) != 0 && time.Now().Before(until) {
				time.Sleep(10 * time.Millisecond)
			}
			if f.scalar(`SELECT count(*) FROM pg_locks WHERE pid=$1 AND locktype IN ('advisory','transactionid','tuple')`, backend) != 0 {
				t.Fatal("request ended but database retained transactional locks")
			}
			must(t, blocker.Rollback(f.ctx))
			store.Guard = originalGuard
			saved := b.call("POST", path, body, id, 200)
			if saved["outcome"] != "accepted" {
				t.Fatal("original identity could not retry", saved)
			}
		})
	}
}

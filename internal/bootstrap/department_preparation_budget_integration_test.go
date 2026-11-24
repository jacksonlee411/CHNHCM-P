//go:build integration

package bootstrap_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	depapp "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	orgdb "chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Delay only a test-owned fact read, respecting its real request context. This
// models a slow preparation independently of database lock contention/timers.
type departmentPreparationDelay struct {
	mu        sync.Mutex
	delays    []time.Duration
	seen      int
	cancelled bool
}

func (x *departmentPreparationDelay) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	if !strings.Contains(data.SQL, "FROM organization.department d WHERE") {
		return ctx
	}
	x.mu.Lock()
	position := x.seen
	x.seen++
	var delay time.Duration
	if position < len(x.delays) {
		delay = x.delays[position]
	}
	x.mu.Unlock()
	if delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-ctx.Done():
			x.mu.Lock()
			x.cancelled = true
			x.mu.Unlock()
		}
	}
	return ctx
}

func (*departmentPreparationDelay) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

func TestDepartmentPreparationSeparatesFactReadsFromLockBudget(t *testing.T) {
	for _, test := range []struct {
		name   string
		delays []time.Duration
		status int
	}{
		{"preliminary_read_over_two_seconds", []time.Duration{2200 * time.Millisecond}, 200},
		{"locked_reread_over_two_seconds", []time.Duration{0, 2200 * time.Millisecond}, 200},
		{"both_reads_share_five_seconds", []time.Duration{2700 * time.Millisecond, 2700 * time.Millisecond}, 503},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := fresh(t)
			b := f.login("admin", f.password)
			bu := b.call("POST", "/api/v1/business-units", createBody("PREP_BU", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
			created := b.call("POST", "/api/v1/departments", departmentCreateBody("PREP_DEP", bu), foundation.NewID(), 201)
			id := created["departmentId"].(string)
			detail := b.call("GET", "/api/v1/departments/"+id, nil, "", 200)
			record := detail["selectedRecord"].(map[string]any)
			path := "/api/v1/departments/" + id + "/records/" + record["recordId"].(string) + "/correct"
			body := map[string]any{"expectedSourceVersion": detail["headSourceVersion"], "expectedRecordRevision": record["revision"], "changes": map[string]any{"name": "准备预算后保存"}}
			service := f.runtime.Department.(depapp.Service)
			original := service.Store
			store := *original.(*orgdb.Store)
			trace := &departmentPreparationDelay{delays: test.delays}
			config := store.Pool.Config().Copy()
			config.ConnConfig.Tracer = trace
			pool, err := pgxpool.NewWithConfig(f.ctx, config)
			must(t, err)
			defer pool.Close()
			store.Pool = pool
			service.Store = &store
			f.runtime.Department = service
			defer func() { service.Store = original; f.runtime.Department = service }()
			key := foundation.NewID()
			started := time.Now()
			reply := b.call("POST", path, body, key, test.status)
			elapsed := time.Since(started)
			trace.mu.Lock()
			seen, cancelled := trace.seen, trace.cancelled
			trace.mu.Unlock()
			t.Logf("preparation=%s seen=%d cancelled=%t status=%d", elapsed, seen, cancelled, test.status)
			if seen < len(test.delays) || elapsed < 2*time.Second || elapsed > 6500*time.Millisecond {
				t.Fatal("preparation delay or bounded deadline not exercised", elapsed, seen)
			}
			if test.status == 503 {
				if !cancelled || elapsed < 4500*time.Millisecond || reply["outcome"] != nil {
					t.Fatal("preparation renewed its deadline or forged a terminal", reply)
				}
				if f.scalar("SELECT (SELECT count(*) FROM organization.department_history h CROSS JOIN LATERAL unnest(h.submission_refs) r WHERE h.ids[r]=$1::uuid)+(SELECT count(*) FROM organization.department_terminal WHERE submission_id=$1)", key) != 0 {
					t.Fatal("preparation timeout persisted a business result")
				}
				// The exhausted delay is a test-only fixture. The same original
				// identity remains valid after rollback, with no hidden replacement.
				reply = b.call("POST", path, body, key, 200)
			} else if cancelled {
				t.Fatal("successful preparation lost its context")
			}
			if reply["outcome"] != "accepted" || reply["submissionId"] != key {
				t.Fatal("ordinary save or same-identity recovery failed", reply)
			}
		})
	}
}

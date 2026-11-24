//go:build integration

package bootstrap_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	dep "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	orgdb "chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type capacityRequestKey struct{}
type capacityTrace struct{ captures chan int }

func (tr *capacityTrace) TraceQueryStart(ctx context.Context, _ *pgx.Conn, d pgx.TraceQueryStartData) context.Context {
	if d.SQL == history.DepartmentQuerySQL {
		if id, ok := ctx.Value(capacityRequestKey{}).(int); ok {
			tr.captures <- id
		}
	}
	return ctx
}
func (tr *capacityTrace) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

// Real HTTP requests occupy twelve production transaction connections. Each
// list captures through its own outer transaction; eight additional requests
// await the pool, with no separate fact-loader connection or cache ownership.
func TestDepartmentCapacitySaturationCancellationAndRecovery(t *testing.T) {
	f := fresh(t)
	actors := []*browser{f.login("admin", f.password)}
	writer := actors[0]
	bu := writer.call("POST", "/api/v1/business-units", createBody("CAPACITY_POOL_BU", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	writer.call("POST", "/api/v1/departments", departmentCreateBody("CAPACITY_POOL_DEPT", bu), foundation.NewID(), 201)
	for len(actors) < 20 {
		actors = append(actors, f.login("admin", f.password))
	}
	svc := f.runtime.Department.(dep.Service)
	store := *svc.Store.(*orgdb.Store)
	cfg := store.Pool.Config().Copy()
	trace := &capacityTrace{captures: make(chan int, 100)}
	cfg.ConnConfig.Tracer = trace
	pool, e := pgxpool.NewWithConfig(f.ctx, cfg)
	must(t, e)
	// Close the original idle pool so the whole runtime retains exactly one API
	// pool. The copied Store initializes a fresh cache for this pool identity.
	f.runtime.Pool.Close()
	f.runtime.Pool = pool
	store.Pool = pool
	svc.Store = &store
	f.runtime.Department = svc
	arrivals := make(chan int, 30)
	original := f.server.Config.Handler
	f.server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(r.Header.Get("X-Capacity-Worker"))
		arrivals <- id
		original.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), capacityRequestKey{}, id)))
	})
	block, e := f.admin.Begin(f.ctx)
	must(t, e)
	defer block.Rollback(context.Background())
	_, e = block.Exec(f.ctx, "LOCK TABLE organization.department_read_segment IN ACCESS EXCLUSIVE MODE")
	must(t, e)
	type reply struct {
		id, status int
		value      map[string]any
		err        error
	}
	answers := make(chan reply, 20)
	cancels := make([]context.CancelFunc, 20)
	var wg sync.WaitGroup
	start := func(i int) {
		ctx, cancel := context.WithCancel(context.Background())
		cancels[i] = cancel
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, err := http.NewRequestWithContext(ctx, "GET", f.server.URL+"/api/v1/departments?page=1&pageSize=20", nil)
			if err != nil {
				answers <- reply{id: i, err: err}
				return
			}
			req.Header.Set("X-Capacity-Worker", fmt.Sprint(i))
			res, err := actors[i].client.Do(req)
			if err != nil {
				answers <- reply{id: i, err: err}
				return
			}
			defer res.Body.Close()
			v := map[string]any{}
			err = json.NewDecoder(res.Body).Decode(&v)
			answers <- reply{i, res.StatusCode, v, err}
		}()
	}
	defer func() {
		for _, cancel := range cancels {
			if cancel != nil {
				cancel()
			}
		}
		wg.Wait()
	}()
	waitID := func(ch <-chan int, want int) {
		t.Helper()
		select {
		case id := <-ch:
			if id != want {
				t.Fatalf("worker=%d want=%d", id, want)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("barrier not reached")
		}
	}
	start(0)
	waitID(trace.captures, 0)
	// Identify two blocked SQL requests before filling the remaining slots.
	start(1)
	waitID(trace.captures, 1)
	for i := 2; i < 12; i++ {
		start(i)
	}
	for i := 2; i < 12; i++ {
		select {
		case <-trace.captures:
		case <-time.After(time.Second):
			t.Fatal("twelve transactions did not enter")
		}
	}
	if pool.Stat().AcquiredConns() != 12 || pool.Config().MaxConns != 12 {
		t.Fatal("wrong API connection budget", pool.Stat())
	}
	for i := 12; i < 20; i++ {
		start(i)
	}
	for i := 0; i < 20; i++ {
		select {
		case <-arrivals:
		case <-time.After(time.Second):
			t.Fatal("HTTP request did not enter handler")
		}
	}
	// No remaining request can start capture while twelve connections are held.
	select {
	case id := <-trace.captures:
		t.Fatal("extra connection obtained", id)
	case <-time.After(50 * time.Millisecond):
	}
	started := time.Now()
	cancels[1]()  // known SQL lock waiter
	cancels[19]() // known pool waiter
	cancels[0]()  // another blocked capture; its SQL and connection must release
	select {
	case id := <-trace.captures:
		if id < 12 || id > 18 {
			t.Fatal("released slot was not acquired by an uncanceled pool waiter", id)
		}
	case <-time.After(time.Second):
		t.Fatal("pool waiter did not obtain a released connection")
	}
	must(t, block.Rollback(f.ctx))
	successful, canceled := 0, 0
	for i := 0; i < 20; i++ {
		select {
		case v := <-answers:
			if v.id == 0 || v.id == 1 || v.id == 19 {
				if v.err == nil {
					t.Fatal("canceled HTTP unexpectedly succeeded", v.id)
				}
				canceled++
				continue
			}
			if v.err != nil || v.status != 200 || v.value["count"] != float64(1) || len(v.value["items"].([]any)) != 1 {
				t.Fatal("recovery failed", v)
			}
			successful++
		case <-time.After(3 * time.Second):
			t.Fatal("requests retained connection ownership")
		}
	}
	if successful != 17 || canceled != 3 || time.Since(started) > 3*time.Second {
		t.Fatal("recovery waited for request deadlines", successful, canceled, time.Since(started))
	}
	deadline := time.Now().Add(2 * time.Second)
	for pool.Stat().AcquiredConns() != 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if pool.Stat().AcquiredConns() != 0 || pool.Stat().TotalConns() > 12 {
		t.Fatal("connection leak", pool.Stat())
	}
	// A normal write after release proves canceled captures did not retain a
	// transaction or shared policy guard.
	writer.call("POST", "/api/v1/departments", departmentCreateBody("CAPACITY_POOL_AFTER", bu), foundation.NewID(), 201)
	t.Logf("20 distinct HTTP sessions, 12 transaction connections; two SQL lock waits and one pool wait canceled; 17 correct reads and later write recovered in %s", time.Since(started))
}

func TestDepartmentFactLoadSQLFailureReleasesOwnership(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	bu := b.call("POST", "/api/v1/business-units", createBody("LOAD_ERROR_BU", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	b.call("POST", "/api/v1/departments", departmentCreateBody("LOAD_ERROR_DEPT", bu), foundation.NewID(), 201)
	block, e := f.admin.Begin(f.ctx)
	must(t, e)
	defer block.Rollback(context.Background())
	_, e = block.Exec(f.ctx, "LOCK TABLE organization.department_read_segment IN ACCESS EXCLUSIVE MODE")
	must(t, e)
	started := time.Now()
	status, value, e := b.response("GET", "/api/v1/departments", nil, "")
	must(t, e)
	elapsed := time.Since(started)
	if status != 503 || value["items"] != nil || elapsed > 3*time.Second {
		t.Fatal("partial result or extended SQL lock budget", status, value, elapsed)
	}
	must(t, block.Rollback(f.ctx))
	got := b.call("GET", "/api/v1/departments", nil, "", 200)
	if got["count"] != float64(1) || len(got["items"].([]any)) != 1 {
		t.Fatal("failed loader retained ownership", got)
	}
	if f.runtime.Pool.Stat().AcquiredConns() != 0 {
		t.Fatal("transaction retained after recovery", f.runtime.Pool.Stat())
	}
	t.Logf("real SQL lock failure after %s; no partial collection; later request reloaded successfully", elapsed)
}

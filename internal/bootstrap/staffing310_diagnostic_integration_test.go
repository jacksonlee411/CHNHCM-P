//go:build integration

package bootstrap_test

import (
	app "chnhcm.local/chnhcm/internal/contexts/staffing/application"
	store "chnhcm.local/chnhcm/internal/contexts/staffing/infra/postgres"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"runtime/pprof"
	"slices"
	"sync"
	"testing"
	"time"
)

type queryCost310 struct {
	SQL          string  `json:"sql"`
	Count        int     `json:"count"`
	Milliseconds float64 `json:"totalMilliseconds"`
	Max          float64 `json:"maxMilliseconds"`
}
type queryStart310 struct {
	start time.Time
	sql   string
}
type queryKey310 struct{}
type queryTrace310 struct {
	mu    sync.Mutex
	costs map[string]*queryCost310
}

func (q *queryTrace310) start(ctx context.Context, sql string) context.Context {
	return context.WithValue(ctx, queryKey310{}, queryStart310{time.Now(), sql})
}
func (q *queryTrace310) end(ctx context.Context) {
	v := ctx.Value(queryKey310{}).(queryStart310)
	ms := float64(time.Since(v.start).Nanoseconds()) / 1e6
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.costs[v.sql] == nil {
		q.costs[v.sql] = &queryCost310{SQL: v.sql}
	}
	c := q.costs[v.sql]
	c.Count++
	c.Milliseconds += ms
	c.Max = max(c.Max, ms)
}
func (q *queryTrace310) TraceQueryStart(ctx context.Context, _ *pgx.Conn, v pgx.TraceQueryStartData) context.Context {
	return q.start(ctx, v.SQL)
}
func (q *queryTrace310) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryEndData) {
	q.end(ctx)
}
func (q *queryTrace310) TraceBatchStart(ctx context.Context, _ *pgx.Conn, v pgx.TraceBatchStartData) context.Context {
	return q.start(ctx, fmt.Sprintf("BATCH %d statements", v.Batch.Len()))
}
func (q *queryTrace310) TraceBatchQuery(context.Context, *pgx.Conn, pgx.TraceBatchQueryData) {}
func (q *queryTrace310) TraceBatchEnd(ctx context.Context, _ *pgx.Conn, _ pgx.TraceBatchEndData) {
	q.end(ctx)
}
func (q *queryTrace310) TraceCopyFromStart(ctx context.Context, _ *pgx.Conn, v pgx.TraceCopyFromStartData) context.Context {
	return q.start(ctx, "COPY "+v.TableName.Sanitize())
}
func (q *queryTrace310) TraceCopyFromEnd(ctx context.Context, _ *pgx.Conn, _ pgx.TraceCopyFromEndData) {
	q.end(ctx)
}

// This opt-in diagnostic isolates the expensive write shape. Its instrumented
// timings are never a substitute for the full, uninstrumented capacity exit.
// Only SQL templates and function stacks are persisted, never SQL arguments.
func TestStaffing310WriteDiagnostic(t *testing.T) {
	path := os.Getenv("CHNHCM_GRADE_G2_WRITE_DIAGNOSTIC")
	if path == "" {
		t.Skip("explicit write profiling only; not a capacity acceptance test")
	}
	mode := os.Getenv("CHNHCM_GRADE_G2_DIAGNOSTIC_MODE")
	if mode == "" {
		mode = "write"
	}
	if mode != "write" && mode != "candidate" {
		t.Fatal("unsupported diagnostic mode")
	}
	gradeCount, datedGrades := 10, 10
	if mode == "candidate" {
		gradeCount, datedGrades = 1000, 20
	}
	h := staffingSetup307(t)
	ids := []string{}
	addRecords := func(kind string, v map[string]any) map[string]any {
		for day := 2; day <= 20; day++ {
			r := h.add(kind, v, "ADD", fmt.Sprintf("2000-01-%02d", day), map[string]any{"name": fmt.Sprintf("诊断资料 %d", day)}, 200)
			v["headSourceVersion"] = r["sourceVersion"]
			v["selectedRecord"] = map[string]any{"recordId": r["selectedRecordId"], "revision": r["selectedRecordRevision"]}
		}
		return h.refresh(kind, v)
	}
	for i := 0; i < gradeCount; i++ {
		v := h.create("grade", fmt.Sprintf("DIAG-G-%04d", i), "")
		if i < datedGrades {
			v = addRecords("grade", v)
		}
		if i < 10 {
			ids = append(ids, v["id"].(string))
		}
	}
	objects, clients := []map[string]any{}, []*browser{}
	for i := 0; i < 20; i++ {
		objects = append(objects, addRecords("gradeladder", create310(h, "gradeladder", fmt.Sprintf("DIAG-L-%02d", i), ids, "")))
		if i == 0 {
			clients = append(clients, h.b)
		} else {
			clients = append(clients, h.f.login("admin", h.f.password))
		}
	}
	service := h.f.runtime.Staffing.(app.Service)
	copyStore := *service.Store.(*store.Store)
	trace := &queryTrace310{costs: map[string]*queryCost310{}}
	config := copyStore.Pool.Config()
	config.ConnConfig.Tracer = trace
	pool, err := pgxpool.NewWithConfig(h.f.ctx, config)
	must(t, err)
	defer pool.Close()
	copyStore.Pool = pool
	h.f.runtime.Staffing = app.Service{Store: &copyStore}
	profile, err := os.Create(path + ".cpu.pprof")
	must(t, err)
	must(t, pprof.StartCPUProfile(profile))
	started := time.Now()
	var mu sync.Mutex
	samples, failures := []float64{}, []string{}
	var wg sync.WaitGroup
	for w := 0; w < 20; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			v := objects[w]
			r := rec307(v)
			for j := 0; j < 5; j++ {
				start := time.Now()
				path := base307("gradeladder") + "/" + v["id"].(string) + "/records/" + r["recordId"].(string) + "/correct"
				body := map[string]any{"expectedSourceVersion": v["headSourceVersion"], "expectedRecordRevision": r["revision"], "changes": map[string]any{"name": fmt.Sprintf("诊断保存 %d %d", w, j)}}
				key := foundation.NewID()
				if mode == "candidate" {
					path = base307("gradeladder") + "/grade-reference-context"
					key = ""
					body = map[string]any{"action": "CORRECT", "objectId": v["id"], "recordId": r["recordId"], "expectedSourceVersion": v["headSourceVersion"], "expectedRecordRevision": r["revision"], "changes": map[string]any{"gradeIds": ids}, "referencePageSize": 20}
				}
				status, out, e := clients[w].response("POST", path, body, key)
				valid := out["outcome"] == "accepted"
				if mode == "candidate" {
					valid = out["totalCount"] == float64(gradeCount) && out["selectedUnavailable"] == false
				}

				mu.Lock()
				samples = append(samples, float64(time.Since(start).Nanoseconds())/1e6)
				if e != nil || status != 200 || !valid {
					failures = append(failures, fmt.Sprintf("session=%d attempt=%d status=%d outcome=%v error=%v", w, j, status, out["outcome"], e))
				}
				mu.Unlock()
				if mode == "write" && e == nil && status == 200 && valid {
					v["headSourceVersion"], r["revision"] = out["sourceVersion"], out["selectedRecordRevision"]
				}
			}
		}(w)
	}
	wg.Wait()
	pprof.StopCPUProfile()
	must(t, profile.Close())
	slices.Sort(samples)
	encoded, err := json.MarshalIndent(map[string]any{"diagnosticOnly": true, "mode": mode, "grades": gradeCount, "ladders": 20, "recordsEach": 20, "membersEach": 10, "sessions": 20, "samples": samples, "errors": failures, "p95": samples[94], "elapsedSeconds": time.Since(started).Seconds(), "queries": trace.costs}, "", "  ")
	must(t, err)
	must(t, os.WriteFile(path+".json", append(encoded, '\n'), 0600))
	if len(failures) != 0 {
		t.Fatal("diagnostic requests had semantic errors", failures)
	}
	t.Logf("instrumented write diagnostic: p95=%.3fms; no capacity pass claim", samples[94])
}

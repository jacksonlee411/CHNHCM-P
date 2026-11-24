//go:build integration

package bootstrap_test

import (
	"testing"
	"time"

	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5/pgxpool"
)

func t07OccupyPool(t *testing.T, f *fixture) []*pgxpool.Conn {
	t.Helper()
	held := []*pgxpool.Conn{}
	for i := int32(0); i < f.runtime.Pool.Config().MaxConns; i++ {
		c, e := f.runtime.Pool.Acquire(f.ctx)
		must(t, e)
		held = append(held, c)
	}
	return held
}
func TestT07PoolWaitThenAuthorizationLockUsesOriginalDeadline(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	b.client.Timeout = 15 * time.Second
	endpoint, input, _ := prepareCommitCommand(b, "T07_POOL_LOCK", "LIFECYCLE_PAIR")
	barrier, e := f.admin.Begin(f.ctx)
	must(t, e)
	defer barrier.Rollback(f.ctx)
	_, e = barrier.Exec(f.ctx, `SELECT FROM access.policy_guard WHERE instance_id=$1 AND principal_id=$2 FOR UPDATE`, f.instance, b.subject)
	must(t, e)
	held := t07OccupyPool(t, f)
	defer func() {
		for _, c := range held {
			c.Release()
		}
	}()
	key := foundation.NewID()
	type answer struct {
		status  int
		out     map[string]any
		err     error
		elapsed time.Duration
	}
	done := make(chan answer, 1)
	start := time.Now()
	go func() {
		s, o, e := b.response("POST", endpoint, input, key)
		done <- answer{s, o, e, time.Since(start)}
	}()
	// The request spends most of its existing ten-second budget awaiting a
	// connection. Its subsequent two-second lock ceiling cannot reset that clock.
	time.Sleep(8500 * time.Millisecond)
	held[0].Release()
	held = held[1:]
	r := <-done
	must(t, r.err)
	if r.status != 503 || r.out["outcome"] != nil || r.elapsed < 9500*time.Millisecond || r.elapsed > 11*time.Second {
		t.Fatal("budget restarted after pool wait", r.status, r.elapsed, r.out)
	}
	must(t, barrier.Rollback(f.ctx))
	for _, c := range held {
		c.Release()
	}
	held = nil
	if f.scalar("SELECT (SELECT count(*) FROM organization.bu_history h CROSS JOIN LATERAL unnest(h.submission_refs) r WHERE h.ids[r]=$1::uuid)+(SELECT count(*) FROM organization.bu_terminal WHERE submission_id=$1)", key) != 0 {
		t.Fatal("deadline wrote terminal")
	}
	if b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)["outcome"] != "not_applied" {
		t.Fatal("lock/connection not reusable")
	}
	t.Logf("8.5s real pool wait followed by real authorization lock: total %s, no renewed budget, original identity resolved", r.elapsed)
}
func TestT07PreviewAndResolveFiveSecondPoolBudgets(t *testing.T) {
	for _, kind := range []string{"preview", "resolve"} {
		t.Run(kind, func(t *testing.T) {
			f := fresh(t)
			b := f.login("admin", f.password)
			b.client.Timeout = 10 * time.Second
			id := b.call("POST", "/api/v1/business-units", createBody("T07_READ_BUDGET", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
			d := detail(b, id)
			held := t07OccupyPool(t, f)
			path := "/api/v1/business-units/" + id + "/preview"
			var body any = map[string]any{"action": "ADD", "anchorRecordId": d["timeline"].([]any)[0].(map[string]any)["recordId"], "expectedSourceVersion": d["headSourceVersion"], "effectiveDate": "2026-02-01", "operationType": "CHANGE", "operationReason": "NAME_ADJUSTMENT", "changes": map[string]any{"name": "新资料"}}
			method := "POST"

			key := foundation.NewID()
			if kind == "resolve" {
				path = "/api/v1/organization/submissions/" + key + "/resolve"
				method = "POST"
				body = nil
			}
			start := time.Now()
			s, out, e := b.response(method, path, body, "")
			for _, c := range held {
				c.Release()
			}
			must(t, e)
			elapsed := time.Since(start)
			if elapsed < 4500*time.Millisecond || elapsed > 6*time.Second {
				t.Fatal("five second budget", elapsed, s, out)
			}
			if kind == "preview" {
				if s != 503 || out["code"] != "M02_EXECUTION_LIMIT" || out["submissionId"] != nil {
					t.Fatal(s, out)
				}
			} else if s != 202 || out["outcome"] != "pending" {
				t.Fatal(s, out)
			}
			if f.scalar("SELECT (SELECT count(*) FROM organization.bu_history h CROSS JOIN LATERAL unnest(h.submission_refs) r WHERE h.ids[r]=$1::uuid)+(SELECT count(*) FROM organization.bu_terminal WHERE submission_id=$1)", key) != 0 {
				t.Fatal("budget exhaustion created a terminal")
			}
			t.Logf("real pool exhaustion %s completed in %s; no business write", kind, elapsed)
		})
	}
}

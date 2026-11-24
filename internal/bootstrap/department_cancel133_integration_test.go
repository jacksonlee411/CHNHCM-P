//go:build integration

package bootstrap_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"chnhcm.local/chnhcm/internal/foundation"
)

// Cancel a real HTTP request while its production statement is blocked behind
// an exclusive relation lock. The holder stays open until the canceled backend
// has released its connection and policy guard, ruling out a successful read
// merely finishing before cancellation.
func TestDepartment133HTTPCancellationReleasesSQLAndGuard(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	bu := b.call("POST", "/api/v1/business-units", createBody("CANCEL133_BU", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	id := b.call("POST", "/api/v1/departments", departmentCreateBody("CANCEL133_DEP", bu), foundation.NewID(), 201)["departmentId"].(string)
	start := map[string]any{"query": map[string]any{}, "selection": map[string]any{"mode": "allMatching", "excludedIds": []string{}}, "page": 1, "pageSize": 20}
	draft := b.call("POST", "/api/v1/departments/selection/start", start, "", 200)["summary"].(map[string]any)
	ref := map[string]any{"query": map[string]any{}, "draftId": draft["draftId"], "expectedRevision": draft["revision"], "page": 1, "pageSize": 20}
	buDetail := b.call("GET", "/api/v1/business-units/"+bu, nil, "", 200)
	buRecord := buDetail["timeline"].([]any)[0].(map[string]any)
	cases := []struct {
		name, method, path string
		body               any
	}{
		{"bu_coverage_preview", "POST", "/api/v1/business-units/" + bu + "/preview", map[string]any{"action": "DEACTIVATE", "expectedSourceVersion": buDetail["headSourceVersion"], "anchorRecordId": buRecord["recordId"], "effectiveDate": "2035-01-01", "operationType": "DEACTIVATE", "operationReason": "ORG_RESTRUCTURING"}},
		{"path", "GET", "/api/v1/departments?filterField=departmentPath&filterOperator=contains&filterValue=CANCEL", nil},
		{"parents", "POST", "/api/v1/departments/references/parents", map[string]any{"context": map[string]any{"mode": "create", "managementBuId": bu, "effectiveDate": "2026-01-01"}, "pageSize": 20}},
		{"start", "POST", "/api/v1/departments/selection/start", start},
		{"resolve", "POST", "/api/v1/departments/selection/resolve", ref},
		{"export", "POST", "/api/v1/departments/export", map[string]any{"query": map[string]any{}, "draftId": draft["draftId"], "expectedRevision": draft["revision"], "sort": "code"}},
		{"neighbor", "POST", "/api/v1/departments/navigation", map[string]any{"query": map[string]any{}, "anchorId": id, "direction": "next", "sort": "code"}},
	}
	wait := func(t *testing.T, label string, fn func() bool) {
		t.Helper()
		until := time.Now().Add(3 * time.Second)
		for !fn() {
			if time.Now().After(until) {
				t.Fatal(label)
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			holder, e := f.admin.Begin(f.ctx)
			must(t, e)
			defer holder.Rollback(f.ctx)
			_, e = holder.Exec(f.ctx, "LOCK TABLE organization.department_read_segment IN ACCESS EXCLUSIVE MODE")
			must(t, e)
			raw, e := json.Marshal(tc.body)
			must(t, e)
			ctx, cancel := context.WithCancel(f.ctx)
			defer cancel()
			var body io.Reader
			if tc.body != nil {
				body = bytes.NewReader(raw)
			}
			req, e := http.NewRequestWithContext(ctx, tc.method, f.server.URL+tc.path, body)
			must(t, e)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Origin", f.server.URL)
			req.Header.Set("X-CSRF-Token", b.csrf)
			done := make(chan error, 1)
			go func() {
				response, err := b.client.Do(req)
				if response != nil {
					response.Body.Close()
					if err == nil {
						err = fmt.Errorf("request returned status %d before cancellation", response.StatusCode)
					}
				}
				done <- err
			}()
			blocked := func() int {
				var n int
				must(t, f.admin.QueryRow(f.ctx, `SELECT count(*) FROM pg_stat_activity WHERE application_name='chnhcm' AND wait_event_type='Lock' AND cardinality(pg_blocking_pids(pid))>0`).Scan(&n))
				return n
			}
			wait(t, "production statement did not reach locked relation", func() bool { return blocked() > 0 })
			cancel()
			select {
			case e = <-done:
				if e == nil {
					t.Fatal("canceled HTTP succeeded")
				}
			case <-time.After(time.Second):
				t.Fatal("HTTP cancellation retained client")
			}
			wait(t, "canceled SQL or connection remains held", func() bool { return blocked() == 0 && f.runtime.Pool.Stat().AcquiredConns() == 0 })
			// The policy row is normally held FOR SHARE until request completion.
			_, e = f.admin.Exec(f.ctx, "SET lock_timeout='200ms';UPDATE access.policy_guard SET policy_no=policy_no;RESET lock_timeout")
			must(t, e)
			must(t, holder.Rollback(f.ctx))
			b.call("GET", "/api/v1/departments", nil, "", 200)
		})
	}
}

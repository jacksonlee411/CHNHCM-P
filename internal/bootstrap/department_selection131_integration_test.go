//go:build integration

package bootstrap_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	dep "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	"chnhcm.local/chnhcm/internal/foundation"
)

// Decorate only the real transaction's capture boundary. No fact or permission
// is mocked; the ordinary HTTP handler and its commit/response path are used.
type selection131Store struct {
	dep.Store
	captured, release chan struct{}
	once              sync.Once
	stats             dep.QueryStats
	extent            dep.QueryExtent
}
type selection131Unit struct {
	dep.Unit
	owner *selection131Store
}

func (s *selection131Store) WithinDepartment(ctx context.Context, c access.Credentials, fn func(dep.Unit) error) error {
	return s.Store.WithinDepartment(ctx, c, func(u dep.Unit) error { return fn(selection131Unit{u, s}) })
}
func (u selection131Unit) CaptureQuery(ctx context.Context, in dep.QueryRequest) (dep.QueryCapture, error) {
	v, e := u.Unit.CaptureQuery(ctx, in)
	if e != nil || in.Extent != u.owner.extent {
		return v, e
	}
	u.owner.once.Do(func() { u.owner.stats = v.Stats; close(u.owner.captured) })
	select {
	case <-u.owner.release:
		return v, nil
	case <-ctx.Done():
		return dep.QueryCapture{}, ctx.Err()
	}
}

type selection131Reply struct {
	status  int
	headers http.Header
	data    []byte
	err     error
}

func selection131HTTP(ctx context.Context, b *browser, path string, body any) selection131Reply {
	raw, _ := json.Marshal(body)
	req, e := http.NewRequestWithContext(ctx, "POST", b.f.server.URL+path, bytes.NewReader(raw))
	if e != nil {
		return selection131Reply{err: e}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", b.f.server.URL)
	req.Header.Set("X-CSRF-Token", b.csrf)
	r, e := b.client.Do(req)
	if e != nil {
		return selection131Reply{err: e}
	}
	defer r.Body.Close()
	raw, e = io.ReadAll(r.Body)
	return selection131Reply{r.StatusCode, r.Header, raw, e}
}
func await131(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(3 * time.Second):
		t.Fatal("real capture barrier not reached")
	}
}

func TestDepartmentSelection131AtomicCaptureAndExport(t *testing.T) {
	for _, operation := range []string{"start-explicit", "start-all", "resolve", "export"} {
		t.Run(operation, func(t *testing.T) {
			f := fresh(t)
			writer, reader := f.login("admin", f.password), f.login("admin", f.password)
			bu := writer.call("POST", "/api/v1/business-units", createBody("S131_BU", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
			old := writer.call("POST", "/api/v1/departments", departmentCreateBody("S131_CHILD", bu), foundation.NewID(), 201)
			choice := map[string]any{"mode": "allMatching", "excludedIds": []string{}}
			if operation == "start-explicit" {
				choice = map[string]any{"mode": "explicit", "ids": []string{old["departmentId"].(string)}}
			}
			body := map[string]any{"query": map[string]any{}, "selection": choice, "page": 1, "pageSize": 20}
			path := "/api/v1/departments/selection/start"
			if operation == "resolve" || operation == "export" {
				first := reader.call("POST", path, body, "", 200)
				body["draftId"] = first["summary"].(map[string]any)["draftId"]
				body["expectedRevision"] = first["summary"].(map[string]any)["revision"]
				delete(body, "selection")
				path = "/api/v1/departments/selection/resolve"
				if operation == "export" {
					path = "/api/v1/departments/export"
					delete(body, "page")
					delete(body, "pageSize")
				}
			}
			svc := f.runtime.Department.(dep.Service)
			barrier := &selection131Store{Store: svc.Store, captured: make(chan struct{}), release: make(chan struct{}), extent: dep.QuerySelection}
			var once sync.Once
			release := func() { once.Do(func() { close(barrier.release) }) }
			defer release()
			if strings.HasPrefix(operation, "start-") {
				barrier.extent = dep.QueryIdentities
			} else if operation == "resolve" {
				barrier.extent = dep.QueryPage
			}
			svc.Store = barrier
			f.runtime.Department = svc
			done := make(chan selection131Reply, 1)
			go func() { done <- selection131HTTP(f.ctx, reader, path, body) }()
			await131(t, barrier.captured)
			writer.call("POST", "/api/v1/departments/"+old["departmentId"].(string)+"/records/"+old["targetRecordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": old["sourceVersion"], "expectedRecordRevision": old["targetRevision"], "changes": map[string]any{"name": "捕获后更名"}}, foundation.NewID(), 200)
			writer.call("POST", "/api/v1/departments", departmentCreateBody("S131_ADDED", bu), foundation.NewID(), 201)
			release()
			reply := <-done
			must(t, reply.err)
			if reply.status != 200 || !strings.HasPrefix(operation, "start-") && (strings.Contains(string(reply.data), "捕获后更名") || strings.Contains(string(reply.data), "S131_ADDED")) {
				t.Fatalf("mixed capture response: %d %s", reply.status, reply.data)
			}
			if operation == "export" {
				if reply.headers.Get("X-Selected-Count") != "1" || !strings.Contains(reply.headers.Get("Content-Type"), "text/csv") {
					t.Fatal("export output boundary")
				}
				next := selection131HTTP(f.ctx, reader, path, body)
				if next.status != 200 || !strings.Contains(string(next.data), "捕获后更名") || strings.Contains(string(next.data), "S131_ADDED") {
					t.Fatal("fixed identity export did not use current data", next.status)
				}
			} else {
				var result map[string]any
				must(t, json.Unmarshal(reply.data, &result))
				p, s := result["page"].(map[string]any), result["summary"].(map[string]any)
				wantCount := float64(1)
				if strings.HasPrefix(operation, "start-") {
					wantCount = 2
				}
				if p["count"] != wantCount || s["selectedCount"] != float64(1) || p["queryKey"] != s["queryKey"] {
					t.Fatal("page/summary snapshots differ")
				}
			}
			t.Logf("%s actual full-set capture: %+v; concurrent name and membership change isolated", operation, barrier.stats)
		})
	}
}

func TestDepartmentSelection131CancelledHTTPHasNoCSV(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	bu := b.call("POST", "/api/v1/business-units", createBody("CANCEL131", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	b.call("POST", "/api/v1/departments", departmentCreateBody("CANCEL131_DEP", bu), foundation.NewID(), 201)
	body := map[string]any{"query": map[string]any{}, "selection": map[string]any{"mode": "allMatching", "excludedIds": []string{}}, "page": 1, "pageSize": 20}
	v := b.call("POST", "/api/v1/departments/selection/start", body, "", 200)
	body["draftId"] = v["summary"].(map[string]any)["draftId"]
	body["expectedRevision"] = v["summary"].(map[string]any)["revision"]
	delete(body, "selection")
	delete(body, "page")
	delete(body, "pageSize")
	svc := f.runtime.Department.(dep.Service)
	barrier := &selection131Store{Store: svc.Store, captured: make(chan struct{}), release: make(chan struct{}), extent: dep.QuerySelection}
	svc.Store = barrier
	f.runtime.Department = svc
	defer close(barrier.release)
	ctx, cancel := context.WithCancel(f.ctx)
	defer cancel()
	done := make(chan selection131Reply, 1)
	go func() { done <- selection131HTTP(ctx, b, "/api/v1/departments/export", body) }()
	await131(t, barrier.captured)
	cancel()
	reply := <-done
	if reply.err == nil || len(reply.data) > 0 || reply.headers.Get("Content-Disposition") != "" {
		t.Fatal("cancel produced CSV")
	}
	// The actual policy guard must be released after cancellation.
	tx, e := f.admin.Begin(f.ctx)
	must(t, e)
	defer tx.Rollback(f.ctx)
	_, e = tx.Exec(f.ctx, "SET LOCAL lock_timeout='500ms'")
	must(t, e)
	_, e = tx.Exec(f.ctx, "SELECT policy_no FROM access.policy_guard WHERE principal_id=$1 FOR UPDATE", b.subject)
	must(t, e)
}

func TestDepartmentSelection131HTTPBudgetHasNoCSV(t *testing.T) {
	testSelection131HTTPBudget(t, "export")
}

func TestDepartmentSelection131ResolveBudgetHasNoPartialPage(t *testing.T) {
	testSelection131HTTPBudget(t, "selection/resolve")
}

func testSelection131HTTPBudget(t *testing.T, endpointPath string) {
	f := fresh(t)
	b := f.login("admin", f.password)
	bu := b.call("POST", "/api/v1/business-units", createBody("BUDGET131", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	b.call("POST", "/api/v1/departments", departmentCreateBody("BUDGET131_DEP", bu), foundation.NewID(), 201)
	body := map[string]any{"query": map[string]any{}, "selection": map[string]any{"mode": "allMatching", "excludedIds": []string{}}, "page": 1, "pageSize": 20}
	v := b.call("POST", "/api/v1/departments/selection/start", body, "", 200)
	body["draftId"] = v["summary"].(map[string]any)["draftId"]
	body["expectedRevision"] = v["summary"].(map[string]any)["revision"]
	delete(body, "selection")
	if endpointPath == "export" {
		delete(body, "page")
		delete(body, "pageSize")
	}
	svc := f.runtime.Department.(dep.Service)
	barrier := &selection131Store{Store: svc.Store, captured: make(chan struct{}), release: make(chan struct{}), extent: dep.QuerySelection}
	if endpointPath == "selection/resolve" {
		barrier.extent = dep.QueryPage
	}
	svc.Store = barrier
	f.runtime.Department = svc
	defer close(barrier.release)
	// Observe the real HTTP handler directly, so the full-capture 10-second
	// budget is not raced by the fixture client's identical transport timeout.
	raw, e := json.Marshal(body)
	must(t, e)
	req := httptest.NewRequest("POST", f.server.URL+"/api/v1/departments/"+endpointPath, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", f.server.URL)
	req.Header.Set("X-CSRF-Token", b.csrf)
	endpoint, e := url.Parse(f.server.URL)
	must(t, e)
	for _, cookie := range b.client.Jar.Cookies(endpoint) {
		req.AddCookie(cookie)
	}
	recorder := httptest.NewRecorder()
	done := make(chan struct{})
	started := time.Now()
	go func() { f.server.Config.Handler.ServeHTTP(recorder, req); close(done) }()
	await131(t, barrier.captured)
	<-done
	elapsed := time.Since(started)
	if elapsed < 9500*time.Millisecond || elapsed > 11*time.Second {
		t.Fatal("full-capture HTTP budget", endpointPath, elapsed)
	}
	if recorder.Code != 503 || recorder.Header().Get("Content-Disposition") != "" || strings.Contains(recorder.Body.String(), "编码,名称") {
		t.Fatal("budget emitted partial file", recorder.Code)
	}
	var rejection map[string]any
	must(t, json.Unmarshal(recorder.Body.Bytes(), &rejection))
	if rejection["page"] != nil || rejection["summary"] != nil {
		t.Fatal("budget emitted partial page", endpointPath)
	}
	t.Logf("%s real handler budget %s; 503 with no attachment, CSV, page or summary", endpointPath, elapsed)
}

type selection131ValidatedStore struct {
	dep.Store
	validated, release chan struct{}
}

func (s selection131ValidatedStore) WithinDepartment(ctx context.Context, c access.Credentials, fn func(dep.Unit) error) error {
	return s.Store.WithinDepartment(ctx, c, func(u dep.Unit) error {
		if e := fn(u); e != nil {
			return e
		}
		close(s.validated)
		select {
		case <-s.release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
}

func TestDepartmentSelection131ExportPolicyBoundary(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	bu := b.call("POST", "/api/v1/business-units", createBody("POLICY131", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	for _, code := range []string{"POLICY131_A", "POLICY131_B"} {
		b.call("POST", "/api/v1/departments", departmentCreateBody(code, bu), foundation.NewID(), 201)
	}
	body := map[string]any{"query": map[string]any{}, "selection": map[string]any{"mode": "allMatching", "excludedIds": []string{}}, "page": 1, "pageSize": 20}
	v := b.call("POST", "/api/v1/departments/selection/start", body, "", 200)
	body["draftId"] = v["summary"].(map[string]any)["draftId"]
	body["expectedRevision"] = v["summary"].(map[string]any)["revision"]
	delete(body, "selection")
	delete(body, "page")
	delete(body, "pageSize")
	service := f.runtime.Department.(dep.Service)
	original := service.Store
	validated, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()
	service.Store = selection131ValidatedStore{original, validated, release}
	f.runtime.Department = service
	done := make(chan selection131Reply, 1)
	go func() { done <- selection131HTTP(f.ctx, b, "/api/v1/departments/export", body) }()
	await131(t, validated)
	// This is after complete set, export rights, field projection AND CSV
	// construction. A real policy writer still cannot commit in that window.
	revoked := make(chan error, 1)
	go func() {
		tx, e := f.admin.Begin(f.ctx)
		if e != nil {
			revoked <- e
			return
		}
		defer tx.Rollback(f.ctx)
		_, e = tx.Exec(f.ctx, "SET LOCAL application_name='q131-t02-export-revoke'")
		if e == nil {
			_, e = tx.Exec(f.ctx, "SELECT policy_no FROM access.policy_guard WHERE principal_id=$1 FOR UPDATE", b.subject)
		}
		if e == nil {
			_, e = tx.Exec(f.ctx, "DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='department.export'", b.subject)
		}
		if e == nil {
			_, e = tx.Exec(f.ctx, "UPDATE access.policy_guard SET policy_no=policy_no+1 WHERE principal_id=$1", b.subject)
		}
		if e == nil {
			e = tx.Commit(f.ctx)
		}
		revoked <- e
	}()
	until := time.Now().Add(3 * time.Second)
	for {
		if f.scalar("SELECT count(*) FROM pg_stat_activity WHERE application_name='q131-t02-export-revoke' AND wait_event_type='Lock'") == 1 {
			break
		}
		if time.Now().After(until) {
			t.Fatal("policy writer never waited on actual guard")
		}
		runtime.Gosched()
	}
	unblock()
	reply := <-done
	must(t, reply.err)
	must(t, <-revoked)
	if reply.status != 200 || reply.headers.Get("X-Selected-Count") != "2" {
		t.Fatal("authorized capture failed", reply.status)
	}
	service.Store = original
	f.runtime.Department = service
	next := selection131HTTP(f.ctx, b, "/api/v1/departments/export", body)
	must(t, next.err)
	if next.status != 403 || next.headers.Get("Content-Disposition") != "" || strings.Contains(string(next.data), "编码,名称") {
		t.Fatal("revocation filtered or emitted partial CSV", next.status)
	}
	t.Log("policy writer observed blocked after complete validation and CSV construction; prior read completed atomically, subsequent export rejected with no file")
}

func TestDepartmentSelection131PartialRevocationNeverExportsRemainder(t *testing.T) {
	for _, action := range []string{"department.read", "department.export"} {
		t.Run(action, func(t *testing.T) {
			f := fresh(t)
			b := f.login("admin", f.password)
			bu := b.call("POST", "/api/v1/business-units", createBody("PARTIAL131", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
			kept := b.call("POST", "/api/v1/departments", departmentCreateBody("PARTIAL131_A", bu), foundation.NewID(), 201)["departmentId"].(string)
			b.call("POST", "/api/v1/departments", departmentCreateBody("PARTIAL131_B", bu), foundation.NewID(), 201)
			body := map[string]any{"query": map[string]any{}, "selection": map[string]any{"mode": "allMatching", "excludedIds": []string{}}, "page": 1, "pageSize": 20}
			first := b.call("POST", "/api/v1/departments/selection/start", body, "", 200)
			body["draftId"] = first["summary"].(map[string]any)["draftId"]
			body["expectedRevision"] = first["summary"].(map[string]any)["revision"]
			delete(body, "selection")
			delete(body, "page")
			delete(body, "pageSize")
			tx, e := f.admin.Begin(f.ctx)
			must(t, e)
			defer tx.Rollback(f.ctx)
			_, e = tx.Exec(f.ctx, "SELECT policy_no FROM access.policy_guard WHERE principal_id=$1 FOR UPDATE", b.subject)
			must(t, e)
			_, e = tx.Exec(f.ctx, "UPDATE access.subject_grant SET scope_kind='department',scope_id=$3 WHERE principal_id=$1 AND action=$2", b.subject, action, kept)
			must(t, e)
			_, e = tx.Exec(f.ctx, "UPDATE access.policy_guard SET policy_no=policy_no+1 WHERE principal_id=$1", b.subject)
			must(t, e)
			must(t, tx.Commit(f.ctx))
			reply := selection131HTTP(f.ctx, b, "/api/v1/departments/export", body)
			must(t, reply.err)
			want := 403

			if reply.status != want || reply.headers.Get("Content-Disposition") != "" || reply.headers.Get("X-Selected-Count") != "" || !strings.Contains(reply.headers.Get("Content-Type"), "application/json") || bytes.HasPrefix(reply.data, []byte{0xef, 0xbb, 0xbf}) {
				t.Fatal("partial revocation emitted remainder", action, reply.status)
			}
		})
	}
}

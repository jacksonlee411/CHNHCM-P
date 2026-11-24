//go:build integration

package bootstrap_test

import (
	"context"
	"net"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"chnhcm.local/chnhcm/internal/contexts/organization/application"
	department "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	orgdb "chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5/pgxpool"
)

// A real read transaction still commits session activity. Losing its COMMIT
// reply must not manufacture a business submission or expose provisional rows.
func TestReadCommitCommunicationLossDoesNotSuggestSubmissionRecovery(t *testing.T) {
	f := fresh(t)
	admin := f.login("admin", f.password)
	reader := f.login("reader", f.readPassword)
	bu := admin.call("POST", "/api/v1/business-units", createBody("READ_ACK_BU", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	admin.call("POST", "/api/v1/departments", departmentCreateBody("READ_ACK_PARENT", bu), foundation.NewID(), 201)
	reference := map[string]any{"context": map[string]any{"mode": "create", "managementBuId": bu, "effectiveDate": "2026-01-01"}, "page": 1, "pageSize": 20}
	parentReference := map[string]any{"context": reference["context"], "pageSize": 20}
	cases := []struct {
		name, method, path string
		body               any
		buService, reader  bool
		count              float64
	}{
		{"bu-list", "GET", "/api/v1/business-units", nil, true, false, 1},
		{"department-list", "GET", "/api/v1/departments", nil, false, false, 1},
		{"parent-admin", "POST", "/api/v1/departments/references/parents", parentReference, false, false, 1},
		{"parent-reader-negative", "POST", "/api/v1/departments/references/parents", parentReference, false, true, 0},
		{"bu-reference-admin", "POST", "/api/v1/departments/references/business-units", reference, false, false, 1},
		{"bu-reference-reader-negative", "POST", "/api/v1/departments/references/business-units", reference, false, true, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := admin
			if tc.reader {
				b = reader
			}
			beforeRead := b.call(tc.method, tc.path, tc.body, "", 200)
			if float64(len(beforeRead["items"].([]any))) != tc.count {
				t.Fatal("independent nonempty/permission-negative fixture not reached", beforeRead["count"])
			}
			orgService := f.runtime.Organization.(application.Service)
			depService := f.runtime.Department.(department.Service)
			orgOriginal, depOriginal := orgService.Store, depService.Store
			base := depOriginal.(*orgdb.Store)
			if tc.buService {
				base = orgOriginal.(*orgdb.Store)
			}
			copyStore := *base
			var armed, sent, dropped atomic.Bool
			config := base.Pool.Config().Copy()
			config.MinConns = 0
			config.MaxConns = 1
			config.ConnConfig.DialFunc = func(ctx context.Context, network, address string) (net.Conn, error) {
				c, e := (&net.Dialer{}).DialContext(ctx, network, address)
				if e != nil {
					return nil, e
				}
				return &t07CommitWire{Conn: c, armed: &armed, sent: &sent, dropped: &dropped}, nil
			}
			pool, e := pgxpool.NewWithConfig(f.ctx, config)
			must(t, e)
			defer pool.Close()
			must(t, pool.Ping(f.ctx))
			copyStore.Pool = pool
			restore := func() {
				orgService.Store = orgOriginal
				depService.Store = depOriginal
				f.runtime.Organization = orgService
				f.runtime.Department = depService
			}
			defer restore()
			if tc.buService {
				orgService.Store = &copyStore
				f.runtime.Organization = orgService
			} else {
				depService.Store = &copyStore
				f.runtime.Department = depService
			}
			beforeBU, beforeDepartment := businessSnapshot(t, f), departmentFacts(f)
			armed.Store(true)
			status, out, e := b.response(tc.method, tc.path, tc.body, "")
			must(t, e)
			if !sent.Load() || !dropped.Load() {
				t.Fatal("real read COMMIT reply was not interrupted")
			}
			t.Logf("R37_READ_COMMIT_BOUNDARY status=%d code=%v sent=%t dropped=%t", status, out["code"], sent.Load(), dropped.Load())
			if status != 503 || out["code"] != "M02_DEPENDENCY_UNAVAILABLE" {
				t.Fatal("read lost-ack response uses business submission recovery", status, out["code"])
			}
			for _, key := range []string{"items", "count", "selectedReference", "outcome", "submissionId"} {
				if out[key] != nil {
					t.Fatal("failed read exposed a provisional result", key)
				}
			}
			message, _ := out["message"].(string)
			if strings.Contains(message, "提交") || strings.Contains(message, "查回") {
				t.Fatal("query error asks user to recover a nonexistent submission", message)
			}
			restore()
			if !reflect.DeepEqual(beforeBU, businessSnapshot(t, f)) || !reflect.DeepEqual(beforeDepartment, departmentFacts(f)) {
				t.Fatal("read failure changed business facts or terminal history")
			}
			after := b.call(tc.method, tc.path, tc.body, "", 200)
			if float64(len(after["items"].([]any))) != tc.count {
				t.Fatal("restored connection did not retain original read/permission result", after["count"])
			}
		})
	}
}

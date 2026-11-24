//go:build integration

package bootstrap_test

import (
	"context"
	"errors"
	"net"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	dep "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	orgdb "chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5/pgxpool"
)

func departmentCreateBody(code, bu string) map[string]any {
	return map[string]any{"code": code, "name": "研发部门", "managementBuId": bu, "effectiveDate": "2026-01-01", "operationType": "CREATE", "operationReason": "NEW_ORGANIZATION"}
}
func departmentFacts(f *fixture) map[string]int {
	out := map[string]int{}
	for _, table := range []string{"department", "department_hierarchy_relation", "department_terminal"} {
		out[table] = f.scalar("SELECT count(*) FROM organization." + table)
	}
	for _, v := range f.history().Departments {
		out["sources"]++
		out["members"] += len(v.State.Timeline.Records)
		out["changes"] += len(v.Operation.History.Items)
		out["paths"] += len(v.Evidence.Segments)
		if v.Operation.EventID != "" {
			out["events"]++
		}
		if v.Operation.RelationEventID != "" {
			out["relation_events"]++
		}
	}
	return out
}

func TestDepartmentRealCommitCommunicationLoss(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	bu := b.call("POST", "/api/v1/business-units", createBody("WIRE_DEPARTMENT", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	for _, mode := range []string{"create_drop_forward", "create_drop_reply", "pair_drop_reply", "no_change_drop_reply", "rejected_drop_reply", "not_applied_drop_reply"} {
		t.Run(mode, func(t *testing.T) {
			body := departmentCreateBody("W"+foundation.NewID()[:8], bu)
			path := "/api/v1/departments"
			if mode == "pair_drop_reply" || mode == "no_change_drop_reply" || mode == "rejected_drop_reply" {
				first := b.call("POST", path, body, foundation.NewID(), 201)
				path += "/" + first["departmentId"].(string)
				body = map[string]any{"expectedSourceVersion": first["sourceVersion"], "expectedRecordRevision": first["targetRevision"], "changes": map[string]any{"name": "研发部门"}}
				if mode == "pair_drop_reply" {
					body = map[string]any{"expectedSourceVersion": first["sourceVersion"], "anchorRecordId": first["targetRecordId"], "deactivationDate": "2026-06-01", "reactivationDate": "2026-07-01", "deactivationReason": "ORG_RESTRUCTURING", "reactivationReason": "BUSINESS_RESUMPTION"}
					path += "/lifecycle-pairs"
				} else {
					path += "/records/" + first["targetRecordId"].(string) + "/correct"
					if mode == "rejected_drop_reply" {
						body["changes"] = map[string]any{"name": ""}
					}
				}
			}
			key := foundation.NewID()
			header := key
			if mode == "not_applied_drop_reply" {
				path = "/api/v1/departments/submissions/" + key + "/resolve"
				body = nil
				header = ""
			}
			svc := f.runtime.Department.(dep.Service)
			original := svc.Store
			copyStore := *original.(*orgdb.Store)
			var armed, sent, dropped atomic.Bool
			config := copyStore.Pool.Config().Copy()
			config.MinConns = 0
			config.MaxConns = 1
			config.ConnConfig.DialFunc = func(ctx context.Context, network, address string) (net.Conn, error) {
				c, e := (&net.Dialer{}).DialContext(ctx, network, address)
				if e != nil {
					return nil, e
				}
				return &t07CommitWire{Conn: c, armed: &armed, sent: &sent, dropped: &dropped, afterForward: mode == "create_drop_forward"}, nil
			}
			pool, e := pgxpool.NewWithConfig(f.ctx, config)
			must(t, e)
			defer pool.Close()
			must(t, pool.Ping(f.ctx))
			copyStore.Pool = pool
			svc.Store = &copyStore
			f.runtime.Department = svc
			defer func() { svc.Store = original; f.runtime.Department = svc }()
			before := departmentFacts(f)
			armed.Store(true)
			status, out, e := b.response("POST", path, body, header)
			must(t, e)
			if !sent.Load() || !dropped.Load() || status != 503 || out["code"] != "M02_RESULT_PENDING" || out["outcome"] != nil || out["items"] != nil {
				t.Fatal("provisional result or no physical COMMIT interruption", status, out)
			}
			svc.Store = original
			f.runtime.Department = svc
			resolved := b.call("POST", "/api/v1/departments/submissions/"+key+"/resolve", nil, "", 200)
			expected := "accepted"
			switch mode {
			case "no_change_drop_reply":
				expected = "no_change"
			case "rejected_drop_reply":
				expected = "rejected"
			case "not_applied_drop_reply":
				expected = "not_applied"
			}
			if resolved["outcome"] != expected && !(mode == "create_drop_forward" && resolved["outcome"] == "not_applied") {
				t.Fatal(mode, resolved)
			}
			if resolved["outcome"] != "accepted" {
				for table, n := range departmentFacts(f) {
					if table != "department_terminal" && n != before[table] {
						t.Fatal("partial business fact", table)
					}
				}
			}
			if resolved["outcome"] == "accepted" {
				want := 201
				if mode == "pair_drop_reply" {
					want = 200
				}
				replay := b.call("POST", path, body, key, want)
				if replay["sourceVersion"] != resolved["sourceVersion"] || replay["replayed"] != true {
					t.Fatal("different original result", replay, resolved)
				}
				if mode == "pair_drop_reply" {
					for _, item := range replay["items"].([]any) {
						if item.(map[string]any)["afterRecord"] == nil {
							t.Fatal("pair result missing exact record")
						}
					}
				}
			}
			if mode == "rejected_drop_reply" {
				b.call("POST", path, body, key, 400)
			}
			if mode == "no_change_drop_reply" {
				replay := b.call("POST", path, body, key, 200)
				if replay["currentStatus"] != "active" || replay["evaluatedBusinessDate"] != resolved["evaluatedBusinessDate"] {
					t.Fatal("no-change status/day lost", replay)
				}
			}
			if f.scalar("SELECT (SELECT count(*) FROM organization.department_history h CROSS JOIN LATERAL unnest(h.submission_refs) r WHERE h.ids[r]=$1::uuid)+(SELECT count(*) FROM organization.department_terminal WHERE submission_id=$1)", key) != 1 {
				t.Fatal("terminal not unique")
			}
			t.Logf("actual COMMIT TCP loss, original identity recovered: %s", resolved["outcome"])
		})
	}
}

type departmentFaultStore struct {
	dep.Store
	after           func(context.Context, dep.Unit) error
	snapshotFailure bool
}
type departmentFaultUnit struct {
	dep.Unit
	snapshotFailure bool
}
type departmentFaultGuard struct{ access.Guard }

func (g departmentFaultGuard) OperatorSnapshot(context.Context) (access.OperatorSnapshot, error) {
	return access.OperatorSnapshot{}, errors.New("controlled X03 snapshot failure")
}
func (u departmentFaultUnit) Access() access.Guard {
	if u.snapshotFailure {
		return departmentFaultGuard{u.Unit.Access()}
	}
	return u.Unit.Access()
}
func (s departmentFaultStore) WithinDepartmentWrite(ctx context.Context, c access.Credentials, fn func(dep.Unit) error) error {
	return s.Store.WithinDepartmentWrite(ctx, c, func(u dep.Unit) error {
		if e := fn(departmentFaultUnit{u, s.snapshotFailure}); e != nil {
			return e
		}
		if s.after != nil {
			return s.after(ctx, u)
		}
		return nil
	})
}
func TestDepartmentSnapshotAndUncommittedCancellation(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	bu := b.call("POST", "/api/v1/business-units", createBody("CANCEL_DEP", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	first := b.call("POST", "/api/v1/departments", departmentCreateBody("D", bu), foundation.NewID(), 201)
	u, e := url.Parse(f.server.URL)
	must(t, e)
	creds := access.Credentials{Token: b.client.Jar.Cookies(u)[0].Value, CSRF: b.csrf, Write: true}
	svc := f.runtime.Department.(dep.Service)
	original := svc.Store
	for _, mode := range []string{"snapshot", "accepted", "no_change", "rejected", "not_applied", "deadline"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(f.ctx)
			defer cancel()
			key := foundation.NewID()
			before := departmentFacts(f)
			svc.Store = departmentFaultStore{Store: original, snapshotFailure: mode == "snapshot", after: func(ctx context.Context, u dep.Unit) error {
				saved, e := u.FindResult(ctx, key)
				if e != nil {
					return e
				}
				if saved == nil {
					t.Fatal("missing provisional terminal")
				}
				want := mode
				if mode == "deadline" {
					want = "accepted"
				}
				if saved.Result.Outcome != want {
					t.Fatal("unexpected provisional outcome", saved.Result.Outcome)
				}
				cancel()
				if mode == "deadline" {
					return context.DeadlineExceeded
				}
				return ctx.Err()
			}}
			var result org.DepartmentResult
			if mode == "not_applied" {
				result, e = svc.Resolve(ctx, creds, key)
			} else {
				name := "更正名称"
				if mode == "no_change" {
					name = "研发部门"
				}
				if mode == "rejected" {
					name = ""
				}
				result, e = svc.Execute(ctx, creds, org.DepartmentCommand{Action: "CORRECT", DepartmentID: first["departmentId"].(string), RecordID: first["targetRecordId"].(string), SubmissionID: key, Digest: make([]byte, 32), Correct: org.DepartmentCorrectInput{ExpectedSourceVersion: first["sourceVersion"].(string), ExpectedRecordRevision: first["targetRevision"].(string), Changes: org.DepartmentChangesInput{"name": &name}}})
			}
			if e == nil || !reflect.DeepEqual(result, org.DepartmentResult{}) {
				t.Fatal("uncommitted result escaped", result, e)
			}
			if !reflect.DeepEqual(before, departmentFacts(f)) {
				t.Fatal("failed commit/snapshot changed facts")
			}
			svc.Store = original
			r, e := svc.Resolve(f.ctx, creds, key)
			must(t, e)
			if r.Outcome != "not_applied" {
				t.Fatal("lock/connection not reusable", r)
			}
		})
	}
}

func TestDepartmentReadAndResolvePoolBudgets(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	b.client.Timeout = 8 * time.Second
	for _, kind := range []string{"changes", "resolve"} {
		t.Run(kind, func(t *testing.T) {
			key := foundation.NewID()
			path := "/api/v1/departments/" + foundation.NewID() + "/changes"
			method := "GET"
			if kind == "resolve" {
				path = "/api/v1/departments/submissions/" + key + "/resolve"
				method = "POST"
			}
			b.client.CloseIdleConnections()
			held := t07OccupyPool(t, f)
			start := time.Now()
			status, out, e := b.response(method, path, nil, "")
			for _, conn := range held {
				conn.Release()
			}
			must(t, e)
			if time.Since(start) < 4500*time.Millisecond || time.Since(start) > 6500*time.Millisecond {
				t.Fatal("read budget", time.Since(start), status, out)
			}
			if kind == "resolve" {
				if status != 202 || out["outcome"] != "pending" {
					t.Fatal(status, out)
				}
			} else if status != 503 || out["code"] != "M02_EXECUTION_LIMIT" {
				t.Fatal(status, out)
			}
			if f.scalar("SELECT (SELECT count(*) FROM organization.department_history h CROSS JOIN LATERAL unnest(h.submission_refs) r WHERE h.ids[r]=$1::uuid)+(SELECT count(*) FROM organization.department_terminal WHERE submission_id=$1)", key) != 0 {
				t.Fatal("timeout terminal")
			}
		})
	}
}

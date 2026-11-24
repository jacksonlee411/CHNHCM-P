//go:build integration && query131

package bootstrap_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

func query131WaitLock(t *testing.T, f *fixture, name string) {
	t.Helper()
	deadline := time.Now().Add(6 * time.Second)
	for {
		var n int
		must(t, f.admin.QueryRow(f.ctx, `SELECT count(*) FROM pg_stat_activity WHERE application_name=$1 AND wait_event_type='Lock' AND cardinality(pg_blocking_pids(pid))>0`, name).Scan(&n))
		if n > 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("server lock barrier not reached", name)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestDepartmentQuery131DatedClosureAndFaults(t *testing.T) {
	f, admin, q := query131Fixture(t)
	a, b := query131BU(t, admin, "DATED_A"), query131BU(t, admin, "DATED_B")
	parent := query131Create(t, admin, "DATED_PARENT", "parent-before", a, "2020-01-01")
	pid := parent["departmentId"].(string)
	body := departmentCreateBody("DATED_LEAF", a)
	body["effectiveDate"], body["name"], body["parentDepartmentId"] = "2020-01-01", "dated-leaf", pid
	leaf := admin.call("POST", "/api/v1/departments", body, foundation.NewID(), 201)
	id := leaf["departmentId"].(string)
	admin.call("POST", "/api/v1/departments/"+id+"/deactivate", map[string]any{"expectedSourceVersion": leaf["sourceVersion"], "anchorRecordId": leaf["targetRecordId"], "effectiveDate": "2026-06-01", "operationType": "DEACTIVATE", "operationReason": "ORG_RESTRUCTURING"}, foundation.NewID(), 201)
	head := admin.call("POST", "/api/v1/departments/"+pid+"/records", map[string]any{"expectedSourceVersion": parent["sourceVersion"], "anchorRecordId": parent["targetRecordId"], "effectiveDate": "2026-07-01", "operationType": "CHANGE", "operationReason": "ORGANIZATION_REASSIGNMENT", "changes": map[string]any{"managementBuId": b, "name": "parent-after"}}, foundation.NewID(), 201)
	query := org.DepartmentListInput{Q: "dated-leaf", Status: "all", Sort: "code"}
	v, _, e := query131PlanRead(t, f, admin, q, query)
	must(t, e)
	if v.Count != 1 || v.Items[0]["pathDay"] != "2026-05-31" || v.Items[0]["departmentPath"] != "parent-before / dated-leaf" {
		t.Fatal("inactive path used current day", v)
	}
	reader := f.login("reader", f.readPassword)
	_, e = f.admin.Exec(f.ctx, `DELETE FROM access.subject_grant WHERE instance_id=$1 AND principal_id=$2 AND action='department.reference.read'`, f.instance, reader.subject)
	must(t, e)
	query131Grant(t, f, reader, "bu", a, "department.reference.read", []string{"name"})
	v, _, e = query131PlanRead(t, f, reader, q, query)
	must(t, e)
	if v.Items[0]["departmentPath"] != nil {
		t.Fatal("historical A relationship granted current B reference rights", v)
	}
	query131Grant(t, f, reader, "bu", b, "department.reference.read", []string{"name"})
	v, _, e = query131PlanRead(t, f, reader, q, query)
	must(t, e)
	if v.Items[0]["departmentPath"] != "parent-before / dated-leaf" {
		t.Fatal("current B reference did not authorize historical path", v)
	}
	admin.call("POST", "/api/v1/departments/"+pid+"/records", map[string]any{"expectedSourceVersion": head["sourceVersion"], "anchorRecordId": head["targetRecordId"], "effectiveDate": "2027-01-01", "operationType": "CHANGE", "operationReason": "ORGANIZATION_REASSIGNMENT", "changes": map[string]any{"name": "parent-future"}}, foundation.NewID(), 201)
	body = departmentCreateBody("DATED_FUTURE", b)
	body["effectiveDate"], body["name"], body["parentDepartmentId"] = "2027-01-01", "future-leaf", pid
	admin.call("POST", "/api/v1/departments", body, foundation.NewID(), 201)
	query.Q = "future-leaf"
	v, _, e = query131PlanRead(t, f, admin, q, query)
	must(t, e)
	if v.Items[0]["effectiveDate"] != "2027-01-01" || v.Items[0]["status"] != "not_effective" || v.Items[0]["pathDay"] != "2027-01-01" || v.Items[0]["departmentPath"] != "parent-future / future-leaf" {
		t.Fatal("future path or first date changed", v)
	}
	// Required closure failures must return no partial count or page.
	// Unrelated query-column auditing no longer belongs to this read.
	for _, tc := range []struct{ name, sql string }{
		{"missing_ancestor", `DELETE FROM organization.department_read_segment WHERE object_id=$1`},
		{"missing_bu", `DELETE FROM organization.bu_read_segment WHERE object_id=$1`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx, g, in := query131Guard(t, f, admin, q)
			defer tx.Rollback(context.Background())
			in.Query = query
			object := pid
			if tc.name == "missing_bu" {
				object = b
			}
			_, e := tx.Exec(f.ctx, tc.sql, object)
			must(t, e)
			got, _, e := history.CapturePlanned131(f.ctx, tx, g, in)
			var fault *foundation.Fault
			if !errors.As(e, &fault) || fault.Status != 503 || got.Count != 0 || len(got.Items) != 0 {
				t.Fatal("closure fault escaped", tc.name, got, e)
			}
		})
	}
	t.Log("current authorization day differs from inactive/future path day; missing required BU/ancestor safely rejected")
}

func TestDepartmentQuery131FullCaptureConcurrency(t *testing.T) {
	f, admin, q := query131Fixture(t)
	bu := query131BU(t, admin, "CAPTURE_BU")
	oldParent := query131Create(t, admin, "CAPTURE_OLD", "parent-old", bu, "2020-01-01")
	newParent := query131Create(t, admin, "CAPTURE_NEW", "parent-new", bu, "2020-01-01")
	body := departmentCreateBody("CAPTURE_LEAF", bu)
	body["name"], body["effectiveDate"], body["parentDepartmentId"] = "capture-leaf", "2020-01-01", oldParent["departmentId"]
	leaf := admin.call("POST", "/api/v1/departments", body, foundation.NewID(), 201)
	reader := f.login("admin", f.password)
	var oldDictionaries []int32
	rows, e := f.admin.Query(f.ctx, `SELECT DISTINCT dictionary_id FROM organization.department_read_segment WHERE object_id::text=ANY($1::text[]) AND span @> $2::date`, []string{oldParent["departmentId"].(string), leaf["departmentId"].(string)}, q.Day)
	must(t, e)
	for rows.Next() {
		var id int32
		must(t, rows.Scan(&id))
		oldDictionaries = append(oldDictionaries, id)
	}
	must(t, rows.Err())
	rows.Close()
	tx, g, in := query131Guard(t, f, reader, q)
	defer tx.Rollback(context.Background())
	in.Query.Q = "capture-leaf"
	_, e = tx.Exec(f.ctx, `SELECT set_config('application_name','q131-full-capture',true)`)
	must(t, e)
	gate, e := f.admin.Begin(f.ctx)
	must(t, e)
	defer gate.Rollback(context.Background())
	_, e = gate.Exec(f.ctx, `SELECT pg_advisory_xact_lock(1310092)`)
	must(t, e)
	sql := strings.Replace(history.Query131PlannedSQL, "FROM organization.department d\n", "FROM organization.department d CROSS JOIN (SELECT pg_advisory_xact_lock(1310092)) gate\n", 1)
	type reply struct {
		c   history.CapturedPlan131
		err error
	}
	done := make(chan reply, 1)
	ctx, cancel := context.WithTimeout(f.ctx, 15*time.Second)
	defer cancel()
	go func() { var r reply; r.err = r.c.Scan(tx.QueryRow(ctx, sql, in.Args()...)); done <- r }()
	query131WaitLock(t, f, "q131-full-capture")
	admin.call("POST", "/api/v1/departments/"+oldParent["departmentId"].(string)+"/records/"+oldParent["targetRecordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": oldParent["sourceVersion"], "expectedRecordRevision": oldParent["targetRevision"], "changes": map[string]any{"name": "parent-replaced"}}, foundation.NewID(), 200)
	admin.call("POST", "/api/v1/departments/"+leaf["departmentId"].(string)+"/records/"+leaf["targetRecordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": leaf["sourceVersion"], "expectedRecordRevision": leaf["targetRevision"], "changes": map[string]any{"parentDepartmentId": newParent["departmentId"]}}, foundation.NewID(), 200)
	query131Create(t, admin, "CAPTURE_NEWCOMER", "capture-leaf newcomer", bu, "2026-01-01")
	must(t, gate.Commit(f.ctx))
	captured := <-done
	must(t, captured.err)
	deleted := 0
	for _, id := range oldDictionaries {
		tag, e := f.admin.Exec(f.ctx, `DELETE FROM organization.department_dictionary WHERE id=$1`, id)
		must(t, e)
		deleted += int(tag.RowsAffected())
	}

	old, _, e := history.ProjectCaptured131(f.ctx, g, in, captured.c)
	must(t, e)
	if deleted != 2 || old.Count != 1 || old.Items[0]["departmentPath"] != "parent-old / capture-leaf" {
		t.Fatal("captured ancestor bytes mixed commits or failed after reclaim", deleted, old)
	}
	must(t, g.RecordSessionActivity(f.ctx))
	must(t, tx.Commit(f.ctx))
	next, _, e := query131PlanRead(t, f, reader, q, in.Query)
	must(t, e)
	if next.Count != 2 {
		t.Fatal("next capture missed addition", next)
	}
	found := false
	for _, item := range next.Items {
		if item["id"] == leaf["departmentId"] {
			found = item["departmentPath"] == "parent-new / capture-leaf"
		}
	}
	if !found {
		t.Fatal("next capture retained old parent", next)
	}
	t.Log("barrier proved count/page/parent names/current scope share one statement; two obsolete dictionaries reclaimed before Go projection")
}

func TestDepartmentQuery131ConcurrentBUTransfer(t *testing.T) {
	f, admin, q := query131Fixture(t)
	a, b := query131BU(t, admin, "TRANSFER_A"), query131BU(t, admin, "TRANSFER_B")
	object := query131Create(t, admin, "TRANSFER_DEPT", "transfer-target", a, "2026-01-01")
	reader := f.login("reader", f.readPassword)
	_, e := f.admin.Exec(f.ctx, `DELETE FROM access.subject_grant WHERE instance_id=$1 AND principal_id=$2 AND action='department.read'`, f.instance, reader.subject)
	must(t, e)
	query131Grant(t, f, reader, "bu", a, "department.read", []string{"name", "managementBuId"})
	tx, g, in := query131Guard(t, f, reader, q)
	defer tx.Rollback(context.Background())
	_, e = tx.Exec(f.ctx, `SELECT set_config('application_name','q131-transfer',true)`)
	must(t, e)
	gate, e := f.admin.Begin(f.ctx)
	must(t, e)
	defer gate.Rollback(context.Background())
	_, e = gate.Exec(f.ctx, `SELECT pg_advisory_xact_lock(1310094)`)
	must(t, e)
	sql := strings.Replace(history.Query131PlannedSQL, "FROM organization.department d\n", "FROM organization.department d CROSS JOIN (SELECT pg_advisory_xact_lock(1310094)) gate\n", 1)
	type reply struct {
		c   history.CapturedPlan131
		err error
	}
	done := make(chan reply, 1)
	ctx, cancel := context.WithTimeout(f.ctx, 15*time.Second)
	defer cancel()
	go func() { var r reply; r.err = r.c.Scan(tx.QueryRow(ctx, sql, in.Args()...)); done <- r }()
	query131WaitLock(t, f, "q131-transfer")
	admin.call("POST", "/api/v1/departments/"+object["departmentId"].(string)+"/records/"+object["targetRecordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": object["sourceVersion"], "expectedRecordRevision": object["targetRevision"], "changes": map[string]any{"managementBuId": b}}, foundation.NewID(), 200)
	must(t, gate.Commit(f.ctx))
	r := <-done
	must(t, r.err)
	old, _, e := history.ProjectCaptured131(f.ctx, g, in, r.c)
	must(t, e)
	if old.Count != 1 || old.Items[0]["managementBuId"] != a {
		t.Fatal("capture mixed current scope with later transfer", old)
	}
	must(t, g.RecordSessionActivity(f.ctx))
	must(t, tx.Commit(f.ctx))
	next, _, e := query131PlanRead(t, f, reader, q, in.Query)
	must(t, e)
	if next.Count != 0 || len(next.Items) != 0 {
		t.Fatal("A rights survived next statement after B transfer", next)
	}
	t.Log("real statement barrier: old count/page/current scope is A; next request loses object after B transfer")
}

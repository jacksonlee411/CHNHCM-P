//go:build integration && query131

package bootstrap_test

import (
	"context"
	"errors"
	"net/url"
	"slices"
	"testing"
	"time"

	accessdb "chnhcm.local/chnhcm/internal/contexts/access/infra/postgres"
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
)

func query131Guard(t *testing.T, f *fixture, b *browser, q history.Query131Input) (pgx.Tx, access.Guard, history.PlannedQuery131) {
	t.Helper()
	tx, e := f.runtime.Pool.Begin(f.ctx)
	must(t, e)
	// Cleanup covers every failure in plan construction and every caller failure.
	t.Cleanup(func() { _ = tx.Rollback(context.Background()) })
	u, e := url.Parse(f.server.URL)
	must(t, e)
	s := accessdb.Store{Pool: f.runtime.Pool, Instance: f.instance}
	g, e := s.Guard(f.ctx, tx, access.Credentials{Token: b.client.Jar.Cookies(u)[0].Value})
	must(t, e)
	at, e := time.Parse(time.RFC3339, q.Day+"T04:00:00Z")
	must(t, e)
	p, e := g.(access.ReadPlanner).PlanReads(f.ctx, q.Enterprise, at, access.DepartmentRead)
	must(t, e)
	return tx, g, history.PlannedQuery131{Base: q, Plan: p, Query: org.DepartmentListInput{Page: q.Page, PageSize: q.PageSize, Status: "all", Sort: "code"}}
}

func query131PlanRead(t *testing.T, f *fixture, b *browser, q history.Query131Input, query org.DepartmentListInput) (history.Query131Result, history.Query131Stats, error) {
	t.Helper()
	return query131ProductRead(t, f, b, q, query)
}

func query131Grant(t *testing.T, f *fixture, b *browser, kind, id, action string, fields []string) {
	t.Helper()
	tag, e := f.admin.Exec(f.ctx, `UPDATE access.subject_grant SET read_fields=$6 WHERE instance_id=$1 AND principal_id=$2 AND scope_kind=$3 AND scope_id=$4 AND action=$5`, f.instance, b.subject, kind, id, action, fields)
	must(t, e)
	if tag.RowsAffected() > 0 {
		return
	}
	_, e = f.admin.Exec(f.ctx, `INSERT INTO access.subject_grant(instance_id,grant_id,principal_id,scope_kind,scope_id,action,read_fields,write_fields) VALUES($1,gen_random_uuid(),$2,$3,$4,$5,$6,ARRAY[]::text[])`, f.instance, b.subject, kind, id, action, fields)
	must(t, e)
}

func TestDepartmentQuery131PlanAndClosure(t *testing.T) {
	f, admin, q := query131Fixture(t)
	a, b := query131BU(t, admin, "PLAN_A"), query131BU(t, admin, "PLAN_B")
	top := query131Create(t, admin, "A_TOP", "root-title", a, "2020-01-01")
	createChild := func(code, name, parent string) map[string]any {
		body := departmentCreateBody(code, a)
		body["name"], body["effectiveDate"], body["parentDepartmentId"] = name, "2026-01-01", parent
		return admin.call("POST", "/api/v1/departments", body, foundation.NewID(), 201)
	}
	parent := createChild("B_PARENT", "parent", top["departmentId"].(string))
	leaf := createChild("C_LEAF", "leaf_%中文𠀀", parent["departmentId"].(string))
	id := leaf["departmentId"].(string)
	query131Create(t, admin, "D_OTHER", "unrelated", b, "2026-01-01")
	query := org.DepartmentListInput{Status: "all", Sort: "code", Q: "LEAF_%中文𠀀"}
	v, stats, e := query131PlanRead(t, f, admin, q, query)
	must(t, e)
	if v.Count != 1 || len(v.Items) != 1 || v.Items[0]["id"] != id || v.Items[0]["departmentPath"] != "root-title / parent / leaf_%中文𠀀" || v.Items[0]["pathDay"] != q.Day || stats.DecodedPageSegments != 3 {
		t.Fatal("page ancestor closure or literal search differs", v, stats)
	}
	query.Q = "c_leaf"
	v, _, e = query131PlanRead(t, f, admin, q, query)
	must(t, e)
	if v.Count != 1 || v.Items[0]["id"] != id {
		t.Fatal("code search differs", v)
	}
	query.Q = ""
	query.Filter = &org.ColumnFilter{Field: "departmentPath", Operator: "contains", Value: "parent / leaf"}
	v, _, e = query131PlanRead(t, f, admin, q, query)
	must(t, e)
	if v.Count != -1 || len(v.Items) != 1 || v.Items[0]["id"] != id {
		t.Fatal("path cursor page differs", v)
	}
	reader := f.login("reader", f.readPassword)
	_, e = f.admin.Exec(f.ctx, `DELETE FROM access.subject_grant WHERE instance_id=$1 AND principal_id=$2 AND action='department.read'`, f.instance, reader.subject)
	must(t, e)
	query131Grant(t, f, reader, "bu", a, "department.read", []string{"code"})
	query131Grant(t, f, reader, "department", id, "department.read", []string{"name"})
	query.Filter = nil
	query.Q = "leaf_%中文𠀀"
	query.Sort = "name"
	v, _, e = query131PlanRead(t, f, reader, q, query)
	must(t, e)
	if v.Count != 1 || v.Items[0]["name"] != "leaf_%中文𠀀" || v.Items[0]["code"] != "C_LEAF" || v.Items[0]["departmentPath"] != nil {
		t.Fatal("overlapping field grants differ", v)
	}
	query.Q = "parent"
	v, _, e = query131PlanRead(t, f, reader, q, query)
	must(t, e)
	if v.Count != 1 {
		t.Fatal("visible code search must still match B_PARENT", v)
	}
	query.Q = "top"
	v, _, e = query131PlanRead(t, f, reader, q, query)
	must(t, e)
	if v.Count != 1 {
		t.Fatal(v)
	}
	query.Q = "root-title"
	v, _, e = query131PlanRead(t, f, reader, q, query)
	must(t, e)
	if v.Count != 0 {
		t.Fatal("hidden name influenced search", v)
	}
	query.Q = "unrelated"
	v, _, e = query131PlanRead(t, f, reader, q, query)
	must(t, e)
	if v.Count != 0 {
		t.Fatal("denied scope influenced search", v)
	}
	query.Q = "no match"
	query.Filter = &org.ColumnFilter{Field: "name", Operator: "contains", Value: "no match"}
	_, _, e = query131PlanRead(t, f, reader, q, query)
	var fault *foundation.Fault
	if !errors.As(e, &fault) || fault.Status != 403 {
		t.Fatal("field denial hidden by empty search", e)
	}
	query.Filter = nil
	query.Q = ""
	v, _, e = query131PlanRead(t, f, reader, q, query)
	must(t, e)
	ids := []string{top["departmentId"].(string), parent["departmentId"].(string)}
	slices.Sort(ids)
	if v.Count != 3 || v.Items[0]["id"] != ids[0] || v.Items[1]["id"] != ids[1] || v.Items[2]["id"] != id {
		t.Fatal("hidden name affected ordering", v)
	}
	// Remove the explicit department grant, move the current ownership, then
	// verify actual old-source HTTP access follows the new BU and history action.
	_, e = f.admin.Exec(f.ctx, `DELETE FROM access.subject_grant WHERE instance_id=$1 AND principal_id=$2 AND (scope_kind='department' OR action='department.history.read')`, f.instance, reader.subject)
	must(t, e)
	query131Grant(t, f, reader, "bu", a, "department.history.read", []string{"name"})
	admin.call("POST", "/api/v1/departments/"+id+"/records/"+leaf["targetRecordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": leaf["sourceVersion"], "expectedRecordRevision": leaf["targetRevision"], "changes": map[string]any{"managementBuId": b, "parentDepartmentId": nil}}, foundation.NewID(), 200)
	query.Q = "c_leaf"
	v, _, e = query131PlanRead(t, f, reader, q, query)
	must(t, e)
	if v.Count != 0 {
		t.Fatal("old A scope survived current B ownership", v)
	}
	reader.call("GET", "/api/v1/departments/"+id+"?sourceVersion="+leaf["sourceVersion"].(string), nil, "", 404)
	query131Grant(t, f, reader, "bu", b, "department.read", []string{"code", "name"})
	query131Grant(t, f, reader, "bu", b, "department.history.read", []string{"name"})
	old := reader.call("GET", "/api/v1/departments/"+id+"?sourceVersion="+leaf["sourceVersion"].(string), nil, "", 200)
	if old["selectedRecord"].(map[string]any)["name"] != "leaf_%中文𠀀" {
		t.Fatal("new B scope did not recover old A history", old)
	}
	_, e = f.admin.Exec(f.ctx, `DELETE FROM access.subject_grant WHERE instance_id=$1 AND principal_id=$2 AND action='department.reference.read'`, f.instance, reader.subject)
	must(t, e)
	query131Grant(t, f, reader, "bu", a, "department.read", []string{"code", "name", "departmentPath", "effectiveDate", "status"})
	query.Q = "b_parent"
	v, _, e = query131PlanRead(t, f, reader, q, query)
	must(t, e)
	if v.Count != 1 || v.Items[0]["departmentPath"] != nil || v.Items[0]["pathMode"] != "restricted" {
		t.Fatal("reference names leaked", v)
	}
	t.Log("real guarded plan, current BU history, field union/search/order/denial, complete ancestor and BU capture, and restricted path passed")
}

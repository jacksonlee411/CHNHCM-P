//go:build integration

package bootstrap_test

import (
	"testing"

	"chnhcm.local/chnhcm/internal/foundation"
)

// A sparse subtree must cross unreadable intermediate departments, retain the
// reference checks above its scope root, and ignore unrelated branches. Literal
// expected members come from normal HTTP creation receipts.
func TestDepartmentScope125SparseAncestorsAndReferenceDenial(t *testing.T) {
	f := fresh(t)
	admin, reader := f.login("admin", f.password), f.login("reader", f.readPassword)
	bu := admin.call("POST", "/api/v1/business-units", createBody("SCOPE125_BU", "2020-01-01"), foundation.NewID(), 201)["buId"].(string)
	create := func(code, parent string) string {
		body := departmentCreateBody(code, bu)
		body["effectiveDate"] = "2020-01-01"
		if parent != "" {
			body["parentDepartmentId"] = parent
		}
		return admin.call("POST", "/api/v1/departments", body, foundation.NewID(), 201)["departmentId"].(string)
	}
	top := create("SCOPE125_A", "")
	scope := create("SCOPE125_B", top)
	hidden := create("SCOPE125_C", scope)
	leaf := create("SCOPE125_D", hidden)
	other := create("SCOPE125_E", "")
	outside := create("SCOPE125_F", other)
	_, e := f.admin.Exec(f.ctx, `WITH original AS (
 DELETE FROM access.subject_grant WHERE instance_id=$1 AND principal_id=$2 AND action='department.read' RETURNING *
 ) INSERT INTO access.subject_grant
 SELECT instance_id,gen_random_uuid(),principal_id,'department',id,action,read_fields,write_fields
 FROM original CROSS JOIN unnest($3::uuid[]) id`, f.instance, reader.subject, []string{scope, leaf, outside})
	must(t, e)
	query := "/api/v1/departments?scopeDepartmentId=" + scope
	check := func(wanted ...string) {
		t.Helper()
		v := reader.call("GET", query, nil, "", 200)
		items := v["items"].([]any)
		if v["count"] != float64(len(wanted)) || len(items) != len(wanted) {
			t.Fatal("sparse scope count", v)
		}
		for i, id := range wanted {
			if items[i].(map[string]any)["id"] != id {
				t.Fatal("sparse scope member", v)
			}
		}
	}
	check(scope, leaf)
	if reader.call("GET", query+"&includeDescendants=false", nil, "", 200)["count"] != float64(1) {
		t.Fatal("self scope included a descendant")
	}
	// The outside cycle is only in this disposable derived index. A traversal
	// of the reader's candidates must terminate without changing this scope.
	_, e = f.admin.Exec(f.ctx, `UPDATE organization.department_read_segment SET q_parent=$1 WHERE instance_id=$2 AND object_id=$3`, outside, f.instance, other)
	must(t, e)
	check(scope, leaf)
	_, e = f.admin.Exec(f.ctx, `UPDATE organization.department_read_segment SET q_parent=NULL WHERE instance_id=$1 AND object_id=$2`, f.instance, other)
	must(t, e)
	// An inactive intermediate is outside today's administrative tree.
	_, e = f.admin.Exec(f.ctx, `UPDATE organization.department_read_segment SET q_state='inactive' WHERE instance_id=$1 AND object_id=$2`, f.instance, hidden)
	must(t, e)
	check(scope)
	_, e = f.admin.Exec(f.ctx, `UPDATE organization.department_read_segment SET q_state='active' WHERE instance_id=$1 AND object_id=$2`, f.instance, hidden)
	must(t, e)
	// Full reference permission still requires the scope's outside ancestors.
	_, e = f.admin.Exec(f.ctx, `UPDATE organization.department_read_segment SET q_parent=$1 WHERE instance_id=$2 AND object_id=$3`, foundation.NewID(), f.instance, top)
	must(t, e)
	reader.call("GET", query+"&q=not-a-match", nil, "", 503)
	_, e = f.admin.Exec(f.ctx, `UPDATE organization.department_read_segment SET q_parent=NULL WHERE instance_id=$1 AND object_id=$2`, f.instance, top)
	must(t, e)
	// Restrict minimal references to exactly the necessary ancestor closure.
	// No reference permission for the unrelated readable outside branch is
	// needed. A missing field on an unreadable intermediate must still deny.
	_, e = f.admin.Exec(f.ctx, `WITH original AS (
 DELETE FROM access.subject_grant WHERE instance_id=$1 AND principal_id=$2 AND action='department.reference.read' RETURNING *
 ) INSERT INTO access.subject_grant
 SELECT instance_id,gen_random_uuid(),principal_id,'department',id,action,ARRAY['code','name'],write_fields
 FROM original CROSS JOIN unnest($3::uuid[]) id`, f.instance, reader.subject, []string{top, scope, hidden, leaf})
	must(t, e)
	check(scope, leaf)
	for _, id := range []string{top, hidden} {
		_, e = f.admin.Exec(f.ctx, `UPDATE access.subject_grant SET read_fields=ARRAY['code'] WHERE instance_id=$1 AND principal_id=$2 AND action='department.reference.read' AND scope_id=$3`, f.instance, reader.subject, id)
		must(t, e)
		reader.call("GET", query+"&q=not-a-match", nil, "", 403)
		_, e = f.admin.Exec(f.ctx, `UPDATE access.subject_grant SET read_fields=ARRAY['code','name'] WHERE instance_id=$1 AND principal_id=$2 AND action='department.reference.read' AND scope_id=$3`, f.instance, reader.subject, id)
		must(t, e)
	}
	check(scope, leaf)
}

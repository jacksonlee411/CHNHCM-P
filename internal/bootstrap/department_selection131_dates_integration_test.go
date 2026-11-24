//go:build integration && query131

package bootstrap_test

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"testing"
)

// Use the real selection HTTP entry, not the ordinary-list capture harness.
// Expected dates, labels and current-BU reference rights are independent of SQL.
func TestDepartmentSelection131DatedPathsAndCurrentReferenceRights(t *testing.T) {
	f, admin, _ := query131Fixture(t)
	a, b := query131BU(t, admin, "R95_A"), query131BU(t, admin, "R95_B")
	parent := query131Create(t, admin, "R95_PARENT", "parent-before", a, "2020-01-01")
	pid := parent["departmentId"].(string)
	child := func(code, name, bu, day string) map[string]any {
		body := departmentCreateBody(code, bu)
		body["name"], body["effectiveDate"], body["parentDepartmentId"] = name, day, pid
		return admin.call("POST", "/api/v1/departments", body, foundation.NewID(), 201)
	}
	stopped := child("R95_STOPPED", "native-stopped-leaf", a, "2020-01-01")
	admin.call("POST", "/api/v1/departments/"+stopped["departmentId"].(string)+"/deactivate", map[string]any{
		"expectedSourceVersion": stopped["sourceVersion"], "anchorRecordId": stopped["targetRecordId"],
		"effectiveDate": "2026-06-01", "operationType": "DEACTIVATE", "operationReason": "ORG_RESTRUCTURING",
	}, foundation.NewID(), 201)
	head := admin.call("POST", "/api/v1/departments/"+pid+"/records", map[string]any{
		"expectedSourceVersion": parent["sourceVersion"], "anchorRecordId": parent["targetRecordId"],
		"effectiveDate": "2026-07-01", "operationType": "CHANGE", "operationReason": "ORGANIZATION_REASSIGNMENT",
		"changes": map[string]any{"managementBuId": b, "name": "parent-after"},
	}, foundation.NewID(), 201)
	admin.call("POST", "/api/v1/departments/"+pid+"/records", map[string]any{
		"expectedSourceVersion": head["sourceVersion"], "anchorRecordId": head["targetRecordId"],
		"effectiveDate": "2027-01-01", "operationType": "CHANGE", "operationReason": "ORGANIZATION_REASSIGNMENT",
		"changes": map[string]any{"name": "parent-future"},
	}, foundation.NewID(), 201)
	current := child("R95_CURRENT", "native-current-leaf", b, "2026-08-01")
	future := child("R95_FUTURE", "native-future-leaf", b, "2027-01-01")
	query131Create(t, admin, "R95_UNRELATED", "unrelated department", b, "2020-01-01")
	reader := f.login("reader", f.readPassword)
	selectAll := func(t *testing.T, actor *browser, query string, count int) (map[string]any, []any) {
		t.Helper()
		status, result, err := actor.response("POST", "/api/v1/departments/selection/start", map[string]any{
			"query":     map[string]any{"q": query, "status": "all"},
			"selection": map[string]any{"mode": "allMatching", "excludedIds": []string{}},
			"page":      1, "pageSize": 20, "sort": "code",
		}, "")
		must(t, err)
		if status != 200 {
			t.Fatal("selection failed", status, result)
		}
		page, summary := result["page"].(map[string]any), result["summary"].(map[string]any)
		items := page["items"].([]any)
		if page["count"] != float64(count) || summary["totalCount"] != nil || summary["selectedCount"] != float64(count) || len(items) != count || page["queryKey"] != summary["queryKey"] {
			t.Fatal("selection page and summary lost the independently expected set", page, summary)
		}
		return page, items
	}
	assertPath := func(t *testing.T, row map[string]any, id, state, day, label string) {
		t.Helper()
		path, ok := row["path"].(map[string]any)
		if !ok {
			t.Fatal("selection row or path missing", id, row)
		}
		if row["id"] != id || row["status"] != state || path["pathBusinessAt"] != day || path["departmentPath"] != ""+label {
			t.Fatal("selection used the wrong dated identity or path", row)
		}
	}
	t.Run("three_path_days", func(t *testing.T) {
		page, items := selectAll(t, admin, "native-", 3)
		rows := map[string]map[string]any{}
		for _, item := range items {
			row := item.(map[string]any)
			rows[row["id"].(string)] = row
		}
		assertPath(t, rows[stopped["departmentId"].(string)], stopped["departmentId"].(string), "inactive", "2026-05-31", "parent-before / native-stopped-leaf")
		assertPath(t, rows[current["departmentId"].(string)], current["departmentId"].(string), "active", page["businessAt"].(string), "parent-after / native-current-leaf")
		assertPath(t, rows[future["departmentId"].(string)], future["departmentId"].(string), "not_effective", "2027-01-01", "parent-future / native-future-leaf")
		if rows[future["departmentId"].(string)]["effectiveDate"] != "2027-01-01" {
			t.Fatal("future first was replaced with the query date")
		}
	})
	t.Run("one_and_no_match", func(t *testing.T) {
		_, items := selectAll(t, admin, "native-stopped-leaf", 1)
		assertPath(t, items[0].(map[string]any), stopped["departmentId"].(string), "inactive", "2026-05-31", "parent-before / native-stopped-leaf")
		selectAll(t, admin, "r95-no-match", 0)
	})
	t.Run("full_path_identity_capture_uses_each_business_day", func(t *testing.T) {
		for _, tc := range []struct{ text, id string }{
			{"parent-before / native-stopped-leaf", stopped["departmentId"].(string)},
			{"parent-after / native-current-leaf", current["departmentId"].(string)},
			{"parent-future / native-future-leaf", future["departmentId"].(string)},
		} {
			result := admin.call("POST", "/api/v1/departments/selection/start", map[string]any{
				"query":     map[string]any{"filterField": "departmentPath", "filterOperator": "contains", "filterValue": tc.text},
				"selection": map[string]any{"mode": "allMatching", "excludedIds": []string{}}, "page": 1, "pageSize": 20,
			}, "", 200)
			page, summary := result["page"].(map[string]any), result["summary"].(map[string]any)
			items := page["items"].([]any)
			if summary["selectedCount"] != float64(1) || page["count"] != nil || len(items) != 1 || items[0].(map[string]any)["id"] != tc.id {
				t.Fatal("full path identity does not match its dated page", result)
			}
		}
	})
	t.Run("current_reference_scope", func(t *testing.T) {
		_, err := f.admin.Exec(f.ctx, `DELETE FROM access.subject_grant WHERE instance_id=$1 AND principal_id=$2 AND action='department.reference.read'`, f.instance, reader.subject)
		must(t, err)
		query131Grant(t, f, reader, "bu", a, "department.reference.read", []string{"name"})
		_, items := selectAll(t, reader, "native-stopped-leaf", 1)
		if items[0].(map[string]any)["path"].(map[string]any)["departmentPath"] != nil {
			t.Fatal("historical A ownership restored current B reference permission")
		}
		query131Grant(t, f, reader, "bu", b, "department.reference.read", []string{"name"})
		_, items = selectAll(t, reader, "native-stopped-leaf", 1)
		assertPath(t, items[0].(map[string]any), stopped["departmentId"].(string), "inactive", "2026-05-31", "parent-before / native-stopped-leaf")
	})
}

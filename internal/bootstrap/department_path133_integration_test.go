//go:build integration

package bootstrap_test

import (
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"fmt"
	"net/url"
	"testing"
)

func TestDepartmentPath133CursorAndCompleteSelection(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	other := f.login("admin", f.password)
	bu := b.call("POST", "/api/v1/business-units", createBody("PATH133_BU", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	ids := []string{}
	for i := 0; i < 25; i++ {
		body := departmentCreateBody(fmt.Sprintf("PATH133_%02d", i), bu)
		body["name"] = "路径匹配"
		ids = append(ids, b.call("POST", "/api/v1/departments", body, foundation.NewID(), 201)["departmentId"].(string))
	}
	in := org.DepartmentListInput{Page: 1, PageSize: 20, Sort: "code", Filter: &org.ColumnFilter{Field: "departmentPath", Operator: "contains", Value: "路径匹配"}}
	origin, _ := url.Parse(f.server.URL)
	first, e := f.runtime.Department.List(f.ctx, foundation.Credentials{Token: b.client.Jar.Cookies(origin)[0].Value}, in)
	must(t, e)
	if len(first.Items) != 20 || !first.HasMore || first.NextCursor == "" {
		t.Fatal("path boundary", len(first.Items), first.HasMore)
	}
	endpoint := "/api/v1/departments?filterField=departmentPath&filterOperator=contains&filterValue=" + url.QueryEscape("路径匹配") + "&cursor=" + url.QueryEscape(first.NextCursor) + "&page=2"
	second := b.call("GET", endpoint, nil, "", 200)
	if second["count"] != nil || len(second["items"].([]any)) != 5 || second["items"].([]any)[0].(map[string]any)["id"] != ids[20] {
		t.Fatal("path continuation", second)
	}
	other.call("GET", endpoint, nil, "", 409)
	start := b.call("POST", "/api/v1/departments/selection/start", map[string]any{"query": map[string]any{"filterField": "departmentPath", "filterOperator": "contains", "filterValue": "路径匹配"}, "selection": map[string]any{"mode": "allMatching", "excludedIds": []string{}}, "page": 1, "pageSize": 20}, "", 200)
	if start["summary"].(map[string]any)["selectedCount"] != float64(25) {
		t.Fatal("full path selection truncated", start)
	}
	// The first fetched window can contain no matches. The ordinary page must
	// continue through the same capture, and a full selection must still exhaust it.
	body := departmentCreateBody("PATH133_ZZ", bu)
	body["name"] = "稀疏末项"
	last := b.call("POST", "/api/v1/departments", body, foundation.NewID(), 201)["departmentId"]
	sparse := b.call("GET", "/api/v1/departments?filterField=departmentPath&filterOperator=contains&filterValue="+url.QueryEscape("稀疏末项"), nil, "", 200)
	if items := sparse["items"].([]any); len(items) != 1 || items[0].(map[string]any)["id"] != last || sparse["hasMore"] != false {
		t.Fatal("sparse path stopped after an empty first window", sparse)
	}
	selected := b.call("POST", "/api/v1/departments/selection/start", map[string]any{"query": map[string]any{"filterField": "departmentPath", "filterOperator": "contains", "filterValue": "稀疏末项"}, "selection": map[string]any{"mode": "allMatching", "excludedIds": []string{}}, "page": 1, "pageSize": 20}, "", 200)
	if selected["summary"].(map[string]any)["selectedCount"] != float64(1) {
		t.Fatal("sparse complete selection differs", selected)
	}
}

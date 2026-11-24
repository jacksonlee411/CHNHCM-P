//go:build integration

package integration_test

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"testing"
)

func TestDepartmentQueryProtocolSelectionAndNavigation(t *testing.T) {
	s := newDepartmentContract(t)
	s.create("Q131_A", "查询协议部门", "BU_A", "", "2026-01-01", s.b, 201)
	query := map[string]any{"q": "查询协议"}
	list := s.b.call("GET", "/api/v1/departments", nil, "", 200)
	if list["selectionVersion"] != nil {
		t.Fatal("retired list digest")
	}
	start := map[string]any{"query": query, "selection": all131(), "page": 1, "pageSize": 20}
	result := s.b.call("POST", "/api/v1/departments/selection/start", start, "", 200)
	summary := result["summary"].(map[string]any)
	if summary["selectedCount"] != float64(1) || summary["draftId"] == nil || summary["selectionVersion"] != nil || summary["totalCount"] != nil {
		t.Fatal(result)
	}
	start["expectedSelectionVersion"] = "retired"
	s.b.call("POST", "/api/v1/departments/selection/start", start, "", 400)
	s.b.call("POST", "/api/v1/departments/selection/resolve", start, "", 400)
	body := draftBody(query, summary)
	body["page"] = 1
	body["pageSize"] = 20
	s.b.call("POST", "/api/v1/departments/selection/resolve", body, "", 200)
	delete(body, "page")
	s.b.call("POST", "/api/v1/departments/selection/resolve", body, "", 400)
	start131(s.b, query, explicit131(foundation.NewID()), 400)
	nav := map[string]any{"query": query, "anchorId": s.aliases["Q131_A"], "direction": "next"}
	v := s.b.call("POST", "/api/v1/departments/navigation", nav, "", 200)
	if v["targetId"] != nil || v["page"] != nil || v["navigationVersion"] != nil {
		t.Fatal(v)
	}
	s.edit("Q131_A", "Q131_A_FIRST", "correct", map[string]any{"changes": map[string]any{"name": "查询协议新名称"}}, s.b, 200)
	s.b.call("POST", "/api/v1/departments/navigation", nav, "", 200)
	nav["query"] = map[string]any{"q": "不存在"}
	s.b.call("POST", "/api/v1/departments/navigation", nav, "", 409)
	nav["query"] = map[string]any{"filterField": "name"}
	s.b.call("POST", "/api/v1/departments/navigation", nav, "", 400)
	nav["query"] = query
	nav["expectedNavigationVersion"] = "retired"
	s.b.call("POST", "/api/v1/departments/navigation", nav, "", 400)
}

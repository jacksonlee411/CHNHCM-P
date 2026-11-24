//go:build integration

package integration_test

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func departmentDraft(b *browser, query map[string]any) map[string]any {
	return start131(b, query, all131(), 200)["summary"].(map[string]any)
}
func draftBody(query, draft map[string]any) map[string]any {
	return map[string]any{"query": query, "draftId": draft["draftId"], "expectedRevision": draft["revision"]}
}
func departmentResolve(b *browser, choice map[string]any, status int) map[string]any {
	body := map[string]any{"page": 1, "pageSize": 20}
	for k, v := range choice {
		body[k] = v
	}
	result := b.call("POST", "/api/v1/departments/selection/resolve", body, "", status)
	if status == 200 {
		return result["summary"].(map[string]any)
	}
	return result
}

func departmentCSVResponse(t *testing.T, b *browser, body any) (int, []byte) {
	t.Helper()
	raw, e := json.Marshal(body)
	must(t, e)
	req, e := http.NewRequest("POST", b.f.server.URL+"/api/v1/departments/export", bytes.NewReader(raw))
	must(t, e)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", b.f.server.URL)
	req.Header.Set("X-CSRF-Token", b.csrf)
	r, e := b.client.Do(req)
	must(t, e)
	defer r.Body.Close()
	raw, e = io.ReadAll(r.Body)
	must(t, e)
	return r.StatusCode, raw
}
func TestDepartmentD19FullQueryCSVAndPathVersion(t *testing.T) {
	s := newDepartmentContract(t)
	s.create("P", "组织效能中心", "BU_A", "", "2026-01-01", s.b, 201)
	for i := 0; i < 137; i++ {
		s.create(fmt.Sprintf("C%03d", i), "组织发展部", "BU_A", "P", "2026-01-01", s.b, 201)
	}
	q := "q=" + url.QueryEscape("组织发展部")
	list := s.b.call("GET", "/api/v1/departments?"+q, nil, "", 200)
	if list["count"] != float64(137) || len(list["items"].([]any)) != 20 {
		t.Fatal("list did not cover 137 authorized rows")
	}
	query := map[string]any{"q": "组织发展部"}
	draft := departmentDraft(s.b, query)
	for _, suffix := range []string{"&page=2", "&pageSize=50", "&sort=name_desc"} {
		v := s.b.call("GET", "/api/v1/departments?"+q+suffix, nil, "", 200)
		if _, ok := v["selectionVersion"]; ok {
			t.Fatal("pagination or sorting reset collection")
		}
	}
	choice := draftBody(query, draft)
	choice["removeIds"] = []string{s.aliases["C136"]}
	selected := departmentResolve(s.b, choice, 200)
	choice = draftBody(query, selected)
	if selected["selectedCount"] != float64(136) {
		t.Fatal("cross-page exclusion failed")
	}
	status, raw := departmentCSVResponse(t, s.b, choice)
	if status != 200 {
		t.Fatalf("CSV status %d: %s", status, raw)
	}
	csvRows, e := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(raw), "\ufeff"))).ReadAll()
	must(t, e)
	if len(csvRows) != 137 || !reflect.DeepEqual(csvRows[0], []string{"编码", "名称", "简称", "BU编码", "BU名称", "部门路径", "状态", "生效日期"}) {
		t.Fatal("CSV count or eight-column contract")
	}
	for _, r := range csvRows[1:] {
		if len(r) != 8 || r[0] == "C136" || r[5] != "组织效能中心 / 组织发展部" || r[6] != "启用" {
			t.Fatal("CSV projection mismatch", r)
		}
	}
	childSource := s.current("C000")["headSourceVersion"]
	s.edit("P", "P_FIRST", "correct", map[string]any{"changes": map[string]any{"name": "组织管理中心"}}, s.b, 200)
	if s.current("C000")["headSourceVersion"] != childSource {
		t.Fatal("ancestor rename rewrote child")
	}
	departmentResolve(s.b, choice, 200)
	list = s.b.call("GET", "/api/v1/departments?"+q, nil, "", 200)
	if list["count"] != float64(137) {
		t.Fatal("current list changed unexpectedly")
	}
	status, raw = departmentCSVResponse(t, s.b, choice)
	if status != 200 || !strings.Contains(string(raw), "组织管理中心") {
		t.Fatal("reselected export not refreshed")
	}
	// Hidden ancestor changes cannot be inferred from a restricted path version.
	s.aliases["C"] = s.aliases["C000"]
	s.profile("path_reader_no_ancestor")
	list = s.actor.call("GET", "/api/v1/departments", nil, "", 200)
	draft = departmentDraft(s.actor, map[string]any{})
	s.edit("P", "P_FIRST", "correct", map[string]any{"changes": map[string]any{"name": "不可见更名"}}, s.b, 200)
	after := s.actor.call("GET", "/api/v1/departments", nil, "", 200)
	if after["count"] != list["count"] {
		t.Fatal("hidden ancestor changed restricted collection version")
	}
	choice = draftBody(map[string]any{}, draft)
	status, raw = departmentCSVResponse(t, s.actor, choice)
	if status != 200 {
		t.Fatal("restricted path export denied")
	}
	csvRows, e = csv.NewReader(strings.NewReader(strings.TrimPrefix(string(raw), "\ufeff"))).ReadAll()
	must(t, e)
	if len(csvRows) != 2 || csvRows[1][5] != "" || csvRows[1][3] != "" || csvRows[1][4] != "" {
		t.Fatal("minimum reference permission expanded")
	}
	mutateLifeGrants(s.f, s.actor.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='department.export'`, s.actor.subject)
	status, raw = departmentCSVResponse(t, s.actor, choice)
	if status != 403 || strings.Contains(string(raw), "编码,名称") {
		t.Fatal("revoked export emitted partial CSV")
	}
}

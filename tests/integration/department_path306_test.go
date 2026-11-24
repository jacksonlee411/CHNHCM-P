//go:build integration

package integration_test

import (
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func TestDepartmentPath306RealGroupAndOriginalDates(t *testing.T) {
	s := newDepartmentContract(t)
	for _, row := range []struct{ alias, name, date string }{{"CURRENT306", "当前部门", "2026-01-01"}, {"STOP306", "旧部门", "2026-01-01"}, {"FUTURE306", "未来部门", "2026-12-01"}} {
		s.create(row.alias, row.name, "BU_A", "", row.date, s.b, 201)
	}
	s.append("STOP306", "STOP_RECORD", "2026-10-01", "DEACTIVATE", nil, s.b, 201)
	originals := map[string]map[string]any{}
	for _, alias := range []string{"CURRENT306", "STOP306", "FUTURE306"} {
		head := s.current(alias)
		originals[alias] = s.exact(alias, head["headSourceVersion"].(string))
	}
	// A real department named like the enterprise must remain in the path once.
	withoutGroup := s.b.call("GET", "/api/v1/departments?filterField=departmentPath&filterOperator=contains&filterValue="+url.QueryEscape("高谷集团"), nil, "", 200)
	if len(withoutGroup["items"].([]any)) != 0 {
		t.Fatal("enterprise name must not match department paths")
	}
	s.create("GROUP306", "高谷集团", "BU_A", "", "2026-01-01", s.b, 201)
	for _, alias := range []string{"CURRENT306", "STOP306", "FUTURE306"} {
		s.edit(alias, alias+"_FIRST", "correct", map[string]any{"changes": map[string]any{"parentDepartmentId": s.aliases["GROUP306"]}}, s.b, 200)
		current := s.current(alias)
		old := originals[alias]
		if depPath(current) != "高谷集团 / "+depPath(old) {
			t.Fatalf("real group path: %s", depPath(current))
		}
		if current["status"] != old["status"] {
			t.Fatal("data completion changed status")
		}
		afterOld := s.exact(alias, old["resolvedSourceVersion"].(string))
		if !reflect.DeepEqual(old["path"], afterOld["path"]) || !reflect.DeepEqual(old["timeline"], afterOld["timeline"]) {
			t.Fatal("old source changed")
		}
		beforeRows, afterRows := old["timeline"].([]any), current["timeline"].([]any)
		if len(beforeRows) != len(afterRows) {
			t.Fatal("data completion added effective records")
		}
		for i := range beforeRows {
			before, after := beforeRows[i].(map[string]any), afterRows[i].(map[string]any)
			if before["recordId"] != after["recordId"] || before["effectiveDate"] != after["effectiveDate"] {
				t.Fatal("original dates/identities changed")
			}
		}
	}
	query := map[string]any{"filterField": "departmentPath", "filterOperator": "contains", "filterValue": "高谷集团"}
	list := s.b.call("GET", "/api/v1/departments?filterField=departmentPath&filterOperator=contains&filterValue="+url.QueryEscape("高谷集团"), nil, "", 200)
	if len(list["items"].([]any)) != 4 {
		t.Fatal("real group must be searchable")
	}
	status, csv := departmentCSVResponse(t, s.b, draftBody(query, departmentDraft(s.b, query)))
	if status != 200 || !strings.Contains(string(csv), "高谷集团 / 未来部门") || strings.Contains(string(csv), "高谷集团 / 高谷集团") {
		t.Fatalf("CSV path status %d", status)
	}
}

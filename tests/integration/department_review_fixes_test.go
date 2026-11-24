//go:build integration

package integration_test

import (
	"encoding/csv"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"chnhcm.local/chnhcm/internal/foundation"
)

func TestDepartmentPairDeletionFieldProjection(t *testing.T) {
	for _, bounded := range []bool{false, true} {
		t.Run(fmt.Sprintf("bounded=%t", bounded), func(t *testing.T) {
			s := newDepartmentContract(t)
			allDepartmentRights(s)
			s.create("C", "原部门名", "BU_A", "", "2026-01-01", s.b, 201)
			s.append("C", "STOP", "2026-03-01", "DEACTIVATE", nil, s.b, 201)
			s.append("C", "RESUME", "2026-04-01", "REACTIVATE", map[string]any{"name": "恢复后部门名"}, s.b, 201)
			if bounded {
				s.append("C", "LATER", "2026-09-01", "CHANGE", map[string]any{"name": "后续独立名称"}, s.b, 201)
			}
			path := "/api/v1/departments/" + s.aliases["C"] + "?selectedRecordId=" + s.records["STOP"]
			pair := func() map[string]any {
				t.Helper()
				detail := s.actor.call("GET", path, nil, "", 200)
				p, ok := detail["pairDeletion"].(map[string]any)
				if !ok {
					t.Fatal("retained deletion permissions must retain the pair summary")
				}
				return p
			}
			visible := func() {
				t.Helper()
				changes := pair()["changes"].([]any)
				if len(changes) != 1 {
					t.Fatalf("want one visible name change, got %v", changes)
				}
				ch := changes[0].(map[string]any)
				var end any
				if bounded {
					end = "2026-09-01"
				}
				if got, exists := ch["endExclusive"]; !exists || got != end || ch["start"] != "2026-04-01" {
					t.Fatalf("authorized interval must distinguish finite and open end: %v", ch)
				}
			}
			visible()
			beforeSources := s.f.scalar("SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.department_history")
			// History grants remain intact: ordinary detail must obey current read fields.
			mutateLifeGrants(s.f, s.actor.subject, `UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'effectiveDate') WHERE principal_id=$1 AND action='department.read'`, s.actor.subject)
			for _, record := range []string{"STOP", "RESUME"} {
				detail := s.actor.call("GET", "/api/v1/departments/"+s.aliases["C"]+"?selectedRecordId="+s.records[record], nil, "", 200)
				if _, present := detail["selectedRecord"].(map[string]any)["effectiveDate"]; present {
					t.Fatal("ordinary selected record still exposes revoked date")
				}
				p, ok := detail["pairDeletion"].(map[string]any)
				if !ok || len(p["changes"].([]any)) != 1 {
					t.Fatal("date revocation must retain authorized name changes")
				}
				var noDates func(any)
				noDates = func(value any) {
					switch v := value.(type) {
					case map[string]any:
						for key, child := range v {
							switch key {
							case "start", "endExclusive", "endInclusive", "effectiveDate":
								t.Errorf("revoked date key %s remains in pair deletion: %v", key, child)
							}
							noDates(child)
						}
					case []any:
						for _, child := range v {
							noDates(child)
						}
					}
				}
				noDates(p)
				ch := p["changes"].([]any)[0].(map[string]any)
				if ch["before"].(map[string]any)["name"] != "恢复后部门名" || ch["after"].(map[string]any)["name"] != "原部门名" {
					t.Fatal("authorized before/after names lost")
				}
			}
			mutateLifeGrants(s.f, s.actor.subject, `UPDATE access.subject_grant SET read_fields=array_remove(array_remove(read_fields,'name'),'shortName') WHERE principal_id=$1 AND action='department.read'`, s.actor.subject)
			if len(pair()["changes"].([]any)) != 0 {
				t.Fatal("hidden-only data differences must not create change summaries")
			}
			mutateLifeGrants(s.f, s.actor.subject, `UPDATE access.subject_grant SET read_fields=read_fields || ARRAY['effectiveDate','name','shortName'] WHERE principal_id=$1 AND action='department.read'`, s.actor.subject)
			visible()
			if s.f.scalar("SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.department_history") != beforeSources {
				t.Fatal("read-time authorization changed business history")
			}
			mutateLifeGrants(s.f, s.actor.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='department.reactivation.delete'`, s.actor.subject)
			if _, exists := s.actor.call("GET", path, nil, "", 200)["pairDeletion"]; exists {
				t.Fatal("missing one delete permission must remove pair capability")
			}
		})
	}
}

func TestDepartmentManagementBUStableFilter(t *testing.T) {
	s := newDepartmentContract(t)
	s.create("A", "同名部门", "BU_A", "", "2026-01-01", s.b, 201)
	s.create("B", "同名部门", "BU_B", "", "2026-01-01", s.b, 201)
	query := map[string]any{"filterField": "managementBuId", "filterOperator": "equals", "filterValue": s.aliases["BU_A"]}
	urlFor := func(q map[string]any) string {
		values := url.Values{}
		for k, v := range q {
			values.Set(k, v.(string))
		}
		return "/api/v1/departments?" + values.Encode()
	}
	assertSelected := func(buName string) map[string]any {
		t.Helper()
		list := s.b.call("GET", urlFor(query), nil, "", 200)
		items := list["items"].([]any)
		if list["count"] != float64(1) || len(items) != 1 || items[0].(map[string]any)["id"] != s.aliases["A"] {
			t.Fatalf("stable ID must select only BU_A department, got %v", list)
		}
		choice := draftBody(query, departmentDraft(s.b, query))
		resolved := departmentResolve(s.b, choice, 200)
		if resolved["selectedCount"] != float64(1) || resolved["canExport"] != true {
			t.Fatal("selection diverged from stable ID list", resolved)
		}
		status, raw := departmentCSVResponse(t, s.b, choice)
		if status != 200 {
			t.Fatalf("CSV HTTP %d: %s", status, raw)
		}
		rows, e := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(raw), "\ufeff"))).ReadAll()
		must(t, e)
		if len(rows) != 2 || rows[1][0] != "A" || rows[1][3] != "BU_A" || rows[1][4] != buName {
			t.Fatal("CSV differs from stable ID selection", rows)
		}
		return choice
	}
	originalChoice := assertSelected("未来业务单元")
	bu := detail(s.b, s.aliases["BU_A"])
	r := bu["selectedRecord"].(map[string]any)
	s.b.call("POST", "/api/v1/business-units/"+s.aliases["BU_A"]+"/records/"+r["recordId"].(string)+"/correct", correctBody(bu, r, map[string]any{"name": "调整后的事业部"}), foundation.NewID(), 200)
	departmentResolve(s.b, originalChoice, 200)
	if status, raw := departmentCSVResponse(t, s.b, originalChoice); status != 200 || !strings.Contains(string(raw), "调整后的事业部") {
		t.Fatal("current BU label export", status)
	}
	choice := assertSelected("调整后的事业部")
	query["filterValue"] = s.aliases["BU_B"]
	if s.b.call("GET", urlFor(query), nil, "", 200)["items"].([]any)[0].(map[string]any)["id"] != s.aliases["B"] {
		t.Fatal("other same-name BU must select its own department")
	}
	query["filterValue"] = foundation.NewID()
	if s.b.call("GET", urlFor(query), nil, "", 200)["count"] != float64(0) {
		t.Fatal("unknown stable BU ID must return empty set")
	}
	for _, invalid := range []struct{ operator, value string }{{"contains", s.aliases["BU_A"]}, {"equals", "BU_A 调整后的事业部"}} {
		query["filterOperator"], query["filterValue"] = invalid.operator, invalid.value
		s.b.call("GET", urlFor(query), nil, "", 400)
		choice["query"] = query
		departmentResolve(s.b, choice, 400)
		s.b.call("POST", "/api/v1/departments/export", choice, "", 400)
	}
	query["filterOperator"], query["filterValue"] = "equals", s.aliases["BU_A"]
	// The readable identity remains usable even when its display label is restricted.
	mutateLifeGrants(s.f, s.b.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='bu.reference.read'`, s.b.subject)
	list := s.b.call("GET", urlFor(query), nil, "", 200)
	if list["count"] != float64(1) || list["items"].([]any)[0].(map[string]any)["managementBu"].(map[string]any)["name"] != nil {
		t.Fatal("stable ID filtering depends on BU label permission")
	}
	choice = draftBody(query, departmentDraft(s.b, query))
	departmentResolve(s.b, choice, 200)
	mutateLifeGrants(s.f, s.b.subject, `UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'managementBuId') WHERE principal_id=$1 AND action='department.read'`, s.b.subject)
	s.b.call("GET", urlFor(query), nil, "", 403)
	departmentResolve(s.b, choice, 403)
	s.b.call("POST", "/api/v1/departments/export", choice, "", 403)
}

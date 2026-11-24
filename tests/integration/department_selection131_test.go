//go:build integration

package integration_test

import (
	"encoding/csv"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"chnhcm.local/chnhcm/internal/foundation"
)

func start131(b *browser, query map[string]any, choice map[string]any, status int) map[string]any {
	return b.call("POST", "/api/v1/departments/selection/start", map[string]any{"query": query, "selection": choice, "page": 1, "pageSize": 20}, "", status)
}
func all131() map[string]any                   { return map[string]any{"mode": "allMatching", "excludedIds": []string{}} }
func explicit131(ids ...string) map[string]any { return map[string]any{"mode": "explicit", "ids": ids} }
func assertPage131(t *testing.T, result map[string]any, total, selected int) (map[string]any, map[string]any) {
	t.Helper()
	p, s := result["page"].(map[string]any), result["summary"].(map[string]any)
	if p["count"] != float64(total) || p["queryKey"] != s["queryKey"] || s["selectedCount"] != float64(selected) {
		t.Fatal("atomic page/summary mismatch", result)
	}
	if _, ok := p["selectionVersion"]; ok {
		t.Fatal("duplicate page version")
	}
	return p, s
}

func TestDepartmentSelection131EstablishmentAndSafeErrors(t *testing.T) {
	s := newDepartmentContract(t)
	s.create("MATCH_A", "命中甲", "BU_A", "", "2026-01-01", s.b, 201)
	securityEvents := s.f.scalar("SELECT count(*) FROM access.security_event")
	query := map[string]any{"q": "MATCH"}
	opened := s.b.call("GET", "/api/v1/departments?q=MATCH", nil, "", 200)
	if opened["count"] != float64(1) {
		t.Fatal(opened)
	}
	s.create("MATCH_B", "命中乙", "BU_A", "", "2026-01-01", s.b, 201)
	var before int
	must(t, s.f.admin.QueryRow(s.f.ctx, "SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.department_history").Scan(&before))
	_, established := assertPage131(t, start131(s.b, query, all131(), 200), 2, 2)
	page, _ := assertPage131(t, start131(s.b, query, explicit131(s.aliases["MATCH_A"]), 200), 2, 1)
	if len(page["items"].([]any)) != 2 {
		t.Fatal(page)
	}
	var after int
	must(t, s.f.admin.QueryRow(s.f.ctx, "SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.department_history").Scan(&after))
	if before != after {
		t.Fatal("selection wrote business sources")
	}
	var audits int
	must(t, s.f.admin.QueryRow(s.f.ctx, "SELECT count(*) FROM information_schema.tables WHERE table_schema='access' AND table_name='evidence_content'").Scan(&audits))
	if audits != 0 {
		t.Fatal("retired authorization process storage returned")
	}
	for _, tc := range []struct {
		name  string
		query map[string]any
		id    string
	}{
		{"unknown", query, foundation.NewID()},
		{"no-longer-matching", map[string]any{"q": "absent"}, s.aliases["MATCH_A"]},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := start131(s.b, tc.query, explicit131(tc.id), 400); got["code"] != "M02_INVALID_SELECTION" {
				t.Fatal(got)
			}
		})
	}
	// Keep an entry grant while removing access to the explicitly selected ID.
	mutateLifeGrants(s.f, s.actor.subject, `UPDATE access.subject_grant SET scope_kind='bu',scope_id=$2 WHERE principal_id=$1 AND action LIKE 'department.%'`, s.actor.subject, s.aliases["BU_B"])
	if got := start131(s.actor, query, explicit131(s.aliases["MATCH_A"]), 400); got["code"] != "M02_INVALID_SELECTION" {
		t.Fatal(got)
	}
	departmentResolve(s.b, draftBody(query, established), 409) // The explicit replacement owns this session now.
	established = departmentDraft(s.b, query)
	s.create("MATCH_C", "建立后新增", "BU_A", "", "2026-01-01", s.b, 201)
	if got := departmentResolve(s.b, draftBody(query, established), 200); got["selectedCount"] != float64(2) {
		t.Fatal("new member joined fixed draft", got)
	}
	mutateLifeGrants(s.f, s.actor.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='department.read'`, s.actor.subject)
	start131(s.actor, query, all131(), 403)
	if s.f.scalar("SELECT count(*) FROM access.security_event") != securityEvents {
		t.Fatal("selection success or denial persisted authorization events")
	}
}

func TestDepartmentSelection131SevenPagesAndIndependentCSV(t *testing.T) {
	s := newDepartmentContract(t)
	s.create("P131", "独立上级", "BU_A", "", "2026-01-01", s.b, 201)
	want := make([][]string, 137)
	for i := range want {
		code, name, short := fmt.Sprintf("CSV131_%03d", i), fmt.Sprintf("名称%03d", 136-i), fmt.Sprintf("简称%03d", i)
		body := departmentBody(code, name, s.aliases["BU_A"], "2026-01-01", s.aliases["P131"])
		body["shortName"] = short
		v := s.b.call("POST", "/api/v1/departments", body, foundation.NewID(), 201)
		s.aliases[code] = v["departmentId"].(string)
		want[i] = []string{code, name, short, "BU_A", "未来业务单元", "独立上级 / " + name, "启用", "2026-01-01"}
	}
	query := map[string]any{"q": "CSV131_"}
	_, summary := assertPage131(t, start131(s.b, query, all131(), 200), 137, 137)
	revision := summary["revision"]
	seen := map[string]bool{}
	for page := 1; page <= 7; page++ {
		result := s.b.call("POST", "/api/v1/departments/selection/resolve", map[string]any{"query": query, "draftId": summary["draftId"], "expectedRevision": revision, "page": page, "pageSize": 20}, "", 200)
		p, sm := assertPage131(t, result, 137, 137)
		if sm["revision"] != revision {
			t.Fatal("page changed version")
		}
		for offset, value := range p["items"].([]any) {
			r := value.(map[string]any)
			code := want[(page-1)*20+offset][0]
			if r["id"] != s.aliases[code] || seen[code] {
				t.Fatal("identity or order", code)
			}
			seen[code] = true
		}
	}
	if len(seen) != 137 {
		t.Fatal("seven-page coverage", len(seen))
	}
	edit := draftBody(query, summary)
	edit["removeIds"] = []string{s.aliases[want[20][0]], s.aliases[want[136][0]]}
	summary = departmentResolve(s.b, edit, 200)
	revision = summary["revision"]
	for _, size := range []int{20, 50, 100} {
		for _, sort := range []string{"code", "name", "name_desc"} {
			result := s.b.call("POST", "/api/v1/departments/selection/resolve", map[string]any{"query": query, "draftId": summary["draftId"], "expectedRevision": revision, "page": 2, "pageSize": size, "sort": sort}, "", 200)
			_, sm := assertPage131(t, result, 137, 135)
			if sm["revision"] != revision {
				t.Fatal("size/sort changed version")
			}
		}
	}
	// Expected cells originate in the creation inputs, not list/CSV responses.
	for _, actor := range []*browser{s.b, s.actor} {
		if actor == s.actor {
			v := departmentDraft(actor, query)
			status, raw := departmentCSVResponse(t, actor, draftBody(query, v))
			if status != 403 || strings.Contains(string(raw), "编码,名称") {
				t.Fatal("reader unexpectedly exported", status)
			}
			mutateLifeGrants(s.f, actor.subject, `INSERT INTO access.subject_grant(instance_id,grant_id,principal_id,scope_kind,scope_id,action,read_fields,write_fields) SELECT instance_id,$2,principal_id,scope_kind,scope_id,'department.export','{}','{}' FROM access.subject_grant WHERE principal_id=$1 AND action='department.read' LIMIT 1`, actor.subject, foundation.NewID())
		}
		for _, mode := range []string{"all", "explicit", "excluded"} {
			for _, sort := range []string{"code", "name", "name_desc"} {
				choice := all131()
				indices := []int{}
				for i := range want {
					if mode == "all" || mode == "explicit" && (i == 0 || i == 136) || mode == "excluded" && i != 20 && i != 136 {
						indices = append(indices, i)
					}
				}
				if mode == "explicit" {
					choice = explicit131(s.aliases[want[0][0]], s.aliases[want[136][0]])
				}
				if mode == "excluded" {
					choice = all131()
				}
				if sort == "name" {
					for i, j := 0, len(indices)-1; i < j; i, j = i+1, j-1 {
						indices[i], indices[j] = indices[j], indices[i]
					}
				}
				v := start131(actor, query, choice, 200)["summary"].(map[string]any)
				if mode == "excluded" {
					edit := draftBody(query, v)
					edit["removeIds"] = []string{s.aliases[want[20][0]], s.aliases[want[136][0]]}
					v = departmentResolve(actor, edit, 200)
				}
				body := draftBody(query, v)
				body["sort"] = sort
				status, raw := departmentCSVResponse(t, actor, body)
				if status != 200 {
					t.Fatalf("CSV %s %s: %d %s", mode, sort, status, raw)
				}
				rows, e := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(raw), "\ufeff"))).ReadAll()
				must(t, e)
				if len(rows) != len(indices)+1 || !reflect.DeepEqual(rows[0], []string{"编码", "名称", "简称", "BU编码", "BU名称", "部门路径", "状态", "生效日期"}) {
					t.Fatal("CSV shape")
				}
				for at, index := range indices {
					if !reflect.DeepEqual(rows[at+1], want[index]) {
						t.Fatalf("independent CSV %s %s row %d: got %v want %v", mode, sort, at, rows[at+1], want[index])
					}
				}
			}
		}
	}
}

func TestDepartmentSelection133CurrentProjectionAndDayExpiry(t *testing.T) {
	s := newDepartmentContract(t)
	s.create("P", "同名路径上级", "BU_A", "", "2026-01-01", s.b, 201)
	s.create("C", "唯一选中子部门", "BU_A", "P", "2026-01-01", s.b, 201)
	s.create("OTHER", "无关兄弟", "BU_A", "P", "2026-01-01", s.b, 201)
	query := map[string]any{"q": "唯一选中子部门"}
	draft := departmentDraft(s.b, query)
	body := draftBody(query, draft)
	s.edit("P", "P_FIRST", "correct", map[string]any{"changes": map[string]any{"shortName": "不可见简称推进路径来源"}}, s.b, 200)
	departmentResolve(s.b, body, 200)
	s.edit("OTHER", "OTHER_FIRST", "correct", map[string]any{"changes": map[string]any{"name": "无关兄弟新名"}}, s.b, 200)
	mutateLifeGrants(s.f, s.b.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='department.record.add'`, s.b.subject)
	if got := departmentResolve(s.b, body, 200); got["draftId"] != draft["draftId"] || got["revision"] != draft["revision"] {
		t.Fatal("unrelated edit invalidated intent", got)
	}
	s.profile("path_reader_no_ancestor")
	actorDraft := departmentDraft(s.actor, map[string]any{})
	s.edit("P", "P_FIRST", "correct", map[string]any{"changes": map[string]any{"name": "隐藏上级新名"}}, s.b, 200)
	actorBody := draftBody(map[string]any{}, actorDraft)
	departmentResolve(s.actor, actorBody, 200)
	mutateLifeGrants(s.f, s.actor.subject, `UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'shortName') WHERE principal_id=$1 AND action='department.read'`, s.actor.subject)
	s.edit("C", "C_FIRST", "correct", map[string]any{"changes": map[string]any{"shortName": "完全不可见的修改"}}, s.b, 200)
	actorBody["page"] = 1
	actorBody["pageSize"] = 20
	result := s.actor.call("POST", "/api/v1/departments/selection/resolve", actorBody, "", 200)
	for _, row := range result["page"].(map[string]any)["items"].([]any) {
		v := row.(map[string]any)
		if v["shortName"] != nil || v["path"].(map[string]any)["departmentPath"] != nil {
			t.Fatal("hidden current material leaked", v)
		}
	}
	mutateLifeGrants(s.f, s.actor.subject, `UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'name') WHERE principal_id=$1 AND action='department.read'`, s.actor.subject)
	result = s.actor.call("POST", "/api/v1/departments/selection/resolve", actorBody, "", 200)
	for _, row := range result["page"].(map[string]any)["items"].([]any) {
		if row.(map[string]any)["name"] != nil {
			t.Fatal("revoked field leaked")
		}
	}
	s.day = "2026-11-05"
	departmentResolve(s.b, body, 409)
}

func TestDepartmentSelection131SelectedReferenceAcrossRealPages(t *testing.T) {
	s := newDepartmentContract(t)
	for i := 0; i < 21; i++ {
		code := fmt.Sprintf("REF131_%02d", i)
		s.create(code, code, "BU_A", "", "2026-01-01", s.b, 201)
	}
	chosen := s.aliases["REF131_20"]
	query := map[string]any{"context": map[string]any{"mode": "create", "effectiveDate": "2026-01-01", "managementBuId": s.aliases["BU_A"]}, "selectedId": chosen, "pageSize": 20}
	for page := 1; page <= 2; page++ {
		v := s.b.call("POST", "/api/v1/departments/references/parents", query, "", 200)
		if v["count"] != nil || v["selectedReference"].(map[string]any)["id"] != chosen {
			t.Fatal("selected reference depends on page", v)
		}
		query["cursor"] = v["nextCursor"]
		items := v["items"].([]any)
		if page == 1 {
			if len(items) != 20 {
				t.Fatal("first candidate page", len(items))
			}
			for _, item := range items {
				if item.(map[string]any)["id"] == chosen {
					t.Fatal("fixture selected reference must be outside first page")
				}
			}
		} else if len(items) != 1 || items[0].(map[string]any)["id"] != chosen {
			t.Fatal("second candidate page does not contain independent selection")
		}
	}
}

//go:build integration

package integration_test

import (
	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/foundation"
	"strings"
	"testing"
)

func departmentBody(code, name, bu, date string, parent any) map[string]any {
	return map[string]any{"code": code, "name": name, "managementBuId": bu, "parentDepartmentId": parent, "effectiveDate": date, "operationType": "CREATE", "operationReason": "NEW_ORGANIZATION"}
}
func TestDepartmentRootChildFirstChain(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	b.call("GET", "/api/v1/departments/context", nil, "", 200)
	bu := b.call("POST", "/api/v1/business-units", createBody("DEPTBU", "2020-01-01"), foundation.NewID(), 201)
	rootKey := foundation.NewID()
	rootBody := departmentBody("ROOT", "总部", bu["buId"].(string), "2020-01-01", nil)
	root := b.call("POST", "/api/v1/departments", rootBody, rootKey, 201)
	id := root["departmentId"].(string)
	child := b.call("POST", "/api/v1/departments", departmentBody("CHILD", "研发部", bu["buId"].(string), "2021-01-01", id), foundation.NewID(), 201)
	childID := child["departmentId"].(string)
	detail := b.call("GET", "/api/v1/departments/"+childID, nil, "", 200)
	selected := detail["selectedRecord"].(map[string]any)
	if selected["name"] != "研发部" || selected["operationSummary"] == nil {
		t.Fatalf("missing department or operation summary: %v", detail)
	}
	path := detail["path"].(map[string]any)
	if !strings.Contains(path["departmentPath"].(string), "总部") {
		t.Fatalf("parent absent in path: %v", path)
	}
	old := b.call("GET", "/api/v1/departments/"+childID+"?sourceVersion="+child["sourceVersion"].(string), nil, "", 200)
	if old["path"].(map[string]any)["departmentPath"] != path["departmentPath"] {
		t.Fatal("exact path differs from first source")
	}
	recovered := b.call("POST", "/api/v1/departments/submissions/"+rootKey+"/resolve", nil, "", 200)
	if recovered["sourceVersion"] != root["sourceVersion"] || recovered["departmentId"] != id {
		t.Fatalf("different recovered result: %v", recovered)
	}
	replay := b.call("POST", "/api/v1/departments", rootBody, rootKey, 201)
	if replay["sourceVersion"] != root["sourceVersion"] || replay["replayed"] != true {
		t.Fatalf("different replay: %v", replay)
	}
	future := b.call("POST", "/api/v1/departments", departmentBody("FUTURE", "未来部门", bu["buId"].(string), "2099-01-01", id), foundation.NewID(), 201)
	futureDetail := b.call("GET", "/api/v1/departments/"+future["departmentId"].(string), nil, "", 200)
	if futureDetail["result"] != "M02_NO_FACT" || futureDetail["selectedRecordIsFuture"] != true {
		t.Fatalf("future first became current: %v", futureDetail)
	}
	list := b.call("GET", "/api/v1/departments", nil, "", 200)
	if list["count"] != float64(3) {
		t.Fatalf("missing list entries: %v", list)
	}
	facts, e := bootstrap.InspectHistory(f.ctx, f.admin, f.instance)
	must(t, e)
	if len(facts.Departments) != 3 {
		t.Fatalf("expected three complete atomic department sources, got %d", len(facts.Departments))
	}
	for _, v := range facts.Departments {
		if !foundation.ValidID(v.Operation.RelationEventID) || v.Operation.Terminal.Outcome != "accepted" || len(v.Evidence.Segments) == 0 {
			t.Fatal("source lacks relation event, accepted terminal or frozen evidence")
		}
	}
	if f.scalar("SELECT count(*) FROM organization.department") != 3 || f.scalar("SELECT count(*) FROM organization.department_hierarchy_relation") != 3 {
		t.Fatal("missing stable identities")
	}
	stop := map[string]any{"expectedSourceVersion": bu["sourceVersion"], "anchorRecordId": bu["targetRecordId"], "effectiveDate": "2030-01-01", "operationType": "DEACTIVATE", "operationReason": "BUSINESS_CLOSURE"}
	b.call("POST", "/api/v1/business-units/"+bu["buId"].(string)+"/deactivate", stop, foundation.NewID(), 409)
	if f.scalar("SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.bu_history") != 1 {
		t.Fatal("BU reverse protection committed an invalid source")
	}
	reader := f.login("reader", f.readPassword)
	mutateLifeGrants(f, reader.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='department.history.read'`, reader.subject)
	r := reader.call("GET", "/api/v1/departments/"+childID, nil, "", 200)
	if r["selectedRecord"].(map[string]any)["operationSummary"] == nil {
		t.Fatal("summary depends on history.read")
	}
	reader.call("GET", "/api/v1/departments/"+childID+"?sourceVersion="+child["sourceVersion"].(string), nil, "", 403)
}

func (b *browser) departmentCurrent(id string) map[string]any {
	return b.call("GET", "/api/v1/departments/"+id, nil, "", 200)
}
func (b *browser) departmentRecord(id, record string) (map[string]any, map[string]any) {
	detail := b.departmentCurrent(id)
	for _, v := range detail["timeline"].([]any) {
		r := v.(map[string]any)
		if r["recordId"] == record {
			return detail, r
		}
	}
	b.f.t.Fatalf("department record missing: %s", record)
	return nil, nil
}
func (b *browser) departmentCorrect(id, record string, extra map[string]any, status int) map[string]any {
	detail, r := b.departmentRecord(id, record)
	body := map[string]any{"expectedSourceVersion": detail["headSourceVersion"], "expectedRecordRevision": r["revision"]}
	for k, v := range extra {
		body[k] = v
	}
	return b.call("POST", "/api/v1/departments/"+id+"/records/"+record+"/correct", body, foundation.NewID(), status)
}
func TestDepartmentMaintenanceAndReverseChecks(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	bu := b.call("POST", "/api/v1/business-units", createBody("ACTIONS", "2020-01-01"), foundation.NewID(), 201)
	root := b.call("POST", "/api/v1/departments", departmentBody("P", "上级原名", bu["buId"].(string), "2020-01-01", nil), foundation.NewID(), 201)
	parent := root["departmentId"].(string)
	first := b.call("POST", "/api/v1/departments", departmentBody("C", "原名", bu["buId"].(string), "2020-01-01", parent), foundation.NewID(), 201)
	id := first["departmentId"].(string)
	record := first["targetRecordId"].(string)
	b.departmentCorrect(parent, root["targetRecordId"].(string), map[string]any{"changes": map[string]any{"name": "上级现名"}}, 200)
	current := b.departmentCurrent(id)
	if !strings.Contains(current["path"].(map[string]any)["departmentPath"].(string), "上级现名") {
		t.Fatal("current path not refreshed")
	}
	exact := b.call("GET", "/api/v1/departments/"+id+"?sourceVersion="+first["sourceVersion"].(string), nil, "", 200)
	if !strings.Contains(exact["path"].(map[string]any)["departmentPath"].(string), "上级原名") {
		t.Fatal("frozen path was rewritten")
	}
	if current["headSourceVersion"] != first["sourceVersion"] {
		t.Fatal("ancestor rename rewrote child source")
	}
	add := b.call("POST", "/api/v1/departments/"+id+"/records", map[string]any{"expectedSourceVersion": first["sourceVersion"], "anchorRecordId": record, "effectiveDate": "2021-01-01", "operationType": "CHANGE", "operationReason": "NAME_ADJUSTMENT", "changes": map[string]any{"name": "新名"}}, foundation.NewID(), 201)
	added := add["targetRecordId"].(string)
	b.departmentCorrect(id, added, map[string]any{"changes": map[string]any{"shortName": "简称"}}, 200)
	detail, r := b.departmentRecord(id, added)
	moved := b.call("POST", "/api/v1/departments/"+id+"/records/"+added+"/redate", map[string]any{"expectedSourceVersion": detail["headSourceVersion"], "expectedRecordRevision": r["revision"], "newEffectiveDate": "2022-01-01"}, foundation.NewID(), 200)
	stop := b.call("POST", "/api/v1/departments/"+id+"/deactivate", map[string]any{"expectedSourceVersion": moved["sourceVersion"], "anchorRecordId": added, "effectiveDate": "2023-01-01", "operationType": "DEACTIVATE", "operationReason": "BUSINESS_CLOSURE"}, foundation.NewID(), 201)
	stopped := b.departmentCurrent(id)
	if stopped["path"].(map[string]any)["pathMode"] != "inactive_before_stop" {
		t.Fatal("inactive path lacks S-1 mode")
	}
	resume := b.call("POST", "/api/v1/departments/"+id+"/reactivate", map[string]any{"expectedSourceVersion": stop["sourceVersion"], "anchorRecordId": stop["targetRecordId"], "effectiveDate": "2024-01-01", "operationType": "REACTIVATE", "operationReason": "BUSINESS_RESUMPTION"}, foundation.NewID(), 201)
	pair := b.call("POST", "/api/v1/departments/"+id+"/lifecycle-pairs", map[string]any{"expectedSourceVersion": resume["sourceVersion"], "anchorRecordId": resume["targetRecordId"], "deactivationDate": "2030-01-01", "deactivationReason": "ORG_RESTRUCTURING", "reactivationDate": "2031-01-01", "reactivationReason": "BUSINESS_RESUMPTION"}, foundation.NewID(), 200)
	if len(pair["items"].([]any)) != 2 {
		t.Fatal("fixed pair not atomic")
	}
	items := pair["items"].([]any)
	a := items[0].(map[string]any)
	z := items[1].(map[string]any)
	b.call("POST", "/api/v1/departments/"+id+"/lifecycle-pairs/delete", map[string]any{"expectedSourceVersion": pair["sourceVersion"], "deactivationRecordId": a["recordId"], "deactivationRevision": a["afterRevisionId"], "reactivationRecordId": z["recordId"], "reactivationRevision": z["afterRevisionId"]}, foundation.NewID(), 200)
	detail, r = b.departmentRecord(id, added)
	b.call("POST", "/api/v1/departments/"+id+"/records/"+added+"/delete", map[string]any{"expectedSourceVersion": detail["headSourceVersion"], "expectedRecordRevision": r["revision"]}, foundation.NewID(), 200)
	before := b.departmentCurrent(parent)
	b.departmentCorrect(parent, root["targetRecordId"].(string), map[string]any{"newEffectiveDate": "2021-01-01"}, 409)
	b.departmentCorrect(parent, root["targetRecordId"].(string), map[string]any{"changes": map[string]any{"parentDepartmentId": id}}, 409)
	if b.departmentCurrent(parent)["headSourceVersion"] != before["headSourceVersion"] {
		t.Fatal("invalid parent change committed")
	}
	b.departmentCorrect(id, record, map[string]any{"changes": map[string]any{"name": "原名"}}, 200)
	// Explicit null clears only the optional parent; future independent records retain theirs.
	b.departmentCorrect(id, record, map[string]any{"changes": map[string]any{"parentDepartmentId": nil}}, 200)
}

func TestDepartmentAssistanceHistoryAndSelection(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	bu := b.call("POST", "/api/v1/business-units", createBody("ASSIST", "2020-01-01"), foundation.NewID(), 201)
	root := b.call("POST", "/api/v1/departments", departmentBody("ROOT", "根", bu["buId"].(string), "2020-01-01", nil), foundation.NewID(), 201)
	parent := root["departmentId"].(string)
	first := b.call("POST", "/api/v1/departments", departmentBody("CHILD", "子", bu["buId"].(string), "2020-01-01", parent), foundation.NewID(), 201)
	id := first["departmentId"].(string)
	record := first["targetRecordId"].(string)
	for _, kind := range []string{"business-units", "parents"} {
		input := map[string]any{"context": map[string]any{"mode": "create", "effectiveDate": "2020-01-01", "managementBuId": bu["buId"]}, "pageSize": 20}
		if kind == "business-units" {
			input["page"] = 1
		}
		v := b.call("POST", "/api/v1/departments/references/"+kind, input, "", 200)
		if len(v["items"].([]any)) < 1 {
			t.Fatalf("empty %s candidates", kind)
		}
	}
	before := f.scalar("SELECT (SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.department_history)+(SELECT count(*) FROM organization.department_terminal)")
	preview := b.call("POST", "/api/v1/departments/"+id+"/preview", map[string]any{"action": "CORRECT", "expectedSourceVersion": first["sourceVersion"], "recordId": record, "expectedRecordRevision": first["targetRevision"], "changes": map[string]any{"name": "预览名"}}, "", 200)
	if preview["valid"] != true || f.scalar("SELECT (SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.department_history)+(SELECT count(*) FROM organization.department_terminal)") != before {
		t.Fatal("preview did not stay read-only")
	}
	add := b.call("POST", "/api/v1/departments/"+id+"/records", map[string]any{"expectedSourceVersion": first["sourceVersion"], "anchorRecordId": record, "effectiveDate": "2021-01-01", "operationType": "CHANGE", "operationReason": "NAME_ADJUSTMENT", "changes": map[string]any{"name": "新名"}}, foundation.NewID(), 201)
	aid := add["targetRecordId"].(string)
	log := b.call("GET", "/api/v1/departments/"+id+"/changes", nil, "", 200)
	if log["total"] != float64(2) {
		t.Fatalf("missing log: %v", log)
	}
	del := b.call("POST", "/api/v1/departments/"+id+"/records/"+aid+"/delete", map[string]any{"expectedSourceVersion": add["sourceVersion"], "expectedRecordRevision": add["targetRevision"]}, foundation.NewID(), 200)
	tomb := b.call("GET", "/api/v1/departments/"+id+"?sourceVersion="+del["sourceVersion"].(string)+"&selectedRecordId="+aid, nil, "", 200)
	selected := tomb["selectedRecord"].(map[string]any)
	if selected["deleted"] != true || selected["operationSummary"].(map[string]any)["operationSourceVersion"] != add["sourceVersion"] {
		t.Fatal("tombstone attributed to deletion actor")
	}
	pinned := b.call("GET", "/api/v1/departments/"+id+"/changes?throughSourceVersion="+log["throughSourceVersion"].(string), nil, "", 200)
	if pinned["total"] != float64(2) {
		t.Fatal("history window grew")
	}
	b.call("GET", "/api/v1/departments", nil, "", 200)
	choice := draftBody(map[string]any{}, departmentDraft(b, map[string]any{}))
	selection := departmentResolve(b, choice, 200)
	if selection["selectedCount"] != float64(2) || selection["canExport"] != true {
		t.Fatal("selection differs")
	}
	b.departmentCorrect(parent, root["targetRecordId"].(string), map[string]any{"changes": map[string]any{"name": "根更名"}}, 200)
	departmentResolve(b, choice, 200)
	b.call("GET", "/api/v1/departments/"+id+"/changes?filterField=changedField&filterOperator=is&filterValue=managementBuId", nil, "", 200)
}

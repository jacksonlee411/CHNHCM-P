//go:build integration

package bootstrap_test

import (
	"chnhcm.local/chnhcm/internal/bootstrap"
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func enterprise304(b *browser) (string, map[string]any) {
	c := b.call("GET", "/api/v1/enterprise/context", nil, "", 200)
	path := "/api/v1/enterprise/" + c["id"].(string)
	return path, b.call("GET", path, nil, "", 200)
}
func record304(v map[string]any) map[string]any { return v["selectedRecord"].(map[string]any) }
func correct304(v map[string]any, changes map[string]any) map[string]any {
	return map[string]any{"expectedSourceVersion": v["headSourceVersion"], "expectedRecordRevision": record304(v)["revision"], "changes": changes}
}
func addition304(v map[string]any, date, kind, name string) map[string]any {
	reason := "NAME_ADJUSTMENT"
	if kind == "DEACTIVATE" {
		reason = "BUSINESS_CLOSURE"
	}
	if kind == "REACTIVATE" {
		reason = "BUSINESS_RESUMPTION"
	}
	body := map[string]any{"expectedSourceVersion": v["headSourceVersion"], "anchorRecordId": record304(v)["recordId"], "effectiveDate": date, "operationType": kind, "operationReason": reason}
	if kind != "DEACTIVATE" {
		body["changes"] = map[string]any{"name": name}
	}
	return body
}
func TestEnterprise304MaintenanceAndRecovery(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	reader := f.login("reader", f.readPassword)
	path, first := enterprise304(b)
	firstID := record304(first)["recordId"].(string)
	var declaration string
	must(t, f.admin.QueryRow(f.ctx, "SELECT row_to_json(e)::text FROM organization.enterprise e").Scan(&declaration))
	original := b.call("GET", path+"?sourceVersion="+first["headSourceVersion"].(string), nil, "", 200)
	snapshots := map[string]map[string]any{first["headSourceVersion"].(string): original}
	receipts := map[string]map[string]any{}
	save := func(route string, body map[string]any, status int) map[string]any {
		key := foundation.NewID()
		r := b.call("POST", path+route, body, key, status)
		receipts[key] = r
		if r["sourceVersion"] != nil {
			src := r["sourceVersion"].(string)
			snapshots[src] = b.call("GET", path+"?sourceVersion="+src, nil, "", 200)
		}
		replay := b.call("POST", path+route, body, key, status)
		if replay["replayed"] != true || replay["sourceVersion"] != r["sourceVersion"] {
			t.Fatal("submission replay differs")
		}
		return r
	}
	body := correct304(first, map[string]any{"name": "企业更名", "nameEn": "Enterprise"})
	body["newEffectiveDate"] = "2019-01-01"
	body["classification"] = map[string]any{"operationType": "CREATE", "operationReason": "NEW_ORGANIZATION"}
	save("/records/"+firstID+"/correct", body, 200)
	_, head := enterprise304(b)
	if record304(head)["recordId"] != firstID || record304(head)["effectiveDate"] != "2019-01-01" {
		t.Fatal("first identity/date")
	}
	save("/records", addition304(head, "2024-01-01", "CHANGE", "第四年名称"), 201)
	head = b.call("GET", path+"?selectedRecordId="+firstID, nil, "", 200)
	inserted := save("/records", addition304(head, "2022-01-01", "CHANGE", "中间名称"), 201)
	rid := inserted["selectedRecordId"].(string)
	head = b.call("GET", path+"?selectedRecordId="+rid, nil, "", 200)
	body = correct304(head, map[string]any{"name": "中间修订", "nameEn": nil})
	body["newEffectiveDate"] = "2023-01-01"
	body["classification"] = map[string]any{"operationType": "CHANGE", "operationReason": "OTHER_DATA_ADJUSTMENT"}
	save("/records/"+rid+"/correct", body, 200)
	head = b.call("GET", path+"?selectedRecordId="+rid, nil, "", 200)
	previewDeleted := b.call("POST", path+"/preview", map[string]any{"action": "DELETE", "expectedSourceVersion": head["headSourceVersion"], "recordId": rid, "expectedRecordRevision": record304(head)["revision"]}, "", 200)
	if previewDeleted["afterRecord"].(map[string]any)["name"] != "企业更名" {
		t.Fatal("deletion preview did not show predecessor")
	}
	originalSummary := record304(head)["operationSummary"]
	save("/records/"+rid+"/delete", map[string]any{"expectedSourceVersion": head["headSourceVersion"], "expectedRecordRevision": record304(head)["revision"]}, 200)
	deleted := b.call("GET", path+"?selectedRecordId="+rid, nil, "", 200)
	if !reflect.DeepEqual(originalSummary, record304(deleted)["operationSummary"]) {
		t.Fatal("deleted original summary changed")
	}
	if deleted["readOnly"] != true || record304(deleted)["deleted"] != true || record304(deleted)["name"] != "中间修订" {
		t.Fatal("deleted original lost")
	}
	head = b.call("GET", path+"?selectedRecordId="+firstID, nil, "", 200)
	save("/lifecycle-pairs", map[string]any{"expectedSourceVersion": head["headSourceVersion"], "anchorRecordId": firstID, "deactivationDate": "2020-01-01", "reactivationDate": "2021-01-01", "deactivationReason": "BUSINESS_CLOSURE", "reactivationReason": "BUSINESS_RESUMPTION"}, 200)
	_, head = enterprise304(b)
	rows := head["timeline"].([]any)
	stop := rows[1].(map[string]any)
	resume := rows[2].(map[string]any)
	pairDetail := b.call("GET", path+"?selectedRecordId="+stop["recordId"].(string), nil, "", 200)
	if pairDetail["pairDeletion"] == nil {
		t.Fatal("pair deletion missing")
	}
	save("/lifecycle-pairs/delete", map[string]any{"expectedSourceVersion": head["headSourceVersion"], "deactivationRecordId": stop["recordId"], "reactivationRecordId": resume["recordId"], "deactivationRevision": stop["revision"], "reactivationRevision": resume["revision"]}, 200)
	_, head = enterprise304(b)
	save("/deactivate", addition304(head, "2025-01-01", "DEACTIVATE", ""), 201)
	_, head = enterprise304(b)
	save("/reactivate", addition304(head, "2026-01-01", "REACTIVATE", "恢复企业"), 201)
	_, head = enterprise304(b)
	n := f.scalar("SELECT sum(cardinality(source_refs)) FROM organization.enterprise_history")
	same := save("/records/"+record304(head)["recordId"].(string)+"/correct", correct304(head, map[string]any{"name": "恢复企业"}), 200)
	if same["outcome"] != "no_change" || n != f.scalar("SELECT sum(cardinality(source_refs)) FROM organization.enterprise_history") {
		t.Fatal("no-change grew source")
	}
	save("/records", addition304(head, "2099-01-01", "CHANGE", "未来企业"), 201)
	_, head = enterprise304(b)
	if record304(head)["name"] != "恢复企业" {
		t.Fatal("future replaced today")
	}
	future := head["timeline"].([]any)
	futureID := future[len(future)-1].(map[string]any)["recordId"].(string)
	if b.call("GET", path+"?selectedRecordId="+futureID, nil, "", 200)["selectedRecordIsFuture"] != true {
		t.Fatal("future marker")
	}
	head = b.call("GET", path+"?selectedRecordId="+firstID, nil, "", 200)
	b.call("POST", path+"/records/"+firstID+"/delete", map[string]any{"expectedSourceVersion": head["headSourceVersion"], "expectedRecordRevision": record304(head)["revision"]}, foundation.NewID(), 403)
	reader.call("POST", path+"/records/"+firstID+"/correct", correct304(head, map[string]any{"name": "无权"}), foundation.NewID(), 403)
	changes := b.call("GET", path+"/changes", nil, "", 200)
	if int(changes["total"].(float64)) != f.scalar("SELECT sum(cardinality(source_refs)) FROM organization.enterprise_history") {
		t.Fatal("change history missing")
	}
	must(t, bootstrap.Initialize(f.ctx, f.admin, bootstrap.InitInput{Instance: f.instance, EnterpriseCode: "LOCAL", EnterpriseName: "本地验证企业", Start: "2020-01-01", Login: "admin", Password: f.password}))
	var after string
	must(t, f.admin.QueryRow(f.ctx, "SELECT row_to_json(e)::text FROM organization.enterprise e").Scan(&after))
	if declaration != after {
		t.Fatal("initialization declaration changed")
	}
	_, current := enterprise304(b)
	if current["headSourceVersion"] != head["headSourceVersion"] {
		t.Fatal("reinitialization replaced maintenance")
	}
	// Reconnect after the real database process stops, then compare all exact sources and results.
	keys := []string{}
	for key, receipt := range receipts {
		if receipt["outcome"] == "accepted" {
			keys = append(keys, key)
		}
	}
	restartTimelineReceipts(t, f, b, keys, "enterprise")
	for src, old := range snapshots {
		now := b.call("GET", path+"?sourceVersion="+src, nil, "", 200)
		if !reflect.DeepEqual(old["timeline"], now["timeline"]) {
			t.Fatal("source changed after restart", src)
		}
	}
	for key, old := range receipts {
		now := b.call("POST", "/api/v1/enterprise/submissions/"+key+"/resolve", nil, "", 200)
		if old["outcome"] != now["outcome"] || old["sourceVersion"] != now["sourceVersion"] {
			t.Fatal("result lost after restart")
		}
	}
	closed := foundation.NewID()
	if b.call("POST", "/api/v1/enterprise/submissions/"+closed+"/resolve", nil, "", 200)["outcome"] != "not_applied" {
		t.Fatal("resolve closure")
	}
	b.call("POST", path+"/records/"+firstID+"/correct", correct304(head, map[string]any{"name": "不得写入"}), closed, 409)
}

func TestEnterprise304References(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	path, head := enterprise304(b)
	firstID := record304(head)["recordId"].(string)
	bu := b.call("POST", "/api/v1/business-units", createBody("REF304", "2020-01-01"), foundation.NewID(), 201)["buId"].(string)
	bp := "/api/v1/business-units/" + bu
	body := correct304(head, nil)
	delete(body, "changes")
	body["newEffectiveDate"] = "2021-01-01"
	rejected := foundation.NewID()
	v := b.call("POST", path+"/records/"+firstID+"/correct", body, rejected, 409)
	if v["code"] != "M02_REFERENCE_CONFLICT" {
		t.Fatal(v)
	}
	preview := map[string]any{"action": "CORRECT", "recordId": firstID}
	for k, v := range body {
		preview[k] = v
	}
	b.call("POST", path+"/preview", preview, "", 409)
	b.call("POST", path+"/deactivate", addition304(head, "2022-01-01", "DEACTIVATE", ""), foundation.NewID(), 409)
	unchanged := b.call("GET", path, nil, "", 200)
	if !reflect.DeepEqual(head["timeline"], unchanged["timeline"]) {
		t.Fatal("failed dependency mutated enterprise")
	}
	depBody := departmentCreateBody("REF_DEP304", bu)
	depBody["effectiveDate"], depBody["name"] = "2020-01-01", "引用部门304"
	dep := b.call("POST", "/api/v1/departments", depBody, foundation.NewID(), 201)["departmentId"].(string)
	dp := "/api/v1/departments/" + dep
	d := b.call("GET", dp, nil, "", 200)
	b.call("POST", dp+"/deactivate", addition304(d, "2022-01-01", "DEACTIVATE", ""), foundation.NewID(), 201)
	d = b.call("GET", bp, nil, "", 200)
	b.call("POST", bp+"/deactivate", addition304(d, "2022-01-01", "DEACTIVATE", ""), foundation.NewID(), 201)
	// Same half-open boundary is legal after dependants are inactive.
	b.call("POST", path+"/deactivate", addition304(head, "2022-01-01", "DEACTIVATE", ""), foundation.NewID(), 201)
	d = b.call("GET", bp, nil, "", 200)
	bad := addition304(d, "2023-01-01", "REACTIVATE", "越界业务单元")
	b.call("POST", bp+"/reactivate", bad, foundation.NewID(), 409)
	pv := map[string]any{"action": "REACTIVATE"}
	for k, v := range bad {
		pv[k] = v
	}
	b.call("POST", bp+"/preview", pv, "", 409)
	b.call("POST", "/api/v1/business-units", createBody("GAP304", "2023-01-01"), foundation.NewID(), 409)
	_, head = enterprise304(b)
	b.call("POST", path+"/reactivate", addition304(head, "2025-01-01", "REACTIVATE", "恢复企业"), foundation.NewID(), 201)
	d = b.call("GET", bp, nil, "", 200)
	b.call("POST", bp+"/reactivate", addition304(d, "2025-01-01", "REACTIVATE", "恢复业务单元"), foundation.NewID(), 201)
	d = b.call("GET", dp, nil, "", 200)
	b.call("POST", dp+"/reactivate", addition304(d, "2025-01-01", "REACTIVATE", "引用部门304"), foundation.NewID(), 201)
	_, head = enterprise304(b)
	head = b.call("GET", path+"?selectedRecordId="+firstID, nil, "", 200)
	b.call("POST", path+"/lifecycle-pairs", map[string]any{"expectedSourceVersion": head["headSourceVersion"], "anchorRecordId": firstID, "deactivationDate": "2020-02-01", "reactivationDate": "2020-03-01", "deactivationReason": "BUSINESS_CLOSURE", "reactivationReason": "BUSINESS_RESUMPTION"}, foundation.NewID(), 409)
	if b.call("POST", "/api/v1/enterprise/submissions/"+rejected+"/resolve", nil, "", 200)["outcome"] != "rejected" {
		t.Fatal("rejected result not retained")
	}
	before := logicalBusinessSnapshot(f)
	b.call("POST", path+"/records/"+firstID+"/correct", correct304(head, map[string]any{"name": "新企业名称"}), foundation.NewID(), 200)
	if !reflect.DeepEqual(before, logicalBusinessSnapshot(f)) {
		t.Fatal("enterprise rename rewrote BU/department sources")
	}
	for _, url := range []string{"/api/v1/organization/context", "/api/v1/departments/context", "/api/v1/locations/context", bp, dp} {
		v := b.call("GET", url, nil, "", 200)
		if v["enterprise"].(map[string]any)["name"] != "恢复企业" {
			t.Fatal("current enterprise name did not follow dated record", url)
		}
	}
	d = b.call("GET", dp, nil, "", 200)
	oldSource := d["headSourceVersion"].(string)
	old := b.call("GET", dp+"?sourceVersion="+oldSource, nil, "", 200)
	_, head = enterprise304(b)
	b.call("POST", path+"/records/"+record304(head)["recordId"].(string)+"/correct", correct304(head, map[string]any{"name": "当前企业路径304"}), foundation.NewID(), 200)
	current := b.call("GET", dp, nil, "", 200)
	if current["path"].(map[string]any)["departmentPath"] != "引用部门304" {
		t.Fatal("current detail path must contain only departments", current["path"])
	}
	for _, query := range []string{"q=REF_DEP304", "filterField=departmentPath&filterOperator=contains&filterValue=304"} {
		v := b.call("GET", "/api/v1/departments?"+query, nil, "", 200)
		items := v["items"].([]any)
		if len(items) != 1 || items[0].(map[string]any)["path"].(map[string]any)["departmentPath"] != "引用部门304" {
			t.Fatal("captured list/path filter includes enterprise", v)
		}
	}
	selection := b.call("POST", "/api/v1/departments/selection/start", map[string]any{"query": map[string]any{"q": "REF_DEP304"}, "selection": map[string]any{"mode": "allMatching", "excludedIds": []string{}}, "page": 1, "pageSize": 20}, "", 200)
	if strings.Contains(fmt.Sprint(selection["page"]), "当前企业路径304") || !strings.Contains(fmt.Sprint(selection["page"]), "引用部门304") {
		t.Fatal("selection path must contain only departments")
	}
	unchangedSource := b.call("GET", dp+"?sourceVersion="+oldSource, nil, "", 200)
	if !reflect.DeepEqual(old["path"], unchangedSource["path"]) {
		t.Fatal("enterprise rename rewrote captured path")
	}
	saved := b.call("POST", dp+"/records/"+record304(current)["recordId"].(string)+"/correct", correct304(current, map[string]any{"name": "路径快照部门"}), foundation.NewID(), 200)
	newSource := b.call("GET", dp+"?sourceVersion="+saved["sourceVersion"].(string), nil, "", 200)
	if newSource["path"].(map[string]any)["departmentPath"] != "路径快照部门" {
		t.Fatal("new department source path must contain only departments", newSource["path"])
	}
}

func TestEnterprise304ConcurrentCoverage(t *testing.T) {
	f := fresh(t)
	a := f.login("admin", f.password)
	b := f.login("admin", f.password)
	path, d := enterprise304(a)
	body := correct304(d, nil)
	delete(body, "changes")
	body["newEffectiveDate"] = "2021-01-01"
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	status := make([]int, 2)
	errs := make([]error, 2)
	go func() {
		defer wg.Done()
		<-start
		status[0], _, errs[0] = a.response("POST", path+"/records/"+record304(d)["recordId"].(string)+"/correct", body, foundation.NewID())
	}()
	go func() {
		defer wg.Done()
		<-start
		status[1], _, errs[1] = b.response("POST", "/api/v1/business-units", createBody("RACE304", "2020-01-01"), foundation.NewID())
	}()
	close(start)
	wg.Wait()
	for _, e := range errs {
		must(t, e)
	}
	if !(status[0] == 200 && status[1] == 400 || status[0] == 409 && status[1] == 201) {
		t.Fatalf("invalid race outcomes %v", status)
	}
	t.Logf("enterprise/BU concurrent coverage outcomes %v", status)
}

func TestEnterprise304AuthorizationAndValidation(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	path, head := enterprise304(b)
	rid := record304(head)["recordId"].(string)
	for _, key := range []string{"name", "classification", "newEffectiveDate", "changes"} {
		body := correct304(head, map[string]any{"name": "不应保存"})
		if key == "name" {
			body["changes"] = map[string]any{"name": nil}
		} else {
			body[key] = nil
		}
		b.call("POST", path+"/records/"+rid+"/correct", body, foundation.NewID(), 400)
	}
	for _, field := range []string{"code", "buId", "parentDepartmentId", "instanceId"} {
		body := correct304(head, map[string]any{field: "非法"})
		b.call("POST", path+"/records/"+rid+"/correct", body, foundation.NewID(), 400)
	}
	b.call("POST", "/api/v1/enterprise/"+foundation.NewID()+"/records/"+rid+"/correct", correct304(head, map[string]any{"name": "非法"}), foundation.NewID(), 404)
	// Same UUID is independent across enterprise and BU submission namespaces.
	key := foundation.NewID()
	bu := b.call("POST", "/api/v1/business-units", createBody("NS304", "2020-01-01"), key, 201)
	en := b.call("POST", path+"/records/"+rid+"/correct", correct304(head, map[string]any{"name": "企业独立提交"}), key, 200)
	if bu["sourceVersion"] == en["sourceVersion"] {
		t.Fatal("cross-object collision")
	}
	f.exec("UPDATE access.subject_grant SET read_fields=array_remove(array_remove(read_fields,'nameEn'),'state') WHERE principal_id=$1 AND action LIKE 'enterprise.%'", b.subject)
	_, head = enterprise304(b)
	if _, ok := record304(head)["nameEn"]; ok {
		t.Fatal("field leaked")
	}
	if head["status"] != nil {
		t.Fatal("status leaked")
	}
	preview := correct304(head, map[string]any{"name": "可预览"})
	preview["action"] = "CORRECT"
	preview["recordId"] = rid
	p := b.call("POST", path+"/preview", preview, "", 200)
	if p["currentStatus"] != nil || p["afterRecord"].(map[string]any)["nameEn"] != nil {
		t.Fatal("preview leaked")
	}
	f.exec("DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='enterprise.history.read'", b.subject)
	_, head = enterprise304(b)
	if head["capabilities"].(map[string]any)["correct"] != false {
		t.Fatal("history required for maintenance")
	}
	b.call("GET", path+"?sourceVersion="+en["sourceVersion"].(string), nil, "", 403)
	b.call("POST", path+"/records/"+rid+"/correct", correct304(head, map[string]any{"name": "不得更正"}), foundation.NewID(), 403)
	b.call("POST", path+"/preview", preview, "", 403)
	if record304(head)["operationSummary"] == nil {
		t.Fatal("current summary incorrectly requires history permission")
	}
	// Direct immutable storage cannot erase sources or the initialization declaration.
	for _, sql := range []string{"UPDATE organization.enterprise SET name='altered'", "DELETE FROM organization.enterprise_history", "DELETE FROM organization.enterprise_profile"} {
		if _, e := f.admin.Exec(f.ctx, sql); e == nil {
			t.Fatal("immutable guard missing", sql)
		}
	}
}

func TestEnterprise304TwentyRecords(t *testing.T) {
	f := freshWithEnterpriseStart(t, "1900-01-01")
	b := f.login("admin", f.password)
	path, head := enterprise304(b)
	started := time.Now()
	for i := 1; i < 20; i++ {
		body := addition304(head, fmt.Sprintf("%04d-01-01", 1900+i), "CHANGE", fmt.Sprintf("企业%d", i))
		b.call("POST", path+"/records", body, foundation.NewID(), 201)
		_, head = enterprise304(b)
	}
	if len(head["timeline"].([]any)) != 20 {
		t.Fatal("twenty-record scenario incomplete")
	}
	t.Logf("20-record functional creation and reads elapsed=%s (not P95)", time.Since(started))
	for _, v := range head["timeline"].([]any) {
		r := v.(map[string]any)
		if strings.TrimSpace(r["name"].(string)) == "" {
			t.Fatal("record missing")
		}
	}
}

// Only disposable migration fixtures use this schema reconstruction. Runtime
// and user environments never remove enterprise history.
func removeEnterprise304Extension(f *fixture) {
	f.exec(`DELETE FROM access.subject_grant WHERE action LIKE 'enterprise.%';
 DROP TABLE organization.enterprise_profile,organization.enterprise_terminal,organization.enterprise_history,organization.enterprise_shape,organization.enterprise_dictionary CASCADE;
 DROP FUNCTION organization.enterprise_stamp_head(),organization.enterprise_append_guard(),organization.enterprise_head_guard(),organization.enterprise_closure_guard(),organization.enterprise_terminal_guard();
 DELETE FROM public.goose_db_version WHERE version_id=14;`)
}

func TestEnterprise304OrderedCoverageLocks(t *testing.T) {
	for _, enterpriseFirst := range []bool{true, false} {
		t.Run(fmt.Sprint(enterpriseFirst), func(t *testing.T) {
			f := fresh(t)
			a := f.login("admin", f.password)
			b := f.login("admin", f.password)
			path, d := enterprise304(a)
			blocker, e := f.admin.Begin(f.ctx)
			must(t, e)
			defer blocker.Rollback(f.ctx)
			_, e = blocker.Exec(f.ctx, "SELECT enterprise_id FROM organization.enterprise WHERE instance_id=$1 FOR UPDATE", f.instance)
			must(t, e)
			body := correct304(d, nil)
			delete(body, "changes")
			body["newEffectiveDate"] = "2021-01-01"
			type reply struct {
				status int
				err    error
			}
			en, bu := make(chan reply, 1), make(chan reply, 1)
			startEnterprise := func() {
				go func() {
					status, _, err := a.response("POST", path+"/records/"+record304(d)["recordId"].(string)+"/correct", body, foundation.NewID())
					en <- reply{status, err}
				}()
			}
			startBU := func() {
				go func() {
					status, _, err := b.response("POST", "/api/v1/business-units", createBody("ORDER304", "2020-01-01"), foundation.NewID())
					bu <- reply{status, err}
				}()
			}
			waitLock := func(suffix string) {
				t.Helper()
				deadline := time.Now().Add(time.Second)
				for {
					var n int
					must(t, f.admin.QueryRow(f.ctx, `SELECT count(*) FROM pg_stat_activity WHERE wait_event_type='Lock' AND query LIKE '%organization.enterprise%' AND query LIKE $1`, "%FOR "+suffix+"%").Scan(&n))
					if n > 0 {
						return
					}
					if time.Now().After(deadline) {
						t.Fatal("expected queued enterprise guard", suffix)
					}
					time.Sleep(5 * time.Millisecond)
				}
			}
			if enterpriseFirst {
				startEnterprise()
				waitLock("UPDATE")
				startBU()
				waitLock("SHARE")
			} else {
				startBU()
				waitLock("SHARE")
				startEnterprise()
				waitLock("UPDATE")
			}
			must(t, blocker.Commit(f.ctx))
			er, br := <-en, <-bu
			must(t, er.err)
			must(t, br.err)
			if enterpriseFirst && (er.status != 200 || br.status != 400) || !enterpriseFirst && (er.status != 409 || br.status != 201) {
				t.Fatalf("ordered coverage %t enterprise=%d BU=%d", enterpriseFirst, er.status, br.status)
			}
		})
	}
}

func TestEnterprise304AtomicPublicationFailure(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	path, head := enterprise304(b)
	row := record304(head)
	before := f.scalar("SELECT sum(cardinality(source_refs)) FROM organization.enterprise_history")
	f.exec(`CREATE FUNCTION public.enterprise304_fail() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION '304 injected publication failure' USING ERRCODE='57014'; END $$;
 CREATE TRIGGER enterprise304_fail BEFORE UPDATE ON organization.enterprise_profile FOR EACH ROW EXECUTE FUNCTION public.enterprise304_fail();`)
	key := foundation.NewID()
	body := correct304(head, map[string]any{"name": "不得部分保存"})
	route := path + "/records/" + row["recordId"].(string) + "/correct"
	b.call("POST", route, body, key, 503)
	f.exec(`DROP TRIGGER enterprise304_fail ON organization.enterprise_profile; DROP FUNCTION public.enterprise304_fail();`)
	_, after := enterprise304(b)
	if !reflect.DeepEqual(head["timeline"], after["timeline"]) || before != f.scalar("SELECT sum(cardinality(source_refs)) FROM organization.enterprise_history") {
		t.Fatal("failed publication left partial source")
	}
	if b.call("POST", "/api/v1/enterprise/submissions/"+key+"/resolve", nil, "", 200)["outcome"] != "not_applied" {
		t.Fatal("rolled-back request was not closed")
	}
	b.call("POST", route, body, key, 409)
	b.call("POST", route, body, foundation.NewID(), 200)
}

func TestEnterprise304ClassificationAndDateFacets(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	path, head := enterprise304(b)
	b.call("POST", path+"/records", addition304(head, "2021-01-01", "CHANGE", "待转换资料"), foundation.NewID(), 201)
	_, head = enterprise304(b)
	rid := record304(head)["recordId"].(string)
	route := path + "/records/" + rid + "/correct"
	input := func(d map[string]any, kind, reason, date string) map[string]any {
		v := correct304(d, nil)
		delete(v, "changes")
		v["classification"] = map[string]any{"operationType": kind, "operationReason": reason}
		if date != "" {
			v["newEffectiveDate"] = date
		}
		return v
	}
	oldSource := head["headSourceVersion"].(string)
	b.call("POST", route, input(head, "DEACTIVATE", "BUSINESS_CLOSURE", "2021-06-01"), foundation.NewID(), 200)
	_, head = enterprise304(b)
	if record304(head)["state"] != "inactive" || record304(head)["name"] != "本地验证企业" {
		t.Fatal("deactivation classification did not inherit predecessor")
	}
	f.exec("UPDATE access.subject_grant SET write_fields=array_remove(write_fields,'operationType') WHERE principal_id=$1 AND action='enterprise.record.reclassify'", b.subject)
	b.call("POST", route, input(head, "CHANGE", "NAME_ADJUSTMENT", ""), foundation.NewID(), 403)
	f.exec("UPDATE access.subject_grant SET write_fields=array_append(write_fields,'operationType') WHERE principal_id=$1 AND action='enterprise.record.reclassify'", b.subject)
	b.call("POST", route, input(head, "CHANGE", "NAME_ADJUSTMENT", "2021-01-01"), foundation.NewID(), 200)
	_, head = enterprise304(b)
	if record304(head)["state"] != "active" {
		t.Fatal("record classification did not restore active state")
	}
	b.call("POST", route, input(head, "REACTIVATE", "BUSINESS_RESUMPTION", ""), foundation.NewID(), 409)
	old := b.call("GET", path+"?sourceVersion="+oldSource, nil, "", 200)
	if record304(old)["name"] != "待转换资料" {
		t.Fatal("classification altered old source")
	}
	f.exec("UPDATE access.subject_grant SET write_fields=ARRAY[]::text[] WHERE principal_id=$1 AND action='enterprise.record.redate'", b.subject)
	body := correct304(head, nil)
	delete(body, "changes")
	body["newEffectiveDate"] = "2021-07-01"
	b.call("POST", route, body, foundation.NewID(), 403)
	b.call("POST", route, correct304(head, map[string]any{"name": "仍可更正资料"}), foundation.NewID(), 200)
}

func TestEnterprise304FutureFirstAndClock(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	path, head := enterprise304(b)
	rid := record304(head)["recordId"].(string)
	now := time.Date(2098, 12, 31, 15, 59, 59, 0, time.UTC)
	restore := bootstrap.SetIntegrationClock(f.runtime, func() time.Time { return now })
	defer restore()
	body := correct304(head, nil)
	delete(body, "changes")
	body["newEffectiveDate"] = "2099-01-01"
	preview := map[string]any{"action": "CORRECT", "recordId": rid}
	for k, v := range body {
		preview[k] = v
	}
	p := b.call("POST", path+"/preview", preview, "", 200)
	if p["currentStatus"] != "not_effective" || p["newNoFactRange"].(map[string]any)["endExclusive"] != "2099-01-01" {
		t.Fatal("future first preview")
	}
	b.call("POST", path+"/records/"+rid+"/correct", body, foundation.NewID(), 200)
	_, head = enterprise304(b)
	if head["status"] != "not_effective" || head["selectedRecordIsFuture"] != true {
		t.Fatal("future first projection")
	}
	b.call("POST", "/api/v1/business-units", createBody("BEFORE304", "2098-12-31"), foundation.NewID(), 400)
	now = now.Add(time.Second)
	_, head = enterprise304(b)
	if head["status"] != "active" || head["currentDate"] != "2099-01-01" || head["selectedRecordIsFuture"] != false {
		t.Fatal("enterprise did not cross Shanghai day")
	}
	b.call("POST", "/api/v1/business-units", createBody("FROM304", "2099-01-01"), foundation.NewID(), 201)
	b.call("POST", path+"/records", addition304(head, "2099-01-02", "CHANGE", "跨日企业名称"), foundation.NewID(), 201)
	for _, url := range []string{"/api/v1/organization/context", "/api/v1/departments/context", "/api/v1/locations/context", "/api/v1/legal-entities/context"} {
		v := b.call("GET", url, nil, "", 200)
		if v["enterprise"].(map[string]any)["name"] == "跨日企业名称" {
			t.Fatal("future enterprise name replaced today", url)
		}
	}
	now = now.Add(24 * time.Hour)
	for _, url := range []string{"/api/v1/organization/context", "/api/v1/departments/context", "/api/v1/locations/context", "/api/v1/legal-entities/context"} {
		v := b.call("GET", url, nil, "", 200)
		if v["enterprise"].(map[string]any)["name"] != "跨日企业名称" || v["currentDate"] != "2099-01-02" {
			t.Fatal("enterprise name did not use captured day", url)
		}
	}
}

func TestEnterprise304LocationFirstDateLocks(t *testing.T) {
	for _, enterpriseFirst := range []bool{true, false} {
		t.Run(fmt.Sprint(enterpriseFirst), func(t *testing.T) {
			f := fresh(t)
			must(t, f.runtime.InitializeCountryCatalog(f.ctx, access.LoginInput{Login: "admin", Password: f.password}))
			country := geoCountry203(t, f, "MO")
			a := f.login("admin", f.password)
			b := f.login("admin", f.password)
			path, d := enterprise304(a)
			blocker, e := f.admin.Begin(f.ctx)
			must(t, e)
			defer blocker.Rollback(f.ctx)
			_, e = blocker.Exec(f.ctx, "SELECT enterprise_id FROM organization.enterprise WHERE instance_id=$1 FOR UPDATE", f.instance)
			must(t, e)
			body := correct304(d, nil)
			delete(body, "changes")
			body["newEffectiveDate"] = "2021-01-01"
			type reply struct {
				status int
				err    error
			}
			en, bu := make(chan reply, 1), make(chan reply, 1)
			startEnterprise := func() {
				go func() {
					status, _, err := a.response("POST", path+"/records/"+record304(d)["recordId"].(string)+"/correct", body, foundation.NewID())
					en <- reply{status, err}
				}()
			}
			startBU := func() {
				go func() {
					status, _, err := b.response("POST", "/api/v1/locations", map[string]any{"code": "ORDER304", "name": "地点", "effectiveDate": "2020-01-01", "countryRegionId": country, "addressLine": "一号", "operationType": "CREATE", "operationReason": "NEW_LOCATION"}, foundation.NewID())
					bu <- reply{status, err}
				}()
			}
			waitLock := func(suffix string) {
				t.Helper()
				deadline := time.Now().Add(time.Second)
				for {
					var n int
					must(t, f.admin.QueryRow(f.ctx, `SELECT count(*) FROM pg_stat_activity WHERE wait_event_type='Lock' AND query LIKE '%organization.enterprise%' AND query LIKE $1`, "%FOR "+suffix+"%").Scan(&n))
					if n > 0 {
						return
					}
					if time.Now().After(deadline) {
						t.Fatal("expected queued enterprise guard", suffix)
					}
					time.Sleep(5 * time.Millisecond)
				}
			}
			if enterpriseFirst {
				startEnterprise()
				waitLock("UPDATE")
				startBU()
				waitLock("SHARE")
			} else {
				startBU()
				waitLock("SHARE")
				startEnterprise()
				waitLock("UPDATE")
			}
			must(t, blocker.Commit(f.ctx))
			er, br := <-en, <-bu
			must(t, er.err)
			must(t, br.err)
			if enterpriseFirst && (er.status != 200 || br.status != 400) || !enterpriseFirst && (er.status != 409 || br.status != 201) {
				t.Fatalf("ordered coverage %t enterprise=%d location=%d", enterpriseFirst, er.status, br.status)
			}
		})
	}
}

func TestEnterprise304LegalEntityFirstDateLocks(t *testing.T) {
	for _, enterpriseFirst := range []bool{true, false} {
		t.Run(fmt.Sprint(enterpriseFirst), func(t *testing.T) {
			f := fresh(t)
			must(t, f.runtime.InitializeCountryCatalog(f.ctx, access.LoginInput{Login: "admin", Password: f.password}))
			country := geoCountry203(t, f, "MO")
			a := f.login("admin", f.password)
			b := f.login("admin", f.password)
			path, d := enterprise304(a)
			blocker, e := f.admin.Begin(f.ctx)
			must(t, e)
			defer blocker.Rollback(f.ctx)
			_, e = blocker.Exec(f.ctx, "SELECT enterprise_id FROM organization.enterprise WHERE instance_id=$1 FOR UPDATE", f.instance)
			must(t, e)
			body := correct304(d, nil)
			delete(body, "changes")
			body["newEffectiveDate"] = "2021-01-01"
			type reply struct {
				status int
				err    error
			}
			en, bu := make(chan reply, 1), make(chan reply, 1)
			startEnterprise := func() {
				go func() {
					status, _, err := a.response("POST", path+"/records/"+record304(d)["recordId"].(string)+"/correct", body, foundation.NewID())
					en <- reply{status, err}
				}()
			}
			startBU := func() {
				go func() {
					status, _, err := b.response("POST", "/api/v1/legal-entities", map[string]any{"code": "ORDER304", "name": "法律实体", "effectiveDate": "2020-01-01", "countryRegionId": country, "subjectType": "legal_person", "operationType": "CREATE", "operationReason": "NEW_LEGAL_ENTITY"}, foundation.NewID())
					bu <- reply{status, err}
				}()
			}
			waitLock := func(suffix string) {
				t.Helper()
				deadline := time.Now().Add(time.Second)
				for {
					var n int
					must(t, f.admin.QueryRow(f.ctx, `SELECT count(*) FROM pg_stat_activity WHERE wait_event_type='Lock' AND query LIKE '%organization.enterprise%' AND query LIKE $1`, "%FOR "+suffix+"%").Scan(&n))
					if n > 0 {
						return
					}
					if time.Now().After(deadline) {
						t.Fatal("expected queued enterprise guard", suffix)
					}
					time.Sleep(5 * time.Millisecond)
				}
			}
			if enterpriseFirst {
				startEnterprise()
				waitLock("UPDATE")
				startBU()
				waitLock("SHARE")
			} else {
				startBU()
				waitLock("SHARE")
				startEnterprise()
				waitLock("UPDATE")
			}
			must(t, blocker.Commit(f.ctx))
			er, br := <-en, <-bu
			must(t, er.err)
			must(t, br.err)
			if enterpriseFirst && (er.status != 200 || br.status != 400) || !enterpriseFirst && (er.status != 409 || br.status != 201) {
				t.Fatalf("ordered coverage %t enterprise=%d legalentity=%d", enterpriseFirst, er.status, br.status)
			}
		})
	}
}

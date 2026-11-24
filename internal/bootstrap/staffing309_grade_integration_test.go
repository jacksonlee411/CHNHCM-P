//go:build integration

package bootstrap_test

import (
	"chnhcm.local/chnhcm/internal/bootstrap"
	staffapp "chnhcm.local/chnhcm/internal/contexts/staffing/application"
	staffdb "chnhcm.local/chnhcm/internal/contexts/staffing/infra/postgres"
	"chnhcm.local/chnhcm/internal/foundation"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestStaffing309DefaultFloorAndInstanceIsolation(t *testing.T) {
	f := freshWithEnterpriseStart(t, "1984-01-01")
	b := f.login("admin", f.password)
	body := map[string]any{"code": "ISOLATED309", "name": "职级", "operationType": "CREATE", "operationReason": "NEW_GRADE"}
	if b.call("POST", "/api/v1/grades", body, foundation.NewID(), 409)["field"] != "effectiveDate" {
		t.Fatal("default date floor did not locate effectiveDate")
	}
	body["effectiveDate"] = "1984-01-01"
	body["enterpriseId"] = foundation.NewID()
	b.call("POST", "/api/v1/grades", body, foundation.NewID(), 400)
	delete(body, "enterpriseId")
	created := b.call("POST", "/api/v1/grades", body, foundation.NewID(), 201)
	// Two initialized instances share this physical DB, so the isolation proof
	// cannot pass merely because the other database has no such identity.
	other := &fixture{t: t, ctx: f.ctx, admin: f.admin, instance: foundation.NewID(), password: f.password}
	must(t, bootstrap.Initialize(f.ctx, f.admin, bootstrap.InitInput{Instance: other.instance, EnterpriseCode: "OTHER", EnterpriseName: "另一企业", Start: "1984-01-01", Login: "admin", Password: other.password}))
	var err error
	other.runtime, err = bootstrap.Open(f.ctx, f.runtime.Pool.Config().ConnString(), other.instance)
	must(t, err)
	t.Cleanup(other.runtime.Close)
	api := &bootstrap.API{Runtime: other.runtime, InsecureLocal: true}
	other.server = httptest.NewUnstartedServer(nil)
	api.Origin = "http://" + other.server.Listener.Addr().String()
	other.server.Config.Handler = api.Handler()
	other.server.Start()
	t.Cleanup(other.server.Close)
	foreign := other.login("admin", other.password)
	id := created["objectId"].(string)
	foreign.call("GET", "/api/v1/grades/"+id, nil, "", 404)
	foreign.call("GET", "/api/v1/grades/"+id+"?sourceVersion="+created["sourceVersion"].(string), nil, "", 404)
	if foreign.call("GET", "/api/v1/grades", nil, "", 200)["totalCount"] != float64(0) {
		t.Fatal("other enterprise Grade leaked")
	}
	foreignGrade := foreign.call("POST", "/api/v1/grades", body, foundation.NewID(), 201)
	if b.call("GET", "/api/v1/grades", nil, "", 200)["totalCount"] != float64(1) {
		t.Fatal("other enterprise code namespace leaked")
	}
	// G2 identities and member lookups use the same physical two-instance proof.
	ladder := map[string]any{"code": "ISOLATED309", "name": "独立阶梯", "operationType": "CREATE", "operationReason": "NEW_GRADE_LADDER", "gradeIds": []string{id}}
	if b.call("POST", "/api/v1/grade-ladders", ladder, foundation.NewID(), 409)["field"] != "effectiveDate" {
		t.Fatal("Ladder default date floor did not locate effectiveDate")
	}
	ladder["effectiveDate"] = "1984-01-01"
	localLadder := b.call("POST", "/api/v1/grade-ladders", ladder, foundation.NewID(), 201)
	b.call("POST", "/api/v1/grade-ladders", ladder, foundation.NewID(), 409)
	foreign.call("POST", "/api/v1/grade-ladders", ladder, foundation.NewID(), 409)
	lid := localLadder["objectId"].(string)
	foreign.call("GET", "/api/v1/grade-ladders/"+lid, nil, "", 404)
	foreign.call("GET", "/api/v1/grade-ladders/"+lid+"?sourceVersion="+localLadder["sourceVersion"].(string), nil, "", 404)
	ladder["gradeIds"] = []string{foreignGrade["objectId"].(string)}
	foreign.call("POST", "/api/v1/grade-ladders", ladder, foundation.NewID(), 201)
	if b.call("GET", "/api/v1/grade-ladders", nil, "", 200)["totalCount"] != float64(1) || foreign.call("GET", "/api/v1/grade-ladders", nil, "", 200)["totalCount"] != float64(1) {
		t.Fatal("Ladder identity or code namespace crossed instances")
	}
	family := foreign.call("POST", "/api/v1/job-families", map[string]any{"code": "ISOLATED310-F", "name": "系列", "effectiveDate": "1984-01-01", "operationType": "CREATE", "operationReason": "NEW_JOB_FAMILY"}, foundation.NewID(), 201)
	job := map[string]any{"code": "ISOLATED310-J", "jobFunctionId": foreign.functionForJobs("1984-01-01"), "name": "职务", "effectiveDate": "1984-01-01", "operationType": "CREATE", "operationReason": "NEW_JOB", "jobFamilyId": family["objectId"], "gradeIds": []string{id}}
	foreign.call("POST", "/api/v1/jobs", job, foundation.NewID(), 409)
	job["gradeIds"] = []string{foreignGrade["objectId"].(string)}
	foreign.call("POST", "/api/v1/jobs", job, foundation.NewID(), 201)
}

func TestStaffing309IdentityShapeAndRecovery(t *testing.T) {
	h := staffingSetup307(t)
	b, f := h.b, h.f
	before := businessSnapshot(t, f)
	key := foundation.NewID()
	body := map[string]any{"code": " g-01 ", "name": " 初级 ", "nameEn": "e\u0301", "description": "说明\r\n第二行", "operationType": "CREATE", "operationReason": "NEW_GRADE"}
	created := b.call("POST", "/api/v1/grades", body, key, 201)
	id := created["objectId"].(string)
	v := h.detail("grade", id, "")
	r := rec307(v)
	if v["code"] != "G-01" || r["name"] != "初级" || r["nameEn"] != "é" || r["description"] != "说明\n第二行" || r["effectiveDate"] != "1900-01-01" {
		t.Fatalf("normalization/default differs: %v", v)
	}
	for _, field := range []string{"jobFamilyId", "responsibilities", "requirements", "gradeIds", "rank", "salary"} {
		if _, present := r[field]; present {
			t.Fatalf("Grade leaked %s", field)
		}
		bad := map[string]any{}
		for k, v := range body {
			bad[k] = v
		}
		bad[field] = "invalid"
		b.call("POST", "/api/v1/grades", bad, foundation.NewID(), 400)
		h.correct("grade", v, map[string]any{field: "invalid"}, nil, 400)
	}
	b.call("POST", "/api/v1/grades", body, key, 201)
	b.call("POST", "/api/v1/grades", body, foundation.NewID(), 409)
	body["name"] = "不同摘要"
	b.call("POST", "/api/v1/grades", body, key, 409)
	// Same code and same submission identity remain legal in a different type.
	body["operationReason"] = "NEW_JOB_FAMILY"
	b.call("POST", "/api/v1/job-families", body, key, 201)
	got := b.call("POST", "/api/v1/grades/submissions/"+key+"/resolve", nil, "", 200)
	if got["kind"] != "grade" || got["objectId"] != id {
		t.Fatal("type namespace crossed")
	}
	unknown := foundation.NewID()
	b.call("POST", "/api/v1/grades/submissions/"+unknown+"/resolve", nil, "", 200)
	body["operationReason"] = "NEW_GRADE"
	if b.call("POST", "/api/v1/grades", body, unknown, 200)["outcome"] != "not_applied" {
		t.Fatal("closed submit reopened")
	}
	h.correct("grade", v, map[string]any{"name": "并发胜者"}, nil, 200)
	h.correct("grade", v, map[string]any{"name": "陈旧写入"}, nil, 409)
	v = h.refresh("grade", v)
	out := h.correct("grade", v, map[string]any{"name": "并发胜者"}, nil, 200)
	if out["outcome"] != "no_change" {
		t.Fatal("unchanged Grade created history")
	}
	if f.scalar("SELECT count(*) FROM staffing.grade_source") != 2 || f.scalar("SELECT count(*) FROM staffing.grade_outbox") != 2 {
		t.Fatal("replay/rejection wrote partial facts")
	}
	for _, suffix := range []string{"_source", "_revision", "_terminal", "_outbox"} {
		if _, err := f.admin.Exec(f.ctx, "DELETE FROM staffing.grade"+suffix); err == nil {
			t.Fatalf("mutable Grade fact: %s", suffix)
		}
	}
	if !reflect.DeepEqual(before, businessSnapshot(t, f)) {
		t.Fatal("Grade changed M02 facts")
	}
	old := b.call("GET", "/api/v1/grades/"+id+"?sourceVersion="+created["sourceVersion"].(string), nil, "", 200)
	if rec307(old)["name"] != "初级" || old["readOnly"] != true {
		t.Fatal("immutable Grade source changed")
	}
	restartTimelineReceipts(t, f, b, []string{key}, "grade")
}

func TestStaffing309PairAndFuture(t *testing.T) {
	h := staffingSetup307(t)
	store := h.f.runtime.Staffing.(staffapp.Service).Store.(*staffdb.Store)
	store.Clock = func() time.Time { return time.Date(2026, 11, 17, 4, 0, 0, 0, time.UTC) }
	v := h.create("grade", "PAIR309", "")
	id := v["id"].(string)
	first := rec307(v)["recordId"].(string)
	h.add("grade", v, "ADD", "2035-01-01", map[string]any{"name": "未来独立资料"}, 200)
	v = h.detail("grade", id, first)
	body := map[string]any{"expectedSourceVersion": v["headSourceVersion"], "anchorRecordId": first, "deactivationDate": "2010-01-01", "reactivationDate": "2020-01-01", "deactivationReason": "GRADE_CLOSURE", "reactivationReason": "GRADE_RESUMPTION", "changes": map[string]any{"name": "恢复资料"}}
	h.b.call("POST", "/api/v1/grades/"+id+"/lifecycle-pairs/insert", body, foundation.NewID(), 200)
	v = h.detail("grade", id, "")
	if rec307(v)["name"] != "恢复资料" || len(v["timeline"].([]any)) != 4 {
		t.Fatal("pair insertion failed")
	}
	pair := v["pairContext"].(map[string]any)
	stop, start := pair["deactivation"].(map[string]any), pair["reactivation"].(map[string]any)
	pairInput := func(v map[string]any) map[string]any {
		p := v["pairContext"].(map[string]any)
		a, b := p["deactivation"].(map[string]any), p["reactivation"].(map[string]any)
		return map[string]any{"expectedSourceVersion": v["headSourceVersion"], "deactivationRecordId": a["recordId"], "deactivationRevision": a["revision"], "reactivationRecordId": b["recordId"], "reactivationRevision": b["revision"]}
	}
	body = pairInput(v)
	body["deactivationDate"] = "2011-01-01"
	body["reactivationDate"] = "2021-01-01"
	body["deactivationReason"] = "GRADE_CLOSURE"
	body["reactivationReason"] = "GRADE_RESUMPTION"
	body["changes"] = map[string]any{"description": "恢复说明"}
	h.b.call("POST", "/api/v1/grades/"+id+"/lifecycle-pairs/maintain", body, foundation.NewID(), 200)
	v = h.detail("grade", id, start["recordId"].(string))
	if rec307(v)["description"] != "恢复说明" || rec307(v)["effectiveDate"] != "2021-01-01" {
		t.Fatal("pair maintenance failed")
	}
	// An isolated stop deletion would invalidate the remaining REACTIVATE.
	sv := h.detail("grade", id, stop["recordId"].(string))
	h.b.call("POST", "/api/v1/grades/"+id+"/records/"+stop["recordId"].(string)+"/delete", map[string]any{"expectedSourceVersion": sv["headSourceVersion"], "expectedRecordRevision": rec307(sv)["revision"]}, foundation.NewID(), 400)
	h.b.call("POST", "/api/v1/grades/"+id+"/lifecycle-pairs/delete", pairInput(v), foundation.NewID(), 200)
	v = h.detail("grade", id, "")
	if len(v["timeline"].([]any)) != 2 || rec307(v)["name"] != "资料 PAIR309" {
		t.Fatal("pair deletion did not restore inherited interval")
	}
	last := v["timeline"].([]any)[1].(map[string]any)
	if last["name"] != "未来独立资料" || last["effectiveDate"] != "2035-01-01" {
		t.Fatal("future snapshot changed")
	}
	future := h.create("grade", "FUTURE309", "")
	h.correct("grade", future, nil, map[string]any{"newEffectiveDate": "2035-02-01"}, 200)
	future = h.refresh("grade", future)
	if future["status"] != "not_effective" {
		t.Fatal("future first is not current active")
	}
	store.Clock = func() time.Time { return time.Date(2035, 1, 31, 15, 59, 59, 0, time.UTC) }
	if h.refresh("grade", future)["status"] != "not_effective" {
		t.Fatal("future first activated before Beijing midnight")
	}
	store.Clock = func() time.Time { return time.Date(2035, 1, 31, 16, 0, 0, 0, time.UTC) }
	active := h.refresh("grade", future)
	if active["status"] != "active" || active["headSourceVersion"] != future["headSourceVersion"] {
		t.Fatal("Beijing effective-day boundary must activate without a new source")
	}
}

func TestStaffing309FieldRightsSortAndCSV(t *testing.T) {
	h := staffingSetup307(t)
	f, b := h.f, h.b
	a := h.create("grade", "CSV309-A", "")
	c := h.create("grade", "CSV309-B", "")
	h.correct("grade", a, map[string]any{"description": "=1+1"}, nil, 200)
	h.correct("grade", c, map[string]any{"description": "原文,含逗号\n第二行\"引号\""}, nil, 200)
	query := map[string]any{"q": "CSV309-"}
	in := map[string]any{"draftId": foundation.NewID(), "query": query}
	draft := b.call("POST", "/api/v1/grades/selection/start", in, "", 200)
	h.create("grade", "CSV309-C", "")
	in["expectedRevision"] = draft["revision"]
	in["sort"] = "-code"
	csv307(t, b, "/api/v1/grades/export", in, 400)
	in["sort"] = "status"
	csv307(t, b, "/api/v1/grades/export", in, 403)
	delete(in, "sort")
	rows := csv307(t, b, "/api/v1/grades/export", in, 200)
	want := [][]string{{"编码", "名称", "英文名称", "说明", "状态", "生效日期"}, {"CSV309-A", "资料 CSV309-A", "Definition CSV309-A", "'=1+1", "启用", "2000-01-01"}, {"CSV309-B", "资料 CSV309-B", "Definition CSV309-B", "原文,含逗号\n第二行\"引号\"", "启用", "2000-01-01"}}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("CSV values differ: %v", rows)
	}
	in["excludedIds"] = []string{c["id"].(string)}
	draft = b.call("POST", "/api/v1/grades/selection/resolve", in, "", 200)
	if draft["count"] != float64(1) {
		t.Fatal("selection exclusion lost")
	}
	in["expectedRevision"] = draft["revision"]
	delete(in, "excludedIds")
	b.call("POST", "/api/v1/grades/selection/cancel", in, "", 200)
	csv307(t, b, "/api/v1/grades/export", in, 409)
	in = map[string]any{"draftId": foundation.NewID(), "query": query}
	draft = b.call("POST", "/api/v1/grades/selection/start", in, "", 200)
	in["expectedRevision"] = draft["revision"]
	store := f.runtime.Staffing.(staffapp.Service).Store.(*staffdb.Store)
	oldClock := store.Clock
	store.Clock = func() time.Time { return time.Now().AddDate(0, 0, 1) }
	csv307(t, b, "/api/v1/grades/export", in, 409)
	store.Clock = oldClock
	in = map[string]any{"draftId": foundation.NewID(), "query": query}
	draft = b.call("POST", "/api/v1/grades/selection/start", in, "", 200)
	in["expectedRevision"] = draft["revision"]
	for _, sort := range []string{"code", "name", "-name", "nameEn", "-nameEn"} {
		b.call("GET", "/api/v1/grades?sort="+sort, nil, "", 200)
	}
	b.call("GET", "/api/v1/grades?sort=-code", nil, "", 400)
	for _, sort := range []string{"status", "effectiveDate", "description"} {
		b.call("GET", "/api/v1/grades?sort="+sort, nil, "", 403)
	}
	b.call("GET", "/api/v1/grades?jobFamilyId="+foundation.NewID(), nil, "", 400)
	reader := f.login("reader", f.readPassword)
	reader.call("GET", "/api/v1/grades/"+a["id"].(string), nil, "", 200)
	reader.call("POST", "/api/v1/grades", map[string]any{"code": "DENIED", "name": "拒绝", "operationType": "CREATE", "operationReason": "NEW_GRADE"}, foundation.NewID(), 403)
	f.exec(`UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'nameEn') WHERE action LIKE 'grade.%' AND principal_id=$1`, b.subject)
	csv307(t, b, "/api/v1/grades/export", in, 403)
	v := h.refresh("grade", a)
	if _, ok := rec307(v)["nameEn"]; ok {
		t.Fatal("hidden field leaked")
	}
	b.call("GET", "/api/v1/grades?sort=nameEn", nil, "", 403)
	b.call("GET", "/api/v1/grades?filterField=nameEn&filterOperator=contains&filterValue=Definition", nil, "", 403)
	f.exec(`UPDATE access.subject_grant SET write_fields=array_remove(write_fields,'description') WHERE action='grade.record.correct' AND principal_id=$1`, b.subject)
	h.correct("grade", v, map[string]any{"description": "无权更改"}, nil, 403)
	f.exec(`DELETE FROM access.subject_grant WHERE action='grade.history.read' AND principal_id=$1`, b.subject)
	h.refresh("grade", a)
	b.call("GET", "/api/v1/grades/"+a["id"].(string)+"?sourceVersion="+a["headSourceVersion"].(string), nil, "", 403)
	b.call("GET", "/api/v1/grades/"+a["id"].(string)+"/changes", nil, "", 403)
	if strings.Contains(rec307(h.refresh("grade", a))["description"].(string), "无权") {
		t.Fatal("forbidden write committed")
	}
}

func TestStaffing309DatedMaintenance(t *testing.T) {
	s := staffingSetup307(t)

	for _, k := range []string{"grade"} {
		t.Run(k, func(t *testing.T) {
			v := s.create(k, "DATED", "")
			id := v["id"].(string)
			first, oldsource := rec307(v)["recordId"].(string), v["headSourceVersion"].(string)
			s.add(k, v, "ADD", "2010-01-01", map[string]any{"name": "独立后项"}, 200)
			v = s.detail(k, id, first)
			s.correct(k, v, map[string]any{"name": "更正首项", "nameEn": nil}, nil, 200)
			v = s.detail(k, id, "")
			if rec307(v)["name"] != "独立后项" {
				t.Fatal("successor changed")
			}
			v = s.detail(k, id, first)
			s.add(k, v, "ADD", "2005-01-01", map[string]any{"name": "插入中间"}, 200)
			v = s.detail(k, id, "")
			middle := v["timeline"].([]any)[1].(map[string]any)
			v = s.detail(k, id, middle["recordId"].(string))
			s.correct(k, v, nil, map[string]any{"newEffectiveDate": "2006-01-01"}, 200)
			v = s.detail(k, id, middle["recordId"].(string))
			s.b.call("POST", base307(k)+"/"+id+"/records/"+middle["recordId"].(string)+"/delete", map[string]any{"expectedSourceVersion": v["headSourceVersion"], "expectedRecordRevision": rec307(v)["revision"]}, foundation.NewID(), 200)
			v = s.detail(k, id, "")
			s.add(k, v, "DEACTIVATE", "2015-01-01", nil, 200)
			v = s.detail(k, id, "")
			s.add(k, v, "REACTIVATE", "2018-01-01", map[string]any{"name": "恢复后独立资料"}, 200)
			v = s.detail(k, id, "")
			if len(v["timeline"].([]any)) != 4 || rec307(v)["name"] != "恢复后独立资料" {
				t.Fatal("lifecycle failed")
			}
			old := s.b.call("GET", base307(k)+"/"+id+"?sourceVersion="+oldsource, nil, "", 200)
			if rec307(old)["name"] != "资料 DATED" || len(old["timeline"].([]any)) != 1 {
				t.Fatal("old source rewritten")
			}
			v = s.detail(k, id, first)
			s.b.call("POST", base307(k)+"/"+id+"/records/"+first+"/delete", map[string]any{"expectedSourceVersion": v["headSourceVersion"], "expectedRecordRevision": rec307(v)["revision"]}, foundation.NewID(), 400)
		})
	}
}

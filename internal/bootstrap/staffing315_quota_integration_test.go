//go:build integration

package bootstrap_test

import (
	"context"
	"maps"
	"net/http"
	"net/http/httptest"
	"testing"

	"chnhcm.local/chnhcm/internal/bootstrap"
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	app "chnhcm.local/chnhcm/internal/contexts/staffing/application"
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	s "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

type quotaGate315 struct {
	app.QuotaStore
	ready, release chan struct{}
}
type quotaTx315 struct {
	app.QuotaTx
	gate *quotaGate315
}

func (g *quotaGate315) WithinQuota(ctx context.Context, c foundation.Credentials, write bool, fn func(app.QuotaTx) error) error {
	return g.QuotaStore.WithinQuota(ctx, c, write, func(u app.QuotaTx) error { return fn(quotaTx315{u, g}) })
}
func (u quotaTx315) SaveQuota(ctx context.Context, p d.QuotaPlan, c s.QuotaCommand, op access.OperatorSnapshot) error {
	close(u.gate.ready)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-u.gate.release:
	}
	return u.QuotaTx.SaveQuota(ctx, p, c, op)
}

func loseResponseAndRestart315(t *testing.T, f *fixture, b *browser, path string, input map[string]any, key string) {
	t.Helper()
	handler := f.server.Config.Handler
	old := f.server
	lost := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Set("Origin", old.URL)
		handler.ServeHTTP(httptest.NewRecorder(), r)
		conn, _, err := w.(http.Hijacker).Hijack()
		if err == nil {
			conn.Close()
		}
	}))
	lost.Start()
	f.server = lost
	_, _, err := b.response("POST", path, input, key)
	if err == nil {
		t.Fatal("应丢失 HTTP 响应")
	}
	f.server = old
	lost.Close()
	dsn := f.runtime.Pool.Config().ConnConfig.ConnString()
	f.server.Close()
	f.runtime.Close()
	f.runtime, err = bootstrap.Open(f.ctx, dsn, f.instance)
	must(t, err)
	t.Cleanup(f.runtime.Close)
	api := &bootstrap.API{Runtime: f.runtime, InsecureLocal: true}
	f.server = httptest.NewUnstartedServer(nil)
	api.Origin = "http://" + f.server.Listener.Addr().String()
	f.server.Config.Handler = api.Handler()
	f.server.Start()
	t.Cleanup(f.server.Close)
}

func TestStaffing315Quota(t *testing.T) {
	h := staffingSetup307(t)
	f := h.f
	b := h.b
	reader := f.login("reader", f.readPassword)
	must(t, f.runtime.InitializeCountryCatalog(f.ctx, access.LoginInput{Login: "admin", Password: f.password}))
	locations := location204{t, f, b, map[string]string{"MO": geoCountry203(t, f, "MO")}}
	loc := locations.create("L315", "MO", "", "2026-01-01")["id"]
	bu := b.call("POST", "/api/v1/business-units", createBody("B315", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	dep := b.call("POST", "/api/v1/departments", departmentCreateBody("D315", bu), foundation.NewID(), 201)["departmentId"]
	family := h.create("jobfamily", "F315", "")
	job := h.create("job", "J315", family["id"].(string))
	pos := func(code string) string {
		return b.call("POST", "/api/v1/positions", map[string]any{"code": code, "name": code, "departmentId": dep, "jobId": job["id"], "locationId": loc, "effectiveDate": "2026-01-01", "operationType": "CREATE", "operationReason": "NEW_POSITION"}, foundation.NewID(), 201)["objectId"].(string)
	}
	a, other := pos("ACCT315"), pos("MGR315")
	const base = "/api/v1/staffing-quotas"
	const jan = "2027-01-01"
	const apr = "2027-04-01"
	const jun = "2027-06-01"
	body := func(id, day string, n any) map[string]any {
		return map[string]any{"positionId": id, "effectiveDate": day, "quota": n, "reason": "年度核编"}
	}
	write := func(path string, input map[string]any) map[string]any {
		return b.call("POST", path, input, foundation.NewID(), 200)
	}
	detail := func(b *browser, id, day string) map[string]any {
		return b.call("GET", base+"/statistics/positions/"+id+"?businessAt="+day, nil, "", 200)
	}
	check := func(value any, want any) {
		t.Helper()
		if value != want {
			t.Fatalf("实际 %v，预期 %v", value, want)
		}
	}
	check(detail(reader, a, jan)["quotaState"], "unset")
	check(detail(reader, a, jan)["usageState"], "unknown")
	retiredKey := foundation.NewID()
	// The unregistered address returns ServeMux's plain-text 404.
	status, _, err := b.response("POST", base+"/"+a, map[string]any{"action": "SET", "effectiveDate": jan, "quantity": 2, "reason": "首次核编"}, retiredKey)
	if status != http.StatusNotFound {
		t.Fatalf("retired quota address: HTTP %d, want 404 (%v)", status, err)
	}
	check(f.scalar("SELECT count(*) FROM staffing.quota_terminal WHERE submission_id=$1", retiredKey), 0)
	check(detail(b, a, jan)["quotaState"], "unset")
	reader.call("POST", base, body(a, jan, 2), foundation.NewID(), 403)
	for _, bad := range []any{-1, 0.5} {
		b.call("POST", base, body(a, jan, bad), foundation.NewID(), 400)
	}
	key := foundation.NewID()
	// Let the real handler commit, then close the connection before writing its
	// response. Recovery uses a newly opened API runtime on the same database.
	loseResponseAndRestart315(t, f, b, base, body(a, jan, 2), key)
	resolve := func(b *browser, id string) map[string]any {
		return b.call("POST", base+"/submissions/"+id+"/resolve", nil, "", 200)
	}
	saved := resolve(b, key)
	check(saved["outcome"], "accepted")
	check(saved["positionId"], a)
	b.call("GET", base+"/positions/"+a+"?businessAt="+jan, nil, "", 400)
	b.call("GET", "/api/v1/positions?businessAt="+jan, nil, "", 400)
	ordinary := b.call("GET", base+"/positions/"+a, nil, "", 200)
	check(ordinary["recordId"], saved["selectedRecordId"])
	check(ordinary["selectedRecordIsFuture"], ordinary["businessAt"].(string) < jan)
	check(b.call("POST", base, body(a, jan, 2), key, 200)["sourceVersion"], saved["sourceVersion"])
	b.call("POST", base, body(a, jan, 3), key, 409)
	check(resolve(reader, key)["outcome"], "not_applied")
	late := foundation.NewID()
	check(resolve(b, late)["outcome"], "not_applied")
	check(b.call("POST", base, body(other, jan, 1), late, 200)["outcome"], "not_applied")
	path := base + "/" + saved["objectId"].(string)
	change := func(day string, n int, source any) map[string]any {
		return write(path+"/adjust", map[string]any{"expectedSourceVersion": source, "effectiveDate": day, "quota": n, "reason": "调编"})
	}
	april := change(apr, 1, saved["sourceVersion"])
	june := change(jun, 4, april["sourceVersion"])
	corr := map[string]any{"expectedSourceVersion": june["sourceVersion"], "expectedRecordRevision": april["selectedRecordRevision"], "quota": 3}
	target := path + "/records/" + april["selectedRecordId"].(string)
	check(write(target+"/correct/preview", corr)["endInclusive"], "2027-05-31")
	invalidReason := maps.Clone(corr)
	invalidReason["reason"] = ""
	b.call("POST", target+"/correct", invalidReason, foundation.NewID(), 400)
	after := write(target+"/correct", corr)
	beforeDelete := write(target+"/delete/preview", map[string]any{"expectedSourceVersion": after["sourceVersion"], "expectedRecordRevision": after["selectedRecordRevision"]})
	check(beforeDelete["effectiveDate"], apr)
	check(beforeDelete["endInclusive"], "2027-05-31")
	check(beforeDelete["quota"], float64(2))
	historyPath := base + "/history/" + saved["objectId"].(string) + "/changes"
	history := reader.call("GET", historyPath+"?filterField=action&filterOperator=is&filterValue=correct", nil, "", 200)
	operation := history["items"].([]any)[0].(map[string]any)
	check(operation["operator"].(map[string]any)["subjectId"], b.subject)
	diff := operation["items"].([]any)[0].(map[string]any)["diffs"].([]any)[0].(map[string]any)
	check(diff["field"], "quota")
	check(diff["before"].(map[string]any)["text"], "1")
	check(diff["after"].(map[string]any)["text"], "3")
	f.exec("UPDATE access.subject_grant SET read_fields=array_remove(array_remove(read_fields,'reason'),'audit') WHERE principal_id=$1 AND action='staffingquota.history.read'", reader.subject)
	restrictedLog := reader.call("GET", historyPath, nil, "", 200)["items"].([]any)[0].(map[string]any)
	check(restrictedLog["operator"], nil)
	check(restrictedLog["recordedAt"], nil)
	check(restrictedLog["items"].([]any)[0].(map[string]any)["afterContext"].(map[string]any)["reasonLabel"], nil)
	reader.call("GET", historyPath+"?filterField=actor&filterOperator=contains&filterValue=admin", nil, "", 403)
	for day, n := range map[string]float64{"2027-03-31": 2, apr: 3, jun: 4} {
		check(detail(reader, a, day)["quota"], n)
	}
	check(detail(reader, a, apr)["endInclusive"], "2027-05-31")
	check(detail(reader, a, apr+"&sourceVersion="+april["sourceVersion"].(string))["quota"], float64(1))
	check(after["selectedRecordId"], april["selectedRecordId"])
	manager := write(base, body(other, jan, 1))
	write(base+"/"+manager["objectId"].(string)+"/adjust", map[string]any{"expectedSourceVersion": manager["sourceVersion"], "effectiveDate": "2027-05-01", "quota": 2, "reason": "五月独立核定"})
	total := func(b *browser, day string, n int, scope string) {
		t.Helper()
		list := b.call("GET", base+"/statistics?businessAt="+day, nil, "", 200)
		groups := list["departments"].([]any)
		if len(groups) != 1 {
			t.Fatalf("部门汇总 %v", groups)
		}
		group := groups[0].(map[string]any)
		check(group["scope"], scope)
		check(group["total"], float64(n))
	}
	total(reader, apr, 4, "complete")
	f.exec("UPDATE access.subject_grant SET scope_kind='position',scope_id=$1 WHERE principal_id=$2 AND action LIKE 'staffingquota.%'", a, reader.subject)
	// Controlled initializer reruns must retain the reader's narrowed position scope.
	must(t, bootstrap.Initialize(f.ctx, f.admin, bootstrap.InitInput{Instance: f.instance, EnterpriseCode: "LOCAL", EnterpriseName: "本地验证企业", Start: "1900-01-01", Login: "reader", Password: f.readPassword, ReadOnly: true}))
	total(reader, apr, 3, "visible")
	reader.call("GET", base+"/statistics/positions/"+other+"?businessAt="+apr, nil, "", 404)
	f.exec("UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'quota') WHERE principal_id=$1 AND action LIKE 'staffingquota.%'", reader.subject)
	restricted := detail(reader, a, apr)
	for _, field := range []string{"quota", "quotaState", "sourceVersion", "records", "history"} {
		if _, ok := restricted[field]; ok {
			t.Fatal("受限字段", field)
		}
	}
	reader.call("GET", base+"/statistics/positions/"+a+"?businessAt="+apr+"&sourceVersion="+april["sourceVersion"].(string), nil, "", 404)
	group := reader.call("GET", base+"/statistics?businessAt="+apr, nil, "", 200)["departments"].([]any)[0].(map[string]any)
	if _, ok := group["total"]; ok {
		t.Fatal("受限合计")
	}
	f.exec("DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='staffingquota.history.read'", reader.subject)
	reader.call("GET", base+"/statistics/positions/"+a+"?businessAt="+apr+"&sourceVersion="+april["sourceVersion"].(string), nil, "", 404)
	f.exec("UPDATE access.subject_grant SET write_fields=ARRAY['effectiveDate'] WHERE principal_id=$1 AND action='staffingquota.redate'", b.subject)
	b.call("POST", target+"/redate", map[string]any{"expectedSourceVersion": after["sourceVersion"], "expectedRecordRevision": after["selectedRecordRevision"], "newEffectiveDate": "2027-04-20"}, foundation.NewID(), 403)
	check(f.scalar("SELECT count(*) FROM staffing.position_source"), 2)
	check(f.scalar("SELECT count(*) FROM staffing.job_source"), 1)
	definition := h.detail("position", a, "")
	first := rec307(definition)
	b.call("POST", "/api/v1/positions/"+a+"/records/"+first["recordId"].(string)+"/redate", map[string]any{"expectedSourceVersion": definition["headSourceVersion"], "expectedRecordRevision": first["revision"], "newEffectiveDate": "2027-02-01"}, foundation.NewID(), 409)
	stop := h.add("position", h.detail("position", other, ""), "DEACTIVATE", "2027-03-01", nil, 200)
	total(b, apr, 3, "complete")
	check(detail(b, other, apr)["quota"], float64(1))
	h.add("position", h.detail("position", other, stop["selectedRecordId"].(string)), "REACTIVATE", jun, nil, 200)
	check(detail(b, other, "2027-05-01")["quota"], float64(2))
	total(b, jun, 6, "complete")
	// Reverse the order with different principals and two real transactions.
	passwd := foundation.NewID()
	must(t, bootstrap.Initialize(f.ctx, f.admin, bootstrap.InitInput{Instance: f.instance, EnterpriseCode: "LOCAL", EnterpriseName: "本地验证企业", Start: "1900-01-01", Login: "quota-admin", Password: passwd}))
	creds := []foundation.Credentials{geoCredentials203(t, f, "admin", f.password), geoCredentials203(t, f, "quota-admin", passwd)}
	for round, code := range []string{"R315A", "R315B"} {
		id := pos(code)
		svc := f.runtime.Staffing.(app.Service)
		gate := &quotaGate315{svc.QuotaStore, make(chan struct{}), make(chan struct{})}
		type answer struct {
			result s.Result
			err    error
		}
		done := make(chan answer, 1)
		second := make(chan answer, 1)
		run := func(svc app.Service, who int, ch chan<- answer) {
			amount := 2 + who
			result, err := svc.ChangeStaffingQuota(f.ctx, creds[who], s.QuotaCommand{SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Action: "create", Input: s.QuotaInput{PositionID: id, Amount: &amount, Reason: "首次核编", EffectiveDate: jan}})
			ch <- answer{result, err}
		}
		go run(app.Service{QuotaStore: gate}, round, done)
		wait307(t, f, gate.ready)
		go run(svc, 1-round, second)
		blocked307(t, f, gate.release)
		close(gate.release)
		one, two := <-done, <-second
		must(t, one.err)
		must(t, two.err)
		check(one.result.Outcome, "accepted")
		check(two.result.FailureStatus, 409)
		check(detail(b, id, jan)["quota"], float64(2+round))
		for _, table := range []string{"quota_source", "quota_record", "quota_outbox"} {
			check(f.scalar("SELECT count(*) FROM staffing."+table+" WHERE object_id=$1", one.result.ObjectID), 1)
		}
	}
	// Zero, explicit withdrawal and subsequent allocation keep one stable root.
	obj := detail(b, other, jun)
	root := obj["quotaId"].(string)
	add := func(op string, source any, n any, date string) map[string]any {
		v := maps.Clone(map[string]any{"expectedSourceVersion": source, "effectiveDate": date, "reason": "维护"})
		if n != nil {
			v["quota"] = n
		}
		return write(base+"/"+root+"/"+op, v)
	}
	zero := add("adjust", obj["sourceVersion"], 0, "2027-07-01")
	check(detail(b, other, "2027-07-01")["quotaState"], "set")
	unset := add("withdraw", zero["sourceVersion"], nil, "2027-08-01")
	check(detail(b, other, "2027-08-01")["quotaState"], "unset")
	again := add("adjust", unset["sourceVersion"], 1, "2027-09-01")
	check(again["objectId"], root)
	recordPath := base + "/" + root + "/records/" + again["selectedRecordId"].(string)
	f.exec("UPDATE access.subject_grant SET write_fields=ARRAY['effectiveDate','quota','reason'] WHERE principal_id=$1 AND action='staffingquota.redate'", b.subject)
	moved := write(recordPath+"/redate", map[string]any{
		"expectedSourceVersion": again["sourceVersion"], "expectedRecordRevision": again["selectedRecordRevision"], "newEffectiveDate": "2027-09-20",
	})
	check(detail(b, other, "2027-09-19")["quotaState"], "unset")
	check(detail(b, other, "2027-09-20")["quota"], float64(1))
	deletingAdmin := f.login("quota-admin", passwd)
	deleted := deletingAdmin.call("POST", recordPath+"/delete", map[string]any{"expectedSourceVersion": moved["sourceVersion"], "expectedRecordRevision": moved["selectedRecordRevision"]}, foundation.NewID(), 200)
	check(deleted["originalSourceVersion"], moved["sourceVersion"])
	check(resolve(deletingAdmin, deleted["submissionId"].(string))["originalSourceVersion"], moved["sourceVersion"])
	stale := b.call("GET", base+"/positions/"+other+"?selectedRecordId="+again["selectedRecordId"].(string), nil, "", 200)
	check(stale["selectionMissing"], true)
	check(stale["actions"], nil)
	current := detail(b, other, "2027-09-20")
	check(current["quotaState"], "unset")
	removed := b.call("GET", base+"/history/"+root+"/changes", nil, "", 200)["items"].([]any)[0].(map[string]any)
	check(removed["action"], "delete")
	check(removed["operator"].(map[string]any)["subjectId"], deletingAdmin.subject)
	check(removed["previousSourceVersion"], moved["sourceVersion"])
	check(detail(b, other, "2027-09-20&sourceVersion="+removed["previousSourceVersion"].(string))["quota"], float64(1))

	// A failure after the root, source and records were written must undo all of them.
	interrupted := pos("FAIL315")
	interruptedKey := foundation.NewID()
	f.exec(`CREATE FUNCTION staffing.fail_quota_event315() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION '315 event persistence interrupted'; END $$;
CREATE TRIGGER fail_quota_event315 BEFORE INSERT ON staffing.quota_outbox FOR EACH ROW EXECUTE FUNCTION staffing.fail_quota_event315()`)
	b.call("POST", base, body(interrupted, jan, 2), interruptedKey, 503)
	check(f.scalar("SELECT count(*) FROM staffing.quota WHERE target_id=$1", interrupted), 0)
	check(f.scalar("SELECT count(*) FROM staffing.quota_source WHERE submission_id=$1", interruptedKey), 0)
	check(f.scalar("SELECT count(*) FROM staffing.quota_terminal WHERE submission_id=$1", interruptedKey), 0)
	f.exec("DROP TRIGGER fail_quota_event315 ON staffing.quota_outbox")
	b.call("POST", base, body(interrupted, jan, 2), interruptedKey, 200)
	check(detail(b, interrupted, jan)["quota"], float64(2))
}

//go:build integration

package integration_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/foundation"
)

func t07Clone(v map[string]any) map[string]any {
	raw, _ := json.Marshal(v)
	var r map[string]any
	_ = json.Unmarshal(raw, &r)
	return r
}

func t07Instance(t *testing.T, f *fixture, start string) *fixture {
	t.Helper()
	other := &fixture{t: t, ctx: f.ctx, admin: f.admin, instance: foundation.NewID(), password: foundation.NewID(), readPassword: foundation.NewID()}
	must(t, bootstrap.Initialize(f.ctx, f.admin, bootstrap.InitInput{DevelopmentFixture: true, Instance: other.instance, EnterpriseCode: "SECOND", EnterpriseName: "独立实例", Start: start, Login: "admin", Password: other.password}))
	var e error
	other.runtime, e = bootstrap.Open(f.ctx, f.runtime.Pool.Config().ConnString(), other.instance)
	must(t, e)
	t.Cleanup(other.runtime.Close)
	other.server = httptest.NewUnstartedServer(nil)
	api := &bootstrap.API{Runtime: other.runtime, InsecureLocal: true, Origin: "http://" + other.server.Listener.Addr().String()}
	other.server.Config.Handler = api.Handler()
	other.server.Start()
	t.Cleanup(other.server.Close)
	return other
}
func TestT07InstanceSubjectAndReadonlyBoundaries(t *testing.T) {
	f := fresh(t)
	g := t07Instance(t, f, "2020-01-01")
	key := foundation.NewID()
	subjects := []*browser{}
	for _, x := range []*fixture{f, g} {
		must(t, bootstrap.Initialize(x.ctx, x.admin, bootstrap.InitInput{DevelopmentFixture: true, Instance: x.instance, EnterpriseCode: map[bool]string{true: "LOCAL", false: "SECOND"}[x == f], EnterpriseName: map[bool]string{true: "本地验证企业", false: "独立实例"}[x == f], Start: "2020-01-01", Login: "second", Password: x.readPassword}))
		subjects = append(subjects, x.login("admin", x.password), x.login("second", x.readPassword))
	}
	saved := make([]map[string]any, len(subjects))
	type writeReply struct {
		index, status int
		body          map[string]any
		err           error
	}
	start := make(chan struct{})
	done := make(chan writeReply, len(subjects))
	for i, b := range subjects {
		go func(i int, b *browser) {
			<-start
			status, out, e := b.response("POST", "/api/v1/business-units", createBody(fmt.Sprintf("T07_SUBJECT_%d", i), "2026-01-01"), key)
			done <- writeReply{i, status, out, e}
		}(i, b)
	}
	close(start)
	for range subjects {
		r := <-done
		must(t, r.err)
		if r.status != 201 {
			t.Fatal("concurrent namespaced create", r.status, r.body)
		}
		saved[r.index] = r.body
	}

	if f.scalar("SELECT (SELECT count(*) FROM organization.bu_history h CROSS JOIN LATERAL unnest(h.submission_refs) r WHERE h.ids[r]=$1::uuid)+(SELECT count(*) FROM organization.bu_terminal WHERE submission_id=$1)", key) != 4 {
		t.Fatal("UUID was not namespaced by both instance and subject")
	}
	for i, b := range subjects {
		got := b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
		if got["sourceVersion"] != saved[i]["sourceVersion"] {
			t.Fatal("crossed subject result")
		}
	}
	a := subjects[0]
	other := subjects[2]
	id := saved[0]["buId"].(string)
	foreignID := saved[2]["buId"].(string)
	other.call("GET", "/api/v1/business-units/"+id, nil, "", 404)
	a.call("GET", "/api/v1/business-units/"+id+"?sourceVersion="+saved[2]["sourceVersion"].(string), nil, "", 404)
	foreignRecord := recordOn(t, detail(other, foreignID), "2026-01-01")
	a.call("POST", "/api/v1/business-units/"+id+"/records/"+foreignRecord["recordId"].(string)+"/correct", correctBody(detail(a, id), foreignRecord, map[string]any{"name": "越界"}), foundation.NewID(), 409)
	reader := f.login("reader", f.readPassword)
	d := detail(reader, id)
	reader.call("POST", "/api/v1/business-units/"+id+"/records", addBody(d, "2026-06-01", map[string]any{"name": "只读新增"}), foundation.NewID(), 403)
	first := recordOn(t, d, "2026-01-01")
	reader.call("POST", "/api/v1/business-units/"+id+"/records/"+first["recordId"].(string)+"/correct", correctBody(d, first, map[string]any{"name": "只读更正"}), foundation.NewID(), 403)
	// Same-host cookies can reach a second port; the configured instance must still
	// reject a token issued by the other instance.
	origin, _ := url.Parse(f.server.URL)
	req, e := http.NewRequest("GET", g.server.URL+"/api/v1/session", nil)
	must(t, e)
	for _, c := range a.client.Jar.Cookies(origin) {
		req.AddCookie(c)
	}
	res, e := http.DefaultClient.Do(req)
	must(t, e)
	res.Body.Close()
	if res.StatusCode != 401 {
		t.Fatal("foreign credential accepted", res.StatusCode)
	}
}

func TestT07DateExtremesPersistWithOpenUpperBound(t *testing.T) {
	f := fresh(t)
	g := t07Instance(t, f, "0001-01-01")
	b := g.login("admin", g.password)
	for i, date := range []string{"0001-01-01", "9999-12-31"} {
		saved := b.call("POST", "/api/v1/business-units", createBody(fmt.Sprintf("T07_EXTREME_%d", i), date), foundation.NewID(), 201)
		d := b.getOnDay(date, "/api/v1/business-units/"+saved["buId"].(string))
		r := recordOn(t, d, date)
		if r["effectiveDate"] != date || r["endExclusive"] != nil {
			t.Fatal(d)
		}
		var lower string
		var infinite bool
		must(t, f.admin.QueryRow(f.ctx, `SELECT lower(r)::text,upper_inf(r) FROM organization.bu_history h JOIN organization.bu_dictionary d ON (d.instance_id,d.enterprise_id,d.id)=(h.instance_id,h.enterprise_id,h.dictionary_id) CROSS JOIN LATERAL unnest(d.ranges) r WHERE h.object_id=$1 AND NOT isempty(r) AND lower(r)=$2::date AND upper_inf(r) LIMIT 1`, saved["buId"], date).Scan(&lower, &infinite))
		if lower != date || !infinite {
			t.Fatal("finite or shifted extreme", lower, infinite)
		}
	}
}

func TestT07CodeDisplayCaseAndNormalizedUniqueness(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	saved := b.call("POST", "/api/v1/business-units", createBody("  Bu_Mixed  ", "2026-01-01"), foundation.NewID(), 201)
	var display, normalized string
	must(t, f.admin.QueryRow(f.ctx, "SELECT code,normalized_code FROM organization.business_unit WHERE bu_id=$1", saved["buId"]).Scan(&display, &normalized))
	if display != "Bu_Mixed" || normalized != "bu_mixed" {
		t.Fatal("display/lookup code normalization", display, normalized)
	}
	b.call("POST", "/api/v1/business-units", createBody("bU_mIxEd", "2026-01-01"), foundation.NewID(), 409)
	id := saved["buId"].(string)
	path := "/api/v1/business-units/" + id
	b.call("POST", path+"/records", addBody(detail(b, id), "2026-03-01", map[string]any{"name": "乙"}), foundation.NewID(), 201)
	d := detail(b, id)
	r := recordOn(t, d, "2026-03-01")
	b.call("POST", path+"/records/"+r["recordId"].(string)+"/delete", deleteBody(d, r), foundation.NewID(), 200)
	b.call("POST", "/api/v1/business-units", createBody("BU_mixed", "2026-01-01"), foundation.NewID(), 409)
	d = detail(b, id)
	b.call("POST", path+"/deactivate", lifeBody(d, "2026-04-01", "DEACTIVATE", nil), foundation.NewID(), 201)
	b.call("POST", "/api/v1/business-units", createBody("bu_MIXED", "2026-01-01"), foundation.NewID(), 409)
	if f.scalar("SELECT count(*) FROM organization.business_unit WHERE normalized_code='bu_mixed'") != 1 {
		t.Fatal("lifecycle released stable code")
	}
}

func TestCompactSameObjectSubmissionRemainsSubjectScoped(t *testing.T) {
	f := fresh(t)
	must(t, bootstrap.Initialize(f.ctx, f.admin, bootstrap.InitInput{DevelopmentFixture: true, Instance: f.instance, EnterpriseCode: "LOCAL", EnterpriseName: "本地验证企业", Start: "2020-01-01", Login: "second", Password: f.readPassword}))
	a, b := f.login("admin", f.password), f.login("second", f.readPassword)
	bu := a.call("POST", "/api/v1/business-units", createBody("SHARED_SUBMISSION", "2026-01-01"), foundation.NewID(), 201)
	for _, kind := range []string{"business-units", "departments"} {
		t.Run(kind, func(t *testing.T) {
			id, resolve := bu["buId"].(string), "organization"
			if kind == "departments" {
				created := a.call("POST", "/api/v1/departments", map[string]any{"code": "SHARED_SUBMISSION", "name": "部门", "managementBuId": id, "effectiveDate": "2026-01-01", "operationType": "CREATE", "operationReason": "NEW_ORGANIZATION"}, foundation.NewID(), 201)
				id, resolve = created["departmentId"].(string), "departments"
			}
			path, key := "/api/v1/"+kind+"/"+id, foundation.NewID()
			bodies, results := []map[string]any{}, []map[string]any{}
			for i, client := range []*browser{a, b} {
				d := client.call("GET", path, nil, "", 200)
				r := selected(d)
				body := map[string]any{"expectedSourceVersion": d["headSourceVersion"], "expectedRecordRevision": r["revision"], "changes": map[string]any{"name": fmt.Sprintf("主体%d的独立修改", i+1)}}
				bodies = append(bodies, body)
				results = append(results, client.call("POST", path+"/records/"+r["recordId"].(string)+"/correct", body, key, 200))
			}
			if results[0]["sourceVersion"] == results[1]["sourceVersion"] {
				t.Fatal("two subjects were merged")
			}
			for i, client := range []*browser{a, b} {
				got := client.call("POST", "/api/v1/"+resolve+"/submissions/"+key+"/resolve", nil, "", 200)
				if got["outcome"] != "accepted" || got["sourceVersion"] != results[i]["sourceVersion"] {
					t.Fatal("same object and UUID crossed subjects", got, results[i])
				}
				r := selected(client.call("GET", path, nil, "", 200))
				replayed := client.call("POST", path+"/records/"+r["recordId"].(string)+"/correct", bodies[i], key, 200)
				if replayed["sourceVersion"] != results[i]["sourceVersion"] {
					t.Fatal("exact replay lost the original subject result", replayed)
				}
			}
		})
	}
}

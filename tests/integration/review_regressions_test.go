//go:build integration

package integration_test

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/foundation"
)

func TestInitialAccountCanAuthenticate(t *testing.T) {
	f := fresh(t)
	initialize := func(login string) error {
		return bootstrap.Initialize(f.ctx, f.admin, bootstrap.InitInput{Instance: f.instance, EnterpriseCode: "LOCAL", EnterpriseName: "本地验证企业", Start: "2020-01-01", Login: login, Password: f.password, ReadOnly: true})
	}
	counts := map[string]int{}
	for _, table := range []string{"principal", "credential", "policy_guard", "subject_grant"} {
		counts[table] = f.scalar("SELECT count(*) FROM access." + table)
	}
	for _, invalid := range []string{" \t ", strings.Repeat("A", 101), strings.Repeat("中", 34)} {
		if initialize(invalid) == nil {
			t.Fatal("initialization accepted a login outside authentication bounds")
		}
		for table, before := range counts {
			if f.scalar("SELECT count(*) FROM access."+table) != before {
				t.Fatalf("rejected initialization left partial %s", table)
			}
		}
	}
	for _, valid := range []string{strings.Repeat("A", 100), strings.Repeat("中", 33) + "A"} {
		must(t, initialize(" \t"+valid+" \n"))
		b := f.login("  "+valid+"  ", f.password)
		principal := b.call("GET", "/api/v1/session", nil, "", 200)
		if principal["login"] != valid {
			t.Fatal("initialization and login must share normalization")
		}
	}
}

func TestBUNameSortUsesCombinedVisibleFields(t *testing.T) {
	f := fresh(t)
	a := f.login("admin", f.password)
	today := a.call("GET", "/api/v1/organization/context", nil, "", 200)["currentDate"].(string)
	first := createBody("SORT_A", today)
	first["name"] = "Alpha"
	second := createBody("SORT_B", today)
	second["name"] = "Zulu"
	idA := a.call("POST", "/api/v1/business-units", first, foundation.NewID(), 201)["buId"].(string)
	idB := a.call("POST", "/api/v1/business-units", second, foundation.NewID(), 201)["buId"].(string)
	// Controlled grants exercise additive field access; no permission editor is introduced.
	grant := func(id string, fields []string) {
		f.exec(`INSERT INTO access.subject_grant(instance_id,grant_id,principal_id,scope_kind,scope_id,action,read_fields,write_fields)
 VALUES($1,$2,$3,'bu',$4,'bu.read',$5,ARRAY[]::text[])`, f.instance, foundation.NewID(), a.subject, id, fields)
	}
	grant(idA, []string{"code"})
	grant(foundation.NewID(), []string{"code"}) // Nonexistent scope cannot narrow visible rows.
	for _, order := range []struct{ sort, first string }{{"name", idA}, {"name_desc", idB}} {
		v := a.call("GET", "/api/v1/business-units?sort="+order.sort, nil, "", 200)
		if v["items"].([]any)[0].(map[string]any)["id"] != order.first {
			t.Fatal("narrow additional grant changed existing name sort")
		}
	}
	// Enterprise code access plus a BU name grant gives that BU both fields.
	f.exec("UPDATE access.subject_grant SET read_fields=ARRAY['code'] WHERE principal_id=$1 AND action='bu.read' AND scope_kind='enterprise'", a.subject)
	f.exec("UPDATE access.subject_grant SET read_fields=ARRAY['name'] WHERE principal_id=$1 AND action='bu.read' AND scope_id=$2", a.subject, idA)
	for _, sort := range []string{"name", "name_desc"} {
		a.call("GET", "/api/v1/business-units?sort="+sort, nil, "", 403)
	}
	v := a.call("GET", "/api/v1/business-units?sort=code", nil, "", 200)
	raw, e := json.Marshal(v)
	must(t, e)
	if strings.Contains(string(raw), "Zulu") || !strings.Contains(string(raw), "Alpha") {
		t.Fatal("code sorting must honor combined fields per visible BU")
	}
	grant(idB, []string{"name"})
	a.call("GET", "/api/v1/business-units?sort=name", nil, "", 200)
	f.exec("DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='bu.read' AND scope_kind='enterprise'", a.subject)
	a.call("GET", "/api/v1/business-units?sort=name_desc", nil, "", 200)
}

func TestTemporaryAuthStorageFailureKeepsSession(t *testing.T) {
	f := fresh(t)
	// Bound the logout lookup in this fixture too; the BU transaction already has a lock timeout.
	f.exec("ALTER ROLE chnhcm_api SET lock_timeout='250ms'")
	f.runtime.Pool.Reset()
	for _, v := range []struct{ name, table, method, path string }{
		{"read_session", "access.principal", "GET", "/api/v1/business-units"},
		{"logout_lookup", "access.session", "DELETE", "/api/v1/session"},
	} {
		t.Run(v.name, func(t *testing.T) {
			b := f.login("admin", f.password)
			tx, e := f.admin.Begin(f.ctx)
			must(t, e)
			defer tx.Rollback(f.ctx)
			_, e = tx.Exec(f.ctx, "LOCK TABLE "+v.table+" IN ACCESS EXCLUSIVE MODE")
			must(t, e)
			status, body, e := b.response(v.method, v.path, nil, "")
			must(t, tx.Rollback(f.ctx))
			must(t, e)
			if status != 503 || body["code"] != "M02_DEPENDENCY_UNAVAILABLE" {
				t.Fatalf("temporary storage failure must be 503, got %d code=%v", status, body["code"])
			}
			// The original cookie still works after the storage failure ends.
			b.call("GET", "/api/v1/session", nil, "", 200)
			b.call("DELETE", "/api/v1/session", nil, "", 204)
			b.call("GET", "/api/v1/session", nil, "", 401)
		})
	}
}

func TestBUFullUnicodeNameSearch(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	today := b.call("GET", "/api/v1/organization/context", nil, "", 200)["currentDate"].(string)
	body := createBody("UNICODE_SEARCH", today)
	body["name"] = strings.Repeat("中", 200)
	b.call("POST", "/api/v1/business-units", body, foundation.NewID(), 201)
	result := b.call("GET", "/api/v1/business-units?q="+url.QueryEscape(body["name"].(string)), nil, "", 200)
	if result["count"] != float64(1) {
		t.Fatal("a valid full Chinese name must match its saved BU")
	}
	b.call("GET", "/api/v1/business-units?q="+url.QueryEscape(strings.Repeat("中", 400)), nil, "", 200)
	b.call("GET", "/api/v1/business-units?q="+url.QueryEscape(strings.Repeat("中", 401)), nil, "", 400)
}

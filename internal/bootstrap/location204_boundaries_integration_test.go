//go:build integration

package bootstrap_test

import (
	"chnhcm.local/chnhcm/internal/bootstrap"
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/location"
	orgdb "chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestLocation204BoundariesAndCapturedReads(t *testing.T) {
	f := freshWithEnterpriseStart(t, "1900-01-01")
	must(t, f.runtime.InitializeCountryCatalog(f.ctx, access.LoginInput{Login: "admin", Password: f.password}))
	b := f.login("admin", f.password)
	h := location204{t, f, b, map[string]string{"MO": geoCountry203(t, f, "MO")}}
	original := h.create("BOUNDARY204", "MO", "", "")
	g := &fixture{t: t, ctx: f.ctx, admin: f.admin, instance: foundation.NewID(), password: foundation.NewID()}
	must(t, bootstrap.Initialize(g.ctx, g.admin, bootstrap.InitInput{Instance: g.instance, EnterpriseCode: "OTHER204", EnterpriseName: "另一企业", Start: "1900-01-01", Login: "admin", Password: g.password}))
	var e error
	g.runtime, e = bootstrap.Open(g.ctx, f.runtime.Pool.Config().ConnConfig.ConnString(), g.instance)
	must(t, e)
	defer g.runtime.Close()
	api := &bootstrap.API{Runtime: g.runtime, InsecureLocal: true}
	g.server = httptest.NewUnstartedServer(nil)
	api.Origin = "http://" + g.server.Listener.Addr().String()
	g.server.Config.Handler = api.Handler()
	g.server.Start()
	defer g.server.Close()
	must(t, g.runtime.InitializeCountryCatalog(g.ctx, access.LoginInput{Login: "admin", Password: g.password}))
	other := g.login("admin", g.password)
	second := location204{t, g, other, map[string]string{"MO": geoCountry203(t, g, "MO")}}
	second.create("BOUNDARY204", "MO", "", "")
	other.call("GET", "/api/v1/locations/"+original["id"].(string), nil, "", 404)
	other.call("GET", "/api/v1/locations/"+original["id"].(string)+"?sourceVersion="+original["headSourceVersion"].(string), nil, "", 404)
	sid := foundation.NewID()
	body := map[string]any{"code": "NAMESPACE204", "name": "提交边界", "countryRegionId": h.countries["MO"], "addressLine": "一号", "operationType": "CREATE", "operationReason": "NEW_LOCATION"}
	saved := b.call("POST", "/api/v1/locations", body, sid, 201)
	b.call("POST", "/api/v1/business-units", createBody("NAMESPACE204", "1900-01-01"), sid, 201)
	body["name"] = "异正文"
	b.call("POST", "/api/v1/locations", body, sid, 409)
	if other.call("POST", "/api/v1/locations/submissions/"+sid+"/resolve", nil, "", 200)["outcome"] != "not_applied" {
		t.Fatal("foreign receipt escaped")
	}
	reader := f.login("reader", f.readPassword)
	if reader.call("POST", "/api/v1/locations/submissions/"+sid+"/resolve", nil, "", 200)["outcome"] != "not_applied" {
		t.Fatal("foreign subject receipt escaped")
	}
	if b.call("POST", "/api/v1/locations/submissions/"+sid+"/resolve", nil, "", 200)["sourceVersion"] != saved["sourceVersion"] {
		t.Fatal("original receipt changed")
	}
	b.call("GET", "/api/v1/locations/"+h.countries["MO"], nil, "", 404)
	f.exec("CREATE TABLE public.g4_original_grants AS SELECT * FROM access.subject_grant")
	t.Run("country-column-exact-name-and-permissions", func(t *testing.T) {
		countryID := h.countries["MO"]
		country := b.call("GET", "/api/v1/country-regions/"+countryID, nil, "", 200)
		record := country["selectedRecord"].(map[string]any)
		oldName := locRecord(original)["countryName"]
		b.call("POST", "/api/v1/country-regions/"+countryID+"/records/"+record["recordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": country["headSourceVersion"], "expectedRecordRevision": record["revision"], "changes": map[string]any{"name": "澳门（列表验证）"}}, foundation.NewID(), 200)
		newer := h.create("COUNTRY-COLUMN204", "MO", "", "")
		row := func(code string) map[string]any {
			return b.call("GET", "/api/v1/locations?q="+code, nil, "", 200)["items"].([]any)[0].(map[string]any)
		}
		if row("BOUNDARY204")["countryName"] != oldName || row("COUNTRY-COLUMN204")["countryName"] != locRecord(newer)["countryName"] || locRecord(newer)["countryName"] == oldName {
			t.Fatal("list country name must follow each saved adoption")
		}
		h.policy("UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'name') WHERE principal_id=$1 AND action='country.history.read'", b.subject)
		if row("BOUNDARY204")["countryName"] != nil {
			t.Fatal("list disclosed country name without adopted-source read permission")
		}
		h.restore()
		h.policy("UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'countryRegionId') WHERE principal_id=$1 AND action='location.read'", b.subject)
		if got := row("BOUNDARY204"); got["countryRegionId"] != nil || got["countryName"] != nil {
			t.Fatal("list disclosed country identity or label without location field permission")
		}
		h.restore()
	})
	future := h.create("HIDDEN-DATE", "MO", "", "2099-01-01")
	h.policy("UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'effectiveDate') WHERE principal_id=$1 AND action IN ('location.read','location.history.read')", b.subject)
	hidden := h.refresh(future)
	if hidden["selectedRecordIsFuture"] == true || locRecord(hidden)["effectiveDate"] != nil || locRecord(hidden)["adoptionDate"] != nil {
		t.Fatal("hidden date leaked")
	}
	h.restore()
	h.add(original, "ADD", "2000-01-01", map[string]any{"addressLine": "第二条"}, 200)
	current := h.refresh(original)
	h.policy("UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'effectiveDate') WHERE principal_id=$1 AND action='location.history.read'", b.subject)
	b.call("POST", "/api/v1/locations/reference-context", map[string]any{"action": "CORRECT", "locationId": original["id"], "expectedSourceVersion": current["headSourceVersion"], "recordId": locRecord(original)["recordId"], "expectedRecordRevision": current["timeline"].([]any)[0].(map[string]any)["revision"]}, "", 403)
	h.restore()
	svc := f.runtime.Location.(app.Service)
	for _, kind := range []string{"list", "export"} {
		t.Run(kind, func(t *testing.T) {
			v := h.create("CAPTURE-"+kind, "MO", "", "")
			id := v["id"].(string)
			name := locRecord(v)["name"].(string)
			creds := geoCredentials203(t, f, "admin", f.password)
			var input org.LocationExportInput
			if kind == "export" {
				empty := []string{}
				draft, e := svc.StartSelection(f.ctx, creds, org.LocationSelectionStartInput{Query: org.LocationSelectionQuery{Q: "CAPTURE-" + kind}, Selection: org.Selection{Mode: "allMatching", ExcludedIDs: &empty}})
				must(t, e)
				input = org.LocationExportInput{Query: org.LocationSelectionQuery{Q: "CAPTURE-" + kind}, DraftID: draft.Summary.DraftID, ExpectedRevision: draft.Summary.Revision}
			}
			base := svc.Store.(*orgdb.Store)
			cfg := base.Pool.Config().Copy()
			reached, release := make(chan struct{}), make(chan struct{})
			prefix := "WITH matched AS MATERIALIZED"
			if kind == "export" {
				prefix = "WITH page AS MATERIALIZED"
			}
			cfg.ConnConfig.Tracer = &snapshotReadBarrier{prefix: prefix, reached: reached, release: release}
			pool, e := pgxpool.NewWithConfig(f.ctx, cfg)
			must(t, e)
			defer pool.Close()
			store := orgdb.Store{Pool: pool, Instance: base.Instance, Clock: base.Clock, RecordedClock: base.RecordedClock, Guard: base.Guard}
			readSvc := app.Service{Store: &store}
			var once sync.Once
			unblock := func() { once.Do(func() { close(release) }) }
			defer unblock()
			type response struct {
				list org.LocationList
				csv  string
				err  error
			}
			done := make(chan response, 1)
			go func() {
				var got response
				if kind == "list" {
					got.list, got.err = readSvc.List(f.ctx, creds, org.LocationQuery{ListInput: org.ListInput{Q: name}})
				} else {
					var out org.LocationExport
					out, got.err = readSvc.ExportSelection(f.ctx, creds, input)
					if got.err == nil {
						var data []byte
						data, got.err = io.ReadAll(out.File)
						out.File.Close()
						got.csv = string(data)
					}
				}
				done <- got
			}()
			select {
			case <-reached:
			case <-time.After(5 * time.Second):
				t.Fatal("capture barrier not reached")
			}
			changed := h.correct(v, map[string]any{"name": "并发新名称", "addressLine": "并发新地址"}, nil, 200)
			unblock()
			got := <-done
			must(t, got.err)
			if kind == "list" {
				if got.list.Count != 1 || len(got.list.Items) != 1 || *got.list.Items[0].Name != name {
					t.Fatal("count and page mixed heads")
				}
			} else if !strings.Contains(got.csv, name) || strings.Contains(got.csv, "并发新名称") {
				t.Fatal("export escaped captured head")
			}
			if h.detail(id, "")["headSourceVersion"] != changed["sourceVersion"] {
				t.Fatal("writer not committed")
			}
		})
	}
}

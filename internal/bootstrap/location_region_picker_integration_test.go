//go:build integration

package bootstrap_test

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/geoarea"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"errors"
	"net/url"
	"testing"
	"time"
)

func TestLocationRegionPickerCandidates(t *testing.T) {
	f := fresh(t)
	login := access.LoginInput{Login: "admin", Password: f.password}
	must(t, f.runtime.InitializeCountryCatalog(f.ctx, login))
	must(t, f.runtime.InitializeGeoAreaCatalog(f.ctx, login))
	svc := f.runtime.GeoArea.(app.Service)
	c := geoCredentials203(t, f, "admin", f.password)
	country := geoCountry203(t, f, "CN")
	for _, query := range []string{"广东省深圳市南山区", "广东 深圳 南山", "广东省 / 深圳市 / 南山区"} {
		result, e := svc.Candidates(f.ctx, c, org.GeoAreaCandidateInput{CountryRegionID: country, AdoptedDate: "1900-01-01", Q: query})
		must(t, e)
		if len(result.Items) != 1 || result.Items[0].Code != "440305" || !result.Items[0].Terminal {
			t.Fatalf("combined path %q: %+v", query, result)
		}
	}
	result, e := svc.Candidates(f.ctx, c, org.GeoAreaCandidateInput{CountryRegionID: country, AdoptedDate: "1900-01-01", Q: "南山"})
	must(t, e)
	if len(result.Items) != 2 || result.Items[0].Code != "230404" || result.Items[1].Code != "440305" {
		t.Fatalf("ambiguous names must retain both full paths: %+v", result)
	}
	result, e = svc.Candidates(f.ctx, c, org.GeoAreaCandidateInput{CountryRegionID: country, AdoptedDate: "1900-01-01", Q: "北京 深圳 南山"})
	must(t, e)
	if len(result.Items) != 0 {
		t.Fatalf("unrelated path fragments: %+v", result)
	}
	result, e = svc.Candidates(f.ctx, c, org.GeoAreaCandidateInput{CountryRegionID: country, AdoptedDate: "1900-01-01", PageSize: 50})
	must(t, e)
	if len(result.Items) != 31 || result.HasMore {
		t.Fatalf("all mainland province choices: %+v", result)
	}
	result, e = svc.Candidates(f.ctx, c, org.GeoAreaCandidateInput{CountryRegionID: country, AdoptedDate: "1900-01-01", Q: "区", PageSize: 50})
	must(t, e)
	if len(result.Items) != 50 || !result.HasMore || result.Cursor == nil {
		t.Fatal("search must remain paged")
	}
	first := result.Items[0].ID
	second, e := svc.Candidates(f.ctx, c, org.GeoAreaCandidateInput{CountryRegionID: country, AdoptedDate: "1900-01-01", Q: "区", PageSize: 50, Cursor: *result.Cursor})
	must(t, e)
	if len(second.Items) != 50 || second.Items[0].Code <= result.Items[49].Code {
		t.Fatal("search continuation changed ordering or repeated a page", first)
	}
	// Search must resolve names at the requested adoption day and still enforce
	// historical name access, including when a path fragment matches an ancestor.
	b := f.login("admin", f.password)
	gd := geoID203(t, f, "440000")
	detail := b.call("GET", "/api/v1/geo-areas/"+gd, nil, "", 200)
	record := detail["selectedRecord"].(map[string]any)
	b.call("POST", "/api/v1/geo-areas/"+gd+"/records", map[string]any{"expectedSourceVersion": detail["headSourceVersion"], "anchorRecordId": record["recordId"], "effectiveDate": "2000-01-01", "operationType": "CHANGE", "operationReason": "DISPLAY_DATA_UPDATE", "changes": map[string]any{"name": "广东新名称"}}, foundation.NewID(), 201)
	result, e = svc.Candidates(f.ctx, c, org.GeoAreaCandidateInput{CountryRegionID: country, AdoptedDate: "1900-01-01", Q: "广东省深圳市南山区"})
	must(t, e)
	if len(result.Items) != 1 || result.Items[0].Path[1].Name != "广东省" {
		t.Fatal("historical path search used current ancestor name")
	}
	reader := f.login("reader", f.readPassword)
	f.exec(`DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='geoarea.history.read'`, reader.subject)
	old := reader.call("GET", "/api/v1/geo-areas/candidates?countryRegionId="+country+"&adoptedDate=1900-01-01&q="+url.QueryEscape("广东省深圳市南山区"), nil, "", 200)
	if len(old["items"].([]any)) != 0 {
		t.Fatal("path search revealed a historical ancestor without history permission")
	}
	checkRegionPickerGuardOrdering(t, f)
	t.Log("combined paths, ambiguous names, unrelated fragments, all provinces, cursor continuation and historical ancestor names passed")
}

func checkRegionPickerGuardOrdering(t *testing.T, f *fixture) {
	b := f.login("reader", f.readPassword)
	path := "/api/v1/geo-areas/candidates?countryRegionId=" + geoCountry203(t, f, "CN") + "&adoptedDate=2020-01-01&pageSize=50&q=" + url.QueryEscape("吉林省")
	cursor := b.call("GET", path, nil, "", 200)["cursor"].(string)
	resume := path + "&cursor=" + url.QueryEscape(cursor)
	expectFailure := func(status int, code string) {
		t.Helper()
		resp := b.call("GET", resume, nil, "", status)
		if resp["code"] != code || resp["items"] != nil || resp["cursor"] != nil {
			t.Fatalf("unexpected candidate failure: %v", resp["code"])
		}
	}
	creds := geoCredentials203(t, f, "reader", f.readPassword)
	store := f.runtime.GeoArea.(app.Service).Store
	writer, err := f.admin.Acquire(f.ctx)
	must(t, err)
	defer writer.Release()
	pid := writer.Conn().PgConn().PID()
	written := make(chan error, 1)
	var before int64
	err = store.WithinGeoArea(f.ctx, creds, func(u app.Unit) error {
		before = u.Access().PolicyVersion()
		go func() {
			_, e := writer.Exec(f.ctx, "UPDATE access.policy_guard SET policy_no=policy_no+1 WHERE instance_id=$1 AND principal_id=$2", f.instance, b.subject)
			written <- e
		}()
		deadline := time.Now().Add(4 * time.Second)
		for f.scalar("SELECT cardinality(pg_blocking_pids($1))", pid) == 0 {
			if time.Now().After(deadline) {
				return errors.New("policy writer did not wait for candidate guard")
			}
			time.Sleep(5 * time.Millisecond)
		}
		return nil
	})
	must(t, err)
	must(t, <-written)
	must(t, store.WithinGeoArea(f.ctx, creds, func(u app.Unit) error {
		if u.Access().PolicyVersion() != before+1 {
			return errors.New("new read missed committed policy version")
		}
		return nil
	}))
	expectFailure(409, "M02_CURSOR_CHANGED")
	f.exec("REVOKE SELECT ON access.policy_guard FROM chnhcm_app")
	expectFailure(503, "M02_DEPENDENCY_UNAVAILABLE")
	f.exec("GRANT SELECT ON access.policy_guard TO chnhcm_app")
	f.exec("UPDATE access.session SET revoked_at=clock_timestamp() WHERE instance_id=$1 AND principal_id=$2", f.instance, b.subject)
	expectFailure(401, "AUTH_REQUIRED")
}

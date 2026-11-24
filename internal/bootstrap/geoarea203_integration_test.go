//go:build integration

package bootstrap_test

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/geoarea"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/geoarea"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/countryseed"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/geoareaseed"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"reflect"
	"runtime/pprof"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestGeoArea203Catalog(t *testing.T) {
	f := fresh(t)
	must(t, f.runtime.InitializeCountryCatalog(f.ctx, access.LoginInput{Login: "admin", Password: f.password}))
	geoPartial203(t, f)
	must(t, f.runtime.InitializeGeoAreaCatalog(f.ctx, access.LoginInput{Login: "admin", Password: f.password}))
	if got := f.scalar("SELECT count(*) FROM organization.geoarea"); got != 3630 {
		t.Fatalf("areas=%d", got)
	}
	if got := f.scalar("SELECT sum(cardinality(source_refs)) FROM organization.geoarea_history"); got != 3630 {
		t.Fatalf("sources=%d", got)
	}
	b := f.login("admin", f.password)
	list := b.call("GET", "/api/v1/geo-areas?pageSize=20", nil, "", 200)
	if list["count"] != float64(3630) {
		t.Fatalf("list=%v", list)
	}
	must(t, f.runtime.InitializeGeoAreaCatalog(f.ctx, access.LoginInput{Login: "admin", Password: f.password}))
	if got := f.scalar("SELECT sum(cardinality(source_refs)) FROM organization.geoarea_history"); got != 3630 {
		t.Fatalf("rerun sources=%d", got)
	}
	geoAllSources203(t, f)
	geoChecks203(t, f)

}

type geoFault203 struct {
	app.Store
	writes int
}

func (s *geoFault203) WithinGeoAreaWrite(ctx context.Context, c access.Credentials, fn func(app.Unit) error) error {
	s.writes++
	if s.writes == 2 {
		return errors.New("injected before second object")
	}
	return s.Store.WithinGeoAreaWrite(ctx, c, fn)
}
func geoCredentials203(t *testing.T, f *fixture, login, password string) access.Credentials {
	v, e := f.runtime.Access.Login(f.ctx, access.LoginInput{Login: login, Password: password}, foundation.NewID())
	must(t, e)
	return access.Credentials{Token: v.Token, CSRF: v.Principal.CSRF, Write: true}
}
func geoPartial203(t *testing.T, f *fixture) (app.CatalogPublication, []org.GeoAreaCommand) {
	c := geoCredentials203(t, f, "admin", f.password)
	p, e := geoareaseed.Fixed()
	must(t, e)
	cp, e := countryseed.Fixed()
	must(t, e)
	ids := map[string]string{}
	for _, row := range cp.Countries {
		cmd, e := cp.Command(f.instance, foundation.NewID(), row)
		must(t, e)
		ids[row.SourceKey] = cmd.Import.ID
	}
	rows, e := p.Prepare(f.instance, ids)
	must(t, e)
	cmds := []org.GeoAreaCommand{}
	for _, row := range rows {
		if row.Node.Structure.CountryCode == "HK" {
			cmds = append(cmds, app.ImportCommand(f.instance, app.Mapping{ID: row.Node.ID, Date: row.Node.EffectiveDate, Origin: d.Origin{Structure: row.Node.Structure, Key: row.SourceKey, SourceVersion: row.SourceVersion, PackageVersion: p.Version, PackageDigest: p.Digest, ArchiveDigest: p.ArchiveDigest}}, row.Name))
		}
	}
	svc := f.runtime.GeoArea.(app.Service)
	failed := app.Service{Store: &geoFault203{Store: svc.Store}}
	e = failed.InitializeCatalog(f.ctx, c, app.CatalogPublication{CountryID: ids["base.hk"], CountryCode: "HK", Version: p.Version, Digest: p.Digest, ArchiveDigest: p.ArchiveDigest, Count: 21}, cmds)
	if e == nil || !strings.Contains(e.Error(), "injected") {
		t.Fatal(e)
	}
	if f.scalar("SELECT count(*) FROM organization.geoarea") != 1 || f.scalar("SELECT count(*) FROM organization.geoarea_package") != 0 {
		t.Fatal("partial publication")
	}
	b := f.login("admin", f.password)
	id := cmds[0].Import.ID
	b.call("GET", "/api/v1/geo-areas/"+id, nil, "", 404)
	b.call("GET", "/api/v1/geo-areas/"+id+"/changes", nil, "", 404)
	list := b.call("GET", "/api/v1/geo-areas", nil, "", 200)
	if list["count"] != float64(0) {
		t.Fatal(list)
	}
	result, e := svc.Resolve(f.ctx, c, cmds[0].SubmissionID)
	must(t, e)
	if result.Outcome != "accepted" || result.GeoAreaID != nil || result.SourceVersion != nil {
		t.Fatal(result)
	}
	var state app.State
	must(t, svc.Store.WithinGeoArea(f.ctx, c, func(u app.Unit) error { var e error; state, e = u.Load(f.ctx, id, ""); return e }))
	r := state.Timeline.Records[0]
	body := map[string]any{"expectedSourceVersion": state.Timeline.Source, "expectedRecordRevision": r.Revision, "changes": map[string]any{"name": "cannot publish"}}
	b.call("POST", "/api/v1/geo-areas/"+id+"/records/"+r.ID+"/correct", body, foundation.NewID(), 404)
	b.call("POST", "/api/v1/geo-areas/"+id+"/preview", map[string]any{"action": "CORRECT", "recordId": r.ID, "expectedRecordRevision": r.Revision, "expectedSourceVersion": state.Timeline.Source, "changes": map[string]any{"name": "cannot publish"}}, "", 404)
	b.call("GET", "/api/v1/geo-areas/candidates?countryRegionId="+ids["base.hk"]+"&adoptedDate=1900-01-01", nil, "", 409)
	t.Log("G3-V04: injected interruption preserves first source; unpublished list/detail/log/preview/write and recovery projection checked")
	return app.CatalogPublication{CountryID: ids["base.hk"], CountryCode: "HK", Version: p.Version, Digest: p.Digest, ArchiveDigest: p.ArchiveDigest, Count: 21}, cmds
}
func geoID203(t *testing.T, f *fixture, code string) string {
	var id string
	must(t, f.admin.QueryRow(f.ctx, `SELECT geoarea_id::text FROM organization.geoarea WHERE instance_id=$1 AND code=$2`, f.instance, code).Scan(&id))
	return id
}
func geoCountry203(t *testing.T, f *fixture, code string) string {
	var id string
	must(t, f.admin.QueryRow(f.ctx, `SELECT country_region_id::text FROM organization.country_region WHERE instance_id=$1 AND code=$2`, f.instance, code).Scan(&id))
	return id
}
func geoChecks203(t *testing.T, f *fixture) {
	b := f.login("admin", f.password)
	reader := f.login("reader", f.readPassword)
	svc := f.runtime.GeoArea.(app.Service)
	c := geoCredentials203(t, f, "admin", f.password)
	countries := map[string]string{}
	for _, code := range []string{"CN", "HK", "TW", "MO"} {
		countries[code] = geoCountry203(t, f, code)
	}
	if f.scalar(`SELECT count(*) FROM organization.geoarea WHERE parent_id IS NULL`) != 66 || f.scalar(`SELECT count(*) FROM organization.geoarea a WHERE NOT EXISTS(SELECT FROM organization.geoarea child WHERE child.parent_id=a.geoarea_id)`) != 3245 {
		t.Fatal("structure denominators")
	}
	for code, count := range map[string]int{"CN": 3209, "HK": 21, "TW": 390, "MO": 10} {
		list := b.call("GET", "/api/v1/geo-areas?countryRegionId="+countries[code], nil, "", 200)
		if list["count"] != float64(count) {
			t.Fatal(code, list)
		}
	}
	for level, count := range map[string]int{"postal-region": 3, "parish": 7} {
		list := reader.call("GET", "/api/v1/geo-areas?filterField=level&filterOperator=equals&filterValue="+level, nil, "", 200)
		if list["count"] != float64(count) {
			t.Fatalf("Macau %s: %v", level, list)
		}
		for _, item := range list["items"].([]any) {
			r := item.(map[string]any)
			path := r["geoAreaPath"].([]any)
			self := path[len(path)-1].(map[string]any)
			if len(path) != 2 || self["id"] != r["id"] || self["name"] != r["name"] || r["level"] != level || r["code"] != nil || r["parentAreaId"] != nil {
				t.Fatalf("Macau path or code: %v", r)
			}
		}
	}
	for _, code := range []string{"441900", "442000", "460400", "620200"} {
		v, e := svc.Candidates(f.ctx, c, org.GeoAreaCandidateInput{CountryRegionID: countries["CN"], AdoptedDate: "1900-01-01", Q: code})
		must(t, e)
		if len(v.Items) != 1 || !v.Items[0].Terminal {
			t.Fatal(code, v)
		}
	}
	root, e := svc.Candidates(f.ctx, c, org.GeoAreaCandidateInput{CountryRegionID: countries["CN"], AdoptedDate: "1900-01-01"})
	must(t, e)
	if len(root.Items) != 20 || !root.HasMore || root.Cursor == nil {
		t.Fatal(root)
	}
	in := org.GeoAreaCandidateInput{CountryRegionID: countries["CN"], AdoptedDate: "1900-01-01", Cursor: *root.Cursor}
	second, e := svc.Candidates(f.ctx, c, in)
	must(t, e)
	if len(second.Items) != 11 || second.HasMore || second.Cursor != nil {
		t.Fatal(second)
	}
	in.Q = "区"
	if _, e = svc.Candidates(f.ctx, c, in); e == nil {
		t.Fatal("cursor query forgery")
	}
	in.Q = ""
	in.Cursor += "x"
	if _, e = svc.Candidates(f.ctx, c, in); e == nil {
		t.Fatal("cursor tamper")
	}
	basic, e := svc.Candidates(f.ctx, c, org.GeoAreaCandidateInput{CountryRegionID: countries["MO"], AdoptedDate: "1900-01-01"})
	must(t, e)
	if basic.Mode != "basic_address" || len(basic.Items) != 0 {
		t.Fatal(basic)
	}
	gd := geoID203(t, f, "440000")
	child, e := svc.Candidates(f.ctx, c, org.GeoAreaCandidateInput{CountryRegionID: countries["CN"], ParentAreaID: gd, AdoptedDate: "1900-01-01", PageSize: 50})
	must(t, e)
	if len(child.Items) != 21 {
		t.Fatal(child)
	}
	search, e := svc.Candidates(f.ctx, c, org.GeoAreaCandidateInput{CountryRegionID: countries["CN"], ParentAreaID: gd, AdoptedDate: "1900-01-01", Q: "区", PageSize: 20})
	must(t, e)
	if len(search.Items) != 20 {
		t.Fatal(search)
	}
	b.call("GET", "/api/v1/geo-areas/candidates?countryRegionId="+countries["HK"]+"&parentAreaId="+gd+"&adoptedDate=1900-01-01", nil, "", 400)
	query := org.GeoAreaQuery{PageSize: 20, Sort: "code", Filter: &org.GeoAreaFilter{Field: "level", Operator: "equals", Value: "province"}}
	list, e := svc.List(f.ctx, c, query)
	must(t, e)
	n, e := svc.Navigate(f.ctx, c, org.GeoAreaNavigationInput{Query: query, AnchorID: list.Items[0].ID, Direction: "next"})
	must(t, e)
	if n.TargetID == nil || *n.TargetID != list.Items[1].ID {
		t.Fatal(n)
	}
	n, e = svc.Navigate(f.ctx, c, org.GeoAreaNavigationInput{Query: query, AnchorID: geoID203(t, f, "441900"), Direction: "next"})
	must(t, e)
	if n.TargetID != nil {
		t.Fatal("absent navigation anchor")
	}
	t.Log("G3-V01/V02/V08: full region counts, roots/leaves, four missing-level leaves, 20+11 pagination, descendants, cursor tamper/query binding, Macau and navigation passed")
	if os.Getenv("CHNHCM_G3_CAPACITY") == "1" {
		if gate := os.Getenv("CHNHCM_G3_CAPACITY_GATE"); gate != "" {
			t.Log("G3-CAPACITY waiting for requested measurement gate")
			for {
				if _, e := os.Stat(gate); e == nil {
					break
				}
				select {
				case <-f.ctx.Done():
					t.Fatal(f.ctx.Err())
				case <-time.After(time.Second):
				}
			}
		}
		reader = f.login("reader", f.readPassword)
		b = geoCapacity203(t, f, countries)
		endpoint, err := url.Parse(f.server.URL)
		must(t, err)
		for _, cookie := range b.client.Jar.Cookies(endpoint) {
			if cookie.Name == "chnhcm_session" {
				c = access.Credentials{Token: cookie.Value, CSRF: b.csrf, Write: true}
			}
		}
	}
	geoSnapshot203(t, f, svc, c, countries["HK"])
	geoLocks203(t, f, svc, c)
	// A parent rename changes today's derived path, never the child's source or original adoption.
	leaf := geoID203(t, f, "440305")
	before := b.call("GET", "/api/v1/geo-areas/"+leaf, nil, "", 200)
	old := b.call("GET", "/api/v1/geo-areas/"+leaf+"?sourceVersion="+before["headSourceVersion"].(string), nil, "", 200)
	parent := b.call("GET", "/api/v1/geo-areas/"+gd, nil, "", 200)
	pr := parent["selectedRecord"].(map[string]any)
	corrected := map[string]any{"expectedSourceVersion": parent["headSourceVersion"], "expectedRecordRevision": pr["revision"], "changes": map[string]any{"name": "广东显示更正203"}}
	submit := foundation.NewID()
	saved := b.call("POST", "/api/v1/geo-areas/"+gd+"/records/"+pr["recordId"].(string)+"/correct", corrected, submit, 200)
	if saved["outcome"] != "accepted" {
		t.Fatal(saved)
	}
	now := b.call("GET", "/api/v1/geo-areas/"+leaf, nil, "", 200)
	if now["headSourceVersion"] != before["headSourceVersion"] || reflect.DeepEqual(now["geoAreaPath"], before["geoAreaPath"]) {
		t.Fatal("parent rename path/source")
	}
	again := b.call("GET", "/api/v1/geo-areas/"+leaf+"?sourceVersion="+before["headSourceVersion"].(string), nil, "", 200)
	if !reflect.DeepEqual(old, again) {
		t.Fatal("original source changed")
	}
	reader.call("POST", "/api/v1/geo-areas/"+gd+"/records/"+pr["recordId"].(string)+"/correct", corrected, foundation.NewID(), 403)
	b.call("POST", "/api/v1/geo-areas/"+gd+"/records/"+pr["recordId"].(string)+"/correct", corrected, foundation.NewID(), 409)
	for _, field := range []string{"code", "scheme", "countryRegionId", "parentAreaId", "source", "effectiveDate"} {
		bad := map[string]any{"expectedSourceVersion": saved["sourceVersion"], "expectedRecordRevision": saved["targetRevision"], "changes": map[string]any{field: "bad"}}
		b.call("POST", "/api/v1/geo-areas/"+gd+"/records/"+pr["recordId"].(string)+"/correct", bad, foundation.NewID(), 400)
	}
	b.call("POST", "/api/v1/geo-areas/"+gd+"/deactivate", map[string]any{"expectedSourceVersion": saved["sourceVersion"], "anchorRecordId": pr["recordId"], "effectiveDate": "1901-01-01", "operationType": "DEACTIVATE", "operationReason": "STOP_SELECTION"}, foundation.NewID(), 201)
	leafRecord := now["selectedRecord"].(map[string]any)
	b.call("POST", "/api/v1/geo-areas/"+leaf+"/records/"+leafRecord["recordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": now["headSourceVersion"], "expectedRecordRevision": leafRecord["revision"], "changes": map[string]any{"name": "南山显示更正203"}}, foundation.NewID(), 200)
	latest := b.call("GET", "/api/v1/geo-areas/"+leaf, nil, "", 200)
	for _, sample := range []struct {
		detail map[string]any
		exact  bool
	}{{latest, false}, {old, true}} {
		path := sample.detail["geoAreaPath"].([]any)
		last := path[len(path)-1].(map[string]any)
		record := sample.detail["selectedRecord"].(map[string]any)
		if last["id"] != leaf || last["name"] != record["name"] || sample.exact && (last["recordId"] != record["recordId"] || last["revision"] != record["revision"] || last["sourceVersion"] != sample.detail["resolvedSourceVersion"]) {
			t.Fatalf("self in current/historical path: %v", last)
		}
	}
	none, e := svc.Candidates(f.ctx, c, org.GeoAreaCandidateInput{CountryRegionID: countries["CN"], ParentAreaID: gd, AdoptedDate: "2026-01-01"})
	must(t, e)
	if len(none.Items) != 0 {
		t.Fatal("inactive ancestor remained selectable")
	}
	oldCandidates, e := svc.Candidates(f.ctx, c, org.GeoAreaCandidateInput{CountryRegionID: countries["CN"], ParentAreaID: gd, AdoptedDate: "1900-01-01", PageSize: 50})
	must(t, e)
	if len(oldCandidates.Items) != 21 {
		t.Fatal("historical selection")
	}
	// Real write-port resolution holds the same head row locks used by maintenance.
	must(t, svc.Store.WithinGeoAreaWrite(f.ctx, c, func(u app.Unit) error {
		a, e := app.ResolveReference(f.ctx, u, leaf, "1900-01-01")
		if e != nil {
			return e
		}
		if len(a.Parents) != 3 || a.Parents[2].ObjectID != leaf {
			return errors.New("incorrect adopted terminal chain")
		}
		u.Confirm()
		return nil
	}))
	if e = svc.Store.WithinGeoAreaWrite(f.ctx, c, func(u app.Unit) error { _, e := app.ResolveReference(f.ctx, u, leaf, "2026-01-01"); return e }); e == nil {
		t.Fatal("locked inactive adoption accepted")
	}
	// Loss of the derived path field suppresses the whole path, including old evidence.
	f.exec(`UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'geoAreaPath') WHERE principal_id=$1 AND action LIKE 'geoarea.%'`, reader.subject)
	hidden := reader.call("GET", "/api/v1/geo-areas/"+leaf, nil, "", 200)
	if _, ok := hidden["geoAreaPath"]; ok {
		t.Fatal("path leaked")
	}
	oldDetail := "/api/v1/geo-areas/" + leaf + "?sourceVersion=" + before["headSourceVersion"].(string)
	for _, url := range []string{"/api/v1/geo-areas/" + leaf, oldDetail} {
		got := reader.call("GET", url, nil, "", 200)
		if got["countryRegionName"] != "中国" || got["geoAreaPath"] != nil {
			t.Fatal("country label and ancestor path must have independent visibility", got["countryRegionName"])
		}
	}
	f.exec(`UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'name') WHERE principal_id=$1 AND action LIKE 'country.%'`, reader.subject)
	if reader.call("GET", "/api/v1/geo-areas/context", nil, "", 200)["filterCountries"] != nil {
		t.Fatal("country options exposed")
	}
	if reader.call("GET", "/api/v1/geo-areas?q=441900", nil, "", 200)["items"].([]any)[0].(map[string]any)["countryRegionName"] != nil {
		t.Fatal("country column exposed")
	}
	reader.call("GET", "/api/v1/geo-areas?filterField=countryRegionId&filterOperator=equals&filterValue="+countries["CN"], nil, "", 403)
	reader.call("POST", "/api/v1/geo-areas/navigation", map[string]any{"direction": "current", "anchorId": leaf, "query": map[string]any{"filter": map[string]any{"field": "countryRegionId", "operator": "equals", "value": countries["CN"]}}}, "", 403)
	for _, url := range []string{"/api/v1/geo-areas/" + leaf, oldDetail} {
		if reader.call("GET", url, nil, "", 200)["countryRegionName"] != nil {
			t.Fatal("restricted country label exposed")
		}
	}
	reader.call("GET", "/api/v1/geo-areas/candidates?countryRegionId="+countries["CN"]+"&adoptedDate=1900-01-01", nil, "", 403)
	t.Log("G3-V05/V06/V07/V09/V10: correction/history, protected structure, inactive ancestor, inherited adoption, real locked resolution and field revocation passed")
}
func geoCapacity203(t *testing.T, f *fixture, countries map[string]string) *browser {
	before := locationBusinessSnapshot204(t, f)
	// Login throttling remains enabled; separate fixture setup from the measured cohort.
	t.Log("G3-CAPACITY: 61s login cooldown outside measurement")
	time.Sleep(61 * time.Second)
	allSessions := make([]*browser, 20)
	for i := range allSessions {
		allSessions[i] = f.login("admin", f.password)
	}
	if path := os.Getenv("CHNHCM_G3_CPU_PROFILE"); path != "" {
		file, err := os.Create(path)
		must(t, err)
		must(t, pprof.StartCPUProfile(file))
		defer func() { pprof.StopCPUProfile(); must(t, file.Close()) }()
	}
	for _, region := range []struct{ code, parent, q string }{{"CN", "440000", "区"}, {"HK", "HK", "區"}, {"TW", "63000", "區"}} {
		for _, shape := range []string{"roots", "children", "search"} {
			independent := geographyCandidateOracle205(t, f, region.code, shape, region.parent, region.q)
			for _, concurrency := range []int{1, 20} {
				in := org.GeoAreaCandidateInput{CountryRegionID: countries[region.code], AdoptedDate: "2026-01-01", PageSize: 20}
				if shape == "children" {
					in.ParentAreaID = geoID203(t, f, region.parent)
				}
				if shape == "search" {
					in.Q = region.q
				}
				sessions := allSessions[:concurrency]
				expected := map[*browser]map[string]any{}
				for _, session := range sessions {
					expected[session] = session.call("GET", candidateURL203(in), nil, "", 200)
					independent(expected[session])
				}
				var wg sync.WaitGroup
				var mu sync.Mutex
				times := []float64{}
				successes := []float64{}
				failures := []error{}
				for _, c := range sessions {
					wg.Add(1)
					go func(c *browser) {
						defer wg.Done()
						for j := 0; j < 200/concurrency; j++ {
							start := time.Now()
							status, v, e := c.response("GET", candidateURL203(in), nil, "")
							if e == nil && status != 200 {
								e = fmt.Errorf("HTTP %d: %v", status, v)
							}
							ms := float64(time.Since(start).Microseconds()) / 1000
							mu.Lock()
							times = append(times, ms)
							if e != nil {
								failures = append(failures, e)
							} else if items, ok := v["items"].([]any); !ok || len(items) == 0 {
								failures = append(failures, fmt.Errorf("empty formal candidates"))
							} else if !reflect.DeepEqual(v, expected[c]) {
								failures = append(failures, errors.New("candidate response differs from this session's source-verified warm-up result"))
							} else {
								successes = append(successes, ms)
							}
							mu.Unlock()
						}
					}(c)
				}
				wg.Wait()
				sort.Float64s(times)
				p95 := times[189]
				sort.Float64s(successes)
				if len(successes) > 0 {
					t.Logf("G3-SUCCESS p50=%.3fms p95=%.3fms p99=%.3fms", successes[(len(successes)-1)/2], successes[(len(successes)*95-1)/100], successes[(len(successes)*99-1)/100])
				}
				t.Logf("G3-CAPACITY region=%s shape=%s sessions=%d n=%d errors=%d p95=%.3fms max=%.3fms adopted=2026-01-01", region.code, shape, concurrency, len(times), len(failures), p95, times[len(times)-1])
				if len(failures) > 0 || concurrency == 1 && p95 > 300 {
					t.Errorf("formal candidate correctness/single-session target failed: %v", failures)
				}
				if concurrency == 20 && p95 > 500 {
					t.Logf("G3-NONBLOCKING 20-session p95 %.3fms exceeds 500ms target; user decision 2026-11-14", p95)
				}
			}
		}
	}
	if !reflect.DeepEqual(before, locationBusinessSnapshot204(t, f)) {
		t.Fatal("G5 candidate measurement changed organization business facts")
	}
	t.Log("G5: 18 candidate groups independently verified against source rows/parent keys; all organization business tables unchanged")
	return allSessions[0]
}

func candidateURL203(in org.GeoAreaCandidateInput) string {
	q := url.Values{"countryRegionId": {in.CountryRegionID}, "adoptedDate": {in.AdoptedDate}, "pageSize": {"20"}}
	if in.ParentAreaID != "" {
		q.Set("parentAreaId", in.ParentAreaID)
	}
	if in.Q != "" {
		q.Set("q", in.Q)
	}
	return "/api/v1/geo-areas/candidates?" + q.Encode()
}

func geoLocks203(t *testing.T, f *fixture, svc app.Service, c access.Credentials) {
	id := geoID203(t, f, "110112")
	v, e := svc.Detail(f.ctx, c, org.DetailInput{ID: id})
	must(t, e)
	ready, release := make(chan struct{}), make(chan struct{})
	adopted := make(chan error, 1)
	written := make(chan error, 1)
	go func() {
		adopted <- svc.Store.WithinGeoAreaWrite(f.ctx, c, func(u app.Unit) error {
			a, e := app.ResolveReference(f.ctx, u, id, "2026-01-01")
			if e != nil {
				close(ready)
				return e
			}
			if a.Parents[len(a.Parents)-1].SourceVersion != v.HeadSourceVersion {
				close(ready)
				return errors.New("adoption did not lock original head")
			}
			close(ready)
			<-release
			u.Confirm()
			return nil
		})
	}()
	<-ready
	name := "通州锁内更正203"
	cmd := org.GeoAreaCommand{Action: "CORRECT", GeoAreaID: id, RecordID: v.SelectedRecord.ID, SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Correct: org.GeoAreaCorrectInput{ExpectedSourceVersion: v.HeadSourceVersion, ExpectedRecordRevision: v.SelectedRecord.Revision, Changes: org.GeoAreaChangesInput{"name": &name}}}
	go func() { _, e := svc.Maintain(f.ctx, c, cmd); written <- e }()
	select {
	case e := <-written:
		close(release)
		<-adopted
		t.Fatalf("writer escaped the held head lock: %v", e)
	case <-time.After(150 * time.Millisecond):
	}
	close(release)
	must(t, <-adopted)
	must(t, <-written)
	t.Log("G3-V09 real concurrent connections: adoption holds terminal head; display write waits then commits")
	// Commit succeeds while the caller loses the acknowledgement. Resolution
	// still returns this object's original source, even after another write.
	v, e = svc.Detail(f.ctx, c, org.DetailInput{ID: id})
	must(t, e)
	cmd.Correct.ExpectedSourceVersion = v.HeadSourceVersion
	cmd.Correct.ExpectedRecordRevision = v.SelectedRecord.Revision
	cmd.SubmissionID = foundation.NewID()
	name = "通州未知提交203"
	lost := app.Service{Store: geoLostAck203{svc.Store}}
	if _, e = lost.Maintain(f.ctx, c, cmd); e == nil {
		t.Fatal("missing injected commit acknowledgement error")
	}
	result, e := svc.Resolve(f.ctx, c, cmd.SubmissionID)
	must(t, e)
	if result.Outcome != "accepted" || result.GeoAreaID == nil || *result.GeoAreaID != id {
		t.Fatal(result)
	}
	again, e := svc.Resolve(f.ctx, c, cmd.SubmissionID)
	must(t, e)
	if !reflect.DeepEqual(result, again) {
		t.Fatal("original receipt changed")
	}
	t.Log("G3-V11 unknown acknowledgement recovered through the original GeoArea submission")
}

type geoLostAck203 struct{ app.Store }

func (s geoLostAck203) WithinGeoAreaWrite(ctx context.Context, c access.Credentials, fn func(app.Unit) error) error {
	if e := s.Store.WithinGeoAreaWrite(ctx, c, fn); e != nil {
		return e
	}
	return errors.New("injected lost acknowledgement after commit")
}

type geoSnapshotStore203 struct {
	app.Store
	after func() error
}

func (s geoSnapshotStore203) WithinGeoArea(ctx context.Context, c access.Credentials, fn func(app.Unit) error) error {
	return s.Store.WithinGeoArea(ctx, c, func(u app.Unit) error { return fn(&geoSnapshotUnit203{Unit: u, after: s.after}) })
}

type geoSnapshotUnit203 struct {
	app.Unit
	after func() error
}

func (u *geoSnapshotUnit203) CandidateRows(ctx context.Context, in org.GeoAreaCandidateInput, key, id string, n int) ([]app.CandidateRow, error) {
	rows, e := u.Unit.CandidateRows(ctx, in, key, id, n)
	if e != nil {
		return nil, e
	}
	if u.after != nil {
		after := u.after
		u.after = nil
		if e = after(); e != nil {
			return nil, e
		}
	}
	return rows, nil
}
func geoSnapshot203(t *testing.T, f *fixture, svc app.Service, c access.Credentials, country string) {
	id := geoID203(t, f, "HK")
	before, e := svc.Detail(f.ctx, c, org.DetailInput{ID: id})
	must(t, e)
	oldName := *before.SelectedRecord.Name
	name := oldName + "快照后更正203"
	wrapped := app.Service{Store: geoSnapshotStore203{Store: svc.Store, after: func() error {
		_, e := svc.Maintain(f.ctx, c, org.GeoAreaCommand{Action: "CORRECT", GeoAreaID: id, RecordID: before.SelectedRecord.ID, SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Correct: org.GeoAreaCorrectInput{ExpectedSourceVersion: before.HeadSourceVersion, ExpectedRecordRevision: before.SelectedRecord.Revision, Changes: org.GeoAreaChangesInput{"name": &name}}})
		return e
	}}}
	rows, e := wrapped.Candidates(f.ctx, c, org.GeoAreaCandidateInput{CountryRegionID: country, ParentAreaID: id, AdoptedDate: "2026-01-01"})
	must(t, e)
	if len(rows.Items) == 0 {
		t.Fatal("empty snapshot candidate page")
	}
	for _, r := range rows.Items {
		if len(r.Path) < 3 || r.Path[1].Name != oldName {
			t.Fatal("candidate mixed newer parent", r)
		}
	}
	fresh, e := svc.Candidates(f.ctx, c, org.GeoAreaCandidateInput{CountryRegionID: country, ParentAreaID: id, AdoptedDate: "2026-01-01"})
	must(t, e)
	if fresh.Items[0].Path[1].Name != name {
		t.Fatal("new request did not observe committed rename")
	}
	t.Log("G3-V08: mutation between cursor fetch and source decoding preserves one snapshot; next request observes new parent")
}

func TestGeoArea203HongKongFocused(t *testing.T) {
	f := fresh(t)
	must(t, f.runtime.InitializeCountryCatalog(f.ctx, access.LoginInput{Login: "admin", Password: f.password}))
	p, cmds := geoPartial203(t, f)
	restartTimelineReceipts(t, f, f.login("admin", f.password), nil, "geoarea")
	if f.scalar("SELECT count(*) FROM organization.geoarea") != 1 {
		t.Fatal("partial restart lost source")
	}
	svc := f.runtime.GeoArea.(app.Service)
	c := geoCredentials203(t, f, "admin", f.password)
	// Two independent sessions resume exactly the same interrupted package.
	c2 := geoCredentials203(t, f, "admin", f.password)
	closed, e := svc.Resolve(f.ctx, c, cmds[1].SubmissionID)
	must(t, e)
	if closed.Outcome != "not_applied" {
		t.Fatal(closed)
	}
	if e = svc.Store.WithinGeoAreaWrite(f.ctx, c, func(u app.Unit) error { return u.PublishCatalog(f.ctx, p, nil) }); e == nil {
		t.Fatal("incomplete package published")
	}
	if f.scalar("SELECT count(*) FROM organization.geoarea_package") != 0 {
		t.Fatal("failed publication did not roll back")
	}

	results := make(chan error, 2)
	for _, cred := range []access.Credentials{c, c2} {
		go func(cred access.Credentials) { results <- svc.InitializeCatalog(f.ctx, cred, p, cmds) }(cred)
	}
	must(t, <-results)
	must(t, <-results)
	if f.scalar("SELECT sum(cardinality(source_refs)) FROM organization.geoarea_history") != 21 {
		t.Fatal("concurrent package duplicated sources")
	}
	list, e := svc.List(f.ctx, c, org.GeoAreaQuery{PageSize: 100})
	must(t, e)
	if list.Count != 21 {
		t.Fatal(list.Count)
	}
	empty, e := svc.List(f.ctx, c, org.GeoAreaQuery{Page: int(^uint(0) >> 1), PageSize: 100})
	must(t, e)
	if empty.Count != 21 || len(empty.Items) != 0 {
		t.Fatal("out-of-range page", empty)
	}
	roots, e := svc.Candidates(f.ctx, c, org.GeoAreaCandidateInput{CountryRegionID: p.CountryID, AdoptedDate: "1900-01-01"})
	must(t, e)
	if len(roots.Items) != 3 {
		t.Fatal(roots)
	}
	for _, row := range roots.Items {
		if len(row.Path) != 2 {
			t.Fatal("root chain", row)
		}
	}
	geoSnapshot203(t, f, svc, c, p.CountryID)
	geoNavigationHTTP203(t, f, p.CountryID)
	again, e := svc.Resolve(f.ctx, c, cmds[1].SubmissionID)
	must(t, e)
	if again.Outcome != "not_applied" {
		t.Fatal("closed original submission overwritten")
	}
	geoPermissions203(t, f, svc, c, p.CountryID)
	geoStopOrdering203(t, f, svc, c)
	geoTerminal203(t, f, svc, c, p.CountryID)
	geoCountryStop203(t, f, svc, c, p.CountryID)
	restartTimelineReceipts(t, f, f.login("admin", f.password), []string{cmds[0].SubmissionID, cmds[2].SubmissionID}, "geoarea")
	t.Log("G3-HK: concurrent interrupted package resume, 21 unique sources, all list paths and root candidates passed")
}

func geoTerminal203(t *testing.T, f *fixture, svc app.Service, c access.Credentials, country string) {
	parent := geoID203(t, f, "HK")
	in := org.GeoAreaCandidateInput{CountryRegionID: country, ParentAreaID: parent, AdoptedDate: "2026-01-01"}
	children, e := svc.Candidates(f.ctx, c, in)
	must(t, e)
	for _, child := range children.Items {
		v, e := svc.Detail(f.ctx, c, org.DetailInput{ID: child.ID})
		must(t, e)
		_, e = svc.Maintain(f.ctx, c, org.GeoAreaCommand{Action: "DEACTIVATE", GeoAreaID: child.ID, SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Add: org.GeoAreaAddInput{ExpectedSourceVersion: v.HeadSourceVersion, AnchorRecordID: v.SelectedRecord.ID, EffectiveDate: "1902-01-01", OperationType: "DEACTIVATE", OperationReason: "STOP_SELECTION"}})
		must(t, e)
	}
	empty, e := svc.Candidates(f.ctx, c, in)
	must(t, e)
	if len(empty.Items) != 0 {
		t.Fatal("stopped children remained selectable")
	}
	roots, e := svc.Candidates(f.ctx, c, org.GeoAreaCandidateInput{CountryRegionID: country, AdoptedDate: "2026-01-01", Q: "HK"})
	must(t, e)
	if len(roots.Items) != 1 || roots.Items[0].ID != parent || roots.Items[0].Terminal {
		t.Fatal("filtered/stopped children changed structural terminal status", roots)
	}
	e = svc.Store.WithinGeoAreaWrite(f.ctx, c, func(u app.Unit) error { _, e := app.ResolveReference(f.ctx, u, parent, "2026-01-01"); return e })
	var fault *foundation.Fault
	if !errors.As(e, &fault) || fault.Code != "M02_REFERENCE_UNAVAILABLE" {
		t.Fatal("nonterminal reference accepted", e)
	}
	t.Log("G3-V08/V09: all direct children inactive and filtered out; root remains structurally nonterminal and write reference rejects it")
}

func geoNavigationHTTP203(t *testing.T, f *fixture, country string) {
	b := f.login("admin", f.password)
	list := b.call("GET", "/api/v1/geo-areas?countryRegionId="+country+"&filterField=level&filterOperator=equals&filterValue=region", nil, "", 200)
	b.call("GET", "/api/v1/geo-areas?rootOnly=true", nil, "", 400)
	b.call("GET", "/api/v1/geo-areas?filterField=level&filterOperator=equals&filterValue=invalid", nil, "", 400)
	b.call("GET", "/api/v1/geo-areas?filterField=countryRegionId&filterOperator=contains&filterValue="+country, nil, "", 400)
	byCountry := b.call("GET", "/api/v1/geo-areas?filterField=countryRegionId&filterOperator=equals&filterValue="+country, nil, "", 200)
	if byCountry["count"] != float64(21) || list["count"] != float64(3) {
		t.Fatal("wrong country or level count", byCountry["count"], list["count"])
	}
	for _, raw := range byCountry["items"].([]any) {
		row := raw.(map[string]any)
		if row["countryRegionId"] != country || row["countryRegionName"] == nil {
			t.Fatal("missing country column")
		}
	}
	rows := list["items"].([]any)
	anchor := rows[0].(map[string]any)["id"]
	query := map[string]any{"countryRegionId": country, "filter": map[string]any{"field": "level", "operator": "equals", "value": "region"}, "status": "all", "sort": "code", "pageSize": 20}
	body := map[string]any{"anchorId": anchor, "direction": "next", "query": query}
	n := b.call("POST", "/api/v1/geo-areas/navigation", body, "", 200)
	if n["targetId"] != rows[1].(map[string]any)["id"] {
		t.Fatal("HTTP navigation disagrees with list", n)
	}
	query["filter"] = map[string]any{"field": "code", "operator": "equals", "value": rows[0].(map[string]any)["code"]}
	n = b.call("POST", "/api/v1/geo-areas/navigation", body, "", 200)
	if n["targetId"] != nil {
		t.Fatal("filtered last anchor has neighbor", n)
	}
	query["businessAt"] = "1900-01-01"
	b.call("POST", "/api/v1/geo-areas/navigation", body, "", 400)
	t.Log("G3-V08/V12 HTTP navigation: omitted optional fields, level query, column predicate and forbidden businessAt checked")
}

func geoCountryStop203(t *testing.T, f *fixture, svc app.Service, c access.Credentials, country string) {
	for _, size := range []int{20, 50, 100} {
		v, e := svc.Candidates(f.ctx, c, org.GeoAreaCandidateInput{CountryRegionID: country, AdoptedDate: "1900-01-01", PageSize: size})
		must(t, e)
		if len(v.Items) != 3 || v.HasMore || v.Cursor != nil {
			t.Fatalf("root final page at size %d: %+v", size, v)
		}
	}
	before := f.scalar("SELECT sum(cardinality(source_refs)) FROM organization.geoarea_history")
	v, e := f.runtime.Country.Detail(f.ctx, c, org.DetailInput{ID: country})
	must(t, e)
	_, e = f.runtime.Country.Maintain(f.ctx, c, org.CountryCommand{Action: "DEACTIVATE", CountryRegionID: country, SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Add: org.CountryAddInput{ExpectedSourceVersion: v.HeadSourceVersion, AnchorRecordID: v.SelectedRecord.ID, EffectiveDate: "1901-01-01", OperationType: "DEACTIVATE", OperationReason: "STOP_SELECTION"}})
	must(t, e)
	_, e = svc.Candidates(f.ctx, c, org.GeoAreaCandidateInput{CountryRegionID: country, AdoptedDate: "2026-01-01"})
	var fault *foundation.Fault
	if !errors.As(e, &fault) || fault.Code != "M02_REFERENCE_UNAVAILABLE" {
		t.Fatalf("inactive country candidates: %v", e)
	}
	if f.scalar("SELECT sum(cardinality(source_refs)) FROM organization.geoarea_history") != before {
		t.Fatal("country stop cascaded area facts")
	}
	id := geoID203(t, f, "EST")
	var adoption d.Adoption
	var firstRecord string
	must(t, svc.Store.WithinGeoArea(f.ctx, c, func(u app.Unit) error {
		v, e := u.Load(f.ctx, id, "")
		if e == nil {
			adoption = v.Timeline.Records[0].Data.Adoption
			firstRecord = v.Timeline.Records[0].ID
		}
		return e
	}))
	area, e := svc.Detail(f.ctx, c, org.DetailInput{ID: id, SelectedRecordID: firstRecord})
	must(t, e)
	name := "国家停用后的区域显示更正"
	_, e = svc.Maintain(f.ctx, c, org.GeoAreaCommand{Action: "CORRECT", GeoAreaID: id, RecordID: area.SelectedRecord.ID, SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Correct: org.GeoAreaCorrectInput{ExpectedSourceVersion: area.HeadSourceVersion, ExpectedRecordRevision: area.SelectedRecord.Revision, Changes: org.GeoAreaChangesInput{"name": &name}}})
	must(t, e)
	must(t, svc.Store.WithinGeoArea(f.ctx, c, func(u app.Unit) error {
		v, e := u.Load(f.ctx, id, "")
		if e == nil && !reflect.DeepEqual(adoption, v.Timeline.Records[0].Data.Adoption) {
			return errors.New("display correction rewrote original adoption")
		}
		return e
	}))
	old, e := svc.Candidates(f.ctx, c, org.GeoAreaCandidateInput{CountryRegionID: country, AdoptedDate: "1900-01-01"})
	must(t, e)
	if len(old.Items) != 3 {
		t.Fatal("country historic candidate qualification changed")
	}
	t.Log("G3-V06/V08: country stop rejects new adoption without cascading area history; display correction keeps original chain; 20/50/100 last pages checked")
}

// Hold an actual successful business write immediately before its transaction
// commits, so the competing reference must re-read the locked terminal head.
type geoBeforeCommit203 struct {
	app.Store
	ready   chan struct{}
	release chan struct{}
}

func (s geoBeforeCommit203) WithinGeoAreaWrite(ctx context.Context, c access.Credentials, fn func(app.Unit) error) error {
	return s.Store.WithinGeoAreaWrite(ctx, c, func(u app.Unit) error {
		e := fn(u)
		close(s.ready)
		if e == nil {
			<-s.release
		}
		return e
	})
}

func geoStopOrdering203(t *testing.T, f *fixture, svc app.Service, c access.Credentials) {
	for _, tc := range []struct {
		code           string
		referenceFirst bool
	}{{"CW", true}, {"WC", false}} {
		t.Run("stop_order_"+tc.code, func(t *testing.T) {
			id := geoID203(t, f, tc.code)
			v, e := svc.Detail(f.ctx, c, org.DetailInput{ID: id})
			must(t, e)
			cmd := org.GeoAreaCommand{Action: "DEACTIVATE", GeoAreaID: id, SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Add: org.GeoAreaAddInput{ExpectedSourceVersion: v.HeadSourceVersion, AnchorRecordID: v.SelectedRecord.ID, EffectiveDate: "1901-01-01", OperationType: "DEACTIVATE", OperationReason: "STOP_SELECTION"}}
			ready, release := make(chan struct{}), make(chan struct{})
			first, second := make(chan error, 1), make(chan error, 1)
			var once sync.Once
			defer once.Do(func() { close(release) })
			reference := func(hold bool) error {
				return svc.Store.WithinGeoAreaWrite(f.ctx, c, func(u app.Unit) error {
					_, e := app.ResolveReference(f.ctx, u, id, "2026-01-01")
					if hold {
						close(ready)
						if e == nil {
							<-release
						}
					}
					if e == nil {
						u.Confirm()
					}
					return e
				})
			}
			if tc.referenceFirst {
				go func() { first <- reference(true) }()
				<-ready
				go func() { _, e := svc.Maintain(f.ctx, c, cmd); second <- e }()
			} else {
				held := app.Service{Store: geoBeforeCommit203{svc.Store, ready, release}}
				go func() { _, e := held.Maintain(f.ctx, c, cmd); first <- e }()
				<-ready
				go func() { second <- reference(false) }()
			}
			select {
			case e := <-second:
				t.Fatalf("competing transaction escaped held head: %v", e)
			case <-time.After(150 * time.Millisecond):
			}
			once.Do(func() { close(release) })
			must(t, <-first)
			e = <-second
			if tc.referenceFirst {
				must(t, e)
			} else {
				var fault *foundation.Fault
				if !errors.As(e, &fault) || fault.Code != "M02_REFERENCE_UNAVAILABLE" {
					t.Fatalf("stopped head accepted: %v", e)
				}
			}
		})
	}
	t.Log("G3-V09 real stop ordering: reference-first permits both commits; stop-first blocks then rejects reference")
}

func geoPermissions203(t *testing.T, f *fixture, svc app.Service, c access.Credentials, country string) {
	b := f.login("admin", f.password)
	reader := f.login("reader", f.readPassword)
	id := geoID203(t, f, "KLN")
	v := b.call("GET", "/api/v1/geo-areas/"+id, nil, "", 200)
	r := v["selectedRecord"].(map[string]any)
	saved := b.call("POST", "/api/v1/geo-areas/"+id+"/records/"+r["recordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": v["headSourceVersion"], "expectedRecordRevision": r["revision"], "changes": map[string]any{"nameEn": "Historical English"}}, foundation.NewID(), 200)
	b.call("POST", "/api/v1/geo-areas/"+id+"/records", map[string]any{"expectedSourceVersion": saved["sourceVersion"], "anchorRecordId": r["recordId"], "effectiveDate": "1901-01-01", "operationType": "CHANGE", "operationReason": "DISPLAY_DATA_UPDATE", "changes": map[string]any{"name": "九龙现用名称"}}, foundation.NewID(), 201)
	f.exec(`UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'nameEn') WHERE principal_id=$1 AND action='geoarea.history.read'`, reader.subject)
	old := reader.call("GET", "/api/v1/geo-areas/candidates?countryRegionId="+country+"&adoptedDate=1900-01-01&q=KLN", nil, "", 200)
	items := old["items"].([]any)
	if len(items) != 1 {
		t.Fatal(old)
	}
	if _, ok := items[0].(map[string]any)["nameEn"]; ok {
		t.Fatal("historical optional field leaked")
	}
	// A visible country and a permitted path do not confer read access to the selected name.
	f.exec(`UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'name') WHERE principal_id=$1 AND action='geoarea.history.read'`, reader.subject)
	history := reader.call("GET", "/api/v1/geo-areas/"+id+"?sourceVersion="+saved["sourceVersion"].(string), nil, "", 200)
	if history["geoAreaPath"] != nil || history["selectedRecord"].(map[string]any)["name"] != nil || history["countryRegionName"] != "香港" {
		t.Fatal("restricted historical self projected", history)
	}
	currentWithName := reader.call("GET", "/api/v1/geo-areas/"+id, nil, "", 200)
	self := currentWithName["geoAreaPath"].([]any)
	if self[len(self)-1].(map[string]any)["name"] != "九龙现用名称" {
		t.Fatal("missing current self", self)
	}
	f.exec(`DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='geoarea.history.read'`, reader.subject)
	old = reader.call("GET", "/api/v1/geo-areas/candidates?countryRegionId="+country+"&adoptedDate=1900-01-01&q=KLN", nil, "", 200)
	if len(old["items"].([]any)) != 0 {
		t.Fatal("historical area remained selectable without history read")
	}
	reader.call("GET", "/api/v1/geo-areas/"+id+"?selectedRecordId="+r["recordId"].(string), nil, "", 403)
	f.exec(`UPDATE access.subject_grant SET read_fields='{}' WHERE principal_id=$1 AND action='geoarea.read'`, reader.subject)
	mo := geoCountry203(t, f, "MO")
	basic := reader.call("GET", "/api/v1/geo-areas/candidates?countryRegionId="+mo+"&adoptedDate=1900-01-01", nil, "", 200)
	if basic["mode"] != "basic_address" {
		t.Fatal(basic)
	}
	f.exec(`UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'name') WHERE principal_id=$1 AND action='country.read'`, reader.subject)
	reader.call("GET", "/api/v1/geo-areas/candidates?countryRegionId="+mo+"&adoptedDate=1900-01-01", nil, "", 403)
	current := b.call("GET", "/api/v1/geo-areas/"+id, nil, "", 200)
	currentRecord := current["selectedRecord"].(map[string]any)
	clearBody := map[string]any{"expectedSourceVersion": current["headSourceVersion"], "expectedRecordRevision": currentRecord["revision"], "changes": map[string]any{"nameEn": nil}}
	b.call("POST", "/api/v1/geo-areas/"+id+"/records/"+currentRecord["recordId"].(string)+"/correct", clearBody, foundation.NewID(), 200)
	current = b.call("GET", "/api/v1/geo-areas/"+id, nil, "", 200)
	currentRecord = current["selectedRecord"].(map[string]any)
	if currentRecord["nameEn"] != nil {
		t.Fatal("explicit null did not clear English")
	}
	count := f.scalar("SELECT sum(cardinality(source_refs)) FROM organization.geoarea_history")
	unchanged := map[string]any{"expectedSourceVersion": current["headSourceVersion"], "expectedRecordRevision": currentRecord["revision"], "changes": map[string]any{"name": currentRecord["name"]}}
	noop := b.call("POST", "/api/v1/geo-areas/"+id+"/records/"+currentRecord["recordId"].(string)+"/correct", unchanged, foundation.NewID(), 200)
	if noop["outcome"] != "no_change" || f.scalar("SELECT sum(cardinality(source_refs)) FROM organization.geoarea_history") != count {
		t.Fatal("no_change grew history", noop)
	}
	t.Log("G3-V05: explicit English null and no_change without source growth passed")
	t.Log("G3-V10: historical optional fields intersect permissions; no history excludes old record; Macau needs country fields but no phantom area fields")
}

func geoAllSources203(t *testing.T, f *fixture) {
	p, e := geoareaseed.Fixed()
	must(t, e)
	countries, e := countryseed.Fixed()
	must(t, e)
	ids := map[string]string{}
	for _, r := range countries.Countries {
		cmd, e := countries.Command(f.instance, foundation.NewID(), r)
		must(t, e)
		ids[r.SourceKey] = cmd.Import.ID
	}
	rows, e := p.Prepare(f.instance, ids)
	must(t, e)
	mo, e := geoareaseed.Macau()
	must(t, e)
	extra, e := mo.Prepare(f.instance, ids)
	must(t, e)
	rows = append(rows, extra...)
	svc := f.runtime.GeoArea.(app.Service)
	c := geoCredentials203(t, f, "admin", f.password)
	for start := 0; start < len(rows); start += 100 {
		end := min(start+100, len(rows))
		must(t, svc.Store.WithinGeoArea(f.ctx, c, func(u app.Unit) error {
			for _, row := range rows[start:end] {
				v, e := u.Load(f.ctx, row.Node.ID, "")
				if e != nil {
					return e
				}
				if len(v.Timeline.Records) != 1 {
					return fmt.Errorf("%s initial record count", row.SourceKey)
				}
				r := v.Timeline.Records[0]
				if r.Date != "1900-01-01" || r.State != "active" || r.Data.Name != row.Name || r.Data.NameEn != nil || r.Data.Structure != row.Node.Structure || v.Timeline.Origin.Key != row.SourceKey || v.Timeline.Origin.SourceVersion != row.SourceVersion {
					return fmt.Errorf("%s imported field mismatch", row.SourceKey)
				}
				a := r.Data.Adoption
				if a.Date != "1900-01-01" || a.Country.ObjectID != row.Node.Structure.CountryID {
					return fmt.Errorf("%s adoption identity", row.SourceKey)
				}
				country, e := u.Country(f.ctx, a.Country.ObjectID, a.Country.SourceVersion)
				if e != nil {
					return e
				}
				cr := country.Timeline.Records[0]
				if a.Country.RecordID != cr.ID || a.Country.Revision != cr.Revision {
					return fmt.Errorf("%s country actual source mismatch", row.SourceKey)
				}
				parent := row.Node.Structure.ParentID
				for i := len(a.Parents) - 1; i >= 0; i-- {
					ref := a.Parents[i]
					if ref.ObjectID != parent {
						return fmt.Errorf("%s adopted parent order", row.SourceKey)
					}
					pv, e := u.Load(f.ctx, parent, ref.SourceVersion)
					if e != nil {
						return e
					}
					pr := pv.Timeline.Records[0]
					if ref.RecordID != pr.ID || ref.Revision != pr.Revision {
						return fmt.Errorf("%s parent actual source mismatch", row.SourceKey)
					}
					parent = pv.Timeline.Origin.ParentID
				}
				if parent != "" {
					return fmt.Errorf("%s truncated adoption chain", row.SourceKey)
				}
			}
			return nil
		}))
	}
	t.Logf("G3-V01/V07: %d imported records independently compared with fixed input, dates, English null, structure and every actual adopted source/record/revision", len(rows))
}

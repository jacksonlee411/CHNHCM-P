//go:build integration

package bootstrap_test

import (
	"chnhcm.local/chnhcm/db/migrations"
	"chnhcm.local/chnhcm/internal/bootstrap"
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	baseapp "chnhcm.local/chnhcm/internal/contexts/organization/application"
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/country"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/countryseed"
	orgdb "chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCountry200SourceFieldRemovalMigration(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	login, e := f.runtime.Access.Login(f.ctx, access.LoginInput{Login: "admin", Password: f.password}, foundation.NewID())
	must(t, e)
	p, e := countryseed.Fixed()
	must(t, e)
	cmd, e := p.Command(f.instance, login.Principal.ID, p.Countries[0])
	must(t, e)
	_, e = f.runtime.Country.Maintain(f.ctx, access.Credentials{Token: login.Token, CSRF: login.Principal.CSRF, Write: true}, cmd)
	must(t, e)
	snapshot := func() map[string]string {
		out := map[string]string{}
		for _, table := range []string{"country_region", "country_dictionary", "country_shape", "country_history", "country_terminal", "country_package"} {
			var digest string
			must(t, f.admin.QueryRow(f.ctx, "SELECT md5(coalesce(string_agg(to_jsonb(t)::text,'' ORDER BY to_jsonb(t)::text),'')) FROM organization."+table+" t").Scan(&digest))
			out[table] = digest
		}
		return out
	}
	before := snapshot()
	removeEmptyLegalEntity206(t, f)
	// Reconstruct the actual 00009 prefix in this disposable DB, including
	// removing later Location/GeoArea extensions and the tree index. Never create a ledger hole.
	removeEnterprise304Extension(f)
	f.exec(`DELETE FROM access.subject_grant WHERE action LIKE 'geoarea.%' OR action LIKE 'location.%';
 DROP TABLE organization.location_selection_block,organization.location_selection_draft,organization.location_read,organization.location_terminal,organization.location,organization.location_history,organization.location_shape,organization.location_dictionary CASCADE;
 DROP FUNCTION organization.location_stamp_head(),organization.location_append_guard(),organization.location_head_guard(),organization.location_closure_guard(),organization.location_terminal_guard();
 DROP TABLE organization.geoarea_package,organization.geoarea_read,organization.geoarea_terminal,organization.geoarea,organization.geoarea_history,organization.geoarea_shape,organization.geoarea_dictionary CASCADE;
 DROP FUNCTION organization.geoarea_stamp_head(),organization.geoarea_append_guard(),organization.geoarea_head_guard(),organization.geoarea_closure_guard(),organization.geoarea_terminal_guard();
 DROP INDEX organization.department_tree_parent;
 DELETE FROM public.goose_db_version WHERE version_id IN (10,11,12,13,16,23);
 DO $$ DECLARE c record; BEGIN
 FOR c IN SELECT conname,pg_get_constraintdef(oid) def FROM pg_constraint WHERE conrelid='access.subject_grant'::regclass AND contype='c' LOOP
 IF position('country.%' IN c.def)>0 OR c.conname IN ('country_resource_fields','subject_grant_scope_kind_check','subject_grant_action_check') THEN
 EXECUTE format('ALTER TABLE access.subject_grant DROP CONSTRAINT %I',c.conname);
 END IF; END LOOP; END $$;`)
	prior, e := migrations.Files.ReadFile("00009_country_catalog.sql")
	must(t, e)
	start := strings.Index(string(prior), "ALTER TABLE access.subject_grant ADD CHECK(scope_kind")
	end := strings.Index(string(prior), "CREATE TABLE organization.country_package")
	if start < 0 || end < start {
		t.Fatal("00009 authorization fixture boundaries")
	}
	f.exec(string(prior)[start:end])
	f.exec(`UPDATE access.subject_grant SET read_fields=array_append(read_fields,'source'),write_fields=array_append(write_fields,'source') WHERE action LIKE 'country.%'`)
	policy := f.scalar("SELECT policy_no FROM access.policy_guard WHERE principal_id=$1", b.subject)
	var otherGrants string
	must(t, f.admin.QueryRow(f.ctx, `SELECT md5(string_agg(to_jsonb(g)::text,'' ORDER BY grant_id)) FROM access.subject_grant g WHERE action NOT LIKE 'country.%'`).Scan(&otherGrants))
	if e := bootstrap.CheckCurrentStorage(f.ctx, f.admin); e == nil {
		t.Fatal("runtime admitted pending field migration")
	}
	must(t, bootstrap.Migrate(f.ctx, f.admin, migrations.Files))
	must(t, bootstrap.CheckCurrentStorage(f.ctx, f.admin))
	if f.scalar("SELECT count(*) FROM access.subject_grant WHERE 'source'=ANY(read_fields) OR 'source'=ANY(write_fields)") != 0 || f.scalar("SELECT policy_no FROM access.policy_guard WHERE principal_id=$1", b.subject) != policy+1 {
		t.Fatal("removed field or policy invalidation missing")
	}
	var afterGrants string
	must(t, f.admin.QueryRow(f.ctx, `SELECT md5(string_agg(to_jsonb(g)::text,'' ORDER BY grant_id)) FROM access.subject_grant g WHERE action NOT LIKE 'country.%'`).Scan(&afterGrants))
	if otherGrants != afterGrants || !reflect.DeepEqual(before, snapshot()) {
		t.Fatal("migration changed other grants or country facts")
	}
	must(t, bootstrap.Migrate(f.ctx, f.admin, migrations.Files))
	if f.scalar("SELECT policy_no FROM access.policy_guard WHERE principal_id=$1", b.subject) != policy+1 {
		t.Fatal("repeated migration changed policy")
	}
	for _, field := range []string{"read_fields", "write_fields"} {
		if _, e := f.admin.Exec(f.ctx, "UPDATE access.subject_grant SET "+field+"=array_append("+field+",'source') WHERE action='country.read'"); e == nil {
			t.Fatal("removed source field can be granted", field)
		}
	}
	detail := b.call("GET", "/api/v1/country-regions/"+cmd.Import.ID, nil, "", 200)
	if _, ok := detail["origin"]; ok || detail["code"] != p.Countries[0].Code {
		t.Fatal("country resource retains provider evidence or lost code")
	}
	t.Log("00009→00010: country facts and other grants unchanged; source grants removed; policy advances once; regrant rejected")
}

func TestCountry200FixedCatalog(t *testing.T) {
	f := fresh(t)
	login, e := f.runtime.Access.Login(f.ctx, access.LoginInput{Login: "admin", Password: f.password}, foundation.NewID())
	must(t, e)
	c := access.Credentials{Token: login.Token, CSRF: login.Principal.CSRF, Write: true}
	store := f.runtime.Organization.(baseapp.Service).Store.(*orgdb.Store)
	svc := app.Service{Store: store}
	p, e := countryseed.Fixed()
	must(t, e)
	for i, row := range p.Countries {
		cmd, e := p.Command(f.instance, login.Principal.ID, row)
		must(t, e)
		result, e := svc.Maintain(f.ctx, c, cmd)
		if e != nil {
			t.Fatalf("row %d %s: %v", i, row.Code, e)
		}
		if result.Outcome != "accepted" || result.CountryRegionID == nil || *result.CountryRegionID != cmd.Import.ID {
			t.Fatal("source identity not retained", row.Code)
		}
	}
	before := f.scalar("SELECT sum(cardinality(source_refs)) FROM organization.country_history")
	for _, row := range p.Countries {
		cmd, e := p.Command(f.instance, login.Principal.ID, row)
		must(t, e)
		result, e := svc.Maintain(f.ctx, c, cmd)
		must(t, e)
		if !result.Replayed {
			t.Fatal("not replayed", row.Code)
		}
	}
	if f.scalar("SELECT sum(cardinality(source_refs)) FROM organization.country_history") != before {
		t.Fatal("repeated import grew history")
	}
	c.Write = false
	last, e := svc.List(f.ctx, c, org.ListInput{Page: int(^uint(0) >> 1), PageSize: 100})
	must(t, e)
	if last.Count != 251 || len(last.Items) != 0 {
		t.Fatal("out-of-range page must be empty")
	}
	expected := map[string]org.CountryCommand{}
	for _, row := range p.Countries {
		cmd, e := p.Command(f.instance, login.Principal.ID, row)
		must(t, e)
		expected[cmd.Import.ID] = cmd
	}
	for page := 1; page <= 3; page++ {
		list, e := svc.List(f.ctx, c, org.ListInput{Page: page, PageSize: 100})
		must(t, e)
		if list.Count != 251 {
			t.Fatal(list.Count)
		}
		for _, item := range list.Items {
			detail, e := svc.Detail(f.ctx, c, org.DetailInput{ID: item.ID})
			must(t, e)
			if detail.SelectedRecord == nil || *detail.SelectedRecord.EffectiveDate != "1900-01-01" {
				t.Fatal("missing initial source")
			}
			source := expected[item.ID].Import
			if detail.Code == nil || *detail.Code != source.Origin.Code || *detail.SelectedRecord.Name != source.Name || detail.SelectedRecord.NameEn == nil || !reflect.DeepEqual(*detail.SelectedRecord.NameEn, source.NameEn) || detail.SelectedRecord.PhoneCode == nil || !reflect.DeepEqual(*detail.SelectedRecord.PhoneCode, source.PhoneCode) {
				t.Fatal("source fields differ", source.Origin.Code)
			}
		}
	}
}

func TestCountry200CatalogRestartAndMaintenance(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	reader := f.login("reader", f.readPassword)
	login, e := f.runtime.Access.Login(f.ctx, access.LoginInput{Login: "admin", Password: f.password}, foundation.NewID())
	must(t, e)
	creds := access.Credentials{Token: login.Token, CSRF: login.Principal.CSRF, Write: true}
	p, e := countryseed.Fixed()
	must(t, e)
	// Interrupted after ten normal commits: no completion marker may exist.
	for _, row := range p.Countries[:10] {
		cmd, e := p.Command(f.instance, login.Principal.ID, row)
		must(t, e)
		_, e = f.runtime.Country.Maintain(f.ctx, creds, cmd)
		must(t, e)
	}
	if f.scalar("SELECT count(*) FROM organization.country_package") != 0 {
		t.Fatal("partial package published")
	}
	cmd, e := p.Command(f.instance, login.Principal.ID, p.Countries[0])
	must(t, e)
	id := cmd.Import.ID
	path := "/api/v1/country-regions/" + id
	original := b.call("GET", path, nil, "", 200)
	if _, ok := original["origin"]; ok {
		t.Fatal("provider evidence leaked into country resource")
	}
	for _, v := range original["capabilities"].(map[string]any)["readFields"].([]any) {
		if v == "source" {
			t.Fatal("removed source field remains in capabilities")
		}
	}
	r := original["selectedRecord"].(map[string]any)
	correction := map[string]any{"expectedSourceVersion": original["headSourceVersion"], "expectedRecordRevision": r["revision"], "changes": map[string]any{"name": "本地显示名称", "phoneCode": nil}}
	key := foundation.NewID()
	saved := b.call("POST", path+"/records/"+r["recordId"].(string)+"/correct", correction, key, 200)
	if saved["outcome"] != "accepted" {
		t.Fatal(saved)
	}
	// A previously closed, unwritten attempt stays closed; restart creates a new
	// submission for this still-absent source identity.
	closedCommand, e := p.Command(f.instance, login.Principal.ID, p.Countries[10])
	must(t, e)
	closedResult, e := f.runtime.Country.Resolve(f.ctx, creds, closedCommand.SubmissionID)
	must(t, e)
	if closedResult.Outcome != "not_applied" {
		t.Fatal(closedResult)
	}
	// Complete through the real login/application entry, then repeat the whole package.
	must(t, f.runtime.InitializeCountryCatalog(f.ctx, access.LoginInput{Login: "admin", Password: f.password}))
	closedResult, e = f.runtime.Country.Resolve(f.ctx, creds, closedCommand.SubmissionID)
	must(t, e)
	if closedResult.Outcome != "not_applied" {
		t.Fatal("closed attempt overwritten")
	}
	before := f.scalar("SELECT sum(cardinality(source_refs)) FROM organization.country_history")
	must(t, f.runtime.InitializeCountryCatalog(f.ctx, access.LoginInput{Login: "admin", Password: f.password}))
	if f.scalar("SELECT sum(cardinality(source_refs)) FROM organization.country_history") != before || before != 252 || f.scalar("SELECT count(*) FROM organization.country_package") != 1 {
		t.Fatal("completion/repetition facts")
	}
	if e = f.runtime.InitializeCountryCatalog(f.ctx, access.LoginInput{Login: "reader", Password: f.readPassword}); e == nil {
		t.Fatal("reader imported")
	}
	current := b.call("GET", path, nil, "", 200)
	r = current["selectedRecord"].(map[string]any)
	if r["name"] != "本地显示名称" || r["phoneCode"] != nil {
		t.Fatal("restart overwrote display data")
	}
	history := b.call("GET", path+"?sourceVersion="+original["headSourceVersion"].(string), nil, "", 200)
	if history["selectedRecord"].(map[string]any)["name"] != p.Countries[0].Name {
		t.Fatal("original source overwritten")
	}
	replay := b.call("POST", path+"/records/"+r["recordId"].(string)+"/correct", correction, key, 200)
	if replay["sourceVersion"] != saved["sourceVersion"] || replay["replayed"] != true {
		t.Fatal("original response not recovered")
	}
	reader.call("GET", path, nil, "", 200)
	reader.call("POST", path+"/records/"+r["recordId"].(string)+"/correct", correction, foundation.NewID(), 403)
	b.call("POST", path+"/records/"+r["recordId"].(string)+"/correct", correction, foundation.NewID(), 409)
	correction["expectedSourceVersion"] = current["headSourceVersion"]
	correction["expectedRecordRevision"] = r["revision"]
	correction["changes"] = map[string]any{"name": "本地显示名称"}
	nochange := b.call("POST", path+"/records/"+r["recordId"].(string)+"/correct", correction, foundation.NewID(), 200)
	if nochange["outcome"] != "no_change" {
		t.Fatal(nochange)
	}
	correction["changes"] = map[string]any{"code": "ZZ"}
	b.call("POST", path+"/records/"+r["recordId"].(string)+"/correct", correction, foundation.NewID(), 400)
	// Initial 1900 is not reused by subsequent changes; a valid pre-enterprise date works.
	add := map[string]any{"expectedSourceVersion": current["headSourceVersion"], "anchorRecordId": r["recordId"], "effectiveDate": "1901-01-01", "operationType": "CHANGE", "operationReason": "DISPLAY_DATA_UPDATE", "changes": map[string]any{"nameEn": "Local label"}}
	change := b.call("POST", path+"/records", add, foundation.NewID(), 201)
	stop := map[string]any{"expectedSourceVersion": change["sourceVersion"], "anchorRecordId": change["selectedRecordId"], "effectiveDate": "1902-01-01", "operationType": "DEACTIVATE", "operationReason": "STOP_SELECTION"}
	stopped := b.call("POST", path+"/deactivate", stop, foundation.NewID(), 201)
	if b.call("GET", path, nil, "", 200)["status"] != "inactive" {
		t.Fatal("stop not projected")
	}
	stop["expectedSourceVersion"] = stopped["sourceVersion"]
	stop["anchorRecordId"] = stopped["selectedRecordId"]
	stop["effectiveDate"] = "1903-01-01"
	stop["operationType"] = "REACTIVATE"
	stop["operationReason"] = "RESUME_SELECTION"
	b.call("POST", path+"/reactivate", stop, foundation.NewID(), 201)
	if b.call("GET", path, nil, "", 200)["status"] != "active" {
		t.Fatal("resume not projected")
	}
	logs := b.call("GET", path+"/changes", nil, "", 200)
	if logs["total"] != float64(5) {
		t.Fatal("missing exact log", logs["total"])
	}
	closed := foundation.NewID()
	if b.call("POST", "/api/v1/country-regions/submissions/"+closed+"/resolve", nil, "", 200)["outcome"] != "not_applied" {
		t.Fatal("unknown not closed")
	}
	b.call("POST", path+"/reactivate", stop, closed, 409)
	// Revoked field rights apply to current reads, history and original results.
	f.exec(`UPDATE access.subject_grant SET read_fields=array_remove(array_remove(read_fields,'name'),'effectiveDate') WHERE principal_id=$1 AND action LIKE 'country.%'`, b.subject)
	f.exec(`UPDATE access.policy_guard SET policy_no=policy_no+1 WHERE principal_id=$1`, b.subject)
	hidden := b.call("GET", path, nil, "", 200)
	hr := hidden["selectedRecord"].(map[string]any)
	if _, ok := hr["name"]; ok {
		t.Fatal("revoked name leaked")
	}
	if _, ok := hidden["origin"]; ok {
		t.Fatal("provider evidence leaked into country resource")
	}
	recovered := b.call("POST", "/api/v1/country-regions/submissions/"+key+"/resolve", nil, "", 200)
	if _, ok := recovered["effectiveDate"]; ok {
		t.Fatal("revoked date leaked in recovery")
	}
	f.exec(`DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='country.history.read'`, b.subject)
	f.exec(`UPDATE access.policy_guard SET policy_no=policy_no+1 WHERE principal_id=$1`, b.subject)
	b.call("GET", path+"?sourceVersion="+original["headSourceVersion"].(string), nil, "", 403)
	b.call("GET", path+"?selectedRecordId="+r["recordId"].(string), nil, "", 403)
	b.call("POST", "/api/v1/country-regions/submissions/"+key+"/resolve", nil, "", 403)
}

type countryFaultStore struct {
	app.Store
	rollback, lostAck bool
}

func (s countryFaultStore) WithinCountryWrite(ctx context.Context, c access.Credentials, fn func(app.Unit) error) error {
	e := s.Store.WithinCountryWrite(ctx, c, func(u app.Unit) error {
		if e := fn(u); e != nil {
			return e
		}
		if s.rollback {
			return context.Canceled
		}
		return nil
	})
	if e == nil && s.lostAck {
		return context.DeadlineExceeded
	}
	return e
}
func TestCountry200SubmissionFaultsAndConcurrency(t *testing.T) {
	f := fresh(t)
	login, e := f.runtime.Access.Login(f.ctx, access.LoginInput{Login: "admin", Password: f.password}, foundation.NewID())
	must(t, e)
	creds := access.Credentials{Token: login.Token, CSRF: login.Principal.CSRF, Write: true}
	svc := f.runtime.Country.(app.Service)
	p, e := countryseed.Fixed()
	must(t, e)
	cmd, e := p.Command(f.instance, login.Principal.ID, p.Countries[0])
	must(t, e)
	results := make(chan org.CountryResult, 8)
	failures := make(chan error, 8)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); r, e := svc.Maintain(f.ctx, creds, cmd); results <- r; failures <- e }()
	}
	wg.Wait()
	close(results)
	close(failures)
	for e := range failures {
		must(t, e)
	}
	source := ""
	for r := range results {
		if r.Outcome != "accepted" || r.SourceVersion == nil {
			t.Fatal(r)
		}
		if source != "" && source != *r.SourceVersion {
			t.Fatal("duplicate source")
		}
		source = *r.SourceVersion
	}
	if f.scalar("SELECT sum(cardinality(source_refs)) FROM organization.country_history") != 1 {
		t.Fatal("concurrent duplicates grew history")
	}
	creds.Write = false
	d, e := svc.Detail(f.ctx, creds, org.DetailInput{ID: cmd.Import.ID})
	must(t, e)
	creds.Write = true
	name := "故障恢复显示名"
	correction := org.CountryCommand{Action: "CORRECT", CountryRegionID: cmd.Import.ID, RecordID: d.SelectedRecord.ID, SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Correct: org.CountryCorrectInput{ExpectedSourceVersion: d.HeadSourceVersion, ExpectedRecordRevision: d.SelectedRecord.Revision, Changes: org.CountryChangesInput{"name": &name}}}
	failed := app.Service{Store: countryFaultStore{Store: svc.Store, rollback: true}}
	r, e := failed.Maintain(f.ctx, creds, correction)
	if e == nil || r.Outcome != "" {
		t.Fatal("uncommitted result escaped", r, e)
	}
	r, e = svc.Resolve(f.ctx, creds, correction.SubmissionID)
	must(t, e)
	if r.Outcome != "not_applied" {
		t.Fatal(r)
	}
	correction.SubmissionID = foundation.NewID()
	lost := app.Service{Store: countryFaultStore{Store: svc.Store, lostAck: true}}
	r, e = lost.Maintain(f.ctx, creds, correction)
	if e == nil || r.Outcome != "" {
		t.Fatal("lost ack exposed confirmed result")
	}
	r, e = svc.Resolve(f.ctx, creds, correction.SubmissionID)
	must(t, e)
	if r.Outcome != "accepted" || r.SourceVersion == nil {
		t.Fatal("committed source not recovered", r)
	}
	replay, e := svc.Maintain(f.ctx, creds, correction)
	must(t, e)
	if !replay.Replayed || *replay.SourceVersion != *r.SourceVersion {
		t.Fatal("recovery changed original source")
	}
	correction.Digest[0] = 1
	if _, e = svc.Maintain(f.ctx, creds, correction); e == nil {
		t.Fatal("digest mismatch accepted")
	}
	if f.scalar("SELECT sum(cardinality(source_refs)) FROM organization.country_history") != 2 {
		t.Fatal("faults grew history")
	}
	// Same full submission UUID in another object namespace cannot recover country facts.
	b := f.login("admin", f.password)
	b.call("POST", "/api/v1/business-units", createBody("COUNTRY_NAMESPACE", "2026-01-01"), cmd.SubmissionID, 201)
	recovered := b.call("POST", "/api/v1/country-regions/submissions/"+cmd.SubmissionID+"/resolve", nil, "", 200)
	if recovered["countryRegionId"] != cmd.Import.ID {
		t.Fatal("submission namespace crossed")
	}
	// Exercise the country's physical bindings, not only the shared BU guards.
	for _, sql := range []string{
		`UPDATE organization.country_region SET head_no=head_no+1`,
		`UPDATE organization.country_history SET envelope=decode(repeat('00',32),'hex')`,
		`UPDATE organization.country_shape SET parts=ARRAY[decode('00','hex')]`,
		`DELETE FROM organization.country_terminal`,
	} {
		if _, e = f.admin.Exec(f.ctx, sql); e == nil {
			t.Fatal("country source protection accepted mutation", sql)
		}
	}
	if f.scalar("SELECT sum(cardinality(source_refs)) FROM organization.country_history") != 2 {
		t.Fatal("rejected SQL changed country history")
	}
	// Catalog scope is instance-only, and SQL retains the old NULL-field constraint.
	if _, e = f.admin.Exec(f.ctx, `UPDATE access.subject_grant SET scope_kind='enterprise' WHERE principal_id=$1 AND action='country.read'`, b.subject); e == nil {
		t.Fatal("country granted through enterprise")
	}
	if _, e = f.admin.Exec(f.ctx, `UPDATE access.subject_grant SET read_fields=ARRAY[NULL]::text[] WHERE principal_id=$1 AND action='country.read'`, b.subject); e == nil {
		t.Fatal("NULL field constraint removed")
	}
}

func TestCountry200ListCapacity(t *testing.T) {
	if os.Getenv("CHNHCM_COUNTRY_LOAD") != "1" {
		t.Skip("country capacity requires an isolated measurement window")
	}
	f := fresh(t)
	must(t, f.runtime.InitializeCountryCatalog(f.ctx, access.LoginInput{Login: "admin", Password: f.password}))
	input, e := countryseed.Fixed()
	must(t, e)
	slices.SortFunc(input.Countries, func(a, b countryseed.Row) int { return strings.Compare(a.Code, b.Code) })
	before := locationBusinessSnapshot204(t, f)
	clients := make([]*browser, 20)
	for i := range clients {
		clients[i] = f.login("reader", f.readPassword)
	}
	for _, sessions := range []int{1, 20} {
		t.Run(fmt.Sprintf("sessions_%d", sessions), func(t *testing.T) {
			type sample struct {
				Millis float64
				OK     bool
			}
			samples := make(chan sample, 200)
			var wg sync.WaitGroup
			gate := make(chan struct{})
			// Warm each connection; the following 200 full HTTP requests are all measured.
			for _, b := range clients[:sessions] {
				b.call("GET", "/api/v1/country-regions", nil, "", 200)
			}
			for i := 0; i < sessions; i++ {
				wg.Add(1)
				go func(worker int, b *browser) {
					defer wg.Done()
					<-gate
					for j := 0; j < 200/sessions; j++ {
						// First/middle/last pages, with an oracle from the frozen input.
						page := []int{1, 7, 13}[(worker*(200/sessions)+j)%3]
						started := time.Now()
						status, body, e := b.response("GET", "/api/v1/country-regions?page="+strconv.Itoa(page), nil, "")
						millis := float64(time.Since(started).Microseconds()) / 1000
						ok := e == nil && status == 200 && body["count"] == float64(251)
						if ok {
							items, valid := body["items"].([]any)
							start, end := (page-1)*20, min(page*20, len(input.Countries))
							ok = valid && len(items) == end-start
							if ok {
								for index, item := range items {
									v, valid := item.(map[string]any)
									id, _ := v["id"].(string)
									want := input.Countries[start+index]
									ok = ok && valid && foundation.ValidID(id) && v["code"] == want.Code && v["name"] == want.Name && v["effectiveDate"] == want.Date && v["status"] == "active"
								}
							}
						}
						samples <- sample{millis, ok}
					}
				}(i, clients[i])
			}
			close(gate)
			wg.Wait()
			close(samples)
			values := []float64{}
			successes := []float64{}
			errors := 0
			for v := range samples {
				values = append(values, v.Millis)
				if !v.OK {
					errors++
				} else {
					successes = append(successes, v.Millis)
				}
			}
			slices.Sort(values)
			p95 := values[189]
			slices.Sort(successes)
			percentile := func(p int) float64 {
				if len(successes) == 0 {
					return 0
				}
				return successes[(len(successes)*p+99)/100-1]
			}
			target := 300.
			if sessions == 20 {
				target = 500
			}
			t.Logf("sessions=%d requests=%d errors=%d all_p95_ms=%.3f success_p50_ms=%.3f success_p95_ms=%.3f success_p99_ms=%.3f max_ms=%.3f target_ms=%.0f", sessions, len(values), errors, p95, percentile(50), percentile(95), percentile(99), values[len(values)-1], target)
			t.Logf("sorted_all_ms=%v", values)
			if errors != 0 || sessions == 1 && p95 > target {
				t.Errorf("country list capacity failed")
			}
			if sessions == 20 && p95 > target {
				t.Log("G5-NONBLOCKING: 20-session P95 exceeds 500ms; preserve the measured result and follow-up responsibility")
			}
		})
	}
	if !reflect.DeepEqual(before, locationBusinessSnapshot204(t, f)) {
		t.Fatal("country capacity changed business facts")
	}
	t.Log("G5: all organization business tables unchanged after 400 HTTP requests; country first/middle/last pages independently match the frozen input")
}

//go:build integration

package bootstrap_test

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestLegalEntity206PairsAndPermissions(t *testing.T) {
	h := legalSetup206(t)
	f, b := h.f, h.b
	f.exec("CREATE TABLE public.legal206_grants AS SELECT * FROM access.subject_grant")
	restore := func() {
		h.policy("DELETE FROM access.subject_grant; INSERT INTO access.subject_grant SELECT * FROM public.legal206_grants")
	}
	v := h.create("PAIR206", "CN", "legal_person")
	id := v["id"].(string)
	first := legalRecord206(v)["recordId"].(string)
	h.add(v, "ADD", "2030-01-01", map[string]any{"name": "独立未来资料"}, 201)
	v = h.detail(id, first)
	insert := map[string]any{"expectedSourceVersion": v["headSourceVersion"], "anchorRecordId": first, "deactivationDate": "2000-01-01", "reactivationDate": "2001-01-01", "deactivationReason": "ENTITY_CLOSURE", "reactivationReason": "ENTITY_RESUMPTION"}
	b.call("POST", legal206Path+"/"+id+"/lifecycle-pairs/insert", insert, foundation.NewID(), 200)
	v = h.refresh(v)
	pairBody := func(v map[string]any) map[string]any {
		rows := v["timeline"].([]any)
		stop, resume := rows[1].(map[string]any), rows[2].(map[string]any)
		return map[string]any{"expectedSourceVersion": v["headSourceVersion"], "deactivationRecordId": stop["recordId"], "deactivationRevision": stop["revision"], "reactivationRecordId": resume["recordId"], "reactivationRevision": resume["revision"]}
	}
	pair := pairBody(v)
	pair["deactivationDate"] = "2000-02-01"
	pair["reactivationDate"] = "2002-01-01"
	pair["deactivationReason"] = "ENTITY_CLOSURE"
	pair["reactivationReason"] = "ENTITY_RESUMPTION"
	pair["changes"] = map[string]any{"registrationNumber": "002"}
	b.call("POST", legal206Path+"/"+id+"/lifecycle-pairs/maintain", pair, foundation.NewID(), 200)
	v = h.refresh(v)
	rows := v["timeline"].([]any)
	stop, resume := rows[1].(map[string]any), rows[2].(map[string]any)
	if stop["registrationNumber"] != "001Ab" || resume["registrationNumber"] != "002" || stop["adoptionDate"] != "1900-01-01" || resume["adoptionDate"] != "2002-01-01" {
		t.Fatal("pair facts", stop, resume)
	}
	pair = pairBody(v)
	pair["deactivationDate"] = "2000-03-01"
	pair["reactivationDate"] = "2002-01-01"
	pair["deactivationReason"] = "ENTITY_CLOSURE"
	pair["reactivationReason"] = "ENTITY_RESUMPTION"
	b.call("POST", legal206Path+"/"+id+"/lifecycle-pairs/maintain", pair, foundation.NewID(), 200)
	v = h.refresh(v)
	if v["timeline"].([]any)[2].(map[string]any)["revision"] != resume["revision"] {
		t.Fatal("untouched resume changed")
	}
	b.call("POST", legal206Path+"/"+id+"/lifecycle-pairs/delete", pairBody(v), foundation.NewID(), 200)
	v = h.refresh(v)
	if len(v["timeline"].([]any)) != 2 {
		t.Fatal("non-atomic pair deletion")
	}
	// Independent date permission and hidden current/history fields.
	h.policy("DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='legalentity.record.redate'", b.subject)
	h.correct(v, nil, map[string]any{"newEffectiveDate": "1901-01-01"}, 403)
	restore()
	h.policy("UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'registrationNumber') WHERE principal_id=$1 AND action LIKE 'legalentity.%'", b.subject)
	hidden := h.refresh(v)
	if _, ok := legalRecord206(hidden)["registrationNumber"]; ok {
		t.Fatal("hidden field leaked")
	}
	b.call("GET", legal206Path+"?sort=registrationNumber", nil, "", 403)
	b.call("GET", legal206Path+"?filterField=registrationNumber&filterOperator=equals&filterValue=001Ab", nil, "", 403)
	restore()
	// Explicit and implicit writes: deleting a change would alter the following stop.
	a := h.create("IMPLICIT206", "MO", "branch")
	h.add(a, "ADD", "2000-01-01", map[string]any{"registrationNumber": "new"}, 201)
	a = h.refresh(a)
	change := legalRecord206(a)["recordId"].(string)
	h.add(a, "DEACTIVATE", "2001-01-01", nil, 201)
	a = h.detail(a["id"].(string), change)
	h.policy("UPDATE access.subject_grant SET write_fields=array_remove(write_fields,'registrationNumber') WHERE principal_id=$1 AND action='legalentity.record.delete'", b.subject)
	b.call("POST", legal206Path+"/"+a["id"].(string)+"/records/"+change+"/delete", map[string]any{"expectedSourceVersion": a["headSourceVersion"], "expectedRecordRevision": legalRecord206(a)["revision"]}, foundation.NewID(), 403)
	restore()
	h.policy("UPDATE access.subject_grant SET write_fields=array_remove(write_fields,'registrationType') WHERE principal_id=$1 AND action='legalentity.record.correct'", b.subject)
	h.correct(a, map[string]any{"registrationNumber": "new2"}, nil, 200)
	a = h.detail(a["id"].(string), change)
	h.correct(a, map[string]any{"registrationType": "证2"}, nil, 403)
	restore()
	h.policy("DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='legalentity.history.read'", b.subject)
	b.call("GET", legal206Path+"/"+id+"?sourceVersion="+v["headSourceVersion"].(string), nil, "", 403)
	b.call("GET", legal206Path+"/"+id+"/changes", nil, "", 403)
	restore()
	h.policy("UPDATE access.subject_grant SET scope_id=$2 WHERE principal_id=$1 AND action LIKE 'legalentity.%'", b.subject, foundation.NewID())
	b.call("GET", legal206Path+"/"+id, nil, "", 403)
	b.call("POST", legal206Path, h.body("OTHER-TENANT", "MO", "other"), foundation.NewID(), 403)
	restore()
	// Missing first/default date at a later enterprise start is separately tested by API.
	for _, sql := range []string{"UPDATE organization.legal_entity SET code='rewritten' WHERE code='PAIR206'", "DELETE FROM organization.legal_entity_history", "UPDATE organization.legal_entity_shape SET parts=parts", "DELETE FROM organization.legal_entity_terminal"} {
		if _, e := f.admin.Exec(f.ctx, sql); e == nil {
			t.Fatal("immutable business fact accepted mutation", sql)
		}
	}
	t.Log("pairs, individual date rights, hidden fields, final implicit write authorization, enterprise scope and immutable storage verified")
}

func TestLegalEntity206ExactCountry(t *testing.T) {
	h := legalSetup206(t)
	b := h.b
	v := h.create("EXACT206", "US", "other")
	name := legalRecord206(v)["countryName"]
	source := v["headSourceVersion"].(string)
	path := "/api/v1/country-regions/" + h.countries["US"]
	directory := b.call("GET", path, nil, "", 200)
	r := directory["selectedRecord"].(map[string]any)
	b.call("POST", path+"/records/"+r["recordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": directory["headSourceVersion"], "expectedRecordRevision": r["revision"], "changes": map[string]any{"name": "合成国家目录新名称"}}, foundation.NewID(), 200)
	h.correct(v, map[string]any{"name": "仅改名称", "registrationNumber": "new"}, nil, 200)
	v = h.refresh(v)
	if legalRecord206(v)["countryName"] != name {
		t.Fatal("unchanged reference adopted new label")
	}
	newer := h.create("EXACT-NEW206", "US", "branch")
	if legalRecord206(newer)["countryName"] != "合成国家目录新名称" {
		t.Fatal("new reference stale")
	}
	contextFor := func(v map[string]any) map[string]any {
		r := legalRecord206(v)
		return b.call("POST", legal206Path+"/reference-context", map[string]any{"action": "CORRECT", "legalEntityId": v["id"], "expectedSourceVersion": v["headSourceVersion"], "recordId": r["recordId"], "expectedRecordRevision": r["revision"]}, "", 200)
	}
	preserved := contextFor(v)
	if preserved["countryName"] != legalRecord206(v)["countryName"] || preserved["countryRestricted"] != false {
		t.Fatal("editing context must reauthorize the exact original label", preserved)
	}
	lateBody := h.body("ADOPTED2020", "US", "other")
	lateBody["effectiveDate"] = "2020-01-01"
	late := b.call("POST", legal206Path, lateBody, foundation.NewID(), 201)
	lateView := h.detail(late["legalEntityId"].(string), "")
	directory = b.call("GET", path, nil, "", 200)
	r = directory["selectedRecord"].(map[string]any)
	b.call("POST", path+"/deactivate", map[string]any{"expectedSourceVersion": directory["headSourceVersion"], "anchorRecordId": r["recordId"], "effectiveDate": "2000-01-01", "operationType": "DEACTIVATE", "operationReason": "STOP_SELECTION"}, foundation.NewID(), 201)
	preserved = contextFor(lateView)
	if preserved["countryName"] != "合成国家目录新名称" || preserved["countryRestricted"] != false {
		t.Fatal("ineligible candidate must not hide authorized old label")
	}
	for _, raw := range preserved["countries"].([]any) {
		if raw.(map[string]any)["id"] == h.countries["US"] {
			t.Fatal("stopped country offered as new candidate")
		}
	}
	h.correct(v, map[string]any{"registrationNumber": "after-stop"}, nil, 200)
	v = h.refresh(v)
	h.correct(v, nil, map[string]any{"newEffectiveDate": "2001-01-01"}, 409)
	h.add(v, "DEACTIVATE", "2001-01-01", nil, 201)
	v = h.refresh(v)
	h.add(v, "REACTIVATE", "2002-01-01", nil, 409)
	h.policy("DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='country.history.read'", b.subject)
	hidden := h.refresh(v)
	if legalRecord206(hidden)["countryName"] != nil || legalRecord206(hidden)["countryRestricted"] != true {
		t.Fatal("old label leaked")
	}
	preserved = contextFor(hidden)
	if preserved["countryName"] != nil || preserved["countryRestricted"] != true {
		t.Fatal("editing context must revoke the original label", preserved)
	}
	old := b.call("GET", legal206Path+"/"+v["id"].(string)+"?sourceVersion="+source, nil, "", 200)
	if legalRecord206(old)["countryName"] != nil {
		t.Fatal("old source leaked label")
	}
	// Global eligibility has no GeoArea grant dependency.
	h.policy("DELETE FROM access.subject_grant WHERE principal_id=$1 AND action LIKE 'geoarea.%'", b.subject)
	h.create("NO-GEO206", "CN", "branch")
	h.policy("DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='country.read'", b.subject)
	b.call("POST", legal206Path, h.body("NO-COUNTRY206", "CN", "branch"), foundation.NewID(), 403)
	t.Log("global country adoption and exact labels survive rename/stop; current/history permissions and recovery eligibility verified")
}

func legalExport206(t *testing.T, h *legal206, body map[string]any, status int) [][]string {
	raw, e := json.Marshal(body)
	must(t, e)
	req, e := http.NewRequest("POST", h.f.server.URL+legal206Path+"/export", strings.NewReader(string(raw)))
	must(t, e)
	req.Header.Set("Origin", h.f.server.URL)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", h.b.csrf)
	response, e := h.b.client.Do(req)
	must(t, e)
	defer response.Body.Close()
	data, e := io.ReadAll(response.Body)
	must(t, e)
	if response.StatusCode != status {
		t.Fatalf("export status=%d expected=%d body=%s", response.StatusCode, status, data)
	}
	if status != 200 {
		return nil
	}
	rows, e := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(data), "\xef\xbb\xbf"))).ReadAll()
	must(t, e)
	return rows
}
func TestLegalEntity206SelectionNavigation(t *testing.T) {
	h := legalSetup206(t)
	b := h.b
	for i := 0; i < 25; i++ {
		h.create(fmt.Sprintf("SELECT206-%02d", i), "MO", "branch")
	}
	query := map[string]any{"q": "SELECT206-", "countryRegionId": h.countries["MO"], "filterField": "subjectType", "filterOperator": "is", "filterValue": "branch"}
	start := b.call("POST", legal206Path+"/selection/start", map[string]any{"query": query, "selection": map[string]any{"mode": "allMatching", "excludedIds": []string{}}, "page": 1, "pageSize": 20}, "", 200)
	sum := start["summary"].(map[string]any)
	if sum["selectedCount"] != float64(25) {
		t.Fatal(start)
	}
	ids := []any{}
	for _, item := range start["page"].(map[string]any)["items"].([]any) {
		ids = append(ids, item.(map[string]any)["id"])
	}
	remove := ids[0].(string)
	edit := map[string]any{"query": query, "draftId": sum["draftId"], "expectedRevision": sum["revision"], "removeIds": []string{remove}, "page": 2, "pageSize": 20, "sort": "code"}
	page := b.call("POST", legal206Path+"/selection/resolve", edit, "", 200)
	sum = page["summary"].(map[string]any)
	if sum["selectedCount"] != float64(24) {
		t.Fatal(page)
	}
	b.call("POST", legal206Path+"/selection/resolve", edit, "", 409)
	newV := h.create("SELECT206-NEW", "MO", "branch")
	changed := h.detail(ids[1].(string), "")
	h.correct(changed, map[string]any{"subjectType": "other", "registrationNumber": "000Z"}, nil, 200)
	exportBody := map[string]any{"query": query, "draftId": sum["draftId"], "expectedRevision": sum["revision"], "sort": "code"}
	rows := legalExport206(t, h, exportBody, 200)
	if len(rows) != 25 || len(rows[0]) != 10 {
		t.Fatal("csv dimensions", len(rows), rows[0])
	}
	seen := false
	for _, row := range rows[1:] {
		if row[0] == "SELECT206-NEW" || row[0] == "SELECT206-00" {
			t.Fatal("membership recaptured")
		}
		if row[0] == "SELECT206-01" {
			seen = true
			if row[4] != "其他组织" || row[6] != "000Z" {
				t.Fatal("current values not exported", row)
			}
		}
	}
	if !seen {
		t.Fatal("captured member disappeared")
	}
	nav := b.call("POST", legal206Path+"/navigation", map[string]any{"query": query, "anchorId": remove, "direction": "next", "sort": "code"}, "", 200)
	if nav["targetId"] != ids[2] || nav["count"] != float64(25) {
		t.Fatal("full-condition navigation", nav, newV["id"])
	}
	h.policy("UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'registrationNumber') WHERE principal_id=$1 AND action='legalentity.export'", b.subject)
	legalExport206(t, h, exportBody, 403)
	before := legalSnapshot206(t, h.f)
	b.call("POST", legal206Path+"/selection/cancel", map[string]any{"draftId": sum["draftId"], "expectedRevision": sum["revision"]}, "", 204)
	if !reflect.DeepEqual(before, legalSnapshot206(t, h.f)) {
		t.Fatal("selection mutated business history")
	}
	if h.f.scalar("SELECT count(*) FROM organization.legal_entity_selection_draft") != 0 {
		t.Fatal("draft retained")
	}
}

// Used only by disposable schema-prefix fixtures, never by runtime migration.
func removeEmptyLegalEntity206(t *testing.T, f *fixture) {
	t.Helper()
	if f.scalar("SELECT count(*) FROM pg_class WHERE oid=to_regclass('organization.legal_purpose_source')") > 0 {
		if f.scalar("SELECT (SELECT count(*) FROM organization.legal_purpose_source)+(SELECT count(*) FROM staffing.assignment_policy_source)") != 0 {
			t.Fatal("prefix fixture contains published T02 rules")
		}
		if f.scalar("SELECT count(*) FROM pg_class WHERE oid=to_regclass('organization.legal_purpose_terminal')") > 0 {
			if f.scalar("SELECT (SELECT count(*) FROM organization.legal_purpose_terminal)+(SELECT count(*) FROM staffing.assignment_policy_terminal)") != 0 {
				t.Fatal("prefix fixture contains T02 receipts")
			}
			f.exec("DROP TABLE organization.legal_purpose_terminal, staffing.assignment_policy_terminal")
		}
		f.exec(`DELETE FROM access.subject_grant WHERE action LIKE 'legalpurpose.%' OR action LIKE 'assignmentpolicy.%';
      ALTER TABLE access.subject_grant DROP CONSTRAINT readiness_rule_scope;
      DROP TABLE organization.legal_purpose_member, organization.legal_purpose_outbox,organization.legal_purpose_source;
      DROP FUNCTION organization.legal_purpose_member_guard();
      DELETE FROM public.goose_db_version WHERE version_id IN (32,33);`)
	}
	if f.scalar("SELECT (to_regclass('staffing.appointment_version') IS NOT NULL)::int") > 0 {
		if f.scalar("SELECT count(*) FROM staffing.appointment_version") != 0 {
			t.Fatal("prefix fixture contains decisions")
		}
		f.exec("DROP TABLE access.appointment_issuer_fixture; DELETE FROM public.goose_db_version WHERE version_id=39")
	}
	if f.scalar("SELECT (to_regclass('staffing.employment') IS NOT NULL)::int") > 0 {
		if f.scalar("SELECT (SELECT count(*) FROM staffing.employment)+(SELECT count(*) FROM staffing.hire_terminal)+(SELECT count(*) FROM access.employment_grant)") != 0 {
			t.Fatal("prefix fixture contains employment data")
		}
		f.exec("DROP TABLE IF EXISTS access.employment_succession_fixture")
		f.exec("DROP TABLE access.employment_lifecycle_fixture,access.employment_promotion_fixture,access.employment_probation_fixture,access.employment_parallel_fixture,access.employment_history_fixture,access.employment_register_fixture,access.employment_location_fixture,access.employment_fixture,access.employment_grant; DELETE FROM public.goose_db_version WHERE version_id=37")
	}
	if f.scalar("SELECT (to_regclass('access.hire_identity_fixture') IS NOT NULL)::int") > 0 {
		f.exec("DROP TABLE access.hire_identity_fixture; DELETE FROM public.goose_db_version WHERE version_id=36")
	}
	if f.scalar("SELECT (to_regclass('recruiting.offer') IS NOT NULL)::int") != 0 &&
		f.scalar("SELECT (SELECT count(*) FROM recruiting.offer)+(SELECT count(*) FROM recruiting.offer_submission)+(SELECT count(*) FROM recruiting.position_intake)+(SELECT count(*) FROM access.offer_grant)") != 0 {
		t.Fatal("prefix fixture contains recruiting facts or grants")
	}
	if f.scalar("SELECT (to_regclass('recruiting.offer') IS NOT NULL)::int") != 0 {
		f.exec("DROP SCHEMA recruiting CASCADE; DROP TABLE access.offer_fixture,access.offer_grant; DELETE FROM public.goose_db_version WHERE version_id=50")
	}
	if f.scalar("SELECT count(*) FROM pg_namespace WHERE nspname='personnel'") > 0 {
		if f.scalar("SELECT (SELECT count(*) FROM personnel.person)+(SELECT count(*) FROM personnel.terminal)+(SELECT count(*) FROM access.person_grant)") != 0 {
			t.Fatal("prefix fixture would remove personnel facts")
		}
		f.exec("DROP SCHEMA personnel CASCADE; DROP TABLE access.person_sensitive_fixture,access.person_data_purpose,access.person_qualification_fixture,access.person_address_family_fixture,access.person_profile_fixture,access.person_fixture,access.person_grant,access.person_policy CASCADE; DROP FUNCTION access.person_grant_guard(); DELETE FROM public.goose_db_version WHERE version_id IN (29,30,31,34,38,40)")
	}

	if f.scalar(`SELECT count(*) FROM pg_namespace WHERE nspname='staffing'`) > 0 {
		if f.scalar("SELECT count(*) FROM pg_class WHERE oid=to_regclass('staffing.quota_terminal')") > 0 && f.scalar("SELECT (SELECT count(*) FROM staffing.quota_source)+(SELECT count(*) FROM staffing.quota_terminal)+(SELECT count(*) FROM staffing.quota_control)") > 0 {
			t.Fatal("prefix fixture contains quota facts")
		}
		if f.scalar("SELECT count(*) FROM pg_class WHERE oid=to_regclass('staffing.hire_rule_source')") > 0 && f.scalar("SELECT count(*) FROM staffing.hire_rule_source") != 0 {
			t.Fatal("prefix fixture contains a customer hire rule")
		}
		for _, table := range []string{"job_family", "job", "grade", "grade_ladder", "position", "position_hierarchy"} {
			if f.scalar("SELECT (SELECT count(*) FROM staffing."+table+")+(SELECT count(*) FROM staffing."+table+"_terminal)") > 0 {
				t.Fatal("prefix fixture would remove Staffing facts", table)
			}
		}
		f.exec(`ALTER TABLE access.subject_grant DROP CONSTRAINT IF EXISTS quota_catalog_scope; DELETE FROM access.subject_grant WHERE action LIKE 'jobfunction.%' OR action LIKE 'jobfamily.%' OR action LIKE 'job.%' OR action LIKE 'grade.%' OR action LIKE 'gradeladder.%' OR action LIKE 'position.%' OR action LIKE 'positionhierarchy.%' OR action LIKE 'quota.%' OR action LIKE 'staffingquota.%' OR action='replenishment.maintain'; DROP SCHEMA staffing CASCADE; DELETE FROM public.goose_db_version WHERE version_id IN (17,18,19,20,21,22,24,25,26,27,28,35,41,42,43,44,45,46,47,48,49,50,51);`)
	}

	t.Helper()
	if f.scalar("SELECT count(*) FROM organization.legal_entity") != 0 {
		t.Fatal("prefix fixture would remove legal entity facts")
	}
	f.exec(`DELETE FROM access.subject_grant WHERE action LIKE 'legalentity.%';
 DROP TABLE organization.legal_entity_selection_block,organization.legal_entity_selection_draft,organization.legal_entity_read,organization.legal_entity_terminal,organization.legal_entity,organization.legal_entity_history,organization.legal_entity_shape,organization.legal_entity_dictionary CASCADE;
 DROP FUNCTION organization.legal_entity_stamp_head(),organization.legal_entity_append_guard(),organization.legal_entity_head_guard(),organization.legal_entity_closure_guard(),organization.legal_entity_terminal_guard();
 DELETE FROM public.goose_db_version WHERE version_id=15;`)
}

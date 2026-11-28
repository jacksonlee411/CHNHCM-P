//go:build integration

package bootstrap_test

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/legalentity"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

const legal206Path = "/api/v1/legal-entities"

type legal206 struct {
	t         *testing.T
	f         *fixture
	b         *browser
	countries map[string]string
}

func legalSetup206(t *testing.T) *legal206 {
	f := freshWithEnterpriseStart(t, "1900-01-01")
	must(t, f.runtime.InitializeCountryCatalog(f.ctx, access.LoginInput{Login: "admin", Password: f.password}))
	h := &legal206{t, f, f.login("admin", f.password), map[string]string{}}
	for _, c := range []string{"CN", "HK", "MO", "TW", "US"} {
		h.countries[c] = geoCountry203(t, f, c)
	}
	return h
}
func legalRecord206(v map[string]any) map[string]any { return v["selectedRecord"].(map[string]any) }
func (h *legal206) body(code, country, kind string) map[string]any {
	return map[string]any{"code": code, "name": "合成法律实体 " + code, "nameEn": "Entity " + code, "countryRegionId": h.countries[country], "subjectType": kind, "registrationType": "登记证", "registrationNumber": "001Ab", "establishingOrganization": "设立单位原文", "operationType": "CREATE", "operationReason": "NEW_LEGAL_ENTITY"}
}
func (h *legal206) create(code, country, kind string) map[string]any {
	v := h.b.call("POST", legal206Path, h.body(code, country, kind), foundation.NewID(), 201)
	return h.detail(v["legalEntityId"].(string), "")
}
func (h *legal206) detail(id, record string) map[string]any {
	path := legal206Path + "/" + id
	if record != "" {
		path += "?selectedRecordId=" + record
	}
	return h.b.call("GET", path, nil, "", 200)
}
func (h *legal206) refresh(v map[string]any) map[string]any { return h.detail(v["id"].(string), "") }
func (h *legal206) correct(v map[string]any, changes map[string]any, extra map[string]any, status int) map[string]any {
	r := legalRecord206(v)
	body := map[string]any{"expectedSourceVersion": v["headSourceVersion"], "expectedRecordRevision": r["revision"]}
	if changes != nil {
		body["changes"] = changes
	}
	for k, v := range extra {
		body[k] = v
	}
	return h.b.call("POST", legal206Path+"/"+v["id"].(string)+"/records/"+r["recordId"].(string)+"/correct", body, foundation.NewID(), status)
}
func (h *legal206) add(v map[string]any, action, date string, changes map[string]any, status int) map[string]any {
	kind, reason, path := "CHANGE", "REGISTRATION_ADJUSTMENT", "/records"
	if action == "DEACTIVATE" {
		kind, reason, path = action, "ENTITY_CLOSURE", "/deactivate"
	}
	if action == "REACTIVATE" {
		kind, reason, path = action, "ENTITY_RESUMPTION", "/reactivate"
	}
	body := map[string]any{"expectedSourceVersion": v["headSourceVersion"], "anchorRecordId": legalRecord206(v)["recordId"], "effectiveDate": date, "operationType": kind, "operationReason": reason}
	if changes != nil {
		body["changes"] = changes
	}
	return h.b.call("POST", legal206Path+"/"+v["id"].(string)+path, body, foundation.NewID(), status)
}
func (h *legal206) policy(sql string, args ...any) {
	h.f.exec(sql, args...)
	h.f.exec("UPDATE access.policy_guard SET policy_no=policy_no+1")
}

func TestLegalEntity206Core(t *testing.T) {
	h := legalSetup206(t)
	f, b := h.f, h.b
	reader := f.login("reader", f.readPassword)
	originalCountries := locationBusinessSnapshot204(t, f)
	for i, kind := range []string{"legal_person", "branch", "unincorporated", "other"} {
		v := h.create(fmt.Sprintf("KIND-%d", i), []string{"CN", "HK", "TW", "US"}[i], kind)
		r := legalRecord206(v)
		if r["effectiveDate"] != "1900-01-01" || r["subjectType"] != kind || r["registrationNumber"] != "001Ab" || r["countryRestricted"] != false {
			t.Fatal("incorrect first facts", r)
		}
		reader.call("GET", legal206Path+"/"+v["id"].(string), nil, "", 200)
	}
	body := h.body("IDENTITY", "MO", "branch")
	sid := foundation.NewID()
	created := b.call("POST", legal206Path, body, sid, 201)
	id := created["legalEntityId"].(string)
	v := h.detail(id, "")
	first := legalRecord206(v)["recordId"].(string)
	replay := b.call("POST", legal206Path, body, sid, 201)
	if replay["legalEntityId"] != id || replay["replayed"] != true {
		t.Fatal("replay changed identity")
	}
	body["name"] = "different"
	b.call("POST", legal206Path, body, sid, 409)
	b.call("POST", legal206Path, body, foundation.NewID(), 409)
	reader.call("POST", legal206Path, h.body("READONLY", "MO", "other"), foundation.NewID(), 403)
	// Same literal code and same submission UUID in a separate object namespace.
	b.call("POST", "/api/v1/business-units", map[string]any{"code": "IDENTITY", "name": "独立业务单元", "effectiveDate": "1900-01-01", "operationType": "CREATE", "operationReason": "NEW_ORGANIZATION"}, sid, 201)
	recovered := b.call("POST", legal206Path+"/submissions/"+sid+"/resolve", nil, "", 200)
	if recovered["legalEntityId"] != id {
		t.Fatal("cross-type result")
	}
	before := v["headSourceVersion"]
	for _, changes := range []map[string]any{{"registrationNumber": nil}, {"registrationType": nil}, {"subjectType": "company"}, {"registrationNumber": "\n001"}, {"establishingOrganization": strings.Repeat("界", 201)}} {
		h.correct(v, changes, nil, 400)
	}
	if h.refresh(v)["headSourceVersion"] != before {
		t.Fatal("invalid patch published")
	}
	h.correct(v, map[string]any{"registrationNumber": "000aB", "establishingOrganization": "新设立单位说明"}, nil, 200)
	v = h.refresh(v)
	if legalRecord206(v)["registrationNumber"] != "000aB" {
		t.Fatal("identifier lost precision")
	}
	old := b.call("GET", legal206Path+"/"+id+"?sourceVersion="+before.(string), nil, "", 200)
	if legalRecord206(old)["registrationNumber"] != "001Ab" {
		t.Fatal("old source overwritten")
	}
	h.correct(v, map[string]any{"registrationType": nil, "registrationNumber": nil}, nil, 200)
	v = h.refresh(v)
	if legalRecord206(v)["registrationNumber"] != nil || legalRecord206(v)["registrationType"] != nil {
		t.Fatal("clear pair")
	}
	h.add(v, "ADD", "2000-01-01", map[string]any{"name": "2000资料"}, 201)
	v = h.refresh(v)
	h.add(v, "DEACTIVATE", "2001-01-01", nil, 201)
	v = h.refresh(v)
	if legalRecord206(v)["name"] != "2000资料" || legalRecord206(v)["state"] != "inactive" {
		t.Fatal("stop inheritance")
	}
	h.add(v, "REACTIVATE", "2002-01-01", map[string]any{"countryRegionId": h.countries["US"], "subjectType": "unincorporated"}, 201)
	v = h.refresh(v)
	if legalRecord206(v)["adoptionDate"] != "2002-01-01" || legalRecord206(v)["countryRegionId"] != h.countries["US"] {
		t.Fatal("resume adoption")
	}
	// Insert between first and later independent records; later facts must survive.
	firstView := h.detail(id, first)
	h.add(firstView, "ADD", "1950-01-01", map[string]any{"name": "中间资料"}, 201)
	v = h.refresh(v)
	if legalRecord206(v)["name"] != "2000资料" || legalRecord206(v)["countryRegionId"] != h.countries["US"] {
		t.Fatal("insert changed successor")
	}
	inserted := v["timeline"].([]any)[1].(map[string]any)
	selected := h.detail(id, inserted["recordId"].(string))
	h.correct(selected, nil, map[string]any{"newEffectiveDate": "1951-01-01"}, 200)
	selected = h.detail(id, inserted["recordId"].(string))
	r := legalRecord206(selected)
	b.call("POST", legal206Path+"/"+id+"/records/"+r["recordId"].(string)+"/delete", map[string]any{"expectedSourceVersion": selected["headSourceVersion"], "expectedRecordRevision": r["revision"]}, foundation.NewID(), 200)
	selected = h.detail(id, first)
	r = legalRecord206(selected)
	b.call("POST", legal206Path+"/"+id+"/records/"+first+"/delete", map[string]any{"expectedSourceVersion": selected["headSourceVersion"], "expectedRecordRevision": r["revision"]}, foundation.NewID(), 409)
	for _, q := range []string{"countryRegionId=" + h.countries["US"] + "&filterField=subjectType&filterOperator=is&filterValue=unincorporated", "q=IDENTITY&sort=registrationNumber_desc", "filterField=establishingOrganization&filterOperator=contains&filterValue=说明"} {
		b.call("GET", legal206Path+"?"+q, nil, "", 200)
	}
	b.call("GET", legal206Path+"?sort=geoAreaId", nil, "", 400)
	b.call("POST", legal206Path+"/reference-context", map[string]any{"action": "CREATE", "effectiveDate": "1900-01-01"}, "", 200)
	log := b.call("GET", legal206Path+"/"+id+"/changes", nil, "", 200)
	if log["total"].(float64) < 8 {
		t.Fatal("missing operation log")
	}
	// No source in the existing directory or location table group was changed.
	after := locationBusinessSnapshot204(t, f)
	for table, digest := range originalCountries {
		if (strings.HasPrefix(table, "country_") || strings.HasPrefix(table, "location") || strings.HasPrefix(table, "geoarea")) && after[table] != digest {
			t.Fatal("unrelated business source changed", table)
		}
	}
	t.Log("L1 independent identity/defaults/4 categories/global countries/registration/full maintenance/history/replay/read-only/query verified")
}

type legalFault206 struct {
	app.Store
	rollback, lostAck bool
}

func (s legalFault206) WithinLegalEntityWrite(ctx context.Context, c access.Credentials, fn func(app.Unit) error) error {
	e := s.Store.WithinLegalEntityWrite(ctx, c, func(u app.Unit) error {
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
func TestLegalEntity206Recovery(t *testing.T) {
	h := legalSetup206(t)
	f := h.f
	svc := f.runtime.LegalEntity.(app.Service)
	c := geoCredentials203(t, f, "admin", f.password)
	command := func(code string) org.LegalEntityCommand {
		return org.LegalEntityCommand{Action: "CREATE", SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Create: org.LegalEntityCreateInput{Code: code, Name: code, CountryRegionID: h.countries["US"], SubjectType: "other", OperationType: "CREATE", OperationReason: "NEW_LEGAL_ENTITY"}}
	}
	cmd := command("ROLLBACK206")
	_, e := (app.Service{Store: legalFault206{Store: svc.Store, rollback: true}}).Maintain(f.ctx, c, cmd)
	if e == nil {
		t.Fatal("rollback missing")
	}
	checkSubmission140(t, f, "legal_entity", cmd.Create.Code, cmd.SubmissionID, 0, 0, "")
	v, e := svc.Resolve(f.ctx, c, cmd.SubmissionID)
	must(t, e)
	if v.Outcome != "not_applied" {
		t.Fatal(v)
	}
	if _, err := svc.Maintain(f.ctx, c, cmd); err == nil || err.Error() != "M02_REQUEST_CLOSED" {
		t.Fatal("rolled-back submission reopened", err)
	}
	checkSubmission140(t, f, "legal_entity", cmd.Create.Code, cmd.SubmissionID, 0, 0, "not_applied")
	cmd = command("LOST206")
	_, e = (app.Service{Store: legalFault206{Store: svc.Store, lostAck: true}}).Maintain(f.ctx, c, cmd)
	if e == nil {
		t.Fatal("lost ack missing")
	}
	checkSubmission140(t, f, "legal_entity", cmd.Create.Code, cmd.SubmissionID, 1, 1, "")
	v, e = svc.Resolve(f.ctx, c, cmd.SubmissionID)
	must(t, e)
	if v.Outcome != "accepted" {
		t.Fatal(v)
	}
	retry, err := svc.Maintain(f.ctx, c, cmd)
	must(t, err)
	if !retry.Replayed || retry.SourceVersion == nil || *retry.SourceVersion != *v.SourceVersion {
		t.Fatal("lost acknowledgement created another source")
	}
	checkSubmission140(t, f, "legal_entity", cmd.Create.Code, cmd.SubmissionID, 1, 1, "")

	before := legalSnapshot206(t, f)
	restartTimelineReceipts(t, f, h.b, []string{cmd.SubmissionID}, "legalentity")
	if !reflect.DeepEqual(before, legalSnapshot206(t, f)) {
		t.Fatal("legal facts changed after database restart")
	}
	v2, e := f.runtime.LegalEntity.Resolve(f.ctx, c, cmd.SubmissionID)
	must(t, e)
	if !reflect.DeepEqual(v, v2) {
		t.Fatal("receipt changed", v, v2)
	}
}
func legalSnapshot206(t *testing.T, f *fixture) map[string]string {
	out := map[string]string{}
	for _, suffix := range []string{"", "_history", "_dictionary", "_shape", "_read", "_terminal"} {
		table := "legal_entity" + suffix
		var hash string
		must(t, f.admin.QueryRow(f.ctx, `SELECT md5(coalesce(string_agg(to_jsonb(t)::text,'' ORDER BY to_jsonb(t)::text),'')) FROM organization.`+table+` t`).Scan(&hash))
		out[table] = hash
	}
	return out
}

func TestLegalEntity206Boundaries(t *testing.T) {
	h := legalSetup206(t)
	b, f := h.b, h.f
	ctx := b.call("GET", legal206Path+"/context", nil, "", 200)
	if ctx["defaultEffectiveDate"] != "1900-01-01" {
		t.Fatal("wrong initial default")
	}
	for _, field := range []string{"use", "managementBuId", "locationId", "geoAreaId", "sourceVersion"} {
		body := h.body("INVALID206", "US", "other")
		body[field] = foundation.NewID()
		b.call("POST", legal206Path, body, foundation.NewID(), 400)
	}
	body := h.body("FUTURE206", "US", "other")
	body["effectiveDate"] = "1899-12-31"
	b.call("POST", legal206Path, body, foundation.NewID(), 400)
	body["effectiveDate"] = "2100-01-01"
	r := b.call("POST", legal206Path, body, foundation.NewID(), 201)
	v := h.detail(r["legalEntityId"].(string), "")
	if v["status"] != "not_effective" || v["selectedRecordIsFuture"] != true || legalRecord206(v)["effectiveDate"] != "2100-01-01" {
		t.Fatal("future first record", v)
	}
	list := b.call("GET", legal206Path+"?q=FUTURE206&status=not_effective", nil, "", 200)
	if list["count"] != float64(1) {
		t.Fatal("future list")
	}
	// All four object kinds may share a literal code and submission UUID, while
	// retaining their own stable identities and original result namespaces.
	key := foundation.NewID()
	le := b.call("POST", legal206Path, h.body("SHARED206", "MO", "branch"), key, 201)
	bu := b.call("POST", "/api/v1/business-units", createBody("SHARED206", "1900-01-01"), key, 201)
	dep := b.call("POST", "/api/v1/departments", departmentCreateBody("SHARED206", bu["buId"].(string)), key, 201)
	loc := b.call("POST", "/api/v1/locations", map[string]any{"code": "SHARED206", "name": "独立地点", "countryRegionId": h.countries["MO"], "addressLine": "一号", "operationType": "CREATE", "operationReason": "NEW_LOCATION"}, key, 201)
	ids := map[any]bool{le["legalEntityId"]: true, bu["buId"]: true, dep["departmentId"]: true, loc["locationId"]: true}
	if len(ids) != 4 {
		t.Fatal("cross-type identity collision")
	}
	before := locationBusinessSnapshot204(t, f)
	v = h.detail(le["legalEntityId"].(string), "")
	h.correct(v, map[string]any{"name": "只更正法律实体"}, nil, 200)
	after := locationBusinessSnapshot204(t, f)
	for table, hash := range before {
		if !strings.HasPrefix(table, "legal_entity") && after[table] != hash {
			t.Fatal("cross-type mutation", table)
		}
	}
	// A new principal cannot resolve another principal's original submission;
	// removed read authority must not be bypassed via context or cached results.
	reader := f.login("reader", f.readPassword)
	foreign := reader.call("POST", legal206Path+"/submissions/"+key+"/resolve", nil, "", 200)
	if foreign["outcome"] != "not_applied" || foreign["legalEntityId"] != nil {
		t.Fatal("another principal recovered original result")
	}
	h.policy("DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='legalentity.read'", b.subject)
	b.call("GET", legal206Path+"/context", nil, "", 403)
	b.call("POST", legal206Path+"/submissions/"+key+"/resolve", nil, "", 404)
}

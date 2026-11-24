//go:build integration

package bootstrap_test

import (
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	orgdb "chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres"
	"encoding/json"
	"net/url"
	"strings"
	"testing"
	"time"

	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/foundation"
)

func draft133Body(summary map[string]any) map[string]any {
	return map[string]any{"query": map[string]any{}, "draftId": summary["draftId"], "expectedRevision": summary["revision"], "page": 1, "pageSize": 20}
}

func TestDepartmentSelectionDraft133FixedIdentityEditsAndCurrentExport(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	other := f.login("admin", f.password)
	bu := b.call("POST", "/api/v1/business-units", createBody("D133_BU", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	first := b.call("POST", "/api/v1/departments", departmentCreateBody("D133_FIRST", bu), foundation.NewID(), 201)
	id := first["departmentId"].(string)
	start := map[string]any{"query": map[string]any{}, "selection": map[string]any{"mode": "allMatching", "excludedIds": []string{}}, "page": 1, "pageSize": 20}
	selection := b.call("POST", "/api/v1/departments/selection/start", start, "", 200)
	summary := selection["summary"].(map[string]any)
	if summary["selectedCount"] != float64(1) || summary["selectionVersion"] != nil {
		t.Fatal("not an identity draft", summary)
	}
	second := b.call("POST", "/api/v1/departments", departmentCreateBody("D133_SECOND", bu), foundation.NewID(), 201)
	b.call("POST", "/api/v1/departments/"+id+"/records/"+first["targetRecordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": first["sourceVersion"], "expectedRecordRevision": first["targetRevision"], "changes": map[string]any{"name": "当前名称"}}, foundation.NewID(), 200)
	body := draft133Body(summary)
	resolved := b.call("POST", "/api/v1/departments/selection/resolve", body, "", 200)
	if resolved["summary"].(map[string]any)["selectedCount"] != float64(1) || len(resolved["selectedIds"].([]any)) != 1 || resolved["selectedIds"].([]any)[0] != id {
		t.Fatal("new matching identity joined fixed selection", resolved)
	}
	other.call("POST", "/api/v1/departments/selection/resolve", body, "", 409)
	// A later invalid membership must roll back any earlier bit changes.
	body["removeIds"] = []string{id, second["departmentId"].(string)}
	b.call("POST", "/api/v1/departments/selection/resolve", body, "", 409)
	delete(body, "removeIds")
	resolved = b.call("POST", "/api/v1/departments/selection/resolve", body, "", 200)
	if resolved["summary"].(map[string]any)["selectedCount"] != float64(1) {
		t.Fatal("rejected edit changed intent")
	}
	body["removeIds"] = []string{id}
	edited := b.call("POST", "/api/v1/departments/selection/resolve", body, "", 200)
	if edited["summary"].(map[string]any)["selectedCount"] != float64(0) {
		t.Fatal(edited)
	}
	b.call("POST", "/api/v1/departments/selection/resolve", body, "", 409)
	body = draft133Body(edited["summary"].(map[string]any))
	body["restoreIds"] = []string{id}
	restored := b.call("POST", "/api/v1/departments/selection/resolve", body, "", 200)
	body = draft133Body(restored["summary"].(map[string]any))
	delete(body, "page")
	delete(body, "pageSize")
	exported := selection131HTTP(f.ctx, b, "/api/v1/departments/export", body)
	must(t, exported.err)
	if exported.status != 200 || exported.headers.Get("X-Selected-Count") != "1" || !strings.Contains(string(exported.data), "当前名称") || strings.Contains(string(exported.data), "D133_SECOND") {
		t.Fatal("export did not use fixed identity/current content", exported.status, string(exported.data))
	}
	// Old complete-projection tokens are no longer accepted by the endpoint.
	b.call("POST", "/api/v1/departments/selection/resolve", map[string]any{"query": map[string]any{}, "selection": start["selection"], "expectedSelectionVersion": strings.Repeat("a", 64), "page": 1, "pageSize": 20}, "", 400)
	// API restart drops temporary intent without touching business sources.
	r, err := bootstrap.Open(f.ctx, f.runtime.Pool.Config().ConnConfig.ConnString(), f.instance)
	must(t, err)
	r.Close()
	body = draft133Body(restored["summary"].(map[string]any))
	b.call("POST", "/api/v1/departments/selection/resolve", body, "", 409)
	selection = b.call("POST", "/api/v1/departments/selection/start", start, "", 200)
	if selection["summary"].(map[string]any)["selectedCount"] != float64(2) {
		t.Fatal("new explicit full capture did not include new member")
	}
	// Logout revokes first and then releases the corresponding draft.
	b.call("DELETE", "/api/v1/session", nil, "", 204)
	var remaining int
	must(t, f.admin.QueryRow(f.ctx, `SELECT count(*) FROM organization.department_selection_draft WHERE instance_id=$1`, f.instance).Scan(&remaining))
	if remaining != 0 {
		t.Fatal("logout left draft", remaining)
	}
}

func TestDepartmentSelectionDraft133ExplicitScopeAndExpiry(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	bu := b.call("POST", "/api/v1/business-units", createBody("D133_EXPLICIT", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	id := b.call("POST", "/api/v1/departments", departmentCreateBody("D133_ONE", bu), foundation.NewID(), 201)["departmentId"].(string)
	start := map[string]any{"query": map[string]any{}, "selection": map[string]any{"mode": "explicit", "ids": []string{id}}, "page": 1, "pageSize": 20}
	v := b.call("POST", "/api/v1/departments/selection/start", start, "", 200)
	body := draft133Body(v["summary"].(map[string]any))
	// Failed replacement leaves the previous draft available.
	start["selection"] = map[string]any{"mode": "explicit", "ids": []string{foundation.NewID()}}
	b.call("POST", "/api/v1/departments/selection/start", start, "", 400)
	b.call("POST", "/api/v1/departments/selection/resolve", body, "", 200)
	_, err := f.admin.Exec(f.ctx, `UPDATE organization.department_selection_draft SET created_at=statement_timestamp()-interval '31 minutes',expires_at=statement_timestamp()-interval '1 minute' WHERE instance_id=$1`, f.instance)
	must(t, err)
	response := b.call("POST", "/api/v1/departments/selection/resolve", body, "", 409)
	raw, _ := json.Marshal(response)
	if strings.Contains(string(raw), id) {
		t.Fatal("expired draft exposes member identity", string(raw))
	}
}

func TestDepartmentSelectionDraft133ConcurrentEditsAndAdmission(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	other := f.login("admin", f.password)
	bu := b.call("POST", "/api/v1/business-units", createBody("D133_QUOTA", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	ids := []string{}
	for _, code := range []string{"D133_Q1", "D133_Q2"} {
		ids = append(ids, b.call("POST", "/api/v1/departments", departmentCreateBody(code, bu), foundation.NewID(), 201)["departmentId"].(string))
	}
	start := map[string]any{"query": map[string]any{}, "selection": map[string]any{"mode": "allMatching", "excludedIds": []string{}}, "page": 1, "pageSize": 20}
	selected := b.call("POST", "/api/v1/departments/selection/start", start, "", 200)["summary"].(map[string]any)
	type result struct {
		status int
		raw    []byte
		err    error
	}
	gate := make(chan struct{})
	results := make(chan result, 2)
	for _, id := range ids {
		body := draft133Body(selected)
		body["removeIds"] = []string{id}
		go func() {
			<-gate
			v := selection131HTTP(f.ctx, b, "/api/v1/departments/selection/resolve", body)
			results <- result{v.status, v.data, v.err}
		}()
	}
	close(gate)
	success, conflict := 0, 0
	for i := 0; i < 2; i++ {
		r := <-results
		must(t, r.err)
		switch r.status {
		case 200:
			success++
			var v map[string]any
			must(t, json.Unmarshal(r.raw, &v))
			selected = v["summary"].(map[string]any)
		case 409:
			conflict++
		default:
			t.Fatalf("concurrent edit status %d", r.status)
		}
	}
	if success != 1 || conflict != 1 || selected["selectedCount"] != float64(1) || selected["revision"] != float64(2) {
		t.Fatal("lost concurrent intent", success, conflict, selected)
	}
	// Fill other-session metadata with empty intent, solely to exercise the real
	// 128-draft admission boundary without 128 synthetic browser logins.
	_, err := f.admin.Exec(f.ctx, `INSERT INTO organization.department_selection_draft SELECT instance_id,enterprise_id,subject_id,gen_random_uuid(),gen_random_uuid(),business_day,query_key,'explicit',1,0,0,created_at,expires_at,n+1 FROM organization.department_selection_draft CROSS JOIN generate_series(1,127) n WHERE instance_id=$1 AND draft_id=$2`, f.instance, selected["draftId"])
	must(t, err)
	other.call("POST", "/api/v1/departments/selection/start", start, "", 429)
	b.call("POST", "/api/v1/departments/selection/resolve", draft133Body(selected), "", 200)
	// Replacing one's own draft is allowed at quota; the preceding identity dies.
	replacement := b.call("POST", "/api/v1/departments/selection/start", start, "", 200)["summary"].(map[string]any)
	b.call("POST", "/api/v1/departments/selection/resolve", draft133Body(selected), "", 409)
	selected = replacement
	tooMany := make([]string, 1025)
	for i := range tooMany {
		tooMany[i] = foundation.NewID()
	}
	body := draft133Body(selected)
	body["addIds"] = tooMany
	limited := b.call("POST", "/api/v1/departments/selection/resolve", body, "", 413)
	if limited["code"] != "M02_SELECTION_BATCH_LIMIT" {
		t.Fatal("batch limit must distinguish temporary capacity", limited)
	}
	b.call("POST", "/api/v1/departments/selection/resolve", draft133Body(selected), "", 200)
	_, err = f.admin.Exec(f.ctx, `UPDATE organization.department_selection_draft SET created_at=statement_timestamp()-interval '31 minutes',expires_at=statement_timestamp()-interval '1 minute' WHERE instance_id=$1 AND draft_id<>$2`, f.instance, selected["draftId"])
	must(t, err)
	store := f.runtime.Department.(app.Service).Store.(*orgdb.Store)
	must(t, store.SweepDrafts(f.ctx))
	var count int
	must(t, f.admin.QueryRow(f.ctx, `SELECT count(*) FROM organization.department_selection_draft WHERE instance_id=$1`, f.instance).Scan(&count))
	if count != 1 {
		t.Fatal("bounded sweep", count)
	}
	other.call("POST", "/api/v1/departments/selection/start", start, "", 200)
	b.call("POST", "/api/v1/departments/selection/cancel", map[string]any{"draftId": selected["draftId"], "expectedRevision": selected["revision"]}, "", 204)
	b.call("POST", "/api/v1/departments/selection/resolve", draft133Body(selected), "", 409)
}

// Expiry may occur while a valid read is still projecting its page. The sweep
// must skip that intent until completion, rather than erase its selected bits.
func TestDepartmentDraft133ExpiryCannotEraseInFlightPage(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	bu := b.call("POST", "/api/v1/business-units", createBody("EXP133_BU", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	b.call("POST", "/api/v1/departments", departmentCreateBody("EXP133_DEP", bu), foundation.NewID(), 201)
	summary := b.call("POST", "/api/v1/departments/selection/start", map[string]any{"query": map[string]any{}, "selection": map[string]any{"mode": "allMatching", "excludedIds": []string{}}, "page": 1, "pageSize": 20}, "", 200)["summary"].(map[string]any)
	_, e := f.admin.Exec(f.ctx, `UPDATE organization.department_selection_draft SET created_at=statement_timestamp()-interval '29 minutes 58 seconds',expires_at=statement_timestamp()+interval '2 seconds' WHERE instance_id=$1 AND draft_id=$2`, f.instance, summary["draftId"])
	must(t, e)
	service := f.runtime.Department.(app.Service)
	store := service.Store.(*orgdb.Store)
	origin, _ := url.Parse(f.server.URL)
	credentials := foundation.Credentials{Token: b.client.Jar.Cookies(origin)[0].Value}
	e = store.WithinDepartment(f.ctx, credentials, func(u app.Unit) error {
		draft, e := u.ReadDraft(f.ctx, summary["draftId"].(string), 1, time.Now().In(foundation.Shanghai).Format("2006-01-02"), summary["queryKey"].(string))
		if e != nil {
			return e
		}
		// PostgreSQL clock is authoritative; poll expiry independently of Go's clock.
		deadline := time.Now().Add(3 * time.Second)
		for {
			var expired bool
			must(t, f.admin.QueryRow(f.ctx, `SELECT expires_at<=clock_timestamp() FROM organization.department_selection_draft WHERE instance_id=$1 AND draft_id=$2`, f.instance, draft.ID).Scan(&expired))
			if expired {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("expiry fixture did not reach its boundary")
			}
			time.Sleep(20 * time.Millisecond)
		}
		must(t, store.SweepDrafts(f.ctx))
		ids, e := u.DraftMembers(f.ctx, draft)
		if e != nil {
			return e
		}
		if len(ids) != 1 {
			t.Fatal("in-flight intent erased")
		}
		return nil
	})
	must(t, e)
	must(t, store.SweepDrafts(f.ctx))
	b.call("POST", "/api/v1/departments/selection/resolve", draft133Body(summary), "", 409)
}

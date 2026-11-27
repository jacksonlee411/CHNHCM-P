//go:build integration

package bootstrap_test

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/location"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
)

type location204 struct {
	t         *testing.T
	f         *fixture
	b         *browser
	countries map[string]string
}

func (h location204) create(code, country, area, date string) map[string]any {
	body := map[string]any{"code": code, "name": "地点 " + code, "countryRegionId": h.countries[country], "addressLine": "一号", "postalCode": "00100", "operationType": "CREATE", "operationReason": "NEW_LOCATION"}
	if area != "" {
		body["geoAreaId"] = geoID203(h.t, h.f, area)
	}
	if date != "" {
		body["effectiveDate"] = date
	}
	saved := h.b.call("POST", "/api/v1/locations", body, foundation.NewID(), 201)
	return h.detail(saved["locationId"].(string), "")
}
func (h location204) detail(id, record string) map[string]any {
	path := "/api/v1/locations/" + id
	if record != "" {
		path += "?selectedRecordId=" + record
	}
	return h.b.call("GET", path, nil, "", 200)
}
func locRecord(v map[string]any) map[string]any { return v["selectedRecord"].(map[string]any) }
func (h location204) correct(v map[string]any, changes map[string]any, extra map[string]any, status int) map[string]any {
	r := locRecord(v)
	body := map[string]any{"expectedSourceVersion": v["headSourceVersion"], "expectedRecordRevision": r["revision"]}
	if changes != nil {
		body["changes"] = changes
	}
	for k, v := range extra {
		body[k] = v
	}
	return h.b.call("POST", "/api/v1/locations/"+v["id"].(string)+"/records/"+r["recordId"].(string)+"/correct", body, foundation.NewID(), status)
}
func (h location204) add(v map[string]any, action, date string, changes map[string]any, status int) map[string]any {
	kind, reason, path := "CHANGE", "ADDRESS_ADJUSTMENT", "/records"
	if action == "DEACTIVATE" {
		kind, reason, path = action, "LOCATION_CLOSURE", "/deactivate"
	}
	if action == "REACTIVATE" {
		kind, reason, path = action, "LOCATION_RESUMPTION", "/reactivate"
	}
	body := map[string]any{"expectedSourceVersion": v["headSourceVersion"], "anchorRecordId": locRecord(v)["recordId"], "effectiveDate": date, "operationType": kind, "operationReason": reason}
	if changes != nil {
		body["changes"] = changes
	}
	if status == 200 {
		status = 201
	}
	return h.b.call("POST", "/api/v1/locations/"+v["id"].(string)+path, body, foundation.NewID(), status)
}
func (h location204) refresh(v map[string]any) map[string]any { return h.detail(v["id"].(string), "") }
func (h location204) policy(sql string, args ...any) {
	h.f.exec(sql, args...)
	h.f.exec("UPDATE access.policy_guard SET policy_no=policy_no+1")
}
func (h location204) restore() {
	h.f.exec("DELETE FROM access.subject_grant; INSERT INTO access.subject_grant SELECT * FROM public.g4_original_grants")
	h.f.exec("UPDATE access.policy_guard SET policy_no=policy_no+1")
}
func TestLocation204Acceptance(t *testing.T) {
	f := freshWithEnterpriseStart(t, "1900-01-01")
	must(t, f.runtime.InitializeCountryCatalog(f.ctx, access.LoginInput{Login: "admin", Password: f.password}))
	must(t, f.runtime.InitializeGeoAreaCatalog(f.ctx, access.LoginInput{Login: "admin", Password: f.password}))
	// Verify imported originals before deliberate directory maintenance; restart
	// later compares all business bytes, including those maintained histories.
	geoAllSources203(t, f)
	h := location204{t, f, f.login("admin", f.password), map[string]string{}}
	for _, c := range []string{"CN", "HK", "MO", "TW", "US"} {
		h.countries[c] = geoCountry203(t, f, c)
	}
	f.exec("CREATE TABLE public.g4_original_grants AS SELECT * FROM access.subject_grant")
	root := t
	run := func(name string, fn func(*testing.T)) {
		root.Run(name, func(t *testing.T) { h.t = t; f.t = t; defer func() { h.t = root; f.t = root }(); fn(t) })
	}

	run("four_regions_and_actual_terminals", func(t *testing.T) {
		for i, v := range []struct{ c, a string }{{"CN", "110108"}, {"CN", "419001"}, {"CN", "320583"}, {"CN", "441900"}, {"CN", "442000"}, {"CN", "460400"}, {"CN", "620200"}, {"HK", "CW"}, {"HK", "SSP"}, {"TW", "63000010"}, {"TW", "65000010"}, {"MO", ""}, {"MO", ""}} {
			v := h.create(fmt.Sprintf("REGION-%02d", i), v.c, v.a, "")
			r := locRecord(v)
			if r["adoptionDate"] != "1900-01-01" || r["postalCode"] != "00100" {
				t.Fatal("default/address")
			}
			if r["addressRestricted"] == true {
				t.Fatal("authorized path restricted")
			}
		}
		for _, v := range []struct{ c, a string }{{"US", ""}, {"MO", "CW"}, {"CN", "110000"}, {"HK", "110108"}, {"CN", ""}} {
			body := map[string]any{"code": "BAD-" + foundation.NewID(), "name": "非法地址", "countryRegionId": h.countries[v.c], "addressLine": "一号", "operationType": "CREATE", "operationReason": "NEW_LOCATION"}
			if v.a != "" {
				body["geoAreaId"] = geoID203(t, f, v.a)
			}
			h.b.call("POST", "/api/v1/locations", body, foundation.NewID(), 409)
		}
		h.b.call("POST", "/api/v1/locations/reference-context", map[string]any{"action": "CREATE", "effectiveDate": "1900-01-01"}, "", 200)
	})
	run("full_lifecycle_and_independent_successors", func(t *testing.T) {
		v := h.create("LIFE", "HK", "CW", "")
		id := v["id"].(string)
		firstID := locRecord(v)["recordId"].(string)
		h.add(v, "ADD", "2000-01-01", map[string]any{"addressLine": "后继地址", "geoAreaId": geoID203(t, f, "SSP")}, 200)
		v = h.refresh(v)
		successor := locRecord(v)["recordId"].(string)
		first := h.detail(id, firstID)
		h.add(first, "ADD", "1990-01-01", map[string]any{"addressLine": "中间地址"}, 200)
		v = h.detail(id, successor)
		if locRecord(v)["addressLine"] != "后继地址" || locRecord(v)["adoptionDate"] != "2000-01-01" {
			t.Fatal("successor propagated")
		}
		h.add(v, "DEACTIVATE", "2010-01-01", nil, 200)
		v = h.refresh(v)
		if locRecord(v)["adoptionDate"] != "2000-01-01" || locRecord(v)["addressLine"] != "后继地址" {
			t.Fatal("deactivation inheritance")
		}
		h.correct(v, nil, map[string]any{"newEffectiveDate": "2011-01-01"}, 200)
		v = h.refresh(v)
		if locRecord(v)["adoptionDate"] != "2000-01-01" {
			t.Fatal("stop re-adopted")
		}
		h.add(v, "REACTIVATE", "2020-01-01", nil, 200)
		v = h.refresh(v)
		if locRecord(v)["adoptionDate"] != "2020-01-01" {
			t.Fatal("resume not adopted")
		}
		h.correct(v, nil, map[string]any{"classification": map[string]any{"operationType": "CHANGE", "operationReason": "OTHER_DATA_ADJUSTMENT"}}, 409)
		first = h.detail(id, firstID)
		h.b.call("POST", "/api/v1/locations/"+id+"/records/"+firstID+"/delete", map[string]any{"expectedSourceVersion": first["headSourceVersion"], "expectedRecordRevision": locRecord(first)["revision"]}, foundation.NewID(), 409)
		v = h.create("PAIR", "MO", "", "")
		id = v["id"].(string)
		h.add(v, "ADD", "2030-01-01", map[string]any{"name": "未来独立"}, 200)
		v = h.detail(id, locRecord(v)["recordId"].(string))
		insert := map[string]any{"expectedSourceVersion": v["headSourceVersion"], "anchorRecordId": locRecord(v)["recordId"], "deactivationDate": "2000-01-01", "reactivationDate": "2001-01-01", "deactivationReason": "LOCATION_CLOSURE", "reactivationReason": "LOCATION_RESUMPTION"}
		h.b.call("POST", "/api/v1/locations/"+id+"/lifecycle-pairs/insert", insert, foundation.NewID(), 200)
		v = h.refresh(v)
		rows := v["timeline"].([]any)
		stop, resume := rows[1].(map[string]any), rows[2].(map[string]any)
		pair := map[string]any{"expectedSourceVersion": v["headSourceVersion"], "deactivationRecordId": stop["recordId"], "deactivationRevision": stop["revision"], "reactivationRecordId": resume["recordId"], "reactivationRevision": resume["revision"], "deactivationDate": "2000-02-01", "reactivationDate": "2002-01-01", "deactivationReason": "LOCATION_CLOSURE", "reactivationReason": "LOCATION_RESUMPTION"}
		h.b.call("POST", "/api/v1/locations/"+id+"/lifecycle-pairs/maintain", pair, foundation.NewID(), 200)
		v = h.refresh(v)
		rows = v["timeline"].([]any)
		stop, resume = rows[1].(map[string]any), rows[2].(map[string]any)
		if resume["adoptionDate"] != "2002-01-01" || stop["adoptionDate"] != "1900-01-01" {
			t.Fatal("pair adoption")
		}
		pair = map[string]any{"expectedSourceVersion": v["headSourceVersion"], "deactivationRecordId": stop["recordId"], "deactivationRevision": stop["revision"], "reactivationRecordId": resume["recordId"], "reactivationRevision": resume["revision"]}
		h.b.call("POST", "/api/v1/locations/"+id+"/lifecycle-pairs/delete", pair, foundation.NewID(), 200)
		if len(h.refresh(v)["timeline"].([]any)) != 2 {
			t.Fatal("pair not atomic")
		}
	})
	run("redate_classification_and_single_deletions", func(t *testing.T) {
		v := h.create("CLASS-DELETE", "HK", "CW", "")
		id := v["id"].(string)
		firstID := locRecord(v)["recordId"].(string)
		h.add(v, "ADD", "2000-01-01", map[string]any{"addressLine": "独立第二条"}, 200)
		v = h.refresh(v)
		secondID := locRecord(v)["recordId"].(string)
		h.add(v, "ADD", "2010-01-01", map[string]any{"addressLine": "独立第三条"}, 200)
		first := h.detail(id, firstID)
		h.correct(first, map[string]any{"name": "首条更正"}, map[string]any{"newEffectiveDate": "1901-01-01"}, 200)
		if locRecord(h.detail(id, firstID))["adoptionDate"] != "1901-01-01" {
			t.Fatal("first redate adoption")
		}
		second := h.detail(id, secondID)
		h.b.call("POST", "/api/v1/locations/"+id+"/records/"+secondID+"/redate", map[string]any{"expectedSourceVersion": second["headSourceVersion"], "expectedRecordRevision": locRecord(second)["revision"], "newEffectiveDate": "2001-01-01"}, foundation.NewID(), 200)
		v = h.refresh(v)
		if locRecord(v)["addressLine"] != "独立第三条" {
			t.Fatal("redate changed independent successor")
		}
		h.correct(v, nil, map[string]any{"classification": map[string]any{"operationType": "DEACTIVATE", "operationReason": "LOCATION_CLOSURE"}}, 200)
		v = h.refresh(v)
		if locRecord(v)["addressLine"] != "独立第二条" || locRecord(v)["adoptionDate"] != "2001-01-01" {
			t.Fatal("classification stop did not inherit")
		}
		h.correct(v, map[string]any{"addressLine": "重新独立"}, map[string]any{"classification": map[string]any{"operationType": "CHANGE", "operationReason": "ADDRESS_ADJUSTMENT"}}, 200)
		v = h.refresh(v)
		if locRecord(v)["adoptionDate"] != "2010-01-01" {
			t.Fatal("classification change did not adopt")
		}
		remove := func(v map[string]any) {
			r := locRecord(v)
			h.b.call("POST", "/api/v1/locations/"+v["id"].(string)+"/records/"+r["recordId"].(string)+"/delete", map[string]any{"expectedSourceVersion": v["headSourceVersion"], "expectedRecordRevision": r["revision"]}, foundation.NewID(), 200)
		}
		source := v["headSourceVersion"].(string)
		remove(h.detail(id, secondID))
		v = h.refresh(v)
		if len(v["timeline"].([]any)) != 2 || locRecord(v)["addressLine"] != "重新独立" || locRecord(v)["adoptionDate"] != "2010-01-01" {
			t.Fatal("delete rewrote independent successor")
		}
		old := h.b.call("GET", "/api/v1/locations/"+id+"?sourceVersion="+source, nil, "", 200)
		if len(old["timeline"].([]any)) != 3 {
			t.Fatal("delete lost original source")
		}
		remove(v)
		h.add(h.refresh(v), "DEACTIVATE", "2000-01-01", nil, 200)
		h.add(h.refresh(v), "REACTIVATE", "2001-01-01", nil, 200)
		remove(h.refresh(v))
		if locRecord(h.refresh(v))["state"] != "inactive" {
			t.Fatal("resume deletion")
		}
		remove(h.refresh(v))
		if len(h.refresh(v)["timeline"].([]any)) != 1 || locRecord(h.refresh(v))["state"] != "active" {
			t.Fatal("stop deletion")
		}
	})
	run("seven_sorts_and_complete_navigation", func(t *testing.T) {
		a := h.create("QUERY204-A", "MO", "", "2000-01-01")
		h.correct(a, map[string]any{"name": "Alpha", "nameEn": "Alpha", "addressLine": "Query address A", "postalCode": "00001"}, nil, 200)
		b := h.create("QUERY204-B", "MO", "", "2001-01-01")
		h.correct(b, map[string]any{"name": "Beta", "nameEn": "Beta", "addressLine": "Query address B", "postalCode": "00002"}, nil, 200)
		h.add(h.refresh(b), "DEACTIVATE", "2002-01-01", nil, 200)
		for _, field := range []string{"code", "name", "nameEn", "addressLine", "postalCode", "status", "effectiveDate"} {
			for _, suffix := range []string{"", "_desc"} {
				got := h.b.call("GET", "/api/v1/locations?q=QUERY204-&sort="+field+suffix+"&pageSize=20", nil, "", 200)
				rows := got["items"].([]any)
				first, last := a["id"], b["id"]
				if suffix != "" {
					first, last = last, first
				}
				if got["count"] != float64(2) || len(rows) != 2 || rows[0].(map[string]any)["id"] != first || rows[1].(map[string]any)["id"] != last {
					t.Fatal("sort mismatch", field, suffix)
				}
			}
		}
		n := h.b.call("POST", "/api/v1/locations/navigation", map[string]any{"query": map[string]any{"q": "QUERY204-", "countryRegionId": h.countries["MO"], "status": "all", "filterField": "addressLine", "filterOperator": "contains", "filterValue": "Query address"}, "sort": "postalCode_desc", "anchorId": b["id"], "direction": "next"}, "", 200)
		if n["targetId"] != a["id"] {
			t.Fatal("full filter navigation")
		}
	})
	run("fields_and_date_authority", func(t *testing.T) {
		defer h.restore()
		v := h.create("FIELDS", "MO", "", "")
		h.correct(v, map[string]any{"postalCode": nil, "addressLine": "　详细地址 楼层房间　", "nameEn": "e\u0301"}, nil, 200)
		v = h.refresh(v)
		r := locRecord(v)
		if r["postalCode"] != nil || r["addressLine"] != "详细地址 楼层房间" || r["nameEn"] != "é" {
			t.Fatal("normalization", r)
		}
		for _, c := range []map[string]any{{"addressLine": strings.Repeat("字", 1002)}, {"postalCode": strings.Repeat("0", 21)}, {"addressLine": "\n一号"}, {"countryRegionId": nil}, {"managementBuId": foundation.NewID()}, {"addressExtra": "已删除"}} {
			h.correct(v, c, nil, 400)
		}
		h.correct(v, map[string]any{"addressLine": strings.Repeat("字", 1001), "postalCode": strings.Repeat("0", 20)}, nil, 200)
		v = h.refresh(v)
		n := f.scalar("SELECT sum(cardinality(source_refs)) FROM organization.location_history")
		no := h.correct(v, map[string]any{"addressLine": locRecord(v)["addressLine"]}, nil, 200)
		if no["outcome"] != "no_change" || f.scalar("SELECT sum(cardinality(source_refs)) FROM organization.location_history") != n {
			t.Fatal("no_change wrote source")
		}
		h.policy("DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='location.record.redate'", h.b.subject)
		h.correct(v, nil, map[string]any{"newEffectiveDate": "1901-01-01"}, 403)
		h.b.call("POST", "/api/v1/locations/"+v["id"].(string)+"/preview", map[string]any{"action": "CORRECT", "expectedSourceVersion": v["headSourceVersion"], "recordId": locRecord(v)["recordId"], "expectedRecordRevision": locRecord(v)["revision"], "newEffectiveDate": "1901-01-01"}, "", 403)
	})
	run("exact_labels_and_directory_stop", func(t *testing.T) {
		defer h.restore()
		v := h.create("OLD-ADDRESS", "HK", "CW", "")
		oldSource := v["headSourceVersion"].(string)
		oldPath, _ := json.Marshal(locRecord(v)["geoAreaPath"])
		area := geoID203(t, f, "CW")
		g := h.b.call("GET", "/api/v1/geo-areas/"+area, nil, "", 200)
		r := g["selectedRecord"].(map[string]any)
		h.b.call("POST", "/api/v1/geo-areas/"+area+"/records/"+r["recordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": g["headSourceVersion"], "expectedRecordRevision": r["revision"], "changes": map[string]any{"name": "中西区新目录名"}}, foundation.NewID(), 200)
		h.correct(v, map[string]any{"addressLine": "原引用二号"}, nil, 200)
		v = h.refresh(v)
		path, _ := json.Marshal(locRecord(v)["geoAreaPath"])
		if string(path) != string(oldPath) {
			t.Fatal("ordinary correction refreshed adoption")
		}
		newV := h.create("NEW-ADDRESS", "HK", "CW", "")
		if !strings.Contains(fmt.Sprint(locRecord(newV)["geoAreaPath"]), "中西区新目录名") {
			t.Fatal("new adoption old version")
		}
		g = h.b.call("GET", "/api/v1/geo-areas/"+area, nil, "", 200)
		r = g["selectedRecord"].(map[string]any)
		h.b.call("POST", "/api/v1/geo-areas/"+area+"/deactivate", map[string]any{"expectedSourceVersion": g["headSourceVersion"], "anchorRecordId": r["recordId"], "effectiveDate": "2000-01-01", "operationType": "DEACTIVATE", "operationReason": "STOP_SELECTION"}, foundation.NewID(), 201)
		h.correct(v, map[string]any{"name": "停选后仍可改名"}, nil, 200)
		v = h.refresh(v)
		h.correct(v, nil, map[string]any{"newEffectiveDate": "2001-01-01"}, 409)
		h.policy("DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='geoarea.history.read'", h.b.subject)
		hidden := h.refresh(v)
		if locRecord(hidden)["geoAreaPath"] != nil || locRecord(hidden)["countryName"] == nil || locRecord(hidden)["addressRestricted"] != true {
			t.Fatal("exact label fields", locRecord(hidden))
		}
		h.correct(hidden, map[string]any{"postalCode": "00001"}, nil, 200)
		h.policy("DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='location.history.read'", h.b.subject)
		h.refresh(v)
		h.b.call("GET", "/api/v1/locations/"+v["id"].(string)+"?sourceVersion="+oldSource, nil, "", 403)
	})
	run("permissions_query_navigation_and_implicit_fields", func(t *testing.T) {
		defer h.restore()
		a := h.create("AUTH-A", "MO", "", "")
		b := h.create("AUTH-B", "MO", "", "")
		h.policy("DELETE FROM access.subject_grant WHERE principal_id=$1 AND action LIKE 'geoarea.%'", h.b.subject)
		h.create("MO-NO-AREA-GRANT", "MO", "", "")
		h.policy("DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='country.read'", h.b.subject)
		body := map[string]any{"code": "NO-COUNTRY", "name": "无国家权", "countryRegionId": h.countries["MO"], "addressLine": "一号", "operationType": "CREATE", "operationReason": "NEW_LOCATION"}
		h.b.call("POST", "/api/v1/locations", body, foundation.NewID(), 403)
		h.restore()
		// Selecting a non-current record requires the independent history fields.
		h.add(a, "ADD", "2000-01-01", map[string]any{"addressLine": "第二条"}, 200)
		oldID := locRecord(a)["recordId"].(string)
		a = h.refresh(a)
		h.policy("UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'addressLine') WHERE principal_id=$1 AND action='location.history.read'", h.b.subject)
		old := h.detail(a["id"].(string), oldID)
		if _, ok := locRecord(old)["addressLine"]; ok {
			t.Fatal("history field leaked")
		}
		if strings.Contains(fmt.Sprint(old["capabilities"].(map[string]any)["readFields"]), "addressLine") {
			t.Fatal("historical capabilities leaked")
		}
		h.restore()
		h.policy("UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'code') WHERE principal_id=$1 AND action='location.read'", h.b.subject)
		listed := h.b.call("GET", "/api/v1/locations?pageSize=20", nil, "", 200)
		for _, row := range listed["items"].([]any) {
			if _, ok := row.(map[string]any)["code"]; ok {
				t.Fatal("hidden code")
			}
		}
		h.b.call("GET", "/api/v1/locations?sort=code", nil, "", 403)
		h.restore()
		query := map[string]any{"q": "AUTH-"}
		nav := map[string]any{"query": query, "anchorId": a["id"], "direction": "next", "sort": "code"}
		n := h.b.call("POST", "/api/v1/locations/navigation", nav, "", 200)
		if n["targetId"] != b["id"] {
			t.Fatal("full query navigation", n)
		}
		nav["query"] = map[string]any{"q": "AUTH-B"}
		n = h.b.call("POST", "/api/v1/locations/navigation", nav, "", 200)
		if n["targetId"] != nil {
			t.Fatal("out-of-scope anchor")
		}
		// A deletion changes inherited stop address, so the final implicit write also needs addressLine.
		h.add(a, "DEACTIVATE", "2010-01-01", nil, 200)
		a = h.refresh(a)
		selected := h.detail(a["id"].(string), a["timeline"].([]any)[1].(map[string]any)["recordId"].(string))
		h.policy("UPDATE access.subject_grant SET write_fields=array_remove(write_fields,'addressLine') WHERE principal_id=$1 AND action='location.record.delete'", h.b.subject)
		h.b.call("POST", "/api/v1/locations/"+a["id"].(string)+"/records/"+locRecord(selected)["recordId"].(string)+"/delete", map[string]any{"expectedSourceVersion": selected["headSourceVersion"], "expectedRecordRevision": locRecord(selected)["revision"]}, foundation.NewID(), 403)
	})

	run("query_selection_export", func(t *testing.T) {
		defer h.restore()
		for i := 0; i < 137; i++ {
			h.create(fmt.Sprintf("SELECT-%03d", i), "MO", "", "")
		}
		query := map[string]any{"q": "SELECT-", "countryRegionId": h.countries["MO"]}
		start := h.b.call("POST", "/api/v1/locations/selection/start", map[string]any{"query": query, "selection": map[string]any{"mode": "allMatching", "excludedIds": []string{}}, "page": 1, "pageSize": 20}, "", 200)
		summary := start["summary"].(map[string]any)
		if summary["selectedCount"] != float64(137) {
			t.Fatal(start)
		}
		ids := start["selectedIds"].([]any)
		removed := ids[0].(string)
		edit := map[string]any{"query": query, "draftId": summary["draftId"], "expectedRevision": summary["revision"], "removeIds": []string{removed}, "page": 7, "pageSize": 20, "sort": "name_desc"}
		next := h.b.call("POST", "/api/v1/locations/selection/resolve", edit, "", 200)
		sum := next["summary"].(map[string]any)
		if sum["selectedCount"] != float64(136) {
			t.Fatal(next)
		}
		// A lost edit response cannot apply the same revision twice; callers must recheck.
		h.b.call("POST", "/api/v1/locations/selection/resolve", edit, "", 409)
		edit["removeIds"] = []string{ids[1].(string)}
		h.b.call("POST", "/api/v1/locations/selection/resolve", edit, "", 409)
		h.create("SELECT-NEW", "MO", "", "")
		body := map[string]any{"query": query, "draftId": sum["draftId"], "expectedRevision": sum["revision"]}
		export := func(status int) []byte {
			raw, _ := json.Marshal(body)
			req, _ := http.NewRequest("POST", f.server.URL+"/api/v1/locations/export", strings.NewReader(string(raw)))
			req.Header.Set("Origin", f.server.URL)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-CSRF-Token", h.b.csrf)
			res, e := h.b.client.Do(req)
			must(t, e)
			defer res.Body.Close()
			data, e := io.ReadAll(res.Body)
			must(t, e)
			if res.StatusCode != status {
				t.Fatalf("export status %d: %s", res.StatusCode, data)
			}
			if status != 200 && res.Header.Get("Content-Disposition") != "" {
				t.Fatal("partial file")
			}
			return data
		}
		data := export(200)
		if !strings.HasPrefix(string(data), "\xef\xbb\xbf") {
			t.Fatal("CSV BOM")
		}
		rows, e := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(data), "\xef\xbb\xbf"))).ReadAll()
		must(t, e)
		if len(rows) != 137 || len(rows[0]) != 9 {
			t.Fatal("CSV dimensions", len(rows))
		}
		for _, r := range rows[1:] {
			if r[0] == "SELECT-NEW" || !strings.Contains(strings.Join(r, "|"), "00100") {
				t.Fatal("CSV fixed members/zero")
			}
		}
		// A corrupt final selected object must reject the whole file, including
		// rows generated by earlier 100-object batches. Ordinary SQL may not
		// rewrite a projection; only this isolated fault injection bypasses it.
		var lastID string
		var envelope []byte
		must(t, f.admin.QueryRow(f.ctx, "SELECT location_id::text,current_envelope FROM organization.location WHERE code='SELECT-136'").Scan(&lastID, &envelope))
		if _, e := f.admin.Exec(f.ctx, "UPDATE organization.location SET current_envelope=decode(repeat('00',32),'hex') WHERE location_id=$1", lastID); e == nil {
			t.Fatal("ordinary SQL rewrote current projection")
		}
		replaceEnvelope := func(value []byte) {
			tx, e := f.admin.Begin(f.ctx)
			must(t, e)
			defer tx.Rollback(f.ctx)
			_, e = tx.Exec(f.ctx, "ALTER TABLE organization.location DISABLE TRIGGER protect_head")
			must(t, e)
			_, e = tx.Exec(f.ctx, "UPDATE organization.location SET current_envelope=$1 WHERE location_id=$2", value, lastID)
			must(t, e)
			_, e = tx.Exec(f.ctx, "ALTER TABLE organization.location ENABLE TRIGGER protect_head")
			must(t, e)
			must(t, tx.Commit(f.ctx))
		}
		defer replaceEnvelope(envelope)
		replaceEnvelope(make([]byte, 32))
		export(503)
		h.b.call("GET", "/api/v1/locations/"+lastID, nil, "", 503)
		replaceEnvelope(envelope)
		export(200)
		originalService := f.runtime.Location
		defer func() { f.runtime.Location = originalService }()
		svc := originalService.(app.Service)
		broken := &locationExportFault204{Store: svc.Store}
		f.runtime.Location = app.Service{Store: broken}
		export(503)
		f.runtime.Location = originalService
		if broken.file == nil || !broken.file.closed || !broken.file.sealed || broken.file.bytes < len(data) {
			t.Fatal("failed final file leaked or failure was too early")
		}
		ctx, cancel := context.WithCancel(f.ctx)
		canceled := &locationExportFault204{Store: svc.Store, cancel: cancel}
		// Selection intent belongs to the originating session. Reuse its actual
		// HTTP cookie so cancellation reaches file sealing, not draft admission.
		endpoint, e := url.Parse(f.server.URL)
		must(t, e)
		credentials := access.Credentials{CSRF: h.b.csrf, Write: true}
		for _, cookie := range h.b.client.Jar.Cookies(endpoint) {
			if cookie.Name == "chnhcm_session" {
				credentials.Token = cookie.Value
			}
		}
		if credentials.Token == "" {
			t.Fatal("missing originating HTTP session")
		}
		out, e := (app.Service{Store: canceled}).ExportSelection(ctx, credentials, org.LocationExportInput{Query: org.LocationSelectionQuery{Q: "SELECT-", CountryRegionID: h.countries["MO"]}, DraftID: sum["draftId"].(string), ExpectedRevision: int64(sum["revision"].(float64))})
		cancel()
		if e == nil || out.File != nil || canceled.file == nil || !canceled.file.closed || !canceled.file.sealed || canceled.file.bytes < len(data) {
			t.Fatal("canceled export retained a file")
		}
		export(200) // Connection, draft and transaction remain usable after both failures.
		h.policy("UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'postalCode') WHERE principal_id=$1 AND action='location.export'", h.b.subject)
		export(403)
	})
	run("namespaces_concurrency_and_guards", func(t *testing.T) {
		const n = 4
		statuses := make(chan int, n)
		var wg sync.WaitGroup
		body := map[string]any{"code": "RACE-CODE", "name": "并发编码", "countryRegionId": h.countries["MO"], "addressLine": "一号", "operationType": "CREATE", "operationReason": "NEW_LOCATION"}
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				status, _, e := h.b.response("POST", "/api/v1/locations", body, foundation.NewID())
				if e != nil {
					statuses <- 0
				} else {
					statuses <- status
				}
			}()
		}
		wg.Wait()
		close(statuses)
		success := 0
		for s := range statuses {
			if s == 201 {
				success++
			} else if s != 409 {
				t.Fatal("code race", s)
			}
		}
		if success != 1 {
			t.Fatal("duplicate accepted")
		}
		if _, e := f.admin.Exec(f.ctx, "UPDATE organization.location SET code='illegal' WHERE code='RACE-CODE'"); e == nil {
			t.Fatal("identity mutated")
		}
	})
	run("submission_rollback_lost_ack_and_restart", func(t *testing.T) {
		svc := f.runtime.Location.(app.Service)
		c := geoCredentials203(t, f, "admin", f.password)
		makeCmd := func(code string) org.LocationCommand {
			return org.LocationCommand{Action: "CREATE", SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Create: org.LocationCreateInput{Code: code, Name: code, CountryRegionID: h.countries["MO"], AddressLine: "一号", OperationType: "CREATE", OperationReason: "NEW_LOCATION"}}
		}
		cmd := makeCmd("ROLLBACK")
		failed := app.Service{Store: locationFault204{Store: svc.Store, rollback: true}}
		_, e := failed.Maintain(f.ctx, c, cmd)
		if e == nil {
			t.Fatal("rollback accepted")
		}
		checkSubmission140(t, f, "location", cmd.Create.Code, cmd.SubmissionID, 0, 0, "")
		r, e := svc.Resolve(f.ctx, c, cmd.SubmissionID)
		must(t, e)
		if r.Outcome != "not_applied" {
			t.Fatal(r)
		}
		if _, err := svc.Maintain(f.ctx, c, cmd); err == nil || err.Error() != "M02_REQUEST_CLOSED" {
			t.Fatal("rolled-back submission reopened", err)
		}
		checkSubmission140(t, f, "location", cmd.Create.Code, cmd.SubmissionID, 0, 0, "not_applied")
		cmd = makeCmd("LOSTACK")
		lost := app.Service{Store: locationFault204{Store: svc.Store, lostAck: true}}
		_, e = lost.Maintain(f.ctx, c, cmd)
		if e == nil {
			t.Fatal("ack fault missing")
		}
		checkSubmission140(t, f, "location", cmd.Create.Code, cmd.SubmissionID, 1, 1, "")
		r, e = svc.Resolve(f.ctx, c, cmd.SubmissionID)
		must(t, e)
		if r.Outcome != "accepted" {
			t.Fatal(r)
		}
		retry, err := svc.Maintain(f.ctx, c, cmd)
		must(t, err)
		if !retry.Replayed || retry.SourceVersion == nil || *retry.SourceVersion != *r.SourceVersion {
			t.Fatal("lost acknowledgement created another source")
		}
		checkSubmission140(t, f, "location", cmd.Create.Code, cmd.SubmissionID, 1, 1, "")

		restartTimelineReceipts(t, f, h.b, []string{cmd.SubmissionID}, "location")
		svc = f.runtime.Location.(app.Service)

		after, e := svc.Resolve(f.ctx, c, cmd.SubmissionID)
		must(t, e)
		if after.SourceVersion == nil || *after.SourceVersion != *r.SourceVersion {
			t.Fatal("restart receipt")
		}
	})
}

type locationFault204 struct {
	app.Store
	rollback, lostAck bool
}

func (s locationFault204) WithinLocationWrite(ctx context.Context, c access.Credentials, fn func(app.Unit) error) error {
	e := s.Store.WithinLocationWrite(ctx, c, func(u app.Unit) error {
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

func TestLocation204EnterpriseDate(t *testing.T) {
	for _, start := range []string{"1984-01-01", "2020-01-01"} {
		t.Run(start, func(t *testing.T) {
			f := freshWithEnterpriseStart(t, start)
			must(t, f.runtime.InitializeCountryCatalog(f.ctx, access.LoginInput{Login: "admin", Password: f.password}))
			b := f.login("admin", f.password)
			body := map[string]any{"code": "DATED", "name": "日期地点", "countryRegionId": geoCountry203(t, f, "MO"), "addressLine": "一号", "operationType": "CREATE", "operationReason": "NEW_LOCATION"}
			ctx := b.call("GET", "/api/v1/locations/context", nil, "", 200)
			if ctx["defaultEffectiveDate"] != "1900-01-01" {
				t.Fatal(ctx)
			}
			b.call("POST", "/api/v1/locations", body, foundation.NewID(), 400)
			body["effectiveDate"] = start
			b.call("POST", "/api/v1/locations", body, foundation.NewID(), 201)
			body["code"] = "FUTURE"
			body["effectiveDate"] = "2099-01-01"
			result := b.call("POST", "/api/v1/locations", body, foundation.NewID(), 201)
			v := b.call("GET", "/api/v1/locations/"+result["locationId"].(string), nil, "", 200)
			if v["status"] != "not_effective" || v["selectedRecordIsFuture"] != true {
				t.Fatal("future projection")
			}
		})
	}
}

// Faults run after all CSV rows reached a real private file, before publication.
type locationExportFault204 struct {
	app.Store
	file   *locationExportFile204
	cancel context.CancelFunc
}

func (s *locationExportFault204) CreateLocationExport(ctx context.Context) (app.ExportFile, error) {
	file, e := s.Store.CreateLocationExport(ctx)
	if e != nil {
		return nil, e
	}
	s.file = &locationExportFile204{ExportFile: file, cancel: s.cancel}
	return s.file, nil
}

type locationExportFile204 struct {
	app.ExportFile
	bytes          int
	sealed, closed bool
	cancel         context.CancelFunc
}

func (f *locationExportFile204) Write(p []byte) (int, error) {
	n, e := f.ExportFile.Write(p)
	f.bytes += n
	return n, e
}
func (f *locationExportFile204) Seal() error {
	f.sealed = true
	if f.cancel != nil {
		f.cancel()
		return context.Canceled
	}
	return io.ErrUnexpectedEOF
}
func (f *locationExportFile204) Close() error { f.closed = true; return f.ExportFile.Close() }

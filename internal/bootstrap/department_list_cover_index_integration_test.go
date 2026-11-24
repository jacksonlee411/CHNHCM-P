//go:build integration

package bootstrap_test

import (
	"net/url"
	"strings"
	"testing"

	"chnhcm.local/chnhcm/internal/foundation"
)

func TestDepartmentCoverIndexValidMaximumNames(t *testing.T) {
	f := fresh(t)
	var installed bool
	must(t, f.admin.QueryRow(f.ctx, `SELECT to_regclass('organization.department_list_span_cover') IS NOT NULL`).Scan(&installed))
	if !installed {
		t.Fatal("list covering index missing from the current migration")
	}
	admin := f.login("admin", f.password)
	bu := admin.call("POST", "/api/v1/business-units", createBody("COVER_WIDE_BU", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	// Distinct supplementary-plane CJK letters keep all 200 characters legal,
	// four UTF-8 bytes each, without a repetitive short-string fixture.
	wide := func(seed int) string {
		value := make([]rune, 200)
		for i := range value {
			value[i] = rune(0x20000 + (i*7919+seed)%40000)
		}
		return string(value)
	}
	code, name, short := strings.Repeat("W", 40), wide(17), wide(3571)
	body := departmentCreateBody(code, bu)
	body["name"], body["shortName"] = name, short
	created := admin.call("POST", "/api/v1/departments", body, foundation.NewID(), 201)
	id, source := created["departmentId"].(string), created["sourceVersion"].(string)
	path := "/api/v1/departments/" + id
	query := "/api/v1/departments?q=" + url.QueryEscape(code)
	check := func(wantedName, wantedShort string) {
		result := admin.call("GET", query, nil, "", 200)
		rows := result["items"].([]any)
		if result["count"] != float64(1) || len(rows) != 1 {
			t.Fatal("maximum-field object missing from list")
		}
		row := rows[0].(map[string]any)
		if row["id"] != id || row["name"] != wantedName || row["shortName"] != wantedShort {
			t.Fatal("maximum fields did not roundtrip")
		}
		var nameBytes, keyBytes, shortBytes int
		must(t, f.admin.QueryRow(f.ctx, `SELECT octet_length(q_name),octet_length(q_name_key),octet_length(q_short_key) FROM organization.department_read_segment WHERE object_id=$1 AND span @> '2026-01-01'::date`, id).Scan(&nameBytes, &keyBytes, &shortBytes))
		if nameBytes != 800 || keyBytes != 800 || shortBytes != 800 {
			t.Fatal("fixture no longer reaches field byte bounds", nameBytes, keyBytes, shortBytes)
		}
	}
	check(name, short)
	name, short = wide(61), wide(12345)
	correction := map[string]any{"expectedSourceVersion": source, "expectedRecordRevision": created["targetRevision"], "changes": map[string]any{"name": name, "shortName": short}}
	accepted := admin.call("POST", path+"/records/"+created["targetRecordId"].(string)+"/correct", correction, foundation.NewID(), 200)
	if accepted["sourceVersion"] == source {
		t.Fatal("correction did not create a new source")
	}
	check(name, short)
	admin.call("GET", path+"?sourceVersion="+source, nil, "", 200)
	selection := admin.call("POST", "/api/v1/departments/selection/start", map[string]any{"query": map[string]any{"q": code}, "selection": map[string]any{"mode": "allMatching", "excludedIds": []string{}}, "page": 1, "pageSize": 20}, "", 200)
	if selection["summary"].(map[string]any)["selectedCount"] != float64(1) {
		t.Fatal("maximum-field selection changed membership")
	}
}

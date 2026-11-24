//go:build integration

package bootstrap_test

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	geo "chnhcm.local/chnhcm/internal/contexts/organization/application/geoarea"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"net/url"
	"strings"
	"testing"
)

func TestLocation204HongKong(t *testing.T) {
	f := freshWithEnterpriseStart(t, "1900-01-01")
	must(t, f.runtime.InitializeCountryCatalog(f.ctx, access.LoginInput{Login: "admin", Password: f.password}))
	publication, commands := geoPartial203(t, f)
	must(t, f.runtime.GeoArea.(geo.Service).InitializeCatalog(f.ctx, geoCredentials203(t, f, "admin", f.password), publication, commands))
	b := f.login("admin", f.password)
	reader := f.login("reader", f.readPassword)
	country, area := geoCountry203(t, f, "HK"), geoID203(t, f, "CW")
	body := map[string]any{"code": "HK-204", "name": "香港办公地点", "nameEn": "Hong Kong office", "countryRegionId": country, "geoAreaId": area, "addressLine": "中环一号", "postalCode": "00100", "operationType": "CREATE", "operationReason": "NEW_LOCATION"}
	sid := foundation.NewID()
	code, saved, err := b.response("POST", "/api/v1/locations", body, sid)
	must(t, err)
	if code != 201 {
		_, cause := f.runtime.Location.Maintain(f.ctx, geoCredentials203(t, f, "admin", f.password), org.LocationCommand{Action: "CREATE", SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Create: org.LocationCreateInput{Code: "DIAG204", Name: "诊断地点", CountryRegionID: country, GeoAreaID: &area, AddressLine: "一号", OperationType: "CREATE", OperationReason: "NEW_LOCATION"}})
		t.Fatalf("HTTP create status=%d code=%v; direct cause=%v", code, saved["code"], cause)
	}
	id := saved["locationId"].(string)
	first := b.call("GET", "/api/v1/locations/"+id, nil, "", 200)
	selected := first["selectedRecord"].(map[string]any)
	if selected["effectiveDate"] != "1900-01-01" || selected["postalCode"] != "00100" || selected["adoptionDate"] != "1900-01-01" || len(selected["geoAreaPath"].([]any)) != 2 {
		t.Fatal("address/default projection", selected)
	}
	replay := b.call("POST", "/api/v1/locations", body, sid, 201)
	if replay["replayed"] != true || replay["locationId"] != id {
		t.Fatal("create replay", replay)
	}
	reader.call("POST", "/api/v1/locations", body, foundation.NewID(), 403)
	reader.call("GET", "/api/v1/locations/"+id, nil, "", 200)
	corrected := b.call("POST", "/api/v1/locations/"+id+"/records/"+selected["recordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": first["headSourceVersion"], "expectedRecordRevision": selected["revision"], "changes": map[string]any{"addressLine": "中环二号 三楼"}}, foundation.NewID(), 200)
	old := b.call("GET", "/api/v1/locations/"+id+"?sourceVersion="+saved["sourceVersion"].(string), nil, "", 200)
	if old["selectedRecord"].(map[string]any)["addressLine"] != "中环一号" {
		t.Fatal("old source mutated", old)
	}
	now := b.call("GET", "/api/v1/locations/"+id, nil, "", 200)
	if now["selectedRecord"].(map[string]any)["addressLine"] != "中环二号 三楼" || now["headSourceVersion"] != corrected["sourceVersion"] {
		t.Fatal(now)
	}
	list := b.call("GET", "/api/v1/locations?countryRegionId="+country+"&filterField=postalCode&filterOperator=equals&filterValue=00100", nil, "", 200)
	if list["count"] != float64(1) {
		t.Fatal(list)
	}
	log := b.call("GET", "/api/v1/locations/"+id+"/changes", nil, "", 200)
	kinds := map[string]string{"status": "state", "operationType": "classification", "operationReason": "classification", "countryRegionId": "reference", "geoAreaId": "reference"}
	seen := map[string]bool{}
	for _, raw := range log["items"].([]any) {
		for _, item := range raw.(map[string]any)["items"].([]any) {
			for _, rawDiff := range item.(map[string]any)["diffs"].([]any) {
				diff := rawDiff.(map[string]any)
				field := diff["field"].(string)
				if kind, ok := kinds[field]; ok {
					after := diff["after"].(map[string]any)
					if after["kind"] != kind {
						t.Fatalf("change %s has kind %v", field, after["kind"])
					}
					if kind == "reference" && after["label"] == nil {
						t.Fatalf("change %s lacks authorized label", field)
					}
					if kind == "state" && after["code"] != "active" {
						t.Fatal("state change has no display code")
					}
					seen[field] = true
				}
			}
		}
	}
	if len(seen) != len(kinds) {
		t.Fatal("incomplete typed change values", seen)
	}
	b.call("POST", "/api/v1/locations/submissions/"+sid+"/resolve", nil, "", 200)
	if f.scalar("SELECT sum(cardinality(source_refs)) FROM organization.location_history") != 2 {
		t.Fatal("unexpected sources")
	}
	// Deleted fields are rejected rather than silently accepted or searchable.
	body["addressExtra"] = "旧地址补充"
	b.call("POST", "/api/v1/locations", body, foundation.NewID(), 400)
	b.call("GET", "/api/v1/locations?sort=addressExtra_desc", nil, "", 400)
	b.call("GET", "/api/v1/locations?filterField=addressExtra&filterOperator=contains&filterValue=x", nil, "", 400)
	current := now["selectedRecord"].(map[string]any)
	longAddress := strings.Repeat("址", 1001)
	b.call("POST", "/api/v1/locations/"+id+"/records/"+current["recordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": now["headSourceVersion"], "expectedRecordRevision": current["revision"], "changes": map[string]any{"addressLine": longAddress}}, foundation.NewID(), 200)
	full := b.call("GET", "/api/v1/locations?filterField=addressLine&filterOperator=equals&filterValue="+url.QueryEscape(longAddress), nil, "", 200)
	if full["count"] != float64(1) || full["items"].([]any)[0].(map[string]any)["addressLine"] != longAddress {
		t.Fatal("full merged address cannot be filtered")
	}
	if _, exists := full["items"].([]any)[0].(map[string]any)["addressExtra"]; exists {
		t.Fatal("removed address field returned")
	}
	t.Log("G4 Hong Kong real HTTP create/default/address/correction/exact source/read-only denial/replay/query verified")
}

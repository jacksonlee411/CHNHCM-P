//go:build integration

package bootstrap_test

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	geoapp "chnhcm.local/chnhcm/internal/contexts/organization/application/geoarea"
	"chnhcm.local/chnhcm/internal/foundation"
	"fmt"
	"testing"
)

func TestDepartmentNavigation302BeyondOrdinaryPage(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	bu := b.call("POST", "/api/v1/business-units", createBody("NAV302_BU", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	ids := make([]string, 31)
	for i := range ids {
		body := departmentCreateBody(fmt.Sprintf("NAV302_%03d", i), bu)
		ids[i] = b.call("POST", "/api/v1/departments", body, foundation.NewID(), 201)["departmentId"].(string)
	}
	query := map[string]any{"q": "NAV302_"}
	for _, client := range []*browser{b, f.login("reader", f.readPassword)} {
		page := client.call("GET", "/api/v1/departments?q=NAV302_&pageSize=20", nil, "", 200)
		if page["count"] != float64(len(ids)) || len(page["items"].([]any)) != 20 {
			t.Fatal("ordinary first page", page)
		}
		for _, anchor := range []int{0, 24, 30} {
			for _, direction := range []string{"current", "previous", "next"} {
				target := anchor
				if direction == "previous" {
					target--
				} else if direction == "next" {
					target++
				}
				body := map[string]any{"query": query, "anchorId": ids[anchor], "direction": direction, "sort": "code"}
				result := client.call("POST", "/api/v1/departments/navigation", body, "", 200)
				if target < 0 || target >= len(ids) {
					if result["targetId"] != nil {
						t.Fatal("navigation boundary", direction, result)
					}
					continue
				}
				if result["targetId"] != ids[target] || result["position"] != float64(target+1) || result["count"] != float64(len(ids)) {
					t.Fatal("global navigation position", anchor, direction, result)
				}
				if target == 0 && result["previousId"] != nil || target > 0 && result["previousId"] != ids[target-1] || target == len(ids)-1 && result["nextId"] != nil || target+1 < len(ids) && result["nextId"] != ids[target+1] {
					t.Fatal("global navigation neighbors", anchor, direction, result)
				}
			}
		}
	}
}

func TestNavigation302CatalogConsumers(t *testing.T) {
	f := fresh(t)
	must(t, f.runtime.InitializeCountryCatalog(f.ctx, access.LoginInput{Login: "admin", Password: f.password}))
	p, commands := geoPartial203(t, f)
	svc := f.runtime.GeoArea.(geoapp.Service)
	must(t, svc.InitializeCatalog(f.ctx, geoCredentials203(t, f, "admin", f.password), p, commands))
	for _, b := range []*browser{f.login("admin", f.password), f.login("reader", f.readPassword)} {
		for _, kind := range []string{"country-regions", "geo-areas"} {
			for _, order := range []string{"code", "name", "name_desc"} {
				var rows []any
				for page := 1; ; page++ {
					list := b.call("GET", fmt.Sprintf("/api/v1/%s?pageSize=100&page=%d&sort=%s", kind, page, order), nil, "", 200)
					rows = append(rows, list["items"].([]any)...)
					if len(rows) >= int(list["count"].(float64)) {
						break
					}
				}
				query := map[string]any{"status": "all"}
				body := map[string]any{"direction": "current", "query": query}
				if kind == "geo-areas" {
					query["sort"] = order
				} else {
					body["sort"] = order
				}
				id := func(index int) any { return rows[index].(map[string]any)["id"] }
				for _, index := range []int{0, len(rows) / 2, len(rows) - 1} {
					body["anchorId"] = id(index)
					n := b.call("POST", "/api/v1/"+kind+"/navigation", body, "", 200)
					if n["targetId"] != id(index) || n["position"] != float64(index+1) || n["count"] != float64(len(rows)) {
						t.Fatal("catalog current position", kind, order, index, n)
					}
					if index == 0 && n["previousId"] != nil || index > 0 && n["previousId"] != id(index-1) || index == len(rows)-1 && n["nextId"] != nil || index+1 < len(rows) && n["nextId"] != id(index+1) {
						t.Fatal("catalog neighbors", kind, order, index, n)
					}
				}
				body["anchorId"], body["direction"] = id(0), "next"
				n := b.call("POST", "/api/v1/"+kind+"/navigation", body, "", 200)
				if n["targetId"] != id(1) || n["position"] != float64(2) || n["count"] != float64(len(rows)) {
					t.Fatal("catalog moved position", kind, order, n)
				}
			}
		}
	}
}

//go:build integration

package bootstrap_test

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"fmt"
	"reflect"
	"testing"
)

func TestLocation205NavigationConsumers(t *testing.T) {
	f := freshWithEnterpriseStart(t, "1900-01-01")
	must(t, f.runtime.InitializeCountryCatalog(f.ctx, access.LoginInput{Login: "admin", Password: f.password}))
	h := location204{t: t, f: f, b: f.login("admin", f.password), countries: map[string]string{"MO": geoCountry203(t, f, "MO")}}
	for i := 0; i < 3; i++ {
		v := h.create(fmt.Sprintf("NAV205-%d", i), "MO", "", "2000-01-01")
		h.correct(v, map[string]any{"name": "相同名称", "addressLine": "导航地址"}, nil, 200)
	}
	before := locationBusinessSnapshot204(t, f)
	for _, b := range []*browser{h.b, f.login("reader", f.readPassword)} {
		for _, field := range []string{"code", "name", "nameEn", "addressLine", "postalCode", "status", "effectiveDate"} {
			for _, suffix := range []string{"", "_desc"} {
				order := field + suffix
				list := b.call("GET", "/api/v1/locations?q=NAV205-&sort="+order, nil, "", 200)
				rows := list["items"].([]any)
				if list["count"] != float64(3) || len(rows) != 3 {
					t.Fatal("expected three navigation fixtures", list)
				}
				id := func(i int) any { return rows[i].(map[string]any)["id"] }
				query := map[string]any{"q": "NAV205-", "status": "all", "countryRegionId": h.countries["MO"], "filterField": "addressLine", "filterOperator": "contains", "filterValue": "导航地址"}
				body := map[string]any{"query": query, "sort": order}
				for i := range rows {
					for direction, offset := range map[string]int{"current": 0, "previous": -1, "next": 1} {
						body["anchorId"], body["direction"] = id(i), direction
						n := b.call("POST", "/api/v1/locations/navigation", body, "", 200)
						at := i + offset
						if at < 0 || at >= len(rows) {
							if n["targetId"] != nil || n["count"] != nil || n["position"] != nil {
								t.Fatal("navigation boundary", n)
							}
							continue
						}
						if n["targetId"] != id(at) || n["position"] != float64(at+1) || n["count"] != float64(3) {
							t.Fatal("position/count", order, direction, n)
						}
						if at == 0 && n["previousId"] != nil || at > 0 && n["previousId"] != id(at-1) || at == 2 && n["nextId"] != nil || at < 2 && n["nextId"] != id(at+1) {
							t.Fatal("neighbors/tie order", order, n)
						}
					}
				}
				body["anchorId"], body["direction"] = foundation.NewID(), "current"
				if n := b.call("POST", "/api/v1/locations/navigation", body, "", 200); n["targetId"] != nil || n["count"] != nil {
					t.Fatal("unknown anchor", n)
				}
			}
		}
	}
	if after := locationBusinessSnapshot204(t, f); !reflect.DeepEqual(before, after) {
		t.Fatal("navigation changed business facts")
	}
	t.Log("G5 publication: both roles, seven sorts in both directions, tied values, full filters, current/previous/next positions, boundaries and original facts passed")
}

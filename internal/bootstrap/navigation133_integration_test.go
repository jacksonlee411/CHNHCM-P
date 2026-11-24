//go:build integration

package bootstrap_test

import (
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"fmt"
	"net/url"
	"slices"
	"testing"
)

func TestNavigation133CurrentAnchorBothObjects(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	bu := b.call("POST", "/api/v1/business-units", createBody("NAV133_BU", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	dep := []string{}
	bus := []string{}
	for i := 0; i < 3; i++ {
		code := fmt.Sprintf("NAV133_%02d", i)
		dep = append(dep, b.call("POST", "/api/v1/departments", departmentCreateBody(code, bu), foundation.NewID(), 201)["departmentId"].(string))
		bus = append(bus, b.call("POST", "/api/v1/business-units", createBody(code, "2026-01-01"), foundation.NewID(), 201)["buId"].(string))
	}
	origin, _ := url.Parse(f.server.URL)
	c := foundation.Credentials{Token: b.client.Jar.Cookies(origin)[0].Value}
	for _, kind := range []string{"departments", "business-units"} {
		ids := dep
		if kind == "business-units" {
			ids = bus
		}
		in := org.NavigationInput{AnchorID: ids[1], Direction: "next", Query: org.SelectionQuery{Q: "NAV133_0"}}
		for _, client := range []*browser{b, f.login("reader", f.readPassword)} {
			for i, id := range ids {
				current := org.NavigationInput{AnchorID: id, Direction: "current", Query: in.Query}
				r := client.call("POST", "/api/v1/"+kind+"/navigation", current, "", 200)
				if r["targetId"] != id || r["position"] != float64(i+1) || r["count"] != float64(3) {
					t.Fatal("current position/count", kind, i, r)
				}
				if i == 0 && r["previousId"] != nil || i == 2 && r["nextId"] != nil || i > 0 && r["previousId"] != ids[i-1] || i < 2 && r["nextId"] != ids[i+1] {
					t.Fatal("neighbors", kind, i, r)
				}
			}
		}
		for _, order := range []string{"name", "name_desc"} {
			ordered := slices.Clone(ids)
			// Equal department names use UUID as the final stable key;
			// BU retains its established code/UUID tie order.
			if kind == "departments" {
				slices.Sort(ordered)
			}
			for i, id := range ordered {
				r := b.call("POST", "/api/v1/"+kind+"/navigation", org.NavigationInput{AnchorID: id, Direction: "current", Query: in.Query, Sort: order}, "", 200)
				if r["position"] != float64(i+1) || r["count"] != float64(3) {
					t.Fatal("stable name ordering", kind, order, r)
				}
			}
		}
		var actual org.NavigationResult
		var e error
		if kind == "departments" {
			actual, e = f.runtime.Department.Navigate(f.ctx, c, org.DepartmentNavigationInput{AnchorID: in.AnchorID, Direction: in.Direction, Sort: in.Sort, Query: org.DepartmentSelectionQuery{Q: in.Query.Q}})
		} else {
			actual, e = f.runtime.Organization.Navigate(f.ctx, c, in)
		}
		must(t, e)
		if actual.TargetID == nil || *actual.TargetID != ids[2] {
			t.Fatal(kind, actual)
		}
		if actual.Position != 3 || actual.Count != 3 || actual.PreviousID == nil || *actual.PreviousID != ids[1] || actual.NextID != nil {
			t.Fatal("moved position", kind, actual)
		}
		in.Direction = "previous"
		r := b.call("POST", "/api/v1/"+kind+"/navigation", in, "", 200)
		if r["targetId"] != ids[0] || r["navigationVersion"] != nil || r["page"] != nil {
			t.Fatal(kind, r)
		}
		in.AnchorID = ids[0]
		r = b.call("POST", "/api/v1/"+kind+"/navigation", in, "", 200)
		if r["targetId"] != nil {
			t.Fatal("first boundary", r)
		}
		in.Query.Q = "missing"
		b.call("POST", "/api/v1/"+kind+"/navigation", in, "", 409)
		in.Query.Q = ""
		in.AnchorID = foundation.NewID()
		b.call("POST", "/api/v1/"+kind+"/navigation", in, "", 409)
	}
	path := map[string]any{"anchorId": dep[1], "direction": "current", "query": map[string]any{"q": "NAV133_0", "filterField": "departmentPath", "filterOperator": "contains", "filterValue": "研发部门"}}
	r := b.call("POST", "/api/v1/departments/navigation", path, "", 200)
	if r["position"] != float64(2) || r["count"] != float64(3) || r["previousId"] != dep[0] || r["nextId"] != dep[2] {
		t.Fatal("path exact totals", r)
	}
	reader := f.login("reader", f.readPassword)
	_, e := f.admin.Exec(f.ctx, `UPDATE access.subject_grant SET scope_kind='department',scope_id=$3 WHERE instance_id=$1 AND principal_id=$2 AND action='department.read'`, f.instance, reader.subject, dep[1])
	must(t, e)
	r = reader.call("POST", "/api/v1/departments/navigation", org.NavigationInput{AnchorID: dep[1], Direction: "current", Query: org.SelectionQuery{Q: "NAV133_0"}}, "", 200)
	if r["position"] != float64(1) || r["count"] != float64(1) || r["previousId"] != nil || r["nextId"] != nil {
		t.Fatal("current authorization must bound totals", r)
	}
	reader.call("POST", "/api/v1/departments/navigation", org.NavigationInput{AnchorID: dep[0], Direction: "current", Query: org.SelectionQuery{Q: "NAV133_0"}}, "", 409)
	_, e = f.admin.Exec(f.ctx, `UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'name') WHERE instance_id=$1 AND principal_id=$2 AND action='department.read'`, f.instance, reader.subject)
	must(t, e)
	reader.call("POST", "/api/v1/departments/navigation", org.NavigationInput{AnchorID: dep[1], Direction: "current", Query: org.SelectionQuery{Q: "研发部门"}}, "", 409)
	reader.call("POST", "/api/v1/departments/navigation", org.NavigationInput{AnchorID: dep[1], Direction: "current", Query: org.SelectionQuery{FilterField: "name", FilterOperator: "contains", FilterValue: "研发"}}, "", 403)
	_, e = f.admin.Exec(f.ctx, `UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'status') WHERE instance_id=$1 AND principal_id=$2 AND action='department.read'`, f.instance, reader.subject)
	must(t, e)
	reader.call("POST", "/api/v1/departments/navigation", org.NavigationInput{AnchorID: dep[1], Direction: "current", Query: org.SelectionQuery{Status: "active"}}, "", 403)
}

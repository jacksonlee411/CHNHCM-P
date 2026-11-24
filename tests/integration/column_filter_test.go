//go:build integration

package integration_test

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"chnhcm.local/chnhcm/internal/foundation"
)

func TestSingleColumnFilter(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	reader := f.login("reader", f.readPassword)
	const root = "/api/v1/business-units"
	get := func(field, op, value, extra string) map[string]any {
		return b.getOnDay("2026-04-01", root+"?filterField="+url.QueryEscape(field)+"&filterOperator="+url.QueryEscape(op)+"&filterValue="+url.QueryEscape(value)+extra)
	}
	assertCount := func(got map[string]any, want int) {
		t.Helper()
		if got["count"] != float64(want) {
			t.Fatalf("count want %d got %v", want, got)
		}
	}
	ids := []string{}
	for i := 0; i < 23; i++ {
		in := createBody(fmt.Sprintf("CF_%02d", i), "2026-01-01")
		in["name"] = "共同 名称"
		if i == 22 {
			in["name"] = "字面%_值"
		}
		ids = append(ids, b.call("POST", root, in, foundation.NewID(), 201)["buId"].(string))
	}
	in := createBody("OTHER_CODE", "2026-01-01")
	in["name"] = "CF_00"
	b.call("POST", root, in, foundation.NewID(), 201)
	in = createBody("FUTURE_CODE", "2027-01-01")
	in["name"] = "未来资料"
	future := b.call("POST", root, in, foundation.NewID(), 201)["buId"].(string)
	d := detail(b, ids[0])
	b.call("POST", root+"/"+ids[0]+"/records", addBody(d, "2026-03-01", map[string]any{"name": "新 名称"}), foundation.NewID(), 201)
	d = detail(b, ids[1])
	b.call("POST", root+"/"+ids[1]+"/deactivate", lifeBody(d, "2026-02-01", "DEACTIVATE", nil), foundation.NewID(), 201)
	before := businessSnapshot(t, f)
	t.Run("literal-independent-columns-and-page", func(t *testing.T) {
		assertCount(b.call("GET", root+"?q=CF_00", nil, "", 200), 2)
		a := get("code", "equals", " cf_00 ", "")
		assertCount(a, 1)
		if a["items"].([]any)[0].(map[string]any)["id"] != ids[0] {
			t.Fatal(a)
		}
		assertCount(get("name", "equals", "CF_00", ""), 1)
		assertCount(get("name", "contains", "%_", ""), 1)
		assertCount(get("name", "equals", "  新 名称  ", ""), 1)
		a = get("code", "contains", "cf_", "&page=99&pageSize=20")
		assertCount(a, 23)
		if a["page"] != float64(2) || len(a["items"].([]any)) != 3 {
			t.Fatal(a)
		}
		assertCount(get("code", "contains", "CF_", "&q="+url.QueryEscape("共同 名称")), 21)
		assertCount(get("name", "equals", "共同 名称", "&sort=name_desc&pageSize=50"), 21)
		assertCount(get("code", "equals", "NOT_FOUND", ""), 0)
	})
	t.Run("status-set-and-resolved-record-date", func(t *testing.T) {
		assertCount(get("status", "isAnyOf", "active,inactive", ""), 24)
		assertCount(get("status", "is", "inactive", ""), 1)
		assertCount(get("status", "isAnyOf", "active,inactive", "&status=not_effective"), 0)
		a := get("effectiveDate", "is", "2027-01-01", "")
		assertCount(a, 1)
		if a["items"].([]any)[0].(map[string]any)["id"] != future || a["items"].([]any)[0].(map[string]any)["status"] != "not_effective" {
			t.Fatal(a)
		}
		for _, tt := range []struct {
			op    string
			count int
		}{{"is", 1}, {"before", 23}, {"after", 1}, {"onOrBefore", 24}, {"onOrAfter", 2}} {
			assertCount(get("effectiveDate", tt.op, "2026-03-01", ""), tt.count)
		}
		// No historical-any-record matching: CF_00's displayed record is March, not January.
		assertCount(get("effectiveDate", "is", "2026-01-01", "&q=CF_00"), 1)
		assertCount(get("name", "equals", "新 名称", ""), 1)
	})
	t.Run("invalid-contract-rejected", func(t *testing.T) {
		for _, query := range []string{
			"filterField=code", "filterValue=aa", "filterField=", "filterField=code&filterOperator=contains&filterValue=",
			"filterField=nameEn&filterOperator=equals&filterValue=secret",
			"filterField=code&filterOperator=isAnyOf&filterValue=a,b", "filterField=status&filterOperator=isAnyOf&filterValue=all",
			"filterField=status&filterOperator=is&filterValue=active,inactive", "filterField=status&filterOperator=isAnyOf&filterValue=active,active",
			"filterField=effectiveDate&filterOperator=is&filterValue=2026-02-30", "filterField=effectiveDate&filterOperator=is&filterValue=0000-01-01",
			"filterField=effectiveDate&filterOperator=is&filterValue=2026-01-01T00:00:00Z",
			"filterField=name&filterOperator=isEmpty&filterValue=aa", "filterField=name&filterOperator=contains&filterValue=" + strings.Repeat("中", 401),
			"filterField=code&filterField=name&filterOperator=equals&filterValue=a",
		} {
			b.call("GET", root+"?"+query, nil, "", 400)
		}
	})
	t.Run("authorized-fields-object-scope-and-count", func(t *testing.T) {
		mutateLifeGrants(f, reader.subject, `UPDATE access.subject_grant SET read_fields=ARRAY['code'] WHERE principal_id=$1 AND action='bu.read'`, reader.subject)
		for _, q := range []string{"filterField=name&filterOperator=contains&filterValue=" + url.QueryEscape("共同"), "filterField=effectiveDate&filterOperator=is&filterValue=2026-01-01"} {
			assertCount(reader.call("GET", root+"?"+q, nil, "", 200), 0)
		}
		assertCount(reader.call("GET", root+"?filterField=code&filterOperator=contains&filterValue=CF_", nil, "", 200), 23)
		mutateLifeGrants(f, reader.subject, `UPDATE access.subject_grant SET scope_kind='bu',scope_id=$2 WHERE principal_id=$1 AND action='bu.read'`, reader.subject, ids[0])
		assertCount(reader.call("GET", root+"?filterField=code&filterOperator=contains&filterValue=CF_", nil, "", 200), 1)
		mutateLifeGrants(f, reader.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='bu.read'`, reader.subject)
		reader.call("GET", root+"?filterField=code&filterOperator=equals&filterValue=CF_00", nil, "", 403)
	})
	if !reflect.DeepEqual(before, businessSnapshot(t, f)) {
		t.Fatal("filtering changed business facts/history")
	}
}

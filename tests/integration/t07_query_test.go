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

func TestT07LiteralSearchStablePaginationAndFields(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	reader := f.login("reader", f.readPassword)
	expected := []string{}
	for i := 0; i < 23; i++ {
		code := fmt.Sprintf("T07_ORDER_%02d", i)
		in := createBody(code, "2026-01-01")
		in["name"] = "相同名称"
		in["nameEn"] = "OnlyEnglishSecret"
		b.call("POST", "/api/v1/business-units", in, foundation.NewID(), 201)
		expected = append(expected, code)
	}
	for i, name := range []string{"甲%_乙", "甲XX乙", strings.Repeat("中", 200)} {
		in := createBody(fmt.Sprintf("T07_LITERAL_%d", i), "2026-01-01")
		in["name"] = name
		b.call("POST", "/api/v1/business-units", in, foundation.NewID(), 201)
	}
	for _, q := range []string{"%_", "甲%", "_乙", strings.Repeat("中", 200)} {
		got := b.call("GET", "/api/v1/business-units?q="+url.QueryEscape(q), nil, "", 200)
		if got["count"] != float64(1) {
			t.Fatal("literal became a pattern or name truncated", q, got)
		}
	}
	for _, sort := range []string{"name", "name_desc"} {
		codes := []string{}
		for _, page := range []int{1, 2} {
			path := fmt.Sprintf("/api/v1/business-units?q=%s&sort=%s&page=%d&pageSize=20", url.QueryEscape("相同名称"), sort, page)
			a := b.call("GET", path, nil, "", 200)
			again := b.call("GET", path, nil, "", 200)
			if !reflect.DeepEqual(a["items"], again["items"]) || a["count"] != float64(23) {
				t.Fatal("unstable page", a, again)
			}
			for _, v := range a["items"].([]any) {
				codes = append(codes, v.(map[string]any)["code"].(string))
			}
		}
		if !reflect.DeepEqual(codes, expected) {
			t.Fatal("name ties lost code order or repeated a row", sort, codes)
		}
	}
	mutateLifeGrants(f, reader.subject, `UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'nameEn') WHERE principal_id=$1`, reader.subject)
	q := reader.call("GET", "/api/v1/business-units?q=OnlyEnglishSecret&sort=name", nil, "", 200)
	if q["count"] != float64(0) {
		t.Fatal("hidden English searched", q)
	}
	q = reader.call("GET", "/api/v1/business-units?q="+url.QueryEscape("相同名称")+"&sort=name", nil, "", 200)
	if q["count"] != float64(23) {
		t.Fatal("Chinese-only reader cannot search Chinese", q)
	}
	mutateLifeGrants(f, reader.subject, `UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'name') WHERE principal_id=$1`, reader.subject)
	q = reader.call("GET", "/api/v1/business-units?q="+url.QueryEscape("相同名称"), nil, "", 200)
	if q["count"] != float64(0) {
		t.Fatal("hidden Chinese searched")
	}
	reader.call("GET", "/api/v1/business-units?sort=name", nil, "", 403)
}

func TestT07FutureFirstAndFourExactHistoricalAnswers(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	in := createBody("T07_FUTURE_HISTORY", "2026-06-01")
	in["name"] = "乙"
	in["nameEn"] = "Beta"
	id := b.call("POST", "/api/v1/business-units", in, foundation.NewID(), 201)["buId"].(string)
	path := "/api/v1/business-units/" + id
	old := b.getOnDay("2026-03-01", path)
	r := recordOn(t, old, "2026-06-01")
	b.call("POST", path+"/records/"+r["recordId"].(string)+"/correct", correctBody(old, r, map[string]any{"name": "丙", "nameEn": "Gamma"}), foundation.NewID(), 200)
	now := b.getOnDay("2026-03-01", path)
	for _, v := range []struct {
		d        map[string]any
		name, en string
	}{{old, "乙", "Beta"}, {now, "丙", "Gamma"}} {
		got := b.getOnDay("2026-03-01", path+"?sourceVersion="+v.d["headSourceVersion"].(string))
		if got["status"] != "not_effective" || got["viewFact"] != nil || !reflect.DeepEqual(got["timeline"], v.d["timeline"]) {
			t.Fatal("future history became current fact", got)
		}
		rr := recordOn(t, got, "2026-06-01")
		if rr["name"] != v.name || rr["nameEn"] != v.en {
			t.Fatal("future display did not use exact source", rr)
		}
	}
	in = createBody("T07_FOUR_HISTORY", "2026-01-01")
	in["name"] = "甲"
	in["nameEn"] = "Alpha"
	id = b.call("POST", "/api/v1/business-units", in, foundation.NewID(), 201)["buId"].(string)
	path = "/api/v1/business-units/" + id
	history := []map[string]any{detail(b, id)}
	b.call("POST", path+"/records", addBody(history[0], "2026-03-01", map[string]any{"name": "乙", "nameEn": "Beta"}), foundation.NewID(), 201)
	history = append(history, detail(b, id))
	r = recordOn(t, history[1], "2026-03-01")
	b.call("POST", path+"/records/"+r["recordId"].(string)+"/correct", correctBody(history[1], r, map[string]any{"name": "丙", "nameEn": "Gamma"}), foundation.NewID(), 200)
	history = append(history, detail(b, id))
	r = recordOn(t, history[2], "2026-03-01")
	b.call("POST", path+"/records/"+r["recordId"].(string)+"/delete", deleteBody(history[2], r), foundation.NewID(), 200)
	history = append(history, detail(b, id))
	before := businessSnapshot(t, f)
	b.call("POST", path+"/records/"+r["recordId"].(string)+"/correct", correctBody(history[3], r, map[string]any{"name": "已删目标"}), foundation.NewID(), 409)
	for table, v := range businessSnapshot(t, f) {
		if table != "terminals" && v != before[table] {
			t.Fatal("deleted correction wrote fact", table)
		}
	}
	for i, v := range history {
		got := b.getOnDay("2026-04-01", path+"?sourceVersion="+v["headSourceVersion"].(string))
		fact := got["viewFact"].(map[string]any)
		if fact["name"] != []string{"甲", "乙", "丙", "甲"}[i] || fact["nameEn"] != []string{"Alpha", "Beta", "Gamma", "Alpha"}[i] || !reflect.DeepEqual(got["timeline"], v["timeline"]) {
			t.Fatal("four source answers", i, got)
		}
	}
}

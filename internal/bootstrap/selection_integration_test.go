//go:build integration

package bootstrap_test

import (
	"bytes"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"testing"
	"time"
)

func selectionBody(list map[string]any, ids []string, all bool) map[string]any {
	choice := map[string]any{"mode": "explicit", "ids": ids}
	if all {
		choice = map[string]any{"mode": "allMatching", "excludedIds": ids}
	}
	return map[string]any{"query": map[string]any{"q": "QS", "status": "all"}, "selection": choice, "expectedSelectionVersion": list["selectionVersion"]}
}
func selectionDownload(t *testing.T, b *browser, body any, status int) []byte {
	t.Helper()
	raw, e := json.Marshal(body)
	must(t, e)
	req, e := http.NewRequest("POST", b.f.server.URL+"/api/v1/business-units/export", bytes.NewReader(raw))
	must(t, e)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", b.f.server.URL)
	req.Header.Set("X-CSRF-Token", b.csrf)
	res, e := b.client.Do(req)
	must(t, e)
	defer res.Body.Close()
	data, e := io.ReadAll(res.Body)
	must(t, e)
	if res.StatusCode != status {
		t.Fatalf("export status=%d want=%d %s", res.StatusCode, status, data)
	}
	if status != 200 {
		if strings.Contains(res.Header.Get("Content-Type"), "csv") || res.Header.Get("Content-Disposition") != "" {
			t.Fatal("failure emitted CSV headers")
		}
	} else if !strings.HasPrefix(string(data), "\xef\xbb\xbf") {
		t.Fatal("missing BOM")
	}
	return data
}
func TestQuerySelectionAuthorizationAndSnapshots(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	reader := f.login("reader", f.readPassword)
	ids := []string{}
	for i := 0; i < 137; i++ {
		v := b.call("POST", "/api/v1/business-units", createBody(fmt.Sprintf("QS%03d", i), "2026-01-01"), foundation.NewID(), 201)
		ids = append(ids, v["buId"].(string))
	}
	b.call("POST", "/api/v1/business-units", createBody("OTHER", "2026-01-01"), foundation.NewID(), 201)
	path := "/api/v1/business-units?q=QS"
	list := b.call("GET", path, nil, "", 200)
	if list["count"] != float64(137) || len(list["items"].([]any)) != 20 {
		t.Fatal("fixture")
	}
	for _, suffix := range []string{"&page=2", "&pageSize=100", "&sort=name_desc", "&q=qs"} {
		p := path + suffix
		if suffix == "&q=qs" {
			p = strings.Replace(path, "q=QS", "q= qs ", 1)
			p = strings.ReplaceAll(p, " ", "%20")
		}
		v := b.call("GET", p, nil, "", 200)
		if v["selectionVersion"] != list["selectionVersion"] || v["queryKey"] != list["queryKey"] {
			t.Fatal("presentation/normalization changed basis")
		}
	}
	body := selectionBody(list, []string{}, true)
	v := b.call("POST", "/api/v1/business-units/selection/resolve", body, "", 200)
	if v["selectedCount"] != float64(137) || v["canExport"] != true {
		t.Fatal(v)
	}
	body = selectionBody(list, []string{ids[24]}, true)
	v = b.call("POST", "/api/v1/business-units/selection/resolve", body, "", 200)
	if v["selectedCount"] != float64(136) {
		t.Fatal(v)
	}
	history := businessSnapshot(t, f)
	before := f.scalar("SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.bu_history")
	data := selectionDownload(t, b, body, 200)
	records, e := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(data), "\xef\xbb\xbf"))).ReadAll()
	must(t, e)
	if len(records) != 137 {
		t.Fatal("page-truncated CSV", len(records))
	}
	for _, r := range records[1:] {
		if r[0] == "QS024" {
			t.Fatal("excluded row exported")
		}
	}
	if f.scalar("SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.bu_history") != before {
		t.Fatal("export wrote organization history")
	}
	empty := selectionBody(list, []string{}, false)
	b.call("POST", "/api/v1/business-units/selection/resolve", empty, "", 200)
	selectionDownload(t, b, empty, 400)
	bad := selectionBody(list, []string{foundation.NewID()}, false)
	b.call("POST", "/api/v1/business-units/selection/resolve", bad, "", 400)
	bad = selectionBody(list, []string{ids[0], ids[0]}, false)
	b.call("POST", "/api/v1/business-units/selection/resolve", bad, "", 400)
	rlist := reader.call("GET", path, nil, "", 200)
	rb := selectionBody(rlist, []string{}, true)
	v = reader.call("POST", "/api/v1/business-units/selection/resolve", rb, "", 200)
	if v["selectedCount"] != float64(137) || v["canExport"] != false {
		t.Fatal("read/export mixed", v)
	}
	selectionDownload(t, reader, rb, 403)
	reader.call("POST", "/api/v1/business-units/selection/resolve", body, "", 409)
	for table, value := range businessSnapshot(t, f) {
		if value != history[table] {
			t.Fatal("selection/export changed organization history", table)
		}
	}
	// Object export denial is atomic even when all other selected objects can export.
	f.exec("DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='bu.export'", b.subject)
	f.exec("INSERT INTO access.subject_grant SELECT instance_id,gen_random_uuid(),principal_id,'bu',$2,'bu.export','{}'::text[],'{}'::text[] FROM access.principal WHERE principal_id=$1", b.subject, ids[0])
	v = b.call("POST", "/api/v1/business-units/selection/resolve", body, "", 200)
	if v["canExport"] != false {
		t.Fatal(v)
	}
	selectionDownload(t, b, body, 403)
	// Replace a member through a supported name correction and the real name
	// filter. Do not disable the immutable-identity constraint: source-pinned
	// caches legitimately retain an object's immutable code.
	filtered := path + "&filterField=name&filterOperator=equals&filterValue=" + url.QueryEscape("未来业务单元")
	filteredBody := func(list map[string]any) map[string]any {
		body := selectionBody(list, []string{}, true)
		query := body["query"].(map[string]any)
		query["filterField"], query["filterOperator"], query["filterValue"] = "name", "equals", "未来业务单元"
		return body
	}
	membership := b.call("GET", filtered, nil, "", 200)
	oldMembership := filteredBody(membership)
	detail := b.call("GET", "/api/v1/business-units/"+ids[0], nil, "", 200)
	record := detail["selectedRecord"].(map[string]any)
	b.call("POST", "/api/v1/business-units/"+ids[0]+"/records/"+record["recordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": detail["headSourceVersion"], "expectedRecordRevision": record["revision"], "changes": map[string]any{"name": "已移出集合"}}, foundation.NewID(), 200)
	b.call("POST", "/api/v1/business-units", createBody("QSREPLACE", "2026-01-01"), foundation.NewID(), 201)
	changed := b.call("GET", filtered, nil, "", 200)
	if changed["count"] != membership["count"] || changed["selectionVersion"] == membership["selectionVersion"] || changed["queryKey"] != membership["queryKey"] {
		t.Fatal("same count replacement missed")
	}
	b.call("POST", "/api/v1/business-units/selection/resolve", oldMembership, "", 409)
	// Invalid protocol, null union arrays and oversized request fail without a CSV.
	for _, choice := range []any{map[string]any{"mode": "allMatching", "excludedIds": nil}, map[string]any{"mode": "allMatching", "excludedIds": []string{}, "ids": []string{}}, map[string]any{"mode": "other", "ids": []string{}}, map[string]any{"mode": "allMatching", "excludedIds": []string{}, "ids": nil}} {
		bad = filteredBody(changed)
		bad["selection"] = choice
		b.call("POST", "/api/v1/business-units/selection/resolve", bad, "", 400)
	}
	body = filteredBody(changed)
	body["query"].(map[string]any)["q"] = strings.Repeat("x", 66000)
	selectionDownload(t, b, body, 400)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", f.server.URL+path, nil)
	if _, e = b.client.Do(req); e == nil {
		t.Fatal("cancelled request completed")
	}
}
func TestQuerySelectionComplete2021Rows(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	start := time.Now()
	ids := make([]string, 0, 2021)
	for i := 0; i < 2021; i++ {
		created := b.call("POST", "/api/v1/business-units", createBody(fmt.Sprintf("QS%04d", i), "2026-01-01"), foundation.NewID(), 201)
		ids = append(ids, created["buId"].(string))
	}
	t.Logf("2021 API-created single-source fixtures: %s", time.Since(start))
	list := b.call("GET", "/api/v1/business-units?q=QS&page=102", nil, "", 200)
	body := selectionBody(list, []string{}, true)
	start = time.Now()
	v := b.call("POST", "/api/v1/business-units/selection/resolve", body, "", 200)
	if v["selectedCount"] != float64(2021) {
		t.Fatal(v)
	}
	t.Logf("full-set resolve 2021: %s", time.Since(start))
	// The same sparse explicit set exceeds 64 KiB and must keep its identity.
	explicit := selectionBody(list, ids, false)
	if checked := b.call("POST", "/api/v1/business-units/selection/resolve", explicit, "", 200); checked["selectedCount"] != float64(len(ids)) {
		t.Fatal("large explicit set changed", checked)
	}
	if csv := selectionDownload(t, b, explicit, 200); strings.Count(string(csv), "\r\n") != len(ids)+1 {
		t.Fatal("large explicit CSV incomplete")
	}
	excluded := selectionBody(list, ids[:2020], true)
	if checked := b.call("POST", "/api/v1/business-units/selection/resolve", excluded, "", 200); checked["selectedCount"] != float64(1) {
		t.Fatal("large exclusions changed", checked)
	}
	runtime.GC()
	var memoryBefore runtime.MemStats
	runtime.ReadMemStats(&memoryBefore)
	stopSample := make(chan struct{})
	peakSample := make(chan uint64, 1)
	go func() {
		ticker := time.NewTicker(2 * time.Millisecond)
		defer ticker.Stop()
		peak := memoryBefore.HeapAlloc
		for {
			select {
			case <-ticker.C:
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				if m.HeapAlloc > peak {
					peak = m.HeapAlloc
				}
			case <-stopSample:
				peakSample <- peak
				return
			}
		}
	}()
	start = time.Now()
	data := selectionDownload(t, b, body, 200)
	rows, e := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(data), "\xef\xbb\xbf"))).ReadAll()
	must(t, e)
	if len(rows) != 2022 {
		t.Fatal("truncated", len(rows))
	}
	t.Logf("CSV 2021 rows: %s, bytes=%d", time.Since(start), len(data))
	close(stopSample)
	peak := <-peakSample
	var memoryAfter runtime.MemStats
	runtime.ReadMemStats(&memoryAfter)
	t.Logf("Go integration process (API+HTTP client), sampled HeapAlloc peak=%d bytes, baseline=%d bytes, TotalAlloc delta=%d bytes; PostgreSQL memory excluded", peak, memoryBefore.HeapAlloc, memoryAfter.TotalAlloc-memoryBefore.TotalAlloc)
}

//go:build integration

package bootstrap_test

import (
	"encoding/csv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/foundation"
)

func TestCurrentMaintenanceFutureRecordsAndMidnightSelection(t *testing.T) {
	f := fresh(t)
	clock := t07Clock(f) // 2026-06-14 23:59:59 Shanghai.
	b := f.login("admin", f.password)
	create := func(code, date, name string) string {
		body := createBody(code, date)
		body["name"] = name
		return b.call("POST", "/api/v1/business-units", body, foundation.NewID(), 201)["buId"].(string)
	}
	current := create("QS_CURRENT", "2026-01-01", "今日甲")
	future := create("QS_FUTURE", "2026-06-15", "未来丙")
	stopped := create("QS_STOPPED", "2026-01-01", "停用丁")
	currentPath := "/api/v1/business-units/" + current
	added := b.call("POST", currentPath+"/records", addBody(detail(b, current), "2026-06-15", map[string]any{"name": "未来乙"}), foundation.NewID(), 201)
	for _, v := range []struct{ action, day, kind, reason string }{{"deactivate", "2026-06-01", "DEACTIVATE", "BUSINESS_CLOSURE"}, {"reactivate", "2026-07-01", "REACTIVATE", "BUSINESS_RESUMPTION"}} {
		d := detail(b, stopped)
		b.call("POST", "/api/v1/business-units/"+stopped+"/"+v.action, map[string]any{"anchorRecordId": lifecycleAnchor(d), "expectedSourceVersion": d["headSourceVersion"], "effectiveDate": v.day, "operationType": v.kind, "operationReason": v.reason}, foundation.NewID(), 201)
	}
	list := b.call("GET", "/api/v1/business-units?q=QS&sort=name", nil, "", 200)
	if list["count"] != float64(3) || list["businessAt"] != "2026-06-14" {
		t.Fatal("current-day collection", list)
	}
	want := map[string][3]string{current: {"今日甲", "active", "2026-01-01"}, future: {"未来丙", "not_effective", "2026-06-15"}, stopped: {"停用丁", "inactive", "2026-06-01"}}
	for _, raw := range list["items"].([]any) {
		row := raw.(map[string]any)
		w := want[row["id"].(string)]
		if row["name"] != w[0] || row["status"] != w[1] || row["effectiveDate"] != w[2] {
			t.Fatal("future replaced today or future-only hidden", row)
		}
	}
	for _, status := range []string{"active", "inactive", "not_effective"} {
		filtered := b.call("GET", "/api/v1/business-units?q=QS&status="+status, nil, "", 200)
		if filtered["count"] != float64(1) || filtered["items"].([]any)[0].(map[string]any)["status"] != status {
			t.Fatal("state filter", filtered)
		}
	}
	if d := detail(b, current); d["selectedRecord"].(map[string]any)["name"] != "今日甲" || d["selectedRecordIsFuture"] != false {
		t.Fatal("ordinary entry did not select today", d)
	}
	selected := b.call("GET", currentPath+"?selectedRecordId="+added["selectedRecordId"].(string), nil, "", 200)
	if selected["selectedRecord"].(map[string]any)["name"] != "未来乙" || selected["selectedRecordIsFuture"] != true {
		t.Fatal("explicit future selection", selected)
	}
	if d := detail(b, future); d["result"] != "M02_NO_FACT" || d["selectedRecordIsFuture"] != true || d["selectedRecord"].(map[string]any)["name"] != "未来丙" {
		t.Fatal("future first detail", d)
	}
	missing := b.call("GET", currentPath+"?selectedRecordId="+foundation.NewID(), nil, "", 200)
	if missing["selectionMissing"] != true || missing["selectedRecord"] != nil {
		t.Fatal("missing selection silently replaced", missing)
	}
	body := selectionBody(list, []string{}, true)
	b.call("POST", "/api/v1/business-units/selection/resolve", body, "", 200)
	rows, err := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(selectionDownload(t, b, body, 200)), "\xef\xbb\xbf"))).ReadAll()
	must(t, err)
	if len(rows) != 4 || !strings.Contains(strings.Join(rows[2], ","), "未生效") {
		t.Fatal("future CSV", rows)
	}
	for _, path := range []string{"/api/v1/business-units?businessAt=2026-07-01", currentPath + "?businessAt=2026-07-01"} {
		b.call("GET", path, nil, "", 400)
	}
	body["query"].(map[string]any)["businessAt"] = "2026-07-01"
	b.call("POST", "/api/v1/business-units/selection/resolve", body, "", 400)
	selectionDownload(t, b, body, 400)
	delete(body["query"].(map[string]any), "businessAt")
	at, _ := time.Parse(time.RFC3339, "2026-06-14T16:00:00Z")
	clock.Store(at.Unix())
	b.call("POST", "/api/v1/business-units/selection/resolve", body, "", 409)
	selectionDownload(t, b, body, 409)
	next := b.call("GET", "/api/v1/business-units?q=QS", nil, "", 200)
	if next["businessAt"] != "2026-06-15" || next["selectionVersion"] == list["selectionVersion"] || detail(b, current)["selectedRecord"].(map[string]any)["name"] != "未来乙" {
		t.Fatal("day advancement did not re-resolve", next)
	}
	selectionDownload(t, b, selectionBody(next, []string{}, true), 200)
	// The selected record's future cue must not restore unreadable date information.
	f.exec("UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'effectiveDate') WHERE principal_id=$1 AND action='bu.read'", b.subject)
	d := detail(b, stopped)
	last := d["timeline"].([]any)[2].(map[string]any)
	d = b.call("GET", "/api/v1/business-units/"+stopped+"?selectedRecordId="+last["recordId"].(string), nil, "", 200)
	if d["selectedRecordIsFuture"] != false || d["selectedRecord"].(map[string]any)["effectiveDate"] != nil {
		t.Fatal("future cue leaked date permission", d)
	}
}

func TestCurrentMaintenanceKeepsDateWhenRequestCrossesMidnight(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	saved := b.call("POST", "/api/v1/business-units", createBody("DAY_BOUNDARY", "2026-06-15"), foundation.NewID(), 201)
	at, err := time.Parse(time.RFC3339, "2026-06-14T15:59:59Z")
	must(t, err)
	var samples atomic.Int64
	restore := bootstrap.SetIntegrationClock(f.runtime, func() time.Time {
		if samples.Add(1) == 1 {
			return at
		}
		return at.Add(time.Second)
	})
	defer restore()
	list := b.call("GET", "/api/v1/business-units", nil, "", 200)
	if list["businessAt"] != "2026-06-14" || list["currentDate"] != "2026-06-14" || list["items"].([]any)[0].(map[string]any)["status"] != "not_effective" {
		t.Fatal("one list response mixed two business days", list)
	}
	samples.Store(0)
	d := detail(b, saved["buId"].(string))
	if d["businessAt"] != "2026-06-14" || d["currentDate"] != "2026-06-14" || d["result"] != "M02_NO_FACT" || d["selectedRecordIsFuture"] != true {
		t.Fatal("detail default and future cue used different days", d)
	}
	samples.Store(0)
	body := selectionBody(list, []string{}, true)
	body["query"].(map[string]any)["q"] = ""
	selectionDownload(t, b, body, 200)
}

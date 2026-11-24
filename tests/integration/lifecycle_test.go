//go:build integration

package integration_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"chnhcm.local/chnhcm/internal/foundation"
)

func lifeBody(d map[string]any, date, kind string, changes map[string]any) map[string]any {
	reason := "BUSINESS_CLOSURE"
	if kind == "REACTIVATE" {
		reason = "BUSINESS_RESUMPTION"
	}
	rows := d["timeline"].([]any)
	in := map[string]any{"anchorRecordId": rows[len(rows)-1].(map[string]any)["recordId"], "expectedSourceVersion": d["headSourceVersion"], "effectiveDate": date, "operationType": kind, "operationReason": reason}
	if changes != nil {
		in["changes"] = changes
	}
	return in
}
func lifeSave(b *browser, id, date, kind string, changes map[string]any) map[string]any {
	return b.call("POST", "/api/v1/business-units/"+id+"/"+strings.ToLower(kind), lifeBody(detail(b, id), date, kind, changes), foundation.NewID(), 201)
}
func TestLifecycleFullJourneySourcesAndExactInheritance(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	body := createBody("LIFECYCLE", "2026-01-01")
	body["name"] = "甲"
	body["nameEn"] = "Alpha"
	created := b.call("POST", "/api/v1/business-units", body, foundation.NewID(), 201)
	id := created["buId"].(string)
	path := "/api/v1/business-units/" + id
	stopped := lifeSave(b, id, "2026-04-01", "DEACTIVATE", nil)
	// The last effective record determines availability, including future stops.
	d := detail(b, id)
	b.call("POST", path+"/records/"+selected(d)["recordId"].(string)+"/correct", correctBody(d, selected(d), map[string]any{"name": "不能更名"}), foundation.NewID(), 400)
	preview := lifeBody(d, "2026-05-01", "REACTIVATE", nil)
	preview["action"] = "REACTIVATE"
	p := b.call("POST", path+"/preview", preview, "", 200)
	if p["outcome"] != "accepted" || p["defaults"] != nil || p["timeline"] != nil {
		t.Fatal(p)
	}
	b.call("POST", path+"/reactivate", lifeBody(d, "2026-05-01", "REACTIVATE", map[string]any{"name": "研发😀"}), foundation.NewID(), 400)
	resumed := lifeSave(b, id, "2026-05-01", "REACTIVATE", map[string]any{"name": "乙", "nameEn": "Beta"})
	lifeSave(b, id, "2026-08-01", "DEACTIVATE", nil)
	lifeSave(b, id, "2026-09-01", "REACTIVATE", map[string]any{"name": "丙", "nameEn": nil})
	b.call("POST", path+"/records", addBody(detail(b, id), "2026-11-01", map[string]any{"name": "丁"}), foundation.NewID(), 201)
	for _, v := range []struct{ date, state, name, en string }{{"2026-03-31", "active", "甲", "Alpha"}, {"2026-04-01", "inactive", "甲", "Alpha"}, {"2026-05-01", "active", "乙", "Beta"}, {"2026-08-01", "inactive", "乙", "Beta"}, {"2026-09-01", "active", "丙", ""}, {"2026-11-01", "active", "丁", ""}} {
		fact := b.getOnDay(v.date, path)["viewFact"].(map[string]any)
		if fact["name"] != v.name || fact["state"] != v.state || v.en != "" && fact["nameEn"] != v.en || v.en == "" && fact["nameEn"] != nil {
			t.Fatal(v, fact)
		}
	}
	d = detail(b, id)
	first := d["timeline"].([]any)[0].(map[string]any)
	b.call("POST", path+"/records/"+first["recordId"].(string)+"/correct", correctBody(d, first, map[string]any{"name": "甲2"}), foundation.NewID(), 200)
	for date, want := range map[string]string{"2026-04-01": "甲2", "2026-05-01": "乙", "2026-08-01": "乙", "2026-09-01": "丙"} {
		if b.getOnDay(date, path)["viewFact"].(map[string]any)["name"] != want {
			t.Fatal(date)
		}
	}
	old := b.getOnDay("2026-04-01", path+"?sourceVersion="+stopped["sourceVersion"].(string))
	if old["viewFact"].(map[string]any)["name"] != "甲" || old["viewFact"].(map[string]any)["endExclusive"] != nil || old["capabilities"].(map[string]any)["reactivate"] != false {
		t.Fatal("old stop source changed", old)
	}
	f.assertBUInheritance()

	if len(f.history().BUs) != 7 || f.eventCount() != 7 {
		t.Fatal("incomplete source/event history")
	}

	if f.recordCount("") != 6 {
		t.Fatal("correction created identity")
	}
	recovered := b.call("POST", "/api/v1/organization/submissions/"+resumed["submissionId"].(string)+"/resolve", nil, "", 200)
	if recovered["sourceVersion"] != resumed["sourceVersion"] || recovered["action"] != "REACTIVATE" || recovered["currentChanged"] != true {
		t.Fatal("restored wrong source", recovered)
	}
	logs := b.call("GET", path+"/changes", nil, "", 200)["items"].([]any)
	if len(logs) != 7 {
		t.Fatal(logs)
	}
}

func TestLifecycleTargetDateTodayFutureAndRejections(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	created := b.call("POST", "/api/v1/business-units", createBody("TARGET_DATE", "2026-01-01"), foundation.NewID(), 201)
	id := created["buId"].(string)
	path := "/api/v1/business-units/" + id
	today := b.call("GET", "/api/v1/organization/context", nil, "", 200)["currentDate"].(string)
	day, e := time.Parse("2006-01-02", today)
	must(t, e)
	stopDay, resumeDay := day.AddDate(0, 0, 2).Format("2006-01-02"), day.AddDate(0, 0, 3).Format("2006-01-02")
	stopped := lifeSave(b, id, stopDay, "DEACTIVATE", nil)
	d := b.call("GET", path+"?selectedRecordId="+stopped["selectedRecordId"].(string), nil, "", 200)
	if d["status"] != "active" || d["capabilities"].(map[string]any)["reactivate"] != true {
		t.Fatal("today incorrectly hides future resume", d)
	}
	input := lifeBody(d, resumeDay, "REACTIVATE", nil)
	key := foundation.NewID()
	saved := b.call("POST", path+"/reactivate", input, key, 201)
	if saved["currentStatus"] != "active" {
		t.Fatal("future affected today", saved)
	}
	if b.call("POST", path+"/reactivate", input, key, 201)["sourceVersion"] != saved["sourceVersion"] {
		t.Fatal("duplicate resume")
	}
	d = detail(b, id)
	before := businessSnapshot(t, f)
	for _, v := range []struct {
		kind, date string
		changes    map[string]any
		status     int
	}{
		{"REACTIVATE", today, nil, 409}, {"DEACTIVATE", stopDay, nil, 409}, {"REACTIVATE", "2025-12-31", nil, 409},
		{"DEACTIVATE", today, map[string]any{}, 400}, {"DEACTIVATE", today, map[string]any{"name": "新"}, 400},
		{"REACTIVATE", resumeDay, map[string]any{}, 400}, {"REACTIVATE", resumeDay, map[string]any{"name": "😀"}, 409},
	} {
		b.call("POST", path+"/"+strings.ToLower(v.kind), lifeBody(d, v.date, v.kind, v.changes), foundation.NewID(), v.status)
	}
	for table, old := range before {
		if table != "terminals" && businessSnapshot(t, f)[table] != old {
			t.Fatal("rejection changed", table)
		}
	}
	// A planned ordinary successor makes a single earlier stop invalid.
	other := b.call("POST", "/api/v1/business-units", createBody("DEPENDENCY", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	b.call("POST", "/api/v1/business-units/"+other+"/records", addBody(detail(b, other), "2026-10-01", map[string]any{"name": "后继"}), foundation.NewID(), 201)
	rejected := b.call("POST", "/api/v1/business-units/"+other+"/deactivate", lifeBody(detail(b, other), "2026-03-01", "DEACTIVATE", nil), foundation.NewID(), 409)
	if !strings.Contains(rejected["message"].(string), "末条记录") {
		t.Fatal("append position not identified", rejected)
	}
}

func TestLifecycleConcurrentSubmissionAndStrictInputs(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	id := b.call("POST", "/api/v1/business-units", createBody("RACE_LIFE", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	path := "/api/v1/business-units/" + id
	for index, kind := range []string{"DEACTIVATE", "REACTIVATE"} {
		date := "2026-04-01"
		if index == 1 {
			date = "2026-05-01"
		}
		in := lifeBody(detail(b, id), date, kind, nil)
		var wg sync.WaitGroup
		out := make(chan int, 2)
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				status, _, err := b.response("POST", path+"/"+strings.ToLower(kind), in, foundation.NewID())
				if err != nil {
					out <- 0
				} else {
					out <- status
				}
			}()
		}
		wg.Wait()
		close(out)
		counts := map[int]int{}
		for status := range out {
			counts[status]++
		}
		if counts[201] != 1 || counts[409] != 1 || f.scalar("SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.bu_history") != index+2 {
			t.Fatal(kind, counts)
		}
	}

	for field, value := range map[string]any{"state": "active", "name": "假", "changes": map[string]any{}, "operationType": "REACTIVATE", "operationReason": "NAME_ADJUSTMENT"} {
		bad := lifeBody(detail(b, id), "2026-08-01", "DEACTIVATE", nil)
		bad[field] = value
		b.call("POST", path+"/deactivate", bad, foundation.NewID(), 400)
	}
	for _, field := range []string{"operationType", "operationReason", "effectiveDate", "expectedSourceVersion"} {
		bad := lifeBody(detail(b, id), "2026-08-01", "REACTIVATE", nil)
		delete(bad, field)
		b.call("POST", path+"/reactivate", bad, foundation.NewID(), 400)
	}

	raw, _ := json.Marshal(lifeBody(detail(b, id), "2026-08-01", "REACTIVATE", nil))
	duplicate := strings.Replace(string(raw), `"effectiveDate":"2026-08-01"`, `"effectiveDate":"2026-08-01","effectiveDate":"2026-09-01"`, 1)
	req, e := http.NewRequest("POST", f.server.URL+path+"/reactivate", strings.NewReader(duplicate))
	must(t, e)
	req.Header.Set("Origin", f.server.URL)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", b.csrf)
	req.Header.Set("Idempotency-Key", foundation.NewID())
	response, e := b.client.Do(req)
	must(t, e)
	response.Body.Close()
	if response.StatusCode != 400 {
		t.Fatal("duplicate lifecycle JSON key accepted", response.StatusCode)
	}

}

//go:build integration

package bootstrap_test

import (
	app "chnhcm.local/chnhcm/internal/contexts/staffing/application"
	db "chnhcm.local/chnhcm/internal/contexts/staffing/infra/postgres"
	"chnhcm.local/chnhcm/internal/foundation"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"
)

// AC025/026: expected values are fixture facts, never reconstructed from the
// endpoint being verified. Exercise the real HTTP, grants, immutable sources and DB.
func TestStaffing308LogSourcesPagingAndFilters(t *testing.T) {
	h := staffingSetup307(t)
	store := h.f.runtime.Staffing.(app.Service).Store.(*db.Store)
	now := time.Date(2026, 1, 15, 0, 2, 3, 123456000, time.UTC)
	store.Clock = func() time.Time { return now }
	family := h.create("jobfamily", "REFERENCE308", "")
	for _, kind := range []string{"jobfamily", "job", "grade"} {
		t.Run(kind, func(t *testing.T) {
			v := h.create(kind, "LOG308", family["id"].(string))
			id := v["id"].(string)
			path := base307(kind) + "/" + id + "/changes"
			first := v["headSourceVersion"].(string)
			for i := 1; i <= 24; i++ {
				h.correct(kind, v, map[string]any{"name": fmt.Sprintf("日志名称%02d", i), "nameEn": fmt.Sprintf("Log %02d", i)}, nil, 200)
				v = h.refresh(kind, v)
			}
			now = time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
			log := h.b.call("GET", path, nil, "", 200)
			if log["total"] != float64(25) || len(log["items"].([]any)) != 20 {
				t.Fatalf("25 groups must page 20: %v", log)
			}
			upper := log["throughSourceVersion"].(string)
			latest := log["items"].([]any)[0].(map[string]any)
			if latest["recordedAt"] != "2026-01-15T00:02:03.123456Z" {
				t.Fatalf("raw timestamp lost: %v", latest["recordedAt"])
			}
			diffs := latest["items"].([]any)[0].(map[string]any)["diffs"].([]any)
			if len(diffs) != 2 {
				t.Fatalf("two fields must stay in one group: %v", diffs)
			}
			for _, entry := range diffs {
				diff := entry.(map[string]any)
				if diff["field"] == "name" && (diff["before"].(map[string]any)["text"] != "日志名称23" || diff["after"].(map[string]any)["text"] != "日志名称24") {
					t.Fatal("incorrect immutable before/after")
				}
			}
			next := h.b.call("GET", path+"?page=2&throughSourceVersion="+upper, nil, "", 200)
			if len(next["items"].([]any)) != 5 {
				t.Fatal("second page must have five operations")
			}
			h.correct(kind, v, map[string]any{"name": "上界之后"}, nil, 200)
			pinned := h.b.call("GET", path+"?throughSourceVersion="+upper, nil, "", 200)
			if pinned["total"] != float64(25) {
				t.Fatal("new source slipped into fixed traversal")
			}
			history := h.b.call("GET", path+"?contextSourceVersion="+first, nil, "", 200)
			if history["total"] != float64(1) || history["throughSourceVersion"] != first {
				t.Fatal("history context escaped")
			}
			h.b.call("GET", path+"?contextSourceVersion="+first+"&throughSourceVersion="+upper, nil, "", 400)
			for _, q := range []string{"unknown=x", "page=1&page=2", "pageSize=19", "throughSourceVersion=", "filterField=actor", "filterField=recordedAt&filterOperator=is&filterValue=0000-01-01", "filterField=effectiveDate&filterOperator=is&filterValue=2026-1-01", "filterField=action&filterOperator=isAnyOf&filterValue=CORRECT,CORRECT", "filterField=action&filterOperator=is&filterValue=REVISE_TIMELINE"} {
				h.b.call("GET", path+"?"+q, nil, "", 400)
			}
			for _, q := range []string{"filterField=action&filterOperator=is&filterValue=CORRECT", "filterField=recordedAt&filterOperator=is&filterValue=2026-01-15", "filterField=changedField&filterOperator=is&filterValue=nameEn"} {
				got := h.b.call("GET", path+"?throughSourceVersion="+upper+"&"+q, nil, "", 200)
				want := float64(24)
				if strings.Contains(q, "recordedAt") || strings.Contains(q, "nameEn") {
					want = 25
				}
				if got["total"] != want {
					t.Fatalf("filter %s total=%v want=%v", q, got["total"], want)
				}
			}
			literal := h.b.call("GET", path+"?filterField=actor&filterOperator=contains&filterValue="+url.QueryEscape("%_\\"), nil, "", 200)
			if literal["total"] != float64(0) {
				t.Fatal("actor text interpreted as wildcard")
			}
			now = time.Date(2026, 1, 15, 0, 2, 3, 123456000, time.UTC)
		})
	}
}
func TestStaffing308LogCurrentGrantsAndDeletedOriginal(t *testing.T) {
	h := staffingSetup307(t)
	f := h.create("jobfamily", "SOURCE308", "")
	for _, kind := range []string{"job", "jobfamily", "grade"} {
		t.Run(kind, func(t *testing.T) {
			v := h.create(kind, "PRIVATE308", f["id"].(string))
			id := v["id"].(string)
			path := base307(kind) + "/" + id
			h.add(kind, v, "ADD", "2020-01-01", map[string]any{"name": "待删除原件", "nameEn": "original"}, 200)
			v = h.refresh(kind, v)
			record := rec307(v)
			originalSummary := record["operationSummary"].(map[string]any)
			rid := record["recordId"].(string)
			h.b.call("POST", path+"/records/"+rid+"/delete", map[string]any{"expectedSourceVersion": v["headSourceVersion"], "expectedRecordRevision": record["revision"]}, foundation.NewID(), 200)
			log := h.b.call("GET", path+"/changes", nil, "", 200)
			group := log["items"].([]any)[0].(map[string]any)
			if group["action"] != "DELETE" {
				t.Fatal("delete action missing")
			}
			var ref map[string]any
			for _, v := range group["items"].([]any) {
				item := v.(map[string]any)
				if item["recordId"] == rid {
					if item["afterRef"] != nil {
						t.Fatal("deleted record has after ref")
					}
					ref = item["beforeRef"].(map[string]any)
				}
			}
			if ref == nil {
				t.Fatal("deleted original absent")
			}
			old := h.b.call("GET", path+"?sourceVersion="+ref["sourceVersion"].(string)+"&selectedRecordId="+rid, nil, "", 200)
			if rec307(old)["name"] != "待删除原件" || rec307(old)["operationSummary"].(map[string]any)["sourceVersion"] != originalSummary["sourceVersion"] {
				t.Fatal("deletion actor replaced original revision summary")
			}
			h.f.exec(`UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'name') WHERE instance_id=$1 AND principal_id=$2 AND action=$3`, h.f.instance, h.b.subject, kind+".history.read")
			hidden := h.b.call("GET", path+"/changes", nil, "", 200)
			data, _ := json.Marshal(hidden)
			if strings.Contains(string(data), "待删除原件") || strings.Contains(string(data), `"field":"name"`) {
				t.Fatal("revoked field leaked")
			}
			h.b.call("GET", path+"/changes?filterField=changedField&filterOperator=is&filterValue=name", nil, "", 403)
			h.f.exec(`DELETE FROM access.subject_grant WHERE instance_id=$1 AND principal_id=$2 AND action=$3`, h.f.instance, h.b.subject, kind+".history.read")
			current := h.b.call("GET", path, nil, "", 200)
			if rec307(current)["operationSummary"] == nil {
				t.Fatal("read-only summary needs no history grant")
			}
			h.b.call("GET", path+"/changes", nil, "", 403)
			h.b.call("GET", path+"?sourceVersion="+ref["sourceVersion"].(string), nil, "", 403)
		})
	}
}
func TestStaffing308InitialSingleSelection(t *testing.T) {
	h := staffingSetup307(t)
	family := h.create("jobfamily", "REF308", "")
	for _, kind := range []string{"jobfamily", "job", "grade"} {
		t.Run(kind, func(t *testing.T) {
			a := h.create(kind, "SEL308-A", family["id"].(string))
			h.create(kind, "SEL308-B", family["id"].(string))
			in := map[string]any{"draftId": foundation.NewID(), "query": map[string]any{"q": "SEL308"}, "initialIds": []string{a["id"].(string)}}
			selected := h.b.call("POST", base307(kind)+"/selection/start", in, "", 200)
			if selected["count"] != float64(1) || len(selected["members"].([]any)) != 2 {
				t.Fatal("initial row checkbox selected whole query")
			}
			h.b.call("POST", base307(kind)+"/selection/start", in, "", 200)
			in["draftId"] = foundation.NewID()
			in["initialIds"] = []string{foundation.NewID()}
			h.b.call("POST", base307(kind)+"/selection/start", in, "", 400)
		})
	}
}

func TestStaffing308ServerMaintenanceAvailability(t *testing.T) {
	h := staffingSetup307(t)
	f := h.create("jobfamily", "WINDOW-REF308", "")
	for _, kind := range []string{"jobfamily", "job", "grade"} {
		v := h.create(kind, "WINDOW308", f["id"].(string))
		first := rec307(v)["recordId"].(string)
		h.add(kind, v, "ADD", "2000-01-02", map[string]any{"name": "紧邻后项"}, 200)
		v = h.detail(kind, v["id"].(string), first)
		if v["additionContext"].(map[string]any)["available"] != false || v["pairInsertion"] != nil {
			t.Fatal("server offered a nonexistent insertion day")
		}
		if v["capabilities"].(map[string]any)["actions"].(map[string]any)["DELETE"] != false {
			t.Fatal("first-record deletion exposed as available")
		}
	}
}

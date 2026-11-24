//go:build integration

package bootstrap_test

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"chnhcm.local/chnhcm/internal/foundation"
)

// The accepted20-record scenario uses ordinary public writes. It imposes no
// product record limit and does not reduce enterprise size or source history.
func TestModel5Timeline20Maintenance(t *testing.T) {
	for _, kind := range []string{"bu", "department"} {
		t.Run(kind, func(t *testing.T) {
			f := fresh(t)
			var model string
			must(t, f.admin.QueryRow(f.ctx, "SELECT contract FROM organization.storage_contract").Scan(&model))
			if model != "m02-compact-5" {
				t.Fatal("selected product model", model)
			}
			b := f.login("admin", f.password)
			route, key, recovery := "business-units", "buId", "organization"
			created := b.call("POST", "/api/v1/business-units", createBody("TIMELINE20_BU", "2020-01-01"), foundation.NewID(), 201)
			if kind == "department" {
				route, key, recovery = "departments", "departmentId", "departments"
				body := departmentCreateBody("TIMELINE20_DEP", created["buId"].(string))
				body["effectiveDate"] = "2020-01-01"
				created = b.call("POST", "/api/v1/departments", body, foundation.NewID(), 201)
			}
			id := created[key].(string)
			path := "/api/v1/" + route + "/" + id
			head := created
			start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
			for i := 1; i < maintenanceTimelineRecords; i++ {
				head = b.call("POST", path+"/records", map[string]any{"expectedSourceVersion": head["sourceVersion"], "anchorRecordId": head["targetRecordId"], "effectiveDate": start.AddDate(0, 0, 2*i).Format("2006-01-02"), "operationType": "CHANGE", "operationReason": "NAME_ADJUSTMENT", "changes": map[string]any{"name": fmt.Sprintf("资料%d", i)}}, foundation.NewID(), 201)
			}
			if kind == "department" && freezeTimelineInput(t, f, id, kind, os.Getenv("CHNHCM_DEPARTMENT_FREEZE_INPUT")) {
				return
			}
			if kind == "bu" && freeze133LongInput(t, f, id) {
				return
			}
			original := head["sourceVersion"].(string)
			originalView := b.call("GET", path+"?sourceVersion="+original, nil, "", 200)
			rows := originalView["timeline"].([]any)
			if len(rows) != maintenanceTimelineRecords {
				t.Fatal("record count", len(rows))
			}
			sourceBytes := func(source string) []byte {
				var fact any
				if kind == "bu" {
					fact = f.buSource(source)
				} else {
					fact = f.departmentSource(source)
				}
				raw, e := json.Marshal(fact)
				must(t, e)
				return raw
			}
			originalFacts := string(sourceBytes(original))
			submissions := []string{}
			for _, position := range []int{0, maintenanceTimelineRecords / 2, maintenanceTimelineRecords - 1} {
				view := b.call("GET", path, nil, "", 200)
				before := view["timeline"].([]any)
				record := before[position].(map[string]any)
				recordID := record["recordId"].(string)
				name := fmt.Sprintf("更正第%d条", position+1)
				submission := foundation.NewID()
				saved := b.call("POST", path+"/records/"+recordID+"/correct", map[string]any{"expectedSourceVersion": head["sourceVersion"], "expectedRecordRevision": record["revision"], "changes": map[string]any{"name": name}}, submission, 200)
				if saved["outcome"] != "accepted" || saved["sourceVersion"] == head["sourceVersion"] {
					t.Fatal("correction not published", saved)
				}
				head = saved
				submissions = append(submissions, submission)
				exact := b.call("GET", path+"?sourceVersion="+saved["sourceVersion"].(string), nil, "", 200)
				after := exact["timeline"].([]any)
				if len(after) != maintenanceTimelineRecords || exact["readOnly"] != true {
					t.Fatal("exact timeline changed shape")
				}
				for i := range before {
					if i != position && !reflect.DeepEqual(before[i], after[i]) {
						t.Fatal("unmodified record changed", i)
					}
				}
				target := after[position].(map[string]any)
				if target["recordId"] != recordID || target["name"] != name || target["revision"] != saved["targetRevision"] {
					t.Fatal("wrong target correction")
				}
				if string(sourceBytes(original)) != originalFacts {
					t.Fatal("original complete source changed")
				}
				resolved := b.call("POST", "/api/v1/"+recovery+"/submissions/"+submission+"/resolve", nil, "", 200)
				if resolved["sourceVersion"] != saved["sourceVersion"] || resolved["outcome"] != "accepted" {
					t.Fatal("original receipt missing")
				}
			}
			beforeRestart := b.call("GET", path+"?sourceVersion="+original, nil, "", 200)
			restartTimelineReceipts(t, f, b, submissions, kind)
			if string(sourceBytes(original)) != originalFacts {
				t.Fatal("original complete source changed after restart")
			}
			if !reflect.DeepEqual(beforeRestart, b.call("GET", path+"?sourceVersion="+original, nil, "", 200)) {
				t.Fatal("original history changed after restart")
			}
			t.Logf("model5 %s:20 records created through public API; first/middle/last corrected;3 original receipts recovered after actual DB restart; unchanged records and full original history preserved", kind)
		})
	}
}

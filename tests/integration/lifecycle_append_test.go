//go:build integration

package integration_test

import (
	"testing"

	"chnhcm.local/chnhcm/internal/foundation"
)

func TestLifecycleAppendPositionDefaultsFutureStateAndHistory(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	id := b.call("POST", "/api/v1/business-units", createBody("APPEND_FUTURE", "2028-01-01"), foundation.NewID(), 201)["buId"].(string)
	path := "/api/v1/business-units/" + id
	d := detail(b, id)
	c := d["lifecycleContext"].(map[string]any)
	if c["defaultEffectiveDate"] != "2028-01-02" || c["minEffectiveDate"] != "2028-01-02" || c["maxEffectiveDate"] != "9999-12-31" || d["capabilities"].(map[string]any)["reactivate"] != false {
		t.Fatal(d)
	}
	first := selected(d)["recordId"]
	stopped := lifeSave(b, id, "2028-02-01", "DEACTIVATE", nil)
	// Selecting an older active record cannot relocate a single operation to the end.
	d = detail(b, id)
	if d["capabilities"].(map[string]any)["deactivate"] != false || d["capabilities"].(map[string]any)["reactivate"] != false || d["lifecycleContext"] != nil {
		t.Fatal("old selection offers lifecycle", d)
	}
	bad := lifeBody(d, "2028-03-01", "REACTIVATE", nil)
	bad["anchorRecordId"] = first
	before := businessSnapshot(t, f)
	key := foundation.NewID()
	b.call("POST", path+"/reactivate", bad, key, 409)
	b.call("POST", path+"/reactivate", bad, key, 409)
	bad["action"] = "REACTIVATE"
	b.call("POST", path+"/preview", bad, "", 409)
	for table, v := range before {
		if table != "terminals" && businessSnapshot(t, f)[table] != v {
			t.Fatal("rejection changed facts", table)
		}
	}
	d = b.call("GET", path+"?selectedRecordId="+stopped["selectedRecordId"].(string), nil, "", 200)
	c = d["lifecycleContext"].(map[string]any)
	if d["status"] != "not_effective" || d["capabilities"].(map[string]any)["reactivate"] != true || d["capabilities"].(map[string]any)["deactivate"] != false || c["defaultEffectiveDate"] != "2028-02-02" || c["defaults"].(map[string]any)["name"] == nil {
		t.Fatal("future tail must allow resume", d)
	}
	input := lifeBody(d, "2028-03-01", "REACTIVATE", map[string]any{"name": "恢复资料"})
	delete(input, "anchorRecordId")
	b.call("POST", path+"/reactivate", input, foundation.NewID(), 400)
	input = lifeBody(d, "2028-01-15", "REACTIVATE", nil)
	input["action"] = "REACTIVATE"
	if r := b.call("POST", path+"/preview", input, "", 409); r["field"] != "effectiveDate" {
		t.Fatal(r)
	}
	resumed := lifeSave(b, id, "2028-03-01", "REACTIVATE", map[string]any{"name": "恢复资料"})
	old := b.call("GET", path+"?sourceVersion="+stopped["sourceVersion"].(string), nil, "", 200)
	if old["lifecycleContext"] != nil || old["capabilities"].(map[string]any)["reactivate"] != false {
		t.Fatal(old)
	}
	recovered := b.call("POST", "/api/v1/organization/submissions/"+resumed["submissionId"].(string)+"/resolve", nil, "", 200)
	if recovered["sourceVersion"] != resumed["sourceVersion"] {
		t.Fatal(recovered)
	}
}

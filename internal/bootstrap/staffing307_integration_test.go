//go:build integration

package bootstrap_test

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"testing"
)

func TestStaffing307Foundation(t *testing.T) {
	f := freshWithEnterpriseStart(t, "1900-01-01")
	b := f.login("admin", f.password)
	before := businessSnapshot(t, f)
	for _, base := range []string{"/api/v1/job-families", "/api/v1/jobs"} {
		b.call("GET", base+"/context", nil, "", 200)
	}
	family := b.call("POST", "/api/v1/job-families", map[string]any{"code": " f-01 ", "name": "研发系列", "description": "第一行\r\n第二行", "operationType": "CREATE", "operationReason": "NEW_JOB_FAMILY"}, foundation.NewID(), 201)
	fid := family["objectId"].(string)
	body := map[string]any{"code": "j-01", "jobFunctionId": b.functionForJobs("1900-01-01"), "name": "开发工程师", "jobFamilyId": fid, "operationType": "CREATE", "operationReason": "NEW_JOB"}
	key := foundation.NewID()
	job := b.call("POST", "/api/v1/jobs", body, key, 201)
	jid := job["objectId"].(string)
	b.call("POST", "/api/v1/jobs", body, key, 201)
	recovered := b.call("POST", "/api/v1/jobs/submissions/"+key+"/resolve", map[string]any{}, "", 200)
	if recovered["objectId"] != jid {
		t.Fatal("wrong receipt")
	}
	detail := b.call("GET", "/api/v1/jobs/"+jid, nil, "", 200)
	r := detail["selectedRecord"].(map[string]any)
	if r["jobFamilyName"] != "研发系列" || r["effectiveDate"] != "1900-01-01" {
		t.Fatalf("wrong staffing projection: %v", r)
	}
	fdetail := b.call("GET", "/api/v1/job-families/"+fid, nil, "", 200)
	fr := fdetail["selectedRecord"].(map[string]any)
	b.call("POST", "/api/v1/job-families/"+fid+"/deactivate", map[string]any{"expectedSourceVersion": fdetail["headSourceVersion"], "anchorRecordId": fr["recordId"], "effectiveDate": "2030-01-01", "operationType": "DEACTIVATE", "operationReason": "FAMILY_CLOSURE"}, foundation.NewID(), 409)
	reader := f.login("reader", f.readPassword)
	reader.call("GET", "/api/v1/jobs/"+jid, nil, "", 200)
	reader.call("POST", "/api/v1/jobs", body, foundation.NewID(), 403)
	unknown := foundation.NewID()
	b.call("POST", "/api/v1/jobs/submissions/"+unknown+"/resolve", map[string]any{}, "", 200)
	out := b.call("POST", "/api/v1/jobs", body, unknown, 200)
	if out["outcome"] != "not_applied" {
		t.Fatal("closed id reopened")
	}
	after := businessSnapshot(t, f)
	for key, value := range before {
		if after[key] != value {
			t.Fatalf("M02 fact changed: %s", key)
		}
	}
	if f.scalar("SELECT count(*) FROM staffing.job_source WHERE instance_id=$1", f.instance) != 1 {
		t.Fatal("duplicate source")
	}
}

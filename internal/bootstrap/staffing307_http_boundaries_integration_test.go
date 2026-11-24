//go:build integration

package bootstrap_test

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestStaffing307HTTPAdmissionIdentityAndConcurrentReads(t *testing.T) {
	h := staffingSetup307(t)
	f, b := h.f, h.b
	key := foundation.NewID()
	body := map[string]any{"code": " shared-307 ", "name": "相同名称", "effectiveDate": "2000-01-01", "operationType": "CREATE", "operationReason": "NEW_JOB_FAMILY"}
	family := b.call("POST", "/api/v1/job-families", body, key, 201)
	fid := family["objectId"].(string)
	body["name"] = "变摘要"
	b.call("POST", "/api/v1/job-families", body, key, 409)
	body["name"] = "相同名称"
	b.call("POST", "/api/v1/job-families", body, foundation.NewID(), 409)
	body["operationReason"] = "NEW_JOB"
	body["jobFunctionId"] = b.functionForJobs("1900-01-01")
	body["jobFamilyId"] = fid
	job := b.call("POST", "/api/v1/jobs", body, key, 201)
	jid := job["objectId"].(string)
	if jid == fid {
		t.Fatal("type identities merged")
	}
	reader := f.login("reader", f.readPassword)
	own := reader.call("POST", "/api/v1/jobs/submissions/"+key+"/resolve", map[string]any{}, "", 200)
	if own["outcome"] != "not_applied" {
		t.Fatal("other actor receipt disclosed")
	}
	for _, k := range []string{"jobfamily", "job"} {
		id := fid
		if k == "job" {
			id = jid
		}
		base := base307(k)
		for _, path := range []string{base + "/bad-id", base + "/bad-id/changes", base + "/bad-id/navigation", base + "?page=-1", base + "?unknown=value"} {
			out := b.call("GET", path, nil, "", 400)
			if !strings.HasPrefix(out["code"].(string), "M03_") {
				t.Fatal("foreign error code")
			}
		}
		current := h.detail(k, id, "")
		before := staffingFacts307(t, f)
		b.call("POST", base+"/"+id+"/preview", map[string]any{"action": "DELETE", "recordId": rec307(current)["recordId"], "expectedSourceVersion": current["headSourceVersion"], "expectedRecordRevision": rec307(current)["revision"], "name": "伪造不相关字段"}, "", 400)
		if !reflect.DeepEqual(before, staffingFacts307(t, f)) {
			t.Fatal("invalid preview changed fact")
		}
	}
	b.call("GET", "/api/v1/jobs?jobFamilyId="+foundation.NewID(), nil, "", 403)
	// Twenty simultaneous requests from one actual authenticated session. Context,
	// navigation and reasons are independent browser requests, not different users.
	for _, client := range []*browser{b, reader} {
		routes := []string{"/api/v1/job-families/context", "/api/v1/job-families/operation-reasons?operationType=DEACTIVATE", "/api/v1/job-families/operation-reasons?operationType=REACTIVATE", "/api/v1/job-families/" + fid + "/navigation", "/api/v1/jobs/" + jid, "/api/v1/jobs", "/api/v1/jobs/" + jid + "/changes"}
		errors := make(chan error, 20)
		var wg sync.WaitGroup
		for w := 0; w < 20; w++ {
			wg.Add(1)
			go func(w int) {
				defer wg.Done()
				for j := 0; j < 5; j++ {
					status, _, e := client.response("GET", routes[(w+j)%len(routes)], nil, "")
					if e == nil && status != 200 {
						e = fmt.Errorf("concurrent read status %d", status)
					}
					if e != nil {
						errors <- e
						return
					}
				}
			}(w)
		}
		wg.Wait()
		close(errors)
		for e := range errors {
			must(t, e)
		}
	}
	// Required nulls, optional clears, non-ASCII profile and control text boundaries.
	body["code"] = "NULL307"
	body["name"] = nil
	b.call("POST", "/api/v1/jobs", body, foundation.NewID(), 400)
	v := h.detail("job", jid, "")
	h.correct("job", v, map[string]any{"nameEn": nil, "requirements": "说明\r\n后行"}, nil, 200)
	v = h.refresh("job", v)
	if rec307(v)["requirements"] != "说明\n后行" || rec307(v)["nameEn"] != "" {
		t.Fatal("optional text semantics changed")
	}
	h.correct("job", v, map[string]any{"name": nil}, nil, 400)
	h.correct("job", v, map[string]any{"jobFamilyId": nil}, nil, 400)
	h.correct("job", v, map[string]any{"requirements": "非法\t制表"}, nil, 400)
}

func TestStaffing307EnterpriseDefaultAndForeignIdentity(t *testing.T) {
	f := freshWithEnterpriseStart(t, "1984-01-01")
	b := f.login("admin", f.password)
	input := map[string]any{"code": "FLOOR-307", "name": "企业下界目录", "operationType": "CREATE", "operationReason": "NEW_JOB_FAMILY"}
	refused := b.call("POST", "/api/v1/job-families", input, foundation.NewID(), 409)
	if refused["field"] != "effectiveDate" {
		t.Fatal("date failure did not locate input")
	}
	input["effectiveDate"] = "1984-01-01"
	family := b.call("POST", "/api/v1/job-families", input, foundation.NewID(), 201)
	delete(input, "effectiveDate")
	input["operationReason"] = "NEW_JOB"
	input["jobFunctionId"] = b.functionForJobs("1984-01-01")
	input["jobFamilyId"] = family["objectId"]
	b.call("POST", "/api/v1/jobs", input, foundation.NewID(), 409)
	input["effectiveDate"] = "1984-01-01"
	b.call("POST", "/api/v1/jobs", input, foundation.NewID(), 201)
	foreign := staffingSetup307(t)
	foreign.b.call("POST", "/api/v1/jobs", input, foundation.NewID(), 409)
	foreign.b.call("GET", "/api/v1/jobs?jobFamilyId="+family["objectId"].(string), nil, "", 403)
	var action string
	must(t, f.admin.QueryRow(f.ctx, "SELECT event->>'action' FROM staffing.job_outbox LIMIT 1").Scan(&action))
	if action != "CREATE" {
		t.Fatal("outbox lacks committed action identity")
	}
}

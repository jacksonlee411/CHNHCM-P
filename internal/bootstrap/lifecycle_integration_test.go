//go:build integration

package bootstrap_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"chnhcm.local/chnhcm/internal/contexts/organization/application"
	"chnhcm.local/chnhcm/internal/foundation"
)

func TestLifecycleWholeWriteRollbackAndRecovery(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	svc := f.runtime.Organization.(application.Service)
	original := svc.Store
	for _, action := range []string{"DEACTIVATE", "REACTIVATE"} {
		for index, mode := range []string{"409", "retry-first", "resolve-first", "race"} {
			id := b.call("POST", "/api/v1/business-units", createBody(fmt.Sprintf("LIFE_%s_%d", action, index), "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
			path := "/api/v1/business-units/" + id
			date, reason := "2026-04-01", "BUSINESS_CLOSURE"
			if action == "REACTIVATE" {
				d := detail(b, id)
				b.call("POST", path+"/deactivate", map[string]any{"anchorRecordId": lifecycleAnchor(d), "expectedSourceVersion": d["headSourceVersion"], "effectiveDate": date, "operationType": "DEACTIVATE", "operationReason": reason}, foundation.NewID(), 201)
				date, reason = "2026-05-01", "BUSINESS_RESUMPTION"
			}
			input := map[string]any{"anchorRecordId": lifecycleAnchor(detail(b, id)), "expectedSourceVersion": detail(b, id)["headSourceVersion"], "effectiveDate": date, "operationType": action, "operationReason": reason}
			before := businessSnapshot(t, f)
			key := foundation.NewID()
			requestPath := path + "/" + strings.ToLower(action)
			failure := error(context.DeadlineExceeded)
			status := 503
			if mode == "409" {
				failure = foundation.Fail(409, "M02_INJECTED_CONFLICT", "受控写后冲突")
				status = 409
			}
			svc.Store = faultStore{original, failure}
			f.runtime.Organization = svc
			b.call("POST", requestPath, input, key, status)
			for table, value := range businessSnapshot(t, f) {
				if table != "terminals" && before[table] != value {
					t.Fatal(action, mode, "partial write", table)
				}
			}
			svc.Store = original
			f.runtime.Organization = svc
			if mode == "retry-first" {
				saved := b.call("POST", requestPath, input, key, 201)
				out := b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
				if out["sourceVersion"] != saved["sourceVersion"] {
					t.Fatal("wrong recovered source")
				}
			}
			if mode == "409" {
				out := b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
				if out["outcome"] != "rejected" {
					t.Fatal(out)
				}
				b.call("POST", requestPath, input, key, 400)
			}
			if mode == "resolve-first" {
				out := b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
				if out["outcome"] != "not_applied" {
					t.Fatal(out)
				}
				b.call("POST", requestPath, input, key, 409)
			}
			if mode == "race" {
				type answer struct {
					kind   string
					status int
					body   map[string]any
					err    error
				}
				answers := make(chan answer, 2)
				go func() {
					status, body, err := b.response("POST", requestPath, input, key)
					answers <- answer{"write", status, body, err}
				}()
				go func() {
					status, body, err := b.response("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "")
					answers <- answer{"resolve", status, body, err}
				}()
				got := map[string]answer{}
				for i := 0; i < 2; i++ {
					v := <-answers
					if v.err != nil {
						t.Fatal(v.err)
					}
					got[v.kind] = v
				}
				if got["resolve"].status != 200 || (got["write"].status != 201 && got["write"].status != 409) {
					t.Fatal(got)
				}
				if got["write"].status == 201 && got["resolve"].body["sourceVersion"] != got["write"].body["sourceVersion"] {
					t.Fatal("race restored wrong result", got)
				}
				if got["write"].status == 409 && got["resolve"].body["outcome"] != "not_applied" {
					t.Fatal("late lifecycle write not closed", got)
				}
				if got["write"].status == 409 {
					for table, value := range businessSnapshot(t, f) {
						if table != "terminals" && before[table] != value {
							t.Fatal("closed lifecycle request wrote facts", action, table)
						}
					}
				}
			}
			if f.scalar("SELECT (SELECT count(*) FROM organization.bu_history h CROSS JOIN LATERAL unnest(h.submission_refs) r WHERE h.ids[r]=$1::uuid)+(SELECT count(*) FROM organization.bu_terminal WHERE submission_id=$1)", key) != 1 {
				t.Fatal("non-unique terminal")
			}
		}
	}
}

func lifecycleAnchor(d map[string]any) any {
	rows := d["timeline"].([]any)
	return rows[len(rows)-1].(map[string]any)["recordId"]
}

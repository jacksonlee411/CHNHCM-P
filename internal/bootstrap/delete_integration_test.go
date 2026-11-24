//go:build integration

package bootstrap_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"chnhcm.local/chnhcm/internal/contexts/organization/application"
	orgdb "chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
)

func prepareDeleteTarget(b *browser, code, kind string) (string, map[string]any, map[string]any) {
	id := b.call("POST", "/api/v1/business-units", createBody(code, "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	path := "/api/v1/business-units/" + id
	if kind == "CHANGE" {
		b.call("POST", path+"/records", addBody(detail(b, id), "2026-06-01", map[string]any{"name": "删除目标"}), foundation.NewID(), 201)
	} else {
		b.call("POST", path+"/deactivate", map[string]any{"anchorRecordId": lifecycleAnchor(detail(b, id)), "expectedSourceVersion": detail(b, id)["headSourceVersion"], "effectiveDate": "2026-05-01", "operationType": "DEACTIVATE", "operationReason": "BUSINESS_CLOSURE"}, foundation.NewID(), 201)
		if kind == "REACTIVATE" {
			b.call("POST", path+"/reactivate", map[string]any{"anchorRecordId": lifecycleAnchor(detail(b, id)), "expectedSourceVersion": detail(b, id)["headSourceVersion"], "effectiveDate": "2026-06-01", "operationType": "REACTIVATE", "operationReason": "BUSINESS_RESUMPTION"}, foundation.NewID(), 201)
		}
	}
	d := detail(b, id)
	rows := d["timeline"].([]any)
	target := rows[len(rows)-1].(map[string]any)
	return path + "/records/" + target["recordId"].(string) + "/delete", map[string]any{"expectedSourceVersion": d["headSourceVersion"], "expectedRecordRevision": target["revision"]}, target
}
func TestDeleteWholeWriteRollbackAndRecoveryRaces(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	svc := f.runtime.Organization.(application.Service)
	original := svc.Store
	for _, kind := range []string{"CHANGE", "DEACTIVATE", "REACTIVATE"} {
		for i, mode := range []string{"409", "retry-first", "resolve-first", "race"} {
			endpoint, in, target := prepareDeleteTarget(b, fmt.Sprintf("DELETE_%s_%d", kind, i), kind)
			before := businessSnapshot(t, f)
			key := foundation.NewID()
			failure := error(context.DeadlineExceeded)
			status := 503
			if mode == "409" {
				failure = foundation.Fail(409, "M02_INJECTED_CONFLICT", "受控写后冲突")
				status = 409
			}
			svc.Store = faultStore{original, failure}
			f.runtime.Organization = svc
			b.call("POST", endpoint, in, key, status)
			for table, value := range businessSnapshot(t, f) {
				if table != "terminals" && before[table] != value {
					t.Fatal(kind, mode, "partial deletion", table)
				}
			}
			svc.Store = original
			f.runtime.Organization = svc
			if mode == "409" {
				out := b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
				if out["outcome"] != "rejected" {
					t.Fatal(out)
				}
				b.call("POST", endpoint, in, key, 400)
			}
			if mode == "retry-first" {
				out := b.call("POST", endpoint, in, key, 200)
				r := b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
				if r["sourceVersion"] != out["sourceVersion"] || r["targetRecordId"] != target["recordId"] || r["targetRevision"] == target["revision"] {
					t.Fatal("wrong direct deletion result", r)
				}
			}
			if mode == "resolve-first" {
				out := b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
				if out["outcome"] != "not_applied" {
					t.Fatal(out)
				}
				b.call("POST", endpoint, in, key, 409)
			}
			if mode == "race" {
				type answer struct {
					kind   string
					status int
					out    map[string]any
					err    error
				}
				answers := make(chan answer, 2)
				go func() { s, v, e := b.response("POST", endpoint, in, key); answers <- answer{"write", s, v, e} }()
				go func() {
					s, v, e := b.response("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "")
					answers <- answer{"resolve", s, v, e}
				}()
				got := map[string]answer{}
				for j := 0; j < 2; j++ {
					a := <-answers
					must(t, a.err)
					got[a.kind] = a
				}
				if got["resolve"].status != 200 || (got["write"].status != 200 && got["write"].status != 409) {
					t.Fatal(got)
				}
				if got["write"].status == 200 && got["resolve"].out["targetRevision"] != got["write"].out["targetRevision"] {
					t.Fatal("non-unique deletion race", got)
				}
				if got["write"].status == 409 && got["resolve"].out["outcome"] != "not_applied" {
					t.Fatal("late delete reopened", got)
				}
			}
			if f.scalar("SELECT (SELECT count(*) FROM organization.bu_history h CROSS JOIN LATERAL unnest(h.submission_refs) r WHERE h.ids[r]=$1::uuid)+(SELECT count(*) FROM organization.bu_terminal WHERE submission_id=$1)", key) != 1 {
				t.Fatal("non-unique terminal")
			}
		}
	}
}
func TestDeleteOriginalDayAcrossMidnight(t *testing.T) {
	for _, kind := range []string{"CHANGE", "DEACTIVATE", "REACTIVATE"} {
		t.Run(kind, func(t *testing.T) {
			f := fresh(t)
			b := f.login("admin", f.password)
			endpoint, in, _ := prepareDeleteTarget(b, "DELETE_MIDNIGHT", kind)
			svc := f.runtime.Organization.(application.Service)
			store := svc.Store.(*orgdb.Store)
			before, _ := time.Parse(time.RFC3339, "2026-06-14T15:59:59Z")
			store.Clock = func(context.Context, pgx.Tx) (time.Time, error) { return before, nil }
			restoreRecorded := f.recordedAt("2026-06-14T16:00:01Z")
			key := foundation.NewID()
			saved := b.call("POST", endpoint, in, key, 200)
			expected := "active"
			if kind == "REACTIVATE" {
				expected = "inactive"
			}
			if saved["evaluatedBusinessDate"] != "2026-06-14" || saved["currentStatus"] != expected {
				t.Fatal("original delete day/state changed", saved)
			}
			restoreRecorded()
			store.Clock = nil
			r := b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
			if r["evaluatedBusinessDate"] != "2026-06-14" || r["evaluatedAt"] != "2026-06-14T15:59:59Z" || r["currentStatus"] != expected {
				t.Fatal("recovery relabeled deletion", r)
			}
			recorded := f.buSource(saved["sourceVersion"].(string)).Operation.Source.RecordedAt.UTC().Format(time.RFC3339Nano)
			if recorded != "2026-06-14T16:00:01Z" {
				t.Fatal(recorded)
			}
		})
	}
}

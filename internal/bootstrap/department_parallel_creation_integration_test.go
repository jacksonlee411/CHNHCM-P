//go:build integration

package bootstrap_test

import (
	"context"
	"testing"
	"time"

	"chnhcm.local/chnhcm/internal/foundation"
)

func TestDepartmentIndependentCreationsShareRangeButBlockMaintenance(t *testing.T) {
	for _, maintained := range []string{"parent", "bu"} {
		t.Run(maintained, func(t *testing.T) {
			f := fresh(t)
			a, b, maintenance := f.login("admin", f.password), f.login("admin", f.password), f.login("admin", f.password)
			bu := a.call("POST", "/api/v1/business-units", createBody("PARALLEL_BU", "2026-01-01"), foundation.NewID(), 201)
			buID := bu["buId"].(string)
			parent := a.call("POST", "/api/v1/departments", departmentCreateBody("PARALLEL_PARENT", buID), foundation.NewID(), 201)
			blocker, err := f.admin.Begin(f.ctx)
			must(t, err)
			defer blocker.Rollback(context.Background())
			_, err = blocker.Exec(f.ctx, "SELECT pg_advisory_xact_lock(127006)")
			must(t, err)
			// Pause a real creation after it acquired all guards and validated the
			// candidate, but before its department and incoming edge become visible.
			f.exec(`CREATE FUNCTION public.pause_new_department() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.code='PARALLEL_HELD' THEN PERFORM pg_advisory_xact_lock(127006); END IF; RETURN NEW; END $$;
CREATE TRIGGER pause_new_department BEFORE INSERT ON organization.department FOR EACH ROW EXECUTE FUNCTION public.pause_new_department()`)
			body := departmentCreateBody("PARALLEL_HELD", buID)
			body["parentDepartmentId"] = parent["departmentId"]
			created := make(chan map[string]any, 1)
			go func() {
				defer close(created)
				created <- a.call("POST", "/api/v1/departments", body, foundation.NewID(), 201)
			}()
			departmentBudgetWait(t, f)
			// Same BU, independent new identity: must commit while the first
			// creation is paused, not wait for its range guard to be released.
			b.call("POST", "/api/v1/departments", departmentCreateBody("PARALLEL_OTHER", buID), foundation.NewID(), 201)
			path, original := "/api/v1/departments/"+parent["departmentId"].(string), parent
			if maintained == "bu" {
				path, original = "/api/v1/business-units/"+buID, bu
			}
			stopped := make(chan map[string]any, 1)
			go func() {
				defer close(stopped)
				stopped <- maintenance.call("POST", path+"/deactivate", map[string]any{"expectedSourceVersion": original["sourceVersion"], "anchorRecordId": original["targetRecordId"], "effectiveDate": "2026-06-01", "operationType": "DEACTIVATE", "operationReason": "ORG_RESTRUCTURING"}, foundation.NewID(), 409)
			}()
			deadline := time.Now().Add(time.Second)
			for f.scalar(`SELECT count(*) FROM pg_stat_activity WHERE usename='chnhcm_api' AND wait_event_type='Lock' AND cardinality(pg_blocking_pids(pid))>0`) < 2 {
				if time.Now().After(deadline) {
					t.Fatal("maintenance did not wait for the in-flight creation")
				}
				time.Sleep(5 * time.Millisecond)
			}
			must(t, blocker.Rollback(f.ctx))
			select {
			case result := <-created:
				if result == nil || result["outcome"] != "accepted" {
					t.Fatal("paused creation did not commit")
				}
			case <-time.After(5 * time.Second):
				t.Fatal("creation did not resume")
			}
			select {
			case result := <-stopped:
				if result == nil {
					t.Fatal("maintenance result missing")
				}
			case <-time.After(5 * time.Second):
				t.Fatal("maintenance did not recheck the committed child")
			}
			current := b.call("GET", path, nil, "", 200)
			if current["headSourceVersion"] != original["sourceVersion"] || len(current["timeline"].([]any)) != 1 {
				t.Fatal("maintenance invalidated a concurrently created reference")
			}
		})
	}
}

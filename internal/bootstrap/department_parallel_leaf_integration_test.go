//go:build integration

package bootstrap_test

import (
	"context"
	"testing"
	"time"

	"chnhcm.local/chnhcm/internal/foundation"
)

func TestDepartmentIndependentLeafWritesProtectNewIncomingEdge(t *testing.T) {
	f := fresh(t)
	a, b, incoming := f.login("admin", f.password), f.login("admin", f.password), f.login("admin", f.password)
	bu := a.call("POST", "/api/v1/business-units", createBody("LEAF_BU", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	held := a.call("POST", "/api/v1/departments", departmentCreateBody("LEAF_HELD", bu), foundation.NewID(), 201)
	other := a.call("POST", "/api/v1/departments", departmentCreateBody("LEAF_OTHER", bu), foundation.NewID(), 201)
	parent := a.call("POST", "/api/v1/departments", departmentCreateBody("LEAF_PARENT", bu), foundation.NewID(), 201)
	blocker, err := f.admin.Begin(f.ctx)
	must(t, err)
	defer blocker.Rollback(context.Background())
	_, err = blocker.Exec(f.ctx, "SELECT pg_advisory_xact_lock(127007)")
	must(t, err)
	// Pause a real existing-leaf write after candidate validation, while its
	// own heads are exclusive and the shared BU range guard remains held.
	f.exec(`CREATE FUNCTION public.pause_leaf_write() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.code='LEAF_HELD' THEN PERFORM pg_advisory_xact_lock(127007); END IF; RETURN NEW; END $$;
CREATE TRIGGER pause_leaf_write BEFORE UPDATE ON organization.department FOR EACH ROW EXECUTE FUNCTION public.pause_leaf_write()`)
	stopped := make(chan map[string]any, 1)
	go func() {
		defer close(stopped)
		stopped <- a.call("POST", "/api/v1/departments/"+held["departmentId"].(string)+"/deactivate", map[string]any{"expectedSourceVersion": held["sourceVersion"], "anchorRecordId": held["targetRecordId"], "effectiveDate": "2026-06-01", "operationType": "DEACTIVATE", "operationReason": "ORG_RESTRUCTURING"}, foundation.NewID(), 201)
	}()
	departmentBudgetWait(t, f)
	// A second existing leaf changes parent and commits before the paused write.
	b.call("POST", "/api/v1/departments/"+other["departmentId"].(string)+"/records/"+other["targetRecordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": other["sourceVersion"], "expectedRecordRevision": other["targetRevision"], "changes": map[string]any{"parentDepartmentId": parent["departmentId"]}}, foundation.NewID(), 200)
	child := departmentCreateBody("LEAF_NEW_CHILD", bu)
	child["parentDepartmentId"] = held["departmentId"]
	rejected := make(chan map[string]any, 1)
	go func() {
		defer close(rejected)
		rejected <- incoming.call("POST", "/api/v1/departments", child, foundation.NewID(), 409)
	}()
	deadline := time.Now().Add(time.Second)
	for f.scalar(`SELECT count(*) FROM pg_stat_activity WHERE usename='chnhcm_api' AND wait_event_type='Lock' AND cardinality(pg_blocking_pids(pid))>0`) < 2 {
		if time.Now().After(deadline) {
			t.Fatal("incoming edge did not wait for the leaf's exclusive heads")
		}
		time.Sleep(5 * time.Millisecond)
	}
	must(t, blocker.Rollback(f.ctx))
	select {
	case value := <-stopped:
		if value == nil || value["outcome"] != "accepted" {
			t.Fatal("independent leaf maintenance did not commit")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("leaf maintenance did not resume")
	}
	select {
	case value := <-rejected:
		if value == nil {
			t.Fatal("incoming edge response missing")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("incoming edge did not recheck the changed parent")
	}
	if f.scalar("SELECT count(*) FROM organization.department WHERE code='LEAF_NEW_CHILD'") != 0 {
		t.Fatal("invalid incoming reference committed")
	}
	current := b.call("GET", "/api/v1/departments/"+other["departmentId"].(string), nil, "", 200)
	if current["selectedRecord"].(map[string]any)["parentDepartment"].(map[string]any)["id"] != parent["departmentId"] {
		t.Fatal("independent leaf relation was not preserved")
	}
}

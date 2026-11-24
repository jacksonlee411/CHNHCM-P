//go:build integration

package integration_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"chnhcm.local/chnhcm/internal/foundation"
)

// Pause after publication and response preparation, at the real COMMIT. A
// reader must see the complete old version until every deferred check succeeds.
func TestDepartment133PublicationCommitAndLateFailure(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprintf("late_failure=%t", fail), func(t *testing.T) {
			s := newDepartmentContract(t)
			s.create("P", "上级", "BU_A", "", "2026-01-01", s.b, 201)
			s.create("C", "部门", "BU_A", "P", "2026-01-01", s.b, 201)
			id := s.aliases["C"]
			reader := s.f.login("admin", s.f.password)
			before := reader.departmentCurrent(id)
			original := s.f.departmentHead(id)
			physical := s.f.physicalOrganizationSnapshot()
			failure := ""
			if fail {
				failure = "RAISE EXCEPTION 'controlled late publication failure';"
			}
			s.f.exec(`CREATE FUNCTION public.publication133_barrier() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN PERFORM pg_advisory_xact_lock_shared(1330134::bigint); ` + failure + ` RETURN NEW; END $$`)
			s.f.exec(`CREATE CONSTRAINT TRIGGER publication133_barrier AFTER INSERT OR UPDATE ON organization.department_history DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION public.publication133_barrier()`)
			tx, err := s.f.admin.Begin(s.f.ctx)
			must(t, err)
			defer tx.Rollback(s.f.ctx)
			_, err = tx.Exec(s.f.ctx, `SELECT pg_advisory_xact_lock(1330134::bigint)`)
			must(t, err)
			key := foundation.NewID()
			input := pairInsertBody(before, selected(before), "2026-06-01", "2026-09-01")
			input["deactivationReason"] = "ORG_RESTRUCTURING"
			replies := make(chan depAnswer, 1)
			go func() {
				status, body, err := s.b.response("POST", "/api/v1/departments/"+id+"/lifecycle-pairs", input, key)
				replies <- depAnswer{status, body, err}
			}()
			until := time.Now().Add(time.Second)
			for s.f.scalar(`SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND usename='chnhcm_api' AND lower(query)='commit' AND wait_event='advisory'`) != 1 {
				if time.Now().After(until) {
					t.Fatal("writer did not reach the deferred commit barrier")
				}
				time.Sleep(5 * time.Millisecond)
			}
			old := reader.departmentCurrent(id)
			if old["headSourceVersion"] != before["headSourceVersion"] || !reflect.DeepEqual(old["timeline"], before["timeline"]) || !reflect.DeepEqual(old["path"], before["path"]) {
				t.Fatal("uncommitted publication became partially visible")
			}
			must(t, tx.Commit(s.f.ctx))
			reply := <-replies
			must(t, reply.err)
			s.f.exec(`DROP TRIGGER publication133_barrier ON organization.department_history`)
			if fail {
				if reply.status != 503 || reply.body["code"] != "M02_RESULT_PENDING" || !reflect.DeepEqual(physical, s.f.physicalOrganizationSnapshot()) {
					t.Fatal("late failure exposed a result or retained partial publication", reply.status, reply.body)
				}
				if s.b.call("POST", "/api/v1/departments/submissions/"+key+"/resolve", nil, "", 200)["outcome"] != "not_applied" {
					t.Fatal("late rollback lost its original submission identity")
				}
				return
			}
			if reply.status != 200 || reply.body["outcome"] != "accepted" {
				t.Fatal(reply.status, reply.body)
			}
			current := reader.departmentCurrent(id)
			if current["headSourceVersion"] != reply.body["sourceVersion"] || len(current["timeline"].([]any)) != 3 {
				t.Fatal("committed source and members disagree")
			}
			restored := s.b.call("POST", "/api/v1/departments/submissions/"+key+"/resolve", nil, "", 200)
			if restored["sourceVersion"] != reply.body["sourceVersion"] || !reflect.DeepEqual(restored["items"], reply.body["items"]) {
				t.Fatal("fresh recovery disagrees with the original item projection")
			}
			s.b.departmentCorrect(s.aliases["P"], s.records["P_FIRST"], map[string]any{"changes": map[string]any{"name": "上级现名"}}, 200)
			restored = s.b.call("POST", "/api/v1/departments/submissions/"+key+"/resolve", nil, "", 200)
			for _, value := range restored["items"].([]any) {
				item := value.(map[string]any)
				if record, ok := item["afterRecord"].(map[string]any); ok && record["parentDepartment"].(map[string]any)["name"] != "上级现名" {
					t.Fatal("independent recovery reused the previous request's reference label")
				}
			}
			frozen := reader.call("GET", "/api/v1/departments/"+id+"?sourceVersion="+before["headSourceVersion"].(string), nil, "", 200)
			if !reflect.DeepEqual(original, s.f.departmentSource(before["headSourceVersion"].(string))) || !reflect.DeepEqual(frozen["path"], before["path"]) {
				t.Fatal("original immutable source changed")
			}
			if !strings.Contains(reader.departmentCurrent(id)["path"].(map[string]any)["departmentPath"].(string), "上级现名") {
				t.Fatal("current path did not use the newly authorized parent name")
			}
		})
	}
}

//go:build integration

package integration_test

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/foundation"
)

func TestDepartmentV06DayChosenAfterLocks(t *testing.T) {
	s := newDepartmentContract(t)
	allDepartmentRights(s)
	s.create("C", "部门", "BU_A", "", "2026-01-01", s.b, 201)
	s.append("C", "C_MOVE", "2027-01-01", "CHANGE", map[string]any{"managementBuId": s.aliases["BU_B"]}, s.b, 201)
	// Only the previous day's BU is visible to the maintenance actor.
	mutateLifeGrants(s.f, s.actor.subject, `UPDATE access.subject_grant SET scope_kind='bu',scope_id=$2 WHERE principal_id=$1 AND action LIKE 'department.%'`, s.actor.subject, s.aliases["BU_A"])
	var clock atomic.Int64
	before, _ := time.Parse(time.RFC3339, "2026-12-31T15:59:59Z")
	after, _ := time.Parse(time.RFC3339, "2026-12-31T16:00:00Z")
	clock.Store(before.Unix())
	var samples atomic.Int64
	restore := bootstrap.SetIntegrationClock(s.f.runtime, func() time.Time { samples.Add(1); return time.Unix(clock.Load(), 0) })
	defer restore()
	detail, r := s.target("C", "C_FIRST")
	body := map[string]any{"expectedSourceVersion": detail["headSourceVersion"], "expectedRecordRevision": r["revision"], "changes": map[string]any{"name": "跨日提交"}}
	path := "/api/v1/departments/" + s.aliases["C"] + "/records/" + r["recordId"].(string) + "/correct"
	for _, actor := range []*browser{s.actor, s.b} {
		clock.Store(before.Unix())
		samples.Store(0)
		tx, e := s.f.admin.Begin(s.f.ctx)
		must(t, e)
		_, e = tx.Exec(s.f.ctx, `SELECT FROM organization.business_unit WHERE bu_id=$1 FOR UPDATE`, s.aliases["BU_A"])
		must(t, e)
		key := foundation.NewID()
		done := make(chan depAnswer, 1)
		go func() {
			status, result, e := actor.response("POST", path, body, key)
			done <- depAnswer{status, result, e}
		}()
		waitDepartmentLocks(t, s.f, 1)
		if samples.Load() != 0 {
			t.Fatal("execution day sampled before relevant locks")
		}
		clock.Store(after.Unix())
		must(t, tx.Rollback(s.f.ctx))
		reply := <-done
		must(t, reply.err)
		if samples.Load() != 1 {
			t.Fatal("write used multiple execution days", samples.Load())
		}
		if actor == s.actor {
			if reply.status != 404 {
				t.Fatal("previous-day BU authorization reused", reply)
			}
		} else if reply.status != 200 || reply.body["evaluatedBusinessDate"] != "2027-01-01" {
			t.Fatal("lock-wait day incorrect", reply)
		}
	}
	for _, path := range []string{"/api/v1/departments", "/api/v1/departments/" + s.aliases["C"]} {
		samples.Store(0)
		s.b.call("GET", path, nil, "", 200)
		if samples.Load() != 1 {
			t.Fatal("read sampled multiple days", path)
		}
	}
	s.actor.call("GET", "/api/v1/departments/"+s.aliases["C"]+"?selectedRecordId="+s.records["C_FIRST"], nil, "", 404)
}
func TestDepartmentV07V20GuardOrderBudgetAndReplan(t *testing.T) {
	s := newDepartmentContract(t)
	s.create("C", "部门", "BU_A", "", "2026-01-01", s.b, 201)
	// Independent literal namespace, not a call into the lock encoder under test.
	raw, e := json.Marshal([]string{"organization.department.admin-relations", s.f.instance, s.aliases["E"], s.aliases["BU_A"], "DEPARTMENT_ADMIN"})
	must(t, e)
	sum := sha256.Sum256(raw)
	key := int64(binary.BigEndian.Uint64(sum[:8]))
	tx, e := s.f.admin.Begin(s.f.ctx)
	must(t, e)
	defer tx.Rollback(context.Background())
	_, e = tx.Exec(s.f.ctx, "SELECT pg_advisory_xact_lock($1)", key)
	must(t, e)
	d, r := s.target("C", "C_FIRST")
	body := map[string]any{"expectedSourceVersion": d["headSourceVersion"], "expectedRecordRevision": r["revision"], "changes": map[string]any{"name": "预算后重试"}}
	path := "/api/v1/departments/" + s.aliases["C"] + "/records/" + r["recordId"].(string) + "/correct"
	id := foundation.NewID()
	start := time.Now()
	v := s.b.call("POST", path, body, id, 503)
	end := time.Now()
	elapsed, wall := end.Sub(start), end.Round(0).Sub(start.Round(0))
	t.Logf("guard request monotonic=%s wall=%s clockDelta=%s code=%v", elapsed, wall, wall-elapsed, v["code"])
	if v["outcome"] != nil || elapsed > 4*time.Second {
		t.Fatal("guard timeout forged terminal or renewed budget", elapsed, v)
	}
	if s.f.scalar("SELECT (SELECT count(*) FROM organization.department_history h CROSS JOIN LATERAL unnest(h.submission_refs) r WHERE h.ids[r]=$1::uuid)+(SELECT count(*) FROM organization.department_terminal WHERE submission_id=$1)", id) != 0 {
		t.Fatal("guard failure persisted terminal")
	}
	must(t, tx.Rollback(s.f.ctx))
	s.b.call("POST", path, body, id, 200)
	// Existing P moves A -> B while a new child waits with initially A-only
	// dependencies. The child must reread/replan to B and reject its old-A parent.
	s.create("P", "父部门", "BU_A", "", "2026-01-01", s.b, 201)
	lock, e := s.f.admin.Begin(s.f.ctx)
	must(t, e)
	defer lock.Rollback(context.Background())
	// A BU barrier releases two compatible shared locks together, so it cannot
	// establish which writer goes first. Queue the parent's exclusive head lock
	// before the child's shared dependency lock and verify that blocking chain.
	_, e = lock.Exec(s.f.ctx, `SELECT FROM organization.department WHERE department_id=$1 FOR UPDATE`, s.aliases["P"])
	must(t, e)
	var blockerPID int
	must(t, lock.QueryRow(s.f.ctx, `SELECT pg_backend_pid()`).Scan(&blockerPID))
	pd, pr := s.target("P", "P_FIRST")
	answers := []chan depAnswer{make(chan depAnswer, 1), make(chan depAnswer, 1)}
	go func() {
		status, v, e := s.b.response("POST", "/api/v1/departments/"+s.aliases["P"]+"/records/"+pr["recordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": pd["headSourceVersion"], "expectedRecordRevision": pr["revision"], "changes": map[string]any{"managementBuId": s.aliases["BU_B"]}}, foundation.NewID())
		answers[0] <- depAnswer{status, v, e}
	}()
	waitDepartmentLocks(t, s.f, 1)
	parentPID := s.f.scalar(`SELECT pid FROM pg_stat_activity WHERE usename='chnhcm_api' AND $1=ANY(pg_blocking_pids(pid))`, blockerPID)
	go func() {
		status, v, e := s.b.response("POST", "/api/v1/departments", departmentBody("NEW_CHILD", "新增子部门", s.aliases["BU_A"], "2026-01-01", s.aliases["P"]), foundation.NewID())
		answers[1] <- depAnswer{status, v, e}
	}()
	waitDepartmentLocks(t, s.f, 2)
	if s.f.scalar(`SELECT count(*) FROM pg_stat_activity WHERE usename='chnhcm_api' AND $1=ANY(pg_blocking_pids(pid))`, parentPID) != 1 {
		t.Fatal("child was not queued behind the parent writer")
	}
	must(t, lock.Rollback(s.f.ctx))
	a, z := <-answers[0], <-answers[1]
	must(t, a.err)
	must(t, z.err)
	if a.status != 200 || z.status != 409 {
		t.Fatal("dependency change not reread/replanned", a, z)
	}
}
func TestDepartmentV23AuthorizationGuardBothOrders(t *testing.T) {
	for _, order := range []string{"revoke_first", "write_first"} {
		t.Run(order, func(t *testing.T) {
			s := newDepartmentContract(t)
			allDepartmentRights(s)
			s.create("C", "部门", "BU_A", "", "2026-01-01", s.b, 201)
			d, r := s.target("C", "C_FIRST")
			body := map[string]any{"expectedSourceVersion": d["headSourceVersion"], "expectedRecordRevision": r["revision"], "changes": map[string]any{"name": "写入"}}
			path := "/api/v1/departments/" + s.aliases["C"] + "/records/" + r["recordId"].(string) + "/correct"
			done := make(chan depAnswer, 1)
			start := func() {
				go func() {
					status, v, e := s.actor.response("POST", path, body, foundation.NewID())
					done <- depAnswer{status, v, e}
				}()
			}
			tx, e := s.f.admin.Begin(s.f.ctx)
			must(t, e)
			defer tx.Rollback(context.Background())
			revoke := func() {
				_, e = tx.Exec(s.f.ctx, `SELECT FROM access.policy_guard WHERE principal_id=$1 FOR UPDATE`, s.actor.subject)
				must(t, e)
				_, e = tx.Exec(s.f.ctx, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='department.record.correct'`, s.actor.subject)
				must(t, e)
			}
			if order == "revoke_first" {
				revoke()
				start()
				waitDepartmentLocks(t, s.f, 1)
				must(t, tx.Commit(s.f.ctx))
				reply := <-done
				must(t, reply.err)
				if reply.status != 403 {
					t.Fatal("stale write grant", reply)
				}
			} else {
				blocker, e := s.f.admin.Begin(s.f.ctx)
				must(t, e)
				defer blocker.Rollback(context.Background())
				_, e = blocker.Exec(s.f.ctx, `SELECT FROM organization.business_unit WHERE bu_id=$1 FOR UPDATE`, s.aliases["BU_A"])
				must(t, e)
				start()
				waitDepartmentLocks(t, s.f, 1)
				revoked := make(chan error, 1)
				go func() {
					_, e := tx.Exec(s.f.ctx, `SELECT FROM access.policy_guard WHERE principal_id=$1 FOR UPDATE`, s.actor.subject)
					if e == nil {
						_, e = tx.Exec(s.f.ctx, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='department.record.correct'`, s.actor.subject)
					}
					if e == nil {
						e = tx.Commit(s.f.ctx)
					}
					revoked <- e
				}()
				until := time.Now().Add(time.Second)
				for s.f.scalar(`SELECT count(*) FROM pg_stat_activity WHERE usename='postgres' AND wait_event_type='Lock'`) == 0 && time.Now().Before(until) {
					time.Sleep(10 * time.Millisecond)
				}
				if s.f.scalar(`SELECT count(*) FROM pg_stat_activity WHERE usename='postgres' AND wait_event_type='Lock'`) == 0 {
					t.Fatal("revocation did not wait for shared policy guard")
				}
				must(t, blocker.Rollback(s.f.ctx))
				reply := <-done
				must(t, reply.err)
				must(t, <-revoked)
				if reply.status != 200 {
					t.Fatal("authorized in-flight write failed", reply)
				}
				s.edit("C", "C_FIRST", "correct", map[string]any{"changes": map[string]any{"name": "第二次"}}, s.actor, 403)
			}
		})
	}
}

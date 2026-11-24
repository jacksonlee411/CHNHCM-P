//go:build integration

package integration_test

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"
)

type depAnswer struct {
	status int
	body   map[string]any
	err    error
}

func waitDepartmentLocks(t *testing.T, f *fixture, count int) {
	t.Helper()
	until := time.Now().Add(5 * time.Second)
	for time.Now().Before(until) {
		if f.scalar(`SELECT count(*) FROM pg_stat_activity WHERE usename='chnhcm_api' AND wait_event_type='Lock' AND cardinality(pg_blocking_pids(pid))>0`) >= count {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("missing independent connection barrier")
}
func TestDepartmentD14EmptySetBothLockOrders(t *testing.T) {
	for _, first := range []string{"department_guard_first", "bu_guard_first"} {
		t.Run(first, func(t *testing.T) {
			s := newDepartmentContract(t)
			bu := s.b.call("GET", "/api/v1/business-units/"+s.aliases["BU_A"], nil, "", 200)
			record := bu["timeline"].([]any)[0].(map[string]any)
			create := departmentBody("C", "组织发展部", s.aliases["BU_A"], "2026-01-01", nil)
			stop := map[string]any{"expectedSourceVersion": bu["headSourceVersion"], "anchorRecordId": record["recordId"], "effectiveDate": "2026-06-01", "operationType": "DEACTIVATE", "operationReason": "ORG_RESTRUCTURING"}
			tx, e := s.f.admin.Begin(s.f.ctx)
			must(t, e)
			defer tx.Rollback(s.f.ctx)
			_, e = tx.Exec(s.f.ctx, `SELECT bu_id FROM organization.business_unit WHERE instance_id=$1 AND bu_id=$2 FOR UPDATE`, s.f.instance, s.aliases["BU_A"])
			must(t, e)
			replies := []chan depAnswer{make(chan depAnswer, 1), make(chan depAnswer, 1)}
			start := func(i int) {
				go func() {
					path, body := "/api/v1/departments", create
					if i == 1 {
						path = "/api/v1/business-units/" + s.aliases["BU_A"] + "/deactivate"
						body = stop
					}
					status, v, e := s.b.response("POST", path, body, foundation.NewID())
					replies[i] <- depAnswer{status, v, e}
				}()
			}
			order := []int{0, 1}
			if first == "bu_guard_first" {
				order = []int{1, 0}
			}
			start(order[0])
			waitDepartmentLocks(t, s.f, 1)
			start(order[1])
			waitDepartmentLocks(t, s.f, 2)
			must(t, tx.Rollback(s.f.ctx))
			a, z := <-replies[order[0]], <-replies[order[1]]
			must(t, a.err)
			must(t, z.err)
			if a.status != 201 || z.status != 409 {
				t.Fatalf("incompatible commits %d/%d: %v %v", a.status, z.status, a.body, z.body)
			}
			for _, department := range s.f.history().Departments {
				if department.Operation.Source.ID != s.f.departmentHead(department.Object).Operation.Source.ID {
					continue
				}
				for _, r := range department.State.Timeline.Records {
					if r.State != "active" {
						continue
					}
					for _, bu := range s.f.buHead(r.Data.BU).State.Timeline.Records {
						if bu.State == "inactive" && (bu.End == "" || r.Date < bu.End) && (r.End == "" || bu.Date < r.End) {
							t.Fatal("committed invalid coverage")
						}
					}
				}
			}
		})
	}
}
func TestDepartmentV20OppositeParents(t *testing.T) {
	s := newDepartmentContract(t)
	s.create("A", "甲", "BU_A", "", "2026-01-01", s.b, 201)
	s.create("B", "乙", "BU_A", "", "2026-01-01", s.b, 201)
	type request struct {
		path string
		body map[string]any
	}
	requests := []request{}
	for _, pair := range [][2]string{{"A", "B"}, {"B", "A"}} {
		detail, r := s.target(pair[0], pair[0]+"_FIRST")
		requests = append(requests, request{
			"/api/v1/departments/" + s.aliases[pair[0]] + "/records/" + r["recordId"].(string) + "/correct",
			map[string]any{"expectedSourceVersion": detail["headSourceVersion"], "expectedRecordRevision": r["revision"], "changes": map[string]any{"parentDepartmentId": s.aliases[pair[1]]}},
		})
	}
	raw, e := json.Marshal([]string{"organization.department.admin-relations", s.f.instance, s.aliases["E"], s.aliases["BU_A"], "DEPARTMENT_ADMIN"})
	must(t, e)
	hash := sha256.Sum256(raw)
	key := int64(binary.BigEndian.Uint64(hash[:8]))
	tx, e := s.f.admin.Begin(s.f.ctx)
	must(t, e)
	defer tx.Rollback(s.f.ctx)
	_, e = tx.Exec(s.f.ctx, "SELECT pg_advisory_xact_lock($1)", key)
	must(t, e)
	answers := make(chan depAnswer, 2)
	clients := []*browser{s.b, s.f.login("admin", s.f.password)}
	for index, request := range requests {
		client := clients[index]
		go func() {
			status, v, e := client.response("POST", request.path, request.body, foundation.NewID())
			answers <- depAnswer{status, v, e}
		}()
	}
	waitDepartmentLocks(t, s.f, 2)
	must(t, tx.Rollback(s.f.ctx))
	a, z := <-answers, <-answers
	must(t, a.err)
	must(t, z.err)
	if a.status+z.status != 609 || (a.status != 200 && z.status != 200) {
		t.Fatalf("cycle writes %d/%d", a.status, z.status)
	}
	loser := a
	if loser.status == 200 {
		loser = z
	}
	if loser.body["code"] != "M02_DEPARTMENT_HIERARCHY_CYCLE" {
		t.Fatalf("unrelated source was treated as stale: %v", loser.body)
	}
}
func TestDepartmentD39AllPersistenceFailurePoints(t *testing.T) {
	s := newDepartmentContract(t)
	s.create("C", "组织发展部", "BU_A", "", "2026-01-01", s.b, 201)
	s.f.exec(`CREATE FUNCTION public.department_test_fail() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'controlled department write failure'; END $$`)
	tables := []string{"department_dictionary", "department_shape", "department_history", "department", "department_hierarchy_relation"}
	for _, table := range tables {
		t.Run(table, func(t *testing.T) {
			s.f.t = t
			s.f.exec("CREATE TRIGGER department_test_fail BEFORE INSERT OR UPDATE ON organization." + table + " FOR EACH ROW EXECUTE FUNCTION public.department_test_fail()")
			before := s.businessDigest()
			physicalBefore := s.f.physicalOrganizationSnapshot()
			terminals := s.f.scalar("SELECT (SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.department_history)+(SELECT count(*) FROM organization.department_terminal)")
			detail, r := s.target("C", "C_FIRST")
			key := foundation.NewID()
			response := s.b.call("POST", "/api/v1/departments/"+s.aliases["C"]+"/deactivate", map[string]any{"expectedSourceVersion": detail["headSourceVersion"], "anchorRecordId": r["recordId"], "effectiveDate": "2026-06-01", "operationType": "DEACTIVATE", "operationReason": "ORG_RESTRUCTURING"}, key, 503)
			if response["outcome"] == "rejected" {
				t.Fatal("technical failure forged business terminal")
			}
			s.assertNoBusinessChange(before)
			if !reflect.DeepEqual(physicalBefore, s.f.physicalOrganizationSnapshot()) {
				t.Fatal("failed write left physical business or encoding rows")
			}
			if s.f.scalar("SELECT (SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.department_history)+(SELECT count(*) FROM organization.department_terminal)") != terminals {
				t.Fatal("failure wrote terminal")
			}
			s.f.exec("DROP TRIGGER department_test_fail ON organization." + table)
			closed := s.b.call("POST", "/api/v1/departments/submissions/"+key+"/resolve", nil, "", 200)
			if closed["outcome"] != "not_applied" {
				t.Fatal("rollback not recoverable")
			}
		})
	}
	s.f.t = t
	s.append("C", "C_STOP", "2026-06-01", "DEACTIVATE", nil, s.b, 201)
	t.Log(fmt.Sprintf("%d real persistence failure points: zero partial facts and reacquirable locks", len(tables)))
}

func TestDepartmentV15CreationAndHeadFailurePoints(t *testing.T) {
	s := newDepartmentContract(t)
	s.f.exec(`CREATE FUNCTION public.department_create_fail() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'controlled creation/head write failure'; END $$`)
	for _, table := range []string{"department", "department_hierarchy_relation", "department_history"} {
		t.Run("insert_"+table, func(t *testing.T) {
			s.f.t = t
			before := s.businessDigest()
			s.f.exec("CREATE TRIGGER creation_failure BEFORE INSERT ON organization." + table + " FOR EACH ROW EXECUTE FUNCTION public.department_create_fail()")
			key := foundation.NewID()
			s.b.call("POST", "/api/v1/departments", departmentBody("FAIL", "部门", s.aliases["BU_A"], "2026-01-01", nil), key, 503)
			s.f.exec("DROP TRIGGER creation_failure ON organization." + table)
			s.assertNoBusinessChange(before)
			if s.b.call("POST", "/api/v1/departments/submissions/"+key+"/resolve", nil, "", 200)["outcome"] != "not_applied" {
				t.Fatal("partial creation")
			}
		})
	}
	s.f.t = t
	s.create("C", "部门", "BU_A", "", "2026-01-01", s.b, 201)
	for _, table := range []string{"department", "department_hierarchy_relation"} {
		t.Run("head_"+table, func(t *testing.T) {
			s.f.t = t
			before := s.businessDigest()
			s.f.exec("CREATE TRIGGER head_failure BEFORE UPDATE ON organization." + table + " FOR EACH ROW EXECUTE FUNCTION public.department_create_fail()")
			s.append("C", "STOP", "2026-06-01", "DEACTIVATE", nil, s.b, 503)
			s.f.exec("DROP TRIGGER head_failure ON organization." + table)
			s.assertNoBusinessChange(before)
		})
	}
	s.f.t = t
}

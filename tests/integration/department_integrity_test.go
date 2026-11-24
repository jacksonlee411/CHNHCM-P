//go:build integration

package integration_test

import (
	"chnhcm.local/chnhcm/internal/bootstrap"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/compact"
	"github.com/jackc/pgx/v5/pgconn"
)

func expectDepartmentSQLState(t *testing.T, e error, code string) {
	t.Helper()
	var p *pgconn.PgError
	if !errors.As(e, &p) || p.Code != code {
		t.Fatalf("want SQLSTATE %s, got %v", code, e)
	}
}
func TestDepartmentV01StorageConstraintsAndSeals(t *testing.T) {
	s := newDepartmentContract(t)
	first := s.create("C", "部门甲", "BU_A", "", "2026-01-01", s.b, 201)
	other := s.create("OTHER", "部门乙", "BU_A", "", "2026-01-01", s.b, 201)
	s.append("C", "C_CHANGE", "2026-06-01", "CHANGE", map[string]any{"name": "部门新名"}, s.b, 201)
	s.append("C", "C_STOP", "2026-10-01", "DEACTIVATE", nil, s.b, 201)
	s.edit("C", "C_CHANGE", "correct", map[string]any{"changes": map[string]any{"name": "更新名称"}}, s.b, 200)
	original := s.f.departmentHead(s.aliases["C"])
	source := original.Operation.Source.ID
	foreign := t07Instance(t, s.f, "2020-01-01")
	foreignActor := foreign.login("admin", foreign.password)
	foreignBU := foreignActor.call("POST", "/api/v1/business-units", createBody("FOREIGN", "2026-01-01"), foundation.NewID(), 201)
	foreignDept := foreignActor.call("POST", "/api/v1/departments", departmentBody("FOREIGN", "其他企业部门", foreignBU["buId"].(string), "2026-01-01", nil), foundation.NewID(), 201)
	foreignEnterprise := foreignActor.departmentCurrent(foreignDept["departmentId"].(string))["enterprise"].(map[string]any)["id"].(string)
	if s.f.scalar(`SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='organization' AND c.relkind='r' AND c.relname LIKE 'department%'`) != 9 {
		t.Fatal("department physical inventory")
	}
	for _, table := range []string{"department_history", "department_shape", "department_dictionary"} {
		_, err := s.f.admin.Exec(s.f.ctx, "UPDATE organization."+table+" SET instance_id=instance_id")
		expectDepartmentSQLState(t, err, "23514")
	}
	for _, table := range []string{"department_history", "department_shape", "department_terminal"} {
		_, err := s.f.runtime.Pool.Exec(s.f.ctx, "DELETE FROM organization."+table)
		expectDepartmentSQLState(t, err, "42501")
	}
	for _, q := range []string{
		"UPDATE organization.department SET parents=ARRAY[gen_random_uuid()] WHERE department_id=$1",
		"UPDATE organization.department SET current_envelope=decode(repeat('00',32),'hex') WHERE department_id=$1",
		"UPDATE organization.department_hierarchy_relation SET head_source_id=head_source_id WHERE department_id=$1",
	} {
		_, err := s.f.admin.Exec(s.f.ctx, q, s.aliases["C"])
		expectDepartmentSQLState(t, err, "23514")
	}
	tx, err := s.f.admin.Begin(s.f.ctx)
	must(t, err)
	_, err = tx.Exec(s.f.ctx, "ALTER TABLE organization.department_history DISABLE TRIGGER protect_history")
	must(t, err)
	_, err = tx.Exec(s.f.ctx, "UPDATE organization.department_history SET enterprise_id=$2 WHERE object_id=$1", s.aliases["C"], foreignEnterprise)
	expectDepartmentSQLState(t, err, "23503")
	must(t, tx.Rollback(s.f.ctx))
	// Each case retains valid physical hashes. Typed closure must reject wrong
	// references and periods, independently of checksum failures.
	cases := []struct {
		name string
		edit func(*bootstrap.DepartmentHistoryFact)
	}{
		{"missing_terminal_close_time", func(v *bootstrap.DepartmentHistoryFact) { v.Operation.TerminalClosedAt = time.Time{} }},
		{"wrong_child", func(v *bootstrap.DepartmentHistoryFact) {
			v.State.Timeline.Records[0].Data.Member.DepartmentID = s.aliases["OTHER"]
		}},
		{"wrong_management_bu", func(v *bootstrap.DepartmentHistoryFact) { v.State.Timeline.Records[0].Data.BU = s.aliases["BU_B"] }},
		{"wrong_revision", func(v *bootstrap.DepartmentHistoryFact) {
			v.State.Timeline.Records[0].Revision = other["targetRevision"].(string)
		}},
		{"wrong_enterprise", func(v *bootstrap.DepartmentHistoryFact) { v.State.Timeline.Enterprise = foreignEnterprise }},
		{"wrong_inheritance", func(v *bootstrap.DepartmentHistoryFact) {
			v.State.Timeline.Records[2].InheritedRevision = other["targetRevision"].(string)
		}},
		{"wrong_predecessor", func(v *bootstrap.DepartmentHistoryFact) {
			for i := range v.Operation.History.Items {
				if v.Operation.History.Items[i].Before != nil {
					v.Operation.History.Items[i].Before.Record.Revision = other["targetRevision"].(string)
					break
				}
			}
		}},
		{"result_wrong_revision", func(v *bootstrap.DepartmentHistoryFact) {
			x := other["targetRevision"].(string)
			v.Operation.Terminal.Result.TargetRevision = &x
		}},
		{"result_invalid_outcome", func(v *bootstrap.DepartmentHistoryFact) { v.Operation.Terminal.Outcome = "no_change" }},
		{"path_wrong_leaf", func(v *bootstrap.DepartmentHistoryFact) {
			p := &v.Evidence.Segments[0]
			p.Nodes[len(p.Nodes)-1].DepartmentID = s.aliases["OTHER"]
		}},
		{"duplicate_facets", func(v *bootstrap.DepartmentHistoryFact) {
			v.Operation.Changes[0].Facets = []string{"CORRECT", "CORRECT"}
		}},
		{"wrong_period", func(v *bootstrap.DepartmentHistoryFact) { v.State.Timeline.Records[0].End = "2026-02-01" }},
		{"overlap", func(v *bootstrap.DepartmentHistoryFact) { v.State.Timeline.Records[0].End = "2026-07-01" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			targetSource, restore := source, original
			if tc.name == "wrong_enterprise" {
				// The formal query writer rejects this current-head identity before
				// it can persist. Keep independent read-time coverage on an old
				// source whose corruption does not replace the valid current head.
				err := bootstrap.MutateIntegrationDepartment(s.f.ctx, s.f.admin, s.f.runtime, s.aliases["C"], source, tc.edit)
				if !errors.Is(err, compact.ErrCorrupt) {
					t.Fatalf("wrong owner accepted by current projection writer: %v", err)
				}
				s.exact("C", source)
				targetSource = first["sourceVersion"].(string)
				restore = s.f.departmentSource(targetSource)
			}
			must(t, bootstrap.MutateIntegrationDepartment(s.f.ctx, s.f.admin, s.f.runtime, s.aliases["C"], targetSource, tc.edit))
			s.b.call("GET", "/api/v1/departments/"+s.aliases["C"]+"?sourceVersion="+targetSource, nil, "", 503)
			must(t, bootstrap.MutateIntegrationDepartment(s.f.ctx, s.f.admin, s.f.runtime, s.aliases["C"], targetSource, func(v *bootstrap.DepartmentHistoryFact) { *v = restore }))
		})
	}
	s.exact("C", source)
	t.Log("native ownership, immutable contents and source-bound projection; typed member, revision, result, path and period counterexamples")
}
func TestDepartmentV12CorruptEvidenceFailsClosed(t *testing.T) {
	s := newDepartmentContract(t)
	p := s.create("P", "上级", "BU_A", "", "2026-01-01", s.b, 201)
	c := s.create("C", "子部门", "BU_A", "P", "2026-01-01", s.b, 201)
	s.append("P", "P_CHANGE", "2026-06-01", "CHANGE", map[string]any{"name": "新上级"}, s.b, 201)
	s.edit("C", "C_FIRST", "correct", map[string]any{"changes": map[string]any{"shortName": "子"}}, s.b, 200)
	current := s.b.departmentCurrent(s.aliases["C"])
	path := current["path"].(map[string]any)
	if len(path["pathSegments"].([]any)) != 2 {
		t.Fatal("period path not segmented", path)
	}
	old := s.b.call("GET", "/api/v1/departments/"+s.aliases["C"]+"?sourceVersion="+c["sourceVersion"].(string), nil, "", 200)
	segment := old["path"].(map[string]any)["pathSegments"].([]any)[0].(map[string]any)
	if _, exists := segment["nodes"]; exists || !strings.HasSuffix(segment["path"].(string), "上级 / 子部门") {
		t.Fatal("historical text projection changed", segment)
	}
	// Exact sources are verified through the independent storage inspection port.
	nodes := s.f.departmentSource(c["sourceVersion"].(string)).Evidence.Segments[0].Nodes
	if nodes[1].Source != p["sourceVersion"] || nodes[2].Source != c["sourceVersion"] {
		t.Fatal("historical/self source refs not exact")
	}
	// Synthetic damaged evidence is isolated to this disposable fixture; prior
	// evidence was verified above. A missing edge must never produce a partial path.
	s.f.editDepartment(c["sourceVersion"].(string), func(v *bootstrap.DepartmentHistoryFact) { v.Evidence.Segments[0].Nodes[1].Edge.MemberID = "" })
	r := s.b.call("GET", "/api/v1/departments/"+s.aliases["C"]+"?sourceVersion="+c["sourceVersion"].(string), nil, "", 503)
	if r["code"] != "M02_DEPARTMENT_PATH_UNAVAILABLE" || r["path"] != nil {
		t.Fatal("damaged path escaped", r)
	}
}

func allDepartmentRights(s *depFixture) {
	mutateLifeGrants(s.f, s.actor.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1`, s.actor.subject)
	mutateLifeGrants(s.f, s.actor.subject, `INSERT INTO access.subject_grant SELECT instance_id,gen_random_uuid(),$1,scope_kind,scope_id,action,read_fields,write_fields FROM access.subject_grant WHERE principal_id=$2`, s.actor.subject, s.b.subject)
}
func TestDepartmentV05MinimumReferenceAndCompleteGrant(t *testing.T) {
	s := newDepartmentContract(t)
	s.create("P", "父", "BU_A", "", "2026-01-01", s.b, 201)
	s.create("C", "子", "BU_A", "P", "2026-01-01", s.b, 201)
	d, r := s.target("C", "C_FIRST")
	query := map[string]any{"context": map[string]any{"mode": "maintain", "departmentId": s.aliases["C"], "expectedSourceVersion": d["headSourceVersion"], "action": "CORRECT", "recordId": r["recordId"], "expectedRecordRevision": r["revision"]}, "pageSize": 20}
	refs := s.actor.call("POST", "/api/v1/departments/references/parents", query, "", 200)
	if len(refs["items"].([]any)) < 1 {
		t.Fatal("read/reference-only user cannot select references")
	}
	s.edit("C", "C_FIRST", "correct", map[string]any{"changes": map[string]any{"name": "无写权"}}, s.actor, 403)
	// Neither individual grant contains both fields. The evaluator must not union them.
	mutateLifeGrants(s.f, s.actor.subject, `INSERT INTO access.subject_grant(instance_id,grant_id,principal_id,scope_kind,scope_id,action,read_fields,write_fields) VALUES($1,gen_random_uuid(),$2,'enterprise',$3,'department.record.correct',ARRAY[]::text[],ARRAY['name']),($1,gen_random_uuid(),$2,'department',$4,'department.record.correct',ARRAY[]::text[],ARRAY['shortName'])`, s.f.instance, s.actor.subject, s.aliases["E"], s.aliases["C"])
	s.edit("C", "C_FIRST", "correct", map[string]any{"changes": map[string]any{"name": "一", "shortName": "二"}}, s.actor, 403)
	s.edit("C", "C_FIRST", "correct", map[string]any{"changes": map[string]any{"name": "单字段允许"}}, s.actor, 200)
	// Full scope with name-only fields cannot write a relationship or impersonate a BU.
	s.edit("C", "C_FIRST", "correct", map[string]any{"changes": map[string]any{"managementBuId": s.aliases["BU_A"]}}, s.actor, 403)
	_, e := s.f.admin.Exec(s.f.ctx, `UPDATE access.subject_grant SET write_fields=ARRAY['shortName'] WHERE principal_id=$1 AND action='bu.create'`, s.b.subject)
	expectDepartmentSQLState(t, e, "23514")
	mutateLifeGrants(s.f, s.actor.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='department.read'`, s.actor.subject)
	s.actor.call("GET", "/api/v1/departments/"+s.aliases["P"], nil, "", 404)
	s.actor.call("POST", "/api/v1/departments/references/parents", query, "", 403)
}
func TestDepartmentV14RecoveryAuthorizationAndV17Locality(t *testing.T) {
	s := newDepartmentContract(t)
	allDepartmentRights(s)
	first := s.create("C", "子部门", "BU_A", "", "2026-01-01", s.actor, 201)
	s.create("UNRELATED", "无关部门", "BU_A", "", "2026-01-01", s.b, 201)
	before := s.b.departmentCurrent(s.aliases["C"])
	s.edit("UNRELATED", "UNRELATED_FIRST", "correct", map[string]any{"changes": map[string]any{"name": "其他新名"}}, s.b, 200)
	after := s.b.departmentCurrent(s.aliases["C"])
	if before["headSourceVersion"] != after["headSourceVersion"] || !reflect.DeepEqual(before["path"], after["path"]) {
		t.Fatal("unrelated same-BU change invalidated target")
	}
	// A stale global-tree token is neither required nor accepted; original own-source saves.
	r := after["selectedRecord"].(map[string]any)
	body := map[string]any{"expectedSourceVersion": first["sourceVersion"], "expectedRecordRevision": r["revision"], "changes": map[string]any{"name": "新名称"}}
	key := foundation.NewID()
	path := "/api/v1/departments/" + s.aliases["C"] + "/records/" + r["recordId"].(string) + "/correct"
	accepted := s.actor.call("POST", path, body, key, 200)
	mutateLifeGrants(s.f, s.actor.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action NOT IN ('department.read','department.history.read','department.reference.read','bu.reference.read')`, s.actor.subject)
	replay := s.actor.call("POST", path, body, key, 200)
	if replay["sourceVersion"] != accepted["sourceVersion"] || replay["replayed"] != true {
		t.Fatal("write revocation reran command")
	}
	different := map[string]any{}
	for k, v := range body {
		different[k] = v
	}
	different["changes"] = map[string]any{"name": "异内容"}
	s.actor.call("POST", path, different, key, 409)
	closeKey := foundation.NewID()
	s.actor.call("POST", "/api/v1/departments/submissions/"+closeKey+"/resolve", nil, "", 200)
	s.actor.call("POST", path, body, closeKey, 409)
	mutateLifeGrants(s.f, s.actor.subject, `UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'name') WHERE principal_id=$1 AND action IN ('department.read','department.history.read')`, s.actor.subject)
	result := s.actor.call("POST", "/api/v1/departments/submissions/"+key+"/resolve", nil, "", 200)
	if result["sourceVersion"] != accepted["sourceVersion"] {
		t.Fatal("field trim rewrote result identity")
	}
	mutateLifeGrants(s.f, s.actor.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='department.history.read'`, s.actor.subject)
	s.actor.call("POST", "/api/v1/departments/submissions/"+key+"/resolve", nil, "", 403)
	mutateLifeGrants(s.f, s.actor.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='department.read'`, s.actor.subject)
	s.actor.call("POST", path, body, key, 404)
	// No BU alias or writable internal context is accepted.
	for _, extra := range []string{"buId", "operator", "path", "departmentTarget", "businessAt", "expectedTreeVersion"} {
		bad := map[string]any{}
		for k, v := range body {
			bad[k] = v
		}
		bad[extra] = "forged"
		s.b.call("POST", path, bad, foundation.NewID(), 400)
	}
	for _, tail := range []string{"?businessAt=2026-01-01", "?sourceVersion=" + s.aliases["BU_A"]} {
		status := 400
		if strings.HasPrefix(tail, "?sourceVersion") {
			status = 404
		}
		s.b.call("GET", "/api/v1/departments/"+s.aliases["C"]+tail, nil, "", status)
	}
}

func TestDepartmentV23DamagedDiagnosticsAfterAuthorization(t *testing.T) {
	s := newDepartmentContract(t)
	allDepartmentRights(s)
	s.create("C", "部门", "BU_A", "", "2026-01-01", s.actor, 201)
	d, r := s.target("C", "C_FIRST")
	key := foundation.NewID()
	body := map[string]any{"expectedSourceVersion": d["headSourceVersion"], "expectedRecordRevision": r["revision"], "changes": map[string]any{"name": ""}}
	path := "/api/v1/departments/" + s.aliases["C"] + "/records/" + r["recordId"].(string) + "/correct"
	s.actor.call("POST", path, body, key, 400)
	s.f.editDepartmentResult(s.actor.subject, key, func(v *bootstrap.IntegrationDepartmentTerminal) { v.Failure.FormatVersion = 999 })
	mutateLifeGrants(s.f, s.actor.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='department.read'`, s.actor.subject)
	s.actor.call("POST", path, body, key, 404)
	s.actor.call("POST", "/api/v1/departments/submissions/"+key+"/resolve", nil, "", 404)
	allDepartmentRights(s)
	v := s.actor.call("POST", "/api/v1/departments/submissions/"+key+"/resolve", nil, "", 503)
	if v["failureDetails"] != nil || v["outcome"] != nil {
		t.Fatal("damaged diagnostic emitted", v)
	}
}
func TestDepartmentV19ExactAdoptionAndIndependentSources(t *testing.T) {
	s := newDepartmentContract(t)
	s.create("OTHER", "无关", "BU_A", "", "2026-01-01", s.b, 201)
	first := s.create("C", "部门", "BU_A", "", "2026-01-01", s.b, 201)
	oldSource := first["sourceVersion"]
	original := s.f.departmentSource(oldSource.(string))
	oldRef := original.State.Timeline.Records[0].Data.Member
	otherOriginal := s.f.departmentHead(s.aliases["OTHER"])
	beforeRelations := 0
	for _, v := range s.f.history().Departments {
		if v.Operation.RelationEventID != "" {
			beforeRelations++
		}
	}
	move := s.append("C", "MOVE", "2026-07-01", "CHANGE", map[string]any{"managementBuId": s.aliases["BU_B"]}, s.b, 201)
	afterRelations := 0
	for _, v := range s.f.history().Departments {
		if v.Operation.RelationEventID != "" {
			afterRelations++
		}
	}
	if afterRelations != beforeRelations+1 || s.f.departmentSource(move["sourceVersion"].(string)).Operation.RelationEventID == "" {
		t.Fatal("cross-BU source/event not singular")
	}
	if !reflect.DeepEqual(otherOriginal, s.f.departmentHead(s.aliases["OTHER"])) {
		t.Fatal("copied unrelated members")
	}
	if !reflect.DeepEqual(oldRef, s.f.departmentSource(oldSource.(string)).State.Timeline.Records[0].Data.Member) {
		t.Fatal("old adopted ref rewritten")
	}

	current := s.current("C")
	source := current["headSourceVersion"]
	// Same-value addition is removed after exact relation binding. No history/event changes.
	before := s.businessDigest()
	d, r := s.target("C", "MOVE")
	result := s.b.call("POST", "/api/v1/departments/"+s.aliases["C"]+"/records", map[string]any{"expectedSourceVersion": d["headSourceVersion"], "anchorRecordId": r["recordId"], "effectiveDate": "2026-08-01", "operationType": "CHANGE", "operationReason": "NAME_ADJUSTMENT", "changes": map[string]any{"name": "部门"}}, foundation.NewID(), 200)
	if result["outcome"] != "no_change" {
		t.Fatal("redundant CHANGE retained")
	}
	s.assertNoBusinessChange(before)
	if s.current("C")["headSourceVersion"] != source {
		t.Fatal("no change advanced source")
	}
	s.append("C", "STOP", "2026-10-01", "DEACTIVATE", nil, s.b, 201)
	s.edit("C", "MOVE", "correct", map[string]any{"changes": map[string]any{"name": "新部门"}}, s.b, 200)
	rows := s.f.departmentHead(s.aliases["C"]).State.Timeline.Records
	found := 0
	for i, r := range rows {
		if r.State == "inactive" {
			if i == 0 || r.InheritedRevision != rows[i-1].Revision || !reflect.DeepEqual(r.Data, rows[i-1].Data) {
				t.Fatal("inactive inheritance not exact")
			}
			found++
		}
	}
	if found != 1 {
		t.Fatal("inactive inheritance missing")
	}

}

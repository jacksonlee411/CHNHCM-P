//go:build integration

package integration_test

import (
	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/foundation"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"testing"
	"time"
)

type depGrant struct {
	ScopeKind string `json:"scope_kind"`
	ScopeID   string `json:"scope_id"`
	Action    string `json:"action"`
	Read      string `json:"read_fields_ref"`
	Write     string `json:"write_fields_ref"`
}
type depContracts struct {
	Profiles map[string][]depGrant `json:"grant_profiles"`
	Fields   map[string][]string   `json:"field_sets"`
	Cases    []map[string]any      `json:"cases"`
}

func departmentContracts(t *testing.T) depContracts {
	t.Helper()
	raw, e := os.ReadFile("../../docs/verification/data/116-r0-review-fixes/contract-fixtures.json")
	must(t, e)
	var v depContracts
	must(t, json.Unmarshal(raw, &v))
	return v
}

type depFixture struct {
	f         *fixture
	b, actor  *browser
	aliases   map[string]string
	records   map[string]string
	submitted map[string]string
	cases     depContracts
	day       string
}

func newDepartmentContract(t *testing.T) *depFixture {
	f := fresh(t, bootstrap.InitInput{EnterpriseName: "高谷集团", Start: "2026-01-01"})
	s := &depFixture{f: f, b: f.login("admin", f.password), actor: f.login("reader", f.readPassword), aliases: map[string]string{}, records: map[string]string{}, submitted: map[string]string{}, cases: departmentContracts(t), day: "2026-11-04"}
	restore := bootstrap.SetIntegrationClock(f.runtime, func() time.Time {
		date, e := time.ParseInLocation("2006-01-02", s.day, foundation.Shanghai)
		must(t, e)
		return date.Add(12 * time.Hour)
	})
	t.Cleanup(restore)
	for _, alias := range []string{"BU_A", "BU_B"} {
		v := s.b.call("POST", "/api/v1/business-units", createBody(alias, "2026-01-01"), foundation.NewID(), 201)
		s.aliases[alias] = v["buId"].(string)
	}
	ctx := s.b.call("GET", "/api/v1/departments/context", nil, "", 200)
	s.aliases["E"] = ctx["enterprise"].(map[string]any)["id"].(string)
	return s
}
func (s *depFixture) profile(name string) {
	s.f.t.Helper()
	grants, ok := s.cases.Profiles[name]
	if !ok {
		s.f.t.Fatal("unknown fixed profile", name)
	}
	tx, e := s.f.admin.Begin(s.f.ctx)
	must(s.f.t, e)
	defer tx.Rollback(s.f.ctx)
	_, e = tx.Exec(s.f.ctx, "SELECT principal_id FROM access.policy_guard WHERE instance_id=$1 AND principal_id=$2 FOR UPDATE", s.f.instance, s.actor.subject)
	must(s.f.t, e)
	_, e = tx.Exec(s.f.ctx, "DELETE FROM access.subject_grant WHERE instance_id=$1 AND principal_id=$2", s.f.instance, s.actor.subject)
	must(s.f.t, e)
	for _, g := range grants {
		id, ok := s.aliases[g.ScopeID]
		if !ok {
			s.f.t.Fatal("unmapped grant scope", g.ScopeID)
		}
		_, e = tx.Exec(s.f.ctx, `INSERT INTO access.subject_grant(instance_id,grant_id,principal_id,scope_kind,scope_id,action,read_fields,write_fields) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, s.f.instance, foundation.NewID(), s.actor.subject, g.ScopeKind, id, g.Action, s.cases.Fields[g.Read], s.cases.Fields[g.Write])
		must(s.f.t, e)
	}
	_, e = tx.Exec(s.f.ctx, "UPDATE access.policy_guard SET policy_no=policy_no+1 WHERE instance_id=$1 AND principal_id=$2", s.f.instance, s.actor.subject)
	must(s.f.t, e)
	must(s.f.t, tx.Commit(s.f.ctx))
}
func (s *depFixture) create(alias, name, bu, parent, date string, b *browser, status int) map[string]any {
	s.f.t.Helper()
	var parentID any
	if parent != "" {
		parentID = s.aliases[parent]
	}
	key := foundation.NewID()
	v := b.call("POST", "/api/v1/departments", departmentBody(alias, name, s.aliases[bu], date, parentID), key, status)
	if status == 201 {
		s.aliases[alias] = v["departmentId"].(string)
		s.records[alias+"_FIRST"] = v["targetRecordId"].(string)
		s.submitted[alias] = key
	}
	return v
}
func (s *depFixture) target(alias, record string) (map[string]any, map[string]any) {
	s.f.t.Helper()
	return s.b.departmentRecord(s.aliases[alias], s.records[record])
}
func (s *depFixture) append(alias, record, date, typ string, changes map[string]any, b *browser, status int) map[string]any {
	s.f.t.Helper()
	detail := s.b.departmentCurrent(s.aliases[alias])
	rows := detail["timeline"].([]any)
	anchor := rows[len(rows)-1].(map[string]any)
	for _, v := range rows {
		r := v.(map[string]any)
		if r["effectiveDate"].(string) < date {
			anchor = r
		}
	}
	reason := map[string]string{"CHANGE": "NAME_ADJUSTMENT", "DEACTIVATE": "ORG_RESTRUCTURING", "REACTIVATE": "BUSINESS_RESUMPTION"}[typ]
	route := "records"
	if typ == "DEACTIVATE" {
		route = "deactivate"
	}
	if typ == "REACTIVATE" {
		route = "reactivate"
	}
	body := map[string]any{"expectedSourceVersion": detail["headSourceVersion"], "anchorRecordId": anchor["recordId"], "effectiveDate": date, "operationType": typ, "operationReason": reason}
	if changes != nil {
		body["changes"] = changes
		if typ == "CHANGE" && (changes["managementBuId"] != nil || changes["parentDepartmentId"] != nil) {
			body["operationReason"] = "ORGANIZATION_REASSIGNMENT"
		}
	}
	key := foundation.NewID()
	v := b.call("POST", "/api/v1/departments/"+s.aliases[alias]+"/"+route, body, key, status)
	if status == 201 {
		s.records[record] = v["targetRecordId"].(string)
		s.submitted[record] = key
	}
	return v
}
func (s *depFixture) edit(alias, record, action string, extra map[string]any, b *browser, status int) map[string]any {
	s.f.t.Helper()
	detail, r := s.target(alias, record)
	body := map[string]any{"expectedSourceVersion": detail["headSourceVersion"], "expectedRecordRevision": r["revision"]}
	for k, v := range extra {
		body[k] = v
	}
	key := foundation.NewID()
	out := b.call("POST", "/api/v1/departments/"+s.aliases[alias]+"/records/"+s.records[record]+"/"+action, body, key, status)
	if status == 200 {
		s.submitted[record+"_"+action] = key
	}
	return out
}
func (s *depFixture) stopBU(alias, date string, status int) map[string]any {
	s.f.t.Helper()
	id := s.aliases[alias]
	detail := s.b.call("GET", "/api/v1/business-units/"+id, nil, "", 200)
	rows := detail["timeline"].([]any)
	last := rows[len(rows)-1].(map[string]any)
	return s.b.call("POST", "/api/v1/business-units/"+id+"/deactivate", map[string]any{"expectedSourceVersion": detail["headSourceVersion"], "anchorRecordId": last["recordId"], "effectiveDate": date, "operationType": "DEACTIVATE", "operationReason": "ORG_RESTRUCTURING"}, foundation.NewID(), status)
}
func (s *depFixture) setup(c map[string]any) {
	s.f.t.Helper()
	before := c["before"].(map[string]any)
	type entry struct {
		alias string
		r     map[string]any
	}
	firsts, rest := []entry{}, []entry{}
	for alias, v := range before {
		if alias != "P" && alias != "C" {
			continue
		}
		rows, ok := v.([]any)
		if !ok {
			continue
		}
		for i, row := range rows {
			e := entry{alias, row.(map[string]any)}
			if i == 0 {
				firsts = append(firsts, e)
			} else {
				rest = append(rest, e)
			}
		}
	}
	sort.Slice(firsts, func(i, j int) bool { return firsts[i].alias > firsts[j].alias })
	for _, e := range firsts {
		parent, _ := e.r["parent"].(string)
		s.create(e.alias, e.r["name"].(string), e.r["bu"].(string), parent, e.r["date"].(string), s.b, 201)
		s.records[e.r["record"].(string)] = s.records[e.alias+"_FIRST"]
	}
	sort.SliceStable(rest, func(i, j int) bool { return rest[i].r["date"].(string) < rest[j].r["date"].(string) })
	for _, e := range rest {
		typ := e.r["operation_type"].(string)
		var changes map[string]any
		if typ != "DEACTIVATE" {
			changes = map[string]any{"name": e.r["name"], "managementBuId": s.aliases[e.r["bu"].(string)], "parentDepartmentId": nil}
			if p, ok := e.r["parent"].(string); ok {
				changes["parentDepartmentId"] = s.aliases[p]
			}
		}
		s.append(e.alias, e.r["record"].(string), e.r["date"].(string), typ, changes, s.b, 201)
	}
	if bu, ok := before["BU_A"].(map[string]any); ok {
		if end, ok := bu["active_end"].(string); ok {
			s.stopBU("BU_A", end, 201)
		}
	}
}
func (s *depFixture) current(alias string) map[string]any {
	return s.b.departmentCurrent(s.aliases[alias])
}
func depPath(detail map[string]any) string {
	p := detail["path"].(map[string]any)
	value, _ := p["departmentPath"].(string)
	return value
}
func (s *depFixture) exact(alias, source string) map[string]any {
	return s.b.call("GET", "/api/v1/departments/"+s.aliases[alias]+"?sourceVersion="+source, nil, "", 200)
}
func (s *depFixture) assertNoBusinessChange(before string) {
	s.f.t.Helper()
	if after := s.businessDigest(); after != before {
		s.f.t.Fatal("rejection changed department/hierarchy business facts")
	}
}
func (s *depFixture) businessDigest() string {
	raw, err := json.Marshal(s.f.history().Departments)
	must(s.f.t, err)
	return fmt.Sprintf("%x", sha256.Sum256(raw))
}

func TestDepartmentFixedContractsD20D39(t *testing.T) {
	cases := departmentContracts(t)
	for _, c := range cases.Cases {
		id := c["id"].(string)
		if id == "D39" {
			continue
		}
		t.Run(id, func(t *testing.T) {
			s := newDepartmentContract(t)
			s.setup(c)
			switch id {
			case "D20":
				s.profile("maintain_A")
				v := s.create("C", "组织发展部", "BU_A", "", "2027-01-01", s.actor, 201)
				detail := s.actor.call("GET", "/api/v1/departments/"+s.aliases["C"], nil, "", 200)
				if detail["status"] != "not_effective" || detail["result"] != "M02_NO_FACT" || detail["path"].(map[string]any)["pathMode"] != "future_first" {
					t.Fatal("future-first violates 123")
				}
				r := s.actor.call("POST", "/api/v1/departments/submissions/"+s.submitted["C"]+"/resolve", nil, "", 200)
				if r["sourceVersion"] != v["sourceVersion"] {
					t.Fatal("original recovery differs")
				}
				if s.actor.call("GET", "/api/v1/departments", nil, "", 200)["count"] != float64(1) {
					t.Fatal("future first absent in list")
				}
				s.day = "2027-01-01"
				if s.actor.call("GET", "/api/v1/departments/"+s.aliases["C"], nil, "", 200)["status"] != "active" {
					t.Fatal("future first did not become active")
				}
			case "D21":
				for _, profile := range []string{"create_A_only", "create_read_A"} {
					s.profile(profile)
					before := s.businessDigest()
					s.create("C", "组织发展部", "BU_A", "", "2027-01-01", s.actor, 403)
					s.assertNoBusinessChange(before)
				}
			case "D22":
				old := s.current("C")["headSourceVersion"].(string)
				s.profile("first_redate_A")
				v := s.edit("C", "C_FIRST", "redate", map[string]any{"newEffectiveDate": "2027-01-01"}, s.actor, 200)
				if v["outcome"] != "accepted" || s.current("C")["status"] != "not_effective" {
					t.Fatal("first redate failed")
				}
				s.actor.call("POST", "/api/v1/departments/submissions/"+s.submitted["C_FIRST_redate"]+"/resolve", nil, "", 200)
				if s.exact("C", old)["selectedRecord"].(map[string]any)["effectiveDate"] != "2026-01-01" {
					t.Fatal("old first source rewritten")
				}
			case "D23":
				for _, profile := range []string{"maintain_A", "correct_AB_without_reclassify"} {
					s.profile(profile)
					status := 403
					if profile == "correct_AB_without_reclassify" {
						status = 200
					}
					s.edit("C", "C_FIRST", "correct", map[string]any{"changes": map[string]any{"managementBuId": s.aliases["BU_B"], "parentDepartmentId": nil}}, s.actor, status)
				}
				r := s.current("C")["selectedRecord"].(map[string]any)
				if r["operationType"] != "CREATE" || r["operationReason"] != "NEW_ORGANIZATION" {
					t.Fatal("relation correction reclassified first")
				}
				s.profile("read_A")
				s.actor.call("GET", "/api/v1/departments/"+s.aliases["C"], nil, "", 404)
				s.profile("read_B")
				s.actor.call("GET", "/api/v1/departments/"+s.aliases["C"], nil, "", 200)
			case "D24":
				var original string
				for _, fact := range s.f.history().Departments {
					if fact.Object == s.aliases["C"] && fact.Operation.Source.Number == 1 {
						original = fact.Operation.Source.ID
					}
				}
				s.day = "2026-12-31"
				s.profile("read_A")
				s.actor.call("GET", "/api/v1/departments", nil, "", 200)
				choice := draftBody(map[string]any{}, departmentDraft(s.actor, map[string]any{}))
				status, _ := departmentCSVResponse(t, s.actor, choice)
				if status != 200 {
					t.Fatal("initial authorized export", status)
				}
				source := s.current("C")["headSourceVersion"].(string)
				for _, day := range []string{"2026-12-31", "2027-01-01"} {
					s.day = day
					for _, profile := range []string{"read_A", "read_B"} {
						s.profile(profile)
						status := 404
						if day == "2026-12-31" && profile == "read_A" || day == "2027-01-01" && profile == "read_B" {
							status = 200
						}
						for _, suffix := range []string{"", "?sourceVersion=" + original, "?sourceVersion=" + source, "?selectedRecordId=" + s.records["C_FIRST"]} {
							s.actor.call("GET", "/api/v1/departments/"+s.aliases["C"]+suffix, nil, "", status)
						}
					}
				}
				s.profile("read_A")
				status, _ = departmentCSVResponse(t, s.actor, choice)
				if status != 409 {
					t.Fatal("old selection exported after scope changed", status)
				}
				if s.current("C")["headSourceVersion"] != source {
					t.Fatal("clock rewrote source")
				}
			case "D25":
				// The fixture's original writer is now restricted to the same read_A profile.
				originalActor := s.actor
				s.actor = s.b
				s.profile("read_A")
				s.actor = originalActor
				s.b.call("POST", "/api/v1/departments/submissions/"+s.submitted["C"]+"/resolve", nil, "", 200)
				s.profile("read_A")
				for _, suffix := range []string{"", "?sourceVersion=" + s.current("C")["headSourceVersion"].(string)} {
					v := s.actor.call("GET", "/api/v1/departments/"+s.aliases["C"]+suffix, nil, "", 200)
					if v["status"] != "inactive" || v["path"].(map[string]any)["pathBusinessAt"] != "2026-05-31" {
						t.Fatal("inactive scope/path")
					}
				}
			case "D26", "D27":
				before := s.current("C")["headSourceVersion"]
				status := 201
				if id == "D27" {
					s.day = "2026-06-15"
					status = 409
				}
				digest := s.businessDigest()
				s.append("P", "P_MOVE", "2026-07-01", "CHANGE", map[string]any{"managementBuId": s.aliases["BU_B"]}, s.b, status)
				if status == 409 {
					s.assertNoBusinessChange(digest)
				}
				if s.current("C")["headSourceVersion"] != before {
					t.Fatal("parent move rewrote child")
				}
			case "D28":
				s.append("C", "resumeBad", "2026-08-01", "REACTIVATE", map[string]any{"managementBuId": s.aliases["BU_A"], "parentDepartmentId": s.aliases["P"]}, s.b, 409)
				s.profile("maintain_A")
				s.append("C", "resumeDenied", "2026-08-01", "REACTIVATE", map[string]any{"managementBuId": s.aliases["BU_B"], "parentDepartmentId": s.aliases["P"]}, s.actor, 403)
				s.profile("maintain_AB")
				s.append("C", "resume", "2026-08-01", "REACTIVATE", map[string]any{"managementBuId": s.aliases["BU_B"], "parentDepartmentId": s.aliases["P"]}, s.actor, 201)
			case "D29", "D30", "D31":
				before := s.businessDigest()
				alias, record, action := "C", "C_STOP", "delete"
				extra := map[string]any{}
				if id != "D29" {
					action = "redate"
					extra["newEffectiveDate"] = "2026-08-01"
				}
				if id == "D31" {
					alias, record = "P", "P_STOP"
					extra["newEffectiveDate"] = "2026-06-01"
				}
				s.edit(alias, record, action, extra, s.b, 409)
				s.assertNoBusinessChange(before)
			case "D32":
				s.edit("C", "C_STOP", "delete", nil, s.b, 409)
				detail, a := s.target("C", "C_STOP")
				_, z := s.target("C", "C_RESUME")
				s.b.call("POST", "/api/v1/departments/"+s.aliases["C"]+"/lifecycle-pairs/delete", map[string]any{"expectedSourceVersion": detail["headSourceVersion"], "deactivationRecordId": a["recordId"], "deactivationRevision": a["revision"], "reactivationRecordId": z["recordId"], "reactivationRevision": z["revision"]}, foundation.NewID(), 200)
				if len(s.current("C")["timeline"].([]any)) != 1 {
					t.Fatal("pair not removed together")
				}
			case "D33":
				old := s.current("C")["headSourceVersion"].(string)
				s.edit("C", "C_STOP", "redate", map[string]any{"newEffectiveDate": "2026-07-01"}, s.b, 200)
				current := s.current("C")
				if depPath(current) != "组织效能中心 / 组织发展部" || current["path"].(map[string]any)["pathBusinessAt"] != "2026-06-30" {
					t.Fatal("stop-date projection wrong")
				}
				if depPath(s.exact("C", old)) != "组织管理中心 / 组织发展部" {
					t.Fatal("old stop path rewritten")
				}
			case "D34", "D35":
				if id == "D34" {
					s.day = "2026-05-01"
					s.append("C", "C_STOP", "2026-08-01", "DEACTIVATE", nil, s.b, 201)
				}
				source := s.current("C")["headSourceVersion"].(string)
				date := "2026-07-01"
				if id == "D35" {
					date = "2026-04-01"
				}
				if id == "D34" {
					s.day = "2026-06-01"
				}
				s.append("P", "P_NAME", date, "CHANGE", map[string]any{"name": "组织管理中心"}, s.b, 201)
				if id == "D35" {
					s.append("P", "P_LATE", "2026-07-01", "CHANGE", map[string]any{"name": "组织运营中心"}, s.b, 201)
				}
				if id == "D34" {
					s.day = "2026-09-01"
				}
				if depPath(s.current("C")) != "组织管理中心 / 组织发展部" || depPath(s.exact("C", source)) != "组织效能中心 / 组织发展部" {
					t.Fatal("current vs explicit head evidence collapsed")
				}
			case "D36":
				s.profile("path_reader_no_ancestor")
				source := s.current("C")["headSourceVersion"].(string)
				for _, suffix := range []string{"", "?sourceVersion=" + source} {
					v := s.actor.call("GET", "/api/v1/departments/"+s.aliases["C"]+suffix, nil, "", 200)
					p := v["path"].(map[string]any)
					if len(p) != 2 || p["pathMode"] != "restricted" {
						t.Fatal("restricted path disclosed metadata", p)
					}
				}
			case "D37":
				for _, action := range []string{"delete", "redate"} {
					s.profile("rollback_B_without_reassign")
					extra := map[string]any{}
					if action == "redate" {
						extra["newEffectiveDate"] = "2026-08-01"
					}
					before := s.businessDigest()
					s.edit("C", "C_MOVE", action, extra, s.actor, 403)
					s.assertNoBusinessChange(before)
				}
				s.profile("maintain_AB")
				s.edit("C", "C_MOVE", "redate", map[string]any{"newEffectiveDate": "2026-08-01"}, s.actor, 200)
				fresh := newDepartmentContract(t)
				fresh.setup(c)
				fresh.profile("maintain_AB")
				fresh.edit("C", "C_MOVE", "delete", nil, fresh.actor, 200)
			case "D38":
				s.profile("lifecycle_A_without_reassign")
				s.append("C", "C_STOP", "2026-06-01", "DEACTIVATE", nil, s.actor, 201)
				s.append("C", "C_RESUME", "2026-08-01", "REACTIVATE", nil, s.actor, 201)
			default:
				t.Fatal("unimplemented fixed case", id)
			}
			t.Logf("fixed fixture %s aliases=%v", id, s.aliases)
		})
	}
}

func TestDepartmentD28RootReactivation(t *testing.T) {
	s := newDepartmentContract(t)
	for _, c := range s.cases.Cases {
		if c["id"] == "D28" {
			s.setup(c)
		}
	}
	s.append("C", "C_RESUME", "2026-08-01", "REACTIVATE", map[string]any{"managementBuId": s.aliases["BU_A"], "parentDepartmentId": nil}, s.b, 201)
	if depPath(s.current("C")) != "组织发展部" {
		t.Fatal("explicit null did not select enterprise root")
	}
}

func TestDepartmentFixedContractsD01D18(t *testing.T) {
	for _, id := range []string{"D01", "D02", "D03", "D04", "D05", "D06", "D07", "D08", "D09", "D10", "D11", "D12", "D13", "D15", "D16", "D17", "D18"} {
		t.Run(id, func(t *testing.T) {
			s := newDepartmentContract(t)
			switch id {
			case "D01", "D15":
				before := s.f.scalar("SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.bu_history")
				s.create("A", "组织发展部", "BU_A", "", "2026-01-01", s.b, 201)
				s.create("B", "组织效能部", "BU_A", "", "2026-01-01", s.b, 201)
				if s.aliases["A"] == s.aliases["B"] || s.f.scalar("SELECT count(*) FROM organization.department") != 2 || s.f.scalar("SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.bu_history") != before {
					t.Fatal("independent identity or table ownership violated")
				}
			case "D02":
				s.create("A", "组织发展部", "BU_A", "", "2026-01-01", s.b, 201)
				before := s.businessDigest()
				s.create("A", "冲突编码", "BU_B", "", "2026-01-01", s.b, 409)
				s.assertNoBusinessChange(before)
			case "D03":
				v := s.create("C", "组织发展部", "BU_A", "", "2026-01-01", s.b, 201)
				s.append("C", "duplicate", "2026-01-01", "CHANGE", map[string]any{"name": "组织发展处"}, s.b, 409)
				s.edit("C", "C_FIRST", "correct", map[string]any{"changes": map[string]any{"name": "组织发展处"}}, s.b, 200)
				if len(s.current("C")["timeline"].([]any)) != 1 || s.exact("C", v["sourceVersion"].(string))["selectedRecord"].(map[string]any)["name"] != "组织发展部" {
					t.Fatal("correction rewrote old source")
				}
			case "D04", "D05", "D06":
				s.create("C", "组织发展部", "BU_A", "", "2026-01-01", s.b, 201)
				stop := "2026-06-01"
				if id == "D05" {
					stop = "2026-06-02"
				}
				s.append("C", "C_STOP", stop, "DEACTIVATE", nil, s.b, 201)
				status := 201
				if id == "D05" {
					status = 409
				}
				s.stopBU("BU_A", "2026-06-01", status)
				if id == "D06" {
					before := s.businessDigest()
					s.edit("C", "C_STOP", "delete", nil, s.b, 409)
					s.assertNoBusinessChange(before)
				}
			case "D07":
				v := s.create("C", "组织发展部", "BU_A", "", "2026-01-01", s.b, 201)
				s.append("C", "C_MOVE", "2026-06-01", "CHANGE", map[string]any{"managementBuId": s.aliases["BU_B"], "parentDepartmentId": nil}, s.b, 201)
				current := s.current("C")
				if current["departmentId"] != v["departmentId"] || current["code"] != "C" || current["selectedRecord"].(map[string]any)["managementBuId"] != s.aliases["BU_B"] {
					t.Fatal("move changed identity")
				}
			case "D08", "D09", "D12":
				s.create("P", "组织效能中心", "BU_A", "", "2026-01-01", s.b, 201)
				v := s.create("C", "组织发展部", "BU_A", "P", "2026-01-01", s.b, 201)
				if depPath(s.current("C")) != "组织效能中心 / 组织发展部" {
					t.Fatal("wrong parent path")
				}
				if id == "D09" {
					before := s.current("C")
					s.append("P", "P_NAME", "2026-06-01", "CHANGE", map[string]any{"name": "组织管理中心"}, s.b, 201)
					list := s.b.call("GET", "/api/v1/departments", nil, "", 200)
					found := false
					for _, item := range list["items"].([]any) {
						r := item.(map[string]any)
						if r["id"] == s.aliases["C"] {
							found = depPath(r) == "组织管理中心 / 组织发展部"
						}
					}
					if !found || s.current("C")["headSourceVersion"] != before["headSourceVersion"] || depPath(s.exact("C", v["sourceVersion"].(string))) != "组织效能中心 / 组织发展部" {
						t.Fatal("current and sealed path are not independent")
					}
				}
				if id == "D12" {
					s.profile("path_reader_no_ancestor")
					v := s.actor.call("GET", "/api/v1/departments/"+s.aliases["C"], nil, "", 200)
					if len(v["path"].(map[string]any)) != 2 {
						t.Fatal("restricted path disclosed ancestry")
					}
				}
			case "D10":
				s.create("A", "甲", "BU_A", "", "2026-01-01", s.b, 201)
				s.create("B", "乙", "BU_A", "", "2026-01-01", s.b, 201)
				s.append("A", "A_PARENT", "2026-08-01", "CHANGE", map[string]any{"parentDepartmentId": s.aliases["B"]}, s.b, 201)
				before := s.businessDigest()
				s.append("B", "B_PARENT", "2026-07-01", "CHANGE", map[string]any{"parentDepartmentId": s.aliases["A"]}, s.b, 409)
				s.assertNoBusinessChange(before)
			case "D11":
				s.create("C", "组织发展部", "BU_A", "", "2026-01-01", s.b, 201)
				s.profile("maintain_A")
				before := s.businessDigest()
				s.append("C", "C_MOVE", "2026-06-01", "CHANGE", map[string]any{"managementBuId": s.aliases["BU_B"]}, s.actor, 403)
				s.assertNoBusinessChange(before)
			case "D13":
				key := foundation.NewID()
				saved := s.b.call("POST", "/api/v1/business-units", createBody("NAMESPACE", "2026-01-01"), key, 201)
				closed := s.b.call("POST", "/api/v1/departments/submissions/"+key+"/resolve", nil, "", 200)
				if closed["outcome"] != "not_applied" {
					t.Fatal("cross-kind recovery read BU")
				}
				own := s.b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
				if own["sourceVersion"] != saved["sourceVersion"] {
					t.Fatal("department closed BU")
				}
			case "D16":
				s.create("C", "组织发展部", "BU_A", "", "2026-01-01", s.b, 201)
				v := s.append("C", "C_NAME", "2026-06-01", "CHANGE", map[string]any{"name": "组织发展处"}, s.b, 201)
				// Synthesize an immutable revision from the previous classification catalog.
				s.f.editDepartmentRevision(v["targetRevision"].(string), func(r *bootstrap.IntegrationDepartmentRevision) {
					r.Reason = "NAME_CHANGE"
					r.TypeLabel = "资料变更"
					r.ReasonLabel = "名称变更"
				})
				before := s.exact("C", v["sourceVersion"].(string))["selectedRecord"].(map[string]any)
				if before["reasonLabel"] != "名称变更" {
					t.Fatal("stored label reinterpreted")
				}
				s.edit("C", "C_NAME", "correct", map[string]any{"changes": map[string]any{"shortName": "发展"}}, s.b, 200)
				after := s.current("C")["selectedRecord"].(map[string]any)
				if after["operationReason"] != "NAME_CHANGE" || after["typeLabel"] != "资料变更" {
					t.Fatal("plain correction requalified old reason")
				}
			case "D17":
				s.stopBU("BU_A", "2026-06-01", 201)
				before := s.businessDigest()
				s.create("C", "组织发展部", "BU_A", "", "2026-01-01", s.b, 409)
				s.assertNoBusinessChange(before)
			case "D18":
				s.create("P", "组织效能中心", "BU_A", "", "2026-01-01", s.b, 201)
				s.create("C", "组织发展部", "BU_A", "P", "2026-01-01", s.b, 201)
				before := s.businessDigest()
				s.append("P", "P_MOVE", "2026-06-01", "CHANGE", map[string]any{"managementBuId": s.aliases["BU_B"]}, s.b, 409)
				s.assertNoBusinessChange(before)
			}
			t.Logf("fixed fixture %s aliases=%v", id, s.aliases)
		})
	}
}

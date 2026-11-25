//go:build integration

package bootstrap_test

import (
	"bytes"
	"chnhcm.local/chnhcm/db/migrations"
	"chnhcm.local/chnhcm/internal/bootstrap"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	staffpub "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	f "chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"maps"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

func rows400(t *testing.T, b *browser, path string, input any) []map[string]any {
	t.Helper()
	raw, e := json.Marshal(input)
	must(t, e)
	req, e := http.NewRequest("POST", b.f.server.URL+path, bytes.NewReader(raw))
	must(t, e)
	req.Header.Set("Origin", b.f.server.URL)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", b.csrf)
	res, e := b.client.Do(req)
	must(t, e)
	defer res.Body.Close()
	var rows []map[string]any
	if res.StatusCode != 200 {
		var v map[string]any
		_ = json.NewDecoder(res.Body).Decode(&v)
		t.Fatalf("%s: %d %v", path, res.StatusCode, v)
	}
	must(t, json.NewDecoder(res.Body).Decode(&rows))
	return rows
}
func TestHire400Readiness(t *testing.T) {
	h := legalSetup206(t)
	b := h.b
	reader := h.f.login("reader", h.f.readPassword)
	a := h.create("A400", "CN", "legal_person")["id"].(string)
	other := h.create("B400", "CN", "branch")["id"].(string)
	base := legal206Path + "/" + a + "/purposes"
	bu := b.call("POST", "/api/v1/business-units", createBody("BU400", "2026-01-01"), f.NewID(), 201)["buId"].(string)
	loc := (&location204{t, h.f, b, h.countries}).create("LOC400", "MO", "", "2026-01-01")["id"].(string)
	save := func(path, head string, row map[string]any) string {
		t.Helper()
		key := f.NewID()
		body := map[string]any{"expectedVersion": head, "row": row}
		v := b.call("POST", path, body, key, 200)["version"].(string)
		if again := b.call("POST", path, body, key, 200); again["version"] != v {
			t.Fatal("retry changed original")
		}
		return v
	}
	first := map[string]any{"id": f.NewID(), "role": "employer", "kind": "declaration", "from": "2027-04-01", "until": "2027-05-01", "basis": "组织授权决定 A"}
	v1 := save(base, "", first)
	original := reader.call("GET", base+"?sourceVersion="+v1, nil, "", 200)
	in := map[string]any{"role": "employer", "from": "2027-04-01", "until": "2027-06-01"}
	gap := b.call("POST", base+"/check", in, "", 409)
	if !strings.Contains(gap["message"].(string), "2027-05-01") {
		t.Fatal(gap)
	}
	first = maps.Clone(original["rows"].([]any)[0].(map[string]any))
	delete(first, "until")
	first["requireBU"] = true
	first["requireLocation"] = true
	head := save(base, v1, first)
	binding := map[string]any{"id": f.NewID(), "role": "employer", "kind": "bu", "target": bu, "from": "2027-04-01", "basis": "适用业务单元"}
	head = save(base, head, binding)
	in["bu"] = bu
	if m := b.call("POST", base+"/check", in, "", 409); !strings.Contains(m["message"].(string), "地点") {
		t.Fatal(m)
	}
	binding["id"] = f.NewID()
	binding["kind"] = "location"
	binding["target"] = loc
	head = save(base, head, binding)
	delete(in, "until")
	in["location"] = loc
	good := rows400(t, b, base+"/check", in)
	if len(good) != 1 || len(good[0]["bindings"].([]any)) != 2 || good[0]["purposeVersion"] != head || good[0]["until"] != nil {
		t.Fatal(good)
	}
	// A reader preview and a submitter's recheck use the same source and bindings.
	cred := geoCredentials203(t, h.f, "admin", h.f.password)
	tx, e := h.f.runtime.Pool.Begin(h.f.ctx)
	must(t, e)
	defer tx.Rollback(h.f.ctx)
	verified, e := h.f.runtime.LegalPurposes.CheckPurpose(dbtx.With(h.f.ctx, tx), cred, a, org.PurposeCheck{Period: f.Period{From: "2027-04-01"}, BU: bu, Location: loc, Role: "employer", Expected: head})
	must(t, e)
	if len(verified) != 1 || verified[0].PurposeVersion != head {
		t.Fatal("wrong locked source")
	}
	must(t, tx.Rollback(h.f.ctx))
	in["adopted"] = good
	h.correct(h.detail(a, ""), map[string]any{"name": "主体更名"}, nil, 200)
	b.call("POST", base+"/check", in, "", 409)
	delete(in, "adopted")
	if v := h.add(h.detail(a, ""), "DEACTIVATE", "2027-05-01", nil, 409); !strings.Contains(v["message"].(string), "2027-05-01") {
		t.Fatal(v)
	}
	// An unused binding changes its source but never divides an adopted term.
	extraBU := b.call("POST", "/api/v1/business-units", createBody("BU401", "2026-01-01"), f.NewID(), 201)["buId"].(string)
	head = save(base, head, map[string]any{"id": f.NewID(), "role": "employer", "kind": "bu", "target": extraBU, "from": "2027-04-15", "basis": "另一业务单元"})
	if parts := rows400(t, b, base+"/check", in); len(parts) != 1 || parts[0]["from"] != "2027-04-01" {
		t.Fatalf("unrelated binding split adoption: %v", parts)
	}
	alt := legal206Path + "/" + other + "/purposes"
	second := maps.Clone(first)
	delete(second, "revision")
	delete(second, "requireBU")
	delete(second, "requireLocation")
	second["id"] = f.NewID()
	second["role"] = "receiver"
	altHead := save(alt, "", second)
	b.call("POST", alt+"/check", in, "", 409)
	second["role"] = "employer"
	second["id"] = f.NewID()
	save(alt, altHead, second)
	candidates := maps.Clone(in)
	candidates["page"] = 1
	if rows := rows400(t, b, legal206Path+"/purpose-candidates", candidates); len(rows) != 2 {
		t.Fatal(rows)
	}
	if got := reader.call("GET", base+"?sourceVersion="+v1, nil, "", 200); !reflect.DeepEqual(got["rows"], original["rows"]) {
		t.Fatal("original changed")
	}
	reader.call("POST", base, map[string]any{"expectedVersion": head, "row": binding}, f.NewID(), 403)
	h.policy("DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='legalpurpose.read'", reader.subject)
	restricted := maps.Clone(in)
	delete(restricted, "location")
	if m := reader.call("POST", base+"/check", restricted, "", 409); m["code"] != "M02_PURPOSE_UNAVAILABLE" || m["message"] != "当前安排不满足办理条件" {
		t.Fatal("restricted preview disclosed the missing binding", m)
	}
	h.policy("DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='legalpurpose.check'", reader.subject)
	reader.call("POST", base+"/check", in, "", 403)
	staff := staffing307{t, h.f, b}
	family := staff.create("jobfamily", "F400", "")["id"].(string)
	job := staff.create("job", "J400", family)["id"].(string)
	dep := b.call("POST", "/api/v1/departments", departmentCreateBody("DEP400", bu), f.NewID(), 201)["departmentId"].(string)
	const path = "/api/v1/assignment-policies"
	enterprise := b.call("GET", path, nil, "", 200)["owner"]
	policy := map[string]any{"id": f.NewID(), "category": "Employee", "purpose": "hire", "mode": "JobOnly", "from": "2027-04-01", "until": "2027-05-01"}
	q := map[string]any{"category": "Employee", "purpose": "hire", "enterprise": enterprise, "from": "2027-04-01", "until": "2027-05-01", "job": job, "department": dep}
	b.call("POST", path+"/check", q, "", 409)
	pv := save(path, "", policy)
	rows := rows400(t, b, path+"/check", q)
	if len(rows) != 1 || rows[0]["policyVersion"] != pv || rows[0]["policy"] == "" || rows[0]["departmentRevision"] == "" || rows[0]["jobRevision"] == "" || rows[0]["position"] != "" || rows[0]["seats"] != float64(0) {
		t.Fatal(rows)
	}
	must(t, bootstrap.Initialize(h.f.ctx, h.f.admin, bootstrap.InitInput{Instance: h.f.instance, EnterpriseCode: "LOCAL", EnterpriseName: "本地验证企业",
		Login: "admin", Password: h.f.password, FirstHireRule: &bootstrap.FirstHireRuleInput{Version: "T02-employee-1", From: "2027-04-01"}}))
	tx, e = h.f.runtime.Pool.Begin(h.f.ctx)
	must(t, e)
	defer tx.Rollback(h.f.ctx)
	// The customer action and both dated source checks share the proposed term
	// and the caller's real transaction before the first Hire can be written.
	term := f.Period{From: "2027-04-01", Until: "2027-05-01"}
	submitCtx := dbtx.With(h.f.ctx, tx)
	rule, e := h.f.runtime.FirstHireRules.FirstHireRule(submitCtx, cred, term.From)
	must(t, e)
	if rule.Version != "T02-employee-1" || rule.Action != "Hire" || rule.InitialState != "Active" || rule.PrimaryPurpose != "personnel_management" {
		t.Fatal("incorrect effective first hire policy")
	}
	employer, e := h.f.runtime.LegalPurposes.CheckPurpose(submitCtx, cred, a, org.PurposeCheck{Period: term, BU: bu, Location: loc, Role: "employer", Expected: head})
	must(t, e)
	if len(employer) != 1 || employer[0].Period != term || employer[0].PurposeVersion != head || employer[0].EntityRevision == "" || employer[0].Declaration == "" || len(employer[0].Bindings) != 2 {
		t.Fatal("locked employer or bindings mismatch")
	}
	guarded, e := h.f.runtime.AssignmentPolicies.CheckAssignment(submitCtx, cred, staffpub.AssignmentCheck{Period: term, Enterprise: enterprise.(string), Category: rule.Relationship, Purpose: "hire", Job: job, Department: dep, Expected: pv})
	must(t, e)
	if len(guarded) != 1 || guarded[0].Period != term || guarded[0].Seats != 0 || guarded[0].Position != "" || guarded[0].Policy != rows[0]["policy"] || guarded[0].JobRevision != rows[0]["jobRevision"] || guarded[0].DepartmentRevision != rows[0]["departmentRevision"] {
		t.Fatal("locked JobOnly adoption mismatch")
	}
	must(t, tx.Rollback(h.f.ctx))
	q["job"] = f.NewID()
	b.call("POST", path+"/check", q, "", 404)
	q["job"] = job
	q["position"] = f.NewID()
	b.call("POST", path+"/check", q, "", 409)
	delete(q, "position")
	policy["id"] = f.NewID()
	b.call("POST", path, map[string]any{"expectedVersion": pv, "row": policy}, f.NewID(), 409)
	policy["from"] = "2027-05-01"
	delete(policy, "until")
	policy["mode"] = "PositionExempt"
	save(path, pv, policy)
	delete(q, "until")
	gap = b.call("POST", path+"/check", q, "", 409)
	if !strings.Contains(gap["message"].(string), "2027-05-01") {
		t.Fatal(gap)
	}
	// A later mode change must not mask an earlier invalid job period.
	staff.add("job", staff.detail("job", job, ""), "DEACTIVATE", "2027-04-20", nil, 200)
	if early := b.call("POST", path+"/check", q, "", 409); !strings.Contains(early["message"].(string), "2027-04-20") {
		t.Fatalf("not the first invalid day: %v", early)
	}
	// Split the second (open) policy with one publication and retain its exact old snapshot.
	before := b.call("GET", path, nil, "", 200)
	oldRows := before["rows"].([]any)
	anchor := oldRows[1].(map[string]any)
	split := maps.Clone(anchor)
	split["id"] = f.NewID()
	delete(split, "revision")
	split["mode"] = "PositionCounted"
	split["from"] = "2027-09-01"
	b.call("POST", path, map[string]any{"expectedVersion": before["version"], "row": split, "insertAfter": anchor["id"]}, f.NewID(), 200)
	after := b.call("GET", path, nil, "", 200)["rows"].([]any)
	if len(after) != 3 || after[1].(map[string]any)["until"] != "2027-09-01" {
		t.Fatal("split not atomic", after)
	}
	if old := b.call("GET", path+"?sourceVersion="+before["version"].(string), nil, "", 200); !reflect.DeepEqual(old["rows"], oldRows) {
		t.Fatal("old membership changed")
	}
	reader.call("GET", path+"?sourceVersion="+pv, nil, "", 200)
	reader.call("POST", path, map[string]any{"expectedVersion": pv, "row": policy}, f.NewID(), 403)
}

func TestHire400RuleRecovery(t *testing.T) {
	h := legalSetup206(t)
	b := h.b
	entity := h.create("R400", "MO", "branch")["id"].(string)
	for _, tc := range []struct {
		path, receiptPath, action string
		row                       map[string]any
	}{
		{legal206Path + "/" + entity + "/purposes", legal206Path + "/purpose-submissions/", "legalpurpose.maintain", map[string]any{"id": f.NewID(), "role": "employer", "kind": "declaration", "from": "2027-04-01", "basis": "本企业授权"}},
		{"/api/v1/assignment-policies", "/api/v1/assignment-policies/submissions/", "assignmentpolicy.maintain", map[string]any{"id": f.NewID(), "category": "Employee", "purpose": "hire", "from": "2027-04-01", "mode": "JobOnly"}},
	} {
		t.Run(tc.action, func(t *testing.T) {
			resolve := func(k string) map[string]any {
				return b.call("POST", tc.receiptPath+k+"/resolve", nil, "", 200)
			}
			key := f.NewID()
			input := map[string]any{"expectedVersion": "", "row": tc.row}
			r := b.call("POST", tc.path, input, key, 200)
			requireReceipt := func(got map[string]any, key, outcome string, v any) {
				t.Helper()
				if got["submissionId"] != key || got["outcome"] != outcome || got["version"] != v {
					t.Fatalf("unexpected receipt: %v", got)
				}
			}
			requireReceipt(resolve(key), key, "accepted", r["version"])
			view := b.call("GET", tc.path, nil, "", 200)
			row := view["rows"].([]any)[0].(map[string]any)
			same := map[string]any{"expectedVersion": r["version"], "row": row}
			k := f.NewID()
			requireReceipt(b.call("POST", tc.path, same, k, 200), k, "no_change", r["version"])
			newRow := maps.Clone(row)
			newRow["from"] = "2027-05-01"
			edit := map[string]any{"expectedVersion": r["version"], "row": newRow}
			b.call("POST", tc.path, edit, f.NewID(), 200)
			requireReceipt(b.call("POST", tc.path, same, k, 200), k, "no_change", r["version"])
			b.call("POST", tc.path, edit, k, 409)
			reject := f.NewID()
			b.call("POST", tc.path, edit, reject, 409)
			requireReceipt(resolve(reject), reject, "rejected", "")
			closed := f.NewID()
			requireReceipt(resolve(closed), closed, "not_applied", "")
			if e := b.call("POST", tc.path, edit, closed, 409); e["code"] != "RULE_SUBMISSION_CLOSED" {
				t.Fatal(e)
			}
			if versions := b.call("GET", tc.path, nil, "", 200)["history"].([]any); len(versions) != 2 {
				t.Fatal("non-publishing requests created a source")
			}
			h.policy("DELETE FROM access.subject_grant WHERE principal_id=$1 AND action=$2", b.subject, tc.action)
			b.call("POST", tc.path, same, k, 403)
			requireReceipt(resolve(k), k, "no_change", r["version"])
			reader := h.f.login("reader", h.f.readPassword)
			other := reader.call("POST", tc.receiptPath+key+"/resolve", nil, "", 200)
			requireReceipt(other, key, "not_applied", "")
			h.policy("DELETE FROM access.subject_grant WHERE principal_id=$1 AND action=$2", b.subject, strings.TrimSuffix(tc.action, ".maintain")+".read")
			b.call("POST", tc.receiptPath+key+"/resolve", nil, "", 403)
		})
	}
}

// Hold the first terminal decision open on one connection and observe the
// second connection waiting in PostgreSQL before committing the first.
func TestHire400ConcurrentRecovery(t *testing.T) {
	h := legalSetup206(t)
	cred := geoCredentials203(t, h.f, "admin", h.f.password)
	entity := h.create("C400", "MO", "branch")["id"].(string)
	policies, purposes := h.f.runtime.AssignmentPolicies, h.f.runtime.LegalPurposes
	for _, tc := range []struct {
		name          string
		save, resolve func(context.Context, string) (f.RuleReceipt, error)
	}{
		{"policy", func(ctx context.Context, key string) (f.RuleReceipt, error) {
			cmd := f.RuleChange[staffpub.AssignmentPolicy]{Row: staffpub.AssignmentPolicy{Period: f.Period{From: "2027-04-01"}, ID: f.NewID(), Purpose: "hire", Category: "Employee", Mode: "JobOnly"}}
			return policies.PublishPolicy(ctx, cred, key, cmd)
		}, func(ctx context.Context, key string) (f.RuleReceipt, error) {
			return policies.ResolvePolicy(ctx, cred, key)
		}},
		{"purpose", func(ctx context.Context, key string) (f.RuleReceipt, error) {
			cmd := f.RuleChange[org.LegalPurpose]{Row: org.LegalPurpose{Period: f.Period{From: "2027-04-01"}, ID: f.NewID(), Role: "employer", Kind: "declaration", Basis: "用人授权"}}
			return purposes.DeclarePurpose(ctx, cred, entity, key, cmd)
		}, func(ctx context.Context, key string) (f.RuleReceipt, error) {
			return purposes.ResolvePurpose(ctx, cred, key)
		}},
	} {
		for _, first := range []string{"save", "resolve"} {
			t.Run(tc.name+"/"+first, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(h.f.ctx, 8*time.Second)
				defer cancel()
				key := f.NewID()
				runFirst, runSecond := tc.save, tc.resolve
				if first == "resolve" {
					runFirst, runSecond = tc.resolve, tc.save
				}
				holding, e := h.f.runtime.Pool.Begin(ctx)
				must(t, e)
				defer holding.Rollback(h.f.ctx)
				waiting, e := h.f.runtime.Pool.Begin(ctx)
				must(t, e)
				defer waiting.Rollback(h.f.ctx)
				firstResult, e := runFirst(dbtx.With(ctx, holding), key)
				must(t, e)
				type completed struct {
					receipt f.RuleReceipt
					err     error
				}
				done := make(chan completed, 1)
				waitingPID := waiting.Conn().PgConn().PID()
				go func() {
					r, e := runSecond(dbtx.With(ctx, waiting), key)
					done <- completed{r, e}
				}()
				for h.f.scalar("SELECT cardinality(pg_blocking_pids($1))", waitingPID) == 0 {
					select {
					case r := <-done:
						t.Fatalf("second call ended before commit: %v", r)
					case <-ctx.Done():
						t.Fatal("second connection never waited for the first")
					case <-time.After(5 * time.Millisecond):
					}
				}
				must(t, holding.Commit(ctx))
				second := <-done
				if first == "save" {
					must(t, second.err)
					if firstResult.Outcome != "accepted" || second.receipt != firstResult {
						t.Fatalf("expected original publication: %+v %+v", firstResult, second)
					}
				} else {
					var failure *f.Fault
					if firstResult.Outcome != "not_applied" || !errors.As(second.err, &failure) || failure.Code != "RULE_SUBMISSION_CLOSED" {
						t.Fatalf("late save escaped the closure: %+v %+v", firstResult, second)
					}
				}
				must(t, waiting.Commit(ctx))
			})
		}
	}
	currentPolicy, e := policies.Policies(h.f.ctx, cred, "")
	must(t, e)
	currentPurpose, e := purposes.Purposes(h.f.ctx, cred, entity, "")
	must(t, e)
	if len(currentPolicy.History) != 1 || len(currentPurpose.History) != 1 {
		t.Fatal("expected one publication in each context", currentPolicy.History, currentPurpose.History)
	}
}

func TestHire400UpgradeFrom28(t *testing.T) {
	pool := newStaffingMigrationPool(t)
	ctx := context.Background()
	// Use the complete shipped 28 prefix with its real lineage-aware upgrader.
	prefix := hireMigrationSet(t, func(version string) bool { return version <= "00028" })
	must(t, bootstrap.Migrate(ctx, pool, prefix))
	id := f.NewID()
	must(t, bootstrap.Initialize(ctx, pool, bootstrap.InitInput{Instance: id, EnterpriseCode: "T400", EnterpriseName: "测试企业", Start: "1900-01-01", Login: "admin", Password: f.NewID()}))
	var original string
	must(t, pool.QueryRow(ctx, "SELECT head_source_id::text FROM organization.enterprise_profile WHERE instance_id=$1", id).Scan(&original))
	const m32 = "00032_hire_readiness.sql"
	b, e := migrations.Files.ReadFile(m32)
	must(t, e)
	prefix[m32] = &fstest.MapFile{Data: b}
	// Force a late failure after Organization has started its migration.
	_, e = pool.Exec(ctx, "CREATE TABLE staffing.assignment_policy_member (fixture boolean)")
	must(t, e)
	if e = bootstrap.Migrate(ctx, pool, prefix); e == nil {
		t.Fatal("expected failed migration")
	}
	var rolledBack bool
	must(t, pool.QueryRow(ctx, "SELECT to_regclass('organization.legal_purpose_source') IS NULL AND (SELECT max(version_id)=28 FROM public.goose_db_version)").Scan(&rolledBack))
	if !rolledBack {
		t.Fatal("partial readiness schema or ledger")
	}
	_, e = pool.Exec(ctx, "DROP TABLE staffing.assignment_policy_member")
	must(t, e)
	// Upgrade an actual 32 database with one previously published policy.
	must(t, bootstrap.Migrate(ctx, pool, prefix))
	if bootstrap.CheckCurrentStorage(ctx, pool) == nil {
		t.Fatal("runtime admitted schema before receipts")
	}
	var owner, subject string
	must(t, pool.QueryRow(ctx, `SELECT e.enterprise_id::text,p.principal_id::text FROM organization.enterprise_profile e
	JOIN access.principal p USING(instance_id) WHERE e.instance_id=$1 AND p.normalized_login='admin'`, id).Scan(&owner, &subject))
	version, key, row, rev := f.NewID(), f.NewID(), f.NewID(), f.NewID()
	tx, e := pool.Begin(ctx)
	must(t, e)
	defer tx.Rollback(ctx)
	_, e = tx.Exec(ctx, `INSERT INTO staffing.assignment_policy_source(instance_id,owner_id,version,subject_id,submission_id,input_digest) VALUES($1,$2,$3,$4,$5,decode(repeat('01',32),'hex'))`, id, owner, version, subject, key)
	must(t, e)
	_, e = tx.Exec(ctx, `INSERT INTO staffing.assignment_policy_member VALUES($1,$2,$3,$4,$5,daterange('2027-04-01',NULL,'[)'),jsonb_build_object('id',$4::uuid,'revision',$5::uuid,'category','Employee','purpose','hire','mode','JobOnly','from','2027-04-01'))`, id, owner, version, row, rev)
	must(t, e)
	_, e = tx.Exec(ctx, "INSERT INTO staffing.assignment_policy_outbox VALUES($1,$2,$3)", id, owner, version)
	must(t, e)
	must(t, tx.Commit(ctx))
	_, e = pool.Exec(ctx, "CREATE TABLE staffing.assignment_policy_terminal (fixture boolean)")
	must(t, e)
	const m33 = "00033_hire_rule_receipts.sql"
	b, e = migrations.Files.ReadFile(m33)
	must(t, e)
	prefix[m33] = &fstest.MapFile{Data: b}
	if e = bootstrap.Migrate(ctx, pool, prefix); e == nil {
		t.Fatal("expected receipt migration conflict")
	}
	must(t, pool.QueryRow(ctx, "SELECT to_regclass('organization.legal_purpose_terminal') IS NULL AND (SELECT max(version_id)=32 FROM public.goose_db_version)").Scan(&rolledBack))
	if !rolledBack {
		t.Fatal("partial receipt schema or ledger")
	}
	_, e = pool.Exec(ctx, "DROP TABLE staffing.assignment_policy_terminal")
	must(t, e)
	must(t, bootstrap.Migrate(ctx, pool, prefix))
	if bootstrap.CheckCurrentStorage(ctx, pool) == nil {
		t.Fatal("runtime admitted e ledger without Personnel")
	}
	// An interrupted fill must resume even though 32 and 33 already exist.
	_, e = pool.Exec(ctx, "CREATE TABLE access.person_profile_fixture (fixture boolean)")
	must(t, e)
	if bootstrap.Migrate(ctx, pool, migrations.Files) == nil {
		t.Fatal("expected profile migration conflict")
	}
	must(t, pool.QueryRow(ctx, `SELECT array_agg(version_id ORDER BY version_id)=ARRAY(SELECT generate_series(1,30)::bigint)||ARRAY[32,33]::bigint[]
		FROM goose_db_version WHERE version_id>0 AND is_applied`).Scan(&rolledBack))
	if !rolledBack || bootstrap.CheckCurrentStorage(ctx, pool) == nil {
		t.Fatal("partial convergence admitted or wrong retry prefix")
	}
	_, e = pool.Exec(ctx, "DROP TABLE access.person_profile_fixture")
	must(t, e)
	must(t, bootstrap.Migrate(ctx, pool, migrations.Files))
	must(t, bootstrap.Migrate(ctx, pool, migrations.Files))
	must(t, bootstrap.CheckCurrentStorage(ctx, pool))
	var preserved bool
	must(t, pool.QueryRow(ctx, `SELECT head_source_id=$2::uuid AND (SELECT count(*)=1 FROM public.goose_db_version WHERE version_id=32 AND is_applied)
 AND (SELECT count(*)=1 FROM public.goose_db_version WHERE version_id=33 AND is_applied)
 AND EXISTS(SELECT FROM staffing.assignment_policy_source s JOIN staffing.assignment_policy_member m USING(instance_id,owner_id,version)
 JOIN staffing.assignment_policy_outbox o USING(instance_id,owner_id,version)
 WHERE s.instance_id=$1 AND s.version=$3 AND s.submission_id=$4 AND m.revision=$5 AND m.fact->>'mode'='JobOnly')
 AND (SELECT count(*)=0 FROM staffing.assignment_policy_terminal)
 AND (SELECT count(*)=2 FROM access.subject_grant WHERE instance_id=$1 AND action IN ('assignmentpolicy.maintain','legalpurpose.maintain'))
 FROM organization.enterprise_profile WHERE instance_id=$1`, id, original, version, key, rev).Scan(&preserved))
	if !preserved {
		t.Fatal("original source, ledger or controlled grants changed")
	}
}

func hireMigrationSet(t *testing.T, include func(string) bool) fstest.MapFS {
	t.Helper()
	files := fstest.MapFS{}
	e := fs.WalkDir(migrations.Files, ".", func(path string, entry fs.DirEntry, e error) error {
		if e != nil || entry.IsDir() || (!strings.Contains(path, "/") && !include(path[:5])) {
			return e
		}
		b, e := migrations.Files.ReadFile(path)
		if e == nil {
			files[path] = &fstest.MapFile{Data: b}
		}
		return e
	})
	must(t, e)
	return files
}

func TestPublic39UpgradeToBankHealth(t *testing.T) {
	pool := newStaffingMigrationPool(t)
	ctx := t.Context()
	must(t, bootstrap.Migrate(ctx, pool, hireMigrationSet(t, func(v string) bool { return v <= "00039" })))
	if bootstrap.CheckCurrentStorage(ctx, pool) == nil {
		t.Fatal("runtime admitted the previous schema")
	}
	const decision = "49000000-0000-4000-8000-000000000039"
	_, err := pool.Exec(ctx, `insert into staffing.appointment_version (instance_id,source,version,edition,number,roster,withdrawn,members,digest,actor,recorded_at)
	 values ($1,$1,$1,1,'合成决定39',false,false,'[]','synthetic',$1,now())`, decision)
	must(t, err)
	_, err = pool.Exec(ctx, "insert into staffing.appointment_outbox values ($1,$1,$1)", decision)
	must(t, err)
	for range 2 {
		must(t, bootstrap.Migrate(ctx, pool, migrations.Files))
	}
	// The positive applied rows must contain every version exactly once.
	// CheckCurrentStorage below verifies the required final version and layout.
	var preserved bool
	must(t, pool.QueryRow(ctx, `select exists(select from staffing.appointment_version v join staffing.appointment_outbox o using(instance_id,source,version) where v.version=$1 and number='合成决定39')
	 and (select count(*)=max(version_id) and count(distinct version_id)=max(version_id) from goose_db_version where is_applied and version_id>0)`, decision).Scan(&preserved))
	if !preserved {
		t.Fatal("prior decision, event or continuous ledger changed")
	}
	must(t, bootstrap.CheckCurrentStorage(ctx, pool))
}

func TestPublic37UpgradeToEducationCredentials(t *testing.T) {
	pool := newStaffingMigrationPool(t)
	ctx := t.Context()
	must(t, bootstrap.Migrate(ctx, pool, hireMigrationSet(t, func(v string) bool { return v <= "00037" })))
	if bootstrap.CheckCurrentStorage(ctx, pool) == nil {
		t.Fatal("runtime admitted the public ledger before migration 38")
	}
	for range 2 {
		must(t, bootstrap.Migrate(ctx, pool, migrations.Files))
	}
	must(t, bootstrap.CheckCurrentStorage(ctx, pool))
}

func TestHire400UpgradeFromPersonnel34(t *testing.T) {
	for _, version := range []string{"00034", "00036", "00038"} {
		t.Run(version, func(t *testing.T) { checkPersonnelUpgrade(t, version) })
	}
}
func checkPersonnelUpgrade(t *testing.T, version string) {
	pool := newStaffingMigrationPool(t)
	ctx := t.Context()
	prior := hireMigrationSet(t, func(v string) bool {
		return v <= "00031" || v <= version && (v == "00034" || v == "00036" || v == "00038")
	})
	must(t, bootstrap.Migrate(ctx, pool, prior))
	if bootstrap.CheckCurrentStorage(ctx, pool) == nil {
		t.Fatal("runtime admitted f ledger without readiness rules")
	}
	// A fault in 33 must keep the original 34 row and the newly committed 32.
	var ledgerIDs []int64
	const originalRows = " FROM goose_db_version WHERE version_id IN (34,36,38) AND version_id <= $1::text::bigint"
	must(t, pool.QueryRow(ctx, "SELECT array_agg(id ORDER BY version_id)"+originalRows, version).Scan(&ledgerIDs))
	_, err := pool.Exec(ctx, "CREATE TABLE staffing.assignment_policy_terminal (fixture boolean)")
	must(t, err)
	if bootstrap.Migrate(ctx, pool, migrations.Files) == nil {
		t.Fatal("expected receipt migration conflict")
	}
	var intact bool
	must(t, pool.QueryRow(ctx, `SELECT array_agg(version_id ORDER BY version_id)=ARRAY(SELECT generate_series(1,32)::bigint)||ARRAY(SELECT v::bigint FROM unnest(ARRAY[34,36,38]) v WHERE v <= $1::text::bigint)
		AND to_regclass('organization.legal_purpose_terminal') IS NULL FROM goose_db_version WHERE version_id>0 AND is_applied`, version).Scan(&intact))
	if !intact || bootstrap.CheckCurrentStorage(ctx, pool) == nil {
		t.Fatal("partial f convergence admitted or wrong retry prefix")
	}
	_, err = pool.Exec(ctx, "DROP TABLE staffing.assignment_policy_terminal")
	must(t, err)
	for range 2 {
		must(t, bootstrap.Migrate(ctx, pool, migrations.Files))
	}
	must(t, bootstrap.CheckCurrentStorage(ctx, pool))
	must(t, pool.QueryRow(ctx, "SELECT array_agg(id ORDER BY version_id)=$2"+originalRows, version, ledgerIDs).Scan(&intact))
	if !intact {
		t.Fatal("an original personnel migration ledger row was replaced")
	}
}

// A consumer can verify source-lock ordering before any Employment exists.
// Only a persisted consumer adoption can later veto a narrowing (L3).
func TestHire400PurposeRecheckOrdering(t *testing.T) {
	h := legalSetup206(t)
	store := h.f.runtime.LegalPurposes
	cred := geoCredentials203(t, h.f, "admin", h.f.password)
	for _, first := range []string{"narrow", "recheck"} {
		t.Run(first, func(t *testing.T) {
			id := h.create("LOCK"+strings.ToUpper(first), "MO", "branch")["id"].(string)
			row := org.LegalPurpose{ID: f.NewID(), Role: "employer", Kind: "declaration", Basis: "首单来源复核", Period: f.Period{From: "2027-04-01"}}
			v, err := store.DeclarePurpose(h.f.ctx, cred, id, f.NewID(), f.RuleChange[org.LegalPurpose]{Row: row})
			must(t, err)
			input := org.PurposeCheck{Role: "employer", Period: row.Period, Expected: v.Version}
			preview, err := store.CheckPurpose(h.f.ctx, cred, id, input)
			must(t, err)
			if len(preview) != 1 {
				t.Fatal("missing original declaration")
			}
			input.Adopted = preview
			row.Revision = preview[0].Declaration
			row.Until = "2027-05-01"
			var edited f.RuleReceipt
			narrow := func(ctx context.Context) error {
				var err error
				edited, err = store.DeclarePurpose(ctx, cred, id, f.NewID(), f.RuleChange[org.LegalPurpose]{Expected: v.Version, Row: row})
				return err
			}
			recheck := func(ctx context.Context) error {
				got, err := store.CheckPurpose(ctx, cred, id, input)
				if err == nil && !reflect.DeepEqual(got, preview) {
					return errors.New("unexpected employer adoption")
				}
				return err
			}
			before, after := narrow, recheck
			if first == "recheck" {
				before, after = recheck, narrow
			}
			ctx, cancel := context.WithTimeout(h.f.ctx, 8*time.Second)
			defer cancel()
			holding, err := h.f.runtime.Pool.Begin(ctx)
			must(t, err)
			defer holding.Rollback(h.f.ctx)
			waiting, err := h.f.runtime.Pool.Begin(ctx)
			must(t, err)
			defer waiting.Rollback(h.f.ctx)
			must(t, before(dbtx.With(ctx, holding)))
			done := make(chan error, 1)
			pid := waiting.Conn().PgConn().PID()
			go func() { done <- after(dbtx.With(ctx, waiting)) }()
			for h.f.scalar("SELECT count(*) FROM unnest(pg_blocking_pids($1)) b WHERE b=$2", pid, holding.Conn().PgConn().PID()) == 0 {
				select {
				case err := <-done:
					t.Fatalf("second operation did not wait: %v", err)
				case <-ctx.Done():
					t.Fatal("no lock wait observed")
				case <-time.After(5 * time.Millisecond):
				}
			}
			must(t, holding.Commit(ctx))
			err = <-done
			if first == "narrow" {
				var fault *f.Fault
				if !errors.As(err, &fault) || fault.Status != 409 {
					t.Fatalf("stale preview accepted: %v", err)
				}
				must(t, waiting.Rollback(ctx))
			} else {
				must(t, err)
				must(t, waiting.Commit(ctx))
			}
			if edited.Outcome != "accepted" || edited.Version == v.Version {
				t.Fatal("narrowing did not publish a new version")
			}
			current, err := store.Purposes(ctx, cred, id, "")
			must(t, err)
			old, err := store.Purposes(ctx, cred, id, v.Version)
			must(t, err)
			if len(current.Rows) != 1 || current.Rows[0].Until != "2027-05-01" || len(old.Rows) != 1 || old.Rows[0].Until != "" || old.Rows[0].Revision != preview[0].Declaration {
				t.Fatal("incorrect current coverage or original source")
			}
		})
	}
}

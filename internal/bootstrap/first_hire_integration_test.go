//go:build integration

package bootstrap_test

import (
	"chnhcm.local/chnhcm/internal/bootstrap"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	f "chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
	"maps"
	"testing"
	"time"
)

func TestFirstHireAndPurposeProtection(t *testing.T) {
	type M = map[string]any
	h := legalSetup206(t)
	b := h.b
	init := bootstrap.InitInput{Instance: h.f.instance, EnterpriseCode: "LOCAL", EnterpriseName: "本地验证企业", Login: "admin", Password: h.f.password, PersonnelProfile: "registrar", PersonalProfileFixture: true, HireIdentityFixture: true, EmploymentFixture: true, FirstHireRule: &bootstrap.FirstHireRuleInput{Version: "T02-employee-1", From: "2027-04-01"}}
	must(t, bootstrap.Initialize(h.f.ctx, h.f.admin, init))
	cred := geoCredentials203(t, h.f, "admin", h.f.password)
	entity := h.create("HIRE37", "CN", "legal_person")["id"].(string)
	row := org.LegalPurpose{ID: f.NewID(), Role: "employer", Kind: "declaration", Basis: "用人单位资格", Period: f.Period{From: "2027-04-01"}}
	legal := h.f.runtime.LegalPurposes
	decl, err := legal.DeclarePurpose(h.f.ctx, cred, entity, f.NewID(), f.RuleChange[org.LegalPurpose]{Row: row})
	must(t, err)
	v, err := legal.Purposes(h.f.ctx, cred, entity, "")
	must(t, err)
	row = v.Rows[0]
	bu := b.call("POST", "/api/v1/business-units", createBody("BUH37", "2026-01-01"), f.NewID(), 201)["buId"].(string)
	dep := b.call("POST", "/api/v1/departments", departmentCreateBody("DH37", bu), f.NewID(), 201)["departmentId"].(string)
	staff := staffing307{t, h.f, b}
	family := staff.create("jobfamily", "FH37", "")["id"].(string)
	job := staff.create("job", "JH37", family)["id"].(string)
	b.call("POST", "/api/v1/assignment-policies", M{"expectedVersion": "", "row": M{"id": f.NewID(), "category": "Employee", "purpose": "hire", "mode": "JobOnly", "from": "2027-04-01"}}, f.NewID(), 200)
	p := b.call("POST", "/api/v1/persons", M{"submissionId": f.NewID(), "name": "首单样本"}, "", 200)
	person := p["id"].(string)
	personPath := "/api/v1/persons/" + person
	in := M{"person": person, "employer": entity, "department": dep, "job": job, "arrival": "2027-04-01", "notice": "人事部三月直接任用通知，四月一日到岗"}
	preview := func() M { return b.call("POST", "/api/v1/hires/preview", in, "", 200) }
	if v := preview(); len(v["missing"].([]any)) != 1 || v["adoption"] != nil {
		t.Fatal(v)
	}
	correct := func(changes []M) {
		p = b.call("POST", personPath+"/corrections", M{"submissionId": f.NewID(), "expectedSourceVersion": p["sourceVersion"], "documents": changes}, "", 200)
	}
	b.call("POST", "/api/v1/hires", in, f.NewID(), 409)
	correct([]M{{"fields": M{"documentKind": "passport", "documentNumber": "SYNTH-H37", "expiryDate": "2027-03-31"}}})
	chdoc := func(fields M) {
		doc := b.call("GET", personPath+"?view=edit", nil, "", 200)["documents"].([]any)[0].(M)
		correct([]M{{"itemId": doc["itemId"], "expectedRevision": doc["revision"], "fields": fields}})
	}
	zero := func(stage string) {
		t.Helper()
		for _, table := range []string{"employment", "assignment", "employment_primary", "employment_action", "employment_outbox", "assignment_segment"} {
			if n := h.f.scalar("SELECT count(*) FROM staffing." + table); n != 0 {
				t.Fatalf("%s: %s=%d", stage, table, n)
			}
		}
		t.Log(stage, "工作关系=0 任职=0 主选择=0 事件=0")
	}
	for _, field := range []string{"bu", "location"} {
		invalid := maps.Clone(in)
		invalid[field] = f.NewID()
		b.call("POST", "/api/v1/hires/preview", invalid, "", 409)
		b.call("POST", "/api/v1/hires", invalid, f.NewID(), 409)
		zero("无效拟任目标 " + field)
	}
	// A future BU is available only in its active interval. Scope the candidate
	// query to the employee and look up the adopted source by its dated name.
	futureInput := createBody("HFUT", "2027-04-01")
	futureInput["name"] = "四月试办单元"
	future := b.call("POST", "/api/v1/business-units", futureInput, f.NewID(), 201)["buId"].(string)
	for _, tc := range []struct {
		kind, day, name, id string
		count               int
	}{
		{"bu", "2027-03-31", "四月试办单元", future, 0}, {"bu", "2027-04-01", "四月试办单元", future, 1},
		{"job", "2027-04-01", "", job, 1}, {"department", "2027-04-01", "", dep, 1},
	} {
		opts := rows400(t, b, "/api/v1/hires/targets", M{"kind": tc.kind, "arrival": tc.day, "person": person, "page": 1, "search": tc.name})
		count := 0
		for _, opt := range opts {
			if opt["id"] == tc.id {
				count++
			}
		}
		if count != tc.count {
			t.Fatalf("kind=%s day=%s got %v", tc.kind, tc.day, opts)
		}
	}
	// Candidate filtering applies one grant to the known arrangement. An
	// unknown employer while choosing the department must not hide that department.
	h.policy("UPDATE access.employment_grant SET employer_id=$2,department_id=$3 WHERE principal_id=$1 AND action='hire'", b.subject, entity, dep)
	otherDep := b.call("POST", "/api/v1/departments", departmentCreateBody("DOTHER37", bu), f.NewID(), 201)["departmentId"].(string)
	scoped := rows400(t, b, "/api/v1/hires/targets", M{"person": person, "kind": "department", "arrival": "2027-04-01", "page": 1})
	if len(scoped) != 1 || scoped[0]["id"] != dep {
		t.Fatal("unexpected department choices", scoped)
	}
	outside := maps.Clone(in)
	outside["department"] = otherDep
	b.call("POST", "/api/v1/hires", outside, f.NewID(), 403)
	zero("拟任部门超出范围")
	in["bu"] = bu
	in["expected"] = preview()["adoption"]
	chdoc(M{"documentNumber": "SYNTH-H37-NEW"})
	b.call("POST", "/api/v1/hires", in, f.NewID(), 409)
	zero("换证")
	delete(in, "expected")
	in["expected"] = preview()["adoption"]
	h.policy("DELETE FROM access.person_grant WHERE principal_id=$1 AND action='hire.check'", b.subject)
	b.call("POST", "/api/v1/hires", in, f.NewID(), 404)
	zero("撤权")
	h.policy("INSERT INTO access.person_grant(instance_id,principal_id,action,purpose,fields) VALUES($1,$2,'hire.check','hire',array['name','document'])", h.f.instance, b.subject)
	// Hold the narrowed declaration uncommitted and observe Hire waiting on that exact connection.
	hold, err := h.f.runtime.Pool.Begin(h.f.ctx)
	must(t, err)
	defer hold.Rollback(h.f.ctx)
	narrow := row
	narrow.Until = "2027-05-01"
	narrowed, err := legal.DeclarePurpose(dbtx.With(h.f.ctx, hold), cred, entity, f.NewID(), f.RuleChange[org.LegalPurpose]{Expected: decl.Version, Row: narrow})
	must(t, err)
	type response struct {
		status int
		err    error
	}
	async := func(path string, body M, key string) <-chan response {
		ch := make(chan response, 1)
		go func() { status, _, err := b.response("POST", path, body, key); ch <- response{status, err} }()
		return ch
	}
	waitOn := func(pid uint32) {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			if h.f.scalar("SELECT count(*) FROM pg_stat_activity WHERE $1::int=any(pg_blocking_pids(pid))", pid) > 0 {
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
		t.Fatal("competing connection did not wait")
	}
	req := async("/api/v1/hires", in, f.NewID())
	waitOn(hold.Conn().PgConn().PID())
	must(t, hold.Commit(h.f.ctx))
	if got := <-req; got.err != nil || got.status != 409 {
		t.Fatal(got)
	}
	zero("先收窄用途")
	v, err = legal.Purposes(h.f.ctx, cred, entity, "")
	must(t, err)
	row = v.Rows[0]
	row.Until = ""
	decl, err = legal.DeclarePurpose(h.f.ctx, cred, entity, f.NewID(), f.RuleChange[org.LegalPurpose]{Expected: narrowed.Version, Row: row})
	must(t, err)
	delete(in, "expected")
	in["expected"] = preview()["adoption"]
	// A receipt failure happens after the business savepoint, including its Outbox, has succeeded.
	rollbackKey := f.NewID()
	_, err = h.f.admin.Exec(h.f.ctx, "ALTER TABLE staffing.hire_terminal ADD CONSTRAINT test_terminal_failure CHECK(outcome <> 'accepted') NOT VALID")
	must(t, err)
	b.call("POST", "/api/v1/hires", in, rollbackKey, 503)
	zero("提交回执落库失败")
	if h.f.scalar("SELECT count(*) FROM staffing.hire_terminal WHERE submission_id=$1", rollbackKey) != 0 {
		t.Fatal("terminal failure left a receipt")
	}
	_, err = h.f.admin.Exec(h.f.ctx, "ALTER TABLE staffing.hire_terminal DROP CONSTRAINT test_terminal_failure")
	must(t, err)
	recovered := b.call("POST", "/api/v1/hires/submissions/"+rollbackKey+"/resolve", nil, "", 200)
	if recovered["outcome"] != "not_applied" || recovered["action"] != nil {
		t.Fatal("incorrect resolution after terminal failure", recovered)
	}
	// Pause the real command at its last business insert, after its employment and choices exist.
	_, err = h.f.admin.Exec(h.f.ctx, `CREATE FUNCTION staffing.hold_test_hire() RETURNS trigger LANGUAGE plpgsql AS $$BEGIN PERFORM pg_advisory_xact_lock(373737);RETURN NEW;END$$;
	CREATE TRIGGER hold_test_hire BEFORE INSERT ON staffing.employment_outbox FOR EACH ROW EXECUTE FUNCTION staffing.hold_test_hire()`)
	must(t, err)
	hold, err = h.f.admin.Begin(h.f.ctx)
	must(t, err)
	defer hold.Rollback(h.f.ctx)
	_, err = hold.Exec(h.f.ctx, "select pg_advisory_xact_lock(373737)")
	must(t, err)
	key := f.NewID()
	req = async("/api/v1/hires", in, key)
	waitOn(hold.Conn().PgConn().PID())
	var writer uint32
	must(t, h.f.admin.QueryRow(h.f.ctx, "SELECT pid FROM pg_stat_activity WHERE $1::int=any(pg_blocking_pids(pid))", hold.Conn().PgConn().PID()).Scan(&writer))
	v, err = legal.Purposes(h.f.ctx, cred, entity, "")
	must(t, err)
	row = v.Rows[0]
	row.Until = "2027-05-01"
	edit := async(legal206Path+"/"+entity+"/purposes", M{"expectedVersion": decl.Version, "row": row}, f.NewID())
	waitOn(writer)
	must(t, hold.Commit(h.f.ctx))
	if got := <-req; got.err != nil || got.status != 200 {
		t.Fatal(got)
	}
	if got := <-edit; got.err != nil || got.status != 409 {
		t.Fatal(got)
	}
	resolve := func(key string) M { return b.call("POST", "/api/v1/hires/submissions/"+key+"/resolve", nil, "", 200) }
	receipt := resolve(key)
	if receipt["outcome"] != "accepted" || receipt["employment"] == nil || receipt["assignment"] == nil {
		t.Fatal(receipt)
	}
	if b.call("POST", "/api/v1/hires", in, key, 200)["action"] != receipt["action"] {
		t.Fatal("duplicate")
	}
	changed := maps.Clone(in)
	changed["notice"] = "另一说明"
	b.call("POST", "/api/v1/hires", changed, key, 409)
	b.call("POST", "/api/v1/hires", in, f.NewID(), 409)
	daily := func(day string, ppl float64) M {
		result := b.call("GET", personPath+"/employment?businessAt="+day, nil, "", 200)
		if result["people"] != ppl || result["assignments"] != ppl || result["seats"] != 0.0 {
			t.Fatal(result)
		}
		return result
	}
	daily("2027-03-31", 0)
	action := daily("2027-04-01", 1)["actions"].([]any)[0].(M)
	if n := h.f.scalar(`SELECT count(*) FROM staffing.assignment_segment WHERE assignment_id=$1 AND period @> date '2027-04-01'
        AND adoption->>'seats'='0' AND adoption->>'position'='' AND adoption->>'jobRevision'<>'' AND adoption->>'departmentRevision'<>''`, receipt["assignment"]); n != 1 {
		t.Fatal("missing adopted JobOnly segment", n)
	}
	if n := h.f.scalar("SELECT count(*) FROM staffing.employment_primary"); n != 2 {
		t.Fatal(n)
	}
	if n := h.f.scalar("SELECT count(*) FROM staffing.employment_action WHERE adoption->'sources'->'identity'->>'personVersion'=$1 AND adoption->'sources'->>'rule'='T02-employee-1'", p["sourceVersion"]); n != 1 {
		t.Fatal("missing immutable adoption")
	}
	b.call("POST", "/api/v1/employment-actions/"+action["id"].(string)+"/cancel", M{"revision": action["revision"], "reason": "任用通知于到岗前撤回"}, f.NewID(), 200)
	daily("2027-04-01", 0)
	if resolve(key)["action"] != receipt["action"] || b.call("POST", "/api/v1/hires", in, key, 200)["action"] != receipt["action"] {
		t.Fatal("lost original result")
	}
	if h.f.scalar("SELECT count(*) FROM staffing.employment_primary") != 0 || h.f.scalar("SELECT count(*) FROM staffing.employment_outbox") != 2 {
		t.Fatal("cancellation effects")
	}
	missing := f.NewID()
	if resolve(missing)["outcome"] != "not_applied" {
		t.Fatal("late submission not closed")
	}
	b.call("POST", "/api/v1/hires", in, missing, 409)
	path := "/api/v1/employment-actions/" + action["id"].(string)
	original := b.call("GET", path, nil, "", 200)
	identity := original["sources"].(M)["identity"].(M)
	if identity["personId"] != person || identity["personVersion"] != p["sourceVersion"] || action["canHistory"] != true {
		t.Fatal("incorrect original source or action permission")
	}
	if n := h.f.scalar(`SELECT count(*) FROM personnel.document_payload
		WHERE person_id=$1 AND version=$2 AND item_id=$3 AND revision=$4 AND attributes->>'expiryDate'='2027-03-31'`,
		person, identity["personVersion"], identity["documentId"], identity["documentRevision"]); n != 1 {
		t.Fatal("hire did not adopt the original document that expired before arrival")
	}
	h.policy("DELETE FROM access.employment_grant WHERE principal_id=$1 AND action='history'", b.subject)
	b.call("GET", path, nil, "", 403)
	if daily("2027-04-01", 0)["actions"].([]any)[0].(M)["canHistory"] != false {
		t.Fatal("history still offered after revocation")
	}
	h.policy("DELETE FROM access.employment_grant WHERE principal_id=$1 AND action='read'", b.subject)
	if got := resolve(key); got["action"] != nil || got["employment"] != nil || got["outcome"] != "accepted" {
		t.Fatal("receipt disclosed after revocation", got)
	}
	if len(daily("2027-04-01", 0)["actions"].([]any)) != 0 {
		t.Fatal("hidden actions disclosed")
	}
}

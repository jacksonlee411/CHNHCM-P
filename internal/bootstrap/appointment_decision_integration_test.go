//go:build integration

package bootstrap_test

import (
	"encoding/json"
	"maps"
	"reflect"
	"testing"
	"time"

	"chnhcm.local/chnhcm/db/migrations"
	"chnhcm.local/chnhcm/internal/bootstrap"
	f "chnhcm.local/chnhcm/internal/foundation"
)

func TestAppointmentDecisionFirstHire(t *testing.T) {
	type M = map[string]any
	h := legalSetup206(t)
	b := h.b
	init := bootstrap.InitInput{Instance: h.f.instance, EnterpriseCode: "LOCAL", EnterpriseName: "本地验证企业", Login: "admin", Password: h.f.password, PersonnelProfile: "registrar", PersonalProfileFixture: true, HireIdentityFixture: true, EmploymentFixture: true, FirstHireRule: &bootstrap.FirstHireRuleInput{Version: "T02-employee-1", From: "2027-04-01"}}
	must(t, bootstrap.Initialize(h.f.ctx, h.f.admin, init))
	init.Login = "second"
	init.Password = f.NewID()
	init.FirstHireRule = nil
	must(t, bootstrap.Initialize(h.f.ctx, h.f.admin, init))
	second := h.f.login(init.Login, init.Password)
	h.policy("DELETE FROM access.employment_grant WHERE principal_id=$1 AND action='appointment.issue'", second.subject)
	entity := h.create("AP39", "CN", "legal_person")["id"].(string)
	b.call("POST", "/api/v1/legal-entities/"+entity+"/purposes", M{"expectedVersion": "", "row": M{"id": f.NewID(), "role": "employer", "kind": "declaration", "basis": "任用试办", "from": "2027-04-01"}}, f.NewID(), 200)
	bu := b.call("POST", "/api/v1/business-units", createBody("BU39", "2026-01-01"), f.NewID(), 201)["buId"].(string)
	dep := b.call("POST", "/api/v1/departments", departmentCreateBody("D39", bu), f.NewID(), 201)["departmentId"].(string)
	staff := staffing307{t, h.f, b}
	family := staff.create("jobfamily", "F39", "")["id"].(string)
	job := staff.create("job", "J39", family)["id"].(string)
	b.call("POST", "/api/v1/assignment-policies", M{"expectedVersion": "", "row": M{"id": f.NewID(), "category": "Employee", "purpose": "hire", "mode": "JobOnly", "from": "2027-04-01"}}, f.NewID(), 200)
	person := func(name string) string {
		return b.call("POST", "/api/v1/persons", M{"submissionId": f.NewID(), "name": name, "primaryDocument": M{"kind": "passport", "number": "SYN-" + f.NewID()}}, "", 200)["id"].(string)
	}
	first, next, replacement := person("原单甲"), person("原单乙"), person("原行更换丙")
	rowA := M{"person": first, "line": f.NewID(), "employer": M{"id": entity, "name": "发文用人单位"}, "department": M{"id": dep, "name": "发文部门"}, "arrival": "2027-04-01"}
	rowB := M{"person": next, "line": f.NewID()}
	source := f.NewID()
	pub := M{"source": source, "number": "合成人事任〔2027〕01", "roster": true, "rows": []M{rowA, rowB}}
	publish := func() string {
		key := f.NewID()
		result := b.call("POST", "/api/v1/appointment-decisions", pub, key, 200)
		if b.call("POST", "/api/v1/appointment-decisions", pub, key, 200)["version"] != result["version"] {
			t.Fatal("publication replay")
		}
		pub["expected"] = result["version"]
		return key
	}
	second.call("POST", "/api/v1/appointment-decisions", pub, f.NewID(), 403)
	ver := publish()
	options := func(who *browser, id string) []M {
		res, e := who.client.Get(h.f.server.URL + "/api/v1/persons/" + id + "/appointment-decisions")
		must(t, e)
		defer res.Body.Close()
		if res.StatusCode != 200 {
			t.Fatal("decision list", res.StatusCode)
		}
		var v []M
		must(t, json.NewDecoder(res.Body).Decode(&v))
		return v
	}
	candidate := func(id string) M {
		found := options(b, id)
		if len(found) != 1 || found[0]["person"] != id {
			t.Fatal(found)
		}
		return found[0]
	}
	input := func(id string) M {
		return M{"person": id, "decision": candidate(id)["ref"], "employer": entity, "department": dep, "job": job, "arrival": "2027-04-01"}
	}
	preview := func(in M) {
		delete(in, "expected")
		in["expected"] = b.call("POST", "/api/v1/hires/preview", in, "", 200)["adoption"]
	}
	// A single-person decision uses the document itself, including when it
	// is withdrawn between preview and the first attempt to save.
	singleSource, singleVersion := f.NewID(), f.NewID()
	single := M{"source": singleSource, "number": "合成人事单39", "roster": false, "rows": []M{{"person": replacement, "bu": M{"id": bu, "name": "原单业务单元"}}}}
	b.call("POST", "/api/v1/appointment-decisions", single, singleVersion, 200)
	singleInput := input(replacement)
	singleInput["bu"] = bu
	preview(singleInput)
	if ref := singleInput["decision"].(M); ref["source"] != singleSource || ref["version"] != singleVersion || ref["line"] != nil || ref["lineRevision"] != nil {
		t.Fatal(ref)
	}
	mismatched := maps.Clone(singleInput)
	delete(mismatched, "bu")
	b.call("POST", "/api/v1/hires", mismatched, f.NewID(), 409)
	single["expected"] = singleVersion
	single["withdrawn"] = true
	single["rows"] = []M{}
	b.call("POST", "/api/v1/appointment-decisions", single, f.NewID(), 200)
	b.call("POST", "/api/v1/hires", singleInput, f.NewID(), 409)
	in := input(first)
	preview(in)
	conflict := maps.Clone(in)
	conflict["arrival"] = "2027-04-02"
	b.call("POST", "/api/v1/hires", conflict, f.NewID(), 409)
	rowA["job"] = M{"id": job, "name": "任用专员"}
	ver = publish()
	b.call("POST", "/api/v1/hires", in, f.NewID(), 409)
	in = input(first)
	preview(in)
	rowA["person"] = replacement
	publish()
	if len(options(b, first)) != 0 {
		t.Fatal("previous roster person still listed")
	}
	b.call("POST", "/api/v1/hires", in, f.NewID(), 409)
	rowA["person"] = first
	ver = publish()
	in = input(first)
	preview(in)
	secondPreview := maps.Clone(in)
	second.call("POST", "/api/v1/hires/preview", secondPreview, "", 403)
	// A hire preview checks both rules and reads the employer, department and job.
	h.policy(`INSERT INTO access.subject_grant
		SELECT instance_id,gen_random_uuid(),$1,scope_kind,scope_id,action,read_fields,write_fields
		FROM access.subject_grant WHERE principal_id=$2 AND action IN ('legalpurpose.check','assignmentpolicy.check','legalentity.read','department.read','department.history.read','job.read','job.history.read')`, second.subject, b.subject)
	secondPreview["expected"] = second.call("POST", "/api/v1/hires/preview", secondPreview, "", 200)["adoption"]
	h.policy("UPDATE access.employment_grant SET department_id=$2 WHERE principal_id=$1 AND action='hire'", second.subject, f.NewID())
	if len(options(second, first)) != 0 {
		t.Fatal("decision outside scope")
	}
	second.call("POST", "/api/v1/hires", secondPreview, f.NewID(), 403)
	h.policy("UPDATE access.employment_grant SET department_id=NULL WHERE principal_id=$1 AND action='hire'", second.subject)
	h.policy("DELETE FROM access.person_grant WHERE principal_id=$1 AND action='hire.check'", second.subject)
	second.call("GET", "/api/v1/persons/"+first+"/appointment-decisions", nil, "", 404)
	h.policy("INSERT INTO access.person_grant(instance_id,principal_id,action,purpose,fields) VALUES($1,$2,'hire.check','hire',array['name','document'])", h.f.instance, second.subject)
	rollback := f.NewID()
	h.f.exec("ALTER TABLE staffing.hire_terminal ADD CONSTRAINT test_rollback CHECK(outcome<>'accepted') NOT VALID")
	b.call("POST", "/api/v1/hires", in, rollback, 503)
	for _, table := range []string{"hire_decision_use", "employment", "assignment", "employment_primary", "employment_outbox", "employment_action", "assignment_segment"} {
		if n := h.f.scalar("SELECT count(*) FROM staffing." + table); n != 0 {
			t.Fatalf("rollback %s %d", table, n)
		}
	}
	h.f.exec("ALTER TABLE staffing.hire_terminal DROP CONSTRAINT test_rollback")
	resolve := func(who *browser, key string) M {
		return who.call("POST", "/api/v1/hires/submissions/"+key+"/resolve", nil, "", 200)
	}
	if resolve(b, rollback)["outcome"] != "not_applied" {
		t.Fatal("unexpected failed transaction receipt")
	}
	// Queue both API transactions at the first business write and the person lock,
	// respectively. Observe two distinct backend PIDs before releasing the writer.
	h.f.exec(`CREATE FUNCTION staffing.test_appointment_wait() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN PERFORM pg_advisory_xact_lock(393939); RETURN NEW; END $$;
 CREATE TRIGGER test_wait BEFORE INSERT ON staffing.employment FOR EACH ROW EXECUTE FUNCTION staffing.test_appointment_wait()`)
	gate, e := h.f.admin.Begin(h.f.ctx)
	must(t, e)
	defer gate.Rollback(h.f.ctx)
	_, e = gate.Exec(h.f.ctx, "SELECT pg_advisory_xact_lock(393939)")
	must(t, e)
	type reply struct {
		status int
		body   M
		err    error
	}
	start := func(who *browser, key string) <-chan reply {
		ch := make(chan reply, 1)
		go func() { s, v, e := who.response("POST", "/api/v1/hires", in, key); ch <- reply{s, v, e} }()
		return ch
	}
	waiter := func(pid uint32) uint32 {
		t.Helper()
		until := time.Now().Add(1800 * time.Millisecond)
		for time.Now().Before(until) {
			n := h.f.scalar("SELECT coalesce(max(pid),0) FROM pg_stat_activity WHERE $1::int=any(pg_blocking_pids(pid))", pid)
			if n != 0 {
				return uint32(n)
			}
			time.Sleep(5 * time.Millisecond)
		}
		t.Fatal("expected competing backend", pid)
		return 0
	}
	keyA, keyB := f.NewID(), f.NewID()
	a := start(b, keyA)
	writer := waiter(gate.Conn().PgConn().PID())
	other := start(second, keyB)
	contender := waiter(writer)
	t.Logf("two concurrent hire connections %d / %d", writer, contender)
	must(t, gate.Commit(h.f.ctx))
	winner, duplicate := <-a, <-other
	if winner.status != 200 || duplicate.status != 200 || winner.err != nil || duplicate.err != nil || winner.body["action"] != duplicate.body["action"] {
		t.Fatal(winner, duplicate)
	}
	for _, table := range []string{"employment", "assignment", "employment_action", "hire_decision_use", "employment_outbox"} {
		if h.f.scalar("SELECT count(*) FROM staffing."+table) != 1 {
			t.Fatal("duplicate", table)
		}
	}
	if h.f.scalar("SELECT count(*) FROM staffing.employment_primary") != 2 {
		t.Fatal("primary choices")
	}
	if resolve(second, keyA)["outcome"] != "not_applied" || resolve(b, keyB)["outcome"] != "not_applied" {
		t.Fatal("private receipt exposed")
	}
	if resolve(b, keyA)["action"] != winner.body["action"] || resolve(second, keyB)["action"] != winner.body["action"] {
		t.Fatal("original recovery")
	}
	checkDay := func(day string, n float64) {
		state := b.call("GET", "/api/v1/persons/"+first+"/employment?businessAt="+day, nil, "", 200)
		if state["people"] != n || state["assignments"] != n || state["seats"] != 0.0 {
			t.Fatal(state)
		}
	}
	checkDay("2027-03-31", 0)
	checkDay("2027-04-01", 1)
	if h.f.scalar("SELECT count(*) FROM staffing.assignment_segment WHERE adoption->>'position'='' AND adoption->>'seats'='0'") != 1 {
		t.Fatal("expected job only")
	}
	path := "/api/v1/employment-actions/" + winner.body["action"].(string)
	original := b.call("GET", path, nil, "", 200)
	if !reflect.DeepEqual(original["appointment"].(M)["ref"], in["decision"]) || original["appointment"].(M)["ref"].(M)["lineRevision"] != ver {
		t.Fatal(original)
	}
	b.call("POST", path+"/cancel", M{"revision": original["action"].(M)["revision"], "reason": "未来安排撤销"}, f.NewID(), 200)
	checkDay("2027-04-01", 0)
	if b.call("POST", "/api/v1/hires", in, keyA, 200)["action"] != winner.body["action"] {
		t.Fatal("replay changed action")
	}
	publish()
	if b.call("POST", "/api/v1/hires", input(first), f.NewID(), 200)["action"] != winner.body["action"] {
		t.Fatal("new revision bypassed original key")
	}
	checkDay("2027-04-01", 0)
	nextInput := input(next)
	preview(nextInput)
	second.call("POST", "/api/v1/hires", nextInput, f.NewID(), 200)
	if h.f.scalar("SELECT count(*) FROM staffing.hire_decision_use") != 2 {
		t.Fatal("independent second line")
	}
	h.policy("DELETE FROM access.employment_grant WHERE principal_id=$1 AND action='read'", second.subject)
	if r := resolve(second, keyB); r["outcome"] != "accepted" || r["action"] != nil {
		t.Fatal(r)
	}
	pub["withdrawn"] = true
	pub["rows"] = []M{}
	publish()
	b.call("POST", "/api/v1/hires", nextInput, f.NewID(), 409)
	if len(options(b, next)) != 0 || !reflect.DeepEqual(b.call("GET", path, nil, "", 200)["appointment"], original["appointment"]) {
		t.Fatal("withdrawal changed original")
	}
}

func TestPublic38UpgradeToAppointmentDecisions(t *testing.T) {
	pool := newStaffingMigrationPool(t)
	ctx := t.Context()
	must(t, bootstrap.Migrate(ctx, pool, hireMigrationSet(t, func(v string) bool { return v <= "00038" })))
	if bootstrap.CheckCurrentStorage(ctx, pool) == nil {
		t.Fatal("accepted 38 without decision storage")
	}
	for range 2 {
		must(t, bootstrap.Migrate(ctx, pool, migrations.Files))
	}
	must(t, bootstrap.CheckCurrentStorage(ctx, pool))
}

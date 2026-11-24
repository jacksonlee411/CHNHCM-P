//go:build integration

package integration_test

import (
	"chnhcm.local/chnhcm/db/migrations"
	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/foundation"
	"encoding/json"
	"fmt"
	"maps"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

const persons = "/api/v1/persons"

func personnelFixture(t *testing.T) *fixture {
	f := fresh(t)
	for _, a := range []struct {
		login, pw, profile string
		read               bool
	}{{"admin", f.password, "registrar", false}, {"reader", f.readPassword, "reader", true}} {
		must(t, bootstrap.Initialize(f.ctx, f.admin, bootstrap.InitInput{Instance: f.instance, EnterpriseCode: "LOCAL", EnterpriseName: "本地验证企业", Start: "2020-01-01", Login: a.login, Password: a.pw, ReadOnly: a.read, PersonnelProfile: a.profile, PersonalProfileFixture: true}))
	}
	return f
}
func TestPersonnelRegistration(t *testing.T) {
	f := personnelFixture(t)
	hr := f.login("admin", f.password)
	reader := f.login("reader", f.readPassword)
	type M = map[string]any
	reg := M{"submissionId": foundation.NewID(), "name": "样本青", "evidence": M{"kind": "form", "reference": "A-001"}, "contacts": []M{{"kind": "phone", "value": "+86 12345"}, {"kind": "email", "value": "Test@EXAMPLE.COM"}}}
	root := hr.call("POST", persons, reg, "", 200)
	id := root["id"].(string)
	version := root["sourceVersion"]
	path := persons + "/" + id
	if hr.call("POST", persons, reg, "", 200)["id"] != id {
		t.Fatal("replay")
	}
	resolve := persons + "/submissions/" + reg["submissionId"].(string) + "/resolve"
	if hr.call("POST", resolve, nil, "", 200)["id"] != id {
		t.Fatal("commit confirmation")
	}
	// Independent session after logout finds the original number.
	hr.call("DELETE", "/api/v1/session", nil, "", 204)
	hr = f.login("admin", f.password)
	lookup := func(b *browser, q string) M {
		ref := b.call("POST", persons+"/queries", M{"keyword": q}, "", 200)["ref"].(string)
		return b.call("GET", persons+"?ref="+ref, nil, "", 200)
	}
	list := lookup(hr, "P00000001")
	if list["total"] != 1.0 {
		t.Fatal("number lookup", list)
	}
	get := func(b *browser, path string) M { return b.call("GET", path, nil, "", 200) }
	first := get(hr, path)
	values := first["contacts"].([]any)
	phone := values[0].(M)
	email := values[1].(M)
	edit := M{"submissionId": foundation.NewID(), "expectedSourceVersion": version, "contacts": []M{{"itemId": phone["itemId"], "expectedRevision": phone["revision"], "kind": "phone", "value": "76543"}}}
	updated := hr.call("POST", path+"/corrections", edit, "", 200)
	differences := func(b *browser) []M {
		res, err := b.client.Get(f.server.URL + path + "/changes/" + updated["sourceVersion"].(string))
		must(t, err)
		defer res.Body.Close()
		if res.StatusCode != 200 {
			t.Fatalf("history difference status %d", res.StatusCode)
		}
		var rows []M
		must(t, json.NewDecoder(res.Body).Decode(&rows))
		return rows
	}
	if rows := differences(hr); len(rows) != 1 || rows[0]["field"] != "电话" || rows[0]["before"] != "+8612345" || rows[0]["after"] != "76543" {
		t.Fatal("projected changes", rows)
	}
	now := get(hr, path)["contacts"].([]any)
	if now[0].(M)["value"] != "76543" || !reflect.DeepEqual(now[1], email) {
		t.Fatal("independent contact")
	}
	if !reflect.DeepEqual(get(hr, path+"/sources/"+version.(string))["contacts"], values) {
		t.Fatal("original source")
	}
	edit["submissionId"] = foundation.NewID()
	hr.call("POST", path+"/corrections", edit, "", 409)
	reader.call("POST", path+"/corrections", edit, "", 403)
	reg["submissionId"] = foundation.NewID()
	if hr.call("POST", persons, reg, "", 200)["id"] != id {
		t.Fatal("business key")
	}
	reg["name"] = "另一人"
	reg["submissionId"] = foundation.NewID()
	hr.call("POST", persons, reg, "", 409)
	reg["name"] = "样本青"
	reg["evidence"] = M{"kind": "form", "reference": "A-002"}
	reg["submissionId"] = foundation.NewID()
	second := hr.call("POST", persons, reg, "", 200)
	if second["id"] == id {
		t.Fatal("two people with the same name received one file")
	}
	num := M{"submissionId": foundation.NewID(), "expectedSourceVersion": updated["sourceVersion"], "oldNumber": "P00000001", "newNumber": " r-01 ", "reason": "correction", "decisionRef": "DEC-1"}
	hr.call("POST", path+"/renumber", num, "", 200)
	row := lookup(hr, "P00000001")["rows"].([]any)[0].(M)
	if row["number"] != "R-01" || row["historical"] != true || row["id"] != id {
		t.Fatal("old number")
	}
	late := foundation.NewID()
	if hr.call("POST", persons+"/submissions/"+late+"/resolve", nil, "", 200)["outcome"] != "not_applied" {
		t.Fatal("negative receipt")
	}
	reg["submissionId"] = late
	hr.call("POST", persons, reg, "", 409)
	_, e := f.admin.Exec(f.ctx, "update access.person_grant set person_ids=array[$3::uuid],fields=array['name','number'] where instance_id=$1 and principal_id=$2", f.instance, reader.subject, id)
	must(t, e)
	if lookup(reader, "样本青")["total"] != 1.0 {
		t.Fatal("scoped count")
	}
	// A separate history grant for the phone alone cannot be joined to the name/number history grant.
	_, e = f.admin.Exec(f.ctx, "insert into access.person_grant(instance_id,principal_id,action,purpose,fields,person_ids) values($1,$2,'history.read','history',array['phone'],array[$3::uuid]),($1,$2,'read','directory',array['name','number','phone'],array[$3::uuid])", f.instance, reader.subject, id)
	must(t, e)
	if rows := differences(reader); len(rows) != 0 {
		t.Fatal("withheld change disclosed", rows)
	}
	if _, ok := get(reader, path+"/sources/"+version.(string))["contacts"]; ok {
		t.Fatal("history after revocation")
	}
	reader.call("GET", persons+"/"+second["id"].(string), nil, "", 404)
	if status, _, _ := reader.response("POST", persons+"/candidates", M{"name": "样本青"}, ""); status != 404 && status != 405 {
		t.Fatalf("retired route %d", status)
	}
	_, e = f.admin.Exec(f.ctx, "delete from access.person_grant where principal_id=$1", reader.subject)
	must(t, e)
	reader.call("GET", persons, nil, "", 403)
	reader.call("GET", path, nil, "", 404)
	_, e = f.admin.Exec(f.ctx, "delete from access.person_grant where principal_id=$1 and action<>'read'", hr.subject)
	must(t, e)
	if hr.call("POST", resolve, nil, "", 200)["id"] != id {
		t.Fatal("read-only confirmation")
	}
	_, e = f.admin.Exec(f.ctx, "delete from access.person_grant where principal_id=$1", hr.subject)
	must(t, e)
	if hr.call("POST", resolve, nil, "", 200)["id"] != nil {
		t.Fatal("receipt after losing read")
	}
	for table, want := range map[string]int{"person": 2, "number": 3, "source": 4, "outbox": 4} {
		var n int
		must(t, f.admin.QueryRow(f.ctx, fmt.Sprintf("select count(*) from personnel.%s", table)).Scan(&n))
		if n != want {
			t.Fatalf("%s %d want %d", table, n, want)
		}
	}
}

func TestPersonnelRetentionAndFieldAccess(t *testing.T) {
	type M = map[string]any
	f := personnelFixture(t)
	hr := f.login("admin", f.password)
	reader := f.login("reader", f.readPassword)
	key := foundation.NewID()
	reg := M{"submissionId": key, "name": "保留样本", "evidence": M{"kind": "personal", "reference": "T-1"}, "contacts": []M{{"kind": "phone", "value": "1234567"}, {"kind": "email", "value": "T@example.test"}}}
	result := hr.call("POST", persons, reg, "", 200)
	id, version := result["id"].(string), result["sourceVersion"].(string)
	path := persons + "/" + id
	source := path + "/sources/" + version
	exec := func(sql string, args ...any) { _, e := f.admin.Exec(f.ctx, sql, args...); must(t, e) }
	exec("update access.person_grant set fields=array['name','number','phone','email.masked'] where principal_id=$1", reader.subject)
	exec("insert into access.person_grant(instance_id,principal_id,action,purpose,fields) values($1,$2,'correct','maintenance',array['phone'])", f.instance, reader.subject)
	detail := reader.call("GET", source, nil, "", 200)
	contacts := detail["contacts"].([]any)
	if contacts[1].(M)["value"] != "T****" || contacts[1].(M)["masked"] != true {
		t.Fatal("mask")
	}
	x := contacts[0].(M)
	move := M{"submissionId": foundation.NewID(), "expectedSourceVersion": version, "contacts": []M{{"itemId": x["itemId"], "expectedRevision": x["revision"], "kind": "email", "value": "T@example.test"}}}
	reader.call("POST", path+"/corrections", move, "", 403)
	if hr.call("GET", path, nil, "", 200)["sourceVersion"] != version {
		t.Fatal("partial write")
	}
	exec("update personnel.person set ordinary_until=clock_timestamp()-interval '1 day',material_until=clock_timestamp()-interval '1 day' where person_id=$1", id)
	reader.call("GET", source, nil, "", 404)
	hr.call("GET", path, nil, "", 404)
	if hr.call("POST", persons+"/submissions/"+key+"/resolve", nil, "", 200)["id"] != nil {
		t.Fatal("expired receipt disclosed person")
	}
	assign := func(action, purpose string) {
		exec("insert into access.person_grant(instance_id,principal_id,action,purpose,fields,source_version,person_ids) values($1,$2,$3,$4,array['evidence'],$5,array[$6::uuid])", f.instance, hr.subject, action, purpose, version, id)
	}
	assign("evidence.read", "evidence")
	v := hr.call("GET", source, nil, "", 200)
	if v["evidence"] == nil || v["name"] != nil || v["contacts"] != nil {
		t.Fatal("evidence-only scope")
	}
	exec("update personnel.person set evidence_until=clock_timestamp()-interval '1 hour',material_evidence_until=clock_timestamp()-interval '1 hour' where person_id=$1", id)
	exec("update personnel.contact_payload set evidence_until=clock_timestamp()-interval '1 hour' where person_id=$1", id)
	exec("insert into personnel.hold values($1,$2,$3,'evidence',clock_timestamp()+interval '1 day','H-1')", f.instance, id, version)
	must(t, bootstrap.Migrate(f.ctx, f.admin, migrations.Files))
	must(t, bootstrap.Migrate(f.ctx, f.admin, migrations.Files))
	hr.call("GET", source, nil, "", 404)
	assign("hold.read", "preservation")
	v = hr.call("GET", source, nil, "", 200)
	if v["evidence"].(M)["reference"] != "T-1" || v["number"] != nil || v["name"] != nil || v["contacts"] != nil {
		t.Fatal("hold read escaped category")
	}
	count := func(table string) int {
		var n int
		must(t, f.admin.QueryRow(f.ctx, "select count(*) from personnel."+table).Scan(&n))
		return n
	}
	if count("identity_payload") != 0 || count("contact_payload") != 0 || count("evidence_payload") != 1 || count("disposition") != 3 {
		t.Fatal("expired payload disposition")
	}
	exec("delete from personnel.hold where person_id=$1", id)
	must(t, bootstrap.Migrate(f.ctx, f.admin, migrations.Files))
	if count("evidence_payload") != 0 || count("number") != 1 || count("material") != 1 || count("disposition") != 4 {
		t.Fatal("hold release or permanent reservations")
	}
	disposed := hr.call("GET", source, nil, "", 200)
	if disposed["disposed"] != true || disposed["evidence"] != nil || disposed["name"] != nil {
		t.Fatal("minimal disposition note")
	}
}

// Hold the same database guard as the command, then enqueue two independent
// HTTP connections in a known order before releasing it.
func TestPersonnelRaces(t *testing.T) {
	type M = map[string]any
	f := personnelFixture(t)
	a, b := f.login("admin", f.password), f.login("admin", f.password)
	type request struct {
		b    *browser
		path string
		body M
	}
	type result struct {
		status int
		value  M
		err    error
	}
	submit := func(in request) <-chan result {
		ch := make(chan result, 1)
		go func() { s, v, e := in.b.response("POST", in.path, in.body, ""); ch <- result{s, v, e} }()
		return ch
	}
	wait := func(n int) {
		limit := time.Now().Add(5 * time.Second)
		for time.Now().Before(limit) {
			var count int
			must(t, f.admin.QueryRow(f.ctx, "select count(*) from pg_stat_activity where datname=current_database() and wait_event_type='Lock'").Scan(&count))
			if count >= n {
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
		t.Fatal("two-connection barrier did not form")
	}
	race := func(key string, left, right request) (result, result) {
		tx, e := f.admin.Begin(f.ctx)
		must(t, e)
		defer tx.Rollback(f.ctx)
		_, e = tx.Exec(f.ctx, "select pg_advisory_xact_lock(hashtextextended($1,0))", f.instance+":"+key)
		must(t, e)
		first := submit(left)
		wait(1)
		second := submit(right)
		wait(2)
		must(t, tx.Commit(f.ctx))
		x, y := <-first, <-second
		must(t, x.err)
		must(t, y.err)
		return x, y
	}
	reg := func(name, ref string) M {
		return M{"submissionId": foundation.NewID(), "name": name, "evidence": M{"kind": "form", "reference": ref}, "contacts": []M{{"kind": "phone", "value": "12345"}}}
	}
	for dir := 0; dir < 2; dir++ {
		if dir == 1 {
			a, b = b, a
		}
		r := reg("同据"+fmt.Sprint(dir), fmt.Sprintf("KEY-%d", dir))
		other := maps.Clone(r)
		other["submissionId"] = foundation.NewID()
		x, y := race("personnel-writes", request{a, persons, r}, request{b, persons, other})
		if x.status != 200 || y.status != 200 || x.value["id"] != y.value["id"] {
			t.Fatal("same material", x, y)
		}
		onlyName := M{"name": "并发同名", "submissionId": foundation.NewID()}
		otherName := maps.Clone(onlyName)
		otherName["submissionId"] = foundation.NewID()
		x, y = race("personnel-writes", request{a, persons, onlyName}, request{b, persons, otherName})
		if x.status != 200 || y.status != 200 || x.value["id"] == y.value["id"] {
			t.Fatal("independent name-only registrations", x, y)
		}
		doc := M{"kind": "passport", "number": fmt.Sprintf("000X-%d", dir)}
		leftReg, rightReg := reg("王伟", fmt.Sprintf("X-%d", dir)), reg("王玮", fmt.Sprintf("Y-%d", dir))
		leftReg["primaryDocument"] = doc
		rightReg["primaryDocument"] = M{"kind": "other", "number": fmt.Sprintf("000x-%d", dir)}
		x, y = race("personnel-writes", request{a, persons, leftReg}, request{b, persons, rightReg})
		if x.status != 200 || y.status != 409 || y.value["existingPersonId"] != x.value["id"] {
			t.Fatal("same primary document", x, y)
		}
		first := x.value
		second := a.call("POST", persons, reg("独立人员"+fmt.Sprint(dir), fmt.Sprintf("Z-%d", dir)), "", 200)
		claim := func(v M, client *browser) request {
			body := M{"submissionId": foundation.NewID(), "expectedSourceVersion": v["sourceVersion"], "documents": []M{{"fields": M{"documentKind": "passport", "documentNumber": fmt.Sprintf("PROFILE-RACE-%d", dir)}, "primary": true}}}
			return request{client, persons + "/" + v["id"].(string) + "/corrections", body}
		}
		one, two := race("personnel-writes", claim(first, a), claim(second, b))
		if one.status != 200 || two.status != 409 || two.value["existingPersonId"] != nil {
			t.Fatal("whole-profile claim was not exclusive")
		}
		first["sourceVersion"] = one.value["sourceVersion"]
		target := fmt.Sprintf("CERT-RACE-%d", dir)
		editBody := M{"submissionId": foundation.NewID(), "expectedSourceVersion": second["sourceVersion"], "primaryDocument": M{"kind": "passport", "number": target}}
		registerBody := M{"submissionId": foundation.NewID(), "name": "另一名员工", "primaryDocument": M{"kind": "other", "number": target}}
		leftDoc, rightDoc := request{a, persons + "/" + second["id"].(string) + "/corrections", editBody}, request{b, persons, registerBody}
		if dir == 1 {
			leftDoc, rightDoc = rightDoc, leftDoc
		}
		x, y = race("personnel-writes", leftDoc, rightDoc)
		if x.status != 200 || y.status != 409 || f.scalar("select count(*) from personnel.document_payload where instance_id=$1 and occupied and number=$2", f.instance, target) != 1 {
			t.Fatal("mixed registration and document change race", x, y)
		}
		renumber := func(br *browser, record M) request {
			id := record["id"].(string)
			v := br.call("GET", persons+"/"+id, nil, "", 200)
			return request{br, persons + "/" + id + "/renumber", M{"submissionId": foundation.NewID(), "expectedSourceVersion": v["sourceVersion"], "oldNumber": v["number"], "newNumber": fmt.Sprintf("N-%d", dir), "reason": "migration", "decisionRef": "D-1"}}
		}
		x, y = race("personnel-writes", renumber(a, first), renumber(b, second))
		if x.status != 200 || y.status != 409 {
			t.Fatal("competing number", x, y)
		}
		path := persons + "/" + first["id"].(string)
		v := a.call("GET", path, nil, "", 200)
		item := v["contacts"].([]any)[0].(M)
		edit := func(br *browser, value string) request {
			return request{br, path + "/corrections", M{"submissionId": foundation.NewID(), "expectedSourceVersion": v["sourceVersion"], "contacts": []M{{"kind": "phone", "value": value, "itemId": item["itemId"], "expectedRevision": item["revision"]}}}}
		}
		x, y = race("personnel-writes", edit(a, "22222"), edit(b, "33333"))
		if x.status != 200 || y.status != 409 {
			t.Fatal("stale item", x, y)
		}
		cmd := reg("查回"+fmt.Sprint(dir), fmt.Sprintf("L-%d", dir))
		sub := cmd["submissionId"].(string)
		left, right := request{a, persons, cmd}, request{b, persons + "/submissions/" + sub + "/resolve", nil}
		if dir == 1 {
			left, right = right, left
		}
		x, y = race("submission:"+a.subject+":"+sub, left, right)
		if dir == 0 && (x.status != 200 || x.value["id"] != y.value["id"]) {
			t.Fatal("resolve after save", x, y)
		}
		if dir == 1 && (x.value["outcome"] != "not_applied" || y.status != 409) {
			t.Fatal("late save", x, y)
		}
	}
	// A forced failure after source allocation rolls back the number and allows
	// the caller's original submission to be conclusively resolved as unaccepted.
	_, e := f.admin.Exec(f.ctx, `create function personnel.fail_fixture() returns trigger language plpgsql as $$ begin raise exception 'fixture';end $$;
 create trigger inject before insert on personnel.outbox for each row execute function personnel.fail_fixture()`)
	must(t, e)
	before := func() string {
		var v string
		must(t, f.admin.QueryRow(f.ctx, "select concat_ws(',',(select count(*) from personnel.person),(select count(*) from personnel.source),(select count(*) from personnel.number),(select count(*) from personnel.terminal),(select last_number from personnel.counter))").Scan(&v))
		return v
	}
	previous := before()
	cmd := reg("回滚样本", "FAIL-1")
	a.call("POST", persons, cmd, "", 503)
	if before() != previous {
		t.Fatal("partial rollback")
	}
	if a.call("POST", persons+"/submissions/"+cmd["submissionId"].(string)+"/resolve", nil, "", 200)["outcome"] != "not_applied" {
		t.Fatal("failed transaction accepted")
	}
	_, e = f.admin.Exec(f.ctx, "drop trigger inject on personnel.outbox")
	must(t, e)
	// Revocation queued behind a command's authorization guard takes effect on
	// its next projection. Reversing the order denies the whole new command.
	tx, e := f.admin.Begin(f.ctx)
	must(t, e)
	defer tx.Rollback(f.ctx)
	_, e = tx.Exec(f.ctx, "select pg_advisory_xact_lock(hashtextextended($1,0))", f.instance+":personnel-writes")
	must(t, e)
	saved := submit(request{a, persons, reg("撤权前", "RIGHT-1")})
	wait(1)
	revoked := make(chan error, 1)
	go func() {
		_, e := f.admin.Exec(f.ctx, "delete from access.person_grant where principal_id=$1 and action='register'", a.subject)
		revoked <- e
	}()
	wait(2)
	must(t, tx.Commit(f.ctx))
	if (<-saved).status != 200 {
		t.Fatal("earlier command")
	}
	must(t, <-revoked)
	tx, e = f.admin.Begin(f.ctx)
	must(t, e)
	defer tx.Rollback(f.ctx)
	_, e = tx.Exec(f.ctx, "update access.policy_guard set policy_no=policy_no+1 where principal_id=$1", a.subject)
	must(t, e)
	late := submit(request{b, persons, reg("撤权后", "RIGHT-2")})
	wait(1)
	must(t, tx.Commit(f.ctx))
	if (<-late).status != 403 {
		t.Fatal("later command")
	}
}

func TestPersonnelCustomerIsolation(t *testing.T) {
	type M = map[string]any
	f := personnelFixture(t)
	other := foundation.NewID()
	must(t, bootstrap.Initialize(f.ctx, f.admin, bootstrap.InitInput{Instance: other, EnterpriseCode: "SECOND", EnterpriseName: "另一客户", Start: "2020-01-01", Login: "admin", Password: f.password, PersonnelProfile: "registrar"}))
	second, e := bootstrap.Open(f.ctx, f.runtime.Pool.Config().ConnString(), other)
	must(t, e)
	t.Cleanup(second.Close)
	server := httptest.NewUnstartedServer(nil)
	api := bootstrap.API{Runtime: second, Origin: "http://" + server.Listener.Addr().String(), InsecureLocal: true}
	server.Config.Handler = api.Handler()
	server.Start()
	t.Cleanup(server.Close)
	g := &fixture{t: t, server: server}
	a, b := f.login("admin", f.password), g.login("admin", f.password)
	reg := M{"submissionId": foundation.NewID(), "name": "同名客户样本", "primaryDocument": M{"kind": "passport", "number": "TEST-SHARED-CLIENTS"}, "evidence": M{"kind": "personal", "reference": "B-01"}}
	x, y := a.call("POST", persons, reg, "", 200), b.call("POST", persons, reg, "", 200)
	if x["id"] == y["id"] {
		t.Fatal("cross-customer binding")
	}
	for i, v := range []M{x, y} {
		br := []*browser{a, b}[i]
		detail := br.call("GET", persons+"/"+v["id"].(string), nil, "", 200)
		if detail["number"] != "P00000001" {
			t.Fatal("cross-customer allocation")
		}
		ref := br.call("POST", persons+"/queries", M{"keyword": "同名客户样本"}, "", 200)["ref"].(string)
		if br.call("GET", persons+"?ref="+ref, nil, "", 200)["total"] != 1.0 {
			t.Fatal("cross-customer search")
		}
		[]*browser{b, a}[i].call("GET", persons+"?ref="+ref, nil, "", 404)
	}
	a.call("GET", persons+"/"+y["id"].(string), nil, "", 404)
	b.call("GET", persons+"/"+x["id"].(string), nil, "", 404)
	// A query reference is session-bound while the submitting principal can
	// still retrieve the original result across logout and a new login.
	ref := a.call("POST", persons+"/queries", M{"keyword": "同名"}, "", 200)["ref"].(string)
	a.call("DELETE", "/api/v1/session", nil, "", 204)
	a = f.login("admin", f.password)
	a.call("GET", persons+"?ref="+ref, nil, "", 404)
	a.call("POST", persons+"/submissions/"+reg["submissionId"].(string)+"/resolve", nil, "", 200)
}

func TestPersonnelMaintenanceAndQueryLifetime(t *testing.T) {
	type M = map[string]any
	f := personnelFixture(t)
	a, reader := f.login("admin", f.password), f.login("reader", f.readPassword)
	original := M{"submissionId": foundation.NewID(), "name": "样本青", "evidence": M{"kind": "form", "reference": "A-001"}}
	p := a.call("POST", persons, original, "", 200)
	id := p["id"].(string)
	path := persons + "/" + id
	before := a.call("GET", path, nil, "", 200)
	if before["contacts"] != nil || before["number"] != "P00000001" {
		t.Fatal("first registration without contacts")
	}
	query := a.call("POST", persons+"/queries", M{"keyword": "样本青"}, "", 200)["ref"].(string)
	reader.call("GET", persons+"?ref="+query, nil, "", 404)
	a.call("DELETE", "/api/v1/session", nil, "", 204)
	a = f.login("admin", f.password)
	a.call("GET", persons+"?ref="+query, nil, "", 404)
	query = a.call("POST", persons+"/queries", M{"keyword": "样本青"}, "", 200)["ref"].(string)
	_, err := f.admin.Exec(f.ctx, "update personnel.query set expires_at=now() where ref=$1", query)
	must(t, err)
	a.call("GET", persons+"?ref="+query, nil, "", 404)
	edit := func(actor *browser, in M, status int) M {
		in["submissionId"] = foundation.NewID()
		in["expectedSourceVersion"] = a.call("GET", path, nil, "", 200)["sourceVersion"]
		return actor.call("POST", path+"/corrections", in, "", status)
	}
	edit(a, M{"contacts": []M{{"kind": "phone", "value": "12345"}, {"kind": "email", "value": "a@example.test"}}}, 200)
	v := a.call("GET", path, nil, "", 200)
	contacts := v["contacts"].([]any)
	_, err = f.admin.Exec(f.ctx, "update access.person_grant set fields=array['name','number'] where principal_id=$1", reader.subject)
	must(t, err)
	_, err = f.admin.Exec(f.ctx, "insert into access.person_grant(instance_id,principal_id,action,purpose,fields) values($1,$2,'correct','maintenance',array['name'])", f.instance, reader.subject)
	must(t, err)
	edit(reader, M{"name": "改后姓名"}, 200)
	if !reflect.DeepEqual(a.call("GET", path, nil, "", 200)["contacts"], contacts) {
		t.Fatal("unprojected items changed")
	}
	original["submissionId"] = foundation.NewID()
	if a.call("POST", persons, original, "", 200)["id"] != id {
		t.Fatal("original material after name correction")
	}
	email := contacts[1].(M)
	edit(a, M{"contacts": []M{{"itemId": email["itemId"], "expectedRevision": email["revision"], "remove": true}}}, 200)
	remaining := a.call("GET", path, nil, "", 200)["contacts"].([]any)
	if len(remaining) != 1 || !reflect.DeepEqual(remaining[0], contacts[0]) || !reflect.DeepEqual(a.call("GET", path+"/sources/"+v["sourceVersion"].(string), nil, "", 200)["contacts"], contacts) {
		t.Fatal("removed sibling or lost original")
	}
	current := a.call("GET", path, nil, "", 200)
	key := foundation.NewID()
	number := M{"submissionId": key, "expectedSourceVersion": current["sourceVersion"], "oldNumber": "P00000001", "newNumber": "R-01", "reason": "migration", "decisionRef": "D-1", "validUntil": "2001-01-01"}
	a.call("POST", path+"/renumber", number, "", 400)
	delete(number, "validUntil")
	number["submissionId"] = foundation.NewID()
	renumbered := a.call("POST", path+"/renumber", number, "", 200)
	if current := a.call("GET", path, nil, "", 200); current["decision"] != nil {
		t.Fatal("current profile exposed historical number chain")
	}
	decision := a.call("GET", path+"/sources/"+renumbered["sourceVersion"].(string), nil, "", 200)["decision"].(M)
	if decision["decisionRef"] != "D-1" || decision["oldNumber"] != "P00000001" || decision["submissionId"] != nil {
		t.Fatal("projected decision")
	}
	second := a.call("POST", persons, M{"submissionId": foundation.NewID(), "name": "乙样本", "evidence": M{"kind": "form", "reference": "B-002"}}, "", 200)
	number["oldNumber"] = "P00000002"
	number["newNumber"] = "P00000001"
	number["expectedSourceVersion"] = second["sourceVersion"]
	number["submissionId"] = foundation.NewID()
	a.call("POST", persons+"/"+second["id"].(string)+"/renumber", number, "", 409)
	number["newNumber"] = "OTHER"
	number["submissionId"] = foundation.NewID()
	number["effectiveDate"] = "2099-01-01"
	a.call("POST", persons+"/"+second["id"].(string)+"/renumber", number, "", 400)
}

func TestPersonnelNameOnlyRegistrationGrant(t *testing.T) {
	type M = map[string]any
	f := personnelFixture(t)
	a := f.login("admin", f.password)
	first := M{"submissionId": foundation.NewID(), "name": "核对样本", "evidence": M{"kind": "form", "reference": "VERIFY-1"}}
	a.call("POST", persons, first, "", 200)
	_, err := f.admin.Exec(f.ctx, "delete from access.person_grant where principal_id=$1 and action<>'register'", a.subject)
	must(t, err)
	entry := a.call("GET", persons+"/context", nil, "", 200)
	if entry["canRead"] != false || entry["canRegister"] != true {
		t.Fatal("register-only menu")
	}
	a.call("GET", persons, nil, "", 403)
	_, err = f.admin.Exec(f.ctx, "update access.person_grant set fields=array['name'] where principal_id=$1 and action='register'", a.subject)
	must(t, err)
	check := M{"name": "核对样本", "submissionId": foundation.NewID()}
	result := a.call("POST", persons, check, "", 200)
	if result["id"] != nil || result["outcome"] != "accepted" {
		t.Fatal("registration without ordinary read")
	}
}

func TestPersonnelPrimaryDocumentLifecycle(t *testing.T) {
	type M = map[string]any
	f := personnelFixture(t)
	hr := f.login("admin", f.password)
	reader := f.login("reader", f.readPassword)
	doc := func(number string) M { return M{"kind": "passport", "number": number} }
	register := func(name, number string) M {
		body := M{"submissionId": foundation.NewID(), "name": name}
		if number != "" {
			body["primaryDocument"] = doc(number)
		}
		return hr.call("POST", persons, body, "", 200)
	}
	first := register("王伟", "")
	id := first["id"].(string)
	path := persons + "/" + id
	correct := func(version any, value any, status int) M {
		return hr.call("POST", path+"/corrections", M{"submissionId": foundation.NewID(), "expectedSourceVersion": version, "primaryDocument": value}, "", status)
	}
	added := correct(first["sourceVersion"], doc(" 000abc "), 200)
	second := register("王伟", "000DEF")
	duplicate := correct(added["sourceVersion"], doc("000def"), 409)
	if duplicate["existingPersonId"] != second["id"] || duplicate["field"] != "documentNumber" {
		t.Fatal("conflict did not identify the permitted file")
	}
	assertDocument := func(b *browser, suffix, number string) {
		v := b.call("GET", path+suffix, nil, "", 200)
		got, ok := v["primaryDocument"].(M)
		if number == "" {
			if ok {
				t.Fatal("unexpected document")
			}
			return
		}
		if !ok || got["number"] != number {
			t.Fatal("document projection", suffix)
		}
	}
	assertDocument(hr, "?view=edit", "000ABC")
	assertDocument(hr, "", "********")
	assertDocument(reader, "?view=fullDocument", "********")
	assertDocument(hr, "/sources/"+first["sourceVersion"].(string), "")
	assertDocument(hr, "/sources/"+added["sourceVersion"].(string)+"?view=fullDocument", "000ABC")
	kept := correct(added["sourceVersion"], doc("000abc"), 200)
	if kept["outcome"] != "no_change" {
		t.Fatal("unchanged document made a revision")
	}
	renamed := hr.call("POST", path+"/corrections", M{"submissionId": foundation.NewID(), "expectedSourceVersion": added["sourceVersion"], "name": "王玮"}, "", 200)
	assertDocument(hr, "?view=edit", "000ABC")
	replaced := correct(renamed["sourceVersion"], doc("000ghi"), 200)
	rows, err := hr.client.Get(f.server.URL + path + "/changes/" + replaced["sourceVersion"].(string))
	must(t, err)
	var changes []M
	must(t, json.NewDecoder(rows.Body).Decode(&changes))
	rows.Body.Close()
	if len(changes) != 1 || changes[0]["field"] != "主要证件" || changes[0]["after"] != "已更换" {
		t.Fatal("missing redacted document change", changes)
	}
	assertDocument(hr, "?view=fullDocument", "000GHI")
	register("可用旧证号", "000abc")
	assertDocument(hr, "/sources/"+added["sourceVersion"].(string)+"?view=fullDocument", "000ABC")
	submitted := foundation.NewID()
	blocked := M{"submissionId": submitted, "name": "不同姓名", "primaryDocument": M{"kind": "other", "number": "000GHI"}}
	hr.call("POST", persons, blocked, "", 409)
	_, err = f.admin.Exec(f.ctx, "delete from access.person_grant where principal_id=$1 and action='verify'", hr.subject)
	must(t, err)
	hidden := hr.call("POST", persons, blocked, "", 409)
	receipt := hr.call("POST", persons+"/submissions/"+submitted+"/resolve", nil, "", 200)
	if hidden["existingPersonId"] != nil || hidden["message"] != "无法按此证件办理，请联系人事核对" || receipt["id"] != nil || receipt["existingPersonId"] != nil {
		t.Fatal("revoked conflict link disclosed")
	}
	cleared := correct(replaced["sourceVersion"], nil, 200)
	assertDocument(hr, "?view=edit", "")
	assertDocument(hr, "/sources/"+replaced["sourceVersion"].(string)+"?view=fullDocument", "000GHI")
	register("撤回后可登记", "000ghi")
	// An explicit null and an omitted field are different commands even with the same submission ID.
	clearKey := foundation.NewID()
	hr.call("POST", path+"/corrections", M{"submissionId": clearKey, "expectedSourceVersion": cleared["sourceVersion"], "primaryDocument": nil}, "", 200)
	hr.call("POST", path+"/corrections", M{"submissionId": clearKey, "expectedSourceVersion": cleared["sourceVersion"]}, "", 409)
	current := correct(cleared["sourceVersion"], doc("HELD"), 200)
	_, err = f.admin.Exec(f.ctx, "update personnel.person set ordinary_until=now()-interval '2 days',evidence_until=now()-interval '1 day' where person_id=$1", id)
	must(t, err)
	_, err = f.admin.Exec(f.ctx, "insert into personnel.hold values($1,$2,$3,'identity',now()+interval '1 day','test')", f.instance, id, current["sourceVersion"])
	must(t, err)
	must(t, bootstrap.DisposeExpiredPersonnel(f.ctx, f.admin))
	hr.call("POST", persons, M{"name": "受限档案", "submissionId": foundation.NewID(), "primaryDocument": doc("held")}, "", 409)
	if f.scalar("select count(*) from personnel.document_payload where person_id=$1 and occupied", id) != 1 {
		t.Fatal("disposing old source released held current number")
	}
	_, err = f.admin.Exec(f.ctx, "delete from personnel.hold where person_id=$1", id)
	must(t, err)
	must(t, bootstrap.DisposeExpiredPersonnel(f.ctx, f.admin))
	register("到期处置后", "HELD")
}

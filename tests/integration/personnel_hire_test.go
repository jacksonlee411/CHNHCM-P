//go:build integration

package integration_test

import (
	"errors"
	"net/url"
	"reflect"
	"testing"
	"time"

	"chnhcm.local/chnhcm/internal/bootstrap"
	person "chnhcm.local/chnhcm/internal/contexts/personnel/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
)

func TestHireIdentityArchiveAndGuards(t *testing.T) {
	f := personnelFixture(t)
	admin := f.login("admin", f.password)
	location, err := url.Parse(f.server.URL)
	must(t, err)
	cred := foundation.Credentials{Token: admin.client.Jar.Cookies(location)[0].Value, CSRF: admin.csrf, Write: true}
	type M = map[string]any
	archive := admin.call("POST", persons, M{"submissionId": foundation.NewID(), "name": "入职用途样本"}, "", 200)
	id := archive["id"].(string)
	in := person.HireIdentityRequest{PersonID: id, Purpose: "hire", Relationship: "Employee", Arrival: "2027-04-01", AcceptedDocuments: []string{"resident_id", "passport"}}
	check := func(want int) person.HireIdentityResult {
		t.Helper()
		result, err := f.runtime.Personnel.CheckIdentity(f.ctx, cred, in)
		var fault *foundation.Fault
		if want == 200 && err != nil || want != 200 && (!errors.As(err, &fault) || fault.Status != want) {
			t.Fatalf("want %d: %v", want, err)
		}
		return result
	}
	check(404)
	init := bootstrap.InitInput{Instance: f.instance, EnterpriseCode: "LOCAL", EnterpriseName: "本地验证企业", Start: "2020-01-01", Login: "admin", Password: f.password, HireIdentityFixture: true}
	must(t, bootstrap.Initialize(f.ctx, f.admin, init))
	if got := check(200); !reflect.DeepEqual(got.Missing, []string{"document"}) {
		t.Fatal(got)
	}
	exec := func(sql string, args ...any) { t.Helper(); _, e := f.admin.Exec(f.ctx, sql, args...); must(t, e) }
	exec("update access.person_grant set fields=array['name','document'] where action='hire.check'")
	check(404)
	exec("update access.person_grant set fields=array['name','document','missing.document','missing.expiryDate'],person_ids=array[$1::uuid] where action='hire.check'", id)
	in.PersonID = foundation.NewID()
	check(404)
	in.PersonID = id
	path := persons + "/" + id
	correct := func(docs []M) {
		t.Helper()
		archive = admin.call("POST", path+"/corrections", M{"submissionId": foundation.NewID(), "expectedSourceVersion": archive["sourceVersion"], "documents": docs}, "", 200)
	}
	correct([]M{{"primary": true, "fields": M{"documentKind": "other", "documentNumber": "SYN-HIRE-OTHER"}}, {"fields": M{"documentKind": "passport", "documentNumber": "SYN-HIRE-PASS", "expiryDate": "2027-03-31"}}})
	row := func() M { return admin.call("GET", path+"?view=edit", nil, "", 200)["documents"].([]any)[1].(M) }
	doc := row()
	if got := check(200); len(got.Missing) != 0 || got.Source.PersonID != id || got.Source.PersonVersion != archive["sourceVersion"] ||
		got.Source.DocumentID != doc["itemId"] || got.Source.DocumentRevision != doc["revision"] {
		t.Fatal("expired passport must identify its owner using the recorded revision", got)
	}
	change := func(fields M) {
		r := row()
		correct([]M{{"itemId": r["itemId"], "expectedRevision": r["revision"], "fields": fields}})
	}
	change(M{"expiryDate": ""})
	if check(200).Source.PersonID != id {
		t.Fatal("ordinary rule rejected unknown expiry")
	}
	in.RequireKnownExpiry = true
	if got := check(200); !reflect.DeepEqual(got.Missing, []string{"expiryDate"}) {
		t.Fatal(got)
	}
	r := row()
	correct([]M{{"itemId": r["itemId"], "expectedRevision": r["revision"], "longTerm": true}})
	if check(200).Source.PersonID != id {
		t.Fatal("explicit long-term document not accepted")
	}
	r = row()
	correct([]M{{"itemId": r["itemId"], "expectedRevision": r["revision"], "longTerm": false, "fields": M{"expiryDate": "2027-04-01"}}})
	in.RequireKnownExpiry = false
	source := check(200).Source
	if source.DocumentID != row()["itemId"] || source.PersonVersion != archive["sourceVersion"] || source.DocumentRevision != row()["revision"] {
		t.Fatal("incorrect adoption", source)
	}
	in.Expected = &source
	change(M{"documentNumber": "SYN-HIRE-RENEW"})
	tx, err := f.runtime.Pool.Begin(f.ctx)
	must(t, err)
	_, err = f.runtime.Personnel.CheckIdentity(dbtx.With(f.ctx, tx), cred, in)
	if err == nil {
		t.Fatal("renewal accepted stale preview")
	}
	must(t, tx.Rollback(f.ctx))
	in.Expected = nil
	source = check(200).Source
	in.Expected = &source
	tx, err = f.runtime.Pool.Begin(f.ctx)
	must(t, err)
	defer tx.Rollback(f.ctx)
	_, err = f.runtime.Personnel.CheckIdentity(dbtx.With(f.ctx, tx), cred, in)
	must(t, err)
	other, err := f.admin.Begin(f.ctx)
	must(t, err)
	defer other.Rollback(f.ctx)
	_, err = other.Exec(f.ctx, "SELECT set_config('lock_timeout','5s',true)")
	must(t, err)
	var available bool
	must(t, other.QueryRow(f.ctx, "select pg_try_advisory_xact_lock(hashtextextended($1,0))", f.instance+":personnel-writes").Scan(&available))
	if available {
		t.Fatal("correction could pass caller's live transaction")
	}
	done := make(chan error, 1)
	go func() {
		_, e := other.Exec(f.ctx, "delete from access.person_grant where action='hire.check'")
		if e == nil {
			e = other.Commit(f.ctx)
		}
		done <- e
	}()
	deadline := time.Now().Add(3 * time.Second)
	for {
		var blocked bool
		must(t, f.admin.QueryRow(f.ctx, "select $1::int=any(pg_blocking_pids($2::int))", tx.Conn().PgConn().PID(), other.Conn().PgConn().PID()).Scan(&blocked))
		if blocked {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("revocation did not wait for outer transaction")
		}
		time.Sleep(10 * time.Millisecond)
	}
	must(t, tx.Rollback(f.ctx))
	must(t, <-done)
	must(t, bootstrap.Initialize(f.ctx, f.admin, init))
	tx, err = f.runtime.Pool.Begin(f.ctx)
	must(t, err)
	defer tx.Rollback(f.ctx)
	_, err = f.runtime.Personnel.CheckIdentity(dbtx.With(f.ctx, tx), cred, in)
	var fault *foundation.Fault
	if !errors.As(err, &fault) || fault.Status != 404 {
		t.Fatal("revoked grant accepted", err)
	}
}

//go:build integration

package bootstrap_test

import (
	"errors"
	"testing"
	"time"

	"chnhcm.local/chnhcm/db/migrations"
	"chnhcm.local/chnhcm/internal/bootstrap"
	accessdb "chnhcm.local/chnhcm/internal/contexts/access/infra/postgres"
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestAccess027AssignmentOrderAndExpiry(t *testing.T) {
	f := fresh(t)
	admin, reader := f.login("admin", f.password), f.login("reader", f.readPassword)
	begin := func() pgx.Tx {
		tx, e := f.admin.Begin(f.ctx)
		must(t, e)
		t.Cleanup(func() { _ = tx.Rollback(f.ctx) })
		return tx
	}
	id := foundation.NewID
	var binding access.VersionBinding
	must(t, f.admin.QueryRow(f.ctx, `SELECT b.role_id,b.scope_id,b.fields_id FROM access.grant_revision b
 JOIN access.bound_permission p USING(instance_id,grant_id) WHERE p.principal_id=$1 AND p.action='bu.create'`, admin.subject).Scan(&binding.Role, &binding.Scope, &binding.Fields))
	change := func(c access.AssignmentChange) {
		t.Helper()
		tx := begin()
		_, err := accessdb.ChangeAssignment(f.ctx, tx, f.instance, c)
		must(t, err)
		must(t, tx.Commit(f.ctx))
	}
	issue := func(source string, until *time.Time) access.AssignmentChange {
		from := time.Now().Add(-time.Minute)
		c := access.AssignmentChange{ID: id(), Subject: reader.subject, Actor: admin.subject, RequestID: id(), Operation: "issue", Source: source, Purpose: "bu", Binding: binding, ValidFrom: &from, ValidUntil: until}
		change(c)
		return c
	}
	revoke := func(c access.AssignmentChange) access.AssignmentChange {
		return access.AssignmentChange{Operation: "revoke", ID: c.ID, Subject: c.Subject, Actor: admin.subject, RequestID: id(), ExpectedRevision: 1}
	}
	save := func(b *browser, code string, want int) {
		t.Helper()
		b.call("POST", "/api/v1/business-units", createBody(code, "2020-01-01"), id(), want)
	}
	checkEmpty := func(code, key string) {
		t.Helper()
		if f.scalar(`SELECT (SELECT count(*) FROM organization.business_unit WHERE code=$1)+(SELECT count(*) FROM organization.bu_terminal WHERE submission_id=$2)`, code, key) != 0 {
			t.Fatal("denial left a business object or receipt", code)
		}
	}
	waitBehind := func(blocker pgx.Tx) {
		t.Helper()
		limit := time.Now().Add(1500 * time.Millisecond)
		for time.Now().Before(limit) {
			if f.scalar(`SELECT count(*) FROM pg_stat_activity WHERE usename='chnhcm_api' AND $1=ANY(pg_blocking_pids(pid))`, int(blocker.Conn().PgConn().PID())) > 0 {
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
		t.Fatal("business connection did not wait for the expected blocker")
	}
	pending := func(code, key string) <-chan int {
		done := make(chan int, 1)
		go func() {
			status, _, err := reader.response("POST", "/api/v1/business-units", createBody(code, "2020-01-01"), key)
			if err != nil {
				status = 0
			}
			done <- status
		}()
		return done
	}
	own, proxy := issue("duty", nil), issue("proxy", nil)
	_, termErr := f.admin.Exec(f.ctx, "UPDATE access.grant_assignment SET purpose='department' WHERE grant_id=$1", own.ID)
	var constraint *pgconn.PgError
	if !errors.As(termErr, &constraint) || constraint.Code != "23514" {
		t.Fatal("assignment terms were not protected", termErr)
	}
	change(own) // Replay of the issue returns its original revision.
	if f.scalar("SELECT count(*) FROM access.grant_revision WHERE grant_id=$1", own.ID) != 1 {
		t.Fatal("duplicate issue")
	}
	save(reader, "X03_OVERLAP", 201)
	change(revoke(proxy))
	save(reader, "X03_DUTY", 201)
	tx := begin()
	_, err := accessdb.ChangeAssignment(f.ctx, tx, f.instance, revoke(own))
	must(t, err)
	key := id()
	done := pending("X03_REVOKED", key)
	waitBehind(tx)
	must(t, tx.Commit(f.ctx))
	if <-done != 403 {
		t.Fatal("revoke-first save was not denied")
	}
	checkEmpty("X03_REVOKED", key)

	// Wait on a real business table after authorization. Only the proxy remains.
	deadline := time.Now().Add(1150 * time.Millisecond)
	issue("proxy", &deadline)
	tx = begin()
	_, err = tx.Exec(f.ctx, "LOCK TABLE organization.bu_shape IN ACCESS EXCLUSIVE MODE")
	must(t, err)
	key = id()
	done = pending("X03_EXPIRED", key)
	waitBehind(tx)
	time.Sleep(time.Until(deadline.Add(10 * time.Millisecond)))
	must(t, tx.Commit(f.ctx))
	if <-done != 403 {
		t.Fatal("authorization survived expiry inside a business lock wait")
	}
	checkEmpty("X03_EXPIRED", key)
	save(reader, "X03_AFTER_EXPIRY", 403)

	// This role contains two live menu actions. The assignment's chosen purpose
	// determines which of them reaches the protected BU command.
	multi := access.PolicyVersion{Kind: "role", ID: id(), DefinitionID: id(), Number: 1, Actions: []string{"bu.create", "legalentity.create"}}
	tx = begin()
	must(t, accessdb.PublishVersion(f.ctx, tx, f.instance, admin.subject, id(), multi))
	must(t, tx.Commit(f.ctx))
	multiBinding := binding
	multiBinding.Role = multi.ID
	from := time.Now().Add(-time.Minute)
	purposed := access.AssignmentChange{Operation: "issue", ID: id(), RequestID: id(), Actor: admin.subject, Subject: reader.subject, Binding: multiBinding, Source: "duty", Purpose: "legalentity", ValidFrom: &from}
	change(purposed)
	save(reader, "X03_OTHER_PURPOSE", 403)
	purposed.ID, purposed.RequestID, purposed.Purpose = id(), id(), "bu"
	change(purposed)
	save(reader, "X03_BU_PURPOSE", 201)
	change(revoke(purposed))

	// Both holders start on one role. Publication alone cannot move a binding.
	adopter := issue("duty", nil)
	newRole := access.PolicyVersion{Kind: "role", ID: id(), DefinitionID: binding.Role, Number: 2, Actions: []string{"bu.create", "bu.export"}}
	tx = begin()
	must(t, accessdb.PublishVersion(f.ctx, tx, f.instance, admin.subject, id(), newRole))
	must(t, tx.Commit(f.ctx))
	if f.scalar("SELECT count(*) FROM access.bound_permission WHERE principal_id=$1 AND action='bu.export'", reader.subject) != 0 {
		t.Fatal("new publication changed an existing holder")
	}
	upgraded := binding
	upgraded.Role = newRole.ID
	change(access.AssignmentChange{ID: adopter.ID, Subject: reader.subject, Actor: admin.subject, RequestID: id(), Operation: "adopt", ExpectedRevision: 1, Binding: upgraded})
	if err := bootstrap.ChangeAccess(f.ctx, f.admin, bootstrap.AccessChange{Instance: f.instance, Login: "reader", Operation: "revoke", Action: "bu.export"}); err == nil {
		t.Fatal("action-only revoke would remove unrelated rights in a multi-action role")
	}
	tx = begin()
	must(t, accessdb.DisableVersion(f.ctx, tx, f.instance, admin.subject, id(), "role", binding.Role))
	must(t, tx.Commit(f.ctx))
	save(admin, "X03_OLD_ROLE", 403)
	save(reader, "X03_NEW_ROLE", 201)

	// Move the original business holder onto the same role and fields too.
	var firstGrant string
	must(t, f.admin.QueryRow(f.ctx, `SELECT a.grant_id FROM access.grant_assignment a JOIN access.grant_revision b USING(instance_id,grant_id,revision)
 WHERE a.principal_id=$1 AND b.role_id=$2`, admin.subject, binding.Role).Scan(&firstGrant))
	change(access.AssignmentChange{ID: firstGrant, Subject: admin.subject, Actor: admin.subject, RequestID: id(), Operation: "adopt", ExpectedRevision: 1, Binding: upgraded})
	save(admin, "X03_BOTH_SHARED", 201)
	// An exclusive field-version change precedes the business guard on another connection.
	tx = begin()
	must(t, accessdb.DisableVersion(f.ctx, tx, f.instance, admin.subject, id(), "fields", binding.Fields))
	key = id()
	done = pending("X03_FIELDS", key)
	waitBehind(tx)
	must(t, tx.Commit(f.ctx))
	if <-done != 403 {
		t.Fatal("disabled shared field version supplied a permission")
	}
	checkEmpty("X03_FIELDS", key)
	save(admin, "X03_SECOND_DISABLED", 403)
	if f.scalar("SELECT count(*) FROM access.grant_revision WHERE grant_id=$1", adopter.ID) != 2 {
		t.Fatal("binding revision lost")
	}

	must(t, bootstrap.ChangeAccess(f.ctx, f.admin, bootstrap.AccessChange{Instance: f.instance, Login: "reader", Operation: "revoke", GrantID: adopter.ID}))
	if f.scalar("SELECT count(*) FROM access.grant_revision WHERE grant_id=$1 AND operation='revoke'", adopter.ID) != 1 {
		t.Fatal("explicit controlled revocation lost its grant identity")
	}

	ordinary := bootstrap.InitInput{Instance: f.instance, EnterpriseCode: "LOCAL", EnterpriseName: "本地验证企业", Start: "2020-01-01", Login: "x03-ordinary", Password: id()}
	must(t, bootstrap.Initialize(f.ctx, f.admin, ordinary))
	newUser := f.login(ordinary.Login, ordinary.Password)
	if f.scalar("SELECT count(*) FROM access.grant_assignment WHERE principal_id=$1", newUser.subject) != 0 {
		t.Fatal("ordinary account received business permissions")
	}
	save(newUser, "X03_ZERO", 403)
}

func TestAccess027Public38Upgrade(t *testing.T) {
	pool := newStaffingMigrationPool(t)
	ctx := t.Context()
	must(t, bootstrap.Migrate(ctx, pool, hireMigrationSet(t, func(v string) bool { return v <= "00038" })))
	inst, password := foundation.NewID(), foundation.NewID()
	for _, login := range []string{"reader", "empty", "disabled"} {
		must(t, bootstrap.Initialize(ctx, pool, bootstrap.InitInput{Instance: inst, EnterpriseCode: "PUBLIC", EnterpriseName: "公共升级验收", Login: login, Password: password}))
	}
	_, err := pool.Exec(ctx, `UPDATE access.principal SET state='disabled' WHERE login='disabled';
		INSERT INTO access.subject_grant
		SELECT p.instance_id,gen_random_uuid(),p.principal_id,'enterprise',e.enterprise_id,action,fields,'{}'
		FROM access.principal p JOIN organization.enterprise e USING(instance_id)
		CROSS JOIN (VALUES ('bu.read',ARRAY['name']),('legalpurpose.check',ARRAY['rule']),
			('assignmentpolicy.read',ARRAY['rule'])) permissions(action,fields)
		WHERE p.login<>'empty';
		CREATE SCHEMA fixture;
		CREATE TABLE fixture.original AS SELECT * FROM access.subject_grant;
		CREATE VIEW access.bound_permission AS SELECT 1`)
	must(t, err)
	if err = bootstrap.Migrate(ctx, pool, migrations.Files); err == nil {
		t.Fatal("expected cutover to roll back on conflicting view")
	}
	var intact bool
	must(t, pool.QueryRow(ctx, `SELECT count(*)=6 AND to_regclass('access.grant_assignment') IS NULL AND
		NOT EXISTS(SELECT FROM goose_db_version WHERE version_id=40 AND is_applied) FROM access.subject_grant`).Scan(&intact))
	if !intact {
		t.Fatal("failed conversion changed old permissions or ledger")
	}
	_, err = pool.Exec(ctx, "DROP VIEW access.bound_permission")
	must(t, err)
	for range 2 {
		must(t, bootstrap.Migrate(ctx, pool, migrations.Files))
	}
	// A migrated ordinary login has no personnel or first-hire fixture marker.
	// Replaying the local startup declarations must preserve that zero.
	must(t, bootstrap.Initialize(ctx, pool, bootstrap.InitInput{
		Instance: inst, EnterpriseCode: "PUBLIC", EnterpriseName: "公共升级验收", Login: "empty", Password: password,
		NewAccountFixturesOnly: true, DevelopmentFixture: true, PersonnelProfile: "registrar",
		PersonalProfileFixture: true, HireIdentityFixture: true, EmploymentFixture: true,
	}))
	must(t, pool.QueryRow(ctx, `SELECT NOT EXISTS(SELECT FROM access.person_grant)
		AND NOT EXISTS(SELECT FROM access.employment_grant)
		AND NOT EXISTS(SELECT FROM access.person_fixture)
		AND NOT EXISTS(SELECT FROM access.employment_fixture)`).Scan(&intact))
	if !intact {
		t.Fatal("restart issued new grants to an existing ordinary account")
	}
	must(t, bootstrap.CheckCurrentStorage(ctx, pool))
	must(t, pool.QueryRow(ctx, `WITH actual AS (
		SELECT instance_id,grant_id,principal_id,scope_kind,scope_id,action,read_fields,write_fields FROM access.bound_permission
	), delta AS (
		(SELECT * FROM fixture.original EXCEPT ALL SELECT * FROM actual) UNION ALL
		(SELECT * FROM actual EXCEPT ALL SELECT * FROM fixture.original)
	) SELECT NOT EXISTS(TABLE delta) AND NOT EXISTS(SELECT FROM access.grant_assignment
		WHERE source<>'migration' OR valid_from IS NOT NULL OR valid_until IS NOT NULL)
		AND NOT EXISTS(SELECT FROM access.grant_revision WHERE actor_id IS NOT NULL OR recorded_at IS NOT NULL)
		AND NOT access.valid_permission($1,'bu',$1,'legalpurpose.check',ARRAY['rule'],'{}')`, inst).Scan(&intact))
	if !intact {
		t.Fatal("cutover changed actual rights, attributed unknown history or lost readiness scope")
	}
}

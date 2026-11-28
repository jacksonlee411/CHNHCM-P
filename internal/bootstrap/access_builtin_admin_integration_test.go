//go:build integration

package bootstrap_test

import (
	"slices"
	"testing"

	"chnhcm.local/chnhcm/db/migrations"
	"chnhcm.local/chnhcm/internal/bootstrap"
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

func TestBuiltinAdminKeepsManagementAcrossAppointmentAndRecovery(t *testing.T) {
	f := fresh(t)
	admin := f.login("admin", f.password)
	m := &managementFixture{fixture: f, root: admin}
	rights := []string{"account", "credential", "role", "policy", "assign", "delegate"}
	check := func(who *browser) {
		t.Helper()
		if got := viewMgmt(t, who, "context").Rights; !slices.Equal(got, rights) {
			t.Fatalf("management rights %v, want %v", got, rights)
		}
		for _, area := range []string{"accounts", "roles", "grants"} {
			who.call("GET", "/api/v1/access/"+area, nil, "", 200)
		}
	}
	check(admin)
	self := viewMgmt(t, admin, "grants?subject="+admin.subject)
	if len(self.Accounts) != 1 || !self.Accounts[0].BuiltinSecurityAdmin || len(self.Accounts[0].Options) != 0 {
		t.Fatal("built-in authority missing or offered for self-edit", self.Accounts)
	}
	reader := f.login("reader", f.readPassword)
	reader.call("GET", "/api/v1/access/accounts", nil, "", 403)
	id := foundation.NewID
	m.cmd(admin, access.ManagementCommand{Operation: "account.create", Owner: f.instance, Login: " Admİn ", Password: id()}, 400)
	create := access.ManagementCommand{Request: id(), Operation: "account.create", Owner: f.instance, Login: "clerk", Password: id()}
	result := m.cmd(admin, create, 200)
	subject := result["target"].(string)
	rev := viewMgmt(t, admin, "accounts?subject="+subject).Accounts[0].Revision
	m.cmd(admin, access.ManagementCommand{Operation: "account.reset", Subject: subject, Revision: rev, Password: id()}, 200)
	var enterprise string
	for _, target := range viewMgmt(t, admin, "roles").Targets {
		if target.Kind == "enterprise" {
			enterprise = target.ID
			break
		}
	}
	role := m.cmd(admin, access.ManagementCommand{Operation: "version.publish",
		Version:   &access.Definition{Kind: "role", Name: "部门名录", Actions: []string{"department.read"}},
		NewScope:  &access.Definition{Kind: "scope", Name: "企业范围", TargetKind: "enterprise", TargetID: enterprise},
		NewFields: &access.Definition{Kind: "fields", Name: "部门名称", Read: []string{"code", "name"}}}, 200)["target"].(string)
	defs := viewMgmt(t, admin, "roles?object="+role).Versions
	i := slices.IndexFunc(defs, func(d access.Definition) bool { return d.ID == role })
	if i < 0 {
		t.Fatal("published role unavailable")
	}
	d := defs[i]
	m.cmd(admin, access.ManagementCommand{Operation: "grant.issue", Subject: subject, Role: role, Scope: d.Scope, Fields: d.Fields, Purpose: "department", Source: "duty"}, 200)
	m.cmd(admin, access.ManagementCommand{Operation: "mandate.issue", Subject: subject, Limit: &access.ManagementLimit{Owner: f.instance, Rights: []string{"account"}, Role: role, Scope: d.Scope, Fields: d.Fields}}, 200)
	// A fresh built-in administrator can write before the first explicit customer appointment.
	in := access.RecoveryCommand{Instance: f.instance, Request: id(), Operation: "bootstrap", Login: "security", Password: id(), Operator: "部署运维", Reason: "独立客户管理员", MaxProxyDays: new(7)}
	current, err := bootstrap.RecoverManagement(f.ctx, f.admin, f.instance, in)
	must(t, err)
	check(admin)
	if replay := m.cmd(admin, create, 200); replay["at"] != result["at"] {
		t.Fatal("appointment changed the receipt key")
	}
	other := f.login(in.Login, in.Password)
	check(other)
	m.cmd(other, access.ManagementCommand{Operation: "account.reset", Subject: admin.subject, Revision: 1, Password: id()}, 403)
	in.Operation, in.ExpectedAdmin, in.Request, in.Login, in.Password = "handover", current.Target, id(), "successor", id()
	next, err := bootstrap.RecoverManagement(f.ctx, f.admin, f.instance, in)
	must(t, err)
	other.call("GET", "/api/v1/session", nil, "", 401)
	successorPassword := in.Password
	successor := f.login(in.Login, successorPassword)
	check(successor)
	check(admin)
	in.Operation, in.ExpectedAdmin, in.Request, in.Login, in.Password = "recover", admin.subject, id(), "admin", id()
	recovered, err := bootstrap.RecoverManagement(f.ctx, f.admin, f.instance, in)
	must(t, err)
	if recovered.Target != admin.subject || next.Target != successor.subject {
		t.Fatal("recovery changed principal identities")
	}
	admin.call("GET", "/api/v1/session", nil, "", 401)
	check(f.login("ADMIN", in.Password))
	check(successor)
	if f.scalar("SELECT count(*) FROM access.management_root WHERE principal_id=$1 AND max_proxy_days=7", next.Target) != 1 {
		t.Fatal("recovering admin replaced the customer's appointment")
	}
	// A handover away from admin ends its old session and appointment, while
	// its built-in rights survive even the subsequent revocation of a business grant.
	adminPassword := in.Password
	in.Operation, in.ExpectedAdmin, in.Request, in.Login, in.Password = "handover", next.Target, id(), "admin", ""
	_, err = bootstrap.RecoverManagement(f.ctx, f.admin, f.instance, in)
	must(t, err)
	oldSession := f.login("admin", adminPassword)
	in.ExpectedAdmin, in.Request, in.Login = admin.subject, id(), "successor"
	_, err = bootstrap.RecoverManagement(f.ctx, f.admin, f.instance, in)
	must(t, err)
	oldSession.call("GET", "/api/v1/session", nil, "", 401)
	successor = f.login("successor", successorPassword)
	grant := m.cmd(successor, access.ManagementCommand{Operation: "grant.issue", Subject: admin.subject, Role: role, Scope: d.Scope, Fields: d.Fields, Purpose: "department", Source: "duty"}, 200)
	m.cmd(successor, access.ManagementCommand{Operation: "grant.revoke", Subject: admin.subject, ID: grant["target"].(string), Revision: 1}, 200)
	check(f.login("admin", adminPassword))
	reader.call("GET", "/api/v1/access/grants", nil, "", 403)
}

func TestBuiltinAdminUpgradeRequiresDeploymentIdentity(t *testing.T) {
	ctx, pool := t.Context(), newStaffingMigrationPool(t)
	must(t, bootstrap.Migrate(ctx, pool, hireMigrationSet(t, func(n string) bool { return n <= "00058" })))
	init := bootstrap.InitInput{Instance: foundation.NewID(), EnterpriseCode: "LOCAL", EnterpriseName: "旧库", Login: "Admin", Password: foundation.NewID(), ReadOnly: true}
	must(t, bootstrap.Initialize(ctx, pool, init))
	must(t, bootstrap.Migrate(ctx, pool, migrations.Files))
	var principal string
	must(t, pool.QueryRow(ctx, `SELECT m.principal_id::text FROM access.management_root m JOIN access.principal p USING(instance_id,principal_id)
 WHERE m.instance_id=$1 AND p.normalized_login='admin'`, init.Instance).Scan(&principal))
	var sealBefore, sealAfter string
	readKeyDigest := func(out *string) {
		t.Helper()
		must(t, pool.QueryRow(ctx, "SELECT encode(sha256(receipt_key),'hex') FROM access.management_root WHERE instance_id=$1", init.Instance).Scan(out))
	}
	readKeyDigest(&sealBefore)
	must(t, bootstrap.Migrate(ctx, pool, migrations.Files))
	must(t, bootstrap.Initialize(ctx, pool, init))
	readKeyDigest(&sealAfter)
	if sealBefore != sealAfter {
		t.Fatal("repeat upgrade or initialization changed the management receipt key")
	}
	var business int
	must(t, pool.QueryRow(ctx, "SELECT count(*) FROM access.grant_assignment WHERE instance_id=$1 AND principal_id=$2", init.Instance, principal).Scan(&business))
	if business != 0 {
		t.Fatal("management initialization granted business access")
	}

	// Reproduce the exact identity declaration used by the previous managed-account
	// creator, in an instance with no deployed admin. Its display/login is insufficient.
	legacy := init
	legacy.Instance, legacy.Login = foundation.NewID(), "operator"
	must(t, bootstrap.Initialize(ctx, pool, legacy))
	accountID := foundation.NewID()
	_, err := pool.Exec(ctx, `INSERT INTO access.principal(instance_id,principal_id,login,normalized_login,state,management_owner,initialization_digest)
 VALUES($1,$2,'Admin','admin','active',$1,sha256(convert_to($2::uuid::text,'UTF8')))`, legacy.Instance, accountID)
	must(t, err)
	must(t, bootstrap.Migrate(ctx, pool, migrations.Files))
	var bound bool
	must(t, pool.QueryRow(ctx, "SELECT EXISTS(SELECT FROM access.management_root WHERE instance_id=$1)", legacy.Instance).Scan(&bound))
	if bound {
		t.Fatal("ordinary legacy account received built-in management")
	}
}

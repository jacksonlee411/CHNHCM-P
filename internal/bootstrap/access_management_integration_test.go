//go:build integration

package bootstrap_test

import (
	"bytes"
	"context"
	"encoding/json"
	"maps"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"chnhcm.local/chnhcm/db/migrations"
	"chnhcm.local/chnhcm/internal/bootstrap"
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

type managementFixture struct {
	*fixture
	root     *browser
	password string
}

func TestAccessManagementUpgradeRetainsRevokedAssignments(t *testing.T) {
	ctx, pool := t.Context(), newStaffingMigrationPool(t)
	must(t, bootstrap.Migrate(ctx, pool, hireMigrationSet(t, func(n string) bool { return n <= "00058" })))
	init := bootstrap.InitInput{Instance: foundation.NewID(), EnterpriseCode: "TEST", EnterpriseName: "升级对账", Login: "old-reader", Password: foundation.NewID(), ReadOnly: true, DevelopmentFixture: true}
	must(t, bootstrap.Initialize(ctx, pool, init))
	must(t, bootstrap.ChangeAccess(ctx, pool, bootstrap.AccessChange{Instance: init.Instance, Login: init.Login, Operation: "revoke", Action: "department.read"}))
	snapshot := func() map[string]string {
		t.Helper()
		out := map[string]string{}
		for _, table := range []string{"principal", "credential", "policy_guard", "grant_assignment", "grant_revision", "policy_version", "security_event", "bound_permission"} {
			var sum string
			// The complete set and all pre-existing columns are compared inside PostgreSQL.
			must(t, pool.QueryRow(ctx, "SELECT md5(coalesce(jsonb_agg(row ORDER BY row)::text,'[]')) FROM (SELECT to_jsonb(t)-'management_owner' AS row FROM access."+table+" t) s").Scan(&sum))
			out[table] = sum
		}
		return out
	}
	before := snapshot()
	_, err := pool.Exec(ctx, "CREATE VIEW access.management_publication AS SELECT true")
	must(t, err)
	if bootstrap.Migrate(ctx, pool, migrations.Files) == nil {
		t.Fatal("expected mid-migration conflict")
	}
	var intact bool
	must(t, pool.QueryRow(ctx, `SELECT to_regclass('access.management_root') IS NULL
	 AND NOT EXISTS(SELECT FROM pg_attribute WHERE attrelid='access.principal'::regclass AND attname='management_owner')
	 AND NOT EXISTS(SELECT FROM goose_db_version WHERE version_id>58 AND is_applied)`).Scan(&intact))
	if !intact || !maps.Equal(before, snapshot()) {
		t.Fatal("failed upgrade changed original ledger, schema or access records")
	}
	_, err = pool.Exec(ctx, "DROP VIEW access.management_publication")
	must(t, err)
	for range 2 {
		must(t, bootstrap.Migrate(ctx, pool, migrations.Files))
	}
	must(t, bootstrap.CheckCurrentStorage(ctx, pool))
	must(t, bootstrap.Initialize(ctx, pool, init))
	must(t, pool.QueryRow(ctx, `SELECT NOT EXISTS(SELECT FROM access.management_root)
	 AND NOT EXISTS(SELECT FROM access.principal WHERE management_owner IS NOT NULL)
	 AND NOT EXISTS(SELECT FROM access.bound_permission WHERE action='department.read')
	 AND EXISTS(SELECT FROM access.bound_permission WHERE action='bu.read')`).Scan(&intact))
	if !intact || !maps.Equal(before, snapshot()) {
		t.Fatal("retry or startup changed the old principal, credential, grant, revocation or actor")
	}
	t.Run("deployment-without-API", func(t *testing.T) {
		bin := filepath.Join(t.TempDir(), "upgrader")
		build := exec.CommandContext(ctx, "go", "build", "-o", bin, "../../cmd/upgrader")
		must(t, build.Run())
		dsn := pool.Config().ConnString()
		run := func(connection string, in access.RecoveryCommand) (access.ManagementResult, error) {
			input, err := json.Marshal(in)
			must(t, err)
			cmd := exec.CommandContext(ctx, bin, "-mode", "access-admin")
			cmd.Env = append(os.Environ(), "CHNHCM_ADMIN_DATABASE_URL="+connection, "CHNHCM_INSTANCE_ID="+init.Instance)
			cmd.Stdin = bytes.NewReader(input)
			var result access.ManagementResult
			out, err := cmd.Output()
			if err == nil {
				err = json.Unmarshal(out, &result)
			}
			return result, err
		}
		pw := foundation.NewID()
		must(t, bootstrap.ConfigureLocalRoles(ctx, pool, pw))
		runtimeDSN, err := url.Parse(dsn)
		must(t, err)
		runtimeDSN.User = url.UserPassword("chnhcm_api", pw)
		in := access.RecoveryCommand{Instance: init.Instance, Operation: "bootstrap", Login: init.Login, Request: foundation.NewID(), Operator: "部署运维", Reason: "隔离升级后的本实例引导"}
		if _, err := run(runtimeDSN.String(), in); err == nil {
			t.Fatal("application database role established a protected administrator")
		}
		first, err := run(dsn, in)
		must(t, err)
		wrongInit := init
		wrongInit.Password = foundation.NewID()
		if bootstrap.Initialize(ctx, pool, wrongInit) == nil {
			t.Fatal("appointment without a credential change suppressed the old credential check")
		}
		in.Operation, in.Request, in.ExpectedAdmin, in.Password = "recover", foundation.NewID(), first.Target, foundation.NewID()
		recovered, err := run(dsn, in)
		must(t, err)
		if recovered.Target != first.Target {
			t.Fatal("deployment recovery changed the original principal")
		}
		must(t, bootstrap.Initialize(ctx, pool, init))
		in.Operation, in.Request, in.Login = "handover", foundation.NewID(), "next-security"
		next, err := run(dsn, in)
		must(t, err)
		var safe bool
		must(t, pool.QueryRow(ctx, `SELECT principal_id=$1 AND NOT EXISTS(SELECT FROM access.bound_permission WHERE action='department.read')
		 FROM access.management_root WHERE instance_id=$2`, next.Target, init.Instance).Scan(&safe))
		if !safe || next.Target == first.Target {
			t.Fatal("deployment handover restored withdrawn business access or retained the wrong appointment")
		}
	})
}

func managed(t *testing.T) *managementFixture {
	t.Helper()
	f := fresh(t)
	password := foundation.NewID()
	in := access.RecoveryCommand{Instance: f.instance, Request: foundation.NewID(), Operation: "bootstrap", Login: "security", Password: password, Reason: "首次引导测试", Operator: "隔离实例维护员", MaxProxyDays: new(7)}
	// Two deployment connections with the same identity must observe one original appointment.
	var wg sync.WaitGroup
	results := make([]access.ManagementResult, 2)
	errs := make([]error, 2)
	for i := range results {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i], errs[i] = bootstrap.RecoverManagement(context.Background(), f.admin, f.instance, in)
		}()
	}
	wg.Wait()
	for _, err := range errs {
		must(t, err)
	}
	if results[0] != results[1] || f.scalar("SELECT count(*) FROM access.management_root") != 1 {
		t.Fatal("first administrator appointment diverged")
	}
	in.Request = foundation.NewID()
	if _, e := bootstrap.RecoverManagement(f.ctx, f.admin, f.instance, in); e == nil {
		t.Fatal("second initial administrator admitted")
	}
	return &managementFixture{f, f.login("security", password), password}
}
func (m *managementFixture) cmd(who *browser, c access.ManagementCommand, want int) map[string]any {
	m.t.Helper()
	if c.Request == "" {
		c.Request = foundation.NewID()
	}
	if c.Instance == "" {
		c.Instance = m.instance
	}
	return who.call("POST", "/api/v1/access/commands", c, "", want)
}
func viewMgmt(t *testing.T, who *browser, path string) access.ManagementView {
	t.Helper()
	v := who.call("GET", "/api/v1/access/"+path, nil, "", 200)
	raw, e := json.Marshal(v)
	must(t, e)
	var out access.ManagementView
	must(t, json.Unmarshal(raw, &out))
	return out
}
func (m *managementFixture) create(login, owner, pwd string) string {
	return m.cmd(m.root, access.ManagementCommand{Operation: "account.create", Login: login, Owner: owner, Password: pwd}, 200)["target"].(string)
}
func TestAccessManagementAccountsAndRecovery(t *testing.T) {
	m := managed(t)
	root := m.root
	id := foundation.NewID
	pwd := id()
	old := access.ManagementCommand{Request: id(), Operation: "account.create", Login: "ordinary", Display: "本地经办人", Owner: m.instance, Password: pwd}
	result := m.cmd(root, old, 200)
	subject := result["target"].(string)
	if m.cmd(root, old, 200)["at"] != result["at"] {
		t.Fatal("replay lost original result")
	}
	old.Password = id()
	m.cmd(root, old, 409)
	first, second := m.login("ordinary", pwd), m.login("ordinary", pwd)
	if m.scalar("SELECT count(*) FROM access.grant_assignment WHERE principal_id=$1", subject) != 0 {
		t.Fatal("implicit business grants")
	}
	first.call("GET", "/api/v1/access/accounts", nil, "", 403)
	view := viewMgmt(t, root, "accounts?subject="+subject)
	if len(view.Events) != 1 || view.Events[0].Actor != "security" {
		t.Fatal("missing account origin", view.Events)
	}
	cmd := access.ManagementCommand{Operation: "account.reset", Subject: subject, Revision: view.Accounts[0].Revision, Request: id(), Password: id()}
	m.cmd(root, cmd, 200)
	m.cmd(root, cmd, 200)
	for _, b := range []*browser{first, second} {
		b.call("GET", "/api/v1/session", nil, "", 401)
	}
	first.call("POST", "/api/v1/session/login", map[string]string{"login": "ordinary", "password": pwd}, "", 401)
	first, second = m.login("ordinary", cmd.Password), m.login("ordinary", cmd.Password)
	first.call("GET", "/api/v1/departments", nil, "", 403)
	rev := viewMgmt(t, root, "accounts?subject="+subject).Accounts[0].Revision
	m.cmd(root, access.ManagementCommand{Operation: "account.disable", Subject: subject, Revision: rev - 1}, 409)
	m.cmd(root, access.ManagementCommand{Operation: "account.disable", Subject: subject, Revision: rev}, 200)
	first.call("GET", "/api/v1/session", nil, "", 401)
	second.call("GET", "/api/v1/session", nil, "", 401)
	first.call("POST", "/api/v1/session/login", map[string]string{"login": "ordinary", "password": cmd.Password}, "", 401)
	m.cmd(root, access.ManagementCommand{Operation: "account.enable", Subject: subject, Revision: rev + 1}, 200)
	first.call("GET", "/api/v1/session", nil, "", 401)
	m.login("ordinary", cmd.Password)
	m.cmd(root, access.ManagementCommand{Operation: "account.disable", Subject: root.subject, Revision: 1}, 403)
	// Completion lookup exposes status only, and a fenced missing submission can never mutate later.
	receipt := root.call("POST", "/api/v1/access/submissions/"+cmd.Request+"/resolve", nil, "", 200)
	raw, _ := json.Marshal(receipt)
	if strings.Contains(string(raw), cmd.Password) || receipt["target"] != subject || receipt["at"] == nil {
		t.Fatal("invalid credential receipt")
	}
	missing := id()
	root.call("POST", "/api/v1/access/submissions/"+missing+"/resolve", nil, "", 200)
	cmd.Request = missing
	m.cmd(root, cmd, 409)
	first = m.login("ordinary", cmd.Password)
	recovery := access.RecoveryCommand{Instance: m.instance, Request: id(), Operation: "handover", ExpectedAdmin: root.subject, Login: "ordinary", Operator: "值班运维", Reason: "客户管理员交接", MaxProxyDays: new(7)}
	_, err := bootstrap.RecoverManagement(m.ctx, m.admin, m.instance, recovery)
	must(t, err)
	root.call("GET", "/api/v1/session", nil, "", 401)
	first.call("GET", "/api/v1/session", nil, "", 401)
	if err := bootstrap.ChangeAccess(m.ctx, m.admin, bootstrap.AccessChange{Instance: m.instance, Login: "ordinary", Operation: "disable"}); err == nil {
		t.Fatal("deployment account change disabled current root")
	}
	former := m.login("security", m.password)
	former.call("GET", "/api/v1/access/accounts", nil, "", 403)
	next := m.login("ordinary", cmd.Password)
	next.call("GET", "/api/v1/access/accounts", nil, "", 200)
	next.call("GET", "/api/v1/access/submissions/"+old.Request, nil, "", 404)
	recovery.Operation = "recover"
	recovery.Request = id()
	if _, err = bootstrap.RecoverManagement(m.ctx, m.admin, m.instance, recovery); err == nil {
		t.Fatal("stale root recovered")
	}
	recovery.ExpectedAdmin = subject
	recovery.Password = id()
	_, err = bootstrap.RecoverManagement(m.ctx, m.admin, m.instance, recovery)
	must(t, err)
	next.call("GET", "/api/v1/session", nil, "", 401)
	recovered := m.login("ordinary", recovery.Password)
	history := viewMgmt(t, recovered, "accounts?subject="+subject).Events
	if len(history) != 6 || history[0].Actor != "值班运维" || history[len(history)-1].Actor != "security" {
		t.Fatal("actor history lost", history)
	}
}

func TestAccessManagementDelegationAndSharedVersions(t *testing.T) {
	m := managed(t)
	id := foundation.NewID
	// Normal business entry prepares two BU identities. The security administrator has no business grants.
	operator := m.login("admin", m.fixture.password)
	for _, code := range []string{"MGMTBUA", "MGMTBUB"} {
		operator.call("POST", "/api/v1/business-units", createBody(code, "2020-01-01"), id(), 201)
	}
	v := viewMgmt(t, m.root, "roles")
	var a, b string
	for _, o := range v.Targets {
		if o.Code == "MGMTBUA" {
			a = o.ID
		}
		if o.Code == "MGMTBUB" {
			b = o.ID
		}
	}
	if a == "" || b == "" {
		t.Fatal("missing management BU identities")
	}
	pwd := id()
	alice := m.create("alice", a, pwd)
	bob := m.create("bob", b, id())
	limitedID := m.create("limited", a, pwd)
	limited := m.login("limited", pwd)
	pub := func(d access.Definition, s, f *access.Definition) string {
		return m.cmd(m.root, access.ManagementCommand{Operation: "version.publish", Version: &d, NewScope: s, NewFields: f}, 200)["target"].(string)
	}
	r := pub(access.Definition{Kind: "role", Name: "甲部门读取", Actions: []string{"department.read"}}, &access.Definition{Kind: "scope", Name: "甲", TargetKind: "bu", TargetID: a}, &access.Definition{Kind: "fields", Name: "身份", Read: []string{"code", "name"}})
	v = viewMgmt(t, m.root, "roles?object="+r)
	var def access.Definition
	for _, d := range v.Versions {
		if d.ID == r {
			def = d
		}
	}
	// These separate version identities describe the same ceiling. Retirement cannot fail merely because the administrator references the version.
	ceilingRole := pub(access.Definition{Kind: "role", Name: "独立管理上限", Actions: []string{"department.read"}}, &access.Definition{Kind: "scope", Name: "上限甲", TargetKind: "bu", TargetID: a}, &access.Definition{Kind: "fields", Name: "上限身份", Read: []string{"code", "name"}})
	var ceiling access.Definition
	for _, d := range viewMgmt(t, m.root, "roles?object="+ceilingRole).Versions {
		if d.ID == ceilingRole {
			ceiling = d
		}
	}
	l := access.ManagementLimit{Owner: a, Rights: []string{"account", "role", "policy", "assign"}, Role: ceilingRole, Scope: ceiling.Scope, Fields: ceiling.Fields, MaxProxyDays: 1}
	mandate := m.cmd(m.root, access.ManagementCommand{Operation: "mandate.issue", Subject: limitedID, Limit: &l}, 200)["target"].(string)
	limited.call("GET", "/api/v1/departments", nil, "", 403)
	limited.call("GET", "/api/v1/access/grants?subject="+bob, nil, "", 403)
	candidates := viewMgmt(t, limited, "accounts")
	for _, p := range candidates.Accounts {
		if p.Owner != a {
			t.Fatal("out-of-scope identity disclosed")
		}
	}
	m.cmd(limited, access.ManagementCommand{Operation: "account.create", Owner: b, Login: "outside", Password: id()}, 403)
	m.cmd(limited, access.ManagementCommand{Operation: "account.reset", Subject: alice, Revision: 1, Password: id()}, 403)
	m.cmd(limited, access.ManagementCommand{Operation: "grant.issue", Subject: limitedID, Role: r, Scope: def.Scope, Fields: def.Fields, Source: "duty"}, 403)
	issue := access.ManagementCommand{Operation: "grant.issue", Subject: alice, Role: r, Scope: def.Scope, Fields: def.Fields, Source: "duty"}
	own := m.cmd(limited, issue, 200)["target"].(string)
	issue.Source = "proxy"
	later := time.Now().Add(48 * time.Hour)
	issue.Until = &later
	m.cmd(limited, issue, 403)
	later = time.Now().Add(time.Hour)
	proxy := m.cmd(limited, issue, 200)["target"].(string)
	revoke := access.ManagementCommand{Operation: "grant.revoke", Subject: alice, ID: proxy, Revision: 1}
	m.cmd(limited, revoke, 200)
	if m.scalar("SELECT count(*) FROM access.bound_permission WHERE principal_id=$1 AND action='department.read'", alice) != 1 {
		t.Fatal("single revoke lost overlapping duty")
	}
	moved := m.create("move-account", a, id())
	move := access.ManagementCommand{Operation: "account.move", Subject: moved, Owner: b, Revision: 1}
	m.cmd(limited, move, 403)
	m.cmd(m.root, move, 200)
	history := viewMgmt(t, m.root, "accounts?subject="+moved).Events
	if len(history) != 2 || history[0].PreviousOwner != a || history[0].AssignedOwner != b || history[1].AssignedOwner != a {
		t.Fatal("account ownership history lost")
	}
	limited.call("GET", "/api/v1/access/accounts?subject="+moved, nil, "", 403)
	future := time.Now().Add(time.Hour)
	issue.Subject = bob
	issue.Source = "duty"
	issue.From = &future
	issue.Until = nil
	m.cmd(limited, issue, 403)
	m.cmd(m.root, issue, 200)
	disable := access.ManagementCommand{Operation: "version.disable", Version: &access.Definition{Kind: "role"}, ID: r, Revision: 1}
	visible := viewMgmt(t, limited, "roles?object="+r+"&panel=references&state=impact").Impact
	if visible == nil || visible.Token != "" || len(visible.References) != 1 || visible.References[0].Subject != alice {
		t.Fatalf("reference boundary: %+v", visible)
	}
	ceilingImpact := viewMgmt(t, m.root, "roles?object="+ceilingRole+"&panel=references&state=impact").Impact
	if ceilingImpact == nil || ceilingImpact.Mandates != 1 || ceilingImpact.Grants != 0 || ceilingImpact.References[0].ID != mandate {
		t.Fatalf("missing management mandate: %+v", ceilingImpact)
	}
	m.cmd(limited, disable, 403)
	for _, pair := range [][2]string{{"scope", def.Scope}, {"fields", def.Fields}} {
		other := disable
		other.Version = &access.Definition{Kind: pair[0]}
		other.ID = pair[1]
		m.cmd(limited, other, 403)
	}
	d := def
	d.Name = "甲部门读取与历史"
	d.Actions = []string{"department.read", "department.history.read"}
	newRole := pub(d, nil, nil)
	if m.scalar("SELECT count(*) FROM access.bound_permission WHERE principal_id=$1 AND action='department.history.read'", alice) != 0 {
		t.Fatal("publication changed old grant")
	}
	adopt := access.ManagementCommand{Operation: "grant.adopt", ID: own, Subject: alice, Revision: 1, Role: newRole, Scope: def.Scope, Fields: def.Fields}
	m.cmd(limited, adopt, 403)
	m.cmd(m.root, adopt, 200)
	if m.scalar("SELECT count(*) FROM access.bound_permission WHERE principal_id=$1 AND action='department.history.read'", alice) != 1 {
		t.Fatal("explicit adoption missing")
	}
	// A local account manager cannot reset/disable an account whose new role exceeds their ceiling.
	m.cmd(limited, access.ManagementCommand{Operation: "account.disable", Subject: alice, Revision: 5}, 403)
	impactView := func(who *browser) *access.VersionImpact {
		return viewMgmt(t, who, "roles?object="+r+"&panel=references&state=impact").Impact
	}
	scoped := impactView(limited)
	if scoped == nil || scoped.Token != "" || scoped.Accounts != 0 {
		t.Fatalf("restricted impact disclosed: %+v", scoped)
	}
	current := impactView(m.root)
	if current == nil || current.Accounts != 1 || current.Grants != 1 || current.Token == "" {
		t.Fatalf("incorrect impact: %+v", current)
	}
	disable.ImpactToken = current.Token
	issue.Subject = m.create("third", b, id())
	m.cmd(m.root, issue, 200)
	m.cmd(m.root, disable, 409)
	current = impactView(m.root)
	if current.Grants != 2 || current.Accounts != 2 || current.Token == disable.ImpactToken {
		t.Fatalf("impact did not follow second assignment: %+v", current)
	}
	disable.ImpactToken = current.Token
	events := viewMgmt(t, m.root, "grants?subject="+alice+"&panel=events&record="+own)
	if len(events.Events) != 2 || events.Events[0].Before == "" || !strings.Contains(events.Events[0].Summary, "V2") || !strings.Contains(events.Events[0].After, "V2") || !strings.Contains(events.Events[1].After, "V1") {
		t.Fatalf("lost historical terms: %+v", events.Events)
	}
	lists := viewMgmt(t, m.root, "grants?q=alice").Accounts
	if len(lists) != 1 || lists[0].ActiveGrants == nil || *lists[0].ActiveGrants != 1 {
		t.Fatalf("grant directory: %+v", lists)
	}
	old := viewMgmt(t, m.root, "grants?subject="+alice+"&source=proxy&state=revoked")
	if old.Total != 1 || len(old.Grants) != 1 || old.Grants[0].ID != proxy || old.Grants[0].CanRevoke || old.Grants[0].Adoption != nil {
		t.Fatalf("unusable history: %+v", old.Grants)
	}
	m.cmd(m.root, disable, 200)
	limited.call("GET", "/api/v1/access/accounts", nil, "", 200)
	if m.scalar("SELECT count(*) FROM access.bound_permission WHERE principal_id=$1 AND action='department.read'", bob) != 0 {
		t.Fatal("future reference survived retirement")
	}
	if m.scalar("SELECT count(*) FROM access.grant_revision WHERE grant_id=$1", own) != 2 {
		t.Fatal("original grant binding lost")
	}
	m.cmd(m.root, access.ManagementCommand{Operation: "mandate.revoke", Subject: limitedID, ID: mandate}, 200)
	limited.call("GET", "/api/v1/access/accounts", nil, "", 403)
}

func TestAccessManagementScopedCredentialAndExpiry(t *testing.T) {
	m := managed(t)
	id := foundation.NewID
	v := viewMgmt(t, m.root, "roles")
	var ent string
	for _, target := range v.Targets {
		if target.Kind == "enterprise" {
			ent = target.ID
		}
	}
	r := m.cmd(m.root, access.ManagementCommand{Operation: "version.publish", Version: &access.Definition{Kind: "role", Name: "经办名单", Actions: []string{"department.read"}}, NewScope: &access.Definition{Kind: "scope", Name: "本企业", TargetKind: "enterprise", TargetID: ent}, NewFields: &access.Definition{Kind: "fields", Name: "名称", Read: []string{"name"}}}, 200)["target"].(string)
	var def access.Definition
	for _, item := range viewMgmt(t, m.root, "roles?object="+r).Versions {
		if item.ID == r {
			def = item
		}
	}
	pwd := id()
	manager := m.create("credential-owner", m.instance, pwd)
	subject := m.create("clerk", m.instance, id())
	clerk := m.login("credential-owner", pwd)
	ceiling := access.ManagementLimit{Owner: m.instance, Rights: []string{"credential"}, Role: r, Scope: def.Scope, Fields: def.Fields, MaxProxyDays: 7}
	m.cmd(m.root, access.ManagementCommand{Operation: "mandate.issue", Subject: manager, Limit: &ceiling}, 200)
	m.cmd(clerk, access.ManagementCommand{Operation: "account.create", Login: "wrong", Owner: m.instance, Password: id()}, 403)
	m.cmd(clerk, access.ManagementCommand{Operation: "account.disable", Subject: subject, Revision: 1}, 403)
	reset := access.ManagementCommand{Request: id(), Operation: "account.reset", Subject: subject, Revision: 1, Password: id()}
	m.cmd(clerk, reset, 200)
	clerk.call("GET", "/api/v1/access/submissions/"+reset.Request, nil, "", 200)
	m.login("clerk", reset.Password)
	// Establish the original initialized account's owner explicitly. Personnel and other legacy actions exceed this mandate.
	legacy := m.login("reader", m.fixture.readPassword)
	a := viewMgmt(t, m.root, "accounts?subject="+legacy.subject).Accounts[0]
	m.cmd(m.root, access.ManagementCommand{Operation: "account.move", Subject: a.ID, Revision: a.Revision, Owner: m.instance}, 200)
	reset.Subject, reset.Revision, reset.Request = a.ID, a.Revision+1, id()
	m.cmd(clerk, reset, 403)
	m.cmd(m.root, reset, 200)
	legacy.call("GET", "/api/v1/session", nil, "", 401)
	must(t, bootstrap.Initialize(m.ctx, m.admin, bootstrap.InitInput{Instance: m.instance, EnterpriseCode: "LOCAL", EnterpriseName: "本地验证企业", Start: "2020-01-01", Login: "reader", Password: m.fixture.readPassword, ReadOnly: true}))
	legacy.call("POST", "/api/v1/session/login", map[string]string{"login": "reader", "password": m.fixture.readPassword}, "", 401)
	if m.login("reader", reset.Password).subject != a.ID {
		t.Fatal("startup changed credential or stable identity")
	}
	// Hold the account insertion after the delegated session and ceiling are checked, then cross its absolute end.
	until := time.Now().Add(2 * time.Second)
	ceiling.Rights = []string{"account"}
	ceiling.Until = &until
	m.cmd(m.root, access.ManagementCommand{Operation: "mandate.issue", Subject: manager, Limit: &ceiling}, 200)
	tx, e := m.admin.Begin(m.ctx)
	must(t, e)
	defer tx.Rollback(m.ctx)
	_, e = tx.Exec(m.ctx, "LOCK access.principal IN SHARE MODE")
	must(t, e)
	done := make(chan int, 1)
	go func() {
		s, _, _ := clerk.response("POST", "/api/v1/access/commands", access.ManagementCommand{Instance: m.instance, Request: id(), Operation: "account.create", Login: "late-account", Owner: m.instance, Password: id()}, "")
		done <- s
	}()
	deadline := time.Now().Add(time.Second)
	for m.scalar("SELECT count(*) FROM pg_stat_activity WHERE usename='chnhcm_api' AND $1=ANY(pg_blocking_pids(pid))", int(tx.Conn().PgConn().PID())) == 0 {
		if time.Now().After(deadline) {
			t.Fatal("management write did not reach blocker")
		}
		time.Sleep(10 * time.Millisecond)
	}
	var stored, released time.Time
	must(t, tx.QueryRow(m.ctx, "SELECT valid_until FROM access.management_mandate WHERE instance_id=$1 AND principal_id=$2 AND rights=ARRAY['account']", m.instance, manager).Scan(&stored))
	_, e = tx.Exec(m.ctx, "SELECT pg_sleep_until($1::timestamptz)", stored)
	must(t, e)
	must(t, tx.QueryRow(m.ctx, "SELECT clock_timestamp()").Scan(&released))
	now := time.Now()
	t.Logf("委派截止 %s，释放前数据库时刻 %s，服务时刻 %s", stored.UTC(), released.UTC(), now.UTC())
	if released.Before(stored) || now.Before(stored) {
		t.Fatal("release precondition is before the stored mandate expiry")
	}
	must(t, tx.Commit(m.ctx))
	status := <-done
	rows := m.scalar("SELECT count(*) FROM access.principal WHERE normalized_login='late-account'")
	if status != 403 || rows != 0 {
		t.Fatalf("after stored deadline: HTTP %d, committed accounts %d", status, rows)
	}
}

func TestAccessGrantWorkspacePagingAndOriginalReceipt(t *testing.T) {
	m := managed(t)
	id := foundation.NewID
	pwd := id()
	operator := m.create("grant-operator", m.instance, pwd)
	subject := m.create("grant-recipient", m.instance, id())
	var enterprise string
	for _, a := range viewMgmt(t, m.root, "roles").Targets {
		if a.Kind == "enterprise" {
			enterprise = a.ID
		}
	}
	role := m.cmd(m.root, access.ManagementCommand{Operation: "version.publish", Version: &access.Definition{Kind: "role", Name: "同名授予", Actions: []string{"department.read"}}, NewScope: &access.Definition{Kind: "scope", Name: "企业范围", TargetKind: "enterprise", TargetID: enterprise}, NewFields: &access.Definition{Kind: "fields", Name: "编码", Read: []string{"code"}}}, 200)["target"].(string)
	var published access.Definition
	for _, d := range viewMgmt(t, m.root, "roles?object="+role).Versions {
		if d.ID == role {
			published = d
		}
	}
	m.cmd(m.root, access.ManagementCommand{Operation: "mandate.issue", Subject: operator, Limit: &access.ManagementLimit{Owner: m.instance, Rights: []string{"assign"}, Role: role, Scope: published.Scope, Fields: published.Fields, MaxProxyDays: 1}}, 200)
	limited := m.login("grant-operator", pwd)
	from := time.Now().UTC().Truncate(time.Second)
	cmd := access.ManagementCommand{Operation: "grant.issue", Subject: subject, Role: role, Scope: published.Scope, Fields: published.Fields, Source: "duty", From: &from}
	firstRequest := id()
	cmd.Request = firstRequest
	original := m.cmd(limited, cmd, 200)["target"].(string)
	for range 22 {
		cmd.Request = id()
		m.cmd(limited, cmd, 200)
	}
	path := "grants?subject=" + subject
	visible := viewMgmt(t, limited, path)
	if visible.Total != 23 || len(visible.Grants) != 20 || !visible.Grants[0].Distinguish || len(visible.Events) != 0 || len(visible.Versions) != 3 {
		t.Fatalf("unbounded or ambiguous first page: %d/%d", len(visible.Grants), visible.Total)
	}
	second := viewMgmt(t, limited, path+"&page=1")
	if second.Total != 23 || second.Page != 1 || len(second.Grants) != 3 {
		t.Fatal("invalid final page", second.Page, second.Total, len(second.Grants))
	}
	anchor := second.Grants[2].ID
	located := viewMgmt(t, limited, path+"&record="+anchor)
	if located.Page != 1 || located.Grants[2].ID != anchor {
		t.Fatal("record link did not locate its page")
	}
	events := viewMgmt(t, limited, path+"&panel=events")
	if events.Total != 23 || len(events.Events) != 20 || events.Events[0].Request == firstRequest {
		t.Fatal("invalid event page")
	}
	got := limited.call("POST", "/api/v1/access/submissions/"+firstRequest+"/resolve", nil, "", 200)
	if got["target"] != original {
		t.Fatal("original receipt outside the visible page disappeared")
	}
	limited.call("GET", "/api/v1/access/"+path+"&panel=mandates", nil, "", 403)
	delegateID := m.create("mandates-only", m.instance, pwd)
	m.cmd(m.root, access.ManagementCommand{Operation: "mandate.issue", Subject: delegateID, Limit: &access.ManagementLimit{Owner: m.instance, Rights: []string{"delegate"}, Role: role, Scope: published.Scope, Fields: published.Fields, MaxProxyDays: 1}}, 200)
	delegate := m.login("mandates-only", pwd)
	directory := viewMgmt(t, delegate, "grants?q=grant-recipient")
	workspace := viewMgmt(t, delegate, path)
	if len(directory.Accounts) != 1 || directory.Accounts[0].ActiveGrants != nil || workspace.Panel != "mandates" || len(workspace.Grants) != 0 || workspace.Total != 0 {
		t.Fatal("delegate-only administrator received business grants or counts")
	}
	delegate.call("GET", "/api/v1/access/"+path+"&panel=grants", nil, "", 403)
	if history := viewMgmt(t, delegate, path+"&panel=events"); history.Total != 0 {
		t.Fatal("business-grant history disclosed to mandate manager")
	}

}

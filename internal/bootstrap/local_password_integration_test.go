//go:build integration

package bootstrap_test

import (
	"testing"

	"chnhcm.local/chnhcm/internal/bootstrap"
)

func TestLocalDevelopmentPasswordChange(t *testing.T) {
	f := fresh(t)
	oldSession := f.login("admin", f.password)
	const password = "admin1234567"
	change := bootstrap.LocalPasswordChange{Instance: f.instance, Login: "admin", PreviousPassword: "wrong-password", Password: password}
	if e := bootstrap.ChangeLocalPassword(f.ctx, f.admin, change); e == nil {
		t.Fatal("accepted incorrect previous password")
	}
	oldSession.call("GET", "/api/v1/session", nil, "", 200)
	change.PreviousPassword = f.password
	must(t, bootstrap.ChangeLocalPassword(f.ctx, f.admin, change))
	oldSession.call("GET", "/api/v1/session", nil, "", 401)
	oldSession.call("POST", "/api/v1/session/login", map[string]string{"login": "admin", "password": f.password}, "", 401)
	current := f.login("admin", password)
	if current.subject != oldSession.subject {
		t.Fatal("password change replaced the account")
	}
	// Recover a DB success followed by an interrupted local config write.
	must(t, bootstrap.ChangeLocalPassword(f.ctx, f.admin, change))
	current.call("GET", "/api/v1/session", nil, "", 200)
	change.Login, change.PreviousPassword = "reader", f.readPassword
	must(t, bootstrap.ChangeLocalPassword(f.ctx, f.admin, change))
	reader := f.login("reader", password)
	var writable int
	must(t, f.admin.QueryRow(f.ctx, `SELECT count(*) FROM access.subject_grant WHERE instance_id=$1 AND principal_id=$2 AND cardinality(write_fields)>0`, f.instance, reader.subject).Scan(&writable))
	if writable != 0 {
		t.Fatal("password change altered reader permissions")
	}
	var events int
	must(t, f.admin.QueryRow(f.ctx, `SELECT count(*) FROM access.security_event WHERE instance_id=$1 AND action='local.password.change'`, f.instance).Scan(&events))
	if events != 2 {
		t.Fatalf("expected two actual password changes, got %d", events)
	}
	// Normal startup still recognizes the same initialized identities and scope.
	for _, account := range []struct {
		login    string
		readOnly bool
	}{{"admin", false}, {"reader", true}} {
		must(t, bootstrap.Initialize(f.ctx, f.admin, bootstrap.InitInput{Instance: f.instance, EnterpriseCode: "LOCAL", EnterpriseName: "本地验证企业", Start: "2020-01-01", Login: account.login, Password: password, ReadOnly: account.readOnly}))
	}
}

//go:build integration

package bootstrap_test

import (
	"errors"
	"reflect"
	"testing"

	"chnhcm.local/chnhcm/db/migrations"
	"chnhcm.local/chnhcm/internal/bootstrap"
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	staffingdb "chnhcm.local/chnhcm/internal/contexts/staffing/infra/postgres"
	f "chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
)

func TestHire400CustomerRule(t *testing.T) {
	h := fresh(t)
	ctx := h.ctx
	cred := geoCredentials203(t, h, "admin", h.password)
	arrival := "2027-04-01"
	supplier := h.runtime.FirstHireRules
	want := hireprep.DirectEmployeeSample()
	missing := func(err error) {
		t.Helper()
		var fault *f.Fault
		if !errors.As(err, &fault) || fault.Code != "HIRE_RULE_UNAVAILABLE" {
			t.Fatalf("expected missing customer rule, got %v", err)
		}
	}
	_, err := supplier.FirstHireRule(ctx, cred, arrival)
	missing(err)
	var subject string
	must(t, h.admin.QueryRow(ctx, "SELECT principal_id::text FROM access.principal WHERE instance_id=$1 AND normalized_login='admin'", h.instance).Scan(&subject))
	period := f.Period{From: arrival, Until: "2027-05-01"}
	tx, err := h.admin.Begin(ctx)
	must(t, err)
	defer tx.Rollback(ctx)
	must(t, staffingdb.InitializeFirstHireRule(ctx, tx, h.instance, subject, want.Version, period))
	adopted, err := supplier.FirstHireRule(dbtx.With(ctx, tx), cred, arrival)
	must(t, err)
	if !reflect.DeepEqual(adopted, want) {
		t.Fatal("caller transaction did not see the entire customer rule")
	}
	must(t, tx.Rollback(ctx))
	_, err = supplier.FirstHireRule(ctx, cred, arrival)
	missing(err)

	input := bootstrap.InitInput{Instance: h.instance, EnterpriseCode: "LOCAL", EnterpriseName: "本地验证企业", Login: "admin", Password: h.password,
		FirstHireRule: &bootstrap.FirstHireRuleInput{Version: want.Version, From: period.From, Until: period.Until}}
	for range 2 {
		must(t, bootstrap.Initialize(ctx, h.admin, input))
	}
	for _, date := range []string{"2027-03-31", period.Until} {
		_, err = supplier.FirstHireRule(ctx, cred, date)
		missing(err)
	}
	for _, date := range []string{arrival, "2027-04-30"} {
		rule, err := supplier.FirstHireRule(ctx, cred, date)
		must(t, err)
		if !reflect.DeepEqual(rule, want) {
			t.Fatalf("incorrect rule for %s", date)
		}
	}
	input.FirstHireRule = &bootstrap.FirstHireRuleInput{Version: want.Version}
	must(t, bootstrap.Initialize(ctx, h.admin, input))
	input.FirstHireRule.From = period.From
	if bootstrap.Initialize(ctx, h.admin, input) == nil {
		t.Fatal("overwrote previous period")
	}
	if h.scalar("SELECT count(*) FROM staffing.hire_rule_source WHERE instance_id=$1 AND subject_id=$2 AND version=$3 AND upper(period)=$4::date", h.instance, subject, want.Version, period.Until) != 1 {
		t.Fatal("original rule or its attribution changed")
	}
	input.Instance = f.NewID()
	input.FirstHireRule.Version = "unknown"
	if bootstrap.Initialize(ctx, h.admin, input) == nil || h.scalar("SELECT count(*) FROM organization.enterprise WHERE instance_id=$1", input.Instance) != 0 {
		t.Fatal("invalid rule left a partial customer initialization")
	}
	input.FirstHireRule = nil
	must(t, bootstrap.Initialize(ctx, h.admin, input))
	other, err := bootstrap.Open(ctx, h.runtime.Pool.Config().ConnConfig.ConnString(), input.Instance)
	must(t, err)
	defer other.Close()
	login, err := other.Access.Login(ctx, access.LoginInput{Login: input.Login, Password: input.Password}, f.NewID())
	must(t, err)
	otherCred := f.Credentials{Token: login.Token, CSRF: login.Principal.CSRF, Write: true}
	_, err = other.FirstHireRules.FirstHireRule(ctx, otherCred, arrival)
	missing(err)
	input.FirstHireRule = &bootstrap.FirstHireRuleInput{Version: want.Version}
	must(t, bootstrap.Initialize(ctx, h.admin, input))
	for _, date := range []string{f.DefaultEffectiveDate, "2099-12-31"} {
		rule, err := other.FirstHireRules.FirstHireRule(ctx, otherCred, date)
		must(t, err)
		if !reflect.DeepEqual(rule, want) {
			t.Fatal("incorrect initial open-ended rule")
		}
	}
	must(t, h.runtime.Access.Logout(ctx, cred))
	if _, err = supplier.FirstHireRule(ctx, cred, arrival); err == nil {
		t.Fatal("revoked session read customer policy")
	}
}

func TestHire400RuleUpgradeFrom34(t *testing.T) {
	pool := newStaffingMigrationPool(t)
	ctx := t.Context()
	must(t, bootstrap.Migrate(ctx, pool, hireMigrationSet(t, func(v string) bool { return v <= "00034" })))
	input := bootstrap.InitInput{Instance: f.NewID(), EnterpriseCode: "OLD34", EnterpriseName: "原客户", Login: "admin", Password: f.NewID()}
	must(t, bootstrap.Initialize(ctx, pool, input))
	var original string
	must(t, pool.QueryRow(ctx, "SELECT head_source_id::text FROM organization.enterprise_profile WHERE instance_id=$1", input.Instance).Scan(&original))
	if bootstrap.CheckCurrentStorage(ctx, pool) == nil {
		t.Fatal("runtime accepted 34")
	}
	_, err := pool.Exec(ctx, "CREATE TABLE staffing.hire_rule_source(fixture boolean)")
	must(t, err)
	if bootstrap.Migrate(ctx, pool, migrations.Files) == nil {
		t.Fatal("conflicting partial schema admitted")
	}
	var last int
	must(t, pool.QueryRow(ctx, "SELECT max(version_id) FROM goose_db_version WHERE is_applied").Scan(&last))
	if last != 34 {
		t.Fatal("failed migration changed the ledger")
	}
	_, err = pool.Exec(ctx, "DROP TABLE staffing.hire_rule_source")
	must(t, err)
	must(t, bootstrap.Migrate(ctx, pool, migrations.Files))
	must(t, bootstrap.Migrate(ctx, pool, migrations.Files))
	must(t, bootstrap.CheckCurrentStorage(ctx, pool))
	var unchanged bool
	must(t, pool.QueryRow(ctx, `SELECT EXISTS(SELECT FROM organization.enterprise_profile WHERE instance_id=$1 AND head_source_id=$2)
	AND NOT EXISTS(SELECT FROM staffing.hire_rule_source)
	AND (SELECT count(*)=1 FROM goose_db_version WHERE version_id=35 AND is_applied)`, input.Instance, original).Scan(&unchanged))
	if !unchanged {
		t.Fatal("upgrade changed old data, assigned a rule, or duplicated the ledger")
	}
}

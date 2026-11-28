//go:build integration

package bootstrap_test

import (
	"crypto/sha256"
	"fmt"
	"maps"
	"strings"
	"testing"
	"testing/fstest"

	"chnhcm.local/chnhcm/db/migrations"
	"chnhcm.local/chnhcm/internal/bootstrap"
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

func TestAccessManagementUpgradeFromOriginal59(t *testing.T) {
	ctx, pool := t.Context(), newStaffingMigrationPool(t)
	previous := hireMigrationSet(t, func(n string) bool { return n <= "00058" })
	// The original 4627 migration is byte-for-byte the SQL now published at 64.
	// Pin the historic bytes before constructing the actual predecessor ledger.
	management, err := migrations.Files.ReadFile("00064_access_management.sql")
	must(t, err)
	if fmt.Sprintf("%x", sha256.Sum256(management)) != "642de72a804a5fdeaa5ea3e762f8d5c6d3c87f2e86f184c864519d1e62d6a7c9" {
		t.Fatal("fixture no longer contains the original 00059 management SQL")
	}
	previous["00059_access_management.sql"] = &fstest.MapFile{Data: management}
	must(t, bootstrap.Migrate(ctx, pool, previous))
	init := bootstrap.InitInput{Instance: foundation.NewID(), EnterpriseCode: "OLD", EnterpriseName: "原安全管理实例", Login: "legacy-operator", Password: foundation.NewID(), ReadOnly: true, DevelopmentFixture: true}
	must(t, bootstrap.Initialize(ctx, pool, init))
	cmd := access.RecoveryCommand{Instance: init.Instance, Operation: "bootstrap", Operator: "部署运维", Reason: "旧实例客户管理员", Request: foundation.NewID(), Login: init.Login}
	receipt, err := bootstrap.RecoverManagement(ctx, pool, init.Instance, cmd)
	must(t, err)
	_, err = pool.Exec(ctx, "CREATE TABLE public.original_ledger AS TABLE public.goose_db_version")
	must(t, err)
	snapshot := func() map[string]string {
		t.Helper()
		digests := map[string]string{}
		for _, name := range []string{"principal", "credential", "grant_assignment", "grant_revision", "policy_version", "management_root", "management_receipt", "security_event"} {
			var digest string
			must(t, pool.QueryRow(ctx, "SELECT md5(jsonb_agg(row ORDER BY row)::text) FROM (SELECT to_jsonb(t) row FROM access."+name+" t) rows").Scan(&digest))
			digests[name] = digest
		}
		return digests
	}
	before := snapshot()
	checkLedger := func(added int) {
		t.Helper()
		var ok bool
		must(t, pool.QueryRow(ctx, `SELECT NOT EXISTS (TABLE public.original_ledger EXCEPT ALL TABLE public.goose_db_version)
			AND (SELECT count(*) FROM public.goose_db_version)=(SELECT count(*) FROM public.original_ledger)+$1`, added).Scan(&ok))
		if !ok || !maps.Equal(before, snapshot()) {
			t.Fatal("an existing ledger row, grant, account, receipt, or recovery key changed")
		}
	}
	if bootstrap.CheckCurrentStorage(ctx, pool) == nil {
		t.Fatal("current runtime accepted the old management-only 59")
	}
	_, err = pool.Exec(ctx, "ALTER TABLE access.management_receipt DISABLE TRIGGER immutable_management_receipt")
	must(t, err)
	if bootstrap.Migrate(ctx, pool, migrations.Files) == nil {
		t.Fatal("upgrader accepted an unprotected original management receipt")
	}
	_, err = pool.Exec(ctx, "ALTER TABLE access.management_receipt ENABLE TRIGGER immutable_management_receipt")
	must(t, err)
	checkLedger(0)
	to60 := hireMigrationSet(t, func(n string) bool { return n <= "00060" })
	const plan = "00060_compensation_plans.sql"
	broken := maps.Clone(to60)
	up, down, found := strings.Cut(string(to60[plan].Data), "-- +goose Down")
	if !found {
		t.Fatal("missing plan migration boundary")
	}
	broken[plan] = &fstest.MapFile{Data: []byte(up + "\nSELECT 1/0;\n-- +goose Down" + down)}
	if bootstrap.Migrate(ctx, pool, broken) == nil {
		t.Fatal("expected a failure inside the plan transaction")
	}
	var untouched bool
	must(t, pool.QueryRow(ctx, "SELECT to_regnamespace('compensation') IS NULL AND to_regnamespace('configuration') IS NULL").Scan(&untouched))
	if !untouched {
		t.Fatal("failed 60 left partial compensation storage")
	}
	checkLedger(0)
	must(t, bootstrap.Migrate(ctx, pool, to60))
	checkLedger(1)
	if bootstrap.CheckCurrentStorage(ctx, pool) == nil {
		t.Fatal("runtime accepted an interrupted upgrade")
	}
	for range 2 {
		must(t, bootstrap.Migrate(ctx, pool, migrations.Files))
		must(t, bootstrap.CheckCurrentStorage(ctx, pool))
		checkLedger(4)
	}
	replay, err := bootstrap.RecoverManagement(ctx, pool, init.Instance, cmd)
	must(t, err)
	if replay.Request != receipt.Request || replay.Target != receipt.Target || !replay.At.Equal(receipt.At) {
		t.Fatal("original deployment request did not return its original result")
	}
}

//go:build integration

package bootstrap_test

import (
	"testing"

	"chnhcm.local/chnhcm/db/migrations"
	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/foundation"
)

func TestPersonnelAddressTypeUpgrade(t *testing.T) {
	pool := newStaffingMigrationPool(t)
	ctx := t.Context()
	must(t, bootstrap.Migrate(ctx, pool, hireMigrationSet(t, func(v string) bool { return v <= "00058" })))
	instance, person, version := foundation.NewID(), foundation.NewID(), foundation.NewID()
	init := bootstrap.InitInput{Instance: instance, EnterpriseCode: "LOCAL", EnterpriseName: "合成实例", Login: "profile", Password: foundation.NewID(), PersonnelProfile: "registrar", PersonalProfileFixture: true}
	must(t, bootstrap.Initialize(ctx, pool, init))
	tx, err := pool.Begin(ctx)
	must(t, err)
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, "insert into personnel.person values($1,$2,'P00000001',$3,'2036-01-01','2037-01-01','legacy','2036-01-01','2037-01-01')", instance, person, version)
	must(t, err)
	_, err = tx.Exec(ctx, "insert into personnel.number values($1,'P00000001',$2)", instance, person)
	must(t, err)
	_, err = tx.Exec(ctx, `insert into personnel.source select $1,$2,$3,null,'P00000001',principal_id,'原经办','2025-01-01','correct',null,'legacy',null from access.principal where instance_id=$1`, instance, person, version)
	must(t, err)
	_, err = tx.Exec(ctx, `insert into personnel.profile_item_payload(instance_id,person_id,version,section,item_id,revision,ordinal,fields,adoptions,ordinary_until,evidence_until,collection_policy)
 values($1,$2,$3,'addresses',gen_random_uuid(),gen_random_uuid(),1,'{"addressType":"通讯地址","street":"旧门牌","addressCountry":"原文"}','{}','2036-01-01','2037-01-01','legacy')`, instance, person, version)
	must(t, err)
	_, err = tx.Exec(ctx, `insert into personnel.terminal select $1,principal_id,gen_random_uuid(),'legacy-digest','accepted',$2,$3,null from access.principal where instance_id=$1`, instance, person, version)
	must(t, err)
	_, err = tx.Exec(ctx, `create schema expected;
 create table expected.item as select to_jsonb(i) value,i.xmin::text tx from personnel.profile_item_payload i;
 create table expected.source as table personnel.source;
 create table expected.receipt as table personnel.terminal;
 create table expected.grants as table access.person_grant`)
	must(t, err)
	must(t, tx.Commit(ctx))
	must(t, bootstrap.Migrate(ctx, pool, migrations.Files))
	var ok bool
	must(t, pool.QueryRow(ctx, `select (select count(*)=4 from personnel.address_type where instance_id=$1)
 and (select count(*)=4 from personnel.address_type_revision where instance_id=$1 and actor_id is null)
 and exists(select from expected.item e join personnel.profile_item_payload i on e.value=(to_jsonb(i)-'address_type_id'-'address_type_revision'-'address_type_day') and e.tx=i.xmin::text where i.address_type_id is null and i.address_type_revision is null and i.address_type_day is null)
 and not exists((table personnel.source except all table expected.source) union all (table expected.source except all table personnel.source))
 and not exists((table personnel.terminal except all table expected.receipt) union all (table expected.receipt except all table personnel.terminal))
 and not exists(table expected.grants except all table access.person_grant)
 and (select count(*)=2 from access.person_grant where purpose='address-types')`, instance).Scan(&ok))
	if !ok {
		t.Fatal("legacy address, receipt, source, rights or initial revision changed")
	}
	_, err = pool.Exec(ctx, "delete from access.person_grant where purpose='address-types'")
	must(t, err)
	must(t, bootstrap.Initialize(ctx, pool, init))
	must(t, pool.QueryRow(ctx, "select not exists(select from access.person_grant where purpose='address-types') and (select count(*)=4 from personnel.address_type_revision)").Scan(&ok))
	if !ok {
		t.Fatal("initializer reinstated revoked rights or changed revisions")
	}
}

//go:build integration

package bootstrap_test

import (
	"crypto/sha256"
	"fmt"
	"net/http/httptest"
	"net/url"
	"reflect"
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

func TestPersonnelAddressTypePeriodUpgrade(t *testing.T) {
	pool := newStaffingMigrationPool(t)
	ctx := t.Context()
	must(t, bootstrap.Migrate(ctx, pool, hireMigrationSet(t, func(v string) bool { return v <= "00058" || v == "00061" })))
	instance := foundation.NewID()
	init := bootstrap.InitInput{Instance: instance, EnterpriseCode: "LOCAL", EnterpriseName: "期间迁移", Login: "profile", Password: foundation.NewID(), PersonnelProfile: "registrar", PersonalProfileFixture: true}
	must(t, bootstrap.Initialize(ctx, pool, init))
	const catalog = "/api/v1/person-address-types"
	var id, oldHead string
	must(t, pool.QueryRow(ctx, "select item_id,head from personnel.address_type where instance_id=$1 and code='MAILING'", instance).Scan(&id, &oldHead))
	submission := foundation.NewID()
	// Exact JSON field order used for 00061 command digests.
	originalJSON := fmt.Sprintf(`{"submissionId":"%s","id":"%s","expectedRevision":"%s","name":"通讯地址","sortOrder":30,"selectable":false}`, submission, id, oldHead)
	digest := fmt.Sprintf("%x", sha256.Sum256([]byte(originalJSON)))
	_, err := pool.Exec(ctx, `with changed as (update personnel.address_type set selectable=false,head=gen_random_uuid()
 where instance_id=$1 and code='MAILING' returning *)
 insert into personnel.address_type_revision(instance_id,item_id,revision,name,sort_order,selectable,actor_id,submission_id,digest)
 select c.instance_id,item_id,head,name,sort_order,false,p.principal_id,$2,$3 from changed c
 join access.principal p using(instance_id)`, instance, submission, digest)
	must(t, err)
	_, err = pool.Exec(ctx, `create schema expected;
 create table expected.items as select item_id,head,code,first_day,selectable from personnel.address_type;
 create table expected.revisions as select to_jsonb(v) original,xmin::text tx from personnel.address_type_revision v`)
	must(t, err)
	must(t, bootstrap.Migrate(ctx, pool, migrations.Files))
	var ok bool
	must(t, pool.QueryRow(ctx, `select
 (select count(*)=4 from personnel.address_type)
 and (select count(*)=5 from personnel.address_type_revision)
 and not exists(select from expected.items e left join personnel.address_type t using(item_id)
   where t.head is distinct from e.head or t.first_day<>e.first_day or t.stop_day is not null
   or t.confirmed_from is distinct from (now() at time zone 'Asia/Shanghai')::date
   or t.undated_stop=e.selectable or (t.valid_during @> t.confirmed_from)<>e.selectable
   or t.valid_during @> (t.confirmed_from-1))
 and not exists(select from expected.revisions e where not exists
   (select from personnel.address_type_revision v where not v.dated and e.tx=v.xmin::text
    and e.original=(to_jsonb(v)-'dated'-'stop_day'-'undated_stop')))`).Scan(&ok))
	if !ok {
		t.Fatal("migration changed identities, receipts, original revisions or invented past availability")
	}
	apiPassword := foundation.NewID()
	must(t, bootstrap.ConfigureLocalRoles(ctx, pool, apiPassword))
	u, err := url.Parse(pool.Config().ConnConfig.ConnString())
	must(t, err)
	u.User = url.UserPassword("chnhcm_api", apiPassword)
	runtime, err := bootstrap.Open(ctx, u.String(), instance)
	must(t, err)
	defer runtime.Close()
	server := httptest.NewUnstartedServer(nil)
	server.Config.Handler = (&bootstrap.API{Runtime: runtime, Origin: "http://" + server.Listener.Addr().String(), InsecureLocal: true}).Handler()
	server.Start()
	defer server.Close()
	f := fixture{t: t, ctx: ctx, admin: pool, instance: instance, runtime: runtime, server: server}
	admin := f.login(init.Login, init.Password)
	receipt := admin.call("POST", catalog+"/submissions/"+submission+"/resolve", nil, "", 200)["item"].(map[string]any)
	replay := admin.call("POST", catalog, map[string]any{"submissionId": submission, "id": id, "expectedRevision": oldHead, "name": "通讯地址", "sortOrder": 30, "selectable": false}, "", 200)
	if !reflect.DeepEqual(replay, receipt) || receipt["selectable"] != false || receipt["stopDay"] != nil || receipt["confirmedFrom"] != nil {
		t.Fatal("legacy receipt changed", receipt)
	}
	current := admin.call("GET", catalog+"/"+id, nil, "", 200)
	item := current["item"].(map[string]any)
	if item["state"] != "inactive" || item["undatedStop"] != true || item["stopDay"] != nil || current["timeline"].([]any)[0].(map[string]any)["state"] != "unknown" {
		t.Fatal("invented legacy date", current)
	}
	corrected := admin.call("POST", catalog, map[string]any{"submissionId": foundation.NewID(), "id": id, "expectedRevision": item["revision"], "sortOrder": 30, "name": "通讯地址", "stopDay": ""}, "", 200)
	if corrected["selectable"] != true || corrected["revision"] == item["revision"] || corrected["confirmedFrom"] != item["confirmedFrom"] {
		t.Fatal("legacy correction lost known boundary")
	}
	history := admin.call("GET", catalog+"/"+id+"/revisions/"+corrected["revision"].(string), nil, "", 200)
	if history["previousRevision"] != receipt["revision"] || history["dated"] != true {
		t.Fatal("correction does not point to prior evidence", history)
	}

}

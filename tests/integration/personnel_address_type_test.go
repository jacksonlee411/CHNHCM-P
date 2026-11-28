//go:build integration

package integration_test

import (
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgconn"
	"maps"
	"reflect"
	"strings"
	"testing"
	"time"

	"chnhcm.local/chnhcm/internal/foundation"
)

func TestPersonnelAddressTypeAuthorizationDependencies(t *testing.T) {
	f := personnelFixture(t)
	hr, catalog := f.login("admin", f.password), f.login("admin", f.password)
	block, err := f.admin.Begin(f.ctx)
	must(t, err)
	defer block.Rollback(f.ctx)
	var role string
	must(t, block.QueryRow(f.ctx, `select v.version_id from access.grant_assignment a
		join access.grant_revision b using (instance_id,grant_id,revision)
		join access.policy_version v on v.instance_id=a.instance_id and v.kind='role' and v.version_id=b.role_id
		where a.instance_id=$1 and a.principal_id=$2 and 'country.read'=any(v.actions)
		limit 1 for update of v`, f.instance, hr.subject).Scan(&role))
	// An actual country reader must wait for its role definition. The M01-only
	// catalog must still finish while the very same definition is locked.
	done := make(chan int, 1)
	go func() {
		status, _, _ := catalog.response("GET", "/api/v1/country-regions/context", nil, "")
		done <- status
	}()
	until := time.Now().Add(10 * time.Second)
	for f.scalar("select count(*) from pg_stat_activity where $1::int=any(pg_blocking_pids(pid))", block.Conn().PgConn().PID()) == 0 {
		if time.Now().After(until) {
			t.Fatal("country reader did not wait for its role")
		}
		time.Sleep(10 * time.Millisecond)
	}
	items := hr.call("GET", "/api/v1/person-address-types", nil, "", 200)["items"].([]any)
	if len(items) != 4 {
		t.Fatal("personnel catalog lost its own authorized choices")
	}
	must(t, block.Commit(f.ctx))
	if <-done != 200 {
		t.Fatal("country reader did not complete after release")
	}
}

func TestPersonnelAddressTypes(t *testing.T) {
	f := personnelFixture(t)
	hr, manager := f.login("admin", f.password), f.login("reader", f.readPassword)
	const catalog = "/api/v1/person-address-types"
	type M = map[string]any
	exec := func(sql string, args ...any) { t.Helper(); _, err := f.admin.Exec(f.ctx, sql, args...); must(t, err) }
	person := hr.call("POST", persons, M{"submissionId": foundation.NewID(), "name": "合成地址试点"}, "", 200)
	path := persons + "/" + person["id"].(string)
	detail := func() M { return hr.call("GET", path, nil, "", 200) }
	candidates := func() []any { return hr.call("GET", path+"/address-type-candidates", nil, "", 200)["items"].([]any) }
	manager.call("GET", catalog, nil, "", 403)
	exec("delete from access.person_grant where instance_id=$1 and principal_id=$2", f.instance, manager.subject)
	exec("insert into access.person_grant(instance_id,principal_id,action,purpose,fields) values($1,$2,'address-type.read','address-types','{}')", f.instance, manager.subject)
	manager.call("GET", path, nil, "", 404)
	create := M{"submissionId": foundation.NewID(), "code": "DORM", "name": "宿舍地址", "sortOrder": 15, "firstDay": "1900-01-01", "selectable": true}
	manager.call("POST", catalog, create, "", 403)
	exec("insert into access.person_grant(instance_id,principal_id,action,purpose,fields) values($1,$2,'address-type.maintain','address-types','{}')", f.instance, manager.subject)
	exec("delete from access.person_grant where instance_id=$1 and principal_id=$2 and purpose='address-types'", f.instance, hr.subject)
	hr.call("GET", catalog, nil, "", 403)
	manager.call("GET", path+"/address-type-candidates", nil, "", 404)
	list := manager.call("GET", catalog, nil, "", 200)
	if len(candidates()) != 4 || len(list["items"].([]any)) != 4 || list["defaultFirstDay"] != "1900-01-01" {
		t.Fatal("initial catalog")
	}
	getItem := func() M { return detail()["addresses"].([]any)[0].(M) }
	post := func(edit M, other M, status int) M {
		t.Helper()
		in := M{"submissionId": foundation.NewID(), "expectedSourceVersion": detail()["sourceVersion"], "addresses": []M{edit}}
		if other != nil {
			in["basic"] = other
		}
		return hr.call("POST", path+"/corrections", in, "", status)
	}
	post(M{"fields": M{"addressType": "居住地址", "addressCountry": "原文", "street": "一号"}}, nil, 200)
	row := getItem()
	rowEdit := func(fields M) M {
		current := getItem()
		return M{"itemId": current["itemId"], "expectedRevision": current["revision"], "fields": fields}
	}
	// Emulate the exact nullable representation of a pre-catalog payload. An unrelated correction must keep it unbound.
	exec("update personnel.profile_item_payload set address_type_id=null,address_type_revision=null,address_type_day=null where instance_id=$1 and item_id=$2", f.instance, row["itemId"])
	post(rowEdit(M{"street": "二号"}), nil, 200)
	if getItem()["addressTypeAdoption"] != nil || getItem()["fields"].(M)["addressType"] != "居住地址" {
		t.Fatal("invented legacy adoption")
	}
	first := manager.call("POST", catalog, create, "", 200)
	if !reflect.DeepEqual(first, manager.call("POST", catalog, create, "", 200)) {
		t.Fatal("create replay")
	}
	sameKey := maps.Clone(create)
	sameKey["name"] = "别名"
	manager.call("POST", catalog, sameKey, "", 409)
	duplicate := maps.Clone(create)
	duplicate["submissionId"] = foundation.NewID()
	manager.call("POST", catalog, duplicate, "", 400)
	choices := candidates()
	if len(choices) != 5 || choices[1].(M)["id"] != first["id"] {
		t.Fatal("eligible ordering")
	}
	choose := func(x M) M {
		v := rowEdit(M{})
		v["addressTypeChoice"] = M{"id": x["id"], "revision": x["revision"]}
		return v
	}
	future := maps.Clone(create)
	future["submissionId"] = foundation.NewID()
	future["code"] = "FUTURE"
	future["firstDay"] = "9999-12-31"
	unavailable := manager.call("POST", catalog, future, "", 200)
	if len(candidates()) != 5 {
		t.Fatal("future option")
	}
	post(choose(unavailable), nil, 409)
	post(choose(first), nil, 200)
	adopted := getItem()["addressTypeAdoption"].(M)
	if adopted["revision"] != first["revision"] || adopted["id"] != first["id"] || adopted["date"] == "" {
		t.Fatal("missing adoption")
	}
	source := detail()["sourceVersion"]
	// Any extra profile rewrite, including a same-value UPDATE, changes the physical row versions.
	type physical struct{ address, profile, revision int }
	count := func() physical {
		return physical{f.scalar("select count(*) from personnel.address_type"), f.scalar("select count(*) from personnel.profile_item_payload"), f.scalar("select count(*) from personnel.address_type_revision")}
	}
	profileFootprint := func() string {
		var out string
		must(t, f.admin.QueryRow(f.ctx, "select md5(string_agg(xmin::text,',' order by version,item_id)) from personnel.profile_item_payload").Scan(&out))
		return out
	}
	sizes := func() [4]int {
		return [4]int{f.scalar("select pg_table_size('personnel.address_type')"), f.scalar("select pg_indexes_size('personnel.address_type')"),
			f.scalar("select pg_table_size('personnel.address_type_revision')"), f.scalar("select pg_indexes_size('personnel.address_type_revision')")}
	}
	before, unchanged := count(), profileFootprint()
	sizeBefore := sizes()
	change := func(x M, name string, active bool) M {
		return M{"submissionId": foundation.NewID(), "id": x["id"], "expectedRevision": x["revision"], "name": name, "sortOrder": x["sortOrder"], "selectable": active}
	}
	rename := change(first, "宿舍通信地址", true)
	rename["sortOrder"] = 5
	second := manager.call("POST", catalog, rename, "", 200)
	if candidates()[0].(M)["id"] != first["id"] {
		t.Fatal("reorder did not affect new candidates")
	}
	after := count()
	sizeAfter := sizes()
	t.Logf("rename storage bytes [current table,indexes,revision table,indexes]: %v -> %v", sizeBefore, sizeAfter)
	if after.address != before.address || after.profile != before.profile || after.revision != before.revision+1 || profileFootprint() != unchanged {
		t.Fatal("rename touched other facts")
	}
	if getItem()["fields"].(M)["addressType"] != "宿舍地址" {
		t.Fatal("rewrote original label")
	}
	manager.call("POST", catalog, change(first, "过期覆盖", false), "", 409)
	manager.call("POST", catalog, change(second, "宿舍通信地址", true), "", 400)
	stale := post(M{"addressTypeChoice": M{"id": first["id"], "revision": first["revision"]}, "fields": M{"addressCountry": "原文", "street": "新增地址"}}, M{"formerName": "必须回滚"}, 409)
	if stale["field"] != "addresses.0.addressType" || detail()["sourceVersion"] != source {
		t.Fatal("failed whole-page atomicity", stale["field"])
	}
	originalSource := manager.call("GET", catalog+"/"+first["id"].(string)+"/revisions/"+first["revision"].(string), nil, "", 200)
	if originalSource["name"] != first["name"] || originalSource["actorId"] != manager.subject || originalSource["recordedAt"] == nil {
		t.Fatal("original revision evidence")
	}
	hr.call("GET", catalog+"/"+first["id"].(string)+"/revisions/"+first["revision"].(string), nil, "", 403)
	stopped := manager.call("POST", catalog, change(second, "宿舍通信地址", false), "", 200)
	if len(candidates()) != 4 {
		t.Fatal("stopped option")
	}
	keep := choose(first)
	keep["fields"] = M{"street": "三号"}
	post(keep, nil, 200)
	if !reflect.DeepEqual(getItem()["addressTypeAdoption"], adopted) {
		t.Fatal("old adoption changed")
	}
	reopened := manager.call("POST", catalog, change(stopped, "宿舍通信地址", true), "", 200)
	post(choose(reopened), nil, 200)
	if x := getItem(); x["fields"].(M)["addressType"] != "宿舍通信地址" || x["addressTypeAdoption"].(M)["revision"] != reopened["revision"] {
		t.Fatal("restore adoption")
	}
	receipt := catalog + "/submissions/" + create["submissionId"].(string) + "/resolve"
	if !reflect.DeepEqual(manager.call("POST", receipt, nil, "", 200)["item"], first) {
		t.Fatal("receipt returned current revision")
	}
	closed := maps.Clone(create)
	closed["submissionId"] = foundation.NewID()
	closed["code"] = "LATE"
	if manager.call("POST", catalog+"/submissions/"+closed["submissionId"].(string)+"/resolve", nil, "", 200)["outcome"] != "not_applied" {
		t.Fatal("resolution fence")
	}
	manager.call("POST", catalog, closed, "", 409)
	// Hold publication after the new choice was locked. A configuration writer cannot pass that choice.
	beforeLock := detail()
	choice := choose(first)
	choice["addressTypeChoice"] = M{"id": choices[0].(M)["id"], "revision": choices[0].(M)["revision"]}
	saving := M{"submissionId": foundation.NewID(), "expectedSourceVersion": beforeLock["sourceVersion"], "addresses": []M{choice}}
	barrier, err := f.admin.Begin(f.ctx)
	must(t, err)
	defer barrier.Rollback(f.ctx)
	_, err = barrier.Exec(f.ctx, "lock table personnel.source in share mode")
	must(t, err)
	done := make(chan int, 1)
	go func() { status, _, _ := hr.response("POST", path+"/corrections", saving, ""); done <- status }()
	deadline := time.Now().Add(15 * time.Second)
	for f.scalar("select count(*) from pg_stat_activity where usename='chnhcm_api' and $1::int=any(pg_blocking_pids(pid))", barrier.Conn().PgConn().PID()) == 0 {
		if time.Now().After(deadline) {
			t.Fatal("save did not reach publication")
		}
		time.Sleep(10 * time.Millisecond)
	}
	_, err = f.admin.Exec(f.ctx, "select 1 from personnel.address_type where instance_id=$1 and item_id=$2 for update nowait", f.instance, choices[0].(M)["id"])
	var locked *pgconn.PgError
	if !errors.As(err, &locked) || locked.Code != "55P03" {
		t.Fatal("chosen configuration revision not guarded at publication", err)
	}
	must(t, barrier.Commit(f.ctx))
	if <-done != 200 {
		t.Fatal("guarded adoption failed")
	}
	// Revoke this row's field between candidate read and the save.
	revoked := choose(first)
	exec("insert into access.person_grant(instance_id,principal_id,action,purpose,fields,item_id,denied) values($1,$2,'profile.correct','personal-profile','{addressType}',$3,true)", f.instance, hr.subject, row["itemId"])
	hr.call("GET", path+"/address-type-candidates?itemId="+row["itemId"].(string), nil, "", 403)
	beforeRevoked := detail()["sourceVersion"]
	post(revoked, M{"formerName": "不能保存"}, 403)
	if detail()["sourceVersion"] != beforeRevoked {
		t.Fatal("revoked save wrote data")
	}
	exec("insert into access.person_grant(instance_id,principal_id,action,purpose,fields,item_id,denied) values($1,$2,'profile.read','personal-profile','{addressType}',$3,true)", f.instance, hr.subject, row["itemId"])
	if detail()["addresses"] != nil {
		t.Fatal("row reference or label was visible after field revocation")
	}
	hr.call("GET", path+"/address-type-candidates?businessAt=2020-01-01", nil, "", 400)
	other := foundation.NewID()
	exec("select personnel.initialize_address_types($1)", other)
	var foreignID, foreignHead string
	must(t, f.admin.QueryRow(f.ctx, "select item_id,head from personnel.address_type where instance_id=$1 limit 1", other).Scan(&foreignID, &foreignHead))
	post(M{"addressTypeChoice": M{"id": foreignID, "revision": foreignHead}, "fields": M{"street": "一号", "addressCountry": "原文"}}, nil, 409)
	exec("delete from access.person_grant where instance_id=$1 and principal_id=$2 and action='address-type.read'", f.instance, manager.subject)
	manager.call("POST", receipt, nil, "", 403)
	rows, err := f.admin.Query(f.ctx, "explain (analyze,costs off) select item_id,head,code,name from personnel.address_type where instance_id=$1 and valid_during @> current_date order by sort_order,item_id", f.instance)
	must(t, err)
	var plan []string
	for rows.Next() {
		var line string
		must(t, rows.Scan(&line))
		plan = append(plan, line)
	}
	must(t, rows.Err())
	rows.Close()
	t.Logf("rename updated one item; revision inserts=%d; profile changes=0; current table+indexes=%d bytes; revision table+indexes=%d bytes; candidates plan=%s", after.revision-before.revision, f.scalar("select pg_total_relation_size('personnel.address_type')"), f.scalar("select pg_total_relation_size('personnel.address_type_revision')"), strings.Join(plan, " | "))
	// Concurrent creation of one code returns just one success.
	exec("insert into access.person_grant(instance_id,principal_id,action,purpose,fields) values($1,$2,'address-type.read','address-types','{}')", f.instance, manager.subject)
	race := func(inputs []M, want int) {
		t.Helper()
		start := make(chan struct{})
		results := make(chan int, len(inputs))
		for _, v := range inputs {
			go func() {
				<-start
				code, _, err := manager.response("POST", catalog, v, "")
				if err != nil {
					code = 0
				}
				results <- code
			}()
		}
		close(start)
		wins, losers := 0, 0
		for range inputs {
			code := <-results
			if code == 200 {
				wins++
			}
			if code == want {
				losers++
			}
		}
		if wins != 1 || losers != len(inputs)-1 {
			t.Fatal(fmt.Sprintf("race success=%d expected rejections=%d", wins, losers))
		}
	}
	a, b := maps.Clone(create), maps.Clone(create)
	a["submissionId"], b["submissionId"] = foundation.NewID(), foundation.NewID()
	a["code"], b["code"] = "RACE", "RACE"
	race([]M{a, b}, 400)
	race([]M{change(reopened, "并发甲", true), change(reopened, "并发乙", true)}, 409)
	// A rejected revision insert rolls back the current row as well. Retrying after a transient failure adds one revision.
	failed := maps.Clone(create)
	failed["submissionId"] = foundation.NewID()
	failed["code"] = "ROLLBACK"
	exec(`create function personnel.test_address_type_failure() returns trigger language plpgsql as $$begin raise exception 'synthetic revision failure'; end$$;
 create trigger test_address_type_failure before insert on personnel.address_type_revision for each row execute function personnel.test_address_type_failure()`)
	manager.call("POST", catalog, failed, "", 503)
	if f.scalar("select count(*) from personnel.address_type where instance_id=$1 and code='ROLLBACK'", f.instance) != 0 {
		t.Fatal("partial configuration write")
	}
	exec("drop trigger test_address_type_failure on personnel.address_type_revision; drop function personnel.test_address_type_failure()")
	manager.call("POST", catalog, failed, "", 200)

}

func TestPersonnelAddressTypePeriods(t *testing.T) {
	f := personnelFixture(t)
	admin, reader := f.login("admin", f.password), f.login("reader", f.readPassword)
	const catalog = "/api/v1/person-address-types"
	type M = map[string]any
	person := admin.call("POST", persons, M{"submissionId": foundation.NewID(), "name": "期间样本"}, "", 200)
	candidates := persons + "/" + person["id"].(string) + "/address-type-candidates"
	today := time.Now().In(foundation.Shanghai)
	stamp := func(offset int) string { return today.AddDate(0, 0, offset).Format("2006-01-02") }
	create := M{"submissionId": foundation.NewID(), "code": "PERIOD", "name": "宿舍地址", "sortOrder": 1, "firstDay": stamp(-3), "stopDay": stamp(1)}
	initial := admin.call("POST", catalog, create, "", 200)
	uri := catalog + "/" + initial["id"].(string)
	has := func() bool {
		for _, o := range admin.call("GET", candidates, nil, "", 200)["items"].([]any) {
			if o.(M)["id"] == initial["id"] {
				return true
			}
		}
		return false
	}
	read := func() M { return admin.call("GET", uri, nil, "", 200) }
	edit := func(base M, day string, want int) M {
		return admin.call("POST", catalog, M{"submissionId": foundation.NewID(), "id": base["id"], "expectedRevision": base["revision"], "name": base["name"], "sortOrder": 1, "stopDay": day}, "", want)
	}
	if v := read(); v["item"].(M)["state"] != "active" || v["selectedDay"] != stamp(-3) || v["timeline"].([]any)[1].(M)["state"] != "inactive" || !has() {
		t.Fatal("future stop suppressed current adoption", v)
	}
	reader.call("GET", uri, nil, "", 403)
	// Grant the observer catalog reads alone, including timeline, with no maintenance.
	_, e := f.admin.Exec(f.ctx, "insert into access.person_grant(instance_id,principal_id,action,purpose,fields) values($1,$2,'address-type.read','address-types','{}')", f.instance, reader.subject)
	must(t, e)
	if v := reader.call("GET", uri, nil, "", 200); v["canMaintain"] != false || len(v["timeline"].([]any)) != 2 {
		t.Fatal("read-only projection", v)
	}
	reader.call("POST", catalog, M{"submissionId": foundation.NewID(), "id": initial["id"], "expectedRevision": initial["revision"], "name": "越权", "sortOrder": 1, "stopDay": ""}, "", 403)
	edit(initial, stamp(-4), 400)
	edit(initial, "2027-02-30", 400)
	stopped := edit(initial, stamp(0), 200)
	if v := read(); v["item"].(M)["state"] != "inactive" || v["selectedDay"] != stamp(0) || has() {
		t.Fatal("stop day included", v)
	}
	admin.call("POST", persons+"/"+person["id"].(string)+"/corrections", M{"submissionId": foundation.NewID(), "expectedSourceVersion": person["sourceVersion"],
		"addresses": []M{{"fields": M{"addressCountry": "合成地区", "street": "一号"}, "addressTypeChoice": M{"id": initial["id"], "revision": stopped["revision"]}}}}, "", 409)
	edit(initial, "", 409)
	source := admin.call("GET", uri+"/revisions/"+initial["revision"].(string), nil, "", 200)
	receipt := admin.call("POST", catalog+"/submissions/"+create["submissionId"].(string)+"/resolve", nil, "", 200)
	if source["stopDay"] != stamp(1) || !reflect.DeepEqual(initial, receipt["item"]) {
		t.Fatal("original scheduled stop lost")
	}
	corrected := edit(stopped, "", 200)
	if v := read(); v["item"].(M)["state"] != "active" || v["item"].(M)["stopDay"] != nil || len(v["timeline"].([]any)) != 1 || !has() || corrected["id"] != initial["id"] {
		t.Fatal("correction lost identity or period", v)
	}
	create["submissionId"], create["code"], create["firstDay"], create["stopDay"] = foundation.NewID(), "FUTURE-PERIOD", stamp(2), stamp(2)
	future := admin.call("POST", catalog, create, "", 200)
	v := admin.call("GET", catalog+"/"+future["id"].(string), nil, "", 200)
	if v["item"].(M)["state"] != "not_effective" || len(v["timeline"].([]any)) != 1 || v["timeline"].([]any)[0].(M)["state"] != "inactive" {
		t.Fatal("empty interval boundary", v)
	}
}

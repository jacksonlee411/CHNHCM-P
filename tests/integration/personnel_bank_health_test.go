//go:build integration

package integration_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"chnhcm.local/chnhcm/db/migrations"
	"chnhcm.local/chnhcm/internal/bootstrap"
	fnd "chnhcm.local/chnhcm/internal/foundation"
)

func TestPersonnelBankHealth(t *testing.T) {
	type M = map[string]any
	f := personnelFixture(t)
	admin := f.login("admin", f.password)
	reader := f.login("reader", f.readPassword)
	register := func(name string) M {
		return admin.call("POST", persons, M{"name": name, "submissionId": fnd.NewID()}, "", 200)
	}
	who := register("账户健康甲")
	other := register("账户健康乙")
	id := who["id"].(string)
	path := persons + "/" + id
	read := func(b *browser, view string) M { return b.call("GET", path+view, nil, "", 200) }
	exec := func(sql string, args ...any) { _, e := f.admin.Exec(f.ctx, sql, args...); must(t, e) }
	save := func(cmd M, status int) M {
		cmd["expectedSourceVersion"] = read(admin, "")["sourceVersion"]
		cmd["submissionId"] = fnd.NewID()
		return admin.call("POST", path+"/corrections", cmd, "", status)
	}
	bank := func(number string) M {
		return M{"bankName": "合成银行", "accountHolder": "合成户名", "currency": "CNY", "accountNumber": number}
	}
	fields := func(x M) M { return x["fields"].(M) }
	first := bank("001 ab")
	first["bankPrimary"] = "yes"
	second := bank("002 CD")
	saved := save(M{"bankAccounts": []M{{"fields": first}, {"fields": second}}, "healthSummary": "合成摘要甲", "basic": M{"formerName": "原档"}}, 200)
	accounts := func(v M) []any { return v["bankAccounts"].([]any) }
	base := read(admin, "")
	rows := accounts(base)
	a, b := rows[0].(M), rows[1].(M)
	item := func(x M) string { return x["itemId"].(string) }
	update := func(x, patch M) M { return M{"itemId": item(x), "expectedRevision": x["revision"], "fields": patch} }
	if len(rows) != 2 || item(a) == item(b) || base["healthSummary"] != "合成摘要甲" {
		t.Fatal("missing independent identities or health")
	}
	for _, view := range []string{"", "?view=edit", "?view=bank-" + item(a)} {
		rows = accounts(read(admin, view))
		want := "********"
		if strings.HasPrefix(view, "?view=bank-") {
			want = "001 ab"
		}
		if fields(rows[0].(M))["accountNumber"] != want || fields(rows[1].(M))["accountNumber"] != "********" {
			t.Fatal("incorrect per-account projection", view)
		}
	}
	assertPrivate := func(v M, markers ...string) {
		t.Helper()
		buf, err := json.Marshal(v)
		must(t, err)
		for _, s := range markers {
			if strings.Contains(string(buf), s) {
				t.Fatalf("projection exposed %s", s)
			}
		}
	}
	assertPrivate(read(reader, "?view=bank-"+item(a)), "bankAccounts", "bankName", "accountNumber", "healthSummary", "合成摘要甲")
	assertPrivate(read(admin, "/sources/"+saved["sourceVersion"].(string)), "bankAccounts", "bankName", "healthSummary", "合成摘要甲")
	// Restricted bank readers see a fixed mask, including on a direct single-account read.
	exec("insert into access.person_grant(instance_id,principal_id,action,purpose,fields,person_ids) values($1,$2,'bank.read','personal-bank',array['accountNumber.masked'],array[$3::uuid])", f.instance, reader.subject, id)
	masked := read(reader, "?view=bank-"+item(a))
	if len(accounts(masked)) != 2 || fields(accounts(masked)[0].(M))["accountNumber"] != "********" {
		t.Fatal("masked grant not honored")
	}
	assertPrivate(masked, "001 ab", "002 CD", "bankName", "accountHolder", "healthSummary")
	exec("insert into access.person_grant(instance_id,principal_id,action,purpose,fields,item_id,denied) values($1,$2,'bank.read','personal-bank',array['accountNumber.masked'],$3,true)", f.instance, reader.subject, item(a))
	restricted := accounts(read(reader, "?view=bank-"+item(a)))
	if len(restricted) != 1 || item(restricted[0].(M)) != item(b) {
		t.Fatal("row with no readable bank field remained visible")
	}
	// Scope health use to a single person, even though general directory access includes both.
	exec("update access.person_grant set person_ids=array[$3::uuid] where instance_id=$1 and principal_id=$2 and purpose='health-summary'", f.instance, admin.subject, id)
	foreignPath := persons + "/" + other["id"].(string)
	assertPrivate(admin.call("GET", foreignPath, nil, "", 200), "healthSummary")
	admin.call("POST", foreignPath+"/corrections", M{"submissionId": fnd.NewID(), "expectedSourceVersion": other["sourceVersion"], "healthSummary": "合成乙"}, "", 403)
	admin.call("POST", foreignPath+"/corrections", M{"submissionId": fnd.NewID(), "expectedSourceVersion": other["sourceVersion"], "bankAccounts": []M{update(b, M{"bankPrimary": "yes"})}}, "", 403)
	duplicate := bank("001AB")
	duplicate["currency"] = "USD"
	conflict := save(M{"bankAccounts": []M{{"fields": duplicate}}, "healthSummary": "不能写入"}, 409)
	if conflict["field"] != "bankAccounts.0.accountNumber" {
		t.Fatal("missing location of the submitted account")
	}
	assertPrivate(conflict, item(a), "001 ab", "合成银行")
	// Reassigning the primary affects the old row as well as the explicitly selected row.
	exec("insert into access.person_grant(instance_id,principal_id,action,purpose,fields,item_id,denied) values($1,$2,'bank.correct','personal-bank',array['bankPrimary'],$3,true)", f.instance, admin.subject, item(a))
	cmd := M{"basic": M{"formerName": "新档"}, "addresses": []M{{"fields": M{"addressType": "居住地址", "addressCountry": "原文", "street": "合成街"}}}, "bankAccounts": []M{update(b, M{"bankPrimary": " yes "})}, "healthSummary": "合成摘要乙"}
	save(cmd, 403)
	exec("delete from access.person_grant where instance_id=$1 and item_id=$2", f.instance, item(a))
	exec("update access.person_data_purpose set valid_until=now() where instance_id=$1 and purpose='health-summary'", f.instance)
	save(M{"bankAccounts": []M{{"fields": bank("003")}}, "healthSummary": "用途到期"}, 403)
	if f.scalar("select count(*) from personnel.bank_account where person_id=$1", id) != 2 {
		t.Fatal("rejected collection left an account identity")
	}
	denied := save(cmd, 403)
	if denied["field"] != nil {
		t.Fatal("revealing denied field")
	}
	assertPrivate(read(admin, "?view=edit"), "healthSummary", "合成摘要")
	if f.scalar("select count(*) from personnel.source where person_id=$1", id) != 2 || f.scalar("select count(*) from personnel.outbox where person_id=$1", id) != 2 {
		t.Fatal("partial source or event")
	}
	exec("update access.person_data_purpose set valid_until='infinity' where instance_id=$1 and purpose='health-summary'", f.instance)
	if !reflect.DeepEqual(read(admin, ""), base) {
		t.Fatal("partial write during denial")
	}
	saved = save(cmd, 200)
	fresh := read(f.login("admin", f.password), "")
	rows = accounts(fresh)
	if len(rows) != 2 || item(rows[0].(M)) != item(a) || item(rows[1].(M)) != item(b) || fields(rows[0].(M))["bankPrimary"] != nil || fields(rows[1].(M))["bankPrimary"] != "yes" || fresh["healthSummary"] != "合成摘要乙" || fresh["basic"].(M)["formerName"] != "新档" || len(fresh["addresses"].([]any)) != 1 {
		t.Fatal("mixed correction failed")
	}
	if f.scalar("select count(*) from personnel.bank_account where person_id=$1", id) != 2 || f.scalar("select count(*) from personnel.outbox where person_id=$1 and action not in ('register','correct')", id) != 0 {
		t.Fatal("unexpected account identity or publication")
	}
	// Correct a mistyped number on the same account. A mask must never replace the stored value.
	for _, mask := range []string{"********", "**** ****"} {
		save(M{"bankAccounts": []M{update(rows[1].(M), M{"accountNumber": mask})}}, 400)
	}
	save(M{"bankAccounts": []M{update(rows[1].(M), M{"accountNumber": "002 CE"})}}, 200)
	corrected := accounts(read(admin, "?view=bank-"+item(b)))[1].(M)
	if item(corrected) != item(b) || corrected["revision"] == rows[1].(M)["revision"] || fields(corrected)["accountNumber"] != "002 CE" {
		t.Fatal("number correction did not preserve the account identity and advance its revision")
	}
	save(M{"healthSummary": ""}, 200)
	if read(admin, "")["healthSummary"] != "" || f.scalar("select count(distinct ordinary_until) from personnel.profile_item_payload where person_id=$1 and section='health'", id) != 1 {
		t.Fatal("clear or retention clock")
	}
	save(M{"bankAccounts": []M{{"itemId": item(a), "expectedRevision": rows[0].(M)["revision"], "remove": true}}}, 200)
	if len(accounts(read(admin, ""))) != 1 || f.scalar("select count(*) from personnel.bank_account where person_id=$1", id) != 2 {
		t.Fatal("removed account lost historical identity")
	}
	exec("update personnel.profile_item_payload set ordinary_until=now()-interval '2 days',evidence_until=now()-interval '1 day' where person_id=$1 and section in ('bankAccounts','health')", id)
	must(t, bootstrap.Migrate(f.ctx, f.admin, migrations.Files))
	if f.scalar("select count(*) from personnel.profile_item_payload where person_id=$1 and section in ('bankAccounts','health')", id) != 0 || f.scalar("select count(distinct category) from personnel.disposition where person_id=$1", id) != 2 {
		t.Fatal("disposition incomplete")
	}
	assertPrivate(admin.call("POST", persons+"/submissions/"+cmd["submissionId"].(string)+"/resolve", nil, "", 200), "bankAccounts", "healthSummary")
}

// A purpose policy edit can delay a detail read beyond the expiry of the reader's grant.
func TestPersonnelHealthReadAfterPolicyWait(t *testing.T) {
	type M = map[string]any
	f := personnelFixture(t)
	admin := f.login("admin", f.password)
	reader := f.login("reader", f.readPassword)
	who := admin.call("POST", persons, M{"submissionId": fnd.NewID(), "name": "健康用途等待"}, "", 200)
	path := persons + "/" + who["id"].(string)
	admin.call("POST", path+"/corrections", M{"submissionId": fnd.NewID(), "expectedSourceVersion": who["sourceVersion"], "healthSummary": "合成摘要"}, "", 200)
	tx, err := f.admin.Begin(f.ctx)
	must(t, err)
	defer tx.Rollback(f.ctx)
	_, err = tx.Exec(f.ctx, "select 1 from access.person_data_purpose where instance_id=$1 and purpose='health-summary' for update", f.instance)
	must(t, err)
	var expiry time.Time
	must(t, f.admin.QueryRow(f.ctx, `insert into access.person_grant(instance_id,principal_id,action,purpose,fields,person_ids,expires_at)
	 values($1,$2,'health.read','health-summary',array['healthSummary'],array[$3::uuid],clock_timestamp()+interval '2 seconds') returning expires_at`, f.instance, reader.subject, who["id"]).Scan(&expiry))
	result := make(chan error, 1)
	go func() {
		status, detail, err := reader.response("GET", path, nil, "")
		if err == nil && (status != 200 || detail["healthSummary"] != nil) {
			err = fmt.Errorf("expired health grant after policy wait, HTTP %d", status)
		}
		result <- err
	}()
	for f.scalar("select count(*) from pg_stat_activity where $1::int=any(pg_blocking_pids(pid))", tx.Conn().PgConn().PID()) == 0 {
		if time.Now().After(expiry) {
			t.Fatal("detail did not reach policy lock before grant expiry")
		}
		time.Sleep(10 * time.Millisecond)
	}
	_, err = tx.Exec(f.ctx, "select pg_sleep(greatest(0,extract(epoch from ($1::timestamptz-clock_timestamp()))))", expiry)
	must(t, err)
	must(t, tx.Commit(f.ctx))
	must(t, <-result)
}

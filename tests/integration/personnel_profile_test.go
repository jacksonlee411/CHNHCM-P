//go:build integration

package integration_test

import (
	"reflect"
	"testing"

	"chnhcm.local/chnhcm/internal/foundation"
)

func TestPersonnelWholeProfile(t *testing.T) {
	type M = map[string]any
	f := personnelFixture(t)
	hr := f.login("admin", f.password)
	reader := f.login("reader", f.readPassword)
	create := func(name, number string) M {
		in := M{"submissionId": foundation.NewID(), "name": name}
		if number != "" {
			in["primaryDocument"] = M{"kind": "passport", "number": number}
		}
		return hr.call("POST", persons, in, "", 200)
	}
	a := create("样本甲", "SYNTH-A")
	other := create("样本乙", "SYNTH-B")
	path := persons + "/" + a["id"].(string)
	get := func() M { return hr.call("GET", path+"?view=edit", nil, "", 200) }
	post := func(in M, status int) M {
		in["submissionId"] = foundation.NewID()
		in["expectedSourceVersion"] = a["sourceVersion"]
		return hr.call("POST", path+"/corrections", in, "", status)
	}
	_, err := f.admin.Exec(f.ctx, "delete from access.person_grant where principal_id=$1 and action='verify'", hr.subject)
	must(t, err)
	initial := get()
	source := a["sourceVersion"]
	input := M{"basic": M{"formerName": "样本曾用名", "birthDate": "2000-02-29"}, "contacts": []M{{"kind": "phone", "value": "12345", "purpose": "工作", "primary": true}, {"kind": "phone", "value": "34567"}, {"kind": "email", "value": "a@example.test", "primary": true}}, "documents": []M{{"fields": M{"documentKind": "passport", "documentNumber": "synth-b", "issuer": "合成签发机关", "issueDate": "2020-01-01", "expiryDate": "2030-01-01"}}}}
	a = post(input, 200)
	replay := hr.call("POST", path+"/corrections", input, "", 200)
	if !reflect.DeepEqual(a, replay) {
		t.Fatal("replay created another source")
	}
	data := get()
	docs := data["documents"].([]any)
	contacts := data["contacts"].([]any)
	if len(docs) != 2 || docs[0].(M)["itemId"] != initial["documents"].([]any)[0].(M)["itemId"] || docs[1].(M)["fields"].(M)["documentNumber"] != "SYNTH-B" || len(contacts) != 3 {
		t.Fatal("missing saved items")
	}
	if hr.call("GET", path+"/sources/"+source.(string), nil, "", 200)["documents"] != nil {
		t.Fatal("profile collection in source")
	}
	row := func(x any, changes M) M {
		v := x.(M)
		changes["itemId"] = v["itemId"]
		changes["expectedRevision"] = v["revision"]
		return changes
	}
	oldCount := f.scalar("select count(*) from personnel.outbox where person_id=$1", a["id"])
	conflict := post(M{"documents": []M{row(docs[1], M{"primary": true})}, "contacts": []M{row(contacts[0], M{"kind": "phone", "value": "54321"})}, "basic": M{"formerName": "不可提交"}}, 409)
	if conflict["existingPersonId"] != nil || !reflect.DeepEqual(data, get()) || f.scalar("select count(*) from personnel.outbox where person_id=$1", a["id"]) != oldCount {
		t.Fatal("conflict changed profile or revealed owner")
	}
	marker := foundation.NewID()
	_, err = f.admin.Exec(f.ctx, `insert into access.person_grant(instance_id,principal_id,action,purpose,fields,grant_id,item_id,denied) values($1,$2,'profile.correct','personal-profile',array['contactPrimary'],$3,$4,true)`, f.instance, hr.subject, marker, contacts[0].(M)["itemId"])
	must(t, err)
	post(M{"contacts": []M{row(contacts[1], M{"primary": true})}, "basic": M{"formerName": "不可提交"}}, 403)
	if get()["sourceVersion"] != a["sourceVersion"] {
		t.Fatal("indirect primary switch modified restricted row")
	}
	_, err = f.admin.Exec(f.ctx, "delete from access.person_grant where grant_id=$1", marker)
	must(t, err)
	a = post(M{"contacts": []M{row(contacts[1], M{"primary": true})}, "basic": M{"formerName": ""}}, 200)
	data = get()
	contacts = data["contacts"].([]any)
	if contacts[0].(M)["primary"] != nil || contacts[1].(M)["primary"] != true || contacts[2].(M)["primary"] != true || data["basic"].(M)["formerName"] != "" {
		t.Fatal("primary switch affected other channel")
	}
	denied := func(action, field string) {
		_, e := f.admin.Exec(f.ctx, "insert into access.person_grant(instance_id,principal_id,action,purpose,fields,person_ids,denied) values($1,$2,$3,'personal-profile',array[$4::text],array[$5::uuid],true)", f.instance, hr.subject, action, field, a["id"])
		must(t, e)
	}
	denied("profile.correct", "issuer")
	post(M{"documents": []M{row(docs[1], M{"remove": true})}, "basic": M{"formerName": "不可提交"}}, 403)
	denied("profile.read", "birthDate")
	hidden := get()["basic"].(M)
	if _, ok := hidden["birthDate"]; ok {
		t.Fatal("hidden date returned")
	}
	post(M{"basic": M{"birthDate": "1999-01-01"}}, 403)
	post(M{"documents": []M{row(hr.call("GET", persons+"/"+other["id"].(string), nil, "", 200)["documents"].([]any)[0], M{"fields": M{"issuer": "无权"}})}}, 403)
	masked := reader.call("GET", path+"?view=document-"+docs[1].(M)["itemId"].(string), nil, "", 200)
	if masked["documents"].([]any)[1].(M)["fields"].(M)["documentNumber"] != "********" {
		t.Fatal("masked reader received full number")
	}
	if data["sourceVersion"] != get()["sourceVersion"] {
		t.Fatal("refused patch created source")
	}
	post(M{"name": ""}, 400)
	post(M{"basic": M{"birthPlace": "测试"}, "proofReference": "文件"}, 400)

	// Fail after preparing all three sections, before the final event is stored.
	beforeFault := get()
	submission := foundation.NewID()
	retry := M{"submissionId": submission, "expectedSourceVersion": a["sourceVersion"],
		"basic":     M{"englishName": "Retry Profile"},
		"documents": []M{row(docs[1], M{"fields": M{"documentName": "重试样本"}})},
		"contacts":  []M{row(contacts[1], M{"kind": "phone", "value": "98765"})}}
	_, err = f.admin.Exec(f.ctx, `create function personnel.fail_profile_event() returns trigger language plpgsql as $$ begin raise exception 'test'; end $$;
		create trigger fail_profile_event before insert on personnel.outbox for each row execute function personnel.fail_profile_event()`)
	must(t, err)
	hr.call("POST", path+"/corrections", retry, "", 503)
	if !reflect.DeepEqual(beforeFault, get()) || f.scalar("select count(*) from personnel.terminal where submission_id=$1", submission) != 0 {
		t.Fatal("database failure left a partial profile or a terminal result")
	}
	_, err = f.admin.Exec(f.ctx, "drop trigger fail_profile_event on personnel.outbox")
	must(t, err)
	result := hr.call("POST", path+"/corrections", retry, "", 200)
	if result["sourceVersion"] == a["sourceVersion"] || !reflect.DeepEqual(result, hr.call("POST", path+"/corrections", retry, "", 200)) {
		t.Fatal("retry failed to recover a single accepted result")
	}
	recovered := get()
	if recovered["basic"].(M)["englishName"] != "Retry Profile" || recovered["documents"].([]any)[1].(M)["fields"].(M)["documentName"] != "重试样本" || recovered["contacts"].([]any)[1].(M)["value"] != "98765" {
		t.Fatal("retried sections did not become visible together")
	}
}

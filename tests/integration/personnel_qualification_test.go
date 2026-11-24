//go:build integration

package integration_test

import (
	"maps"
	"reflect"
	"testing"

	"chnhcm.local/chnhcm/db/migrations"
	"chnhcm.local/chnhcm/internal/bootstrap"
	fnd "chnhcm.local/chnhcm/internal/foundation"
)

func TestPersonnelEducationCredentials(t *testing.T) {
	type M = map[string]any
	f := personnelFixture(t)
	a := f.login("admin", f.password)
	otherSession := f.login("admin", f.password)
	reader := f.login("reader", f.readPassword)
	register := func(name string) M {
		return a.call("POST", persons, M{"submissionId": fnd.NewID(), "name": name}, "", 200)
	}
	person := register("教育凭证经办样本")
	id := person["id"].(string)
	path := persons + "/" + id
	get := func() M { return a.call("GET", path+"?view=edit", nil, "", 200) }
	send := func(body M, status int) M {
		body["submissionId"] = fnd.NewID()
		body["expectedSourceVersion"] = get()["sourceVersion"]
		return a.call("POST", path+"/corrections", body, "", status)
	}
	sql := func(q string, args ...any) { _, err := f.admin.Exec(f.ctx, q, args...); must(t, err) }
	count := func() string {
		var digest string
		must(t, f.admin.QueryRow(f.ctx, `select concat_ws(',',(select head from personnel.person where person_id=$1),(select count(*) from personnel.source where person_id=$1),(select count(*) from personnel.profile_item_payload where person_id=$1),(select count(*) from personnel.outbox where person_id=$1))`, id).Scan(&digest))
		return digest
	}
	row := func(fields M) M { return M{"fields": fields} }
	change := func(old M, fields M) M {
		return M{"itemId": old["itemId"], "expectedRevision": old["revision"], "fields": fields}
	}
	find := func(data M, group, key, value string) M {
		t.Helper()
		list, _ := data[group].([]any)
		for _, entry := range list {
			r := entry.(M)
			if r["fields"].(M)[key] == value {
				return r
			}
		}
		t.Fatalf("missing %s / %s", group, value)
		return nil
	}
	study := M{"educationType": "学习经历", "school": "在读院校", "completion": "在读", "startMonth": "2025-09"}
	diploma := M{"educationType": "已取得学历", "school": "学历院校", "educationLevel": "本科", "highestEducation": "yes", "endMonth": "2022-06"}
	degree := M{"educationType": "学位授予", "school": "授予院校", "degree": "学士", "awardDate": "2022-06-30", "highestDegree": "yes"}
	cert := M{"credentialKind": "职业资格", "credentialName": "资格样本", "certificateNumber": "SYNTH-CERT-001", "issuedOn": "2023-01-01", "validFrom": "2023-02-01", "validUntil": "2025-01-31"}
	title := M{"credentialKind": "职称", "credentialName": "职称样本", "indefinite": "yes", "issuedOn": "2020-01-01"}
	base := count()
	for _, tc := range []struct {
		group  string
		fields M
		key    string
	}{
		{"education", M{"educationType": "学位授予", "school": "校"}, "degree"},
		{"education", M{"educationType": "已取得学历", "school": "校"}, "educationLevel"},
		{"education", M{"educationType": "学习经历", "school": "校", "startMonth": "2025-13"}, "startMonth"},
		{"education", M{"educationType": "学习经历", "school": "校", "completion": "虚构状态"}, "completion"},
		{"education", M{"educationType": "学习经历", "school": "校", "completion": "在读", "degree": "拟获", "highestDegree": "yes"}, "highestDegree"},
		{"education", M{"educationType": "学习经历", "school": "校", "completion": "在读", "educationLevel": "本科", "highestEducation": "yes"}, "highestEducation"},
		{"credentials", M{"credentialKind": "职业资格", "credentialName": "期限", "validFrom": "2025-03-01", "validUntil": "2025-02-28"}, "validUntil"},
	} {
		fault := send(M{tc.group: []M{row(tc.fields)}, "basic": M{"formerName": "应回滚"}}, 400)
		if fault["field"] != tc.group+".0."+tc.key || count() != base {
			t.Fatal("validation or atomicity", tc.key, fault)
		}
	}
	cmd := M{"education": []M{row(study), row(diploma), row(degree)}, "credentials": []M{row(cert), row(title)}, "basic": M{"formerName": "混合草稿"}}
	sql(`create function personnel.fail_qualification() returns trigger language plpgsql as $$ begin raise exception 'fixture'; end $$;
 create trigger fail_qualification before insert on personnel.outbox for each row execute function personnel.fail_qualification()`)
	send(cmd, 503)
	if count() != base || a.call("POST", persons+"/submissions/"+cmd["submissionId"].(string)+"/resolve", nil, "", 200)["outcome"] != "not_applied" {
		t.Fatal("late failure persisted")
	}
	sql("drop trigger fail_qualification on personnel.outbox")
	saved := send(cmd, 200)
	after := count()
	receipt := otherSession.call("POST", persons+"/submissions/"+cmd["submissionId"].(string)+"/resolve", nil, "", 200)
	if receipt["sourceVersion"] != saved["sourceVersion"] || !reflect.DeepEqual(saved, otherSession.call("POST", path+"/corrections", cmd, "", 200)) || after != count() {
		t.Fatal("lost response replay changed result")
	}
	otherSession.call("POST", path+"/corrections", M{"submissionId": fnd.NewID(), "expectedSourceVersion": person["sourceVersion"], "education": []M{row(study)}}, "", 409)
	data := get()
	school := find(data, "education", "school", "在读院校")
	dim := find(data, "education", "school", "学历院校")
	award := find(data, "education", "school", "授予院校")
	issued := find(data, "credentials", "credentialName", "资格样本")
	if len(data["education"].([]any)) != 3 || len(data["credentials"].([]any)) != 2 || school["fields"].(M)["endMonth"] != nil || dim["fields"].(M)["highestEducation"] != "yes" || award["fields"].(M)["highestDegree"] != "yes" || issued["fields"].(M)["certificateNumber"] != cert["certificateNumber"] {
		t.Fatal("incorrect adopted facts")
	}
	if find(a.call("GET", path, nil, "", 200), "credentials", "credentialName", "资格样本")["fields"].(M)["certificateNumber"] != "********" {
		t.Fatal("default full number")
	}
	fullPath := path + "?view=credential-" + issued["itemId"].(string)
	if find(a.call("GET", fullPath, nil, "", 200), "credentials", "credentialName", "资格样本")["fields"].(M)["certificateNumber"] != cert["certificateNumber"] || find(reader.call("GET", fullPath, nil, "", 200), "credentials", "credentialName", "资格样本")["fields"].(M)["certificateNumber"] != "********" {
		t.Fatal("full versus restricted certificate read")
	}
	duplicate := maps.Clone(diploma)
	delete(duplicate, "highestEducation")
	send(M{"education": []M{row(duplicate)}}, 409)
	if count() != after {
		t.Fatal("duplicate or stale submission persisted")
	}
	deny := func(field string, item M) string {
		key := fnd.NewID()
		sql("insert into access.person_grant(instance_id,principal_id,grant_id,action,purpose,fields,item_id,denied) values($1,$2,$3,'qualification.correct','education-credentials',array[$4::text],$5,true)", f.instance, a.subject, key, field, item["itemId"])
		return key
	}
	allow := func(key string) { sql("delete from access.person_grant where grant_id=$1", key) }
	// Moving one highest mark needs write access on the previously marked row.
	mark := M{"completion": "已毕业", "educationLevel": "硕士研究生", "highestEducation": "yes"}
	grant := deny("highestEducation", dim)
	send(M{"education": []M{change(school, mark)}, "basic": M{"formerName": "整笔拒绝"}}, 403)
	allow(grant)
	// Changing type clears the old start month even when the command omitted it.
	grant = deny("startMonth", school)
	typeChange := M{"educationType": "学位授予", "degree": "硕士"}
	send(M{"education": []M{change(school, typeChange)}}, 403)
	allow(grant)
	grant = deny("degree", award)
	send(M{"education": []M{change(award, M{"educationType": "已取得学历", "educationLevel": "本科"})}, "credentials": []M{change(issued, M{"specialty": "应回滚"})}}, 403)
	allow(grant)
	grant = deny("validUntil", issued)
	send(M{"credentials": []M{change(issued, M{"indefinite": "yes"})}}, 403)
	allow(grant)
	if count() != after {
		t.Fatal("implicitly changed a protected value")
	}
	send(M{"education": []M{change(school, mark)}}, 200)
	data = get()
	school = find(data, "education", "school", "在读院校")
	if find(data, "education", "school", "学历院校")["fields"].(M)["highestEducation"] != nil || find(data, "education", "school", "授予院校")["revision"] != award["revision"] {
		t.Fatal("highest marks coupled")
	}
	renewal := maps.Clone(cert)
	renewal["issuedOn"] = "2025-01-01"
	delete(renewal, "certificateNumber")
	send(M{"education": []M{change(school, typeChange)}, "credentials": []M{row(renewal)}}, 200)
	data = get()
	converted := find(data, "education", "school", "在读院校")["fields"].(M)
	if len(data["credentials"].([]any)) != 3 || converted["degree"] != "硕士" || converted["startMonth"] != nil || converted["completion"] != nil || converted["highestEducation"] != nil || converted["highestDegree"] != nil {
		t.Fatal("conversion or renewal lost identity")
	}
	foreign := register("教育凭证另一人")
	for _, tc := range []struct {
		group string
		item  M
	}{{"education", award}, {"credentials", issued}} {
		fault := a.call("POST", persons+"/"+foreign["id"].(string)+"/corrections", M{"submissionId": fnd.NewID(), "expectedSourceVersion": foreign["sourceVersion"], tc.group: []M{{"itemId": tc.item["itemId"], "expectedRevision": tc.item["revision"], "remove": true}}}, "", 403)
		if fault["field"] != nil {
			t.Fatal("foreign row exposed")
		}
	}
	sql("update personnel.profile_item_payload set ordinary_until=now()-interval '1 second' where person_id=$1 and item_id=$2", id, issued["itemId"])
	freshAdmin := f.login("admin", f.password)
	visible := freshAdmin.call("GET", fullPath, nil, "", 200)["credentials"].([]any)
	if len(visible) != 2 {
		t.Fatal("expired evidence still visible")
	}
	for _, r := range visible {
		if r.(M)["itemId"] == issued["itemId"] {
			t.Fatal("expired row disclosed")
		}
	}
	if freshAdmin.call("POST", persons+"/submissions/"+cmd["submissionId"].(string)+"/resolve", nil, "", 200)["credentials"] != nil {
		t.Fatal("receipt disclosed expired payload")
	}
	// Expired editions are disposed independently. A hold applies to one exact version and category.
	sql("update personnel.profile_item_payload set ordinary_until=now()-interval '2 days', evidence_until=now()-interval '1 day' where person_id=$1 and section in ('education','credentials')", id)
	sql("insert into personnel.hold values($1,$2,$3,'education',now()+interval '1 day','合成保全')", f.instance, id, saved["sourceVersion"])
	// Take a database copy before the disposition, just as an earlier backup would contain the payload.
	sql("create table public.qualification_backup as select * from personnel.profile_item_payload where person_id=$1", id)
	must(t, bootstrap.Migrate(f.ctx, f.admin, migrations.Files))
	verifyDisposition := func() {
		var kept, other, disposed int
		must(t, f.admin.QueryRow(f.ctx, `select count(*) filter(where version=$2 and section='education'),count(*) filter(where version<>$2 or section<>'education'),(select count(*) from personnel.disposition where person_id=$1 and category in ('education','credentials')) from personnel.profile_item_payload where person_id=$1`, id, saved["sourceVersion"]).Scan(&kept, &other, &disposed))
		if kept != 3 || other != 0 || disposed == 0 {
			t.Fatal("incorrect disposition or hold", kept, other, disposed)
		}
		current := freshAdmin.call("GET", fullPath, nil, "", 200)
		if current["education"] != nil || current["credentials"] != nil {
			t.Fatal("disposed or held education/credentials exposed")
		}
	}
	verifyDisposition()
	sql("insert into personnel.profile_item_payload select * from public.qualification_backup on conflict do nothing")
	must(t, bootstrap.Migrate(f.ctx, f.admin, migrations.Files))
	verifyDisposition()

}

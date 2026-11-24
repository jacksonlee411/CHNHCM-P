//go:build integration

package integration_test

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	fnd "chnhcm.local/chnhcm/internal/foundation"
	"reflect"
	"testing"
)

func TestPersonnelAddressFamily(t *testing.T) {
	type M = map[string]any
	f := personnelFixture(t)
	hr := f.login("admin", f.password)
	reader := f.login("reader", f.readPassword)
	root := hr.call("POST", persons, M{"submissionId": fnd.NewID(), "name": "合成家庭资料"}, "", 200)
	id := root["id"].(string)
	path := persons + "/" + id
	get := func() M { return hr.call("GET", path+"?view=edit", nil, "", 200) }
	post := func(in M, status int) M {
		in["submissionId"] = fnd.NewID()
		in["expectedSourceVersion"] = get()["sourceVersion"]
		return hr.call("POST", path+"/corrections", in, "", status)
	}
	edit := func(row any, fields M) M {
		x := row.(M)
		return M{"itemId": x["itemId"], "expectedRevision": x["revision"], "fields": fields}
	}
	exec := func(sql string, args ...any) { _, e := f.admin.Exec(f.ctx, sql, args...); must(t, e) }
	addr := func(kind, street string) M {
		return M{"fields": M{"addressType": kind, "addressCountry": "历史地名", "street": street, "postalCode": "00123"}}
	}
	first := addr("居住地址", "样本一")
	first["fields"].(M)["addressPrimary"] = "yes"
	one := M{"fields": M{"familyName": "样本甲", "relationship": "配偶", "emergency": "yes"}}
	two := M{"fields": M{"familyName": "样本乙", "relationship": "子女"}}
	input := M{"addresses": []M{first, addr("通讯地址", "样本二")}, "family": []M{one, two}, "contacts": []M{{"kind": "phone", "value": "22222"}}, "basic": M{"formerName": "样本原名"}}
	if fault := post(input, 400); fault["field"] != "family.0.emergency" || get()["sourceVersion"] != root["sourceVersion"] {
		t.Fatal("mixed draft partially saved or incorrect error location", fault)
	}
	one["fields"].(M)["familyPhone"] = "12345"
	saved := post(input, 200)
	data := get()
	if !reflect.DeepEqual(saved, hr.call("POST", path+"/corrections", input, "", 200)) {
		t.Fatal("receipt changed")
	}
	rows := data["addresses"].([]any)
	family := data["family"].([]any)
	if len(rows) != 2 || len(family) != 2 || rows[0].(M)["fields"].(M)["postalCode"] != "00123" || data["basic"].(M)["formerName"] != "样本原名" {
		t.Fatal("mixed save")
	}
	if reader.call("GET", path, nil, "", 200)["family"].([]any)[0].(M)["fields"].(M)["familyPhone"] != "****2345" {
		t.Fatal("reader family mask")
	}
	for _, target := range []any{rows[0], rows[1]} {
		grant := fnd.NewID()
		exec("insert into access.person_grant(instance_id,principal_id,grant_id,action,purpose,fields,item_id,denied) values($1,$2,$3,'profile.correct','personal-profile','{addressPrimary}',$4,true)", f.instance, hr.subject, grant, target.(M)["itemId"])
		post(M{"addresses": []M{edit(rows[1], M{"addressPrimary": "yes"})}, "family": []M{edit(family[1], M{"familyJob": "不应写入"})}}, 403)
		if get()["sourceVersion"] != data["sourceVersion"] {
			t.Fatal("unauthorized primary switch")
		}
		exec("delete from access.person_grant where grant_id=$1", grant)
	}
	change := M{"addresses": []M{edit(rows[1], M{"addressPrimary": "yes"})}, "family": []M{edit(family[0], M{"familyPhone": ""})}}
	post(change, 400)
	change["family"] = []M{edit(family[0], M{"familyPhone": "", "familyEmail": "kin@example.test"}), edit(family[1], M{"familyPhone": "67890", "emergency": "yes"})}
	post(change, 200)
	data = get()
	rows = data["addresses"].([]any)
	family = data["family"].([]any)
	if rows[0].(M)["fields"].(M)["addressPrimary"] != nil || rows[1].(M)["fields"].(M)["addressPrimary"] != "yes" || data["contacts"].([]any)[0].(M)["value"] != "22222" || family[0].(M)["fields"].(M)["emergency"] != "yes" || family[1].(M)["fields"].(M)["emergency"] != "yes" {
		t.Fatal("address or emergency switches")
	}
	post(M{"family": []M{edit(family[0], M{"familyEmail": "", "emergency": ""})}}, 200)
	foreign := hr.call("POST", persons, M{"name": "合成另一人", "submissionId": fnd.NewID()}, "", 200)
	for _, group := range []string{"addresses", "family"} {
		row := get()[group].([]any)[0].(M)
		hr.call("POST", persons+"/"+foreign["id"].(string)+"/corrections", M{"submissionId": fnd.NewID(), "expectedSourceVersion": foreign["sourceVersion"], group: []M{{"itemId": row["itemId"], "expectedRevision": row["revision"], "remove": true}}}, "", 403)
	}
	data = get()
	exec("insert into access.person_grant(instance_id,principal_id,action,purpose,fields,denied) values($1,$2,'family.correct','family-contact','{familyJob}',true)", f.instance, hr.subject)
	post(M{"family": []M{edit(data["family"].([]any)[0], M{"familyJob": "样本职务"})}, "addresses": []M{addr("其他地址", "应回滚")}, "basic": M{"formerName": "应回滚"}}, 403)
	if get()["sourceVersion"] != data["sourceVersion"] {
		t.Fatal("revoked draft partially saved")
	}
	reader.call("POST", path+"/corrections", M{"submissionId": fnd.NewID(), "expectedSourceVersion": data["sourceVersion"], "addresses": []M{first}}, "", 403)
	exec("insert into access.person_grant(instance_id,principal_id,action,purpose,fields,denied,item_id) values($1,$2,'family.read','family-contact','{familyName}',true,$3)", f.instance, reader.subject, family[1].(M)["itemId"])
	if len(reader.call("GET", path, nil, "", 200)["family"].([]any)) != 1 {
		t.Fatal("restricted count")
	}
	for _, group := range []string{"addresses", "family"} {
		row := data[group].([]any)[1].(M)
		post(M{group: []M{{"itemId": row["itemId"], "expectedRevision": row["revision"], "remove": true}}}, 200)
		if len(get()[group].([]any)) != 1 {
			t.Fatal("remove failed", group)
		}
	}
	history := hr.call("GET", path+"/sources/"+saved["sourceVersion"].(string), nil, "", 200)
	if history["family"] != nil || history["addresses"] != nil {
		t.Fatal("private extension in historical response")
	}
}

func TestPersonnelFamilyRejectedReceipt(t *testing.T) {
	type M = map[string]any
	f := personnelFixture(t)
	hr := f.login("admin", f.password)
	person := hr.call("POST", persons, M{"name": "家庭办理回执样本", "submissionId": fnd.NewID()}, "", 200)
	id := person["id"].(string)
	path := persons + "/" + id
	hr.call("POST", path+"/corrections", M{"submissionId": fnd.NewID(), "expectedSourceVersion": person["sourceVersion"], "family": []M{{"fields": M{"familyName": "合成亲属", "relationship": "父母", "emergency": "yes", "familyPhone": "12345"}}}}, "", 200)
	detail := hr.call("GET", path, nil, "", 200)
	row := detail["family"].([]any)[0].(M)
	submission := fnd.NewID()
	edit := M{"submissionId": submission, "expectedSourceVersion": detail["sourceVersion"], "family": []M{{"itemId": row["itemId"], "expectedRevision": row["revision"], "fields": M{"familyPhone": ""}}}}
	fault := hr.call("POST", path+"/corrections", edit, "", 400)
	if fault["field"] != "family.0.emergency" || !reflect.DeepEqual(fault, hr.call("POST", path+"/corrections", edit, "", 400)) {
		t.Fatal("original field error")
	}
	_, err := f.admin.Exec(f.ctx, "insert into access.person_grant(instance_id,principal_id,action,purpose,fields,item_id,denied) values($1,$2,'family.read','family-contact','{emergency}',$3,true)", f.instance, hr.subject, row["itemId"])
	must(t, err)
	limited := hr.call("POST", path+"/corrections", edit, "", 403)
	if limited["field"] != nil || limited["message"] == fault["message"] {
		t.Fatal("replayed field detail after scope change")
	}
	out := hr.call("POST", persons+"/submissions/"+submission+"/resolve", nil, "", 200)
	if out["outcome"] != "rejected" || hr.call("GET", path, nil, "", 200)["sourceVersion"] != detail["sourceVersion"] {
		t.Fatal("rejected terminal or source changed")
	}
}

func TestPersonnelExpiredPrimaryAddress(t *testing.T) {
	f := personnelFixture(t)
	hr := f.login("admin", f.password)
	type M = map[string]any
	person := hr.call("POST", persons, M{"name": "地址到期样本", "submissionId": fnd.NewID()}, "", 200)
	id := person["id"].(string)
	path := persons + "/" + id
	one := M{"addressType": "居住地址", "addressCountry": "原文地名", "street": "合成旧址", "addressPrimary": "yes"}
	two := M{"addressType": "通讯地址", "addressCountry": "原文地名", "street": "合成新址"}
	hr.call("POST", path+"/corrections", M{"expectedSourceVersion": person["sourceVersion"], "submissionId": fnd.NewID(), "addresses": []M{{"fields": one}, {"fields": two}}}, "", 200)
	current := hr.call("GET", path, nil, "", 200)
	rows := current["addresses"].([]any)
	old, next := rows[0].(M), rows[1].(M)
	_, err := f.admin.Exec(f.ctx, "update personnel.profile_item_payload set ordinary_until=now()-interval '1 second' where instance_id=$1 and person_id=$2 and item_id=$3", f.instance, id, old["itemId"])
	must(t, err)
	saved := hr.call("POST", path+"/corrections", M{"expectedSourceVersion": current["sourceVersion"], "submissionId": fnd.NewID(), "addresses": []M{{"itemId": next["itemId"], "expectedRevision": next["revision"], "fields": M{"addressPrimary": "yes"}}}}, "", 200)
	visible := hr.call("GET", path, nil, "", 200)["addresses"].([]any)
	if len(visible) != 1 || visible[0].(M)["itemId"] != next["itemId"] || visible[0].(M)["fields"].(M)["addressPrimary"] != "yes" {
		t.Fatal("current primary address")
	}
	var retained, copied int
	must(t, f.admin.QueryRow(f.ctx, `select count(*) filter (where version=$4 and revision=$5 and fields->>'addressPrimary'='yes'), count(*) filter (where version=$6) from personnel.profile_item_payload where instance_id=$1 and person_id=$2 and item_id=$3`, f.instance, id, old["itemId"], current["sourceVersion"], old["revision"], saved["sourceVersion"]).Scan(&retained, &copied))
	if retained != 1 || copied != 0 {
		t.Fatalf("sealed primary %d, copied %d", retained, copied)
	}
}

func TestPersonnelAddressDirectory(t *testing.T) {
	type M = map[string]any
	f := personnelFixture(t)
	login := access.LoginInput{Login: "admin", Password: f.password}
	must(t, f.runtime.InitializeCountryCatalog(f.ctx, login))
	hr := f.login("admin", f.password)
	root := hr.call("POST", persons, M{"submissionId": fnd.NewID(), "name": "目录样本", "contacts": []M{{"kind": "phone", "value": "12345"}}}, "", 200)
	path := persons + "/" + root["id"].(string)
	get := func() M { return hr.call("GET", path+"?view=edit", nil, "", 200) }
	post := func(in M, status int) M {
		in["submissionId"] = fnd.NewID()
		in["expectedSourceVersion"] = get()["sourceVersion"]
		return hr.call("POST", path+"/corrections", in, "", status)
	}
	var country string
	must(t, f.admin.QueryRow(f.ctx, "select country_region_id from organization.country_region where instance_id=$1 and code='CN'", f.instance).Scan(&country))
	date := hr.call("GET", "/api/v1/geo-areas/context", nil, "", 200)["currentDate"].(string)
	unavailable := hr.call("GET", "/api/v1/geo-areas/candidates?countryRegionId="+country+"&adoptedDate="+date, nil, "", 409)
	if unavailable["code"] != "M02_CATALOG_NOT_READY" {
		t.Fatal("catalog unavailable was not identified")
	}
	must(t, f.runtime.InitializeGeoAreaCatalog(f.ctx, login))
	candidate := func(query string) M {
		return hr.call("GET", "/api/v1/geo-areas/candidates?countryRegionId="+country+"&adoptedDate="+date+query, nil, "", 200)["items"].([]any)[0].(M)
	}
	choice := func(item M) M {
		nodes := item["path"].([]any)
		ancestors := []M{}
		for _, x := range nodes[:len(nodes)-1] {
			n := x.(M)
			ancestors = append(ancestors, M{"id": n["id"], "sourceVersion": n["sourceVersion"]})
		}
		return M{"id": item["id"], "sourceVersion": nodes[len(nodes)-1].(M)["sourceVersion"], "ancestors": ancestors}
	}
	province := candidate("&q=440000")
	area := candidate("&parentAreaId=" + province["id"].(string) + "&q=440305")
	choices := M{"addressCountry": M{"id": country, "sourceVersion": province["path"].([]any)[0].(M)["sourceVersion"]}, "province": choice(province), "area": choice(area)}
	in := M{"basic": M{"englishName": "Catalog"}, "family": []M{{"fields": M{"familyName": "样本", "relationship": "其他"}}}, "addresses": []M{{"fields": M{"addressType": "通讯地址", "street": "合成地址"}, "choices": choices}}}
	var original []string
	must(t, f.admin.QueryRow(f.ctx, "select read_fields from access.subject_grant where principal_id=$1 and action='geoarea.read'", hr.subject).Scan(&original))
	_, err := f.admin.Exec(f.ctx, "update access.subject_grant set read_fields='{}' where principal_id=$1 and action='geoarea.read'", hr.subject)
	must(t, err)
	fault := post(in, 403)
	if fault["code"] != "PERSON_PLACE_DENIED" || get()["sourceVersion"] != root["sourceVersion"] {
		t.Fatal("withdrawn catalog grant", fault)
	}
	_, err = f.admin.Exec(f.ctx, "update access.subject_grant set read_fields=$2 where principal_id=$1 and action='geoarea.read'", hr.subject, original)
	must(t, err)
	geo := "/api/v1/geo-areas/" + area["id"].(string)
	detail := hr.call("GET", geo, nil, "", 200)
	rec := detail["selectedRecord"].(M)
	changed := hr.call("POST", geo+"/records/"+rec["recordId"].(string)+"/correct", M{"expectedSourceVersion": detail["headSourceVersion"], "expectedRecordRevision": rec["revision"], "changes": M{"name": "已修订地区"}}, fnd.NewID(), 200)
	if fault := post(in, 409); fault["field"] != "addresses.0.area" || get()["sourceVersion"] != root["sourceVersion"] {
		t.Fatal("stale selection was not atomic", fault)
	}
	choices["area"] = choice(candidate("&parentAreaId=" + province["id"].(string) + "&q=440305"))
	choices["province"] = choice(candidate("&q=110000"))
	post(in, 409)
	choices["province"] = choice(province)
	post(in, 200)
	data := get()
	address := data["addresses"].([]any)[0]
	a := address.(M)["adoptions"].(M)["area"].(M)
	if a["sourceVersion"] != changed["sourceVersion"] || a["date"] != date || a["recordId"] != rec["recordId"] || address.(M)["fields"].(M)["area"] != "已修订地区" {
		t.Fatal("incomplete adoption")
	}
	// The candidate includes the current ancestry. An unchanged saved province keeps its original adoption.
	provincePath := "/api/v1/geo-areas/" + province["id"].(string)
	pd := hr.call("GET", provincePath, nil, "", 200)
	pr := pd["selectedRecord"].(M)
	hr.call("POST", provincePath+"/records/"+pr["recordId"].(string)+"/correct", M{"expectedSourceVersion": pd["headSourceVersion"], "expectedRecordRevision": pr["revision"], "changes": M{"nameEn": "Revised province"}}, fnd.NewID(), 200)
	edit := M{"itemId": address.(M)["itemId"], "expectedRevision": address.(M)["revision"], "choices": M{"area": choices["area"]}}
	reselect := M{"addresses": []M{edit}}
	post(reselect, 409)
	if get()["sourceVersion"] != data["sourceVersion"] {
		t.Fatal("stale ancestor partially saved")
	}
	edit["choices"] = M{"area": choice(candidate("&parentAreaId=" + province["id"].(string) + "&q=440305"))}
	post(reselect, 200)
	updated := get()["addresses"].([]any)[0].(M)
	if updated["revision"] == address.(M)["revision"] || !reflect.DeepEqual(updated["adoptions"].(M)["province"], address.(M)["adoptions"].(M)["province"]) {
		t.Fatal("reselect must update the area ancestry and preserve the stored province")
	}
	edit["expectedRevision"] = updated["revision"]
	if r := post(reselect, 200); r["outcome"] != "no_change" {
		t.Fatal("identical adoption created another revision")
	}
	address = updated
	hr.call("POST", geo+"/deactivate", M{"expectedSourceVersion": changed["sourceVersion"], "anchorRecordId": rec["recordId"], "effectiveDate": date, "operationType": "DEACTIVATE", "operationReason": "STOP_SELECTION"}, fnd.NewID(), 201)
	post(in, 409)
	phone := data["contacts"].([]any)[0].(M)
	post(M{"contacts": []M{{"itemId": phone["itemId"], "expectedRevision": phone["revision"], "kind": "phone", "value": "67890"}}}, 200)
	if !reflect.DeepEqual(address, get()["addresses"].([]any)[0]) || get()["contacts"].([]any)[0].(M)["value"] != "67890" {
		t.Fatal("unrelated edit lost original adoption")
	}
}

//go:build integration

package integration_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"chnhcm.local/chnhcm/db/migrations"
	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/foundation"
)

type logBody = map[string]any

type personnelLogPage struct {
	Page, PageSize, Total int
	ThroughSourceVersion  string
	ReadFields            []string
	Items                 []struct {
		SourceVersion, PreviousSourceVersion, Operator, Action, RecordedAt string
		Diffs                                                              []struct {
			Field, FieldKey, ItemID, Before, After string
			CanReveal                              bool
		}
	}
}

func readPersonnelLog(t *testing.T, b *browser, path string, query logBody) personnelLogPage {
	t.Helper()
	raw, err := json.Marshal(b.call("POST", path+"/changes", query, "", 200))
	must(t, err)
	var page personnelLogPage
	must(t, json.Unmarshal(raw, &page))
	return page
}

func personnelLogFilter(field, operator, value string) logBody {
	return logBody{"filterField": field, "filterOperator": operator, "filterValue": value}
}

func TestPersonnelLogPaginationAndFilters(t *testing.T) {
	f := personnelFixture(t)
	hr := f.login("admin", f.password)
	created := hr.call("POST", persons, logBody{"submissionId": foundation.NewID(), "name": "初始姓名"}, "", 200)
	id := created["id"].(string)
	path := persons + "/" + id
	versions := []string{created["sourceVersion"].(string)}
	appendEdit := func() {
		result := hr.call("POST", path+"/corrections", logBody{
			"submissionId": foundation.NewID(), "expectedSourceVersion": versions[len(versions)-1],
			"name": fmt.Sprintf("第%02d次更正", len(versions)),
		}, "", 200)
		versions = append(versions, result["sourceVersion"].(string))
	}
	for i := 0; i < 21; i++ {
		appendEdit()
	}
	first := readPersonnelLog(t, hr, path, logBody{})
	if first.Page != 1 || first.PageSize != 20 || first.Total != 22 || len(first.Items) != 20 || first.ThroughSourceVersion != versions[21] {
		t.Fatalf("unexpected first page: page %d / size %d / total %d / groups %d", first.Page, first.PageSize, first.Total, len(first.Items))
	}
	for i, group := range first.Items {
		if group.SourceVersion != versions[21-i] || group.PreviousSourceVersion != versions[20-i] || group.Action != "correct" || group.Operator != "admin" {
			t.Fatalf("group %d broke recorded source chain", i)
		}
	}

	// A subsequent writer commits before the reader asks for page two.
	appendEdit()
	second := readPersonnelLog(t, hr, path, logBody{"throughSourceVersion": first.ThroughSourceVersion, "page": 2})
	if second.Total != 22 || second.Page != 2 || len(second.Items) != 2 || second.Items[0].SourceVersion != versions[1] || second.Items[1].SourceVersion != versions[0] || second.Items[1].PreviousSourceVersion != "" {
		t.Fatal("intervening write displaced an operation on the pinned second page")
	}
	for _, size := range []int{50, 100} {
		page := readPersonnelLog(t, hr, path, logBody{"pageSize": size})
		if page.PageSize != size || page.Total != 23 || len(page.Items) != 23 || page.Items[0].SourceVersion != versions[22] {
			t.Fatalf("fresh page at size %d did not include the latest correction", size)
		}
	}
	last := readPersonnelLog(t, hr, path, logBody{"page": 42})
	if last.Page != 2 || len(last.Items) != 3 || last.Items[2].SourceVersion != versions[0] {
		t.Fatal("out of range did not return the last page")
	}

	// Assign all corrections the same instant, before the registration's recorded
	// time. Provenance order still follows the commits, while date predicates use Beijing.
	_, err := f.admin.Exec(f.ctx, `update personnel.source set recorded_at=case when action='register'
	then timestamptz '2026-01-02 16:00:00.123456+00' else timestamptz '2026-01-02 15:59:59.999999+00' end where person_id=$1`, id)
	must(t, err)
	pinned := readPersonnelLog(t, hr, path, logBody{"pageSize": 50, "throughSourceVersion": versions[21]})
	if len(pinned.Items) != 22 {
		t.Fatal("missing operations with equal or reversed timestamps")
	}
	for i, row := range pinned.Items {
		if row.SourceVersion != versions[21-i] {
			t.Fatalf("timestamp displaced source at offset %d", i)
		}
	}
	for _, test := range []struct {
		field, op, value string
		count            int
	}{
		{"recordedAt", "is", "2026-01-03", 1}, {"recordedAt", "onOrAfter", "2026-01-03", 1}, {"recordedAt", "after", "2026-01-03", 0},
		{"recordedAt", "before", "2026-01-03", 22}, {"recordedAt", "onOrBefore", "2026-01-03", 23},
		{"actor", "contains", "dmi", 23}, {"actor", "equals", "dmi", 0}, {"action", "isAnyOf", "register,renumber", 1},
		{"changedField", "is", "number", 1}, {"changedField", "isAnyOf", "name,number", 23},
	} {
		page := readPersonnelLog(t, hr, path, personnelLogFilter(test.field, test.op, test.value))
		if page.Total != test.count || len(page.Items) != min(20, test.count) {
			t.Errorf("%s %s %s got %d operations, want %d", test.field, test.op, test.value, page.Total, test.count)
		}
	}
	for _, query := range []logBody{
		{"page": nil}, {"pageSize": nil}, {"page": 0}, {"page": -1}, {"pageSize": 21},
		{"throughSourceVersion": nil}, {"throughSourceVersion": ""},
		{"filterField": "actor"}, personnelLogFilter("actor", "before", "admin"),
		personnelLogFilter("recordedAt", "is", "2026-02-30"), personnelLogFilter("unknown", "equals", "value"),
	} {
		hr.call("POST", path+"/changes", query, "", 400)
	}
	other := hr.call("POST", persons, logBody{"name": "另一档案", "submissionId": foundation.NewID()}, "", 200)
	foreign := hr.call("POST", path+"/changes", logBody{"throughSourceVersion": other["sourceVersion"]}, "", 404)
	missing := hr.call("POST", path+"/changes", logBody{"throughSourceVersion": foundation.NewID()}, "", 404)
	if foreign["code"] != missing["code"] || foreign["message"] != missing["message"] {
		t.Fatal("foreign bound disclosed existence")
	}
}

func TestPersonnelLogAuthorizedSourceDifferences(t *testing.T) {
	f := personnelFixture(t)
	hr, reader := f.login("admin", f.password), f.login("reader", f.readPassword)
	reg := hr.call("POST", persons, logBody{"name": "旧姓名", "submissionId": foundation.NewID(), "primaryDocument": logBody{"kind": "passport", "number": "LOG-PRIVATE-01"}, "contacts": []logBody{{"kind": "phone", "value": "13810001111"}, {"kind": "phone", "value": "13820002222"}}}, "", 200)
	id, version := reg["id"].(string), reg["sourceVersion"].(string)
	path := persons + "/" + id
	part := hr.call("GET", path, nil, "", 200)["contacts"].([]any)
	update := func(index int, value string) logBody {
		contact := part[index].(logBody)
		return logBody{"kind": "phone", "itemId": contact["itemId"], "expectedRevision": contact["revision"], "value": value}
	}
	edit := hr.call("POST", path+"/corrections", logBody{"submissionId": foundation.NewID(), "expectedSourceVersion": version,
		"name": "新姓名", "contacts": []logBody{update(0, "13999991111"), update(1, "13999993333")}}, "", 200)
	correction := edit["sourceVersion"].(string)
	page := readPersonnelLog(t, hr, path, logBody{})
	if page.Total != 2 || len(page.Items) != 2 || len(page.Items[0].Diffs) != 3 {
		t.Fatal("one multi-item correction must form a single group with three changes")
	}
	if page.Items[0].PreviousSourceVersion != version || page.Items[0].SourceVersion != correction {
		t.Fatal("wrong correction or parent")
	}
	wantPhones := map[string][2]string{
		part[0].(logBody)["itemId"].(string): {"****1111", "****1111"},
		part[1].(logBody)["itemId"].(string): {"****2222", "****3333"},
	}
	for _, diff := range page.Items[0].Diffs {
		switch diff.FieldKey {
		case "name":
			if diff.Before != "旧姓名" || diff.After != "新姓名" {
				t.Fatal("name difference lost old revision")
			}
		case "phone":
			want, ok := wantPhones[diff.ItemID]
			if !ok || [2]string{diff.Before, diff.After} != want || !diff.CanReveal {
				t.Fatal("default phone summary or exact disclosure capability was incorrect")
			}
			delete(wantPhones, diff.ItemID)
		default:
			t.Fatalf("unexpected change field %s", diff.FieldKey)
		}
	}
	if len(wantPhones) != 0 {
		t.Fatal("same-kind contacts were merged")
	}
	readExact := func(b *browser) []logBody {
		r, err := b.client.Get(f.server.URL + path + "/changes/" + correction)
		must(t, err)
		defer r.Body.Close()
		if r.StatusCode != 200 {
			t.Fatalf("exact difference status %d", r.StatusCode)
		}
		var rows []logBody
		must(t, json.NewDecoder(r.Body).Decode(&rows))
		return rows
	}
	textOf := func(v any) string { raw, e := json.Marshal(v); must(t, e); return string(raw) }
	logText := textOf(hr.call("POST", path+"/changes", logBody{}, "", 200))
	for _, secret := range []string{"13810001111", "13999991111", "13820002222", "13999993333", "LOG-PRIVATE-01"} {
		if strings.Contains(logText, secret) {
			t.Fatal("list included a private plaintext value")
		}
	}
	exactText := textOf(readExact(hr))
	if !strings.Contains(exactText, "13810001111") || !strings.Contains(exactText, "13999991111") {
		t.Fatal("explicit authorized difference lost the equal-mask phone edit")
	}
	original := hr.call("GET", path+"/sources/"+version, nil, "", 200)
	if original["name"] != "旧姓名" || original["contacts"].([]any)[0].(logBody)["value"] != "13810001111" {
		t.Fatal("exact source linked to current data")
	}
	exec := func(sql string, args ...any) { _, e := f.admin.Exec(f.ctx, sql, args...); must(t, e) }
	// A later correction changes just the private prefix of the second phone.
	// Actors with different visibility get different filtered counts, but all
	// authorized actions still have an unfiltered group.
	latestContacts := hr.call("GET", path, nil, "", 200)["contacts"].([]any)
	secondPhone := latestContacts[1].(logBody)
	hiddenEdit := hr.call("POST", path+"/corrections", logBody{
		"submissionId": foundation.NewID(), "expectedSourceVersion": correction,
		"contacts": []logBody{{"kind": "phone", "itemId": secondPhone["itemId"], "expectedRevision": secondPhone["revision"], "value": "13777773333"}},
	}, "", 200)
	if full := readPersonnelLog(t, hr, path, personnelLogFilter("changedField", "is", "phone")); full.Total != 3 {
		t.Fatal("full history reader lost a real same-mask change")
	}
	exec("update access.person_grant set fields=array['name','number','phone.masked'] where principal_id=$1 and action in ('read','history.read')", reader.subject)
	masked := readPersonnelLog(t, reader, path, personnelLogFilter("changedField", "is", "phone"))
	if masked.Total != 2 || len(masked.Items) != 2 || len(masked.Items[0].Diffs) != 2 {
		t.Fatal("masked actor should see name and only the visibly changed phone")
	}
	allMasked := readPersonnelLog(t, reader, path, logBody{})
	if allMasked.Total != 3 || len(allMasked.Items) != 3 || allMasked.Items[0].SourceVersion != hiddenEdit["sourceVersion"] || len(allMasked.Items[0].Diffs) != 0 {
		t.Fatal("an authorized operation with no visible difference lost its metadata or disclosed a hidden value")
	}
	for _, diff := range masked.Items[0].Diffs {
		if diff.FieldKey == "phone" && (diff.ItemID != part[1].(logBody)["itemId"] || diff.Before != "****2222" || diff.After != "****3333" || diff.CanReveal) {
			t.Fatal("masked actor inferred hidden contact value")
		}
	}
	exec("update access.person_grant set fields=array['name','number'] where principal_id=$1 and action='history.read'", reader.subject)
	denied := reader.call("POST", path+"/changes", personnelLogFilter("changedField", "is", "phone"), "", 400)
	unknown := reader.call("POST", path+"/changes", personnelLogFilter("changedField", "is", "not_a_field"), "", 400)
	if denied["field"] != "filterValue" || !reflect.DeepEqual(denied, unknown) {
		t.Fatal("unknown and revoked field exposed different results")
	}
	short := readPersonnelLog(t, reader, path, logBody{"throughSourceVersion": correction})
	if short.Total != 2 || len(short.Items) != 2 || slices.Contains(short.ReadFields, "phone") || len(short.Items[0].Diffs) != 1 || short.Items[0].Diffs[0].FieldKey != "name" {
		t.Fatal("mask, hidden field or hidden count survived revocation")
	}
	if got := textOf(readExact(reader)); strings.Contains(got, "1111") || strings.Contains(got, "2222") || strings.Contains(got, "电话") {
		t.Fatal("revoked field leaked in an exact difference")
	}

	// Only the older snapshot of one item has reached its retention deadline.
	// The later value must never appear as a new registration of that contact.
	exec("update personnel.contact_payload set ordinary_until=clock_timestamp()-interval '2 days', evidence_until=clock_timestamp()-interval '1 day' where person_id=$1 and version=$2 and item_id=$3", id, version, part[0].(logBody)["itemId"])
	checkExpiredSide := func(stage string) {
		t.Helper()
		restricted := readPersonnelLog(t, hr, path, logBody{})
		for _, group := range restricted.Items {
			if group.SourceVersion != version && group.SourceVersion != correction {
				continue
			}
			for _, diff := range group.Diffs {
				if diff.ItemID == part[0].(logBody)["itemId"] {
					t.Fatalf("%s item was presented as an addition or removal", stage)
				}
			}
		}
		if got := textOf(readExact(hr)); strings.Contains(got, "13810001111") || strings.Contains(got, "13999991111") {
			t.Fatalf("%s item appeared in exact differences", stage)
		}
	}
	checkExpiredSide("expired")
	must(t, bootstrap.Migrate(f.ctx, f.admin, migrations.Files))
	checkExpiredSide("disposed")
	exec("delete from access.person_grant where principal_id=$1 and action='history.read'", reader.subject)
	reader.call("POST", path+"/changes", logBody{}, "", 404)
	reader.call("GET", path+"/sources/"+correction, nil, "", 404)
	reader.call("GET", path, nil, "", 200)
}

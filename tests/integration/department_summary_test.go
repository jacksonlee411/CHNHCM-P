//go:build integration

package integration_test

import (
	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/foundation"
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDepartmentV08V11OperationHistoryAttribution(t *testing.T) {
	s := newDepartmentContract(t)
	allDepartmentRights(s)
	s.f.exec(`UPDATE access.principal SET display_name=CASE login WHEN 'admin' THEN '甲操作员' ELSE '乙操作员' END`)
	first := s.create("C", "资料一", "BU_A", "", "2026-01-01", s.b, 201)
	second := s.append("C", "C_TWO", "2026-06-01", "CHANGE", map[string]any{"name": "资料二"}, s.actor, 201)
	_, r1 := s.target("C", "C_FIRST")
	third := s.append("C", "C_THREE", "2026-10-01", "CHANGE", map[string]any{"name": "资料三"}, s.b, 201)
	fourth := s.edit("C", "C_TWO", "correct", map[string]any{"changes": map[string]any{"name": "资料二新"}}, s.actor, 200)
	_, r2 := s.target("C", "C_TWO")
	_, r3 := s.target("C", "C_THREE")
	path := "/api/v1/departments/" + s.aliases["C"]
	read := func(source, record string, b *browser) map[string]any {
		q := url.Values{"selectedRecordId": {record}}
		if source != "" {
			q.Set("sourceVersion", source)
		}
		return b.call("GET", path+"?"+q.Encode(), nil, "", 200)
	}
	assertSummary := func(d map[string]any, revision, source, actor string) {
		t.Helper()
		summaryIs(t, d, d["selectedRecord"].(map[string]any)["recordId"].(string), revision, source, actor)
	}
	assertSummary(read("", s.records["C_FIRST"], s.b), r1["revision"].(string), second["sourceVersion"].(string), s.actor.subject)
	assertSummary(read("", s.records["C_TWO"], s.b), r2["revision"].(string), fourth["sourceVersion"].(string), s.actor.subject)
	assertSummary(read("", s.records["C_THREE"], s.b), r3["revision"].(string), third["sourceVersion"].(string), s.b.subject)
	s.f.exec(`UPDATE access.principal SET display_name='后续改名',login='renamed' WHERE principal_id=$1`, s.b.subject)
	if summaryOf(t, read(first["sourceVersion"].(string), s.records["C_FIRST"], s.actor))["operator"].(map[string]any)["displayName"] != "甲操作员" {
		t.Fatal("identity snapshot rewritten")
	}
	fifth := s.edit("C", "C_TWO", "delete", nil, s.b, 200)
	tomb := read(fifth["sourceVersion"].(string), s.records["C_TWO"], s.actor)
	if tomb["selectedRecord"].(map[string]any)["deleted"] != true {
		t.Fatal("missing tombstone")
	}
	assertSummary(tomb, r2["revision"].(string), fourth["sourceVersion"].(string), s.actor.subject)
	log := s.actor.call("GET", path+"/changes", nil, "", 200)
	if log["total"] != float64(5) {
		t.Fatal("missing history", log)
	}
	deleted := log["items"].([]any)[0].(map[string]any)
	item := changeItem(t, deleted, s.records["C_TWO"])
	if deleted["operator"].(map[string]any)["subjectId"] != s.b.subject || item["afterRef"] != nil || item["beforeRef"].(map[string]any)["revision"] != r2["revision"] || item["beforeRef"].(map[string]any)["sourceVersion"] != fourth["sourceVersion"] {
		t.Fatal("deletion link attribution", item)
	}
	for _, field := range []string{"name", "effectiveDate", "operationReason"} {
		filtered := s.actor.call("GET", path+"/changes?"+logFilter("changedField", "is", field), nil, "", 200)
		found := false
		for _, row := range filtered["items"].([]any) {
			if row.(map[string]any)["sourceVersion"] == fourth["sourceVersion"] {
				found = true
			}
		}
		if found != (field == "name") {
			t.Fatal("unchanged context counted as difference", field, filtered)
		}
	}
	s.actor.call("GET", path+"/changes?contextSourceVersion="+third["sourceVersion"].(string)+"&throughSourceVersion="+fourth["sourceVersion"].(string), nil, "", 400)
	if s.actor.call("GET", path+"/changes?throughSourceVersion="+fourth["sourceVersion"].(string), nil, "", 200)["total"] != float64(4) {
		t.Fatal("history window moved")
	}
	// Service operator with no display name remains an exact complete snapshot.
	s.f.exec(`UPDATE access.principal SET display_name=NULL,principal_kind='service' WHERE principal_id=$1`, s.actor.subject)
	sixth := s.edit("C", "C_THREE", "correct", map[string]any{"changes": map[string]any{"shortName": "服务更新"}}, s.actor, 200)
	full := s.actor.call("GET", path, nil, "", 200)
	op := summaryOf(t, full)["operator"].(map[string]any)
	if op["kind"] != "service" || op["displayName"] != nil || op["login"] != "reader" || op["identityEvidence"] != "captured" {
		t.Fatal("service identity incomplete", op)
	}
	// A reused revision's summary remains independent of the head's new actor.
	assertSummary(read("", s.records["C_FIRST"], s.actor), s.b.call("GET", path+"?selectedRecordId="+s.records["C_FIRST"], nil, "", 200)["selectedRecord"].(map[string]any)["revision"].(string), fifth["sourceVersion"].(string), s.b.subject)
	mutateLifeGrants(s.f, s.actor.subject, `UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'name') WHERE principal_id=$1 AND action='department.history.read'`, s.actor.subject)
	history := s.actor.call("GET", path+"/changes", nil, "", 200)
	if !containsField(history["detailReadFields"].([]any), "name") {
		t.Fatal("current detail fields incorrectly intersected with history")
	}
	for _, row := range history["items"].([]any) {
		for _, entry := range row.(map[string]any)["items"].([]any) {
			it := entry.(map[string]any)
			for _, key := range []string{"beforeContext", "afterContext"} {
				if v, ok := it[key].(map[string]any); ok && v["name"] != nil {
					t.Fatal("history field leaked")
				}
			}
		}
	}
	s.actor.call("GET", path+"/changes?"+logFilter("changedField", "is", "name"), nil, "", 403)
	priorSummary := summaryOf(t, full)
	mutateLifeGrants(s.f, s.actor.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='department.history.read'`, s.actor.subject)
	basic := s.actor.call("GET", path, nil, "", 200)
	if !reflect.DeepEqual(priorSummary, summaryOf(t, basic)) {
		t.Fatal("summary requires log rights")
	}
	s.actor.call("GET", path+"/changes", nil, "", 403)
	s.actor.call("GET", path+"?sourceVersion="+sixth["sourceVersion"].(string), nil, "", 403)
	s.f.exec(`UPDATE access.principal SET state='disabled' WHERE principal_id=$1`, s.actor.subject)
	if !reflect.DeepEqual(summaryOf(t, s.b.call("GET", path, nil, "", 200)), priorSummary) {
		t.Fatal("deactivated actor rewrote captured summary")
	}
}
func containsField(fields []any, want string) bool {
	for _, field := range fields {
		if field == want {
			return true
		}
	}
	return false
}

// H45 preserves the 023 count/actor/date oracle; only nameEn maps to shortName.
func TestDepartmentH45PaginationAndPrivacy(t *testing.T) {
	s := newDepartmentContract(t)
	allDepartmentRights(s)
	s.f.exec(`UPDATE access.principal SET display_name=CASE login WHEN 'admin' THEN '甲操作员' ELSE '乙操作员' END`)
	first := s.create("H45", "样本1", "BU_A", "", "2026-01-01", s.b, 201)
	sources := []string{first["sourceVersion"].(string)}
	for n := 2; n <= 45; n++ {
		who := s.b
		if n%2 == 0 {
			who = s.actor
		}
		changes := map[string]any{"name": fmt.Sprintf("样本%d", n)}
		if n%3 == 0 {
			changes["shortName"] = fmt.Sprintf("EN%d", n)
		}
		saved := s.edit("H45", "H45_FIRST", "correct", map[string]any{"changes": changes}, who, 200)
		sources = append(sources, saved["sourceVersion"].(string))
	}
	path := "/api/v1/departments/" + s.aliases["H45"]
	upper := "throughSourceVersion=" + sources[44]
	seen := map[string]bool{}
	for page, want := range []int{20, 20, 5} {
		r := logRead(s.b, path, fmt.Sprintf("%s&page=%d", upper, page+1))
		if r["total"] != float64(45) || len(r["items"].([]any)) != want {
			t.Fatal("H45 pages", r)
		}
		for _, v := range r["items"].([]any) {
			id := v.(map[string]any)["sourceVersion"].(string)
			if seen[id] {
				t.Fatal("duplicate page member")
			}
			seen[id] = true
		}
	}
	if len(seen) != 45 {
		t.Fatal("missing H45 source")
	}
	for _, size := range []int{20, 50, 100} {
		r := logRead(s.b, path, fmt.Sprintf("%s&pageSize=%d&page=999", upper, size))
		wantPage, wantCount := 1, 45
		if size == 20 {
			wantPage, wantCount = 3, 5
		}
		if r["page"] != float64(wantPage) || len(r["items"].([]any)) != wantCount {
			t.Fatal("H45 clamp", r)
		}
	}
	for _, tc := range []struct {
		field, op, value string
		want             int
	}{
		{"actor", "equals", " ADMIN ", 23}, {"actor", "contains", "乙操作", 22}, {"changedField", "is", "shortName", 15}, {"changedField", "isAnyOf", "name,shortName", 45},
		{"action", "is", "CORRECT", 44}, {"action", "is", "CREATE", 1}, {"action", "isAnyOf", "CREATE,CORRECT", 45}, {"effectiveDate", "is", "2026-01-01", 45}, {"actor", "contains", "%_\\", 0},
	} {
		r := logRead(s.b, path, upper+"&"+logFilter(tc.field, tc.op, tc.value))
		if r["total"] != float64(tc.want) {
			t.Fatalf("H45 filter %s/%s: %v", tc.field, tc.value, r["total"])
		}
	}
	for _, q := range []string{"page=", "page=0", "pageSize=10", "throughSourceVersion=", "contextSourceVersion=", "contextSourceVersion=x", "page=1&page=2", "filterField=actor", logFilter("action", "isAnyOf", "CREATE,CREATE"), logFilter("actor", "contains", strings.Repeat("甲", 401)), logFilter("changedField", "is", "unknown"), logFilter("recordedAt", "is", "2026-02-30"), "businessAt=2026-01-01", "filterValue=x&filterValue=y"} {
		s.b.call("GET", path+"/changes?"+q, nil, "", 400)
	}
	s.b.call("GET", path+"/changes?throughSourceVersion="+foundation.NewID(), nil, "", 404)
	other := s.create("OTHER", "其他", "BU_A", "", "2026-01-01", s.b, 201)
	s.b.call("GET", path+"/changes?throughSourceVersion="+other["sourceVersion"].(string), nil, "", 404)
	g := t07Instance(t, s.f, "2020-01-01")
	foreign := g.login("admin", g.password)
	bu := foreign.call("POST", "/api/v1/business-units", createBody("FOREIGN", "2026-01-01"), foundation.NewID(), 201)
	dep := foreign.call("POST", "/api/v1/departments", departmentBody("FOREIGN", "外部部门", bu["buId"].(string), "2026-01-01", nil), foundation.NewID(), 201)
	for _, key := range []string{"throughSourceVersion", "contextSourceVersion"} {
		s.b.call("GET", path+"/changes?"+key+"="+dep["sourceVersion"].(string), nil, "", 404)
	}
	foreign.call("GET", path+"/changes", nil, "", 404)
	if logRead(s.b, path, "filterField=&filterOperator=&filterValue=")["total"] != float64(45) {
		t.Fatal("empty filter")
	}
	mutateLifeGrants(s.f, s.actor.subject, `UPDATE access.subject_grant SET read_fields=ARRAY['code'] WHERE principal_id=$1 AND action='department.history.read'`, s.actor.subject)
	restricted := logRead(s.actor, path, "")
	if !containsField(restricted["detailReadFields"].([]any), "shortName") {
		t.Fatal("detail metadata narrowed")
	}
	historical := logRead(s.actor, path, "contextSourceVersion="+sources[44])
	if !reflect.DeepEqual(historical["detailReadFields"], historical["readFields"]) {
		t.Fatal("historical fields")
	}
	delete(restricted, "detailReadFields")
	raw, _ := json.Marshal(restricted)
	for _, secret := range []string{"样本", "EN45", "effectiveDate", "operationReason", "shortName", "endInclusive"} {
		if strings.Contains(string(raw), secret) {
			t.Fatal("hidden history leak", secret)
		}
	}
	if restricted["total"] != float64(45) || len(directChange(t, restricted["items"].([]any)[0].(map[string]any))["diffs"].([]any)) != 0 {
		t.Fatal("generic history row")
	}
	for _, q := range []string{logFilter("changedField", "is", "shortName"), logFilter("changedField", "isAnyOf", "state,period"), logFilter("effectiveDate", "is", "2026-01-01")} {
		s.actor.call("GET", path+"/changes?"+q, nil, "", 403)
	}
	// Later writes cannot move the captured page window.
	s.edit("H45", "H45_FIRST", "correct", map[string]any{"changes": map[string]any{"name": "窗口之后"}}, s.b, 200)
	if logRead(s.b, path, upper)["total"] != float64(45) {
		t.Fatal("fixed upper moved")
	}
}

func TestDepartmentPairFutureAndNoChangeSummaryOracles(t *testing.T) {
	s := newDepartmentContract(t)
	allDepartmentRights(s)
	s.create("P", "配对", "BU_A", "", "2026-01-01", s.b, 201)
	path := "/api/v1/departments/" + s.aliases["P"]
	d := s.b.departmentCurrent(s.aliases["P"])
	input := pairInsertBody(d, selected(d), "2026-06-01", "2026-09-01")
	input["deactivationReason"] = "ORG_RESTRUCTURING"
	p2 := s.b.call("POST", path+"/lifecycle-pairs", input, foundation.NewID(), 200)
	d = s.b.departmentCurrent(s.aliases["P"])
	stop, resume := recordOn(t, d, "2026-06-01"), recordOn(t, d, "2026-09-01")
	s.records["P_RESUME"] = resume["recordId"].(string)
	p3 := s.edit("P", "P_RESUME", "correct", map[string]any{"changes": map[string]any{"name": "恢复修订"}}, s.actor, 200)
	d = s.b.departmentCurrent(s.aliases["P"])
	stop, resume = recordOn(t, d, "2026-06-01"), recordOn(t, d, "2026-09-01")
	p4 := s.b.call("POST", path+"/lifecycle-pairs/delete", pairDeleteBody(d, stop, resume), foundation.NewID(), 200)
	for _, tc := range []struct {
		r             map[string]any
		source, actor string
	}{{stop, p2["sourceVersion"].(string), s.b.subject}, {resume, p3["sourceVersion"].(string), s.actor.subject}} {
		for _, src := range []string{p3["sourceVersion"].(string), p4["sourceVersion"].(string)} {
			original := s.b.call("GET", path+"?sourceVersion="+src+"&selectedRecordId="+tc.r["recordId"].(string), nil, "", 200)
			summaryIs(t, original, tc.r["recordId"].(string), tc.r["revision"].(string), tc.source, tc.actor)
			if src == p4["sourceVersion"] && selected(original)["deleted"] != true {
				t.Fatal("pair original missing")
			}
		}
		found := false
		for _, v := range p4["items"].([]any) {
			it := v.(map[string]any)
			if it["recordId"] == tc.r["recordId"] {
				found = true
				if it["beforeRecord"].(map[string]any)["revision"] != tc.r["revision"] {
					t.Fatal("pair result original")
				}
			}
		}
		if !found {
			t.Fatal("pair result absent")
		}
	}
	log := logRead(s.b, path, "")["items"].([]any)[0].(map[string]any)
	if len(log["items"].([]any)) != 3 || log["operator"].(map[string]any)["subjectId"] != s.b.subject {
		t.Fatal("pair log attribution")
	}
	s.create("F", "未来甲", "BU_A", "", "2026-11-10", s.b, 201)
	f2 := s.append("F", "F_TWO", "2026-12-01", "CHANGE", map[string]any{"name": "未来乙"}, s.b, 201)
	_, r2 := s.target("F", "F_TWO")
	f3 := s.edit("F", "F_TWO", "delete", nil, s.actor, 200)
	path = "/api/v1/departments/" + s.aliases["F"]
	current := s.b.departmentCurrent(s.aliases["F"])
	r1 := selected(current)
	if current["result"] != "M02_NO_FACT" || current["selectedRecordIsFuture"] != true {
		t.Fatal("future default")
	}
	summaryIs(t, current, r1["recordId"].(string), r1["revision"].(string), f3["sourceVersion"].(string), s.actor.subject)
	for _, src := range []string{f2["sourceVersion"].(string), f3["sourceVersion"].(string)} {
		original := s.b.call("GET", path+"?sourceVersion="+src+"&selectedRecordId="+r2["recordId"].(string), nil, "", 200)
		summaryIs(t, original, r2["recordId"].(string), r2["revision"].(string), f2["sourceVersion"].(string), s.b.subject)
	}
	for _, q := range []string{"", "&sourceVersion=" + f3["sourceVersion"].(string)} {
		missing := s.b.call("GET", path+"?selectedRecordId="+foundation.NewID()+q, nil, "", 200)
		if missing["selectedRecord"] != nil || missing["selectionMissing"] != true {
			t.Fatal("missing selection defaulted")
		}
	}
	for _, patch := range []map[string]any{nil, {}} {
		result := s.append("F", "UNUSED", "2026-12-01", "CHANGE", patch, s.b, 200)
		if result["outcome"] != "no_change" {
			t.Fatal("empty addition not no_change", result)
		}
		if s.b.departmentCurrent(s.aliases["F"])["headSourceVersion"] != f3["sourceVersion"] {
			t.Fatal("empty addition changed source")
		}
	}
	f4 := s.append("F", "F_NEW", "2026-12-01", "CHANGE", map[string]any{"name": "未来重建"}, s.b, 201)
	if f4["targetRecordId"] == r2["recordId"] {
		t.Fatal("deleted identity reused")
	}
	mutateLifeGrants(s.f, s.actor.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='department.history.read'`, s.actor.subject)
	visible := s.actor.call("GET", path, nil, "", 200)
	if summaryOf(t, visible) == nil {
		t.Fatal("ordinary summary lost")
	}
	s.actor.call("GET", path+"/changes", nil, "", 403)
	s.actor.call("GET", path+"?sourceVersion="+f4["sourceVersion"].(string), nil, "", 403)
	mutateLifeGrants(s.f, s.actor.subject, `UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'effectiveDate') WHERE principal_id=$1 AND action='department.read'`, s.actor.subject)
	hidden := s.actor.call("GET", path, nil, "", 200)
	if hidden["selectedRecordIsFuture"] == true || selected(hidden)["effectiveDate"] != nil || hidden["path"].(map[string]any)["pathMode"] != "restricted" || !reflect.DeepEqual(summaryOf(t, visible), summaryOf(t, hidden)) {
		t.Fatal("date loss leaked or summary changed")
	}
	mutateLifeGrants(s.f, s.actor.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='department.read'`, s.actor.subject)
	s.actor.call("GET", path, nil, "", 404)
}

func TestDepartmentS1S7SummaryOracles(t *testing.T) {
	s := newDepartmentContract(t)
	allDepartmentRights(s)
	s.create("ROOT", "总部", "BU_A", "", "2026-01-01", s.b, 201)
	at := time.Date(2026, 11, 3, 9, 0, 0, 0, foundation.Shanghai)
	restore := bootstrap.SetIntegrationClock(s.f.runtime, func() time.Time { return at })
	defer restore()
	s.f.exec(`UPDATE access.principal SET display_name=CASE login WHEN 'admin' THEN '李明' ELSE '王芳' END,login=CASE login WHEN 'admin' THEN 'liming' ELSE 'wangfang' END`)
	s.create("S", "华东一", "BU_A", "ROOT", "2026-01-01", s.b, 201)
	at = time.Date(2026, 11, 3, 9, 1, 0, 0, foundation.Shanghai)
	s2 := s.append("S", "R2", "2026-06-01", "CHANGE", map[string]any{"name": "华东二"}, s.actor, 201)
	_, r1 := s.target("S", "S_FIRST")
	at = time.Date(2026, 11, 3, 9, 2, 0, 0, foundation.Shanghai)
	s3 := s.append("S", "R3", "2026-10-01", "CHANGE", map[string]any{"name": "华东三"}, s.b, 201)
	_, r2 := s.target("S", "R2")
	at = time.Date(2026, 11, 3, 9, 3, 0, 0, foundation.Shanghai)
	s4 := s.edit("S", "R3", "correct", map[string]any{"changes": map[string]any{"name": "华东三新", "shortName": "East Three"}}, s.actor, 200)
	path := "/api/v1/departments/" + s.aliases["S"]
	read := func(src, record string) *map[string]any {
		q := url.Values{"selectedRecordId": {record}}
		if src != "" {
			q.Set("sourceVersion", src)
		}
		v := s.b.call("GET", path+"?"+q.Encode(), nil, "", 200)
		return &v
	}
	summaryIs(t, *read("", s.records["S_FIRST"]), s.records["S_FIRST"], r1["revision"].(string), s2["sourceVersion"].(string), s.actor.subject)
	summaryIs(t, *read("", s.records["R2"]), s.records["R2"], r2["revision"].(string), s3["sourceVersion"].(string), s.b.subject)
	currentR3 := *read("", s.records["R3"])
	summaryIs(t, currentR3, s.records["R3"], selected(currentR3)["revision"].(string), s4["sourceVersion"].(string), s.actor.subject)
	mutateLifeGrants(s.f, s.actor.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='department.history.read'`, s.actor.subject)
	for _, tc := range []struct{ record, source, stamp string }{{s.records["S_FIRST"], s2["sourceVersion"].(string), "2026-11-03T09:01:00+08:00"}, {s.records["R3"], s4["sourceVersion"].(string), "2026-11-03T09:03:00+08:00"}} {
		basic := s.actor.call("GET", path+"?selectedRecordId="+tc.record, nil, "", 200)
		summary := summaryOf(t, basic)
		stamp, err := time.Parse(time.RFC3339, summary["recordedAt"].(string))
		must(t, err)
		if summary["operationSourceVersion"] != tc.source || stamp.In(foundation.Shanghai).Format(time.RFC3339) != tc.stamp {
			t.Fatal("fixed S4 summary", summary)
		}
	}
	s.actor.call("GET", path+"/changes", nil, "", 403)
	s.actor.call("GET", path+"?sourceVersion="+s4["sourceVersion"].(string), nil, "", 403)
	allDepartmentRights(s)
	s.edit("ROOT", "ROOT_FIRST", "correct", map[string]any{"changes": map[string]any{"name": "新总部"}}, s.b, 200)
	if !reflect.DeepEqual(summaryOf(t, currentR3), summaryOf(t, *read("", s.records["R3"]))) {
		t.Fatal("ancestor renamed summary")
	}
	at = time.Date(2026, 11, 3, 9, 4, 0, 0, foundation.Shanghai)
	s5 := s.edit("S", "R2", "correct", map[string]any{"changes": map[string]any{"name": "华东二新"}}, s.b, 200)
	_, r2 = s.target("S", "R2")
	no := s.edit("S", "R2", "correct", map[string]any{"changes": map[string]any{"name": "华东二新"}}, s.b, 200)
	if no["outcome"] != "no_change" || s.current("S")["headSourceVersion"] != s5["sourceVersion"] {
		t.Fatal("same correction changed source")
	}
	at = time.Date(2026, 11, 3, 9, 5, 0, 0, foundation.Shanghai)
	s6 := s.edit("S", "R2", "delete", nil, s.actor, 200)
	current := *read("", s.records["S_FIRST"])
	summaryIs(t, current, s.records["S_FIRST"], selected(current)["revision"].(string), s6["sourceVersion"].(string), s.actor.subject)
	if s.append("S", "EMPTY", "2026-06-01", "CHANGE", map[string]any{}, s.b, 200)["outcome"] != "no_change" {
		t.Fatal("S7 empty patch")
	}
	at = time.Date(2026, 11, 3, 9, 6, 0, 0, foundation.Shanghai)
	s7 := s.append("S", "R4", "2026-06-01", "CHANGE", map[string]any{"name": "华东四"}, s.b, 201)
	if s7["targetRecordId"] == s.records["R2"] {
		t.Fatal("S7 reused R2")
	}
	for _, src := range []string{s5["sourceVersion"].(string), s6["sourceVersion"].(string)} {
		original := *read(src, s.records["R2"])
		summaryIs(t, original, s.records["R2"], r2["revision"].(string), s5["sourceVersion"].(string), s.b.subject)
		if selected(original)["name"] != "华东二新" {
			t.Fatal("S7 rewrote original")
		}
	}
	// Current S7 still separates ordinary summary from history permission.
	allDepartmentRights(s)
	mutateLifeGrants(s.f, s.actor.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='department.history.read'`, s.actor.subject)
	if summaryOf(t, s.actor.call("GET", path, nil, "", 200)) == nil {
		t.Fatal("summary depends on history")
	}
	s.actor.call("GET", path+"/changes", nil, "", 403)
	s.actor.call("GET", path+"?sourceVersion="+s7["sourceVersion"].(string), nil, "", 403)
}

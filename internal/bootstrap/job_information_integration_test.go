//go:build integration

package bootstrap_test

import (
	"maps"
	"reflect"
	"testing"

	"chnhcm.local/chnhcm/internal/bootstrap"
	f "chnhcm.local/chnhcm/internal/foundation"
)

func TestAssignmentLocationChangesPreserveBusinessAndRevisionIdentity(t *testing.T) {
	type M = map[string]any
	h := legalSetup206(t)
	b := h.b
	init := bootstrap.InitInput{Instance: h.f.instance, EnterpriseCode: "LOCAL", EnterpriseName: "本地验证企业", Login: "admin", Password: h.f.password, PersonnelProfile: "registrar", PersonalProfileFixture: true, HireIdentityFixture: true, EmploymentFixture: true, FirstHireRule: &bootstrap.FirstHireRuleInput{Version: "T02-employee-1", From: "2027-04-01"}}
	must(t, bootstrap.Initialize(h.f.ctx, h.f.admin, init))
	entity := h.create("LOC42", "CN", "legal_person")["id"].(string)
	b.call("POST", "/api/v1/legal-entities/"+entity+"/purposes", M{"expectedVersion": "", "row": M{"id": f.NewID(), "role": "employer", "kind": "declaration", "basis": "工作地点试办用人资格", "from": "2027-04-01"}}, f.NewID(), 200)
	bu := b.call("POST", "/api/v1/business-units", createBody("BU42", "2026-01-01"), f.NewID(), 201)["buId"].(string)
	dep := b.call("POST", "/api/v1/departments", departmentCreateBody("D42", bu), f.NewID(), 201)["departmentId"].(string)
	staff := staffing307{t, h.f, b}
	family := staff.create("jobfamily", "F42", "")["id"].(string)
	job := staff.create("job", "J42", family)["id"].(string)
	b.call("POST", "/api/v1/assignment-policies", M{"expectedVersion": "", "row": M{"id": f.NewID(), "category": "Employee", "purpose": "hire", "mode": "JobOnly", "from": "2027-04-01"}}, f.NewID(), 200)
	person := b.call("POST", "/api/v1/persons", M{"submissionId": f.NewID(), "name": "工作地点试办陈禾", "primaryDocument": M{"kind": "passport", "number": "TEST-LOC42"}}, "", 200)["id"].(string)
	loc := &location204{t, h.f, b, h.countries}
	var locations []string
	for _, code := range []string{"A42", "B42", "C42", "D42"} {
		locations = append(locations, loc.create(code, "MO", "", "2026-01-01")["id"].(string))
	}
	hire := b.call("POST", "/api/v1/hires", M{"person": person, "employer": entity, "department": dep, "job": job, "arrival": "2027-04-01", "location": locations[0], "notice": "合成首任通知"}, f.NewID(), 200)
	base := "/api/v1/assignments/" + hire["assignment"].(string)
	view := func(date string, source string) M {
		q := base + "?businessAt=" + date
		if source != "" {
			q += "&sourceVersion=" + source
		}
		return b.call("GET", q, nil, "", 200)
	}
	location := func(date, source string) string {
		t.Helper()
		return view(date, source)["state"].(M)["location"].(M)["id"].(string)
	}
	list := func(day string) M {
		return b.call("POST", "/api/v1/assignments/query", M{"businessAt": day, "page": 1, "pageSize": 20}, "", 200)
	}
	if before := list("2027-03-31"); before["total"] != 0.0 || before["people"] != 0.0 {
		t.Fatal(before)
	}
	directory := list("2027-04-01")
	if directory["total"] != 1.0 || directory["people"] != 1.0 || directory["rows"].([]any)[0].(M)["id"] != hire["assignment"] || directory["departments"].([]any)[0].(M)["id"] != dep {
		t.Fatal(directory)
	}
	start := view("2027-04-01", "")
	if start["recordNumber"] != 0.0 || start["canChangeLocation"] != true || location("2027-04-30", "") != locations[0] {
		t.Fatal(start)
	}
	first := M{"businessDate": "2027-05-01", "expectedSourceVersion": start["sourceVersion"], "location": locations[1]}
	newPath := base + "/location-changes"
	firstKey := f.NewID()
	saved := b.call("POST", newPath, first, firstKey, 200)
	again := b.call("POST", newPath, first, firstKey, 200)
	if !reflect.DeepEqual(saved, again) || saved["recordNumber"] != 0.0 {
		t.Fatal(saved, again)
	}
	may := view("2027-05-01", "")
	if location("2027-05-01", "") != locations[1] || len(may["actions"].([]any)) != 2 {
		t.Fatal(may)
	}
	options := rows400(t, b, base+"/location-options", M{"businessDate": "2027-05-01", "page": 1, "search": "C42"})
	if len(options) != 1 || options[0]["id"] != locations[2] {
		t.Fatal("未取得获权真实地点", options)
	}
	target := saved["action"].(string)
	correctPath := newPath + "/" + target + "/correct"
	correct := M{"businessDate": "2027-05-01", "expectedSourceVersion": may["sourceVersion"], "expectedRevision": saved["revision"], "location": locations[2]}
	changed := b.call("POST", correctPath, correct, f.NewID(), 200)
	if changed["action"] != target || changed["revision"] == saved["revision"] {
		t.Fatal(changed)
	}
	if location("2027-04-30", "") != locations[0] || location("2027-05-01", "") != locations[2] || location("2027-05-01", may["sourceVersion"].(string)) != locations[1] {
		t.Fatal("当日或旧来源地点不符")
	}
	orig := "/api/v1/employment-actions/" + target
	latest := b.call("GET", orig, nil, "", 200)
	old := b.call("GET", orig+"?adoptedRef="+saved["revision"].(string), nil, "", 200)
	after := func(v M) any { return v["locationChange"].(M)["after"].(M)["id"] }
	if after(old) != locations[1] || after(latest) != locations[2] || len(latest["revisions"].([]any)) != 2 {
		t.Fatal(old, latest)
	}
	b.call("POST", correctPath, correct, f.NewID(), 409)
	current := view("2027-05-01", "")
	if len(current["actions"].([]any)) != 2 {
		t.Fatal(current)
	}
	next := M{"businessDate": "2027-06-01", "expectedSourceVersion": current["sourceVersion"], "location": locations[3]}
	h.f.exec("ALTER TABLE staffing.employment_outbox ADD CONSTRAINT test_no_location_event CHECK(false) NOT VALID")
	b.call("POST", newPath, next, f.NewID(), 503)
	h.f.exec("ALTER TABLE staffing.employment_outbox DROP CONSTRAINT test_no_location_event")
	if got := view("2027-06-01", ""); got["sourceVersion"] != current["sourceVersion"] || len(got["actions"].([]any)) != 2 {
		t.Fatal("回滚未还原", got)
	}
	second := b.call("POST", newPath, next, f.NewID(), 200)
	if second["action"] == target || location("2027-06-01", "") != locations[3] {
		t.Fatal(second)
	}
	head := view("2027-06-01", "")
	if len(head["actions"].([]any)) != 3 {
		t.Fatal(head)
	}
	type result struct {
		code int
		err  error
	}
	results := make(chan result, 2)
	for _, id := range []string{locations[0], locations[1]} {
		cmd := maps.Clone(correct)
		cmd["expectedSourceVersion"] = head["sourceVersion"]
		cmd["expectedRevision"] = changed["revision"]
		cmd["location"] = id
		go func() { status, _, e := b.response("POST", correctPath, cmd, f.NewID()); results <- result{status, e} }()
	}
	counts := map[int]int{}
	for range 2 {
		r := <-results
		must(t, r.err)
		counts[r.code]++
	}
	if counts[200] != 1 || counts[409] != 1 || location("2027-06-01", "") != locations[3] {
		t.Fatal("竞争更正或独立后继异常", counts)
	}
	if got := b.call("POST", "/api/v1/hires/submissions/"+firstKey+"/resolve", nil, "", 200); !reflect.DeepEqual(got, saved) {
		t.Fatal("断线查回未保留首次提交结果", got)
	}
	h.policy("DELETE FROM access.employment_grant WHERE principal_id=$1 AND action IN ('location.change','location.correct')", b.subject)
	must(t, bootstrap.Initialize(h.f.ctx, h.f.admin, init))
	if view("2027-06-01", "")["canChangeLocation"] != false {
		t.Fatal("重新初始化恢复了撤权")
	}
	b.call("POST", newPath, next, f.NewID(), 403)
	h.policy("DELETE FROM access.employment_grant WHERE principal_id=$1 AND action='history'", b.subject)
	b.call("GET", orig+"?adoptedRef="+saved["revision"].(string), nil, "", 403)
	b.call("GET", base+"?sourceVersion="+may["sourceVersion"].(string), nil, "", 403)
	h.policy("DELETE FROM access.employment_grant WHERE principal_id=$1 AND action='read'", b.subject)
	if after := list("2027-06-01"); after["total"] != 0.0 || len(after["departments"].([]any)) != 0 {
		t.Fatal("撤权后名录或树泄露", after)
	}
	b.call("GET", base, nil, "", 403)
	if got := b.call("POST", "/api/v1/hires/submissions/"+firstKey+"/resolve", nil, "", 200); got["recordNumber"] != nil || got["revision"] != nil {
		t.Fatal("撤权后回执披露", got)
	}
}

//go:build integration

package bootstrap_test

import (
	"reflect"
	"strings"
	"testing"

	"chnhcm.local/chnhcm/internal/foundation"
)

const plans210 = "/api/v1/compensation/plans"
const rates210 = "/api/v1/compensation/grade-rates"

type planBody = map[string]any

func fixed210(code, name, amount, frequency string) planBody {
	return planBody{"id": foundation.NewID(), "code": code, "name": name, "nature": "ALLOWANCE", "method": "FIXED", "currency": "CNY", "frequency": frequency, "unit": "PER_" + frequency, "precision": 2, "amount": amount}
}
func sourceRow210(grade string, rate planBody, frequency string) planBody {
	return planBody{"id": foundation.NewID(), "code": "BASIC", "name": "基本工资", "nature": "BASIC", "method": "GRADE_RATE", "currency": "CNY", "frequency": frequency, "unit": "PER_" + frequency, "precision": 2, "rates": []planBody{{"gradeId": grade, "rateId": rate["objectId"]}}}
}
func target210(base, percent, freq string) planBody {
	return planBody{"id": foundation.NewID(), "code": "BONUS", "name": "目标奖金", "nature": "BONUS", "method": "PERCENT", "currency": "CNY", "frequency": freq, "unit": "PER_" + freq, "precision": 2, "baseId": base, "percent": percent, "rounding": "HALF_UP"}
}
func newPlan210(code string, scope planBody, rows ...planBody) planBody {
	return planBody{"action": "CREATE", "code": code, "effectiveDate": "2027-01-01", "reason": "建立计划", "changes": planBody{"name": code + " 制度", "scope": scope, "components": rows}}
}
func command210(action string, prior planBody, day string, changes planBody) planBody {
	out := planBody{"action": action, "recordId": prior["recordId"], "expectedRevision": prior["revision"], "expectedVersion": prior["sourceVersion"]}
	if day != "" {
		out["effectiveDate"], out["reason"] = day, "薪酬制度调整"
	}
	if changes != nil {
		out["changes"] = changes
	}
	return out
}
func resultPath210(base string, result planBody) string {
	return base + "/" + result["objectId"].(string)
}
func component210(rec planBody, code string) planBody {
	for _, i := range rec["components"].([]any) {
		r := i.(planBody)
		if r["code"] == code {
			return r
		}
	}
	return nil
}
func TestCompensationPlanFixedAndPermissions(t *testing.T) {
	h := staffingSetup307(t)
	b, f := h.b, h.f
	enterprise := planBody{"mode": "ENTERPRISE"}
	allow := fixed210("ALLOW", "岗位津贴", "0", "MONTH")
	input := newPlan210("ALLOW-M", enterprise, allow)
	id := foundation.NewID()
	first := b.call("POST", plans210, input, id, 200)
	path := resultPath210(plans210, first)
	if v := b.call("POST", plans210, input, id, 200); !reflect.DeepEqual(v, first) {
		t.Fatal(v)
	}
	b.call("POST", plans210, input, foundation.NewID(), 409)
	if f.scalar("SELECT count(*) FROM configuration.plan_scope_version") != 1 {
		t.Fatal("failed duplicate left a policy")
	}
	initial := b.call("GET", path, nil, "", 200)["record"].(planBody)
	scope := initial["scope"].(planBody)["version"]
	if component210(initial, "ALLOW")["amount"] != "0.00" {
		t.Fatal(initial)
	}
	if got := b.call("GET", plans210+"?q=ALLOW-M", nil, "", 200)["total"]; got != float64(1) {
		t.Fatal(got)
	}
	allow["amount"] = "300"
	add := command210("ADD", first, "2027-10-01", planBody{"components": []planBody{allow}})
	second := b.call("POST", path, add, foundation.NewID(), 200)
	b.call("POST", path, add, foundation.NewID(), 409)
	for day, want := range map[string]string{"2027-09-30": "0.00", "2027-10-01": "300.00"} {
		r := b.call("GET", path+"?businessAt="+day, nil, "", 200)["record"].(planBody)
		if r["scope"].(planBody)["version"] != scope || component210(r, "ALLOW")["amount"] != want {
			t.Fatal(r)
		}
	}
	invalid := newPlan210("BAD", planBody{}, allow)
	b.call("POST", plans210, invalid, foundation.NewID(), 422)
	f.exec("UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'amount') WHERE principal_id=$1 AND action='compplan.read'", b.subject)
	f.exec("UPDATE access.subject_grant SET write_fields=ARRAY['name'] WHERE principal_id=$1 AND action='compplan.correct'", b.subject)
	changed := b.call("POST", path, command210("CORRECT", second, "", planBody{"name": "仅改名称"}), foundation.NewID(), 200)
	r := b.call("GET", path+"?businessAt=2027-10-01", nil, "", 200)["record"].(planBody)
	if r["name"] != "仅改名称" || component210(r, "ALLOW")["amount"] != nil {
		t.Fatal(r)
	}
	allow["amount"] = nil
	b.call("POST", path, command210("CORRECT", changed, "", planBody{"components": []planBody{allow}}), foundation.NewID(), 403)
	reader := f.login("reader", f.readPassword)
	r = reader.call("GET", path+"?businessAt=2027-10-01", nil, "", 200)["record"].(planBody)
	if component210(r, "ALLOW")["amount"] != "300.00" {
		t.Fatal("hidden amount lost when renaming", r)
	}
	if r := b.call("GET", path+"/history?sourceVersion="+second["sourceVersion"].(string), nil, "", 200)["items"].([]any)[0].(planBody)["after"].(planBody); component210(r, "ALLOW")["amount"] != nil {
		t.Fatal("history disclosed restricted amount")
	}
	if f.scalar("SELECT count(*) FROM compensation.plan_version") != 3 {
		t.Fatal("unexpected plan version count")
	}
}
func TestCompensationPlanSharedRatesAndAdoptions(t *testing.T) {
	h := staffingSetup307(t)
	b := h.b
	grade := h.create("grade", "P5", "")["id"].(string)
	createRate := func(freq string, amounts planBody) planBody {
		return b.call("POST", rates210, planBody{"action": "CREATE", "gradeId": grade, "currency": "CNY", "frequency": freq, "unit": "PER_" + freq, "reason": "建立标准", "effectiveDate": "2027-01-01", "changes": amounts}, foundation.NewID(), 200)
	}
	monthly := createRate("MONTH", planBody{"form": "RANGE", "minimum": "10000", "maximum": "15000", "guidance": "12500"})
	annual := createRate("YEAR", planBody{"form": "FIXED", "amount": "180000.04"})
	enterprise := planBody{"mode": "ENTERPRISE"}
	mbasic := sourceRow210(grade, monthly, "MONTH")
	base := sourceRow210(grade, annual, "YEAR")
	allow := fixed210("ALLOW", "岗位津贴", "300", "MONTH")
	oallow := fixed210("ALLOW", "岗位津贴", "200", "MONTH")
	pro := b.call("POST", plans210, newPlan210("PRO-M", enterprise, mbasic, allow, target210(mbasic["id"].(string), "10", "MONTH")), foundation.NewID(), 200)
	ops := b.call("POST", plans210, newPlan210("OPS-M", enterprise, sourceRow210(grade, monthly, "MONTH"), oallow), foundation.NewID(), 200)
	bonus := target210(base["id"].(string), "12.50", "YEAR")
	mgt := b.call("POST", plans210, newPlan210("MGT-Y", enterprise, base, bonus), foundation.NewID(), 200)
	record := func(path string, date string) planBody {
		return b.call("GET", path+"?businessAt="+date, nil, "", 200)["record"].(planBody)
	}
	mgtpath := resultPath210(plans210, mgt)
	propath := resultPath210(plans210, pro)
	opspath := resultPath210(plans210, ops)
	monthpath := resultPath210(rates210, monthly)
	if v := component210(record(mgtpath, "2027-01-01"), "BONUS")["targets"].([]any)[0].(planBody)["amount"]; v != "22500.01" {
		t.Fatal(v)
	}
	if v := component210(record(propath, "2027-01-01"), "BONUS")["targets"]; v != nil {
		t.Fatal("range became personal amount", v)
	}
	bonus["baseId"] = bonus["id"]
	b.call("POST", mgtpath, command210("CORRECT", mgt, "", planBody{"components": []planBody{base, bonus}}), foundation.NewID(), 422)
	july := b.call("POST", monthpath, command210("ADD", monthly, "2027-07-01", planBody{"minimum": "11000", "guidance": "13500", "maximum": "16000"}), foundation.NewID(), 200)
	getRate := func(record planBody) planBody { return component210(record, "BASIC")["rates"].([]any)[0].(planBody) }
	for _, path := range []string{propath, opspath} {
		for day, want := range map[string]string{"2027-06-30": "15000.00", "2027-07-01": "16000.00"} {
			link := getRate(record(path, day))
			if link["rateId"] != monthly["objectId"] || link["record"].(planBody)["maximum"] != want || link["adopted"].(planBody)["sourceVersion"] != monthly["sourceVersion"] {
				t.Fatal(link)
			}
		}
	}
	// The unchanged January record is still the source when a new plan is saved after the July adjustment.
	late := b.call("POST", plans210, newPlan210("AFTER", enterprise, sourceRow210(grade, monthly, "MONTH")), foundation.NewID(), 200)
	l := getRate(record(resultPath210(plans210, late), "2027-01-01"))
	if l["adoptedHead"] != july["sourceVersion"] || l["adopted"].(planBody)["sourceVersion"] != monthly["sourceVersion"] {
		t.Fatal(l)
	}
	allow["amount"] = "350"
	oct := b.call("POST", propath, command210("ADD", pro, "2027-10-01", planBody{"components": []planBody{mbasic, allow}}), foundation.NewID(), 200)
	if v := component210(record(opspath, "2027-10-01"), "ALLOW")["amount"]; v != "200.00" {
		t.Fatal(v)
	}
	b.call("POST", monthpath, command210("DELETE", july, "", nil), foundation.NewID(), 200)
	current := record(propath, "2027-10-01")
	if getRate(current)["record"].(planBody)["maximum"] != "15000.00" || component210(current, "ALLOW")["amount"] != "350.00" {
		t.Fatal(current)
	}
	orig := b.call("GET", propath+"/history?sourceVersion="+oct["sourceVersion"].(string), nil, "", 200)["items"].([]any)[0].(planBody)["after"].(planBody)
	if getRate(orig)["record"].(planBody)["maximum"] != "16000.00" || getRate(orig)["adopted"].(planBody)["sourceVersion"] != july["sourceVersion"] {
		t.Fatal(orig)
	}
	rd := b.call("GET", monthpath, nil, "", 200)
	stopRate := command210("DEACTIVATE", planBody{"sourceVersion": rd["sourceVersion"], "recordId": rd["record"].(planBody)["recordId"], "revision": rd["record"].(planBody)["revision"]}, "2028-01-01", nil)
	b.call("POST", monthpath, stopRate, foundation.NewID(), 422)
	for i, p := range []planBody{oct, ops, late} {
		b.call("POST", []string{propath, opspath, resultPath210(plans210, late)}[i], command210("DEACTIVATE", p, "2028-01-01", nil), foundation.NewID(), 200)
		if i < 2 {
			b.call("POST", monthpath, stopRate, foundation.NewID(), 422)
		}
	}
	b.call("POST", monthpath, stopRate, foundation.NewID(), 200)
}

func TestCompensationPlanScopedReferences(t *testing.T) {
	h := legalSetup206(t)
	b := h.b
	st := &staffing307{t, h.f, b}
	entity := h.create("A210", "CN", "legal_person")["id"].(string)
	other := h.create("B210", "CN", "legal_person")["id"].(string)
	family := st.create("jobfamily", "PRO210", "")
	prof := family["id"].(string)
	ops := st.create("jobfamily", "OPS210", "")["id"].(string)
	member := func(dim, id string, exclude bool) planBody {
		return planBody{"dimension": dim, "id": id, "exclude": exclude}
	}
	choices := []planBody{member("legalEntity", entity, false), member("legalEntity", other, false), member("jobFamily", prof, false), member("jobFamily", ops, false), member("jobFamily", ops, true)}
	scope := planBody{"mode": "SELECTED", "choices": choices}
	row := fixed210("ALLOW", "岗位津贴", "300", "MONTH")
	p := b.call("POST", plans210, newPlan210("SCOPED210", scope, row), foundation.NewID(), 200)
	path := resultPath210(plans210, p)
	info := b.call("GET", path, nil, "", 200)["record"].(planBody)["scope"].(planBody)
	if len(info["members"].([]any)) != 5 {
		t.Fatal(info)
	}
	for _, c := range []struct {
		legal, fam string
		want       bool
	}{{entity, prof, true}, {other, prof, true}, {entity, ops, false}, {"", prof, false}} {
		got := b.call("GET", path+"/applicability?businessAt=2027-01-01&legalEntityId="+c.legal+"&jobFamilyId="+c.fam, nil, "", 200)
		if got["applicable"] != c.want {
			t.Fatal(c, got)
		}
	}
	st.add("jobfamily", family, "DEACTIVATE", "2027-03-01", nil, 200)
	paused := st.detail("jobfamily", prof, "")["timeline"].([]any)[1].(planBody)
	pausedSource := st.detail("jobfamily", prof, paused["recordId"].(string))
	st.add("jobfamily", pausedSource, "REACTIVATE", "2027-04-01", nil, 200)
	before := h.f.scalar("SELECT count(*) FROM configuration.plan_scope_version")
	gap := b.call("POST", plans210, newPlan210("GAP210", scope, row), foundation.NewID(), 422)
	if gap["field"] != "scope" {
		t.Fatal(gap)
	}
	if h.f.scalar("SELECT count(*) FROM configuration.plan_scope_version") != before {
		t.Fatal("failed scope adopted")
	}
	// A name-only correction does not re-adopt the old scope or any hidden amounts.
	h.policy("DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='legalentity.read'", b.subject)
	limited := b.call("GET", path, nil, "", 200)["record"].(planBody)["scope"].(planBody)
	if limited["restricted"] != true {
		t.Fatal(limited)
	}
	relabel := b.call("POST", path, command210("CORRECT", p, "", planBody{"name": "限定范围计划"}), foundation.NewID(), 200)
	b.call("POST", path, command210("CORRECT", relabel, "", planBody{"scope": planBody{"mode": "ENTERPRISE"}}), foundation.NewID(), 403)
}

func TestCompensationPlanReferenceOnlyAndGradeMapping(t *testing.T) {
	h := staffingSetup307(t)
	b, f := h.b, h.f
	p4 := h.create("grade", "P4", "")["id"].(string)
	p5 := h.create("grade", "P5", "")["id"].(string)
	newRate := func(grade string) planBody {
		return b.call("POST", rates210, planBody{"action": "CREATE", "gradeId": grade, "currency": "CNY", "frequency": "MONTH", "unit": "PER_MONTH", "reason": "建立标准", "effectiveDate": "2027-01-01", "changes": planBody{"form": "FIXED", "amount": "4200.00"}}, foundation.NewID(), 200)
	}
	r4, r5 := newRate(p4), newRate(p5)
	f.exec("DELETE FROM access.subject_grant WHERE principal_id=$1 AND action IN ('graderate.read','graderate.history.read')", b.subject)
	b.call("GET", resultPath210(rates210, r4), nil, "", 404)
	query := plans210 + "/rate-candidates?businessAt=2027-01-01&currency=CNY&frequency=MONTH&q=P"
	choices := b.call("GET", query, nil, "", 200)["items"].([]any)
	if len(choices) != 2 {
		t.Fatal(choices)
	}
	for _, value := range choices {
		r := value.(planBody)
		if r["name"] == nil || r["amount"] != nil || r["form"] != nil {
			t.Fatal(r)
		}
	}
	grades := []planBody{{"dimension": "grade", "id": p4}, {"dimension": "grade", "id": p5}}
	scope := planBody{"mode": "SELECTED", "choices": grades}
	row := sourceRow210(p4, r4, "MONTH")
	allowance := fixed210("ALLOW", "本计划津贴", "300", "MONTH")
	input := newPlan210("GRADE-MAP", scope, row, allowance)
	if m := b.call("POST", plans210, input, foundation.NewID(), 422); m["field"] != "components.0.rates" {
		t.Fatal(m)
	}
	row["rates"] = []planBody{{"gradeId": p4, "rateId": r4["objectId"]}, {"gradeId": p5, "rateId": r5["objectId"]}}
	saved := b.call("POST", plans210, input, foundation.NewID(), 200)
	doc := b.call("GET", resultPath210(plans210, saved), nil, "", 200)["record"].(planBody)
	links := component210(doc, "BASIC")["rates"].([]any)
	if len(links) != 2 {
		t.Fatal(links)
	}
	for _, value := range links {
		link := value.(planBody)
		if link["record"] != nil || link["open"] == true || link["adopted"] != nil || link["name"] == nil {
			t.Fatal(link)
		}
	}
	f.exec("DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='graderate.reference'", b.subject)
	// This correction changes only the plan's separate fixed allowance. The
	// original standard mapping is not re-adopted under unrelated permissions.
	patch := planBody{"components": []planBody{{"id": row["id"]}, {"id": allowance["id"], "amount": "325"}}}
	b.call("POST", resultPath210(plans210, saved), command210("CORRECT", saved, "", patch), foundation.NewID(), 200)
	reader := f.login("reader", f.readPassword)
	after := reader.call("GET", resultPath210(plans210, saved), nil, "", 200)["record"].(planBody)
	if component210(after, "ALLOW")["amount"] != "325.00" || component210(after, "BASIC")["rates"].([]any)[0].(planBody)["adopted"].(planBody)["sourceVersion"] != r4["sourceVersion"] {
		t.Fatal(after)
	}
	b.call("GET", query, nil, "", 403)
	for _, rate := range []any{r4["objectId"], foundation.NewID()} {
		row["rates"] = []planBody{{"gradeId": p4, "rateId": rate}}
		b.call("POST", plans210, newPlan210("DENIED-M", planBody{"mode": "ENTERPRISE"}, row), foundation.NewID(), 403)
	}
}

// A scope retaining a grade records its exact prior name; only adopting the
// source anew rechecks the corrected catalog segment and complete coverage.
func TestCompensationPlanGradeChangeAndOriginalScope(t *testing.T) {
	h := staffingSetup307(t)
	b := h.b
	grade := h.create("grade", "P5-ORIGIN", "")
	id := grade["id"].(string)
	rate := b.call("POST", rates210, planBody{"action": "CREATE", "gradeId": id, "currency": "CNY", "frequency": "YEAR", "unit": "PER_YEAR", "effectiveDate": "2027-01-01", "reason": "建立标准", "changes": planBody{"form": "FIXED", "amount": "180000"}}, foundation.NewID(), 200)
	basic := sourceRow210(id, rate, "YEAR")
	bonus := target210(basic["id"].(string), "12.50", "YEAR")
	scope := planBody{"mode": "SELECTED", "choices": []planBody{{"dimension": "grade", "id": id}}}
	initial := b.call("POST", plans210, newPlan210("P5-ORIG-Y", scope, basic, bonus), foundation.NewID(), 200)
	path := resultPath210(plans210, initial)
	first := func() planBody {
		return b.call("GET", path+"/history?sourceVersion="+initial["sourceVersion"].(string), nil, "", 200)["items"].([]any)[0].(planBody)["after"].(planBody)
	}
	before := first()
	member := before["scope"].(planBody)["members"].([]any)[0].(planBody)
	if member["name"] != "资料 P5-ORIGIN" || member["sourceVersion"] != grade["headSourceVersion"] {
		t.Fatal("initial scope source differs", member)
	}
	for _, rows := range [][]planBody{{bonus}, {basic, {"id": bonus["id"], "currency": "USD"}}} {
		refused := b.call("POST", path, command210("CORRECT", initial, "", planBody{"components": rows}), foundation.NewID(), 422)
		if !strings.HasPrefix(refused["field"].(string), "components.") {
			t.Fatal(refused)
		}
	}
	h.correct("grade", grade, planBody{"name": "更名后专业五级"}, nil, 200)
	grade = h.detail("grade", id, "")
	family := h.create("jobfamily", "PRO-ORIGIN", "")
	refined := planBody{"mode": "SELECTED", "choices": []planBody{{"dimension": "grade", "id": id}, {"dimension": "jobFamily", "id": family["id"]}}}
	changed := b.call("POST", path, command210("ADD", initial, "2027-02-01", planBody{"scope": refined, "description": "将适用对象限于专业系列"}), foundation.NewID(), 200)
	day := b.call("GET", path+"?businessAt=2027-02-01", nil, "", 200)["record"].(planBody)
	var adoptedName any
	for _, m := range day["scope"].(planBody)["members"].([]any) {
		if m.(planBody)["dimension"] == "grade" {
			adoptedName = m.(planBody)["name"]
		}
	}
	if adoptedName != "更名后专业五级" || !reflect.DeepEqual(before, first()) {
		t.Fatal("corrected source or original lost")
	}
	h.add("grade", grade, "DEACTIVATE", "2027-03-01", nil, 200)
	stopped := h.detail("grade", id, "")["timeline"].([]any)[1].(planBody)
	h.add("grade", h.detail("grade", id, stopped["recordId"].(string)), "REACTIVATE", "2027-04-01", nil, 200)
	rejected := b.call("POST", path, command210("ADD", changed, "2027-02-15", planBody{"description": "涉及期间缺口"}), foundation.NewID(), 422)
	if !strings.Contains(rejected["message"].(string), "2027-03-01") || !reflect.DeepEqual(before, first()) {
		t.Fatal("missing March gap or changed prior adoption", rejected)
	}
	if h.f.scalar("SELECT count(*) FROM compensation.plan_version") != 2 {
		t.Fatal("failed changes wrote a plan revision")
	}
}

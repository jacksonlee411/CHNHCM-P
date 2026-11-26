//go:build integration

package bootstrap_test

import (
	f "chnhcm.local/chnhcm/internal/foundation"
	"slices"
	"testing"
)

func TestEarlierAssignmentEditRetainsRealLaterTransfer(t *testing.T) {
	type M = map[string]any
	fixture := setupAssignment(t)
	b := fixture.h.b
	id := fixture.hire["assignment"].(string)
	base := "/api/v1/assignments/" + id
	state := func(day string) M { return b.call("GET", base+"?businessAt="+day, nil, "", 200) }
	current := func() any { return state("2027-04-01")["sourceVersion"] }
	loc := func(day string, value string, insert bool) M {
		return b.call("POST", base+"/location-changes", M{"expectedSourceVersion": current(), "businessDate": day, "location": value, "insert": insert}, f.NewID(), 200)
	}
	targetDept := departmentCreateBody("D44", fixture.bu)
	targetDept["name"] = "七月产品部"
	dep2 := b.call("POST", "/api/v1/departments", targetDept, f.NewID(), 201)["departmentId"].(string)
	policy := b.call("GET", "/api/v1/assignment-policies", nil, "", 200)
	b.call("POST", "/api/v1/assignment-policies", M{"expectedVersion": policy["version"], "row": M{"id": f.NewID(), "category": "Employee", "purpose": "transfer", "mode": "JobOnly", "from": "2027-04-01"}}, f.NewID(), 200)
	loc("2027-05-01", fixture.locations[1], false)
	transfer := b.call("POST", base+"/transfers", M{"expectedSourceVersion": current(), "businessDate": "2027-07-01", "department": dep2, "job": fixture.job, "notice": "产品部七月调令"}, f.NewID(), 200)
	beforeInsert := current().(string)
	june := loc("2027-06-01", fixture.locations[2], true)
	day := state("2027-07-01")["state"].(M)
	if day["location"].(M)["id"] != fixture.locations[2] || day["department"] != "七月产品部" || state("2027-06-30")["state"].(M)["department"] != "研发部门" {
		t.Fatal("独立后继与补录地点", day)
	}
	old := b.call("GET", base+"?sourceVersion="+beforeInsert+"&businessAt=2027-07-01", nil, "", 200)
	if old["state"].(M)["location"].(M)["id"] != fixture.locations[1] {
		t.Fatal("旧来源被改写", old)
	}
	list := b.call("POST", "/api/v1/assignments/query", M{"businessAt": "2027-07-01", "department": dep2, "page": 1, "pageSize": 20}, "", 200)
	if list["total"] != 1.0 || list["rows"].([]any)[0].(M)["id"] != id {
		t.Fatal(list)
	}
	orig := "/api/v1/employment-actions/" + june["action"].(string)
	read := func() M { return b.call("GET", orig, nil, "", 200)["action"].(M) }
	redated := b.call("POST", base+"/actions/"+june["action"].(string)+"/redate", M{"expectedSourceVersion": current(), "expectedRevision": read()["revision"], "businessDate": "2027-06-10"}, f.NewID(), 200)
	if redated["action"] != june["action"] || state("2027-06-09")["state"].(M)["location"].(M)["id"] != fixture.locations[1] {
		t.Fatal("改日没有恢复原期间", redated)
	}
	key := f.NewID()
	del := M{"expectedSourceVersion": current(), "expectedRevision": read()["revision"]}
	endpoint := base + "/actions/" + june["action"].(string) + "/delete"
	deleted := b.call("POST", endpoint, del, key, 200)
	if deleted["action"] != june["action"] || read()["operation"] != "Delete" || state("2027-07-01")["state"].(M)["location"].(M)["id"] != fixture.locations[1] {
		t.Fatal("删除后的日状态及关联", deleted)
	}
	if again := b.call("POST", endpoint, del, key, 200); again["revision"] != deleted["revision"] {
		t.Fatal("删除重送", again)
	}
	b.call("GET", orig+"?adoptedRef="+june["revision"].(string), nil, "", 200)
	next := loc("2027-06-02", fixture.locations[3], true)
	transferOriginal := "/api/v1/employment-actions/" + transfer["action"].(string)
	transferRevision := b.call("GET", transferOriginal, nil, "", 200)["action"].(M)["revision"]
	b.call("POST", base+"/transfers/"+transfer["action"].(string)+"/correct", M{"expectedSourceVersion": current(), "expectedRevision": transferRevision, "businessDate": "2027-07-01", "department": dep2, "job": fixture.job,
		"notice": "七月调令要求先到指定工作地点履职", "requires": M{"action": next["action"], "revision": next["revision"], "location": fixture.locations[3]}}, f.NewID(), 200)
	prior := current()
	b.call("POST", base+"/actions/"+next["action"].(string)+"/delete", M{"expectedSourceVersion": prior, "expectedRevision": next["revision"]}, f.NewID(), 409)
	if current() != prior || state("2027-07-01")["state"].(M)["location"].(M)["id"] != fixture.locations[3] {
		t.Fatal("缺必要依赖的删除没有回滚")
	}
	options := rows400(t, b, base+"/transfer-options", M{"kind": "department", "businessDate": "2027-06-01", "search": "七月产品部", "page": 1})
	if len(options) != 1 || options[0]["id"] != dep2 {
		t.Fatal("调入候选", options)
	}
	reg := func(q string) M { return b.call("GET", "/api/v1/employment-actions?"+q, nil, "", 200) }
	if v := reg("kind=DataChange&status=deleted"); v["total"] != 1.0 || v["rows"].([]any)[0].(M)["id"] != june["action"] {
		t.Fatal(v)
	}
	for _, side := range []struct {
		department, visible, invisible, other string
		atJuly                                float64
	}{
		{fixture.dep, "fromDepartment", "toDepartment", dep2, 0}, {dep2, "toDepartment", "fromDepartment", fixture.dep, 1},
	} {
		fixture.h.policy("UPDATE access.employment_grant SET department_id=$2 WHERE principal_id=$1 AND action IN ('read','history','register.read','transfer')", b.subject, side.department)
		listed := reg("kind=Transfer")
		if listed["total"] != 1.0 {
			t.Fatal("单侧办理计数", listed)
		}
		row := listed["rows"].([]any)[0].(M)
		if row[side.visible] != side.department || row[side.invisible] != nil || len(listed[side.invisible+"s"].([]any)) != 0 || reg("kind=Transfer&" + side.invisible + "=" + side.other)["total"] != 0.0 {
			t.Fatal("单侧部门或筛选泄漏", listed)
		}
		original := b.call("GET", transferOriginal, nil, "", 200)
		if original["personNumber"] == nil || original["personName"] == nil {
			t.Fatal("单侧原件缺少独立获权身份", original)
		}
		if side.atJuly == 0 {
			v := state("2027-04-01")
			if slices.Contains(v["effectiveDates"].([]any), any("2027-07-01")) {
				t.Fatal("日期节点泄露调入部门", v["effectiveDates"])
			}
		}
		evidence := original["transfer"].(M)
		if side.atJuly == 0 && evidence["after"].(M)["department"] != nil || side.atJuly == 1 && evidence["beforeDepartment"] != nil {
			t.Fatal("原件单侧裁剪", evidence)
		}
		personView := b.call("GET", "/api/v1/persons/"+fixture.person+"/employment?businessAt=2027-07-01", nil, "", 200)
		if personView["assignments"] != side.atJuly || personView["people"] != side.atJuly {
			t.Fatal("调动后的可见任职人数", personView)
		}
	}
	fixture.h.policy("UPDATE access.employment_grant SET department_id=NULL WHERE principal_id=$1 AND action='read'", b.subject)
	fixture.h.policy("UPDATE access.employment_grant SET department_id=$2 WHERE principal_id=$1 AND action='history'", b.subject, fixture.dep)
	oldScope := b.call("GET", base+"?businessAt=2027-04-01&sourceVersion="+prior.(string), nil, "", 200)
	if slices.Contains(oldScope["effectiveDates"].([]any), any("2027-07-01")) {
		t.Fatal("旧来源日期必须按历史授权裁剪")
	}
	b.call("GET", base+"?businessAt=2027-07-01&sourceVersion="+prior.(string), nil, "", 403)
	fixture.h.policy("DELETE FROM access.employment_grant WHERE principal_id=$1 AND action='history'", b.subject)
	b.call("GET", transferOriginal+"?adoptedRef="+transfer["revision"].(string), nil, "", 403)

}

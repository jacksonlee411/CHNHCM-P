//go:build integration

package bootstrap_test

import (
	f "chnhcm.local/chnhcm/internal/foundation"
	"fmt"
	"testing"
)

func TestPersonnelDirectorySelectionMembership(t *testing.T) {
	type M = map[string]any
	x := setupAssignment(t)
	b := x.h.b
	people := []string{}
	for i := 0; i < 22; i++ {
		p := b.call("POST", "/api/v1/persons", M{"submissionId": f.NewID(), "name": fmt.Sprintf("名录选择样本%02d", i)}, "", 200)
		people = append(people, p["id"].(string))
	}
	ref := b.call("POST", "/api/v1/persons/queries", M{"keyword": "名录选择样本"}, "", 200)["ref"]
	read := func(page int, sort string, all bool, ids []string, version string) M {
		return b.call("POST", "/api/v1/personnel/directory/query", M{"ref": ref, "page": page, "pageSize": 20, "sort": sort, "selection": M{"all": all, "ids": ids, "version": version}}, "", 200)
	}
	expire := func(id string) {
		x.h.f.exec("UPDATE personnel.person SET ordinary_until=clock_timestamp()-interval '1 second' WHERE person_id=$1", id)
	}
	expire(people[21])
	first := read(1, "number", true, []string{}, "")
	version := first["selection"].(M)["version"].(string)
	if first["total"] != 21.0 || first["selection"].(M)["count"] != 21.0 {
		t.Fatal("全选跨页数量错误")
	}
	for _, test := range []struct {
		all   bool
		count float64
	}{{true, 20}, {false, 1}} {
		result := read(2, "-name", test.all, []string{people[0]}, version)
		selected := result["selection"].(M)
		if selected["version"] != version || selected["changed"] != false || selected["count"] != test.count {
			t.Fatal("翻页和排序后选择错误", selected)
		}
	}
	expire(people[0])
	x.h.f.exec("UPDATE personnel.person SET ordinary_until=clock_timestamp()+interval '1 month' WHERE person_id=$1", people[21])
	swapped := read(1, "number", true, []string{}, version)
	if swapped["total"] != 21.0 || swapped["selection"].(M)["changed"] != true || swapped["selection"].(M)["count"] != 0.0 {
		t.Fatal("同数换人仍沿用旧选择")
	}
	invalid := read(1, "number", false, []string{people[0]}, "")
	if invalid["selection"].(M)["changed"] != true || invalid["selection"].(M)["count"] != 0.0 {
		t.Fatal("失权人员仍可勾选")
	}
	// Only the leaf has an assignment. The parent path must be both reachable
	// and authorized, including for the nearest future term.
	parents := []string{}
	for i := range 5 {
		body := departmentCreateBody(fmt.Sprintf("P405D%d", i), x.bu)
		if i > 0 {
			body["parentDepartmentId"] = parents[i-1]
		}
		id := b.call("POST", "/api/v1/departments", body, f.NewID(), 201)["departmentId"].(string)
		parents = append(parents, id)
	}
	cfg := b.call("GET", "/api/v1/assignment-policies", nil, "", 200)
	b.call("POST", "/api/v1/assignment-policies", M{"expectedVersion": cfg["version"], "row": M{"id": f.NewID(), "category": "Employee", "purpose": "secondary", "mode": "JobOnly", "from": "2027-04-01"}}, f.NewID(), 200)
	path := "/api/v1/assignments/" + x.hire["assignment"].(string)
	versionAt := b.call("GET", path+"?businessAt=2027-04-03", nil, "", 200)["sourceVersion"]
	b.call("POST", path+"/additional-assignments", M{"expectedSourceVersion": versionAt, "from": "2027-04-03", "job": x.job, "department": parents[4], "notice": "五级树测试", "parallel": false}, f.NewID(), 200)
	scope := func(direct bool) M {
		return b.call("POST", "/api/v1/personnel/directory/query", M{"page": 1, "pageSize": 20, "department": parents[0], "directOnly": direct}, "", 200)
	}
	joined := scope(false)
	if joined["total"] != 1.0 || joined["assignments"] != 1.0 || scope(true)["total"] != 0.0 {
		t.Fatal("祖先和直属部门范围不符", joined)
	}
	links := map[string]any{}
	for _, node := range joined["departments"].([]any) {
		item := node.(M)
		links[item["id"].(string)] = item["parent"]
	}
	for i := 1; i < len(parents); i++ {
		if links[parents[i]] != parents[i-1] {
			t.Fatal("路径缺失", i, links)
		}
	}
	x.h.f.exec("UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'parentDepartmentId') WHERE principal_id=$1 AND action='department.history.read'", b.subject)
	limited := scope(false)
	if limited["total"] != 0.0 {
		t.Fatal("缺少上级关系权限时仍能从祖先命中")
	}
	for _, node := range limited["departments"].([]any) {
		if node.(M)["id"] == parents[0] {
			t.Fatal("返回了无权推导的祖先")
		}
	}

}

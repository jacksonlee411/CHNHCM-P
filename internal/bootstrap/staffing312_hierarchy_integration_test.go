//go:build integration

package bootstrap_test

import (
	"context"
	"strings"
	"testing"

	"chnhcm.local/chnhcm/internal/bootstrap"
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	app "chnhcm.local/chnhcm/internal/contexts/staffing/application"
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	staff "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

type hierarchyStore312 struct {
	app.Store
	ready, release chan struct{}
}
type hierarchyUnit312 struct {
	app.Unit
	gate hierarchyStore312
}

func (gate hierarchyStore312) Within(ctx context.Context, c foundation.Credentials, write bool, fn func(app.Unit) error) error {
	return gate.Store.Within(ctx, c, write, func(unit app.Unit) error { return fn(hierarchyUnit312{unit, gate}) })
}
func (unit hierarchyUnit312) SaveHierarchy(ctx context.Context, enterprise string, result staff.Result, number int, edges d.HierarchyEdges, actor access.OperatorSnapshot) error {
	close(unit.gate.ready)
	select {
	case <-unit.gate.release:
		return unit.Unit.SaveHierarchy(ctx, enterprise, result, number, edges, actor)
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestStaffing312Hierarchy(t *testing.T) {
	h := staffingSetup307(t)
	secondPassword := foundation.NewID()
	must(t, bootstrap.Initialize(h.f.ctx, h.f.admin, bootstrap.InitInput{DevelopmentFixture: true, Instance: h.f.instance, EnterpriseCode: "LOCAL", EnterpriseName: "本地验证企业", Start: "1900-01-01", Login: "hierarchy-admin", Password: secondPassword}))
	secondAdmin := h.f.login("hierarchy-admin", secondPassword)
	if secondAdmin.subject == h.b.subject {
		t.Fatal("并发参与者应有不同的管理员主体")
	}
	left := geoCredentials203(t, h.f, "admin", h.f.password)
	right := geoCredentials203(t, h.f, "hierarchy-admin", secondPassword)
	reader := h.f.login("reader", h.f.readPassword)
	must(t, h.f.runtime.InitializeCountryCatalog(h.f.ctx, access.LoginInput{Login: "admin", Password: h.f.password}))
	locations := location204{t, h.f, h.b, map[string]string{"MO": geoCountry203(t, h.f, "MO")}}
	loc := locations.create("H312", "MO", "", "2026-01-01")["id"]
	bu := h.b.call("POST", "/api/v1/business-units", createBody("H312", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	dep := h.b.call("POST", "/api/v1/departments", departmentCreateBody("H312", bu), foundation.NewID(), 201)["departmentId"]
	family := h.create("jobfamily", "H312", "")
	job := h.create("job", "H312", family["id"].(string))
	pos := func(code string) string {
		return h.b.call("POST", "/api/v1/positions", map[string]any{"code": code, "name": code, "departmentId": dep, "jobId": job["id"], "locationId": loc, "effectiveDate": "2026-01-01", "operationType": "CREATE", "operationReason": "NEW_POSITION"}, foundation.NewID(), 201)["objectId"].(string)
	}
	child, first, second, other := pos("A"), pos("B"), pos("C"), pos("D")
	const base = "/api/v1/position-hierarchies/"
	const jan = "2027-01-01"
	const apr = "2027-04-01"
	body := func(expected, date, child string, parent any) map[string]any {
		return map[string]any{"expectedVersion": expected, "effectiveDate": date, "childId": child, "parentId": parent}
	}
	publish := func(b map[string]any, status int) map[string]any {
		return h.b.call("POST", base+"reporting/publish", b, foundation.NewID(), status)
	}
	one := body("", jan, child, first)
	reader.call("POST", base+"reporting/publish", one, foundation.NewID(), 403)
	h.b.call("POST", base+"reporting/publish", map[string]any{"expectedVersion": "", "effectiveDate": jan, "childId": child}, foundation.NewID(), 400)
	key := foundation.NewID()
	saved := h.b.call("POST", base+"reporting/publish", one, key, 200)
	v1 := saved["sourceVersion"].(string)
	if h.b.call("POST", base+"reporting/publish", one, key, 200)["replayed"] != true {
		t.Fatal("原提交未复用")
	}
	resolve := base + "submissions/" + key + "/resolve"
	unknown := base + "submissions/" + foundation.NewID() + "/resolve"
	receipts := h.f.scalar("SELECT count(*) FROM staffing.position_hierarchy_terminal")
	h.b.call("POST", unknown+"?purpose=functional", nil, "", 400)
	h.b.call("POST", unknown, map[string]any{"purpose": "functional"}, "", 400)
	if h.f.scalar("SELECT count(*) FROM staffing.position_hierarchy_terminal") != receipts {
		t.Fatal("无效查回请求产生了终态回执")
	}
	if got := h.b.call("POST", resolve, nil, "", 200); got["sourceVersion"] != v1 || got["outcome"] != "accepted" {
		t.Fatal("未查回原发布版本", got)
	}
	if got := secondAdmin.call("POST", resolve, nil, "", 200); got["outcome"] != "not_applied" || got["sourceVersion"] != nil {
		t.Fatal("另一主体可以查回原发布", got)
	}
	h.b.call("POST", base+"reporting/publish", body("", jan, child, second), key, 409)
	if got := publish(body(v1, jan, child, first), 200); got["outcome"] != "no_change" || got["sourceVersion"] != v1 {
		t.Fatal("未更改关系却创建了新版本", got)
	}
	v2 := publish(body(v1, apr, child, second), 200)["sourceVersion"].(string)
	parentAt := func(version, date, id string) any {
		url := base + "reporting?businessAt=" + date
		if version != "" {
			url += "&sourceVersion=" + version
		}
		view := reader.call("GET", url, nil, "", 200)
		if view["canPublish"] != false {
			t.Fatal("只读账号有发布能力")
		}
		for _, node := range view["nodes"].([]any) {
			v := node.(map[string]any)
			if v["id"] == id {
				return v["parentId"]
			}
		}
		return nil
	}
	if parentAt("", "2027-03-31", child) != first || parentAt("", apr, child) != second || parentAt(v1, apr, child) != first {
		t.Fatal("旧版或未来边界错误")
	}
	if !strings.Contains(publish(body(v2, jan, second, child), 409)["message"].(string), apr) {
		t.Fatal("未查出未来闭环")
	}
	fail := h.add("position", h.detail("position", second, ""), "DEACTIVATE", "2027-05-01", nil, 409)
	if !strings.Contains(fail["message"].(string), "2027-05-01") {
		t.Fatal(fail)
	}
	v3 := publish(body(v2, "2027-05-01", child, nil), 200)["sourceVersion"].(string)
	h.add("position", h.detail("position", second, ""), "DEACTIVATE", "2027-05-01", nil, 200)
	fail = publish(body(v3, "2027-05-01", child, second), 409)
	if !strings.Contains(fail["message"].(string), "2027-05-01") || parentAt(v2, "2027-05-01", child) != second {
		t.Fatal("停用或旧版不正确", fail)
	}
	for _, route := range []struct{ method, path string }{{"GET", "functional"}, {"GET", "functional/structure?mode=versions"}, {"POST", "functional/publish"}} {
		status, _, _ := h.b.response(route.method, base+route.path, nil, "")
		if status != 404 {
			t.Fatalf("obsolete route %s %s returned %d", route.method, route.path, status)
		}
	}
	h.b.call("GET", base+"reporting?purpose=functional", nil, "", 400)
	h.b.call("POST", base+"reporting/publish?purpose=functional", body(v3, jan, child, first), foundation.NewID(), 400)
	original := reader.call("GET", base+"reporting?businessAt="+apr+"&sourceVersion="+v1, nil, "", 200)
	if _, present := original["purpose"]; present || h.f.scalar("SELECT count(*) FROM staffing.position_hierarchy_outbox WHERE event ? 'purpose'") != 0 {
		t.Fatal("管理汇报读取或新发布事件仍包含用途字段")
	}
	if len(original["nodes"].([]any)) != 4 {
		t.Fatal("原发布快照缺少未挂接的岗位")
	}
	later := pos("LATER")
	h.correct("position", h.detail("position", other, ""), map[string]any{"name": "新名称"}, nil, 200)
	frozen := reader.call("GET", base+"reporting?businessAt="+apr+"&sourceVersion="+v1, nil, "", 200)
	for _, item := range frozen["nodes"].([]any) {
		node := item.(map[string]any)
		if node["id"] == later || node["id"] == other && node["name"] != "D" {
			t.Fatal("旧版混入后来岗位或名称", node)
		}
	}
	for _, statement := range []string{
		"UPDATE staffing.position_hierarchy_version SET version_no=99 WHERE source_id=$1",
		"DELETE FROM staffing.position_hierarchy_node WHERE source_id=$1",
		"DELETE FROM staffing.position_hierarchy_edge WHERE source_id=$1",
		"INSERT INTO staffing.position_hierarchy_edge SELECT instance_id,hierarchy_id,source_id,parent_id,child_id,period FROM staffing.position_hierarchy_edge WHERE source_id=$1",
	} {
		if _, err := h.f.admin.Exec(h.f.ctx, statement, v1); err == nil {
			t.Fatal("可篡改已发布结构", statement)
		}
	}
	h.f.exec("DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='positionhierarchy.history.read'", reader.subject)
	reader.call("GET", base+"reporting?businessAt="+apr+"&sourceVersion="+v1, nil, "", 403)
	raceChild, raceParent := pos("RACE-A"), pos("RACE-B")
	for _, reverse := range []bool{false, true} {
		sub, parent := raceChild, raceParent
		if reverse {
			sub, parent = parent, sub
		}
		expected := ""
		view := h.b.call("GET", base+"reporting", nil, "", 200)
		expected = view["sourceVersion"].(string)
		if reverse {
			expected = publish(body(expected, "2027-02-01", raceChild, nil), 200)["sourceVersion"].(string)
		} else {
			expected = publish(body(expected, jan, other, first), 200)["sourceVersion"].(string)
		}
		service := h.f.runtime.Staffing.(app.Service)
		gate := hierarchyStore312{service.Store, make(chan struct{}), make(chan struct{})}
		type answer struct {
			res staff.Result
			err error
		}
		win, lose := make(chan answer, 1), make(chan answer, 1)
		command := func(svc app.Service, c foundation.Credentials, child, parent string, target chan answer) {
			res, err := svc.PublishHierarchy(h.f.ctx, c, foundation.NewID(), make([]byte, 32), staff.PublishHierarchy{ExpectedVersion: expected, EffectiveDate: "2027-02-01", ChildID: child, ParentID: &parent})
			target <- answer{res, err}
		}
		go command(app.Service{Store: gate}, left, sub, parent, win)
		wait307(t, h.f, gate.ready)
		go command(service, right, parent, sub, lose)
		blocked307(t, h.f, gate.release)
		close(gate.release)
		a, b := <-win, <-lose
		must(t, a.err)
		must(t, b.err)
		if a.res.Outcome != "accepted" || b.res.Failure == nil || b.res.Failure.Code != "M03_VERSION_CONFLICT" {
			t.Fatal(a, b)
		}
		fail = secondAdmin.call("POST", base+"reporting/publish", body(a.res.SourceVersion, "2027-02-01", parent, sub), foundation.NewID(), 409)
		if !strings.Contains(fail["message"].(string), "成环") {
			t.Fatal(fail)
		}
	}
	if h.f.scalar("SELECT count(*) FROM staffing.position_hierarchy_version") != 7 || h.f.scalar("SELECT count(*) FROM staffing.position_hierarchy_outbox") != 7 {
		t.Fatal("半笔发布或重复事件")
	}
	for _, order := range []string{"publish-first", "stop-first"} {
		t.Run(order, func(t *testing.T) {
			target := pos(order + "-P")
			leaf := pos(order + "-C")
			detail := h.detail("position", target, "")
			latest := h.b.call("GET", base+"reporting", nil, "", 200)
			stop := staff.Command{Kind: staff.Position, ObjectID: target, SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Input: staff.Input{Action: "DEACTIVATE", ExpectedSourceVersion: detail["headSourceVersion"].(string), AnchorRecordID: rec307(detail)["recordId"].(string), EffectiveDate: "2027-05-01", OperationType: "DEACTIVATE", OperationReason: "POSITION_CLOSURE"}}
			link := staff.PublishHierarchy{ExpectedVersion: latest["sourceVersion"].(string), EffectiveDate: apr, ChildID: leaf, ParentID: &target}
			service := h.f.runtime.Staffing.(app.Service)
			ready, release := make(chan struct{}), make(chan struct{})
			type answer struct {
				res staff.Result
				err error
			}
			one, two := make(chan answer, 1), make(chan answer, 1)
			publish := func(svc app.Service, c foundation.Credentials, ch chan answer) {
				res, err := svc.PublishHierarchy(h.f.ctx, c, foundation.NewID(), make([]byte, 32), link)
				ch <- answer{res, err}
			}
			deactivate := func(svc app.Service, c foundation.Credentials, ch chan answer) {
				res, err := svc.Execute(h.f.ctx, c, stop)
				ch <- answer{res, err}
			}
			if order == "publish-first" {
				go publish(app.Service{Store: hierarchyStore312{service.Store, ready, release}}, left, one)
				wait307(t, h.f, ready)
				go deactivate(service, right, two)
			} else {
				go deactivate(app.Service{Store: positionGate312{service.Store, ready, release}}, left, one)
				wait307(t, h.f, ready)
				go publish(service, right, two)
			}
			blocked307(t, h.f, release)
			close(release)
			winner, loser := <-one, <-two
			must(t, winner.err)
			must(t, loser.err)
			if winner.res.Outcome != "accepted" || loser.res.Failure == nil || !strings.Contains(loser.res.Failure.Message, "2027-05-01") {
				t.Fatalf("岗位变更与发布未互斥：%+v / %+v", winner, loser)
			}
		})
	}
}

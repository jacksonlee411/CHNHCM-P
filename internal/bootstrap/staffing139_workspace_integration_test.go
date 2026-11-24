//go:build integration

package bootstrap_test

import (
	app "chnhcm.local/chnhcm/internal/contexts/staffing/application"
	db "chnhcm.local/chnhcm/internal/contexts/staffing/infra/postgres"
	"fmt"
	"testing"
	"time"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

func TestStaffing139DepartmentRange(t *testing.T) {
	h := staffingSetup307(t)
	now := time.Date(2027, 3, 15, 12, 0, 0, 0, foundation.Shanghai)
	store := h.f.runtime.Staffing.(app.Service).Store.(*db.Store)
	store.Clock = func() time.Time { return now }
	must(t, h.f.runtime.InitializeCountryCatalog(h.f.ctx, access.LoginInput{Login: "admin", Password: h.f.password}))
	locations := location204{t, h.f, h.b, map[string]string{"MO": geoCountry203(t, h.f, "MO")}}
	loc := locations.create("L139", "MO", "", "2026-01-01")["id"]
	bu := h.b.call("POST", "/api/v1/business-units", createBody("BU139", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	dep := func(code, parent string) string {
		body := departmentCreateBody(code, bu)
		if parent != "" {
			body["parentDepartmentId"] = parent
		}
		return h.b.call("POST", "/api/v1/departments", body, foundation.NewID(), 201)["departmentId"].(string)
	}
	root := dep("S139", "")
	sub, other := dep("S139A", root), dep("E139", "")
	family := h.create("jobfamily", "F139", "")
	job := h.create("job", "J139", family["id"].(string))
	const base = "/api/v1/positions"
	pos := func(code, dep, date string) string {
		body := map[string]any{"name": code, "code": code, "departmentId": dep, "locationId": loc, "jobId": job["id"], "effectiveDate": date, "operationType": "CREATE", "operationReason": "NEW_POSITION"}
		return h.b.call("POST", base, body, foundation.NewID(), 201)["objectId"].(string)
	}
	first := pos("P139A", root, "2026-01-01")
	future := pos("P139B", sub, "2099-01-01")
	pos("P139C", other, "2026-01-01")
	scope := "?scopeDepartmentId=" + root
	result := h.b.call("GET", base+scope, nil, "", 200)
	if result["totalCount"] != float64(2) {
		t.Fatal("含子部门应有两个岗位", result)
	}
	if row := result["items"].([]any)[1].(map[string]any); row["id"] != future || row["status"] != "not_effective" {
		t.Fatal("未来岗位应沿拟属部门入选", row)
	}
	single := h.b.call("GET", base+scope+"&includeDescendants=false", nil, "", 200)
	if single["totalCount"] != float64(1) || single["items"].([]any)[0].(map[string]any)["id"] != first {
		t.Fatal("仅本部门", single)
	}
	nav := h.b.call("GET", base+"/"+first+"/navigation"+scope, nil, "", 200)
	if nav["nextId"] != future || nav["totalCount"] != float64(2) {
		t.Fatal("范围内前后导航", nav)
	}
	in := map[string]any{"query": map[string]any{"scopeDepartmentId": root}, "draftId": foundation.NewID()}
	selection := h.b.call("POST", base+"/selection/start", in, "", 200)
	in["expectedRevision"] = selection["revision"]
	in["query"] = map[string]any{"scopeDepartmentId": root, "includeDescendants": true}
	in["excludedIds"] = []string{future}
	selection = h.b.call("POST", base+"/selection/resolve", in, "", 200)
	in["expectedRevision"] = selection["revision"]
	pos("P139NEW", sub, "2026-01-01")
	h.correct("position", h.detail("position", first, ""), map[string]any{"departmentId": other}, nil, 200)
	live := h.b.call("GET", base+scope, nil, "", 200)["items"].([]any)
	for _, row := range live {
		if row.(map[string]any)["id"] == first {
			t.Fatal("岗位调出后仍在实时范围中")
		}
	}
	rows := csv307(t, h.b, base+"/export", in, 200)
	if len(rows) != 2 || rows[1][0] != "P139A" {
		t.Fatal("导出必须沿用捕获成员", rows)
	}
	in["query"] = map[string]any{"scopeDepartmentId": root, "includeDescendants": false}
	h.b.call("POST", base+"/selection/resolve", in, "", 409)
	h.b.call("GET", base+"?scopeDepartmentId="+foundation.NewID(), nil, "", 409)
	for _, query := range []string{"?scopeDepartmentId=", "?includeDescendants=false", scope + "&includeDescendants=maybe"} {
		h.b.call("GET", base+query, nil, "", 400)
	}
	h.b.call("GET", "/api/v1/grades"+scope, nil, "", 400)
	t.Run("固定范围与当前授权", func(t *testing.T) {
		in["query"] = map[string]any{"scopeDepartmentId": root}
		var grant, enterprise string
		must(t, h.f.admin.QueryRow(h.f.ctx, "SELECT grant_id::text,scope_id::text FROM access.subject_grant WHERE instance_id=$1 AND principal_id=$2 AND action='department.read' AND scope_kind='enterprise'", h.f.instance, h.b.subject).Scan(&grant, &enterprise))
		h.f.exec("UPDATE access.subject_grant SET scope_kind='department',scope_id=$2 WHERE grant_id=$1", grant, root)
		extra := foundation.NewID()
		h.f.exec("INSERT INTO access.subject_grant SELECT instance_id,$2::uuid,principal_id,'department',$3::uuid,action,read_fields,write_fields FROM access.subject_grant WHERE grant_id=$1", grant, extra, other)
		rows := csv307(t, h.b, base+"/export", in, 200)
		if len(rows) != 2 || rows[1][0] != "P139A" {
			t.Fatal("被排除岗位失权", rows)
		}
		h.f.exec("DELETE FROM access.subject_grant WHERE grant_id=$1", extra)
		csv307(t, h.b, base+"/export", in, 403)
		h.f.exec("UPDATE access.subject_grant SET scope_kind='enterprise',scope_id=$2 WHERE grant_id=$1", grant, enterprise)
		anchor := h.b.call("POST", "/api/v1/departments", departmentCreateBody("TEMP139", bu), foundation.NewID(), 201)
		id := anchor["departmentId"].(string)
		moved := pos("MOVED139", id, "2026-01-01")
		capture := map[string]any{"draftId": foundation.NewID(), "query": map[string]any{"scopeDepartmentId": id}}
		fixed := h.b.call("POST", base+"/selection/start", capture, "", 200)
		capture["expectedRevision"] = fixed["revision"]
		h.correct("position", h.detail("position", moved, ""), map[string]any{"departmentId": other}, nil, 200)
		h.b.call("POST", "/api/v1/departments/"+id+"/deactivate", map[string]any{"expectedSourceVersion": anchor["sourceVersion"], "anchorRecordId": anchor["targetRecordId"], "effectiveDate": "2027-03-01", "operationType": "DEACTIVATE", "operationReason": "ORG_RESTRUCTURING"}, foundation.NewID(), 201)
		h.b.call("GET", base+"?scopeDepartmentId="+id, nil, "", 409)
		rows = csv307(t, h.b, base+"/export", capture, 200)
		if len(rows) != 2 || rows[1][0] != "MOVED139" {
			t.Fatal("锚点停用后导出原成员", rows)
		}
	})
	t.Run("分页结构及旧岗位节点", func(t *testing.T) {
		const hierarchy = "/api/v1/position-hierarchies/reporting"
		parent := pos("H139P", other, "2026-01-01")
		later := pos("H139F", other, "2026-01-01")
		dated := pos("H139A", root, "2026-01-01")
		changed := h.add("position", h.detail("position", dated, ""), "ADD", "2027-04-01", map[string]any{"name": "区域专员"}, 200)
		publish := func(expected, date, superior string) string {
			return h.b.call("POST", hierarchy+"/publish", map[string]any{"expectedVersion": expected, "effectiveDate": date, "childId": dated, "parentId": superior}, foundation.NewID(), 200)["sourceVersion"].(string)
		}
		v1 := publish("", "2027-01-01", parent)
		v2 := publish(v1, "2027-06-01", later)
		h.correct("position", h.detail("position", dated, changed["selectedRecordId"].(string)), map[string]any{"name": "区域客户专员"}, nil, 200)
		v3 := publish(v2, "2027-04-01", first)
		query := hierarchy + "/structure?businessAt=2027-04-01&"
		current := h.b.call("GET", query+"mode=relation&nodeId="+dated, nil, "", 200)
		if current["sourceVersion"] != v3 || current["endInclusive"] != "2027-05-31" || current["nextStart"] != "2027-06-01" || current["parent"].(map[string]any)["id"] != first {
			t.Fatal("后续安排", current)
		}
		old := h.b.call("GET", query+"mode=relation&nodeId="+dated+"&sourceVersion="+v1, nil, "", 200)
		selected := old["selected"].(map[string]any)
		if selected["name"] != "区域专员" || old["parent"].(map[string]any)["id"] != parent || old["canPublish"] != false {
			t.Fatal("只读封存关系", old)
		}
		detail := h.b.call("GET", base+"/"+dated+"?sourceVersion="+selected["positionSource"].(string)+"&selectedRecordId="+selected["recordId"].(string), nil, "", 200)
		if detail["readOnly"] != true || rec307(detail)["name"] != "区域专员" {
			t.Fatal("封存岗位资料", detail)
		}
		path := h.b.call("GET", query+"mode=path&nodeId="+dated, nil, "", 200)["nodes"].([]any)
		if len(path) != 2 || path[0].(map[string]any)["id"] != first || path[1].(map[string]any)["id"] != dated {
			t.Fatal("定位路径", path)
		}
		for _, mode := range []string{"nodes", "search&q=139", "versions"} {
			page := h.b.call("GET", query+"pageSize=1&mode="+mode, nil, "", 200)
			key := "nodes"
			if mode == "versions" {
				key = "versions"
			}
			if len(page[key].([]any)) != 1 || page["more"] != true {
				t.Fatal("分支、搜索及历史须有界", mode, page)
			}
		}
		reader := h.f.login("reader", h.f.readPassword)
		reader.call("GET", query+"mode=relation&nodeId="+dated+"&sourceVersion="+v1, nil, "", 200)
		h.f.exec("DELETE FROM access.subject_grant WHERE instance_id=$1 AND principal_id=$2 AND action='position.history.read'", h.f.instance, reader.subject)
		reader.call("GET", query+"mode=node&nodeId="+dated+"&sourceVersion="+v1, nil, "", 403)
		if allowed := reader.call("GET", hierarchy+"/structure?mode=nodes", nil, "", 200)["canReadHistory"]; allowed != false {
			t.Fatal("目录泄漏岗位历史", allowed)
		}
		reader.call("GET", hierarchy+"/structure?mode=versions", nil, "", 403)
		reader.call("GET", base+"/"+dated, nil, "", 200)
		noPositions := h.b.call("GET", hierarchy+"/structure?mode=nodes&businessAt=1900-01-01", nil, "", 200)
		if len(noPositions["nodes"].([]any)) != 0 {
			t.Fatal("空日期", noPositions)
		}
		cleared := h.b.call("POST", hierarchy+"/publish", map[string]any{"childId": dated, "parentId": nil, "effectiveDate": "2027-04-01", "expectedVersion": v3}, foundation.NewID(), 200)
		none := h.b.call("GET", query+"mode=relation&nodeId="+dated, nil, "", 200)
		if none["parent"] != nil || none["selected"] == nil || none["nextStart"] != "2027-06-01" {
			t.Fatal("解除上级或后继安排", none)
		}
		noChange := h.b.call("POST", hierarchy+"/publish", map[string]any{"childId": dated, "parentId": nil, "effectiveDate": "2027-04-01", "expectedVersion": cleared["sourceVersion"]}, foundation.NewID(), 200)
		if noChange["outcome"] != "no_change" {
			t.Fatal("同值应不新增版本", noChange)
		}
		h.b.call("GET", hierarchy+"/structure?mode=nodes&sourceVersion="+foundation.NewID(), nil, "", 404)
	})
	t.Run("停用岗及未来部门", func(t *testing.T) {
		stopped := pos("INACTIVE139", sub, "2026-01-01")
		h.add("position", h.detail("position", stopped, ""), "DEACTIVATE", "2027-03-01", nil, 200)
		inactive := h.b.call("GET", base+scope+"&status=inactive", nil, "", 200)
		if inactive["totalCount"] != float64(1) || inactive["items"].([]any)[0].(map[string]any)["id"] != stopped {
			t.Fatal(inactive)
		}
		body := departmentCreateBody("FUTURE139", bu)
		body["effectiveDate"] = "2027-05-01"
		body["parentDepartmentId"] = root
		delayed := h.b.call("POST", "/api/v1/departments", body, foundation.NewID(), 201)["departmentId"].(string)
		later := pos("FUTURE139", delayed, "2027-05-01")
		scoped := h.b.call("GET", base+scope+"&q=FUTURE139", nil, "", 200)
		all := h.b.call("GET", base+"?q=FUTURE139", nil, "", 200)
		if scoped["totalCount"] != float64(0) || all["items"].([]any)[0].(map[string]any)["id"] != later {
			t.Fatal(scoped, all)
		}
	})
	t.Run("深层、宽分支和末页路径", func(t *testing.T) {
		const endpoint = "/api/v1/position-hierarchies/reporting"
		chain := []string{pos("TREE139ROOT", other, "2026-01-01")}
		head := h.b.call("GET", endpoint, nil, "", 200)["sourceVersion"].(string)
		addChild := func(code, parent string) string {
			id := pos(code, other, "2026-01-01")
			head = h.b.call("POST", endpoint+"/publish", map[string]any{"childId": id, "parentId": parent, "effectiveDate": "2026-01-01", "expectedVersion": head}, foundation.NewID(), 200)["sourceVersion"].(string)
			return id
		}
		for i := 1; i <= 5; i++ {
			chain = append(chain, addChild(fmt.Sprintf("CHAIN139%d", i), chain[i-1]))
		}
		tail := ""
		for i := 1; i <= 34; i++ {
			tail = addChild(fmt.Sprintf("WIDE139%02d", i), chain[5])
		}
		query := endpoint + "/structure?businessAt=2027-04-01&mode="
		path := h.b.call("GET", query+"path&nodeId="+tail, nil, "", 200)["nodes"].([]any)
		if len(path) != 7 || path[6].(map[string]any)["siblingPage"] != float64(2) {
			t.Fatal("应定位第七层第二页", path)
		}
		for i, id := range chain {
			if path[i].(map[string]any)["id"] != id {
				t.Fatal("错误的祖先", path)
			}
		}
		page := h.b.call("GET", query+"nodes&nodeId="+chain[5]+"&page=2", nil, "", 200)
		items := page["nodes"].([]any)
		if len(items) != 4 || page["more"] != false || items[3].(map[string]any)["id"] != tail {
			t.Fatal("末页精确窗口", page)
		}
		search := h.b.call("GET", query+"search&q=WIDE13934", nil, "", 200)
		if len(search["nodes"].([]any)) != 1 || search["nodes"].([]any)[0].(map[string]any)["id"] != tail {
			t.Fatal("查找末页目标", search)
		}
	})
	in["query"] = map[string]any{"scopeDepartmentId": root}
	now = now.AddDate(0, 0, 1)
	csv307(t, h.b, base+"/export", in, 409)
	now = now.AddDate(0, 0, -1)
	h.f.exec("UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'departmentId') WHERE instance_id=$1 AND principal_id=$2 AND action='position.read'", h.f.instance, h.b.subject)
	h.b.call("GET", base+scope, nil, "", 403)
	h.b.call("GET", base+"/"+first+"/navigation"+scope, nil, "", 403)
	csv307(t, h.b, base+"/export", in, 403)
	h.f.exec("UPDATE access.subject_grant SET read_fields=array_append(read_fields,'departmentId') WHERE instance_id=$1 AND principal_id=$2 AND action='position.read'", h.f.instance, h.b.subject)
	h.f.exec("DELETE FROM access.subject_grant WHERE instance_id=$1 AND principal_id=$2 AND action='department.read'", h.f.instance, h.b.subject)
	h.b.call("GET", base+scope, nil, "", 403)
	h.b.call("GET", base+"/"+first+"/navigation"+scope, nil, "", 403)
	in["query"] = map[string]any{"scopeDepartmentId": root}
	csv307(t, h.b, base+"/export", in, 403)
	if got := h.b.call("GET", base+"?q=P139", nil, "", 200)["totalCount"]; got != float64(4) {
		t.Fatal("普通岗位权限应独立", got)
	}
}

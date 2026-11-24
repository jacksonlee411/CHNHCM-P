//go:build integration

package bootstrap_test

import (
	"chnhcm.local/chnhcm/internal/bootstrap"
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"context"
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"chnhcm.local/chnhcm/internal/foundation"
)

// Narrow-fact prototype plus real public endpoint pagination and scope checks.
// Independent expected identities come from the normal creation receipts.
func TestDepartmentTree300LifecycleAndCrossBU(t *testing.T) {
	f := fresh(t)
	day, _ := time.Parse(time.RFC3339, "2026-06-30T15:59:59Z")
	restore := bootstrap.SetIntegrationClock(f.runtime, func() time.Time { return day })
	defer restore()
	b := f.login("admin", f.password)
	buA := b.call("POST", "/api/v1/business-units", createBody("TREE_STATE_A", "2020-01-01"), foundation.NewID(), 201)["buId"].(string)
	buB := b.call("POST", "/api/v1/business-units", createBody("TREE_STATE_B", "2020-01-01"), foundation.NewID(), 201)["buId"].(string)
	create := func(code, bu, parent, date string) map[string]any {
		body := departmentCreateBody(code, bu)
		body["effectiveDate"] = date
		if parent != "" {
			body["parentDepartmentId"] = parent
		}
		return b.call("POST", "/api/v1/departments", body, foundation.NewID(), 201)
	}
	a := create("STATE_A", buA, "", "2020-01-01")["departmentId"].(string)
	other := create("STATE_B", buB, "", "2020-01-01")["departmentId"].(string)
	invalid := departmentCreateBody("STATE_CROSS", buB)
	invalid["effectiveDate"] = "2020-01-01"
	invalid["parentDepartmentId"] = a
	b.call("POST", "/api/v1/departments", invalid, foundation.NewID(), 409)
	leaf := create("STATE_C", buA, a, "2020-01-01")
	id := leaf["departmentId"].(string)
	create("STATE_F", buA, a, "2099-01-01")
	base := "/api/v1/departments/" + id
	stop := b.call("POST", base+"/deactivate", map[string]any{"expectedSourceVersion": leaf["sourceVersion"], "anchorRecordId": leaf["targetRecordId"], "effectiveDate": "2026-06-01", "operationType": "DEACTIVATE", "operationReason": "ORG_RESTRUCTURING"}, foundation.NewID(), 201)
	b.call("POST", base+"/reactivate", map[string]any{"expectedSourceVersion": stop["sourceVersion"], "anchorRecordId": stop["targetRecordId"], "effectiveDate": "2026-07-01", "operationType": "REACTIVATE", "operationReason": "BUSINESS_RESUMPTION"}, foundation.NewID(), 201)
	before := f.scalar("SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.department_history")
	roots := b.call("GET", "/api/v1/departments/tree/nodes", nil, "", 200)["items"].([]any)
	if len(roots) != 2 || roots[0].(map[string]any)["id"] != a || roots[1].(map[string]any)["id"] != other || roots[0].(map[string]any)["hasChildren"] != false {
		t.Fatal("inactive/future relation leaked into roots", roots)
	}
	if len(b.call("GET", "/api/v1/departments/tree/nodes?parentId="+a, nil, "", 200)["items"].([]any)) != 0 {
		t.Fatal("inactive child shown")
	}
	b.call("GET", base+"/tree-location", nil, "", 409)
	b.call("GET", "/api/v1/departments?scopeDepartmentId="+id, nil, "", 409)
	all := b.call("GET", "/api/v1/departments", nil, "", 200)
	if all["count"] != float64(4) {
		t.Fatal("all omitted inactive or future", all)
	}
	if b.call("GET", "/api/v1/departments?scopeDepartmentId="+a+"&status=inactive", nil, "", 200)["count"] != float64(0) {
		t.Fatal("inactive filter escaped current branch")
	}
	day = day.Add(time.Second)
	chain := b.call("GET", base+"/tree-location", nil, "", 200)["items"].([]any)
	if len(chain) != 2 || chain[0].(map[string]any)["id"] != a || chain[1].(map[string]any)["id"] != id {
		t.Fatal("reactivation parent lost", chain)
	}
	query := url.Values{"scopeDepartmentId": {a}, "filterField": {"managementBuId"}, "filterOperator": {"equals"}, "filterValue": {buB}}
	filtered := b.call("GET", "/api/v1/departments?"+query.Encode(), nil, "", 200)
	if filtered["count"] != float64(0) {
		t.Fatal("BU filter replaced administrative scope", filtered)
	}
	query.Set("scopeDepartmentId", other)
	filtered = b.call("GET", "/api/v1/departments?"+query.Encode(), nil, "", 200)
	if filtered["count"] != float64(1) || filtered["items"].([]any)[0].(map[string]any)["id"] != other {
		t.Fatal("separate BU root lost", filtered)
	}
	if f.scalar("SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.department_history") != before {
		t.Fatal("tree reads changed source history")
	}
}

func TestDepartmentTree300NarrowReadPrototype(t *testing.T) {
	f := fresh(t)
	admin := f.login("admin", f.password)
	bu := admin.call("POST", "/api/v1/business-units", createBody("TREE300_BU", "2020-01-01"), foundation.NewID(), 201)["buId"].(string)
	create := func(code, parent, date string) string {
		body := departmentCreateBody(code, bu)
		body["effectiveDate"] = date
		if parent != "" {
			body["parentDepartmentId"] = parent
		}
		return admin.call("POST", "/api/v1/departments", body, foundation.NewID(), 201)["departmentId"].(string)
	}
	root := create("TREE_ROOT", "", "2020-01-01")
	expected := []string{}
	for i := 0; i < 51; i++ {
		expected = append(expected, create(fmt.Sprintf("TREE_%03d", i), root, "2020-01-01"))
		if i == 48 || i == 49 {
			boundary := admin.call("GET", "/api/v1/departments/tree/nodes?parentId="+root, nil, "", 200)
			if len(boundary["items"].([]any)) != i+1 || boundary["hasMore"] != false || boundary["hasPrevious"] != false {
				t.Fatal("49/50 page boundary", boundary)
			}
		}
	}
	create("TREE_FUTURE", root, "2027-01-01")
	leaf := create("TREE_LEAF", expected[25], "2020-01-01")
	origin, e := url.Parse(f.server.URL)
	must(t, e)
	credentials := access.Credentials{Token: admin.client.Jar.Cookies(origin)[0].Value}
	roots, e := f.runtime.Department.TreeNodes(f.ctx, credentials, org.DepartmentTreeInput{PageSize: 20})
	must(t, e)
	if len(roots.Items) != 1 || roots.Items[0].ID != root || roots.Items[0].HasChildren == nil || !*roots.Items[0].HasChildren {
		t.Fatal("root tree", roots)
	}
	nodes, e := f.runtime.Department.TreeNodes(f.ctx, credentials, org.DepartmentTreeInput{ParentID: &root, PageSize: 20})
	must(t, e)
	if len(nodes.Items) != 20 || !nodes.HasMore || nodes.HasPrevious {
		t.Fatal("first child page", nodes)
	}
	again, e := f.runtime.Department.TreeNodes(f.ctx, credentials, org.DepartmentTreeInput{ParentID: &root, PageSize: 20, Cursor: nodes.NextCursor})
	must(t, e)
	if len(again.Items) != 20 || again.Items[0].ID != expected[20] || !again.HasPrevious {
		t.Fatal("next child page", again)
	}
	replayed, e := f.runtime.Department.TreeNodes(f.ctx, credentials, org.DepartmentTreeInput{ParentID: &root, PageSize: 20, Cursor: nodes.NextCursor})
	must(t, e)
	if !reflect.DeepEqual(again, replayed) {
		t.Fatal("valid cursor replay changed static result")
	}
	back, e := f.runtime.Department.TreeNodes(f.ctx, credentials, org.DepartmentTreeInput{ParentID: &root, PageSize: 20, Cursor: again.PreviousCursor})
	must(t, e)
	if !reflect.DeepEqual(nodes.Items, back.Items) {
		t.Fatal("previous window changed", back)
	}
	located, e := f.runtime.Department.TreeLocation(f.ctx, credentials, leaf)
	must(t, e)
	if len(located.Items) != 3 || located.Items[2].ID != leaf {
		t.Fatal("location", located)
	}
	around, e := f.runtime.Department.TreeNodes(f.ctx, credentials, org.DepartmentTreeInput{ParentID: &root, PageSize: 20, AroundID: expected[50]})
	must(t, e)
	if len(around.Items) != 1 || !around.HasPrevious || around.HasMore {
		t.Fatal("around last", around)
	}
	for _, suffix := range []string{"?businessAt=2026-01-01", "?parentId=", "?pageSize=21", "?pageSize=0", "?aroundId=", "?includeDescendants=true"} {
		admin.call("GET", "/api/v1/departments/tree/nodes"+suffix, nil, "", 400)
	}
	admin.call("GET", "/api/v1/departments/tree/nodes?parentId="+root+"&aroundId="+leaf, nil, "", 409)
	for _, suffix := range []string{"?parentId=" + leaf + "&pageSize=20", "?parentId=" + root + "&pageSize=50"} {
		admin.call("GET", "/api/v1/departments/tree/nodes"+suffix+"&cursor="+url.QueryEscape(nodes.NextCursor), nil, "", 409)
	}
	admin.call("GET", "/api/v1/departments/tree/nodes?cursor=broken", nil, "", 409)
	reader := f.login("reader", f.readPassword)
	reader.call("GET", "/api/v1/departments/tree/nodes?parentId="+root+"&pageSize=20&cursor="+url.QueryEscape(nodes.NextCursor), nil, "", 409)
	restoreClock := bootstrap.SetIntegrationClock(f.runtime, func() time.Time { return time.Now().Add(24 * time.Hour) })
	admin.call("GET", "/api/v1/departments/tree/nodes?parentId="+root+"&pageSize=20&cursor="+url.QueryEscape(nodes.NextCursor), nil, "", 409)
	restoreClock()
	admin.call("GET", "/api/v1/departments/tree/nodes?pageSize=20", nil, "", 200)
	admin.call("GET", "/api/v1/departments/tree/search?q=TREE_050", nil, "", 200)
	admin.call("GET", "/api/v1/departments/"+leaf+"/tree-location", nil, "", 200)
	scoped := admin.call("GET", "/api/v1/departments?scopeDepartmentId="+root, nil, "", 200)
	if scoped["count"] != float64(53) {
		t.Fatal("subtree count", scoped)
	}
	own := admin.call("GET", "/api/v1/departments?scopeDepartmentId="+root+"&includeDescendants=false", nil, "", 200)
	if own["count"] != float64(1) {
		t.Fatal("own count", own)
	}
	admin.call("GET", "/api/v1/departments?includeDescendants=true", nil, "", 400)
	admin.call("GET", "/api/v1/departments?scopeDepartmentId="+foundation.NewID(), nil, "", 409)
	selected := admin.call("POST", "/api/v1/departments/selection/start", map[string]any{"query": map[string]any{"scopeDepartmentId": root}, "selection": map[string]any{"mode": "allMatching", "excludedIds": []string{}}, "page": 1, "pageSize": 20}, "", 200)
	if selected["summary"].(map[string]any)["selectedCount"] != float64(53) {
		t.Fatal("scope selection", selected)
	}
	// Scope combines with each existing column predicate; paths retain their
	// cursor contract instead of inventing a total. Expected counts are literal
	// fixture facts, not a second call to the implementation under test.
	for _, tc := range []struct {
		field, operator, value string
		count                  int
	}{
		{"code", "equals", "tree_050", 1},
		{"name", "contains", "研发", 53},
		{"shortName", "equals", "not-present", 0},
		{"effectiveDate", "onOrBefore", "2020-01-01", 53},
		{"managementBuId", "equals", bu, 53},
		{"status", "isAnyOf", "inactive,active", 53},
		{"departmentPath", "contains", "研发", -1},
	} {
		params := url.Values{"scopeDepartmentId": {root}, "filterField": {tc.field}, "filterOperator": {tc.operator}, "filterValue": {tc.value}}
		got := admin.call("GET", "/api/v1/departments?"+params.Encode(), nil, "", 200)
		if tc.count < 0 {
			if got["count"] != nil || len(got["items"].([]any)) != 20 || got["hasMore"] != true {
				t.Fatal("scoped path cursor", got)
			}
		} else if got["count"] != float64(tc.count) {
			t.Fatal("scope intersection", tc.field, got)
		}
	}
	fifty := admin.call("GET", "/api/v1/departments/tree/nodes?parentId="+root, nil, "", 200)
	if len(fifty["items"].([]any)) != 50 || fifty["hasMore"] != true {
		t.Fatal("default fifty boundary", fifty)
	}
	lastOne := admin.call("GET", "/api/v1/departments/tree/nodes?parentId="+root+"&cursor="+url.QueryEscape(fifty["nextCursor"].(string)), nil, "", 200)
	if len(lastOne["items"].([]any)) != 1 || lastOne["hasMore"] != false {
		t.Fatal("fifty-one boundary", lastOne)
	}
	navigation := map[string]any{"query": map[string]any{"scopeDepartmentId": root}, "anchorId": expected[0], "direction": "next"}
	neighbor := admin.call("POST", "/api/v1/departments/navigation", navigation, "", 200)
	if neighbor["targetId"] != expected[1] {
		t.Fatal("scoped next identity", neighbor)
	}
	navigation["anchorId"] = root
	navigation["query"] = map[string]any{"scopeDepartmentId": root, "includeDescendants": false}
	if admin.call("POST", "/api/v1/departments/navigation", navigation, "", 200)["targetId"] != nil {
		t.Fatal("own-only navigation escaped scope")
	}
	navigation["query"] = map[string]any{"scopeDepartmentId": nil}
	admin.call("POST", "/api/v1/departments/navigation", navigation, "", 400)
	// Corrupt only this disposable fixture's derived edge and restore it. A
	// cycle or missing necessary parent must never produce a partial tree/page.
	for _, parent := range []string{leaf, foundation.NewID()} {
		_, e = f.admin.Exec(f.ctx, `UPDATE organization.department_read_segment SET q_parent=$1 WHERE instance_id=$2 AND object_id=$3`, parent, f.instance, root)
		must(t, e)
		admin.call("GET", "/api/v1/departments/"+leaf+"/tree-location", nil, "", 503)
		admin.call("GET", "/api/v1/departments?scopeDepartmentId="+root, nil, "", 503)
	}
	_, e = f.admin.Exec(f.ctx, `UPDATE organization.department_read_segment SET q_parent=NULL WHERE instance_id=$1 AND object_id=$2`, f.instance, root)
	must(t, e)
	admin.call("GET", "/api/v1/departments/"+leaf+"/tree-location", nil, "", 200)
	tx, err := f.admin.Begin(f.ctx)
	must(t, err)
	defer tx.Rollback(context.Background())
	const pageSQL = `SELECT object_id::text,q_code_key FROM organization.department_read_segment
 WHERE instance_id=$1 AND span @> DATE '2026-01-01' AND q_state='active'
 AND q_parent=$2::uuid AND (q_code_key COLLATE "C",object_id) > ($3::text COLLATE "C",$4::uuid)
 ORDER BY q_code_key COLLATE "C",object_id LIMIT 21`
	read := func(key, id string) ([]string, []string) {
		rows, e := tx.Query(f.ctx, pageSQL, f.instance, root, key, id)
		must(t, e)
		defer rows.Close()
		ids, keys := []string{}, []string{}
		for rows.Next() {
			var a, b string
			must(t, rows.Scan(&a, &b))
			ids = append(ids, a)
			keys = append(keys, b)
		}
		must(t, rows.Err())
		return ids, keys
	}
	zero := "00000000-0000-0000-0000-000000000000"
	first, keys := read("", zero)
	if len(first) != 21 || !reflect.DeepEqual(first[:20], expected[:20]) {
		t.Fatal("first window differs", first)
	}
	replay, _ := read("", zero)
	if !reflect.DeepEqual(first, replay) {
		t.Fatal("read cursor replay differs")
	}
	second, secondKeys := read(keys[19], first[19])
	if len(second) != 21 || !reflect.DeepEqual(second[:20], expected[20:40]) {
		t.Fatal("second window differs")
	}
	last, _ := read(secondKeys[19], second[19])
	if !reflect.DeepEqual(last, expected[40:]) {
		t.Fatal("last page or future exclusion differs", last)
	}
	// Filter after traversing structure. The middle parent is intentionally not
	// among the visible IDs, but its visible descendant must remain reachable.
	const scopeSQL = `WITH RECURSIVE current AS MATERIALIZED (
 SELECT object_id,q_parent FROM organization.department_read_segment WHERE instance_id=$1 AND span @> DATE '2026-01-01' AND q_state='active'
 ), branch(id) AS (SELECT object_id FROM current WHERE object_id=$2::uuid
 UNION SELECT c.object_id FROM current c JOIN branch p ON c.q_parent=p.id)
 SELECT id::text FROM branch WHERE id=$3::uuid`
	rows, e := tx.Query(f.ctx, scopeSQL, f.instance, root, leaf)
	must(t, e)
	ids := []string{}
	for rows.Next() {
		var id string
		must(t, rows.Scan(&id))
		ids = append(ids, id)
	}
	must(t, rows.Err())
	rows.Close()
	if !reflect.DeepEqual(ids, []string{leaf}) {
		t.Fatal("row filtering pruned intermediate structural edge", ids)
	}
	t.Log("current-model prototype: 51 siblings across three windows, repeatable position, future exclusion, and traversal before row filtering passed; no capacity or public authorization claim")
}

func TestDepartmentTree300AuthorizationAndFixedDraft(t *testing.T) {
	f := fresh(t)
	admin := f.login("admin", f.password)
	reader := f.login("reader", f.readPassword)
	bu := admin.call("POST", "/api/v1/business-units", createBody("TREE_AUTH_BU", "2020-01-01"), foundation.NewID(), 201)["buId"].(string)
	create := func(code, parent string) string {
		body := departmentCreateBody(code, bu)
		body["effectiveDate"] = "2020-01-01"
		if parent != "" {
			body["parentDepartmentId"] = parent
		}
		return admin.call("POST", "/api/v1/departments", body, foundation.NewID(), 201)["departmentId"].(string)
	}
	a := create("TREE_A", "")
	b := create("TREE_B", a)
	c := create("TREE_C", b)
	// Explicitly grant this reader export for the current member.
	_, e := f.admin.Exec(f.ctx, `INSERT INTO access.subject_grant(instance_id,grant_id,principal_id,scope_kind,scope_id,action,read_fields,write_fields) SELECT instance_id,gen_random_uuid(),principal_id,'department',$3,'department.export',read_fields,ARRAY[]::text[] FROM access.subject_grant WHERE instance_id=$1 AND principal_id=$2 AND action='department.read'`, f.instance, reader.subject, c)
	must(t, e)
	// Preserve references but allow only A and C as operational objects.
	_, e = f.admin.Exec(f.ctx, `UPDATE access.subject_grant SET scope_kind='department',scope_id=$3 WHERE instance_id=$1 AND principal_id=$2 AND action='department.read'`, f.instance, reader.subject, a)
	must(t, e)
	_, e = f.admin.Exec(f.ctx, `INSERT INTO access.subject_grant(instance_id,grant_id,principal_id,scope_kind,scope_id,action,read_fields,write_fields) SELECT instance_id,gen_random_uuid(),principal_id,scope_kind,$3,action,read_fields,write_fields FROM access.subject_grant WHERE instance_id=$1 AND principal_id=$2 AND action='department.read'`, f.instance, reader.subject, c)
	must(t, e)
	roots := reader.call("GET", "/api/v1/departments/tree/nodes", nil, "", 200)["items"].([]any)
	if len(roots) != 1 || roots[0].(map[string]any)["hasChildren"] != false {
		t.Fatal("hidden child exposed", roots)
	}
	// A visible root cannot advertise an operational child whose required
	// fields are denied. This is independent of the root's own field rights.
	_, e = f.admin.Exec(f.ctx, `INSERT INTO access.subject_grant(instance_id,grant_id,principal_id,scope_kind,scope_id,action,read_fields,write_fields) SELECT instance_id,gen_random_uuid(),principal_id,scope_kind,$3,action,read_fields,write_fields FROM access.subject_grant WHERE instance_id=$1 AND principal_id=$2 AND action='department.read' AND scope_id=$4`, f.instance, reader.subject, b, a)
	must(t, e)
	visible := reader.call("GET", "/api/v1/departments/tree/nodes", nil, "", 200)["items"].([]any)
	if visible[0].(map[string]any)["hasChildren"] != true {
		t.Fatal("visible child not discoverable")
	}
	var childFields []string
	must(t, f.admin.QueryRow(f.ctx, `SELECT read_fields FROM access.subject_grant WHERE instance_id=$1 AND principal_id=$2 AND action='department.read' AND scope_id=$3`, f.instance, reader.subject, b).Scan(&childFields))
	for _, field := range []string{"code", "name", "status", "effectiveDate", "parentDepartmentId", "departmentPath"} {
		_, e = f.admin.Exec(f.ctx, `UPDATE access.subject_grant SET read_fields=array_remove($4::text[],$5) WHERE instance_id=$1 AND principal_id=$2 AND action='department.read' AND scope_id=$3`, f.instance, reader.subject, b, childFields, field)
		must(t, e)
		reader.call("GET", "/api/v1/departments/tree/nodes", nil, "", 403)
	}
	_, e = f.admin.Exec(f.ctx, `DELETE FROM access.subject_grant WHERE instance_id=$1 AND principal_id=$2 AND action='department.read' AND scope_id=$3`, f.instance, reader.subject, b)
	must(t, e)
	var originalFields []string
	must(t, f.admin.QueryRow(f.ctx, `SELECT read_fields FROM access.subject_grant WHERE instance_id=$1 AND principal_id=$2 AND action='department.read' AND scope_id=$3`, f.instance, reader.subject, a).Scan(&originalFields))
	for _, field := range []string{"code", "name", "status", "effectiveDate", "parentDepartmentId", "departmentPath"} {
		_, e = f.admin.Exec(f.ctx, `UPDATE access.subject_grant SET read_fields=array_remove($4::text[],$5) WHERE instance_id=$1 AND principal_id=$2 AND action='department.read' AND scope_id=$3`, f.instance, reader.subject, a, originalFields, field)
		must(t, e)
		reader.call("GET", "/api/v1/departments/tree/nodes", nil, "", 403)
		reader.call("GET", "/api/v1/departments/"+a+"/tree-location", nil, "", 409)
		reader.call("GET", "/api/v1/departments?scopeDepartmentId="+a, nil, "", 409)
	}
	_, e = f.admin.Exec(f.ctx, `UPDATE access.subject_grant SET read_fields=$4 WHERE instance_id=$1 AND principal_id=$2 AND action='department.read' AND scope_id=$3`, f.instance, reader.subject, a, originalFields)
	must(t, e)
	found := reader.call("GET", "/api/v1/departments/tree/search?q=TREE_C", nil, "", 200)["items"].([]any)
	if len(found) != 1 || found[0].(map[string]any)["id"] != c {
		t.Fatal("readable descendant not searchable")
	}
	chain := reader.call("GET", "/api/v1/departments/"+c+"/tree-location", nil, "", 200)["items"].([]any)
	if len(chain) != 3 || chain[1].(map[string]any)["kind"] != "context" || chain[1].(map[string]any)["hasChildren"] != nil {
		t.Fatal("reference ancestor acquired operations", chain)
	}
	scoped := reader.call("GET", "/api/v1/departments?scopeDepartmentId="+a, nil, "", 200)
	if scoped["count"] != float64(2) {
		t.Fatal("unreadable intermediate pruned descendant", scoped)
	}
	query := map[string]any{"scopeDepartmentId": a}
	start := map[string]any{"query": query, "selection": map[string]any{"mode": "allMatching", "excludedIds": []string{}}, "page": 1, "pageSize": 20}
	selected := reader.call("POST", "/api/v1/departments/selection/start", start, "", 200)
	summary := selected["summary"].(map[string]any)
	edit := draft133Body(summary)
	edit["query"] = query
	edit["removeIds"] = []string{a}
	selected = reader.call("POST", "/api/v1/departments/selection/resolve", edit, "", 200)
	summary = selected["summary"].(map[string]any)
	if summary["selectedCount"] != float64(1) {
		t.Fatal(summary)
	}
	body := draft133Body(summary)
	body["query"] = map[string]any{"scopeDepartmentId": a, "includeDescendants": true}
	reader.call("POST", "/api/v1/departments/selection/resolve", body, "", 200) // default=true has identical query binding.
	_, e = f.admin.Exec(f.ctx, `DELETE FROM access.subject_grant WHERE instance_id=$1 AND principal_id=$2 AND action='department.read' AND scope_id=$3`, f.instance, reader.subject, a)
	must(t, e)
	reader.call("GET", "/api/v1/departments?scopeDepartmentId="+a, nil, "", 409)
	unavailable := reader.call("POST", "/api/v1/departments/selection/resolve", body, "", 200)
	if unavailable["pageState"] != "scope_unavailable" || unavailable["page"] != nil || unavailable["selectedIds"] != nil || unavailable["memberIds"] != nil || !reflect.DeepEqual(unavailable["summary"], summary) {
		t.Fatal("scope failure destroyed or leaked draft page", unavailable)
	}
	body["removeIds"] = []string{c}
	reader.call("POST", "/api/v1/departments/selection/resolve", body, "", 409)
	delete(body, "removeIds")
	again := reader.call("POST", "/api/v1/departments/selection/resolve", body, "", 200)
	if !reflect.DeepEqual(again["summary"], summary) {
		t.Fatal("rejected edit consumed revision or members")
	}
	exportBody := map[string]any{"query": query, "draftId": summary["draftId"], "expectedRevision": summary["revision"]}
	exported := selection131HTTP(f.ctx, reader, "/api/v1/departments/export", exportBody)
	must(t, exported.err)
	if exported.status != 200 || exported.headers.Get("X-Selected-Count") != "1" {
		t.Fatalf("excluded unavailable anchor blocked current member export: %d %s", exported.status, exported.data)
	}
	_, e = f.admin.Exec(f.ctx, `DELETE FROM access.subject_grant WHERE instance_id=$1 AND principal_id=$2 AND action='department.reference.read'`, f.instance, reader.subject)
	must(t, e)
	reader.call("GET", "/api/v1/departments/"+c+"/tree-location", nil, "", 409)
	denied := selection131HTTP(f.ctx, reader, "/api/v1/departments/export", exportBody)
	must(t, denied.err)
	if denied.status != 403 || denied.headers.Get("Content-Disposition") != "" {
		t.Fatal("reference revocation leaked CSV", denied.status)
	}
}

func TestDepartmentTree300FutureMoveAndFixedMembers(t *testing.T) {
	f := fresh(t)
	day := time.Date(2026, 6, 30, 15, 59, 59, 0, time.UTC)
	restore := bootstrap.SetIntegrationClock(f.runtime, func() time.Time { return day })
	defer restore()
	b := f.login("admin", f.password)
	bu := b.call("POST", "/api/v1/business-units", createBody("CLOCK300_BU", "2020-01-01"), foundation.NewID(), 201)["buId"].(string)
	create := func(code, parent string) map[string]any {
		body := departmentCreateBody(code, bu)
		body["effectiveDate"] = "2020-01-01"
		if parent != "" {
			body["parentDepartmentId"] = parent
		}
		return b.call("POST", "/api/v1/departments", body, foundation.NewID(), 201)
	}
	a := create("CLOCK_A", "")["departmentId"].(string)
	other := create("CLOCK_B", "")["departmentId"].(string)
	leaf := create("CLOCK_LEAF", a)
	id := leaf["departmentId"].(string)
	b.call("POST", "/api/v1/departments/"+id+"/records", map[string]any{"expectedSourceVersion": leaf["sourceVersion"], "anchorRecordId": leaf["selectedRecordId"], "effectiveDate": "2026-07-01", "operationType": "CHANGE", "operationReason": "ORGANIZATION_REASSIGNMENT", "changes": map[string]any{"parentDepartmentId": other}}, foundation.NewID(), 201)
	chain := b.call("GET", "/api/v1/departments/"+id+"/tree-location", nil, "", 200)["items"].([]any)
	if chain[0].(map[string]any)["id"] != a {
		t.Fatal("future move leaked early")
	}
	day = day.Add(time.Second)
	chain = b.call("GET", "/api/v1/departments/"+id+"/tree-location", nil, "", 200)["items"].([]any)
	if chain[0].(map[string]any)["id"] != other {
		t.Fatal("today move did not activate")
	}
	query := map[string]any{"scopeDepartmentId": other}
	selected := b.call("POST", "/api/v1/departments/selection/start", map[string]any{"query": query, "selection": map[string]any{"mode": "allMatching", "excludedIds": []string{}}, "page": 1, "pageSize": 20}, "", 200)
	summary := selected["summary"].(map[string]any)
	if summary["selectedCount"] != float64(2) {
		t.Fatal(summary)
	}
	newer := create("CLOCK_NEW", other)["departmentId"].(string)
	detail := b.call("GET", "/api/v1/departments/"+id, nil, "", 200)
	record := detail["selectedRecord"].(map[string]any)
	b.call("POST", "/api/v1/departments/"+id+"/records/"+record["recordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": detail["headSourceVersion"], "expectedRecordRevision": record["revision"], "changes": map[string]any{"parentDepartmentId": a}}, foundation.NewID(), 200)
	body := draft133Body(summary)
	body["query"] = query
	resolved := b.call("POST", "/api/v1/departments/selection/resolve", body, "", 200)
	if resolved["summary"].(map[string]any)["selectedCount"] != float64(2) || len(resolved["selectedIds"].([]any)) != 1 {
		t.Fatal("moved-out identity lost or entrant selected", resolved)
	}
	exportBody := map[string]any{"query": query, "draftId": summary["draftId"], "expectedRevision": summary["revision"]}
	csv := selection131HTTP(f.ctx, b, "/api/v1/departments/export", exportBody)
	must(t, csv.err)
	if csv.status != 200 || !strings.Contains(string(csv.data), "CLOCK_LEAF") || strings.Contains(string(csv.data), "CLOCK_NEW") || strings.Contains(string(csv.data), newer) {
		t.Fatal("fixed export changed membership", csv.status)
	}
	base := "/api/v1/departments/" + id
	exactBefore := b.call("GET", base+"?sourceVersion="+leaf["sourceVersion"].(string), nil, "", 200)
	detail = b.call("GET", base, nil, "", 200)
	record = detail["selectedRecord"].(map[string]any)
	added := b.call("POST", base+"/records", map[string]any{"expectedSourceVersion": detail["headSourceVersion"], "anchorRecordId": record["recordId"], "effectiveDate": "2026-07-03", "operationType": "CHANGE", "operationReason": "ORGANIZATION_REASSIGNMENT", "changes": map[string]any{"name": "未来更名部门", "parentDepartmentId": other}}, foundation.NewID(), 201)
	chain = b.call("GET", base+"/tree-location", nil, "", 200)["items"].([]any)
	if chain[0].(map[string]any)["id"] != a || chain[1].(map[string]any)["name"] == "未来更名部门" {
		t.Fatal("future name or relation became current")
	}
	newRecord := added["targetRecordId"].(string)
	redated := b.call("POST", base+"/records/"+newRecord+"/redate", map[string]any{"expectedSourceVersion": added["sourceVersion"], "expectedRecordRevision": added["targetRevision"], "newEffectiveDate": "2026-07-02"}, foundation.NewID(), 200)
	day = day.Add(24 * time.Hour)
	chain = b.call("GET", base+"/tree-location", nil, "", 200)["items"].([]any)
	if chain[0].(map[string]any)["id"] != other || chain[1].(map[string]any)["name"] != "未来更名部门" {
		t.Fatal("redated current relation/name not reflected")
	}
	b.call("POST", base+"/records/"+newRecord+"/delete", map[string]any{"expectedSourceVersion": redated["sourceVersion"], "expectedRecordRevision": redated["targetRevision"]}, foundation.NewID(), 200)
	chain = b.call("GET", base+"/tree-location", nil, "", 200)["items"].([]any)
	if chain[0].(map[string]any)["id"] != a || chain[1].(map[string]any)["name"] == "未来更名部门" {
		t.Fatal("deleted current record left stale tree")
	}
	secondBU := b.call("POST", "/api/v1/business-units", createBody("CLOCK300_SECOND_BU", "2020-01-01"), foundation.NewID(), 201)["buId"].(string)
	detail = b.call("GET", base, nil, "", 200)
	record = detail["selectedRecord"].(map[string]any)
	b.call("POST", base+"/records/"+record["recordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": detail["headSourceVersion"], "expectedRecordRevision": record["revision"], "changes": map[string]any{"managementBuId": secondBU, "parentDepartmentId": nil}}, foundation.NewID(), 200)
	chain = b.call("GET", base+"/tree-location", nil, "", 200)["items"].([]any)
	if len(chain) != 1 || chain[0].(map[string]any)["id"] != id || chain[0].(map[string]any)["parentId"] != nil {
		t.Fatal("legal BU transfer retained old tree parent")
	}
	exactAfter := b.call("GET", base+"?sourceVersion="+leaf["sourceVersion"].(string), nil, "", 200)
	if !reflect.DeepEqual(exactBefore["path"], exactAfter["path"]) || !reflect.DeepEqual(exactBefore["timeline"], exactAfter["timeline"]) {
		t.Fatal("tree maintenance altered original source facts")
	}
}

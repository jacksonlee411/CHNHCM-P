//go:build integration

package bootstrap_test

import (
	"chnhcm.local/chnhcm/internal/bootstrap"
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	app "chnhcm.local/chnhcm/internal/contexts/staffing/application"
	s "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"fmt"
	"slices"
	"strings"
	"testing"
)

func TestStaffing315DepartmentAllocation(t *testing.T) {
	h := staffingSetup307(t)
	f, b := h.f, h.b
	reader := f.login("reader", f.readPassword)
	must(t, f.runtime.InitializeCountryCatalog(f.ctx, access.LoginInput{Login: "admin", Password: f.password}))
	locs := location204{t, f, b, map[string]string{"MO": geoCountry203(t, f, "MO")}}
	loc := locs.create("ALLOC315", "MO", "", "2026-01-01")["id"]
	bu := b.call("POST", "/api/v1/business-units", createBody("ALLOC315", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	dept := func(code string) string {
		return b.call("POST", "/api/v1/departments", departmentCreateBody(code, bu), foundation.NewID(), 201)["departmentId"].(string)
	}
	fam := h.create("jobfamily", "ALLOC315", "")
	job := h.create("job", "ALLOC315", fam["id"].(string))
	pos := func(dep, code string) string {
		return b.call("POST", "/api/v1/positions", map[string]any{"departmentId": dep, "locationId": loc, "jobId": job["id"], "code": code, "name": code, "effectiveDate": "2026-01-01", "operationType": "CREATE", "operationReason": "NEW_POSITION"}, foundation.NewID(), 201)["objectId"].(string)
	}
	const base = "/api/v1/staffing-quotas"
	const jan = "2027-01-01"
	const may = "2027-05-01"
	const jun = "2027-06-01"
	depPath := func(id string) string { return base + "/departments/" + id }
	get := func(actor *browser, id, day string) map[string]any {
		return actor.call("GET", depPath(id)+"?businessAt="+day, nil, "", 200)
	}
	position := func(id, day string) map[string]any {
		return b.call("GET", base+"/statistics/positions/"+id+"?businessAt="+day, nil, "", 200)
	}
	row := func(id string, n any, source any) map[string]any {
		v := map[string]any{}
		if n == nil {
			v["withdraw"] = true
		} else {
			v["quota"] = n
		}
		if source != nil {
			v["expectedSourceVersion"] = source
		}
		if id != "" {
			v["positionId"] = id
		}
		return v
	}
	input := func(day string, total map[string]any, rows ...map[string]any) map[string]any {
		v := map[string]any{"effectiveDate": day, "reason": "年度编制调整"}
		if total != nil {
			v["total"] = total
		}
		if len(rows) > 0 {
			v["positions"] = rows
		}
		return v
	}
	save := func(id string, v map[string]any, want int) map[string]any {
		return b.call("POST", depPath(id)+"/allocations", v, foundation.NewID(), want)
	}
	check := func(a, b any) {
		t.Helper()
		if a != b {
			t.Fatalf("实际 %v，预期 %v", a, b)
		}
	}
	checkDates := func(view map[string]any, want ...string) {
		t.Helper()
		var got []string
		if values, ok := view["allocationDates"].([]any); ok {
			for _, value := range values {
				got = append(got, value.(string))
			}
		}
		if !slices.Equal(got, want) {
			t.Fatalf("分配变化日期 %v，预期 %v", got, want)
		}
	}
	create := func(id string, n int) {
		b.call("POST", base, map[string]any{"positionId": id, "quota": n, "effectiveDate": jan, "reason": "岗位独立核编"}, foundation.NewID(), 200)
	}
	empty, finance := dept("CS315"), dept("FIN315")
	// With no position or quota yet, the list stays small and the directory
	// can still page through all eligible departments and select the empty one.
	optionsPath := base + "/department-options?businessAt=" + jan
	firstPage := reader.call("GET", optionsPath+"&pageSize=1", nil, "", 200)
	check(firstPage["hasMore"], true)
	secondPage := reader.call("GET", optionsPath+"&pageSize=1&page=2", nil, "", 200)
	check(secondPage["hasMore"], false)
	if firstPage["items"].([]any)[0].(map[string]any)["id"] == secondPage["items"].([]any)[0].(map[string]any)["id"] {
		t.Fatal("部门候选页重复")
	}
	list := reader.call("GET", base+"/statistics?businessAt="+jan, nil, "", 200)
	check(list["canBrowseDepartments"], true)
	check(len(list["departments"].([]any)), 0)
	selected := reader.call("GET", base+"/statistics?businessAt="+jan+"&departmentId="+empty, nil, "", 200)
	check(selected["departments"].([]any)[0].(map[string]any)["departmentId"], empty)
	choices := reader.call("GET", optionsPath+"&q=cs315", nil, "", 200)
	check(choices["items"].([]any)[0].(map[string]any)["id"], empty)
	f.exec("UPDATE access.subject_grant SET scope_kind='department',scope_id=$1 WHERE principal_id=$2 AND action='department.history.read'", empty, reader.subject)
	scopedOptions := reader.call("GET", optionsPath, nil, "", 200)
	check(len(scopedOptions["items"].([]any)), 1)
	check(scopedOptions["items"].([]any)[0].(map[string]any)["id"], empty)
	f.exec("UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'code') WHERE principal_id=$1 AND action='department.history.read'", reader.subject)
	masked := reader.call("GET", optionsPath+"&q=cs315", nil, "", 200)
	check(len(masked["items"].([]any)), 0)
	masked = reader.call("GET", optionsPath, nil, "", 200)
	check(masked["items"].([]any)[0].(map[string]any)["code"], nil)
	f.exec("UPDATE access.subject_grant SET read_fields=array_append(read_fields,'code') WHERE principal_id=$1 AND action='department.history.read'", reader.subject)
	f.exec("UPDATE access.subject_grant SET scope_kind='enterprise',scope_id=(SELECT scope_id FROM access.subject_grant WHERE principal_id=$1 AND action='department.read' AND scope_kind='enterprise' LIMIT 1) WHERE principal_id=$1 AND action='department.history.read'", reader.subject)
	early := save(empty, input("1900-01-01", row("", 20, nil)), 409)
	check(early["message"], "生效日不能早于部门首日 2026-01-01")
	checkDates(get(b, empty, jan))
	check(get(b, empty, jan)["defaultEffectiveDate"], "1900-01-01")
	save(empty, input(jan, row("", 20, nil)), 200)
	check(get(reader, empty, jan)["remaining"], float64(20))
	unchangedSource := get(b, empty, jan)["sourceVersion"]
	same := save(empty, input(may, row("", 20, unchangedSource)), 200)
	check(same["outcome"], "no_change")
	check(get(b, empty, may)["sourceVersion"], unchangedSource)
	a, z := pos(finance, "AC315"), pos(finance, "MG315")
	create(a, 3)
	create(z, 1)
	check(get(b, finance, may)["defaultEffectiveDate"], may)
	short := save(finance, input(jan, row("", 3, nil)), 409)
	if !strings.Contains(short["message"].(string), "1 编") {
		t.Fatal(short)
	}
	save(finance, input(jan, row("", 5, nil)), 200)
	unassigned := pos(finance, "FUT315")
	futureReceipt := b.call("POST", base, map[string]any{"positionId": unassigned, "effectiveDate": jun, "quota": 0, "reason": "未来零额度"}, foundation.NewID(), 200)
	check(get(reader, finance, jan)["unsetCount"], float64(1))
	check(get(reader, finance, jun)["unsetCount"], float64(0))
	b.call("POST", base+"/"+futureReceipt["objectId"].(string)+"/withdraw", map[string]any{"effectiveDate": "2027-07-01", "expectedSourceVersion": futureReceipt["sourceVersion"], "reason": "暑期撤去"}, foundation.NewID(), 200)
	check(get(reader, finance, "2027-07-01")["unsetCount"], float64(1))
	h.add("position", h.detail("position", unassigned, ""), "DEACTIVATE", "2027-08-01", nil, 200)
	check(get(reader, finance, "2027-08-01")["unsetCount"], float64(0))
	checkDates(get(reader, finance, jan), "2026-01-01", jan, jun, "2027-07-01", "2027-08-01")
	filtered := reader.call("GET", base+"/statistics?businessAt="+jan+"&departmentId="+finance+"&q=AC315", nil, "", 200)
	check(filtered["totalCount"], float64(1))
	check(filtered["departments"].([]any)[0].(map[string]any)["unsetCount"], float64(1))
	v := get(reader, finance, jan)
	check(v["total"], float64(4))
	check(v["remaining"], float64(1))
	check(v["canSetTotal"], false)
	reader.call("POST", depPath(finance)+"/allocations", input(may, row("", 6, v["sourceVersion"])), foundation.NewID(), 403)
	f.exec("UPDATE access.subject_grant SET scope_kind='position',scope_id=$1 WHERE principal_id=$2 AND action LIKE 'staffingquota.%'", a, reader.subject)
	reader.call("GET", optionsPath, nil, "", 403)
	partial := get(reader, finance, jan)
	checkDates(partial, "2026-01-01", jan)
	f.exec("UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'effectiveDate') WHERE principal_id=$1 AND action='staffingquota.history.read'", reader.subject)
	checkDates(get(reader, finance, jan))
	f.exec("UPDATE access.subject_grant SET read_fields=array_append(read_fields,'effectiveDate') WHERE principal_id=$1 AND action='staffingquota.history.read'", reader.subject)
	check(partial["scope"], "visible")
	check(partial["total"], float64(3))
	check(partial["unsetCount"], float64(0))
	for _, k := range []string{"remaining", "quota", "quotaState", "sourceVersion", "history"} {
		if _, ok := partial[k]; ok {
			t.Fatal("范围泄露", k)
		}
	}
	checkVisiblePositions := func() {
		// The second grant can name its position, but cannot identify its department.
		f.exec(`INSERT INTO access.subject_grant
			SELECT instance_id, gen_random_uuid(), principal_id, 'position', $1, action,
				array_remove(read_fields,'departmentId'), write_fields
			FROM access.subject_grant WHERE principal_id=$2 AND scope_id=$3
				AND action IN ('staffingquota.read','staffingquota.history.read')`, z, reader.subject, a)
		defer f.exec("DELETE FROM access.subject_grant WHERE principal_id=$1 AND scope_id=$2 AND action LIKE 'staffingquota.%'", reader.subject, z)
		list := func(query string) map[string]any {
			return reader.call("GET", base+"/statistics?businessAt="+jan+"&pageSize=1&"+query, nil, "", 200)
		}
		for i, id := range []string{a, z} {
			page := list(fmt.Sprintf("page=%d", i+1))
			check(page["totalCount"], float64(2))
			item := page["items"].([]any)[0].(map[string]any)
			check(item["positionId"], id)
			if id == z {
				check(item["departmentId"], nil)
			}
			check(page["positionSummary"].(map[string]any)["total"], float64(4))
			check(page["departments"].([]any)[0].(map[string]any)["total"], float64(3))
		}
		onlyZ := list("q=MG315&anchor=" + z)
		check(onlyZ["positionSummary"].(map[string]any)["total"], float64(1))
		nav := onlyZ["navigation"].(map[string]any)
		check(nav["position"], float64(1))
		check(nav["totalCount"], float64(1))
		check(nav["previousId"], nil)
		check(nav["nextId"], nil)
		check(list("q=FUT315")["positionSummary"], nil)
		f.exec("UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'status') WHERE principal_id=$1 AND scope_id=$2 AND action='staffingquota.history.read'", reader.subject, a)
		defer f.exec("UPDATE access.subject_grant SET read_fields=array_append(read_fields,'status') WHERE principal_id=$1 AND scope_id=$2 AND action='staffingquota.history.read'", reader.subject, a)
		page := list("page=1")
		check(len(page["positionSummary"].(map[string]any)), 0)
		// Now both positions can identify the same department. One publishable
		// amount must not become that department's displayed subtotal.
		f.exec("UPDATE access.subject_grant SET read_fields=array_append(read_fields,'departmentId') WHERE principal_id=$1 AND scope_id=$2 AND action IN ('staffingquota.read','staffingquota.history.read')", reader.subject, z)
		detail := get(reader, finance, jan)
		for _, key := range []string{"total", "unsetCount", "remaining"} {
			if _, shown := detail[key]; shown {
				t.Fatalf("缺少岗位状态读权时返回了 %s", key)
			}
		}
	}
	checkVisiblePositions()
	save(finance, input(may, row("", nil, v["sourceVersion"])), 200)
	check(get(b, finance, may)["quotaState"], "unset")
	check(get(b, finance, may)["remaining"], nil)
	check(position(a, may)["quota"], float64(3))
	eng := dept("ENG315")
	dev, qa := pos(eng, "DEV315"), pos(eng, "QA315")
	save(eng, input(jan, row("", 10, nil), row(dev, 6, nil), row(qa, 3, nil)), 200)
	origin := get(b, eng, jan)
	left, right := position(dev, jan), position(qa, jan)
	single := func(view map[string]any, day string, n int, want int) {
		b.call("POST", base+"/"+view["quotaId"].(string)+"/adjust", map[string]any{"expectedSourceVersion": view["sourceVersion"], "effectiveDate": day, "quota": n, "reason": "岗位调整"}, foundation.NewID(), want)
	}
	single(right, may, 5, 409)
	single(right, jun, 4, 200)
	right = position(qa, jan)
	fail := save(eng, input(may, row("", 8, origin["sourceVersion"]), row(dev, 5, left["sourceVersion"])), 409)
	if !strings.Contains(fail["message"].(string), jun) {
		t.Fatal("首个冲突日", fail)
	}
	check(get(b, eng, may)["quota"], float64(10))
	check(position(dev, may)["quota"], float64(6))
	save(eng, input(may, row("", 8, origin["sourceVersion"])), 409)
	save(eng, input(may, row("", 8, origin["sourceVersion"]), row(dev, 4, left["sourceVersion"])), 200)
	check(get(b, eng, "2027-04-30")["quota"], float64(10))
	check(get(b, eng, jun)["remaining"], float64(0))
	// Two target writes and their terminal must roll back together on an infrastructure fault.
	later := "2027-07-01"
	capNow := get(b, eng, later)
	payload := input(later, row("", 10, capNow["sourceVersion"]), row(dev, 5, position(dev, later)["sourceVersion"]), row(qa, 5, right["sourceVersion"]))
	key := foundation.NewID()
	path := depPath(eng) + "/allocations"
	f.exec(`CREATE FUNCTION staffing.fail_last315() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF (SELECT count(*) FROM staffing.quota_outbox WHERE event->>'submissionId'=NEW.event->>'submissionId')=2 THEN RAISE EXCEPTION 'third row interrupted'; END IF; RETURN NEW; END $$;
 CREATE TRIGGER fail_last315 BEFORE INSERT ON staffing.quota_outbox FOR EACH ROW EXECUTE FUNCTION staffing.fail_last315()`)
	b.call("POST", path, payload, key, 503)
	check(get(b, eng, later)["quota"], float64(8))
	check(position(dev, later)["quota"], float64(4))
	check(f.scalar("SELECT count(*) FROM staffing.quota_source WHERE submission_id=$1", key), 0)
	check(f.scalar("SELECT count(*) FROM staffing.quota_terminal WHERE submission_id=$1", key), 0)
	f.exec("DROP TRIGGER fail_last315 ON staffing.quota_outbox")
	loseResponseAndRestart315(t, f, b, path, payload, key)
	again := b.call("POST", base+"/submissions/"+key+"/resolve", nil, "", 200)
	check(len(again["quotaChanges"].([]any)), 3)
	check(get(b, eng, later)["remaining"], float64(0))
	check(position(dev, later)["quota"], float64(5))
	check(position(qa, later)["quota"], float64(5))
	check(len(b.call("POST", path, payload, key, 200)["quotaChanges"].([]any)), 3)
	b.call("POST", path, input(jan, row("", 50, nil)), key, 409)
	// Historical sources remain accessible at their original amounts. A later staffing
	// restart or transfer must fit the cap of the direct receiving department.
	snap := get(b, eng, later+"&sourceVersion="+origin["sourceVersion"].(string))
	check(snap["quota"], float64(10))
	check(snap["remaining"], nil)
	check(snap["allocationDates"], nil)
	selectedTotal := "&selectedRecordId=" + origin["recordId"].(string)
	originalRecord := get(b, eng, later+selectedTotal+"&sourceVersion="+origin["sourceVersion"].(string))
	check(originalRecord["businessAt"], jan)
	check(originalRecord["quota"], float64(10))
	check(originalRecord["recordId"], origin["recordId"])
	check(originalRecord["actions"], nil)
	check(len(originalRecord["members"].([]any)), 0)
	currentRecord := get(b, eng, later+selectedTotal)
	check(currentRecord["businessAt"], jan)
	check(currentRecord["total"], float64(9))
	missing := get(b, eng, jan+"&selectedRecordId="+foundation.NewID())
	check(missing["selectionMissing"], true)
	check(missing["quota"], nil)
	check(missing["actions"], nil)
	// Selecting a position-only change cannot address the older total for editing.
	check(get(b, eng, jun)["actions"], nil)
	b.call("GET", depPath(eng)+"?sourceVersion="+origin["sourceVersion"].(string)+"&selectedRecordId="+foundation.NewID(), nil, "", 404)
	stop := h.add("position", h.detail("position", dev, ""), "DEACTIVATE", "2027-08-01", nil, 200)
	check(get(b, eng, "2027-08-01")["remaining"], float64(5))
	save(eng, input("2027-08-01", row("", 5, get(b, eng, later)["sourceVersion"])), 200)
	h.add("position", h.detail("position", dev, stop["selectedRecordId"].(string)), "REACTIVATE", "2027-09-01", nil, 409)
	h.add("position", h.detail("position", a, ""), "ADD", "2027-09-01", map[string]any{"departmentId": eng}, 409)
	floor := b.call("GET", "/api/v1/departments/"+empty, nil, "", 200)
	r := floor["selectedRecord"].(map[string]any)
	b.call("POST", "/api/v1/departments/"+empty+"/records/"+r["recordId"].(string)+"/redate", map[string]any{"expectedSourceVersion": floor["headSourceVersion"], "expectedRecordRevision": r["revision"], "newEffectiveDate": may}, foundation.NewID(), 409)
	save(empty, input(may, row("", 0, get(b, empty, jan)["sourceVersion"])), 200)
	check(get(b, empty, may)["remaining"], float64(0))
	// From 10 / 6 / 3, independent future nodes and a mixed grant change the outcome.
	plan := dept("PLAN315")
	pDev, pQA := pos(plan, "DPLAN315"), pos(plan, "QPLAN315")
	save(plan, input(jan, row("", 10, nil), row(pDev, 6, nil), row(pQA, 3, nil)), 200)
	origDev, origQA := position(pDev, jan), position(pQA, jan)
	origCap := get(b, plan, jan)
	group := input(jan, nil, row(pDev, 5, origDev["sourceVersion"]), row(pQA, 5, origQA["sourceVersion"]))
	f.exec("UPDATE access.subject_grant SET scope_kind='position',scope_id=$1 WHERE principal_id=$2 AND action='staffingquota.correct'", pDev, b.subject)
	save(plan, group, 403)
	check(position(pDev, jan)["quota"], float64(6))
	check(position(pQA, jan)["quota"], float64(3))
	f.exec("UPDATE access.subject_grant SET scope_kind='enterprise',scope_id=(SELECT scope_id FROM access.subject_grant WHERE principal_id=$1 AND action='staffingquota.read' AND scope_kind='enterprise') WHERE principal_id=$1 AND action='staffingquota.correct'", b.subject)
	save(plan, input(may, row("", 8, origCap["sourceVersion"]), row(pDev, 5, origDev["sourceVersion"])), 200)
	check(get(b, plan, may)["remaining"], float64(0))
	check(get(b, plan, "2027-04-30")["remaining"], float64(1))
	save(plan, input("2027-03-01", row("", 11, get(b, plan, may)["sourceVersion"]), row(pQA, 5, origQA["sourceVersion"])), 409) // the May successor is independent
	save(plan, input(jun, row("", 11, get(b, plan, may)["sourceVersion"]), row(pDev, 6, position(pDev, jun)["sourceVersion"]), row(pQA, 5, origQA["sourceVersion"])), 200)
	check(get(b, plan, jun)["remaining"], float64(0))
	// The group receipt follows the current scope and historical field grant.
	f.exec("UPDATE access.subject_grant SET scope_kind='position',scope_id=$1 WHERE principal_id=$2 AND action='staffingquota.history.read'", dev, b.subject)
	reduced := b.call("POST", base+"/submissions/"+key+"/resolve", nil, "", 200)["quotaChanges"].([]any)
	check(len(reduced), 1)
	check(reduced[0].(map[string]any)["targetId"], dev)
	f.exec("UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'departmentId') WHERE principal_id=$1 AND action='staffingquota.history.read'", b.subject)
	hiddenDept := b.call("POST", base+"/submissions/"+key+"/resolve", nil, "", 200)
	check(hiddenDept["departmentId"], nil)
	check(len(hiddenDept["quotaChanges"].([]any)), 1)
	f.exec("UPDATE access.subject_grant SET read_fields=array_append(read_fields,'departmentId') WHERE principal_id=$1 AND action='staffingquota.history.read'", b.subject)
	f.exec("UPDATE access.subject_grant SET scope_kind='enterprise',scope_id=(SELECT scope_id FROM access.subject_grant WHERE principal_id=$1 AND action='staffingquota.read' AND scope_kind='enterprise') WHERE principal_id=$1 AND action='staffingquota.history.read'", b.subject)
	// AC094: only active, directly assigned posts count on each date.
	from, to := dept("FROM315"), dept("TO315")
	pA, pB, pC := pos(from, "A094"), pos(from, "B094"), pos(from, "C094")
	save(from, input(jan, row("", 6, nil), row(pA, 2, nil), row(pB, 3, nil)), 200)
	save(to, input(jan, row("", 2, nil)), 200)
	save(from, input("2028-01-01", nil, row(pC, 1, nil)), 200)
	check(position(pC, may)["quotaState"], "unset")
	check(get(b, from, "2027-03-01")["remaining"], float64(1))
	stopped := h.add("position", h.detail("position", pB, ""), "DEACTIVATE", "2027-04-01", nil, 200)
	stoppedView := h.detail("position", pB, stopped["selectedRecordId"].(string))
	restored := h.add("position", stoppedView, "REACTIVATE", jun, nil, 200)
	check(get(b, from, may)["remaining"], float64(4))
	check(position(pB, may)["quota"], float64(3))
	check(get(b, from, jun)["remaining"], float64(1))
	f.exec("UPDATE access.subject_grant SET scope_id=$1 WHERE principal_id=$2 AND scope_kind='position' AND action LIKE 'staffingquota.%'", pA, reader.subject)
	for _, day := range []string{"2027-03-01", may, jun} {
		limited := get(reader, from, day)
		checkDates(limited, "2026-01-01", jan)
		check(limited["total"], float64(2))
		check(len(limited["members"].([]any)), 1)
		check(limited["remaining"], nil)
	}
	bview := h.detail("position", pB, restored["selectedRecordId"].(string))
	future := "2027-07-01"
	h.add("position", bview, "ADD", future, map[string]any{"departmentId": to}, 409)
	check(position(pB, future)["departmentId"], from)
	save(to, input(future, row("", 3, get(b, to, future)["sourceVersion"])), 200)
	h.add("position", bview, "ADD", future, map[string]any{"departmentId": to}, 200)
	checkDates(get(b, from, future), "2026-01-01", jan, "2027-04-01", jun, future, "2028-01-01")
	checkDates(get(b, to, future), jan, future)
	f.exec("UPDATE access.subject_grant SET scope_id=$1 WHERE principal_id=$2 AND scope_kind='position' AND action LIKE 'staffingquota.%'", pB, reader.subject)
	leftDepartment := get(reader, from, future)
	checkDates(leftDepartment, "2026-01-01", jan, "2027-04-01", jun, future)
	check(leftDepartment["total"], float64(0))
	check(leftDepartment["remaining"], nil)
	check(len(leftDepartment["members"].([]any)), 0)
	reader.call("GET", depPath(eng)+"?businessAt="+future, nil, "", 404)
	check(get(b, from, future)["total"], float64(2))
	check(get(b, to, future)["total"], float64(3))
	check(get(b, from, jun)["total"], float64(5))
	// A position-scoped clerk may not infer the number of hidden funded posts from a refusal.
	f.exec("UPDATE access.subject_grant SET scope_kind='position',scope_id=$1 WHERE principal_id=$2 AND action IN ('staffingquota.read','staffingquota.history.read')", pB, b.subject)
	narrow := position(pB, future)
	denial := b.call("POST", base+"/"+narrow["quotaId"].(string)+"/adjust", map[string]any{"effectiveDate": "2027-08-01", "quota": 4, "reason": "增编", "expectedSourceVersion": narrow["sourceVersion"]}, foundation.NewID(), 409)
	check(denial["message"], "本次调整超过部门编制上限")
	f.exec("UPDATE access.subject_grant SET scope_kind='enterprise',scope_id=(SELECT scope_id FROM access.subject_grant WHERE principal_id=$1 AND action='staffingquota.adjust' AND scope_kind='enterprise') WHERE principal_id=$1 AND action IN ('staffingquota.read','staffingquota.history.read')", b.subject)
	// Both administrators saw one remaining place. Hold the first transaction after validation.
	passwd := foundation.NewID()
	must(t, bootstrap.Initialize(f.ctx, f.admin, bootstrap.InitInput{DevelopmentFixture: true, Instance: f.instance, EnterpriseCode: "LOCAL", EnterpriseName: "本地验证企业", Start: "1900-01-01", Login: "allocation-admin", Password: passwd}))
	creds := []foundation.Credentials{geoCredentials203(t, f, "admin", f.password), geoCredentials203(t, f, "allocation-admin", passwd)}
	for round, suffix := range []string{"A", "B"} {
		dep := dept("RACE315" + suffix)
		p0, p1 := pos(dep, "R315D"+suffix), pos(dep, "R315Q"+suffix)
		save(dep, input(jan, row("", 10, nil), row(p0, 6, nil), row(p1, 3, nil)), 200)
		p := []map[string]any{position(p0, jan), position(p1, jan)}
		svc := f.runtime.Staffing.(app.Service)
		gate := &quotaGate315{svc.QuotaStore, make(chan struct{}), make(chan struct{})}
		type verdict struct {
			r s.Result
			e error
		}
		ch1, ch2 := make(chan verdict, 1), make(chan verdict, 1)
		run := func(svc app.Service, i int, ch chan<- verdict) {
			n := []int{7, 4}[i]
			cmd := s.QuotaCommand{Action: "adjust", QuotaID: p[i]["quotaId"].(string), SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Input: s.QuotaInput{ExpectedSource: p[i]["sourceVersion"].(string), EffectiveDate: may, Amount: &n, Reason: "增加一编"}}
			if i == 1 {
				cmd.Action = "allocate"
				cmd.QuotaID = ""
				cmd.DepartmentID = dep
				cmd.Allocation = &s.AllocationInput{EffectiveDate: may, Reason: "同日增编", Positions: []s.AllocationRow{{PositionID: p1, ExpectedSource: p[i]["sourceVersion"].(string), Amount: &n}}}
				cmd.Input = s.QuotaInput{}
			}
			res, e := svc.ChangeStaffingQuota(f.ctx, creds[i], cmd)
			ch <- verdict{res, e}
		}
		go run(app.Service{QuotaStore: gate}, round, ch1)
		wait307(t, f, gate.ready)
		go run(svc, 1-round, ch2)
		blocked307(t, f, gate.release)
		close(gate.release)
		one, two := <-ch1, <-ch2
		must(t, one.e)
		must(t, two.e)
		check(one.r.Outcome, "accepted")
		check(two.r.FailureStatus, 409)
		check(get(b, dep, may)["remaining"], float64(0))
	}
}

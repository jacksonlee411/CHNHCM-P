//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/foundation"
)

func operationActors(f *fixture) (*browser, *browser) {
	a, b := f.login("admin", f.password), f.login("reader", f.readPassword)
	mutateLifeGrants(f, b.subject, `INSERT INTO access.subject_grant SELECT instance_id,gen_random_uuid(),$1,scope_kind,scope_id,action,read_fields,write_fields FROM access.subject_grant WHERE principal_id=$2 AND action LIKE 'bu.%' AND action NOT IN ('bu.read','bu.history.read','bu.reference.read')`, b.subject, a.subject)
	_, err := f.admin.Exec(f.ctx, `UPDATE access.principal SET display_name=CASE login WHEN 'admin' THEN '甲操作员' ELSE '乙操作员' END`)
	must(f.t, err)
	return a, b
}
func summaryOf(t *testing.T, d map[string]any) map[string]any {
	t.Helper()
	s, ok := selected(d)["operationSummary"].(map[string]any)
	if !ok {
		t.Fatal("selected record lacks operation summary")
	}
	return s
}
func summaryIs(t *testing.T, d map[string]any, r, revision, source, subject string) {
	t.Helper()
	s := summaryOf(t, d)
	if s["recordId"] != r || s["revision"] != revision || s["operationSourceVersion"] != source || s["operator"].(map[string]any)["subjectId"] != subject {
		t.Fatalf("wrong operation attribution: %v", s)
	}
}
func changeItem(t *testing.T, c map[string]any, id string) map[string]any {
	t.Helper()
	for _, v := range c["items"].([]any) {
		item := v.(map[string]any)
		if item["recordId"] == id {
			return item
		}
	}
	t.Fatal("change item missing")
	return nil
}
func directChange(t *testing.T, c map[string]any) map[string]any {
	t.Helper()
	for _, v := range c["items"].([]any) {
		i := v.(map[string]any)
		if i["changeKind"] == "direct" {
			return i
		}
	}
	t.Fatal("direct change missing")
	return nil
}
func logRead(b *browser, path, query string) map[string]any {
	return b.call("GET", path+"/changes?"+query, nil, "", 200)
}
func logFilter(field, op, value string) string {
	return url.Values{"filterField": {field}, "filterOperator": {op}, "filterValue": {value}}.Encode()
}
func correctOperation(b *browser, id, date string, changes map[string]any) map[string]any {
	d := detail(b, id)
	r := recordOn(b.f.t, d, date)
	return b.call("POST", "/api/v1/business-units/"+id+"/records/"+r["recordId"].(string)+"/correct", correctBody(d, r, changes), foundation.NewID(), 200)
}

// Independent S1..S7 oracle: expected identities, actors, dates, membership and
// changed fields are stated here, never obtained by executing production diffs.
func TestOperationSummaryRevisionAndDeletionOracle(t *testing.T) {
	f := fresh(t)
	a, b := operationActors(f)
	body := createBody("OP_S", "2026-01-01")
	body["name"] = "华东一"
	body["nameEn"] = nil
	s1 := a.call("POST", "/api/v1/business-units", body, foundation.NewID(), 201)
	id := s1["buId"].(string)
	path := "/api/v1/business-units/" + id
	r1 := selected(detail(a, id))
	s1r1 := r1["revision"].(string)
	summaryIs(t, detail(a, id), r1["recordId"].(string), s1r1, s1["sourceVersion"].(string), a.subject)
	s2 := b.call("POST", path+"/records", addBody(detail(b, id), "2026-06-01", map[string]any{"name": "华东二"}), foundation.NewID(), 201)
	d2 := detail(a, id)
	r12 := recordOn(t, d2, "2026-01-01")
	r2 := recordOn(t, d2, "2026-06-01")
	s3 := a.call("POST", path+"/records", addBody(d2, "2026-10-01", map[string]any{"name": "华东三"}), foundation.NewID(), 201)
	s4 := correctOperation(b, id, "2026-10-01", map[string]any{"name": "华东三新", "nameEn": "East Three"})
	s5 := correctOperation(a, id, "2026-06-01", map[string]any{"name": "华东二新"})
	d5 := detail(a, id)
	r23 := recordOn(t, d5, "2026-06-01")
	r32 := recordOn(t, d5, "2026-10-01")
	read := func(source, r string) map[string]any {
		q := url.Values{"selectedRecordId": {r}}
		if source != "" {
			q.Set("sourceVersion", source)
		}
		return a.call("GET", path+"?"+q.Encode(), nil, "", 200)
	}
	summaryIs(t, read("", r1["recordId"].(string)), r1["recordId"].(string), r12["revision"].(string), s2["sourceVersion"].(string), b.subject)
	summaryIs(t, read("", r2["recordId"].(string)), r2["recordId"].(string), r23["revision"].(string), s5["sourceVersion"].(string), a.subject)
	summaryIs(t, read("", r32["recordId"].(string)), r32["recordId"].(string), r32["revision"].(string), s4["sourceVersion"].(string), b.subject)
	// A later rename/deactivation of an account must not rewrite past identity.
	_, err := f.admin.Exec(f.ctx, `UPDATE access.principal SET display_name='新显示名',login='renamed_admin' WHERE principal_id=$1`, a.subject)
	must(t, err)
	if summaryOf(t, read(s1["sourceVersion"].(string), r1["recordId"].(string)))["operator"].(map[string]any)["displayName"] != "甲操作员" {
		t.Fatal("identity was joined to today's account")
	}
	s6 := deleteRecord(b, id, r23)
	source6 := s6["sourceVersion"].(string)
	deleted := read(source6, r2["recordId"].(string))
	if selected(deleted)["deleted"] != true || selected(deleted)["revision"] != s6["targetRevision"] {
		t.Fatal("tombstone identity changed")
	}
	summaryIs(t, deleted, r2["recordId"].(string), r23["revision"].(string), s5["sourceVersion"].(string), a.subject)
	summaryIs(t, read(s5["sourceVersion"].(string), r2["recordId"].(string)), r2["recordId"].(string), r23["revision"].(string), s5["sourceVersion"].(string), a.subject)
	summaryIs(t, detail(a, id), r32["recordId"].(string), r32["revision"].(string), s4["sourceVersion"].(string), b.subject)
	logs := logRead(a, path, "")
	ops := logs["items"].([]any)
	if logs["total"] != float64(6) || logs["throughSourceVersion"] != source6 || logs["contextSourceVersion"] != nil {
		t.Fatal("wrong log window")
	}
	deletion := ops[0].(map[string]any)
	item := changeItem(t, deletion, r2["recordId"].(string))
	if deletion["operator"].(map[string]any)["subjectId"] != b.subject || item["afterRef"] != nil || item["afterContext"] != nil || item["beforeRef"].(map[string]any)["revision"] != r23["revision"] || item["beforeRef"].(map[string]any)["sourceVersion"] != s5["sourceVersion"] {
		t.Fatal("delete link/actor is not exact")
	}
	correction := changeItem(t, ops[1].(map[string]any), r2["recordId"].(string))
	diffs := correction["diffs"].([]any)
	c := correction["afterContext"].(map[string]any)
	if len(diffs) != 1 || diffs[0].(map[string]any)["field"] != "name" || c["effectiveDate"] != "2026-06-01" || c["endInclusive"] != "2026-09-30" || c["operationType"] != "CHANGE" || c["nameEn"] != nil {
		t.Fatal("context mistaken for a change", correction)
	}
	derived := changeItem(t, ops[3].(map[string]any), r2["recordId"].(string))
	if derived["changeKind"] != "derived" || derived["diffs"].([]any)[0].(map[string]any)["field"] != "period" {
		t.Fatal("period revision not accounted")
	}
	// Frozen source bounds: current can read S5; historical S4 cannot read S5.
	a.call("GET", path+"/changes?contextSourceVersion="+s4["sourceVersion"].(string)+"&throughSourceVersion="+s5["sourceVersion"].(string), nil, "", 400)
	if logRead(a, path, "throughSourceVersion="+s5["sourceVersion"].(string))["total"] != float64(5) || logRead(a, path, "contextSourceVersion="+s4["sourceVersion"].(string))["total"] != float64(4) || logRead(a, path, "contextSourceVersion="+s4["sourceVersion"].(string)+"&throughSourceVersion="+s3["sourceVersion"].(string))["total"] != float64(3) {
		t.Fatal("context upper boundary")
	}
	// A no-change insertion does not consume a record/source or alter the summary.
	before := businessSnapshot(t, f)
	d6 := detail(a, id)
	no := a.call("POST", path+"/records", addBody(d6, "2026-06-01", map[string]any{}), foundation.NewID(), 200)
	if no["outcome"] != "no_change" {
		t.Fatal("expected no_change")
	}
	after := businessSnapshot(t, f)
	for table, v := range before {
		if table != "terminals" && after[table] != v {
			t.Fatal("no_change changed facts", table)
		}
	}
	s7 := a.call("POST", path+"/records", addBody(d6, "2026-06-01", map[string]any{"name": "华东四"}), foundation.NewID(), 201)
	if s7["selectedRecordId"] == r2["recordId"] {
		t.Fatal("reused deleted identity")
	}
	if logRead(a, path, "throughSourceVersion="+source6)["total"] != float64(6) || logRead(a, path, "")["total"] != float64(7) {
		t.Fatal("snapshot drift")
	}
	for _, field := range []string{"name", "effectiveDate", "operationReason"} {
		found := false
		for _, row := range logRead(a, path, logFilter("changedField", "is", field))["items"].([]any) {
			if row.(map[string]any)["sourceVersion"] == s5["sourceVersion"] {
				found = true
			}
		}
		if found != (field == "name") {
			t.Fatal("unchanged context matched difference filter", field)
		}
	}
	mutateLifeGrants(f, a.subject, `UPDATE access.principal SET state='disabled' WHERE principal_id=$1`, a.subject)
	frozen := b.call("GET", path+"?sourceVersion="+s5["sourceVersion"].(string)+"&selectedRecordId="+r2["recordId"].(string), nil, "", 200)
	identity := summaryOf(t, frozen)["operator"].(map[string]any)
	if identity["displayName"] != "甲操作员" || identity["login"] != "admin" {
		t.Fatal("disabled/renamed identity changed sealed summary")
	}
	t.Log("S1–S7: unchanged/derived/direct/deleted revision attribution; precise links; frozen identities and source windows; no-change and new identity verified")
}

func TestOperationSummaryPairAndFutureOracles(t *testing.T) {
	f := fresh(t)
	a, b := operationActors(f)
	restore := bootstrap.SetIntegrationClock(f.runtime, func() time.Time { return time.Date(2026, 11, 4, 4, 0, 0, 0, time.UTC) })
	defer restore()
	p1 := a.call("POST", "/api/v1/business-units", createBody("OP_PAIR", "2026-01-01"), foundation.NewID(), 201)
	id := p1["buId"].(string)
	path := "/api/v1/business-units/" + id
	d := detail(a, id)
	input := pairInsertBody(d, selected(d), "2026-06-01", "2026-09-01")
	input["deactivationReason"] = "ORG_RESTRUCTURING"
	p2 := a.call("POST", path+"/lifecycle-pairs", input, foundation.NewID(), 200)
	p3 := correctOperation(b, id, "2026-09-01", map[string]any{"name": "恢复修订"})
	d = detail(a, id)
	stop, resume := recordOn(t, d, "2026-06-01"), recordOn(t, d, "2026-09-01")
	p4 := a.call("POST", path+"/lifecycle-pairs/delete", pairDeleteBody(d, stop, resume), foundation.NewID(), 200)
	for _, tc := range []struct {
		r        map[string]any
		s, actor string
	}{{stop, p2["sourceVersion"].(string), a.subject}, {resume, p3["sourceVersion"].(string), b.subject}} {
		original := a.call("GET", path+"?sourceVersion="+p4["sourceVersion"].(string)+"&selectedRecordId="+tc.r["recordId"].(string), nil, "", 200)
		if selected(original)["deleted"] != true || selected(original)["revision"] == tc.r["revision"] {
			t.Fatal("pair tombstone lost")
		}
		summaryIs(t, original, tc.r["recordId"].(string), tc.r["revision"].(string), tc.s, tc.actor)
	}
	if len(logRead(a, path, "")["items"].([]any)[0].(map[string]any)["items"].([]any)) != 3 {
		t.Fatal("pair must include predecessor period change")
	}
	f1 := a.call("POST", "/api/v1/business-units", createBody("OP_FUTURE", "2026-11-10"), foundation.NewID(), 201)
	id = f1["buId"].(string)
	path = "/api/v1/business-units/" + id
	f2 := a.call("POST", path+"/records", addBody(detail(a, id), "2026-12-01", map[string]any{"name": "未来乙"}), foundation.NewID(), 201)
	d = detail(a, id)
	r1, r2 := recordOn(t, d, "2026-11-10"), recordOn(t, d, "2026-12-01")
	f3 := deleteRecord(b, id, r2)
	current := detail(a, id)
	if current["result"] != "M02_NO_FACT" || current["viewFact"] != nil || current["selectedRecordIsFuture"] != true {
		t.Fatal("future default lost")
	}
	newR1 := selected(current)
	summaryIs(t, current, r1["recordId"].(string), newR1["revision"].(string), f3["sourceVersion"].(string), b.subject)
	summaryIs(t, a.call("GET", path+"?sourceVersion="+f3["sourceVersion"].(string)+"&selectedRecordId="+r2["recordId"].(string), nil, "", 200), r2["recordId"].(string), r2["revision"].(string), f2["sourceVersion"].(string), a.subject)
	missing := a.call("GET", path+"?selectedRecordId="+foundation.NewID(), nil, "", 200)
	if missing["selectedRecord"] != nil || missing["selectionMissing"] != true {
		t.Fatal("missing target silently defaulted")
	}
	mutateLifeGrants(f, b.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='bu.history.read'`, b.subject)
	reader := detail(b, id)
	summaryIs(t, reader, r1["recordId"].(string), newR1["revision"].(string), f3["sourceVersion"].(string), b.subject)
	b.call("GET", path+"/changes", nil, "", 403)
	b.call("GET", path+"?sourceVersion="+f3["sourceVersion"].(string), nil, "", 403)
	mutateLifeGrants(f, b.subject, `UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'effectiveDate') WHERE principal_id=$1 AND action='bu.read'`, b.subject)
	hidden := detail(b, id)
	if hidden["selectedRecordIsFuture"] == true || selected(hidden)["effectiveDate"] != nil || summaryOf(t, hidden)["operationSourceVersion"] != f3["sourceVersion"] {
		t.Fatal("date loss leaked future or hid legal summary")
	}
	mutateLifeGrants(f, b.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='bu.read'`, b.subject)
	b.call("GET", path, nil, "", 404)
	t.Log("P1–P4 and F1–F3: per-record deletion originals; NO_FACT selected summary; default missing target; history/date/object permission separation verified")
}

func TestChangeLogPaginationFiltersPrivacyAndBudget(t *testing.T) {
	f := fresh(t)
	a, b := operationActors(f)
	body := createBody("OP_H", "2026-01-01")
	body["name"] = "样本1"
	body["nameEn"] = nil
	initial := a.call("POST", "/api/v1/business-units", body, foundation.NewID(), 201)
	id := initial["buId"].(string)
	path := "/api/v1/business-units/" + id
	sources := []string{initial["sourceVersion"].(string)}
	revision := initial["targetRevision"]
	if revision == nil {
		revision = selected(detail(a, id))["revision"]
	}
	record := selected(detail(a, id))["recordId"].(string)
	for n := 2; n <= 45; n++ {
		who := a
		if n%2 == 0 {
			who = b
		}
		changes := map[string]any{"name": fmt.Sprintf("样本%d", n)}
		if n%3 == 0 {
			changes["nameEn"] = fmt.Sprintf("EN%d", n)
		}
		saved := who.call("POST", path+"/records/"+record+"/correct", map[string]any{"expectedSourceVersion": sources[len(sources)-1], "expectedRecordRevision": revision, "changes": changes}, foundation.NewID(), 200)
		sources = append(sources, saved["sourceVersion"].(string))
		revision = saved["targetRevision"]
	}
	upper := "throughSourceVersion=" + sources[44]
	invalidRange := a.call("GET", path+"/changes?contextSourceVersion="+sources[0]+"&throughSourceVersion="+sources[44], nil, "", 400)
	if invalidRange["code"] != "M02_INVALID_LOG_RANGE" {
		t.Fatal("invalid range must have a resettable reason", invalidRange)
	}
	seen := map[string]bool{}
	for page, want := range []int{20, 20, 5} {
		res := logRead(a, path, fmt.Sprintf("%s&page=%d", upper, page+1))
		if res["total"] != float64(45) || len(res["items"].([]any)) != want {
			t.Fatal("operation pagination count")
		}
		for _, raw := range res["items"].([]any) {
			source := raw.(map[string]any)["sourceVersion"].(string)
			if seen[source] {
				t.Fatal("duplicate page member")
			}
			seen[source] = true
		}
	}
	if len(seen) != 45 {
		t.Fatal("missing page member")
	}
	for _, size := range []int{50, 100} {
		r := logRead(a, path, fmt.Sprintf("%s&pageSize=%d&page=999", upper, size))
		if len(r["items"].([]any)) != 45 || r["page"] != float64(1) {
			t.Fatal("size/clamp")
		}
	}
	for _, tc := range []struct {
		field, op, value string
		want             int
	}{{"actor", "equals", " ADMIN ", 23}, {"actor", "contains", "乙操作", 22}, {"changedField", "is", "nameEn", 15}, {"changedField", "isAnyOf", "name,nameEn", 45}, {"action", "is", "CORRECT", 44}, {"action", "is", "CREATE", 1}, {"action", "isAnyOf", "CREATE,CORRECT", 45}, {"effectiveDate", "is", "2026-01-01", 45}, {"actor", "contains", "%_\\", 0}} {
		r := logRead(a, path, upper+"&"+logFilter(tc.field, tc.op, tc.value))
		if r["total"] != float64(tc.want) {
			t.Fatalf("filter %s %s: total %v", tc.field, tc.value, r["total"])
		}
	}
	for _, q := range []string{"page=", "page=0", "pageSize=10", "throughSourceVersion=", "contextSourceVersion=", "contextSourceVersion=x", "page=1&page=2", "filterField=actor", logFilter("action", "isAnyOf", "CREATE,CREATE"), logFilter("actor", "contains", strings.Repeat("甲", 401)), logFilter("changedField", "is", "unknown"), logFilter("recordedAt", "is", "2026-02-30"), "businessAt=2026-01-01", "filterValue=x&filterValue=y"} {
		a.call("GET", path+"/changes?"+q, nil, "", 400)
	}
	for _, tc := range []struct{ query, field string }{
		{"page=x", "page"}, {"pageSize=10", "pageSize"}, {"throughSourceVersion=invalid", "throughSourceVersion"}, {"filterField=actor", "filter"},
	} {
		rejected := a.call("GET", path+"/changes?"+tc.query, nil, "", 400)
		if rejected["field"] != tc.field {
			t.Fatal("log reset must identify only its invalid parameter", tc.query, rejected)
		}
	}
	a.call("GET", path+"/changes?throughSourceVersion="+foundation.NewID(), nil, "", 404)
	other := a.call("POST", "/api/v1/business-units", createBody("OP_OTHER", "2026-01-01"), foundation.NewID(), 201)
	a.call("GET", path+"/changes?throughSourceVersion="+other["sourceVersion"].(string), nil, "", 404)
	g := t07Instance(t, f, "2020-01-01")
	foreign := g.login("admin", g.password)
	foreignSource := foreign.call("POST", "/api/v1/business-units", createBody("OP_FOREIGN", "2026-01-01"), foundation.NewID(), 201)
	for _, key := range []string{"throughSourceVersion", "contextSourceVersion"} {
		a.call("GET", path+"/changes?"+key+"="+foreignSource["sourceVersion"].(string), nil, "", 404)
	}
	foreign.call("GET", path+"/changes", nil, "", 404)

	if logRead(a, path, "filterField=&filterOperator=&filterValue=")["total"] != float64(45) {
		t.Fatal("empty filter triplet")
	}
	// Current and history field permissions are intersected before filters/counts.
	mutateLifeGrants(f, b.subject, `UPDATE access.subject_grant SET read_fields=ARRAY['code'] WHERE principal_id=$1 AND action='bu.history.read'`, b.subject)
	restricted := logRead(b, path, "")
	if !strings.Contains(fmt.Sprint(restricted["detailReadFields"]), "nameEn") {
		t.Fatal("ordinary detail fields incorrectly narrowed to history intersection")
	}
	historical := logRead(b, path, "contextSourceVersion="+sources[44])
	if !reflect.DeepEqual(historical["detailReadFields"], historical["readFields"]) {
		t.Fatal("historical detail fields must use history intersection")
	}
	// Ordinary detail permissions are metadata only, separate from log contents.
	delete(restricted, "detailReadFields")
	raw, _ := json.Marshal(restricted)
	for _, secret := range []string{"样本", "EN45", "effectiveDate", "operationReason", "nameEn", "endInclusive"} {
		if strings.Contains(string(raw), secret) {
			t.Fatal("hidden field leaked", secret)
		}
	}
	if restricted["total"] != float64(45) || len(directChange(t, restricted["items"].([]any)[0].(map[string]any))["diffs"].([]any)) != 0 {
		t.Fatal("generic operation row missing")
	}
	for _, q := range []string{logFilter("changedField", "is", "nameEn"), logFilter("changedField", "isAnyOf", "state,period"), logFilter("effectiveDate", "is", "2026-01-01")} {
		b.call("GET", path+"/changes?"+q, nil, "", 403)
	}
	// Grow the same real API history to 1000. Budgets are independent of list/export.
	started := time.Now()
	for n := 46; n <= 1000; n++ {
		changes := map[string]any{"name": fmt.Sprintf("样本%d", n)}
		if n%3 == 0 {
			changes["nameEn"] = fmt.Sprintf("EN%d", n)
		}
		saved := a.call("POST", path+"/records/"+record+"/correct", map[string]any{"expectedSourceVersion": sources[len(sources)-1], "expectedRecordRevision": revision, "changes": changes}, foundation.NewID(), 200)
		sources = append(sources, saved["sourceVersion"].(string))
		revision = saved["targetRevision"]
	}
	t.Logf("H1000 fixture writes: %s", time.Since(started))
	for _, q := range []string{"page=1", "page=50", "pageSize=100", logFilter("actor", "equals", "admin"), logFilter("changedField", "is", "nameEn")} {
		start := time.Now()
		result := logRead(a, path, q)
		elapsed := time.Since(start)
		encoded, _ := json.Marshal(result)
		if elapsed > 2*time.Second || len(encoded) > 2*1024*1024 {
			t.Fatalf("1000-operation response budget exceeded: elapsed=%s bytes=%d", elapsed, len(encoded))
		}
		t.Logf("H1000 %s: total=%v elapsed=%s bytes=%d", q, result["total"], elapsed, len(encoded))
	}
	if logRead(a, path, logFilter("changedField", "is", "nameEn"))["total"] != float64(333) {
		t.Fatal("H1000 English oracle")
	}
	// One cold connection-pool pass followed by two warm 20-reader waves.
	f.runtime.Pool.Reset()
	var memoryBefore, memoryAfter runtime.MemStats
	runtime.ReadMemStats(&memoryBefore)
	for wave := 0; wave < 2; wave++ {
		type measurement struct {
			duration time.Duration
			status   int
			total    float64
			err      error
		}
		measurements := make(chan measurement, 20)
		startGate := make(chan struct{})
		var readers sync.WaitGroup
		for n := 0; n < 20; n++ {
			readers.Add(1)
			go func(n int) {
				defer readers.Done()
				<-startGate
				start := time.Now()
				status, result, err := a.response("GET", path+"/changes?pageSize=20&page="+fmt.Sprint(n+1), nil, "")
				total, _ := result["total"].(float64)
				measurements <- measurement{time.Since(start), status, total, err}
			}(n)
		}
		close(startGate)
		readers.Wait()
		close(measurements)
		times := []time.Duration{}
		for m := range measurements {
			if m.err != nil || m.status != 200 || m.total != 1000 {
				t.Fatalf("20-reader wave %d failed status=%d elapsed=%s error=%v", wave, m.status, m.duration, m.err)
			}
			times = append(times, m.duration)
		}
		sort.Slice(times, func(i, j int) bool { return times[i] < times[j] })
		t.Logf("H1000 20 concurrent wave=%d p50=%s p95=%s max=%s", wave, times[9], times[18], times[19])
	}
	runtime.ReadMemStats(&memoryAfter)
	t.Logf("API test process heap before=%d after=%d allocated delta=%d bytes; SQL is one source/revision stream plus fixed auth/window reads", memoryBefore.HeapAlloc, memoryAfter.HeapAlloc, memoryAfter.TotalAlloc-memoryBefore.TotalAlloc)
	if logRead(a, path, upper)["total"] != float64(45) {
		t.Fatal("pinned history gained future writes")
	}
	blocker, err := f.admin.Begin(f.ctx)
	must(t, err)
	defer blocker.Rollback(context.Background())
	_, err = blocker.Exec(f.ctx, "LOCK TABLE organization.bu_history IN ACCESS EXCLUSIVE MODE")
	must(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", f.server.URL+path+"/changes", nil)
	done := make(chan error, 1)
	go func() {
		res, e := a.client.Do(req)
		if res != nil {
			res.Body.Close()
		}
		done <- e
	}()
	deadline := time.Now().Add(time.Second)
	for f.scalar("SELECT count(*) FROM pg_stat_activity WHERE usename='chnhcm_api' AND wait_event_type='Lock'") == 0 {
		if time.Now().After(deadline) {
			t.Fatal("query did not reach lock barrier")
		}
		time.Sleep(5 * time.Millisecond)
	}
	cancelled := time.Now()
	cancel()
	if err := <-done; err == nil {
		t.Fatal("in-flight cancellation ignored")
	}
	for f.scalar("SELECT count(*) FROM pg_stat_activity WHERE usename='chnhcm_api' AND xact_start IS NOT NULL") != 0 || f.runtime.Pool.Stat().AcquiredConns() != 0 {
		if time.Since(cancelled) > 2*time.Second {
			t.Fatal("cancelled query retained transaction/connection")
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Logf("in-flight cancellation to confirmed DB transaction/connection release: %s", time.Since(cancelled))
	must(t, blocker.Rollback(f.ctx))
	t.Log("H45/H1000 pagination, stable bounds, literal actor filter, permission trimming, invalid query and latency/response-size budgets verified")
}

func TestOperationIdentityLegacyServiceAndAtomicFailure(t *testing.T) {
	f := fresh(t)
	a, _ := operationActors(f)
	// Privileged fixture represents sealed ID-only operator evidence.
	old := a.call("POST", "/api/v1/business-units", createBody("OP_LEGACY", "2026-01-01"), foundation.NewID(), 201)
	id := old["buId"].(string)
	path := "/api/v1/business-units/" + id
	f.editBU(old["sourceVersion"].(string), func(v *bootstrap.BUHistoryFact) {
		op := &v.Operation.Source.Operator
		op.DisplayName = nil
		op.Login = nil
		op.Kind = "unknown"
		op.IdentityEvidence = "subject_id_only"
		for i := range v.State.Summaries {
			v.State.Summaries[i].Operator = *op
		}
	})
	op := summaryOf(t, detail(a, id))["operator"].(map[string]any)
	if op["identityEvidence"] != "subject_id_only" || op["login"] != nil || op["displayName"] != nil || op["kind"] != "unknown" {
		t.Fatal("old subject fabricated", op)
	}
	if logRead(a, path, logFilter("actor", "equals", a.subject))["total"] != float64(1) || logRead(a, path, logFilter("actor", "equals", "admin"))["total"] != float64(0) {
		t.Fatal("legacy filter uses live identity")
	}
	_, err := f.admin.Exec(f.ctx, `UPDATE access.principal SET display_name=NULL,principal_kind='service' WHERE principal_id=$1`, a.subject)
	must(t, err)
	saved := correctOperation(a, id, "2026-01-01", map[string]any{"name": "服务操作"})
	op = summaryOf(t, detail(a, id))["operator"].(map[string]any)
	if op["displayName"] != nil || op["kind"] != "service" || op["login"] != "admin" || op["identityEvidence"] != "captured" {
		t.Fatal("service snapshot", op)
	}
	for _, field := range []string{"operator", "recordedAt", "subjectId"} {
		d := detail(a, id)
		input := correctBody(d, selected(d), map[string]any{"name": "伪造拒绝"})
		input[field] = "forged"
		a.call("POST", path+"/records/"+selected(d)["recordId"].(string)+"/correct", input, foundation.NewID(), 400)
	}
	// A failed X03 identity read must roll back every business effect. Deliberately
	// corrupt only this isolated fixture's principal category below its DB guard.
	f.exec(`ALTER TABLE access.principal DROP CONSTRAINT principal_principal_kind_check;UPDATE access.principal SET principal_kind='invalid' WHERE login='admin'`)
	before := businessSnapshot(t, f)
	d := detail(a, id)
	r := selected(d)
	a.call("POST", path+"/records/"+r["recordId"].(string)+"/correct", correctBody(d, r, map[string]any{"name": "不得落库"}), foundation.NewID(), 503)
	after := businessSnapshot(t, f)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("identity failure left partial business facts")
	}
	if summaryOf(t, detail(a, id))["operationSourceVersion"] != saved["sourceVersion"] {
		t.Fatal("failed write changed summary")
	}
	// Recorded-day boundaries use Shanghai; equal/backward clocks do not reorder.
	f.exec(`UPDATE access.principal SET principal_kind='human' WHERE login='admin'`)
	last := saved["sourceVersion"].(string)
	for n, at := range []string{"2026-11-02T15:59:59.999Z", "2026-11-02T16:00:00Z", "2026-11-03T15:59:59Z", "2026-11-03T16:00:00Z", "2026-11-02T16:00:00Z"} {
		last = appendHistoryFixture(t, f, a, id, last, fmt.Sprintf("时刻%d", n), at)
	}
	result := logRead(a, path, logFilter("recordedAt", "is", "2026-11-03"))
	if result["total"] != float64(3) || result["items"].([]any)[0].(map[string]any)["sourceVersion"] != last {
		t.Fatal("Shanghai boundary/order")
	}
	for operator, count := range map[string]int{"before": 1, "after": 3, "onOrBefore": 4, "onOrAfter": 6} {
		if logRead(a, path, logFilter("recordedAt", operator, "2026-11-03"))["total"] != float64(count) {
			t.Fatal("Shanghai date operator", operator)
		}
	}
	t.Log("legacy-only evidence, service/missing-name snapshot, transactional identity failure, Shanghai midnight and clock reversal verified")
}

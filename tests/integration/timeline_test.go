//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgconn"
	"net/http"
	"strings"
	"testing"
	"time"

	"chnhcm.local/chnhcm/db/migrations"
	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/foundation"
)

func detail(b *browser, id string) map[string]any {
	return b.call("GET", "/api/v1/business-units/"+id, nil, "", 200)
}
func addBody(d map[string]any, date string, changes map[string]any, explicitAnchor ...string) map[string]any {
	if len(explicitAnchor) > 0 {
		return map[string]any{"anchorRecordId": explicitAnchor[0], "expectedSourceVersion": d["headSourceVersion"], "effectiveDate": date, "operationType": "CHANGE", "operationReason": "NAME_ADJUSTMENT", "changes": changes}
	}
	// Fixtures select their explicit insertion gap from their known T0 dates.
	// This is request setup, not a production fallback for missing anchors.
	rows := d["timeline"].([]any)
	anchor := rows[0].(map[string]any)
	for _, raw := range rows {
		r := raw.(map[string]any)
		if r["effectiveDate"].(string) < date && r["effectiveDate"].(string) >= anchor["effectiveDate"].(string) {
			anchor = r
		}
	}
	return map[string]any{"anchorRecordId": anchor["recordId"], "expectedSourceVersion": d["headSourceVersion"], "effectiveDate": date, "operationType": "CHANGE", "operationReason": "NAME_ADJUSTMENT", "changes": changes}
}
func correctBody(d map[string]any, record map[string]any, changes map[string]any) map[string]any {
	v := map[string]any{"expectedSourceVersion": d["headSourceVersion"], "expectedRecordRevision": record["revision"]}
	if changes != nil {
		v["changes"] = changes
	}
	return v
}
func selected(d map[string]any) map[string]any { return d["selectedRecord"].(map[string]any) }
func TestTimelineJourneyAndRecovery(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	body := createBody("JOURNEY", "2026-01-01")
	body["name"] = "甲"
	body["nameEn"] = "Alpha"
	created := b.call("POST", "/api/v1/business-units", body, foundation.NewID(), 201)
	id := created["buId"].(string)
	base := "/api/v1/business-units/" + id
	for _, v := range []struct{ date, name, en string }{{"2026-06-01", "乙", "Beta"}, {"2026-10-01", "丙", "Gamma"}} {
		b.call("POST", base+"/records", addBody(detail(b, id), v.date, map[string]any{"name": v.name, "nameEn": v.en}), foundation.NewID(), 201)
	}
	before := detail(b, id)
	oldSource := before["headSourceVersion"].(string)
	input := addBody(before, "2026-03-01", map[string]any{"name": "新"})
	key := foundation.NewID()
	added := b.call("POST", base+"/records", input, key, 201)
	records := detail(b, id)["timeline"].([]any)
	first := records[0].(map[string]any)
	changed := b.call("POST", base+"/records/"+first["recordId"].(string)+"/correct", correctBody(detail(b, id), first, map[string]any{"name": "甲2"}), foundation.NewID(), 200)
	for _, v := range []struct{ date, name, en string }{{"2026-01-01", "甲2", "Alpha"}, {"2026-03-01", "新", "Alpha"}, {"2026-06-01", "乙", "Beta"}, {"2026-10-01", "丙", "Gamma"}} {
		r := b.getOnDay(v.date, base)["viewFact"].(map[string]any)
		if r["name"] != v.name || r["nameEn"] != v.en {
			t.Fatalf("snapshot %s: %v", v.date, r)
		}
	}
	old := b.getOnDay("2026-03-01", base+"?sourceVersion="+oldSource)
	if old["viewFact"].(map[string]any)["name"] != "甲" || old["viewFact"].(map[string]any)["endExclusive"] != "2026-06-01" {
		t.Fatal("original source changed")
	}
	newSource := added["sourceVersion"].(string)
	mid := b.getOnDay("2026-03-01", base+"?sourceVersion="+newSource)
	if mid["viewFact"].(map[string]any)["name"] != "新" || mid["viewFact"].(map[string]any)["nameEn"] != "Alpha" {
		t.Fatal("new exact source wrong")
	}
	// Recover the actual original result after a newer correction and API connection reset.
	f.runtime.Pool.Reset()
	must(t, f.runtime.Ready(f.ctx))
	resolved := b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
	if resolved["sourceVersion"] != newSource || resolved["currentChanged"] != true || resolved["evaluatedBusinessDate"] == nil {
		t.Fatal(resolved)
	}
	replay := b.call("POST", base+"/records", input, key, 201)
	if replay["sourceVersion"] != newSource || replay["replayed"] != true {
		t.Fatal("replay changed original result")
	}
	if f.scalar("SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.bu_history") != 5 || f.eventCount() != 5 {
		t.Fatal("duplicate sources")
	}
	logs := b.call("GET", base+"/changes", nil, "", 200)["items"].([]any)
	if directChange(t, logs[0].(map[string]any))["afterContext"].(map[string]any)["name"] != "甲2" || directChange(t, logs[1].(map[string]any))["afterContext"].(map[string]any)["name"] != "新" {
		t.Fatal("logs select wrong direct record")
	}
	// No-change must not allocate identities or facts; explicit null is a genuine change.
	d := detail(b, id)
	beforeFacts := f.revisionCount(false)
	nochange := b.call("POST", base+"/records", addBody(d, "2026-04-01", map[string]any{"name": "新"}), foundation.NewID(), 200)
	if nochange["outcome"] != "no_change" || f.revisionCount(false) != beforeFacts || f.recordCount("") != 4 {
		t.Fatal("nochange made facts")
	}
	target := d["timeline"].([]any)[1].(map[string]any)
	nochangeKey := foundation.NewID()
	nochange = b.call("POST", base+"/records/"+target["recordId"].(string)+"/correct", correctBody(d, target, map[string]any{"name": "新"}), nochangeKey, 200)
	recoveredNochange := b.call("POST", "/api/v1/organization/submissions/"+nochangeKey+"/resolve", nil, "", 200)
	if recoveredNochange["selectedRecordId"] != target["recordId"] || recoveredNochange["observedSourceVersion"] != d["headSourceVersion"] {
		t.Fatal("no-change correction lost original observed selection")
	}
	if nochange["outcome"] != "no_change" {
		t.Fatal("correction same value")
	}
	b.call("POST", base+"/records/"+target["recordId"].(string)+"/correct", correctBody(d, target, map[string]any{"nameEn": nil}), foundation.NewID(), 200)
	r := b.getOnDay("2026-03-01", base)["viewFact"].(map[string]any)
	if _, exists := r["nameEn"]; exists {
		t.Fatal("English clear not persisted")
	}
	// Keep exact original source and first correction references after later writes.
	b.call("GET", base+"?sourceVersion="+changed["sourceVersion"].(string), nil, "", 200)
}
func TestTimelineRejectionsAndCurrentPermissions(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	created := b.call("POST", "/api/v1/business-units", createBody("RULES", "2026-01-01"), foundation.NewID(), 201)
	id := created["buId"].(string)
	path := "/api/v1/business-units/" + id
	d := detail(b, id)
	for _, date := range []string{"2026-01-01", "2025-12-31", "2019-12-31"} {
		key := foundation.NewID()
		b.call("POST", path+"/records", addBody(d, date, map[string]any{"name": "X"}), key, map[bool]int{true: 409, false: 400}[date == "2026-01-01"])
		r := b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
		if r["outcome"] != "rejected" || r["failureDetails"] == nil {
			t.Fatal("lost safe rejection")
		}
	}
	for _, changes := range []map[string]any{{"state": "active"}, {"name": nil}, {"name": strings.Repeat("中", 201)}, {"name": "bad\nname"}} {
		b.call("POST", path+"/records", addBody(d, "2026-03-01", changes), foundation.NewID(), 400)
	}
	stale := addBody(d, "2026-04-01", map[string]any{"name": "stale"})
	today := b.call("GET", "/api/v1/organization/context", nil, "", 200)["currentDate"].(string)
	key := foundation.NewID()
	body := addBody(d, today, map[string]any{"name": "今日"})
	accepted := b.call("POST", path+"/records", body, key, 201)
	b.call("POST", path+"/records", stale, foundation.NewID(), 409)
	// Narrow field authority under the exact current subject guard.
	mutateLifeGrants(f, b.subject, `UPDATE access.subject_grant
		SET write_fields=ARRAY['name','effectiveDate','operationType','operationReason']
		WHERE instance_id=$1 AND principal_id=$2 AND action LIKE 'bu.%'`, f.instance, b.subject)
	b.call("POST", path+"/records", addBody(detail(b, id), "2027-06-01", map[string]any{"nameEn": "Future unit"}), foundation.NewID(), 403)
	b.call("POST", path+"/records", addBody(detail(b, id), "2027-06-01", map[string]any{"name": "未来"}), foundation.NewID(), 201)
	// CN field fixture: no English read/write, but omitted English remains a complete independent snapshot.
	mutateLifeGrants(f, b.subject, `UPDATE access.subject_grant SET read_fields=ARRAY['name']
		WHERE instance_id=$1 AND principal_id=$2 AND action LIKE 'bu.%'`, f.instance, b.subject)
	current := detail(b, id)
	first := current["timeline"].([]any)[0].(map[string]any)
	b.call("POST", path+"/records/"+first["recordId"].(string)+"/correct", correctBody(current, first, map[string]any{"name": "部分字段更正"}), foundation.NewID(), 200)
	var english string
	for _, r := range f.buHead(id).State.Timeline.Records {
		if r.ID == first["recordId"] && r.Data.NameEn != nil {
			english = *r.Data.NameEn
		}
	}
	if english != "Future unit" {
		t.Fatal("hidden English overwritten")
	}
	for _, r := range detail(b, id)["timeline"].([]any) {
		if _, ok := r.(map[string]any)["nameEn"]; ok {
			t.Fatal("CN response leaked English")
		}
	}
	// Revoke write only: original result and exact retry remain readable.
	must(t, bootstrap.ChangeAccess(f.ctx, f.admin, bootstrap.AccessChange{Instance: f.instance, Login: "admin", Operation: "revoke", Action: "bu.record.add"}))
	got := b.call("POST", path+"/records", body, key, 201)
	if got["sourceVersion"] != accepted["sourceVersion"] {
		t.Fatal("write revocation lost result")
	}
	b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
	b.call("POST", path+"/records", addBody(detail(b, id), "2028-01-01", map[string]any{"name": "Denied"}, first["recordId"].(string)), foundation.NewID(), 403)
	must(t, bootstrap.ChangeAccess(f.ctx, f.admin, bootstrap.AccessChange{Instance: f.instance, Login: "admin", Operation: "revoke", Action: "bu.history.read"}))
	b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 403)
	if f.scalar("SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.bu_history") != 4 {
		t.Fatal("rejections wrote facts")
	}
}

func TestTimelineConcurrentVersionAndResolve(t *testing.T) {
	f := fresh(t)
	a := f.login("admin", f.password)
	b := f.login("admin", f.password)
	created := a.call("POST", "/api/v1/business-units", createBody("RACE", "2026-01-01"), foundation.NewID(), 201)
	id := created["buId"].(string)
	path := "/api/v1/business-units/" + id + "/records"
	d := detail(a, id)
	ch := make(chan int, 2)
	start := make(chan struct{})
	for i, client := range []*browser{a, b} {
		go func(i int, c *browser) {
			<-start
			s, _, _ := c.response("POST", path, addBody(d, fmt.Sprintf("2026-0%d-01", i+3), map[string]any{"name": "X"}), foundation.NewID())
			ch <- s
		}(i, client)
	}
	close(start)
	x, y := <-ch, <-ch
	if x+y != 610 {
		t.Fatalf("source race %d/%d", x, y)
	}
	key := foundation.NewID()
	input := addBody(detail(a, id), "2026-08-01", map[string]any{"name": "Y"})
	go func() { s, _, _ := a.response("POST", path, input, key); ch <- s }()
	go func() {
		s, _, _ := b.response("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "")
		ch <- s
	}()
	x, y = <-ch, <-ch
	if x != 200 && y != 200 {
		t.Fatalf("resolve race %d/%d", x, y)
	}
	if f.scalar("SELECT (SELECT count(*) FROM organization.bu_history h CROSS JOIN LATERAL unnest(h.submission_refs) r WHERE h.ids[r]=$1::uuid)+(SELECT count(*) FROM organization.bu_terminal WHERE submission_id=$1)", key) != 1 {
		t.Fatal("race terminal count")
	}
	r := a.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
	if r["outcome"] != "accepted" && r["outcome"] != "not_applied" {
		t.Fatal(r)
	}
}
func TestCurrentStorageMismatchFailsClosed(t *testing.T) {
	f := fresh(t)
	empty, e := bootstrap.Open(f.ctx, f.runtime.Pool.Config().ConnString(), foundation.NewID())
	if e == nil {
		empty.Close()
		t.Fatal("uninitialized instance opened business runtime")
	}
	f.exec("UPDATE organization.storage_contract SET contract='broken'")
	if e := bootstrap.Migrate(f.ctx, f.admin, migrations.Files); e == nil {
		t.Fatal("mismatched storage migrated")
	}
	if e := f.runtime.Ready(f.ctx); e == nil {
		t.Fatal("mismatch reported ready")
	}
	// Open must reject before mounting any business handler.
	cfg := f.runtime.Pool.Config()
	runtime, e := bootstrap.Open(f.ctx, cfg.ConnString(), f.instance)
	if e == nil {
		runtime.Close()
		t.Fatal("mismatch opened runtime")
	}
}

func TestTimelineAggregateTimeoutAndBoundedCleanup(t *testing.T) {
	t.Run("ADD", func(t *testing.T) { testTimelineAggregateTimeout(t, "ADD") })
	t.Run("DELETE", func(t *testing.T) { testTimelineAggregateTimeout(t, "DELETE") })
	t.Run("REDATE", func(t *testing.T) { testTimelineAggregateTimeout(t, "REDATE") })
}
func testTimelineAggregateTimeout(t *testing.T, action string) {
	f := fresh(t)
	b := f.login("admin", f.password)
	b.client.Timeout = 18 * time.Second
	saved := b.call("POST", "/api/v1/business-units", createBody("TIMEOUT", "2026-01-01"), foundation.NewID(), 201)
	id := saved["buId"].(string)
	path := "/api/v1/business-units/" + id
	for i := 2; i <= 20; i++ {
		date := fmt.Sprintf("2026-01-%02d", i)
		b.call("POST", path+"/records", addBody(detail(b, id), date, map[string]any{"name": fmt.Sprint(i)}), foundation.NewID(), 201)
	}
	d := detail(b, id)
	key := foundation.NewID()
	before := businessSnapshot(t, f)
	// State, operation and projection are separate SQL writes. Delay each 4s
	// (<5s); cumulative work must stop at the original 10s budget.
	f.exec(`CREATE FUNCTION organization.test_slow_member() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN PERFORM pg_sleep(4); RETURN NEW; END $$`)
	for _, table := range []string{"bu_shape"} {
		f.exec("CREATE TRIGGER test_slow_member BEFORE INSERT ON organization." + table + " FOR EACH STATEMENT EXECUTE FUNCTION organization.test_slow_member()")
	}
	endpoint := path + "/records"
	input := addBody(d, "2026-02-01", map[string]any{"name": "pending"})
	if action != "ADD" {
		r := recordOn(t, d, "2026-01-10")
		input = deleteBody(d, r)
		endpoint += "/" + r["recordId"].(string) + "/delete"
		if action == "REDATE" {
			r = recordOn(t, d, "2026-01-20")
			input = redateBody(d, r, "2026-02-01")
			endpoint = redatePath(id, r)
		}
	}
	started := time.Now()
	b.call("POST", endpoint, input, key, 503)
	elapsed := time.Since(started)
	for _, table := range []string{"bu_shape"} {
		f.exec("DROP TRIGGER test_slow_member ON organization." + table)
	}
	f.exec("DROP FUNCTION organization.test_slow_member()")
	if elapsed < 9*time.Second || elapsed > 13*time.Second {
		t.Fatalf("overall timeout/cleanup=%s", elapsed)
	}
	for table, v := range before {
		current := businessSnapshot(t, f)[table]
		if current != v {
			t.Fatalf("timeout changed %s", table)
		}
	}
	// Observe lock release on a real new transaction; local return alone does not prove it.
	tx, e := f.admin.Begin(f.ctx)
	must(t, e)
	_, e = tx.Exec(f.ctx, "SET LOCAL lock_timeout='1s'")
	must(t, e)
	_, e = tx.Exec(f.ctx, "SELECT head_source_id FROM organization.business_unit WHERE bu_id=$1 FOR UPDATE", id)
	must(t, e)
	must(t, tx.Rollback(f.ctx))
	r := b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
	if r["outcome"] != "not_applied" {
		t.Fatal(r)
	}
	t.Logf("20-record cumulative SQL timeout=%s; all business tables unchanged; BU lock released; resolve=not_applied", elapsed)
}

func TestTimelineStrictProtocolAndCurrentFormat(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	saved := b.call("POST", "/api/v1/business-units", createBody("PROTOCOL", "2026-01-01"), foundation.NewID(), 201)
	id := saved["buId"].(string)
	base := "/api/v1/business-units/" + id
	d := detail(b, id)
	head := d["headSourceVersion"].(string)
	for _, raw := range []string{
		fmt.Sprintf(`{"expectedSourceVersion":%q,"effectiveDate":"2026-03-01","operationType":"CHANGE","operationReason":"NAME_ADJUSTMENT"}`, head),
		fmt.Sprintf(`{"expectedSourceVersion":%q,"effectiveDate":null,"operationType":"CHANGE","operationReason":"NAME_ADJUSTMENT","changes":{"name":"X"}}`, head),
		fmt.Sprintf(`{"expectedSourceVersion":%q,"effectiveDate":"2026-03-01","operationType":"CHANGE","operationReason":"NAME_ADJUSTMENT","changes":{"name":"X","name":"Y"}}`, head),
		fmt.Sprintf(`{"expectedSourceVersion":%q,"effectiveDate":"2026-03-01","operationType":"CHANGE","operationReason":"NAME_ADJUSTMENT","changes":{"Name":"X"}}`, head),
	} {
		req, e := http.NewRequest("POST", f.server.URL+base+"/records", strings.NewReader(raw))
		must(t, e)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", f.server.URL)
		req.Header.Set("X-CSRF-Token", b.csrf)
		req.Header.Set("Idempotency-Key", foundation.NewID())
		r, e := b.client.Do(req)
		must(t, e)
		r.Body.Close()
		if r.StatusCode != 400 {
			t.Fatalf("strict input status %d", r.StatusCode)
		}
	}
	target := selected(d)
	path := base + "/records/" + target["recordId"].(string) + "/correct"
	for _, field := range []string{"effectiveDate", "newEffectiveDate", "operationType", "operationReason", "note", "state"} {
		body := correctBody(d, target, map[string]any{"name": "X"})
		body[field] = "forbidden"
		b.call("POST", path, body, foundation.NewID(), 400)
	}
	stale := correctBody(d, target, map[string]any{"name": "X"})
	stale["expectedRecordRevision"] = foundation.NewID()
	b.call("POST", path, stale, foundation.NewID(), 409)
	// Valid identifiers can locate rejected inputs without claiming they are actual stored sources.
	badSource := addBody(d, "2026-03-01", map[string]any{"name": "X"})
	badSource["expectedSourceVersion"] = foundation.NewID()
	key := foundation.NewID()
	b.call("POST", base+"/records", badSource, key, 409)
	r := b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
	if r["outcome"] != "rejected" {
		t.Fatal(r)
	}
	// Accepted outcomes live only in the source directory. The standalone
	// terminal table cannot claim an accepted outcome, even with valid roots.
	_, e := f.admin.Exec(f.ctx, `INSERT INTO organization.bu_terminal(instance_id,enterprise_id,subject_id,submission_id,request_object_id,outcome,ids,dictionary_id,operation,envelope)
 SELECT instance_id,enterprise_id,subject_id,$1,request_object_id,'accepted',ids,dictionary_id,operation,envelope FROM organization.bu_terminal WHERE submission_id=$2`, foundation.NewID(), key)
	var pg *pgconn.PgError
	if !errors.As(e, &pg) || pg.Code != "23514" {
		t.Fatalf("accepted source ownership: %v", e)
	}
	for _, query := range []string{"UPDATE organization.business_unit SET first_record_id=first_record_id", "UPDATE organization.business_unit SET code=code", "UPDATE organization.bu_shape SET parts=parts", "DELETE FROM organization.bu_terminal"} {
		if _, e = f.runtime.Pool.Exec(f.ctx, query); e == nil {
			t.Fatal("runtime changed protected identity/history")
		}
	}
	if f.scalar("SELECT count(*) FROM information_schema.tables WHERE table_schema='organization' AND table_name IN ('source_member','record_revision','effective_record','command_result')") != 0 {
		t.Fatal("obsolete storage facade remains")
	}

}

func TestTimelineClientCancellationReleasesLocks(t *testing.T) {
	t.Run("ADD", func(t *testing.T) { testTimelineClientCancellation(t, "ADD") })
	t.Run("DELETE", func(t *testing.T) { testTimelineClientCancellation(t, "DELETE") })
	t.Run("REDATE", func(t *testing.T) { testTimelineClientCancellation(t, "REDATE") })
}
func testTimelineClientCancellation(t *testing.T, action string) {
	f := fresh(t)
	b := f.login("admin", f.password)
	saved := b.call("POST", "/api/v1/business-units", createBody("CANCEL", "2026-01-01"), foundation.NewID(), 201)
	id := saved["buId"].(string)
	if action != "ADD" {
		b.call("POST", "/api/v1/business-units/"+id+"/records", addBody(detail(b, id), "2026-03-01", map[string]any{"name": "delete target"}), foundation.NewID(), 201)
	}
	before := businessSnapshot(t, f)
	barrier, e := f.admin.Begin(f.ctx)
	must(t, e)
	defer barrier.Rollback(f.ctx)
	input := addBody(detail(b, id), "2026-03-01", map[string]any{"name": "cancelled"})
	endpoint := "/api/v1/business-units/" + id + "/records"
	if action != "ADD" {
		d := detail(b, id)
		r := recordOn(t, d, "2026-03-01")
		input = deleteBody(d, r)
		endpoint += "/" + r["recordId"].(string) + "/delete"
		if action == "REDATE" {
			input = redateBody(d, r, "2026-04-01")
			endpoint = redatePath(id, r)
		}
	}
	_, e = barrier.Exec(f.ctx, "LOCK TABLE organization.bu_shape IN ACCESS EXCLUSIVE MODE")
	must(t, e)
	raw, e := json.Marshal(input)
	must(t, e)
	ctx, cancel := context.WithCancel(f.ctx)
	defer cancel()
	key := foundation.NewID()
	req, e := http.NewRequestWithContext(ctx, "POST", f.server.URL+endpoint, bytes.NewReader(raw))
	must(t, e)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", f.server.URL)
	req.Header.Set("X-CSRF-Token", b.csrf)
	req.Header.Set("Idempotency-Key", key)
	done := make(chan error, 1)
	go func() {
		r, e := b.client.Do(req)
		if r != nil {
			r.Body.Close()
		}
		done <- e
	}()
	f.waitBlocked("chnhcm_api")
	started := time.Now()
	cancel()
	if e = <-done; !errors.Is(e, context.Canceled) {
		t.Fatalf("cancel response %v", e)
	}
	// Client cancellation can return before the server observes the disconnect.
	// Keep the shape-write barrier until the BU lock is released, otherwise a valid
	// commit can win that race and this would test response loss, not rollback.
	tx, e := f.admin.Begin(f.ctx)
	must(t, e)
	defer tx.Rollback(f.ctx)
	_, e = tx.Exec(f.ctx, "SET LOCAL lock_timeout='2s'")
	must(t, e)
	_, e = tx.Exec(f.ctx, "SELECT head_source_id FROM organization.business_unit WHERE bu_id=$1 FOR UPDATE", id)
	must(t, e)
	must(t, tx.Rollback(f.ctx))
	elapsed := time.Since(started)
	if elapsed > 3*time.Second {
		t.Fatalf("cancel/lock release exceeded bound: %s", elapsed)
	}
	must(t, barrier.Rollback(f.ctx))
	after := businessSnapshot(t, f)
	for table, v := range before {
		if after[table] != v {
			t.Fatalf("cancel changed %s", table)
		}
	}
	resolved := b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
	if resolved["outcome"] != "not_applied" {
		t.Fatal(resolved)
	}
	t.Logf("client cancellation through business write; DB BU lock released in %s; original source unchanged", elapsed)
}

func TestTimelineDamagedResultKeepsIdentityAndCurrentReadPriority(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	created := b.call("POST", "/api/v1/business-units", createBody("DECODE", "2026-01-01"), foundation.NewID(), 201)
	id := created["buId"].(string)
	key := foundation.NewID()
	body := addBody(detail(b, id), "2026-01-01", map[string]any{"name": "X"})
	path := "/api/v1/business-units/" + id + "/records"
	b.call("POST", path, body, key, 409)
	// Privileged, isolated corruption fixture. Runtime has no UPDATE on this table.
	f.editBUResult(b.subject, key, func(v *bootstrap.IntegrationBUTerminal) { v.Failure.FormatVersion = 999 })
	b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 503)
	b.call("POST", path, body, key, 503)
	must(t, bootstrap.ChangeAccess(f.ctx, f.admin, bootstrap.AccessChange{Instance: f.instance, Login: "admin", Operation: "revoke", Action: "bu.history.read"}))
	b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 403)
	if f.scalar("SELECT (SELECT count(*) FROM organization.bu_history h CROSS JOIN LATERAL unnest(h.submission_refs) r WHERE h.ids[r]=$1::uuid)+(SELECT count(*) FROM organization.bu_terminal WHERE submission_id=$1)", key) != 1 || f.scalar("SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.bu_history") != 1 {
		t.Fatal("damaged result was reopened or changed facts")
	}
}

func TestTimelineReadRejectionsWithoutPersistentDecisions(t *testing.T) {
	f := fresh(t)
	a := f.login("admin", f.password)
	reader := f.login("reader", f.readPassword)
	saved := a.call("POST", "/api/v1/business-units", createBody("READ_AUDIT", "2026-01-01"), foundation.NewID(), 201)
	id := saved["buId"].(string)
	source := saved["sourceVersion"].(string)
	before := businessSnapshot(t, f)
	securityEvents := f.scalar("SELECT count(*) FROM access.security_event")
	preview := map[string]any{"action": "ADD", "anchorRecordId": foundation.NewID(), "expectedSourceVersion": source, "effectiveDate": "2026-03-01", "operationType": "CHANGE", "operationReason": "NAME_ADJUSTMENT", "changes": map[string]any{"name": "X"}}
	reader.call("POST", "/api/v1/business-units/"+id+"/preview", preview, "", 403)
	if f.scalar("SELECT count(*) FROM access.security_event") != securityEvents {
		t.Fatal("preview denial persisted authorization")
	}
	must(t, bootstrap.ChangeAccess(f.ctx, f.admin, bootstrap.AccessChange{Instance: f.instance, Login: "admin", Operation: "revoke", Action: "bu.history.read"}))
	a.call("GET", "/api/v1/business-units/"+id+"?sourceVersion="+source, nil, "", 403)
	if f.scalar("SELECT count(*) FROM access.security_event") != securityEvents+1 {
		t.Fatal("history denial persisted authorization or lost the actual revocation event")
	}
	must(t, bootstrap.ChangeAccess(f.ctx, f.admin, bootstrap.AccessChange{Instance: f.instance, Login: "admin", Operation: "revoke", Action: "bu.read"}))
	a.call("GET", "/api/v1/business-units/"+id, nil, "", 404)
	a.call("GET", "/api/v1/business-units", nil, "", 403)
	if f.scalar("SELECT count(*) FROM access.security_event") != securityEvents+2 {
		t.Fatal("query denial persisted authorization or lost the actual revocation events")
	}
	after := businessSnapshot(t, f)
	for table, v := range before {
		if after[table] != v {
			t.Fatalf("read rejection changed business table %s", table)
		}
	}
}

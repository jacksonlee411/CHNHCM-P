//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"chnhcm.local/chnhcm/db/migrations"
	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type fixture struct {
	t                                *testing.T
	ctx                              context.Context
	admin                            *pgxpool.Pool
	runtime                          *bootstrap.Runtime
	server                           *httptest.Server
	instance, password, readPassword string
}
type browser struct {
	f             *fixture
	client        *http.Client
	csrf, subject string
}

func must(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}
func fresh(t *testing.T, options ...bootstrap.InitInput) *fixture {
	t.Helper()
	ctx := context.Background()
	// No privileged reaper sidecar: each fixture owns its container and explicitly terminates it.
	t.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
	image, e := os.ReadFile("../../deploy/local/postgres-image.txt")
	must(t, e)
	password := foundation.NewID() + foundation.NewID()
	command := []string{"postgres", "-c", "synchronous_commit=on", "-c", "timezone=UTC"}
	c, e := tc.GenericContainer(ctx, tc.GenericContainerRequest{ContainerRequest: tc.ContainerRequest{
		Image: strings.TrimSpace(string(image)), ExposedPorts: []string{"5432/tcp"},
		Env:        map[string]string{"POSTGRES_PASSWORD": password, "POSTGRES_DB": "chnhcm", "POSTGRES_INITDB_ARGS": "--encoding=UTF8 --locale=C"},
		Cmd:        command,
		WaitingFor: wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(60 * time.Second),
	}, Started: true})
	must(t, e)
	t.Cleanup(func() {
		if e := c.Terminate(context.Background()); e != nil {
			t.Error("fixture cleanup failed")
		}
	})
	host, e := c.Host(ctx)
	must(t, e)
	port, e := c.MappedPort(ctx, "5432/tcp")
	must(t, e)
	dsn := func(user, pass string) string {
		u := url.URL{Scheme: "postgres", User: url.UserPassword(user, pass), Host: host + ":" + port.Port(), Path: "/chnhcm", RawQuery: "sslmode=disable"}
		return u.String()
	}
	admin, e := pgxpool.New(ctx, dsn("postgres", password))
	must(t, e)
	t.Cleanup(admin.Close)
	must(t, bootstrap.Migrate(ctx, admin, migrations.Files))
	fixtureSQL, e := os.ReadFile("../../tests/fixtures/access/policy.sql")
	must(t, e)
	_, e = admin.Exec(ctx, string(fixtureSQL))
	must(t, e)
	apiPass := foundation.NewID()
	must(t, bootstrap.ConfigureLocalRoles(ctx, admin, apiPass))
	f := &fixture{t: t, ctx: ctx, admin: admin, instance: foundation.NewID(), password: foundation.NewID(), readPassword: foundation.NewID()}
	enterpriseName, enterpriseStart := "本地验证企业", "2020-01-01"
	if len(options) > 0 {
		enterpriseName = options[0].EnterpriseName
		enterpriseStart = options[0].Start
	}
	for _, a := range []struct {
		login, pass string
		readOnly    bool
	}{{"admin", f.password, false}, {"reader", f.readPassword, true}} {
		must(t, bootstrap.Initialize(ctx, admin, bootstrap.InitInput{DevelopmentFixture: true, Instance: f.instance, EnterpriseCode: "LOCAL", EnterpriseName: enterpriseName, Start: enterpriseStart, Login: a.login, Password: a.pass, ReadOnly: a.readOnly}))
	}
	f.runtime, e = bootstrap.Open(ctx, dsn("chnhcm_api", apiPass), f.instance)
	must(t, e)
	t.Cleanup(f.runtime.Close)
	api := &bootstrap.API{Runtime: f.runtime, InsecureLocal: true}
	f.server = httptest.NewUnstartedServer(nil)
	api.Origin = "http://" + f.server.Listener.Addr().String()
	f.server.Config.Handler = api.Handler()
	f.server.Start()
	t.Cleanup(f.server.Close)
	must(t, f.runtime.Ready(ctx))
	return f
}
func (f *fixture) login(name, password string) *browser {
	f.t.Helper()
	jar, e := cookiejar.New(nil)
	must(f.t, e)
	b := &browser{f: f, client: &http.Client{Jar: jar, Timeout: 10 * time.Second}}
	r := b.call("POST", "/api/v1/session/login", map[string]any{"login": name, "password": password}, "", 200)
	b.csrf = r["csrf"].(string)
	b.subject = r["subjectId"].(string)
	return b
}
func (b *browser) response(method, path string, body any, key string) (int, map[string]any, error) {
	var data []byte
	var e error
	if body != nil {
		data, e = json.Marshal(body)
		if e != nil {
			return 0, nil, e
		}
	}
	req, e := http.NewRequest(method, b.f.server.URL+path, bytes.NewReader(data))
	if e != nil {
		return 0, nil, e
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", b.f.server.URL)
	req.Header.Set("X-CSRF-Token", b.csrf)
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	res, e := b.client.Do(req)
	if e != nil {
		return 0, nil, e
	}
	defer res.Body.Close()
	raw, e := io.ReadAll(res.Body)
	if e != nil {
		return 0, nil, e
	}
	v := map[string]any{}
	if len(raw) > 0 {
		e = json.Unmarshal(raw, &v)
	}
	return res.StatusCode, v, e
}
func (b *browser) call(method, path string, body any, key string, status int) map[string]any {
	b.f.t.Helper()
	s, v, e := b.response(method, path, body, key)
	must(b.f.t, e)
	if s != status {
		b.f.t.Fatalf("%s %s: status %d want %d; code=%v message=%v", method, path, s, status, v["code"], v["message"])
	}
	return v
}
func createBody(code, date string) map[string]any {
	return map[string]any{"code": code, "name": "未来业务单元", "nameEn": "Future unit", "effectiveDate": date, "operationType": "CREATE", "operationReason": "NEW_ORGANIZATION"}
}
func (f *fixture) scalar(query string, args ...any) int {
	f.t.Helper()
	var n int
	must(f.t, f.admin.QueryRow(f.ctx, query, args...).Scan(&n))
	return n
}

func TestBUCreateQuerySourceHistory(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	ctx := b.call("GET", "/api/v1/organization/context", nil, "", 200)
	today := ctx["currentDate"].(string)
	now, e := time.Parse("2006-01-02", today)
	must(t, e)
	future := now.AddDate(0, 0, 2).Format("2006-01-02")
	body := createBody("FUTURE01", future)
	key := foundation.NewID()
	saved := b.call("POST", "/api/v1/business-units", body, key, 201)
	id := saved["buId"].(string)
	source := saved["sourceVersion"].(string)
	if f.scalar("SELECT count(*) FROM organization.business_unit") != 1 || len(f.history().BUs) != 1 || f.recordCount("") != 1 || f.revisionCount(false) != 1 || f.eventCount() != 1 {
		t.Fatal("create must store exactly one complete source, revision and event")
	}

	if f.scalar("SELECT coalesce(sum(head_no-delivered_no),0) FROM organization.business_unit") != 1 {
		t.Fatal("outbox must remain undelivered")
	}
	detail := b.getOnDay(today, "/api/v1/business-units/"+id)
	if detail["result"] != "M02_NO_FACT" || detail["viewFact"] != nil || detail["selectedRecord"] == nil {
		t.Fatal("future selection must not become today's fact")
	}
	list := b.getOnDay(today, "/api/v1/business-units")
	if list["count"] != float64(1) {
		t.Fatal("future first record absent from list")
	}
	for _, old := range []string{
		"knownAt=2026-10-30T00%3A00%3A00Z", "knownAt=", "knownAt=bad&sourceVersion=" + source,
		"knownAt=old;ignored=x", "sourceVersion=", "sourceVersion=%ZZ",
		"sourceVersion=" + source + "&sourceVersion=" + source,
		"selectedRecordId=", "selectedRecordId=%ZZ", "businessAt=" + today + "&businessAt=" + today,
	} {
		b.call("GET", "/api/v1/business-units/"+id+"?"+old, nil, "", 400)
	}
	for _, path := range []string{"/api/v1/business-units?q=%ZZ", "/api/v1/business-units?page=1&page=2", "/api/v1/business-units/" + id + "/changes?page=1;ignored=x"} {
		b.call("GET", path, nil, "", 400)
	}
	b.call("GET", "/api/v1/business-units/"+id+"?sourceVersion="+foundation.NewID(), nil, "", 404)
	exact := b.getOnDay(future, "/api/v1/business-units/"+id+"?sourceVersion="+source)
	if exact["result"] != "M02_FACT" || exact["recordedAt"] == nil || exact["committedAt"] != nil || exact["readOnly"] != true {
		t.Fatal("exact source must be readable immediately without capture")
	}
	replay := b.call("POST", "/api/v1/business-units", body, key, 201)
	if replay["buId"] != id || replay["replayed"] != true {
		t.Fatal("retry created another result")
	}
	resolved := b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
	if resolved["sourceVersion"] != source {
		t.Fatal("resolve lost original result")
	}
	body["name"] = "不同内容"
	b.call("POST", "/api/v1/business-units", body, key, 409)
	late := foundation.NewID()
	b.call("POST", "/api/v1/organization/submissions/"+late+"/resolve", nil, "", 200)
	b.call("POST", "/api/v1/business-units", createBody("LATE", today), late, 409)
	reader := f.login("reader", f.readPassword)
	reader.call("POST", "/api/v1/business-units", createBody("DENIED", today), foundation.NewID(), 403)
	if f.scalar("SELECT count(*) FROM organization.business_unit") != 1 {
		t.Fatal("replay/rejected attempts changed business facts")
	}
	t.Logf("PostgreSQL 18.6 real create/query/resolve/source history verified; source=%s", source)
}

func (f *fixture) exec(sql string, args ...any) {
	f.t.Helper()
	_, e := f.admin.Exec(f.ctx, sql, args...)
	must(f.t, e)
}
func (f *fixture) waitBlocked(user string) {
	f.t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if f.scalar(`SELECT count(*) FROM pg_locks l JOIN pg_stat_activity a USING(pid) WHERE NOT l.granted AND a.usename=$1`, user) > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	f.t.Fatal("expected observable database lock wait")
}
func TestBUConcurrencyRollbackAndSealedHistory(t *testing.T) {
	f := fresh(t)
	a, b := f.login("admin", f.password), f.login("admin", f.password)
	today := a.call("GET", "/api/v1/organization/context", nil, "", 200)["currentDate"].(string)
	type answer struct {
		s int
		v map[string]any
		e error
	}
	race := func(key1, key2 string, body1, body2 map[string]any) []answer {
		start := make(chan struct{})
		out := make(chan answer, 2)
		for i, client := range []*browser{a, b} {
			go func(i int, c *browser) {
				<-start
				keys := []string{key1, key2}
				bodies := []map[string]any{body1, body2}
				s, v, e := c.response("POST", "/api/v1/business-units", bodies[i], keys[i])
				out <- answer{s, v, e}
			}(i, client)
		}
		close(start)
		results := []answer{<-out, <-out}
		for _, r := range results {
			must(t, r.e)
		}
		return results
	}
	key := foundation.NewID()
	body := createBody("ONCE", today)
	same := race(key, key, body, body)
	if same[0].s != 201 || same[1].s != 201 || same[0].v["sourceVersion"] != same[1].v["sourceVersion"] {
		t.Fatalf("concurrent idempotency results: %d/%d", same[0].s, same[1].s)
	}
	collision := race(foundation.NewID(), foundation.NewID(), createBody("CaseCode", today), createBody("casecode", today))
	if collision[0].s+collision[1].s != 610 {
		t.Fatalf("case-insensitive uniqueness results: %d/%d", collision[0].s, collision[1].s)
	}
	baseline := businessSnapshot(t, f)
	f.exec(`CREATE FUNCTION organization.test_outbox_failure() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected outbox failure'; END $$; CREATE TRIGGER test_outbox_failure BEFORE INSERT ON organization.bu_history FOR EACH ROW EXECUTE FUNCTION organization.test_outbox_failure()`)
	failed := foundation.NewID()
	a.call("POST", "/api/v1/business-units", createBody("ROLLBACK", today), failed, 503)
	f.exec("DROP TRIGGER test_outbox_failure ON organization.bu_history; DROP FUNCTION organization.test_outbox_failure()")
	if !reflect.DeepEqual(baseline, businessSnapshot(t, f)) {
		t.Fatal("partial source/event/history transaction leaked")
	}

	if f.scalar("SELECT (SELECT count(*) FROM organization.bu_history h CROSS JOIN LATERAL unnest(h.submission_refs) r WHERE h.ids[r]=$1::uuid)+(SELECT count(*) FROM organization.bu_terminal WHERE submission_id=$1)", failed) != 0 {
		t.Fatal("failed transaction left a definitive command result")
	}
	resolved := a.call("POST", "/api/v1/organization/submissions/"+failed+"/resolve", nil, "", 200)
	if resolved["outcome"] != "not_applied" {
		t.Fatal("definite rollback cannot be recovered as accepted")
	}
	// Attempt to modify a prior source root without changing its directory.
	_, sealErr := f.runtime.Pool.Exec(f.ctx, `UPDATE organization.bu_history SET states[1]=states[1]+1`)
	var sealed *pgconn.PgError
	if !errors.As(sealErr, &sealed) || sealed.Code != "23514" {
		t.Fatalf("committed source was mutable: %v", sealErr)
	}
	for _, sql := range []string{
		"UPDATE organization.bu_shape SET parts=ARRAY[decode('00','hex')]",
		"DELETE FROM organization.bu_history", "TRUNCATE organization.bu_terminal CASCADE",
		"UPDATE access.security_event SET decision='allow'", "DELETE FROM access.security_event", "TRUNCATE access.security_event",
		"ALTER TABLE organization.bu_history DISABLE TRIGGER ALL",
	} {
		if _, err := f.runtime.Pool.Exec(f.ctx, sql); err == nil {
			t.Fatalf("runtime role unexpectedly allowed %s", sql)
		}
	}

}

func TestBUCurrentAuthorizationAndSession(t *testing.T) {
	f := fresh(t)
	a := f.login("admin", f.password)
	today := a.call("GET", "/api/v1/organization/context", nil, "", 200)["currentDate"].(string)
	body := createBody("SECRET_CODE", today)
	key := foundation.NewID()
	saved := a.call("POST", "/api/v1/business-units", body, key, 201)
	id := saved["buId"].(string)
	originalTerminal := f.buSource(saved["sourceVersion"].(string)).Operation.Terminal
	// A controlled fixture narrows fields while holding the same exclusive guard used by the upgrader.
	mutateLifeGrants(f, a.subject, `UPDATE access.subject_grant
		SET read_fields=ARRAY['name'], write_fields=array_remove(write_fields,'nameEn')
		WHERE instance_id=$1 AND principal_id=$2 AND action LIKE 'bu.%'`, f.instance, a.subject)
	d := a.call("GET", "/api/v1/business-units/"+id, nil, "", 200)
	raw, _ := json.Marshal(d)
	if bytes.Contains(raw, []byte("SECRET_CODE")) || bytes.Contains(raw, []byte("Future unit")) {
		t.Fatal("masked fields leaked in detail")
	}
	record := d["selectedRecord"].(map[string]any)
	if _, ok := record["effectiveDate"]; ok {
		t.Fatal("masked effectiveDate leaked in detail")
	}
	list := a.call("GET", "/api/v1/business-units", nil, "", 200)
	item := list["items"].([]any)[0].(map[string]any)
	if _, ok := item["effectiveDate"]; ok {
		t.Fatal("masked date leaked in list")
	}
	list = a.call("GET", "/api/v1/business-units?q=SECRET_CODE", nil, "", 200)
	if list["count"] != float64(0) {
		t.Fatal("search used unreadable code")
	}
	res := a.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
	if _, ok := res["effectiveDate"]; ok {
		t.Fatal("resolve leaked masked date")
	}
	a.call("POST", "/api/v1/business-units", createBody("BAD_FIELD", today), foundation.NewID(), 403)
	allowedBody := createBody("ALLOWED_FIELD", today)
	delete(allowedBody, "nameEn")
	a.call("POST", "/api/v1/business-units", allowedBody, foundation.NewID(), 201)
	must(t, bootstrap.ChangeAccess(f.ctx, f.admin, bootstrap.AccessChange{Instance: f.instance, Login: "admin", Operation: "revoke", Action: "bu.history.read"}))
	a.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 403)
	// Request body still lacks current field-write authorization; use the accepted body without English.
	secondKey := foundation.NewID()
	secondBody := createBody("HISTORY_REPLAY", today)
	delete(secondBody, "nameEn")
	a.call("POST", "/api/v1/business-units", secondBody, secondKey, 201)
	a.call("POST", "/api/v1/business-units", secondBody, secondKey, 403)
	if after := f.buSource(saved["sourceVersion"].(string)).Operation.Terminal; !reflect.DeepEqual(after, originalTerminal) {
		t.Fatal("current grant edits changed the original business result")
	}
	must(t, bootstrap.ChangeAccess(f.ctx, f.admin, bootstrap.AccessChange{Instance: f.instance, Login: "admin", Operation: "revoke", Action: "bu.read"}))
	a.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 404)
	// Relogin rotates the old cookie, and logout ends the new real server-side session.
	reader := f.login("reader", f.readPassword)
	u, _ := url.Parse(f.server.URL)
	oldCookies := reader.client.Jar.Cookies(u)
	v := reader.call("POST", "/api/v1/session/login", map[string]any{"login": "reader", "password": f.readPassword}, "", 200)
	reader.csrf = v["csrf"].(string)
	staleJar, _ := cookiejar.New(nil)
	staleJar.SetCookies(u, oldCookies)
	stale := &browser{f: f, client: &http.Client{Jar: staleJar}}
	stale.call("GET", "/api/v1/session", nil, "", 401)
	reader.call("DELETE", "/api/v1/session", nil, "", 204)
	reader.call("GET", "/api/v1/session", nil, "", 401)
	if e := bootstrap.Initialize(f.ctx, f.admin, bootstrap.InitInput{DevelopmentFixture: true, Instance: f.instance, EnterpriseCode: "LOCAL", EnterpriseName: "本地验证企业", Start: "2020-01-01", Login: "admin", Password: f.password, ReadOnly: true}); e == nil {
		t.Fatal("changed initialization intent accepted")
	}
}

func TestBURevocationOrderingAndStrictInput(t *testing.T) {
	f := fresh(t)
	a := f.login("admin", f.password)
	today := a.call("GET", "/api/v1/organization/context", nil, "", 200)["currentDate"].(string)
	tx, e := f.admin.Begin(f.ctx)
	must(t, e)
	_, e = tx.Exec(f.ctx, "SELECT policy_no FROM access.policy_guard WHERE instance_id=$1 AND principal_id=$2 FOR UPDATE", f.instance, a.subject)
	must(t, e)
	ch := make(chan int, 1)
	go func() {
		s, _, _ := a.response("POST", "/api/v1/business-units", createBody("REVOKED", today), foundation.NewID())
		ch <- s
	}()
	f.waitBlocked("chnhcm_api")
	_, e = tx.Exec(f.ctx, "DELETE FROM access.subject_grant WHERE instance_id=$1 AND principal_id=$2 AND action='bu.create'", f.instance, a.subject)
	must(t, e)
	_, e = tx.Exec(f.ctx, "UPDATE access.policy_guard SET policy_no=policy_no+1 WHERE instance_id=$1 AND principal_id=$2", f.instance, a.subject)
	must(t, e)
	must(t, tx.Commit(f.ctx))
	if status := <-ch; status != 403 {
		t.Fatalf("request after revocation returned %d", status)
	}
	if f.scalar("SELECT count(*) FROM organization.business_unit") != 0 {
		t.Fatal("revoked command wrote data")
	}
	for _, raw := range []string{`{"code":"A","code":"B"}`, `{"unexpected":true}`, `{"name":"x"} {}`, `null`, `[]`} {
		req, e := http.NewRequest("POST", f.server.URL+"/api/v1/business-units", strings.NewReader(raw))
		must(t, e)
		req.Header.Set("Origin", f.server.URL)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CSRF-Token", a.csrf)
		req.Header.Set("Idempotency-Key", foundation.NewID())
		r, e := a.client.Do(req)
		must(t, e)
		r.Body.Close()
		if r.StatusCode != 400 {
			t.Fatalf("strict input %q returned %d", raw, r.StatusCode)
		}
	}
	for _, origin := range []string{"", "https://untrusted.invalid"} {
		req, _ := http.NewRequest("DELETE", f.server.URL+"/api/v1/session", nil)
		req.Header.Set("Origin", origin)
		req.Header.Set("X-CSRF-Token", a.csrf)
		r, e := a.client.Do(req)
		must(t, e)
		r.Body.Close()
		if r.StatusCode != 403 {
			t.Fatal("invalid Origin accepted")
		}
	}
	oldCSRF := a.csrf
	a.csrf = ""
	a.call("POST", "/api/v1/organization/submissions/"+foundation.NewID()+"/resolve", nil, "", 403)
	a.csrf = oldCSRF
	must(t, bootstrap.ChangeAccess(f.ctx, f.admin, bootstrap.AccessChange{Instance: f.instance, Login: "admin", Operation: "disable"}))
	a.call("GET", "/api/v1/session", nil, "", 401)
}

func TestBUSliceLoad(t *testing.T) {
	if os.Getenv("CHNHCM_LOAD") != "1" {
		t.Skip("explicit 2000 BU capacity sample")
	}
	f := fresh(t)
	b := f.login("admin", f.password)
	today := b.call("GET", "/api/v1/organization/context", nil, "", 200)["currentDate"].(string)
	clients := []*browser{b}
	for i := 1; i < 20; i++ {
		clients = append(clients, f.login("admin", f.password))
	}
	start := time.Now()
	var latencyMu sync.Mutex
	var writeLatency []time.Duration
	var last map[string]any
	var wg sync.WaitGroup
	errCh := make(chan error, 20)
	for i, c := range clients {
		wg.Add(1)
		go func(worker int, b *browser) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				begin := time.Now()
				s, result, e := b.response("POST", "/api/v1/business-units", createBody(fmt.Sprintf("LOAD_%02d_%03d", worker, j), today), foundation.NewID())
				latencyMu.Lock()
				writeLatency = append(writeLatency, time.Since(begin))
				last = result
				latencyMu.Unlock()
				if e != nil || s != 201 {
					errCh <- fmt.Errorf("load status %d; transport error %t", s, e != nil)
					return
				}
			}
		}(i, c)
	}
	wg.Wait()
	close(errCh)
	for e := range errCh {
		t.Error(e)
	}
	if f.scalar("SELECT count(*) FROM organization.business_unit") != 2000 {
		t.Fatal("load sample incomplete")
	}
	t.Logf("2000 BUs / 20 real sessions / one source each: write elapsed=%s", time.Since(start))
	percentile := func(values []time.Duration, p int) time.Duration {
		sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
		return values[(len(values)-1)*p/100]
	}
	t.Logf("HTTP create milliseconds: p50=%d p95=%d p99=%d", percentile(writeLatency, 50).Milliseconds(), percentile(writeLatency, 95).Milliseconds(), percentile(writeLatency, 99).Milliseconds())
	for _, kind := range []string{"list", "detail", "sourceVersion"} {
		latencies := []time.Duration{}
		endpoint := "/api/v1/business-units?pageSize=100"
		if kind != "list" {
			endpoint = "/api/v1/business-units/" + last["buId"].(string)
		}
		if kind == "sourceVersion" {
			endpoint += "?sourceVersion=" + last["sourceVersion"].(string)
		}
		for i := 0; i < 5; i++ {
			for _, client := range clients {
				wg.Add(1)
				go func(b *browser) {
					defer wg.Done()
					begin := time.Now()
					status, _, err := b.response("GET", endpoint, nil, "")
					latencyMu.Lock()
					latencies = append(latencies, time.Since(begin))
					latencyMu.Unlock()
					if err != nil || status != 200 {
						t.Errorf("%s status=%d", kind, status)
					}
				}(client)
			}
			wg.Wait()
		}
		t.Logf("HTTP %s 100 samples/20 concurrent: p50=%dms p95=%dms p99=%dms", kind, percentile(latencies, 50).Milliseconds(), percentile(latencies, 95).Milliseconds(), percentile(latencies, 99).Milliseconds())
	}
	var size int64
	must(t, f.admin.QueryRow(f.ctx, "SELECT pg_database_size(current_database())").Scan(&size))
	t.Logf("database bytes after sample=%d", size)
	list := b.call("GET", "/api/v1/business-units?page=999&pageSize=100", nil, "", 200)
	if list["page"] != float64(20) || list["count"] != float64(2000) {
		t.Fatal("page clamping/count inconsistent")
	}
}

func TestBUCommitVisibilityAndRevocationOrdering(t *testing.T) {
	f := fresh(t)
	a := f.login("admin", f.password)
	today := a.call("GET", "/api/v1/organization/context", nil, "", 200)["currentDate"].(string)
	// Hold only the test-owned outbox table. Source inserts are uncommitted until this barrier releases.
	barrier, e := f.admin.Begin(f.ctx)
	must(t, e)
	defer barrier.Rollback(f.ctx)
	_, e = barrier.Exec(f.ctx, "LOCK TABLE organization.bu_shape IN ACCESS EXCLUSIVE MODE")
	must(t, e)
	type result struct {
		status int
		body   map[string]any
		err    error
	}
	ch := make(chan result, 1)
	go func() {
		s, v, e := a.response("POST", "/api/v1/business-units", createBody("COMMIT_BARRIER", today), foundation.NewID())
		ch <- result{s, v, e}
	}()
	f.waitBlocked("chnhcm_api")
	if f.scalar("SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.bu_history") != 0 {
		t.Fatal("uncommitted source became visible")
	}
	// A revoke started behind the request's shared policy guard cannot overtake that request.
	revokeDone := make(chan error, 1)
	go func() {
		revokeDone <- bootstrap.ChangeAccess(f.ctx, f.admin, bootstrap.AccessChange{Instance: f.instance, Login: "admin", Operation: "revoke", Action: "bu.create"})
	}()
	f.waitBlocked("postgres")
	must(t, barrier.Commit(f.ctx))
	accepted := <-ch
	must(t, accepted.err)
	if accepted.status != 201 {
		t.Fatalf("request already holding shared guard should finish: %d", accepted.status)
	}
	must(t, <-revokeDone)
	id := accepted.body["buId"].(string)
	a.call("GET", "/api/v1/business-units/"+id+"?sourceVersion="+accepted.body["sourceVersion"].(string), nil, "", 200)
	a.call("POST", "/api/v1/business-units", createBody("AFTER_REVOKE", today), foundation.NewID(), 403)
	f.runtime.Pool.Reset()
	must(t, f.runtime.Ready(f.ctx))
	a.call("GET", "/api/v1/business-units/"+id, nil, "", 200)
}

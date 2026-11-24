//go:build integration

// Composition-root integration probes may wire task-owned adapters while business
// journey tests continue to consume only public contracts under tests/integration.
package bootstrap_test

import (
	"bytes"
	"chnhcm.local/chnhcm/db/migrations"
	"chnhcm.local/chnhcm/internal/bootstrap"
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/application"
	orgdb "chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

type fixture struct {
	t                                *testing.T
	ctx                              context.Context
	admin                            *pgxpool.Pool
	runtime                          *bootstrap.Runtime
	server                           *httptest.Server
	container                        tc.Container
	instance, password, readPassword string
}
type browser struct {
	f                  *fixture
	client             *http.Client
	csrf, subject      string
	jobFunctionFixture string
}

func must(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}
func fresh(t *testing.T) *fixture {
	t.Helper()
	return freshWithEnterpriseStart(t, "2020-01-01")
}

func freshWithEnterpriseStart(t *testing.T, start string) *fixture {
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
	apiPass := foundation.NewID()
	must(t, bootstrap.ConfigureLocalRoles(ctx, admin, apiPass))
	f := &fixture{t: t, ctx: ctx, admin: admin, container: c, instance: foundation.NewID(), password: foundation.NewID(), readPassword: foundation.NewID()}
	for _, a := range []struct {
		login, pass string
		readOnly    bool
	}{{"admin", f.password, false}, {"reader", f.readPassword, true}} {
		must(t, bootstrap.Initialize(ctx, admin, bootstrap.InitInput{Instance: f.instance, EnterpriseCode: "LOCAL", EnterpriseName: "本地验证企业", Start: start, Login: a.login, Password: a.pass, ReadOnly: a.readOnly}))
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

func (f *fixture) exec(sql string, args ...any) {
	f.t.Helper()
	_, e := f.admin.Exec(f.ctx, sql, args...)
	must(f.t, e)
}
func businessSnapshot(t *testing.T, f *fixture) map[string]string {
	t.Helper()
	return logicalBusinessSnapshot(f)
}

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

// A task-owned adapter injects faults after real business writes, without adding production hooks.
type faultStore struct {
	application.Store
	failure error
}
type faultUnit struct {
	application.Unit
	failure error
}

func (u faultUnit) Persist(ctx context.Context, w application.Write) (org.Result, error) {
	r, e := u.Unit.Persist(ctx, w)
	if e != nil {
		return r, e
	}
	return r, u.failure
}
func (s faultStore) WithinWrite(ctx context.Context, c access.Credentials, fn func(application.Unit) error) error {
	return s.Store.WithinWrite(ctx, c, func(u application.Unit) error { return fn(faultUnit{u, s.failure}) })
}
func TestTimelineWholeSavepointAndResolveRaces(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	created := b.call("POST", "/api/v1/business-units", createBody("ATOMIC", "2026-01-01"), foundation.NewID(), 201)
	id := created["buId"].(string)
	path := "/api/v1/business-units/" + id + "/records"
	svc := f.runtime.Organization.(application.Service)
	original := svc.Store
	input := addBody(detail(b, id), "2026-03-01", map[string]any{"name": "change"})
	before := businessSnapshot(t, f)
	key := foundation.NewID()
	svc.Store = faultStore{original, foundation.Fail(409, "M02_INJECTED_CONFLICT", "受控冲突")}
	f.runtime.Organization = svc
	b.call("POST", path, input, key, 409)
	if f.recordCount("") != 1 || f.scalar("SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.bu_history") != 1 {
		t.Fatal("409 committed orphan identity or partial source")
	}
	after := businessSnapshot(t, f)
	for _, table := range []string{"business_unit", "sources"} {
		if before[table] != after[table] {
			t.Fatalf("savepoint changed %s", table)
		}
	}
	svc.Store = original
	f.runtime.Organization = svc
	r := b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
	if r["outcome"] != "rejected" {
		t.Fatal("rejection missing")
	}
	b.call("POST", path, input, key, 400) // Original safe rejection persists, never reexecutes.
	for _, resolveFirst := range []bool{false, true} {
		unknown := foundation.NewID()
		svc.Store = faultStore{original, context.DeadlineExceeded}
		f.runtime.Organization = svc
		b.call("POST", path, input, unknown, 503)
		if f.scalar("SELECT (SELECT count(*) FROM organization.bu_history h CROSS JOIN LATERAL unnest(h.submission_refs) r WHERE h.ids[r]=$1::uuid)+(SELECT count(*) FROM organization.bu_terminal WHERE submission_id=$1)", unknown) != 0 {
			t.Fatal("outer rollback wrote rejection in another transaction")
		}
		svc.Store = original
		f.runtime.Organization = svc
		if resolveFirst {
			b.call("POST", "/api/v1/organization/submissions/"+unknown+"/resolve", nil, "", 200)
			b.call("POST", path, input, unknown, 409)
		} else {
			b.call("POST", path, input, unknown, 201)
			b.call("POST", "/api/v1/organization/submissions/"+unknown+"/resolve", nil, "", 200)
			input = addBody(detail(b, id), "2026-04-01", map[string]any{"name": "second"})
		}
		if f.scalar("SELECT (SELECT count(*) FROM organization.bu_history h CROSS JOIN LATERAL unnest(h.submission_refs) r WHERE h.ids[r]=$1::uuid)+(SELECT count(*) FROM organization.bu_terminal WHERE submission_id=$1)", unknown) != 1 {
			t.Fatal("not a unique terminal")
		}
	}
}
func TestTimelineOriginalBusinessDateAcrossMidnight(t *testing.T) {
	for _, action := range []string{"ADD", "DEACTIVATE", "REACTIVATE"} {
		t.Run(action, func(t *testing.T) {
			f := fresh(t)
			b := f.login("admin", f.password)
			saved := b.call("POST", "/api/v1/business-units", createBody("MIDNIGHT", "2026-01-01"), foundation.NewID(), 201)
			id := saved["buId"].(string)
			path := "/api/v1/business-units/" + id
			expectedState := "active"
			if action == "REACTIVATE" {
				b.call("POST", path+"/deactivate", map[string]any{"anchorRecordId": lifecycleAnchor(detail(b, id)), "expectedSourceVersion": detail(b, id)["headSourceVersion"], "effectiveDate": "2026-06-01", "operationType": "DEACTIVATE", "operationReason": "BUSINESS_CLOSURE"}, foundation.NewID(), 201)
				expectedState = "inactive"
			}
			// Injectable database Clock adapter, production still executes SELECT clock_timestamp().
			svc := f.runtime.Organization.(application.Service)
			store := svc.Store.(*orgdb.Store)
			before, _ := time.Parse(time.RFC3339, "2026-06-14T15:59:59Z")
			store.Clock = func(context.Context, pgx.Tx) (time.Time, error) { return before, nil }
			restoreRecorded := f.recordedAt("2026-06-14T16:00:01Z")
			key := foundation.NewID()
			input := addBody(detail(b, id), "2026-06-15", map[string]any{"name": "跨日"})
			endpoint := path + "/records"
			if action != "ADD" {
				reason := "BUSINESS_CLOSURE"
				if action == "REACTIVATE" {
					reason = "BUSINESS_RESUMPTION"
				}
				input = map[string]any{"anchorRecordId": lifecycleAnchor(detail(b, id)), "expectedSourceVersion": detail(b, id)["headSourceVersion"], "effectiveDate": "2026-06-15", "operationType": action, "operationReason": reason}
				endpoint = path + "/" + strings.ToLower(action)
			}
			result := b.call("POST", endpoint, input, key, 201)
			if result["evaluatedBusinessDate"] != "2026-06-14" || result["currentStatus"] != expectedState {
				t.Fatal("evaluated business date/state changed", result)
			}
			restoreRecorded()
			store.Clock = nil
			recovered := b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
			if recovered["evaluatedBusinessDate"] != "2026-06-14" || recovered["evaluatedAt"] != "2026-06-14T15:59:59Z" || recovered["currentStatus"] != expectedState {
				t.Fatal("recovery relabeled original day/state", recovered)
			}
			source := b.call("GET", path+"?sourceVersion="+result["sourceVersion"].(string), nil, "", 200)
			if source["recordedAt"] != "2026-06-14T16:00:01Z" {
				t.Fatal("actual recording timestamp lost")
			}
		})
	}
}

//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"testing"
	"time"

	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/foundation"
)

func TestT07FirstLoginRequiresSameOriginWithoutCSRF(t *testing.T) {
	f := fresh(t)
	raw, e := json.Marshal(map[string]any{"login": "admin", "password": f.password})
	must(t, e)
	for _, origin := range []string{"", "https://foreign.invalid"} {
		req, e := http.NewRequest("POST", f.server.URL+"/api/v1/session/login", bytes.NewReader(raw))
		must(t, e)
		req.Header.Set("Content-Type", "application/json")
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		res, e := http.DefaultClient.Do(req)
		must(t, e)
		res.Body.Close()
		if res.StatusCode != 403 || f.scalar("SELECT count(*) FROM access.session") != 0 {
			t.Fatal("first login accepted missing or foreign Origin", res.StatusCode)
		}
	}
	b := f.login("admin", f.password) // No cookie or CSRF exists before this request.
	b.call("GET", "/api/v1/session", nil, "", 200)
	if f.scalar("SELECT count(*) FROM access.session") != 1 {
		t.Fatal("same-origin first login failed")
	}
}

func TestT07AuthenticationAndRevocationLockOrdering(t *testing.T) {
	f := fresh(t)
	for _, action := range []string{"login", "visit"} {
		for _, change := range []string{"disable", "credential_fixture"} {
			for _, first := range []string{"request", "change"} {
				t.Run(action+"_"+change+"_"+first, func(t *testing.T) {
					login := "t07_" + foundation.NewID()[:8]
					password := foundation.NewID()
					must(t, bootstrap.Initialize(f.ctx, f.admin, bootstrap.InitInput{Instance: f.instance, EnterpriseCode: "LOCAL", EnterpriseName: "本地验证企业", Start: "2020-01-01", Login: login, Password: password}))
					b := f.login(login, password)
					subject := b.subject
					ctx, cancel := context.WithTimeout(f.ctx, 6*time.Second)
					defer cancel()
					barrier, e := f.admin.Begin(ctx)
					must(t, e)
					defer barrier.Rollback(f.ctx)
					_, e = barrier.Exec(ctx, "SELECT FROM access.policy_guard WHERE instance_id=$1 AND principal_id=$2 FOR UPDATE", f.instance, subject)
					must(t, e)
					type reply struct {
						status int
						err    error
					}
					request := make(chan reply, 1)
					changed := make(chan error, 1)
					startRequest := func() {
						go func() {
							method, path, in := "GET", "/api/v1/session", any(nil)
							if action == "login" {
								method = "POST"
								path = "/api/v1/session/login"
								in = map[string]any{"login": login, "password": password}
							}
							s, _, e := b.response(method, path, in, "")
							request <- reply{s, e}
						}()
					}
					startChange := func() {
						go func() {
							if change == "disable" {
								changed <- bootstrap.ChangeAccess(ctx, f.admin, bootstrap.AccessChange{Instance: f.instance, Login: login, Operation: "disable"})
								return
							}
							// Current session protocol under a controlled credential-change fixture.
							// This is not a claim that a password-management business UI is implemented.
							tx, e := f.admin.Begin(ctx)
							if e != nil {
								changed <- e
								return
							}
							defer tx.Rollback(f.ctx)
							_, e = tx.Exec(ctx, "SELECT FROM access.policy_guard WHERE instance_id=$1 AND principal_id=$2 FOR UPDATE", f.instance, subject)
							if e == nil {
								_, e = tx.Exec(ctx, `UPDATE access.credential SET password_hash=decode(repeat('00',32),'hex') WHERE instance_id=$1 AND principal_id=$2`, f.instance, subject)
							}
							if e == nil {
								_, e = tx.Exec(ctx, `UPDATE access.session SET revoked_at=clock_timestamp() WHERE instance_id=$1 AND principal_id=$2;`, f.instance, subject)
							}
							if e == nil {
								_, e = tx.Exec(ctx, `UPDATE access.policy_guard SET policy_no=policy_no+1 WHERE instance_id=$1 AND principal_id=$2`, f.instance, subject)
							}
							if e == nil {
								e = tx.Commit(ctx)
							}
							changed <- e
						}()
					}
					waiters := func(n int) {
						t.Helper()
						until := time.Now().Add(time.Second)
						for time.Now().Before(until) {
							if f.scalar(`SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND cardinality(pg_blocking_pids(pid))>0`) >= n {
								return
							}
							time.Sleep(3 * time.Millisecond)
						}
						t.Fatalf("did not observe %d real policy lock waiters", n)
					}
					if first == "request" {
						startRequest()
						waiters(1)
						startChange()
					} else {
						startChange()
						waiters(1)
						startRequest()
					}
					waiters(2)
					must(t, barrier.Commit(ctx))
					res := <-request
					must(t, res.err)
					must(t, <-changed)
					want := 401
					if first == "request" {
						want = 200
					}
					if res.status != want {
						t.Fatal("wrong locked order", res.status, want)
					}
					b.call("GET", "/api/v1/session", nil, "", 401)
					if f.scalar("SELECT count(*) FROM access.session WHERE instance_id=$1 AND principal_id=$2 AND revoked_at IS NULL", f.instance, subject) != 0 {
						t.Fatal("revocation missed a racing new session")
					}
					t.Log("two actual PostgreSQL policy-lock waiters; ordering and all issued sessions verified")
				})
			}
		}
	}
}
func TestT07SessionTouchDoesNotExtendAbsoluteExpiry(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	var absolute time.Time
	must(t, f.admin.QueryRow(f.ctx, `UPDATE access.session SET absolute_expires_at=clock_timestamp()+interval '2 seconds',idle_expires_at=clock_timestamp()+interval '1 second' WHERE principal_id=$1 RETURNING absolute_expires_at`, b.subject).Scan(&absolute))
	for i := 0; i < 3; i++ {
		b.call("GET", "/api/v1/session", nil, "", 200)
		var got time.Time
		must(t, f.admin.QueryRow(f.ctx, "SELECT absolute_expires_at FROM access.session WHERE principal_id=$1", b.subject).Scan(&got))
		if !got.Equal(absolute) {
			t.Fatal("touch extended absolute expiry")
		}
	}
	// The deadline belongs to PostgreSQL's clock; the host may be adjusted separately.
	_, e := f.admin.Exec(f.ctx, `SELECT pg_sleep(GREATEST(EXTRACT(EPOCH FROM ($1::timestamptz-clock_timestamp())),0)+0.1)`, absolute)
	must(t, e)
	b.call("GET", "/api/v1/session", nil, "", 401)
	// A new credential verification remains possible: expiry did not disable the account.
	jar, e := cookiejar.New(nil)
	must(t, e)
	other := &browser{f: f, client: &http.Client{Jar: jar, Timeout: 10 * time.Second}}
	got := other.call("POST", "/api/v1/session/login", map[string]any{"login": "admin", "password": f.password}, "", 200)
	if got["subjectId"] != b.subject {
		t.Fatal(fmt.Sprint(got["subjectId"]))
	}
}

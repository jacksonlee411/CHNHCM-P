//go:build integration

package bootstrap_test

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/application"
	department "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	orgdb "chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestSameSessionRequestsProceedWhileBusinessReadIsOpen(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	u, e := url.Parse(f.server.URL)
	must(t, e)
	c := access.Credentials{Token: b.client.Jar.Cookies(u)[0].Value}
	store := f.runtime.Organization.(application.Service).Store.(*orgdb.Store)
	for _, kind := range []string{"BU", "department"} {
		t.Run(kind, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(f.ctx, 8*time.Second)
			defer cancel()
			entered, release := make(chan struct{}), make(chan struct{})
			done := make(chan error, 1)
			pause := func() error {
				close(entered)
				select {
				case <-release:
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			go func() {
				if kind == "BU" {
					done <- store.Within(ctx, c, func(application.Unit) error { return pause() })
				} else {
					done <- store.WithinDepartment(ctx, c, func(department.Unit) error { return pause() })
				}
			}()
			select {
			case <-entered:
			case <-ctx.Done():
				t.Fatal("first request did not acquire its authorization guard")
			}
			// No second session, lowered lock budget, or timing sleep: the second
			// requests must finish while the first remains inside business work.
			for _, route := range []string{"/api/v1/departments/context", "/api/v1/departments", "/api/v1/business-units", "/api/v1/session"} {
				status, _, err := b.response("GET", route, nil, "")
				must(t, err)
				if status != 200 {
					t.Fatalf("same-session %s blocked behind business read: %d", route, status)
				}
			}
			var lastSeen time.Time
			must(t, f.admin.QueryRow(ctx, "SELECT last_seen_at FROM access.session WHERE principal_id=$1", b.subject).Scan(&lastSeen))
			// Session-row concurrency does not release the policy guard: revocation
			// must still serialize with the in-flight business transaction.
			tx, err := f.admin.Begin(ctx)
			must(t, err)
			_, err = tx.Exec(ctx, "SELECT FROM access.policy_guard WHERE principal_id=$1 FOR UPDATE NOWAIT", b.subject)
			var pg *pgconn.PgError
			if !errors.As(err, &pg) || pg.Code != "55P03" {
				t.Fatal("open read failed to protect current policy", err)
			}
			must(t, tx.Rollback(ctx))
			close(release)
			must(t, <-done)
			var after time.Time
			must(t, f.admin.QueryRow(ctx, "SELECT last_seen_at FROM access.session WHERE principal_id=$1", b.subject).Scan(&after))
			if !after.Equal(lastSeen) {
				t.Fatal("older request completion moved session activity backwards")
			}
			tx, err = f.admin.Begin(ctx)
			must(t, err)
			_, err = tx.Exec(ctx, "SELECT FROM access.policy_guard WHERE principal_id=$1 FOR UPDATE NOWAIT", b.subject)
			must(t, err)
			must(t, tx.Rollback(ctx))
		})
	}
}

type failingActivityGuard struct{ access.Guard }

func (g failingActivityGuard) RecordSessionActivity(context.Context) error {
	return errors.New("controlled session activity persistence failure")
}

func TestSessionActivityFailureRollsBackBusinessAndWithholdsResponse(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	service := f.runtime.Organization.(application.Service)
	store := *service.Store.(*orgdb.Store)
	guard := store.Guard
	store.Guard = func(ctx context.Context, tx pgx.Tx, c access.Credentials) (access.Guard, error) {
		g, err := guard(ctx, tx, c)
		return failingActivityGuard{g}, err
	}
	service.Store = &store
	f.runtime.Organization = service
	for _, mode := range []string{"write", "read", "read-denial"} {
		if mode == "read-denial" {
			f.exec("DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='bu.read'", b.subject)
		}
		before := f.scalar("SELECT count(*) FROM access.security_event")
		method, route, body := "POST", "/api/v1/business-units", any(createBody("ACTIVITY_FAIL", "2026-01-01"))
		if mode != "write" {
			method, body = "GET", nil
		}
		status, value, err := b.response(method, route, body, foundation.NewID())
		must(t, err)
		if status != 503 || value["outcome"] != nil || value["items"] != nil {
			t.Fatal("failed activity persistence exposed provisional result", status)
		}
		if f.scalar("SELECT count(*) FROM organization.business_unit") != 0 || f.scalar("SELECT count(*) FROM access.security_event") != before {
			t.Fatal("session activity failure committed business facts or an unexpected security event")
		}
	}
}

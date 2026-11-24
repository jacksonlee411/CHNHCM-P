//go:build integration

package bootstrap_test

import (
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	"context"
	"testing"
	"time"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/application"
	orgdb "chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
)

type t07AfterPersistStore struct {
	application.Store
	after func() error
}
type t07AfterPersistUnit struct {
	application.Unit
	after func() error
}

func (u t07AfterPersistUnit) Persist(ctx context.Context, w application.Write) (org.Result, error) {
	r, e := u.Unit.Persist(ctx, w)
	if e != nil {
		return r, e
	}
	return r, u.after()
}
func (s t07AfterPersistStore) WithinWrite(ctx context.Context, c access.Credentials, fn func(application.Unit) error) error {
	return s.Store.WithinWrite(ctx, c, func(u application.Unit) error { return fn(t07AfterPersistUnit{u, s.after}) })
}
func TestT07PanicCancellationAndFailedSavepointCleanup(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	for _, mode := range []string{"panic_after_revisions", "cancel_after_outbox", "rollback_to_missing_savepoint", "connection_lost_before_rollback"} {
		t.Run(mode, func(t *testing.T) {
			endpoint, input, _ := prepareCommitCommand(b, "T07_CLEAN_"+foundation.NewID()[:8], "LIFECYCLE_PAIR")
			svc := f.runtime.Organization.(application.Service)
			original := svc.Store
			copyStore := *original.(*orgdb.Store)
			var tx pgx.Tx
			var cancel context.CancelFunc
			guard := copyStore.Guard
			copyStore.Guard = func(ctx context.Context, current pgx.Tx, c access.Credentials) (access.Guard, error) {
				tx = current
				return guard(ctx, current, c)
			}
			observed := false
			wrapped := t07AfterPersistStore{Store: &copyStore, after: func() error {
				var object string
				var number int32
				if e := tx.QueryRow(context.Background(), `SELECT object_id::text,chunk_no FROM organization.bu_history WHERE written_xid=pg_current_xact_id() ORDER BY chunk_no DESC LIMIT 1`).Scan(&object, &number); e != nil {
					return e
				}
				chunk, e := history.LoadChunk(context.Background(), tx, history.BusinessUnit, f.instance, object, number)
				if e != nil {
					return e
				}
				raw, e := history.LoadShape(context.Background(), tx, history.BusinessUnit, f.instance, chunk.Enterprise, chunk.Operations[len(chunk.Operations)-1])
				if e != nil {
					return e
				}
				op, e := history.DecodeBUOperation(context.Background(), chunk.Values, raw)
				if e != nil {
					return e
				}
				if len(op.Revisions) < 1 || op.EventID == "" {
					panic("fixture did not reach complete business writes")
				}

				observed = true
				switch mode {
				case "panic_after_revisions":
					panic("task-owned panic after real writes")
				case "cancel_after_outbox":
					cancel()
					return context.Canceled
				case "rollback_to_missing_savepoint":
					if _, e := tx.Exec(context.Background(), "RELEASE SAVEPOINT sp_1"); e != nil {
						return e
					}
				case "connection_lost_before_rollback":
					_ = tx.Conn().PgConn().Conn().Close()
				}
				return foundation.Fail(409, "M02_SOURCE_CONFLICT", "受控业务拒绝")
			}}
			if mode == "cancel_after_outbox" {
				svc.Store = t07CancelableStore{Store: wrapped, cancel: &cancel}
			} else {
				svc.Store = wrapped
			}
			f.runtime.Organization = svc
			before := businessSnapshot(t, f)
			key := foundation.NewID()
			start := time.Now()
			status, out, e := b.response("POST", endpoint, input, key)
			svc.Store = original
			f.runtime.Organization = svc
			if !observed {
				t.Fatal("write barrier not reached")
			}
			if time.Since(start) > 3*time.Second {
				t.Fatal("cleanup unbounded", time.Since(start))
			}
			if mode == "panic_after_revisions" {
				if e == nil {
					t.Fatal("panic unexpectedly returned confirmed HTTP", status, out)
				}
			} else {
				must(t, e)
				if status != 503 || out["outcome"] != nil {
					t.Fatal("failed cleanup leaked terminal", status, out)
				}
			}
			for table, n := range businessSnapshot(t, f) {
				if n != before[table] {
					t.Fatal("exception left partial fact/terminal", table)
				}
			}
			resolved := b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
			if resolved["outcome"] != "not_applied" {
				t.Fatal(resolved)
			}
			b.call("POST", endpoint, input, key, 409)
			t.Log("actual revision/outbox writes rolled back; cleanup bounded; original identity closed, no background terminal writer")
		})
	}
}

type t07CancelableStore struct {
	application.Store
	cancel *context.CancelFunc
}

func (s t07CancelableStore) WithinWrite(ctx context.Context, c access.Credentials, fn func(application.Unit) error) error {
	inner, cancel := context.WithCancel(ctx)
	defer cancel()
	*s.cancel = cancel
	return s.Store.WithinWrite(inner, c, fn)
}

func TestT07BusinessCommitDoesNotWriteAuthorizationEvents(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	for _, deferred := range []bool{false, true} {
		t.Run(map[bool]string{false: "security_event_insert", true: "security_event_deferred"}[deferred], func(t *testing.T) {
			endpoint, input, _ := prepareCommitCommand(b, "T07_AUTH_"+foundation.NewID()[:8], "LIFECYCLE_PAIR")
			before := f.scalar("SELECT count(*) FROM access.security_event")
			key := foundation.NewID()
			f.exec(`CREATE FUNCTION access.t07_deny_audit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'controlled security event failure'; END $$`)
			if deferred {
				f.exec(`CREATE CONSTRAINT TRIGGER t07_deny_audit AFTER INSERT ON access.security_event DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION access.t07_deny_audit()`)
			} else {
				f.exec(`CREATE TRIGGER t07_deny_audit BEFORE INSERT ON access.security_event FOR EACH ROW EXECUTE FUNCTION access.t07_deny_audit()`)
			}
			status, out, e := b.response("POST", endpoint, input, key)
			f.exec(`DROP TRIGGER t07_deny_audit ON access.security_event; DROP FUNCTION access.t07_deny_audit()`)
			must(t, e)
			if status < 200 || status >= 300 || out["outcome"] != "accepted" {
				t.Fatal("business command depended on authorization persistence", status, out)
			}
			if f.scalar("SELECT count(*) FROM access.security_event") != before {
				t.Fatal("business authorization generated a security event")
			}
			if b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)["outcome"] != "accepted" {
				t.Fatal("original result recovery failed")
			}

		})
	}
}

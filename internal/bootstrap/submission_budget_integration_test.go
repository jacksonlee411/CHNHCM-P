//go:build integration

package bootstrap_test

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	buapp "chnhcm.local/chnhcm/internal/contexts/organization/application"
	depapp "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	orgdb "chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"testing"
	"time"
)

func TestSubmissionBudgetWithoutDatabaseLockTimer(t *testing.T) {
	for _, kind := range []string{"bu", "department"} {
		t.Run(kind, func(t *testing.T) {
			f := fresh(t)
			b := f.login("admin", f.password)
			id := foundation.NewID()
			bu := f.runtime.Organization.(buapp.Service)
			dep := f.runtime.Department.(depapp.Service)
			originalBU, originalDep := bu.Store, dep.Store
			store := *originalDep.(*orgdb.Store)
			guard := store.Guard
			store.Guard = func(ctx context.Context, tx pgx.Tx, c access.Credentials) (access.Guard, error) {
				g, e := guard(ctx, tx, c)
				if e == nil {
					_, e = tx.Exec(ctx, "SET LOCAL lock_timeout='0'")
				}
				return g, e
			}
			bu.Store, dep.Store = &store, &store
			f.runtime.Organization, f.runtime.Department = bu, dep
			defer func() {
				bu.Store, dep.Store = originalBU, originalDep
				f.runtime.Organization, f.runtime.Department = bu, dep
			}()
			// Namespace bytes are independent of the implementation being checked.
			parts := []string{f.instance, b.subject, id}
			route := "/api/v1/organization/submissions/" + id + "/resolve"
			table := "bu_terminal"
			if kind == "department" {
				parts = []string{"organization.department", f.instance, b.subject, id}
				route = "/api/v1/departments/submissions/" + id + "/resolve"
				table = "department_terminal"
			}
			raw, e := json.Marshal(parts)
			must(t, e)
			sum := sha256.Sum256(raw)
			blocker, e := f.admin.Begin(f.ctx)
			must(t, e)
			defer blocker.Rollback(context.Background())
			_, e = blocker.Exec(f.ctx, "SELECT pg_advisory_xact_lock($1)", int64(binary.BigEndian.Uint64(sum[:8])))
			must(t, e)
			start := time.Now()
			status, body, e := b.response("POST", route, nil, "")
			elapsed := time.Since(start)
			must(t, e)
			t.Logf("submission %s elapsed=%s status=%d", kind, elapsed, status)
			if status != 202 || body["outcome"] != "pending" || elapsed > 4*time.Second {
				t.Fatal("submission guard depended on wall-clock DB timer", elapsed, status, body)
			}
			if f.scalar("SELECT count(*) FROM organization."+table+" WHERE submission_id=$1", id) != 0 {
				t.Fatal("timeout fabricated terminal")
			}
			must(t, blocker.Rollback(f.ctx))
			v := b.call("POST", route, nil, "", 200)
			if v["outcome"] != "not_applied" {
				t.Fatal("released identity could not resolve", v)
			}
		})
	}
}

//go:build integration

package bootstrap_test

import (
	"bytes"
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"chnhcm.local/chnhcm/internal/contexts/organization/application"
	orgdb "chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Deliberately swallow the actual PostgreSQL savepoint-rollback response. The
// next socket read waits until pgx's cleanup deadline; no fabricated timeout is
// returned by the application or store.
type t07RollbackWire struct {
	net.Conn
	armed, hit *atomic.Bool
	dropping   bool
}

func (c *t07RollbackWire) Write(p []byte) (int, error) {
	if c.armed.Load() && len(p) > 5 && p[0] == 'Q' && bytes.HasPrefix(p[5:], []byte("rollback to savepoint")) {
		c.dropping = true
		c.hit.Store(true)
	}
	return c.Conn.Write(p)
}
func (c *t07RollbackWire) Read(p []byte) (int, error) {
	if !c.dropping {
		return c.Conn.Read(p)
	}
	buf := make([]byte, len(p))
	for {
		_, e := c.Conn.Read(buf)
		if e != nil {
			return 0, e
		}
	}
}
func TestT07BlockedRollbackReplyUsesOneCleanupDeadline(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	endpoint, input, _ := prepareCommitCommand(b, "T07_ROLLBACK_WIRE", "LIFECYCLE_PAIR")
	svc := f.runtime.Organization.(application.Service)
	original := svc.Store
	base := original.(*orgdb.Store)
	copyStore := *base
	cfg := base.Pool.Config().Copy()
	cfg.MinConns = 0
	cfg.MaxConns = 1
	var armed, hit atomic.Bool
	cfg.ConnConfig.DialFunc = func(ctx context.Context, network, address string) (net.Conn, error) {
		c, e := (&net.Dialer{}).DialContext(ctx, network, address)
		if e != nil {
			return nil, e
		}
		return &t07RollbackWire{Conn: c, armed: &armed, hit: &hit}, nil
	}
	pool, e := pgxpool.NewWithConfig(f.ctx, cfg)
	must(t, e)
	defer pool.Close()
	must(t, pool.Ping(f.ctx))
	copyStore.Pool = pool
	var cleanupAt time.Time
	svc.Store = t07AfterPersistStore{Store: &copyStore, after: func() error {
		armed.Store(true)
		cleanupAt = time.Now()
		return foundation.Fail(409, "M02_SOURCE_CONFLICT", "受控拒绝")
	}}
	f.runtime.Organization = svc
	defer func() { svc.Store = original; f.runtime.Organization = svc }()
	before := businessSnapshot(t, f)
	key := foundation.NewID()
	status, out, e := b.response("POST", endpoint, input, key)
	elapsed := time.Since(cleanupAt)
	must(t, e)
	svc.Store = original
	f.runtime.Organization = svc
	if !hit.Load() || elapsed < 1800*time.Millisecond || elapsed > 3*time.Second || status != 503 || out["outcome"] != nil {
		t.Fatal("cleanup deadline not actually exercised", hit.Load(), elapsed, status, out)
	}
	// The socket close must release the remaining outer transaction as well.
	until := time.Now().Add(time.Second)
	for f.scalar("SELECT count(*) FROM pg_stat_activity WHERE application_name='chnhcm' AND state='idle in transaction'") != 0 && time.Now().Before(until) {
		time.Sleep(3 * time.Millisecond)
	}
	for table, value := range businessSnapshot(t, f) {
		if before[table] != value {
			t.Fatal("blocked cleanup leaked", table)
		}
	}
	resolved := b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
	if resolved["outcome"] != "not_applied" {
		t.Fatal(resolved)
	}
	t.Logf("actual rollback reply swallowed; savepoint and outer cleanup shared deadline: %s; zero business/terminal residual", elapsed)
}

//go:build integration

package bootstrap_test

import (
	"bytes"
	"context"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"chnhcm.local/chnhcm/internal/contexts/organization/application"
	orgdb "chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5/pgxpool"
)

// The driver sends a real COMMIT over its real TCP connection. The fixture
// closes that socket before returning any COMMIT reply bytes to pgx. Unlike an
// application-level replacement error, pgx itself observes a broken connection.
type t07CommitWire struct {
	net.Conn
	armed, sent, dropped *atomic.Bool
	afterForward         bool
	commit               bool
}

func (c *t07CommitWire) Write(p []byte) (int, error) {
	if c.armed.Load() && len(p) > 5 && p[0] == 'Q' && bytes.Equal(p[5:], []byte("commit\x00")) && c.sent.CompareAndSwap(false, true) {
		c.commit = true
		n, e := c.Conn.Write(p)
		if c.afterForward {
			c.dropped.Store(true)
			_ = c.Conn.Close()
		}
		return n, e
	}
	return c.Conn.Write(p)
}
func (c *t07CommitWire) Read(p []byte) (int, error) {
	if c.commit && !c.afterForward {
		// Read into a private buffer so no reply reaches the driver. Observing the
		// server reply makes this branch deterministic without faking a DB commit.
		_, _ = c.Conn.Read(make([]byte, len(p)))
		c.dropped.Store(true)
		_ = c.Conn.Close()
		return 0, io.ErrUnexpectedEOF
	}
	return c.Conn.Read(p)
}
func TestT07RealCommitCommunicationLossPreservesIdentity(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	for _, mode := range []string{"drop_after_forward_before_ack", "drop_reply_bytes", "rejected_drop_reply"} {
		afterForward := mode == "drop_after_forward_before_ack"
		t.Run(mode, func(t *testing.T) {
			endpoint, input, _ := prepareCommitCommand(b, "T07_WIRE_"+foundation.NewID()[:8], "LIFECYCLE_PAIR")
			svc := f.runtime.Organization.(application.Service)
			original := svc.Store
			base := original.(*orgdb.Store)
			copyStore := *base
			var armed, sent, dropped atomic.Bool
			config := base.Pool.Config().Copy()
			config.MinConns = 0
			config.MaxConns = 1
			config.ConnConfig.DialFunc = func(ctx context.Context, network, address string) (net.Conn, error) {
				c, e := (&net.Dialer{}).DialContext(ctx, network, address)
				if e != nil {
					return nil, e
				}
				return &t07CommitWire{Conn: c, armed: &armed, sent: &sent, dropped: &dropped, afterForward: afterForward}, nil
			}
			pool, e := pgxpool.NewWithConfig(f.ctx, config)
			must(t, e)
			defer pool.Close()
			must(t, pool.Ping(f.ctx))
			copyStore.Pool = pool
			svc.Store = &copyStore
			if mode == "rejected_drop_reply" {
				svc.Store = t07AfterPersistStore{Store: &copyStore, after: func() error { return foundation.Fail(409, "M02_SOURCE_CONFLICT", "受控写后冲突") }}
			}
			f.runtime.Organization = svc
			defer func() { svc.Store = original; f.runtime.Organization = svc }()
			key := foundation.NewID()
			before := businessSnapshot(t, f)
			beforeSources := f.scalar("SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.bu_history")
			armed.Store(true)
			status, out, e := b.response("POST", endpoint, input, key)
			must(t, e)
			if !sent.Load() || !dropped.Load() {
				t.Fatal("did not interrupt real COMMIT TCP exchange")
			}
			if status != 503 || out["code"] != "M02_RESULT_PENDING" || out["outcome"] != nil || out["items"] != nil {
				t.Fatal("broken COMMIT exposed provisional result", status, out)
			}
			svc.Store = original
			f.runtime.Organization = svc
			start := time.Now()
			resolved := b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
			if time.Since(start) > 5*time.Second {
				t.Fatal("resolve exceeded its own budget")
			}
			if mode == "rejected_drop_reply" && resolved["outcome"] != "rejected" {
				t.Fatal("rejected COMMIT must recover its original rejection", resolved)
			}
			if resolved["outcome"] == "accepted" {
				if !afterForward && f.scalar("SELECT count(*) FROM organization.bu_history h CROSS JOIN LATERAL unnest(h.submission_refs) r WHERE h.ids[r]=$1::uuid", key) != 1 {
					t.Fatal("lost reply did not follow committed outcome")
				}
				retry := b.call("POST", endpoint, input, key, 200)
				if retry["sourceVersion"] != resolved["sourceVersion"] {
					t.Fatal("retry changed original source")
				}
				if f.scalar("SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.bu_history") != beforeSources+1 {
					t.Fatal("accepted source not unique")
				}
			} else if resolved["outcome"] == "rejected" && mode == "rejected_drop_reply" {
				retry := b.call("POST", endpoint, input, key, 409)
				if retry["code"] != "M02_SOURCE_CONFLICT" {
					t.Fatal(retry)
				}
				for table, n := range businessSnapshot(t, f) {
					if table != "terminals" && n != before[table] {
						t.Fatal("rejected wire left business fact", table)
					}
				}
			} else if resolved["outcome"] == "not_applied" && afterForward {
				b.call("POST", endpoint, input, key, 409)
				for table, n := range businessSnapshot(t, f) {
					if table != "terminals" && n != before[table] {
						t.Fatal("uncommitted source leaked", table)
					}
				}
			} else {
				t.Fatal("unexpected recovered result", resolved)
			}
			if f.scalar("SELECT (SELECT count(*) FROM organization.bu_history h CROSS JOIN LATERAL unnest(h.submission_refs) r WHERE h.ids[r]=$1::uuid)+(SELECT count(*) FROM organization.bu_terminal WHERE submission_id=$1)", key) != 1 {
				t.Fatal("terminal not unique")
			}
			t.Logf("actual COMMIT bytes forwarded; physical TCP closed; HTTP503 pending; original identity resolved=%v, unique terminal", resolved["outcome"])
		})
	}
}

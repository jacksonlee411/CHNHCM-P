//go:build integration

package bootstrap_test

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/application"
	orgdb "chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"sync"
	"testing"
	"time"
)

type independent207Store struct {
	application.Store
	mode, a, b, shared      string
	readyA, readyB, release chan struct{}
	onceA, onceB            sync.Once
	states                  chan string
}
type independent207Unit struct {
	application.Unit
	owner *independent207Store
}

func (s *independent207Store) WithinWrite(ctx context.Context, c access.Credentials, fn func(application.Unit) error) error {
	return s.Store.WithinWrite(ctx, c, func(u application.Unit) error { return fn(independent207Unit{u, s}) })
}
func (u independent207Unit) LockSubmission(ctx context.Context, id string) error {
	s := u.owner
	lockID := id
	if s.mode == "collision" {
		lockID = s.shared
	}
	e := u.Unit.LockSubmission(ctx, lockID)
	if e == nil && s.mode != "observe" {
		if id == s.a {
			s.onceA.Do(func() { close(s.readyA) })
		}
		if id == s.b {
			s.onceB.Do(func() { close(s.readyB) })
		}
		if s.mode == "deadlock" || s.mode == "collision" && id == s.a {
			select {
			case <-s.release:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		if s.mode == "deadlock" {
			other := s.a
			if id == s.a {
				other = s.b
			}
			e = u.Unit.LockSubmission(ctx, other)
		}
	}
	if e != nil {
		var p *pgconn.PgError
		if errors.As(e, &p) {
			s.states <- p.Code
		}
	}
	return e
}

type independent207Reply struct {
	status int
	body   map[string]any
	err    error
}

func independent207Call(b *browser, path string, body any, key string) <-chan independent207Reply {
	done := make(chan independent207Reply, 1)
	go func() { s, v, e := b.response("POST", path, body, key); done <- independent207Reply{s, v, e} }()
	return done
}
func independent207Wait(t *testing.T, c <-chan struct{}) {
	t.Helper()
	select {
	case <-c:
	case <-time.After(3 * time.Second):
		t.Fatal("transaction did not reach real-lock barrier")
	}
}
func independent207Queued(t *testing.T, f *fixture, n int) {
	t.Helper()
	until := time.Now().Add(time.Second)
	for time.Now().Before(until) {
		if f.scalar(`SELECT count(*) FROM pg_locks WHERE locktype='advisory' AND NOT granted`) >= n {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("did not observe %d real advisory waiters", n)
}
func TestIndependentTL207RealDeadlock(t *testing.T) {
	f := fresh(t)
	a := f.login("admin", f.password)
	b := f.login("admin", f.password)
	pa, ia, _ := prepareCommitCommand(a, "TL207_DEAD_A", "LIFECYCLE_PAIR")
	pb, ib, _ := prepareCommitCommand(b, "TL207_DEAD_B", "LIFECYCLE_PAIR")
	ka, kb := foundation.NewID(), foundation.NewID()
	svc := f.runtime.Organization.(application.Service)
	original := svc.Store
	s := &independent207Store{Store: original, mode: "deadlock", a: ka, b: kb, readyA: make(chan struct{}), readyB: make(chan struct{}), release: make(chan struct{}), states: make(chan string, 4)}
	svc.Store = s
	f.runtime.Organization = svc
	released := false
	defer func() {
		if !released {
			close(s.release)
		}
		svc.Store = original
		f.runtime.Organization = svc
	}()
	aa := independent207Call(a, pa, ia, ka)
	bb := independent207Call(b, pb, ib, kb)
	independent207Wait(t, s.readyA)
	independent207Wait(t, s.readyB)
	close(s.release)
	released = true
	ar, br := <-aa, <-bb
	must(t, ar.err)
	must(t, br.err)
	svc.Store = original
	f.runtime.Organization = svc
	states := []string{}
	for len(s.states) > 0 {
		states = append(states, <-s.states)
	}
	found := false
	for _, code := range states {
		if code == "40P01" {
			found = true
		}
	}
	if !found {
		t.Fatal("fixture did not reach PostgreSQL deadlock", states)
	}
	cases := []struct {
		client *browser
		path   string
		body   any
		key    string
		reply  independent207Reply
	}{{a, pa, ia, ka, ar}, {b, pb, ib, kb, br}}
	winners, losers := 0, 0
	for _, c := range cases {
		count := f.scalar("SELECT count(*) FROM organization.bu_history h CROSS JOIN LATERAL unnest(h.submission_refs) r WHERE h.ids[r]=$1::uuid", c.key)
		events := f.eventsFor(c.key)
		if c.reply.status == 200 {
			winners++
			if c.reply.body["outcome"] != "accepted" || count != 1 || events != 1 {
				t.Fatal("winner has partial result")
			}
			r := c.client.call("POST", "/api/v1/organization/submissions/"+c.key+"/resolve", nil, "", 200)
			if r["sourceVersion"] != c.reply.body["sourceVersion"] {
				t.Fatal("winner result drift")
			}
		}
		if c.reply.status == 503 {
			losers++
			if c.reply.body["outcome"] != nil || count != 0 || events != 0 || f.scalar("SELECT (SELECT count(*) FROM organization.bu_history h CROSS JOIN LATERAL unnest(h.submission_refs) r WHERE h.ids[r]=$1::uuid)+(SELECT count(*) FROM organization.bu_terminal WHERE submission_id=$1)", c.key) != 0 {
				t.Fatal("deadlock victim falsely closed or wrote facts")
			}
			r := c.client.call("POST", "/api/v1/organization/submissions/"+c.key+"/resolve", nil, "", 200)
			if r["outcome"] != "not_applied" {
				t.Fatal("deadlock victim could not close")
			}
			c.client.call("POST", c.path, c.body, c.key, 409)
		}
	}
	if winners != 1 || losers != 1 {
		t.Fatal("deadlock did not leave one accepted and one pending", ar.status, br.status)
	}
	t.Logf("real PostgreSQL SQLSTATE=%v; one HTTP200 accepted, one HTTP503 pending; victim zero source/outbox and resolve closes original identity", states)
}
func TestIndependentTL207RealSubmissionLockTimeout(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	path, input, _ := prepareCommitCommand(b, "TL207_LOCK_TIMEOUT", "LIFECYCLE_PAIR")
	key := foundation.NewID()
	blocker, e := f.admin.Begin(f.ctx)
	must(t, e)
	defer blocker.Rollback(context.Background())
	_, e = blocker.Exec(f.ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "unrelated-placeholder")
	must(t, e)
	// Ask the real Unit to acquire the exact full submission key on the owner's transaction through its documented hash convention.
	var fullKey int64
	fullKey = independent207Key(f.instance, b.subject, key)
	_, e = blocker.Exec(f.ctx, `SELECT pg_advisory_xact_lock($1)`, fullKey)
	must(t, e)
	svc := f.runtime.Organization.(application.Service)
	original := svc.Store
	// This probe specifically proves PostgreSQL's physical lock-timeout error.
	// The production Go and SQL ceilings are both two seconds; choosing an
	// earlier SQL timer in this fixture avoids racing those equal deadlines.
	physical := *original.(*orgdb.Store)
	guard := physical.Guard
	physical.Guard = func(ctx context.Context, tx pgx.Tx, c access.Credentials) (access.Guard, error) {
		g, e := guard(ctx, tx, c)
		if e == nil {
			_, e = tx.Exec(ctx, "SET LOCAL lock_timeout='750ms'")
		}
		return g, e
	}
	s := &independent207Store{Store: &physical, mode: "observe", states: make(chan string, 4)}
	svc.Store = s
	f.runtime.Organization = svc
	defer func() { svc.Store = original; f.runtime.Organization = svc }()
	started := time.Now()
	answer := independent207Call(b, path, input, key)
	independent207Queued(t, f, 1)
	r := <-answer
	elapsed := time.Since(started)
	must(t, r.err)
	svc.Store = original
	f.runtime.Organization = svc
	code := ""
	if len(s.states) > 0 {
		code = <-s.states
	}
	if code != "55P03" || r.status != 503 || r.body["outcome"] != nil || elapsed > 4*time.Second {
		t.Fatal("lock timeout was not bounded/pending", code, r.status, elapsed)
	}
	if f.scalar("SELECT (SELECT count(*) FROM organization.bu_history h CROSS JOIN LATERAL unnest(h.submission_refs) r WHERE h.ids[r]=$1::uuid)+(SELECT count(*) FROM organization.bu_terminal WHERE submission_id=$1)", key) != 0 || f.scalar("SELECT count(*) FROM organization.bu_history h CROSS JOIN LATERAL unnest(h.submission_refs) r WHERE h.ids[r]=$1::uuid", key) != 0 || f.eventsFor(key) != 0 {
		t.Fatal("lock timeout left facts or fabricated terminal")
	}
	must(t, blocker.Rollback(f.ctx))
	closed := b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
	if closed["outcome"] != "not_applied" {
		t.Fatal("timeout original identity not closable")
	}
	b.call("POST", path, input, key, 409)
	t.Logf("real PostgreSQL SQLSTATE=%s submission-lock timeout=%s; no terminal/source/outbox, explicit resolve closes original identity", code, elapsed)
}
func TestIndependentTL207ControlledAdvisoryCollisionKeepsFullIdentities(t *testing.T) {
	f := fresh(t)
	a := f.login("admin", f.password)
	b := f.login("admin", f.password)
	pa, ia, _ := prepareCommitCommand(a, "TL207_COLLISION_A", "LIFECYCLE_PAIR")
	pb, ib, _ := prepareCommitCommand(b, "TL207_COLLISION_B", "LIFECYCLE_PAIR")
	ka, kb := foundation.NewID(), foundation.NewID()
	svc := f.runtime.Organization.(application.Service)
	original := svc.Store
	s := &independent207Store{Store: original, mode: "collision", a: ka, b: kb, shared: foundation.NewID(), readyA: make(chan struct{}), readyB: make(chan struct{}), release: make(chan struct{}), states: make(chan string, 4)}
	// Force the same advisory hash in both Go and the native lock guard. Full
	// submission identities and native uniqueness checks remain unchanged.
	var originalKeyFunction string
	must(t, f.admin.QueryRow(f.ctx, "SELECT pg_get_functiondef('organization.submission_key(text,uuid,uuid,uuid)'::regprocedure)").Scan(&originalKeyFunction))
	f.exec(fmt.Sprintf(`CREATE OR REPLACE FUNCTION organization.submission_key(kind text,instance_id uuid,subject_id uuid,submission_id uuid) RETURNS bigint LANGUAGE sql IMMUTABLE STRICT AS 'SELECT %d::bigint'`, independent207Key(f.instance, a.subject, s.shared)))
	defer f.exec(originalKeyFunction)

	svc.Store = s
	f.runtime.Organization = svc
	released := false
	defer func() {
		if !released {
			close(s.release)
		}
		svc.Store = original
		f.runtime.Organization = svc
	}()
	aa := independent207Call(a, pa, ia, ka)
	independent207Wait(t, s.readyA)
	bb := independent207Call(b, pb, ib, kb)
	independent207Queued(t, f, 1)
	close(s.release)
	released = true
	ar, br := <-aa, <-bb
	must(t, ar.err)
	must(t, br.err)
	svc.Store = original
	f.runtime.Organization = svc
	if ar.status != 200 || br.status != 200 || ar.body["submissionId"] != ka || br.body["submissionId"] != kb || ar.body["sourceVersion"] == br.body["sourceVersion"] || ar.body["buId"] == br.body["buId"] {
		t.Fatal("advisory collision merged full identities", ar.status, br.status)
	}
	for _, c := range []struct {
		client *browser
		path   string
		body   any
		key    string
		reply  independent207Reply
	}{{a, pa, ia, ka, ar}, {b, pb, ib, kb, br}} {
		if f.scalar("SELECT count(*) FROM organization.bu_history h CROSS JOIN LATERAL unnest(h.submission_refs) r WHERE h.ids[r]=$1::uuid", c.key) != 1 || f.scalar("SELECT count(*) FROM organization.bu_history h CROSS JOIN LATERAL unnest(h.submission_refs) r WHERE h.ids[r]=$1::uuid", c.key) != 1 {
			t.Fatal("collision terminal/source count")
		}
		replay := c.client.call("POST", c.path, c.body, c.key, 200)
		resolved := c.client.call("POST", "/api/v1/organization/submissions/"+c.key+"/resolve", nil, "", 200)
		if replay["sourceVersion"] != c.reply.body["sourceVersion"] || resolved["sourceVersion"] != c.reply.body["sourceVersion"] {
			t.Fatal("collision lookup used advisory key instead of full identity")
		}
	}
	t.Log("controlled same advisory key on two distinct full submission identities: real DB waiter observed; two independent accepted terminals/sources and exact replay/resolve remain distinct")
}

func independent207Key(instance, subject, key string) int64 {
	b, _ := json.Marshal([]string{instance, subject, key})
	h := sha256.Sum256(b)
	return int64(binary.BigEndian.Uint64(h[:8]))
}

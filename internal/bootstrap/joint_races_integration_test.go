//go:build integration

package bootstrap_test

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/application"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// All gates surround real Unit operations; no production hooks or alternate
// transaction paths are introduced. The third request queues on the real DB
// submission lock while the predetermined winner is still holding it.
type jointBarrierStore struct {
	application.Store
	mode                                          string
	calls                                         atomic.Int32
	persists                                      atomic.Int32
	reached, releaseA, winnerReady, releaseWinner chan struct{}
}
type jointBarrierUnit struct {
	application.Unit
	owner *jointBarrierStore
	call  int32
}

func (s *jointBarrierStore) WithinWrite(ctx context.Context, c access.Credentials, fn func(application.Unit) error) error {
	call := s.calls.Add(1)
	return s.Store.WithinWrite(ctx, c, func(u application.Unit) error { return fn(jointBarrierUnit{u, s, call}) })
}
func (u jointBarrierUnit) LockSubmission(ctx context.Context, id string) error {
	if u.call == 3 && u.owner.mode != "reject" && u.owner.mode != "accepted-original" {
		select {
		case <-u.owner.winnerReady:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return u.Unit.LockSubmission(ctx, id)
}
func waitJointGate(ctx context.Context, c <-chan struct{}) error {
	select {
	case <-c:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (u jointBarrierUnit) Persist(ctx context.Context, w application.Write) (org.Result, error) {
	u.owner.persists.Add(1)
	r, e := u.Unit.Persist(ctx, w)
	if e != nil {
		return r, e
	}
	if u.call == 1 {
		close(u.owner.reached)
		if e = waitJointGate(ctx, u.owner.releaseA); e != nil {
			return r, e
		}
		if u.owner.mode == "reject" {
			return r, foundation.Fail(409, "M02_SOURCE_CONFLICT", "受控写后冲突")
		}
		if u.owner.mode == "accepted-original" {
			return r, nil
		}
		return r, context.DeadlineExceeded
	}
	if u.call == 2 && u.owner.mode == "retry-first" {
		close(u.owner.winnerReady)
		if e = waitJointGate(ctx, u.owner.releaseWinner); e != nil {
			return r, e
		}
	}
	return r, nil
}
func (u jointBarrierUnit) StoreTerminal(ctx context.Context, t application.Terminal) error {
	e := u.Unit.StoreTerminal(ctx, t)
	if e != nil {
		return e
	}
	if u.call == 2 && u.owner.mode == "resolve-first" && t.Outcome == "not_applied" {
		close(u.owner.winnerReady)
		return waitJointGate(ctx, u.owner.releaseWinner)
	}
	return nil
}

type jointReply struct {
	status int
	body   map[string]any
	err    error
}

func waitJointSignal(t *testing.T, c <-chan struct{}) {
	t.Helper()
	select {
	case <-c:
	case <-time.After(3 * time.Second):
		t.Fatal("joint barrier was not reached")
	}
}
func waitJointLocks(t *testing.T, f *fixture, count int) {
	t.Helper()
	deadline := time.Now().Add(1500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if f.scalar(`SELECT count(*) FROM pg_locks l JOIN pg_stat_activity a USING(pid) WHERE l.locktype='advisory' AND NOT l.granted AND a.datname=current_database()`) >= count {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("expected requests were not waiting on the real submission lock", count)
}
func TestJointThreeRequestDeterministicTerminalOrders(t *testing.T) {
	for _, mode := range []string{"reject", "retry-first", "resolve-first", "accepted-original"} {
		t.Run(mode, func(t *testing.T) {
			f := fresh(t)
			b := f.login("admin", f.password)
			time.Sleep(3100 * time.Millisecond)
			bRetry := f.login("admin", f.password)
			time.Sleep(3100 * time.Millisecond)
			bResolve := f.login("admin", f.password)
			endpoint, input, _ := prepareCommitCommand(b, "BARRIER_"+fmt.Sprint(len(mode)), "LIFECYCLE_PAIR")
			if mode == "accepted-original" {
				path := strings.TrimSuffix(endpoint, "/lifecycle-pairs")
				id := strings.TrimPrefix(path, "/api/v1/business-units/")
				initial := detail(b, id)
				first := initial["timeline"].([]any)[0].(map[string]any)
				b.call("POST", path+"/records/"+first["recordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": initial["headSourceVersion"], "expectedRecordRevision": first["revision"], "changes": map[string]any{"name": "甲", "nameEn": "Alpha"}}, foundation.NewID(), 200)
				for i, date := range []string{"2026-06-01", "2026-10-01"} {
					b.call("POST", path+"/records", addBody(detail(b, id), date, map[string]any{"name": []string{"乙", "丙"}[i], "nameEn": []string{"Beta", "Gamma"}[i]}), foundation.NewID(), 201)
				}
				d := detail(b, id)
				input = map[string]any{"expectedSourceVersion": d["headSourceVersion"], "anchorRecordId": d["timeline"].([]any)[0].(map[string]any)["recordId"], "deactivationDate": "2026-03-01", "reactivationDate": "2026-04-01", "deactivationReason": "BUSINESS_CLOSURE", "reactivationReason": "BUSINESS_RESUMPTION"}

			}
			key := foundation.NewID()
			svc := f.runtime.Organization.(application.Service)
			original := svc.Store
			barrier := &jointBarrierStore{Store: original, mode: mode, reached: make(chan struct{}), releaseA: make(chan struct{}), winnerReady: make(chan struct{}), releaseWinner: make(chan struct{})}
			t.Cleanup(func() {
				for _, c := range []chan struct{}{barrier.releaseA, barrier.releaseWinner} {
					select {
					case <-c:
					default:
						close(c)
					}
				}
			})
			svc.Store = barrier
			f.runtime.Organization = svc
			submit := func(client *browser) <-chan jointReply {
				out := make(chan jointReply, 1)
				go func() { s, v, e := client.response("POST", endpoint, input, key); out <- jointReply{s, v, e} }()
				return out
			}
			resolve := func() <-chan jointReply {
				out := make(chan jointReply, 1)
				go func() {
					s, v, e := bResolve.response("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "")
					out <- jointReply{s, v, e}
				}()
				return out
			}
			a := submit(b)
			waitJointSignal(t, barrier.reached)
			var retry, recovery <-chan jointReply
			if mode == "resolve-first" {
				recovery = resolve()
				waitJointLocks(t, f, 1)
				retry = submit(bRetry)
			} else {
				retry = submit(bRetry)
				waitJointLocks(t, f, 1)
				recovery = resolve()
			}
			if mode == "reject" || mode == "accepted-original" {
				waitJointLocks(t, f, 2)
			}
			close(barrier.releaseA)
			if mode != "reject" && mode != "accepted-original" {
				waitJointSignal(t, barrier.winnerReady)
				waitJointLocks(t, f, 1)
				close(barrier.releaseWinner)
			}
			ar, br, rr := <-a, <-retry, <-recovery
			must(t, ar.err)
			must(t, br.err)
			must(t, rr.err)
			svc.Store = original
			f.runtime.Organization = svc
			if f.scalar("SELECT (SELECT count(*) FROM organization.bu_history h CROSS JOIN LATERAL unnest(h.submission_refs) r WHERE h.ids[r]=$1::uuid)+(SELECT count(*) FROM organization.bu_terminal WHERE submission_id=$1)", key) != 1 {
				t.Fatal("competing requests left multiple terminals")
			}
			want := "not_applied"
			sources, items, persists := 1, 0, int32(1)
			switch mode {
			case "reject":
				want = "rejected"
				if ar.status != 409 || br.status != 409 || rr.status != 200 || !reflect.DeepEqual(ar.body["failureDetails"], rr.body["failureDetails"]) {
					t.Fatal(ar, br, rr)
				}
			case "accepted-original":
				want = "accepted"
				sources, items, persists = 5, 2, 1
				if ar.status != 200 || br.status != 200 || rr.status != 200 || ar.body["sourceVersion"] != br.body["sourceVersion"] || br.body["sourceVersion"] != rr.body["sourceVersion"] {
					t.Fatal(ar, br, rr)
				}
			case "retry-first":
				want = "accepted"
				sources, items, persists = 2, 2, 2
				if ar.status != 503 || br.status != 200 || rr.status != 200 || br.body["sourceVersion"] != rr.body["sourceVersion"] {
					t.Fatal(ar, br, rr)
				}
			case "resolve-first":
				if ar.status != 503 || br.status != 409 || br.body["code"] != "M02_REQUEST_CLOSED" || rr.status != 200 {
					t.Fatal(ar, br, rr)
				}
			}
			if rr.body["outcome"] != want || f.scalar("SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.bu_history") != sources || f.jointItemCount() != items || barrier.persists.Load() != persists {
				t.Fatal("wrong terminal or partial facts", mode, rr.body)
			}
			if mode == "reject" {
				changed := map[string]any{}
				for k, v := range input {
					changed[k] = v
				}
				changed["expectedSourceVersion"] = foundation.NewID()
				mismatch := b.call("POST", endpoint, changed, key, 409)
				if mismatch["code"] != "M02_REQUEST_MISMATCH" {
					t.Fatal("rejected content mismatch", mismatch)
				}
			}
			t.Logf("order=%s: A=%d retry=%d resolve=%d terminal=%s; real advisory-lock waiters observed", mode, ar.status, br.status, rr.status, want)
		})
	}
}

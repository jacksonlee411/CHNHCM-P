//go:build integration

package bootstrap_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/application"
	"chnhcm.local/chnhcm/internal/contexts/organization/domain"
	"chnhcm.local/chnhcm/internal/foundation"
)

type t07BUSerialStore struct {
	application.Store
	ready, release chan struct{}
	held           atomic.Bool
}
type t07BUSerialUnit struct {
	application.Unit
	owner *t07BUSerialStore
}

func (s *t07BUSerialStore) WithinWrite(ctx context.Context, c access.Credentials, fn func(application.Unit) error) error {
	return s.Store.WithinWrite(ctx, c, func(u application.Unit) error { return fn(t07BUSerialUnit{u, s}) })
}
func (u t07BUSerialUnit) LoadTimeline(ctx context.Context, id string, lock bool) (application.Timeline, error) {
	out, e := u.Unit.LoadTimeline(ctx, id, lock)
	if !lock {
		return out, e
	}
	return u.afterBULock(ctx, out, e)
}
func (u t07BUSerialUnit) LockUnchangedBUCoverage(ctx context.Context, base application.Timeline, candidate domain.Candidate) (application.Timeline, error) {
	out, e := u.Unit.LockUnchangedBUCoverage(ctx, base, candidate)
	return u.afterBULock(ctx, out, e)
}
func (u t07BUSerialUnit) afterBULock(ctx context.Context, out application.Timeline, e error) (application.Timeline, error) {
	if e == nil && u.owner.held.CompareAndSwap(false, true) {
		close(u.owner.ready)
		select {
		case <-u.owner.release:
		case <-ctx.Done():
			return out, ctx.Err()
		}
	}
	return out, e
}
func TestT07BUConcurrencyMatrixRealSeparateSessions(t *testing.T) {
	f := fresh(t)
	a := f.login("admin", f.password)
	b := f.login("admin", f.password)
	for _, kind := range []string{"add", "correct", "redate", "delete", "stop", "sameDayAdd", "differentBU"} {
		orders := 2
		if kind == "sameDayAdd" || kind == "differentBU" {
			orders = 1
		}
		for order := 0; order < orders; order++ {
			t.Run(fmt.Sprintf("%s_order_%d", kind, order), func(t *testing.T) {
				create := createBody("T07_RACE_"+foundation.NewID()[:8], "2026-01-01")
				create["name"], create["nameEn"] = "甲", "Alpha"
				id := a.call("POST", "/api/v1/business-units", create, foundation.NewID(), 201)["buId"].(string)
				base := "/api/v1/business-units/" + id
				a.call("POST", base+"/records", addBody(detail(a, id), "2026-06-01", map[string]any{"name": "乙", "nameEn": "Beta"}), foundation.NewID(), 201)
				a.call("POST", base+"/records", addBody(detail(a, id), "2026-10-01", map[string]any{"name": "丙", "nameEn": "Gamma"}), foundation.NewID(), 201)
				d := detail(a, id)
				r := d["timeline"].([]any)[1].(map[string]any)
				rp := base + "/records/" + r["recordId"].(string)
				c2 := d["timeline"].([]any)[2].(map[string]any)
				c2p := base + "/records/" + c2["recordId"].(string)
				pa := rp + "/redate"
				ia := map[string]any{"expectedSourceVersion": d["headSourceVersion"], "expectedRecordRevision": r["revision"], "newEffectiveDate": "2026-07-01"}
				wa := 200
				pb := ""
				var ib map[string]any
				wb := 200
				switch kind {
				case "add":
					pb = base + "/records"
					ib = addBody(d, "2026-08-01", map[string]any{"name": "新", "nameEn": "Zulu"})
					wb = 201
				case "correct":
					pb = c2p + "/correct"
					ib = map[string]any{"expectedSourceVersion": d["headSourceVersion"], "expectedRecordRevision": c2["revision"], "changes": map[string]any{"name": "新", "nameEn": "Zulu"}}
				case "redate":
					pb = rp + "/redate"
					ib = map[string]any{"expectedSourceVersion": d["headSourceVersion"], "expectedRecordRevision": r["revision"], "newEffectiveDate": "2026-09-01"}
				case "delete":
					pb = c2p + "/delete"
					ib = map[string]any{"expectedSourceVersion": d["headSourceVersion"], "expectedRecordRevision": c2["revision"]}
				case "stop":
					pb = base + "/deactivate"
					ib = map[string]any{"anchorRecordId": lifecycleAnchor(d), "expectedSourceVersion": d["headSourceVersion"], "effectiveDate": "2026-12-01", "operationType": "DEACTIVATE", "operationReason": "BUSINESS_CLOSURE"}
					wb = 201
				case "sameDayAdd":
					pa = base + "/records"
					ia = addBody(d, "2026-08-01", map[string]any{"name": "先"})
					wa = 201
					pb = pa
					ib = addBody(d, "2026-08-01", map[string]any{"name": "后"})
					wb = 201
				case "differentBU":
					other := a.call("POST", "/api/v1/business-units", createBody("T07_RACE_OTHER_"+foundation.NewID()[:8], "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
					pb = "/api/v1/business-units/" + other + "/records"
					ib = addBody(detail(b, other), "2026-08-01", map[string]any{"name": "独立"})
					wb = 201
				}
				if order == 1 {
					pa, pb = pb, pa
					ia, ib = ib, ia
					wa, wb = wb, wa
				}
				old := businessSnapshot(t, f)["sources"]
				svc := f.runtime.Organization.(application.Service)
				original := svc.Store
				barrier := &t07BUSerialStore{Store: original, ready: make(chan struct{}), release: make(chan struct{})}
				svc.Store = barrier
				f.runtime.Organization = svc
				released := false
				defer func() {
					if !released {
						close(barrier.release)
					}
					svc.Store = original
					f.runtime.Organization = svc
				}()
				ka, kb := foundation.NewID(), foundation.NewID()
				aa := independent207Call(a, pa, ia, ka)
				independent207Wait(t, barrier.ready)
				bb := independent207Call(b, pb, ib, kb)
				var br independent207Reply
				if kind == "differentBU" {
					select {
					case br = <-bb:
					case <-time.After(time.Second):
						t.Fatal("independent BU blocked behind unrelated BU")
					}
				} else {
					// A holds a real row lock. B uses a separate authenticated session and
					// reaches the actual PostgreSQL transaction/tuple lock wait.
					observed := false
					until := time.Now().Add(time.Second)
					for time.Now().Before(until) {
						if f.scalar(`SELECT count(*) FROM pg_stat_activity WHERE application_name='chnhcm' AND wait_event_type='Lock' AND cardinality(pg_blocking_pids(pid))>0`) > 0 {
							observed = true
							break
						}
						time.Sleep(3 * time.Millisecond)
					}
					if !observed {
						t.Fatal("no real concurrent BU lock waiter")
					}
				}
				close(barrier.release)
				released = true
				ar := <-aa
				if kind != "differentBU" {
					br = <-bb
				}
				must(t, ar.err)
				must(t, br.err)
				svc.Store = original
				f.runtime.Organization = svc
				if ar.status != wa || ar.body["outcome"] != "accepted" {
					t.Fatal("first writer", ar)
				}
				if kind == "differentBU" {
					if br.status != wb || br.body["outcome"] != "accepted" {
						t.Fatal(br)
					}
				} else {
					if br.status != 409 || br.body["code"] != "M02_SOURCE_CONFLICT" {
						t.Fatal("later stale writer", br)
					}
					current := detail(a, id)
					if current["headSourceVersion"] != ar.body["sourceVersion"] {
						t.Fatal("wrong final head")
					}
					historical := a.call("GET", base+"?sourceVersion="+d["headSourceVersion"].(string), nil, "", 200)
					if len(historical["timeline"].([]any)) != 3 || historical["timeline"].([]any)[1].(map[string]any)["effectiveDate"] != "2026-06-01" {
						t.Fatal("original source changed")
					}
				}
				if old == businessSnapshot(t, f)["sources"] || f.scalar("SELECT (SELECT count(*) FROM organization.bu_history h CROSS JOIN LATERAL unnest(h.submission_refs) r WHERE h.ids[r] IN ($1::uuid,$2::uuid))+(SELECT count(*) FROM organization.bu_terminal WHERE submission_id IN ($1,$2))", ka, kb) != 2 {
					t.Fatal("missing unique complete outcomes")
				}
				t.Log("separate sessions; observed actual lock waiting (or independent-BU progress), original source and unique outcomes checked")
			})
		}
	}
}

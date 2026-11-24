//go:build integration

package bootstrap_test

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/application"
	orgdb "chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type snapshotReadTraceKey struct{}

type snapshotReadBarrier struct {
	prefix           string
	reached, release chan struct{}
	once             sync.Once
}

func (b *snapshotReadBarrier) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	return context.WithValue(ctx, snapshotReadTraceKey{}, strings.HasPrefix(data.SQL, b.prefix))
}

func (b *snapshotReadBarrier) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	if matched, _ := ctx.Value(snapshotReadTraceKey{}).(bool); !matched || data.Err != nil {
		return
	}
	b.once.Do(func() {
		close(b.reached)
		select {
		case <-b.release:
		case <-ctx.Done():
		}
	})
}

func TestT07ReadsPinOneSourceDuringConcurrentCorrection(t *testing.T) {
	f := fresh(t)
	writer := f.login("admin", f.password)
	reader := f.login("admin", f.password)
	u, e := url.Parse(f.server.URL)
	must(t, e)
	creds := access.Credentials{Token: reader.client.Jar.Cookies(u)[0].Value}
	for _, kind := range []string{"detail", "list"} {
		t.Run(kind, func(t *testing.T) {
			in := createBody("T07_READ_"+kind, "2026-01-01")
			in["name"] = "旧名称" + kind
			saved := writer.call("POST", "/api/v1/business-units", in, foundation.NewID(), 201)
			id := saved["buId"].(string)
			old := detail(writer, id)
			record := old["timeline"].([]any)[0].(map[string]any)
			svc := f.runtime.Organization.(application.Service)
			store := *svc.Store.(*orgdb.Store)
			reached, release := make(chan struct{}), make(chan struct{})
			prefix := "-- name: GetUnit :one"
			if kind == "list" {
				prefix = "SELECT b.bu_id::text,b.enterprise_id::text,b.code"
			}
			// Pause this reader after the actual head/list query has completed.
			// The barrier is independent of how often business time is sampled.
			cfg := store.Pool.Config()
			cfg.ConnConfig.Tracer = &snapshotReadBarrier{prefix: prefix, reached: reached, release: release}
			readerPool, err := pgxpool.NewWithConfig(f.ctx, cfg)
			must(t, err)
			defer readerPool.Close()
			var releaseOnce sync.Once
			releaseRead := func() { releaseOnce.Do(func() { close(release) }) }
			defer releaseRead()
			store.Pool = readerPool
			svc.Store = &store
			type response struct {
				d org.Detail
				l org.List
				e error
			}
			done := make(chan response, 1)
			go func() {
				v := response{}
				if kind == "detail" {
					v.d, v.e = svc.Detail(f.ctx, creds, org.DetailInput{ID: id, BusinessAt: "2026-03-01"})
				} else {
					v.l, v.e = svc.List(f.ctx, creds, org.ListInput{BusinessAt: "2026-03-01", Q: "旧名称" + kind, Status: "all", Sort: "code", Page: 1, PageSize: 20})
				}
				done <- v
			}()
			select {
			case <-reached:
			case <-time.After(3 * time.Second):
				t.Fatal("read barrier not reached")
			}
			input := map[string]any{"expectedSourceVersion": old["headSourceVersion"], "expectedRecordRevision": record["revision"], "changes": map[string]any{"name": "新名称" + kind}}
			status, newSource, err := writer.response("POST", "/api/v1/business-units/"+id+"/records/"+record["recordId"].(string)+"/correct", input, foundation.NewID())
			if status == 200 && err == nil && kind == "detail" {
				newSource = writer.call("POST", "/api/v1/business-units/"+id+"/records", addBody(detail(writer, id), "2026-06-01", map[string]any{"name": "并发后继"}), foundation.NewID(), 201)
			}
			releaseRead()
			must(t, err)
			if status != 200 {
				t.Fatal("concurrent correction could not commit", status, newSource)
			}
			got := <-done
			must(t, got.e)
			if kind == "detail" {
				if got.d.PairInsertion == nil || got.d.PairInsertion.AnchorRecordID != record["recordId"] || got.d.PairInsertion.MaxDate != "9999-12-31" {
					t.Fatal("pair context escaped pinned source", got.d.PairInsertion)
				}

				if got.d.HeadSourceVersion == nil || *got.d.HeadSourceVersion != old["headSourceVersion"] || got.d.ResolvedSourceVersion == nil || *got.d.ResolvedSourceVersion != old["headSourceVersion"] || len(got.d.Timeline) != 1 || got.d.ViewFact == nil || *got.d.ViewFact.Name != "旧名称"+kind || got.d.Timeline[0].Revision != record["revision"] || !got.d.Capabilities.Correct {
					t.Fatal("detail mixed concurrent sources", got.d)
				}
			} else {
				if got.l.Count != 1 || len(got.l.Items) != 1 || got.l.Items[0].ID != id || *got.l.Items[0].Name != "旧名称"+kind {
					t.Fatal("list count and items mixed versions", got.l)
				}
				next := writer.call("GET", "/api/v1/business-units?q="+url.QueryEscape("旧名称"+kind), nil, "", 200)
				if next["count"] != float64(0) {
					t.Fatal("following list did not advance", next)
				}
			}
			current := detail(writer, id)
			if current["headSourceVersion"] != newSource["sourceVersion"] || reflect.DeepEqual(current["timeline"], old["timeline"]) {
				t.Fatal("writer did not advance")
			}
			t.Log(fmt.Sprintf("%s read held after head/list snapshot; another actual connection committed before reader returned", kind))
		})
	}
}

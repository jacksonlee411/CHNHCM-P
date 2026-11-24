//go:build integration

package bootstrap_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/contexts/organization/application"
	orgdb "chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
)

func t07Clock(f *fixture) *atomic.Int64 {
	clock := &atomic.Int64{}
	now, _ := time.Parse(time.RFC3339, "2026-06-14T15:59:59Z")
	clock.Store(now.Unix())
	svc := f.runtime.Organization.(application.Service)
	svc.Store.(*orgdb.Store).Clock = func(context.Context, pgx.Tx) (time.Time, error) { return time.Unix(clock.Load(), 0), nil }
	return clock
}
func TestT07BusinessDayNaturalTransitionsAndOldPreview(t *testing.T) {
	f := fresh(t)
	clock := t07Clock(f)
	b := f.login("admin", f.password)
	id := b.call("POST", "/api/v1/business-units", createBody("T07_NATURAL", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	path := "/api/v1/business-units/" + id
	for _, v := range []struct{ action, date, kind, reason string }{{"deactivate", "2026-06-15", "DEACTIVATE", "BUSINESS_CLOSURE"}, {"reactivate", "2026-06-16", "REACTIVATE", "BUSINESS_RESUMPTION"}} {
		in := map[string]any{"anchorRecordId": lifecycleAnchor(detail(b, id)), "expectedSourceVersion": detail(b, id)["headSourceVersion"], "effectiveDate": v.date, "operationType": v.kind, "operationReason": v.reason}
		b.call("POST", path+"/"+v.action, in, foundation.NewID(), 201)
	}
	before := businessSnapshot(t, f)
	for _, v := range []struct{ instant, date, state string }{{"2026-06-14T15:59:59Z", "2026-06-14", "active"}, {"2026-06-14T16:00:00Z", "2026-06-15", "inactive"}, {"2026-06-15T15:59:59Z", "2026-06-15", "inactive"}, {"2026-06-15T16:00:00Z", "2026-06-16", "active"}} {
		now, _ := time.Parse(time.RFC3339, v.instant)
		clock.Store(now.Unix())
		d := detail(b, id)
		if d["status"] != v.state || d["businessAt"] != v.date {
			t.Fatal("natural day boundary", v, d)
		}
	}
	if !reflect.DeepEqual(before, businessSnapshot(t, f)) {
		t.Fatal("natural arrival appended facts or events")
	}
	for i, day := range []string{"2026-06-15", "2026-06-16"} {
		now, _ := time.Parse(time.RFC3339, "2026-06-14T15:59:59Z")
		clock.Store(now.Unix())
		id := b.call("POST", "/api/v1/business-units", createBody(fmt.Sprintf("T07_PREVIEW_%d", i), "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
		path := "/api/v1/business-units/" + id
		prepared := detail(b, id)
		if prepared["additionContext"].(map[string]any)["defaultEffectiveDate"] != "2026-06-14" {
			t.Fatal("default did not use server day", prepared)
		}
		in := addBody(prepared, "2026-06-15", map[string]any{"name": "跨日资料"})
		preview := map[string]any{}
		for k, v := range in {
			preview[k] = v
		}
		preview["action"] = "ADD"
		p := b.call("POST", path+"/preview", preview, "", 200)
		if p["outcome"] != "accepted" {
			t.Fatal(p)
		}
		now, _ = time.Parse("2006-01-02", day)
		clock.Store(now.Unix())
		saved := b.call("POST", path+"/records", in, foundation.NewID(), 201)
		if saved["evaluatedBusinessDate"] != day || saved["currentStatus"] != "active" {
			t.Fatal("reused preview sample", saved)
		}
		if saved["effectiveDate"] != "2026-06-15" {
			t.Fatal("target date changed", saved)
		}
	}
	for i, day := range []string{"2026-06-15", "2026-06-16"} {
		now, _ := time.Parse(time.RFC3339, "2026-06-14T15:59:59Z")
		clock.Store(now.Unix())
		id := b.call("POST", "/api/v1/business-units", createBody(fmt.Sprintf("T07_JOINT_PREVIEW_%d", i), "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
		path := "/api/v1/business-units/" + id
		d := detail(b, id)
		input := map[string]any{"expectedSourceVersion": d["headSourceVersion"], "anchorRecordId": d["timeline"].([]any)[0].(map[string]any)["recordId"], "deactivationDate": "2026-06-15", "reactivationDate": "2026-06-16", "deactivationReason": "BUSINESS_CLOSURE", "reactivationReason": "BUSINESS_RESUMPTION"}

		now, _ = time.Parse("2006-01-02", day)
		clock.Store(now.Unix())
		saved := b.call("POST", path+"/lifecycle-pairs", input, foundation.NewID(), 200)
		want := "当天生效"
		if i == 1 {
			want = "过去补登记"
		}
		if saved["evaluatedBusinessDate"] != day || func() any {
			for _, x := range saved["items"].([]any) {
				r := x.(map[string]any)
				if r["effectiveDate"] == "2026-06-15" {
					return r["dateDescription"]
				}
			}
			return nil
		}() != want {
			t.Fatal("joint reused old preview day/description", saved)
		}
	}

	// A displayed default is request input, not a rule to recompute after midnight.
	instant, _ := time.Parse(time.RFC3339, "2026-06-14T15:59:59Z")
	clock.Store(instant.Unix())
	id = b.call("POST", "/api/v1/business-units", createBody("POSITION_DEFAULT_CLOCK", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	prepared := detail(b, id)
	shown := prepared["additionContext"].(map[string]any)["defaultEffectiveDate"].(string)
	if shown != "2026-06-14" {
		t.Fatal("wrong displayed server-day default", prepared)
	}
	captured := addBody(prepared, shown, map[string]any{"name": "默认跨日资料"})
	key := foundation.NewID()
	instant, _ = time.Parse(time.RFC3339, "2026-06-14T16:00:00Z")
	clock.Store(instant.Unix())
	if detail(b, id)["additionContext"].(map[string]any)["defaultEffectiveDate"] != "2026-06-15" {
		t.Fatal("preparation did not use Shanghai day")
	}
	accepted := b.call("POST", "/api/v1/business-units/"+id+"/records", captured, key, 201)
	if accepted["effectiveDate"] != "2026-06-14" || accepted["evaluatedBusinessDate"] != "2026-06-15" {
		t.Fatal("midnight rewrote captured date", accepted)
	}
	instant, _ = time.Parse(time.RFC3339, "2026-06-15T16:00:00Z")
	clock.Store(instant.Unix())
	for _, recovered := range []map[string]any{b.call("POST", "/api/v1/business-units/"+id+"/records", captured, key, 201), b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)} {
		if recovered["sourceVersion"] != accepted["sourceVersion"] || recovered["effectiveDate"] != "2026-06-14" || recovered["evaluatedBusinessDate"] != "2026-06-15" {
			t.Fatal("next day replaced original submission", recovered)
		}
	}

}

func TestT07BusinessDayBoundaryBrowser(t *testing.T) {
	if os.Getenv("CHNHCM_T07_CLOCK_BROWSER") != "1" {
		t.Skip("explicit browser lane after make build")
	}
	dist, e := filepath.Abs("../../apps/web/dist")
	must(t, e)
	_, e = os.Stat(filepath.Join(dist, "index.html"))
	must(t, e)
	f := fresh(t)
	clock := t07Clock(f)
	server := httptest.NewUnstartedServer(nil)
	api := &bootstrap.API{Runtime: f.runtime, Origin: "http://" + server.Listener.Addr().String(), InsecureLocal: true}
	handler := api.Handler()
	files := http.FileServer(http.Dir(dist))
	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Harness-only clock control on an ephemeral localhost test server; never
		// registered by API.Handler or a running development/production server.
		if r.URL.Path == "/__fixture_clock" && r.Method == "POST" {
			v := r.URL.Query().Get("instant")
			if v != "2026-06-14T15:59:59Z" && v != "2026-06-14T16:00:00Z" && v != "2026-06-30T15:59:59Z" && v != "2026-06-30T16:00:00Z" && v != "2026-07-31T16:00:00Z" {
				w.WriteHeader(400)
				return
			}
			at, _ := time.Parse(time.RFC3339, v)
			clock.Store(at.Unix())
			w.WriteHeader(204)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			handler.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			files.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(dist, "index.html"))
	})
	server.Start()
	defer server.Close()
	ctx, cancel := context.WithTimeout(f.ctx, 3*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "node", "../../tests/e2e/t07-clock.mjs", server.URL)
	raw, _ := json.Marshal(map[string]string{"login": "admin", "password": f.password})
	cmd.Stdin = bytes.NewReader(raw)
	out, e := cmd.CombinedOutput()
	if e != nil {
		t.Fatalf("clock browser: %s", out)
	}
	if f.scalar("SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.bu_history") != 3 || f.eventCount() != 3 {
		t.Fatal("natural arrival generated another source/event")
	}
	t.Log(strings.TrimSpace(string(out)))
}

//go:build integration

package bootstrap_test

import (
	"bytes"
	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/contexts/organization/application"
	orgdb "chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres"
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestJointMidnightBrowser(t *testing.T) {
	if os.Getenv("CHNHCM_MIDNIGHT_BROWSER") != "1" {
		t.Skip("supplementary browser lane: run after make build with CHNHCM_MIDNIGHT_BROWSER=1")
	}
	dist, e := filepath.Abs("../../apps/web/dist")
	must(t, e)
	_, e = os.Stat(filepath.Join(dist, "index.html"))
	must(t, e)
	f := fresh(t)
	svc := f.runtime.Organization.(application.Service)
	store := svc.Store.(*orgdb.Store)
	before, _ := time.Parse(time.RFC3339, "2026-06-14T15:59:59Z")
	store.Clock = func(context.Context, pgx.Tx) (time.Time, error) { return before, nil }
	restoreRecorded := f.recordedAt("2026-06-14T16:00:01Z")
	defer restoreRecorded()
	server := httptest.NewUnstartedServer(nil)
	api := &bootstrap.API{Runtime: f.runtime, Origin: "http://" + server.Listener.Addr().String(), InsecureLocal: true}
	handler := api.Handler()
	files := http.FileServer(http.Dir(dist))
	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
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
	ctx, cancel := context.WithTimeout(f.ctx, 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "node", "../../tests/e2e/joint-midnight.mjs", server.URL)
	body, _ := json.Marshal(map[string]string{"login": "admin", "password": f.password})
	cmd.Stdin = bytes.NewReader(body)
	output, e := cmd.CombinedOutput()
	if e != nil {
		t.Fatalf("supplementary browser: %s", strings.TrimSpace(string(output)))
	}
	t.Log(strings.TrimSpace(string(output)))
}

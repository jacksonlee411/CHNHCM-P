//go:build integration

package bootstrap_test

import (
	"bytes"
	"chnhcm.local/chnhcm/internal/bootstrap"
	app "chnhcm.local/chnhcm/internal/contexts/staffing/application"
	db "chnhcm.local/chnhcm/internal/contexts/staffing/infra/postgres"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStaffing308ClockBrowser(t *testing.T) {
	if os.Getenv("CHNHCM_STAFFING308_CLOCK_BROWSER") != "1" {
		t.Skip("explicit fixed-clock browser after make build")
	}
	dist, e := filepath.Abs("../../apps/web/dist")
	must(t, e)
	_, e = os.Stat(filepath.Join(dist, "index.html"))
	must(t, e)
	const enterpriseStart = "1984-01-01"
	fixture := freshWithEnterpriseStart(t, enterpriseStart)
	h := &staffing307{t, fixture, fixture.login("admin", fixture.password)}
	store := h.f.runtime.Staffing.(app.Service).Store.(*db.Store)
	now := time.Date(2026, 1, 15, 0, 2, 3, 123456000, time.UTC)
	store.Clock = func() time.Time { return now }
	h.b.functionForJobs(enterpriseStart)
	f := h.create("jobfamily", "CLOCK308-F", "")
	j := h.create("job", "CLOCK308-J", f["id"].(string))
	now = time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	server := httptest.NewUnstartedServer(nil)
	handler := (&bootstrap.API{Runtime: h.f.runtime, Origin: "http://" + server.Listener.Addr().String(), InsecureLocal: true}).Handler()
	files := http.FileServer(http.Dir(dist))
	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			handler.ServeHTTP(w, r)
		} else if strings.HasPrefix(r.URL.Path, "/assets/") {
			files.ServeHTTP(w, r)
		} else {
			http.ServeFile(w, r, filepath.Join(dist, "index.html"))
		}
	})
	server.Start()
	defer server.Close()
	ctx, cancel := context.WithTimeout(h.f.ctx, 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "node", "../../tests/e2e/staffing308-clock.mjs", server.URL)
	input, _ := json.Marshal(map[string]any{"password": h.f.password, "objects": []map[string]any{{"segment": "job-families", "id": f["id"]}, {"segment": "jobs", "id": j["id"]}}})
	cmd.Stdin = bytes.NewReader(input)
	out, e := cmd.CombinedOutput()
	if e != nil {
		t.Fatalf("Staffing clock browser: %s", out)
	}
	t.Log(strings.TrimSpace(string(out)))
}

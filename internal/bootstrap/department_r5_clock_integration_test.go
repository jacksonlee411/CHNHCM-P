//go:build integration

package bootstrap_test

import (
	"bytes"
	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestDepartmentR5BusinessDayBrowser(t *testing.T) {
	if os.Getenv("CHNHCM_R5_CLOCK_BROWSER") != "1" {
		t.Skip("explicit R5 browser lane after make build")
	}
	dist, e := filepath.Abs("../../apps/web/dist")
	must(t, e)
	_, e = os.Stat(filepath.Join(dist, "index.html"))
	must(t, e)
	f := fresh(t)
	var clock atomic.Int64
	before, _ := time.Parse(time.RFC3339, "2026-06-30T15:59:59Z")
	clock.Store(before.Unix())
	restore := bootstrap.SetIntegrationClock(f.runtime, func() time.Time { return time.Unix(clock.Load(), 0) })
	defer restore()
	admin, reader := f.login("admin", f.password), f.login("reader", f.readPassword)
	a := admin.call("POST", "/api/v1/business-units", createBody("R5_CLOCK_A", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	b := admin.call("POST", "/api/v1/business-units", createBody("R5_CLOCK_B", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	created := admin.call("POST", "/api/v1/departments", map[string]any{"code": "R5_CLOCK_C", "name": "跨日转出部门", "managementBuId": a, "parentDepartmentId": nil, "effectiveDate": "2026-01-01", "operationType": "CREATE", "operationReason": "NEW_ORGANIZATION"}, foundation.NewID(), 201)
	id := created["departmentId"].(string)
	admin.call("POST", "/api/v1/departments/"+id+"/records", map[string]any{"expectedSourceVersion": created["sourceVersion"], "anchorRecordId": created["selectedRecordId"], "effectiveDate": "2026-07-01", "operationType": "CHANGE", "operationReason": "ORGANIZATION_REASSIGNMENT", "changes": map[string]any{"managementBuId": b, "name": "七月转入部门"}}, foundation.NewID(), 201)
	tx, e := f.admin.Begin(f.ctx)
	must(t, e)
	defer tx.Rollback(f.ctx)
	_, e = tx.Exec(f.ctx, `SELECT principal_id FROM access.policy_guard WHERE instance_id=$1 AND principal_id=$2 FOR UPDATE`, f.instance, reader.subject)
	must(t, e)
	_, e = tx.Exec(f.ctx, `UPDATE access.subject_grant SET scope_kind='bu',scope_id=$3 WHERE instance_id=$1 AND principal_id=$2 AND action IN ('department.read','department.history.read')`, f.instance, reader.subject, a)
	must(t, e)
	_, e = tx.Exec(f.ctx, `INSERT INTO access.subject_grant(instance_id,grant_id,principal_id,scope_kind,scope_id,action,read_fields,write_fields) SELECT instance_id,gen_random_uuid(),principal_id,scope_kind,scope_id,'department.export',read_fields,ARRAY[]::text[] FROM access.subject_grant WHERE instance_id=$1 AND principal_id=$2 AND action='department.read'`, f.instance, reader.subject)
	must(t, e)
	_, e = tx.Exec(f.ctx, `UPDATE access.policy_guard SET policy_no=policy_no+1 WHERE instance_id=$1 AND principal_id=$2`, f.instance, reader.subject)
	must(t, e)
	must(t, tx.Commit(f.ctx))
	count := f.scalar("SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.department_history")
	server := httptest.NewUnstartedServer(nil)
	handler := (&bootstrap.API{Runtime: f.runtime, Origin: "http://" + server.Listener.Addr().String(), InsecureLocal: true}).Handler()
	files := http.FileServer(http.Dir(dist))
	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/__fixture_clock" && r.Method == "POST" {
			v := r.URL.Query().Get("instant")
			if v != "2026-06-30T15:59:59Z" && v != "2026-06-30T16:00:00Z" {
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
	cmd := exec.CommandContext(ctx, "node", "../../tests/e2e/department-clock.mjs", server.URL)
	raw, _ := json.Marshal(map[string]any{"login": "reader", "password": f.readPassword, "id": id, "source": created["sourceVersion"]})
	cmd.Stdin = bytes.NewReader(raw)
	out, e := cmd.CombinedOutput()
	if e != nil {
		t.Fatalf("R5 clock browser: %s", out)
	}
	t.Log(strings.TrimSpace(string(out)))
	if f.scalar("SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.department_history") != count {
		t.Fatal("day passage wrote a department source")
	}
}

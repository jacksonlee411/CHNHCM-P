//go:build integration

package bootstrap_test

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/application"
	orgdb "chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"encoding/csv"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"net/url"
	"strings"
	"testing"
	"time"
)

type exportPauseGuard struct {
	access.Guard
	entered chan struct{}
	release <-chan struct{}
}

func (g exportPauseGuard) Decide(ctx context.Context, in access.Intent) (access.Decision, error) {
	d, e := g.Guard.Decide(ctx, in)
	if e != nil || in.Action != "bu.export" {
		return d, e
	}
	close(g.entered)
	select {
	case <-g.release:
		return d, nil
	case <-ctx.Done():
		return d, ctx.Err()
	}
}
func TestQuerySelectionSourceFieldsAndCancellation(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	created := b.call("POST", "/api/v1/business-units", createBody("QS0", "2026-01-01"), foundation.NewID(), 201)
	path := "/api/v1/business-units?q=QS"
	old := b.call("GET", path, nil, "", 200)
	detail := b.call("GET", "/api/v1/business-units/"+created["buId"].(string), nil, "", 200)
	record := detail["selectedRecord"].(map[string]any)
	b.call("POST", "/api/v1/business-units/"+created["buId"].(string)+"/records/"+created["targetRecordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": created["sourceVersion"], "expectedRecordRevision": record["revision"], "changes": map[string]any{"name": "资料来源变更"}}, foundation.NewID(), 200)
	freshList := b.call("GET", path, nil, "", 200)
	if freshList["count"] != old["count"] || freshList["selectionVersion"] == old["selectionVersion"] {
		t.Fatal("source-only change not detected")
	}
	b.call("POST", "/api/v1/business-units/selection/resolve", selectionBody(old, []string{}, true), "", 409)
	f.exec("DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='bu.record.add'", b.subject)
	if after := b.call("GET", path, nil, "", 200); after["selectionVersion"] != freshList["selectionVersion"] {
		t.Fatal("unrelated write rights changed selection version")
	}
	f.exec("UPDATE access.subject_grant SET read_fields='{}'::text[] WHERE principal_id=$1 AND action='bu.read'", b.subject)
	freshList = b.call("GET", "/api/v1/business-units", nil, "", 200)
	body := selectionBody(freshList, []string{}, true)
	body["query"].(map[string]any)["q"] = ""
	raw := selectionDownload(t, b, body, 200)
	rows, e := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(raw), "\xef\xbb\xbf"))).ReadAll()
	must(t, e)
	if len(rows) != 2 || rows[1][0] != "" || rows[1][1] != "" || rows[1][3] != "" || rows[1][2] != "启用" {
		t.Fatal("export permission leaked read fields", rows)
	}
	// Restore only read fields. The test uses the real guard and database, with a
	// task-owned pause after its actual export decision to cancel in-flight work.
	f.exec("UPDATE access.subject_grant SET read_fields=ARRAY['code','name','effectiveDate'] WHERE principal_id=$1 AND action='bu.read'", b.subject)
	list := b.call("GET", path, nil, "", 200)
	service := f.runtime.Organization.(application.Service)
	store := *service.Store.(*orgdb.Store)
	guard := store.Guard
	entered := make(chan struct{})
	release := make(chan struct{})
	store.Guard = func(ctx context.Context, tx pgx.Tx, c access.Credentials) (access.Guard, error) {
		g, e := guard(ctx, tx, c)
		return exportPauseGuard{g, entered, release}, e
	}
	service.Store = &store
	endpoint, _ := url.Parse(f.server.URL)
	creds := access.Credentials{Token: b.client.Jar.Cookies(endpoint)[0].Value, CSRF: b.csrf, Write: true}
	body = selectionBody(list, []string{}, true)
	encoded, _ := json.Marshal(body)
	var in org.ExportRequest
	must(t, json.Unmarshal(encoded, &in))
	ctx, cancel := context.WithCancel(f.ctx)
	done := make(chan error, 1)
	go func() {
		out, e := service.ExportSelection(ctx, creds, in)
		if len(out.Data) != 0 {
			done <- foundation.Fail(500, "TEST", "cancel returned data")
			return
		}
		done <- e
	}()
	<-entered
	start := time.Now()
	cancel()
	if e = <-done; e == nil {
		t.Fatal("cancel accepted")
	}
	if time.Since(start) > time.Second {
		t.Fatal("cancel failed to release promptly")
	}
	// An exclusive policy writer must now acquire the guard rather than hang.
	tx, e := f.admin.Begin(f.ctx)
	must(t, e)
	defer tx.Rollback(f.ctx)
	_, e = tx.Exec(f.ctx, "SET LOCAL lock_timeout='500ms'")
	must(t, e)
	_, e = tx.Exec(f.ctx, "SELECT principal_id FROM access.policy_guard WHERE principal_id=$1 FOR UPDATE", b.subject)
	must(t, e)
	t.Logf("in-flight export cancellation and exclusive policy reacquisition: %s", time.Since(start))
}

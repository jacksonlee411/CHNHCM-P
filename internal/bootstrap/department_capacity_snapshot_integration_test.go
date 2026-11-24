//go:build integration

package bootstrap_test

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	dep "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	orgdb "chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestDepartmentListPinsFactsAcrossConcurrentParentCorrection(t *testing.T) {
	f := fresh(t)
	writer, reader := f.login("admin", f.password), f.login("admin", f.password)
	bu := writer.call("POST", "/api/v1/business-units", createBody("PINNED_BU", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	parent := writer.call("POST", "/api/v1/departments", departmentCreateBody("PINNED_PARENT", bu), foundation.NewID(), 201)
	body := departmentCreateBody("PINNED_CHILD", bu)
	body["parentDepartmentId"] = parent["departmentId"]
	child := writer.call("POST", "/api/v1/departments", body, foundation.NewID(), 201)
	var dictionaries []int32
	must(t, f.admin.QueryRow(f.ctx, `SELECT array_agg(DISTINCT dictionary_id) FROM organization.department_read_segment WHERE object_id IN ($1::uuid,$2::uuid)`, parent["departmentId"], child["departmentId"]).Scan(&dictionaries))
	svc := f.runtime.Department.(dep.Service)
	store := *svc.Store.(*orgdb.Store)
	reached, release := make(chan struct{}), make(chan struct{})
	barrier := &snapshotReadBarrier{prefix: history.DepartmentQuerySQL, reached: reached, release: release}
	cfg := store.Pool.Config().Copy()
	cfg.ConnConfig.Tracer = barrier
	pool, e := pgxpool.NewWithConfig(f.ctx, cfg)
	must(t, e)
	defer pool.Close()
	store.Pool = pool
	svc.Store = &store
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()
	u, e := url.Parse(f.server.URL)
	must(t, e)
	credentials := access.Credentials{Token: reader.client.Jar.Cookies(u)[0].Value}
	type reply struct {
		out org.DepartmentList
		err error
	}
	done := make(chan reply, 1)
	go func() {
		v, e := svc.List(context.Background(), credentials, org.DepartmentListInput{Page: 1, PageSize: 20})
		done <- reply{v, e}
	}()
	select {
	case <-reached:
	case <-time.After(3 * time.Second):
		t.Fatal("capture barrier not reached")
	}
	name := "并发更新后的父部门"
	writer.call("POST", "/api/v1/departments/"+parent["departmentId"].(string)+"/records/"+parent["targetRecordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": parent["sourceVersion"], "expectedRecordRevision": parent["targetRevision"], "changes": map[string]any{"name": name}}, foundation.NewID(), 200)
	// Changing the child's BU and parent after capture must not splice new scope
	// or ancestry into the old page. Both replaced read dictionaries are reclaimed.
	otherBU := writer.call("POST", "/api/v1/business-units", createBody("PINNED_OTHER_BU", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	writer.call("POST", "/api/v1/departments/"+child["departmentId"].(string)+"/records/"+child["targetRecordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": child["sourceVersion"], "expectedRecordRevision": child["targetRevision"], "changes": map[string]any{"managementBuId": otherBU, "parentDepartmentId": nil}}, foundation.NewID(), 200)
	writer.call("POST", "/api/v1/departments", departmentCreateBody("PINNED_NEWCOMER", bu), foundation.NewID(), 201)
	deleted, e := f.admin.Exec(f.ctx, `DELETE FROM organization.department_dictionary WHERE id=ANY($1::int[])`, dictionaries)
	must(t, e)
	if deleted.RowsAffected() == 0 {
		t.Fatal("no captured dictionary was actually reclaimed")
	}
	unblock()
	old := <-done
	must(t, old.err)
	if old.out.Count != 2 || len(old.out.Items) != 2 {
		t.Fatal("count mixed concurrent addition", old.out.Count)
	}
	raw, e := json.Marshal(old.out)
	must(t, e)
	if strings.Contains(string(raw), "selectionVersion") || strings.Contains(string(raw), otherBU) {
		t.Fatal("old page leaked new ownership or collection digest")
	}

	var oldPath string
	for _, row := range old.out.Items {
		if row.ID == child["departmentId"] {
			if row.Path.Path == nil {
				t.Fatal("missing pinned path")
			}
			oldPath = *row.Path.Path
		}
	}
	if oldPath == "" || strings.Contains(oldPath, name) || !strings.Contains(oldPath, "研发部门") {
		t.Fatal("mixed parent sources", oldPath)
	}
	// A later request must use the new head while the unchanged child's fact head
	// remains identical; no head hash or global cache can hide this dependency.
	current := writer.call("GET", "/api/v1/departments?page=1&pageSize=20", nil, "", 200)
	found := false
	for _, v := range current["items"].([]any) {
		row := v.(map[string]any)
		if row["id"] == child["departmentId"] {
			found = true
			if strings.Contains(row["path"].(map[string]any)["departmentPath"].(string), name) || row["managementBu"].(map[string]any)["id"] != otherBU {
				t.Fatal("new parent source was not observed", row)
			}
		}
	}
	if _, exists := current["selectionVersion"]; !found || exists || current["count"] != float64(3) {
		t.Fatal("collection failed to reflect parent correction")
	}
	t.Log("formal capture retained count, path and current scope through concurrent rename, BU transfer, parent change, addition and actual dictionary deletion; next request observed commits; GET has no selection version")
}

//go:build integration

package bootstrap_test

import (
	"context"
	"fmt"
	"net/url"
	"testing"
	"time"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

func TestDepartmentParent133CursorQualifiedBoundaryAndOwnership(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	other := f.login("admin", f.password)
	bu := b.call("POST", "/api/v1/business-units", createBody("P133_BU", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	ids := []string{}
	for i := 0; i < 25; i++ {
		id := b.call("POST", "/api/v1/departments", departmentCreateBody(fmt.Sprintf("P133_%02d", i), bu), foundation.NewID(), 201)["departmentId"].(string)
		ids = append(ids, id)
	}
	in := org.DepartmentParentQuery{Context: org.DepartmentReferenceContext{Mode: "create", ManagementBUID: bu, EffectiveDate: "2026-01-01"}, PageSize: 20}
	origin, _ := url.Parse(f.server.URL)
	// Calling the real application once preserves the internal SQL error if
	// capture fails; assertions and subsequent pages use the actual HTTP route.
	qualifications := 0
	observed := foundation.WithStageObserver(f.ctx, func(name string, _ time.Duration) {
		if name == "references.qualify" {
			qualifications++
		}
	})
	first, err := f.runtime.Department.Parents(observed, foundation.Credentials{Token: b.client.Jar.Cookies(origin)[0].Value}, in)
	must(t, err)
	if len(first.Items) != 20 || !first.HasMore || first.NextCursor == "" {
		t.Fatal("qualified page boundary", len(first.Items), first.HasMore)
	}
	if qualifications != 21 {
		t.Fatal("dense qualification did not stop at 21", qualifications)
	}
	for i, item := range first.Items {
		if item.ID != ids[i] {
			t.Fatal("stable ordering", i)
		}
	}
	in.Cursor = first.NextCursor
	second := b.call("POST", "/api/v1/departments/references/parents", in, "", 200)
	if second["count"] != nil || second["page"] != nil || second["hasMore"] != false || len(second["items"].([]any)) != 5 {
		t.Fatal("cursor page contract", second)
	}
	if second["items"].([]any)[0].(map[string]any)["id"] != ids[20] {
		t.Fatal("lookahead member skipped")
	}
	other.call("POST", "/api/v1/departments/references/parents", in, "", 409)
	in.Cursor = first.NextCursor + "x"
	b.call("POST", "/api/v1/departments/references/parents", in, "", 409)
	in.Cursor = first.NextCursor
	in.Q = "no match"
	b.call("POST", "/api/v1/departments/references/parents", in, "", 409)
	in.Cursor = ""
	empty := b.call("POST", "/api/v1/departments/references/parents", in, "", 200)
	if empty["hasMore"] != false || len(empty["items"].([]any)) != 0 {
		t.Fatal(empty)
	}
}

type parent133Store struct {
	app.Store
	afterCommon func()
}
type parent133Unit struct {
	app.Unit
	afterCommon func()
}

func (s parent133Store) WithinDepartment(ctx context.Context, c access.Credentials, fn func(app.Unit) error) error {
	return s.Store.WithinDepartment(ctx, c, func(u app.Unit) error { return fn(parent133Unit{u, s.afterCommon}) })
}
func (u parent133Unit) WalkParents(ctx context.Context, q app.ParentRequest, visit func(app.ParentFrame) (bool, error)) error {
	return u.Unit.WalkParents(ctx, q, func(frame app.ParentFrame) (bool, error) {
		more, err := visit(frame)
		if frame.Common && err == nil && u.afterCommon != nil {
			u.afterCommon()
		}
		return more, err
	})
}

func TestDepartmentParent133SparseFetchUsesOneSnapshotAndFutureCoverage(t *testing.T) {
	f := fresh(t)
	reader := f.login("admin", f.password)
	writer := f.login("admin", f.password)
	bu := writer.call("POST", "/api/v1/business-units", createBody("P133_SPARSE_BU", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	for i := 0; i < 65; i++ {
		body := departmentCreateBody(fmt.Sprintf("P133_A_%02d", i), bu)
		body["effectiveDate"] = "2099-01-01"
		writer.call("POST", "/api/v1/departments", body, foundation.NewID(), 201)
	}
	var last map[string]any
	for i := 0; i < 21; i++ {
		body := departmentCreateBody(fmt.Sprintf("P133_B_%02d", i), bu)
		body["name"] = "快照原名"
		last = writer.call("POST", "/api/v1/departments", body, foundation.NewID(), 201)
	}
	stop := writer.call("POST", "/api/v1/departments", departmentCreateBody("P133_C_STOP", bu), foundation.NewID(), 201)
	writer.call("POST", "/api/v1/departments/"+stop["departmentId"].(string)+"/deactivate", map[string]any{"expectedSourceVersion": stop["sourceVersion"], "anchorRecordId": stop["targetRecordId"], "effectiveDate": "2099-01-01", "operationType": "DEACTIVATE", "operationReason": "ORG_RESTRUCTURING"}, foundation.NewID(), 201)
	service := f.runtime.Department.(app.Service)
	service.Store = parent133Store{Store: service.Store, afterCommon: func() {
		writer.call("POST", "/api/v1/departments/"+last["departmentId"].(string)+"/records/"+last["targetRecordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": last["sourceVersion"], "expectedRecordRevision": last["targetRevision"], "changes": map[string]any{"name": "后续新名"}}, foundation.NewID(), 200)
	}}
	q := org.DepartmentParentQuery{Context: org.DepartmentReferenceContext{Mode: "create", ManagementBUID: bu, EffectiveDate: "2026-01-01"}, PageSize: 50}
	origin, _ := url.Parse(f.server.URL)
	result, err := service.Parents(f.ctx, foundation.Credentials{Token: reader.client.Jar.Cookies(origin)[0].Value}, q)
	must(t, err)
	if len(result.Items) != 21 || result.HasMore {
		t.Fatal("sparse prefix or far-future stop broke qualification", len(result.Items), result.HasMore)
	}
	if *result.Items[20].Name != "快照原名" {
		t.Fatal("later FETCH mixed source snapshots")
	}
	next, err := f.runtime.Department.Parents(f.ctx, foundation.Credentials{Token: reader.client.Jar.Cookies(origin)[0].Value}, q)
	must(t, err)
	if *next.Items[20].Name != "后续新名" {
		t.Fatal("next request did not refresh facts")
	}
}

func TestDepartmentParent133MissingRequiredAncestorFails(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	bu := b.call("POST", "/api/v1/business-units", createBody("P133_MISSING_BU", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	parent := b.call("POST", "/api/v1/departments", departmentCreateBody("P133_ANCESTOR", bu), foundation.NewID(), 201)["departmentId"].(string)
	body := departmentCreateBody("P133_REQUIRED_CHILD", bu)
	body["parentDepartmentId"] = parent
	b.call("POST", "/api/v1/departments", body, foundation.NewID(), 201)
	query := map[string]any{"context": map[string]any{"mode": "create", "effectiveDate": "2026-01-01", "managementBuId": bu}, "q": "P133_REQUIRED_CHILD", "pageSize": 20}
	if len(b.call("POST", "/api/v1/departments/references/parents", query, "", 200)["items"].([]any)) != 1 {
		t.Fatal("required ancestor fixture")
	}
	_, err := f.admin.Exec(f.ctx, `DELETE FROM organization.department_read_segment WHERE instance_id=$1 AND object_id=$2`, f.instance, parent)
	must(t, err)
	result := b.call("POST", "/api/v1/departments/references/parents", query, "", 503)
	if result["items"] != nil {
		t.Fatal("missing required material became partial candidates")
	}
}

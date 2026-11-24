//go:build integration && query131

package bootstrap_test

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"net/url"
	"sync"
	"testing"
	"time"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

type parent131Store struct {
	app.Store
	before  func()
	after   func()
	capture *app.ParentCapture
}
type parent131Unit struct {
	app.Unit
	before  func()
	after   func()
	capture *app.ParentCapture
}

func (s parent131Store) WithinDepartment(ctx context.Context, c access.Credentials, fn func(app.Unit) error) error {
	return s.Store.WithinDepartment(ctx, c, func(u app.Unit) error { return fn(parent131Unit{u, s.before, s.after, s.capture}) })
}
func (u parent131Unit) WalkParents(ctx context.Context, q app.ParentRequest, visit func(app.ParentFrame) (bool, error)) error {
	if u.before != nil {
		u.before()
	}
	if u.capture != nil {
		*u.capture = app.ParentCapture{}
	}
	return u.Unit.WalkParents(ctx, q, func(frame app.ParentFrame) (bool, error) {
		if u.capture != nil {
			u.capture.Stats.Segments += frame.Capture.Stats.Segments
			u.capture.Stats.DecodedDepartments += frame.Capture.Stats.DecodedDepartments
		}
		if frame.Common && u.after != nil {
			u.after()
		}
		return visit(frame)
	})
}

// Exhaust the active production walker only for storage fault assertions.
type parent133Observed struct {
	Candidates []string
	Stats      app.QueryStats
}

func captureParent133(ctx context.Context, tx pgx.Tx, g access.Guard, q app.ParentRequest) (out parent133Observed, err error) {
	err = history.WalkDepartmentParents(ctx, tx, g, q, func(frame app.ParentFrame) (bool, error) {
		if !frame.Common {
			out.Candidates = append(out.Candidates, frame.ID)
		}
		out.Stats.Segments += frame.Capture.Stats.Segments
		out.Stats.DecodedDepartments += frame.Capture.Stats.DecodedDepartments
		return true, nil
	})
	if err != nil {
		out = parent133Observed{}
	}
	return
}

func TestDepartmentParent131CaptureWindowsAndSourceConflict(t *testing.T) {
	f, admin, _ := query131Fixture(t)
	bu := query131BU(t, admin, "P131_BU")
	p := query131Create(t, admin, "P131_PARENT", "old label", bu, "2020-01-01")
	for _, day := range []string{"2021-01-01", "2022-01-01", "2023-01-01", "2024-01-01", "2025-01-01"} {
		p = admin.call("POST", "/api/v1/departments/"+p["departmentId"].(string)+"/records", map[string]any{"expectedSourceVersion": p["sourceVersion"], "anchorRecordId": p["targetRecordId"], "effectiveDate": day, "operationType": "CHANGE", "operationReason": "NAME_ADJUSTMENT", "changes": map[string]any{"name": day}}, foundation.NewID(), 201)
	}
	service := f.runtime.Department.(app.Service)
	var captured app.ParentCapture
	service.Store = parent131Store{Store: service.Store, capture: &captured}
	origin, e := url.Parse(f.server.URL)
	must(t, e)
	cred := access.Credentials{Token: admin.client.Jar.Cookies(origin)[0].Value}
	in := org.DepartmentParentQuery{PageSize: 20, Context: org.DepartmentReferenceContext{Mode: "create", ManagementBUID: bu, EffectiveDate: "2026-01-01"}}
	out, e := service.Parents(f.ctx, cred, in)
	must(t, e)
	if len(out.Items) != 1 || out.Items[0].ID != p["departmentId"] {
		t.Fatal("covering predecessor lost", out)
	}
	if captured.Stats.DecodedDepartments != 0 || captured.Stats.Segments != 1 {
		t.Fatal("parent query loaded historical department material or unrelated periods", captured.Stats)
	}
	t.Logf("window capture: %+v", captured.Stats)
	// Search is based on today's authorized label, independently of W's day.
	in.Context.EffectiveDate = "2021-06-01"
	in.Q = "2025-01-01"
	out, e = service.Parents(f.ctx, cred, in)
	must(t, e)
	if len(out.Items) != 1 {
		t.Fatal("search used proposed-day label", out)
	}
	in.Q = "2021-01-01"
	out, e = service.Parents(f.ctx, cred, in)
	must(t, e)
	if len(out.Items) != 0 {
		t.Fatal("historical label leaked through search", out)
	}
	// Real committed update after the target preparation, before the capture.
	in = org.DepartmentParentQuery{PageSize: 20, Context: org.DepartmentReferenceContext{Mode: "maintain", DepartmentID: p["departmentId"].(string), ExpectedSourceVersion: p["sourceVersion"].(string), Action: "CORRECT", RecordID: p["targetRecordId"].(string), ExpectedRecordRevision: p["targetRevision"].(string)}}
	original := service.Store.(parent131Store).Store
	service.Store = parent131Store{Store: original, before: func() {
		admin.call("POST", "/api/v1/departments/"+p["departmentId"].(string)+"/records/"+p["targetRecordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": p["sourceVersion"], "expectedRecordRevision": p["targetRevision"], "changes": map[string]any{"name": "after preparation"}}, foundation.NewID(), 200)
	}}
	_, e = service.Parents(f.ctx, cred, in)
	var fault *foundation.Fault
	if !errors.As(e, &fault) || fault.Code != "M02_SOURCE_CONFLICT" {
		t.Fatal("old W rebound to new source", e)
	}
}

func TestDepartmentParent131CaptureFaults(t *testing.T) {
	f, admin, q := query131Fixture(t)
	bu := query131BU(t, admin, "P131_FAULT_BU")
	p := query131Create(t, admin, "P131_FAULT_PARENT", "parent", bu, "2020-01-01")
	for _, tc := range []struct{ name, sql string }{
		{"wrong_source", `UPDATE organization.department_read_segment SET source_id='00000000-0000-4000-8000-000000000001' WHERE object_id=$1`},
		{"bad_edge", `UPDATE organization.department_read_segment SET q_parent='00000000-0000-4000-8000-000000000003' WHERE object_id=$1`},
		{"missing_bu", `DELETE FROM organization.bu_read_segment WHERE object_id=$1`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx, g, _ := query131Guard(t, f, admin, q)
			defer tx.Rollback(context.Background())
			id := p["departmentId"].(string)
			if tc.name == "missing_bu" {
				id = bu
			}
			_, e := tx.Exec(f.ctx, tc.sql, id)
			must(t, e)
			at, e := time.Parse(time.RFC3339, q.Day+"T04:00:00Z")
			must(t, e)
			plan, e := g.(access.ReadPlanner).PlanReads(f.ctx, q.Enterprise, at, access.DepartmentReference)
			must(t, e)
			base := d.Timeline{ID: foundation.NewID(), Enterprise: q.Enterprise, Records: []d.Record{{ID: foundation.NewID(), Date: "2026-01-01", State: "active", Data: d.Data{BU: bu}}}}
			in := app.ParentRequest{Plan: plan, BU: bu, Base: base, Windows: []d.ParentWindow{{Start: "2026-01-01", BU: bu}}}
			got, e := captureParent133(f.ctx, tx, g, in)
			var fault *foundation.Fault
			if !errors.As(e, &fault) || fault.Status != 503 || len(got.Candidates) != 0 {
				t.Fatal("corrupt capture became empty/partial success", e, got.Stats)
			}
		})
	}
	// Every qualifier must be present, including a far-future boundary.
	p = admin.call("POST", "/api/v1/departments/"+p["departmentId"].(string)+"/deactivate", map[string]any{"expectedSourceVersion": p["sourceVersion"], "anchorRecordId": p["targetRecordId"], "effectiveDate": "2099-01-01", "operationType": "DEACTIVATE", "operationReason": "ORG_RESTRUCTURING"}, foundation.NewID(), 201)
	out := admin.call("POST", "/api/v1/departments/references/parents", map[string]any{"context": map[string]any{"mode": "create", "managementBuId": bu, "effectiveDate": "2026-01-01"}, "pageSize": 20}, "", 200)
	if len(out["items"].([]any)) != 0 {
		t.Fatal("far future stop omitted", out)
	}
}

func TestDepartmentParent131QueryDoesNotAuditUnrelatedStorage(t *testing.T) {
	f, admin, q := query131Fixture(t)
	bu := query131BU(t, admin, "P131_SCOPE_BU")
	otherBU := query131BU(t, admin, "P131_OTHER_BU")
	parent := query131Create(t, admin, "P131_SCOPE_PARENT", "available parent", bu, "2020-01-01")
	other := query131Create(t, admin, "P131_OTHER_PARENT", "unrelated department", otherBU, "2020-01-01")
	for _, tc := range []struct{ name, sql string }{
		{"unused_projection", `UPDATE organization.department_read_segment SET q_name='outside query',q_seal=decode(repeat('00',32),'hex') WHERE object_id=$1`},
		{"unused_segments", `DELETE FROM organization.department_read_segment WHERE object_id=$1`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx, g, _ := query131Guard(t, f, admin, q)
			defer tx.Rollback(context.Background())
			_, e := tx.Exec(f.ctx, tc.sql, other["departmentId"])
			must(t, e)
			at, e := time.Parse(time.RFC3339, q.Day+"T04:00:00Z")
			must(t, e)
			plan, e := g.(access.ReadPlanner).PlanReads(f.ctx, q.Enterprise, at, access.DepartmentReference)
			must(t, e)
			base := d.Timeline{ID: foundation.NewID(), Enterprise: q.Enterprise, Records: []d.Record{{ID: foundation.NewID(), Date: "2026-01-01", State: "active", Data: d.Data{BU: bu}}}}
			got, e := captureParent133(f.ctx, tx, g, app.ParentRequest{Plan: plan, BU: bu, Base: base, Windows: []d.ParentWindow{{Start: "2026-01-01", BU: bu}}})
			must(t, e)
			if len(got.Candidates) != 1 || got.Candidates[0] != parent["departmentId"] || got.Stats.DecodedDepartments != 0 || got.Stats.Segments != 1 {
				t.Fatal("unrelated storage changed the parent query", got.Candidates, got.Stats)
			}
		})
	}
}

func TestDepartmentParent131FutureFirstUsesProposedDateAndCurrentLabel(t *testing.T) {
	_, admin, _ := query131Fixture(t)
	bu := query131BU(t, admin, "P131_FUTURE_BU")
	parent := query131Create(t, admin, "P131_FUTURE_PARENT", "future parent", bu, "2098-01-01")
	for _, tc := range []struct {
		day   string
		count float64
	}{{"2026-01-01", 0}, {"2098-01-01", 1}} {
		out := admin.call("POST", "/api/v1/departments/references/parents", map[string]any{"context": map[string]any{"mode": "create", "managementBuId": bu, "effectiveDate": tc.day}, "pageSize": 20}, "", 200)
		if float64(len(out["items"].([]any))) != tc.count {
			t.Fatal("proposed date qualification changed", tc.day, out)
		}
		if tc.count != 0 {
			item := out["items"].([]any)[0].(map[string]any)
			if item["id"] != parent["departmentId"] || item["name"] != "future parent" {
				t.Fatal("future-first current label lost", item)
			}
		}
	}
}

func TestDepartmentParent131CapturedFactsSurviveConcurrentWrites(t *testing.T) {
	f, writer, _ := query131Fixture(t)
	reader := f.login("admin", f.password)
	bu := query131BU(t, writer, "P131_PIN_BU")
	parent := query131Create(t, writer, "P131_PIN_PARENT", "before capture", bu, "2020-01-01")
	service := f.runtime.Department.(app.Service)
	origin, e := url.Parse(f.server.URL)
	must(t, e)
	cred := access.Credentials{Token: reader.client.Jar.Cookies(origin)[0].Value}
	in := org.DepartmentParentQuery{PageSize: 20, Context: org.DepartmentReferenceContext{Mode: "create", ManagementBUID: bu, EffectiveDate: "2026-01-01"}}
	captured, release := make(chan struct{}), make(chan struct{})
	original := service.Store
	service.Store = parent131Store{Store: original, after: func() { close(captured); <-release }}
	type answer struct {
		out org.DepartmentParents
		err error
	}
	done := make(chan answer, 1)
	go func() { v, e := service.Parents(f.ctx, cred, in); done <- answer{v, e} }()
	select {
	case <-captured:
	case <-time.After(3 * time.Second):
		close(release)
		t.Fatal("actual capture barrier not reached")
	}
	// Release even if the writer assertion fails, so cleanup never leaks a read.
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()
	writer.call("POST", "/api/v1/departments/"+parent["departmentId"].(string)+"/records/"+parent["targetRecordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": parent["sourceVersion"], "expectedRecordRevision": parent["targetRevision"], "changes": map[string]any{"name": "after capture"}}, foundation.NewID(), 200)
	query131Create(t, writer, "P131_PIN_NEW", "new member", bu, "2020-01-01")
	unblock()
	answerValue := <-done
	must(t, answerValue.err)
	if len(answerValue.out.Items) != 1 || *answerValue.out.Items[0].Name != "before capture" {
		t.Fatal("mixed facts or labels after capture", answerValue.out)
	}
	service.Store = original
	current, e := service.Parents(f.ctx, cred, in)
	must(t, e)
	if len(current.Items) != 2 {
		t.Fatal("subsequent capture did not observe commit", current)
	}
	t.Log("actual cursor first-frame barrier retained original labels/qualification; later request observed committed rename and member")
}

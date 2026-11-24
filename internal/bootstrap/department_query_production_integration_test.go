//go:build integration && query131

package bootstrap_test

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"
	"time"

	"chnhcm.local/chnhcm/internal/bootstrap"
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
)

// These wrappers observe the formal capture port. They do not supply facts or
// authorization decisions: both the real store and application projector run.
type query131ObservedStore struct {
	app.Store
	stats *app.QueryStats
}

func (s query131ObservedStore) WithinDepartment(ctx context.Context, c access.Credentials, fn func(app.Unit) error) error {
	return s.Store.WithinDepartment(ctx, c, func(u app.Unit) error { return fn(query131ObservedUnit{u, s.stats}) })
}

type query131ObservedUnit struct {
	app.Unit
	stats *app.QueryStats
}

func (u query131ObservedUnit) CaptureQuery(ctx context.Context, q app.QueryRequest) (app.QueryCapture, error) {
	v, e := u.Unit.CaptureQuery(ctx, q)
	*u.stats = v.Stats
	return v, e
}

func (u query131ObservedUnit) WalkQuery(ctx context.Context, q app.QueryRequest, visit func(app.QueryFrame) (bool, error)) error {
	*u.stats = app.QueryStats{}
	return u.Unit.WalkQuery(ctx, q, func(frame app.QueryFrame) (bool, error) {
		u.stats.Objects += frame.Capture.Stats.Objects
		u.stats.Segments += frame.Capture.Stats.Segments
		u.stats.MaterialBytes += frame.Capture.Stats.MaterialBytes
		u.stats.DecodedDepartments += frame.Capture.Stats.DecodedDepartments
		return visit(frame)
	})
}

func query131ProductRead(t *testing.T, f *fixture, b *browser, q history.Query131Input, in org.DepartmentListInput) (history.Query131Result, history.Query131Stats, error) {
	t.Helper()
	at, e := time.Parse(time.RFC3339, q.Day+"T04:00:00Z")
	must(t, e)
	restore := bootstrap.SetIntegrationClock(f.runtime, func() time.Time { return at })
	defer restore()
	service := f.runtime.Department.(app.Service)
	var work app.QueryStats
	service.Store = query131ObservedStore{service.Store, &work}
	origin, e := url.Parse(f.server.URL)
	must(t, e)
	in.Page, in.PageSize = q.Page, q.PageSize
	page, e := service.List(f.ctx, access.Credentials{Token: b.client.Jar.Cookies(origin)[0].Value}, in)
	stats := history.Query131Stats{Objects: work.Objects, Segments: work.Segments, Dictionaries: work.Dictionaries, Shapes: work.Shapes, ResponseBytes: work.MaterialBytes, DecodedPageSegments: work.DecodedDepartments}
	if e != nil {
		return history.Query131Result{}, stats, e
	}
	raw, e := json.Marshal(page)
	must(t, e)
	var dto map[string]any
	must(t, json.Unmarshal(raw, &dto))
	if _, present := dto["selectionVersion"]; present {
		t.Fatal("ordinary GET payload computes a selection version")
	}
	result := history.Query131Result{Count: int64(page.Count)}
	for _, v := range dto["items"].([]any) {
		row := v.(map[string]any)
		if path, ok := row["path"].(map[string]any); ok {
			row["departmentPath"], row["pathDay"], row["pathMode"] = path["departmentPath"], path["pathBusinessAt"], path["pathMode"]
		}
		result.Items = append(result.Items, row)
	}
	return result, stats, nil
}

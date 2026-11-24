//go:build integration && query131

package bootstrap_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	app "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

func TestDepartmentQuery131IndependentPagesAndGrammar(t *testing.T) {
	f, admin, q := query131Fixture(t)
	bu := query131BU(t, admin, "PAGE_BU")
	query := org.DepartmentListInput{Sort: "code", Status: "all"}
	v, _, e := query131PlanRead(t, f, admin, q, query)
	must(t, e)
	if v.Count != 0 || len(v.Items) != 0 {
		t.Fatal(v)
	}
	ids := []string{}
	checks := 1
	for _, count := range []int{1, 20, 21, 137} {
		for len(ids) < count {
			i := len(ids)
			body := departmentCreateBody(fmt.Sprintf("PAGE_%03d", i), bu)
			body["name"], body["effectiveDate"] = fmt.Sprintf("N%03d Ω_%%中文𠀀", 136-i), "2026-01-01"
			if i == 136 {
				body["shortName"] = "SHORT_%"
			}
			created := admin.call("POST", "/api/v1/departments", body, foundation.NewID(), 201)
			ids = append(ids, created["departmentId"].(string))
		}
		for _, size := range []int{20, 50, 100} {
			for _, order := range []string{"code", "name", "name_desc"} {
				query.Sort = order
				for page := 1; page <= (count+size-1)/size+1; page++ {
					q.Page, q.PageSize = page, size
					v, stats, e := query131PlanRead(t, f, admin, q, query)
					must(t, e)
					start, end := min((page-1)*size, count), min(page*size, count)
					if v.Count != int64(count) || len(v.Items) != end-start || stats.DecodedPageSegments != end-start {
						t.Fatal("count/page work", count, size, page, v, stats)
					}
					for j, item := range v.Items {
						i := start + j
						if order == "name" {
							i = count - 1 - i
						}
						name := fmt.Sprintf("N%03d Ω_%%中文𠀀", 136-i)
						if item["id"] != ids[i] || item["name"] != name || item["effectiveDate"] != "2026-01-01" || item["status"] != "active" || item["departmentPath"] != ""+name {
							t.Fatal("literal expected row", i, item)
						}
					}
					checks++
				}
			}
		}
	}
	q.Page, q.PageSize, query.Sort = 200000000, 20, "code"
	v, _, e = query131PlanRead(t, f, admin, q, query)
	must(t, e)
	if v.Count != 137 || len(v.Items) != 0 {
		t.Fatal("valid large page must preserve count and return an empty page", v)
	}
	checks++
	q.Page = 1
	for _, tc := range []struct {
		filter org.ColumnFilter
		count  int64
	}{
		{org.ColumnFilter{Field: "name", Operator: "contains", Value: "  Ω_%中文𠀀  "}, 137},
		{org.ColumnFilter{Field: "code", Operator: "equals", Value: "  page_136  "}, 1},
		{org.ColumnFilter{Field: "shortName", Operator: "equals", Value: " short_% "}, 1},
		{org.ColumnFilter{Field: "departmentPath", Operator: "contains", Value: " N000 Ω_% "}, 1},
		{org.ColumnFilter{Field: "effectiveDate", Operator: "before", Value: "2026-01-01"}, 0},
		{org.ColumnFilter{Field: "effectiveDate", Operator: "onOrBefore", Value: "2026-01-01"}, 137},
		{org.ColumnFilter{Field: "managementBuId", Operator: "equals", Value: bu}, 137},
		{org.ColumnFilter{Field: "status", Operator: "isAnyOf", Value: "inactive,active,active"}, 137},
	} {
		query.Filter = &tc.filter
		v, _, e = query131PlanRead(t, f, admin, q, query)
		must(t, e)
		if tc.filter.Field == "departmentPath" && (v.Count != -1 || int64(len(v.Items)) != tc.count) || tc.filter.Field != "departmentPath" && v.Count != tc.count {
			t.Fatal(tc.filter, v)
		}
		checks++
	}
	for _, filter := range []*org.ColumnFilter{
		{Field: "departmentPath", Operator: "unknown", Value: "x"},
		{Field: "name", Operator: "contains", Value: " "},
		{Field: "managementBuId", Operator: "contains", Value: bu},
		{Field: "effectiveDate", Operator: "is", Value: "2026-02-30"},
	} {
		query.Filter = filter
		v, _, e = query131PlanRead(t, f, admin, q, query)
		var fault *foundation.Fault
		if !errors.As(e, &fault) || fault.Status != 400 || v.Count != 0 {
			t.Fatal("invalid grammar accepted", filter, v, e)
		}
		checks++
	}
	tx, g, in := query131Guard(t, f, admin, q)
	defer tx.Rollback(context.Background())
	in.Query = org.DepartmentListInput{Sort: "code", Status: "all"}
	var plan []byte
	must(t, f.admin.QueryRow(f.ctx, "EXPLAIN (ANALYZE,BUFFERS,FORMAT JSON) "+history.Query131PlannedSQL, in.Args()...).Scan(&plan))
	query131Log(t, "Q131_PLANNED_EXPLAIN", json.RawMessage(plan))
	_, _, e = history.CapturePlanned131(f.ctx, tx, g, in)
	must(t, e)
	t.Logf("%d independent literal count/order/page/filter checks; 0/1/20/21/137 objects, pages 20/50/100, all three sorts", checks)
}

func TestDepartmentQuery131DeepCycleAndCancellation(t *testing.T) {
	f, admin, q := query131Fixture(t)
	bu := query131BU(t, admin, "DEEP_BU")
	parent, root := "", ""
	names := []string{"本地验证企业"}
	for i := 0; i < 100; i++ {
		name := fmt.Sprintf("depth-%03d", i)
		body := departmentCreateBody(fmt.Sprintf("DEPTH_%03d", i), bu)
		body["name"], body["effectiveDate"] = name, "2026-01-01"
		if parent != "" {
			body["parentDepartmentId"] = parent
		}
		r := admin.call("POST", "/api/v1/departments", body, foundation.NewID(), 201)
		parent = r["departmentId"].(string)
		if i == 0 {
			root = parent
		}
		names = append(names, name)
	}
	query := org.DepartmentListInput{Q: "depth-099", Sort: "code", Status: "all"}
	v, stats, e := query131PlanRead(t, f, admin, q, query)
	must(t, e)
	if v.Count != 1 || stats.DecodedPageSegments != 100 || v.Items[0]["departmentPath"] != strings.Join(names, " / ") {
		t.Fatal("deep path truncated", v, stats)
	}
	t.Run("cycle", func(t *testing.T) {
		tx, g, in := query131Guard(t, f, admin, q)
		defer tx.Rollback(context.Background())
		in.Query = query
		_, e := tx.Exec(f.ctx, `UPDATE organization.department_read_segment SET q_parent=$1 WHERE object_id=$2`, parent, root)
		must(t, e)

		var captured history.CapturedPlan131
		must(t, captured.Scan(tx.QueryRow(f.ctx, history.Query131PlannedSQL, in.Args()...)))
		var payload struct{ Cycle bool }
		must(t, json.Unmarshal(captured.Raw, &payload))
		if captured.Stats.Bad != 0 || !payload.Cycle {
			t.Fatal("recursive cycle barrier untested", captured.Stats, payload)
		}
		v, _, e := history.ProjectCaptured131(f.ctx, g, in, captured)
		var fault *foundation.Fault
		if !errors.As(e, &fault) || fault.Status != 503 || v.Count != 0 || len(v.Items) != 0 {
			t.Fatal("cycle returned partial success", v, e)
		}
		// The full-selection closure terminates with UNION by identity/date.
		// It consumes the required original path facts without auditing whether
		// privileged SQL has replaced the redundant query edge.
		ctx, cancel := context.WithTimeout(f.ctx, 3*time.Second)
		defer cancel()
		full, e := history.CaptureDepartmentQuery(ctx, tx, g, app.QueryRequest{Plan: in.Plan, Input: org.DepartmentListInput{Q: query.Q, Page: 1, PageSize: 20}, Extent: app.QuerySelection})
		must(t, e)
		if full.Count != 1 || len(full.PageIDs) != 1 || ctx.Err() != nil {
			t.Fatal("full capture did not terminate with the original path facts", full.Stats, e)
		}

	})
	t.Run("cancel", func(t *testing.T) {
		tx, _, in := query131Guard(t, f, admin, q)
		defer tx.Rollback(context.Background())
		in.Query = query
		_, e := tx.Exec(f.ctx, `SELECT set_config('application_name','q131-cancel',true)`)
		must(t, e)
		gate, e := f.admin.Begin(f.ctx)
		must(t, e)
		defer gate.Rollback(context.Background())
		_, e = gate.Exec(f.ctx, `SELECT pg_advisory_xact_lock(1310093)`)
		must(t, e)
		sql := strings.Replace(history.Query131PlannedSQL, "FROM organization.department d\n", "FROM organization.department d CROSS JOIN (SELECT pg_advisory_xact_lock(1310093)) gate\n", 1)
		ctx, cancel := context.WithTimeout(f.ctx, 10*time.Second)
		defer cancel()
		done := make(chan error, 1)
		go func() { var c history.CapturedPlan131; done <- c.Scan(tx.QueryRow(ctx, sql, in.Args()...)) }()
		query131WaitLock(t, f, "q131-cancel")
		cancel()
		if e := <-done; !errors.Is(e, context.Canceled) {
			t.Fatal("capture ignored cancellation", e)
		}
	})
	t.Log("depth 100 captured without cutoff; recursive query cycle terminates; blocked capture cancelled without partial page")
}

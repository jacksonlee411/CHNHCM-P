//go:build integration && query131

package history

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	"chnhcm.local/chnhcm/internal/contexts/organization/domain/dated"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
)

// Exercise the same capture statement as the formal list port.
var Query131PlannedSQL = DepartmentQuerySQL

type PlannedQuery131 struct {
	Base  Query131Input
	Plan  access.ReadPlan
	Query org.DepartmentListInput
}

func (q PlannedQuery131) Args() []any {
	filter, operator, value := "", "", ""
	if q.Query.Filter != nil {
		filter, operator, value = q.Query.Filter.Field, q.Query.Filter.Operator, q.Query.Filter.Value
	}
	query, _ := json.Marshal(map[string]string{"q": strings.ToLower(strings.TrimSpace(q.Query.Q)), "sort": q.Query.Sort, "status": q.Query.Status, "field": filter, "operator": operator, "value": value})
	plan, _ := json.Marshal(q.Plan)
	return []any{q.Base.Instance, q.Base.Enterprise, q.Base.Day, plan, query, q.Base.PageSize, (q.Base.Page - 1) * q.Base.PageSize, false}
}

type CapturedPlan131 struct {
	Stats         Query131Stats
	Denied, Count int64
	Raw           []byte
}

func (c *CapturedPlan131) Scan(row pgx.Row) error {
	return row.Scan(&c.Stats.Objects, &c.Stats.Segments, &c.Stats.Bad, &c.Stats.Dictionaries, &c.Stats.Shapes, &c.Denied, &c.Count, &c.Raw)
}

func CapturePlanned131(ctx context.Context, tx pgx.Tx, g access.Guard, q PlannedQuery131) (Query131Result, Query131Stats, error) {
	var c CapturedPlan131
	var err error
	if q, err = q.normalized(); err != nil {
		return Query131Result{}, c.Stats, err
	}
	if err := q.validate(g); err != nil {
		return Query131Result{}, c.Stats, err
	}
	if err := c.Scan(tx.QueryRow(ctx, Query131PlannedSQL, q.Args()...)); err != nil {
		return Query131Result{}, c.Stats, err
	}
	return ProjectCaptured131(ctx, g, q, c)
}

func (q PlannedQuery131) normalized() (PlannedQuery131, error) {
	q.Query.Page, q.Query.PageSize = q.Base.Page, q.Base.PageSize
	v, err := app.NormalizeQuery(q.Query)
	q.Query = v
	return q, err
}

func (q PlannedQuery131) validate(g access.Guard) error {
	planner, ok := g.(access.ReadPlanner)
	if !ok || q.Plan.Origin != planner || q.Plan.Instance != q.Base.Instance || q.Plan.Enterprise != q.Base.Enterprise || q.Plan.Day != q.Base.Day || q.Plan.Action != access.DepartmentRead {
		return errors.New("query plan guard/date/identity mismatch")
	}
	if !q.Plan.Allowed.All && len(q.Plan.Allowed.BUs) == 0 && len(q.Plan.Allowed.Departments) == 0 {
		return foundation.Fail(403, "ACCESS_DENIED", "无部门读取权限")
	}
	if !foundation.ValidDate(q.Base.Day) || q.Base.Page < 1 || !slices.Contains([]int{20, 50, 100}, q.Base.PageSize) || q.Base.Page > int(^uint(0)>>1)/q.Base.PageSize || !slices.Contains([]string{"code", "name", "name_desc"}, q.Query.Sort) || !slices.Contains([]string{"all", "active", "inactive", "not_effective"}, q.Query.Status) {
		return errors.New("invalid planned query")
	}
	return nil
}

// ProjectCaptured131 uses only captured facts and the originating policy guard.
// It remains usable after current rows/dictionaries are replaced or reclaimed.
func ProjectCaptured131(ctx context.Context, g access.Guard, q PlannedQuery131, c CapturedPlan131) (out Query131Result, stats Query131Stats, err error) {
	stats = c.Stats
	stats.ResponseBytes = len(c.Raw)
	if q, err = q.normalized(); err != nil {
		return out, stats, err
	}
	bad := func() (Query131Result, Query131Stats, error) {
		return Query131Result{}, stats, foundation.Fail(503, "M02_DEPENDENCY_UNAVAILABLE", "部门列表暂时无法加载")
	}
	if err = q.validate(g); err != nil {
		return out, stats, err
	}
	if c.Stats.Bad != 0 {
		return bad()
	}
	if c.Denied != 0 {
		return out, stats, foundation.Fail(403, "ACCESS_DENIED", "无权使用该筛选字段")
	}
	var payload struct {
		Cycle, Missing   bool
		Enterprise       capturedEnterprise `json:"enterpriseProfile"`
		Page             []string
		Departments, BUs []map[string]any
	}
	if json.Unmarshal(c.Raw, &payload) != nil || payload.Cycle || payload.Missing {
		return bad()
	}
	enterpriseName, e := decodeQueryEnterpriseName(ctx, q.Base.Enterprise, q.Base.Day, payload.Enterprise)
	if e != nil {
		return bad()
	}
	byDay := map[string]d.Snapshot{}
	text := func(item map[string]any, k string) string { v, _ := item[k].(string); return v }
	for _, item := range payload.Departments {
		day := text(item, "queryDay")
		raw, e := json.Marshal(item)
		if e != nil {
			return bad()
		}
		var captured capturedDepartment
		if json.Unmarshal(raw, &captured) != nil {
			return bad()
		}
		t, e := decodeQueryDepartment(ctx, q.Base.Enterprise, captured)
		if e != nil {
			return bad()
		}

		s := byDay[day]
		s.Departments = append(s.Departments, t)
		byDay[day] = s
		stats.DecodedPageSegments++
	}
	for _, item := range payload.BUs {
		if item["valid"] != true || text(item, "enterprise") != q.Base.Enterprise {
			return bad()
		}
		raw, e := json.Marshal(item)
		if e != nil {
			return bad()
		}
		var captured capturedBU
		if json.Unmarshal(raw, &captured) != nil || captured.Source != captured.SegmentSource {
			return bad()
		}
		values, data, e := queryValues(captured.IDs, captured.Material)
		if e != nil {
			return bad()
		}
		decoded, e := DecodeBURead(ctx, values, data)
		if e != nil {
			return bad()
		}
		b := decoded.Timeline
		if b.ID != text(item, "id") || b.Source != text(item, "source") || len(b.Records) == 0 || b.First != b.Records[0].ID {
			return bad()
		}
		b.Code = text(item, "code")
		day := text(item, "day")
		s := byDay[day]
		s.BusinessUnits = append(s.BusinessUnits, b)
		byDay[day] = s
	}
	for day, s := range byDay {
		byDay[day] = s.ReadIndexed()
	}
	current := byDay[q.Base.Day]
	planner := g.(access.ReadPlanner)
	refPlan, e := planner.PlanReads(ctx, q.Base.Enterprise, q.Plan.At, access.DepartmentReference)
	if e != nil {
		return out, stats, e
	}
	buPlan, e := planner.PlanReads(ctx, q.Base.Enterprise, q.Plan.At, access.BUReference)
	if e != nil {
		return out, stats, e
	}
	decide := func(id string, plan access.ReadPlan) (access.Decision, error) {
		t := current.Department(id)
		if t == nil {
			return access.Decision{}, errors.New("missing current authorization fact")
		}
		r := d.ScopeRecord(*t, q.Base.Day)
		if r == nil {
			return access.Decision{}, errors.New("missing current member")
		}
		mode := "current_record"
		if q.Base.Day < t.Records[0].Date {
			mode = "future_first"
		}
		return planner.DecidePlan(ctx, plan, access.ReadIntent{Action: string(plan.Action), Kind: "object", EnterpriseID: q.Base.Enterprise, DepartmentTarget: &access.DepartmentTarget{EnterpriseID: q.Base.Enterprise, DepartmentID: id, ScopeFact: access.ScopeFact{EvaluatedAt: q.Plan.At, BusinessDate: q.Base.Day, Mode: mode, ManagementBUID: r.Data.BU, CurrentSourceVersion: t.Source, RecordID: r.ID, Revision: r.Revision}}})
	}
	out.Items = []map[string]any{}
	for _, id := range payload.Page {
		if e = ctx.Err(); e != nil {
			return Query131Result{}, stats, e
		}
		t := current.Department(id)
		if t == nil {
			return bad()
		}
		r := d.ScopeRecord(*t, q.Base.Day)
		if r == nil {
			return bad()
		}
		decision, e := decide(id, q.Plan)
		if e != nil || !decision.Allowed {
			return bad()
		}
		has := func(field string) bool { return slices.Contains(decision.ReadFields, field) }
		item := map[string]any{"id": id, "pathMode": "restricted"}
		for field, value := range map[string]string{"code": t.Code, "name": r.Data.Name, "effectiveDate": r.Date, "status": d.Status(*t, q.Base.Day), "managementBuId": r.Data.BU} {
			if has(field) {
				item[field] = value
			}
		}
		if has("shortName") {
			item["shortName"] = r.Data.ShortName
		}
		if has("managementBuId") {
			bd, e := planner.DecidePlan(ctx, buPlan, access.ReadIntent{Action: "bu.reference.read", Kind: "reference", EnterpriseID: q.Base.Enterprise, BUID: r.Data.BU})
			if e != nil {
				return bad()
			}
			if bd.Allowed && slices.Contains(bd.ReadFields, "name") {
				b := current.BU(r.Data.BU)
				if b == nil {
					return bad()
				}
				br := d.BUAt(b.Records, q.Base.Day)
				if br == nil {
					br = &b.Records[0]
				}
				item["managementBuName"] = br.Name
			}
		}
		if has("departmentPath") && has("effectiveDate") && has("status") {
			day, mode := q.Base.Day, "current"
			switch d.Status(*t, q.Base.Day) {
			case "inactive":
				day, mode = dated.NextDate(r.Date, -1), "inactive_before_stop"
			case "not_effective":
				day, mode = t.Records[0].Date, "future_first"
			}
			snapshot, ok := byDay[day]
			if !ok {
				return bad()
			}
			nodes, e := d.PathAt(ctx, snapshot, id, day, enterpriseName)
			if e != nil {
				return bad()
			}
			visible := true
			names := []string{}
			for _, node := range nodes {
				if node.Kind == "department" {
					rd, e := decide(node.DepartmentID, refPlan)
					if e != nil {
						return bad()
					}
					if !rd.Allowed || !slices.Contains(rd.ReadFields, "name") {
						visible = false
					}
				}
				names = append(names, node.Name)
			}
			if visible {
				item["departmentPath"] = strings.Join(names, " / ")
				item["pathMode"] = mode
				item["pathDay"] = day
			}
		}
		if f := q.Query.Filter; f != nil && f.Field == "departmentPath" {
			value, _ := item["departmentPath"].(string)
			value = strings.ToLower(value)
			if f.Operator == "contains" && !strings.Contains(value, f.Value) || f.Operator == "equals" && value != f.Value {
				continue
			}
		}
		out.Items = append(out.Items, item)
	}
	out.Count = c.Count
	if f := q.Query.Filter; f != nil && f.Field == "departmentPath" {
		out.Count = int64(len(out.Items))
		start := min((q.Base.Page-1)*q.Base.PageSize, len(out.Items))
		end := min(start+q.Base.PageSize, len(out.Items))
		out.Items = out.Items[start:end]
	}
	return out, stats, nil
}

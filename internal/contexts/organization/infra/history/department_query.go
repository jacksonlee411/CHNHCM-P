package history

import (
	"context"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/compact"
	"github.com/jackc/pgx/v5"
)

//go:embed department_query_facts.sql
var departmentQueryFactsSQL string

//go:embed query_department_material.sql
var queryDepartmentMaterialSQL string

//go:embed department_query_scope.sql
var departmentQueryScopeSQL string

//go:embed department_query.sql
var departmentQueryTailRawSQL string

var departmentQueryTailSQL = strings.Replace(departmentQueryTailRawSQL, "/*ENTERPRISE_ITEM*/", queryEnterpriseMaterialSQL, 1)

// This is also the statement exercised by the concurrency/fault harness.
var DepartmentQuerySQL = departmentQueryFactsSQL + departmentQueryScopeSQL + strings.Replace(departmentQueryTailSQL, "/*DEPARTMENT_ITEM*/", queryDepartmentMaterialSQL, 1)

type capturedMaterial struct {
	Words, Binary, Starts, Ends, Parts []string
}
type capturedDepartment struct {
	ID, Code, Source, SegmentSource, Relation, FirstRecord string
	QueryDay                                               string
	IDs                                                    []string
	Material                                               capturedMaterial `json:"_material"`
}
type capturedBU struct {
	ID, Enterprise, Code, Source, SegmentSource, Day string
	IDs                                              []string
	Valid                                            bool
	Material                                         capturedMaterial `json:"_material"`
}
type departmentQueryPayload struct {
	Navigation             *org.NavigationResult
	TreeChildren           map[string]app.TreeChildren
	TreePrevious, TreeNext bool
	Cycle, Missing         bool
	Enterprise             capturedEnterprise `json:"enterpriseProfile"`
	Page                   []string
	Departments            []capturedDepartment
	BUs                    []capturedBU
}

func queryUnavailable() error {
	return foundation.Fail(503, "M02_DEPENDENCY_UNAVAILABLE", "部门列表暂时无法加载")
}

// Decode only the immutable bytes captured by the statement. A dictionary may
// have been reclaimed already; no storage read is allowed during projection.
// Decode the existing compact material once for its business facts. Source
// identity is checked by the caller; storage digests are not recomputed here.
func queryValues(ids []string, m capturedMaterial) (compact.Values, []byte, error) {
	v := compact.Values{Strings: m.Words, IDs: ids}
	for _, value := range m.Binary {
		b, err := hex.DecodeString(value)
		if err != nil {
			return v, nil, err
		}
		v.Binary = append(v.Binary, b)
	}
	if len(m.Starts) != len(m.Ends) {
		return v, nil, compact.ErrCorrupt
	}
	for i, start := range m.Starts {
		v.Ranges = append(v.Ranges, compact.Range{Start: start, End: m.Ends[i]})
	}
	var data []byte
	for _, part := range m.Parts {
		b, err := hex.DecodeString(part)
		if err != nil {
			return v, nil, err
		}
		data = append(data, b...)
	}
	return v, data, nil
}

func decodeQueryDepartment(ctx context.Context, enterprise string, item capturedDepartment) (d.Timeline, error) {
	if !foundation.ValidDate(item.QueryDay) || item.Source != item.SegmentSource {
		return d.Timeline{}, compact.ErrCorrupt
	}
	values, data, err := queryValues(item.IDs, item.Material)
	if err != nil {
		return d.Timeline{}, err
	}
	decoded, err := DecodeDepartmentRead(ctx, values, data)
	if err != nil {
		return d.Timeline{}, err
	}
	t := decoded.Domain()
	if t.ID != item.ID || t.Enterprise != enterprise || t.Source != item.Source || t.First != item.FirstRecord || t.RelationSource != item.Relation || len(t.Records) == 0 {
		return d.Timeline{}, compact.ErrCorrupt
	}
	t.Code = item.Code
	return t, nil
}

func CaptureDepartmentQuery(ctx context.Context, tx pgx.Tx, guard access.Guard, q app.QueryRequest) (out app.QueryCapture, err error) {
	planner, ok := guard.(access.ReadPlanner)
	p := q.Plan
	principal := guard.Principal()
	if !ok || p.Origin != planner || p.Instance != principal.InstanceID || p.Subject != principal.ID || p.Session != principal.SessionID || p.Action != access.DepartmentRead || !foundation.ValidID(p.Enterprise) || p.At.IsZero() || p.Day != p.At.In(foundation.Shanghai).Format("2006-01-02") || !foundation.ValidDate(p.Day) || q.Extent > app.QueryIdentities {
		return out, queryUnavailable()
	}
	if !p.Allowed.All && len(p.Allowed.BUs) == 0 && len(p.Allowed.Departments) == 0 {
		return out, foundation.Fail(403, "ACCESS_DENIED", "无部门读取权限")
	}
	q.Input, err = app.NormalizeQuery(q.Input)
	if err != nil {
		return out, err
	}
	if q.Tree != nil {
		return captureTreeQuery(ctx, tx, guard, q)
	}
	field, operator, value := "", "", ""
	if f := q.Input.Filter; f != nil {
		field, operator, value = f.Field, f.Operator, f.Value
	}
	parameters := map[string]any{"q": q.Input.Q, "sort": q.Input.Sort, "status": q.Input.Status, "field": field, "operator": operator, "value": value}
	if q.Navigation != "" {
		if !foundation.ValidID(q.AnchorID) || (q.Navigation != "current" && q.Navigation != "previous" && q.Navigation != "next") || field == "departmentPath" || q.Extent != app.QueryPage {
			return out, queryUnavailable()
		}
		parameters["anchor"], parameters["direction"] = q.AnchorID, q.Navigation
	}
	if e := addTreeScopeParameters(ctx, guard, q, parameters); e != nil {
		return out, e
	}
	if q.OnlyIDs != nil {
		parameters["ids"] = q.OnlyIDs
	}
	query, _ := json.Marshal(parameters)
	plan, _ := json.Marshal(queryPlan(p))
	// Permission and filter values determine the necessary work in every capture.
	// Cache parameter descriptions while planning with the actual bound values.
	args := []any{pgx.QueryExecModeCacheDescribe, p.Instance, p.Enterprise, p.Day, plan, query}
	if q.Extent == app.QueryIdentities {
		// This query deliberately stops before path traversal, dictionaries,
		// decoded business data and collection digests. Path text needs its Go
		// matcher and is never accepted by this identity-only endpoint.
		if field == "departmentPath" {
			return out, queryUnavailable()
		}
		prefix, _, _ := strings.Cut(departmentQueryScopeSQL, "), ordered AS MATERIALIZED (")
		statement := departmentQueryFactsSQL + prefix + ") SELECT (SELECT denied FROM forbidden), ARRAY(SELECT department_id::text FROM matching ORDER BY department_id LIMIT 131073)"
		var denied int64
		if err = tx.QueryRow(ctx, statement, args...).Scan(&denied, &out.PageIDs); err != nil {
			return out, err
		}
		if denied != 0 {
			return out, scopeQueryError(denied)
		}
		out.Count = len(out.PageIDs)
		return out, nil
	}
	if q.Extent == app.QuerySelection {
		return captureDepartmentSet(ctx, tx, p, q, plan, query)
	}
	if field == "departmentPath" {
		return out, queryUnavailable()
	}
	var bad, denied, count int64
	var raw []byte
	done := foundation.BeginStage(ctx, "department.query.capture")
	args = append(args, q.Input.PageSize, (q.Input.Page-1)*q.Input.PageSize, q.Extent == app.QuerySelection)
	statement := DepartmentQuerySQL
	if q.Navigation != "" {
		statement = DepartmentNavigationSQL
	}
	err = tx.QueryRow(ctx, statement, args...).Scan(&out.Stats.Objects, &out.Stats.Segments, &bad, &out.Stats.Dictionaries, &out.Stats.Shapes, &denied, &count, &raw)
	done()
	if err != nil {
		return out, err
	}
	if bad != 0 || count < 0 || uint64(count) > uint64(^uint(0)>>1) || raw == nil {
		return app.QueryCapture{}, queryUnavailable()
	}
	if denied != 0 {
		return app.QueryCapture{}, scopeQueryError(denied)
	}
	decoded, e := decodeDepartmentQuery(ctx, p.Enterprise, p.Day, int(count), raw, nil)
	decoded.Stats.Objects = out.Stats.Objects
	decoded.Stats.Segments = out.Stats.Segments
	decoded.Stats.Dictionaries = out.Stats.Dictionaries
	decoded.Stats.Shapes = out.Stats.Shapes
	return decoded, e
}

func decodeDepartmentQuery(ctx context.Context, enterprise, day string, count int, raw []byte, capturedName *string) (out app.QueryCapture, err error) {
	var payload departmentQueryPayload
	if json.Unmarshal(raw, &payload) != nil || payload.Cycle || payload.Missing {
		return app.QueryCapture{}, queryUnavailable()
	}
	out.TreeChildren = payload.TreeChildren
	out.Navigation = payload.Navigation
	out.TreePrevious, out.TreeNext = payload.TreePrevious, payload.TreeNext
	out.Count = int(count)
	out.PageIDs = payload.Page
	if capturedName != nil {
		out.EnterpriseName = *capturedName
	} else {
		out.EnterpriseName, err = decodeQueryEnterpriseName(ctx, enterprise, day, payload.Enterprise)
		if err != nil {
			return app.QueryCapture{}, err
		}
	}
	out.ByDay = map[string]d.Snapshot{}
	out.Stats.MaterialBytes = len(raw)
	done := foundation.BeginStage(ctx, "department.query.decode")
	defer done()
	for _, item := range payload.Departments {
		if err = ctx.Err(); err != nil {
			return app.QueryCapture{}, err
		}
		timeline, e := decodeQueryDepartment(ctx, enterprise, item)
		if e != nil {
			return app.QueryCapture{}, errors.Join(queryUnavailable(), e)
		}
		s := out.ByDay[item.QueryDay]
		s.Departments = append(s.Departments, timeline)
		out.ByDay[item.QueryDay] = s
		out.Stats.DecodedDepartments++
	}
	for _, item := range payload.BUs {
		if err = ctx.Err(); err != nil {
			return app.QueryCapture{}, err
		}
		if !item.Valid || item.Enterprise != enterprise || item.Source != item.SegmentSource || !foundation.ValidDate(item.Day) {
			return app.QueryCapture{}, queryUnavailable()
		}
		values, data, e := queryValues(item.IDs, item.Material)
		if e != nil {
			return app.QueryCapture{}, errors.Join(queryUnavailable(), e)
		}
		decoded, e := DecodeBURead(ctx, values, data)
		if e != nil {
			return app.QueryCapture{}, errors.Join(queryUnavailable(), e)
		}
		b := decoded.Timeline
		if b.ID != item.ID || b.Source != item.Source || len(b.Records) == 0 || b.First != b.Records[0].ID {
			return app.QueryCapture{}, queryUnavailable()
		}
		b.Code = item.Code
		s := out.ByDay[item.Day]
		s.BusinessUnits = append(s.BusinessUnits, b)
		out.ByDay[item.Day] = s
		out.Stats.DecodedBUs++
	}
	for day, s := range out.ByDay {
		out.ByDay[day] = s.ReadIndexed()
	}
	return out, nil
}

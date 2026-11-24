package department

import (
	"context"
	"errors"
	"time"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

// QueryExtent describes the consumer, never an integrity or authorization bypass.
type QueryExtent uint8

const (
	QueryPage QueryExtent = iota
	QuerySelection
	QueryIdentities
)

type TreeRead struct {
	Mode, ParentID, AroundID, AfterKey, AfterID string
	Reverse                                     bool
	Reference                                   access.ReadPlan
}
type TreeChildren struct {
	HasChildren bool
	Denied      bool
}
type QueryRequest struct {
	Tree              *TreeRead
	Plan              access.ReadPlan
	Input             org.DepartmentListInput
	Extent            QueryExtent
	OnlyIDs           []string
	AfterKey, AfterID string
	AnchorID          string
	Navigation        string
	Reverse           bool
}

type QueryStats struct {
	Objects, Segments, Dictionaries, Shapes       int64
	MaterialBytes, DecodedDepartments, DecodedBUs int
	FetchedFrames, TransferredMaterialBytes       int
}

// QueryCapture contains only facts decoded from one statement's captured bytes.
// Each date has its own lookup spans; authorization always uses Plan.Day.
type QueryCapture struct {
	Navigation             *org.NavigationResult
	TreeChildren           map[string]TreeChildren
	TreePrevious, TreeNext bool
	EnterpriseName         string
	Count                  int
	PageIDs                []string
	ByDay                  map[string]d.Snapshot
	Stats                  QueryStats
}

func queryKey(in org.DepartmentListInput, subject string) string {
	return fingerprint(struct {
		Q, Status, Subject string
		Scope              string
		Descendants        *bool
		Filter             *org.ColumnFilter
	}{in.Q, in.Status, subject, in.ScopeDepartmentID, in.IncludeDescendants, in.Filter})
}

func queryRows(ctx context.Context, u Unit, in org.DepartmentListInput, extent QueryExtent) (out org.DepartmentList, rows []listMember, captured QueryCapture, now time.Time, err error) {
	return queryRowsIDs(ctx, u, in, extent, nil)
}

func queryRowsIDs(ctx context.Context, u Unit, in org.DepartmentListInput, extent QueryExtent, ids []string) (out org.DepartmentList, rows []listMember, captured QueryCapture, now time.Time, err error) {
	if in.Filter != nil && in.Filter.Field == "departmentPath" {
		return pathRows(ctx, u, in, extent, ids)
	}
	ent, err := u.Enterprise(ctx, false)
	if err != nil {
		return
	}
	now, err = u.Now(ctx)
	if err != nil {
		return
	}
	planner, ok := u.Access().(access.ReadPlanner)
	if !ok {
		err = foundation.Fail(503, "M02_DEPENDENCY_UNAVAILABLE", "部门资料暂时无法加载")
		return
	}
	plan, err := planner.PlanReads(ctx, ent.ID, now, access.DepartmentRead)
	if err != nil {
		return
	}
	if !plan.Allowed.All && len(plan.Allowed.BUs) == 0 && len(plan.Allowed.Departments) == 0 {
		err = denied()
		return
	}
	captured, err = u.CaptureQuery(ctx, QueryRequest{Plan: plan, Input: in, Extent: extent, OnlyIDs: ids})
	if err != nil {
		return
	}
	rows, err = projectQuery(ctx, u, in, plan, captured, ent, now)
	if err != nil {
		return
	}
	count := captured.Count

	out = org.DepartmentList{Count: count, Page: in.Page, PageSize: in.PageSize, BusinessAt: plan.Day, QueryKey: queryKey(in, plan.Instance+":"+plan.Subject)}
	return
}

func projectQuery(ctx context.Context, u Unit, in org.DepartmentListInput, plan access.ReadPlan, captured QueryCapture, ent org.Enterprise, now time.Time) (rows []listMember, err error) {
	planner := u.Access().(access.ReadPlanner)
	ent.Name = captured.EnterpriseName
	ent.Names = nil
	current := captured.ByDay[plan.Day]
	guard, err := capturedAuthorization(ctx, u, current, plan, planner)
	if err != nil {
		return
	}
	p := newProjector(ctx, projectedReadUnit{Unit: u, guard: guard}, current, ent, now)
	p.pathSnapshots = captured.ByDay
	rows = make([]listMember, 0, len(captured.PageIDs))
	for _, id := range captured.PageIDs {
		if err = ctx.Err(); err != nil {
			return
		}
		t := current.Department(id)
		if t == nil {
			err = foundation.Fail(503, "M02_DEPENDENCY_UNAVAILABLE", "部门列表暂时无法加载")
			return
		}
		r := d.ScopeRecord(*t, plan.Day)
		decision := guard.objectDecision(id)
		if r == nil || !decision.Allowed {
			err = foundation.Fail(503, "M02_DEPENDENCY_UNAVAILABLE", "部门列表暂时无法加载")
			return
		}
		v := listMember{t: t, r: r, fields: guard.objectMask(id)}
		if v.has("status") {
			status := d.Status(*t, plan.Day)
			v.status = &status
		}
		if v.has("managementBuId") {
			ref := p.ref(r.Data.BU, "bu")
			if ref.err != nil {
				err = ref.err
				return
			}
			v.bu = ref.value
		}
		v.path, err = p.listPathValue(t, r, v.fields)
		if err != nil {
			err = unavailableQueryPath(err)
			return
		}
		if in.Filter == nil || in.Filter.Field != "departmentPath" || v.matches(in) {
			rows = append(rows, v)
		}
	}
	return
}

type QueryFrame struct {
	Anchor  bool
	Key, ID string
	Capture QueryCapture
}

// Keep the safe public fault while retaining cancellation/corruption as the
// cause. In particular, a deadline during recursive path projection must not
// disappear behind a generic dependency fault.
func unavailableQueryPath(cause error) error {
	return errors.Join(foundation.Fail(503, "M02_DEPENDENCY_UNAVAILABLE", "部门路径暂时无法加载，请重试"), cause)
}

// Evaluate only the captured page/ancestor closure with the X03-compiled sets.
func capturedAuthorization(ctx context.Context, u Unit, s d.Snapshot, plan access.ReadPlan, planner access.ReadPlanner) (projectedReads, error) {
	capacity := 2*len(s.Departments) + len(s.BusinessUnits)
	g := projectedReads{Guard: u.Access(), snapshot: s, departments: make([]projectedDepartmentReads, len(s.Departments)), bus: map[string]projectedBUReads{}, decisions: make([]access.Decision, 0, capacity), masks: make([]uint16, 0, capacity)}
	ref, err := planner.PlanReads(ctx, plan.Enterprise, plan.At, access.DepartmentReference)
	if err != nil {
		return g, err
	}
	bu, err := planner.PlanReads(ctx, plan.Enterprise, plan.At, access.BUReference)
	if err != nil {
		return g, err
	}
	add := func(p access.ReadPlan, in access.ReadIntent) (int, error) {
		decision, e := planner.DecidePlan(ctx, p, in)
		if e != nil {
			return 0, e
		}
		i := len(g.decisions)
		g.decisions = append(g.decisions, decision)
		g.masks = append(g.masks, readMask(decision.ReadFields))
		return i, nil
	}
	for i, t := range s.Departments {
		target := factOnDay(t, plan.At, plan.Day, "")
		object, e := add(plan, access.ReadIntent{Action: string(plan.Action), Kind: "object", EnterpriseID: plan.Enterprise, DepartmentTarget: &target})
		if e != nil {
			return g, e
		}
		reference, e := add(ref, access.ReadIntent{Action: string(ref.Action), Kind: "reference", EnterpriseID: plan.Enterprise, DepartmentTarget: &target})
		if e != nil {
			return g, e
		}
		g.departments[i] = projectedDepartmentReads{target: &target, object: object, reference: reference}
	}
	for _, b := range s.BusinessUnits {
		i, e := add(bu, access.ReadIntent{Action: string(bu.Action), Kind: "reference", EnterpriseID: plan.Enterprise, BUID: b.ID})
		if e != nil {
			return g, e
		}
		g.bus[b.ID] = projectedBUReads{enterprise: plan.Enterprise, reference: i}
	}
	return g, nil
}

// Identity capture needs path matching and current field authorization, not all
// eight CSV columns, reference labels, collection hashes or list summaries.
func pathMatchingIDs(ctx context.Context, u Unit, in org.DepartmentListInput, plan access.ReadPlan, captured QueryCapture, ent org.Enterprise) ([]string, error) {
	return pathMatchingIDsLimit(ctx, u, in, plan, captured, ent, DraftMaxMembers)
}

func pathMatchingIDsLimit(ctx context.Context, u Unit, in org.DepartmentListInput, plan access.ReadPlan, captured QueryCapture, ent org.Enterprise, limit int) ([]string, error) {
	current := captured.ByDay[plan.Day]
	guard, err := capturedAuthorization(ctx, u, current, plan, u.Access().(access.ReadPlanner))
	if err != nil {
		return nil, err
	}
	ent.Name = captured.EnterpriseName
	ent.Names = nil
	p := newProjector(ctx, projectedReadUnit{Unit: u, guard: guard}, current, ent, plan.At)
	p.pathSnapshots = captured.ByDay
	ids := []string{}
	for _, id := range captured.PageIDs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		t := current.Department(id)
		if t == nil {
			return nil, foundation.Fail(503, "M02_DEPENDENCY_UNAVAILABLE", "部门列表暂时无法加载")
		}
		r := d.ScopeRecord(*t, plan.Day)
		if r == nil || !guard.objectDecision(id).Allowed {
			return nil, foundation.Fail(503, "M02_DEPENDENCY_UNAVAILABLE", "部门列表暂时无法加载")
		}
		fields := guard.objectMask(id)
		path, err := p.listPathValue(t, r, fields)
		if err != nil {
			return nil, unavailableQueryPath(err)
		}
		row := listMember{t: t, r: r, fields: fields, path: path}
		if row.matches(in) {
			ids = append(ids, id)
			if limit > 0 && len(ids) > limit {
				return nil, DraftMemberLimit()
			}
		}
	}
	return ids, nil
}

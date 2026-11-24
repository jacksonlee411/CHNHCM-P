package history

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
)

func addTreeScopeParameters(ctx context.Context, guard access.Guard, q app.QueryRequest, params map[string]any) error {
	params["scope"] = q.Input.ScopeDepartmentID
	if q.Input.ScopeDepartmentID == "" {
		return nil
	}
	params["descendants"] = q.Input.IncludeDescendants == nil || *q.Input.IncludeDescendants
	planner, ok := guard.(access.ReadPlanner)
	if !ok {
		return queryUnavailable()
	}
	reference, e := planner.PlanReads(ctx, q.Plan.Enterprise, q.Plan.At, access.DepartmentReference)
	if e != nil {
		return e
	}
	params["scopeReference"] = queryPlan(reference)
	// A department-only read set can test membership by following those
	// candidates' current parents, without expanding every unreadable sibling.
	sparse := !q.Plan.Allowed.All && len(q.Plan.Allowed.BUs) == 0
	params["scopeSparse"] = sparse
	if sparse {
		params["scopeCandidates"] = q.Plan.Allowed.Departments
	}
	// With enterprise-wide minimal reference rights, branch edges already
	// establish every descendant's ancestry inside the branch. Only the scope
	// root's outside ancestors need the separate existence/reference walk.
	params["scopeReferencesComplete"] = reference.Allowed.All && reference.Fields["code"].All && reference.Fields["name"].All
	complete := true
	for _, field := range []string{"code", "name", "status", "effectiveDate", "parentDepartmentId", "departmentPath"} {
		complete = complete && treeSetCovers(q.Plan.Fields[field], q.Plan.Allowed)
	}
	params["scopeFieldsComplete"] = complete
	// With complete operational fields and no scalar filter, the captured
	// branch already supplies every matching identity, code and path day.
	// Path text is still matched by Go against captured source material.
	params["scopeMatchAll"] = q.Plan.Allowed.All && complete && q.Input.Q == "" && q.Input.Status == "all" && q.Input.Sort == "code" && (q.Input.Filter == nil || q.Input.Filter.Field == "departmentPath")
	return nil
}
func scopeQueryError(code int64) error {
	if code == -1 {
		return foundation.Fail(409, "M02_TREE_SCOPE_UNAVAILABLE", "当前范围不可用")
	}
	if code == -2 {
		return queryUnavailable()
	}
	return foundation.Fail(403, "ACCESS_DENIED", "无权使用该筛选字段")
}

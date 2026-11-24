package department

import (
	"context"
	"strings"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

func treeQuery(in org.DepartmentTreeInput, mode string) (org.DepartmentListInput, error) {
	if in.PageSize == 0 {
		in.PageSize = 50
		if mode == "search" {
			in.PageSize = 20
		}
	}
	if in.ParentID != nil && !foundation.ValidID(*in.ParentID) || in.AroundID != "" && !foundation.ValidID(in.AroundID) || in.Cursor != "" && in.AroundID != "" {
		return org.DepartmentListInput{}, foundation.Fail(400, "M02_INVALID_INPUT", "部门树查询参数无效")
	}
	if mode == "search" && (in.ParentID != nil || in.AroundID != "" || strings.TrimSpace(in.Q) == "") {
		return org.DepartmentListInput{}, foundation.Fail(400, "M02_INVALID_INPUT", "请输入定位词")
	}
	q := org.DepartmentListInput{Q: in.Q, Page: 1, PageSize: in.PageSize, Sort: "code", Status: "active"}
	return q, normalizeDepartmentList(&q)
}
func treeCapture(ctx context.Context, u Unit, q org.DepartmentListInput, t *TreeRead) (QueryCapture, access.ReadPlan, projectedReads, error) {
	plan, e := selectionPlan(ctx, u)
	if e != nil {
		return QueryCapture{}, plan, projectedReads{}, e
	}
	planner := u.Access().(access.ReadPlanner)
	t.Reference, e = planner.PlanReads(ctx, plan.Enterprise, plan.At, access.DepartmentReference)
	if e != nil {
		return QueryCapture{}, plan, projectedReads{}, e
	}
	captured, e := u.CaptureQuery(ctx, QueryRequest{Plan: plan, Input: q, Tree: t})
	if e != nil {
		return captured, plan, projectedReads{}, e
	}
	reads, e := capturedAuthorization(ctx, u, captured.ByDay[plan.Day], plan, planner)
	return captured, plan, reads, e
}
func treeChildren(items []org.DepartmentTreeNode, captured QueryCapture) error {
	for i := range items {
		if items[i].Kind != "node" {
			continue
		}
		child, ok := captured.TreeChildren[items[i].ID]
		if !ok {
			return foundation.Fail(503, "M02_DEPENDENCY_UNAVAILABLE", "部门树子项读取不完整")
		}
		if child.Denied {
			return denied()
		}
		value := child.HasChildren
		items[i].HasChildren = &value
	}
	return nil
}
func (s Service) TreeNodes(ctx context.Context, c access.Credentials, in org.DepartmentTreeInput) (org.DepartmentTreePage, error) {
	return s.treePage(ctx, c, in, "nodes")
}
func (s Service) TreeSearch(ctx context.Context, c access.Credentials, in org.DepartmentTreeInput) (org.DepartmentTreePage, error) {
	return s.treePage(ctx, c, in, "search")
}
func (s Service) treePage(ctx context.Context, c access.Credentials, in org.DepartmentTreeInput, mode string) (out org.DepartmentTreePage, err error) {
	q, err := treeQuery(in, mode)
	if err != nil {
		return out, err
	}
	err = s.Store.WithinDepartment(ctx, c, func(raw Unit) error {
		now, e := raw.Now(ctx)
		if e != nil {
			return e
		}
		u := selectionUnit{raw, now}
		plan, e := selectionPlan(ctx, u)
		if e != nil {
			return e
		}
		t := &TreeRead{Mode: mode, AroundID: in.AroundID}
		if in.ParentID != nil {
			t.ParentID = *in.ParentID
		}
		binding := queryCursor{Kind: "department-tree", Instance: plan.Instance, Enterprise: plan.Enterprise, Subject: plan.Subject, Session: plan.Session, Day: plan.Day, Query: fingerprint(struct {
			Mode, Parent, Q string
			Size            int
		}{mode, t.ParentID, q.Q, q.PageSize}), Source: "next"}
		if in.Cursor != "" {
			t.AfterKey, t.AfterID, e = binding.decode(in.Cursor)
			if e != nil {
				binding.Source = "previous"
				t.AfterKey, t.AfterID, e = binding.decode(in.Cursor)
				t.Reverse = true
			}
			if e != nil {
				return e
			}
		}
		captured, plan, reads, e := treeCapture(ctx, u, q, t)
		if e != nil {
			return e
		}
		snapshot := captured.ByDay[plan.Day]
		if in.ParentID != nil {
			if _, e = projectTreePath(ctx, snapshot, reads, *in.ParentID, plan.Day); e != nil {
				return e
			}
		}
		items := []org.DepartmentTreeNode{}
		for _, id := range captured.PageIDs {
			path, e := projectTreePath(ctx, snapshot, reads, id, plan.Day)
			if e != nil {
				return e
			}
			items = append(items, path[len(path)-1])
		}
		if in.AroundID != "" && (len(items) == 0 || items[0].ID != in.AroundID) {
			return treeScopeUnavailable()
		}
		if e = treeChildren(items, captured); e != nil {
			return e
		}
		out = org.DepartmentTreePage{Items: items, BusinessAt: plan.Day, HasPrevious: captured.TreePrevious, HasMore: captured.TreeNext}
		if len(items) > 0 {
			if out.HasMore {
				binding.Source = "next"
				last := items[len(items)-1]
				out.NextCursor = binding.encode(strings.ToLower(last.Code), last.ID)
			}
			if out.HasPrevious {
				binding.Source = "previous"
				first := items[0]
				out.PreviousCursor = binding.encode(strings.ToLower(first.Code), first.ID)
			}
		}
		return nil
	})
	if err != nil {
		out = org.DepartmentTreePage{}
	}
	return
}
func (s Service) TreeLocation(ctx context.Context, c access.Credentials, id string) (out org.DepartmentTreeLocation, err error) {
	if !foundation.ValidID(id) {
		return out, foundation.Fail(400, "M02_INVALID_INPUT", "所选部门无效，请重新选择")
	}
	err = s.Store.WithinDepartment(ctx, c, func(raw Unit) error {
		now, e := raw.Now(ctx)
		if e != nil {
			return e
		}
		u := selectionUnit{raw, now}
		q := org.DepartmentListInput{Page: 1, PageSize: 20, Status: "active", Sort: "code"}
		captured, plan, reads, e := treeCapture(ctx, u, q, &TreeRead{Mode: "location", AroundID: id})
		if e != nil {
			return e
		}
		items, e := projectTreePath(ctx, captured.ByDay[plan.Day], reads, id, plan.Day)
		if e != nil {
			return e
		}
		if e = treeChildren(items, captured); e != nil {
			return e
		}
		out = org.DepartmentTreeLocation{Items: items, BusinessAt: plan.Day}
		return nil
	})
	if err != nil {
		out = org.DepartmentTreeLocation{}
	}
	return
}

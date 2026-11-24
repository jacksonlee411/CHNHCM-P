package department

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
)

func (s Service) Navigate(ctx context.Context, c access.Credentials, in org.DepartmentNavigationInput) (out org.DepartmentNavigationPage, err error) {
	if !foundation.ValidID(in.AnchorID) || (in.Direction != "current" && in.Direction != "previous" && in.Direction != "next") {
		return out, foundation.Fail(400, "M02_INVALID_INPUT", "导航参数无效")
	}
	q, err := selectionQuery(in.Query, in.Sort, 1, 20)
	if err != nil {
		return
	}
	err = s.Store.WithinDepartment(ctx, c, func(raw Unit) error {
		now, e := raw.Now(ctx)
		if e != nil {
			return e
		}
		u := selectionUnit{raw, now}
		ent, e := u.Enterprise(ctx, false)
		if e != nil {
			return e
		}
		plan, e := selectionPlan(ctx, u)
		if e != nil {
			return e
		}
		if q.Filter == nil || q.Filter.Field != "departmentPath" {
			captured, e := u.CaptureQuery(ctx, QueryRequest{Plan: plan, Input: q, AnchorID: in.AnchorID, Navigation: in.Direction})
			if e != nil {
				return e
			}
			rows, e := projectQuery(ctx, u, q, plan, captured, ent, now)
			if e != nil {
				return e
			}
			found := false
			for _, row := range rows {
				if row.t.ID == in.AnchorID {
					found = true
				}
			}
			if !found {
				return foundation.Fail(409, "M02_NAVIGATION_CHANGED", "当前记录不在查询结果中")
			}
			if captured.Navigation != nil {
				out = *captured.Navigation
			}
			return nil
		}
		if in.Direction == "current" {
			captured, e := u.CaptureQuery(ctx, QueryRequest{Plan: plan, Input: q, Extent: QuerySelection})
			if e != nil {
				return e
			}
			ids, e := pathMatchingIDsLimit(ctx, u, q, plan, captured, ent, 0)
			if e != nil {
				return e
			}
			for i, id := range ids {
				if id != in.AnchorID {
					continue
				}
				out.TargetID, out.Position, out.Count = &ids[i], i+1, len(ids)
				if i > 0 {
					out.PreviousID = &ids[i-1]
				}
				if i+1 < len(ids) {
					out.NextID = &ids[i+1]
				}
				return nil
			}
			return foundation.Fail(409, "M02_NAVIGATION_CHANGED", "当前记录不在查询结果中")
		}
		return u.WalkQuery(ctx, QueryRequest{Plan: plan, Input: q, AnchorID: in.AnchorID, Reverse: in.Direction == "previous"}, func(frame QueryFrame) (bool, error) {
			rows, e := projectQuery(ctx, u, q, plan, frame.Capture, ent, now)
			if e != nil {
				return false, e
			}
			if frame.Anchor {
				if len(rows) != 1 {
					return false, foundation.Fail(409, "M02_NAVIGATION_CHANGED", "原对象已不在当前查询中，请返回列表")
				}
				return true, nil
			}
			if len(rows) == 0 {
				return true, nil
			}
			id := rows[0].t.ID
			out.TargetID = &id
			return false, nil
		})
	})
	return
}

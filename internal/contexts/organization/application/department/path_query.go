package department

import (
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"time"
)

// Path matching is evaluated by Go over one statement snapshot. Ordinary pages
// stop after the lookahead match; fixed selection capture deliberately exhausts it.
func pathRows(ctx context.Context, u Unit, in org.DepartmentListInput, extent QueryExtent, ids []string) (out org.DepartmentList, rows []listMember, captured QueryCapture, now time.Time, err error) {
	if extent == QueryPage && in.Page > 1 && in.Cursor == "" {
		err = foundation.Fail(400, "M02_INVALID_INPUT", "查询位置已失效，请重新查询")
		return
	}
	ent, err := u.Enterprise(ctx, false)
	if err != nil {
		return
	}
	now, err = u.Now(ctx)
	if err != nil {
		return
	}
	u = selectionUnit{u, now}
	plan, err := selectionPlan(ctx, u)
	if err != nil {
		return
	}
	binding := queryCursor{Kind: "department-path", Instance: plan.Instance, Enterprise: plan.Enterprise, Subject: plan.Subject, Session: plan.Session, Day: plan.Day, Query: fingerprint(struct {
		Key, Sort string
		Size      int
	}{queryKey(in, plan.Instance+":"+plan.Subject), in.Sort, in.PageSize})}
	q := QueryRequest{Plan: plan, Input: in, Extent: extent, OnlyIDs: ids}
	if extent == QueryPage && in.Cursor != "" {
		q.AfterKey, q.AfterID, err = binding.decode(in.Cursor)
		if err != nil {
			return
		}
	}
	rows = make([]listMember, 0, in.PageSize)
	lastKey, lastID := "", ""
	err = u.WalkQuery(ctx, q, func(frame QueryFrame) (bool, error) {
		values, e := projectQuery(ctx, u, in, plan, frame.Capture, ent, now)
		if e != nil {
			return false, e
		}
		if len(values) == 0 {
			return true, nil
		}
		if extent == QueryPage && len(rows) == in.PageSize {
			out.HasMore = true
			out.NextCursor = binding.encode(lastKey, lastID)
			return false, nil
		}
		rows = append(rows, values...)
		lastKey, lastID = frame.Key, frame.ID
		return true, nil
	})
	if err != nil {
		return
	}
	out.Count = -1
	out.Page = in.Page
	out.PageSize = in.PageSize
	out.BusinessAt = plan.Day
	out.QueryKey = queryKey(in, plan.Instance+":"+plan.Subject)
	return
}

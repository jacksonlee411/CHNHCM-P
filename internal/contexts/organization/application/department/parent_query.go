package department

import (
	"context"
	"time"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

// ParentRequest is prepared by the common Go kernel. It is never an HTTP DTO
// or reusable write permission. Capture must check Source in its own snapshot.
type ParentRequest struct {
	Plan                                 access.ReadPlan
	Target, Source, BU, Selected, Search string
	Base                                 d.Timeline
	Windows                              []d.ParentWindow
	AfterCode, AfterID                   string
}

type ParentFrame struct {
	Common   bool
	ID, Code string
	Capture  ParentCapture
}
type ParentCapture struct {
	Facts, Labels d.Snapshot
	Stats         QueryStats
}

func prepareParent(ctx context.Context, t d.Timeline, q org.DepartmentReferenceContext, start string) (d.Timeline, map[string]bool, bool, error) {
	base := t
	rows, applicable, err := referenceRows(ctx, t, q, "department", "", start)
	if err != nil || !applicable {
		return base, nil, applicable, err
	}
	marker := foundation.NewID()
	marked, _, err := referenceRows(ctx, t, q, "department", marker, start)
	if err != nil {
		return base, nil, false, err
	}
	if len(marked) != len(rows) {
		return base, nil, false, foundation.Fail(503, "M02_DEPENDENCY_UNAVAILABLE", "适用期间暂时无法加载")
	}
	variable := map[string]bool{}
	for i, a := range marked {
		b := rows[i]
		if a.ID != b.ID || a.Date != b.Date || a.End != b.End || a.State != b.State || a.Data.BU != b.Data.BU {
			return base, nil, false, foundation.Fail(503, "M02_DEPENDENCY_UNAVAILABLE", "候选期间不一致")
		}
		if a.Data.Parent == marker && a.Data.Parent != b.Data.Parent {
			variable[b.ID] = true
		}
	}
	base.Records = rows
	return base, variable, true, nil
}

func parentPlan(ctx context.Context, u Unit, ent string, now time.Time) (access.ReadPlan, error) {
	planner, ok := u.Access().(access.ReadPlanner)
	if !ok {
		return access.ReadPlan{}, foundation.Fail(503, "M02_DEPENDENCY_UNAVAILABLE", "部门资料暂时无法加载")
	}
	return planner.PlanReads(ctx, ent, now, access.DepartmentReference)
}

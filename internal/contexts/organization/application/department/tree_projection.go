package department

import (
	"context"
	"slices"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

// TreePathItem is a current, authorized path projection. Context ancestors are
// references only: they do not acquire selection, expansion or detail rights.
type TreePathItem = org.DepartmentTreeNode

var treeFields = []string{"code", "name", "status", "effectiveDate", "parentDepartmentId", "departmentPath"}

func treeScopeUnavailable() error {
	return foundation.Fail(409, "M02_TREE_SCOPE_UNAVAILABLE", "当前范围不可用")
}

func treeFieldsAllowed(decision access.Decision) bool {
	if !decision.Allowed {
		return false
	}
	for _, field := range treeFields {
		if !slices.Contains(decision.ReadFields, field) {
			return false
		}
	}
	return true
}

// projectTreePath consumes one captured current snapshot and its request-local
// authorization decisions. It never reads again or interprets historical paths.
func projectTreePath(ctx context.Context, snapshot d.Snapshot, reads projectedReads, id, day string) ([]TreePathItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !treeFieldsAllowed(reads.objectDecision(id)) {
		return nil, treeScopeUnavailable()
	}
	target := snapshot.Department(id)
	if target == nil || d.Status(*target, day) != "active" {
		return nil, treeScopeUnavailable()
	}
	chain, err := d.PathAt(ctx, snapshot, id, day, "")
	if err != nil {
		return nil, unavailableQueryPath(err)
	}
	out := make([]TreePathItem, 0, len(chain))
	for _, part := range chain {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if part.Kind != "department" {
			continue
		}
		reference := reads.referenceDecision(part.DepartmentID, "department")
		if !reference.Allowed || !slices.Contains(reference.ReadFields, "code") || !slices.Contains(reference.ReadFields, "name") {
			return nil, treeScopeUnavailable()
		}
		timeline := snapshot.Department(part.DepartmentID)
		if timeline == nil {
			return nil, foundation.Fail(503, "M02_DEPENDENCY_UNAVAILABLE", "部门列表暂时无法加载")
		}
		item := TreePathItem{ID: timeline.ID, Code: timeline.Code, Name: part.Name, Kind: "context"}
		if part.Edge.Parent != "" {
			parent := part.Edge.Parent
			item.ParentID = &parent
		}
		if treeFieldsAllowed(reads.objectDecision(timeline.ID)) {
			item.Kind = "node"
		}
		out = append(out, item)
	}
	return out, nil
}

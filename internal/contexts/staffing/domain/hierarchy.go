package domain

import (
	"cmp"
	"slices"

	"chnhcm.local/chnhcm/internal/foundation"
)

type HierarchyEdge struct{ Child, Parent, Start, End string }
type HierarchyEdges []HierarchyEdge

func (edges HierarchyEdges) Assign(child, parent, date string) HierarchyEdges {
	next := slices.Clone(edges)
	replaced := false
	for index := range next {
		if next[index].Child == child && next[index].Start == date {
			next[index].Parent = parent
			replaced = true
		}
	}
	if !replaced {
		next = append(next, HierarchyEdge{Child: child, Parent: parent, Start: date})
	}
	slices.SortFunc(next, func(left, right HierarchyEdge) int {
		return cmp.Or(cmp.Compare(left.Child, right.Child), cmp.Compare(left.Start, right.Start))
	})
	for index := range next {
		if next[index].Child == child {
			next[index].End = ""
			if index+1 < len(next) && next[index+1].Child == child {
				next[index].End = next[index+1].Start
			}
		}
	}
	return next
}
func (edges HierarchyEdges) At(date string) map[string]string {
	parents := map[string]string{}
	for _, edge := range edges {
		if edge.Parent != "" && edge.Start <= date && (edge.End == "" || date < edge.End) {
			parents[edge.Child] = edge.Parent
		}
	}
	return parents
}
func (edges HierarchyEdges) Validate(positions map[string]Timeline) error {
	boundaries := map[string]bool{}
	first, reason := "", ""
	reject := func(date, message string) {
		if first == "" || date < first {
			first, reason = date, message
		}
	}
	for _, edge := range edges {
		if !foundation.ValidDate(edge.Start) || edge.End != "" && (!foundation.ValidDate(edge.End) || edge.End <= edge.Start) {
			return foundation.Fail(400, "M03_INVALID_INPUT", "岗位关系日期无效")
		}
		boundaries[edge.Start] = true
		if edge.End != "" {
			boundaries[edge.End] = true
		}
		if edge.Parent == "" {
			continue
		}
		for _, id := range []string{edge.Child, edge.Parent} {
			if date := PositionConflict(positions[id], edge.Start, edge.End, ""); date != "" {
				reject(date, "父子岗位须在整个关系期间启用")
			}
		}
	}
	dates := make([]string, 0, len(boundaries))
	for date := range boundaries {
		dates = append(dates, date)
	}
	slices.Sort(dates)
	for _, date := range dates {
		if first != "" && first < date {
			break
		}
		parents := map[string]string{}
		for _, edge := range edges {
			if edge.Start <= date && (edge.End == "" || date < edge.End) {
				if _, exists := parents[edge.Child]; exists {
					reject(date, "同一岗位在同一天只能有一个管理上级")
				}
				parents[edge.Child] = edge.Parent
			}
		}
		done := map[string]bool{}
		for child := range parents {
			if done[child] {
				continue
			}
			path := map[string]bool{}
			for node := child; node != "" && !done[node]; node = parents[node] {
				if path[node] {
					reject(date, "岗位上下级关系成环")
					break
				}
				path[node] = true
			}
			for node := range path {
				done[node] = true
			}
		}
	}
	if first != "" {
		return foundation.Fail(409, "M03_HIERARCHY_CONFLICT", first+"："+reason)
	}
	return nil
}

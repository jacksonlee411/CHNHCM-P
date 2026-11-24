package postgres

import (
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"sort"
	"strings"
)

func (u *unit) buNavigationFacts(ctx context.Context, wanted map[string]string, enterprise, day string) (map[string]d.BusinessUnit, error) {
	var ids []string
	if wanted != nil {
		ids = make([]string, 0, len(wanted))
		for id := range wanted {
			ids = append(ids, id)
		}
	}
	return history.CaptureBUNavigation(ctx, u.tx, u.instance, enterprise, day, ids)
}
func compareBUEntry(a, b buListEntry, sort string) int {
	if sort == "name" || sort == "name_desc" {
		n := strings.Compare(a.name, b.name)
		if sort == "name_desc" {
			n = -n
		}
		if n != 0 {
			return n
		}
	}
	if n := strings.Compare(a.sortCode, b.sortCode); n != 0 {
		return n
	}
	return strings.Compare(a.summary.ID, b.summary.ID)
}
func (u *unit) Navigate(ctx context.Context, in org.ListInput, ent org.Enterprise, anchor, direction string) (out org.NavigationResult, err error) {
	rows, err := u.buListMatches(ctx, in, ent, true)
	if err != nil {
		return
	}
	sort.Slice(rows, func(i, j int) bool { return compareBUEntry(rows[i], rows[j], in.Sort) < 0 })
	index := -1
	for i := range rows {
		if rows[i].summary.ID == anchor {
			index = i
			break
		}
	}
	if index < 0 {
		return out, foundation.Fail(409, "M02_NAVIGATION_CHANGED", "当前记录不在查询结果中")
	}
	if direction == "previous" {
		index--
	} else if direction == "next" {
		index++
	}
	if index < 0 || index >= len(rows) {
		return out, nil
	}
	out.TargetID, out.Position, out.Count = &rows[index].summary.ID, index+1, len(rows)
	if index > 0 {
		out.PreviousID = &rows[index-1].summary.ID
	}
	if index+1 < len(rows) {
		out.NextID = &rows[index+1].summary.ID
	}
	return
}

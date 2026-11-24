package application

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
)

func (s Service) Navigate(ctx context.Context, c access.Credentials, in org.NavigationInput) (out org.NavigationResult, err error) {
	if !foundation.ValidID(in.AnchorID) || (in.Direction != "current" && in.Direction != "previous" && in.Direction != "next") {
		return out, foundation.Fail(400, "M02_INVALID_INPUT", "导航参数无效")
	}
	q := org.ListInput{Q: in.Query.Q, Status: in.Query.Status, Sort: in.Sort, Page: 1, PageSize: 20}
	if in.Query.FilterField != "" || in.Query.FilterOperator != "" || in.Query.FilterValue != "" {
		q.Filter = &org.ColumnFilter{Field: in.Query.FilterField, Operator: in.Query.FilterOperator, Value: in.Query.FilterValue}
	}
	if err = normalizeList(&q); err != nil {
		return
	}
	err = s.Store.Within(ctx, c, func(u Unit) error {
		ent, e := u.Enterprise(ctx, false)
		if e != nil {
			return e
		}
		now, e := u.Now(ctx)
		if e != nil {
			return e
		}
		q.BusinessAt = now.In(foundation.Shanghai).Format("2006-01-02")
		out, e = u.Navigate(ctx, q, ent, in.AnchorID, in.Direction)
		return e
	})
	return
}

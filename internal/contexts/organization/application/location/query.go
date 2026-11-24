package location

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"slices"
	"strings"
	"unicode/utf8"
)

var queryFields = []string{"code", "name", "nameEn", "addressLine", "postalCode", "status", "effectiveDate"}

func queryInput(v org.LocationSelectionQuery, sort string) org.LocationQuery {
	q := org.LocationQuery{ListInput: org.ListInput{Q: v.Q, Status: v.Status, Sort: sort, Page: 1, PageSize: 100}, CountryRegionID: v.CountryRegionID}
	if v.FilterField != "" || v.FilterOperator != "" || v.FilterValue != "" {
		q.Filter = &org.ColumnFilter{Field: v.FilterField, Operator: v.FilterOperator, Value: v.FilterValue}
	}
	return q
}
func normalizeList(q *org.LocationQuery) error {
	bad := func() error { return fault("M02_INVALID_INPUT", "", "查询条件无效") }
	if q.BusinessAt != "" || q.CountryRegionID != "" && !foundation.ValidID(q.CountryRegionID) {
		return bad()
	}
	if q.Page == 0 {
		q.Page = 1
	}
	if q.PageSize == 0 {
		q.PageSize = 20
	}
	if q.Page < 1 || !slices.Contains([]int{20, 50, 100}, q.PageSize) || q.Page-1 > int(^uint(0)>>1)/q.PageSize {
		return bad()
	}
	if q.Status == "all" {
		q.Status = ""
	}
	if !slices.Contains([]string{"", "active", "inactive", "not_effective"}, q.Status) {
		return bad()
	}
	q.Q = strings.ToLower(strings.TrimSpace(q.Q))
	if utf8.RuneCountInString(q.Q) > 400 {
		return bad()
	}
	if q.Sort != "" && !slices.Contains(queryFields, strings.TrimSuffix(q.Sort, "_desc")) {
		return bad()
	}
	if q.Filter != nil {
		f := *q.Filter
		switch f.Field {
		case "status":
			values := strings.Split(f.Value, ",")
			for _, v := range values {
				if !slices.Contains([]string{"active", "inactive", "not_effective"}, v) {
					return bad()
				}
			}
			slices.Sort(values)
			f.Value = strings.Join(slices.Compact(values), ",")
			if !slices.Contains([]string{"is", "isAnyOf"}, f.Operator) {
				return bad()
			}
		case "effectiveDate":
			if !foundation.ValidDate(f.Value) || !slices.Contains([]string{"is", "before", "after", "onOrBefore", "onOrAfter"}, f.Operator) {
				return bad()
			}
		default:
			f.Value = strings.ToLower(strings.TrimSpace(f.Value))
			limit := 500
			if f.Field == "addressLine" {
				limit = 1001
			}
			if !slices.Contains(queryFields, f.Field) || !slices.Contains([]string{"contains", "equals"}, f.Operator) || f.Value == "" || utf8.RuneCountInString(f.Value) > limit {
				return bad()
			}
		}
		q.Filter = &f
	}
	return nil
}
func authorizeQuery(ctx context.Context, u Unit, q org.LocationQuery, fields []string) error {
	has := func(f string) bool { return slices.Contains(fields, f) }
	if q.Sort != "" && !has(strings.TrimSuffix(q.Sort, "_desc")) || q.Status != "" && !has("status") || q.Filter != nil && !has(q.Filter.Field) || q.Q != "" && !has("code") && !has("name") {
		return denied()
	}
	if q.CountryRegionID != "" {
		if !has("countryRegionId") {
			return denied()
		}
		r, e := u.Access().Decide(ctx, access.Intent{Action: "country.read", Kind: "country-catalog"})
		if e != nil {
			return e
		}
		if !r.Allowed {
			return denied()
		}
	}
	return nil
}

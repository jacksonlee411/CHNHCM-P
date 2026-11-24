package application

import (
	"chnhcm.local/chnhcm/internal/contexts/organization/application/changelog"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"slices"
)

var changeFields = []string{"name", "nameEn", "effectiveDate", "period", "state", "operationType", "operationReason"}

func NormalizeChangeQuery(in *org.ChangeQuery) error { return changelog.Normalize(in, changeFields) }

func ChangeFieldAllowed(field string, fields []string) bool {
	if field == "state" {
		return true
	}
	if field == "period" {
		field = "effectiveDate"
	}
	return slices.Contains(fields, field)
}

func ChangeMatches(c org.Change, f *org.ColumnFilter, recordedDate string) bool {
	items := []changelog.Item{}
	for _, it := range c.Items {
		v := changelog.Item{Diffs: it.Diffs}
		for _, r := range []*org.ChangeRecordContext{it.BeforeContext, it.AfterContext} {
			if r != nil {
				v.Dates = append(v.Dates, r.EffectiveDate)
			}
		}
		items = append(items, v)
	}
	return changelog.Matches(c.Action, c.Operator, items, f, recordedDate)
}

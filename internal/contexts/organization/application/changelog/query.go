// Package changelog validates and filters authorized history projections.
package changelog

import (
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"slices"
	"strings"
	"unicode/utf8"
)

func Normalize(in *org.ChangeQuery, changeFields []string) error {
	invalid := func(field string) error {
		return &foundation.Fault{Status: 400, Code: "M02_INVALID_INPUT", Field: field, Message: "变更日志查询条件无效"}
	}
	if in.Page < 1 {
		in.Page = 1
	}
	if in.PageSize == 0 {
		in.PageSize = 20
	}
	if !slices.Contains([]int{20, 50, 100}, in.PageSize) {
		return invalid("pageSize")
	}
	for field, id := range map[string]string{"contextSourceVersion": in.ContextSourceVersion, "throughSourceVersion": in.ThroughSourceVersion} {
		if id != "" && !foundation.ValidID(id) {
			return invalid(field)
		}
	}
	f := in.Filter
	if f == nil {
		return nil
	}
	if f.Field == "" && f.Operator == "" && f.Value == "" {
		in.Filter = nil
		return nil
	}
	if f.Field == "" || f.Operator == "" || f.Value == "" {
		return invalid("filter")
	}
	switch f.Field {
	case "recordedAt", "effectiveDate":
		if !foundation.ValidDate(f.Value) || !slices.Contains([]string{"is", "before", "after", "onOrBefore", "onOrAfter"}, f.Operator) {
			return invalid("filter")
		}
	case "actor":
		f.Value = strings.ToLower(strings.TrimSpace(f.Value))
		if f.Value == "" || utf8.RuneCountInString(f.Value) > 400 || !slices.Contains([]string{"contains", "equals"}, f.Operator) {
			return invalid("filter")
		}
	case "action", "changedField":
		if !slices.Contains([]string{"is", "isAnyOf"}, f.Operator) {
			return invalid("filter")
		}
		allowed := changeFields
		if f.Field == "action" {
			allowed = []string{"CREATE", "ADD", "CORRECT", "REDATE", "DELETE", "DEACTIVATE", "REACTIVATE", "REVISE_TIMELINE"}
		}
		values := strings.Split(f.Value, ",")
		seen := map[string]bool{}
		if f.Operator == "is" && len(values) != 1 {
			return invalid("filter")
		}
		for _, v := range values {
			if seen[v] || !slices.Contains(allowed, v) {
				return invalid("filter")
			}
			seen[v] = true
		}
	default:
		return invalid("filter")
	}
	return nil
}

// Date comparison includes both endpoints of a changed effective date. Recorded
// timestamps are converted to a Shanghai business date before this comparison.
func MatchDate(value string, f org.ColumnFilter) bool {
	switch f.Operator {
	case "is":
		return value == f.Value
	case "before":
		return value < f.Value
	case "after":
		return value > f.Value
	case "onOrBefore":
		return value <= f.Value
	case "onOrAfter":
		return value >= f.Value
	}
	return false
}
func Matches(action string, operator org.OperatorSnapshot, items []Item, f *org.ColumnFilter, recordedDate string) bool {
	if f == nil {
		return true
	}
	switch f.Field {
	case "recordedAt":
		return MatchDate(recordedDate, *f)
	case "actor":
		values := []string{}
		if operator.IdentityEvidence == "subject_id_only" {
			values = append(values, operator.SubjectID)
		} else {
			for _, v := range []*string{operator.DisplayName, operator.Login} {
				if v != nil {
					values = append(values, *v)
				}
			}
		}
		for _, v := range values {
			v = strings.ToLower(strings.TrimSpace(v))
			if f.Operator == "equals" && v == f.Value || f.Operator == "contains" && strings.Contains(v, f.Value) {
				return true
			}
		}
	case "action":
		return slices.Contains(strings.Split(f.Value, ","), action)
	case "effectiveDate":
		for _, item := range items {
			for _, date := range item.Dates {
				if date != "" && MatchDate(date, *f) {
					return true
				}
			}
		}
	case "changedField":
		values := strings.Split(f.Value, ",")
		for _, item := range items {
			for _, d := range item.Diffs {
				if slices.Contains(values, d.Field) {
					return true
				}
			}
		}
	}
	return false
}

type Item struct {
	Dates []string
	Diffs []org.ChangeDiff
}

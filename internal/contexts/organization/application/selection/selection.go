// Package selection shares complete-set selection and safe CSV cells; adapters own projections and rights.
package selection

import (
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"strings"
	"unicode"
)

func Invalid() error {
	return foundation.Fail(400, "M02_INVALID_SELECTION", "选择内容无效，请重新选择")
}
func Rows[T any](rows []T, choice org.Selection, idOf func(T) string) ([]T, error) {
	var ids *[]string
	switch choice.Mode {
	case "explicit":
		if choice.ExcludedIDs != nil {
			return nil, Invalid()
		}
		ids = choice.IDs
	case "allMatching":
		if choice.IDs != nil {
			return nil, Invalid()
		}
		ids = choice.ExcludedIDs
	default:
		return nil, Invalid()
	}
	if ids == nil {
		return nil, Invalid()
	}
	available := map[string]bool{}
	for _, row := range rows {
		available[idOf(row)] = true
	}
	declared := map[string]bool{}
	for _, id := range *ids {
		if !foundation.ValidID(id) || declared[id] || !available[id] {
			return nil, Invalid()
		}
		declared[id] = true
	}
	selected := make([]T, 0, len(rows))
	for _, row := range rows {
		if declared[idOf(row)] == (choice.Mode == "explicit") {
			selected = append(selected, row)
		}
	}
	return selected, nil
}
func CSVText(value string) string {
	first := strings.TrimLeftFunc(value, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) })
	if strings.HasPrefix(value, "\t") || strings.HasPrefix(value, "\r") || strings.HasPrefix(value, "\n") || first != "" && strings.ContainsRune("=+-@", []rune(first)[0]) {
		return "'" + value
	}
	return value
}

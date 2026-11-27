package application

import (
	"cmp"
	"slices"

	config "chnhcm.local/chnhcm/internal/contexts/configuration/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

func Invalid(message string) error {
	return &foundation.Fault{Status: 422, Code: "X02_INVALID_SCOPE", Field: "scope", Message: message}
}

// Order the independent choices so reordering a selection retains its policy version.
func Normalize(scope config.Scope) (config.Scope, error) {
	scope.Choices = slices.Clone(scope.Choices)
	if scope.Mode == "ENTERPRISE" && len(scope.Choices) == 0 {
		return scope, nil
	}
	if scope.Mode != "SELECTED" || len(scope.Choices) == 0 {
		return scope, Invalid("请明确选择企业通用或指定适用对象")
	}
	seen := make(map[config.Choice]bool, len(scope.Choices))
	for _, c := range scope.Choices {
		if !slices.Contains([]string{"legalEntity", "jobFamily", "grade"}, c.Dimension) || !foundation.ValidID(c.ID) {
			return scope, Invalid("请选择有效的法律实体、职务系列或职级")
		}
		if seen[c] {
			return scope, Invalid("适用条件中包含重复选择")
		}
		seen[c] = true
	}
	slices.SortFunc(scope.Choices, func(a, b config.Choice) int {
		if n := cmp.Compare(a.Dimension, b.Dimension); n != 0 {
			return n
		}
		if n := cmp.Compare(a.ID, b.ID); n != 0 {
			return n
		}
		if a.Exclude == b.Exclude {
			return 0
		}
		if a.Exclude {
			return 1
		}
		return -1
	})
	return scope, nil
}

func Choices(p config.Policy) config.Scope {
	s := config.Scope{Mode: p.Mode}
	for _, m := range p.Members {
		s.Choices = append(s.Choices, m.Choice)
	}
	return s
}

// WithinDimension requires an explicit candidate when that dimension is used.
// An exclusion takes precedence even when the same member is also allowed.
func WithinDimension(scope config.Scope, dimension, id string) bool {
	hasAllow, matched, used := false, false, false
	for _, c := range scope.Choices {
		if c.Dimension != dimension {
			continue
		}
		used = true
		if c.Exclude && c.ID == id {
			return false
		}
		if !c.Exclude {
			hasAllow = true
			matched = matched || c.ID == id
		}
	}
	return !used || id != "" && (!hasAllow || matched)
}
func Matches(p config.Policy, subjects map[string]string) bool {
	if p.Mode == "ENTERPRISE" {
		return true
	}
	if p.Mode != "SELECTED" || len(p.Members) == 0 {
		return false
	}
	scope := Choices(p)
	for _, dim := range []string{"legalEntity", "jobFamily", "grade"} {
		if !WithinDimension(scope, dim, subjects[dim]) {
			return false
		}
	}
	return true
}

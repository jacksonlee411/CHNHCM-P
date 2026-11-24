package department

import (
	"context"
	"fmt"
	"slices"
)

// ReferenceValidator freezes reverse dependencies once per query. Each probe
// replaces only the target, then discovers its complete dated ancestor closure.
// Qualification is still performed by Validate over all affected boundaries.
type ReferenceValidator struct {
	snapshot    Snapshot
	target      string
	descendants []string
}

func NewReferenceValidator(s Snapshot, target string) *ReferenceValidator {
	if s.departmentIndex == nil || s.buIndex == nil || s.children == nil {
		s = s.Indexed()
	}
	ds := []string{target}
	seen := map[string]bool{target: true}
	for i := 0; i < len(ds); i++ {
		for _, child := range s.children[ds[i]] {
			if !seen[child] {
				seen[child] = true
				ds = append(ds, child)
			}
		}
	}
	return &ReferenceValidator{snapshot: s, target: target, descendants: ds}
}

func (v *ReferenceValidator) Validate(ctx context.Context, candidate Timeline) error {
	if candidate.ID != v.target {
		return fmt.Errorf("reference probe target changed")
	}
	s := v.snapshot
	s.replacement = &candidate
	ds := slices.Clone(v.descendants)
	seen, bus := make(map[string]bool, len(ds)), map[string]bool{}
	for _, id := range ds {
		seen[id] = true
	}
	for i := 0; i < len(ds); i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		if t := s.Department(ds[i]); t != nil {
			for j := range t.Records {
				r := &t.Records[j]
				if r.Data.BU != "" {
					bus[r.Data.BU] = true
				}
				if p := r.Data.Parent; p != "" && !seen[p] {
					seen[p] = true
					ds = append(ds, p)
				}
			}
		}
	}
	deps := Dependencies{Departments: ds, BUs: make([]string, 0, len(bus))}
	for id := range bus {
		deps.BUs = append(deps.BUs, id)
	}
	slices.Sort(deps.Departments)
	slices.Sort(deps.BUs)
	return Validate(ctx, s, deps)
}

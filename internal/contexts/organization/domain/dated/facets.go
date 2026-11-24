package dated

import "slices"

// DirectFacets excludes derived periods and inheritance. A no-change input may
// still receive a derived revision because another item changed its predecessor.
func (k Kernel[T, P]) DirectFacets(it Item[P], base, after []Record[T]) []string {
	if it.Action == "DELETE" {
		return []string{"DELETE"}
	}
	if it.Action == "ADD" {
		if slices.ContainsFunc(after, func(r Record[T]) bool { return r.ID == CandidateID(it) }) {
			return []string{"ADD"}
		}
		return []string{}
	}
	var b, a Record[T]
	for _, r := range base {
		if r.ID == it.RecordID {
			b = r
		}
	}
	for _, r := range after {
		if r.ID == it.RecordID {
			a = r
		}
	}
	facets := []string{}
	if !k.Policy.EqualData(b.Data, a.Data) {
		facets = append(facets, "CORRECT")
	}
	if b.Date != a.Date {
		facets = append(facets, "REDATE")
	}
	if b.Type != a.Type || b.Reason != a.Reason {
		facets = append(facets, "RECLASSIFY")
	}
	// Inherited value changes on a final DEACTIVATE are derived, not field corrections.
	if a.Type == "DEACTIVATE" {
		facets = slices.DeleteFunc(facets, func(f string) bool { return f == "CORRECT" })
	}
	return facets
}

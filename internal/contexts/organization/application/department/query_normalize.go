package department

import org "chnhcm.local/chnhcm/internal/contexts/organization/public"

// NormalizeQuery is the shared grammar for lists, selections and diagnostics.
func NormalizeQuery(in org.DepartmentListInput) (org.DepartmentListInput, error) {
	err := normalizeDepartmentList(&in)
	return in, err
}

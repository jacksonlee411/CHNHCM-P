package public

import "context"

// EarliestDate is evaluated from current heads, including inactive objects, in
// the enterprise writer's guarded transaction. It never discloses child details.
type EarliestDate interface {
	EarliestStaffingDate(context.Context, string) (string, error)
}

type PositionUse struct{ Start, End, GradeID string }
type OrganizationDependencies interface {
	DepartmentQuotaStart(context.Context, string, string) (string, error)
	EarliestDate
	PositionUses(context.Context, string, string, string) ([]PositionUse, error)
}

// PurposeUse describes a still-effective employment's need for legal coverage.
// Consumers require the enclosing legal-entity guard in the caller transaction.
type PurposeUse struct {
	From, Until, BU, Location string
}

package public

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"time"
)

// EnterpriseFloor supplies identity and the first-date floor under the caller's
// shared enterprise guard. Active coverage does not constrain Staffing catalogs.
type EnterpriseFloor interface {
	StaffingEnterprise(context.Context, bool) (Enterprise, error)
}

type PositionReferenceRecord struct {
	ID, Revision, Start, End, Name, Parent string
	Active                                 bool
}
type PositionReference struct {
	ID, Source, Code string
	Records          []PositionReferenceRecord
	DepartmentScope  *PositionDepartmentScope
}
type PositionDepartmentScope struct{ BusinessDate, Mode, ManagementBUID, RecordID, Revision string }
type DepartmentChoice struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Code string `json:"code,omitempty"`
}
type PositionDirectory interface {
	SearchQuotaDepartments(context.Context, foundation.Credentials, string, string, string, time.Time, int, int) ([]DepartmentChoice, error)
	// Match only the supplied department identities in today's authorized administrative branch.
	PositionDepartments(context.Context, foundation.Credentials, string, time.Time, string, bool, []string) ([]string, error)
	PositionCandidateIDs(context.Context, string, string, string, string, int, int) ([]string, error)
	PositionReferences(context.Context, string, string, []string, bool, time.Time) ([]PositionReference, error)
}

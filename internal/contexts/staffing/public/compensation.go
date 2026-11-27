package public

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
)

// CatalogSegment identifies one exact effective Staffing record. Its enclosing
// source is the source observed within the consumer's transaction.
type CatalogSegment struct {
	foundation.Period
	RecordID, Revision, Name string
	Active                   bool
}
type CompensationGrade struct {
	ID, Source, Code string
	Segments         []CatalogSegment
}
type CompensationGrades interface {
	CompensationGrade(context.Context, string, string, bool) (CompensationGrade, error)
}

package public

import (
	"context"

	"chnhcm.local/chnhcm/internal/foundation"
)

type CatalogSegment = foundation.CatalogSegment
type CompensationGrade = foundation.CatalogSource

type CompensationGrades interface {
	CompensationGrade(context.Context, string, string, bool) (CompensationGrade, error)
	CompensationJobFamily(context.Context, string, string, bool) (foundation.CatalogSource, error)
}

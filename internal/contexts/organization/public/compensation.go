package public

import (
	"context"

	"chnhcm.local/chnhcm/internal/foundation"
)

// CompensationLegalEntities exposes the actual entity timeline for plan applicability.
type CompensationLegalEntities interface {
	CompensationLegalEntity(context.Context, string, string, bool) (foundation.CatalogSource, error)
}

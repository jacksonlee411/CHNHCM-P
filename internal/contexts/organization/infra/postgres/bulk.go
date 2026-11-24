package postgres

import "chnhcm.local/chnhcm/internal/contexts/organization/domain"

type revisionInsert struct {
	domain.Record
	PreviousRevision *string
	Deleted          bool
}

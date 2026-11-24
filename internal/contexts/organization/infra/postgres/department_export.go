package postgres

import (
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	legalentity "chnhcm.local/chnhcm/internal/contexts/organization/application/legalentity"
	location "chnhcm.local/chnhcm/internal/contexts/organization/application/location"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/exportfile"
	"context"
)

func (s *Store) CreateExport(ctx context.Context) (app.ExportFile, error) {
	return exportfile.Default.Create(ctx)
}

func (s *Store) CreateLocationExport(ctx context.Context) (location.ExportFile, error) {
	return exportfile.Default.Create(ctx)
}

func (s *Store) CreateLegalEntityExport(ctx context.Context) (legalentity.ExportFile, error) {
	return exportfile.Default.Create(ctx)
}

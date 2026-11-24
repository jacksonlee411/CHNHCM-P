package legalentity

import (
	country "chnhcm.local/chnhcm/internal/contexts/organization/application/country"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/legalentity"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"context"
	"slices"
)

func projectCountry(ctx context.Context, u Unit, r d.Record, fields []string, _ string, out *org.LegalEntityRecord) error {
	if !slices.Contains(fields, "countryRegionId") {
		return nil
	}
	name, e := country.ProjectCountry(ctx, u.References(), r.Data.Adoption)
	if e != nil {
		return e
	}
	out.CountryName = name
	out.CountryRestricted = name == nil
	return nil
}

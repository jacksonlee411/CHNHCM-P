package location

import (
	geo "chnhcm.local/chnhcm/internal/contexts/organization/application/geoarea"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/location"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"context"
	"slices"
)

func projectAddress(ctx context.Context, u Unit, r d.Record, fields []string, today string, out *org.LocationRecord) error {
	if !slices.Contains(fields, "countryRegionId") && !slices.Contains(fields, "geoAreaPath") {
		return nil
	}
	path, e := geo.ProjectAdoption(ctx, u.References(), r.Data.Adoption, today)
	if e != nil {
		return e
	}
	out.AddressRestricted = len(path) != len(r.Data.Adoption.Parents)+1
	if len(path) > 0 && slices.Contains(fields, "countryRegionId") {
		out.CountryName = ptr(path[0].Name)
	}
	if !out.AddressRestricted && slices.Contains(fields, "geoAreaPath") {
		out.GeoAreaPath = path[1:]
	}
	return nil
}

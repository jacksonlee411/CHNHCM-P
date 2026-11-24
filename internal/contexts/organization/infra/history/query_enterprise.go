package history

import (
	"context"
	_ "embed"

	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/enterprise"
)

//go:embed query_enterprise_material.sql
var queryEnterpriseMaterialSQL string

type capturedEnterprise struct {
	ID, Source string
	IDs        []string
	Material   capturedMaterial `json:"_material"`
}

// The name is decoded from the same statement snapshot as department facts.
// Business-day selection remains in the shared Go dated kernel.
func decodeQueryEnterpriseName(ctx context.Context, id, day string, item capturedEnterprise) (string, error) {
	values, data, err := queryValues(item.IDs, item.Material)
	if err != nil {
		return "", queryUnavailable()
	}
	state, err := DecodeEnterpriseState(ctx, values, data)
	t := state.State.Timeline
	if err != nil || item.ID != id || t.ID != id || t.Source != item.Source || len(t.Records) == 0 {
		return "", queryUnavailable()
	}
	r := d.Kernel.FactAt(t.Records, day)
	if r == nil {
		r = &t.Records[0]
	}
	return r.Data.Name, nil
}

package postgres

import (
	"context"
	"errors"

	"chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
)

func (s *Store) CompensationLegalEntity(ctx context.Context, enterprise, id string, writing bool) (out foundation.CatalogSource, err error) {
	tx, ok := dbtx.From(ctx)
	if !ok {
		return out, errors.New("legal entity adoption requires a transaction")
	}
	s.historyCatalog.Enter()
	defer s.historyCatalog.Leave()
	u := &unit{tx: tx, instance: s.Instance, catalog: &s.historyCatalog}
	ent, err := u.Enterprise(ctx, writing)
	if err != nil || ent.ID != enterprise {
		return out, err
	}
	lu := &legalentityUnit{unit: u, enterprise: ent}
	if writing {
		if err = lu.lockEntity(ctx, id, false); err != nil {
			return out, err
		}
	}
	st, err := lu.Load(ctx, id, "")
	if err != nil || st.Timeline.ID == "" {
		return out, err
	}
	t := st.Timeline
	out = foundation.CatalogSource{ID: t.ID, Code: t.Code, Source: t.Source}
	for _, r := range t.Records {
		out.Segments = append(out.Segments, foundation.CatalogSegment{
			Period: foundation.Period{From: r.Date, Until: r.End}, RecordID: r.ID, Revision: r.Revision, Name: r.Data.Name, Active: r.State == "active",
		})
	}
	return out, nil
}

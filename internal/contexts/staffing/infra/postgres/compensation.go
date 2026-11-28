package postgres

import (
	"context"
	"errors"

	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	st "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
	"github.com/jackc/pgx/v5"
)

// CompensationGrade observes the grade head under the same object guard as
// Staffing maintenance. No salary data or compensation decisions live here.
func (s *Store) CompensationGrade(ctx context.Context, enterprise, id string, writing bool) (st.CompensationGrade, error) {
	return s.compensationCatalog(ctx, enterprise, id, writing, d.Grade)
}

func (s *Store) CompensationJobFamily(ctx context.Context, enterprise, id string, writing bool) (foundation.CatalogSource, error) {
	return s.compensationCatalog(ctx, enterprise, id, writing, d.Family)
}

func (s *Store) compensationCatalog(ctx context.Context, enterprise, id string, writing bool, kind d.Kind) (foundation.CatalogSource, error) {
	var out foundation.CatalogSource
	tx, ok := dbtx.From(ctx)
	if !ok {
		return out, errors.New("staffing catalog requires a transaction")
	}
	if writing {
		if err := lock(ctx, tx, true, objectGuard(kind), s.Instance, id); err != nil {
			return out, err
		}
	}
	var raw []byte
	query := "SELECT head FROM staffing.grade WHERE instance_id=$1 AND enterprise_id=$2 AND id=$3"
	if kind == d.Family {
		query = "SELECT head FROM staffing.job_family WHERE instance_id=$1 AND enterprise_id=$2 AND id=$3"
	}
	err := tx.QueryRow(ctx, query, s.Instance, enterprise, id).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	t, err := decode(raw, kind)
	if err != nil {
		return out, err
	}
	out = foundation.CatalogSource{ID: t.ID, Source: t.Source, Code: t.Code}
	for _, r := range t.Records {
		out.Segments = append(out.Segments, st.CatalogSegment{Period: foundation.Period{From: r.Date, Until: r.End}, RecordID: r.ID, Revision: r.Revision, Name: r.Data["name"], Active: r.Active()})
	}
	return out, nil
}

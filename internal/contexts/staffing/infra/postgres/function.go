package postgres

import (
	"context"
	"errors"

	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	"chnhcm.local/chnhcm/internal/contexts/staffing/infra/postgres/generated"
	"chnhcm.local/chnhcm/internal/foundation"
)

// Old snapshots remain byte-for-byte intact. An absent reference can only be
// hydrated from the exact cutover tuple, including when a later edit inherits it.
func (u *unit) hydrateFunctions(ctx context.Context, timelines []*d.Timeline) error {
	type key struct{ job, source, record, revision string }
	pending := map[key]bool{}
	var objects, sources []string
	for _, t := range timelines {
		for i := range t.Records {
			r := &t.Records[i]
			ref := r.Function
			id := r.Data["jobFunctionId"]
			if ref == (d.FunctionEvidence{}) && id == "" {
				ref = d.FunctionEvidence{Source: t.Source, RecordID: r.ID, Revision: r.Revision}
				r.Function = ref
			}
			if ref.FunctionID != id || !foundation.ValidID(ref.Source) || !foundation.ValidID(ref.RecordID) || !foundation.ValidID(ref.Revision) {
				return errors.New("incomplete job function evidence")
			}
			if id != "" {
				if !foundation.ValidID(id) || !foundation.ValidDate(ref.BusinessAt) || ref.BusinessAt > r.Date {
					return errors.New("invalid job function adoption")
				}
				continue
			}
			if ref.BusinessAt != "" {
				return errors.New("invalid legacy job function evidence")
			}
			k := key{t.ID, ref.Source, ref.RecordID, ref.Revision}
			if _, exists := pending[k]; !exists {
				objects = append(objects, t.ID)
				sources = append(sources, ref.Source)
				pending[k] = true
			}
		}
	}
	if len(pending) == 0 {
		return nil
	}
	rows, err := generated.New(u.tx).LegacyJobFunctionEvidence(ctx, generated.LegacyJobFunctionEvidenceParams{Instance: u.store.Instance, Objects: objects, Sources: sources})
	if err != nil {
		return err
	}
	for _, r := range rows {
		delete(pending, key{r.Job, r.Source, r.Record, r.Revision})
	}
	if len(pending) > 0 {
		return errors.New("missing pre-cutover job function evidence")
	}
	return nil
}

package postgres

import (
	"chnhcm.local/chnhcm/internal/contexts/organization/application/buauth"
	"context"
	"slices"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/domain"

	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
)

func correctionCapabilities(ctx context.Context, selected, first, start string, rows []buMemberRow, read, names []string, decide func(string, []string) (access.Decision, error)) (*org.CorrectionCapabilities, error) {
	out := &org.CorrectionCapabilities{Facets: []string{}, ClassificationTypes: []string{}}
	var target *buMemberRow
	base := make([]domain.Record, 0, len(rows))
	for i := range rows {
		r := &rows[i]
		base = append(base, domain.Record{ID: r.RecordID, Date: r.EffectiveDate})
		if r.RecordID == selected {
			target = r
		}
	}
	if target == nil {
		return out, nil
	}
	if slices.Contains(read, "operationType") && slices.Contains(read, "operationReason") {
		original, e := decide(buauth.ReclassifyAction(target.OperationType), []string{"operationType", "operationReason"})
		if e != nil {
			return nil, e
		}
		if original.Allowed {
			kinds := []string{"CHANGE", "DEACTIVATE", "REACTIVATE"}
			if selected == first {
				kinds = []string{"CREATE"}
			}
			for _, kind := range kinds {
				d, e := decide(buauth.ReclassifyAction(kind), []string{"operationType", "operationReason"})
				if e != nil {
					return nil, e
				}
				if d.Allowed {
					out.ClassificationTypes = append(out.ClassificationTypes, kind)
				}
			}
		}
	}
	if len(out.ClassificationTypes) > 0 {
		out.Facets = append(out.Facets, "RECLASSIFY")
	}
	canNames := false
	for _, f := range []string{"name", "nameEn"} {
		if slices.Contains(read, f) && slices.Contains(names, f) {
			canNames = true
		}
	}
	if canNames && (target.OperationType != "DEACTIVATE" || slices.ContainsFunc(out.ClassificationTypes, func(t string) bool { return t != "DEACTIVATE" })) {
		out.Facets = append(out.Facets, "CORRECT")
	}
	if slices.Contains(read, "effectiveDate") {
		d, e := decide(buauth.RedateAction(target.OperationType), []string{"effectiveDate"})
		if e != nil {
			return nil, e
		}
		if d.Allowed {
			out.Facets = append(out.Facets, "REDATE")
			out.MinEffectiveDate, out.MaxEffectiveDate, e = domain.CorrectionBounds(base, selected, start)
			if e != nil {
				return nil, e
			}
		}
	}
	return out, nil
}

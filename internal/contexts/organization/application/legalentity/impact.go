package legalentity

import (
	"chnhcm.local/chnhcm/internal/contexts/organization/domain/dated"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
)

func impact(v dated.Impact) *org.TimelineImpact {
	out := &org.TimelineImpact{DataAffectedRanges: []org.DateRange{}, StructuralChanges: []org.Structure{}, AffectedRecordIDs: v.AffectedRecordIDs, PeriodChanges: []org.Period{}}
	for _, r := range v.DataAffectedRanges {
		out.DataAffectedRanges = append(out.DataAffectedRanges, org.DateRange{Start: r.Start, End: r.End})
	}
	for _, s := range v.StructuralChanges {
		out.StructuralChanges = append(out.StructuralChanges, org.Structure{RecordID: s.RecordID, ItemID: s.ItemID, BeforeRevision: s.BeforeRevision, AfterRevision: s.AfterRevision, Facets: s.Facets})
	}
	convert := func(r *dated.Range) *org.DateRange {
		if r == nil {
			return nil
		}
		return &org.DateRange{Start: r.Start, End: r.End}
	}
	for _, p := range v.PeriodChanges {
		out.PeriodChanges = append(out.PeriodChanges, org.Period{RecordID: p.RecordID, ItemID: p.ItemID, Before: convert(p.Before), After: convert(p.After)})
	}
	return out
}

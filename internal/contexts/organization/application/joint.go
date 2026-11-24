package application

import (
	"errors"
	"slices"
	"strings"

	"chnhcm.local/chnhcm/internal/contexts/organization/domain"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

func validatePairIdentity(items []domain.Item) error {
	ids, targets := map[string]bool{}, map[string]bool{}
	for _, it := range items {
		if !foundation.ValidID(it.ID) || ids[it.ID] || it.Action != "ADD" && targets[it.RecordID] {
			return foundation.Fail(400, "M02_REVISION_ITEM_CONFLICT", "当前修改包含重复或无效的记录")
		}
		ids[it.ID] = true
		if it.Action != "ADD" {
			if !foundation.ValidID(it.RecordID) || !foundation.ValidID(it.ExpectedRevision) {
				return foundation.Fail(400, "M02_INVALID_INPUT", "修改内容不完整，请重新打开表单")
			}
			targets[it.RecordID] = true
		}
	}
	return nil
}
func pairItemMetadata(items []domain.Item, base []domain.Record) []org.RequestItem {
	out := []org.RequestItem{}
	for _, it := range items {
		v := org.RequestItem{ItemID: it.ID, Action: it.Action, RecordID: it.RecordID, ExpectedRecordRevision: it.ExpectedRevision, DeclaredFields: []string{}, ProcessingFacets: []string{}}
		if it.Action == "ADD" {
			v.ProcessingFacets = []string{"ADD"}
			v.EffectiveDate = it.Date
			v.NewType = it.Type
			v.DeclaredFields = []string{"effectiveDate", "operationType", "operationReason"}
		}
		if it.Action == "DELETE" {
			v.ProcessingFacets = []string{"DELETE"}
		}
		for _, b := range base {
			if b.ID == it.RecordID && b.Revision == it.ExpectedRevision {
				v.OriginalType = b.Type
			}
		}
		if v.NewType == "" && it.Action != "ADD" {
			v.NewType = v.OriginalType
		}
		if it.Action == "ADD" {
			v.ProcessingFacets = []string{"ADD"}
		}
		slices.Sort(v.DeclaredFields)
		v.DeclaredFields = slices.Compact(v.DeclaredFields)
		slices.Sort(v.ProcessingFacets)
		out = append(out, v)
	}
	slices.SortFunc(out, func(a, b org.RequestItem) int { return strings.Compare(a.ItemID, b.ItemID) })
	return out
}
func jointFailure(err error, metadata ...[]org.RequestItem) *org.FailureDetails {
	var f *foundation.Fault
	var item *domain.ItemFault
	issue := org.Issue{Code: "M02_INVALID_INPUT", Locations: []org.Location{}}
	if errors.As(err, &f) {
		issue.Code = f.Code
	}
	loc := org.Location{}
	if f != nil {
		loc.FieldPath = f.Field
	}
	if errors.As(err, &item) {
		loc.ItemID = item.ItemID
	}
	if loc.ItemID == "" {
		loc.FieldPath = ""
	}
	if len(metadata) > 0 {
		for _, it := range metadata[0] {
			if it.ItemID == loc.ItemID && it.Action == "UPDATE" && loc.FieldPath == "effectiveDate" {
				loc.FieldPath = "newEffectiveDate"
			}
		}
	}
	if loc.ItemID != "" || loc.FieldPath != "" {
		issue.Locations = append(issue.Locations, loc)
	}
	return &org.FailureDetails{FormatVersion: 1, Issues: []org.Issue{issue}}
}
func jointOutcomes(base []domain.Record, c domain.Candidate, items []domain.Item) []org.ItemResult {
	out := []org.ItemResult{}
	for _, it := range items {
		facets := domain.DirectFacets(it, base, c.Records)
		v := org.ItemResult{ItemID: it.ID, Outcome: "no_change", RecordID: it.RecordID, ProcessingFacets: facets, EffectiveDate: it.Date}
		if len(facets) > 0 {
			v.Outcome = "accepted"
		}
		for _, b := range base {
			if b.ID == it.RecordID {
				r := b.Revision
				v.BeforeRevisionID = &r
				if v.EffectiveDate == "" {
					v.EffectiveDate = b.Date
				}
			}
		}
		if it.Action == "ADD" {
			v.RecordID = ""
		}
		if it.Action == "ADD" && v.Outcome == "accepted" {
			v.RecordID = domain.CandidateID(it)
		}
		out = append(out, v)
	}
	slices.SortFunc(out, func(a, b org.ItemResult) int { return strings.Compare(a.ItemID, b.ItemID) })
	return out
}
func trimJointResult(r *org.Result, fields []string) {
	for i := range r.Items {
		v := &r.Items[i]
		if !slices.Contains(fields, "effectiveDate") {
			v.EffectiveDate = ""
			v.DateDescription = ""
		}
		for _, p := range []*org.Record{v.BeforeRecord, v.AfterRecord} {
			if p == nil {
				continue
			}
			if !slices.Contains(fields, "name") {
				p.Name = nil
			}
			if !slices.Contains(fields, "nameEn") {
				p.NameEn = nil
			}
			if !slices.Contains(fields, "effectiveDate") {
				p.EffectiveDate = ""
				p.EndExclusive = nil
				p.EndInclusive = nil
			}
			if !slices.Contains(fields, "operationType") {
				p.OperationType = ""
				p.TypeLabel = ""
			}
			if !slices.Contains(fields, "operationReason") {
				p.OperationReason = ""
				p.ReasonLabel = ""
			}
		}
	}
}

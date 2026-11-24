package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"

	"chnhcm.local/chnhcm/internal/contexts/organization/application"
	"chnhcm.local/chnhcm/internal/contexts/organization/domain"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

func (u *unit) persistJointItems(ctx context.Context, tl application.Timeline, source string, t *application.Terminal, c domain.Candidate) error {
	records := make(map[string]domain.Record, len(c.Records)+len(c.Deleted))
	for _, members := range [][]domain.Record{c.Records, c.Deleted} {
		for _, member := range members {
			records[member.ID] = member
		}
	}
	for i := range t.ItemOutcomes {
		v := &t.ItemOutcomes[i]
		if v.Outcome != "accepted" {
			continue
		}
		record, ok := records[v.RecordID]
		if !ok || record.Revision == "" {
			return errors.New("joint direct revision missing")
		}
		v.EffectiveDate = record.Date
		v.AfterRevisionID = &record.Revision
	}
	return nil
}

func strictJSON(raw []byte, out any) error {
	d := json.NewDecoder(strings.NewReader(string(raw)))
	d.DisallowUnknownFields()
	return d.Decode(out)
}
func decodeJointTerminal(request, outcomes []byte, outcome string) ([]org.ItemResult, error) {
	var req []org.RequestItem
	var out []org.ItemResult
	bad := resultUnavailable()
	if strictJSON(request, &req) != nil || strictJSON(outcomes, &out) != nil || req == nil || out == nil {
		return nil, bad
	}
	ids := map[string]org.RequestItem{}
	previous := ""
	for _, it := range req {
		if !foundation.ValidID(it.ItemID) || it.ItemID <= previous || it.DeclaredFields == nil || it.ProcessingFacets == nil || !slices.IsSorted(it.DeclaredFields) {
			return nil, bad
		}
		previous = it.ItemID
		if !slices.Contains([]string{"ADD", "UPDATE", "DELETE"}, it.Action) {
			return nil, bad
		}
		if it.Action != "ADD" && (!foundation.ValidID(it.RecordID) || !foundation.ValidID(it.ExpectedRecordRevision)) {
			return nil, bad
		}
		for n, f := range it.DeclaredFields {
			if !slices.Contains([]string{"name", "nameEn", "effectiveDate", "operationType", "operationReason"}, f) || n > 0 && it.DeclaredFields[n-1] == f {
				return nil, bad
			}
		}
		if !validRequestItem(it, outcome) {
			return nil, bad
		}
		ids[it.ItemID] = it
	}
	if outcome == "rejected" {
		if len(out) != 0 {
			return nil, bad
		}
		return out, nil
	}
	if len(req) == 0 || len(out) != len(req) {
		return nil, bad
	}
	accepted := 0
	seen := map[string]bool{}
	for _, v := range out {
		m, ok := ids[v.ItemID]
		if !ok || seen[v.ItemID] || !slices.Contains([]string{"accepted", "no_change"}, v.Outcome) || v.ProcessingFacets == nil || v.BeforeRecord != nil || v.AfterRecord != nil || v.DateDescription != "" {
			return nil, bad
		}
		seen[v.ItemID] = true
		if v.Outcome == "accepted" {
			accepted++
			if !foundation.ValidID(v.RecordID) || v.AfterRevisionID == nil || !foundation.ValidID(*v.AfterRevisionID) || len(v.ProcessingFacets) == 0 {
				return nil, bad
			}
		}
		if v.Outcome == "no_change" && (v.AfterRevisionID != nil || len(v.ProcessingFacets) > 0) {
			return nil, bad
		}
		if m.Action == "ADD" {
			if v.BeforeRevisionID != nil || v.Outcome == "no_change" && v.RecordID != "" {
				return nil, bad
			}
		} else if v.RecordID != m.RecordID || v.BeforeRevisionID == nil || *v.BeforeRevisionID != m.ExpectedRecordRevision {
			return nil, bad
		}
	}
	if (outcome == "accepted") != (accepted > 0) {
		return nil, bad
	}
	return out, nil
}
func (u *unit) hydrateJointResult(ctx context.Context, out *org.Result, raw []byte, expected *string) error {
	if out.BUID == nil || expected == nil {
		return resultUnavailable()
	}
	var metadata []org.RequestItem
	if strictJSON(raw, &metadata) != nil {
		return resultUnavailable()
	}
	before, e := u.readBU(ctx, *out.BUID, *expected)
	if e != nil {
		return e
	}
	records := map[string]*org.Record{}
	members := map[string]bool{}
	add := func(r domain.Record, deleted, member bool) {
		p := application.PublicRecord(r, []string{"name", "nameEn", "effectiveDate", "operationType", "operationReason"})
		p.Deleted = deleted
		key := r.ID + "/" + r.Revision
		records[key] = &p
		members[key] = member
	}
	for _, r := range before.State.Timeline.Records {
		add(r, false, true)
	}
	type directItem struct {
		record, after, action string
		before                *string
		facets                []string
	}
	direct := map[string]directItem{}
	if out.Outcome == "accepted" {
		if out.SourceVersion == nil {
			return resultUnavailable()
		}
		after, e := u.readBU(ctx, *out.BUID, *out.SourceVersion)
		if e != nil {
			return e
		}
		for _, r := range after.Operation.Revisions {
			add(r.Record, r.Deleted, false)
		}
		for _, it := range after.Operation.Terminal.ItemOutcomes {
			if it.Outcome != "accepted" {
				continue
			}
			action := ""
			for _, m := range after.Operation.Terminal.Items {
				if m.ItemID == it.ItemID {
					action = m.Action
				}
			}
			direct[it.ItemID] = directItem{it.RecordID, value(it.AfterRevisionID), action, it.BeforeRevisionID, it.ProcessingFacets}
		}
		n := 0
		for _, it := range out.Items {
			if it.Outcome == "accepted" {
				n++
			}
		}
		if len(direct) != n {
			return resultUnavailable()
		}
	}
	for i := range out.Items {
		v := &out.Items[i]
		if v.BeforeRevisionID != nil {
			key := v.RecordID + "/" + *v.BeforeRevisionID
			if !members[key] || records[key] == nil {
				return resultUnavailable()
			}
			v.BeforeRecord = records[key]
		}
		if v.Outcome == "accepted" {
			if out.SourceVersion == nil || v.AfterRevisionID == nil {
				return resultUnavailable()
			}
			d, ok := direct[v.ItemID]
			if !ok || d.record != v.RecordID || d.after != *v.AfterRevisionID || !slices.Equal(d.facets, v.ProcessingFacets) || (d.before == nil) != (v.BeforeRevisionID == nil) || d.before != nil && *d.before != *v.BeforeRevisionID {
				return resultUnavailable()
			}
			for _, m := range metadata {
				if m.ItemID == v.ItemID && m.Action != d.action {
					return resultUnavailable()
				}
			}
			r := records[v.RecordID+"/"+d.after]
			if r == nil {
				return resultUnavailable()
			}
			v.AfterRecord = r
			v.EffectiveDate = r.EffectiveDate
		}
		expectedDate := ""
		originalAction := ""
		for _, m := range metadata {
			if m.ItemID == v.ItemID {
				expectedDate = m.EffectiveDate
				originalAction = m.Action
				if expectedDate == "" && v.BeforeRecord != nil {
					expectedDate = v.BeforeRecord.EffectiveDate
				}
			}
		}
		if v.EffectiveDate != expectedDate {
			return resultUnavailable()
		}
		v.EffectiveDate = expectedDate
		if originalAction == "ADD" && out.EvaluatedBusinessDate != nil && v.EffectiveDate != "" {
			v.DateDescription = "当天生效"
			if v.EffectiveDate < *out.EvaluatedBusinessDate {
				v.DateDescription = "过去补登记"
			}
			if v.EffectiveDate > *out.EvaluatedBusinessDate {
				v.DateDescription = "未来安排"
			}
		}
	}
	return nil
}

func validRequestItem(v org.RequestItem, outcome string) bool {
	fields := map[string]bool{}
	for _, f := range v.DeclaredFields {
		fields[f] = true
	}
	facets := map[string]bool{}
	for _, f := range v.ProcessingFacets {
		if facets[f] || !slices.Contains([]string{"ADD", "DELETE", "CORRECT", "REDATE", "RECLASSIFY"}, f) {
			return false
		}
		facets[f] = true
	}
	if v.EffectiveDate != "" && !foundation.ValidDate(v.EffectiveDate) {
		return false
	}
	if outcome != "rejected" && len(facets) == 0 {
		return false
	}
	switch v.Action {
	case "ADD":
		if len(facets) != 1 || !facets["ADD"] || v.RecordID != "" || v.ExpectedRecordRevision != "" || v.OriginalType != "" {
			return false
		}
		for _, f := range []string{"effectiveDate", "operationType", "operationReason"} {
			if !fields[f] {
				return false
			}
		}
		if outcome != "rejected" && (v.EffectiveDate == "" || !slices.Contains([]string{"CHANGE", "DEACTIVATE", "REACTIVATE"}, v.NewType)) {
			return false
		}
		if v.NewType == "DEACTIVATE" && (fields["name"] || fields["nameEn"]) {
			return outcome == "rejected"
		}
	case "DELETE":
		if len(facets) != 1 || !facets["DELETE"] || len(fields) != 0 || v.EffectiveDate != "" {
			return false
		}
	case "UPDATE":
		if facets["ADD"] || facets["DELETE"] {
			return false
		}
		if facets["CORRECT"] != (fields["name"] || fields["nameEn"]) || facets["REDATE"] != fields["effectiveDate"] || facets["RECLASSIFY"] != fields["operationType"] || facets["RECLASSIFY"] != fields["operationReason"] {
			return false
		}
		if facets["REDATE"] != (v.EffectiveDate != "") {
			return false
		}
	default:
		return false
	}
	if outcome != "rejected" && v.Action != "ADD" {
		if !slices.Contains([]string{"CREATE", "CHANGE", "DEACTIVATE", "REACTIVATE"}, v.OriginalType) || !slices.Contains([]string{"CREATE", "CHANGE", "DEACTIVATE", "REACTIVATE"}, v.NewType) {
			return false
		}
	}
	return true
}
func validJointDiagnostics(d org.FailureDetails, items []org.RequestItem) bool {
	ids := map[string]org.RequestItem{}
	for _, it := range items {
		ids[it.ItemID] = it
	}
	seen := map[string]bool{}
	for _, issue := range d.Issues {
		if !slices.Contains([]string{"M02_INVALID_INPUT", "M02_FORBIDDEN_INPUT", "M02_REVISION_ITEM_CONFLICT", "M02_RECORD_NOT_FOUND", "M02_PAIR_NOT_FOUND", "M02_REVISION_CONFLICT", "M02_SOURCE_CONFLICT", "M02_FIRST_RECORD_PROTECTED", "M02_TIMELINE_INVALID", "M02_ACTION_NOT_ALLOWED", "M02_DATE_CONFLICT", "M02_REFERENCE_CONFLICT", "ACCESS_DENIED"}, issue.Code) {
			return false
		}
		for _, l := range issue.Locations {
			if l.ItemID == "" {
				if l.FieldPath != "" {
					return false
				}
				continue
			}
			it, exists := ids[l.ItemID]
			if !exists {
				return false
			}
			paths := []string{""}
			switch it.Action {
			case "ADD":
				paths = append(paths, "effectiveDate", "operationType", "operationReason", "changes", "changes.name", "changes.nameEn")
			case "UPDATE":
				paths = append(paths, "newEffectiveDate", "changes", "changes.name", "changes.nameEn", "classification.operationType", "classification.operationReason")
			}
			if !slices.Contains(paths, l.FieldPath) {
				return false
			}
			key := issue.Code + ":" + l.ItemID + ":" + l.FieldPath
			if seen[key] {
				return false
			}
			seen[key] = true
		}
	}
	return true
}

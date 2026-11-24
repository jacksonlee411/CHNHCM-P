package enterprise

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/application/changelog"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/enterprise"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"reflect"
	"slices"
	"strings"
	"time"
)

var changeFields = []string{"name", "nameEn", "effectiveDate", "period", "state", "operationType", "operationReason"}

func changeAllowed(field string, fields []string) bool {
	if field == "period" {
		field = "effectiveDate"
	}
	return slices.Contains(fields, field)
}
func changeValue(r *org.EnterpriseRecord, field string) org.ChangeValue {
	if r == nil {
		return org.ChangeValue{Kind: "missing"}
	}
	text := func(v *string) org.ChangeValue {
		if v == nil {
			return org.ChangeValue{Kind: "empty"}
		}
		return org.ChangeValue{Kind: "text", Text: v}
	}
	switch field {
	case "name":
		return text(r.Name)
	case "nameEn":
		if r.NameEn != nil {
			return text(*r.NameEn)
		}
	case "effectiveDate":
		v := text(r.EffectiveDate)
		v.Kind = "date"
		return v
	case "period":
		v := org.ChangeValue{Kind: "period", EndExclusive: r.EndExclusive, EndInclusive: r.EndInclusive}
		if r.EffectiveDate != nil {
			v.Start = *r.EffectiveDate
		}
		return v
	case "state":
		v := text(r.State)
		v.Kind = "state"
		return v
	case "operationType":
		v := text(r.OperationType)
		if r.TypeLabel != nil {
			v.Label = *r.TypeLabel
		}
		return v
	case "operationReason":
		v := text(r.OperationReason)
		if r.ReasonLabel != nil {
			v.Label = *r.ReasonLabel
		}
		return v
	}
	return org.ChangeValue{Kind: "empty"}
}
func (s Service) Changes(ctx context.Context, c access.Credentials, q org.ChangeQuery) (out org.EnterpriseChanges, err error) {
	if !foundation.ValidID(q.ID) {
		return out, fault("M02_INVALID_INPUT", "", "企业标识无效")
	}
	if err = changelog.Normalize(&q, changeFields); err != nil {
		return
	}
	err = s.read(ctx, c, func(u Unit) error {
		fields, e := readable(ctx, u, true)
		if e != nil {
			return e
		}
		detailFields, e := readable(ctx, u, q.ContextSourceVersion != "")
		if e != nil {
			return e
		}
		if q.Filter != nil {
			f := q.Filter
			if f.Field == "effectiveDate" && !changeAllowed("effectiveDate", fields) {
				return denied()
			}
			if f.Field == "changedField" {
				for _, field := range strings.Split(f.Value, ",") {
					if !changeAllowed(field, fields) {
						return denied()
					}
				}
			}
		}
		state, e := u.Load(ctx, q.ID, q.ContextSourceVersion)
		if e != nil {
			return e
		}
		if state.Timeline.ID == "" {
			return missing()
		}
		through := q.ThroughSourceVersion
		if through == "" {
			through = state.Timeline.Source
		}
		sources, e := u.History(ctx, q.ID, through)
		if e != nil {
			return e
		}
		if len(sources) == 0 {
			return missing()
		}
		if sources[0].Number > state.Timeline.SourceNo {
			return fault("M02_INVALID_INPUT", "throughSourceVersion", "日志上界不能超过所查看的来源")
		}
		out = org.EnterpriseChanges{Items: []org.EnterpriseChange{}, ReadFields: fields, DetailReadFields: detailFields, Page: q.Page, PageSize: q.PageSize, ContextSourceVersion: ptr(q.ContextSourceVersion), ThroughSourceVersion: through}
		matches := []org.EnterpriseChange{}
		for _, op := range sources {
			row := org.EnterpriseChange{ID: op.ID, SourceVersion: op.ID, PreviousSourceVersion: ptr(op.Previous), RecordedAt: op.RecordedAt.Format(time.RFC3339Nano), Action: op.Action, Operator: op.Operator, Items: []org.EnterpriseChangeItem{}}
			filterItems := []changelog.Item{}
			for _, revision := range op.Revisions {
				item := org.EnterpriseChangeItem{ChangeKind: "direct", Diffs: []org.ChangeDiff{}}
				dates := []string{}
				for i, r := range []*d.Record{revision.Before, revision.After} {
					if r == nil {
						continue
					}
					v := project(*r, fields, nil)
					item.RecordID = r.ID
					if v.EffectiveDate != nil {
						dates = append(dates, *v.EffectiveDate)
					}
					if i == 0 {
						item.BeforeContext = &v
						item.BeforeRef = &org.ChangeRef{SourceVersion: op.Previous, RecordID: r.ID, Revision: r.Revision}
					} else {
						item.AfterContext = &v
						item.AfterRef = &org.ChangeRef{SourceVersion: op.ID, RecordID: r.ID, Revision: r.Revision}
					}
				}
				if op.Terminal.Result.TargetRecordID != nil && item.RecordID != *op.Terminal.Result.TargetRecordID && !(op.Action == "INSERT_LIFECYCLE_PAIR" && revision.Before == nil) && !(op.Action == "DELETE_LIFECYCLE_PAIR" && revision.After == nil) {
					item.ChangeKind = "derived"
				}
				for _, field := range changeFields {
					if !changeAllowed(field, fields) {
						continue
					}
					a, b := changeValue(item.BeforeContext, field), changeValue(item.AfterContext, field)
					if !reflect.DeepEqual(a, b) {
						item.Diffs = append(item.Diffs, org.ChangeDiff{Field: field, Before: a, After: b})
					}
				}
				filterItems = append(filterItems, changelog.Item{Dates: dates, Diffs: item.Diffs})
				row.Items = append(row.Items, item)
			}
			if changelog.Matches(row.Action, row.Operator, filterItems, q.Filter, op.RecordedAt.In(foundation.Shanghai).Format("2006-01-02")) {
				matches = append(matches, row)
			}
		}
		out.Total = len(matches)
		out.Page = min(q.Page, max(1, (out.Total+q.PageSize-1)/q.PageSize))
		start := min((out.Page-1)*q.PageSize, out.Total)
		out.Items = append(out.Items, matches[start:min(start+q.PageSize, out.Total)]...)
		return nil
	})
	return
}

var _ org.EnterpriseService = Service{}

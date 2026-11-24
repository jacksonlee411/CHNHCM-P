package department

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/application/changelog"
	"chnhcm.local/chnhcm/internal/contexts/organization/application/submission"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"reflect"
	"slices"
	"strings"
	"time"
)

var departmentChangeFields = []string{"name", "shortName", "managementBuId", "parentDepartmentId", "effectiveDate", "period", "state", "operationType", "operationReason"}

func changeFieldAllowed(field string, fields []string) bool {
	if field == "period" {
		field = "effectiveDate"
	}
	if field == "state" {
		field = "status"
	}
	return slices.Contains(fields, field)
}
func departmentChangeValue(r *org.DepartmentRecord, field string) org.ChangeValue {
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
	case "shortName":
		if r.ShortName == nil {
			return org.ChangeValue{Kind: "empty"}
		}
		return text(*r.ShortName)
	case "managementBuId", "parentDepartmentId":
		id, ref := r.ManagementBUID, r.ManagementBU
		if field == "parentDepartmentId" {
			id, ref = r.ParentDepartmentID, r.ParentDepartment
		}
		v := text(id)
		if id != nil {
			v.Kind = "reference"
			if ref != nil && ref.Name != nil {
				v.Label = *ref.Name
			}
		}
		return v
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
		v := org.ChangeValue{Kind: "state"}
		if r.State != nil {
			v.Code = *r.State
		}
		return v
	case "operationType", "operationReason":
		code, label := r.OperationType, r.TypeLabel
		if field == "operationReason" {
			code, label = r.OperationReason, r.ReasonLabel
		}
		v := org.ChangeValue{Kind: "classification"}
		if code != nil {
			v.Code = *code
		}
		if label != nil {
			v.Label = *label
		}
		return v
	}
	return org.ChangeValue{Kind: "empty"}
}
func historyProjection(p *projector, source HistorySource, fields []string) (org.DepartmentChange, error) {
	out := org.DepartmentChange{SourceVersion: source.ID, PreviousSourceVersion: ptr(source.Previous), RecordedAt: source.At, Operator: source.Operator, Action: source.Action, Items: []org.DepartmentChangeItem{}}
	for _, it := range source.Items {
		item := org.DepartmentChangeItem{RecordID: it.RecordID, ChangeKind: it.Role, Diffs: []org.ChangeDiff{}}
		if it.Before != nil {
			r, e := p.record(it.Before.Record, fields, nil)
			if e != nil {
				return out, e
			}
			item.BeforeContext = &r
			item.BeforeRef = &org.ChangeRef{SourceVersion: it.BeforeSource, RecordID: it.RecordID, Revision: it.Before.Record.Revision}
		}
		if it.After != nil && !it.After.Deleted {
			r, e := p.record(it.After.Record, fields, nil)
			if e != nil {
				return out, e
			}
			item.AfterContext = &r
			item.AfterRef = &org.ChangeRef{SourceVersion: source.ID, RecordID: it.RecordID, Revision: it.After.Record.Revision}
		}
		for _, field := range departmentChangeFields {
			if !changeFieldAllowed(field, fields) {
				continue
			}
			a, b := departmentChangeValue(item.BeforeContext, field), departmentChangeValue(item.AfterContext, field)
			if (field == "shortName" || field == "parentDepartmentId") && (a.Kind == "missing" && b.Kind == "empty" || a.Kind == "empty" && b.Kind == "missing") {
				continue
			}
			changed := !reflect.DeepEqual(a, b)
			if field == "operationReason" && it.Before != nil && it.After != nil && it.Before.Record.Type != it.After.Record.Type {
				changed = true
			}
			if changed {
				item.Diffs = append(item.Diffs, org.ChangeDiff{Field: field, Before: a, After: b})
			}
		}
		out.Items = append(out.Items, item)
	}
	return out, nil
}
func (s Service) Changes(ctx context.Context, c access.Credentials, in org.DepartmentChangeQuery) (out org.DepartmentChanges, err error) {
	q := org.ChangeQuery{ID: in.DepartmentID, Page: in.Page, PageSize: in.PageSize, ContextSourceVersion: in.ContextSourceVersion, ThroughSourceVersion: in.ThroughSourceVersion, Filter: in.Filter}
	if !foundation.ValidID(q.ID) {
		return out, foundation.Fail(400, "M02_INVALID_INPUT", "所选部门无效，请重新选择")
	}
	if e := changelog.Normalize(&q, departmentChangeFields); e != nil {
		return out, e
	}
	err = s.Store.WithinDepartment(ctx, c, func(u Unit) error {
		ent, e := u.Enterprise(ctx, false)
		if e != nil {
			return e
		}
		now, e := u.Now(ctx)
		if e != nil {
			return e
		}
		read, e := u.Snapshot(ctx, q.ID, "")
		if e != nil {
			return e
		}
		t := read.Snapshot.Department(q.ID)
		if t == nil {
			return submission.RejectRead(missing())
		}
		fields, e := readable(ctx, u, *t, now, true)
		if e != nil {
			return submission.RejectRead(e)
		}
		detailFields := fields
		if q.ContextSourceVersion == "" {
			detailFields, e = readable(ctx, u, *t, now, false)
			if e != nil {
				return submission.RejectRead(e)
			}
		}
		if q.Filter != nil {
			if q.Filter.Field == "effectiveDate" && !changeFieldAllowed("effectiveDate", fields) {
				return submission.RejectRead(denied())
			}
			if q.Filter.Field == "changedField" {
				for _, field := range strings.Split(q.Filter.Value, ",") {
					if !changeFieldAllowed(field, fields) {
						return submission.RejectRead(denied())
					}
				}
			}
		}
		through := q.ThroughSourceVersion
		if through == "" {
			through = t.Source
			if q.ContextSourceVersion != "" {
				through = q.ContextSourceVersion
			}
		}
		sources, e := u.History(ctx, q.ID, through)
		if e != nil {
			return e
		}
		if len(sources) == 0 {
			return submission.RejectRead(missing())
		}
		if q.ContextSourceVersion != "" && q.ContextSourceVersion != through {
			contextSource, e := u.Source(ctx, q.ID, q.ContextSourceVersion)
			if e != nil {
				return e
			}
			if sources[0].No > contextSource.No {
				return submission.RejectRead(foundation.Fail(400, "M02_INVALID_LOG_RANGE", "日志查询范围无效，请重置日志条件"))
			}
		}

		out = org.DepartmentChanges{ReadFields: fields, DetailReadFields: detailFields, Items: []org.DepartmentChange{}, Page: q.Page, PageSize: q.PageSize, ThroughSourceVersion: through, ContextSourceVersion: ptr(q.ContextSourceVersion)}
		projectedPage := q.Filter == nil
		if projectedPage {
			out.Total = len(sources)
			out.Page = min(q.Page, max(1, (len(sources)+q.PageSize-1)/q.PageSize))
			start := min((out.Page-1)*q.PageSize, len(sources))
			sources = sources[start:min(start+q.PageSize, len(sources))]
		}
		p := newProjector(ctx, u, read.Snapshot, ent, now)
		all := []org.DepartmentChange{}
		for _, source := range sources {
			if e := ctx.Err(); e != nil {
				return e
			}
			v, e := historyProjection(p, source, fields)
			if e != nil {
				return e
			}
			at, e := time.Parse(time.RFC3339Nano, source.At)
			if e != nil {
				return e
			}
			matching := []changelog.Item{}
			for _, it := range v.Items {
				row := changelog.Item{Diffs: it.Diffs}
				for _, r := range []*org.DepartmentRecord{it.BeforeContext, it.AfterContext} {
					if r != nil && r.EffectiveDate != nil {
						row.Dates = append(row.Dates, *r.EffectiveDate)
					}
				}
				matching = append(matching, row)
			}
			if changelog.Matches(v.Action, v.Operator, matching, q.Filter, at.In(foundation.Shanghai).Format("2006-01-02")) {
				all = append(all, v)
			}
		}
		if projectedPage {
			out.Items = all
			return nil
		}
		out.Total = len(all)
		out.Page = min(q.Page, max(1, (len(all)+q.PageSize-1)/q.PageSize))
		start := min((out.Page-1)*q.PageSize, len(all))
		out.Items = all[start:min(start+q.PageSize, len(all))]
		return nil
	})
	return
}

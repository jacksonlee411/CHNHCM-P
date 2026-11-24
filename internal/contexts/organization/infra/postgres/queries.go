package postgres

import (
	"chnhcm.local/chnhcm/internal/contexts/organization/application/buauth"
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/application"
	"chnhcm.local/chnhcm/internal/contexts/organization/domain"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres/generated"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
)

func (u *unit) List(ctx context.Context, in org.ListInput, ent org.Enterprise, currentDate string) (org.List, error) {
	return u.list(ctx, in, ent, currentDate, false)
}
func (u *unit) SelectionList(ctx context.Context, in org.ListInput, ent org.Enterprise, currentDate string) (org.List, error) {
	return u.list(ctx, in, ent, currentDate, true)
}
func (u *unit) list(ctx context.Context, in org.ListInput, ent org.Enterprise, currentDate string, all bool) (out org.List, err error) {
	matches, err := u.buListMatches(ctx, in, ent, false)
	if err != nil {
		return out, err
	}
	slices.SortFunc(matches, func(a, b buListEntry) int { return compareBUEntry(a, b, in.Sort) })
	basis := make([][2]string, 0, len(matches))
	for _, v := range matches {
		basis = append(basis, [2]string{v.summary.ID, v.source})
	}
	slices.SortFunc(basis, func(a, b [2]string) int { return strings.Compare(a[0], b[0]) })
	raw, _ := json.Marshal(basis)
	out.QueryKey, out.SelectionVersion, err = selectionVersion(in, ent, u.guard, raw)
	if err != nil {
		return out, err
	}
	out.Count = len(matches)
	out.Page = min(in.Page, max(1, (out.Count+in.PageSize-1)/in.PageSize))
	out.Items = []org.Summary{}
	start, end := (out.Page-1)*in.PageSize, min(out.Page*in.PageSize, out.Count)
	if all {
		start, end = 0, out.Count
	}
	for _, v := range matches[start:end] {
		out.Items = append(out.Items, v.summary)
	}
	out.PageSize = in.PageSize
	out.BusinessAt = in.BusinessAt
	out.CurrentDate = currentDate
	return
}

type buListEntry struct {
	summary                org.Summary
	source, sortCode, name string
}

func (u *unit) buListMatches(ctx context.Context, in org.ListInput, ent org.Enterprise, navigation bool) ([]buListEntry, error) {
	var err error
	grants := []access.Grant{}
	for _, g := range u.guard.Grants() {
		if g.Action == "bu.read" {
			grants = append(grants, g)
		}
	}
	_, err = u.guard.Decide(ctx, access.Intent{Action: "bu.read", EnterpriseID: ent.ID, Kind: "list", Fields: []string{}})
	if err != nil {
		return nil, err
	}
	if len(grants) == 0 {
		return nil, application.RejectRead(foundation.Fail(403, "ACCESS_DENIED", "当前没有业务单元读取权限"))
	}
	wanted := map[string]string{}
	for _, g := range grants {
		if g.ScopeKind == "enterprise" && g.ScopeID == ent.ID {
			wanted = nil
			break
		}
		if g.ScopeKind == "bu" {
			wanted[g.ScopeID] = ""
		}
	}
	var facts map[string]d.BusinessUnit
	var e error
	if navigation {
		facts, e = u.buNavigationFacts(ctx, wanted, ent.ID, in.BusinessAt)
	} else {
		facts, e = u.buListFacts(ctx, wanted, in.BusinessAt)
	}
	if e != nil {
		return nil, e
	}
	matches := []buListEntry{}
	canSort := true
	for id, tl := range facts {
		fields := map[string]bool{}
		visible := false
		for _, g := range grants {
			if g.ScopeKind == "enterprise" && g.ScopeID == ent.ID || g.ScopeKind == "bu" && g.ScopeID == id {
				visible = true
				for _, f := range g.ReadFields {
					fields[f] = true
				}
			}
		}
		if !visible {
			continue
		}
		if !fields["name"] {
			canSort = false
		}
		fact := d.BUAt(tl.Records, in.BusinessAt)
		status := "not_effective"
		if fact == nil && len(tl.Records) > 0 {
			first := tl.Records[0]
			fact = &first
		} else if fact != nil {
			status = fact.State
		}
		if fact == nil {
			return nil, resultUnavailable()
		}
		code, name := strings.ToLower(tl.Code), strings.ToLower(fact.Name)
		q := strings.ToLower(strings.TrimSpace(in.Q))
		if q != "" && !(fields["code"] && strings.Contains(code, q) || fields["name"] && strings.Contains(name, q)) {
			continue
		}
		if in.Status != "" && in.Status != "all" && in.Status != status {
			continue
		}
		if f := in.Filter; f != nil {
			v := ""
			allowed := true
			switch f.Field {
			case "code":
				v = code
				allowed = fields["code"]
			case "name":
				v = name
				allowed = fields["name"]
			case "effectiveDate":
				v = fact.Date
				allowed = fields["effectiveDate"]
			case "status":
				v = status
			}
			ok := false
			switch f.Operator {
			case "equals", "is":
				ok = v == f.Value
			case "contains":
				ok = strings.Contains(v, f.Value)
			case "before":
				ok = v < f.Value
			case "after":
				ok = v > f.Value
			case "onOrBefore":
				ok = v <= f.Value
			case "onOrAfter":
				ok = v >= f.Value
			case "isAnyOf":
				ok = slices.Contains(strings.Split(f.Value, ","), v)
			}
			if f.Field == "status" {
				ok = slices.Contains(strings.Split(f.Value, ","), v)
			}
			if !allowed || !ok {
				continue
			}
		}
		item := org.Summary{ID: id, Status: status}
		sortCode := id
		if fields["code"] {
			item.Code = tl.Code
			sortCode = code
		}
		if fields["name"] {
			n := fact.Name
			item.Name = &n
		}
		if fields["effectiveDate"] {
			item.EffectiveDate = fact.Date
		}
		matches = append(matches, buListEntry{item, tl.Source, sortCode, fact.Name})
	}
	if in.Sort != "code" && in.Sort != "" && !canSort {
		return nil, application.RejectRead(foundation.Fail(403, "ACCESS_DENIED", "当前字段权限不允许按名称排序"))
	}
	return matches, nil
}

func (u *unit) Detail(ctx context.Context, in org.DetailInput, ent org.Enterprise, currentDate string) (out org.Detail, err error) {
	d, err := u.guard.Decide(ctx, access.Intent{Action: "bu.read", EnterpriseID: ent.ID, BUID: in.ID, Fields: []string{}, Kind: "detail"})
	if err != nil {
		return
	}
	if !d.Allowed {
		return out, application.RejectRead(foundation.Fail(404, "M02_OBJECT_NOT_FOUND", "业务单元不存在或当前不可见"))
	}
	history := in.SourceVersion != ""
	fields := d.ReadFields
	historyAllowed := false
	for _, g := range u.guard.Grants() {
		if g.Action == "bu.history.read" && ((g.ScopeKind == "enterprise" && g.ScopeID == ent.ID) || (g.ScopeKind == "bu" && g.ScopeID == in.ID)) {
			historyAllowed = true
		}
	}
	if history {
		h, e := u.guard.Decide(ctx, access.Intent{Action: "bu.history.read", EnterpriseID: ent.ID, BUID: in.ID, Fields: []string{}, Kind: "history"})
		if e != nil {
			return out, e
		}
		if !h.Allowed {
			return out, application.RejectRead(foundation.Fail(403, "ACCESS_DENIED", "当前权限不允许读取历史"))
		}
		fields = []string{}
		for _, f := range d.ReadFields {
			if slices.Contains(h.ReadFields, f) {
				fields = append(fields, f)
			}
		}
	}
	bu, err := u.q.GetUnit(ctx, generated.GetUnitParams{InstanceID: u.instance, BuID: in.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return out, application.RejectRead(foundation.Fail(404, "M02_OBJECT_NOT_FOUND", "业务单元不存在或当前不可见"))
	}
	if err != nil {
		return
	}
	out = org.Detail{NameGuidance: domain.NameGuidance, ID: bu.BuID, Code: bu.Code, Enterprise: ent, BusinessAt: in.BusinessAt, CurrentDate: currentDate, Timeline: []org.Record{}, ReadOnly: true, Capabilities: org.Capabilities{AddWriteFields: []string{}, CorrectWriteFields: []string{}, DeactivateWriteFields: []string{}, ReactivateWriteFields: []string{}, History: historyAllowed, ReadFields: fields, WriteFields: []string{}}}
	if !slices.Contains(fields, "code") {
		out.Code = ""
	}
	source := bu.HeadSourceID
	if in.SourceVersion != "" {
		source = in.SourceVersion
	}
	stored, err := u.readBU(ctx, in.ID, source)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, application.RejectRead(foundation.Fail(404, "M02_SOURCE_NOT_FOUND", "无法查看该历史资料"))
	}
	if err != nil {
		return
	}
	out.ResolvedSourceVersion = &source
	if !history {
		out.HeadSourceVersion = &source
	}
	recorded := stored.Operation.Source.RecordedAt.UTC().Format(time.RFC3339Nano)
	out.RecordedAt = &recorded
	out.Actor = &stored.Operation.Source.Subject
	evaluated := stored.Operation.Terminal.EvaluatedAt
	if evaluated == nil {
		return out, resultUnavailable()
	}
	evaluatedText := evaluated.UTC().Format(time.RFC3339Nano)
	day := evaluated.In(foundation.Shanghai).Format("2006-01-02")
	out.EvaluatedAt = &evaluatedText
	out.EvaluatedBusinessDate = &day
	rows := buMembers(stored.State.Timeline.Records)
	out.Result = "M02_NO_FACT"
	out.Status = "not_effective"
	for _, r := range rows {
		record := application.PublicRecord(domain.Record{ID: r.RecordID, Revision: r.RevisionID, Date: r.EffectiveDate, End: r.EndDate, State: r.State, Type: r.OperationType, Reason: r.OperationReason, TypeLabel: r.TypeLabel, ReasonLabel: r.ReasonLabel, Data: domain.BUData{Name: r.Name, NameEn: r.NameEn}}, fields)
		out.Timeline = append(out.Timeline, record)
		if r.EffectiveDate <= in.BusinessAt && (r.EndDate == "" || in.BusinessAt < r.EndDate) {
			copy := record
			out.ViewFact = &copy
			out.Result = "M02_FACT"
			out.Status = record.State
		}
		if record.ID == in.SelectedRecordID {
			copy := record
			out.SelectedRecord = &copy
		}
	}
	if history && in.SelectedRecordID != "" && out.SelectedRecord == nil {
		out.SelectedRecord, err = u.deletedRecord(ctx, in.ID, source, in.SelectedRecordID, fields)
		if err != nil {
			return out, err
		}
	}
	if in.SelectedRecordID != "" && out.SelectedRecord == nil {
		out.SelectionMissing = true
	} else if in.SelectedRecordID == "" {
		out.SelectedRecord = out.ViewFact
		if out.SelectedRecord == nil && len(out.Timeline) > 0 {
			r := out.Timeline[0]
			out.SelectedRecord = &r
		}
	}
	if out.SelectedRecord != nil {
		out.SelectedRecord.OperationSummary, err = u.operationSummary(ctx, in.ID, source, out.SelectedRecord)
		if err != nil {
			return out, err
		}
	}
	out.SelectedRecordIsFuture = out.SelectedRecord != nil && out.SelectedRecord.EffectiveDate > out.CurrentDate
	if !history {
		// Only identical capability queries share this request-local decision. No
		// write authorization or cross-request result is cached.
		decisions := map[string]access.Decision{}
		decide := func(action string, fields []string) (access.Decision, error) {
			key := action + "|" + strings.Join(fields, ",")
			if d, ok := decisions[key]; ok {
				return d, nil
			}
			d, e := u.guard.Decide(ctx, access.Intent{Action: action, EnterpriseID: ent.ID, BUID: in.ID, Kind: "capability", Fields: fields})
			if e == nil {
				decisions[key] = d
			}
			return d, e
		}
		base := make([]domain.Record, 0, len(rows))
		for _, r := range rows {
			base = append(base, domain.Record{ID: r.RecordID, Date: r.EffectiveDate, State: r.State, Data: domain.BUData{Name: r.Name, NameEn: r.NameEn}})
		}
		for _, a := range []string{"DEACTIVATE", "REACTIVATE"} {
			permission := map[string]string{"DEACTIVATE": "bu.deactivate", "REACTIVATE": "bu.reactivate"}[a]
			decision, e := decide(permission, []string{"effectiveDate", "operationType", "operationReason"})
			if e != nil {
				return out, e
			}
			available := false
			if decision.Allowed && out.SelectedRecord != nil && slices.ContainsFunc(base, func(r domain.Record) bool { return r.ID == out.SelectedRecord.ID }) {
				window, e := domain.PrepareLifecycleAppend(base, out.SelectedRecord.ID, a, ent.ValidFrom, out.CurrentDate)
				if e != nil {
					return out, e
				}
				available = window.Available
				if available {
					c := &org.LifecycleContext{AnchorRecordID: out.SelectedRecord.ID, OperationType: a, Defaults: &org.AdditionDefaults{}}
					if slices.Contains(fields, "effectiveDate") {
						c.MinEffectiveDate, c.MaxEffectiveDate, c.DefaultEffectiveDate = window.Min, window.Max, window.Default
					}
					if slices.Contains(fields, "name") {
						v := window.Anchor.Data.Name
						c.Defaults.Name = &v
					}
					if slices.Contains(fields, "nameEn") {
						c.Defaults.NameEn = window.Anchor.Data.NameEn
					}
					out.LifecycleContext = c
				}
			}
			if a == "DEACTIVATE" {
				out.Capabilities.Deactivate = available
				out.Capabilities.DeactivateWriteFields = decision.WriteFields
			} else {
				out.Capabilities.Reactivate = available
				out.Capabilities.ReactivateWriteFields = decision.WriteFields
			}
		}
		add, e := decide("bu.record.add", []string{"effectiveDate", "operationType", "operationReason"})
		if e != nil {
			return out, e
		}
		correct, e := decide("bu.record.correct", []string{})
		if e != nil {
			return out, e
		}
		out.Capabilities.Add = add.Allowed && (slices.Contains(add.WriteFields, "name") || slices.Contains(add.WriteFields, "nameEn"))
		selectedType := ""
		if out.SelectedRecord != nil {
			for _, r := range rows {
				if r.RecordID == out.SelectedRecord.ID {
					selectedType = r.OperationType
				}
			}
		}
		out.Capabilities.Correct = correct.Allowed && out.SelectedRecord != nil && selectedType != "DEACTIVATE" && (slices.Contains(correct.WriteFields, "name") || slices.Contains(correct.WriteFields, "nameEn"))
		if out.Capabilities.Add && slices.Contains(fields, "effectiveDate") && out.SelectedRecord != nil {
			base := make([]domain.Record, 0, len(rows))
			for _, r := range rows {
				base = append(base, domain.Record{ID: r.RecordID, Date: r.EffectiveDate, State: r.State, Data: domain.BUData{Name: r.Name, NameEn: r.NameEn}})
			}
			window, e := domain.PrepareAddition(base, out.SelectedRecord.ID, ent.ValidFrom, out.CurrentDate)
			if e != nil {
				return out, e
			}
			c := &org.AdditionContext{AnchorRecordID: out.SelectedRecord.ID, Mode: window.Mode, NextRecordID: window.Next, Available: window.Available}
			editable := false
			for _, field := range []string{"name", "nameEn"} {
				if slices.Contains(fields, field) && slices.Contains(add.WriteFields, field) {
					editable = true
				}
			}
			c.Available = c.Available && editable
			if c.Available {
				c.MinEffectiveDate, c.MaxEffectiveDate, c.DefaultEffectiveDate = window.Min, window.Max, window.Default
				c.Defaults = &org.AdditionDefaults{}
				if slices.Contains(fields, "name") {
					v := window.Anchor.Data.Name
					c.Defaults.Name = &v
				}
				if slices.Contains(fields, "nameEn") {
					c.Defaults.NameEn = window.Anchor.Data.NameEn
				}
			}
			out.AdditionContext = c
		}
		out.Capabilities.AddWriteFields = add.WriteFields
		out.Capabilities.CorrectWriteFields = correct.WriteFields
		if out.SelectedRecord != nil && out.SelectedRecord.ID != bu.FirstRecordID {
			if action := buauth.DeleteAction(selectedType); action != "" {
				deletion, e := decide(action, []string{})
				if e != nil {
					return out, e
				}
				out.Capabilities.Delete = deletion.Allowed
			}
		}
		if out.SelectedRecord != nil {
			action := buauth.RedateAction(selectedType)
			decision, e := decide(action, []string{"effectiveDate"})
			if e != nil {
				return out, e
			}
			out.Capabilities.Redate = decision.Allowed
		}
		if out.SelectedRecord != nil {
			out.Capabilities.Correction, err = correctionCapabilities(ctx, out.SelectedRecord.ID, bu.FirstRecordID, ent.ValidFrom, rows, fields, correct.WriteFields, decide)
			if err != nil {
				return out, err
			}
			out.Capabilities.Correct = len(out.Capabilities.Correction.Facets) > 0
		}
		err = u.pairContexts(ctx, &out, bu.FirstRecordID, rows, decide)
		if err != nil {
			return out, err
		}
		out.ReadOnly = !(out.PairInsertion != nil || out.PairDeletion != nil || out.Capabilities.Add || out.Capabilities.Correct || out.Capabilities.Deactivate || out.Capabilities.Reactivate || out.Capabilities.Delete || out.Capabilities.Redate)
	}

	return
}

// A deleted result is a sealed revision outside the live membership. It is only
// read in an explicitly historical source, after current object/history checks.
func (u *unit) deletedRecord(ctx context.Context, bu, source, id string, fields []string) (*org.Record, error) {
	v, e := u.readBU(ctx, bu, source)
	if e != nil {
		return nil, e
	}
	for _, r := range v.Operation.Revisions {
		if r.Deleted && r.Record.ID == id {
			out := application.PublicRecord(r.Record, fields)
			out.Deleted = true
			return &out, nil
		}
	}
	return nil, nil
}

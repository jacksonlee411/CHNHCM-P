package enterprise

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/application/submission"
	base "chnhcm.local/chnhcm/internal/contexts/organization/domain"
	"chnhcm.local/chnhcm/internal/contexts/organization/domain/dated"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/enterprise"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"errors"
	"slices"
)

func (s Service) read(ctx context.Context, c access.Credentials, fn func(Unit) error) error {
	return s.Store.WithinEnterprise(ctx, c, func(u Unit) error {
		e := fn(u)
		var f *foundation.Fault
		if errors.As(e, &f) && f.Status < 500 {
			return submission.RejectRead(e)
		}
		return e
	})
}
func emptyCaps() org.Capabilities {
	return org.Capabilities{ReadFields: []string{}, WriteFields: []string{}, AddWriteFields: []string{}, CorrectWriteFields: []string{}, DeactivateWriteFields: []string{}, ReactivateWriteFields: []string{}}
}
func (s Service) Context(ctx context.Context, c access.Credentials) (out org.EnterpriseContext, err error) {
	err = s.read(ctx, c, func(u Unit) error {
		fields, e := readable(ctx, u, false)
		if e != nil {
			return e
		}
		now, e := u.Now(ctx)
		if e != nil {
			return e
		}
		history, e := u.Access().Decide(ctx, intent("history.read", nil))
		if e != nil {
			return e
		}
		caps := emptyCaps()
		caps.ReadFields = fields
		caps.History = history.Allowed
		// Source initialization is intentionally not a public creation capability.
		id, e := u.Identity(ctx)
		if e != nil {
			return e
		}
		out = org.EnterpriseContext{ID: id, CurrentDate: now.In(foundation.Shanghai).Format("2006-01-02"), DefaultEffectiveDate: foundation.DefaultEffectiveDate, Timezone: "Asia/Shanghai", NameMaxLength: base.NameMaxLength, NameGuidance: base.NameGuidance, Capabilities: caps}
		return nil
	})
	return
}
func project(r d.Record, fields []string, summary *org.OperationSummary) org.EnterpriseRecord {
	out := org.EnterpriseRecord{ID: r.ID, Revision: r.Revision}
	has := func(f string) bool { return slices.Contains(fields, f) }
	if has("effectiveDate") {
		out.EffectiveDate = ptr(r.Date)
		end := ptr(r.End)
		out.EndExclusive = &end
		inclusive := ptr(dated.NextDate(r.End, -1))
		out.EndInclusive = &inclusive
	}
	if has("name") {
		out.Name = &r.Data.Name
	}
	if has("nameEn") {
		v := (d.Policy{}).Clone(r.Data).NameEn
		out.NameEn = &v
	}
	if has("state") {
		out.State = ptr(r.State)
	}
	if has("operationType") {
		out.OperationType = ptr(r.Type)
		out.TypeLabel = ptr(r.TypeLabel)
	}
	if has("operationReason") {
		out.OperationReason = ptr(r.Reason)
		out.ReasonLabel = ptr(r.ReasonLabel)
	}
	out.OperationSummary = summary
	return out
}
func status(t d.Timeline, today string) string {
	r := d.Kernel.FactAt(t.Records, today)
	if r == nil {
		return "not_effective"
	}
	return r.State
}
func (s Service) Detail(ctx context.Context, c access.Credentials, in org.DetailInput) (out org.EnterpriseDetail, err error) {
	if !foundation.ValidID(in.ID) || in.SourceVersion != "" && !foundation.ValidID(in.SourceVersion) || in.SelectedRecordID != "" && !foundation.ValidID(in.SelectedRecordID) || in.BusinessAt != "" {
		return out, fault("M02_INVALID_INPUT", "", "对象、记录或来源标识无效")
	}
	err = s.read(ctx, c, func(u Unit) error {
		fields, e := readable(ctx, u, in.SourceVersion != "")
		if e != nil {
			return e
		}
		now, e := u.Now(ctx)
		if e != nil {
			return e
		}
		today := now.In(foundation.Shanghai).Format("2006-01-02")
		current, e := u.Load(ctx, in.ID, "")
		if e != nil {
			return e
		}
		if current.Timeline.ID == "" {
			return missing()
		}
		state := current
		if in.SourceVersion != "" && in.SourceVersion != current.Timeline.Source {
			state, e = u.Load(ctx, in.ID, in.SourceVersion)
			if e != nil {
				return e
			}
			if state.Timeline.ID == "" {
				return foundation.Fail(404, "M02_SOURCE_NOT_FOUND", "企业来源不存在")
			}
		}
		t := state.Timeline
		history, e := u.Access().Decide(ctx, intent("history.read", nil))
		if e != nil {
			return e
		}
		caps := emptyCaps()
		caps.ReadFields = fields
		caps.History = history.Allowed
		out = org.EnterpriseDetail{ID: t.ID, CurrentDate: today, NameGuidance: base.NameGuidance, HeadSourceVersion: current.Timeline.Source, ResolvedSourceVersion: t.Source, ReadOnly: in.SourceVersion != "", Timeline: []org.EnterpriseRecord{}, Capabilities: caps}
		if slices.Contains(fields, "code") {
			out.Code = ptr(t.Code)
		}
		if slices.Contains(fields, "state") {
			out.Status = ptr(status(t, today))
		}
		selected := in.SelectedRecordID
		if selected == "" {
			r := d.Kernel.FactAt(t.Records, today)
			if r != nil {
				selected = r.ID
			} else {
				selected = t.First
			}
		}
		var original *d.Record
		currentID := t.First
		if fact := d.Kernel.FactAt(t.Records, today); fact != nil {
			currentID = fact.ID
		}
		historyFields := []string{}
		if history.Allowed {
			historyFields, e = readable(ctx, u, true)
			if e != nil {
				return e
			}
		}
		if selected != currentID && !history.Allowed {
			return denied()
		}
		if selected != currentID {
			out.Capabilities.ReadFields = historyFields
		}
		for _, r := range t.Records {
			rowFields := fields
			if r.ID != currentID {
				if !history.Allowed {
					continue
				}
				rowFields = historyFields
			}
			var summary *org.OperationSummary
			for _, v := range state.Summaries {
				if v.RecordID == r.ID && v.Revision == r.Revision {
					x := v
					summary = &x
					break
				}
			}
			v := project(r, rowFields, summary)
			out.Timeline = append(out.Timeline, v)
			if r.ID == selected {
				out.SelectedRecord = &v
				x := r
				original = &x
				out.SelectedRecordIsFuture = slices.Contains(rowFields, "effectiveDate") && r.Date > today
			}
		}
		if out.SelectedRecord == nil && in.SelectedRecordID != "" && history.Allowed {
			ops, e := u.History(ctx, in.ID, t.Source)
			if e != nil {
				return e
			}
			for _, op := range ops {
				for _, rev := range op.Revisions {
					if rev.After == nil && rev.Before != nil && rev.Before.ID == selected {
						previous, e := u.Load(ctx, in.ID, op.Previous)
						if e != nil {
							return e
						}
						var summary *org.OperationSummary
						for _, s := range previous.Summaries {
							if s.RecordID == selected && s.Revision == rev.Before.Revision {
								x := s
								summary = &x
								break
							}
						}
						v := project(*rev.Before, historyFields, summary)
						v.Deleted = true
						out.SelectedRecord = &v
						out.ReadOnly = true
						break
					}
				}
				if out.SelectedRecord != nil {
					break
				}
			}
		}
		out.SelectionMissing = out.SelectedRecord == nil
		if out.ReadOnly || original == nil || !history.Allowed {
			return nil
		}
		return populateCapabilities(ctx, u, &out, t, *original, now)
	})
	return
}
func (s Service) Preview(ctx context.Context, c access.Credentials, id string, in org.EnterprisePreviewInput) (out org.EnterprisePreview, err error) {
	cmd := org.EnterpriseCommand{Action: in.Action, EnterpriseID: id, RecordID: in.RecordID, SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Delete: org.DeleteInput{ExpectedSourceVersion: in.ExpectedSourceVersion, ExpectedRecordRevision: in.ExpectedRecordRevision}, Add: org.EnterpriseAddInput{ExpectedSourceVersion: in.ExpectedSourceVersion, AnchorRecordID: in.AnchorRecordID, EffectiveDate: in.EffectiveDate, OperationType: in.OperationType, OperationReason: in.OperationReason, Changes: in.Changes}, Correct: org.EnterpriseCorrectInput{ExpectedSourceVersion: in.ExpectedSourceVersion, ExpectedRecordRevision: in.ExpectedRecordRevision, Changes: in.Changes, NewEffectiveDate: in.NewEffectiveDate, Classification: in.Classification}}
	_, declared, e := metadata(cmd)
	if e != nil {
		return out, e
	}
	err = s.read(ctx, c, func(u Unit) error {
		fields, e := readable(ctx, u, true)
		if e != nil {
			return e
		}
		state, e := u.Load(ctx, id, "")
		if e != nil {
			return e
		}
		if state.Timeline.ID == "" {
			return missing()
		}
		if state.Timeline.Source != in.ExpectedSourceVersion {
			return fault("M02_SOURCE_CONFLICT", "", "来源已变化，请重新核对资料")
		}
		if e = authorizeCommand(ctx, u, state.Timeline, cmd, declared); e != nil {
			return e
		}
		candidate, e := candidate(ctx, state.Timeline, cmd)
		if e != nil {
			return e
		}
		after := state.Timeline
		after.Records = candidate.Records
		if e = u.ValidateReferences(ctx, state.Timeline, after); e != nil {
			return e
		}
		now, e := u.Now(ctx)
		if e != nil {
			return e
		}
		today := now.In(foundation.Shanghai).Format("2006-01-02")
		if slices.Contains(fields, "state") {
			out.PreviousCurrentStatus = status(state.Timeline, today)
			out.CurrentStatus = status(after, today)
		}
		out.ReadFields = fields
		if cmd.Action == "DELETE" && len(candidate.Deleted) > 0 {
			if r := d.Kernel.FactAt(candidate.Records, candidate.Deleted[0].Date); r != nil {
				v := project(*r, fields, nil)
				out.AfterRecord = &v
			}
		}
		if slices.Contains(fields, "effectiveDate") && after.Records[0].Date > state.Timeline.Records[0].Date {
			out.NewNoFactRange = &org.PreviewDateRange{Start: state.Timeline.Records[0].Date, EndExclusive: after.Records[0].Date, EndInclusive: dated.NextDate(after.Records[0].Date, -1)}
		}
		for _, r := range candidate.Records {
			if r.ID == in.RecordID || !foundation.ValidID(r.ID) {
				v := project(r, fields, nil)
				out.AfterRecord = &v
			}
		}
		return nil
	})
	return
}

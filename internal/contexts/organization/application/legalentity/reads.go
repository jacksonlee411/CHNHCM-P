package legalentity

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/application/enterprisecheck"
	"chnhcm.local/chnhcm/internal/contexts/organization/application/submission"
	base "chnhcm.local/chnhcm/internal/contexts/organization/domain"
	"chnhcm.local/chnhcm/internal/contexts/organization/domain/dated"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/legalentity"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"errors"
	"slices"
)

func (s Service) read(ctx context.Context, c access.Credentials, fn func(Unit) error) error {
	return s.Store.WithinLegalEntity(ctx, c, func(u Unit) error {
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
func (s Service) Context(ctx context.Context, c access.Credentials) (out org.LegalEntityContext, err error) {
	err = s.read(ctx, c, func(u Unit) error {
		fields, e := readable(ctx, u, false)
		if e != nil {
			return e
		}
		now, e := u.Now(ctx)
		if e != nil {
			return e
		}
		history, e := u.Access().Decide(ctx, intent(u, "history.read", nil))
		if e != nil {
			return e
		}
		caps := emptyCaps()
		caps.ReadFields = fields
		caps.History = history.Allowed
		create, e := u.Access().Decide(ctx, intent(u, "create", nil))
		if e != nil {
			return e
		}
		caps.Create = create.Allowed
		caps.WriteFields = create.WriteFields
		out = org.LegalEntityContext{CurrentDate: now.In(foundation.Shanghai).Format("2006-01-02"), DefaultEffectiveDate: foundation.DefaultEffectiveDate, Timezone: "Asia/Shanghai", NameMaxLength: base.NameMaxLength, NameGuidance: base.NameGuidance, Capabilities: caps, Enterprise: enterprisecheck.Display(u.EnterpriseScope(), now)}
		return nil
	})
	return
}
func (s Service) List(ctx context.Context, c access.Credentials, in org.LegalEntityQuery) (out org.LegalEntityList, err error) {
	if err = normalizeList(&in); err != nil {
		return
	}
	err = s.read(ctx, c, func(u Unit) error {
		fields, e := readable(ctx, u, false)
		if e != nil {
			return e
		}
		now, e := u.Now(ctx)
		if e != nil {
			return e
		}
		if e = authorizeQuery(ctx, u, in, fields); e != nil {
			return e
		}
		out, e = u.List(ctx, in, fields, now.In(foundation.Shanghai).Format("2006-01-02"))
		if e == nil {
			out.QueryKey = queryKey(u, in)
			out.BusinessAt = out.CurrentDate
		}
		return e
	})
	return
}
func project(r d.Record, fields []string, summary *org.OperationSummary) org.LegalEntityRecord {
	out := org.LegalEntityRecord{ID: r.ID, Revision: r.Revision}
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
	if has("establishingOrganization") {
		v := (d.Policy{}).Clone(r.Data).EstablishingOrganization
		out.EstablishingOrganization = &v
	}
	if has("registrationNumber") {
		v := (d.Policy{}).Clone(r.Data).RegistrationNumber
		out.RegistrationNumber = &v
	}
	if has("status") {
		out.State = ptr(r.State)
	}
	if has("countryRegionId") {
		out.CountryRegionID = ptr(r.Data.CountryID)
	}
	if has("subjectType") {
		out.SubjectType = ptr(r.Data.SubjectType)
	}
	if has("registrationType") {
		v := (d.Policy{}).Clone(r.Data).RegistrationType
		out.RegistrationType = &v
	}
	if has("effectiveDate") {
		out.AdoptionDate = ptr(r.Data.Adoption.Date)
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
func (s Service) Detail(ctx context.Context, c access.Credentials, in org.DetailInput) (out org.LegalEntityDetail, err error) {
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
				return foundation.Fail(404, "M02_SOURCE_NOT_FOUND", "法律实体来源不存在")
			}
		}
		t := state.Timeline
		history, e := u.Access().Decide(ctx, intent(u, "history.read", nil))
		if e != nil {
			return e
		}
		caps := emptyCaps()
		caps.ReadFields = fields
		caps.History = history.Allowed
		out = org.LegalEntityDetail{ID: t.ID, CurrentDate: today, NameGuidance: base.NameGuidance, HeadSourceVersion: current.Timeline.Source, ResolvedSourceVersion: t.Source, ReadOnly: in.SourceVersion != "", Timeline: []org.LegalEntityRecord{}, Capabilities: caps}
		if in.SourceVersion == "" && slices.Contains(fields, "name") {
			out.CanReadPurposes, e = access.RuleAllowed(ctx, u.Access(), u.EnterpriseScope().ID, "legalpurpose.read")
			if e != nil {
				return e
			}
		}

		if slices.Contains(fields, "code") {
			out.Code = ptr(t.Code)
		}
		if slices.Contains(fields, "status") {
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
		for _, r := range t.Records {
			rowFields := fields
			if r.ID != currentID {
				if !history.Allowed {
					continue
				}
				rowFields = historyFields
			}
			var summary *org.OperationSummary
			if history.Allowed || r.ID == currentID {
				for _, v := range state.Summaries {
					if v.RecordID == r.ID && v.Revision == r.Revision {
						x := v
						summary = &x
						break
					}
				}
			}
			v := project(r, rowFields, summary)
			if e = projectCountry(ctx, u, r, rowFields, today, &v); e != nil {
				return e
			}
			out.Timeline = append(out.Timeline, v)
			if r.ID == selected {
				out.SelectedRecord = &v
				x := r
				original = &x
				out.SelectedRecordIsFuture = slices.Contains(rowFields, "effectiveDate") && r.Date > today
			}
		}
		if original == nil && in.SourceVersion != "" {
			ops, e := u.History(ctx, t.ID, t.Source)
			if e != nil {
				return e
			}
			if len(ops) > 0 {
				for _, rev := range ops[0].Revisions {
					if rev.Before != nil && rev.Before.ID == selected && rev.After == nil {
						v := project(*rev.Before, fields, nil)
						v.Deleted = true
						if e = projectCountry(ctx, u, *rev.Before, fields, today, &v); e != nil {
							return e
						}
						out.SelectedRecord = &v
						out.ReadOnly = true
						return nil
					}
				}
			}
		}
		if selected != currentID {
			out.Capabilities.ReadFields = historyFields
		}
		out.SelectionMissing = out.SelectedRecord == nil
		if out.ReadOnly || original == nil || !history.Allowed {
			return nil
		}
		return populateCapabilities(ctx, u, &out, t, *original, today)
	})
	return
}
func (s Service) Preview(ctx context.Context, c access.Credentials, id string, in org.LegalEntityPreviewInput) (out org.LegalEntityPreview, err error) {
	cmd := org.LegalEntityCommand{Action: in.Action, LegalEntityID: id, RecordID: in.RecordID, SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Add: org.LegalEntityAddInput{ExpectedSourceVersion: in.ExpectedSourceVersion, AnchorRecordID: in.AnchorRecordID, EffectiveDate: in.EffectiveDate, OperationType: in.OperationType, OperationReason: in.OperationReason, Changes: in.Changes}, Correct: org.LegalEntityCorrectInput{ExpectedSourceVersion: in.ExpectedSourceVersion, ExpectedRecordRevision: in.ExpectedRecordRevision, Changes: in.Changes, NewEffectiveDate: in.NewEffectiveDate, Classification: in.Classification}, Delete: org.DeleteInput{ExpectedSourceVersion: in.ExpectedSourceVersion, ExpectedRecordRevision: in.ExpectedRecordRevision}, Redate: org.RedateInput{ExpectedSourceVersion: in.ExpectedSourceVersion, ExpectedRecordRevision: in.ExpectedRecordRevision, NewEffectiveDate: in.NewEffectiveDate}}
	_, declared, e := metadata(cmd)
	if e != nil {
		return out, e
	}
	err = s.read(ctx, c, func(u Unit) error {
		fields, e := readable(ctx, u, true)
		if e != nil {
			return e
		}
		if e = requireCommand(ctx, u, cmd, declared); e != nil {
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
		candidate, e := candidate(ctx, u, state.Timeline, cmd)
		if e != nil {
			return e
		}
		out.ReadFields = fields
		if slices.Contains(fields, "effectiveDate") {
			out.Impact = impact(candidate.Impact)
		}
		for _, r := range candidate.Records {
			if r.ID == in.RecordID || !slices.ContainsFunc(state.Timeline.Records, func(old d.Record) bool { return old.ID == r.ID }) {
				v := project(r, fields, nil)
				now, e := u.Now(ctx)
				if e != nil {
					return e
				}
				if e = projectCountry(ctx, u, r, fields, now.In(foundation.Shanghai).Format("2006-01-02"), &v); e != nil {
					return e
				}
				out.AfterRecord = &v
			}
		}
		return nil
	})
	return
}

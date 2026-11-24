package postgres

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/application"
	"chnhcm.local/chnhcm/internal/contexts/organization/application/buauth"
	"chnhcm.local/chnhcm/internal/contexts/organization/domain"

	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"context"
	"reflect"
	"slices"
)

func (u *unit) pairContexts(ctx context.Context, out *org.Detail, first string, rows []buMemberRow, decide func(string, []string) (access.Decision, error)) error {
	if out.SelectedRecord == nil {
		return nil
	}
	base := make([]domain.Record, 0, len(rows))
	for _, r := range rows {
		base = append(base, domain.Record{ID: r.RecordID, Revision: r.RevisionID, Date: r.EffectiveDate, End: r.EndDate, Type: r.OperationType, Reason: r.OperationReason, TypeLabel: r.TypeLabel, ReasonLabel: r.ReasonLabel, State: r.State, Data: domain.BUData{Name: r.Name, NameEn: r.NameEn}})
	}
	fields := out.Capabilities.ReadFields
	// Pair permissions are independent of single-append availability.
	stopPermission, e := decide("bu.deactivate", []string{"effectiveDate", "operationType", "operationReason"})
	if e != nil {
		return e
	}
	resumePermission, e := decide("bu.reactivate", []string{"effectiveDate", "operationType", "operationReason"})
	if e != nil {
		return e
	}
	if stopPermission.Allowed && resumePermission.Allowed && slices.Contains(fields, "effectiveDate") {
		w, e := domain.PrepareLifecyclePair(base, out.SelectedRecord.ID, out.Enterprise.ValidFrom)
		if e != nil {
			return e
		}
		c := &org.PairInsertionContext{AnchorRecordID: out.SelectedRecord.ID, Available: w.Available}
		if w.Available {
			c.MinDate = w.Min
			c.MaxDate = w.Max
			c.MaxStopDate = domain.NextDate(w.Max, -1)
			c.DefaultStopDate = w.Min
			c.DefaultResumeDate = domain.NextDate(w.Min, 1)
		}
		out.PairInsertion = c
	}
	stop, resume, e := domain.LifecyclePair(base, out.SelectedRecord.ID)
	if e != nil {
		return nil
	}
	for _, kind := range []string{"DEACTIVATE", "REACTIVATE"} {
		d, e := decide(buauth.DeleteAction(kind), []string{})
		if e != nil {
			return e
		}
		if !d.Allowed {
			return nil
		}
	}
	items := []domain.Item{{ID: stop.ID, Action: "DELETE", RecordID: stop.ID, ExpectedRevision: stop.Revision}, {ID: resume.ID, Action: "DELETE", RecordID: resume.ID, ExpectedRevision: resume.Revision}}
	candidate, e := domain.Calculate(ctx, base, first, out.Enterprise.ValidFrom, items)
	if e != nil {
		return e
	}
	changes, e := domain.PairNameChanges(ctx, base, candidate.Records)
	if e != nil {
		return e
	}
	result := &org.PairDeletionContext{Deactivation: application.PublicRecord(stop, fields), Reactivation: application.PublicRecord(resume, fields), NameChanges: []org.PairNameChange{}}
	for _, v := range changes {
		b, a := application.PublicRecord(v.Before, fields), application.PublicRecord(v.After, fields)
		if reflect.DeepEqual(b.Name, a.Name) && reflect.DeepEqual(b.NameEn, a.NameEn) {
			continue
		}
		p := org.PairNameChange{BeforeName: b.Name, BeforeNameEn: b.NameEn, AfterName: a.Name, AfterNameEn: a.NameEn}
		if slices.Contains(fields, "effectiveDate") {
			p.Start = v.Start
			if v.End != "" {
				end := v.End
				p.End = &end
				last := domain.NextDate(end, -1)
				p.EndInclusive = &last
			}
		}
		result.NameChanges = append(result.NameChanges, p)
	}
	out.PairDeletion = result
	return nil
}

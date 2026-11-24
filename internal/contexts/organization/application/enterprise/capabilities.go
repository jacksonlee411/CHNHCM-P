package enterprise

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/domain/dated"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/enterprise"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"reflect"
	"slices"
	"strings"
	"time"
)

func populateCapabilities(ctx context.Context, u Unit, out *org.EnterpriseDetail, t d.Timeline, r d.Record, now time.Time) error {

	decide := func(action string, fields []string) (access.Decision, error) {
		if action == "" {
			return access.Decision{}, nil
		}
		return u.Access().Decide(ctx, intent(strings.TrimPrefix(action, "enterprise."), fields))
	}
	read := out.Capabilities.ReadFields
	dataFields := []string{"name", "nameEn"}
	correct, e := decide("enterprise.record.correct", nil)
	if e != nil {
		return e
	}
	out.Capabilities.CorrectWriteFields = intersection(intersection(correct.WriteFields, read), dataFields)
	facets := []string{}
	if len(out.Capabilities.CorrectWriteFields) > 0 && r.Type != "DEACTIVATE" {
		facets = append(facets, "CORRECT")
	}
	date, e := decide(categoryAction(r.Type, "redate"), []string{"effectiveDate"})
	if e != nil {
		return e
	}
	correction := &org.CorrectionCapabilities{Facets: facets, ClassificationTypes: []string{}}
	if date.Allowed && slices.Contains(read, "effectiveDate") {
		correction.Facets = append(correction.Facets, "REDATE")
		correction.MinEffectiveDate, correction.MaxEffectiveDate, e = d.Kernel.CorrectionBounds(t.Records, r.ID, d.MinDate)
		if e != nil {
			return e
		}
		out.Capabilities.Redate = true
	}
	if slices.Contains(read, "operationType") && slices.Contains(read, "operationReason") {
		original, e := decide(categoryAction(r.Type, "reclassify"), []string{"operationType", "operationReason"})
		if e != nil {
			return e
		}
		if original.Allowed {
			types := []string{"CHANGE", "DEACTIVATE", "REACTIVATE"}
			if r.ID == t.First {
				types = []string{"CREATE"}
			}
			for _, typ := range types {
				v, e := decide(categoryAction(typ, "reclassify"), []string{"operationType", "operationReason"})
				if e != nil {
					return e
				}
				if v.Allowed {
					correction.ClassificationTypes = append(correction.ClassificationTypes, typ)
				}
			}
			if len(correction.ClassificationTypes) > 0 {
				correction.Facets = append(correction.Facets, "RECLASSIFY")
			}
		}
	}
	out.Capabilities.Correction = correction
	out.Capabilities.Correct = len(correction.Facets) > 0
	deletion, e := decide(categoryAction(r.Type, "delete"), nil)
	if e != nil {
		return e
	}
	out.Capabilities.Delete = deletion.Allowed && r.ID != t.First
	history, e := decide("enterprise.history.read", nil)
	if e != nil {
		return e
	}
	out.Capabilities.History = history.Allowed
	today := now.In(foundation.Shanghai).Format("2006-01-02")
	required := []string{"effectiveDate", "operationType", "operationReason"}
	add, e := decide("enterprise.record.add", required)
	if e != nil {
		return e
	}
	out.Capabilities.AddWriteFields = intersection(add.WriteFields, read)
	window, e := d.Kernel.PrepareAddition(t.Records, r.ID, d.MinDate, today)
	if e != nil {
		return e
	}
	if add.Allowed && window.Available && len(intersection(read, required)) == len(required) {
		defaults := project(window.Anchor, read, nil)
		out.Capabilities.Add = true
		out.AdditionContext = &org.EnterpriseAdditionContext{AnchorRecordID: r.ID, Mode: window.Mode, Available: true, MinEffectiveDate: window.Min, MaxEffectiveDate: ptr(window.Max), DefaultEffectiveDate: window.Default, Defaults: defaults}
	}
	for _, kind := range []string{"DEACTIVATE", "REACTIVATE"} {
		action := "enterprise.deactivate"
		if kind == "REACTIVATE" {
			action = "enterprise.reactivate"
		}
		permission, e := decide(action, required)
		if e != nil {
			return e
		}
		fields := intersection(permission.WriteFields, read)
		if kind == "DEACTIVATE" {
			out.Capabilities.DeactivateWriteFields = fields
		} else {
			out.Capabilities.ReactivateWriteFields = fields
		}
		w, e := d.Kernel.PrepareLifecycleAppend(t.Records, r.ID, kind, d.MinDate, today)
		if e != nil {
			return e
		}
		if permission.Allowed && w.Available && len(intersection(read, required)) == len(required) {
			defaults := project(w.Anchor, read, nil)
			out.LifecycleContext = &org.EnterpriseLifecycleContext{AnchorRecordID: r.ID, OperationType: kind, MinEffectiveDate: w.Min, MaxEffectiveDate: ptr(w.Max), DefaultEffectiveDate: w.Default, Defaults: defaults}
			if kind == "DEACTIVATE" {
				out.Capabilities.Deactivate = true
			} else {
				out.Capabilities.Reactivate = true
			}
		}
	}
	stop, e := decide("enterprise.deactivate", required)
	if e != nil {
		return e
	}
	resume, e := decide("enterprise.reactivate", required)
	if e != nil {
		return e
	}
	w, e := d.Kernel.PrepareLifecyclePair(t.Records, r.ID, d.MinDate)
	if e != nil {
		return e
	}
	if stop.Allowed && resume.Allowed && w.Available && len(intersection(read, required)) == len(required) {
		out.PairInsertion = &org.PairInsertionContext{AnchorRecordID: r.ID, Available: true, MinDate: w.Min, MaxStopDate: dated.NextDate(w.Max, -1), MaxDate: w.Max, DefaultStopDate: w.Min, DefaultResumeDate: dated.NextDate(w.Min, 1)}
	}
	stopDelete, e := decide("enterprise.deactivation.delete", nil)
	if e != nil {
		return e
	}
	resumeDelete, e := decide("enterprise.reactivation.delete", nil)
	if e != nil {
		return e
	}
	if stopDelete.Allowed && resumeDelete.Allowed {
		a, b, e := d.Kernel.LifecyclePair(t.Records, r.ID)
		if e == nil {
			candidate, e := d.Kernel.Calculate(ctx, t.Records, t.First, d.MinDate, []d.Item{{ID: "deactivation", Action: "DELETE", RecordID: a.ID, ExpectedRevision: a.Revision}, {ID: "reactivation", Action: "DELETE", RecordID: b.ID, ExpectedRevision: b.Revision}})
			if e == nil {
				before := project(a, read, nil)
				after := project(b, read, nil)
				out.PairDeletion = &org.EnterprisePairDeletionContext{Deactivation: before, Reactivation: after, Changes: []org.EnterpriseDataChange{}}
				changes, e := d.Kernel.DataChanges(ctx, t.Records, candidate.Records)
				if e != nil {
					return e
				}
				for _, ch := range changes {
					b := project(ch.Before, read, nil)
					a := project(ch.After, read, nil)
					// Only differences in readable business fields belong in the summary.
					if reflect.DeepEqual(b.Name, a.Name) && reflect.DeepEqual(b.NameEn, a.NameEn) {
						continue
					}
					change := org.EnterpriseDataChange{Before: b, After: a}
					if slices.Contains(read, "effectiveDate") {
						change.Start = ptr(ch.Start)
						end := ptr(ch.End)
						change.EndExclusive = &end
						inclusive := ptr(dated.NextDate(ch.End, -1))
						change.EndInclusive = &inclusive
					}
					out.PairDeletion.Changes = append(out.PairDeletion.Changes, change)
				}
			}
		}
	}
	return nil
}

func intersection(a, b []string) []string {
	out := []string{}
	for _, v := range a {
		if slices.Contains(b, v) {
			out = append(out, v)
		}
	}
	return out
}

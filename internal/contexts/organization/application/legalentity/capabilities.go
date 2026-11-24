package legalentity

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/domain/dated"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/legalentity"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"context"
	"reflect"
	"slices"
)

func intersection(a, b []string) []string {
	out := []string{}
	for _, v := range a {
		if slices.Contains(b, v) {
			out = append(out, v)
		}
	}
	return out
}
func populateCapabilities(ctx context.Context, u Unit, out *org.LegalEntityDetail, t d.Timeline, r d.Record, today string) error {
	decide := func(action string, fields []string) (access.Decision, error) {
		return u.Access().Decide(ctx, intent(u, action, fields))
	}
	read := out.Capabilities.ReadFields
	minimum := u.EnterpriseScope().ValidFrom
	correct, e := decide("record.correct", nil)
	if e != nil {
		return e
	}
	out.Capabilities.CorrectWriteFields = intersection(intersection(correct.WriteFields, read), dataFields)
	correction := &org.CorrectionCapabilities{Facets: []string{}, ClassificationTypes: []string{}}
	if correct.Allowed && len(out.Capabilities.CorrectWriteFields) > 0 && r.Type != "DEACTIVATE" {
		correction.Facets = append(correction.Facets, "CORRECT")
	}
	redate, e := decide("record.redate", []string{"effectiveDate"})
	if e != nil {
		return e
	}
	if redate.Allowed && slices.Contains(read, "effectiveDate") {
		correction.Facets = append(correction.Facets, "REDATE")
		out.Capabilities.Redate = true
		correction.MinEffectiveDate, correction.MaxEffectiveDate, e = d.Kernel.CorrectionBounds(t.Records, r.ID, minimum)
		if e != nil {
			return e
		}
	}
	classification, e := decide("record.correct", []string{"operationType", "operationReason"})
	if e != nil {
		return e
	}
	if classification.Allowed && slices.Contains(read, "operationType") && slices.Contains(read, "operationReason") {
		correction.Facets = append(correction.Facets, "RECLASSIFY")
		correction.ClassificationTypes = []string{"CHANGE", "DEACTIVATE", "REACTIVATE"}
		if r.ID == t.First {
			correction.ClassificationTypes = []string{"CREATE"}
		}
	}
	out.Capabilities.Correction = correction
	out.Capabilities.Correct = len(correction.Facets) > 0
	deletion, e := decide("record.delete", nil)
	if e != nil {
		return e
	}
	out.Capabilities.Delete = deletion.Allowed && r.ID != t.First
	add, e := decide("record.add", dateFields)
	if e != nil {
		return e
	}
	out.Capabilities.AddWriteFields = intersection(add.WriteFields, read)
	w, e := d.Kernel.PrepareAddition(t.Records, r.ID, minimum, today)
	if e != nil {
		return e
	}
	projectDefault := func(r d.Record) (org.LegalEntityRecord, error) {
		v := project(r, read, nil)
		e := projectCountry(ctx, u, r, read, today, &v)
		return v, e
	}
	if add.Allowed && w.Available && len(intersection(read, dateFields)) == len(dateFields) {
		defaults, e := projectDefault(w.Anchor)
		if e != nil {
			return e
		}
		out.Capabilities.Add = true
		out.AdditionContext = &org.LegalEntityAdditionContext{Available: true, AnchorRecordID: r.ID, Mode: w.Mode, MinEffectiveDate: w.Min, MaxEffectiveDate: ptr(w.Max), DefaultEffectiveDate: w.Default, Defaults: defaults}
	}
	for _, kind := range []string{"DEACTIVATE", "REACTIVATE"} {
		action := "deactivate"
		if kind == "REACTIVATE" {
			action = "reactivate"
		}
		allowed, e := decide(action, dateFields)
		if e != nil {
			return e
		}
		fields := intersection(allowed.WriteFields, read)
		if kind == "DEACTIVATE" {
			out.Capabilities.DeactivateWriteFields = fields
		} else {
			out.Capabilities.ReactivateWriteFields = fields
		}
		life, e := d.Kernel.PrepareLifecycleAppend(t.Records, r.ID, kind, minimum, today)
		if e != nil {
			return e
		}
		if allowed.Allowed && life.Available && len(intersection(read, dateFields)) == len(dateFields) {
			defaults, e := projectDefault(life.Anchor)
			if e != nil {
				return e
			}
			out.LifecycleContext = &org.LegalEntityLifecycleContext{AnchorRecordID: r.ID, OperationType: kind, MinEffectiveDate: life.Min, MaxEffectiveDate: ptr(life.Max), DefaultEffectiveDate: life.Default, Defaults: defaults}
			if kind == "DEACTIVATE" {
				out.Capabilities.Deactivate = true
			} else {
				out.Capabilities.Reactivate = true
			}
		}
	}
	pair, e := decide("lifecycle.pair", dateFields)
	if e != nil {
		return e
	}
	if pair.Allowed && len(intersection(read, dateFields)) == len(dateFields) {
		window, e := d.Kernel.PrepareLifecyclePair(t.Records, r.ID, minimum)
		if e != nil {
			return e
		}
		if window.Available {
			out.PairInsertion = &org.PairInsertionContext{Available: true, AnchorRecordID: r.ID, MinDate: window.Min, MaxDate: window.Max, MaxStopDate: dated.NextDate(window.Max, -1), DefaultStopDate: window.Min, DefaultResumeDate: dated.NextDate(window.Min, 1)}
		}
		stop, resume, e := d.Kernel.LifecyclePair(t.Records, r.ID)
		if e == nil {
			c, e := d.Kernel.Calculate(ctx, t.Records, t.First, minimum, []d.Item{{ID: "stop", Action: "DELETE", RecordID: stop.ID, ExpectedRevision: stop.Revision}, {ID: "resume", Action: "DELETE", RecordID: resume.ID, ExpectedRevision: resume.Revision}})
			if e == nil {
				a, e := projectDefault(stop)
				if e != nil {
					return e
				}
				b, e := projectDefault(resume)
				if e != nil {
					return e
				}
				out.PairDeletion = &org.LegalEntityPairContext{Deactivation: a, Reactivation: b, Changes: []org.LegalEntityDataChange{}}
				changes, e := d.Kernel.DataChangesWithAdoptions(ctx, t.Records, c.Records)
				if e != nil {
					return e
				}
				for _, change := range changes {
					before, e := projectDefault(change.Before)
					if e != nil {
						return e
					}
					after, e := projectDefault(change.After)
					if e != nil {
						return e
					}
					if !slices.ContainsFunc(dataFields, func(field string) bool {
						return !reflect.DeepEqual(changeValue(&before, field), changeValue(&after, field))
					}) {
						continue
					}
					end := ptr(change.End)
					inclusive := ptr(dated.NextDate(change.End, -1))
					out.PairDeletion.Changes = append(out.PairDeletion.Changes, org.LegalEntityDataChange{Start: ptr(change.Start), EndExclusive: &end, EndInclusive: &inclusive, Before: before, After: after})
				}
			}
		}
	}
	return nil
}

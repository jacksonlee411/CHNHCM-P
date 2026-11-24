//go:build integration

// Fixed pre-00016 encoders used solely to seed prior-format conversion tests.
package bootstrap

import (
	p2 "chnhcm.local/chnhcm/internal/contexts/organization/infra/history/model"
	compact "chnhcm.local/chnhcm/internal/platform/compact"
	context "context"
	time "time"
)

var priorExtras = map[string]string{}

func priorExtra(line string) *string {
	if v, ok := priorExtras[line]; ok {
		return &v
	}
	return nil
}

func encodePriorLocationState(ctx context.Context, p *compact.Pool, v p2.LocationState) ([]byte, error) {
	w := compact.NewWriter(ctx, p, 16)
	w.String(string(v.State.Timeline.ID))
	w.String(string(v.State.Timeline.Enterprise))
	w.String(string(v.State.Timeline.Code))
	w.String(string(v.State.Timeline.First))
	w.String(string(v.State.Timeline.Source))
	w.Int(int64(v.State.Timeline.SourceNo))
	w.Bool(v.State.Timeline.Records != nil)
	w.Uint(uint64(len(v.State.Timeline.Records)))
	for n269 := range v.State.Timeline.Records {
		w.String(string(v.State.Timeline.Records[n269].ID))
		w.String(string(v.State.Timeline.Records[n269].Revision))
		w.Int(int64(v.State.Timeline.Records[n269].RevisionNo))
		w.Range(v.State.Timeline.Records[n269].Date, v.State.Timeline.Records[n269].End)
		w.String(string(v.State.Timeline.Records[n269].Data.Name))
		w.Bool(v.State.Timeline.Records[n269].Data.NameEn != nil)
		if v.State.Timeline.Records[n269].Data.NameEn != nil {
			w.String(string((*v.State.Timeline.Records[n269].Data.NameEn)))
		}
		w.String(string(v.State.Timeline.Records[n269].Data.CountryID))
		w.Bool(v.State.Timeline.Records[n269].Data.AreaID != nil)
		if v.State.Timeline.Records[n269].Data.AreaID != nil {
			w.String(string((*v.State.Timeline.Records[n269].Data.AreaID)))
		}
		w.String(string(v.State.Timeline.Records[n269].Data.AddressLine))
		w.Bool(priorExtra(v.State.Timeline.Records[n269].Data.AddressLine) != nil)
		if priorExtra(v.State.Timeline.Records[n269].Data.AddressLine) != nil {
			w.String(*priorExtra(v.State.Timeline.Records[n269].Data.AddressLine))
		}
		w.Bool(v.State.Timeline.Records[n269].Data.PostalCode != nil)
		if v.State.Timeline.Records[n269].Data.PostalCode != nil {
			w.String(string((*v.State.Timeline.Records[n269].Data.PostalCode)))
		}
		w.String(string(v.State.Timeline.Records[n269].Data.Adoption.Date))
		w.String(string(v.State.Timeline.Records[n269].Data.Adoption.Country.ObjectID))
		w.String(string(v.State.Timeline.Records[n269].Data.Adoption.Country.SourceVersion))
		w.String(string(v.State.Timeline.Records[n269].Data.Adoption.Country.RecordID))
		w.String(string(v.State.Timeline.Records[n269].Data.Adoption.Country.Revision))
		w.Bool(v.State.Timeline.Records[n269].Data.Adoption.Parents != nil)
		w.Uint(uint64(len(v.State.Timeline.Records[n269].Data.Adoption.Parents)))
		for n270 := range v.State.Timeline.Records[n269].Data.Adoption.Parents {
			w.String(string(v.State.Timeline.Records[n269].Data.Adoption.Parents[n270].ObjectID))
			w.String(string(v.State.Timeline.Records[n269].Data.Adoption.Parents[n270].SourceVersion))
			w.String(string(v.State.Timeline.Records[n269].Data.Adoption.Parents[n270].RecordID))
			w.String(string(v.State.Timeline.Records[n269].Data.Adoption.Parents[n270].Revision))
		}
		w.String(string(v.State.Timeline.Records[n269].Type))
		w.String(string(v.State.Timeline.Records[n269].Reason))
		w.String(string(v.State.Timeline.Records[n269].TypeLabel))
		w.String(string(v.State.Timeline.Records[n269].ReasonLabel))
		w.String(string(v.State.Timeline.Records[n269].State))
		w.String(string(v.State.Timeline.Records[n269].InheritedRecord))
		w.String(string(v.State.Timeline.Records[n269].InheritedRevision))
	}
	w.Bool(v.State.Summaries != nil)
	w.Uint(uint64(len(v.State.Summaries)))
	for n271 := range v.State.Summaries {
		w.String(string(v.State.Summaries[n271].RecordID))
		w.String(string(v.State.Summaries[n271].Revision))
		w.String(string(v.State.Summaries[n271].OperationSourceVersion))
		w.String(string(v.State.Summaries[n271].RecordedAt))
		w.String(string(v.State.Summaries[n271].Operator.SubjectID))
		w.Bool(v.State.Summaries[n271].Operator.DisplayName != nil)
		if v.State.Summaries[n271].Operator.DisplayName != nil {
			w.String(string((*v.State.Summaries[n271].Operator.DisplayName)))
		}
		w.Bool(v.State.Summaries[n271].Operator.Login != nil)
		if v.State.Summaries[n271].Operator.Login != nil {
			w.String(string((*v.State.Summaries[n271].Operator.Login)))
		}
		w.String(string(v.State.Summaries[n271].Operator.Kind))
		w.String(string(v.State.Summaries[n271].Operator.IdentityEvidence))
	}
	return w.Finish()
}
func encodePriorLocationOperation(ctx context.Context, p *compact.Pool, v p2.LocationOperation) ([]byte, error) {
	w := compact.NewWriter(ctx, p, 17)
	w.String(string(v.Operation.ID))
	w.String(string(v.Operation.Previous))
	w.String(string(v.Operation.Subject))
	w.String(string(v.Operation.Submission))
	w.String(string(v.Operation.Action))
	w.String(string(v.Operation.EventID))
	w.Int(int64(v.Operation.Number))
	w.String(v.Operation.RecordedAt.UTC().Format(time.RFC3339Nano))
	w.String(string(v.Operation.Operator.SubjectID))
	w.Bool(v.Operation.Operator.DisplayName != nil)
	if v.Operation.Operator.DisplayName != nil {
		w.String(string((*v.Operation.Operator.DisplayName)))
	}
	w.Bool(v.Operation.Operator.Login != nil)
	if v.Operation.Operator.Login != nil {
		w.String(string((*v.Operation.Operator.Login)))
	}
	w.String(string(v.Operation.Operator.Kind))
	w.String(string(v.Operation.Operator.IdentityEvidence))
	w.Bool(v.Operation.Revisions != nil)
	w.Uint(uint64(len(v.Operation.Revisions)))
	for n281 := range v.Operation.Revisions {
		w.Bool(v.Operation.Revisions[n281].Before != nil)
		if v.Operation.Revisions[n281].Before != nil {
			w.String(string((*v.Operation.Revisions[n281].Before).ID))
			w.String(string((*v.Operation.Revisions[n281].Before).Revision))
			w.Int(int64((*v.Operation.Revisions[n281].Before).RevisionNo))
			w.Range((*v.Operation.Revisions[n281].Before).Date, (*v.Operation.Revisions[n281].Before).End)
			w.String(string((*v.Operation.Revisions[n281].Before).Data.Name))
			w.Bool((*v.Operation.Revisions[n281].Before).Data.NameEn != nil)
			if (*v.Operation.Revisions[n281].Before).Data.NameEn != nil {
				w.String(string((*(*v.Operation.Revisions[n281].Before).Data.NameEn)))
			}
			w.String(string((*v.Operation.Revisions[n281].Before).Data.CountryID))
			w.Bool((*v.Operation.Revisions[n281].Before).Data.AreaID != nil)
			if (*v.Operation.Revisions[n281].Before).Data.AreaID != nil {
				w.String(string((*(*v.Operation.Revisions[n281].Before).Data.AreaID)))
			}
			w.String(string((*v.Operation.Revisions[n281].Before).Data.AddressLine))
			w.Bool(priorExtra((*v.Operation.Revisions[n281].Before).Data.AddressLine) != nil)
			if priorExtra((*v.Operation.Revisions[n281].Before).Data.AddressLine) != nil {
				w.String(*priorExtra((*v.Operation.Revisions[n281].Before).Data.AddressLine))
			}
			w.Bool((*v.Operation.Revisions[n281].Before).Data.PostalCode != nil)
			if (*v.Operation.Revisions[n281].Before).Data.PostalCode != nil {
				w.String(string((*(*v.Operation.Revisions[n281].Before).Data.PostalCode)))
			}
			w.String(string((*v.Operation.Revisions[n281].Before).Data.Adoption.Date))
			w.String(string((*v.Operation.Revisions[n281].Before).Data.Adoption.Country.ObjectID))
			w.String(string((*v.Operation.Revisions[n281].Before).Data.Adoption.Country.SourceVersion))
			w.String(string((*v.Operation.Revisions[n281].Before).Data.Adoption.Country.RecordID))
			w.String(string((*v.Operation.Revisions[n281].Before).Data.Adoption.Country.Revision))
			w.Bool((*v.Operation.Revisions[n281].Before).Data.Adoption.Parents != nil)
			w.Uint(uint64(len((*v.Operation.Revisions[n281].Before).Data.Adoption.Parents)))
			for n282 := range (*v.Operation.Revisions[n281].Before).Data.Adoption.Parents {
				w.String(string((*v.Operation.Revisions[n281].Before).Data.Adoption.Parents[n282].ObjectID))
				w.String(string((*v.Operation.Revisions[n281].Before).Data.Adoption.Parents[n282].SourceVersion))
				w.String(string((*v.Operation.Revisions[n281].Before).Data.Adoption.Parents[n282].RecordID))
				w.String(string((*v.Operation.Revisions[n281].Before).Data.Adoption.Parents[n282].Revision))
			}
			w.String(string((*v.Operation.Revisions[n281].Before).Type))
			w.String(string((*v.Operation.Revisions[n281].Before).Reason))
			w.String(string((*v.Operation.Revisions[n281].Before).TypeLabel))
			w.String(string((*v.Operation.Revisions[n281].Before).ReasonLabel))
			w.String(string((*v.Operation.Revisions[n281].Before).State))
			w.String(string((*v.Operation.Revisions[n281].Before).InheritedRecord))
			w.String(string((*v.Operation.Revisions[n281].Before).InheritedRevision))
		}
		w.Bool(v.Operation.Revisions[n281].After != nil)
		if v.Operation.Revisions[n281].After != nil {
			w.String(string((*v.Operation.Revisions[n281].After).ID))
			w.String(string((*v.Operation.Revisions[n281].After).Revision))
			w.Int(int64((*v.Operation.Revisions[n281].After).RevisionNo))
			w.Range((*v.Operation.Revisions[n281].After).Date, (*v.Operation.Revisions[n281].After).End)
			w.String(string((*v.Operation.Revisions[n281].After).Data.Name))
			w.Bool((*v.Operation.Revisions[n281].After).Data.NameEn != nil)
			if (*v.Operation.Revisions[n281].After).Data.NameEn != nil {
				w.String(string((*(*v.Operation.Revisions[n281].After).Data.NameEn)))
			}
			w.String(string((*v.Operation.Revisions[n281].After).Data.CountryID))
			w.Bool((*v.Operation.Revisions[n281].After).Data.AreaID != nil)
			if (*v.Operation.Revisions[n281].After).Data.AreaID != nil {
				w.String(string((*(*v.Operation.Revisions[n281].After).Data.AreaID)))
			}
			w.String(string((*v.Operation.Revisions[n281].After).Data.AddressLine))
			w.Bool(priorExtra((*v.Operation.Revisions[n281].After).Data.AddressLine) != nil)
			if priorExtra((*v.Operation.Revisions[n281].After).Data.AddressLine) != nil {
				w.String(*priorExtra((*v.Operation.Revisions[n281].After).Data.AddressLine))
			}
			w.Bool((*v.Operation.Revisions[n281].After).Data.PostalCode != nil)
			if (*v.Operation.Revisions[n281].After).Data.PostalCode != nil {
				w.String(string((*(*v.Operation.Revisions[n281].After).Data.PostalCode)))
			}
			w.String(string((*v.Operation.Revisions[n281].After).Data.Adoption.Date))
			w.String(string((*v.Operation.Revisions[n281].After).Data.Adoption.Country.ObjectID))
			w.String(string((*v.Operation.Revisions[n281].After).Data.Adoption.Country.SourceVersion))
			w.String(string((*v.Operation.Revisions[n281].After).Data.Adoption.Country.RecordID))
			w.String(string((*v.Operation.Revisions[n281].After).Data.Adoption.Country.Revision))
			w.Bool((*v.Operation.Revisions[n281].After).Data.Adoption.Parents != nil)
			w.Uint(uint64(len((*v.Operation.Revisions[n281].After).Data.Adoption.Parents)))
			for n283 := range (*v.Operation.Revisions[n281].After).Data.Adoption.Parents {
				w.String(string((*v.Operation.Revisions[n281].After).Data.Adoption.Parents[n283].ObjectID))
				w.String(string((*v.Operation.Revisions[n281].After).Data.Adoption.Parents[n283].SourceVersion))
				w.String(string((*v.Operation.Revisions[n281].After).Data.Adoption.Parents[n283].RecordID))
				w.String(string((*v.Operation.Revisions[n281].After).Data.Adoption.Parents[n283].Revision))
			}
			w.String(string((*v.Operation.Revisions[n281].After).Type))
			w.String(string((*v.Operation.Revisions[n281].After).Reason))
			w.String(string((*v.Operation.Revisions[n281].After).TypeLabel))
			w.String(string((*v.Operation.Revisions[n281].After).ReasonLabel))
			w.String(string((*v.Operation.Revisions[n281].After).State))
			w.String(string((*v.Operation.Revisions[n281].After).InheritedRecord))
			w.String(string((*v.Operation.Revisions[n281].After).InheritedRevision))
		}
	}
	w.String(string(v.Operation.Terminal.ID))
	w.String(string(v.Operation.Terminal.ObjectID))
	w.String(string(v.Operation.Terminal.Action))
	w.String(string(v.Operation.Terminal.Outcome))
	w.String(string(v.Operation.Terminal.ErrorCode))
	w.Bytes(v.Operation.Terminal.Digest)
	w.String(string(v.Operation.Terminal.Result.SubmissionID))
	w.String(string(v.Operation.Terminal.Result.Outcome))
	w.Bool(v.Operation.Terminal.Result.LocationID != nil)
	if v.Operation.Terminal.Result.LocationID != nil {
		w.String(string((*v.Operation.Terminal.Result.LocationID)))
	}
	w.Bool(v.Operation.Terminal.Result.SourceVersion != nil)
	if v.Operation.Terminal.Result.SourceVersion != nil {
		w.String(string((*v.Operation.Terminal.Result.SourceVersion)))
	}
	w.Bool(v.Operation.Terminal.Result.SelectedRecordID != nil)
	if v.Operation.Terminal.Result.SelectedRecordID != nil {
		w.String(string((*v.Operation.Terminal.Result.SelectedRecordID)))
	}
	w.Bool(v.Operation.Terminal.Result.TargetRecordID != nil)
	if v.Operation.Terminal.Result.TargetRecordID != nil {
		w.String(string((*v.Operation.Terminal.Result.TargetRecordID)))
	}
	w.Bool(v.Operation.Terminal.Result.TargetRevision != nil)
	if v.Operation.Terminal.Result.TargetRevision != nil {
		w.String(string((*v.Operation.Terminal.Result.TargetRevision)))
	}
	w.Bool(v.Operation.Terminal.Result.EffectiveDate != nil)
	if v.Operation.Terminal.Result.EffectiveDate != nil {
		w.String(string((*v.Operation.Terminal.Result.EffectiveDate)))
	}
	w.Bool(v.Operation.Terminal.Result.RecordedAt != nil)
	if v.Operation.Terminal.Result.RecordedAt != nil {
		w.String(string((*v.Operation.Terminal.Result.RecordedAt)))
	}
	w.Bool(v.Operation.Terminal.Result.ErrorCode != nil)
	if v.Operation.Terminal.Result.ErrorCode != nil {
		w.String(string((*v.Operation.Terminal.Result.ErrorCode)))
	}
	w.String(string(v.Operation.Terminal.Result.Action))
	w.Bool(v.Operation.Terminal.Result.Replayed)
	w.Bool(v.Operation.Terminal.Result.CurrentChanged)
	return w.Finish()
}

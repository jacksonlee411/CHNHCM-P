// This reader is used only by migration 00016. Runtime accepts only the current format.
package bootstrap

import (
	p13 "chnhcm.local/chnhcm/internal/contexts/organization/application/location"
	p10 "chnhcm.local/chnhcm/internal/contexts/organization/domain/geoarea"
	p12 "chnhcm.local/chnhcm/internal/contexts/organization/domain/location"
	p2 "chnhcm.local/chnhcm/internal/contexts/organization/infra/history/model"
	p4 "chnhcm.local/chnhcm/internal/contexts/organization/public"
	compact "chnhcm.local/chnhcm/internal/platform/compact"
	context "context"
	time "time"
)

func mergeAddress(line, extra string) string {
	if extra == "" {
		return line
	}
	if line == "" {
		return extra
	}
	return line + " " + extra
}

func decodePriorLocationState(ctx context.Context, p compact.Values, data []byte) (v p2.LocationState, err error) {
	r := compact.NewReader(ctx, p, data, 16)
	v.State.Timeline.ID = string(r.String())
	v.State.Timeline.Enterprise = string(r.String())
	v.State.Timeline.Code = string(r.String())
	v.State.Timeline.First = string(r.String())
	v.State.Timeline.Source = string(r.String())
	v.State.Timeline.SourceNo = int64(r.Int())
	n272, n273 := r.Bool(), r.Len()
	if n272 {
		v.State.Timeline.Records = make([]p12.Record, n273)
	} else if n273 != 0 {
		r.Err = compact.ErrCorrupt
	}
	for n274 := range v.State.Timeline.Records {
		v.State.Timeline.Records[n274].ID = string(r.String())
		v.State.Timeline.Records[n274].Revision = string(r.String())
		v.State.Timeline.Records[n274].RevisionNo = int64(r.Int())
		v.State.Timeline.Records[n274].Date, v.State.Timeline.Records[n274].End = r.Range()
		v.State.Timeline.Records[n274].Data.Name = string(r.String())
		if r.Bool() {
			v.State.Timeline.Records[n274].Data.NameEn = new(string)
			(*v.State.Timeline.Records[n274].Data.NameEn) = string(r.String())
		}
		v.State.Timeline.Records[n274].Data.CountryID = string(r.String())
		if r.Bool() {
			v.State.Timeline.Records[n274].Data.AreaID = new(string)
			(*v.State.Timeline.Records[n274].Data.AreaID) = string(r.String())
		}
		v.State.Timeline.Records[n274].Data.AddressLine = string(r.String())
		if r.Bool() {
			v.State.Timeline.Records[n274].Data.AddressLine = mergeAddress(v.State.Timeline.Records[n274].Data.AddressLine, r.String())
		}
		if r.Bool() {
			v.State.Timeline.Records[n274].Data.PostalCode = new(string)
			(*v.State.Timeline.Records[n274].Data.PostalCode) = string(r.String())
		}
		v.State.Timeline.Records[n274].Data.Adoption.Date = string(r.String())
		v.State.Timeline.Records[n274].Data.Adoption.Country.ObjectID = string(r.String())
		v.State.Timeline.Records[n274].Data.Adoption.Country.SourceVersion = string(r.String())
		v.State.Timeline.Records[n274].Data.Adoption.Country.RecordID = string(r.String())
		v.State.Timeline.Records[n274].Data.Adoption.Country.Revision = string(r.String())
		n275, n276 := r.Bool(), r.Len()
		if n275 {
			v.State.Timeline.Records[n274].Data.Adoption.Parents = make([]p10.Reference, n276)
		} else if n276 != 0 {
			r.Err = compact.ErrCorrupt
		}
		for n277 := range v.State.Timeline.Records[n274].Data.Adoption.Parents {
			v.State.Timeline.Records[n274].Data.Adoption.Parents[n277].ObjectID = string(r.String())
			v.State.Timeline.Records[n274].Data.Adoption.Parents[n277].SourceVersion = string(r.String())
			v.State.Timeline.Records[n274].Data.Adoption.Parents[n277].RecordID = string(r.String())
			v.State.Timeline.Records[n274].Data.Adoption.Parents[n277].Revision = string(r.String())
		}
		v.State.Timeline.Records[n274].Type = string(r.String())
		v.State.Timeline.Records[n274].Reason = string(r.String())
		v.State.Timeline.Records[n274].TypeLabel = string(r.String())
		v.State.Timeline.Records[n274].ReasonLabel = string(r.String())
		v.State.Timeline.Records[n274].State = string(r.String())
		v.State.Timeline.Records[n274].InheritedRecord = string(r.String())
		v.State.Timeline.Records[n274].InheritedRevision = string(r.String())
	}
	n278, n279 := r.Bool(), r.Len()
	if n278 {
		v.State.Summaries = make([]p4.OperationSummary, n279)
	} else if n279 != 0 {
		r.Err = compact.ErrCorrupt
	}
	for n280 := range v.State.Summaries {
		v.State.Summaries[n280].RecordID = string(r.String())
		v.State.Summaries[n280].Revision = string(r.String())
		v.State.Summaries[n280].OperationSourceVersion = string(r.String())
		v.State.Summaries[n280].RecordedAt = string(r.String())
		v.State.Summaries[n280].Operator.SubjectID = string(r.String())
		if r.Bool() {
			v.State.Summaries[n280].Operator.DisplayName = new(string)
			(*v.State.Summaries[n280].Operator.DisplayName) = string(r.String())
		}
		if r.Bool() {
			v.State.Summaries[n280].Operator.Login = new(string)
			(*v.State.Summaries[n280].Operator.Login) = string(r.String())
		}
		v.State.Summaries[n280].Operator.Kind = string(r.String())
		v.State.Summaries[n280].Operator.IdentityEvidence = string(r.String())
	}
	err = r.Finish()
	return
}
func decodePriorLocationOperation(ctx context.Context, p compact.Values, data []byte) (v p2.LocationOperation, err error) {
	r := compact.NewReader(ctx, p, data, 17)
	v.Operation.ID = string(r.String())
	v.Operation.Previous = string(r.String())
	v.Operation.Subject = string(r.String())
	v.Operation.Submission = string(r.String())
	v.Operation.Action = string(r.String())
	v.Operation.EventID = string(r.String())
	v.Operation.Number = int64(r.Int())
	if r.Err == nil {
		v.Operation.RecordedAt, r.Err = time.Parse(time.RFC3339Nano, r.String())
	}
	v.Operation.Operator.SubjectID = string(r.String())
	if r.Bool() {
		v.Operation.Operator.DisplayName = new(string)
		(*v.Operation.Operator.DisplayName) = string(r.String())
	}
	if r.Bool() {
		v.Operation.Operator.Login = new(string)
		(*v.Operation.Operator.Login) = string(r.String())
	}
	v.Operation.Operator.Kind = string(r.String())
	v.Operation.Operator.IdentityEvidence = string(r.String())
	n284, n285 := r.Bool(), r.Len()
	if n284 {
		v.Operation.Revisions = make([]p13.Revision, n285)
	} else if n285 != 0 {
		r.Err = compact.ErrCorrupt
	}
	for n286 := range v.Operation.Revisions {
		if r.Bool() {
			v.Operation.Revisions[n286].Before = new(p12.Record)
			(*v.Operation.Revisions[n286].Before).ID = string(r.String())
			(*v.Operation.Revisions[n286].Before).Revision = string(r.String())
			(*v.Operation.Revisions[n286].Before).RevisionNo = int64(r.Int())
			(*v.Operation.Revisions[n286].Before).Date, (*v.Operation.Revisions[n286].Before).End = r.Range()
			(*v.Operation.Revisions[n286].Before).Data.Name = string(r.String())
			if r.Bool() {
				(*v.Operation.Revisions[n286].Before).Data.NameEn = new(string)
				(*(*v.Operation.Revisions[n286].Before).Data.NameEn) = string(r.String())
			}
			(*v.Operation.Revisions[n286].Before).Data.CountryID = string(r.String())
			if r.Bool() {
				(*v.Operation.Revisions[n286].Before).Data.AreaID = new(string)
				(*(*v.Operation.Revisions[n286].Before).Data.AreaID) = string(r.String())
			}
			(*v.Operation.Revisions[n286].Before).Data.AddressLine = string(r.String())
			if r.Bool() {
				(*v.Operation.Revisions[n286].Before).Data.AddressLine = mergeAddress((*v.Operation.Revisions[n286].Before).Data.AddressLine, r.String())
			}
			if r.Bool() {
				(*v.Operation.Revisions[n286].Before).Data.PostalCode = new(string)
				(*(*v.Operation.Revisions[n286].Before).Data.PostalCode) = string(r.String())
			}
			(*v.Operation.Revisions[n286].Before).Data.Adoption.Date = string(r.String())
			(*v.Operation.Revisions[n286].Before).Data.Adoption.Country.ObjectID = string(r.String())
			(*v.Operation.Revisions[n286].Before).Data.Adoption.Country.SourceVersion = string(r.String())
			(*v.Operation.Revisions[n286].Before).Data.Adoption.Country.RecordID = string(r.String())
			(*v.Operation.Revisions[n286].Before).Data.Adoption.Country.Revision = string(r.String())
			n287, n288 := r.Bool(), r.Len()
			if n287 {
				(*v.Operation.Revisions[n286].Before).Data.Adoption.Parents = make([]p10.Reference, n288)
			} else if n288 != 0 {
				r.Err = compact.ErrCorrupt
			}
			for n289 := range (*v.Operation.Revisions[n286].Before).Data.Adoption.Parents {
				(*v.Operation.Revisions[n286].Before).Data.Adoption.Parents[n289].ObjectID = string(r.String())
				(*v.Operation.Revisions[n286].Before).Data.Adoption.Parents[n289].SourceVersion = string(r.String())
				(*v.Operation.Revisions[n286].Before).Data.Adoption.Parents[n289].RecordID = string(r.String())
				(*v.Operation.Revisions[n286].Before).Data.Adoption.Parents[n289].Revision = string(r.String())
			}
			(*v.Operation.Revisions[n286].Before).Type = string(r.String())
			(*v.Operation.Revisions[n286].Before).Reason = string(r.String())
			(*v.Operation.Revisions[n286].Before).TypeLabel = string(r.String())
			(*v.Operation.Revisions[n286].Before).ReasonLabel = string(r.String())
			(*v.Operation.Revisions[n286].Before).State = string(r.String())
			(*v.Operation.Revisions[n286].Before).InheritedRecord = string(r.String())
			(*v.Operation.Revisions[n286].Before).InheritedRevision = string(r.String())
		}
		if r.Bool() {
			v.Operation.Revisions[n286].After = new(p12.Record)
			(*v.Operation.Revisions[n286].After).ID = string(r.String())
			(*v.Operation.Revisions[n286].After).Revision = string(r.String())
			(*v.Operation.Revisions[n286].After).RevisionNo = int64(r.Int())
			(*v.Operation.Revisions[n286].After).Date, (*v.Operation.Revisions[n286].After).End = r.Range()
			(*v.Operation.Revisions[n286].After).Data.Name = string(r.String())
			if r.Bool() {
				(*v.Operation.Revisions[n286].After).Data.NameEn = new(string)
				(*(*v.Operation.Revisions[n286].After).Data.NameEn) = string(r.String())
			}
			(*v.Operation.Revisions[n286].After).Data.CountryID = string(r.String())
			if r.Bool() {
				(*v.Operation.Revisions[n286].After).Data.AreaID = new(string)
				(*(*v.Operation.Revisions[n286].After).Data.AreaID) = string(r.String())
			}
			(*v.Operation.Revisions[n286].After).Data.AddressLine = string(r.String())
			if r.Bool() {
				(*v.Operation.Revisions[n286].After).Data.AddressLine = mergeAddress((*v.Operation.Revisions[n286].After).Data.AddressLine, r.String())
			}
			if r.Bool() {
				(*v.Operation.Revisions[n286].After).Data.PostalCode = new(string)
				(*(*v.Operation.Revisions[n286].After).Data.PostalCode) = string(r.String())
			}
			(*v.Operation.Revisions[n286].After).Data.Adoption.Date = string(r.String())
			(*v.Operation.Revisions[n286].After).Data.Adoption.Country.ObjectID = string(r.String())
			(*v.Operation.Revisions[n286].After).Data.Adoption.Country.SourceVersion = string(r.String())
			(*v.Operation.Revisions[n286].After).Data.Adoption.Country.RecordID = string(r.String())
			(*v.Operation.Revisions[n286].After).Data.Adoption.Country.Revision = string(r.String())
			n290, n291 := r.Bool(), r.Len()
			if n290 {
				(*v.Operation.Revisions[n286].After).Data.Adoption.Parents = make([]p10.Reference, n291)
			} else if n291 != 0 {
				r.Err = compact.ErrCorrupt
			}
			for n292 := range (*v.Operation.Revisions[n286].After).Data.Adoption.Parents {
				(*v.Operation.Revisions[n286].After).Data.Adoption.Parents[n292].ObjectID = string(r.String())
				(*v.Operation.Revisions[n286].After).Data.Adoption.Parents[n292].SourceVersion = string(r.String())
				(*v.Operation.Revisions[n286].After).Data.Adoption.Parents[n292].RecordID = string(r.String())
				(*v.Operation.Revisions[n286].After).Data.Adoption.Parents[n292].Revision = string(r.String())
			}
			(*v.Operation.Revisions[n286].After).Type = string(r.String())
			(*v.Operation.Revisions[n286].After).Reason = string(r.String())
			(*v.Operation.Revisions[n286].After).TypeLabel = string(r.String())
			(*v.Operation.Revisions[n286].After).ReasonLabel = string(r.String())
			(*v.Operation.Revisions[n286].After).State = string(r.String())
			(*v.Operation.Revisions[n286].After).InheritedRecord = string(r.String())
			(*v.Operation.Revisions[n286].After).InheritedRevision = string(r.String())
		}
	}
	v.Operation.Terminal.ID = string(r.String())
	v.Operation.Terminal.ObjectID = string(r.String())
	v.Operation.Terminal.Action = string(r.String())
	v.Operation.Terminal.Outcome = string(r.String())
	v.Operation.Terminal.ErrorCode = string(r.String())
	v.Operation.Terminal.Digest = r.Bytes()
	v.Operation.Terminal.Result.SubmissionID = string(r.String())
	v.Operation.Terminal.Result.Outcome = string(r.String())
	if r.Bool() {
		v.Operation.Terminal.Result.LocationID = new(string)
		(*v.Operation.Terminal.Result.LocationID) = string(r.String())
	}
	if r.Bool() {
		v.Operation.Terminal.Result.SourceVersion = new(string)
		(*v.Operation.Terminal.Result.SourceVersion) = string(r.String())
	}
	if r.Bool() {
		v.Operation.Terminal.Result.SelectedRecordID = new(string)
		(*v.Operation.Terminal.Result.SelectedRecordID) = string(r.String())
	}
	if r.Bool() {
		v.Operation.Terminal.Result.TargetRecordID = new(string)
		(*v.Operation.Terminal.Result.TargetRecordID) = string(r.String())
	}
	if r.Bool() {
		v.Operation.Terminal.Result.TargetRevision = new(string)
		(*v.Operation.Terminal.Result.TargetRevision) = string(r.String())
	}
	if r.Bool() {
		v.Operation.Terminal.Result.EffectiveDate = new(string)
		(*v.Operation.Terminal.Result.EffectiveDate) = string(r.String())
	}
	if r.Bool() {
		v.Operation.Terminal.Result.RecordedAt = new(string)
		(*v.Operation.Terminal.Result.RecordedAt) = string(r.String())
	}
	if r.Bool() {
		v.Operation.Terminal.Result.ErrorCode = new(string)
		(*v.Operation.Terminal.Result.ErrorCode) = string(r.String())
	}
	v.Operation.Terminal.Result.Action = string(r.String())
	v.Operation.Terminal.Result.Replayed = r.Bool()
	v.Operation.Terminal.Result.CurrentChanged = r.Bool()
	err = r.Finish()
	return
}

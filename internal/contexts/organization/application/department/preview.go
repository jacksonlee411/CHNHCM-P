package department

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/application/enterprisecheck"
	"chnhcm.local/chnhcm/internal/contexts/organization/application/submission"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
)

func (s Service) Preview(ctx context.Context, c access.Credentials, id string, in org.DepartmentPreviewInput) (out org.DepartmentPreview, err error) {
	cmd := org.DepartmentCommand{Action: in.Action, DepartmentID: id, RecordID: in.RecordID, SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Add: org.DepartmentAddInput{ExpectedSourceVersion: in.ExpectedSourceVersion, AnchorRecordID: in.AnchorRecordID, EffectiveDate: in.EffectiveDate, OperationType: in.OperationType, OperationReason: in.OperationReason, Changes: in.Changes}, Correct: org.DepartmentCorrectInput{ExpectedSourceVersion: in.ExpectedSourceVersion, ExpectedRecordRevision: in.ExpectedRecordRevision, NewEffectiveDate: in.NewEffectiveDate, Classification: in.Classification, Changes: in.Changes}, Delete: org.DepartmentDeleteInput{ExpectedSourceVersion: in.ExpectedSourceVersion, ExpectedRecordRevision: in.ExpectedRecordRevision}, Redate: org.DepartmentRedateInput{ExpectedSourceVersion: in.ExpectedSourceVersion, ExpectedRecordRevision: in.ExpectedRecordRevision, NewEffectiveDate: in.NewEffectiveDate}}
	terminal, e := commandMetadata(cmd)
	if e != nil {
		return out, e
	}
	cmd.Fields = terminal.Fields
	err = s.Store.WithinDepartment(ctx, c, func(u Unit) error {
		ent, e := u.Enterprise(ctx, false)
		if e != nil {
			return e
		}
		now, e := u.Now(ctx)
		if e != nil {
			return e
		}
		bu, parent, _, _, _ := explicitTargets(cmd, d.Timeline{})
		read, e := u.PreviewSnapshot(ctx, id, []string{bu}, []string{parent}, false)
		if e != nil {
			return e
		}
		t := read.Snapshot.Department(id)
		if t == nil {
			return submission.RejectRead(missing())
		}
		fields, e := readable(ctx, u, *t, now, false)
		if e != nil {
			return submission.RejectRead(e)
		}
		out.ReadFields = fields
		target := fact(*t, now, "")
		if _, e = require(ctx, u, requirements(cmd, *t, target)); e != nil {
			return submission.RejectRead(e)
		}
		if e = preauthorizeRelations(ctx, u, read.Snapshot, *t, cmd, target, now); e != nil {
			return submission.RejectRead(e)
		}
		if t.Source != in.ExpectedSourceVersion {
			return submission.RejectRead(foundation.Fail(409, "M02_SOURCE_CONFLICT", "资料已更新，请核对最新内容后保存"))
		}
		work, items, e := localCandidate(ctx, cmd, *t, ent.ValidFrom, d.Kernel)
		if e != nil {
			return submission.RejectRead(e)
		}
		// Compare all active dated edges, not just declared fields or identity
		// deltas: moving the first date or changing state also affects children.
		if d.Plan(*t, work.Records).Changed {
			read, e = u.PreviewSnapshot(ctx, id, []string{bu}, []string{parent}, true)
			if e != nil {
				return e
			}
			t = read.Snapshot.Department(id)
			if t == nil || t.Source != in.ExpectedSourceVersion {
				return submission.RejectRead(foundation.Fail(409, "M02_SOURCE_CONFLICT", "资料已更新，请核对最新内容后保存"))
			}
		}
		candidate, _, _, e := qualifyPreparedCandidate(ctx, u, read, *t, cmd, target, now, ent, work, items)
		if e != nil {
			return submission.RejectRead(e)
		}
		if e = enterprisecheck.Records(ent, candidate.Records); e != nil {
			return submission.RejectRead(e)
		}
		result := org.DepartmentResult{Impact: impact(candidate.Impact)}
		trimResult(&result, fields)
		out.Valid = true
		out.NoChange = candidate.NoChange
		out.Impact = result.Impact
		return nil
	})
	return
}

package application

import (
	"bytes"
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	s "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"errors"
	"slices"
	"time"
)

func quotaNotFound() error {
	return foundation.Fail(404, "M03_NOT_FOUND", "未找到该岗位或额度记录")
}
func quotaDenied() error {
	return foundation.Fail(403, "M03_FORBIDDEN", "当前没有办理此项编制业务的权限")
}
func quotaDecision(ctx context.Context, u QuotaTx, enterprise, position, action string, fields []string) (access.Decision, error) {
	kind := "quota-target"
	if position == "" {
		kind = "quota-list"
	}
	return u.Access().Decide(ctx, access.Intent{Kind: kind, EnterpriseID: enterprise, PositionID: position, Action: "staffingquota." + action, Fields: fields})
}
func quotaPrepare(ctx context.Context, u QuotaTx, c s.QuotaCommand, lock bool) (d.QuotaPlan, error) {
	var empty d.QuotaPlan
	ent, err := u.Enterprise(ctx, lock)
	if err != nil {
		return empty, err
	}
	// Check action authority before disclosing target existence or conflicting values.
	purpose := c.Action
	if purpose == "allocate" {
		purpose = "correct"
	}
	gate, err := quotaDecision(ctx, u, ent.ID, "", purpose, nil)
	if err != nil {
		return empty, err
	}
	if !gate.Allowed {
		return empty, quotaDenied()
	}
	target := d.QuotaTarget{Kind: "position", ID: c.Input.PositionID}
	if c.DepartmentID != "" {
		target = d.QuotaTarget{Kind: "department", ID: c.DepartmentID}
	}
	if c.Action != "create" {
		if !foundation.ValidID(c.QuotaID) {
			return empty, quotaNotFound()
		}
		target, err = u.QuotaIdentity(ctx, c.QuotaID)
		if err != nil {
			return empty, err
		}
	}
	if !foundation.ValidID(target.ID) {
		return empty, quotaNotFound()
	}
	read, err := allocationDecision(ctx, u, ent.ID, target, "read", nil)
	if err != nil {
		return empty, err
	}
	if !read.Allowed {
		return empty, quotaNotFound()
	}
	fields := []string{"effectiveDate", "reason", "quota"}
	if c.Action == "correct" {
		fields = nil
		if c.Input.Amount != nil {
			fields = append(fields, "quota")
		}
		if c.Input.ReasonProvided || c.Input.Reason != "" {
			fields = append(fields, "reason")
		}
	}
	if c.Action == "allocate" {
		fields = []string{"reason", "quota"}
	}
	if c.Action == "redate" || c.Action == "delete" {
		fields = []string{"effectiveDate"}
	}
	dec, err := allocationDecision(ctx, u, ent.ID, target, purpose, fields)
	if err != nil {
		return empty, err
	}
	if !dec.Allowed || !has(read.ReadFields, fields...) || c.Action != "create" && !has(read.ReadFields, "quota", "sourceVersion") {
		return empty, quotaDenied()
	}
	if lock {
		if err = lockAllocationTargets(ctx, u, []d.QuotaTarget{target}); err != nil {
			return empty, err
		}
	}
	position, err := allocationTimeline(ctx, u, ent.ID, target, c.Input.EffectiveDate != "" && c.Input.EffectiveDate != today(u), lock)
	if err != nil {
		return empty, err
	}
	before, err := u.LoadTargetQuota(ctx, target, "")
	if err != nil {
		return empty, err
	}
	date := c.Input.EffectiveDate
	if c.Action == "redate" {
		date = c.Input.NewDate
	}
	dates := []string{date}
	if c.RecordID != "" {
		if i := slices.IndexFunc(before.Records, func(r d.QuotaRecord) bool { return r.ID == c.RecordID }); i >= 0 {
			dates = append(dates, before.Records[i].Date)
		}
	}
	for _, day := range dates {
		if day != "" && day != today(u) {
			hist, err := allocationRights(ctx, u, ent.ID, target, true)
			if err != nil {
				return empty, err
			}
			if !hist.Allowed || !has(hist.ReadFields, fields...) || c.Action != "create" && !has(hist.ReadFields, "quota", "sourceVersion") {
				return empty, quotaDenied()
			}
			if target.Kind == "department" {
				if _, err := allocationTimeline(ctx, u, ent.ID, target, true, false); err != nil {
					return empty, err
				}
			}
			break
		}
	}
	// Moving or removing a boundary extends a neighbour's value into the old interval.
	// Recheck quota write authority when the actual allocation changes there.
	if c.Action == "redate" || c.Action == "delete" {
		index := slices.IndexFunc(before.Records, func(r d.QuotaRecord) bool { return r.ID == c.RecordID })
		if index >= 0 && (c.Action == "delete" || date != before.Records[index].Date) && before.BoundaryChangesAmount(index) {
			allowed, e := allocationDecision(ctx, u, ent.ID, target, purpose, []string{"effectiveDate", "quota"})
			if e != nil {
				return empty, e
			}
			if !allowed.Allowed || !has(read.ReadFields, "quota") {
				return empty, quotaDenied()
			}
		}
	}
	plan, err := d.ChangeQuota(before, d.QuotaChange{Action: c.Action, Source: c.Input.ExpectedSource, RecordID: c.RecordID, Revision: c.Input.ExpectedRevision, Date: date, Reason: c.Input.Reason, ReasonProvided: c.Input.ReasonProvided, Amount: c.Input.Amount}, position)
	if err != nil || !plan.Changed || target.Kind != "position" {
		return plan, err
	}
	usage, err := u.PositionUsage(ctx, target.ID)
	if err != nil {
		return empty, err
	}
	return plan, usage.CheckTightening(before, plan.After, position, position)
}
func quotaRedact(ctx context.Context, u QuotaTx, out *s.Result) error {
	out.Kind = s.Quota
	if out.Replayed && out.Failure != nil {
		out.Failure = foundation.Fail(out.FailureStatus, out.Failure.Code, "原提交未生效")
	}
	if out.DepartmentID != "" {
		ent, err := u.Enterprise(ctx, false)
		if err != nil {
			return err
		}
		dep := d.QuotaTarget{Kind: "department", ID: out.DepartmentID}
		_, err = allocationTimeline(ctx, u, ent.ID, dep, out.Replayed, false)
		if err != nil {
			var f *foundation.Fault
			if !errors.As(err, &f) || f.Status != 404 {
				return err
			}
			out.DepartmentID = ""
			out.QuotaChanges = nil
			return nil
		}
		visible := make([]s.AllocationRef, 0, len(out.QuotaChanges))
		showDepartment := false
		for _, row := range out.QuotaChanges {
			read, err := allocationRights(ctx, u, ent.ID, d.QuotaTarget{Kind: row.TargetKind, ID: row.TargetID}, out.Replayed)
			if err != nil {
				return err
			}
			if read.Allowed && has(read.ReadFields, "quota", "sourceVersion") {
				if row.TargetKind == "department" && !has(read.ReadFields, "departmentId", "status") {
					continue
				}
				showDepartment = showDepartment || has(read.ReadFields, "departmentId")
				visible = append(visible, row)
			}
		}
		if !showDepartment {
			out.DepartmentID = ""
		}
		out.QuotaChanges = visible
	}
	if out.ObjectID == "" {
		return nil
	}
	target, err := u.QuotaIdentity(ctx, out.ObjectID)
	if err != nil {
		return err
	}
	ent, err := u.Enterprise(ctx, false)
	if err != nil {
		return err
	}
	read, err := allocationDecision(ctx, u, ent.ID, target, "read", nil)
	if err != nil {
		return err
	}
	history, err := allocationDecision(ctx, u, ent.ID, target, "history.read", nil)
	if err != nil {
		return err
	}
	if target.Kind == "department" {
		_, e := allocationTimeline(ctx, u, ent.ID, target, out.Replayed, false)
		if e != nil {
			var f *foundation.Fault
			if !errors.As(e, &f) || f.Status != 404 {
				return e
			}
			read.Allowed = false
		}
	}
	if !read.Allowed || !has(read.ReadFields, "quota", "sourceVersion") {
		out.ObjectID = ""
		out.SourceVersion = ""
		out.SelectedRecordID = ""
		out.SelectedRecordRevision = ""
	}
	if out.Replayed && (!history.Allowed || !has(history.ReadFields, "quota", "sourceVersion")) {
		out.SourceVersion = ""
		out.SelectedRecordID = ""
		out.SelectedRecordRevision = ""
	}
	if out.SourceVersion != "" && out.SelectedRecordID != "" {
		if target.Kind == "position" {
			out.PositionID = target.ID
		} else {
			out.DepartmentID = target.ID
		}
		// Old immutable receipts also resolve the exact pre-deletion source.
		if out.Action == "delete" && history.Allowed && has(history.ReadFields, "quota", "sourceVersion") {
			previous, e := u.PreviousQuotaSource(ctx, out.ObjectID, out.SourceVersion)
			if e != nil {
				return e
			}
			out.OriginalSourceVersion = previous
		}
	}
	return nil
}
func (svc Service) ChangeStaffingQuota(ctx context.Context, creds foundation.Credentials, c s.QuotaCommand) (out s.Result, err error) {
	if !foundation.ValidID(c.SubmissionID) || len(c.Digest) != 32 {
		return out, foundation.Fail(400, "M03_INVALID_INPUT", "提交号无效")
	}
	const k = d.Kind(s.Quota)
	err = svc.QuotaStore.WithinQuota(ctx, creds, true, func(u QuotaTx) error {
		if e := u.LockSubmission(ctx, k, c.SubmissionID); e != nil {
			return e
		}
		old, e := u.Terminal(ctx, k, c.SubmissionID)
		if e != nil {
			return e
		}
		if old.Exists {
			if old.Result.Outcome != "not_applied" && !bytes.Equal(old.Digest, c.Digest) {
				return foundation.Fail(409, "M03_SUBMISSION_MISMATCH", "原提交号已用于其他内容")
			}
			out = old.Result
			out.Replayed = true
			return quotaRedact(ctx, u, &out)
		}
		if c.Action == "replay" {
			return foundation.Fail(409, "M03_QUOTA_ACTION_RETIRED", "此提交尚未办理，请从岗位编制重新开始")
		}
		out = s.Result{Kind: s.Quota, Action: c.Action, SubmissionID: c.SubmissionID}
		e = u.Business(ctx, func() error {
			plans, actions, e := allocationPlans(ctx, u, c, true)
			if e != nil {
				return e
			}
			out.Outcome = "no_change"
			if c.Allocation != nil {
				out.DepartmentID = c.DepartmentID
			}
			for index, p := range plans {
				command := c
				command.Action = actions[index]
				i := slices.IndexFunc(p.After.Records, func(r d.QuotaRecord) bool { return r.ID == p.Selected })
				if c.Allocation == nil {
					out.ObjectID = p.After.ID
					out.SourceVersion = p.After.Source
					out.SelectedRecordID = p.Selected
					if i >= 0 {
						out.SelectedRecordRevision = p.After.Records[i].Revision
					}
				}
				if !p.Changed {
					continue
				}
				op, e := u.Access().OperatorSnapshot(ctx)
				if e != nil {
					return e
				}
				if i >= 0 {
					a := d.Audit{Action: command.Action, SubjectID: op.SubjectID, IdentityKind: op.Kind, RecordedAt: u.Now().Format(time.RFC3339Nano), SourceVersion: p.After.Source}
					if op.DisplayName != nil {
						a.DisplayName = *op.DisplayName
					}
					if op.Login != nil {
						a.Login = *op.Login
					}
					p.After.Records[i].Audit = a
				}
				if e = u.Access().ValidateUse(); e != nil {
					return e
				}
				if e = u.SaveQuota(ctx, p, command, op); e != nil {
					return e
				}
				out.Outcome = "accepted"
				if c.Allocation != nil {
					t := p.After.Target()
					out.QuotaChanges = append(out.QuotaChanges, s.AllocationRef{TargetKind: t.Kind, TargetID: t.ID, QuotaID: p.After.ID, SourceVersion: p.After.Source, RecordID: p.Selected})
				}
			}
			return nil
		})
		if e != nil {
			var f *foundation.Fault
			if !errors.As(e, &f) || f.Status >= 500 {
				return e
			}
			out = s.Result{Kind: s.Quota, SubmissionID: c.SubmissionID, Action: c.Action, Outcome: "rejected", Failure: f, FailureStatus: f.Status}
		}
		if e = u.Close(ctx, k, c.SubmissionID, c.Digest, out); e != nil {
			return e
		}
		return quotaRedact(ctx, u, &out)
	})
	return
}
func (svc Service) ResolveQuota(ctx context.Context, c foundation.Credentials, id string) (out s.Result, err error) {
	if !foundation.ValidID(id) {
		return out, foundation.Fail(400, "M03_INVALID_INPUT", "提交号无效")
	}
	err = svc.QuotaStore.WithinQuota(ctx, c, true, func(u QuotaTx) error {
		const k = d.Kind(s.Quota)
		if e := u.LockSubmission(ctx, k, id); e != nil {
			return e
		}
		old, e := u.Terminal(ctx, k, id)
		if e != nil {
			return e
		}
		if old.Exists {
			out = old.Result
			out.Replayed = true
			return quotaRedact(ctx, u, &out)
		}
		out = s.Result{Kind: s.Quota, SubmissionID: id, Outcome: "not_applied"}
		return u.Close(ctx, k, id, nil, out)
	})
	return
}
func (svc Service) PreviewQuota(ctx context.Context, c foundation.Credentials, cmd s.QuotaCommand) (out s.View, err error) {
	err = svc.QuotaStore.WithinQuota(ctx, c, false, func(u QuotaTx) error {
		plans, _, e := allocationPlans(ctx, u, cmd, false)
		if e != nil {
			return e
		}
		ent, e := u.Enterprise(ctx, false)
		if e != nil {
			return e
		}
		p := plans[0]
		dec, e := allocationRights(ctx, u, ent.ID, p.After.Target(), true)
		if e != nil {
			return e
		}
		out = s.View{}
		if !dec.Allowed {
			return nil
		}
		if cmd.Action == "delete" {
			i := slices.IndexFunc(p.Before.Records, func(r d.QuotaRecord) bool { return r.ID == p.Selected })
			// The earlier amount replaces only the deleted record's previous span.
			if i > 0 {
				if has(dec.ReadFields, "effectiveDate") {
					out["effectiveDate"] = p.Before.Records[i].Date
					if i+1 < len(p.Before.Records) {
						out["endInclusive"] = dayBefore(p.Before.Records[i+1].Date)
					}
				}
				if has(dec.ReadFields, "quota") {
					prior := p.Before.Records[i-1].Amount
					out["quotaState"] = "unset"
					if prior != nil {
						out["quotaState"] = "set"
					}
					out["quota"] = prior
				}
			}
			return nil
		}
		for i, r := range p.After.Records {
			if r.ID == p.Selected {
				out = quotaRecord(p.After, i, dec.ReadFields)
				break
			}
		}
		return nil
	})
	return
}

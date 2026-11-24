package postgres

import (
	"context"
	"strings"
	"unicode/utf8"

	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	f "chnhcm.local/chnhcm/internal/foundation"
)

func (u *unit) validatePreceding(ctx context.Context, a h.Action, day string, ref *h.ActionRef) error {
	if ref == nil {
		return nil
	}
	if !ref.Valid() {
		return assignmentConflict()
	}
	prior, e := u.action(ctx, ref.Action)
	if e != nil {
		return e
	}
	if prior.Revision != ref.Revision || prior.BusinessDate != day || prior.Operation == "Delete" || prior.Operation == "Withdraw" || prior.Employment != a.Employment {
		return assignmentConflict()
	}
	if h.RelationAction(prior.Kind) {
		_, visible, e := u.relationVisibleAssignment(ctx, prior, "history")
		if e != nil {
			return e
		}
		if !visible {
			return f.Fail(403, "ACTION_DEPENDENCY", "当前权限不允许引用原办理")
		}
	} else {
		if prior.Assignment != a.Assignment {
			return assignmentConflict()
		}
		disclosed, e := u.actionDisclosure(ctx, prior, true)
		if e != nil {
			return e
		}
		if !disclosed.from && !disclosed.to {
			return f.Fail(403, "ACTION_DEPENDENCY", "当前权限不允许引用原办理")
		}
	}
	return nil
}

func (st *Store) RecordPromotion(ctx context.Context, c f.Credentials, id, action, kind, key string, in h.PromotionCommand) (h.Receipt, error) {
	return st.employmentCommand(ctx, c, key, []any{kind, id, action, in}, func(u *unit) (employmentResult, error) {
		return u.promotion(ctx, id, action, kind, in)
	})
}
func (u *unit) promotion(ctx context.Context, id, action, kind string, in h.PromotionCommand) (out employmentResult, err error) {
	notice, reason := strings.TrimSpace(in.Notice), strings.TrimSpace(in.Reason)
	withdraw := kind == "withdraw"
	if !in.Basis.Valid() || !withdraw && in.Basis != nil || !in.Previous.Valid() || !in.Precedes.Valid() || !f.ValidID(in.ExpectedSource) || action != "" && (!f.ValidID(action) || !f.ValidID(in.ExpectedRevision)) ||
		action == "" && in.ExpectedRevision != "" || !withdraw && (!f.ValidDate(in.BusinessDate) || !f.ValidID(in.Grade)) ||
		(in.Job != "") != (in.Department != "") || in.Job != "" && (!f.ValidID(in.Job) || !f.ValidID(in.Department)) ||
		in.TransferNotice != "" && kind != "pair" || kind == "pair" && in.Job == "" ||
		kind == "grade" && (in.Job != "" || in.Precedes != nil) || utf8.RuneCountInString(notice) > 1000 || utf8.RuneCountInString(reason) > 1000 || utf8.RuneCountInString(in.TransferNotice) > 1000 {
		return out, f.Fail(400, "PROMOTION_INPUT", "请核对任期、生效日期、职级与依据")
	}
	if withdraw && (action == "" || reason == "" && notice == "" && in.Basis == nil || in.Grade != "" || in.BusinessDate != "" || in.Job != "" || in.Previous != nil || in.Precedes != nil) {
		return out, f.Fail(400, "PROMOTION_WITHDRAW", "请引用原通知或简述误录原因")
	}
	var a h.Action
	if id == "" {
		orig, e := u.action(ctx, action)
		if e != nil {
			return out, e
		}
		id = orig.Assignment
	}
	a, err = u.assignmentIdentity(ctx, id)
	if err != nil {
		return out, err
	}
	if e := lock(ctx, u.tx, false, "staffing.hire.person", u.store.Instance, a.Person); e != nil {
		return out, e
	}
	a, err = u.assignmentIdentity(ctx, id)
	if err != nil {
		return out, err
	}
	day := in.BusinessDate
	if withdraw {
		orig, e := u.action(ctx, action)
		if e != nil {
			return out, e
		}
		day = orig.BusinessDate
	}
	if !a.Includes(day) {
		return out, f.Fail(409, "PROMOTION_PERIOD", "所选日期不在实际任期内")
	}
	scoped, e := u.assignmentAt(ctx, id, day, "")
	if e != nil {
		return out, e
	}
	permit := "promotion"
	switch kind {
	case "grade":
		permit = "grade.record"
	case "withdraw":
		permit = "promotion.withdraw"
	case "promotion", "pair":
	default:
		return out, f.Fail(400, "PROMOTION_INPUT", "无效的办理类型")
	}
	if e = u.requireEmployment(ctx, permit, a.Person, a.Employer, scoped.Department); e != nil {
		return out, e
	}
	if a.Cancelled || a.SourceVersion != in.ExpectedSource {
		return out, assignmentConflict()
	}
	initial, rows, e := u.historyRows(ctx, a)
	if e != nil {
		return out, e
	}
	originals := map[string]savedRecord{}
	index := -1
	next := -1
	for i, r := range rows {
		originals[r.ID] = r
		if r.ID == action {
			index = i
		}
		if in.Precedes != nil && r.ID == in.Precedes.Action {
			next = i
		}
	}
	old := savedRecord{}
	var target savedRecord
	if action != "" {
		if index < 0 {
			return out, assignmentConflict()
		}
		old = rows[index]
		target = old
		if old.Kind != "Promotion" || old.Removed || old.Revision != in.ExpectedRevision || old.Day != day {
			return out, assignmentConflict()
		}
		for _, dept := range affectedDepartments(old) {
			if e = u.requireEmployment(ctx, permit, a.Person, a.Employer, dept); e != nil {
				return out, e
			}
		}
	} else {
		target.ID = f.NewID()
		target.Kind = "Promotion"
		if kind == "grade" {
			target.Kind = "GradeRecord"
		}
		index = len(rows)
		rows = append(rows, target)
	}
	target.Revision = f.NewID()
	target.Day = day
	if withdraw {
		target.Operation = "Withdraw"
		target.Removed = true
		ev := *target.Evidence.Promotion
		ev.PriorSource = a.SourceVersion
		if in.Basis != nil {
			if e = u.validatePreceding(ctx, a, day, in.Basis); e != nil {
				return out, e
			}
			original, e := u.action(ctx, in.Basis.Action)
			if e != nil {
				return out, e
			}
			if original.Notice == "" {
				return out, f.Fail(409, "PROMOTION_BASIS", "所选办理缺少可引用通知")
			}
			ev.WithdrawalBasis = in.Basis
			notice = original.Notice
		}
		ev.Withdrawal = reason
		if ev.Withdrawal == "" {
			ev.Withdrawal = notice
		}
		target.Evidence.Promotion = &ev
		for i, row := range rows {
			if row.Previous != nil && row.Previous.Action == action {
				rows[i].Previous = old.Previous
			}
		}
		rows[index] = target
		return u.publishRecordEdit(ctx, a, initial, rows, originals, index, permit, "")
	}
	if notice == "" && action == "" {
		return out, f.Fail(400, "PROMOTION_NOTICE", "请填写原任用依据")
	}
	if notice != "" {
		if old.ID == "" || old.Notice != notice {
			target.Evidence.NoticeRef = &h.ActionRef{Action: target.ID, Revision: target.Revision}
		}
		target.Notice = notice
	}
	if reason == "" {
		if old.Evidence.Promotion != nil {
			reason = old.Evidence.Promotion.Reason
		} else if kind == "grade" {
			reason = "补录原任用职级"
		} else {
			reason = "职级晋升"
		}
	}
	target.Operation = "Add"
	if action != "" {
		target.Operation = "Correct"
	}
	if in.Previous != nil {
		if e = u.validatePreceding(ctx, a, day, in.Previous); e != nil {
			return out, e
		}
		target.Previous = in.Previous
	}
	job := scoped.Job
	if old.Grade != nil {
		job = old.Grade.Job
	}
	if in.Precedes != nil {
		if e = u.validatePreceding(ctx, a, day, in.Precedes); e != nil {
			return out, e
		}
		if next < 0 || rows[next].Kind != "Transfer" || rows[next].ID == target.ID {
			return out, assignmentConflict()
		}
		successor := &rows[next]
		if successor.Previous != nil && successor.Previous.Action != target.ID {
			if target.Previous != nil && target.Previous.Action != successor.Previous.Action {
				return out, assignmentConflict()
			}
			target.Previous = successor.Previous
		}
		successor.Previous = &h.ActionRef{Action: target.ID, Revision: target.Revision}
		target.Operation = "Insert"
		job = successor.Evidence.Transfer.BeforeJob
	}
	if in.Job != "" {
		job = in.Job
	}
	grade, e := u.gradeForJob(ctx, job, in.Grade, oneDay(day))
	if e != nil {
		return out, e
	}
	target.Grade = grade
	target.Evidence.Promotion = &h.PromotionEvidence{After: *grade, Reason: reason}
	pair := ""
	if in.Job != "" {
		for _, dept := range []string{scoped.Department, in.Department} {
			for _, right := range []string{"read", "transfer", permit} {
				if e = u.requireEmployment(ctx, right, a.Person, a.Employer, dept); e != nil {
					return out, e
				}
			}
		}
		transfer := &h.JobTransfer{Department: in.Department, Job: in.Job}
		if action == "" && scoped.Department == in.Department && scoped.Job == in.Job {
			return out, f.Fail(409, "TRANSFER_UNCHANGED", "部门和职务均未改变")
		}
		if kind == "pair" {
			companion := savedRecord{Operation: "Add", Notice: strings.TrimSpace(in.TransferNotice)}
			if companion.Notice == "" {
				companion.Notice = target.Notice
			}
			companion.Kind = "Transfer"
			companion.Day = day
			companion.ID = f.NewID()
			companion.Revision = f.NewID()
			companion.Evidence.NoticeRef = target.Evidence.NoticeRef
			if strings.TrimSpace(in.TransferNotice) != "" {
				companion.Evidence.NoticeRef = &h.ActionRef{Action: companion.ID, Revision: companion.Revision}
			}
			companion.Transfer = transfer
			companion.Previous = &h.ActionRef{Action: target.ID, Revision: target.Revision}
			pair = companion.ID
			rows = append(rows, companion)
		} else {
			target.Transfer = transfer
		}
	}
	if action != "" && in.Grade == old.Grade.ID && reason == old.Evidence.Promotion.Reason && target.Notice == old.Notice && in.Job == "" && in.Precedes == nil && in.Previous == nil {
		return out, f.Fail(409, "PROMOTION_UNCHANGED", "原晋升内容未改变")
	}
	rows[index] = target
	return u.publishRecordEdit(ctx, a, initial, rows, originals, index, permit, pair)
}
func affectedDepartments(r savedRecord) []string {
	deps := []string{r.Evidence.Promotion.Department}
	if r.Transfer != nil {
		deps = append(deps, r.Transfer.Department)
	}
	return deps
}

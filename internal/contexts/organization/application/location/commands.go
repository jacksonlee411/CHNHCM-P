package location

import (
	"context"
	"errors"
	"time"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/application/recordhistory"
	"chnhcm.local/chnhcm/internal/contexts/organization/application/submission"
	domain "chnhcm.local/chnhcm/internal/contexts/organization/domain"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/location"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

func ptr(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}
func (s Service) Maintain(ctx context.Context, c access.Credentials, cmd org.LocationCommand) (org.LocationResult, error) {
	if cmd.Action == "CREATE" && cmd.Create.EffectiveDate == "" {
		cmd.Create.EffectiveDate = foundation.DefaultEffectiveDate
	}
	terminal, fields, e := metadata(cmd)
	if e != nil {
		return org.LocationResult{}, e
	}
	reply, e := submission.Run(ctx, func(ctx context.Context, fn func(Unit) error) error { return s.Store.WithinLocationWrite(ctx, c, fn) }, func(u Unit, reply *submission.Reply[org.LocationResult]) error {
		p := u.Access().Principal()
		terminal.Digest = submission.BindDigest(p.InstanceID, p.ID, terminal.Digest)
		saved, e := submission.Lookup[org.LocationResult, Terminal](ctx, u, terminal.ID)
		if e != nil {
			return e
		}
		if saved != nil {
			*reply = submission.Replay(ctx, u, saved, terminal.Digest, submission.ReplayPolicy[org.LocationResult]{Outcome: func(r org.LocationResult) string { return r.Outcome }, Read: func(ctx context.Context, v *Saved) error { return readSaved(ctx, u, v) }, MarkReplayed: func(r *org.LocationResult) { r.Replayed = true }})
			return nil
		}
		deny := func(e error) error {
			var f *foundation.Fault
			if !errors.As(e, &f) || f.Status >= 500 {
				return e
			}
			reply.Rejection = e
			u.Confirm()
			return nil
		}
		readFields, e := readable(ctx, u, cmd.Action != "CREATE")
		if e != nil {
			return deny(e)
		}
		if e = requireCommand(ctx, u, cmd, fields); e != nil {
			return deny(e)
		}
		if e = u.Lock(ctx, terminal.ObjectID); e != nil {
			return e
		}
		before, e := u.Load(ctx, terminal.ObjectID, "")
		if e != nil {
			return e
		}
		if cmd.Action != "CREATE" && before.Timeline.ID == "" {
			return deny(missing())
		}
		reject := func(e error) error {
			var f *foundation.Fault
			if !errors.As(e, &f) || f.Status >= 500 {
				return e
			}
			terminal.Outcome = "rejected"
			terminal.ErrorCode = f.Code
			terminal.Result = org.LocationResult{SubmissionID: terminal.ID, Outcome: "rejected", ErrorCode: ptr(f.Code)}
			if e = u.StoreTerminal(ctx, terminal); e != nil {
				return e
			}
			reply.Result = terminal.Result
			reply.Rejection = f
			u.Confirm()
			return nil
		}
		if cmd.Action == "CREATE" {
			if before.Timeline.ID != "" {
				return reject(fault("M02_CODE_CONFLICT", "code", "来源对象已存在，请使用原来源映射查回"))
			}
			code, e := domain.NormalizeCode(cmd.Create.Code)
			if e != nil {
				return reject(e)
			}
			before.Timeline = d.Timeline{ID: terminal.ObjectID, Enterprise: u.EnterpriseScope().ID, Code: code, First: foundation.NewID()}
		} else if before.Timeline.Source != expected(cmd) {
			return reject(fault("M02_SOURCE_CONFLICT", "", "来源已变化，请重新核对资料"))
		}
		candidate, e := candidate(ctx, u, before.Timeline, cmd)
		if e != nil {
			return reject(e)
		}
		if candidate.NoChange {
			selected := cmd.RecordID
			if selected == "" {
				selected = cmd.Add.AnchorRecordID
			}
			terminal.Outcome = "no_change"
			terminal.Result = org.LocationResult{SubmissionID: terminal.ID, Outcome: "no_change", LocationID: ptr(terminal.ObjectID), SourceVersion: ptr(before.Timeline.Source), SelectedRecordID: ptr(selected), Action: cmd.Action}
			if e = u.StoreTerminal(ctx, terminal); e != nil {
				return e
			}
			reply.Result = terminal.Result
			u.Confirm()
			return nil
		}
		now, e := u.Now(ctx)
		if e != nil {
			return e
		}
		operator, e := u.Access().OperatorSnapshot(ctx)
		if e != nil {
			return e
		}
		write := assemble(before, candidate, cmd, terminal, now, operator)
		e = u.Business(ctx, func() error { r, e := u.Persist(ctx, write); reply.Result = r; return e })
		if e != nil {
			return reject(e)
		}
		trimResult(&reply.Result, readFields)
		u.Confirm()
		return nil
	})
	if e != nil {
		return org.LocationResult{}, e
	}
	return reply.Result, reply.Rejection
}
func assemble(before State, candidate d.Candidate, cmd org.LocationCommand, terminal Terminal, now time.Time, operator access.OperatorSnapshot) Write {
	t := before.Timeline
	t.Source = foundation.NewID()
	t.SourceNo++
	recordedAt := now.UTC().Truncate(time.Microsecond)
	summary := org.OperationSummary{OperationSourceVersion: t.Source, RecordedAt: recordedAt.Format(time.RFC3339Nano), Operator: org.OperatorSnapshot(operator)}
	history := recordhistory.Assemble[Revision](d.Kernel, before.Timeline.Records, candidate, cmd.RecordID, before.Summaries, summary)
	t.Records = history.Records
	op := Operation{ID: t.Source, Previous: before.Timeline.Source, Subject: operator.SubjectID, Submission: terminal.ID, Action: cmd.Action, EventID: foundation.NewID(), Number: t.SourceNo, RecordedAt: recordedAt, Operator: summary.Operator, Revisions: history.Revisions}
	terminal.Outcome = "accepted"
	terminal.Result = org.LocationResult{
		SubmissionID: terminal.ID, Outcome: terminal.Outcome, LocationID: ptr(t.ID), SourceVersion: ptr(t.Source),
		SelectedRecordID: ptr(history.Selected), TargetRecordID: ptr(history.Target.ID), TargetRevision: ptr(history.Target.Revision),
		EffectiveDate: ptr(history.Target.Date), Action: cmd.Action, RecordedAt: ptr(summary.RecordedAt),
	}
	op.Terminal = terminal
	return Write{Before: before.Timeline, After: State{Timeline: t, Summaries: history.Summaries}, Operation: op, authorized: true}
}
func (s Service) Resolve(ctx context.Context, c access.Credentials, id string) (org.LocationResult, error) {
	if !foundation.ValidID(id) {
		return org.LocationResult{}, fault("M02_INVALID_INPUT", "", "提交标识无效")
	}
	reply, e := submission.Run(ctx, func(ctx context.Context, fn func(Unit) error) error { return s.Store.WithinLocationWrite(ctx, c, fn) }, func(u Unit, reply *submission.Reply[org.LocationResult]) error {
		var e error
		*reply, e = submission.Resolve[org.LocationResult, Terminal](ctx, u, id, submission.ResolvePolicy[org.LocationResult, Terminal]{PrepareRead: func(context.Context) error { return nil }, Read: func(ctx context.Context, v *Saved) error { return readSaved(ctx, u, v) }, Close: func(ctx context.Context) (submission.Closure[org.LocationResult, Terminal], error) {
			var closed submission.Closure[org.LocationResult, Terminal]
			if _, e := require(ctx, u, "read", nil); e != nil {
				closed.Reply.Rejection = e
				return closed, nil
			}
			result := org.LocationResult{SubmissionID: id, Outcome: "not_applied", ErrorCode: ptr("M02_REQUEST_CLOSED")}
			closed.Terminal = Terminal{ID: id, Outcome: "not_applied", Result: result}
			closed.Reply.Result = result
			return closed, nil
		}})
		return e
	})
	if e != nil {
		return org.LocationResult{}, e
	}
	return reply.Result, reply.Rejection
}

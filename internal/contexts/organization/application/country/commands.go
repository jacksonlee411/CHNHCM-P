package country

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/application/submission"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/country"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"errors"
	"slices"
	"time"
)

func ptr(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}
func (s Service) Maintain(ctx context.Context, c access.Credentials, cmd org.CountryCommand) (org.CountryResult, error) {
	terminal, fields, e := metadata(cmd)
	if e != nil {
		return org.CountryResult{}, e
	}
	reply, e := submission.Run(ctx, func(ctx context.Context, fn func(Unit) error) error { return s.Store.WithinCountryWrite(ctx, c, fn) }, func(u Unit, reply *submission.Reply[org.CountryResult]) error {
		p := u.Access().Principal()
		terminal.Digest = submission.BindDigest(p.InstanceID, p.ID, terminal.Digest)
		saved, e := submission.Lookup[org.CountryResult, Terminal](ctx, u, terminal.ID)
		if e != nil {
			return e
		}
		if saved != nil {
			*reply = submission.Replay(ctx, u, saved, terminal.Digest, submission.ReplayPolicy[org.CountryResult]{Outcome: func(r org.CountryResult) string { return r.Outcome }, Read: func(ctx context.Context, v *Saved) error { return readSaved(ctx, u, v) }, MarkReplayed: func(r *org.CountryResult) { r.Replayed = true }})
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
		readFields, e := readable(ctx, u, true)
		if e != nil {
			return deny(e)
		}
		if _, e = require(ctx, u, action(cmd), fields); e != nil {
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
			terminal.Result = org.CountryResult{SubmissionID: terminal.ID, Outcome: "rejected", ErrorCode: ptr(f.Code)}
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
			before.Timeline = d.Timeline{ID: terminal.ObjectID, First: foundation.NewID(), Origin: origin(cmd.Import.Origin)}
		} else if before.Timeline.Source != expected(cmd) {
			return reject(fault("M02_SOURCE_CONFLICT", "", "来源已变化，请重新核对资料"))
		}
		candidate, e := candidate(ctx, before.Timeline, cmd)
		if e != nil {
			return reject(e)
		}
		if candidate.NoChange {
			selected := cmd.RecordID
			if selected == "" {
				selected = cmd.Add.AnchorRecordID
			}
			terminal.Outcome = "no_change"
			terminal.Result = org.CountryResult{SubmissionID: terminal.ID, Outcome: "no_change", CountryRegionID: ptr(terminal.ObjectID), SourceVersion: ptr(before.Timeline.Source), SelectedRecordID: ptr(selected), Action: cmd.Action}
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
		return org.CountryResult{}, e
	}
	return reply.Result, reply.Rejection
}
func assemble(before State, candidate d.Candidate, cmd org.CountryCommand, terminal Terminal, now time.Time, operator access.OperatorSnapshot) Write {
	t := before.Timeline
	t.Source = foundation.NewID()
	t.SourceNo++
	t.Records = d.Kernel.CloneRecords(candidate.Records)
	op := Operation{ID: t.Source, Previous: before.Timeline.Source, Subject: operator.SubjectID, Submission: terminal.ID, Action: cmd.Action, EventID: foundation.NewID(), Number: t.SourceNo, RecordedAt: now.UTC().Truncate(time.Microsecond), Operator: org.OperatorSnapshot(operator), Revisions: []Revision{}}
	summaries := slices.Clone(before.Summaries)
	selected := cmd.RecordID
	for i := range t.Records {
		r := &t.Records[i]
		changed := cmd.Action == "CREATE" || slices.Contains(candidate.Impact.AffectedRecordIDs, r.ID)
		if !foundation.ValidID(r.ID) {
			r.ID = foundation.NewID()
			selected = r.ID
		}
		if changed {
			r.Revision = foundation.NewID()
			r.RevisionNo++
		}
		if r.Type == "DEACTIVATE" && i > 0 {
			r.Data = (d.Policy{}).Clone(t.Records[i-1].Data)
			r.InheritedRecord = t.Records[i-1].ID
			r.InheritedRevision = t.Records[i-1].Revision
		}
		if changed {
			var previous *d.Record
			for _, v := range before.Timeline.Records {
				if v.ID == r.ID {
					x := v
					previous = &x
					break
				}
			}
			next := *r
			op.Revisions = append(op.Revisions, Revision{Before: previous, After: &next})
			summaries = slices.DeleteFunc(summaries, func(v org.OperationSummary) bool { return v.RecordID == r.ID })
			summaries = append(summaries, org.OperationSummary{RecordID: r.ID, Revision: r.Revision, OperationSourceVersion: t.Source, RecordedAt: op.RecordedAt.Format(time.RFC3339Nano), Operator: op.Operator})
		}
	}
	if cmd.Action == "CREATE" {
		selected = t.First
	}
	terminal.Outcome = "accepted"
	terminal.Result = org.CountryResult{SubmissionID: terminal.ID, Outcome: "accepted", CountryRegionID: ptr(t.ID), SourceVersion: ptr(t.Source), SelectedRecordID: ptr(selected), TargetRecordID: ptr(selected), Action: cmd.Action, RecordedAt: ptr(op.RecordedAt.Format(time.RFC3339Nano))}
	for _, r := range t.Records {
		if r.ID == selected {
			terminal.Result.TargetRevision = ptr(r.Revision)
			terminal.Result.EffectiveDate = ptr(r.Date)
		}
	}
	op.Terminal = terminal
	return Write{Before: before.Timeline, After: State{Timeline: t, Summaries: summaries}, Operation: op, authorized: true}
}
func (s Service) Resolve(ctx context.Context, c access.Credentials, id string) (org.CountryResult, error) {
	if !foundation.ValidID(id) {
		return org.CountryResult{}, fault("M02_INVALID_INPUT", "", "提交标识无效")
	}
	reply, e := submission.Run(ctx, func(ctx context.Context, fn func(Unit) error) error { return s.Store.WithinCountryWrite(ctx, c, fn) }, func(u Unit, reply *submission.Reply[org.CountryResult]) error {
		var e error
		*reply, e = submission.Resolve[org.CountryResult, Terminal](ctx, u, id, submission.ResolvePolicy[org.CountryResult, Terminal]{PrepareRead: func(context.Context) error { return nil }, Read: func(ctx context.Context, v *Saved) error { return readSaved(ctx, u, v) }, Close: func(ctx context.Context) (submission.Closure[org.CountryResult, Terminal], error) {
			var closed submission.Closure[org.CountryResult, Terminal]
			if _, e := require(ctx, u, "read", nil); e != nil {
				closed.Reply.Rejection = e
				return closed, nil
			}
			result := org.CountryResult{SubmissionID: id, Outcome: "not_applied", ErrorCode: ptr("M02_REQUEST_CLOSED")}
			closed.Terminal = Terminal{ID: id, Outcome: "not_applied", Result: result}
			closed.Reply.Result = result
			return closed, nil
		}})
		return e
	})
	if e != nil {
		return org.CountryResult{}, e
	}
	return reply.Result, reply.Rejection
}

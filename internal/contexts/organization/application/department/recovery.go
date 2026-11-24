package department

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/application/submission"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"strings"
	"time"
)

func authorizeSaved(ctx context.Context, u Unit, saved *Saved) error {
	id := saved.RequestObjectID
	if saved.Result.DepartmentID != nil {
		id = *saved.Result.DepartmentID
	}
	if id == "" {
		return u.CompleteResult(ctx, saved)
	}
	now, e := u.Now(ctx)
	if e != nil {
		return e
	}
	read, e := u.Snapshot(ctx, id, "")
	if e != nil {
		return e
	}
	current := read.Snapshot.Department(id)
	if current == nil {
		return missing()
	}
	fields, e := readable(ctx, u, *current, now, true)
	if e != nil {
		return e
	}
	if e = u.CompleteResult(ctx, saved); e != nil {
		return e
	}
	if e = populateResultItems(ctx, u, &saved.Result, fields, now); e != nil {
		return e
	}
	trimResult(&saved.Result, fields)
	if saved.Result.SourceVersion != nil {
		saved.Result.CurrentChanged = *saved.Result.SourceVersion != current.Source
	}
	if saved.Result.ObservedSourceVersion != nil {
		saved.Result.CurrentChanged = *saved.Result.ObservedSourceVersion != current.Source
	}
	// Persisted diagnostics have no replayable hidden object payload. A current
	// result read may conservatively reduce a qualified relation error further.
	if saved.Result.ErrorCode != nil && strings.HasPrefix(*saved.Result.ErrorCode, "M02_DEPARTMENT_") {
		code := "M02_DEPARTMENT_REFERENCE_CONFLICT"
		saved.Result.ErrorCode = &code
		saved.Result.FailureDetails = &org.FailureDetails{FormatVersion: 1, Issues: []org.Issue{{Code: code, Locations: []org.Location{}}}}
	}
	return nil
}
func (s Service) Resolve(ctx context.Context, c access.Credentials, id string) (org.DepartmentResult, error) {
	if !foundation.ValidID(id) {
		return org.DepartmentResult{}, foundation.Fail(400, "M02_INVALID_INPUT", "保存请求无效")
	}
	reply, e := submission.Run(ctx, func(ctx context.Context, fn func(Unit) error) error { return s.Store.WithinDepartmentWrite(ctx, c, fn) }, func(u Unit, r *submission.Reply[org.DepartmentResult]) error {
		var err error
		*r, err = submission.Resolve[org.DepartmentResult, Terminal](ctx, u, id, submission.ResolvePolicy[org.DepartmentResult, Terminal]{PrepareRead: func(context.Context) error { return nil }, Read: func(ctx context.Context, v *Saved) error { return authorizeSaved(ctx, u, v) }, Close: func(ctx context.Context) (submission.Closure[org.DepartmentResult, Terminal], error) {
			var closure submission.Closure[org.DepartmentResult, Terminal]
			_, e := require(ctx, u, access.Intent{Action: "submission.resolve", Fields: []string{}, Kind: "resolve-own"})
			if e != nil {
				return closure, e
			}
			code := "M02_REQUEST_CLOSED"
			closure.Terminal = Terminal{ID: id, Outcome: "not_applied", ErrorCode: code}
			closure.Reply.Result = org.DepartmentResult{SubmissionID: id, Outcome: "not_applied", ErrorCode: &code}
			return closure, nil
		}})
		return err
	})
	if e != nil {
		return org.DepartmentResult{}, e
	}
	return reply.Result, reply.Rejection
}

// Project exact item revisions only after current result read authorization.
// The terminal stores references, never a reusable authorization projection.
func populateResultItems(ctx context.Context, u Unit, result *org.DepartmentResult, fields []string, now time.Time) error {
	if result.Outcome != "accepted" || len(result.Items) == 0 || result.DepartmentID == nil || result.SourceVersion == nil {
		return nil
	}
	read, e := u.Snapshot(ctx, *result.DepartmentID, *result.SourceVersion)
	if e != nil {
		return e
	}
	ent, e := u.Enterprise(ctx, false)
	if e != nil {
		return e
	}
	source, e := u.Source(ctx, *result.DepartmentID, *result.SourceVersion)
	if e != nil {
		return e
	}
	p := newProjector(ctx, u, read.Snapshot, ent, now)
	for i := range result.Items {
		item := &result.Items[i]
		found := false
		for _, change := range source.Items {
			if change.RecordID != item.RecordID {
				continue
			}
			found = true
			if change.Before != nil {
				v, e := p.record(change.Before.Record, fields, &change.Before.Summary)
				if e != nil {
					return e
				}
				item.BeforeRecord = &v
			}
			if change.After != nil && !change.After.Deleted {
				v, e := p.record(change.After.Record, fields, &change.After.Summary)
				if e != nil {
					return e
				}
				item.AfterRecord = &v
			}
		}
		if !found {
			return foundation.Fail(503, "M02_DEPENDENCY_UNAVAILABLE", "提交结果不可用")
		}
	}
	return nil
}

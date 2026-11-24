package application

import (
	"context"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/application/enterprisecheck"
	"chnhcm.local/chnhcm/internal/contexts/organization/application/submission"
	"chnhcm.local/chnhcm/internal/contexts/organization/domain"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

type Saved = submission.Saved[org.Result]
type Unit interface {
	Navigate(context.Context, org.ListInput, org.Enterprise, string, string) (org.NavigationResult, error)
	Access() access.Guard
	Now(context.Context) (time.Time, error)
	Enterprise(context.Context, bool) (org.Enterprise, error)
	submission.Repository[org.Result, Terminal]
	LoadTimeline(context.Context, string, bool) (Timeline, error)
	LockUnchangedBUCoverage(context.Context, Timeline, domain.Candidate) (Timeline, error)
	ValidateReferences(context.Context, Timeline, domain.Candidate) error
	Persist(context.Context, Write) (org.Result, error)
	List(context.Context, org.ListInput, org.Enterprise, string) (org.List, error)
	SelectionList(context.Context, org.ListInput, org.Enterprise, string) (org.List, error)
	Detail(context.Context, org.DetailInput, org.Enterprise, string) (org.Detail, error)
	Changes(context.Context, org.ChangeQuery, org.Enterprise) (org.Changes, error)
}
type Store interface {
	Within(context.Context, access.Credentials, func(Unit) error) error
	// Nil confirms the outer commit; callback results must not escape on error.
	WithinWrite(context.Context, access.Credentials, func(Unit) error) error
}
type Service struct{ Store Store }

func permission() error {
	return foundation.Fail(403, "ACCESS_DENIED", "当前权限不允许此操作")
}
func notFound() error {
	return foundation.Fail(404, "M02_OBJECT_NOT_FOUND", "业务单元不存在或当前不可见")
}
func normalizeDate(v string, now time.Time) (string, error) {
	if v == "" {
		v = now.In(foundation.Shanghai).Format("2006-01-02")
	}
	if !foundation.ValidDate(v) {
		return "", foundation.Fail(400, "M02_INVALID_INPUT", "查看日期无效")
	}
	return v, nil
}
func allowed(grants []access.Grant, action, enterprise, bu string) bool {
	for _, g := range grants {
		if g.Action == action && ((g.ScopeKind == "enterprise" && g.ScopeID == enterprise) || (g.ScopeKind == "bu" && g.ScopeID == bu && bu != "")) {
			return true
		}
	}
	return false
}
func (s Service) Context(ctx context.Context, c access.Credentials) (out org.Context, err error) {
	var rejected error
	err = s.Store.Within(ctx, c, func(u Unit) error {
		e, e1 := u.Enterprise(ctx, false)
		if e1 != nil {
			return e1
		}
		now, e1 := u.Now(ctx)
		if e1 != nil {
			return e1
		}
		g := u.Access()
		hasRead := allowed(g.Grants(), "bu.read", e.ID, "")
		for _, grant := range g.Grants() {
			if grant.Action == "bu.read" && grant.ScopeKind == "bu" {
				hasRead = true
			}
		}
		if !hasRead {
			rejected = permission()
			_, e1 = g.Decide(ctx, access.Intent{Action: "bu.read", EnterpriseID: e.ID, Kind: "context", Fields: []string{}})
			return e1
		}
		read, e1 := g.Decide(ctx, access.Intent{Action: "bu.read", EnterpriseID: e.ID, Kind: "context", Fields: []string{}})
		if e1 != nil {
			return e1
		}
		create, e1 := g.Decide(ctx, access.Intent{Action: "bu.create", EnterpriseID: e.ID, Kind: "capability", Fields: []string{"code", "name", "effectiveDate", "operationType", "operationReason"}})
		if e1 != nil {
			return e1
		}
		out = org.Context{Enterprise: enterprisecheck.Display(e, now), CurrentDate: now.In(foundation.Shanghai).Format("2006-01-02"), DefaultEffectiveDate: foundation.DefaultEffectiveDate, Timezone: "Asia/Shanghai", NameMaxLength: domain.NameMaxLength, NameGuidance: domain.NameGuidance, CodeMaxLength: 40, Capabilities: org.Capabilities{AddWriteFields: []string{}, CorrectWriteFields: []string{}, DeactivateWriteFields: []string{}, ReactivateWriteFields: []string{}, Create: create.Allowed && allowed(g.Grants(), "bu.read", e.ID, ""), History: allowed(g.Grants(), "bu.history.read", e.ID, ""), ReadFields: read.ReadFields, WriteFields: create.WriteFields}}
		return nil
	})
	if err == nil {
		err = rejected
	}
	return
}

func (s Service) Resolve(ctx context.Context, c access.Credentials, id string) (out org.Result, err error) {
	if !foundation.ValidID(id) {
		return out, foundation.Fail(400, "M02_INVALID_INPUT", "保存请求无效")
	}
	reply, err := submission.Run(ctx, func(ctx context.Context, fn func(Unit) error) error {
		return s.Store.WithinWrite(ctx, c, fn)
	}, func(u Unit, reply *submission.Reply[org.Result]) error {
		var enterprise string
		var e error
		*reply, e = submission.Resolve[org.Result, Terminal](ctx, u, id, submission.ResolvePolicy[org.Result, Terminal]{
			PrepareRead: func(ctx context.Context) error {
				ent, e := u.Enterprise(ctx, false)
				enterprise = ent.ID
				return e
			},
			Read: func(ctx context.Context, saved *Saved) error { return authorizeSaved(ctx, u, saved, enterprise) },
			Close: func(ctx context.Context) (submission.Closure[org.Result, Terminal], error) {
				var closure submission.Closure[org.Result, Terminal]
				d, e := u.Access().Decide(ctx, access.Intent{Action: "submission.resolve", Fields: []string{}, Kind: "resolve-own"})
				if e != nil {
					return closure, e
				}
				if !d.Allowed {
					closure.Reply.Rejection = permission()
					return closure, nil
				}
				code := "M02_REQUEST_CLOSED"
				closure.Terminal = Terminal{ID: id, Outcome: "not_applied", ErrorCode: code}
				closure.Reply.Result = org.Result{SubmissionID: id, Outcome: "not_applied", ErrorCode: &code}
				return closure, nil
			},
		})
		return e
	})
	if err != nil {
		return org.Result{}, err
	}
	out, err = reply.Result, reply.Rejection
	return
}

// An old accepted result is historical information; every recovery path checks current rights.
func authorizeResult(ctx context.Context, u Unit, result *org.Result, enterprise string) error {
	intent := access.Intent{Action: "bu.read", EnterpriseID: enterprise, BUID: *result.BUID, Kind: "result", Fields: []string{}}
	history := intent
	history.Action = "bu.history.read"
	fields, err := submission.AuthorizeHistory(ctx, u.Access(), submission.ReadRequirement{Intent: intent, Denied: notFound()}, submission.ReadRequirement{Intent: history, Denied: permission()})
	if err != nil {
		return err
	}
	result.CurrentReadFields = fields.Object
	TrimResult(result, fields.History)
	return nil
}

func (s Service) List(ctx context.Context, c access.Credentials, in org.ListInput) (out org.List, err error) {
	if err = normalizeList(&in); err != nil {
		return out, err
	}
	err = s.Store.Within(ctx, c, func(u Unit) error {
		e, e1 := u.Enterprise(ctx, false)
		if e1 != nil {
			return e1
		}
		now, e1 := u.Now(ctx)
		if e1 != nil {
			return e1
		}
		in.BusinessAt, e1 = normalizeDate(in.BusinessAt, now)
		if e1 != nil {
			return e1
		}
		out, e1 = u.List(ctx, in, e, now.In(foundation.Shanghai).Format("2006-01-02"))
		return e1
	})
	return
}
func normalizeList(in *org.ListInput) error {
	if in.Page < 1 {
		in.Page = 1
	}
	if in.PageSize == 0 {
		in.PageSize = 20
	}
	if !slices.Contains([]int{20, 50, 100}, in.PageSize) {
		return foundation.Fail(400, "M02_INVALID_INPUT", "页大小无效")
	}
	if in.Sort == "" {
		in.Sort = "code"
	}
	if !slices.Contains([]string{"code", "name", "name_desc"}, in.Sort) || !slices.Contains([]string{"", "all", "active", "inactive", "not_effective"}, in.Status) || utf8.RuneCountInString(in.Q) > 400 {
		return foundation.Fail(400, "M02_INVALID_INPUT", "筛选条件无效")
	}
	if in.Filter != nil {
		filter := *in.Filter
		valid := false
		switch filter.Field {
		case "code", "name":
			filter.Value = strings.ToLower(strings.TrimSpace(filter.Value))
			valid = slices.Contains([]string{"contains", "equals"}, filter.Operator) && filter.Value != "" && utf8.RuneCountInString(filter.Value) <= 400
		case "status":
			values := strings.Split(filter.Value, ",")
			valid = (filter.Operator == "is" && len(values) == 1 || filter.Operator == "isAnyOf" && len(values) >= 1 && len(values) <= 3)
			seen := map[string]bool{}
			for _, value := range values {
				valid = valid && !seen[value] && slices.Contains([]string{"active", "inactive", "not_effective"}, value)
				seen[value] = true
			}
		case "effectiveDate":
			valid = foundation.ValidDate(filter.Value) && slices.Contains([]string{"is", "before", "after", "onOrBefore", "onOrAfter"}, filter.Operator)
		}
		if !valid {
			return foundation.Fail(400, "M02_INVALID_INPUT", "列筛选条件无效")
		}
		in.Filter = &filter
	}
	in.Q = strings.ToLower(strings.TrimSpace(in.Q))
	if in.Status == "" {
		in.Status = "all"
	}
	if in.Filter != nil && in.Filter.Field == "status" {
		values := strings.Split(in.Filter.Value, ",")
		slices.Sort(values)
		in.Filter.Value = strings.Join(values, ",")
		if len(values) == 1 {
			in.Filter.Operator = "is"
		}
	}
	return nil
}
func (s Service) Detail(ctx context.Context, c access.Credentials, in org.DetailInput) (out org.Detail, err error) {
	if !foundation.ValidID(in.ID) {
		return out, notFound()
	}
	if in.SourceVersion != "" && !foundation.ValidID(in.SourceVersion) {
		return out, foundation.Fail(400, "M02_INVALID_INPUT", "所选历史资料无效")
	}
	if in.SelectedRecordID != "" && !foundation.ValidID(in.SelectedRecordID) {
		return out, foundation.Fail(400, "M02_INVALID_INPUT", "所选记录无效")
	}
	err = s.Store.Within(ctx, c, func(u Unit) error {
		e, e1 := u.Enterprise(ctx, false)
		if e1 != nil {
			return e1
		}
		now, e1 := u.Now(ctx)
		if e1 != nil {
			return e1
		}
		in.BusinessAt, e1 = normalizeDate(in.BusinessAt, now)
		if e1 != nil {
			return e1
		}
		out, e1 = u.Detail(ctx, in, enterprisecheck.Display(e, now), now.In(foundation.Shanghai).Format("2006-01-02"))
		return e1
	})
	return
}
func (s Service) Changes(ctx context.Context, c access.Credentials, in org.ChangeQuery) (out org.Changes, err error) {
	if !foundation.ValidID(in.ID) {
		return out, notFound()
	}
	if err = NormalizeChangeQuery(&in); err != nil {
		return out, err
	}
	err = s.Store.Within(ctx, c, func(u Unit) error {
		ent, e := u.Enterprise(ctx, false)
		if e != nil {
			return e
		}
		out, e = u.Changes(ctx, in, ent)
		return e
	})
	return
}

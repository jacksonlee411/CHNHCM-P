package geoarea

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/application/submission"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"slices"
)

func denied() error { return foundation.Fail(403, "ACCESS_DENIED", "当前权限不允许此操作") }
func missing() error {
	return foundation.Fail(404, "M02_OBJECT_NOT_FOUND", "行政区划与地址分区不存在或当前不可见")
}

// The catalog scope is the authenticated instance. Callers cannot supply a
// foreign instance, enterprise or BU to broaden this target.
func intent(action string, fields []string) access.Intent {
	return access.Intent{Action: "geoarea." + action, Kind: "geoarea-catalog", Fields: fields}
}
func require(ctx context.Context, u Unit, action string, fields []string) (access.Decision, error) {
	r, e := u.Access().Decide(ctx, intent(action, fields))
	if e != nil {
		return r, e
	}
	if !r.Allowed {
		return r, denied()
	}
	return r, nil
}
func readable(ctx context.Context, u Unit, history bool) ([]string, error) {
	if history {
		r, e := submission.AuthorizeHistory(ctx, u.Access(), submission.ReadRequirement{Intent: intent("read", nil), Denied: missing()}, submission.ReadRequirement{Intent: intent("history.read", nil), Denied: denied()})
		return r.History, e
	}
	r, e := require(ctx, u, "read", nil)
	if e != nil {
		return nil, e
	}
	return r.ReadFields, nil
}
func readSaved(ctx context.Context, u Unit, saved *Saved) error {
	fields, e := readable(ctx, u, saved.RequestObjectID != "")
	if e != nil {
		return e
	}
	if saved.RequestObjectID != "" {
		ok, e := u.Published(ctx, saved.RequestObjectID)
		if e != nil {
			return e
		}
		if !ok {
			saved.Result = org.GeoAreaResult{SubmissionID: saved.Result.SubmissionID, Outcome: saved.Result.Outcome, ErrorCode: saved.Result.ErrorCode}
			return nil
		}
	}
	trimResult(&saved.Result, fields)
	if saved.Result.SourceVersion != nil && saved.RequestObjectID != "" {
		current, e := u.Load(ctx, saved.RequestObjectID, "")
		if e != nil {
			return e
		}
		saved.Result.CurrentChanged = current.Timeline.Source != *saved.Result.SourceVersion
	}
	return nil
}
func action(cmd org.GeoAreaCommand) string {
	switch cmd.Action {
	case "CREATE":
		return "create"
	case "ADD":
		return "record.add"
	case "CORRECT":
		return "record.correct"
	case "DEACTIVATE":
		return "deactivate"
	case "REACTIVATE":
		return "reactivate"
	default:
		return "unsupported"
	}
}

func trimResult(result *org.GeoAreaResult, fields []string) {
	if !slices.Contains(fields, "effectiveDate") {
		result.EffectiveDate = nil
	}
}

func published(ctx context.Context, u Unit, id string) error {
	ok, e := u.Published(ctx, id)
	if e != nil {
		return e
	}
	if !ok {
		return missing()
	}
	return nil
}

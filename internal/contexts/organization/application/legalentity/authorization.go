package legalentity

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
	return foundation.Fail(404, "M02_OBJECT_NOT_FOUND", "法律实体不存在或当前不可见")
}

// Enterprise scope comes from the authenticated unit, never from request input.
func intent(u Unit, action string, fields []string) access.Intent {
	return access.Intent{Action: "legalentity." + action, Kind: "legalentity", EnterpriseID: u.EnterpriseScope().ID, Fields: fields}
}
func require(ctx context.Context, u Unit, action string, fields []string) (access.Decision, error) {
	r, e := u.Access().Decide(ctx, intent(u, action, fields))
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
		r, e := submission.AuthorizeHistory(ctx, u.Access(), submission.ReadRequirement{Intent: intent(u, "read", nil), Denied: missing()}, submission.ReadRequirement{Intent: intent(u, "history.read", nil), Denied: denied()})
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
func action(cmd org.LegalEntityCommand) string {
	switch cmd.Action {
	case "CREATE":
		return "create"
	case "ADD":
		return "record.add"
	case "CORRECT":
		return "record.correct"
	case "DELETE":
		return "record.delete"
	case "REDATE":
		return "record.redate"
	case "INSERT_LIFECYCLE_PAIR", "DELETE_LIFECYCLE_PAIR", "MAINTAIN_LIFECYCLE_PAIR":
		return "lifecycle.pair"
	case "DEACTIVATE":
		return "deactivate"
	case "REACTIVATE":
		return "reactivate"
	default:
		return "unsupported"
	}
}

func trimResult(result *org.LegalEntityResult, fields []string) {
	if !slices.Contains(fields, "effectiveDate") {
		result.EffectiveDate = nil
	}
}

// A combined correction preserves the independent date-maintenance grant.
func requireCommand(ctx context.Context, u Unit, cmd org.LegalEntityCommand, fields []string) error {
	if cmd.Action != "CORRECT" {
		_, e := require(ctx, u, action(cmd), fields)
		return e
	}
	correction := slices.DeleteFunc(slices.Clone(fields), func(f string) bool { return f == "effectiveDate" })
	if len(correction) > 0 || cmd.Correct.NewEffectiveDate == "" {
		if _, e := require(ctx, u, "record.correct", correction); e != nil {
			return e
		}
	}
	if cmd.Correct.NewEffectiveDate != "" {
		_, e := require(ctx, u, "record.redate", []string{"effectiveDate"})
		return e
	}
	return nil
}

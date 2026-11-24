package enterprise

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/application/submission"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/enterprise"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"slices"
	"strings"
)

func denied() error { return foundation.Fail(403, "ACCESS_DENIED", "当前权限不允许此操作") }
func missing() error {
	return foundation.Fail(404, "M02_OBJECT_NOT_FOUND", "企业不存在或当前不可见")
}

// The catalog scope is the authenticated instance. Callers cannot supply a
// foreign instance, enterprise or BU to broaden this target.
func intent(action string, fields []string) access.Intent {
	return access.Intent{Action: "enterprise." + action, Kind: "enterprise-maintenance", Fields: fields}
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
func action(cmd org.EnterpriseCommand) string {
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

func trimResult(result *org.EnterpriseResult, fields []string) {
	if !slices.Contains(fields, "effectiveDate") {
		result.EffectiveDate = nil
	}
}

func categoryAction(kind, operation string) string {
	category := "record"
	switch kind {
	case "CREATE":
		category = "first"
	case "DEACTIVATE":
		category = "deactivation"
	case "REACTIVATE":
		category = "reactivation"
	}
	return category + "." + operation
}
func authorizeCommand(ctx context.Context, u Unit, t d.Timeline, c org.EnterpriseCommand, fields []string) error {
	check := func(a string, f []string) error { _, e := require(ctx, u, a, f); return e }
	kind := "CHANGE"
	for _, r := range t.Records {
		if r.ID == c.RecordID {
			kind = r.Type
		}
	}
	switch c.Action {
	case "CORRECT":
		if c.Correct.Changes != nil {
			if e := check("record.correct", fields); e != nil {
				return e
			}
		}
		final := kind
		if c.Correct.Classification != nil {
			final = c.Correct.Classification.OperationType
		}
		for _, k := range slices.Compact([]string{kind, final}) {
			if c.Correct.NewEffectiveDate != "" {
				if e := check(categoryAction(k, "redate"), []string{"effectiveDate"}); e != nil {
					return e
				}
			}
			if c.Correct.Classification != nil {
				if e := check(categoryAction(k, "reclassify"), []string{"operationType", "operationReason"}); e != nil {
					return e
				}
			}
		}
		return nil
	case "DELETE":
		return check(categoryAction(kind, "delete"), nil)
	case "INSERT_LIFECYCLE_PAIR", "DELETE_LIFECYCLE_PAIR":
		for _, k := range []string{"DEACTIVATE", "REACTIVATE"} {
			a := strings.ToLower(k)
			f := []string{"effectiveDate", "operationType", "operationReason"}
			if c.Action == "DELETE_LIFECYCLE_PAIR" {
				a = categoryAction(k, "delete")
				f = nil
			}
			if e := check(a, f); e != nil {
				return e
			}
		}
		return nil
	default:
		return check(action(c), fields)
	}
}

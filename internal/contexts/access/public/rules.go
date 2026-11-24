package public

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"slices"
	"strings"
)

// A configuration record is disclosed as a unit under its dedicated rule grant.
func RuleAllowed(ctx context.Context, g Guard, enterprise, action string) (bool, error) {
	in := Intent{Action: action, Kind: "rule", EnterpriseID: enterprise}
	if strings.HasSuffix(action, ".maintain") {
		in.Fields = []string{"rule"}
	}
	d, err := g.Decide(ctx, in)
	return d.Allowed && slices.Contains(d.ReadFields, "rule"), err
}
func RequireRule(ctx context.Context, g Guard, enterprise, action string) error {
	ok, err := RuleAllowed(ctx, g, enterprise, action)
	if err != nil {
		return err
	}
	if !ok {
		return foundation.Fail(403, "ACCESS_DENIED", "当前权限不允许此操作")
	}
	return nil
}

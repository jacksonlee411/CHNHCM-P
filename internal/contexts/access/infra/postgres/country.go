package postgres

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"github.com/jackc/pgx/v5"
	"strings"
)

func (g *guard) matchesGrant(grant access.Grant, in access.Intent) bool {
	if !strings.HasPrefix(in.Action, "country.") && !strings.HasPrefix(in.Action, "geoarea.") && !strings.HasPrefix(in.Action, "enterprise.") {
		return matchesValidatedIntent(grant, in)
	}
	return (in.Kind == "enterprise-maintenance" && strings.HasPrefix(in.Action, "enterprise.") || in.Kind == "country-catalog" && strings.HasPrefix(in.Action, "country.") || in.Kind == "geoarea-catalog" && strings.HasPrefix(in.Action, "geoarea.")) && in.EnterpriseID == "" && in.BUID == "" && in.DepartmentTarget == nil && in.AffectedScope == nil && grant.Action == in.Action && grant.ScopeKind == "instance" && grant.ScopeID == g.p.InstanceID
}

// The development initializer explicitly grants source creation to writable
// accounts. CLI execution itself never supplies an authorization bypass.
func initializeCountryGrants(ctx context.Context, tx pgx.Tx, instance, subject string, readOnly bool) error {
	fields := []string{"code", "name", "nameEn", "phoneCode", "state", "effectiveDate", "operationType", "operationReason"}
	actions := []string{"country.read", "country.history.read"}
	if !readOnly {
		actions = append(actions, "country.create", "country.record.add", "country.record.correct", "country.deactivate", "country.reactivate")
	}
	changed := false
	for _, action := range actions {
		writes := []string{}
		switch action {
		case "country.create":
			writes = []string{"code", "name", "nameEn", "phoneCode", "effectiveDate", "operationType", "operationReason"}
		case "country.record.correct":
			writes = []string{"name", "nameEn", "phoneCode"}
		case "country.record.add", "country.reactivate":
			writes = []string{"name", "nameEn", "phoneCode", "effectiveDate", "operationType", "operationReason"}
		case "country.deactivate":
			writes = []string{"effectiveDate", "operationType", "operationReason"}
		}
		tag, e := tx.Exec(ctx, `INSERT INTO access.subject_grant VALUES($1,$2,$3,'instance',$1,$4,$5,$6) ON CONFLICT(instance_id,principal_id,scope_kind,scope_id,action) DO NOTHING`, instance, foundation.NewID(), subject, action, fields, writes)
		if e != nil {
			return e
		}
		changed = changed || tag.RowsAffected() > 0
	}
	if changed {
		_, e := tx.Exec(ctx, `UPDATE access.policy_guard SET policy_no=policy_no+1 WHERE instance_id=$1 AND principal_id=$2`, instance, subject)
		return e
	}
	return nil
}

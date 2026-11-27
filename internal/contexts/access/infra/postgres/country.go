package postgres

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"

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
func initializeCountryGrants(batch *pgx.Batch, instance, subject string, readOnly bool) {
	fields := []string{"code", "name", "nameEn", "phoneCode", "state", "effectiveDate", "operationType", "operationReason"}
	actions := []string{"country.read", "country.history.read"}
	if !readOnly {
		actions = append(actions, "country.create", "country.record.add", "country.record.correct", "country.deactivate", "country.reactivate")
	}
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
		queueDevelopmentGrant(batch, instance, subject, "instance", instance, action, fields, writes)
	}
}

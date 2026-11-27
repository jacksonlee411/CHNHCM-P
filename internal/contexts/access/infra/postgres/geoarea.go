package postgres

import (
	"github.com/jackc/pgx/v5"
)

// The development initializer explicitly grants source creation to writable
// accounts. CLI execution itself never supplies an authorization bypass.
func initializeGeoAreaGrants(batch *pgx.Batch, instance, subject string, readOnly bool) {
	fields := []string{"code", "name", "nameEn", "scheme", "nature", "level", "countryRegionId", "parentAreaId", "geoAreaPath", "state", "effectiveDate", "operationType", "operationReason"}
	actions := []string{"geoarea.read", "geoarea.history.read"}
	if !readOnly {
		actions = append(actions, "geoarea.create", "geoarea.record.add", "geoarea.record.correct", "geoarea.deactivate", "geoarea.reactivate")
	}
	for _, action := range actions {
		writes := []string{}
		switch action {
		case "geoarea.create":
			writes = []string{"code", "name", "nameEn", "scheme", "nature", "level", "countryRegionId", "parentAreaId", "effectiveDate", "operationType", "operationReason"}
		case "geoarea.record.correct":
			writes = []string{"name", "nameEn"}
		case "geoarea.record.add", "geoarea.reactivate":
			writes = []string{"name", "nameEn", "effectiveDate", "operationType", "operationReason"}
		case "geoarea.deactivate":
			writes = []string{"effectiveDate", "operationType", "operationReason"}
		}
		queueDevelopmentGrant(batch, instance, subject, "instance", instance, action, fields, writes)
	}
}

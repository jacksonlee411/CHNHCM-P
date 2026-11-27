package postgres

import (
	"github.com/jackc/pgx/v5"
	"strings"
)

func initializeLocationGrants(batch *pgx.Batch, instance, enterprise, subject string, readOnly bool) {
	fields := []string{"code", "name", "nameEn", "countryRegionId", "geoAreaId", "geoAreaPath", "addressLine", "postalCode", "status", "effectiveDate", "operationType", "operationReason"}
	writable := []string{"name", "nameEn", "countryRegionId", "geoAreaId", "addressLine", "postalCode", "effectiveDate", "operationType", "operationReason"}
	actions := []string{"location.read", "location.history.read", "location.export"}
	if !readOnly {
		actions = append(actions, "location.create", "location.record.add", "location.record.correct", "location.record.delete", "location.record.redate", "location.deactivate", "location.reactivate", "location.lifecycle.pair")
	}
	for _, action := range actions {
		writes := []string{}
		if !strings.HasSuffix(action, ".read") && action != "location.export" {
			writes = writable
		}
		if action == "location.create" {
			writes = append([]string{"code"}, writable...)
		}
		queueDevelopmentGrant(batch, instance, subject, "enterprise", enterprise, action, fields, writes)
	}
}

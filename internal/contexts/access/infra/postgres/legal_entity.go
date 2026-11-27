package postgres

import (
	"github.com/jackc/pgx/v5"
	"strings"
)

func initializeLegalEntityGrants(batch *pgx.Batch, instance, enterprise, subject string, readOnly bool) {
	fields := []string{"code", "name", "nameEn", "countryRegionId", "subjectType", "registrationType", "registrationNumber", "establishingOrganization", "status", "effectiveDate", "operationType", "operationReason"}
	writable := []string{"name", "nameEn", "countryRegionId", "subjectType", "registrationType", "registrationNumber", "establishingOrganization", "effectiveDate", "operationType", "operationReason"}
	actions := []string{"legalentity.read", "legalentity.history.read", "legalentity.export"}
	if !readOnly {
		actions = append(actions, "legalentity.create", "legalentity.record.add", "legalentity.record.correct", "legalentity.record.delete", "legalentity.record.redate", "legalentity.deactivate", "legalentity.reactivate", "legalentity.lifecycle.pair")
	}
	for _, action := range actions {
		writes := []string{}
		if !strings.HasSuffix(action, ".read") && action != "legalentity.export" {
			writes = writable
		}
		if action == "legalentity.create" {
			writes = append([]string{"code"}, writable...)
		}
		queueDevelopmentGrant(batch, instance, subject, "enterprise", enterprise, action, fields, writes)
	}
}

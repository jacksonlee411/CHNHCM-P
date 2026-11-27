package postgres

import (
	"github.com/jackc/pgx/v5"
	"strings"
)

func initializeEnterpriseGrants(batch *pgx.Batch, instance, subject string, readOnly bool) {
	fields := []string{"code", "name", "nameEn", "state", "effectiveDate", "operationType", "operationReason"}
	actions := []string{"read", "history.read"}
	if !readOnly {
		actions = append(actions, "record.add", "record.correct", "record.delete", "deactivation.delete", "reactivation.delete", "record.redate", "first.redate", "deactivation.redate", "reactivation.redate", "deactivate", "reactivate", "first.reclassify", "record.reclassify", "deactivation.reclassify", "reactivation.reclassify")
	}
	for _, action := range actions {
		writes := []string{}
		switch {
		case action == "record.correct":
			writes = []string{"name", "nameEn"}
		case strings.HasSuffix(action, ".redate"):
			writes = []string{"effectiveDate"}
		case strings.HasSuffix(action, ".reclassify"):
			writes = []string{"operationType", "operationReason"}
		case action == "record.add" || action == "reactivate":
			writes = []string{"name", "nameEn", "effectiveDate", "operationType", "operationReason"}
		case action == "deactivate":
			writes = []string{"effectiveDate", "operationType", "operationReason"}
		}
		queueDevelopmentGrant(batch, instance, subject, "instance", instance, "enterprise."+action, fields, writes)
	}
}

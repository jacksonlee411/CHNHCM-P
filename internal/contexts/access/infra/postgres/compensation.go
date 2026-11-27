package postgres

import (
	"github.com/jackc/pgx/v5"
	"slices"
)

func initializeGradeRateGrants(batch *pgx.Batch, instance, enterprise, subject string, reader bool) {
	identity := []string{"gradeId", "currency", "frequency", "unit", "precision"}
	values := []string{"form", "amount", "minimum", "guidance", "maximum", "effectiveDate", "reason"}
	read := append(append(slices.Clone(identity), values...), "status", "operator")
	for _, action := range []string{"read", "reference", "history.read"} {
		queueDevelopmentGrant(batch, instance, subject, "enterprise", enterprise, "graderate."+action, read, []string{})
	}
	if reader {
		return
	}
	for _, action := range []string{"create", "add", "correct", "delete", "deactivate", "reactivate"} {
		write := values
		if action == "create" {
			write = append(slices.Clone(identity), values...)
		}
		queueDevelopmentGrant(batch, instance, subject, "enterprise", enterprise, "graderate."+action, read, write)
	}
}

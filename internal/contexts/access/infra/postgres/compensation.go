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

func initializeCompensationPlanGrants(batch *pgx.Batch, instance, enterprise, subject string, reader bool) {
	write := []string{"name", "description", "scope", "components", "amount", "rates", "percentage", "effectiveDate", "reason"}
	read := append(slices.Clone(write), "code", "status", "operator")
	for _, act := range []string{"read", "history.read"} {
		queueDevelopmentGrant(batch, instance, subject, "enterprise", enterprise, "compplan."+act, read, []string{})
	}
	if reader {
		return
	}
	for _, act := range []string{"create", "add", "correct", "delete", "deactivate", "reactivate"} {
		fields := write
		if act == "create" {
			fields = append(slices.Clone(fields), "code")
		}
		queueDevelopmentGrant(batch, instance, subject, "enterprise", enterprise, "compplan."+act, read, fields)
	}
}

package postgres

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"github.com/jackc/pgx/v5"
	"strings"
)

func initializeLegalEntityGrants(ctx context.Context, tx pgx.Tx, instance, enterprise, subject string, readOnly bool) error {
	fields := []string{"code", "name", "nameEn", "countryRegionId", "subjectType", "registrationType", "registrationNumber", "establishingOrganization", "status", "effectiveDate", "operationType", "operationReason"}
	writable := []string{"name", "nameEn", "countryRegionId", "subjectType", "registrationType", "registrationNumber", "establishingOrganization", "effectiveDate", "operationType", "operationReason"}
	actions := []string{"legalentity.read", "legalentity.history.read", "legalentity.export"}
	if !readOnly {
		actions = append(actions, "legalentity.create", "legalentity.record.add", "legalentity.record.correct", "legalentity.record.delete", "legalentity.record.redate", "legalentity.deactivate", "legalentity.reactivate", "legalentity.lifecycle.pair")
	}
	changed := false
	for _, action := range actions {
		writes := []string{}
		if !strings.HasSuffix(action, ".read") && action != "legalentity.export" {
			writes = writable
		}
		if action == "legalentity.create" {
			writes = append([]string{"code"}, writable...)
		}
		tag, e := tx.Exec(ctx, `INSERT INTO access.subject_grant VALUES($1,$2,$3,'enterprise',$4,$5,$6,$7) ON CONFLICT(instance_id,principal_id,scope_kind,scope_id,action) DO NOTHING`, instance, foundation.NewID(), subject, enterprise, action, fields, writes)
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

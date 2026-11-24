package postgres

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"github.com/jackc/pgx/v5"
	"strings"
)

func initializeLocationGrants(ctx context.Context, tx pgx.Tx, instance, enterprise, subject string, readOnly bool) error {
	fields := []string{"code", "name", "nameEn", "countryRegionId", "geoAreaId", "geoAreaPath", "addressLine", "postalCode", "status", "effectiveDate", "operationType", "operationReason"}
	writable := []string{"name", "nameEn", "countryRegionId", "geoAreaId", "addressLine", "postalCode", "effectiveDate", "operationType", "operationReason"}
	actions := []string{"location.read", "location.history.read", "location.export"}
	if !readOnly {
		actions = append(actions, "location.create", "location.record.add", "location.record.correct", "location.record.delete", "location.record.redate", "location.deactivate", "location.reactivate", "location.lifecycle.pair")
	}
	changed := false
	for _, action := range actions {
		writes := []string{}
		if !strings.HasSuffix(action, ".read") && action != "location.export" {
			writes = writable
		}
		if action == "location.create" {
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

package postgres

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"github.com/jackc/pgx/v5"
)

// The development initializer explicitly grants source creation to writable
// accounts. CLI execution itself never supplies an authorization bypass.
func initializeGeoAreaGrants(ctx context.Context, tx pgx.Tx, instance, subject string, readOnly bool) error {
	fields := []string{"code", "name", "nameEn", "scheme", "nature", "level", "countryRegionId", "parentAreaId", "geoAreaPath", "state", "effectiveDate", "operationType", "operationReason"}
	actions := []string{"geoarea.read", "geoarea.history.read"}
	if !readOnly {
		actions = append(actions, "geoarea.create", "geoarea.record.add", "geoarea.record.correct", "geoarea.deactivate", "geoarea.reactivate")
	}
	changed := false
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

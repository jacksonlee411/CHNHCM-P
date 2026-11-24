package postgres

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"github.com/jackc/pgx/v5"
	"strings"
)

func initializeEnterpriseGrants(ctx context.Context, tx pgx.Tx, instance, subject string, readOnly bool) error {
	fields := []string{"code", "name", "nameEn", "state", "effectiveDate", "operationType", "operationReason"}
	actions := []string{"read", "history.read"}
	if !readOnly {
		actions = append(actions, "record.add", "record.correct", "record.delete", "deactivation.delete", "reactivation.delete", "record.redate", "first.redate", "deactivation.redate", "reactivation.redate", "deactivate", "reactivate", "first.reclassify", "record.reclassify", "deactivation.reclassify", "reactivation.reclassify")
	}
	changed := false
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
		tag, e := tx.Exec(ctx, `INSERT INTO access.subject_grant VALUES($1,$2,$3,'instance',$1,$4,$5,$6) ON CONFLICT(instance_id,principal_id,scope_kind,scope_id,action) DO NOTHING`, instance, foundation.NewID(), subject, "enterprise."+action, fields, writes)
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

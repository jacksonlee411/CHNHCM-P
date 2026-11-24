package postgres

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"github.com/jackc/pgx/v5"
	"strings"
)

func initializeStaffingGrants(ctx context.Context, tx pgx.Tx, instance, enterprise, subject string, readOnly, newPrincipal bool) error {
	for _, kind := range []string{"jobfamily", "jobfunction", "job", "grade", "gradeladder", "position", "positionhierarchy", "staffingquota"} {
		fields := []string{"code", "name", "nameEn", "status", "effectiveDate", "operationType", "operationReason"}
		if kind == "position" {
			fields = append(fields, "departmentId", "jobId", "locationId", "gradeId")
		} else if kind == "jobfunction" || kind == "jobfamily" || kind == "grade" || kind == "gradeladder" {
			fields = append(fields, "description")
		} else {
			fields = append(fields, "jobFamilyId", "responsibilities", "requirements")
		}
		if kind == "job" {
			fields = append(fields, "jobFunctionId")
		}
		if kind == "job" || kind == "gradeladder" {
			fields = append(fields, "gradeIds")
		}
		actions := []string{"read", "history.read", "export"}
		if !readOnly {
			actions = append(actions, "create", "record.add", "record.correct", "record.delete", "record.redate", "deactivate", "reactivate", "lifecycle.pair")
		}
		if kind == "positionhierarchy" {
			fields = []string{}
			actions = []string{"read", "history.read"}
			if !readOnly {
				actions = append(actions, "publish")
			}
		}
		if kind == "staffingquota" {
			// Re-initializing a subject must preserve its explicitly migrated,
			// narrowed or revoked quota authority.
			if !newPrincipal {
				continue
			}
			fields = []string{"code", "name", "departmentId", "status", "effectiveDate", "quota", "reason", "audit", "usageState", "sourceVersion"}
			actions = []string{"read", "history.read"}
			if !readOnly {
				actions = append(actions, "create", "adjust", "correct", "redate", "delete", "withdraw")
			}
		}
		changed := false
		for _, action := range actions {
			write := []string{}
			if !strings.HasSuffix(action, "read") && action != "export" {
				for _, f := range fields {
					if f != "operator" && f != "status" && (f != "code" || action == "create") {
						write = append(write, f)
					}
				}
			}
			if kind == "staffingquota" && len(write) > 0 {
				write = []string{"effectiveDate", "quota", "reason"}
			}
			tag, e := tx.Exec(ctx, `INSERT INTO access.subject_grant VALUES($1,$2,$3,'enterprise',$4,$5,$6,$7) ON CONFLICT(instance_id,principal_id,scope_kind,scope_id,action) DO NOTHING`, instance, foundation.NewID(), subject, enterprise, kind+"."+action, fields, write)
			if e != nil {
				return e
			}
			changed = changed || tag.RowsAffected() > 0
		}
		if changed {
			if _, e := tx.Exec(ctx, `UPDATE access.policy_guard SET policy_no=policy_no+1 WHERE instance_id=$1 AND principal_id=$2`, instance, subject); e != nil {
				return e
			}
		}
	}
	return nil
}

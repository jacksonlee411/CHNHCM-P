package postgres

import (
	"github.com/jackc/pgx/v5"
	"strings"
)

func initializeStaffingGrants(batch *pgx.Batch, instance, enterprise, subject string, readOnly bool) {
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
			fields = []string{"code", "name", "departmentId", "status", "effectiveDate", "quota", "reason", "audit", "usageState", "sourceVersion"}
			actions = []string{"read", "history.read"}
			if !readOnly {
				actions = append(actions, "create", "adjust", "correct", "redate", "delete", "withdraw")
			}
		}
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
			queueDevelopmentGrant(batch, instance, subject, "enterprise", enterprise, kind+"."+action, fields, write)
		}
	}
}

//go:build integration

package bootstrap_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5/pgconn"
)

func replace133Assignment(assignments, field, value string) string {
	return strings.Replace(assignments, field+"="+field+"||"+field+"[1]", field+"="+value, 1)
}

// The optimization may trust only the already-checked immutable prefix. These
// ordinary application-role statements must still reject damage to that prefix,
// new local references, new foreign roots and the material it references.
func TestHistory133AppendChecksNewReferencesAndProtectsPrefix(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	bu := b.call("POST", "/api/v1/business-units", createBody("APPEND133", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	dep := b.call("POST", "/api/v1/departments", departmentCreateBody("APPEND133_DEP", bu), foundation.NewID(), 201)["departmentId"].(string)
	foreignInstance, foreignEnterprise := foundation.NewID(), foundation.NewID()
	f.exec(`INSERT INTO organization.enterprise VALUES($1,$2,'FOREIGN','foreign','[2020-01-01,)',decode(repeat('00',32),'hex'))`, foreignInstance, foreignEnterprise)
	for _, owner := range []struct{ kind, object string }{{"bu", bu}, {"department", dep}} {
		t.Run(owner.kind, func(t *testing.T) {
			var foreignShape int32
			must(t, f.admin.QueryRow(f.ctx, `INSERT INTO organization.`+owner.kind+`_shape(instance_id,enterprise_id,parts) VALUES($1,$2,ARRAY[decode('0708','hex')]) RETURNING id`, foreignInstance, foreignEnterprise).Scan(&foreignShape))
			appendOne := `source_refs=source_refs||source_refs[1],subject_refs=subject_refs||subject_refs[1],submission_refs=submission_refs||submission_refs[1],states=states||states[1],operations=operations||operations[1],evidence=evidence||evidence[1]`
			for _, tc := range []struct{ name, statement, code string }{
				{"old_root", `UPDATE organization.%s_history SET states[1]=2147483647 WHERE object_id=$1`, "23514"},
				{"old_id", `UPDATE organization.%s_history SET ids[1]=gen_random_uuid() WHERE object_id=$1`, "23514"},
				{"new_source_ref", `UPDATE organization.%s_history SET ` + replace133Assignment(appendOne, "source_refs", "source_refs||0") + ` WHERE object_id=$1`, "23514"},
				{"new_subject_ref", `UPDATE organization.%s_history SET ` + replace133Assignment(appendOne, "subject_refs", "subject_refs||NULL::integer") + ` WHERE object_id=$1`, "23514"},
				{"new_submission_ref", `UPDATE organization.%s_history SET ` + replace133Assignment(appendOne, "submission_refs", "submission_refs||2147483647") + ` WHERE object_id=$1`, "23514"},
				{"missing_state", `UPDATE organization.%s_history SET ` + replace133Assignment(appendOne, "states", "states||2147483647") + ` WHERE object_id=$1`, "23503"},
				{"foreign_operation", `UPDATE organization.%s_history SET ` + replace133Assignment(appendOne, "operations", fmt.Sprintf("operations||%d", foreignShape)) + ` WHERE object_id=$1`, "23503"},
				{"null_evidence", `UPDATE organization.%s_history SET ` + replace133Assignment(appendOne, "evidence", "evidence||NULL::integer") + ` WHERE object_id=$1`, "23514"},
				{"delete_shape", `DELETE FROM organization.%[1]s_shape WHERE id=(SELECT states[1] FROM organization.%[1]s_history WHERE object_id=$1)`, "42501"},
				{"change_shape", `UPDATE organization.%[1]s_shape SET parts=ARRAY[decode('09','hex')] WHERE id=(SELECT states[1] FROM organization.%[1]s_history WHERE object_id=$1)`, "42501"},
			} {
				t.Run(tc.name, func(t *testing.T) {
					tx, err := f.admin.Begin(f.ctx)
					must(t, err)
					defer tx.Rollback(f.ctx)
					_, err = tx.Exec(f.ctx, "SET LOCAL ROLE chnhcm_app")
					must(t, err)
					_, err = tx.Exec(f.ctx, fmt.Sprintf(tc.statement, owner.kind), owner.object)
					var pg *pgconn.PgError
					if !errors.As(err, &pg) || pg.Code != tc.code {
						t.Fatalf("damage must be rejected by native guard: %v", err)
					}
				})
			}
			// Sealing adds no member; all retained data remains exactly unchanged.
			tx, err := f.admin.Begin(f.ctx)
			must(t, err)
			defer tx.Rollback(f.ctx)
			_, err = tx.Exec(f.ctx, `UPDATE organization.`+owner.kind+`_history SET sealed=true WHERE object_id=$1`, owner.object)
			must(t, err)
		})
	}
}

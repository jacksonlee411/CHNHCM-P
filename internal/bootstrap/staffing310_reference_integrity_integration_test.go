//go:build integration

package bootstrap_test

import (
	"errors"
	"fmt"
	"testing"

	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"time"
)

// Structural reference checks must reject missing/cross-object evidence even
// below the Go service. A mixed valid/invalid batch must leave no partial rows.
func TestStaffing310MemberReferenceIntegrity(t *testing.T) {
	h := staffingSetup307(t)
	f := h.f
	family := h.create("jobfamily", "REF-INTEGRITY-F", "")
	a, b := h.create("grade", "REF-INTEGRITY-A", ""), h.create("grade", "REF-INTEGRITY-B", "")
	for _, kind := range []string{"job", "gradeladder"} {
		t.Run(kind, func(t *testing.T) {
			v := create310(h, kind, "REF-INTEGRITY", []string{a["id"].(string)}, family["id"].(string))
			name := "job"
			if kind == "gradeladder" {
				name = "grade_ladder"
			}
			insert := "INSERT INTO staffing." + name + "_grade_member (instance_id,object_id,source_id,record_id,grade_id,position,business_at,adopted_source,adopted_record,adopted_revision) VALUES ($1,$2,$3,$4,$5,2,'2000-01-01',$6,$7,$8)"
			args := func() []any {
				return []any{f.instance, v["id"], v["headSourceVersion"], rec307(v)["recordId"], b["id"], b["headSourceVersion"], rec307(b)["recordId"], rec307(b)["revision"]}
			}
			for _, tc := range []struct {
				name   string
				column int
			}{{"manifest", 2}, {"source", 5}, {"record", 6}, {"revision", 7}, {"grade", 4}, {"instance", 0}} {
				t.Run(tc.name, func(t *testing.T) {
					values := args()
					values[tc.column] = foundation.NewID()
					tx, err := f.admin.Begin(f.ctx)
					must(t, err)
					_, err = tx.Exec(f.ctx, insert, values...)
					if err == nil {
						err = tx.Commit(f.ctx)
					} else {
						_ = tx.Rollback(f.ctx)
					}
					var pg *pgconn.PgError
					if !errors.As(err, &pg) || pg.Code != "23503" {
						t.Fatalf("missing %s reference: %v", tc.name, err)
					}
				})
			}
			t.Run("cross-existing-grade", func(t *testing.T) {
				values := args()
				values[5], values[6], values[7] = a["headSourceVersion"], rec307(a)["recordId"], rec307(a)["revision"]
				_, err := f.admin.Exec(f.ctx, insert, values...)
				var pg *pgconn.PgError
				if !errors.As(err, &pg) || pg.Code != "23503" {
					t.Fatalf("existing evidence belonged to a different Grade: %v", err)
				}
			})
			// Both values have fresh Grade keys/positions. The second has no
			// adopted source; neither row may survive the rejected statement.
			values := append(args(), foundation.NewID())
			_, err := f.admin.Exec(f.ctx, insert+",($1,$2,$3,$4,$9,3,'2000-01-01',$6,$7,$8)", values...)
			var pg *pgconn.PgError
			if !errors.As(err, &pg) || pg.Code != "23503" {
				t.Fatalf("mixed batch: %v", err)
			}
			// COPY runs the same statement-level structural check and rolls
			// back the complete transport batch on a missing adopted reference.
			row := []any{values[0], values[1], values[2], values[3], values[4], 2, pgtype.Date{Time: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true}, values[5], values[6], values[7]}
			bad := append([]any{}, row...)
			bad[4], bad[5] = foundation.NewID(), 3
			tx, e := f.admin.Begin(f.ctx)
			must(t, e)
			_, e = tx.CopyFrom(f.ctx, pgx.Identifier{"staffing", name + "_grade_member"}, []string{"instance_id", "object_id", "source_id", "record_id", "grade_id", "position", "business_at", "adopted_source", "adopted_record", "adopted_revision"}, pgx.CopyFromRows([][]any{row, bad}))
			_ = tx.Rollback(f.ctx)
			if !errors.As(e, &pg) || pg.Code != "23503" {
				t.Fatalf("mixed COPY batch: %v", e)
			}
			if got := f.scalar("SELECT count(*) FROM staffing."+name+"_grade_member WHERE object_id=$1", v["id"]); got != 1 {
				t.Fatalf("partial batch rows=%d", got)
			}
			for _, table := range []string{"grade_source", "grade_revision", name + "_grade_manifest", name + "_grade_member"} {
				for _, statement := range []string{"DELETE FROM staffing.%s WHERE instance_id=$1", "UPDATE staffing.%s SET instance_id=instance_id WHERE instance_id=$1"} {
					_, err = f.admin.Exec(f.ctx, fmt.Sprintf(statement, table), f.instance)
					if !errors.As(err, &pg) || pg.Code != "23514" {
						t.Fatalf("mutable %s: %v", table, err)
					}
				}
			}
		})
	}
}

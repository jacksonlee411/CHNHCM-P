//go:build integration && query131

package bootstrap_test

import (
	"context"
	"errors"
	"testing"

	dep "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	"chnhcm.local/chnhcm/internal/foundation"
)

func TestDepartmentQuery131NecessaryFactsAndUnrelatedStorage(t *testing.T) {
	f, admin, q := query131Fixture(t)
	bu := query131BU(t, admin, "NECESSARY_BU")
	wanted := query131Create(t, admin, "NECESSARY_MATCH", "wanted-row", bu, "2026-01-01")
	unrelated := query131Create(t, admin, "ZZ_UNRELATED", "unrelated-row", bu, "2026-01-01")
	for _, tc := range []struct {
		name, sql string
		required  bool
	}{
		{"unrelated_seal", `UPDATE organization.department_read_segment SET q_seal=decode(repeat('00',32),'hex') WHERE object_id=$1`, false},
		{"unrelated_history_directory", `UPDATE organization.department_read_segment SET ids[1]=gen_random_uuid() WHERE object_id=$1`, false},
		{"unrelated_missing_segments", `DELETE FROM organization.department_read_segment WHERE object_id=$1`, false},
		{"required_source_conflict", `UPDATE organization.department_read_segment SET source_id=gen_random_uuid() WHERE object_id=$1`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx, guard, in := query131Guard(t, f, admin, q)
			defer tx.Rollback(context.Background())
			id := unrelated["departmentId"]
			if tc.required {
				id = wanted["departmentId"]
			}
			_, err := tx.Exec(f.ctx, tc.sql, id)
			must(t, err)
			for _, extent := range []dep.QueryExtent{dep.QueryPage, dep.QuerySelection} {
				in.Query.Q = "wanted-row"
				got, err := history.CaptureDepartmentQuery(f.ctx, tx, guard, dep.QueryRequest{Plan: in.Plan, Input: in.Query, Extent: extent})
				if tc.required {
					var fault *foundation.Fault
					if !errors.As(err, &fault) || fault.Status != 503 || got.Count != 0 || len(got.PageIDs) != 0 || len(got.ByDay) != 0 {
						t.Fatal("required source failure returned partial facts", extent, got.Stats, err)
					}
					continue
				}
				must(t, err)
				if got.Count != 1 || len(got.PageIDs) != 1 || got.PageIDs[0] != wanted["departmentId"] || got.Stats.Objects != 1 || got.Stats.DecodedDepartments != 1 {
					t.Fatal("unrelated storage affected the matching result", extent, got.Stats, got.PageIDs)
				}
				in.Query.Q = "no-match"
				empty, err := history.CaptureDepartmentQuery(f.ctx, tx, guard, dep.QueryRequest{Plan: in.Plan, Input: in.Query, Extent: extent})
				must(t, err)
				if empty.Count != 0 || len(empty.PageIDs) != 0 || empty.Stats.DecodedDepartments != 0 {
					t.Fatal("empty result fetched unrelated source material", extent, empty.Stats)
				}
			}
		})
	}
}

//go:build integration && query131

package bootstrap_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	dep "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Preserve the real MVCC statement and native values, but deliberately deliver
// facts before their dictionaries. SQL row order is not a protocol.
type selection131MaterialTx struct {
	pgx.Tx
	filter string
}

func (tx selection131MaterialTx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	const prefix = "DECLARE selection_facts133 NO SCROLL CURSOR FOR "
	if sql == prefix+history.DepartmentSelectionQuerySQL {
		sql = history.DepartmentSelectionQuerySQL
		if tx.filter == "corrupt inline dictionary" {
			sql = "SELECT kind,item,CASE WHEN kind=3 AND words IS NOT NULL THEN ARRAY['corrupted material']::text[] ELSE words END AS words,binaries,ranges,parts,dictionary_digest,shape_digest,ids FROM (" + sql + ") material ORDER BY kind DESC"
		} else if tx.filter == "BU missing shape" {
			sql = "SELECT kind,item,words,binaries,ranges,CASE WHEN kind=2 THEN NULL::bytea[] ELSE parts END AS parts,dictionary_digest,shape_digest,ids FROM (" + sql + ") material ORDER BY kind DESC"
		} else if tx.filter == "BU conflicting source" {
			sql = "SELECT kind,CASE WHEN kind=2 THEN item[1:5]||ARRAY[gen_random_uuid()::text] ELSE item END AS item,words,binaries,ranges,parts,dictionary_digest,shape_digest,ids FROM (" + sql + ") material ORDER BY kind DESC"
		} else if tx.filter == "BU conflicting identity" {
			sql = "SELECT kind,CASE WHEN kind=2 THEN ARRAY[gen_random_uuid()::text]||item[2:6] ELSE item END AS item,words,binaries,ranges,parts,dictionary_digest,shape_digest,ids FROM (" + sql + ") material ORDER BY kind DESC"
		} else if tx.filter == "conflicting source" {
			sql = "SELECT kind,CASE WHEN kind=3 THEN item[1:6]||ARRAY[gen_random_uuid()::text] ELSE item END AS item,words,binaries,ranges,parts,dictionary_digest,shape_digest,ids FROM (" + sql + ") material ORDER BY kind DESC"
		} else {
			sql = "SELECT * FROM (" + sql + ") material WHERE " + tx.filter + " ORDER BY kind DESC"
		}
		sql = prefix + sql
	}
	return tx.Tx.Exec(ctx, sql, args...)
}

func TestDepartmentSelection131MaterialOrderAndCompleteness(t *testing.T) {
	f, admin, q := query131Fixture(t)
	bu := query131BU(t, admin, "MATERIAL_BU")
	for _, code := range []string{"MATERIAL_A", "MATERIAL_B"} {
		day := "2026-01-01"
		if code == "MATERIAL_A" {
			// Future-first paths capture both D and the future label day from
			// one immutable dictionary, alongside ordinary inline dictionaries.
			day = "2027-01-01"
		}
		query131Create(t, admin, code, "shared words with independent identities", bu, day)
	}
	query131Create(t, admin, "MATERIAL_UNIQUE", "unique material travels with its fact", bu, "2026-01-01")
	for _, filter := range []string{"true", "kind <> -2", "kind <> -1", "kind <> 0", "corrupt inline dictionary", "conflicting source", "BU missing shape", "BU conflicting source", "BU conflicting identity"} {
		t.Run(filter, func(t *testing.T) {
			tx, guard, in := query131Guard(t, f, admin, q)
			defer tx.Rollback(context.Background())
			request := dep.QueryRequest{Plan: in.Plan, Input: in.Query, Extent: dep.QuerySelection}
			want, err := history.CaptureDepartmentQuery(f.ctx, tx, guard, request)
			must(t, err)
			got, err := history.CaptureDepartmentQuery(f.ctx, selection131MaterialTx{tx, filter}, guard, request)
			if filter != "true" {
				var fault *foundation.Fault
				if !errors.As(err, &fault) || fault.Status != 503 || got.Count != 0 || len(got.PageIDs) != 0 || len(got.ByDay) != 0 {
					t.Fatal("incomplete material returned partial capture", filter, got.Stats, err)
				}
				return
			}
			must(t, err)
			if got.Count != 3 || !reflect.DeepEqual(got.PageIDs, want.PageIDs) || got.EnterpriseName != want.EnterpriseName || got.Stats.DecodedDepartments != want.Stats.DecodedDepartments {
				t.Fatal("material order changed the captured set", got.Stats, want.Stats)
			}
			for day, expected := range want.ByDay {
				actual := got.ByDay[day]
				for _, row := range expected.Departments {
					if !reflect.DeepEqual(actual.Department(row.ID), expected.Department(row.ID)) {
						t.Fatal("shared material changed an independent source or identity")
					}
				}
				for _, row := range expected.BusinessUnits {
					if !reflect.DeepEqual(actual.BU(row.ID), expected.BU(row.ID)) {
						t.Fatal("material order changed captured BU facts")
					}
				}
			}
		})
	}
}

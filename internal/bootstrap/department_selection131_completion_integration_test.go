//go:build integration && query131

package bootstrap_test

import (
	dep "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"testing"
)

// Use the real captured MVCC rows, then inject a driver completion error. This
// targets the boundary after rows may already have populated a private capture;
// it does not claim to reproduce a PostgreSQL server failure or timeout.
var selectionCompletionFailure = errors.New("injected selection row completion failure")

type selectionCompletionTx struct {
	pgx.Tx
	rowsRead *int
}

func (tx selectionCompletionTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	rows, err := tx.Tx.Query(ctx, sql, args...)
	if err != nil || sql != "FETCH FORWARD 256 FROM selection_facts133" {
		return rows, err
	}
	return &selectionCompletionRows{Rows: rows, rowsRead: tx.rowsRead}, nil
}

type selectionCompletionRows struct {
	pgx.Rows
	rowsRead *int
}

func (r *selectionCompletionRows) Next() bool {
	next := r.Rows.Next()
	if next {
		*r.rowsRead++
	}
	return next
}
func (r *selectionCompletionRows) Err() error {
	if err := r.Rows.Err(); err != nil {
		return err
	}
	return selectionCompletionFailure
}
func TestDepartmentSelection131CompletionFailureReturnsNoCapture(t *testing.T) {
	f, admin, q := query131Fixture(t)
	bu := query131BU(t, admin, "R117_BU")
	for _, code := range []string{"R117_A", "R117_B", "R117_C"} {
		query131Create(t, admin, code, "completion boundary "+code, bu, "2026-01-01")
	}
	tx, guard, in := query131Guard(t, f, admin, q)
	defer tx.Rollback(context.Background())
	request := dep.QueryRequest{Plan: in.Plan, Input: in.Query, Extent: dep.QuerySelection}
	want, err := history.CaptureDepartmentQuery(f.ctx, tx, guard, request)
	must(t, err)
	if want.Count != 3 || len(want.PageIDs) != 3 {
		t.Fatal("invalid complete control capture", want.Count)
	}
	rowsRead := 0
	got, err := history.CaptureDepartmentQuery(f.ctx, selectionCompletionTx{Tx: tx, rowsRead: &rowsRead}, guard, request)
	if !errors.Is(err, selectionCompletionFailure) || rowsRead < 4 || got.Count != 0 || len(got.PageIDs) != 0 || len(got.ByDay) != 0 || got.EnterpriseName != "" || got.Stats.MaterialBytes != 0 {
		t.Fatal("completion error returned partial capture", rowsRead, got.Stats, err)
	}
}

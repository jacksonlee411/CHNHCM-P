//go:build integration

package bootstrap_test

import (
	"context"
	"errors"
	"testing"

	"chnhcm.local/chnhcm/internal/contexts/organization/application/submission"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres/transaction"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestReadFailure133PreservesSQLCauseAndReleasesTransaction(t *testing.T) {
	f := fresh(t)
	activity := 0
	got := transaction.Run(f.ctx, f.admin, false, func(s *transaction.Scope) error {
		s.BeforeCommit(func(ctx context.Context) error { activity++; _, e := s.Tx().Exec(ctx, "SELECT 1"); return e })
		_, e := s.Tx().Exec(f.ctx, "SET LOCAL statement_timeout='10ms'")
		if e != nil {
			return e
		}
		_, e = s.Tx().Exec(f.ctx, "SELECT pg_sleep(0.1)")
		return submission.RejectRead(e)
	})
	var pgerr *pgconn.PgError
	if !errors.As(got, &pgerr) || pgerr.Code != "57014" || activity != 0 {
		t.Fatalf("cause or activity: %v %d", got, activity)
	}
	denied := foundation.Fail(403, "ACCESS_DENIED", "拒绝读取")
	got = transaction.Run(f.ctx, f.admin, false, func(s *transaction.Scope) error {
		s.BeforeCommit(func(ctx context.Context) error { activity++; _, e := s.Tx().Exec(ctx, "SELECT 1"); return e })
		return submission.RejectRead(denied)
	})
	if !errors.Is(got, denied) || activity != 1 {
		t.Fatalf("healthy denial changed: %v %d", got, activity)
	}
	must(t, f.admin.Ping(f.ctx))
	if f.admin.Stat().AcquiredConns() != 0 {
		t.Fatal("connection retained")
	}
}

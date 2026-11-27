package postgres

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	st "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	Pool       *pgxpool.Pool
	Instance   string
	Guard      func(context.Context, pgx.Tx, foundation.Credentials) (access.Guard, error)
	Enterprise org.EnterpriseFloor
	Grades     st.CompensationGrades
	Clock      func() time.Time
}

type unit struct {
	tx         pgx.Tx
	guard      access.Guard
	enterprise org.Enterprise
	today      string
}

func (s *Store) within(ctx context.Context, credentials foundation.Credentials, write bool, fn func(context.Context, *unit) error) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	opts := pgx.TxOptions{IsoLevel: pgx.ReadCommitted}
	if !write {
		// Touch the session before starting a consistent snapshot. Parallel menu
		// and detail reads must not update the same row in repeatable read.
		if err := pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
			g, e := s.Guard(ctx, tx, credentials)
			if e != nil {
				return e
			}
			return g.RecordSessionActivity(ctx)
		}); err != nil {
			return err
		}
		opts.IsoLevel = pgx.RepeatableRead
	}
	return pgx.BeginTxFunc(ctx, s.Pool, opts, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SET LOCAL lock_timeout='3s'; SET LOCAL statement_timeout='10s'"); err != nil {
			return err
		}
		g, err := s.Guard(ctx, tx, credentials)
		if err != nil {
			return err
		}
		bound := dbtx.With(g.Bind(ctx), tx)
		ent, err := s.Enterprise.StaffingEnterprise(bound, write)
		if err != nil {
			return err
		}
		var now time.Time
		if s.Clock != nil {
			now = s.Clock()
		} else if err := tx.QueryRow(ctx, "SELECT transaction_timestamp()").Scan(&now); err != nil {
			return err
		}
		if err = fn(bound, &unit{tx: tx, guard: g, enterprise: ent, today: now.In(foundation.Shanghai).Format(time.DateOnly)}); err != nil {
			return err
		}
		if write {
			return g.RecordSessionActivity(ctx)
		}
		return g.ValidateUse()
	})
}

func (u *unit) decision(ctx context.Context, action string, fields []string) (access.Decision, error) {
	return u.guard.Decide(ctx, access.Intent{Action: action, EnterpriseID: u.enterprise.ID, Fields: fields, Kind: "object"})
}
func unavailable() error { return foundation.Fail(404, "M09_UNAVAILABLE", "所选资料不可用") }
func denied() error {
	return foundation.Fail(403, "M09_FORBIDDEN", "当前权限不能办理此操作")
}
func (u *unit) require(ctx context.Context, action string, fields []string) (access.Decision, error) {
	d, err := u.decision(ctx, action, fields)
	if err != nil {
		return d, err
	}
	if !d.Allowed {
		return d, denied()
	}
	return d, nil
}
func has(fields []string, field string) bool { return slices.Contains(fields, field) }

func (u *unit) lock(ctx context.Context, key string) error {
	_, err := u.tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,0))", key)
	return err
}
func optional(text string) any {
	if text == "" {
		return nil
	}
	return text
}
func fault(err error) (*foundation.Fault, bool) {
	var f *foundation.Fault
	ok := errors.As(err, &f)
	return f, ok
}

// Object discovery is hidden on a denied read. Session and infrastructure faults retain their meaning.
func (u *unit) requireRateRead(ctx context.Context) (access.Decision, error) {
	result, err := u.decision(ctx, "graderate.read", nil)
	if err == nil && !result.Allowed {
		err = unavailable()
	}
	return result, err
}

func (u *unit) rateWriteGrant(ctx context.Context, action string, fields []string) (access.Decision, error) {
	read, err := u.requireRateRead(ctx)
	if err != nil {
		return read, err
	}
	for _, field := range fields {
		if !has(read.ReadFields, field) {
			return read, denied()
		}
	}
	return u.require(ctx, "graderate."+strings.ToLower(action), fields)
}

// Package transaction owns PostgreSQL transaction mechanics, never object SQL.
package transaction

import (
	"context"
	"errors"
	"time"

	"chnhcm.local/chnhcm/internal/contexts/organization/application/submission"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
)

type Beginner interface {
	Begin(context.Context) (pgx.Tx, error)
}

type Scope struct {
	tx              pgx.Tx
	confirmed       bool
	cleanupDeadline time.Time
	beforeCommit    func(context.Context) error
}

func (s *Scope) Tx() pgx.Tx { return s.tx }
func (s *Scope) Confirm()   { s.confirmed = true }

// BeforeCommit finalizes infrastructure state after business work has stopped.
// It runs in the same transaction and request budget, including committed read
// rejections, but never on a failed or unconfirmed write.
func (s *Scope) BeforeCommit(fn func(context.Context) error) { s.beforeCommit = fn }

func (s *Scope) cleanupContext() (context.Context, context.CancelFunc) {
	if s.cleanupDeadline.IsZero() {
		s.cleanupDeadline = time.Now().Add(2 * time.Second)
	}
	return context.WithDeadline(context.Background(), s.cleanupDeadline)
}

func Run(ctx context.Context, db Beginner, write bool, fn func(*Scope) error) error {
	connected := foundation.BeginStage(ctx, "transaction.acquire")
	tx, err := db.Begin(ctx)
	connected()
	if err != nil {
		return err
	}
	defer foundation.BeginStage(ctx, "transaction.hold")()
	s := &Scope{tx: tx}
	defer func() {
		clean, cancel := s.cleanupContext()
		defer cancel()
		_ = tx.Rollback(clean)
	}()
	if _, err = tx.Exec(ctx, "SET LOCAL jit=off; SET LOCAL lock_timeout='2s'; SET LOCAL statement_timeout='5s'; SET LOCAL idle_in_transaction_session_timeout='12s'"); err != nil {
		return err
	}
	callbackErr := fn(s)
	if callbackErr != nil {
		var readRejection *submission.ReadRejection
		var fault *foundation.Fault
		// Read intent alone cannot make an infrastructure failure committable.
		// Only an explicit business refusal may finish session activity.
		if write || !errors.As(callbackErr, &readRejection) ||
			!errors.As(readRejection.Cause, &fault) || fault.Status < 400 || fault.Status >= 500 {
			return callbackErr
		}
	}
	if write && !s.confirmed {
		return errors.New("write transaction has no explicit terminal/read commit intent")
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if s.beforeCommit != nil {
		activity := foundation.BeginStage(ctx, "access.session_activity")
		err = s.beforeCommit(ctx)
		activity()
		if err != nil {
			return err
		}
	}
	committed := foundation.BeginStage(ctx, "transaction.commit")
	err = tx.Commit(ctx)
	committed()
	if err != nil {
		if !write {
			return foundation.Fail(503, "M02_DEPENDENCY_UNAVAILABLE", "查询暂不可用，请重试")
		}
		return foundation.Fail(503, "M02_RESULT_PENDING", "提交结果尚未确认，请查看保存结果")
	}
	return callbackErr
}

func (s *Scope) Business(ctx context.Context, fn func(pgx.Tx) error) (err error) {
	tx, err := s.tx.Begin(ctx)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if committed {
			return // A released savepoint must not start a later rollback's budget.
		}
		clean, cancel := s.cleanupContext()
		defer cancel()
		if rollbackErr := tx.Rollback(clean); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			// Never write a terminal if savepoint rollback is uncertain.
			_ = s.tx.Conn().Close(clean)
			err = foundation.Fail(503, "M02_RESULT_PENDING", "暂时无法确认保存结果")
		} else {
			var fault *foundation.Fault
			if errors.As(err, &fault) && fault.Status < 500 && ctx.Err() == nil {
				s.cleanupDeadline = time.Time{}
			}
		}
	}()
	if err = fn(tx); err != nil {
		return err
	}
	err = tx.Commit(ctx)
	committed = err == nil
	return err
}

package transaction

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"chnhcm.local/chnhcm/internal/contexts/organization/application/submission"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type commitBoundaryTx struct {
	pgx.Tx
	commitError        error
	commits, rollbacks int
}

func (t *commitBoundaryTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (t *commitBoundaryTx) Commit(context.Context) error   { t.commits++; return t.commitError }
func (t *commitBoundaryTx) Rollback(context.Context) error { t.rollbacks++; return nil }

type commitBoundaryDB struct{ tx *commitBoundaryTx }

func (b commitBoundaryDB) Begin(context.Context) (pgx.Tx, error) { return b.tx, nil }

func TestReadAndWriteCommitFailuresHaveDifferentRecoverySemantics(t *testing.T) {
	for _, mode := range []string{"read", "denied-read", "confirmed-write"} {
		for _, failure := range []struct {
			name string
			err  error
		}{{"lost-ack", io.ErrUnexpectedEOF}, {"deadline-at-commit", context.DeadlineExceeded}} {
			t.Run(mode+"/"+failure.name, func(t *testing.T) {
				tx := &commitBoundaryTx{commitError: failure.err}
				write := mode == "confirmed-write"
				activity := 0
				err := Run(context.Background(), commitBoundaryDB{tx}, write, func(s *Scope) error {
					s.BeforeCommit(func(context.Context) error { activity++; return nil })
					if write {
						s.Confirm()
					}
					if mode == "denied-read" {
						return submission.RejectRead(foundation.Fail(403, "ACCESS_DENIED", "拒绝读取"))
					}
					return nil
				})
				want := "M02_DEPENDENCY_UNAVAILABLE"
				if write {
					want = "M02_RESULT_PENDING"
				}
				var fault *foundation.Fault
				faultCode, faultStatus := "none", 0
				if errors.As(err, &fault) {
					faultCode, faultStatus = fault.Code, fault.Status
				}
				t.Logf("R37_UNIT_COMMIT mode=%s failure=%s code=%s status=%d activity=%d commits=%d rollbacks=%d", mode, failure.name, faultCode, faultStatus, activity, tx.commits, tx.rollbacks)
				if fault == nil || fault.Status != 503 || fault.Code != want {
					t.Fatalf("%s commit failure: %v", mode, err)
				}
				if !write && (strings.Contains(fault.Message, "提交") || strings.Contains(fault.Message, "查回")) {
					t.Fatal("read failure asks for a nonexistent submission", fault)
				}
				if activity != 1 || tx.commits != 1 || tx.rollbacks != 1 {
					t.Fatal("commit/cleanup or activity boundary changed", activity, tx.commits, tx.rollbacks)
				}
			})
		}
	}
}

func TestSuccessfulCommitKeepsReadRejectionAndWriteConfirmation(t *testing.T) {
	for _, mode := range []string{"read", "denied-read", "confirmed-write"} {
		t.Run(mode, func(t *testing.T) {
			tx := &commitBoundaryTx{}
			var rejection error
			if mode == "denied-read" {
				rejection = submission.RejectRead(foundation.Fail(403, "ACCESS_DENIED", "拒绝读取"))
			}
			got := Run(context.Background(), commitBoundaryDB{tx}, mode == "confirmed-write", func(s *Scope) error {
				if mode == "confirmed-write" {
					s.Confirm()
				}
				return rejection
			})
			if got != rejection || tx.commits != 1 || tx.rollbacks != 1 {
				t.Fatal("successful transaction changed its business result", got, tx.commits, tx.rollbacks)
			}
		})
	}
}

func TestWriteWithoutConfirmationStillCannotCommit(t *testing.T) {
	tx := &commitBoundaryTx{}
	got := Run(context.Background(), commitBoundaryDB{tx}, true, func(*Scope) error { return nil })
	if got == nil || tx.commits != 0 || tx.rollbacks != 1 {
		t.Fatal("unconfirmed write escaped", got, tx.commits, tx.rollbacks)
	}
}

func TestReadInfrastructureFailureSkipsActivityAndKeepsFirstCause(t *testing.T) {
	for _, cause := range []error{&pgconn.PgError{Code: "57014"}, context.Canceled, context.DeadlineExceeded, io.ErrUnexpectedEOF, foundation.Fail(503, "M02_DEPENDENCY_UNAVAILABLE", "不可用")} {
		tx := &commitBoundaryTx{}
		activity := 0
		got := Run(context.Background(), commitBoundaryDB{tx}, false, func(s *Scope) error {
			s.BeforeCommit(func(context.Context) error { activity++; return &pgconn.PgError{Code: "25P02"} })
			return submission.RejectRead(cause)
		})
		if !errors.Is(got, cause) || activity != 0 || tx.commits != 0 || tx.rollbacks != 1 {
			t.Errorf("first cause %T lost or failed transaction continued: got=%v activity=%d commits=%d rollbacks=%d", cause, got, activity, tx.commits, tx.rollbacks)
		}
	}
}

// A preliminary relationship-lock savepoint may finish before the business
// savepoint fails. Successful release must not consume its cleanup allowance.
type savepointParent struct {
	pgx.Tx
	child *commitBoundaryTx
}

func (p savepointParent) Begin(context.Context) (pgx.Tx, error) { return p.child, nil }
func TestSuccessfulSavepointLeavesLaterCleanupBudgetUnstarted(t *testing.T) {
	child := &commitBoundaryTx{}
	s := &Scope{tx: savepointParent{child: child}}
	if err := s.Business(context.Background(), func(pgx.Tx) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if child.commits != 1 || child.rollbacks != 0 || !s.cleanupDeadline.IsZero() {
		t.Fatal("successful savepoint consumed cleanup budget", child.commits, child.rollbacks, s.cleanupDeadline)
	}
	cause := errors.New("later business failure")
	if err := s.Business(context.Background(), func(pgx.Tx) error { return cause }); err != cause {
		t.Fatal(err)
	}
	if child.rollbacks != 1 || s.cleanupDeadline.IsZero() {
		t.Fatal("failed business savepoint did not retain outer cleanup deadline")
	}
}

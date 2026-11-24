package submission_test

import (
	"context"
	"errors"
	"testing"

	"chnhcm.local/chnhcm/internal/contexts/organization/application/submission"
)

// A deliberately non-BU result checks the port without importing BU DTOs.
type receipt struct {
	Reference string
	Outcome   string
	Secret    string
	Replayed  bool
}

func TestUnconfirmedRepliesCannotCloseRecovery(t *testing.T) {
	pending := errors.New("commit acknowledgement lost")
	rejected := errors.New("business rejection")
	for _, outcome := range []string{"accepted", "rejected", "no_change", "not_applied"} {
		for _, committed := range []bool{false, true} {
			t.Run(outcome+map[bool]string{true: "/confirmed", false: "/unconfirmed"}[committed], func(t *testing.T) {
				within := func(ctx context.Context, fn func(int) error) error {
					if err := fn(1); err != nil {
						return err
					}
					if !committed {
						return pending
					}
					return nil
				}
				got, err := submission.Run(context.Background(), within, func(_ int, reply *submission.Reply[receipt]) error {
					reply.Result = receipt{Reference: "immutable-original", Outcome: outcome}
					if outcome == "rejected" {
						reply.Rejection = rejected
					}
					return nil
				})
				if !committed {
					if !errors.Is(err, pending) || got != (submission.Reply[receipt]{}) {
						t.Fatalf("unconfirmed result escaped: %+v, %v", got, err)
					}
				} else if err != nil || got.Result.Reference != "immutable-original" || got.Result.Outcome != outcome || (outcome == "rejected" && !errors.Is(got.Rejection, rejected)) {
					t.Fatalf("confirmed original lost: %+v, %v", got, err)
				}
			})
		}
	}
}

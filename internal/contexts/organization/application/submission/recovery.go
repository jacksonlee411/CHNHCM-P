package submission

import (
	"context"
	"crypto/subtle"

	"chnhcm.local/chnhcm/internal/foundation"
)

// Lookup serializes late commands and resolve against the same object repository.
func Lookup[R, T any](ctx context.Context, repo Repository[R, T], id string) (*Saved[R], error) {
	if err := repo.LockSubmission(ctx, id); err != nil {
		return nil, err
	}
	return repo.FindResult(ctx, id)
}

// Read projects a stored result using current object/history rights. It must not
// reevaluate the old command, write permission, business date or qualifications.
type Read[R any] func(context.Context, *Saved[R]) error

type ReplayPolicy[R any] struct {
	Outcome      func(R) string
	Read         Read[R]
	MarkReplayed func(*R)
}

// Replay runs only for an existing result under Lookup's lock. Identity and
// digest errors precede authorization; authorization precedes damaged diagnostics.
func Replay[R any](ctx context.Context, confirm interface{ Confirm() }, saved *Saved[R], digest []byte, policy ReplayPolicy[R]) Reply[R] {
	confirm.Confirm()
	if policy.Outcome(saved.Result) == "not_applied" {
		return Reply[R]{Rejection: foundation.Fail(409, "M02_REQUEST_CLOSED", "本次提交已确认未保存，请核对后重新提交")}
	}
	if subtle.ConstantTimeCompare(saved.Digest, digest) != 1 {
		return Reply[R]{Rejection: foundation.Fail(409, "M02_REQUEST_MISMATCH", "保存内容已变化，请查看上次保存结果")}
	}
	reply := readSaved(ctx, saved, policy.Read)
	if reply.Rejection == nil {
		policy.MarkReplayed(&reply.Result)
	}
	return reply
}

func readSaved[R any](ctx context.Context, saved *Saved[R], read Read[R]) Reply[R] {
	if err := read(ctx, saved); err != nil {
		return Reply[R]{Rejection: err}
	}
	if saved.DecodeError != nil {
		return Reply[R]{Rejection: saved.DecodeError}
	}
	return Reply[R]{Result: saved.Result}
}

// Closure separates a safe authorization denial from a technical error. Only an
// authorized missing-result lookup may persist the object's not_applied terminal.
type Closure[R, T any] struct {
	Terminal T
	Reply    Reply[R]
}

type ResolvePolicy[R, T any] struct {
	PrepareRead func(context.Context) error
	Read        Read[R]
	Close       func(context.Context) (Closure[R, T], error)
}

func Resolve[R, T any](ctx context.Context, repo Repository[R, T], id string, policy ResolvePolicy[R, T]) (Reply[R], error) {
	saved, err := Lookup(ctx, repo, id)
	if err != nil {
		return Reply[R]{}, err
	}
	if saved != nil {
		if err := policy.PrepareRead(ctx); err != nil {
			return Reply[R]{}, err
		}
		reply := readSaved(ctx, saved, policy.Read)
		repo.Confirm()
		return reply, nil
	}
	closure, err := policy.Close(ctx)
	if err != nil {
		return Reply[R]{}, err
	}
	if closure.Reply.Rejection == nil {
		if err := repo.StoreTerminal(ctx, closure.Terminal); err != nil {
			return Reply[R]{}, err
		}
	}
	repo.Confirm()
	return closure.Reply, nil
}

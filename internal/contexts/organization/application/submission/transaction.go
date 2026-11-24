// Package submission coordinates confirmed results without owning object facts or SQL.
package submission

import "context"

// Saved keeps exact stored results separate from their current readable projection.
type Saved[R any] struct {
	Result          R
	Digest          []byte
	RequestObjectID string
	DecodeError     error
}

// Repository is bound to one object kind, instance and authenticated subject.
// Business must finish savepoint rollback before a rejected terminal can be stored.
type Repository[R, T any] interface {
	LockSubmission(context.Context, string) error
	FindResult(context.Context, string) (*Saved[R], error)
	StoreTerminal(context.Context, T) error
	Confirm()
	Business(context.Context, func() error) error
}

// Reply is provisional inside a transaction, including a business rejection.
type Reply[R any] struct {
	Result    R
	Rejection error
}

// Run releases a reply only after within confirms its outer commit. The callback
// must keep results in reply; a failed commit cannot close client recovery.
func Run[U, R any](ctx context.Context, within func(context.Context, func(U) error) error, work func(U, *Reply[R]) error) (Reply[R], error) {
	var reply Reply[R]
	err := within(ctx, func(u U) error { return work(u, &reply) })
	if err != nil {
		return Reply[R]{}, err
	}
	return reply, nil
}

// ReadRejection permits committing necessary session activity before returning
// a denied read. It does not persist an authorization decision.
// Write transactions never accept this intent, regardless of HTTP error status.
type ReadRejection struct{ Cause error }

func (e *ReadRejection) Error() string { return e.Cause.Error() }
func (e *ReadRejection) Unwrap() error { return e.Cause }
func RejectRead(err error) error {
	if err == nil {
		return nil
	}
	return &ReadRejection{Cause: err}
}

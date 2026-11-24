package submission_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/application/submission"
	"chnhcm.local/chnhcm/internal/foundation"
)

type terminal struct{ Reference, Outcome string }
type repository struct {
	saved     *submission.Saved[receipt]
	stored    []terminal
	trace     []string
	failStore error
}

func (r *repository) LockSubmission(context.Context, string) error {
	r.trace = append(r.trace, "lock")
	return nil
}
func (r *repository) FindResult(context.Context, string) (*submission.Saved[receipt], error) {
	r.trace = append(r.trace, "find")
	if r.saved == nil {
		return nil, nil
	}
	copy := *r.saved
	return &copy, nil
}
func (r *repository) StoreTerminal(_ context.Context, t terminal) error {
	r.trace = append(r.trace, "store")
	if r.failStore != nil {
		return r.failStore
	}
	r.stored = append(r.stored, t)
	r.saved = &submission.Saved[receipt]{Result: receipt{Reference: t.Reference, Outcome: t.Outcome}}
	return nil
}
func (r *repository) Confirm() { r.trace = append(r.trace, "confirm") }
func (r *repository) Business(context.Context, func() error) error {
	panic("recovery must never execute business writes")
}

func TestReplayUsesOriginalFactsAndCurrentReadAuthority(t *testing.T) {
	ctx := context.Background()
	denied := errors.New("current read revoked")
	corrupt := errors.New("damaged diagnostics")
	digest := []byte("bound original request")
	for _, tc := range []struct {
		name, outcome               string
		different, revoked, damaged bool
		code                        string
		want                        error
	}{
		{name: "accepted", outcome: "accepted"},
		{name: "rejected", outcome: "rejected"},
		{name: "no change", outcome: "no_change"},
		{name: "closed precedes digest", outcome: "not_applied", different: true, revoked: true, damaged: true, code: "M02_REQUEST_CLOSED"},
		{name: "mismatch precedes read", outcome: "accepted", different: true, revoked: true, damaged: true, code: "M02_REQUEST_MISMATCH"},
		{name: "read precedes damaged result", outcome: "accepted", revoked: true, damaged: true, want: denied},
		{name: "damaged result", outcome: "rejected", damaged: true, want: corrupt},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &repository{saved: &submission.Saved[receipt]{Result: receipt{Reference: "old-source", Outcome: tc.outcome, Secret: "private"}, Digest: digest}}
			if tc.damaged {
				r.saved.DecodeError = corrupt
			}
			old, err := submission.Lookup[receipt, terminal](ctx, r, "original-id")
			if err != nil {
				t.Fatal(err)
			}
			request := digest
			if tc.different {
				request = []byte("another request")
			}
			reads := 0
			reply := submission.Replay(ctx, r, old, request, submission.ReplayPolicy[receipt]{
				Outcome: func(r receipt) string { return r.Outcome },
				Read: func(_ context.Context, s *submission.Saved[receipt]) error {
					reads++
					if tc.revoked {
						return denied
					}
					s.Result.Secret = ""
					return nil
				},
				MarkReplayed: func(r *receipt) { r.Replayed = true },
			})
			if tc.code != "" {
				var f *foundation.Fault
				if !errors.As(reply.Rejection, &f) || f.Code != tc.code || reads != 0 {
					t.Fatalf("identity priority lost: %+v, reads=%d", reply, reads)
				}
			} else if tc.want != nil {
				if !errors.Is(reply.Rejection, tc.want) || reads != 1 {
					t.Fatalf("read priority lost: %+v", reply)
				}
			} else if reply.Rejection != nil || reply.Result != (receipt{Reference: "old-source", Outcome: tc.outcome, Replayed: true}) {
				t.Fatalf("original result changed: %+v", reply)
			}
			if reply.Rejection != nil && reply.Result != (receipt{}) {
				t.Fatal("denied result leaked")
			}
			if !reflect.DeepEqual(r.trace, []string{"lock", "find", "confirm"}) || len(r.stored) != 0 || r.saved.Result.Secret != "private" {
				t.Fatalf("recovery changed stored evidence: %+v", r)
			}
		})
	}
}

func TestResolveClosesOnlyAuthorizedMissingResult(t *testing.T) {
	ctx := context.Background()
	denied := errors.New("read denied")
	technical := errors.New("storage unavailable")
	for _, tc := range []struct {
		name                                              string
		existing, deny, failPrepare, failClose, failStore bool
	}{
		{name: "missing"}, {name: "existing", existing: true}, {name: "missing denied", deny: true},
		{name: "existing denied", existing: true, deny: true}, {name: "prepare failure", existing: true, failPrepare: true},
		{name: "decision failure", failClose: true}, {name: "terminal write failure", failStore: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &repository{}
			if tc.existing {
				r.saved = &submission.Saved[receipt]{Result: receipt{Reference: "original", Outcome: "no_change"}}
			}
			if tc.failStore {
				r.failStore = technical
			}
			prepares, reads, closes := 0, 0, 0
			got, err := submission.Resolve[receipt, terminal](ctx, r, "id", submission.ResolvePolicy[receipt, terminal]{
				PrepareRead: func(context.Context) error {
					prepares++
					if tc.failPrepare {
						return technical
					}
					return nil
				},
				Read: func(context.Context, *submission.Saved[receipt]) error {
					reads++
					if tc.deny {
						return denied
					}
					return nil
				},
				Close: func(context.Context) (submission.Closure[receipt, terminal], error) {
					closes++
					if tc.failClose {
						return submission.Closure[receipt, terminal]{}, technical
					}
					if tc.deny {
						return submission.Closure[receipt, terminal]{Reply: submission.Reply[receipt]{Rejection: denied}}, nil
					}
					return submission.Closure[receipt, terminal]{Terminal: terminal{Reference: "id", Outcome: "not_applied"}, Reply: submission.Reply[receipt]{Result: receipt{Reference: "id", Outcome: "not_applied"}}}, nil
				},
			})
			fatal := tc.failPrepare || tc.failClose || tc.failStore
			if fatal {
				if !errors.Is(err, technical) || got != (submission.Reply[receipt]{}) || r.trace[len(r.trace)-1] == "confirm" {
					t.Fatalf("failed recovery confirmed: %+v %v %v", got, err, r.trace)
				}
			} else if tc.deny {
				if err != nil || !errors.Is(got.Rejection, denied) || got.Result != (receipt{}) || len(r.stored) != 0 {
					t.Fatal("denial escaped or closed submission")
				}
			} else {
				expected := receipt{Reference: "id", Outcome: "not_applied"}
				if tc.existing {
					expected = receipt{Reference: "original", Outcome: "no_change"}
				}
				if err != nil || got.Rejection != nil || got.Result != expected {
					t.Fatalf("wrong original/closed result: %+v %v", got, err)
				}
			}
			if tc.existing {
				if prepares != 1 || closes != 0 || len(r.stored) != 0 || (!tc.failPrepare && reads != 1) {
					t.Fatal("existing result reexecuted")
				}
			} else if prepares != 0 || reads != 0 || closes != 1 {
				t.Fatal("missing result read invented object")
			}
			if !fatal && r.trace[len(r.trace)-1] != "confirm" {
				t.Fatal("safe read did not confirm")
			}
		})
	}
}

type currentGuard struct {
	access.Guard
	object, history access.Decision
	actions         []string
}

func (g *currentGuard) Decide(_ context.Context, in access.Intent) (access.Decision, error) {
	g.actions = append(g.actions, in.Action)
	if in.Action == "fixture.read" {
		return g.object, nil
	}
	if in.Action == "fixture.history.read" {
		return g.history, nil
	}
	panic("recovery must not check write permissions")
}
func TestCurrentHistoryCannotExpandOrMutateObjectFields(t *testing.T) {
	denied := errors.New("denied")
	object := submission.ReadRequirement{Intent: access.Intent{Action: "fixture.read"}, Denied: denied}
	history := submission.ReadRequirement{Intent: access.Intent{Action: "fixture.history.read"}, Denied: denied}
	g := &currentGuard{object: access.Decision{Allowed: true, ReadFields: []string{"name", "code"}}, history: access.Decision{Allowed: true, ReadFields: []string{"code", "secret"}}}
	got, err := submission.AuthorizeHistory(context.Background(), g, object, history)
	if err != nil || !reflect.DeepEqual(got.Object, []string{"name", "code"}) || !reflect.DeepEqual(got.History, []string{"code"}) {
		t.Fatalf("field authority expanded: %+v %v", got, err)
	}
	got.Object[0] = "changed"
	got.History[0] = "changed"
	if !reflect.DeepEqual(g.object.ReadFields, []string{"name", "code"}) || g.history.ReadFields[0] != "code" {
		t.Fatal("authorization slices were mutated")
	}
	g.actions = nil
	g.object.Allowed = false
	got, err = submission.AuthorizeHistory(context.Background(), g, object, history)
	if !errors.Is(err, denied) || got.Object != nil || got.History != nil || !reflect.DeepEqual(g.actions, []string{"fixture.read"}) {
		t.Fatal("object revocation bypassed")
	}
	g.actions = nil
	g.object.Allowed = true
	g.history.Allowed = false
	got, err = submission.AuthorizeHistory(context.Background(), g, object, history)
	if !errors.Is(err, denied) || got.Object != nil || got.History != nil {
		t.Fatal("history revocation bypassed")
	}
}

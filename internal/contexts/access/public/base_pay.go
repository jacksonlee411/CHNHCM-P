package public

import "context"

// The caller first establishes employment visibility for the selected term.
// A restricted plan grant applies only to that plan.
type BasePayRights struct {
	Read, History, First, Adjust, Correct, Withdraw, UseAll bool
	Plans                                                   map[string]bool
}
type BasePayAuthorizer interface {
	BasePayRights(context.Context, string, string) (BasePayRights, error)
}

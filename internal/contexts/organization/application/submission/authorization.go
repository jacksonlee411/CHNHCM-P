package submission

import (
	"context"
	"slices"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
)

// ReadRequirement carries an object-owned intent and its safe denial response.
type ReadRequirement struct {
	Intent access.Intent
	Denied error
}

type ReadFields struct {
	Object  []string
	History []string
}

// AuthorizeHistory never expands object fields through history permission, and
// does not mutate the decision slices owned by the current authorization guard.
func AuthorizeHistory(ctx context.Context, guard access.Guard, object, history ReadRequirement) (ReadFields, error) {
	read, err := guard.Decide(ctx, object.Intent)
	if err != nil {
		return ReadFields{}, err
	}
	if !read.Allowed {
		return ReadFields{}, object.Denied
	}
	historical, err := guard.Decide(ctx, history.Intent)
	if err != nil {
		return ReadFields{}, err
	}
	if !historical.Allowed {
		return ReadFields{}, history.Denied
	}
	fields := slices.Clone(read.ReadFields)
	fields = slices.DeleteFunc(fields, func(f string) bool { return !slices.Contains(historical.ReadFields, f) })
	return ReadFields{Object: slices.Clone(read.ReadFields), History: fields}, nil
}

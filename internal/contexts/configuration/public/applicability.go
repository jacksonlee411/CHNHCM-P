// Package public exposes explicit applicability policies bound to the owning object's transaction.
package public

import (
	"context"

	"chnhcm.local/chnhcm/internal/foundation"
)

type Choice struct {
	Dimension string `json:"dimension"`
	ID        string `json:"id"`
	Exclude   bool   `json:"exclude,omitempty"`
}

type Scope struct {
	Mode    string   `json:"mode"`
	Choices []Choice `json:"choices,omitempty"`
}

type Member struct {
	Choice
	Code     string `json:"code,omitempty"`
	Name     string `json:"name"`
	Source   string `json:"sourceVersion"`
	Record   string `json:"recordId"`
	Revision string `json:"revision"`
}

type Policy struct {
	Version     string            `json:"version"`
	Mode        string            `json:"mode"`
	ObservedFor foundation.Period `json:"observedFor"`
	Members     []Member          `json:"members,omitempty"`
}

type Applicability interface {
	Normalize(Scope) (Scope, error)
	PermitsDimension(Scope, string, string) bool
	SavePlanScope(context.Context, string, string, Scope, foundation.Period, []Member) (Policy, error)
	ReadPlanScope(context.Context, string, string, string) (Policy, error)
	Matches(Policy, map[string]string) bool
}

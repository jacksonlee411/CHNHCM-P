package public

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
)

type LegalPurpose struct {
	foundation.Period `json:",inline"`
	ID                string `json:"id"`
	Revision          string `json:"revision,omitempty"`
	Role              string `json:"role"`
	Kind              string `json:"kind"` // declaration, bu or location
	Basis             string `json:"basis"`
	Target            string `json:"target,omitempty"`
	RequireBU         bool   `json:"requireBU,omitempty"`
	RequireLocation   bool   `json:"requireLocation,omitempty"`
}
type PurposeCheck struct {
	Adopted           []LegalAdoption `json:"adopted,omitempty"`
	foundation.Period `json:",inline"`
	Role              string `json:"role"`
	BU                string `json:"bu,omitempty"`
	Location          string `json:"location,omitempty"`
	Expected          string `json:"expectedVersion,omitempty"`
}
type LegalAdoption struct {
	foundation.Period `json:",inline"`
	EntityName        string   `json:"entityName,omitempty"`
	EntityVersion     string   `json:"entityVersion"`
	EntityRevision    string   `json:"entityRevision"`
	PurposeVersion    string   `json:"purposeVersion"`
	Declaration       string   `json:"declaration"`
	Bindings          []string `json:"bindings"`
}
type PurposeCandidates struct {
	PurposeCheck `json:",inline"`
	Search       string `json:"search,omitempty"`
	Page         int    `json:"page"`
}
type EligibleEntity struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type LegalPurposes interface {
	Candidates(context.Context, foundation.Credentials, PurposeCandidates) ([]EligibleEntity, error)
	Purposes(context.Context, foundation.Credentials, string, string) (foundation.DatedRules[LegalPurpose], error)
	DeclarePurpose(context.Context, foundation.Credentials, string, string, foundation.RuleChange[LegalPurpose]) (foundation.RuleReceipt, error)
	ResolvePurpose(context.Context, foundation.Credentials, string) (foundation.RuleReceipt, error)
	CheckPurpose(context.Context, foundation.Credentials, string, PurposeCheck) ([]LegalAdoption, error)
}

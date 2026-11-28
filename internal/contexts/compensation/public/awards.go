package public

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
)

type PayAllowance struct {
	Name   string `json:"name"`
	Amount string `json:"amount"`
}
type PayBasis struct {
	PlanID        string         `json:"planId,omitempty"`
	PlanName      string         `json:"planName,omitempty"`
	PlanCode      string         `json:"planCode,omitempty"`
	PlanSource    string         `json:"planSource,omitempty"`
	PlanRecord    string         `json:"planRecord,omitempty"`
	PlanRevision  string         `json:"planRevision,omitempty"`
	ScopeVersion  string         `json:"scopeVersion,omitempty"`
	ComponentID   string         `json:"componentId,omitempty"`
	ComponentName string         `json:"componentName,omitempty"`
	RateID        string         `json:"rateId,omitempty"`
	RateSource    string         `json:"rateSource,omitempty"`
	RateRecord    string         `json:"rateRecord,omitempty"`
	RateRevision  string         `json:"rateRevision,omitempty"`
	Grade         string         `json:"grade,omitempty"`
	Minimum       string         `json:"minimum,omitempty"`
	Guidance      string         `json:"guidance,omitempty"`
	Maximum       string         `json:"maximum,omitempty"`
	Allowances    []PayAllowance `json:"allowances,omitempty"`
	Token         string         `json:"token,omitempty"`
}
type PayAward struct {
	ID          string            `json:"awardId"`
	Revision    string            `json:"revision"`
	Assignment  string            `json:"assignmentId"`
	Day         string            `json:"effectiveDate"`
	Action      string            `json:"action"`
	Amount      string            `json:"amount"`
	Reason      string            `json:"reason"`
	Note        string            `json:"note,omitempty"`
	RecordedAt  string            `json:"recordedAt"`
	Operator    string            `json:"operator,omitempty"`
	Basis       *PayBasis         `json:"basis,omitempty"`
	Previous    string            `json:"previous,omitempty"`
	Current     bool              `json:"current"`
	Restoration *AwardRestoration `json:"restoration,omitempty"`
}
type AwardRestoration struct {
	Amount string `json:"amount,omitempty"`
}
type PayQuery struct {
	Day     string
	Page    int
	Plan    string
	Prepare bool
}
type PayDetail struct {
	CandidateCount    int        `json:"candidateCount"`
	CandidatePage     int        `json:"candidatePage"`
	CandidatePageSize int        `json:"candidatePageSize"`
	SelectedBasis     *PayBasis  `json:"selectedBasis,omitempty"`
	Assignment        string     `json:"assignmentId"`
	BusinessAt        string     `json:"businessAt"`
	Active            bool       `json:"active"`
	Version           string     `json:"version"`
	Actions           []string   `json:"actions"`
	Current           *PayAward  `json:"current,omitempty"`
	Policy            *PayBasis  `json:"policy,omitempty"`
	Message           string     `json:"message,omitempty"`
	Candidates        []PayBasis `json:"candidates"`
	Items             []PayAward `json:"items"`
}
type ConfirmAward struct {
	Action          string  `json:"action"`
	EffectiveDate   string  `json:"effectiveDate"`
	ExpectedVersion string  `json:"expectedVersion"`
	ID              string  `json:"awardId,omitempty"`
	Revision        string  `json:"expectedRevision,omitempty"`
	Plan            string  `json:"planId,omitempty"`
	Component       string  `json:"componentId,omitempty"`
	BasisToken      string  `json:"basisToken,omitempty"`
	Amount          *string `json:"amount,omitempty"`
	Reason          *string `json:"reason,omitempty"`
	Note            string  `json:"note,omitempty"`
}
type AwardReceipt struct {
	Submission string   `json:"submissionId"`
	Award      PayAward `json:"award"`
}
type RecoveryInput struct {
	Assignment string        `json:"assignmentId,omitempty"`
	Command    *ConfirmAward `json:"command,omitempty"`
}
type AwardResolution struct {
	Submission   string    `json:"submissionId"`
	Outcome      string    `json:"outcome"`
	Award        *PayAward `json:"award,omitempty"`
	Confirmation string    `json:"confirmation,omitempty"`
}
type AwardService interface {
	PayAccess(context.Context, foundation.Credentials, string, string) (bool, error)
	BasePay(context.Context, foundation.Credentials, string, PayQuery) (PayDetail, error)
	ConfirmAward(context.Context, foundation.Credentials, string, string, string, ConfirmAward) (AwardReceipt, error)
	AwardOriginal(context.Context, foundation.Credentials, string, string) (PayAward, error)
	AwardReceipt(context.Context, foundation.Credentials, string) (AwardReceipt, error)
	ResolveAward(context.Context, foundation.Credentials, string, RecoveryInput) (AwardResolution, error)
}

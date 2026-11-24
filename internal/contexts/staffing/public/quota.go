package public

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
)

const Quota Kind = "staffingquota"

type QuotaInput struct {
	PositionID       string `json:"positionId,omitempty"`
	ExpectedSource   string `json:"expectedSourceVersion,omitempty"`
	ExpectedRevision string `json:"expectedRecordRevision,omitempty"`
	EffectiveDate    string `json:"effectiveDate,omitempty"`
	NewDate          string `json:"newEffectiveDate,omitempty"`
	Amount           *int   `json:"quota,omitempty"`
	Reason           string `json:"reason,omitempty"`
	ReasonProvided   bool   `json:"-"`
}
type AllocationRow struct {
	PositionID     string `json:"positionId,omitempty"`
	ExpectedSource string `json:"expectedSourceVersion,omitempty"`
	Amount         *int   `json:"quota,omitempty"`
	Withdraw       bool   `json:"withdraw,omitempty"`
}
type AllocationInput struct {
	EffectiveDate string          `json:"effectiveDate"`
	Reason        string          `json:"reason"`
	Total         *AllocationRow  `json:"total,omitempty"`
	Positions     []AllocationRow `json:"positions,omitempty"`
}
type AllocationRef struct {
	TargetKind    string `json:"targetKind"`
	TargetID      string `json:"targetId"`
	QuotaID       string `json:"quotaId,omitempty"`
	SourceVersion string `json:"sourceVersion,omitempty"`
	RecordID      string `json:"recordId,omitempty"`
}
type QuotaCommand struct {
	DepartmentID                            string
	Allocation                              *AllocationInput
	Action, QuotaID, RecordID, SubmissionID string
	Digest                                  []byte
	Input                                   QuotaInput
}
type QuotaQuery struct {
	DepartmentDetail, DepartmentOptions, Statistics                 bool
	Day, PositionID, Source, Search, DepartmentID, RecordID, Anchor string
	Page, PageSize                                                  int
}
type QuotaService interface {
	QuotaChanges(context.Context, foundation.Credentials, string, LogQuery) (ChangeLog, error)
	ChangeStaffingQuota(context.Context, foundation.Credentials, QuotaCommand) (Result, error)
	PreviewQuota(context.Context, foundation.Credentials, QuotaCommand) (View, error)
	ResolveQuota(context.Context, foundation.Credentials, string) (Result, error)
	Quotas(context.Context, foundation.Credentials, QuotaQuery) (View, error)
}

// RegularUseRequest is sent by a source owner inside its existing transaction.
// M03 resolves and adopts policy and capacity after acquiring its guards.
type RegularUseRequest struct {
	Enterprise, Position, Owner, Category, Purpose string
	From, Until                                    string
}

type RegularUseSegment struct{ From, Until, Policy, Effect string }

type RegularQuotaGuard interface {
	CheckRegularUse(context.Context, RegularUseRequest) ([]RegularUseSegment, error)
}

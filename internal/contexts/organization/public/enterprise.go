package public

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
)

type EnterpriseContext struct {
	ID                   string       `json:"id"`
	CurrentDate          string       `json:"currentDate"`
	DefaultEffectiveDate string       `json:"defaultEffectiveDate"`
	Timezone             string       `json:"timezone"`
	NameMaxLength        int          `json:"nameMaxLength"`
	NameGuidance         string       `json:"nameGuidance"`
	Capabilities         Capabilities `json:"capabilities"`
}
type EnterpriseRecord struct {
	ID               string            `json:"recordId"`
	Revision         string            `json:"revision"`
	Deleted          bool              `json:"deleted"`
	EffectiveDate    *string           `json:"effectiveDate,omitempty"`
	EndExclusive     **string          `json:"endExclusive,omitempty"`
	EndInclusive     **string          `json:"endInclusive,omitempty"`
	Name             *string           `json:"name,omitempty"`
	NameEn           **string          `json:"nameEn,omitempty"`
	State            *string           `json:"state,omitempty"`
	OperationType    *string           `json:"operationType,omitempty"`
	OperationReason  *string           `json:"operationReason,omitempty"`
	TypeLabel        *string           `json:"typeLabel,omitempty"`
	ReasonLabel      *string           `json:"reasonLabel,omitempty"`
	OperationSummary *OperationSummary `json:"operationSummary,omitempty"`
}
type EnterpriseAdditionContext struct {
	Available            bool             `json:"available"`
	AnchorRecordID       string           `json:"anchorRecordId"`
	Mode                 string           `json:"mode"`
	MinEffectiveDate     string           `json:"minEffectiveDate"`
	MaxEffectiveDate     *string          `json:"maxEffectiveDate"`
	DefaultEffectiveDate string           `json:"defaultEffectiveDate"`
	Defaults             EnterpriseRecord `json:"defaults"`
}
type EnterpriseLifecycleContext struct {
	AnchorRecordID       string           `json:"anchorRecordId"`
	OperationType        string           `json:"operationType"`
	MinEffectiveDate     string           `json:"minEffectiveDate"`
	MaxEffectiveDate     *string          `json:"maxEffectiveDate"`
	DefaultEffectiveDate string           `json:"defaultEffectiveDate"`
	Defaults             EnterpriseRecord `json:"defaults"`
}
type EnterpriseDetail struct {
	PairInsertion          *PairInsertionContext          `json:"pairInsertion,omitempty"`
	PairDeletion           *EnterprisePairDeletionContext `json:"pairDeletion,omitempty"`
	ID                     string                         `json:"id"`
	Code                   *string                        `json:"code,omitempty"`
	CurrentDate            string                         `json:"currentDate"`
	Status                 *string                        `json:"status,omitempty"`
	NameGuidance           string                         `json:"nameGuidance"`
	HeadSourceVersion      string                         `json:"headSourceVersion"`
	ResolvedSourceVersion  string                         `json:"resolvedSourceVersion"`
	SelectedRecord         *EnterpriseRecord              `json:"selectedRecord"`
	Timeline               []EnterpriseRecord             `json:"timeline"`
	ReadOnly               bool                           `json:"readOnly"`
	SelectionMissing       bool                           `json:"selectionMissing"`
	SelectedRecordIsFuture bool                           `json:"selectedRecordIsFuture"`
	Capabilities           Capabilities                   `json:"capabilities"`
	AdditionContext        *EnterpriseAdditionContext     `json:"additionContext,omitempty"`
	LifecycleContext       *EnterpriseLifecycleContext    `json:"lifecycleContext,omitempty"`
}
type EnterpriseResult struct {
	SubmissionID     string  `json:"submissionId"`
	Outcome          string  `json:"outcome"`
	EnterpriseID     *string `json:"enterpriseId,omitempty"`
	SourceVersion    *string `json:"sourceVersion,omitempty"`
	SelectedRecordID *string `json:"selectedRecordId,omitempty"`
	TargetRecordID   *string `json:"targetRecordId,omitempty"`
	TargetRevision   *string `json:"targetRevision,omitempty"`
	EffectiveDate    *string `json:"effectiveDate,omitempty"`
	RecordedAt       *string `json:"recordedAt,omitempty"`
	ErrorCode        *string `json:"errorCode,omitempty"`
	Action           string  `json:"action,omitempty"`
	Replayed         bool    `json:"replayed"`
	CurrentChanged   bool    `json:"currentChanged"`
}
type EnterpriseChangesInput map[string]*string
type EnterpriseAddInput struct {
	ExpectedSourceVersion string                 `json:"expectedSourceVersion"`
	AnchorRecordID        string                 `json:"anchorRecordId"`
	EffectiveDate         string                 `json:"effectiveDate"`
	OperationType         string                 `json:"operationType"`
	OperationReason       string                 `json:"operationReason"`
	Changes               EnterpriseChangesInput `json:"changes,omitempty"`
}
type EnterpriseCorrectInput struct {
	NewEffectiveDate       string                 `json:"newEffectiveDate,omitempty"`
	Classification         *Classification        `json:"classification,omitempty"`
	ExpectedSourceVersion  string                 `json:"expectedSourceVersion"`
	ExpectedRecordRevision string                 `json:"expectedRecordRevision"`
	Changes                EnterpriseChangesInput `json:"changes,omitempty"`
}

type EnterpriseCommand struct {
	Delete                                       DeleteInput
	InsertPair                                   InsertLifecyclePairInput
	DeletePair                                   DeleteLifecyclePairInput
	Action, EnterpriseID, RecordID, SubmissionID string
	Digest                                       []byte
	Add                                          EnterpriseAddInput
	Correct                                      EnterpriseCorrectInput
}
type EnterprisePreviewInput struct {
	NewEffectiveDate       string                 `json:"newEffectiveDate,omitempty"`
	Classification         *Classification        `json:"classification,omitempty"`
	Action                 string                 `json:"action"`
	ExpectedSourceVersion  string                 `json:"expectedSourceVersion"`
	AnchorRecordID         string                 `json:"anchorRecordId,omitempty"`
	RecordID               string                 `json:"recordId,omitempty"`
	ExpectedRecordRevision string                 `json:"expectedRecordRevision,omitempty"`
	EffectiveDate          string                 `json:"effectiveDate,omitempty"`
	OperationType          string                 `json:"operationType,omitempty"`
	OperationReason        string                 `json:"operationReason,omitempty"`
	Changes                EnterpriseChangesInput `json:"changes,omitempty"`
}
type EnterprisePreview struct {
	NewNoFactRange        *PreviewDateRange `json:"newNoFactRange,omitempty"`
	PreviousCurrentStatus string            `json:"previousCurrentStatus,omitempty"`
	CurrentStatus         string            `json:"currentStatus,omitempty"`
	ReadFields            []string          `json:"readFields"`
	AfterRecord           *EnterpriseRecord `json:"afterRecord,omitempty"`
}
type EnterpriseChangeItem struct {
	ChangeKind    string            `json:"changeKind"`
	BeforeRef     *ChangeRef        `json:"beforeRef"`
	AfterRef      *ChangeRef        `json:"afterRef"`
	Diffs         []ChangeDiff      `json:"diffs"`
	RecordID      string            `json:"recordId"`
	BeforeContext *EnterpriseRecord `json:"beforeContext"`
	AfterContext  *EnterpriseRecord `json:"afterContext"`
}
type EnterpriseChange struct {
	ID                    string                 `json:"id"`
	SourceVersion         string                 `json:"sourceVersion"`
	PreviousSourceVersion *string                `json:"previousSourceVersion"`
	RecordedAt            string                 `json:"recordedAt"`
	Action                string                 `json:"action"`
	Operator              OperatorSnapshot       `json:"operator"`
	Items                 []EnterpriseChangeItem `json:"items"`
}
type EnterpriseChanges struct {
	ReadFields           []string           `json:"readFields"`
	DetailReadFields     []string           `json:"detailReadFields"`
	ContextSourceVersion *string            `json:"contextSourceVersion"`
	ThroughSourceVersion string             `json:"throughSourceVersion"`
	Items                []EnterpriseChange `json:"items"`
	Total                int                `json:"total"`
	Page                 int                `json:"page"`
	PageSize             int                `json:"pageSize"`
}
type EnterpriseService interface {
	Context(context.Context, foundation.Credentials) (EnterpriseContext, error)
	Detail(context.Context, foundation.Credentials, DetailInput) (EnterpriseDetail, error)
	Maintain(context.Context, foundation.Credentials, EnterpriseCommand) (EnterpriseResult, error)
	Preview(context.Context, foundation.Credentials, string, EnterprisePreviewInput) (EnterprisePreview, error)
	Resolve(context.Context, foundation.Credentials, string) (EnterpriseResult, error)
	Changes(context.Context, foundation.Credentials, ChangeQuery) (EnterpriseChanges, error)
}

type EnterprisePairDeletionContext struct {
	Deactivation EnterpriseRecord       `json:"deactivation"`
	Reactivation EnterpriseRecord       `json:"reactivation"`
	Changes      []EnterpriseDataChange `json:"changes"`
}
type EnterpriseDataChange struct {
	Start        *string          `json:"start,omitempty"`
	EndExclusive **string         `json:"endExclusive,omitempty"`
	EndInclusive **string         `json:"endInclusive,omitempty"`
	Before       EnterpriseRecord `json:"before"`
	After        EnterpriseRecord `json:"after"`
}

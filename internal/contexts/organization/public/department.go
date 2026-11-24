package public

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"io"
)

type DepartmentCreateInput struct {
	Code               string  `json:"code"`
	Name               string  `json:"name"`
	ShortName          *string `json:"shortName,omitempty"`
	ManagementBUID     string  `json:"managementBuId"`
	ParentDepartmentID *string `json:"parentDepartmentId,omitempty"`
	EffectiveDate      string  `json:"effectiveDate"`
	OperationType      string  `json:"operationType"`
	OperationReason    string  `json:"operationReason"`
}
type DepartmentChangesInput map[string]*string
type DepartmentAddInput struct {
	ExpectedSourceVersion string                 `json:"expectedSourceVersion"`
	AnchorRecordID        string                 `json:"anchorRecordId"`
	EffectiveDate         string                 `json:"effectiveDate"`
	OperationType         string                 `json:"operationType"`
	OperationReason       string                 `json:"operationReason"`
	Changes               DepartmentChangesInput `json:"changes,omitempty"`
}
type DepartmentCorrectInput struct {
	ExpectedSourceVersion  string                 `json:"expectedSourceVersion"`
	ExpectedRecordRevision string                 `json:"expectedRecordRevision"`
	NewEffectiveDate       string                 `json:"newEffectiveDate,omitempty"`
	Classification         *Classification        `json:"classification,omitempty"`
	Changes                DepartmentChangesInput `json:"changes,omitempty"`
}
type DepartmentDeleteInput struct {
	ExpectedSourceVersion  string `json:"expectedSourceVersion"`
	ExpectedRecordRevision string `json:"expectedRecordRevision"`
}
type DepartmentRedateInput struct {
	ExpectedSourceVersion  string `json:"expectedSourceVersion"`
	ExpectedRecordRevision string `json:"expectedRecordRevision"`
	NewEffectiveDate       string `json:"newEffectiveDate"`
}
type DepartmentDeactivateInput struct {
	ExpectedSourceVersion string `json:"expectedSourceVersion"`
	AnchorRecordID        string `json:"anchorRecordId"`
	EffectiveDate         string `json:"effectiveDate"`
	OperationType         string `json:"operationType"`
	OperationReason       string `json:"operationReason"`
}
type DepartmentReactivateInput DepartmentAddInput
type DepartmentInsertLifecyclePairInput InsertLifecyclePairInput
type DepartmentDeleteLifecyclePairInput DeleteLifecyclePairInput

// DepartmentCommand is application input, not an open HTTP union or generic items API.
type DepartmentCommand struct {
	Action, DepartmentID, RecordID, SubmissionID string
	Digest                                       []byte
	Fields                                       []string
	Create                                       DepartmentCreateInput
	Add                                          DepartmentAddInput
	Correct                                      DepartmentCorrectInput
	Delete                                       DepartmentDeleteInput
	Redate                                       DepartmentRedateInput
	InsertPair                                   DepartmentInsertLifecyclePairInput
	DeletePair                                   DepartmentDeleteLifecyclePairInput
}
type DepartmentResult struct {
	SubmissionID          string                 `json:"submissionId"`
	Outcome               string                 `json:"outcome"`
	DepartmentID          *string                `json:"departmentId,omitempty"`
	SourceVersion         *string                `json:"sourceVersion,omitempty"`
	TargetRecordID        *string                `json:"targetRecordId,omitempty"`
	TargetRevision        *string                `json:"targetRevision,omitempty"`
	SelectedRecordID      *string                `json:"selectedRecordId,omitempty"`
	ExpectedSourceVersion *string                `json:"expectedSourceVersion,omitempty"`
	ObservedSourceVersion *string                `json:"observedSourceVersion,omitempty"`
	Action                string                 `json:"action,omitempty"`
	EffectiveDate         *string                `json:"effectiveDate,omitempty"`
	PreviousEffectiveDate *string                `json:"previousEffectiveDate,omitempty"`
	RecordedAt            *string                `json:"recordedAt,omitempty"`
	EvaluatedAt           *string                `json:"evaluatedAt,omitempty"`
	EvaluatedBusinessDate *string                `json:"evaluatedBusinessDate,omitempty"`
	CurrentStatus         string                 `json:"currentStatus,omitempty"`
	ErrorCode             *string                `json:"errorCode,omitempty"`
	FailureDetails        *FailureDetails        `json:"failureDetails,omitempty"`
	Items                 []DepartmentItemResult `json:"items,omitempty"`
	Impact                *TimelineImpact        `json:"impact,omitempty"`
	Replayed              bool                   `json:"replayed"`
	CurrentChanged        bool                   `json:"currentChanged"`
}
type DepartmentReference struct {
	ID   string  `json:"id"`
	Code *string `json:"code,omitempty"`
	Name *string `json:"name,omitempty"`
}
type DepartmentPathSegment struct {
	EndInclusive *string `json:"endInclusive"`
	Start        string  `json:"start"`
	EndExclusive *string `json:"endExclusive"`
	Path         string  `json:"path"`
}
type DepartmentPath struct {
	PathMode       string                  `json:"pathMode"`
	PathVersion    string                  `json:"pathVersion"`
	Path           *string                 `json:"departmentPath,omitempty"`
	PathBusinessAt *string                 `json:"pathBusinessAt,omitempty"`
	PathSegments   []DepartmentPathSegment `json:"pathSegments,omitempty"`
}
type DepartmentRecord struct {
	ID                 string               `json:"recordId"`
	Revision           string               `json:"revision"`
	Deleted            bool                 `json:"deleted"`
	EffectiveDate      *string              `json:"effectiveDate,omitempty"`
	EndExclusive       **string             `json:"endExclusive,omitempty"`
	EndInclusive       **string             `json:"endInclusive,omitempty"`
	Name               *string              `json:"name,omitempty"`
	ShortName          **string             `json:"shortName,omitempty"`
	ManagementBUID     *string              `json:"managementBuId,omitempty"`
	ManagementBU       *DepartmentReference `json:"managementBu,omitempty"`
	ParentDepartmentID *string              `json:"parentDepartmentId,omitempty"`
	ParentDepartment   *DepartmentReference `json:"parentDepartment,omitempty"`
	State              *string              `json:"state,omitempty"`
	OperationType      *string              `json:"operationType,omitempty"`
	OperationReason    *string              `json:"operationReason,omitempty"`
	TypeLabel          *string              `json:"typeLabel,omitempty"`
	ReasonLabel        *string              `json:"reasonLabel,omitempty"`
	OperationSummary   *OperationSummary    `json:"operationSummary,omitempty"`
}
type DepartmentContext struct {
	DefaultEffectiveDate string       `json:"defaultEffectiveDate"`
	Enterprise           Enterprise   `json:"enterprise"`
	CurrentDate          string       `json:"currentDate"`
	Timezone             string       `json:"timezone"`
	NameMaxLength        int          `json:"nameMaxLength"`
	NameGuidance         string       `json:"nameGuidance"`
	CodeMaxLength        int          `json:"codeMaxLength"`
	FieldCatalog         []string     `json:"fieldCatalog"`
	Capabilities         Capabilities `json:"capabilities"`
}
type DepartmentDetailInput struct{ DepartmentID, SelectedRecordID, SourceVersion string }
type DepartmentDetail struct {
	NameGuidance           string                         `json:"nameGuidance"`
	DepartmentID           string                         `json:"departmentId"`
	Code                   *string                        `json:"code,omitempty"`
	Enterprise             Enterprise                     `json:"enterprise"`
	CurrentDate            string                         `json:"currentDate"`
	Status                 string                         `json:"status"`
	Result                 string                         `json:"result"`
	ResolvedSourceVersion  string                         `json:"resolvedSourceVersion"`
	HeadSourceVersion      string                         `json:"headSourceVersion"`
	SelectedRecord         *DepartmentRecord              `json:"selectedRecord"`
	SelectionMissing       bool                           `json:"selectionMissing"`
	SelectedRecordIsFuture bool                           `json:"selectedRecordIsFuture"`
	Timeline               []DepartmentRecord             `json:"timeline"`
	Path                   DepartmentPath                 `json:"path"`
	ReadOnly               bool                           `json:"readOnly"`
	Capabilities           Capabilities                   `json:"capabilities"`
	AdditionContext        *DepartmentAdditionContext     `json:"additionContext,omitempty"`
	LifecycleContext       *DepartmentLifecycleContext    `json:"lifecycleContext,omitempty"`
	PairInsertion          *PairInsertionContext          `json:"pairInsertion,omitempty"`
	PairDeletion           *DepartmentPairDeletionContext `json:"pairDeletion,omitempty"`
}
type DepartmentSummary struct {
	ID            string               `json:"id"`
	Code          *string              `json:"code,omitempty"`
	Name          *string              `json:"name,omitempty"`
	ShortName     *string              `json:"shortName,omitempty"`
	ManagementBU  *DepartmentReference `json:"managementBu,omitempty"`
	Status        *string              `json:"status,omitempty"`
	EffectiveDate *string              `json:"effectiveDate,omitempty"`
	Path          DepartmentPath       `json:"path"`
}
type DepartmentListInput struct {
	ScopeDepartmentID  string
	IncludeDescendants *bool
	Q, Status, Sort    string
	Cursor             string
	Page, PageSize     int
	Filter             *ColumnFilter
}
type DepartmentList struct {
	HasMore    bool                `json:"hasMore,omitempty"`
	NextCursor string              `json:"nextCursor,omitempty"`
	Items      []DepartmentSummary `json:"items"`
	Count      int                 `json:"count"`
	Page       int                 `json:"page"`
	PageSize   int                 `json:"pageSize"`
	BusinessAt string              `json:"businessAt"`
	QueryKey   string              `json:"queryKey"`
}

// Navigation resolves one current neighbor from a stable object anchor.
type DepartmentNavigationInput struct {
	Query     DepartmentSelectionQuery `json:"query"`
	AnchorID  string                   `json:"anchorId"`
	Direction string                   `json:"direction"`
	Sort      string                   `json:"sort,omitempty"`
}
type DepartmentNavigationPage = NavigationResult
type DepartmentSelectionStartInput struct {
	Cursor    string                   `json:"cursor,omitempty"`
	Query     DepartmentSelectionQuery `json:"query"`
	Selection Selection                `json:"selection"`
	Page      int                      `json:"page"`
	PageSize  int                      `json:"pageSize"`
	Sort      string                   `json:"sort,omitempty"`
}
type DepartmentSelectionInput struct {
	Cursor           string                   `json:"cursor,omitempty"`
	Query            DepartmentSelectionQuery `json:"query"`
	DraftID          string                   `json:"draftId"`
	ExpectedRevision int64                    `json:"expectedRevision"`
	AddIDs           []string                 `json:"addIds,omitempty"`
	RemoveIDs        []string                 `json:"removeIds,omitempty"`
	RestoreIDs       []string                 `json:"restoreIds,omitempty"`
	Page             int                      `json:"page"`
	PageSize         int                      `json:"pageSize"`
	Sort             string                   `json:"sort,omitempty"`
}
type DepartmentSelectionPage struct {
	PageState   string                    `json:"pageState"`
	BusinessAt  string                    `json:"businessAt"`
	MemberIDs   []string                  `json:"memberIds"`
	Page        DepartmentList            `json:"page"`
	Summary     DepartmentSelectionResult `json:"summary"`
	SelectedIDs []string                  `json:"selectedIds"`
}
type DepartmentDraftInput struct {
	DraftID          string `json:"draftId"`
	ExpectedRevision int64  `json:"expectedRevision"`
}
type DepartmentExportInput struct {
	Query            DepartmentSelectionQuery `json:"query"`
	DraftID          string                   `json:"draftId"`
	ExpectedRevision int64                    `json:"expectedRevision"`
	Sort             string                   `json:"sort,omitempty"`
}
type DepartmentSelectionResult struct {
	QueryKey      string `json:"queryKey"`
	DraftID       string `json:"draftId"`
	Revision      int64  `json:"revision"`
	SelectedCount int    `json:"selectedCount"`
	CanExport     bool   `json:"canExport"`
}
type DepartmentExport struct {
	File       io.ReadCloser
	Count      int
	BusinessAt string
}

type DepartmentService interface {
	TreeNodes(context.Context, foundation.Credentials, DepartmentTreeInput) (DepartmentTreePage, error)
	TreeSearch(context.Context, foundation.Credentials, DepartmentTreeInput) (DepartmentTreePage, error)
	TreeLocation(context.Context, foundation.Credentials, string) (DepartmentTreeLocation, error)
	Changes(context.Context, foundation.Credentials, DepartmentChangeQuery) (DepartmentChanges, error)
	Preview(context.Context, foundation.Credentials, string, DepartmentPreviewInput) (DepartmentPreview, error)
	References(context.Context, foundation.Credentials, string, DepartmentReferenceQuery) (DepartmentReferences, error)
	Parents(context.Context, foundation.Credentials, DepartmentParentQuery) (DepartmentParents, error)
	Navigate(context.Context, foundation.Credentials, DepartmentNavigationInput) (DepartmentNavigationPage, error)
	CancelSelection(context.Context, foundation.Credentials, DepartmentDraftInput) error
	StartSelection(context.Context, foundation.Credentials, DepartmentSelectionStartInput) (DepartmentSelectionPage, error)
	ResolveSelection(context.Context, foundation.Credentials, DepartmentSelectionInput) (DepartmentSelectionPage, error)
	ExportSelection(context.Context, foundation.Credentials, DepartmentExportInput) (DepartmentExport, error)
	Execute(context.Context, foundation.Credentials, DepartmentCommand) (DepartmentResult, error)
	Resolve(context.Context, foundation.Credentials, string) (DepartmentResult, error)
	Context(context.Context, foundation.Credentials) (DepartmentContext, error)
	Detail(context.Context, foundation.Credentials, DepartmentDetailInput) (DepartmentDetail, error)
	List(context.Context, foundation.Credentials, DepartmentListInput) (DepartmentList, error)
}

type DepartmentItemResult struct {
	ItemID           string            `json:"itemId"`
	Outcome          string            `json:"outcome"`
	RecordID         string            `json:"recordId,omitempty"`
	BeforeRevisionID *string           `json:"beforeRevisionId"`
	AfterRevisionID  *string           `json:"afterRevisionId"`
	EffectiveDate    string            `json:"effectiveDate,omitempty"`
	ProcessingFacets []string          `json:"processingFacets"`
	BeforeRecord     *DepartmentRecord `json:"beforeRecord,omitempty"`
	AfterRecord      *DepartmentRecord `json:"afterRecord,omitempty"`
}
type DepartmentAdditionContext struct {
	AnchorRecordID       string            `json:"anchorRecordId"`
	Mode                 string            `json:"mode"`
	NextRecordID         string            `json:"nextRecordId,omitempty"`
	Available            bool              `json:"available"`
	MinEffectiveDate     string            `json:"minEffectiveDate,omitempty"`
	MaxEffectiveDate     string            `json:"maxEffectiveDate,omitempty"`
	DefaultEffectiveDate string            `json:"defaultEffectiveDate,omitempty"`
	Defaults             *DepartmentRecord `json:"defaults,omitempty"`
}
type DepartmentLifecycleContext struct {
	AnchorRecordID       string            `json:"anchorRecordId"`
	OperationType        string            `json:"operationType"`
	MinEffectiveDate     string            `json:"minEffectiveDate,omitempty"`
	MaxEffectiveDate     string            `json:"maxEffectiveDate,omitempty"`
	DefaultEffectiveDate string            `json:"defaultEffectiveDate,omitempty"`
	Defaults             *DepartmentRecord `json:"defaults,omitempty"`
}
type DepartmentPairDeletionContext struct {
	Deactivation DepartmentRecord       `json:"deactivation"`
	Reactivation DepartmentRecord       `json:"reactivation"`
	Changes      []DepartmentDataChange `json:"changes"`
}
type DepartmentDataChange struct {
	Start        *string          `json:"start,omitempty"`
	EndExclusive **string         `json:"endExclusive,omitempty"`
	EndInclusive **string         `json:"endInclusive,omitempty"`
	Before       DepartmentRecord `json:"before"`
	After        DepartmentRecord `json:"after"`
}

// Preview is a closed single-action union at the HTTP boundary.
type DepartmentPreviewInput struct {
	Action                 string                 `json:"action"`
	ExpectedSourceVersion  string                 `json:"expectedSourceVersion"`
	RecordID               string                 `json:"recordId,omitempty"`
	ExpectedRecordRevision string                 `json:"expectedRecordRevision,omitempty"`
	AnchorRecordID         string                 `json:"anchorRecordId,omitempty"`
	EffectiveDate          string                 `json:"effectiveDate,omitempty"`
	NewEffectiveDate       string                 `json:"newEffectiveDate,omitempty"`
	OperationType          string                 `json:"operationType,omitempty"`
	OperationReason        string                 `json:"operationReason,omitempty"`
	Classification         *Classification        `json:"classification,omitempty"`
	Changes                DepartmentChangesInput `json:"changes,omitempty"`
}
type DepartmentPreview struct {
	Valid      bool            `json:"valid"`
	NoChange   bool            `json:"noChange"`
	Impact     *TimelineImpact `json:"impact,omitempty"`
	ReadFields []string        `json:"readFields"`
}
type DepartmentReferenceClassification struct {
	OperationType   string `json:"operationType"`
	OperationReason string `json:"operationReason,omitempty"`
}
type DepartmentReferenceContext struct {
	Mode                   string                             `json:"mode"`
	DepartmentID           string                             `json:"departmentId,omitempty"`
	ExpectedSourceVersion  string                             `json:"expectedSourceVersion,omitempty"`
	Action                 string                             `json:"action,omitempty"`
	RecordID               string                             `json:"recordId,omitempty"`
	ExpectedRecordRevision string                             `json:"expectedRecordRevision,omitempty"`
	AnchorRecordID         string                             `json:"anchorRecordId,omitempty"`
	EffectiveDate          string                             `json:"effectiveDate,omitempty"`
	NewEffectiveDate       string                             `json:"newEffectiveDate,omitempty"`
	ManagementBUID         string                             `json:"managementBuId,omitempty"`
	OperationType          string                             `json:"operationType,omitempty"`
	OperationReason        string                             `json:"operationReason,omitempty"`
	Classification         *DepartmentReferenceClassification `json:"classification,omitempty"`
	Changes                DepartmentChangesInput             `json:"changes,omitempty"`
}
type DepartmentReferenceQuery struct {
	Context    DepartmentReferenceContext `json:"context"`
	Q          string                     `json:"q,omitempty"`
	SelectedID string                     `json:"selectedId,omitempty"`
	Page       int                        `json:"page"`
	PageSize   int                        `json:"pageSize"`
}
type DepartmentReferenceCandidate struct {
	ID          string  `json:"id"`
	Code        *string `json:"code,omitempty"`
	Name        *string `json:"name,omitempty"`
	Eligibility string  `json:"eligibility"`
}
type DepartmentReferences struct {
	SelectedReference *DepartmentReference           `json:"selectedReference,omitempty"`
	Applicable        bool                           `json:"applicable"`
	Items             []DepartmentReferenceCandidate `json:"items"`
	Count             int                            `json:"count"`
	Page              int                            `json:"page"`
	PageSize          int                            `json:"pageSize"`
}

type DepartmentParentQuery struct {
	Context    DepartmentReferenceContext `json:"context"`
	Q          string                     `json:"q,omitempty"`
	SelectedID string                     `json:"selectedId,omitempty"`
	Cursor     string                     `json:"cursor,omitempty"`
	PageSize   int                        `json:"pageSize"`
}
type DepartmentParents struct {
	SelectedReference *DepartmentReference           `json:"selectedReference,omitempty"`
	Applicable        bool                           `json:"applicable"`
	Items             []DepartmentReferenceCandidate `json:"items"`
	HasMore           bool                           `json:"hasMore"`
	NextCursor        string                         `json:"nextCursor,omitempty"`
}

type DepartmentChangeQuery struct {
	DepartmentID                               string
	Page, PageSize                             int
	ContextSourceVersion, ThroughSourceVersion string
	Filter                                     *ColumnFilter
}
type DepartmentChangeItem struct {
	RecordID      string            `json:"recordId"`
	ChangeKind    string            `json:"changeKind"`
	BeforeRef     *ChangeRef        `json:"beforeRef"`
	AfterRef      *ChangeRef        `json:"afterRef"`
	BeforeContext *DepartmentRecord `json:"beforeContext"`
	AfterContext  *DepartmentRecord `json:"afterContext"`
	Diffs         []ChangeDiff      `json:"diffs"`
}
type DepartmentChange struct {
	SourceVersion         string                 `json:"sourceVersion"`
	PreviousSourceVersion *string                `json:"previousSourceVersion"`
	RecordedAt            string                 `json:"recordedAt"`
	Operator              OperatorSnapshot       `json:"operator"`
	Action                string                 `json:"action"`
	Items                 []DepartmentChangeItem `json:"items"`
}
type DepartmentChanges struct {
	ReadFields           []string           `json:"readFields"`
	DetailReadFields     []string           `json:"detailReadFields"`
	Items                []DepartmentChange `json:"items"`
	Total                int                `json:"total"`
	Page                 int                `json:"page"`
	PageSize             int                `json:"pageSize"`
	ContextSourceVersion *string            `json:"contextSourceVersion"`
	ThroughSourceVersion string             `json:"throughSourceVersion"`
}

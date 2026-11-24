package public

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
)

type CreateInput struct {
	Code            string  `json:"code"`
	Name            string  `json:"name"`
	NameEn          *string `json:"nameEn,omitempty"`
	EffectiveDate   string  `json:"effectiveDate"`
	OperationType   string  `json:"operationType"`
	OperationReason string  `json:"operationReason"`
}
type CreateCommand struct {
	Input        CreateInput
	Digest       []byte
	SubmissionID string
	Fields       []string
}
type Enterprise struct {
	ActivePeriods []EnterpriseActivePeriod `json:"-"`
	Names         []EnterpriseName         `json:"-"`
	ID            string                   `json:"id"`
	Code          string                   `json:"code,omitempty"`
	Name          string                   `json:"name"`
	ValidFrom     string                   `json:"validFrom"`
}
type EnterpriseName struct{ Date, Name string }
type Capabilities struct {
	Correction            *CorrectionCapabilities `json:"correction,omitempty"`
	Redate                bool                    `json:"redate"`
	Delete                bool                    `json:"delete"`
	Deactivate            bool                    `json:"deactivate"`
	Reactivate            bool                    `json:"reactivate"`
	DeactivateWriteFields []string                `json:"deactivateWriteFields"`
	ReactivateWriteFields []string                `json:"reactivateWriteFields"`
	Add                   bool                    `json:"add"`
	Correct               bool                    `json:"correct"`
	AddWriteFields        []string                `json:"addWriteFields"`
	CorrectWriteFields    []string                `json:"correctWriteFields"`
	Create                bool                    `json:"create"`
	History               bool                    `json:"history"`
	ReadFields            []string                `json:"readFields"`
	WriteFields           []string                `json:"writeFields"`
}
type Context struct {
	DefaultEffectiveDate string       `json:"defaultEffectiveDate"`
	Enterprise           Enterprise   `json:"enterprise"`
	CurrentDate          string       `json:"currentDate"`
	Timezone             string       `json:"timezone"`
	Capabilities         Capabilities `json:"capabilities"`
	NameMaxLength        int          `json:"nameMaxLength"`
	NameGuidance         string       `json:"nameGuidance"`
	CodeMaxLength        int          `json:"codeMaxLength"`
}
type Record struct {
	OperationSummary *OperationSummary `json:"operationSummary,omitempty"`
	ItemID           string            `json:"itemId,omitempty"`
	Deleted          bool              `json:"deleted"`
	ID               string            `json:"recordId,omitempty"`
	Revision         string            `json:"revision,omitempty"`
	EndExclusive     *string           `json:"endExclusive"`
	EndInclusive     *string           `json:"endInclusive"`
	EffectiveDate    string            `json:"effectiveDate,omitempty"`
	Name             *string           `json:"name,omitempty"`
	NameEn           *string           `json:"nameEn,omitempty"`
	State            string            `json:"state"`
	OperationType    string            `json:"operationType"`
	OperationReason  string            `json:"operationReason"`
	TypeLabel        string            `json:"typeLabel"`
	ReasonLabel      string            `json:"reasonLabel"`
}
type Summary struct {
	ID            string  `json:"id"`
	Code          string  `json:"code,omitempty"`
	Name          *string `json:"name,omitempty"`
	Status        string  `json:"status"`
	EffectiveDate string  `json:"effectiveDate,omitempty"`
}

// BusinessAt supports internal dated reads. HTTP never supplies it and defaults to the server day.
type ListInput struct {
	BusinessAt, Q, Status, Sort string
	Page, PageSize              int
	Filter                      *ColumnFilter
}

// ColumnFilter is one organization query predicate, independent of the UI widget model.
type ColumnFilter struct{ Field, Operator, Value string }
type List struct {
	QueryKey         string    `json:"queryKey"`
	SelectionVersion string    `json:"selectionVersion"`
	Items            []Summary `json:"items"`
	Count            int       `json:"count"`
	Page             int       `json:"page"`
	PageSize         int       `json:"pageSize"`
	BusinessAt       string    `json:"businessAt"`
	CurrentDate      string    `json:"currentDate"`
}

// BusinessAt is internal dated-read context; ordinary HTTP callers only select records and sources.
type DetailInput struct{ ID, BusinessAt, SourceVersion, SelectedRecordID string }
type Detail struct {
	LifecycleContext       *LifecycleContext     `json:"lifecycleContext,omitempty"`
	PairInsertion          *PairInsertionContext `json:"pairInsertion,omitempty"`
	PairDeletion           *PairDeletionContext  `json:"pairDeletion,omitempty"`
	AdditionContext        *AdditionContext      `json:"additionContext,omitempty"`
	NameGuidance           string                `json:"nameGuidance"`
	EvaluatedAt            *string               `json:"evaluatedAt,omitempty"`
	EvaluatedBusinessDate  *string               `json:"evaluatedBusinessDate,omitempty"`
	ID                     string                `json:"id"`
	Code                   string                `json:"code,omitempty"`
	Enterprise             Enterprise            `json:"enterprise"`
	BusinessAt             string                `json:"businessAt"`
	CurrentDate            string                `json:"currentDate"`
	Status                 string                `json:"status"`
	Result                 string                `json:"result"`
	ViewFact               *Record               `json:"viewFact"`
	SelectedRecordIsFuture bool                  `json:"selectedRecordIsFuture"`
	SelectedRecord         *Record               `json:"selectedRecord"`
	Timeline               []Record              `json:"timeline"`
	HeadSourceVersion      *string               `json:"headSourceVersion,omitempty"`
	ResolvedSourceVersion  *string               `json:"resolvedSourceVersion"`
	RecordedAt             *string               `json:"recordedAt,omitempty"`
	Actor                  *string               `json:"actor,omitempty"`
	ReadOnly               bool                  `json:"readOnly"`
	SelectionMissing       bool                  `json:"selectionMissing"`
	Capabilities           Capabilities          `json:"capabilities"`
}
type Result struct {
	RecordedAt            *string         `json:"recordedAt,omitempty"`
	Items                 []ItemResult    `json:"items,omitempty"`
	CurrentReadFields     []string        `json:"-"` // Response-only current object scope, never stored in the terminal.
	SubmissionID          string          `json:"submissionId"`
	Outcome               string          `json:"outcome"`
	BUID                  *string         `json:"buId,omitempty"`
	SourceVersion         *string         `json:"sourceVersion,omitempty"`
	TargetRecordID        *string         `json:"targetRecordId,omitempty"`
	TargetRevision        *string         `json:"targetRevision,omitempty"`
	SelectedRecordID      *string         `json:"selectedRecordId,omitempty"`
	EffectiveDate         *string         `json:"effectiveDate,omitempty"`
	PreviousEffectiveDate *string         `json:"previousEffectiveDate,omitempty"`
	ErrorCode             *string         `json:"errorCode,omitempty"`
	Action                string          `json:"action,omitempty"`
	ObservedSourceVersion *string         `json:"observedSourceVersion,omitempty"`
	ExpectedSourceVersion *string         `json:"expectedSourceVersion,omitempty"`
	EvaluatedAt           *string         `json:"evaluatedAt,omitempty"`
	EvaluatedBusinessDate *string         `json:"evaluatedBusinessDate,omitempty"`
	CurrentStatus         string          `json:"currentStatus,omitempty"`
	Impact                *TimelineImpact `json:"impact,omitempty"`
	FailureDetails        *FailureDetails `json:"failureDetails,omitempty"`
	Replayed              bool            `json:"replayed"`
	CurrentChanged        bool            `json:"currentChanged"`
}
type Reason struct {
	Default bool   `json:"default"`
	Code    string `json:"code,omitempty"`
	Name    string `json:"name"`
}
type Service interface {
	Navigate(context.Context, foundation.Credentials, NavigationInput) (NavigationResult, error)
	ResolveSelection(context.Context, foundation.Credentials, SelectionRequest) (SelectionResult, error)
	ExportSelection(context.Context, foundation.Credentials, ExportRequest) (Export, error)
	Maintain(context.Context, foundation.Credentials, MaintainCommand) (Result, error)
	Preview(context.Context, foundation.Credentials, string, PreviewInput) (Preview, error)
	Context(context.Context, foundation.Credentials) (Context, error)
	Create(context.Context, foundation.Credentials, CreateCommand) (Result, error)
	List(context.Context, foundation.Credentials, ListInput) (List, error)
	Detail(context.Context, foundation.Credentials, DetailInput) (Detail, error)
	Resolve(context.Context, foundation.Credentials, string) (Result, error)
	Changes(context.Context, foundation.Credentials, ChangeQuery) (Changes, error)
}

type ChangesInput map[string]*string
type AddInput struct {
	AnchorRecordID        string       `json:"anchorRecordId"`
	ExpectedSourceVersion string       `json:"expectedSourceVersion"`
	EffectiveDate         string       `json:"effectiveDate"`
	Changes               ChangesInput `json:"changes"`
	OperationType         string       `json:"operationType"`
	OperationReason       string       `json:"operationReason"`
}
type DeactivateInput struct {
	AnchorRecordID        string `json:"anchorRecordId"`
	ExpectedSourceVersion string `json:"expectedSourceVersion"`
	EffectiveDate         string `json:"effectiveDate"`
	OperationType         string `json:"operationType"`
	OperationReason       string `json:"operationReason"`
}
type ReactivateInput struct {
	AnchorRecordID        string       `json:"anchorRecordId"`
	ExpectedSourceVersion string       `json:"expectedSourceVersion"`
	EffectiveDate         string       `json:"effectiveDate"`
	OperationType         string       `json:"operationType"`
	OperationReason       string       `json:"operationReason"`
	Changes               ChangesInput `json:"changes,omitempty"`
}
type CorrectInput struct {
	ExpectedSourceVersion  string          `json:"expectedSourceVersion"`
	ExpectedRecordRevision string          `json:"expectedRecordRevision"`
	Changes                ChangesInput    `json:"changes,omitempty"`
	NewEffectiveDate       string          `json:"newEffectiveDate,omitempty"`
	Classification         *Classification `json:"classification,omitempty"`
}
type DeleteInput struct {
	ExpectedSourceVersion  string `json:"expectedSourceVersion"`
	ExpectedRecordRevision string `json:"expectedRecordRevision"`
}
type RedateInput struct {
	ExpectedSourceVersion  string `json:"expectedSourceVersion"`
	ExpectedRecordRevision string `json:"expectedRecordRevision"`
	NewEffectiveDate       string `json:"newEffectiveDate"`
}
type MaintainCommand struct {
	InsertPair                           InsertLifecyclePairInput
	DeletePair                           DeleteLifecyclePairInput
	Action, BUID, RecordID, SubmissionID string
	Add                                  AddInput
	Correct                              CorrectInput
	Delete                               DeleteInput
	Redate                               RedateInput
	Fields                               []string
	Digest                               []byte
}
type PreviewInput struct {
	AnchorRecordID         string          `json:"anchorRecordId,omitempty"`
	Action                 string          `json:"action"`
	ExpectedSourceVersion  string          `json:"expectedSourceVersion"`
	RecordID               string          `json:"recordId,omitempty"`
	ExpectedRecordRevision string          `json:"expectedRecordRevision,omitempty"`
	EffectiveDate          string          `json:"effectiveDate,omitempty"`
	NewEffectiveDate       string          `json:"newEffectiveDate,omitempty"`
	OperationType          string          `json:"operationType,omitempty"`
	OperationReason        string          `json:"operationReason,omitempty"`
	Changes                ChangesInput    `json:"changes,omitempty"`
	Classification         *Classification `json:"classification,omitempty"`
}

// PreviewDateRange describes the finite gap created by moving a first date later.
type PreviewDateRange struct {
	Start        string `json:"start"`
	EndExclusive string `json:"endExclusive"`
	EndInclusive string `json:"endInclusive"`
}

type DateRange struct {
	Start string  `json:"start"`
	End   *string `json:"endExclusive"`
}
type Structure struct {
	RecordID       string   `json:"recordId,omitempty"`
	ItemID         string   `json:"itemId,omitempty"`
	BeforeRevision *string  `json:"beforeRevisionId"`
	AfterRevision  *string  `json:"afterRevisionId"`
	Facets         []string `json:"facets"`
}
type Period struct {
	RecordID string     `json:"recordId,omitempty"`
	ItemID   string     `json:"itemId,omitempty"`
	Before   *DateRange `json:"before"`
	After    *DateRange `json:"after"`
}
type TimelineImpact struct {
	AffectedItemIDs    []string    `json:"affectedItemIds,omitempty"`
	DataAffectedRanges []DateRange `json:"dataAffectedRanges"`
	StructuralChanges  []Structure `json:"structuralChanges"`
	AffectedRecordIDs  []string    `json:"affectedRecordIds"`
	PeriodChanges      []Period    `json:"periodChanges"`
}
type Location struct {
	ItemID    string `json:"itemId,omitempty"`
	FieldPath string `json:"fieldPath,omitempty"`
}
type Issue struct {
	Code      string     `json:"code"`
	Locations []Location `json:"locations"`
}
type FailureDetails struct {
	FormatVersion int     `json:"formatVersion"`
	Issues        []Issue `json:"issues"`
}
type Preview struct {
	ReadFields            []string          `json:"readFields"`
	NewNoFactRange        *PreviewDateRange `json:"newNoFactRange,omitempty"`
	AfterRecord           *Record           `json:"afterRecord,omitempty"`
	PreviousCurrentStatus string            `json:"previousCurrentStatus,omitempty"`
	Outcome               string            `json:"outcome"`
	ExpectedSourceVersion string            `json:"expectedSourceVersion"`
	Impact                TimelineImpact    `json:"impact"`
	EvaluatedAt           string            `json:"evaluatedAt"`
	EvaluatedBusinessDate string            `json:"evaluatedBusinessDate"`
	CurrentStatus         string            `json:"currentStatus"`
}

type NavigationInput struct {
	Query     SelectionQuery `json:"query"`
	AnchorID  string         `json:"anchorId"`
	Direction string         `json:"direction"`
	Sort      string         `json:"sort,omitempty"`
}
type NavigationResult struct {
	TargetID   *string `json:"targetId"`
	PreviousID *string `json:"previousId,omitempty"`
	NextID     *string `json:"nextId,omitempty"`
	Position   int     `json:"position,omitempty"`
	Count      int     `json:"count,omitempty"`
}

type EnterpriseActivePeriod struct{ Start, End string }

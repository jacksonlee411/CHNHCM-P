package public

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"io"
)

type LegalEntityContext struct {
	CurrentDate          string       `json:"currentDate"`
	DefaultEffectiveDate string       `json:"defaultEffectiveDate"`
	Timezone             string       `json:"timezone"`
	NameMaxLength        int          `json:"nameMaxLength"`
	NameGuidance         string       `json:"nameGuidance"`
	Enterprise           Enterprise   `json:"enterprise"`
	Capabilities         Capabilities `json:"capabilities"`
}
type LegalEntityRecord struct {
	ID                       string            `json:"recordId"`
	Revision                 string            `json:"revision"`
	Deleted                  bool              `json:"deleted"`
	EffectiveDate            *string           `json:"effectiveDate,omitempty"`
	EndExclusive             **string          `json:"endExclusive,omitempty"`
	EndInclusive             **string          `json:"endInclusive,omitempty"`
	Name                     *string           `json:"name,omitempty"`
	NameEn                   **string          `json:"nameEn,omitempty"`
	CountryRegionID          *string           `json:"countryRegionId,omitempty"`
	CountryName              *string           `json:"countryName,omitempty"`
	AdoptionDate             *string           `json:"adoptionDate,omitempty"`
	SubjectType              *string           `json:"subjectType,omitempty"`
	RegistrationType         **string          `json:"registrationType,omitempty"`
	RegistrationNumber       **string          `json:"registrationNumber,omitempty"`
	EstablishingOrganization **string          `json:"establishingOrganization,omitempty"`
	CountryRestricted        bool              `json:"countryRestricted"`
	State                    *string           `json:"state,omitempty"`
	OperationType            *string           `json:"operationType,omitempty"`
	OperationReason          *string           `json:"operationReason,omitempty"`
	TypeLabel                *string           `json:"typeLabel,omitempty"`
	ReasonLabel              *string           `json:"reasonLabel,omitempty"`
	OperationSummary         *OperationSummary `json:"operationSummary,omitempty"`
}
type LegalEntitySummary struct {
	ID                       string  `json:"id"`
	Code                     *string `json:"code,omitempty"`
	Name                     *string `json:"name,omitempty"`
	NameEn                   *string `json:"nameEn,omitempty"`
	CountryRegionID          *string `json:"countryRegionId,omitempty"`
	SubjectType              *string `json:"subjectType,omitempty"`
	RegistrationType         *string `json:"registrationType,omitempty"`
	RegistrationNumber       *string `json:"registrationNumber,omitempty"`
	EstablishingOrganization *string `json:"establishingOrganization,omitempty"`
	Status                   *string `json:"status,omitempty"`
	EffectiveDate            *string `json:"effectiveDate,omitempty"`
}
type LegalEntityList struct {
	QueryKey    string               `json:"queryKey"`
	BusinessAt  string               `json:"businessAt"`
	Items       []LegalEntitySummary `json:"items"`
	Count       int                  `json:"count"`
	Page        int                  `json:"page"`
	PageSize    int                  `json:"pageSize"`
	CurrentDate string               `json:"currentDate"`
}
type LegalEntityAdditionContext struct {
	Available            bool              `json:"available"`
	AnchorRecordID       string            `json:"anchorRecordId"`
	Mode                 string            `json:"mode"`
	MinEffectiveDate     string            `json:"minEffectiveDate"`
	MaxEffectiveDate     *string           `json:"maxEffectiveDate"`
	DefaultEffectiveDate string            `json:"defaultEffectiveDate"`
	Defaults             LegalEntityRecord `json:"defaults"`
}
type LegalEntityLifecycleContext struct {
	AnchorRecordID       string            `json:"anchorRecordId"`
	OperationType        string            `json:"operationType"`
	MinEffectiveDate     string            `json:"minEffectiveDate"`
	MaxEffectiveDate     *string           `json:"maxEffectiveDate"`
	DefaultEffectiveDate string            `json:"defaultEffectiveDate"`
	Defaults             LegalEntityRecord `json:"defaults"`
}
type LegalEntityDetail struct {
	CanReadPurposes        bool                         `json:"canReadPurposes,omitempty"`
	PairInsertion          *PairInsertionContext        `json:"pairInsertion,omitempty"`
	PairDeletion           *LegalEntityPairContext      `json:"pairDeletion,omitempty"`
	ID                     string                       `json:"id"`
	Code                   *string                      `json:"code,omitempty"`
	CurrentDate            string                       `json:"currentDate"`
	Status                 *string                      `json:"status,omitempty"`
	NameGuidance           string                       `json:"nameGuidance"`
	HeadSourceVersion      string                       `json:"headSourceVersion"`
	ResolvedSourceVersion  string                       `json:"resolvedSourceVersion"`
	SelectedRecord         *LegalEntityRecord           `json:"selectedRecord"`
	Timeline               []LegalEntityRecord          `json:"timeline"`
	ReadOnly               bool                         `json:"readOnly"`
	SelectionMissing       bool                         `json:"selectionMissing"`
	SelectedRecordIsFuture bool                         `json:"selectedRecordIsFuture"`
	Capabilities           Capabilities                 `json:"capabilities"`
	AdditionContext        *LegalEntityAdditionContext  `json:"additionContext,omitempty"`
	LifecycleContext       *LegalEntityLifecycleContext `json:"lifecycleContext,omitempty"`
}
type LegalEntityResult struct {
	SubmissionID     string  `json:"submissionId"`
	Outcome          string  `json:"outcome"`
	LegalEntityID    *string `json:"legalEntityId,omitempty"`
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
type LegalEntityChangesInput map[string]*string
type LegalEntityAddInput struct {
	ExpectedSourceVersion string                  `json:"expectedSourceVersion"`
	AnchorRecordID        string                  `json:"anchorRecordId"`
	EffectiveDate         string                  `json:"effectiveDate"`
	OperationType         string                  `json:"operationType"`
	OperationReason       string                  `json:"operationReason"`
	Changes               LegalEntityChangesInput `json:"changes,omitempty"`
}
type LegalEntityCorrectInput struct {
	NewEffectiveDate       string                  `json:"newEffectiveDate,omitempty"`
	Classification         *Classification         `json:"classification,omitempty"`
	ExpectedSourceVersion  string                  `json:"expectedSourceVersion"`
	ExpectedRecordRevision string                  `json:"expectedRecordRevision"`
	Changes                LegalEntityChangesInput `json:"changes,omitempty"`
}

type LegalEntityCreateInput struct {
	Code                     string  `json:"code"`
	Name                     string  `json:"name"`
	NameEn                   *string `json:"nameEn,omitempty"`
	EffectiveDate            string  `json:"effectiveDate,omitempty"`
	OperationType            string  `json:"operationType"`
	OperationReason          string  `json:"operationReason"`
	CountryRegionID          string  `json:"countryRegionId"`
	SubjectType              string  `json:"subjectType"`
	RegistrationType         *string `json:"registrationType,omitempty"`
	RegistrationNumber       *string `json:"registrationNumber,omitempty"`
	EstablishingOrganization *string `json:"establishingOrganization,omitempty"`
}
type LegalEntityCommand struct {
	Action, LegalEntityID, RecordID, SubmissionID string
	Digest                                        []byte
	Create                                        LegalEntityCreateInput
	Delete                                        DeleteInput
	Redate                                        RedateInput
	InsertPair                                    InsertLifecyclePairInput
	DeletePair                                    DeleteLifecyclePairInput
	MaintainPair                                  LegalEntityMaintainPairInput
	Add                                           LegalEntityAddInput
	Correct                                       LegalEntityCorrectInput
}
type LegalEntityPreviewInput struct {
	NewEffectiveDate       string                  `json:"newEffectiveDate,omitempty"`
	Classification         *Classification         `json:"classification,omitempty"`
	Action                 string                  `json:"action"`
	ExpectedSourceVersion  string                  `json:"expectedSourceVersion"`
	AnchorRecordID         string                  `json:"anchorRecordId,omitempty"`
	RecordID               string                  `json:"recordId,omitempty"`
	ExpectedRecordRevision string                  `json:"expectedRecordRevision,omitempty"`
	EffectiveDate          string                  `json:"effectiveDate,omitempty"`
	OperationType          string                  `json:"operationType,omitempty"`
	OperationReason        string                  `json:"operationReason,omitempty"`
	Changes                LegalEntityChangesInput `json:"changes,omitempty"`
}
type LegalEntityPreview struct {
	Impact      *TimelineImpact    `json:"impact,omitempty"`
	ReadFields  []string           `json:"readFields"`
	AfterRecord *LegalEntityRecord `json:"afterRecord,omitempty"`
}
type LegalEntityChangeItem struct {
	ChangeKind    string             `json:"changeKind"`
	BeforeRef     *ChangeRef         `json:"beforeRef"`
	AfterRef      *ChangeRef         `json:"afterRef"`
	Diffs         []ChangeDiff       `json:"diffs"`
	RecordID      string             `json:"recordId"`
	BeforeContext *LegalEntityRecord `json:"beforeContext"`
	AfterContext  *LegalEntityRecord `json:"afterContext"`
}
type LegalEntityChange struct {
	ID                    string                  `json:"id"`
	SourceVersion         string                  `json:"sourceVersion"`
	PreviousSourceVersion *string                 `json:"previousSourceVersion"`
	RecordedAt            string                  `json:"recordedAt"`
	Action                string                  `json:"action"`
	Operator              OperatorSnapshot        `json:"operator"`
	Items                 []LegalEntityChangeItem `json:"items"`
}
type LegalEntityChanges struct {
	ReadFields           []string            `json:"readFields"`
	DetailReadFields     []string            `json:"detailReadFields"`
	ContextSourceVersion *string             `json:"contextSourceVersion"`
	ThroughSourceVersion string              `json:"throughSourceVersion"`
	Items                []LegalEntityChange `json:"items"`
	Total                int                 `json:"total"`
	Page                 int                 `json:"page"`
	PageSize             int                 `json:"pageSize"`
}
type LegalEntityService interface {
	ReferenceContext(context.Context, foundation.Credentials, LegalEntityReferenceInput) (LegalEntityReferenceContext, error)
	StartSelection(context.Context, foundation.Credentials, LegalEntitySelectionStartInput) (LegalEntitySelectionPage, error)
	ResolveSelection(context.Context, foundation.Credentials, LegalEntitySelectionInput) (LegalEntitySelectionPage, error)
	CancelSelection(context.Context, foundation.Credentials, LegalEntityDraftInput) error
	ExportSelection(context.Context, foundation.Credentials, LegalEntityExportInput) (LegalEntityExport, error)
	Context(context.Context, foundation.Credentials) (LegalEntityContext, error)
	List(context.Context, foundation.Credentials, LegalEntityQuery) (LegalEntityList, error)
	Detail(context.Context, foundation.Credentials, DetailInput) (LegalEntityDetail, error)
	Maintain(context.Context, foundation.Credentials, LegalEntityCommand) (LegalEntityResult, error)
	Preview(context.Context, foundation.Credentials, string, LegalEntityPreviewInput) (LegalEntityPreview, error)
	Resolve(context.Context, foundation.Credentials, string) (LegalEntityResult, error)
	Changes(context.Context, foundation.Credentials, ChangeQuery) (LegalEntityChanges, error)
	Navigate(context.Context, foundation.Credentials, LegalEntityNavigationInput) (NavigationResult, error)
}

type LegalEntityQuery struct {
	ListInput
	CountryRegionID string `json:"countryRegionId,omitempty"`
}
type LegalEntityNavigationInput struct {
	Query     LegalEntitySelectionQuery `json:"query"`
	Sort      string                    `json:"sort,omitempty"`
	AnchorID  string                    `json:"anchorId"`
	Direction string                    `json:"direction"`
}
type LegalEntitySelectionQuery struct {
	Q               string `json:"q,omitempty"`
	Status          string `json:"status,omitempty"`
	FilterField     string `json:"filterField,omitempty"`
	FilterOperator  string `json:"filterOperator,omitempty"`
	FilterValue     string `json:"filterValue,omitempty"`
	CountryRegionID string `json:"countryRegionId,omitempty"`
}
type LegalEntityDataChange struct {
	Start        *string           `json:"start,omitempty"`
	EndExclusive **string          `json:"endExclusive,omitempty"`
	EndInclusive **string          `json:"endInclusive,omitempty"`
	Before       LegalEntityRecord `json:"before"`
	After        LegalEntityRecord `json:"after"`
}
type LegalEntityPairContext struct {
	Changes      []LegalEntityDataChange `json:"changes"`
	Deactivation LegalEntityRecord       `json:"deactivation"`
	Reactivation LegalEntityRecord       `json:"reactivation"`
}
type LegalEntityMaintainPairInput struct {
	ExpectedSourceVersion string                  `json:"expectedSourceVersion"`
	DeactivationRecordID  string                  `json:"deactivationRecordId"`
	DeactivationRevision  string                  `json:"deactivationRevision"`
	ReactivationRecordID  string                  `json:"reactivationRecordId"`
	ReactivationRevision  string                  `json:"reactivationRevision"`
	DeactivationDate      string                  `json:"deactivationDate"`
	ReactivationDate      string                  `json:"reactivationDate"`
	DeactivationReason    string                  `json:"deactivationReason"`
	ReactivationReason    string                  `json:"reactivationReason"`
	Changes               LegalEntityChangesInput `json:"changes,omitempty"`
}

type LegalEntitySelectionStartInput struct {
	Query     LegalEntitySelectionQuery `json:"query"`
	Selection Selection                 `json:"selection"`
	Page      int                       `json:"page"`
	PageSize  int                       `json:"pageSize"`
	Sort      string                    `json:"sort,omitempty"`
}
type LegalEntitySelectionInput struct {
	Query            LegalEntitySelectionQuery `json:"query"`
	DraftID          string                    `json:"draftId"`
	ExpectedRevision int64                     `json:"expectedRevision"`
	AddIDs           []string                  `json:"addIds,omitempty"`
	RemoveIDs        []string                  `json:"removeIds,omitempty"`
	RestoreIDs       []string                  `json:"restoreIds,omitempty"`
	Page             int                       `json:"page"`
	PageSize         int                       `json:"pageSize"`
	Sort             string                    `json:"sort,omitempty"`
}
type LegalEntitySelectionPage struct {
	PageState   string                     `json:"pageState"`
	BusinessAt  string                     `json:"businessAt"`
	MemberIDs   []string                   `json:"memberIds"`
	Page        LegalEntityList            `json:"page"`
	Summary     LegalEntitySelectionResult `json:"summary"`
	SelectedIDs []string                   `json:"selectedIds"`
}
type LegalEntityDraftInput struct {
	DraftID          string `json:"draftId"`
	ExpectedRevision int64  `json:"expectedRevision"`
}
type LegalEntityExportInput struct {
	Query            LegalEntitySelectionQuery `json:"query"`
	DraftID          string                    `json:"draftId"`
	ExpectedRevision int64                     `json:"expectedRevision"`
	Sort             string                    `json:"sort,omitempty"`
}
type LegalEntitySelectionResult struct {
	QueryKey      string `json:"queryKey"`
	DraftID       string `json:"draftId"`
	Revision      int64  `json:"revision"`
	SelectedCount int    `json:"selectedCount"`
	CanExport     bool   `json:"canExport"`
}
type LegalEntityExport struct {
	File       io.ReadCloser
	Count      int
	BusinessAt string
}

type LegalEntityReferenceInput struct {
	Action                 string                  `json:"action"`
	LegalEntityID          string                  `json:"legalEntityId,omitempty"`
	ExpectedSourceVersion  string                  `json:"expectedSourceVersion,omitempty"`
	RecordID               string                  `json:"recordId,omitempty"`
	ExpectedRecordRevision string                  `json:"expectedRecordRevision,omitempty"`
	AnchorRecordID         string                  `json:"anchorRecordId,omitempty"`
	EffectiveDate          string                  `json:"effectiveDate,omitempty"`
	NewEffectiveDate       string                  `json:"newEffectiveDate,omitempty"`
	Classification         *Classification         `json:"classification,omitempty"`
	Changes                LegalEntityChangesInput `json:"changes,omitempty"`
}
type LegalEntityCountryOption struct {
	ID   string `json:"id"`
	Code string `json:"code,omitempty"`
	Name string `json:"name"`
}
type LegalEntityReferenceContext struct {
	NewAdoption       bool                       `json:"newAdoption"`
	AdoptedDate       string                     `json:"adoptedDate"`
	Countries         []LegalEntityCountryOption `json:"countries"`
	CountryName       *string                    `json:"countryName,omitempty"`
	CountryRestricted bool                       `json:"countryRestricted"`
}

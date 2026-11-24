package public

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"io"
)

type LocationContext struct {
	CurrentDate          string       `json:"currentDate"`
	DefaultEffectiveDate string       `json:"defaultEffectiveDate"`
	Timezone             string       `json:"timezone"`
	NameMaxLength        int          `json:"nameMaxLength"`
	NameGuidance         string       `json:"nameGuidance"`
	Enterprise           Enterprise   `json:"enterprise"`
	Capabilities         Capabilities `json:"capabilities"`
}
type LocationRecord struct {
	ID                string            `json:"recordId"`
	Revision          string            `json:"revision"`
	Deleted           bool              `json:"deleted"`
	EffectiveDate     *string           `json:"effectiveDate,omitempty"`
	EndExclusive      **string          `json:"endExclusive,omitempty"`
	EndInclusive      **string          `json:"endInclusive,omitempty"`
	Name              *string           `json:"name,omitempty"`
	NameEn            **string          `json:"nameEn,omitempty"`
	CountryRegionID   *string           `json:"countryRegionId,omitempty"`
	GeoAreaID         **string          `json:"geoAreaId,omitempty"`
	CountryName       *string           `json:"countryName,omitempty"`
	GeoAreaPath       []GeoAreaPathNode `json:"geoAreaPath,omitempty"`
	AdoptionDate      *string           `json:"adoptionDate,omitempty"`
	AddressLine       *string           `json:"addressLine,omitempty"`
	PostalCode        **string          `json:"postalCode,omitempty"`
	AddressRestricted bool              `json:"addressRestricted"`
	State             *string           `json:"state,omitempty"`
	OperationType     *string           `json:"operationType,omitempty"`
	OperationReason   *string           `json:"operationReason,omitempty"`
	TypeLabel         *string           `json:"typeLabel,omitempty"`
	ReasonLabel       *string           `json:"reasonLabel,omitempty"`
	OperationSummary  *OperationSummary `json:"operationSummary,omitempty"`
}
type LocationSummary struct {
	ID              string  `json:"id"`
	Code            *string `json:"code,omitempty"`
	Name            *string `json:"name,omitempty"`
	NameEn          *string `json:"nameEn,omitempty"`
	CountryRegionID *string `json:"countryRegionId,omitempty"`
	CountryName     *string `json:"countryName,omitempty"`
	GeoAreaID       *string `json:"geoAreaId,omitempty"`
	AddressLine     *string `json:"addressLine,omitempty"`
	PostalCode      *string `json:"postalCode,omitempty"`
	Status          *string `json:"status,omitempty"`
	EffectiveDate   *string `json:"effectiveDate,omitempty"`
}
type LocationList struct {
	QueryKey    string            `json:"queryKey"`
	BusinessAt  string            `json:"businessAt"`
	Items       []LocationSummary `json:"items"`
	Count       int               `json:"count"`
	Page        int               `json:"page"`
	PageSize    int               `json:"pageSize"`
	CurrentDate string            `json:"currentDate"`
}
type LocationAdditionContext struct {
	Available            bool           `json:"available"`
	AnchorRecordID       string         `json:"anchorRecordId"`
	Mode                 string         `json:"mode"`
	MinEffectiveDate     string         `json:"minEffectiveDate"`
	MaxEffectiveDate     *string        `json:"maxEffectiveDate"`
	DefaultEffectiveDate string         `json:"defaultEffectiveDate"`
	Defaults             LocationRecord `json:"defaults"`
}
type LocationLifecycleContext struct {
	AnchorRecordID       string         `json:"anchorRecordId"`
	OperationType        string         `json:"operationType"`
	MinEffectiveDate     string         `json:"minEffectiveDate"`
	MaxEffectiveDate     *string        `json:"maxEffectiveDate"`
	DefaultEffectiveDate string         `json:"defaultEffectiveDate"`
	Defaults             LocationRecord `json:"defaults"`
}
type LocationDetail struct {
	PairInsertion          *PairInsertionContext     `json:"pairInsertion,omitempty"`
	PairDeletion           *LocationPairContext      `json:"pairDeletion,omitempty"`
	ID                     string                    `json:"id"`
	Code                   *string                   `json:"code,omitempty"`
	CurrentDate            string                    `json:"currentDate"`
	Status                 *string                   `json:"status,omitempty"`
	NameGuidance           string                    `json:"nameGuidance"`
	HeadSourceVersion      string                    `json:"headSourceVersion"`
	ResolvedSourceVersion  string                    `json:"resolvedSourceVersion"`
	SelectedRecord         *LocationRecord           `json:"selectedRecord"`
	Timeline               []LocationRecord          `json:"timeline"`
	ReadOnly               bool                      `json:"readOnly"`
	SelectionMissing       bool                      `json:"selectionMissing"`
	SelectedRecordIsFuture bool                      `json:"selectedRecordIsFuture"`
	Capabilities           Capabilities              `json:"capabilities"`
	AdditionContext        *LocationAdditionContext  `json:"additionContext,omitempty"`
	LifecycleContext       *LocationLifecycleContext `json:"lifecycleContext,omitempty"`
}
type LocationResult struct {
	SubmissionID     string  `json:"submissionId"`
	Outcome          string  `json:"outcome"`
	LocationID       *string `json:"locationId,omitempty"`
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
type LocationChangesInput map[string]*string
type LocationAddInput struct {
	ExpectedSourceVersion string               `json:"expectedSourceVersion"`
	AnchorRecordID        string               `json:"anchorRecordId"`
	EffectiveDate         string               `json:"effectiveDate"`
	OperationType         string               `json:"operationType"`
	OperationReason       string               `json:"operationReason"`
	Changes               LocationChangesInput `json:"changes,omitempty"`
}
type LocationCorrectInput struct {
	NewEffectiveDate       string               `json:"newEffectiveDate,omitempty"`
	Classification         *Classification      `json:"classification,omitempty"`
	ExpectedSourceVersion  string               `json:"expectedSourceVersion"`
	ExpectedRecordRevision string               `json:"expectedRecordRevision"`
	Changes                LocationChangesInput `json:"changes,omitempty"`
}

type LocationCreateInput struct {
	Code            string  `json:"code"`
	Name            string  `json:"name"`
	NameEn          *string `json:"nameEn,omitempty"`
	EffectiveDate   string  `json:"effectiveDate,omitempty"`
	OperationType   string  `json:"operationType"`
	OperationReason string  `json:"operationReason"`
	CountryRegionID string  `json:"countryRegionId"`
	GeoAreaID       *string `json:"geoAreaId,omitempty"`
	AddressLine     string  `json:"addressLine"`
	PostalCode      *string `json:"postalCode,omitempty"`
}
type LocationCommand struct {
	Action, LocationID, RecordID, SubmissionID string
	Digest                                     []byte
	Create                                     LocationCreateInput
	Delete                                     DeleteInput
	Redate                                     RedateInput
	InsertPair                                 InsertLifecyclePairInput
	DeletePair                                 DeleteLifecyclePairInput
	MaintainPair                               LocationMaintainPairInput
	Add                                        LocationAddInput
	Correct                                    LocationCorrectInput
}
type LocationPreviewInput struct {
	NewEffectiveDate       string               `json:"newEffectiveDate,omitempty"`
	Classification         *Classification      `json:"classification,omitempty"`
	Action                 string               `json:"action"`
	ExpectedSourceVersion  string               `json:"expectedSourceVersion"`
	AnchorRecordID         string               `json:"anchorRecordId,omitempty"`
	RecordID               string               `json:"recordId,omitempty"`
	ExpectedRecordRevision string               `json:"expectedRecordRevision,omitempty"`
	EffectiveDate          string               `json:"effectiveDate,omitempty"`
	OperationType          string               `json:"operationType,omitempty"`
	OperationReason        string               `json:"operationReason,omitempty"`
	Changes                LocationChangesInput `json:"changes,omitempty"`
}
type LocationPreview struct {
	Impact      *TimelineImpact `json:"impact,omitempty"`
	ReadFields  []string        `json:"readFields"`
	AfterRecord *LocationRecord `json:"afterRecord,omitempty"`
}
type LocationChangeItem struct {
	ChangeKind    string          `json:"changeKind"`
	BeforeRef     *ChangeRef      `json:"beforeRef"`
	AfterRef      *ChangeRef      `json:"afterRef"`
	Diffs         []ChangeDiff    `json:"diffs"`
	RecordID      string          `json:"recordId"`
	BeforeContext *LocationRecord `json:"beforeContext"`
	AfterContext  *LocationRecord `json:"afterContext"`
}
type LocationChange struct {
	ID                    string               `json:"id"`
	SourceVersion         string               `json:"sourceVersion"`
	PreviousSourceVersion *string              `json:"previousSourceVersion"`
	RecordedAt            string               `json:"recordedAt"`
	Action                string               `json:"action"`
	Operator              OperatorSnapshot     `json:"operator"`
	Items                 []LocationChangeItem `json:"items"`
}
type LocationChanges struct {
	ReadFields           []string         `json:"readFields"`
	DetailReadFields     []string         `json:"detailReadFields"`
	ContextSourceVersion *string          `json:"contextSourceVersion"`
	ThroughSourceVersion string           `json:"throughSourceVersion"`
	Items                []LocationChange `json:"items"`
	Total                int              `json:"total"`
	Page                 int              `json:"page"`
	PageSize             int              `json:"pageSize"`
}
type LocationService interface {
	ReferenceContext(context.Context, foundation.Credentials, LocationReferenceInput) (LocationReferenceContext, error)
	StartSelection(context.Context, foundation.Credentials, LocationSelectionStartInput) (LocationSelectionPage, error)
	ResolveSelection(context.Context, foundation.Credentials, LocationSelectionInput) (LocationSelectionPage, error)
	CancelSelection(context.Context, foundation.Credentials, LocationDraftInput) error
	ExportSelection(context.Context, foundation.Credentials, LocationExportInput) (LocationExport, error)
	Context(context.Context, foundation.Credentials) (LocationContext, error)
	List(context.Context, foundation.Credentials, LocationQuery) (LocationList, error)
	Detail(context.Context, foundation.Credentials, DetailInput) (LocationDetail, error)
	Maintain(context.Context, foundation.Credentials, LocationCommand) (LocationResult, error)
	Preview(context.Context, foundation.Credentials, string, LocationPreviewInput) (LocationPreview, error)
	Resolve(context.Context, foundation.Credentials, string) (LocationResult, error)
	Changes(context.Context, foundation.Credentials, ChangeQuery) (LocationChanges, error)
	Navigate(context.Context, foundation.Credentials, LocationNavigationInput) (NavigationResult, error)
}

type LocationQuery struct {
	ListInput
	CountryRegionID string `json:"countryRegionId,omitempty"`
}
type LocationNavigationInput struct {
	Query     LocationSelectionQuery `json:"query"`
	Sort      string                 `json:"sort,omitempty"`
	AnchorID  string                 `json:"anchorId"`
	Direction string                 `json:"direction"`
}
type LocationSelectionQuery struct {
	Q               string `json:"q,omitempty"`
	Status          string `json:"status,omitempty"`
	FilterField     string `json:"filterField,omitempty"`
	FilterOperator  string `json:"filterOperator,omitempty"`
	FilterValue     string `json:"filterValue,omitempty"`
	CountryRegionID string `json:"countryRegionId,omitempty"`
}
type LocationDataChange struct {
	Start        *string        `json:"start,omitempty"`
	EndExclusive **string       `json:"endExclusive,omitempty"`
	EndInclusive **string       `json:"endInclusive,omitempty"`
	Before       LocationRecord `json:"before"`
	After        LocationRecord `json:"after"`
}
type LocationPairContext struct {
	Changes      []LocationDataChange `json:"changes"`
	Deactivation LocationRecord       `json:"deactivation"`
	Reactivation LocationRecord       `json:"reactivation"`
}
type LocationMaintainPairInput struct {
	ExpectedSourceVersion string               `json:"expectedSourceVersion"`
	DeactivationRecordID  string               `json:"deactivationRecordId"`
	DeactivationRevision  string               `json:"deactivationRevision"`
	ReactivationRecordID  string               `json:"reactivationRecordId"`
	ReactivationRevision  string               `json:"reactivationRevision"`
	DeactivationDate      string               `json:"deactivationDate"`
	ReactivationDate      string               `json:"reactivationDate"`
	DeactivationReason    string               `json:"deactivationReason"`
	ReactivationReason    string               `json:"reactivationReason"`
	Changes               LocationChangesInput `json:"changes,omitempty"`
}

type LocationSelectionStartInput struct {
	Query     LocationSelectionQuery `json:"query"`
	Selection Selection              `json:"selection"`
	Page      int                    `json:"page"`
	PageSize  int                    `json:"pageSize"`
	Sort      string                 `json:"sort,omitempty"`
}
type LocationSelectionInput struct {
	Query            LocationSelectionQuery `json:"query"`
	DraftID          string                 `json:"draftId"`
	ExpectedRevision int64                  `json:"expectedRevision"`
	AddIDs           []string               `json:"addIds,omitempty"`
	RemoveIDs        []string               `json:"removeIds,omitempty"`
	RestoreIDs       []string               `json:"restoreIds,omitempty"`
	Page             int                    `json:"page"`
	PageSize         int                    `json:"pageSize"`
	Sort             string                 `json:"sort,omitempty"`
}
type LocationSelectionPage struct {
	PageState   string                  `json:"pageState"`
	BusinessAt  string                  `json:"businessAt"`
	MemberIDs   []string                `json:"memberIds"`
	Page        LocationList            `json:"page"`
	Summary     LocationSelectionResult `json:"summary"`
	SelectedIDs []string                `json:"selectedIds"`
}
type LocationDraftInput struct {
	DraftID          string `json:"draftId"`
	ExpectedRevision int64  `json:"expectedRevision"`
}
type LocationExportInput struct {
	Query            LocationSelectionQuery `json:"query"`
	DraftID          string                 `json:"draftId"`
	ExpectedRevision int64                  `json:"expectedRevision"`
	Sort             string                 `json:"sort,omitempty"`
}
type LocationSelectionResult struct {
	QueryKey      string `json:"queryKey"`
	DraftID       string `json:"draftId"`
	Revision      int64  `json:"revision"`
	SelectedCount int    `json:"selectedCount"`
	CanExport     bool   `json:"canExport"`
}
type LocationExport struct {
	File       io.ReadCloser
	Count      int
	BusinessAt string
}

type LocationReferenceInput struct {
	Action                 string               `json:"action"`
	LocationID             string               `json:"locationId,omitempty"`
	ExpectedSourceVersion  string               `json:"expectedSourceVersion,omitempty"`
	RecordID               string               `json:"recordId,omitempty"`
	ExpectedRecordRevision string               `json:"expectedRecordRevision,omitempty"`
	AnchorRecordID         string               `json:"anchorRecordId,omitempty"`
	EffectiveDate          string               `json:"effectiveDate,omitempty"`
	NewEffectiveDate       string               `json:"newEffectiveDate,omitempty"`
	Classification         *Classification      `json:"classification,omitempty"`
	Changes                LocationChangesInput `json:"changes,omitempty"`
}
type LocationCountryOption struct {
	ID           string `json:"id"`
	Code         string `json:"code,omitempty"`
	Name         string `json:"name"`
	RequiresArea bool   `json:"requiresArea"`
}
type LocationReferenceContext struct {
	NewAdoption bool                    `json:"newAdoption"`
	AdoptedDate string                  `json:"adoptedDate"`
	Countries   []LocationCountryOption `json:"countries"`
}

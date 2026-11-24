package public

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
)

// GeoAreaOrigin is fixed offline initialization input, never an HTTP resource.
type GeoAreaStructure struct {
	CountryID   string `json:"countryRegionId"`
	CountryCode string `json:"countryCode"`
	Scheme      string `json:"scheme"`
	Code        string `json:"code"`
	Nature      string `json:"nature"`
	Level       string `json:"level"`
	ParentID    string `json:"parentAreaId"`
}
type GeoAreaOrigin struct {
	GeoAreaStructure
	Key, SourceVersion, PackageVersion, PackageDigest, ArchiveDigest string
}
type GeoAreaPathNode struct {
	ID            string `json:"id"`
	Code          string `json:"code"`
	Name          string `json:"name"`
	Level         string `json:"level,omitempty"`
	SourceVersion string `json:"sourceVersion,omitempty"`
	RecordID      string `json:"recordId,omitempty"`
	Revision      string `json:"revision,omitempty"`
}
type GeoAreaCountryOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type GeoAreaContext struct {
	FilterCountries      []GeoAreaCountryOption `json:"filterCountries,omitempty"`
	CurrentDate          string                 `json:"currentDate"`
	DefaultEffectiveDate string                 `json:"defaultEffectiveDate"`
	Timezone             string                 `json:"timezone"`
	NameMaxLength        int                    `json:"nameMaxLength"`
	NameGuidance         string                 `json:"nameGuidance"`
	Capabilities         Capabilities           `json:"capabilities"`
}
type GeoAreaRecord struct {
	CountryRegionID *string           `json:"countryRegionId,omitempty"`
	ParentAreaID    **string          `json:"parentAreaId,omitempty"`
	Scheme          *string           `json:"scheme,omitempty"`
	Nature          *string           `json:"nature,omitempty"`
	Level           *string           `json:"level,omitempty"`
	GeoAreaPath     []GeoAreaPathNode `json:"geoAreaPath,omitempty"`

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
type GeoAreaSummary struct {
	CountryRegionName *string           `json:"countryRegionName,omitempty"`
	CountryRegionID   *string           `json:"countryRegionId,omitempty"`
	ParentAreaID      **string          `json:"parentAreaId,omitempty"`
	Scheme            *string           `json:"scheme,omitempty"`
	Nature            *string           `json:"nature,omitempty"`
	Level             *string           `json:"level,omitempty"`
	GeoAreaPath       []GeoAreaPathNode `json:"geoAreaPath,omitempty"`

	ID            string  `json:"id"`
	Code          *string `json:"code,omitempty"`
	Name          *string `json:"name,omitempty"`
	NameEn        *string `json:"nameEn,omitempty"`
	Status        *string `json:"status,omitempty"`
	EffectiveDate *string `json:"effectiveDate,omitempty"`
}
type GeoAreaList struct {
	Items       []GeoAreaSummary `json:"items"`
	Count       int              `json:"count"`
	Page        int              `json:"page"`
	PageSize    int              `json:"pageSize"`
	CurrentDate string           `json:"currentDate"`
}
type GeoAreaAdditionContext struct {
	Available            bool          `json:"available"`
	AnchorRecordID       string        `json:"anchorRecordId"`
	Mode                 string        `json:"mode"`
	MinEffectiveDate     string        `json:"minEffectiveDate"`
	MaxEffectiveDate     *string       `json:"maxEffectiveDate"`
	DefaultEffectiveDate string        `json:"defaultEffectiveDate"`
	Defaults             GeoAreaRecord `json:"defaults"`
}
type GeoAreaLifecycleContext struct {
	AnchorRecordID       string        `json:"anchorRecordId"`
	OperationType        string        `json:"operationType"`
	MinEffectiveDate     string        `json:"minEffectiveDate"`
	MaxEffectiveDate     *string       `json:"maxEffectiveDate"`
	DefaultEffectiveDate string        `json:"defaultEffectiveDate"`
	Defaults             GeoAreaRecord `json:"defaults"`
}
type GeoAreaDetail struct {
	CountryRegionName *string           `json:"countryRegionName,omitempty"`
	PathDate          string            `json:"pathDate"`
	CountryRegionID   *string           `json:"countryRegionId,omitempty"`
	ParentAreaID      **string          `json:"parentAreaId,omitempty"`
	Scheme            *string           `json:"scheme,omitempty"`
	Nature            *string           `json:"nature,omitempty"`
	Level             *string           `json:"level,omitempty"`
	GeoAreaPath       []GeoAreaPathNode `json:"geoAreaPath,omitempty"`

	ID                     string                   `json:"id"`
	Code                   *string                  `json:"code,omitempty"`
	CurrentDate            string                   `json:"currentDate"`
	Status                 *string                  `json:"status,omitempty"`
	NameGuidance           string                   `json:"nameGuidance"`
	HeadSourceVersion      string                   `json:"headSourceVersion"`
	ResolvedSourceVersion  string                   `json:"resolvedSourceVersion"`
	SelectedRecord         *GeoAreaRecord           `json:"selectedRecord"`
	Timeline               []GeoAreaRecord          `json:"timeline"`
	ReadOnly               bool                     `json:"readOnly"`
	SelectionMissing       bool                     `json:"selectionMissing"`
	SelectedRecordIsFuture bool                     `json:"selectedRecordIsFuture"`
	Capabilities           Capabilities             `json:"capabilities"`
	AdditionContext        *GeoAreaAdditionContext  `json:"additionContext,omitempty"`
	LifecycleContext       *GeoAreaLifecycleContext `json:"lifecycleContext,omitempty"`
}
type GeoAreaResult struct {
	SubmissionID     string  `json:"submissionId"`
	Outcome          string  `json:"outcome"`
	GeoAreaID        *string `json:"geoAreaId,omitempty"`
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
type GeoAreaChangesInput map[string]*string
type GeoAreaAddInput struct {
	ExpectedSourceVersion string              `json:"expectedSourceVersion"`
	AnchorRecordID        string              `json:"anchorRecordId"`
	EffectiveDate         string              `json:"effectiveDate"`
	OperationType         string              `json:"operationType"`
	OperationReason       string              `json:"operationReason"`
	Changes               GeoAreaChangesInput `json:"changes,omitempty"`
}
type GeoAreaCorrectInput struct {
	ExpectedSourceVersion  string              `json:"expectedSourceVersion"`
	ExpectedRecordRevision string              `json:"expectedRecordRevision"`
	Changes                GeoAreaChangesInput `json:"changes"`
}

// GeoAreaImport is accepted only by the fixed offline source adapter, not HTTP.
type GeoAreaImport struct {
	ID            string
	Origin        GeoAreaOrigin
	EffectiveDate string
	Name          string
	NameEn        *string
}
type GeoAreaCommand struct {
	Action, GeoAreaID, RecordID, SubmissionID string
	Digest                                    []byte
	Import                                    GeoAreaImport
	Add                                       GeoAreaAddInput
	Correct                                   GeoAreaCorrectInput
}
type GeoAreaPreviewInput struct {
	Action                 string              `json:"action"`
	ExpectedSourceVersion  string              `json:"expectedSourceVersion"`
	AnchorRecordID         string              `json:"anchorRecordId,omitempty"`
	RecordID               string              `json:"recordId,omitempty"`
	ExpectedRecordRevision string              `json:"expectedRecordRevision,omitempty"`
	EffectiveDate          string              `json:"effectiveDate,omitempty"`
	OperationType          string              `json:"operationType,omitempty"`
	OperationReason        string              `json:"operationReason,omitempty"`
	Changes                GeoAreaChangesInput `json:"changes,omitempty"`
}
type GeoAreaPreview struct {
	ReadFields  []string       `json:"readFields"`
	AfterRecord *GeoAreaRecord `json:"afterRecord,omitempty"`
}
type GeoAreaChangeItem struct {
	ChangeKind    string         `json:"changeKind"`
	BeforeRef     *ChangeRef     `json:"beforeRef"`
	AfterRef      *ChangeRef     `json:"afterRef"`
	Diffs         []ChangeDiff   `json:"diffs"`
	RecordID      string         `json:"recordId"`
	BeforeContext *GeoAreaRecord `json:"beforeContext"`
	AfterContext  *GeoAreaRecord `json:"afterContext"`
}
type GeoAreaChange struct {
	ID                    string              `json:"id"`
	SourceVersion         string              `json:"sourceVersion"`
	PreviousSourceVersion *string             `json:"previousSourceVersion"`
	RecordedAt            string              `json:"recordedAt"`
	Action                string              `json:"action"`
	Operator              OperatorSnapshot    `json:"operator"`
	Items                 []GeoAreaChangeItem `json:"items"`
}
type GeoAreaChanges struct {
	ReadFields           []string        `json:"readFields"`
	DetailReadFields     []string        `json:"detailReadFields"`
	ContextSourceVersion *string         `json:"contextSourceVersion"`
	ThroughSourceVersion string          `json:"throughSourceVersion"`
	Items                []GeoAreaChange `json:"items"`
	Total                int             `json:"total"`
	Page                 int             `json:"page"`
	PageSize             int             `json:"pageSize"`
}
type GeoAreaService interface {
	Candidates(context.Context, foundation.Credentials, GeoAreaCandidateInput) (GeoAreaCandidates, error)
	Context(context.Context, foundation.Credentials) (GeoAreaContext, error)
	List(context.Context, foundation.Credentials, GeoAreaQuery) (GeoAreaList, error)
	Detail(context.Context, foundation.Credentials, DetailInput) (GeoAreaDetail, error)
	Maintain(context.Context, foundation.Credentials, GeoAreaCommand) (GeoAreaResult, error)
	Preview(context.Context, foundation.Credentials, string, GeoAreaPreviewInput) (GeoAreaPreview, error)
	Resolve(context.Context, foundation.Credentials, string) (GeoAreaResult, error)
	Changes(context.Context, foundation.Credentials, ChangeQuery) (GeoAreaChanges, error)
	Navigate(context.Context, foundation.Credentials, GeoAreaNavigationInput) (NavigationResult, error)
}

type GeoAreaQuery struct {
	Q               string         `json:"q,omitempty"`
	Status          string         `json:"status,omitempty"`
	Sort            string         `json:"sort,omitempty"`
	Page            int            `json:"page,omitempty"`
	PageSize        int            `json:"pageSize,omitempty"`
	Filter          *GeoAreaFilter `json:"filter,omitempty"`
	CountryRegionID string         `json:"countryRegionId,omitempty"`
	Level           string         `json:"level,omitempty"`
	ParentAreaID    string         `json:"parentAreaId,omitempty"`
}
type GeoAreaFilter struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}
type GeoAreaNavigationInput struct {
	Query     GeoAreaQuery `json:"query"`
	AnchorID  string       `json:"anchorId"`
	Direction string       `json:"direction"`
}

type GeoAreaCandidateInput struct {
	CountryRegionID string `json:"countryRegionId"`
	ParentAreaID    string `json:"parentAreaId"`
	AdoptedDate     string `json:"adoptedDate"`
	Q               string `json:"q"`
	PageSize        int    `json:"pageSize"`
	Cursor          string `json:"cursor"`
}
type GeoAreaCandidate struct {
	ID       string            `json:"id"`
	Code     string            `json:"code"`
	Name     string            `json:"name"`
	Level    string            `json:"level"`
	Nature   *string           `json:"nature,omitempty"`
	Scheme   *string           `json:"scheme,omitempty"`
	NameEn   *string           `json:"nameEn,omitempty"`
	Path     []GeoAreaPathNode `json:"path"`
	Terminal bool              `json:"terminal"`
}
type GeoAreaCandidates struct {
	Mode    string             `json:"mode"`
	Items   []GeoAreaCandidate `json:"items"`
	HasMore bool               `json:"hasMore"`
	Cursor  *string            `json:"cursor"`
}

package public

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
)

// CountryOrigin is offline initialization evidence. It is not an HTTP resource
// or a separately authorized user field.
type CountryOrigin struct {
	Scheme         string `json:"scheme"`
	Key            string `json:"sourceKey"`
	Code           string `json:"code"`
	Name           string `json:"name"`
	NameEn         string `json:"nameEn"`
	PackageVersion string `json:"packageVersion"`
	PackageDigest  string `json:"packageDigest"`
	ArchiveDigest  string `json:"archiveDigest"`
}
type CountryContext struct {
	CurrentDate          string       `json:"currentDate"`
	DefaultEffectiveDate string       `json:"defaultEffectiveDate"`
	Timezone             string       `json:"timezone"`
	NameMaxLength        int          `json:"nameMaxLength"`
	NameGuidance         string       `json:"nameGuidance"`
	Capabilities         Capabilities `json:"capabilities"`
}
type CountryRecord struct {
	ID               string            `json:"recordId"`
	Revision         string            `json:"revision"`
	Deleted          bool              `json:"deleted"`
	EffectiveDate    *string           `json:"effectiveDate,omitempty"`
	EndExclusive     **string          `json:"endExclusive,omitempty"`
	EndInclusive     **string          `json:"endInclusive,omitempty"`
	Name             *string           `json:"name,omitempty"`
	NameEn           **string          `json:"nameEn,omitempty"`
	PhoneCode        **string          `json:"phoneCode,omitempty"`
	State            *string           `json:"state,omitempty"`
	OperationType    *string           `json:"operationType,omitempty"`
	OperationReason  *string           `json:"operationReason,omitempty"`
	TypeLabel        *string           `json:"typeLabel,omitempty"`
	ReasonLabel      *string           `json:"reasonLabel,omitempty"`
	OperationSummary *OperationSummary `json:"operationSummary,omitempty"`
}
type CountrySummary struct {
	SourceVersion string  `json:"sourceVersion"`
	ID            string  `json:"id"`
	Code          *string `json:"code,omitempty"`
	Name          *string `json:"name,omitempty"`
	NameEn        *string `json:"nameEn,omitempty"`
	PhoneCode     *string `json:"phoneCode,omitempty"`
	Status        *string `json:"status,omitempty"`
	EffectiveDate *string `json:"effectiveDate,omitempty"`
}
type CountryList struct {
	Items       []CountrySummary `json:"items"`
	Count       int              `json:"count"`
	Page        int              `json:"page"`
	PageSize    int              `json:"pageSize"`
	CurrentDate string           `json:"currentDate"`
}
type CountryAdditionContext struct {
	Available            bool          `json:"available"`
	AnchorRecordID       string        `json:"anchorRecordId"`
	Mode                 string        `json:"mode"`
	MinEffectiveDate     string        `json:"minEffectiveDate"`
	MaxEffectiveDate     *string       `json:"maxEffectiveDate"`
	DefaultEffectiveDate string        `json:"defaultEffectiveDate"`
	Defaults             CountryRecord `json:"defaults"`
}
type CountryLifecycleContext struct {
	AnchorRecordID       string        `json:"anchorRecordId"`
	OperationType        string        `json:"operationType"`
	MinEffectiveDate     string        `json:"minEffectiveDate"`
	MaxEffectiveDate     *string       `json:"maxEffectiveDate"`
	DefaultEffectiveDate string        `json:"defaultEffectiveDate"`
	Defaults             CountryRecord `json:"defaults"`
}
type CountryDetail struct {
	ID                     string                   `json:"id"`
	Code                   *string                  `json:"code,omitempty"`
	CurrentDate            string                   `json:"currentDate"`
	Status                 *string                  `json:"status,omitempty"`
	NameGuidance           string                   `json:"nameGuidance"`
	HeadSourceVersion      string                   `json:"headSourceVersion"`
	ResolvedSourceVersion  string                   `json:"resolvedSourceVersion"`
	SelectedRecord         *CountryRecord           `json:"selectedRecord"`
	Timeline               []CountryRecord          `json:"timeline"`
	ReadOnly               bool                     `json:"readOnly"`
	SelectionMissing       bool                     `json:"selectionMissing"`
	SelectedRecordIsFuture bool                     `json:"selectedRecordIsFuture"`
	Capabilities           Capabilities             `json:"capabilities"`
	AdditionContext        *CountryAdditionContext  `json:"additionContext,omitempty"`
	LifecycleContext       *CountryLifecycleContext `json:"lifecycleContext,omitempty"`
}
type CountryResult struct {
	SubmissionID     string  `json:"submissionId"`
	Outcome          string  `json:"outcome"`
	CountryRegionID  *string `json:"countryRegionId,omitempty"`
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
type CountryChangesInput map[string]*string
type CountryAddInput struct {
	ExpectedSourceVersion string              `json:"expectedSourceVersion"`
	AnchorRecordID        string              `json:"anchorRecordId"`
	EffectiveDate         string              `json:"effectiveDate"`
	OperationType         string              `json:"operationType"`
	OperationReason       string              `json:"operationReason"`
	Changes               CountryChangesInput `json:"changes,omitempty"`
}
type CountryCorrectInput struct {
	ExpectedSourceVersion  string              `json:"expectedSourceVersion"`
	ExpectedRecordRevision string              `json:"expectedRecordRevision"`
	Changes                CountryChangesInput `json:"changes"`
}

// CountryImport is accepted only by the fixed offline source adapter, not HTTP.
type CountryImport struct {
	ID                string
	Origin            CountryOrigin
	EffectiveDate     string
	Name              string
	NameEn, PhoneCode *string
}
type CountryCommand struct {
	Action, CountryRegionID, RecordID, SubmissionID string
	Digest                                          []byte
	Import                                          CountryImport
	Add                                             CountryAddInput
	Correct                                         CountryCorrectInput
}
type CountryPreviewInput struct {
	Action                 string              `json:"action"`
	ExpectedSourceVersion  string              `json:"expectedSourceVersion"`
	AnchorRecordID         string              `json:"anchorRecordId,omitempty"`
	RecordID               string              `json:"recordId,omitempty"`
	ExpectedRecordRevision string              `json:"expectedRecordRevision,omitempty"`
	EffectiveDate          string              `json:"effectiveDate,omitempty"`
	OperationType          string              `json:"operationType,omitempty"`
	OperationReason        string              `json:"operationReason,omitempty"`
	Changes                CountryChangesInput `json:"changes,omitempty"`
}
type CountryPreview struct {
	ReadFields  []string       `json:"readFields"`
	AfterRecord *CountryRecord `json:"afterRecord,omitempty"`
}
type CountryChangeItem struct {
	ChangeKind    string         `json:"changeKind"`
	BeforeRef     *ChangeRef     `json:"beforeRef"`
	AfterRef      *ChangeRef     `json:"afterRef"`
	Diffs         []ChangeDiff   `json:"diffs"`
	RecordID      string         `json:"recordId"`
	BeforeContext *CountryRecord `json:"beforeContext"`
	AfterContext  *CountryRecord `json:"afterContext"`
}
type CountryChange struct {
	ID                    string              `json:"id"`
	SourceVersion         string              `json:"sourceVersion"`
	PreviousSourceVersion *string             `json:"previousSourceVersion"`
	RecordedAt            string              `json:"recordedAt"`
	Action                string              `json:"action"`
	Operator              OperatorSnapshot    `json:"operator"`
	Items                 []CountryChangeItem `json:"items"`
}
type CountryChanges struct {
	ReadFields           []string        `json:"readFields"`
	DetailReadFields     []string        `json:"detailReadFields"`
	ContextSourceVersion *string         `json:"contextSourceVersion"`
	ThroughSourceVersion string          `json:"throughSourceVersion"`
	Items                []CountryChange `json:"items"`
	Total                int             `json:"total"`
	Page                 int             `json:"page"`
	PageSize             int             `json:"pageSize"`
}
type CountryService interface {
	Context(context.Context, foundation.Credentials) (CountryContext, error)
	List(context.Context, foundation.Credentials, ListInput) (CountryList, error)
	Detail(context.Context, foundation.Credentials, DetailInput) (CountryDetail, error)
	Maintain(context.Context, foundation.Credentials, CountryCommand) (CountryResult, error)
	Preview(context.Context, foundation.Credentials, string, CountryPreviewInput) (CountryPreview, error)
	Resolve(context.Context, foundation.Credentials, string) (CountryResult, error)
	Changes(context.Context, foundation.Credentials, ChangeQuery) (CountryChanges, error)
	Navigate(context.Context, foundation.Credentials, NavigationInput) (NavigationResult, error)
}

// AdoptedCountry identifies the exact active record selected by another context.
type AdoptedCountry struct{ ID, SourceVersion, RecordID, Revision, Date, Name string }

// ProfileCatalog resolves published address references under the caller's current
// permissions and holds their source locks through the profile submission.
type ProfileCatalog interface {
	AdoptCountry(ctx context.Context, credentials foundation.Credentials, id, expectedSource, day string) (AdoptedCountry, error)
	AdoptProfileArea(ctx context.Context, credentials foundation.Credentials, id, day string) ([]GeoAreaPathNode, error)
}

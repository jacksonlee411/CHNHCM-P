// Package public is Staffing's transport-independent value contract.
package public

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
)

type Kind string

const (
	Family      Kind = "jobfamily"
	JobFunction Kind = "jobfunction"
	Job         Kind = "job"
	Grade       Kind = "grade"
	Ladder      Kind = "gradeladder"
	Position    Kind = "position"
)

type Classification struct {
	OperationType   string `json:"operationType"`
	OperationReason string `json:"operationReason"`
}
type Input struct {
	DepartmentID           *string            `json:"departmentId,omitempty"`
	JobID                  *string            `json:"jobId,omitempty"`
	LocationID             *string            `json:"locationId,omitempty"`
	GradeID                *string            `json:"gradeId,omitempty"`
	GradeIDs               *[]string          `json:"gradeIds,omitempty"`
	ChangedGradeIDs        *[]string          `json:"-"`
	ReferencePageSize      int                `json:"referencePageSize,omitempty"`
	ReferenceQ             string             `json:"referenceQ,omitempty"`
	ReferencePage          int                `json:"referencePage,omitempty"`
	ObjectID               string             `json:"objectId,omitempty"`
	Action                 string             `json:"action,omitempty"`
	ExpectedSourceVersion  string             `json:"expectedSourceVersion,omitempty"`
	RecordID               string             `json:"recordId,omitempty"`
	ExpectedRecordRevision string             `json:"expectedRecordRevision,omitempty"`
	AnchorRecordID         string             `json:"anchorRecordId,omitempty"`
	Code                   string             `json:"code,omitempty"`
	Name                   *string            `json:"name,omitempty"`
	NameEn                 *string            `json:"nameEn,omitempty"`
	Description            *string            `json:"description,omitempty"`
	JobFunctionID          *string            `json:"jobFunctionId,omitempty"`
	JobFunctionPresent     bool               `json:"-"`
	JobFamilyID            *string            `json:"jobFamilyId,omitempty"`
	Responsibilities       *string            `json:"responsibilities,omitempty"`
	Requirements           *string            `json:"requirements,omitempty"`
	EffectiveDate          string             `json:"effectiveDate,omitempty"`
	NewEffectiveDate       string             `json:"newEffectiveDate,omitempty"`
	OperationType          string             `json:"operationType,omitempty"`
	OperationReason        string             `json:"operationReason,omitempty"`
	Changes                map[string]*string `json:"changes,omitempty"`
	Classification         *Classification    `json:"classification,omitempty"`
	DeactivationRecordID   string             `json:"deactivationRecordId,omitempty"`
	DeactivationRevision   string             `json:"deactivationRevision,omitempty"`
	ReactivationRecordID   string             `json:"reactivationRecordId,omitempty"`
	ReactivationRevision   string             `json:"reactivationRevision,omitempty"`
	DeactivationDate       string             `json:"deactivationDate,omitempty"`
	ReactivationDate       string             `json:"reactivationDate,omitempty"`
	DeactivationReason     string             `json:"deactivationReason,omitempty"`
	ReactivationReason     string             `json:"reactivationReason,omitempty"`
}
type Command struct {
	Kind                   Kind
	ObjectID, SubmissionID string
	Digest                 []byte
	Input                  Input
}
type Result struct {
	PositionID             string            `json:"positionId,omitempty"`
	OriginalSourceVersion  string            `json:"originalSourceVersion,omitempty"`
	DepartmentID           string            `json:"departmentId,omitempty"`
	QuotaChanges           []AllocationRef   `json:"quotaChanges,omitempty"`
	Kind                   Kind              `json:"kind"`
	SelectedRecordRevision string            `json:"selectedRecordRevision,omitempty"`
	FailureStatus          int               `json:"failureStatus,omitempty"`
	SubmissionID           string            `json:"submissionId"`
	Outcome                string            `json:"outcome"`
	ObjectID               string            `json:"objectId,omitempty"`
	SourceVersion          string            `json:"sourceVersion,omitempty"`
	SelectedRecordID       string            `json:"selectedRecordId,omitempty"`
	Action                 string            `json:"action,omitempty"`
	Replayed               bool              `json:"replayed,omitempty"`
	Failure                *foundation.Fault `json:"failure,omitempty"`
}
type Query struct {
	ScopeDepartmentID  string `json:"scopeDepartmentId,omitempty"`
	IncludeDescendants *bool  `json:"includeDescendants,omitempty"`
	Q                  string `json:"q,omitempty"`
	Status             string `json:"status,omitempty"`
	JobFamilyID        string `json:"jobFamilyId,omitempty"`
	JobFunctionID      string `json:"jobFunctionId,omitempty"`
	JobFunctionMissing bool   `json:"jobFunctionMissing,omitempty"`
	FilterField        string `json:"filterField,omitempty"`
	FilterOperator     string `json:"filterOperator,omitempty"`
	FilterValue        string `json:"filterValue,omitempty"`
	Sort               string `json:"sort,omitempty"`
	Page               int    `json:"page,omitempty"`
	PageSize           int    `json:"pageSize,omitempty"`
}
type View map[string]any
type SelectionInput struct {
	InitialIDs       []string `json:"initialIds,omitempty"`
	DraftID          string   `json:"draftId"`
	ExpectedRevision string   `json:"expectedRevision,omitempty"`
	Query            Query    `json:"query"`
	ExcludedIDs      []string `json:"excludedIds,omitempty"`
	Sort             string   `json:"sort,omitempty"`
}
type Selection struct {
	State       string   `json:"state"`
	DraftID     string   `json:"draftId"`
	Revision    string   `json:"revision"`
	BusinessAt  string   `json:"businessAt"`
	Members     []string `json:"members"`
	ExcludedIDs []string `json:"excludedIds"`
	Count       int      `json:"count"`
}
type Export struct {
	Bytes []byte
	Count int
	Date  string
}
type Service interface {
	QuotaService
	Structure(context.Context, foundation.Credentials, StructureQuery) (StructureView, error)
	HierarchyAt(context.Context, foundation.Credentials, string, string) (HierarchyView, error)
	PublishHierarchy(context.Context, foundation.Credentials, string, []byte, PublishHierarchy) (Result, error)
	Context(context.Context, foundation.Credentials, Kind) (View, error)
	Reasons(context.Context, foundation.Credentials, Kind, string) (View, error)
	List(context.Context, foundation.Credentials, Kind, Query) (View, error)
	Detail(context.Context, foundation.Credentials, Kind, string, string, string) (View, error)
	Changes(context.Context, foundation.Credentials, Kind, string, LogQuery) (ChangeLog, error)
	Navigation(context.Context, foundation.Credentials, Kind, string, Query) (View, error)
	Preview(context.Context, foundation.Credentials, Command) (View, error)
	Execute(context.Context, foundation.Credentials, Command) (Result, error)
	Resolve(context.Context, foundation.Credentials, Kind, string) (Result, error)
	ClassificationReferenceContext(context.Context, foundation.Credentials, Kind, Command) (View, error)
	PositionReferences(context.Context, foundation.Credentials, string, Command) (View, error)
	GradeReferenceContext(context.Context, foundation.Credentials, Command) (View, error)
	Select(context.Context, foundation.Credentials, Kind, string, SelectionInput) (Selection, error)
	Export(context.Context, foundation.Credentials, Kind, SelectionInput) (Export, error)
}

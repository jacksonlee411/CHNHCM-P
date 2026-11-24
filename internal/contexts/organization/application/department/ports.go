// Package department adapts the shared dated/submission mechanisms to Department.
package department

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/application/submission"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"context"
	"io"
	"time"
)

type ReadSnapshot struct {
	ListFacts *ListFacts
	Snapshot  d.Snapshot
	Selected  *d.Timeline
	Evidence  []d.PathSegment
	Summaries map[string]org.OperationSummary
}
type LockRequest struct {
	DepartmentID, Code string
	BUs, Parents       []string
	Creating           bool
}
type Terminal struct {
	ID, EnterpriseID, DepartmentID, RecordID, Revision, ExpectedSource, Action, Outcome, ErrorCode string
	Digest                                                                                         []byte
	Fields                                                                                         []string
	Items                                                                                          []org.RequestItem
	EvaluatedAt                                                                                    *time.Time
	Failure                                                                                        *org.FailureDetails
	Result                                                                                         org.DepartmentResult
}
type Change struct {
	RecordID, BeforeRevision, AfterRevision, Role, Action string
	Facets                                                []string
}
type EdgeChange struct {
	EdgeID                   string         `json:"edgeId"`
	BeforeRevisionID         *string        `json:"beforeRevisionId"`
	AfterRevisionID          *string        `json:"afterRevisionId"`
	BeforeManagementBUID     *string        `json:"beforeManagementBuId"`
	AfterManagementBUID      *string        `json:"afterManagementBuId"`
	BeforeParentDepartmentID *string        `json:"beforeParentDepartmentId"`
	AfterParentDepartmentID  *string        `json:"afterParentDepartmentId"`
	BeforeRange              *org.DateRange `json:"beforeRange"`
	AfterRange               *org.DateRange `json:"afterRange"`
}
type Write struct {
	Before, After              d.Timeline
	Candidate                  d.Candidate
	NewRecords, DeletedRecords []d.Record
	NewEdges, DeletedEdges     []d.Edge
	RelationChanged            bool
	Changes                    []Change
	EdgeChanges                []EdgeChange
	Evidence                   []d.PathSegment
	Operator                   access.OperatorSnapshot
	Terminal                   Terminal
	authorized                 bool
}

func (w Write) Validated() bool { return w.authorized }

type Saved = submission.Saved[org.DepartmentResult]
type Unit interface {
	Drafts
	Access() access.Guard
	Now(context.Context) (time.Time, error)
	Enterprise(context.Context, bool) (org.Enterprise, error)
	submission.Repository[org.DepartmentResult, Terminal]
	Snapshot(context.Context, string, string) (ReadSnapshot, error)
	PreviewSnapshot(context.Context, string, []string, []string, bool) (ReadSnapshot, error)
	CaptureQuery(context.Context, QueryRequest) (QueryCapture, error)
	WalkQuery(context.Context, QueryRequest, func(QueryFrame) (bool, error)) error
	WalkParents(context.Context, ParentRequest, func(ParentFrame) (bool, error)) error
	CreationBUs(context.Context) ([]string, error)
	BUReferenceSnapshot(context.Context, string) (ReadSnapshot, error)
	History(context.Context, string, string) ([]HistorySource, error)
	Source(context.Context, string, string) (HistorySource, error)
	CompleteResult(context.Context, *Saved) error
	Lock(context.Context, LockRequest) (ReadSnapshot, error)
	LockUnchangedRelations(context.Context, LockRequest, d.Timeline, []d.Record) (ReadSnapshot, error)
	Persist(context.Context, Write) (org.DepartmentResult, error)
}
type Store interface {
	CreateExport(context.Context) (ExportFile, error)
	WithinDepartment(context.Context, access.Credentials, func(Unit) error) error
	WithinDepartmentWrite(context.Context, access.Credentials, func(Unit) error) error
}

// ExportFile belongs to one request. Seal makes the complete output readable;
// Close releases the private temporary storage on every success/failure path.
type ExportFile interface {
	io.Writer
	io.ReadCloser
	Seal() error
}
type Service struct{ Store Store }

type HistoryRevision struct {
	Record  d.Record
	Deleted bool
	Summary org.OperationSummary
}
type HistoryItem struct {
	RecordID, BeforeSource, Role, Action string
	Before, After                        *HistoryRevision
}
type HistorySource struct {
	ID, Previous, At, Action string
	No                       int64
	Operator                 org.OperatorSnapshot
	Items                    []HistoryItem
}

package application

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	s "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"time"
)

type StoredResult struct {
	Digest []byte
	Result s.Result
	Exists bool
}
type Source struct {
	Timeline   d.Timeline
	Command    s.Command
	Operator   access.OperatorSnapshot
	RecordedAt time.Time
}
type Draft struct {
	s.Selection
	Query     s.Query
	Expiry    time.Time
	Cancelled bool
}
type HierarchyUnit interface {
	ReadStructure(context.Context, string, s.StructureQuery, bool, bool) (s.StructureView, error)
	LockHierarchy(context.Context, string) error
	LockHierarchyNodes(context.Context, []string) error
	HierarchyVersions(context.Context, string) (string, []s.HierarchyVersion, error)
	HierarchyEdges(context.Context, string, string) (d.HierarchyEdges, error)
	CurrentHierarchyUses(context.Context, string, string) (d.HierarchyEdges, error)
	HierarchyPositions(context.Context, string, string, string, string) ([]d.Timeline, error)
	SaveHierarchy(context.Context, string, s.Result, int, d.HierarchyEdges, access.OperatorSnapshot) error
}
type Unit interface {
	AllocationTx
	LoadPositionQuota(context.Context, string, string) (d.StaffingQuota, error)
	PositionDepartments(context.Context, string, bool, []string) ([]string, error)
	HierarchyUnit
	PositionCandidateIDs(context.Context, string, string, string, []string, int, int) ([]string, error)
	PositionSources(context.Context, string, []string, bool) ([]PositionSource, error)
	PositionUses(context.Context, string, string, string) ([]s.PositionUse, error)
	Access() access.Guard
	Now() time.Time
	Enterprise(context.Context, bool) (org.Enterprise, error)
	LockSubmission(context.Context, d.Kind, string) error
	Terminal(context.Context, d.Kind, string) (StoredResult, error)
	Close(context.Context, d.Kind, string, []byte, s.Result) error
	LockObject(context.Context, d.Kind, string, []string, []string, []string) (d.Timeline, error)
	Load(context.Context, d.Kind, string, string) (d.Timeline, error)
	ReferenceClassifications(context.Context, d.Kind) ([]d.Timeline, error)
	ReferenceGrades(context.Context) ([]d.Timeline, error)
	Current(context.Context, d.Kind) ([]d.Timeline, error)
	CurrentPage(context.Context, d.Kind, int, int, bool, bool) ([]d.Timeline, int, error)
	Heads(context.Context, d.Kind, []string) ([]d.Timeline, error)
	Dependents(context.Context, d.Kind, string) ([]d.Timeline, error)
	GradeConsumerHeads(context.Context) ([]d.Timeline, error)
	Sources(context.Context, d.Kind, string) ([]Source, error)
	Business(context.Context, func() error) error
	Save(context.Context, d.Plan, s.Command, access.OperatorSnapshot) error
	Draft(context.Context, d.Kind, string, bool) (Draft, bool, error)
	SaveDraft(context.Context, d.Kind, Draft) error
}
type Store interface {
	WithinSelection(context.Context, foundation.Credentials, func(Unit) error) error
	Within(context.Context, foundation.Credentials, bool, func(Unit) error) error
}
type Service struct {
	Store      Store
	QuotaStore QuotaStore
}

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

type QuotaStore interface {
	WithinQuota(context.Context, foundation.Credentials, bool, func(QuotaTx) error) error
}
type QuotaCandidate struct {
	PositionID, Code, RecordID string
	Position                   d.QuotaPeriod
	Amount                     int64
}
type QuotaHistoryEntry struct {
	Source, Action, RecordID string
	RecordedAt               time.Time
	Operator                 access.OperatorSnapshot
	Snapshot                 d.StaffingQuota
}
type AllocationTx interface {
	PositionUsage(context.Context, string) (d.QuotaUsage, error)
	LockQuotaDepartments(context.Context, []string) error
	DepartmentMembers(context.Context, string) ([]d.AllocatedPosition, error)
	LoadTargetQuota(context.Context, d.QuotaTarget, string) (d.StaffingQuota, error)
}
type QuotaTx interface {
	Access() access.Guard
	Now() time.Time
	Enterprise(context.Context, bool) (org.Enterprise, error)
	LockQuotaTarget(context.Context, string) error
	Load(context.Context, d.Kind, string, string) (d.Timeline, error)
	PositionPeriods(context.Context, string) (d.QuotaPeriods, error)
	PositionSources(context.Context, string, []string, bool) ([]PositionSource, error)
	LoadPositionQuota(context.Context, string, string) (d.StaffingQuota, error)
	QuotaIdentity(context.Context, string) (d.QuotaTarget, error)
	AllocationTx
	SearchQuotaDepartments(context.Context, foundation.Credentials, string, string, string, int, int) ([]org.DepartmentChoice, error)
	DepartmentQuotaIDs(context.Context) ([]string, error)
	QuotaCandidates(context.Context, string) ([]QuotaCandidate, error)
	PreviousQuotaSource(context.Context, string, string) (string, error)
	QuotaHistory(context.Context, string) ([]QuotaHistoryEntry, error)
	Business(context.Context, func() error) error
	SaveQuota(context.Context, d.QuotaPlan, s.QuotaCommand, access.OperatorSnapshot) error
	LockSubmission(context.Context, d.Kind, string) error
	Terminal(context.Context, d.Kind, string) (StoredResult, error)
	Close(context.Context, d.Kind, string, []byte, s.Result) error
}

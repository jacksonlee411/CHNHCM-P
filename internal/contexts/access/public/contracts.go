// Package public contains Access contracts without persistence types.
package public

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"time"
)

type Credentials = foundation.Credentials
type Principal struct {
	ID         string `json:"subjectId"`
	SessionID  string `json:"-"`
	Login      string `json:"login"`
	CSRF       string `json:"csrf"`
	InstanceID string `json:"instanceId"`
}
type Grant struct {
	ScopeKind   string
	ScopeID     string
	Action      string
	ReadFields  []string
	WriteFields []string
}
type Intent struct {
	// PermissionOnly asks for the allow/deny outcome and no field projection.
	// A command can then survive expiry of rights it never used in its result.
	PermissionOnly   bool
	PositionID       string
	DepartmentTarget *DepartmentTarget
	AffectedScope    *AffectedScope
	Requirements     []Intent
	ItemID           string
	Action           string
	EnterpriseID     string
	BUID             string
	Fields           []string
	Kind             string
}

// ReadIntent contains only the target and purpose of one guarded projection.
// Command declarations cannot enter this batch. Scope matching and field rights
// remain identical to Decide; callers may share an immutable target pointer.
type ReadIntent struct {
	DepartmentTarget                 *DepartmentTarget
	EnterpriseID, BUID, Action, Kind string
}

func (r ReadIntent) Intent() Intent {
	return Intent{DepartmentTarget: r.DepartmentTarget, EnterpriseID: r.EnterpriseID,
		BUID: r.BUID, Action: r.Action, Kind: r.Kind}
}

// ScopeFact is supplied by M02 from current locked facts, never from HTTP input.
type ScopeFact struct {
	EvaluatedAt          time.Time
	BusinessDate         string
	Mode                 string
	ManagementBUID       string
	CurrentSourceVersion string
	RecordID             string
	Revision             string
}
type DepartmentTarget struct {
	EnterpriseID string
	DepartmentID string
	ScopeFact    ScopeFact
}
type AffectedScope struct {
	ManagementBUID string
}
type Decision struct {
	// DeniedItemID identifies the first failed explicit requirement in a compound intent.
	DeniedItemID            string
	Allowed                 bool
	ReadFields, WriteFields []string
}

// ReadDecisions is request-local evaluation material, never an audit format or
// a persisted permission cache. Index preserves one current decision per input;
// identical matching grant sequences may refer to one immutable field union.
type ReadDecisions struct {
	Values []Decision
	Index  []int
}
type Guard interface {
	// Bind carries this transaction's decision history to nested business ports.
	Bind(context.Context) context.Context
	// RecordSessionActivity persists the authenticated request's access time in
	// this transaction, after business work and immediately before commit. No
	// further business locks may be acquired after this session-row update.
	RecordSessionActivity(context.Context) error
	ValidateUse() error
	OperatorSnapshot(context.Context) (OperatorSnapshot, error)
	// PolicyVersion is the version locked by this transaction for its principal.
	PolicyVersion() int64
	Principal() Principal
	Grants() []Grant
	Decide(context.Context, Intent) (Decision, error)
	// DecideReads evaluates read-only projections under this same guarded policy.
	// It does not persist authorization calculations.
	DecideReads(context.Context, []ReadIntent) (ReadDecisions, error)
}

// OperatorSnapshot is immutable operation evidence, never an authorization token.
type OperatorSnapshot struct {
	SubjectID        string  `json:"subjectId"`
	DisplayName      *string `json:"displayName"`
	Login            *string `json:"login"`
	Kind             string  `json:"kind"`
	IdentityEvidence string  `json:"identityEvidence"`
}

type LoginInput struct {
	Login         string `json:"login"`
	Password      string `json:"password"`
	PreviousToken string `json:"-"`
}
type LoginResult struct {
	Principal Principal
	Token     string
}
type Service interface {
	Login(context.Context, LoginInput, string) (LoginResult, error)
	Session(context.Context, Credentials) (Principal, error)
	Logout(context.Context, Credentials) error
}

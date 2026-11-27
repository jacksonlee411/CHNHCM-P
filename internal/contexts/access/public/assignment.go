package public

import "time"

// Version IDs name immutable publications. DefinitionID groups successive publications.
type PolicyVersion struct {
	Kind, ID, DefinitionID  string
	Number                  int64
	Actions                 []string
	ScopeKind, ScopeID      string
	ReadFields, WriteFields []string
}

type VersionBinding struct{ Role, Scope, Fields string }

// AssignmentChange is accepted by the Access writer inside a caller-owned,
// authorized transaction. Issue uses revision zero. Adopt and revoke name the
// observed revision. The request ID belongs to exactly one command payload.
type AssignmentChange struct {
	Operation, ID, Subject, RequestID, Actor string
	ExpectedRevision                         int64
	Binding                                  VersionBinding
	Source, Purpose                          string
	ValidFrom, ValidUntil                    *time.Time
}

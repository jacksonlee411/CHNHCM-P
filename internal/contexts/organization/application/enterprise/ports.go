// Package enterprise adapts dated and submission to the instance enterprise catalog.
package enterprise

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/application/submission"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/enterprise"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"context"
	"time"
)

type Terminal struct {
	ID, ObjectID, Action, Outcome, ErrorCode string
	Digest                                   []byte
	Result                                   org.EnterpriseResult
}
type Revision struct{ Before, After *d.Record }
type Operation struct {
	ID, Previous, Subject, Submission, Action, EventID string
	Number                                             int64
	RecordedAt                                         time.Time
	Operator                                           org.OperatorSnapshot
	Revisions                                          []Revision
	Terminal                                           Terminal
}
type State struct {
	Timeline  d.Timeline
	Summaries []org.OperationSummary
}
type Write struct {
	Before     d.Timeline
	After      State
	Operation  Operation
	authorized bool
}

func (w Write) Validated() bool { return w.authorized }

type Saved = submission.Saved[org.EnterpriseResult]
type Unit interface {
	Identity(context.Context) (string, error)
	ValidateReferences(context.Context, d.Timeline, d.Timeline) error
	Access() access.Guard
	Now(context.Context) (time.Time, error)
	submission.Repository[org.EnterpriseResult, Terminal]
	Lock(context.Context, string) error
	Load(context.Context, string, string) (State, error)
	History(context.Context, string, string) ([]Operation, error)
	Persist(context.Context, Write) (org.EnterpriseResult, error)
}
type Store interface {
	WithinEnterprise(context.Context, access.Credentials, func(Unit) error) error
	WithinEnterpriseWrite(context.Context, access.Credentials, func(Unit) error) error
}
type Service struct{ Store Store }

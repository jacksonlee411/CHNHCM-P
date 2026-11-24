// Package legalentity adapts dated and submission to the enterprise legalentity object.
package legalentity

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	country "chnhcm.local/chnhcm/internal/contexts/organization/application/country"
	"chnhcm.local/chnhcm/internal/contexts/organization/application/submission"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/legalentity"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"context"
	"io"
	"time"
)

type Terminal struct {
	ID, ObjectID, Action, Outcome, ErrorCode string
	Digest                                   []byte
	Result                                   org.LegalEntityResult
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

type Saved = submission.Saved[org.LegalEntityResult]
type Unit interface {
	Countries(context.Context) ([]country.State, error)
	Drafts
	CaptureExport(context.Context, org.LegalEntityQuery, []string, string, []string) ([]string, error)
	QueryIDs(context.Context, org.LegalEntityQuery, []string, string, []string) ([]string, error)
	LoadMany(context.Context, []string) ([]State, error)
	EnterpriseScope() org.Enterprise
	References() country.ReferenceUnit
	Access() access.Guard
	Now(context.Context) (time.Time, error)
	submission.Repository[org.LegalEntityResult, Terminal]
	Lock(context.Context, string) error
	Load(context.Context, string, string) (State, error)
	List(context.Context, org.LegalEntityQuery, []string, string) (org.LegalEntityList, error)
	Navigate(context.Context, org.LegalEntityNavigationInput, []string, string) (org.NavigationResult, error)
	History(context.Context, string, string) ([]Operation, error)
	Persist(context.Context, Write) (org.LegalEntityResult, error)
}
type ExportFile interface {
	io.Writer
	io.ReadCloser
	Seal() error
}
type Store interface {
	CreateLegalEntityExport(context.Context) (ExportFile, error)
	WithinLegalEntity(context.Context, access.Credentials, func(Unit) error) error
	WithinLegalEntityWrite(context.Context, access.Credentials, func(Unit) error) error
}
type Service struct{ Store Store }

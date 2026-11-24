// Package location adapts dated and submission to the enterprise location object.
package location

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	country "chnhcm.local/chnhcm/internal/contexts/organization/application/country"
	geo "chnhcm.local/chnhcm/internal/contexts/organization/application/geoarea"
	"chnhcm.local/chnhcm/internal/contexts/organization/application/submission"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/location"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"context"
	"io"
	"time"
)

type Terminal struct {
	ID, ObjectID, Action, Outcome, ErrorCode string
	Digest                                   []byte
	Result                                   org.LocationResult
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

type Saved = submission.Saved[org.LocationResult]
type Unit interface {
	AddressCountries(context.Context) ([]country.State, error)
	Drafts
	CaptureExport(context.Context, org.LocationQuery, []string, string, []string) ([]string, error)
	QueryIDs(context.Context, org.LocationQuery, []string, string, []string) ([]string, error)
	LoadMany(context.Context, []string) ([]State, error)
	EnterpriseScope() org.Enterprise
	References() geo.ReferenceUnit
	Access() access.Guard
	Now(context.Context) (time.Time, error)
	submission.Repository[org.LocationResult, Terminal]
	Lock(context.Context, string) error
	Load(context.Context, string, string) (State, error)
	List(context.Context, org.LocationQuery, []string, string) (org.LocationList, error)
	Navigate(context.Context, org.LocationNavigationInput, []string, string) (org.NavigationResult, error)
	History(context.Context, string, string) ([]Operation, error)
	Persist(context.Context, Write) (org.LocationResult, error)
}
type ExportFile interface {
	io.Writer
	io.ReadCloser
	Seal() error
}
type Store interface {
	CreateLocationExport(context.Context) (ExportFile, error)
	WithinLocation(context.Context, access.Credentials, func(Unit) error) error
	WithinLocationWrite(context.Context, access.Credentials, func(Unit) error) error
}
type Service struct{ Store Store }

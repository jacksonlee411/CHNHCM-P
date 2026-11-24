// Package country adapts dated and submission to the instance country catalog.
package country

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/application/submission"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/country"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"context"
	"time"
)

type Terminal struct {
	ID, ObjectID, Action, Outcome, ErrorCode string
	Digest                                   []byte
	Result                                   org.CountryResult
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

type Saved = submission.Saved[org.CountryResult]
type Unit interface {
	CatalogComplete(context.Context, CatalogPublication) (bool, error)
	PublishCatalog(context.Context, CatalogPublication) error
	Access() access.Guard
	Now(context.Context) (time.Time, error)
	submission.Repository[org.CountryResult, Terminal]
	Lock(context.Context, string) error
	Load(context.Context, string, string) (State, error)
	List(context.Context, org.ListInput, []string, string) (org.CountryList, error)
	History(context.Context, string, string) ([]Operation, error)
	Persist(context.Context, Write) (org.CountryResult, error)
}
type Store interface {
	WithinCountry(context.Context, access.Credentials, func(Unit) error) error
	WithinCountryWrite(context.Context, access.Credentials, func(Unit) error) error
}
type Service struct{ Store Store }

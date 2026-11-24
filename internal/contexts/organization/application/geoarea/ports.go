// Package geoarea adapts dated and submission to the instance geoarea catalog.
package geoarea

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	countryapp "chnhcm.local/chnhcm/internal/contexts/organization/application/country"
	"chnhcm.local/chnhcm/internal/contexts/organization/application/submission"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/geoarea"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"context"
	"time"
)

type Terminal struct {
	ID, ObjectID, Action, Outcome, ErrorCode string
	Digest                                   []byte
	Result                                   org.GeoAreaResult
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

type Saved = submission.Saved[org.GeoAreaResult]
type Unit interface {
	AddressCountries(context.Context) ([]countryapp.State, error)
	PathHeads(context.Context, d.Structure) (CandidateRow, error)
	RegionPublished(context.Context, string) (bool, error)
	Published(context.Context, string) (bool, error)
	IsTerminal(context.Context, string) (bool, error)
	CatalogComplete(context.Context, CatalogPublication) (bool, error)
	PublishCatalog(context.Context, CatalogPublication, []Mapping) error
	Mappings(context.Context, string) ([]Mapping, error)
	Country(context.Context, string, string) (countryapp.State, error)
	LockReference(context.Context, string, string) error
	CandidateRows(context.Context, org.GeoAreaCandidateInput, string, string, int) ([]CandidateRow, error)
	Navigate(context.Context, org.GeoAreaNavigationInput, []string, string) (org.NavigationResult, error)
	Access() access.Guard
	Now(context.Context) (time.Time, error)
	submission.Repository[org.GeoAreaResult, Terminal]
	Lock(context.Context, string) error
	Load(context.Context, string, string) (State, error)
	LoadSources(context.Context, map[string]string) (map[string]State, error)
	List(context.Context, org.GeoAreaQuery, []string, string) (org.GeoAreaList, error)
	History(context.Context, string, string) ([]Operation, error)
	Persist(context.Context, Write) (org.GeoAreaResult, error)
}
type Store interface {
	WithinGeoArea(context.Context, access.Credentials, func(Unit) error) error
	WithinGeoAreaWrite(context.Context, access.Credentials, func(Unit) error) error
}
type Service struct{ Store Store }

// Package model defines the only persisted Organization history format. Types
// here carry complete logical facts; storage chunks only change representation.
package model

import (
	app "chnhcm.local/chnhcm/internal/contexts/organization/application"
	countryapp "chnhcm.local/chnhcm/internal/contexts/organization/application/country"
	dapp "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	enterpriseapp "chnhcm.local/chnhcm/internal/contexts/organization/application/enterprise"
	geoareaapp "chnhcm.local/chnhcm/internal/contexts/organization/application/geoarea"
	legalentityapp "chnhcm.local/chnhcm/internal/contexts/organization/application/legalentity"
	locationapp "chnhcm.local/chnhcm/internal/contexts/organization/application/location"
	"chnhcm.local/chnhcm/internal/contexts/organization/domain"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"time"
)

type Source struct {
	ID, Previous, Subject, Submission, Action string
	Number                                    int64
	RecordedAt                                time.Time
	Operator                                  org.OperatorSnapshot
}

type BUState struct {
	Timeline  app.Timeline
	Summaries []org.OperationSummary
}
type BURevision struct {
	Record   domain.Record
	Previous string
	Deleted  bool
	Before   *domain.Record
}
type BUOperation struct {
	TerminalClosedAt time.Time
	Source           Source
	Revisions        []BURevision
	Terminal         app.Terminal
	EventID          string
	Impact           org.TimelineImpact
}
type DepartmentState struct {
	Timeline  d.Timeline
	Summaries []org.OperationSummary
}
type DepartmentOperation struct {
	TerminalClosedAt         time.Time
	Source                   Source
	History                  dapp.HistorySource
	Terminal                 dapp.Terminal
	EventID, RelationEventID string
	RelationPrevious         string
	Changes                  []dapp.Change
	EdgeChanges              []dapp.EdgeChange
	NewEdges, DeletedEdges   []d.Edge
	DeletedRecords           []d.Record
}
type DepartmentEvidence struct{ Segments []d.PathSegment }

// Read indexes contain only immutable source members. They are disposable
// derivatives, never history, authorization, status or a composed path.
type DepartmentReadRecord struct {
	ID, Revision, Date, End, State, Name string
	ShortName                            *string
	BU, Parent                           string
}
type DepartmentRead struct {
	ID, Enterprise, First, Source, RelationSource string
	Records                                       []DepartmentReadRecord
	Edges                                         []d.Edge
}

func ReadDepartment(t d.Timeline) DepartmentRead {
	v := DepartmentRead{ID: t.ID, Enterprise: t.Enterprise, First: t.First, Source: t.Source, RelationSource: t.RelationSource, Edges: t.Edges}
	for _, r := range t.Records {
		v.Records = append(v.Records, DepartmentReadRecord{ID: r.ID, Revision: r.Revision, Date: r.Date, End: r.End, State: r.State, Name: r.Data.Name, ShortName: r.Data.ShortName, BU: r.Data.BU, Parent: r.Data.Parent})
	}
	return v
}
func (v DepartmentRead) Domain() d.Timeline {
	t := d.Timeline{ID: v.ID, Enterprise: v.Enterprise, First: v.First, Source: v.Source, RelationSource: v.RelationSource, Edges: v.Edges}
	for _, r := range v.Records {
		t.Records = append(t.Records, d.Record{ID: r.ID, Revision: r.Revision, Date: r.Date, End: r.End, State: r.State, Data: d.Data{Name: r.Name, ShortName: r.ShortName, BU: r.BU, Parent: r.Parent}})
	}
	return t
}

type BURead struct{ Timeline d.BusinessUnit }

// The typed operation format fixes the event kind and schema; event identity,
// source, correlation, and payload remain immutable facts in that operation.
type BUEvent struct {
	ID, Source, Correlation, Type string
	SchemaVersion                 int
	Impact                        org.TimelineImpact
}

func (o BUOperation) Event() BUEvent {
	return BUEvent{o.EventID, o.Source.ID, o.Source.Submission, "BusinessUnitSourceChanged", 2, o.Impact}
}

// Country uses the same fixed codec and history envelopes with its own types.
type CountryState struct{ State countryapp.State }
type CountryOperation struct{ Operation countryapp.Operation }
type CountryTerminal struct{ Terminal countryapp.Terminal }
type CountryEvidence struct{ PackageDigest string }

type GeoAreaState struct{ State geoareaapp.State }
type GeoAreaOperation struct{ Operation geoareaapp.Operation }
type GeoAreaTerminal struct{ Terminal geoareaapp.Terminal }
type GeoAreaEvidence struct{ PackageDigest string }

type EnterpriseState struct{ State enterpriseapp.State }
type EnterpriseOperation struct{ Operation enterpriseapp.Operation }
type EnterpriseTerminal struct{ Terminal enterpriseapp.Terminal }
type EnterpriseEvidence struct{ InitializationDigest string }
type LocationState struct{ State locationapp.State }
type LocationOperation struct{ Operation locationapp.Operation }
type LocationTerminal struct{ Terminal locationapp.Terminal }
type LocationEvidence struct{ ObjectID string }

type LegalEntityState struct{ State legalentityapp.State }
type LegalEntityOperation struct{ Operation legalentityapp.Operation }
type LegalEntityTerminal struct{ Terminal legalentityapp.Terminal }
type LegalEntityEvidence struct{ ObjectID string }

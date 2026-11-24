package public

import (
	f "chnhcm.local/chnhcm/internal/foundation"
	"context"
)

type Terms struct {
	Number      string `json:"number"`
	Person      string `json:"person"`
	Employer    string `json:"employer"`
	Department  string `json:"department,omitempty"`
	Job         string `json:"job,omitempty"`
	Position    string `json:"position,omitempty"`
	Arrival     string `json:"arrival"`
	ReserveFrom string `json:"reserveFrom"`
	LastDay     string `json:"lastDay,omitempty"`
}

// The offer owns the immutable arrangement and employer evidence it adopted.
type AssignmentSegment struct {
	QuotaSource         string `json:"quotaSource,omitempty"`
	QuotaRevision       string `json:"quotaRevision,omitempty"`
	RecruitingCoverage  string `json:"recruitingCoverage,omitempty"`
	DepartmentID        string `json:"departmentId,omitempty"`
	JobID               string `json:"jobId,omitempty"`
	PositionName        string `json:"positionName,omitempty"`
	PositionSource      string `json:"positionSource,omitempty"`
	PositionRevision    string `json:"positionRevision,omitempty"`
	ReplenishmentSource string `json:"replenishmentSource,omitempty"`
	CoverageSource      string `json:"coverageSource,omitempty"`
	DepartmentName      string `json:"departmentName,omitempty"`
	JobName             string `json:"jobName,omitempty"`
	f.Period            `json:",inline"`
	PolicyVersion       string `json:"policyVersion"`
	Policy              string `json:"policy"`
	DepartmentSource    string `json:"departmentSource"`
	DepartmentRevision  string `json:"departmentRevision"`
	JobSource           string `json:"jobSource"`
	JobRevision         string `json:"jobRevision"`
	Position            string `json:"position"`
	Seats               int    `json:"seats"`
}
type LegalAdoption struct {
	f.Period       `json:",inline"`
	EntityName     string   `json:"entityName,omitempty"`
	EntityVersion  string   `json:"entityVersion"`
	EntityRevision string   `json:"entityRevision"`
	PurposeVersion string   `json:"purposeVersion"`
	Declaration    string   `json:"declaration"`
	Bindings       []string `json:"bindings"`
}
type PublishedTerms struct {
	Terms
	ReservationRequired bool            `json:"reservationRequired"`
	PersonName          string          `json:"personName"`
	EmployerName        string          `json:"employerName"`
	EmployerAdoption    []LegalAdoption `json:"employerAdoption"`
}

type Change struct {
	Operation    string `json:"operation"`
	Expected     string `json:"expected,omitempty"`
	BusinessDate string `json:"businessDate"`
	Basis        string `json:"basis"`
	Terms        *Terms `json:"terms,omitempty"`
}
type Revision struct {
	Version      string `json:"version"`
	Operation    string `json:"operation"`
	Status       string `json:"status"`
	BusinessDate string `json:"businessDate"`
	Basis        string `json:"basis"`
	RecordedAt   string `json:"recordedAt"`
}
type Offer struct {
	ID string `json:"id"`
	PublishedTerms
	Revision
	Segments  []AssignmentSegment `json:"segments"`
	Revisions []Revision          `json:"revisions"`
	Actions   []string            `json:"actions"`
}
type Receipt struct {
	SubmissionID string `json:"submissionId"`
	Outcome      string `json:"outcome"`
	ID           string `json:"id,omitempty"`
	Version      string `json:"version,omitempty"`
}
type Directory struct {
	CanIssue bool    `json:"canIssue"`
	Offers   []Offer `json:"offers"`
}
type Usage struct {
	Coverage string
	From     string
	Segments []Reservation
}
type Reservation struct {
	f.Period
	Source, Policy, Effect, Department, Job string
}
type Entry struct {
	CanRead  bool `json:"canRead"`
	CanIssue bool `json:"canIssue"`
}
type Offers interface {
	Entry(context.Context, f.Credentials, string) (Entry, error)
	List(context.Context, f.Credentials, string) (Directory, error)
	Get(context.Context, f.Credentials, string, string) (Offer, error)
	Submit(context.Context, f.Credentials, string, string, Change) (Receipt, error)
	Resolve(context.Context, f.Credentials, string) (Receipt, error)
}

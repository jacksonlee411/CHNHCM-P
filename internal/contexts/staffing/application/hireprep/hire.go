package hireprep

import (
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	staff "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	f "chnhcm.local/chnhcm/internal/foundation"
	"context"
)

type Adoption struct {
	Decision *DecisionRef              `json:"decision,omitempty"`
	Identity IdentitySource            `json:"identity"`
	Rule     string                    `json:"rule"`
	Legal    []org.LegalAdoption       `json:"legal"`
	Segments []staff.AssignmentSegment `json:"segments"`
}
type Hire struct {
	Person     string       `json:"person"`
	Employer   string       `json:"employer"`
	Department string       `json:"department"`
	Job        string       `json:"job"`
	BU         string       `json:"bu,omitempty"`
	Location   string       `json:"location,omitempty"`
	Arrival    string       `json:"arrival"`
	LastDay    string       `json:"lastDay,omitempty"`
	Decision   *DecisionRef `json:"decision,omitempty"`
	Notice     string       `json:"notice,omitempty"`
	Expected   *Adoption    `json:"expected,omitempty"`
}
type Preview struct {
	Adoption *Adoption `json:"adoption,omitempty"`
	Missing  []string  `json:"missing,omitempty"`
}
type Receipt struct {
	Submission   string `json:"submissionId"`
	Outcome      string `json:"outcome"`
	Action       string `json:"action,omitempty"`
	Employment   string `json:"employment,omitempty"`
	Assignment   string `json:"assignment,omitempty"`
	RecordNumber *int   `json:"recordNumber,omitempty"`
}
type Action struct {
	ID           string `json:"id"`
	Revision     string `json:"revision"`
	Kind         string `json:"kind"`
	Employment   string `json:"employment"`
	Assignment   string `json:"assignment"`
	RecordNumber int    `json:"recordNumber"`
	PrimaryJob   bool   `json:"primaryJob"`
	Person       string `json:"person"`
	Employer     string `json:"employer"`
	Department   string `json:"department"`
	Job          string `json:"job"`
	Notice       string `json:"notice"`
	RecordedAt   string `json:"recordedAt"`
	Original     string `json:"original,omitempty"`
	f.Period
	Cancelled  bool `json:"cancelled"`
	CanCancel  bool `json:"canCancel"`
	CanHistory bool `json:"canHistory"`
}
type Overview struct {
	CanHire     bool     `json:"canHire"`
	Actions     []Action `json:"actions"`
	People      int      `json:"people"`
	Assignments int      `json:"assignments"`
	Seats       int      `json:"seats"`
}
type Cancel struct {
	Revision string `json:"revision"`
	Reason   string `json:"reason"`
}
type Original struct {
	Appointment       *DecisionOption `json:"appointment,omitempty"`
	Action            Action          `json:"action"`
	Sources           *Adoption       `json:"sources,omitempty"`
	Reason            string          `json:"reason,omitempty"`
	State             string          `json:"state,omitempty"`
	PrimaryPurpose    string          `json:"primaryPurpose,omitempty"`
	PrimaryEmployment string          `json:"primaryEmployment,omitempty"`
	PrimaryAssignment string          `json:"primaryAssignment,omitempty"`
	OriginalRevision  string          `json:"originalRevision,omitempty"`
}
type EmployerSearch struct {
	Person     string `json:"person"`
	Arrival    string `json:"arrival"`
	LastDay    string `json:"lastDay,omitempty"`
	BU         string `json:"bu,omitempty"`
	Location   string `json:"location,omitempty"`
	Department string `json:"department,omitempty"`
	Search     string `json:"search,omitempty"`
	Page       int    `json:"page"`
}
type TargetSearch struct {
	Person     string `json:"person"`
	Kind       string `json:"kind"`
	Arrival    string `json:"arrival"`
	LastDay    string `json:"lastDay,omitempty"`
	Employer   string `json:"employer,omitempty"`
	Department string `json:"department,omitempty"`
	Search     string `json:"search,omitempty"`
	Page       int    `json:"page"`
}
type Hires interface {
	PublishDecision(context.Context, f.Credentials, string, DecisionPublication) (DecisionEdition, error)
	HireDecisions(context.Context, f.Credentials, string) ([]DecisionOption, error)
	HireTargets(context.Context, f.Credentials, TargetSearch) ([]org.EligibleEntity, error)
	HireEmployers(context.Context, f.Credentials, EmployerSearch) ([]org.EligibleEntity, error)
	OriginalAction(context.Context, f.Credentials, string) (Original, error)
	PreviewHire(context.Context, f.Credentials, Hire) (Preview, error)
	Hire(context.Context, f.Credentials, string, Hire) (Receipt, error)
	CancelHire(context.Context, f.Credentials, string, string, Cancel) (Receipt, error)
	ResolveHire(context.Context, f.Credentials, string) (Receipt, error)
	PersonEmployment(context.Context, f.Credentials, string, string) (Overview, error)
}

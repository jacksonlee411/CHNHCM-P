package hireprep

import (
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	staff "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	f "chnhcm.local/chnhcm/internal/foundation"
	"context"
)

type Adoption struct {
	Grade    *GradeRef                 `json:"grade,omitempty"`
	Decision *DecisionRef              `json:"decision,omitempty"`
	Identity IdentitySource            `json:"identity"`
	Rule     string                    `json:"rule"`
	Legal    []org.LegalAdoption       `json:"legal"`
	Segments []staff.AssignmentSegment `json:"segments"`
}
type Hire struct {
	Position   string       `json:"position,omitempty"`
	RehireOf   *ActionRef   `json:"rehireOf,omitempty"`
	Grade      string       `json:"grade,omitempty"`
	TrialStart string       `json:"trialStart,omitempty"`
	Person     string       `json:"person"`
	Employer   string       `json:"employer"`
	Department string       `json:"department,omitempty"`
	Job        string       `json:"job,omitempty"`
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
	Related      []ActionRef `json:"related,omitempty"`
	Revision     string      `json:"revision,omitempty"`
	Submission   string      `json:"submissionId"`
	Outcome      string      `json:"outcome"`
	Action       string      `json:"action,omitempty"`
	Employment   string      `json:"employment,omitempty"`
	Assignment   string      `json:"assignment,omitempty"`
	RecordNumber *int        `json:"recordNumber,omitempty"`
}
type Action struct {
	CanRehire            bool       `json:"canRehire"`
	Previous             *ActionRef `json:"-"`
	CanCorrectPromotion  bool       `json:"canCorrectPromotion"`
	CanWithdrawPromotion bool       `json:"canWithdrawPromotion"`
	Operation            string     `json:"operation"`
	CanCorrectDuty       bool       `json:"canCorrectDuty"`
	CanCorrectTransfer   bool       `json:"canCorrectTransfer"`
	CanMaintain          bool       `json:"canMaintain"`
	CanDelete            bool       `json:"canDelete"`
	CanCorrectLocation   bool       `json:"canCorrectLocation"`
	BusinessDate         string     `json:"businessDate"`
	SourceVersion        string     `json:"sourceVersion"`
	ID                   string     `json:"id"`
	Revision             string     `json:"revision"`
	Kind                 string     `json:"kind"`
	Employment           string     `json:"employment"`
	Assignment           string     `json:"assignment"`
	RecordNumber         int        `json:"recordNumber"`
	PrimaryJob           bool       `json:"primaryJob"`
	Person               string     `json:"person"`
	Employer             string     `json:"employer"`
	Department           string     `json:"department"`
	Job                  string     `json:"job"`
	Notice               string     `json:"notice"`
	RecordedAt           string     `json:"recordedAt"`
	Original             string     `json:"original,omitempty"`
	f.Period
	Cancelled  bool `json:"cancelled"`
	CanCancel  bool `json:"canCancel"`
	CanHistory bool `json:"canHistory"`
}
type TermLink struct {
	Assignment   string `json:"assignment"`
	RecordNumber int    `json:"recordNumber"`
	From         string `json:"from"`
}
type Overview struct {
	RehireFrom  *ActionRef `json:"rehireFrom,omitempty"`
	Terms       []TermLink `json:"terms"`
	CanHire     bool       `json:"canHire"`
	Actions     []Action   `json:"actions"`
	People      int        `json:"people"`
	Assignments int        `json:"assignments"`
	Seats       int        `json:"seats"`
}
type Cancel struct {
	Basis          *ActionRef `json:"basis,omitempty"`
	ExpectedSource string     `json:"expectedSourceVersion,omitempty"`
	Notice         string     `json:"notice,omitempty"`
	Revision       string     `json:"revision"`
	Reason         string     `json:"reason,omitempty"`
}
type RelationSummary struct {
	f.Period
	EmployerName string     `json:"employerName"`
	Terms        []TermLink `json:"terms"`
}
type ExchangeSearch struct {
	Ref          string `json:"ref"`
	BusinessDate string `json:"businessDate"`
	Page         int    `json:"page"`
}
type ExchangeOption struct {
	Assignment   string `json:"assignment"`
	Source       string `json:"sourceVersion"`
	Name         string `json:"name"`
	RecordNumber int    `json:"recordNumber"`
	Position     string `json:"positionName"`
	Target       string `json:"targetName"`
}
type ExchangeCommand struct {
	Operation    string     `json:"operation"`
	Expected     string     `json:"expectedSourceVersion"`
	PeerExpected string     `json:"peerSourceVersion"`
	Peer         string     `json:"peerAssignment"`
	BusinessDate string     `json:"businessDate,omitempty"`
	Notice       string     `json:"notice,omitempty"`
	Action       *ActionRef `json:"action,omitempty"`
}
type ExchangeLink struct {
	PeerSource string     `json:"peerSourceVersion,omitempty"`
	CanRedate  bool       `json:"canRedate"`
	CanDelete  bool       `json:"canDelete"`
	Group      string     `json:"group"`
	Peer       string     `json:"peerAssignment,omitempty"`
	Original   *ActionRef `json:"peerAction,omitempty"`
}
type Original struct {
	Exchange          *ExchangeLink      `json:"exchange,omitempty"`
	Manager           *ManagerFact       `json:"manager,omitempty"`
	Exit              *ExitEvidence      `json:"exit,omitempty"`
	RehireOf          *ActionRef         `json:"rehireOf,omitempty"`
	NoticeRef         *ActionRef         `json:"noticeRef,omitempty"`
	Promotion         *PromotionEvidence `json:"promotion,omitempty"`
	Previous          *ActionRef         `json:"previous,omitempty"`
	Relation          *RelationSummary   `json:"relation,omitempty"`
	TrialStart        string             `json:"trialStart,omitempty"`
	Probation         *ProbationEvidence `json:"probation,omitempty"`
	Secondary         *SecondaryEvidence `json:"secondary,omitempty"`
	Transfer          *TransferEvidence  `json:"transfer,omitempty"`
	Requires          *RequiredFact      `json:"requires,omitempty"`
	DepartmentName    string             `json:"departmentName,omitempty"`
	Revisions         []Revision         `json:"revisions,omitempty"`
	LocationChange    *LocationEvidence  `json:"locationChange,omitempty"`
	Appointment       *DecisionOption    `json:"appointment,omitempty"`
	Action            Action             `json:"action"`
	Sources           *Adoption          `json:"sources,omitempty"`
	Reason            string             `json:"reason,omitempty"`
	State             string             `json:"state,omitempty"`
	PrimaryPurpose    string             `json:"primaryPurpose,omitempty"`
	PrimaryEmployment string             `json:"primaryEmployment,omitempty"`
	PrimaryAssignment string             `json:"primaryAssignment,omitempty"`
	OriginalRevision  string             `json:"originalRevision,omitempty"`
}
type EmployerSearch struct {
	RehireOf   *ActionRef `json:"rehireOf,omitempty"`
	Person     string     `json:"person"`
	Arrival    string     `json:"arrival"`
	LastDay    string     `json:"lastDay,omitempty"`
	BU         string     `json:"bu,omitempty"`
	Location   string     `json:"location,omitempty"`
	Department string     `json:"department,omitempty"`
	Search     string     `json:"search,omitempty"`
	Page       int        `json:"page"`
}
type TargetSearch struct {
	RehireOf   *ActionRef `json:"rehireOf,omitempty"`
	Job        string     `json:"job,omitempty"`
	Person     string     `json:"person"`
	Kind       string     `json:"kind"`
	Arrival    string     `json:"arrival"`
	LastDay    string     `json:"lastDay,omitempty"`
	Employer   string     `json:"employer,omitempty"`
	Department string     `json:"department,omitempty"`
	Search     string     `json:"search,omitempty"`
	Page       int        `json:"page"`
}
type ArrangementQuery struct {
	Position string     `json:"position,omitempty"`
	Person   string     `json:"person"`
	Arrival  string     `json:"arrival"`
	LastDay  string     `json:"lastDay,omitempty"`
	RehireOf *ActionRef `json:"rehireOf,omitempty"`
}
type ArrangementMode struct {
	Department       *org.EligibleEntity `json:"department,omitempty"`
	Job              *org.EligibleEntity `json:"job,omitempty"`
	JobRequired      bool                `json:"jobRequired"`
	PositionRequired bool                `json:"positionRequired"`
}
type Hires interface {
	ExchangeOptions(context.Context, f.Credentials, string, ExchangeSearch) ([]ExchangeOption, error)
	ExchangePositions(context.Context, f.Credentials, string, string, ExchangeCommand) (Receipt, error)
	HireArrangement(context.Context, f.Credentials, ArrangementQuery) (ArrangementMode, error)
	JobInformation
	PublishDecision(context.Context, f.Credentials, string, DecisionPublication) (DecisionEdition, error)
	HireDecisions(context.Context, f.Credentials, string, *ActionRef) ([]DecisionOption, error)
	HireTargets(context.Context, f.Credentials, TargetSearch) ([]org.EligibleEntity, error)
	HireEmployers(context.Context, f.Credentials, EmployerSearch) ([]org.EligibleEntity, error)
	OriginalAction(context.Context, f.Credentials, string, string) (Original, error)
	PreviewHire(context.Context, f.Credentials, Hire) (Preview, error)
	Hire(context.Context, f.Credentials, string, Hire) (Receipt, error)
	CancelHire(context.Context, f.Credentials, string, string, Cancel) (Receipt, error)
	ResolveHire(context.Context, f.Credentials, string) (Receipt, error)
	PersonEmployment(context.Context, f.Credentials, string, string) (Overview, error)
}

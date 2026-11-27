package public

import (
	"context"
	"encoding/json"

	"chnhcm.local/chnhcm/internal/foundation"
)

// A changes map distinguishes an omitted protected value from an explicit null.
// Each supported field is decoded and authorized separately at the command boundary.
type PlanChanges map[string]json.RawMessage

type WritePlan struct {
	Action           string      `json:"action"`
	Code             string      `json:"code,omitempty"`
	EffectiveDate    string      `json:"effectiveDate,omitempty"`
	Reason           string      `json:"reason,omitempty"`
	RecordID         string      `json:"recordId,omitempty"`
	ExpectedVersion  string      `json:"expectedVersion,omitempty"`
	ExpectedRevision string      `json:"expectedRevision,omitempty"`
	Changes          PlanChanges `json:"changes,omitempty"`
}
type ScopeMember struct {
	Dimension string `json:"dimension"`
	ID        string `json:"id"`
	Exclude   bool   `json:"exclude,omitempty"`
	Code      string `json:"code,omitempty"`
	Name      string `json:"name,omitempty"`
	Source    string `json:"sourceVersion,omitempty"`
	RecordID  string `json:"recordId,omitempty"`
	Revision  string `json:"revision,omitempty"`
}
type PlanScope struct {
	Restricted bool          `json:"restricted,omitempty"`
	Mode       string        `json:"mode"`
	Version    string        `json:"version"`
	Members    []ScopeMember `json:"members,omitempty"`
}
type PlanRate struct {
	GradeID     string      `json:"gradeId"`
	RateID      string      `json:"rateId"`
	Name        string      `json:"name,omitempty"`
	Code        string      `json:"code,omitempty"`
	BusinessAt  string      `json:"businessAt,omitempty"`
	Record      *RateRecord `json:"record,omitempty"`
	Adopted     *Reference  `json:"adopted,omitempty"`
	AdoptedHead string      `json:"adoptedHead,omitempty"`
	Open        bool        `json:"open,omitempty"`
}
type Target struct {
	GradeID string `json:"gradeId,omitempty"`
	Amount  string `json:"amount"`
}
type PlanComponent struct {
	ID        string     `json:"id"`
	Code      string     `json:"code"`
	Name      string     `json:"name"`
	Nature    string     `json:"nature"`
	Currency  string     `json:"currency"`
	Frequency string     `json:"frequency"`
	Unit      string     `json:"unit"`
	Precision int        `json:"precision"`
	Method    string     `json:"method"`
	Amount    string     `json:"amount,omitempty"`
	Rates     []PlanRate `json:"rates,omitempty"`
	BaseID    string     `json:"baseId,omitempty"`
	Percent   string     `json:"percent,omitempty"`
	Rounding  string     `json:"rounding,omitempty"`
	Targets   []Target   `json:"targets,omitempty"`
}
type PlanRecord struct {
	ID          string          `json:"recordId"`
	Revision    string          `json:"revision"`
	Start       string          `json:"effectiveDate,omitempty"`
	Until       string          `json:"endExclusive,omitempty"`
	Status      string          `json:"status,omitempty"`
	Operation   string          `json:"operation,omitempty"`
	Reason      string          `json:"reason,omitempty"`
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	Scope       *PlanScope      `json:"scope,omitempty"`
	Components  []PlanComponent `json:"components,omitempty"`
}
type Plan struct {
	ID      string     `json:"id"`
	Version string     `json:"sourceVersion"`
	Code    string     `json:"code,omitempty"`
	Record  PlanRecord `json:"record"`
}
type PlanOptions struct {
	ScopeDimensions []string          `json:"scopeDimensions"`
	Units           map[string]string `json:"units"`
}
type PlanContext struct {
	PlanOptions
	Reasons     []string `json:"reasons"`
	DefaultDate string   `json:"defaultEffectiveDate"`
	Writable    []string `json:"writable"`
}
type PlanDetail struct {
	PlanOptions
	Plan
	Today      string              `json:"today"`
	BusinessAt string              `json:"businessAt"`
	Readable   []string            `json:"readable"`
	Dates      []DateNode          `json:"dates"`
	History    bool                `json:"history"`
	Actions    map[string][]string `json:"actions"`
	Reasons    map[string][]string `json:"reasons"`
	Windows    map[string]Window   `json:"windows"`
}
type PlanList struct {
	Readable []string `json:"readable"`
	Items    []Plan   `json:"items"`
	Total    int      `json:"total"`
	Today    string   `json:"today"`
	Create   bool     `json:"create"`
}
type PlanChange struct {
	Code       string      `json:"code,omitempty"`
	Version    string      `json:"sourceVersion"`
	Action     string      `json:"action"`
	RecordedAt string      `json:"recordedAt"`
	Operator   string      `json:"operator,omitempty"`
	Before     *PlanRecord `json:"before,omitempty"`
	After      *PlanRecord `json:"after,omitempty"`
}
type RateCandidate struct {
	From      string `json:"effectiveDate,omitempty"`
	Until     string `json:"endExclusive,omitempty"`
	RateID    string `json:"rateId"`
	GradeID   string `json:"gradeId"`
	Name      string `json:"name"`
	Code      string `json:"code,omitempty"`
	Currency  string `json:"currency"`
	Frequency string `json:"frequency"`
	Unit      string `json:"unit"`
	Precision int    `json:"precision"`
}
type PlanService interface {
	PlanContext(context.Context, foundation.Credentials) (PlanContext, error)
	ListPlans(context.Context, foundation.Credentials, RateQuery) (PlanList, error)
	ReadPlan(context.Context, foundation.Credentials, string, string, string) (PlanDetail, error)
	PlanHistory(context.Context, foundation.Credentials, string, string) ([]PlanChange, error)
	SavePlan(context.Context, foundation.Credentials, string, string, WritePlan) (Receipt, error)
	PlanReceipt(context.Context, foundation.Credentials, string) (Receipt, error)
	ResolvePlan(context.Context, foundation.Credentials, string) (Receipt, error)
	RateCandidates(context.Context, foundation.Credentials, string, string, string, string, int) ([]RateCandidate, error)
	PlanApplicable(context.Context, foundation.Credentials, string, string, map[string]string) (bool, error)
}
type ConfigurationService interface {
	RateService
	PlanService
}

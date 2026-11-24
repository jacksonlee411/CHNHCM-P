package public

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
)

type AssignmentPolicy struct {
	foundation.Period `json:",inline"`
	ID                string `json:"id"`
	Revision          string `json:"revision,omitempty"`
	Category          string `json:"category"`
	Purpose           string `json:"purpose"`
	Mode              string `json:"mode"` // JobOnly, PositionExempt or PositionCounted
}
type AssignmentCheck struct {
	Adopted           []AssignmentSegment `json:"adopted,omitempty"`
	foundation.Period `json:",inline"`
	Enterprise        string `json:"enterprise"`
	Category          string `json:"category"`
	Purpose           string `json:"purpose"`
	Department        string `json:"department,omitempty"`
	Job               string `json:"job,omitempty"`
	Position          string `json:"position,omitempty"`
	Expected          string `json:"expectedVersion,omitempty"`
}
type AssignmentSegment struct {
	DepartmentID        string `json:"departmentId,omitempty"`
	JobID               string `json:"jobId,omitempty"`
	PositionName        string `json:"positionName,omitempty"`
	PositionSource      string `json:"positionSource,omitempty"`
	PositionRevision    string `json:"positionRevision,omitempty"`
	ReplenishmentSource string `json:"replenishmentSource,omitempty"`
	CoverageSource      string `json:"coverageSource,omitempty"`
	DepartmentName      string `json:"departmentName,omitempty"`
	JobName             string `json:"jobName,omitempty"`
	foundation.Period   `json:",inline"`
	PolicyVersion       string `json:"policyVersion"`
	Policy              string `json:"policy"`
	DepartmentSource    string `json:"departmentSource"`
	DepartmentRevision  string `json:"departmentRevision"`
	JobSource           string `json:"jobSource"`
	JobRevision         string `json:"jobRevision"`
	Position            string `json:"position"`
	Seats               int    `json:"seats"`
}
type AssignmentPolicies interface {
	Policies(context.Context, foundation.Credentials, string) (foundation.DatedRules[AssignmentPolicy], error)
	PublishPolicy(context.Context, foundation.Credentials, string, foundation.RuleChange[AssignmentPolicy]) (foundation.RuleReceipt, error)
	ResolvePolicy(context.Context, foundation.Credentials, string) (foundation.RuleReceipt, error)
	CheckAssignment(context.Context, foundation.Credentials, AssignmentCheck) ([]AssignmentSegment, error)
}

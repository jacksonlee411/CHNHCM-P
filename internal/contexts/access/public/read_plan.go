package public

import (
	"context"
	"slices"
	"time"
)

// ReadAction admits only guarded read projections; it cannot authorize writes.
type ReadAction string

const (
	DepartmentRead      ReadAction = "department.read"
	DepartmentReference ReadAction = "department.reference.read"
	DepartmentHistory   ReadAction = "department.history.read"
	DepartmentExport    ReadAction = "department.export"
	BURead              ReadAction = "bu.read"
	BUReference         ReadAction = "bu.reference.read"
)

// ReadSet is a union of identities compiled by X03, not a grant/permission DSL.
// All is relative to the plan's one enterprise. SQL may only test membership.
type ReadSet struct {
	All         bool     `json:"all"`
	BUs         []string `json:"bus"`
	Departments []string `json:"departments"`
}

func (s ReadSet) Contains(bu, department string) bool {
	return s.All || slices.Contains(s.BUs, bu) || slices.Contains(s.Departments, department)
}

// ReadPlan is immutable, request-local material under the originating Guard.
// Current M02 scope facts must be validated independently of any historical path
// date. The plan is never an HTTP input, cached grant result or write capability.
type ReadPlan struct {
	Origin                                      ReadPlanner `json:"-"`
	Instance, Enterprise, Subject, Session, Day string
	At                                          time.Time
	Action                                      ReadAction
	Allowed                                     ReadSet            `json:"allowed"`
	Fields                                      map[string]ReadSet `json:"fields"`
}

type ReadPlanner interface {
	PlanReads(context.Context, string, time.Time, ReadAction) (ReadPlan, error)
	DecidePlan(context.Context, ReadPlan, ReadIntent) (Decision, error)
}

package public

import "context"

type PayAssignment struct {
	Person        string `json:"personId"`
	Assignment    string `json:"assignmentId"`
	Employment    string `json:"employmentId"`
	Employer      string `json:"employerId"`
	Source        string `json:"assignmentSource"`
	Department    string `json:"departmentId"`
	Job           string `json:"jobId"`
	Family        string `json:"familyId,omitempty"`
	JobSource     string `json:"jobSource,omitempty"`
	JobRevision   string `json:"jobRevision,omitempty"`
	Grade         string `json:"gradeId,omitempty"`
	GradeName     string `json:"gradeName,omitempty"`
	SegmentFrom   string `json:"segmentFrom,omitempty"`
	SegmentUntil  string `json:"segmentUntil,omitempty"`
	JobRecord     string `json:"jobRecord,omitempty"`
	GradeSource   string `json:"gradeSource,omitempty"`
	GradeAdoption string `json:"gradeAdoption,omitempty"`
	From          string `json:"from"`
	Until         string `json:"until,omitempty"`
	Active        bool   `json:"active"`
}
type PayAssignments interface {
	PayAssignment(context.Context, string, string, string, string, bool) (PayAssignment, error)
}

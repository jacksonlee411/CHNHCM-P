package hireprep

import (
	"context"

	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	f "chnhcm.local/chnhcm/internal/foundation"
)

type LocationRef struct {
	ID       string `json:"id"`
	Name     string `json:"name,omitempty"`
	Source   string `json:"sourceVersion,omitempty"`
	Revision string `json:"revision,omitempty"`
}

type LocationChange struct {
	BusinessDate     string `json:"businessDate"`
	Location         string `json:"location"`
	ExpectedSource   string `json:"expectedSourceVersion"`
	ExpectedRevision string `json:"expectedRevision,omitempty"`
}

type LocationEvidence struct {
	Replaced *LocationRef        `json:"replaced,omitempty"`
	Reason   string              `json:"reason"`
	Before   *LocationRef        `json:"before,omitempty"`
	After    LocationRef         `json:"after"`
	Legal    []org.LegalAdoption `json:"legal"`
}

type Revision struct {
	Operator   string `json:"operator"`
	ID         string `json:"id"`
	Actor      string `json:"actor"`
	RecordedAt string `json:"recordedAt"`
}

type State struct {
	f.Period
	Department string       `json:"department"`
	Job        string       `json:"job"`
	Location   *LocationRef `json:"location,omitempty"`
	Seats      int          `json:"seats"`
}

type AssignmentView struct {
	f.Period
	EmployerName      string   `json:"employerName"`
	Relationship      string   `json:"relationship"`
	Name              string   `json:"name"`
	Number            string   `json:"number"`
	Person            string   `json:"person"`
	Assignment        string   `json:"assignment"`
	Employment        string   `json:"employment"`
	RecordNumber      int      `json:"recordNumber"`
	PrimaryJob        bool     `json:"primaryJob"`
	SourceVersion     string   `json:"sourceVersion"`
	BusinessAt        string   `json:"businessAt"`
	State             *State   `json:"state,omitempty"`
	Actions           []Action `json:"actions"`
	CanChangeLocation bool     `json:"canChangeLocation"`
}
type LocationSearch struct {
	BusinessDate string `json:"businessDate"`
	Search       string `json:"search,omitempty"`
	Page         int    `json:"page"`
}
type AssignmentQuery struct {
	Anchor     string `json:"anchor,omitempty"`
	BusinessAt string `json:"businessAt,omitempty"`
	Department string `json:"department,omitempty"`
	Person     string `json:"person,omitempty"`
	Ref        string `json:"ref,omitempty"`
	Page       int    `json:"page"`
	PageSize   int    `json:"pageSize"`
	Sort       string `json:"sort,omitempty"`
}
type EmployeeRow struct {
	ID             string `json:"id"`
	Person         string `json:"person"`
	Employment     string `json:"employment"`
	Name           string `json:"name"`
	Number         string `json:"number"`
	RecordNumber   int    `json:"recordNumber"`
	PrimaryJob     bool   `json:"primaryJob"`
	Department     string `json:"department"`
	DepartmentName string `json:"departmentName"`
	JobName        string `json:"jobName"`
	f.Period
}
type EmployeeDepartment struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Parent string `json:"parent,omitempty"`
}
type AssignmentLink struct {
	Person     string `json:"person"`
	Assignment string `json:"assignment"`
}
type AssignmentNav struct {
	Position int             `json:"position"`
	Previous *AssignmentLink `json:"previous,omitempty"`
	Next     *AssignmentLink `json:"next,omitempty"`
}
type EmployeeDirectory struct {
	Navigation  *AssignmentNav       `json:"navigation,omitempty"`
	Rows        []EmployeeRow        `json:"rows"`
	Departments []EmployeeDepartment `json:"departments"`
	People      int                  `json:"people"`
	Total       int                  `json:"total"`
	BusinessAt  string               `json:"businessAt"`
	Keyword     string               `json:"keyword"`
}
type JobInformation interface {
	CanListRegister(context.Context, f.Credentials) (bool, error)
	ListRegister(context.Context, f.Credentials, RegisterQuery) (RegisterResult, error)
	CanListAssignments(context.Context, f.Credentials) (bool, error)
	ListAssignments(context.Context, f.Credentials, AssignmentQuery) (EmployeeDirectory, error)
	AssignmentInfo(context.Context, f.Credentials, string, string, string) (AssignmentView, error)
	LocationOptions(context.Context, f.Credentials, string, LocationSearch) ([]LocationRef, error)
	ChangeLocation(context.Context, f.Credentials, string, string, string, LocationChange) (Receipt, error)
}

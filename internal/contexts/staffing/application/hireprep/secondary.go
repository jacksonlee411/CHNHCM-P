package hireprep

import (
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	staff "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	f "chnhcm.local/chnhcm/internal/foundation"
	"strings"
	"unicode/utf8"
)

func StartsAssignment(kind string) bool {
	return kind == "Hire" || kind == "Rehire" || kind == "AdditionalAssignment" || kind == "ParallelEmployment"
}

type Secondary struct {
	BU         string `json:"bu,omitempty"`
	Location   string `json:"location,omitempty"`
	From       string `json:"from"`
	LastDay    string `json:"lastDay,omitempty"`
	Parallel   bool   `json:"parallel"`
	Employer   string `json:"employer,omitempty"`
	Department string `json:"department"`
	Job        string `json:"job"`
	Notice     string `json:"notice"`
	Expected   string `json:"expectedSourceVersion"`
}

func (in Secondary) Validate() error {
	if in.BU != "" && !f.ValidID(in.BU) || in.Location != "" && !f.ValidID(in.Location) || !f.ValidID(in.Expected) || !f.ValidID(in.Job) || !f.ValidID(in.Department) || in.Parallel != (in.Employer != "") || (in.Parallel && !f.ValidID(in.Employer)) || strings.TrimSpace(in.Notice) == "" || utf8.RuneCountInString(in.Notice) > 1000 {
		return f.Fail(400, "SECONDARY_INPUT", "请核对所属关系、任职安排及任用依据")
	}
	return nil
}

type SecondaryOption struct {
	BU         string `json:"bu,omitempty"`
	Location   string `json:"location,omitempty"`
	From       string `json:"from"`
	LastDay    string `json:"lastDay,omitempty"`
	Parallel   bool   `json:"parallel"`
	Kind       string `json:"kind"`
	Employer   string `json:"employer,omitempty"`
	Department string `json:"department,omitempty"`
	Search     string `json:"search,omitempty"`
	Page       int    `json:"page"`
}
type SecondaryEvidence struct {
	RelationshipPrimary bool                      `json:"relationshipPrimary"`
	Legal               []org.LegalAdoption       `json:"legal"`
	Segments            []staff.AssignmentSegment `json:"segments"`
}

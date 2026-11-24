package hireprep

import (
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	f "chnhcm.local/chnhcm/internal/foundation"
	"strings"
	"unicode/utf8"
)

// These are the identities and display names on the issued original.
// An omitted arrangement may be completed during hiring.
type DecisionRow struct {
	Position   *org.EligibleEntity `json:"position,omitempty"`
	Person     string              `json:"person"`
	Line       string              `json:"line,omitempty"`
	Employer   *org.EligibleEntity `json:"employer,omitempty"`
	Department *org.EligibleEntity `json:"department,omitempty"`
	BU         *org.EligibleEntity `json:"bu,omitempty"`
	Location   *org.EligibleEntity `json:"location,omitempty"`
	Job        *org.EligibleEntity `json:"job,omitempty"`
	Arrival    string              `json:"arrival,omitempty"`
	LastDay    string              `json:"lastDay,omitempty"`
}
type DecisionPublication struct {
	Source    string        `json:"source"`
	Expected  string        `json:"expected,omitempty"`
	Number    string        `json:"number"`
	Roster    bool          `json:"roster"`
	Withdrawn bool          `json:"withdrawn,omitempty"`
	Rows      []DecisionRow `json:"rows"`
}
type DecisionEdition struct {
	Source  string `json:"source"`
	Version string `json:"version"`
}
type DecisionOption struct {
	Ref    DecisionRef `json:"ref"`
	Number string      `json:"number"`
	DecisionRow
}

func (p DecisionPublication) Validate() error {
	invalid := f.Fail(400, "APPOINTMENT_INPUT", "请核对原单编号、人员行和拟任安排")
	if !f.ValidID(p.Source) || (p.Expected != "" && !f.ValidID(p.Expected)) || strings.TrimSpace(p.Number) == "" || strings.TrimSpace(p.Number) != p.Number || utf8.RuneCountInString(p.Number) > 120 {
		return invalid
	}
	if p.Withdrawn {
		if p.Expected == "" || len(p.Rows) != 0 {
			return invalid
		}
		return nil
	}
	if len(p.Rows) == 0 || (!p.Roster && len(p.Rows) != 1) {
		return invalid
	}
	people, lines := map[string]bool{}, map[string]bool{}
	for _, row := range p.Rows {
		if !f.ValidID(row.Person) || people[row.Person] || lines[row.Line] || (p.Roster && !f.ValidID(row.Line)) || (!p.Roster && row.Line != "") {
			return invalid
		}
		people[row.Person], lines[row.Line] = true, true
		if row.Arrival != "" && !f.ValidDate(row.Arrival) || row.LastDay != "" && (!f.ValidDate(row.LastDay) || row.LastDay < row.Arrival) {
			return invalid
		}
		for _, target := range []*org.EligibleEntity{row.Employer, row.Department, row.Job, row.Position, row.BU, row.Location} {
			if target != nil && (!f.ValidID(target.ID) || strings.TrimSpace(target.Name) == "" || utf8.RuneCountInString(target.Name) > 400) {
				return invalid
			}
		}
	}
	return nil
}

package domain

import (
	f "chnhcm.local/chnhcm/internal/foundation"
	"strings"
	"time"
	"unicode/utf8"
)

type Terms struct {
	Number, Person, Employer, Department, Job, Position, Arrival, ReserveFrom, LastDay string
}

func ValidateTerms(t Terms) (f.Period, error) {
	span := f.Period{From: t.ReserveFrom}
	if t.LastDay != "" {
		end, e := time.Parse(time.DateOnly, t.LastDay)
		if e != nil {
			return span, f.Fail(400, "OFFER_INPUT", "任用结束日期无效")
		}
		span.Until = end.AddDate(0, 0, 1).Format(time.DateOnly)
	}
	if !span.Valid() || !f.ValidDate(t.Arrival) || !span.Includes(t.Arrival) || !f.ValidID(t.Person) || !f.ValidID(t.Employer) ||
		strings.TrimSpace(t.Number) == "" || utf8.RuneCountInString(t.Number) > 160 ||
		t.Position != "" && !f.ValidID(t.Position) || (t.Position == "" || t.Department != "") && !f.ValidID(t.Department) || (t.Position == "" || t.Job != "") && !f.ValidID(t.Job) {
		return f.Period{}, f.Fail(400, "OFFER_INPUT", "请核对录用编号、人员、任用安排和日期")
	}
	return span, nil
}
func Next(status, op string, held bool) (string, error) {
	switch op {
	case "issue":
		if status == "" {
			return "issued", nil
		}
	case "reschedule":
		if status == "accepted" || status == "issued" {
			return "issued", nil
		}
	case "accept":
		if status == "issued" {
			return "accepted", nil
		}
	case "decline":
		if status == "issued" || status == "accepted" {
			return "declined", nil
		}
	case "release":
		if (status == "issued" || status == "accepted") && held {
			return status, nil
		}
	case "reserve":
		if (status == "issued" || status == "accepted") && !held {
			return status, nil
		}
	}
	return "", f.Fail(409, "OFFER_STATE", "录用已变化，请核对当前状态和份额")
}

// PromisedArrangement compares each day of the current commitment with the issued
// binding. Refreshed source versions may split an otherwise identical interval.
type Binding struct {
	f.Period
	Position, DepartmentID, JobID string
	Seats                         int
}

func PromisedArrangement(saved, current []Binding) bool {
	var days []string
	for _, segment := range saved {
		days = append(days, segment.From, segment.Until)
	}
	for _, proposed := range current {
		for _, span := range proposed.Split(days) {
			found := false
			for _, original := range saved {
				if original.Includes(span.From) && original.Position == proposed.Position && original.Seats == proposed.Seats &&
					original.DepartmentID == proposed.DepartmentID && original.JobID == proposed.JobID {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}
	return len(current) > 0
}

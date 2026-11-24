package hireprep

import (
	"slices"

	staff "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	f "chnhcm.local/chnhcm/internal/foundation"
)

// A transfer defines the two target identifiers and adopts the applicable M03
// segments. A separate location action changes only location.
type JobTransfer struct {
	Department string                    `json:"department,omitempty"`
	Job        string                    `json:"job,omitempty"`
	Segments   []staff.AssignmentSegment `json:"segments,omitempty"`
}
type Placement struct {
	f.Period
	Department string
	Job        string
	Adoption   staff.AssignmentSegment
	Location   *LocationRef
	Grade      *GradeRef
}
type RequiredFact struct {
	Action     string `json:"action"`
	Revision   string `json:"revision"`
	Location   string `json:"location,omitempty"`
	Department string `json:"department,omitempty"`
	Job        string `json:"job,omitempty"`
}
type HistoryEntry struct {
	ID, Revision, Day, Kind string
	Removed                 bool
	Location                *LocationRef
	Transfer                *JobTransfer
	Grade                   *GradeRef
	Previous                *ActionRef
	Requires                *RequiredFact
}
type HistoryChange struct {
	Alternate        *Placement
	Before, After    Placement
	RequiredRevision string
}
type HistoryProjection struct {
	Segments []Placement
	Changes  map[string]HistoryChange
}

func historyConflict(message string) error {
	return f.Fail(409, "ASSIGNMENT_HISTORY_CONFLICT", message)
}

// ProjectHistory uses business dates and explicit facts. Independent field
// changes on a date read the same starting state. No registration order is used.
func ProjectHistory(term f.Period, initial []Placement, history []HistoryEntry) (HistoryProjection, error) {
	result := HistoryProjection{Changes: make(map[string]HistoryChange)}
	days := []string{term.From}
	if !term.Valid() || len(initial) == 0 {
		return result, historyConflict("入职原来源未覆盖任期")
	}
	for _, s := range initial {
		days = append(days, s.From, s.Until)
	}
	entries := make(map[string]HistoryEntry)
	byDay := map[string][]HistoryEntry{}
	for _, r := range history {
		if r.Removed {
			continue
		}
		if !term.Includes(r.Day) {
			return result, historyConflict("办理日期不在任期内")
		}
		entries[r.ID] = r
		days = append(days, r.Day)
		byDay[r.Day] = append(byDay[r.Day], r)
		if r.Transfer != nil {
			for _, segment := range r.Transfer.Segments {
				if term.Includes(segment.From) {
					days = append(days, segment.From)
				}
				if term.Includes(segment.Until) {
					days = append(days, segment.Until)
				}
			}
		}
	}
	var activeTransfer *JobTransfer
	var activeLocation *LocationRef
	var activeGrade *GradeRef
	for _, span := range term.Split(days) {
		day := span.From
		base, ok := placementAt(initial, day)
		if !ok {
			return result, historyConflict("入职原来源存在未覆盖期间")
		}
		state := base
		// An adoption can end exactly where the next transfer starts. The last
		// outgoing interval still supplies the before-image of the new action.
		if activeTransfer != nil {
			if e := applyTransfer(&state, activeTransfer, day); e != nil {
				if len(result.Segments) == 0 || !slices.ContainsFunc(byDay[day], func(r HistoryEntry) bool { return r.Transfer != nil }) {
					return result, e
				}
				last := result.Segments[len(result.Segments)-1]
				state.Department, state.Job, state.Adoption = last.Department, last.Job, last.Adoption
			}
		}
		if activeLocation != nil {
			state.Location = activeLocation
		}
		if activeGrade != nil {
			state.Grade = activeGrade
		}
		dayStart := state
		var location, transfer, grade bool
		ordered, err := OrderDay(byDay[day])
		if err != nil {
			return result, err
		}
		for _, r := range ordered {
			before := dayStart
			if r.Previous != nil {
				if pred, ok := entries[r.Previous.Action]; ok {
					if pred.Day > day {
						return result, historyConflict("前序办理晚于本项生效日期")
					}
					if pred.Day == day {
						before = result.Changes[pred.ID].After
					}
				}
			}
			change := HistoryChange{Before: before}
			if req := r.Requires; req != nil {
				pred, ok := entries[req.Action]
				if !ok || pred.Day >= day || (req.Location != "" && (before.Location == nil || before.Location.ID != req.Location)) ||
					(req.Department != "" && before.Department != req.Department) || (req.Job != "" && before.Job != req.Job) {
					return result, historyConflict("后续办理所依赖的原事实无法保留")
				}
				change.RequiredRevision = pred.Revision
			}
			after := before
			if r.Kind == "Promotion" || r.Kind == "GradeRecord" {
				if grade || r.Grade == nil || (r.Kind == "Promotion" && (before.Grade == nil || before.Grade.ID == r.Grade.ID)) || (r.Kind == "GradeRecord" && before.Grade != nil) {
					return result, historyConflict("请核对该任期的实际原职级、拟任职级及同日办理")
				}
				grade = true
				activeGrade = r.Grade
				state.Grade, after.Grade = r.Grade, r.Grade
			}
			switch r.Kind {
			case "DataChange":
				if r.Location == nil || location {
					return result, historyConflict("同日的工作地点存在冲突")
				}
				location = true
				activeLocation = r.Location
				state.Location = r.Location
				after.Location = r.Location
			case "GradeRecord":
			case "Promotion", "Transfer":
				if r.Kind == "Promotion" && r.Transfer == nil {
					break
				}
				if r.Transfer == nil || transfer {
					return result, historyConflict("同日的调动存在冲突")
				}
				transfer = true
				activeTransfer = r.Transfer
				if e := applyTransfer(&state, r.Transfer, day); e != nil {
					return result, e
				}
				after.Department, after.Job, after.Adoption = state.Department, state.Job, state.Adoption
			default:
				return result, historyConflict("该办理类型不能用于职务历史维护")
			}
			change.After = after
			result.Changes[r.ID] = change
		}
		// An unlinked promotion and transfer can have two lawful before-images.
		// Preserve that distinction for authorization, qualification and reporting.
		var promotion, move HistoryEntry
		for _, r := range ordered {
			if r.Kind == "Promotion" {
				promotion = r
			}
			if r.Kind == "Transfer" {
				move = r
			}
		}
		if promotion.ID != "" && move.ID != "" && !follows(promotion, move.ID, entries) && !follows(move, promotion.ID, entries) {
			alternative := result.Changes[move.ID].After
			change := result.Changes[promotion.ID]
			change.Alternate = &alternative
			result.Changes[promotion.ID] = change
		}
		state.Period = span
		result.Segments = append(result.Segments, state)
	}
	return result, nil
}

func follows(item HistoryEntry, ancestor string, entries map[string]HistoryEntry) bool {
	day := item.Day
	for item.Previous != nil {
		if item.Previous.Action == ancestor {
			return true
		}
		item = entries[item.Previous.Action]
		if item.Day != day {
			return false
		}
	}
	return false
}

func placementAt(rows []Placement, day string) (Placement, bool) {
	for _, r := range rows {
		if r.Includes(day) {
			return r, true
		}
	}
	return Placement{}, false
}
func applyTransfer(state *Placement, transfer *JobTransfer, day string) error {
	for _, s := range transfer.Segments {
		if s.Includes(day) {
			state.Department, state.Job, state.Adoption = transfer.Department, transfer.Job, s
			return nil
		}
	}
	return historyConflict("调动后的部门、职务或任职政策未覆盖当期")
}

// RecordMaintenance applies an operation to the original business identity.
// Requires identifies a fact explicitly cited by the decision.
type RecordMaintenance struct {
	ExpectedSource   string        `json:"expectedSourceVersion"`
	ExpectedRevision string        `json:"expectedRevision,omitempty"`
	BusinessDate     string        `json:"businessDate,omitempty"`
	Department       string        `json:"department,omitempty"`
	Job              string        `json:"job,omitempty"`
	Notice           string        `json:"notice,omitempty"`
	Insert           bool          `json:"insert,omitempty"`
	Requires         *RequiredFact `json:"requires,omitempty"`
	Previous         *ActionRef    `json:"previous,omitempty"`
}
type TransferEvidence struct {
	BeforeGrade          *GradeRef   `json:"beforeGrade,omitempty"`
	AfterGrade           *GradeRef   `json:"afterGrade,omitempty"`
	BeforeDepartment     string      `json:"beforeDepartment,omitempty"`
	BeforeJob            string      `json:"beforeJob,omitempty"`
	BeforeDepartmentName string      `json:"beforeDepartmentName,omitempty"`
	BeforeJobName        string      `json:"beforeJobName,omitempty"`
	After                JobTransfer `json:"after"`
}

package hireprep

import f "chnhcm.local/chnhcm/internal/foundation"

// GradeRef records the personal adoption and the M03 job rule used at the time.
type GradeRef struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Source      string `json:"sourceVersion"`
	Revision    string `json:"revision"`
	Job         string `json:"job"`
	JobSource   string `json:"jobSource"`
	JobRevision string `json:"jobRevision"`
}
type ActionRef struct {
	Action   string `json:"action"`
	Revision string `json:"revision"`
}
type PromotionEvidence struct {
	Before         *GradeRef `json:"before,omitempty"`
	After          GradeRef  `json:"after"`
	Department     string    `json:"department"`
	DepartmentName string    `json:"departmentName"`
	Reason         string    `json:"reason"`
	Withdrawal     string    `json:"withdrawal,omitempty"`
}
type GradeSearch struct {
	BusinessDate string `json:"businessDate"`
	Job          string `json:"job,omitempty"`
	Search       string `json:"search,omitempty"`
	Page         int    `json:"page"`
}

// Precedes permits a later registration to identify an already recorded successor.
// Previous expresses documented order, independent of legal prerequisites.
type PromotionCommand struct {
	ExpectedSource   string     `json:"expectedSourceVersion"`
	ExpectedRevision string     `json:"expectedRevision,omitempty"`
	BusinessDate     string     `json:"businessDate,omitempty"`
	Grade            string     `json:"grade,omitempty"`
	Department       string     `json:"department,omitempty"`
	Job              string     `json:"job,omitempty"`
	Notice           string     `json:"notice,omitempty"`
	TransferNotice   string     `json:"transferNotice,omitempty"`
	Reason           string     `json:"reason,omitempty"`
	Previous         *ActionRef `json:"previous,omitempty"`
	Precedes         *ActionRef `json:"precedes,omitempty"`
}

func (r *ActionRef) Valid() bool {
	return r == nil || f.ValidID(r.Action) && f.ValidID(r.Revision)
}

// The caller validates references outside the assignment replay (such as the
// relation's exact confirmation). Only linked same-day assignment edits affect
// each other's before-images. Peers without a link have no implied order.
func OrderDay(rows []HistoryEntry) ([]HistoryEntry, error) {
	byID := make(map[string]HistoryEntry, len(rows))
	for _, r := range rows {
		byID[r.ID] = r
	}
	states := map[string]int{}
	ordered := make([]HistoryEntry, 0, len(rows))
	var visit func(HistoryEntry) error
	visit = func(r HistoryEntry) error {
		switch states[r.ID] {
		case 1:
			return historyConflict("同日办理先后不能成环")
		case 2:
			return nil
		}
		states[r.ID] = 1
		if r.Previous != nil {
			if p, ok := byID[r.Previous.Action]; ok {
				if err := visit(p); err != nil {
					return err
				}
			}
		}
		states[r.ID] = 2
		ordered = append(ordered, r)
		return nil
	}
	for _, r := range rows {
		if err := visit(r); err != nil {
			return nil, err
		}
	}
	return ordered, nil
}

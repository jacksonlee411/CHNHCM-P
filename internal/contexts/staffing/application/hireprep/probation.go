package hireprep

import f "chnhcm.local/chnhcm/internal/foundation"

type ProbationCommand struct {
	BusinessDate   string `json:"businessDate"`
	ExpectedSource string `json:"expectedSourceVersion"`
	Notice         string `json:"notice"`
	Department     string `json:"department,omitempty"`
	// An actual same-day transfer named in the notice as an earlier fact.
	AfterAction   string `json:"afterAction,omitempty"`
	AfterRevision string `json:"afterRevision,omitempty"`
}
type ProbationEvidence struct {
	Start          string `json:"start"`
	Department     string `json:"department,omitempty"`
	DepartmentName string `json:"departmentName,omitempty"`
	Attribution    string `json:"attribution"`
	AfterAction    string `json:"afterAction,omitempty"`
	AfterRevision  string `json:"afterRevision,omitempty"`
}
type ProbationState struct {
	Start     string `json:"start,omitempty"`
	Confirmed string `json:"confirmed,omitempty"`
	Status    string `json:"status"`
}

func (p ProbationState) At(day string) ProbationState {
	p.Status = "unrecorded"
	if p.Start != "" && p.Start <= day {
		p.Status = "probation"
	}
	if p.Confirmed != "" && p.Confirmed <= day {
		p.Status = "confirmed"
	}
	return p
}
func (p ProbationState) CanRecord(kind, day string, term f.Period) error {
	if !f.ValidDate(day) || !term.Includes(day) {
		return f.Fail(409, "PROBATION_PERIOD", "请选择实际工作关系内的业务日期")
	}
	if kind == "ProbationStart" {
		if p.Start != "" {
			return f.Fail(409, "PROBATION_EXISTS", "本工作关系已有实际试用开始日")
		}
		return nil
	}
	if kind != "ConfirmEmployment" || p.Start == "" || p.Start > day {
		return f.Fail(409, "PROBATION_REQUIRED", "请先核对本工作关系真实的试用开始日期")
	}
	if p.Confirmed != "" {
		return f.Fail(409, "ALREADY_CONFIRMED", "本工作关系已有转正办理")
	}
	return nil
}
func RelationAction(kind string) bool {
	return kind == "ProbationStart" || kind == "ConfirmEmployment" || kind == "Leave"
}

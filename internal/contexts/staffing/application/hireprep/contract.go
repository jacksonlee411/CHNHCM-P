package hireprep

import (
	f "chnhcm.local/chnhcm/internal/foundation"
	"strings"
	"unicode/utf8"
)

type ContractRef struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}
type ContractProcess struct {
	Version     string `json:"version"`
	Operation   string `json:"operation"`
	Day         string `json:"businessDate"`
	Operator    string `json:"operator"`
	RecordedAt  string `json:"recordedAt"`
	Basis       string `json:"basis"`
	LeaveAction string `json:"leaveAction,omitempty"`
}
type Contract struct {
	ContractRef
	Employment string `json:"employment"`
	Number     string `json:"number"`
	Signed     string `json:"signed"`
	f.Period
	Previous  string            `json:"previous,omitempty"`
	Next      string            `json:"next,omitempty"`
	End       string            `json:"firstInactive,omitempty"`
	Processes []ContractProcess `json:"processes"`
}
type ContractList struct {
	Required  []ContractRef `json:"required,omitempty"`
	CanManage bool          `json:"canManage"`
	Items     []Contract    `json:"items"`
}
type ContractCommand struct {
	Operation string       `json:"operation"`
	Number    string       `json:"number,omitempty"`
	Signed    string       `json:"signed,omitempty"`
	From      string       `json:"from,omitempty"`
	LastDay   string       `json:"lastDay,omitempty"`
	Ref       *ContractRef `json:"ref,omitempty"`
	End       string       `json:"firstInactive,omitempty"`
	Basis     string       `json:"basis,omitempty"`
	Expected  string       `json:"expectedSourceVersion"`
}

func (in ContractCommand) Validate() error {
	invalid := f.Fail(400, "CONTRACT_INPUT", "请核对合同编号、签订日、实际期限及所选原合同")
	if !f.ValidID(in.Expected) || in.Ref != nil && (!f.ValidID(in.Ref.ID) || !f.ValidID(in.Ref.Version)) {
		return invalid
	}
	if in.Operation == "end" {
		if in.Ref == nil || !f.ValidDate(in.End) || strings.TrimSpace(in.Basis) == "" || utf8.RuneCountInString(in.Basis) > 1000 || in.Signed != "" || in.Number != "" || in.From != "" || in.LastDay != "" {
			return invalid
		}
		return nil
	}
	if in.Operation != "sign" && in.Operation != "renew" || (in.Operation == "renew") != (in.Ref != nil) || !f.ValidDate(in.Signed) || !f.ValidDate(in.From) || in.LastDay != "" && (!f.ValidDate(in.LastDay) || in.LastDay < in.From) || in.End != "" || in.Basis != "" || strings.TrimSpace(in.Number) == "" || utf8.RuneCountInString(in.Number) > 160 {
		return invalid
	}
	return nil
}

type ContractReceipt struct {
	SubmissionID string `json:"submissionId"`
	Outcome      string `json:"outcome"`
	ID           string `json:"id,omitempty"`
	Version      string `json:"version,omitempty"`
}

func (c Contract) RemainsAt(day string) bool {
	return (c.End == "" || c.End > day) && (c.Until == "" || c.Until > day)
}

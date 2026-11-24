package public

// OperatorSnapshot is immutable organization evidence, projected from the X03 port.
type OperatorSnapshot struct {
	SubjectID        string  `json:"subjectId"`
	DisplayName      *string `json:"displayName"`
	Login            *string `json:"login"`
	Kind             string  `json:"kind"`
	IdentityEvidence string  `json:"identityEvidence"`
}

type OperationSummary struct {
	RecordID               string           `json:"recordId"`
	Revision               string           `json:"revision"`
	OperationSourceVersion string           `json:"operationSourceVersion"`
	RecordedAt             string           `json:"recordedAt"`
	Operator               OperatorSnapshot `json:"operator"`
}
type ChangeQuery struct {
	ID                                         string
	Page, PageSize                             int
	ContextSourceVersion, ThroughSourceVersion string
	Filter                                     *ColumnFilter
}
type ChangeRef struct {
	SourceVersion string `json:"sourceVersion"`
	RecordID      string `json:"recordId"`
	Revision      string `json:"revision"`
}
type ChangeRecordContext struct {
	EffectiveDate   string   `json:"effectiveDate,omitempty"`
	EndExclusive    **string `json:"endExclusive,omitempty"`
	EndInclusive    **string `json:"endInclusive,omitempty"`
	Name            *string  `json:"name,omitempty"`
	NameEn          **string `json:"nameEn,omitempty"`
	State           string   `json:"state"`
	OperationType   string   `json:"operationType,omitempty"`
	TypeLabel       string   `json:"typeLabel,omitempty"`
	OperationReason string   `json:"operationReason,omitempty"`
	ReasonLabel     string   `json:"reasonLabel,omitempty"`
}

// A missing record, an authorized empty value and an unreadable field are distinct.
// Unreadable fields never produce a ChangeDiff.
type ChangeValue struct {
	Kind         string   `json:"kind"`
	Text         *string  `json:"text,omitempty"`
	Start        string   `json:"start,omitempty"`
	EndExclusive **string `json:"endExclusive,omitempty"`
	EndInclusive **string `json:"endInclusive,omitempty"`
	Code         string   `json:"code,omitempty"`
	Label        string   `json:"label,omitempty"`
}
type ChangeDiff struct {
	Field  string      `json:"field"`
	Before ChangeValue `json:"before"`
	After  ChangeValue `json:"after"`
}
type ChangeItem struct {
	RecordID      string               `json:"recordId"`
	ChangeKind    string               `json:"changeKind"`
	BeforeRef     *ChangeRef           `json:"beforeRef"`
	AfterRef      *ChangeRef           `json:"afterRef"`
	BeforeContext *ChangeRecordContext `json:"beforeContext"`
	AfterContext  *ChangeRecordContext `json:"afterContext"`
	Diffs         []ChangeDiff         `json:"diffs"`
}
type Change struct {
	SourceVersion         string           `json:"sourceVersion"`
	PreviousSourceVersion *string          `json:"previousSourceVersion"`
	RecordedAt            string           `json:"recordedAt"`
	Operator              OperatorSnapshot `json:"operator"`
	Action                string           `json:"action"`
	Items                 []ChangeItem     `json:"items"`
}
type Changes struct {
	ReadFields           []string `json:"readFields"`
	DetailReadFields     []string `json:"detailReadFields"`
	Items                []Change `json:"items"`
	Total                int      `json:"total"`
	Page                 int      `json:"page"`
	PageSize             int      `json:"pageSize"`
	ContextSourceVersion *string  `json:"contextSourceVersion"`
	ThroughSourceVersion string   `json:"throughSourceVersion"`
}

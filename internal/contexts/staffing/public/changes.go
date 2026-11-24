package public

// LogQuery pins a log traversal to an immutable source, independently of time.
type LogQuery struct {
	Page, PageSize                             int
	FilterField, FilterOperator, FilterValue   string
	ContextSourceVersion, ThroughSourceVersion string
}
type ChangeOperator struct {
	SubjectID        string  `json:"subjectId"`
	DisplayName      *string `json:"displayName"`
	Login            *string `json:"login"`
	Kind             string  `json:"kind"`
	IdentityEvidence string  `json:"identityEvidence"`
}
type ChangeValue struct {
	Members      []GradeMemberView `json:"members,omitempty"`
	Kind         string            `json:"kind"`
	Text         string            `json:"text,omitempty"`
	Label        string            `json:"label,omitempty"`
	Code         string            `json:"code,omitempty"`
	Start        string            `json:"start,omitempty"`
	EndInclusive string            `json:"endInclusive,omitempty"`
}
type ChangeDiff struct {
	Field  string      `json:"field"`
	Before ChangeValue `json:"before"`
	After  ChangeValue `json:"after"`
}
type ChangeRef struct {
	SourceVersion string `json:"sourceVersion"`
	RecordID      string `json:"recordId"`
	Revision      string `json:"revision"`
}
type ChangeItem struct {
	RecordID      string       `json:"recordId"`
	ChangeKind    string       `json:"changeKind"`
	BeforeRef     *ChangeRef   `json:"beforeRef"`
	AfterRef      *ChangeRef   `json:"afterRef"`
	BeforeContext View         `json:"beforeContext"`
	AfterContext  View         `json:"afterContext"`
	Diffs         []ChangeDiff `json:"diffs"`
}
type ChangeGroup struct {
	SourceVersion         string         `json:"sourceVersion"`
	PreviousSourceVersion *string        `json:"previousSourceVersion"`
	RecordedAt            string         `json:"recordedAt,omitempty"`
	Operator              ChangeOperator `json:"operator,omitzero"`
	Action                string         `json:"action"`
	Items                 []ChangeItem   `json:"items"`
}
type ChangeLog struct {
	Items                []ChangeGroup `json:"items"`
	Total                int           `json:"total"`
	Page                 int           `json:"page"`
	PageSize             int           `json:"pageSize"`
	ContextSourceVersion *string       `json:"contextSourceVersion"`
	ThroughSourceVersion string        `json:"throughSourceVersion"`
	ReadFields           []string      `json:"readFields"`
	DetailReadFields     []string      `json:"detailReadFields"`
}

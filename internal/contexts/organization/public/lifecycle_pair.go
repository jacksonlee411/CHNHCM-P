package public

type InsertLifecyclePairInput struct {
	ExpectedSourceVersion string `json:"expectedSourceVersion"`
	AnchorRecordID        string `json:"anchorRecordId"`
	DeactivationDate      string `json:"deactivationDate"`
	DeactivationReason    string `json:"deactivationReason"`
	ReactivationDate      string `json:"reactivationDate"`
	ReactivationReason    string `json:"reactivationReason"`
}
type DeleteLifecyclePairInput struct {
	ExpectedSourceVersion string `json:"expectedSourceVersion"`
	DeactivationRecordID  string `json:"deactivationRecordId"`
	DeactivationRevision  string `json:"deactivationRevision"`
	ReactivationRecordID  string `json:"reactivationRecordId"`
	ReactivationRevision  string `json:"reactivationRevision"`
}
type PairInsertionContext struct {
	AnchorRecordID    string `json:"anchorRecordId"`
	Available         bool   `json:"available"`
	MinDate           string `json:"minDate,omitempty"`
	MaxStopDate       string `json:"maxStopDate,omitempty"`
	MaxDate           string `json:"maxDate,omitempty"`
	DefaultStopDate   string `json:"defaultStopDate,omitempty"`
	DefaultResumeDate string `json:"defaultResumeDate,omitempty"`
}
type PairNameChange struct {
	Start        string  `json:"start,omitempty"`
	End          *string `json:"endExclusive"`
	EndInclusive *string `json:"endInclusive"`
	BeforeName   *string `json:"beforeName,omitempty"`
	BeforeNameEn *string `json:"beforeNameEn,omitempty"`
	AfterName    *string `json:"afterName,omitempty"`
	AfterNameEn  *string `json:"afterNameEn,omitempty"`
}
type PairDeletionContext struct {
	Deactivation Record           `json:"deactivation"`
	Reactivation Record           `json:"reactivation"`
	NameChanges  []PairNameChange `json:"nameChanges"`
}

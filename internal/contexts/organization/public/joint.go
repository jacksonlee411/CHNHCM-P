package public

type Classification struct {
	OperationType   string `json:"operationType,omitempty"`
	OperationReason string `json:"operationReason,omitempty"`
}

// RequestItem holds only protocol location and declared authority, never names.
type RequestItem struct {
	ItemID                 string   `json:"itemId"`
	Action                 string   `json:"action"`
	RecordID               string   `json:"recordId,omitempty"`
	ExpectedRecordRevision string   `json:"expectedRecordRevision,omitempty"`
	EffectiveDate          string   `json:"effectiveDate,omitempty"`
	ProcessingFacets       []string `json:"processingFacets"`
	DeclaredFields         []string `json:"declaredFields"`
	OriginalType           string   `json:"originalType,omitempty"`
	NewType                string   `json:"newType,omitempty"`
}
type ItemResult struct {
	DateDescription  string   `json:"dateDescription,omitempty"`
	ItemID           string   `json:"itemId"`
	Outcome          string   `json:"outcome"`
	RecordID         string   `json:"recordId,omitempty"`
	BeforeRevisionID *string  `json:"beforeRevisionId"`
	AfterRevisionID  *string  `json:"afterRevisionId"`
	EffectiveDate    string   `json:"effectiveDate,omitempty"`
	ProcessingFacets []string `json:"processingFacets"`
	BeforeRecord     *Record  `json:"beforeRecord,omitempty"`
	AfterRecord      *Record  `json:"afterRecord,omitempty"`
}

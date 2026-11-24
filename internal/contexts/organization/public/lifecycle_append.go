package public

type LifecycleContext struct {
	AnchorRecordID       string            `json:"anchorRecordId"`
	OperationType        string            `json:"operationType"`
	MinEffectiveDate     string            `json:"minEffectiveDate,omitempty"`
	MaxEffectiveDate     string            `json:"maxEffectiveDate,omitempty"`
	DefaultEffectiveDate string            `json:"defaultEffectiveDate,omitempty"`
	Defaults             *AdditionDefaults `json:"defaults"`
}

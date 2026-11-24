package public

type AdditionDefaults struct {
	Name   *string `json:"name,omitempty"`
	NameEn *string `json:"nameEn,omitempty"`
}
type AdditionContext struct {
	AnchorRecordID       string            `json:"anchorRecordId"`
	Mode                 string            `json:"mode"`
	NextRecordID         string            `json:"nextRecordId,omitempty"`
	Available            bool              `json:"available"`
	MinEffectiveDate     string            `json:"minEffectiveDate,omitempty"`
	MaxEffectiveDate     string            `json:"maxEffectiveDate,omitempty"`
	DefaultEffectiveDate string            `json:"defaultEffectiveDate,omitempty"`
	Defaults             *AdditionDefaults `json:"defaults,omitempty"`
}

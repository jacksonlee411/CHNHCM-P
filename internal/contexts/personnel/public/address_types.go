package public

import "time"

type AddressTypeChoice struct {
	ID       string `json:"id"`
	Revision string `json:"revision"`
}
type AddressTypeAdoption struct {
	AddressTypeChoice
	Date string `json:"date"`
}
type AddressTypeOption struct {
	AddressTypeChoice
	Code string `json:"code"`
	Name string `json:"name"`
}
type AddressType struct {
	AddressTypeOption
	Order      int    `json:"sortOrder"`
	FirstDay   string `json:"firstDay"`
	Selectable bool   `json:"selectable"`
}
type AddressTypeList struct {
	Items           []AddressType `json:"items"`
	CanMaintain     bool          `json:"canMaintain"`
	DefaultFirstDay string        `json:"defaultFirstDay"`
}
type AddressTypeCommand struct {
	SubmissionID     string `json:"submissionId"`
	ID               string `json:"id,omitempty"`
	ExpectedRevision string `json:"expectedRevision,omitempty"`
	Code             string `json:"code,omitempty"`
	Name             string `json:"name"`
	Order            int    `json:"sortOrder"`
	FirstDay         string `json:"firstDay,omitempty"`
	Selectable       *bool  `json:"selectable"`
}

type AddressTypeReceipt struct {
	Outcome string       `json:"outcome"`
	Item    *AddressType `json:"item,omitempty"`
}

type AddressTypeSource struct {
	AddressType
	ActorID    *string   `json:"actorId,omitempty"`
	RecordedAt time.Time `json:"recordedAt"`
}

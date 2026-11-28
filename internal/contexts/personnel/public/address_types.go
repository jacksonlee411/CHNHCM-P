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
	Order         int    `json:"sortOrder"`
	FirstDay      string `json:"firstDay"`
	Selectable    bool   `json:"selectable"`
	StopDay       string `json:"stopDay,omitempty"`
	ConfirmedFrom string `json:"confirmedFrom,omitempty"`
	UndatedStop   bool   `json:"undatedStop,omitempty"`
}
type AddressTypeView struct {
	AddressType
	State string `json:"state"`
}
type AddressTypePoint struct {
	Day   string `json:"day"`
	State string `json:"state"`
}
type AddressTypeDetail struct {
	Item           AddressTypeView    `json:"item"`
	Timeline       []AddressTypePoint `json:"timeline"`
	SelectedDay    string             `json:"selectedDay"`
	BusinessDay    string             `json:"businessDay"`
	DefaultStopDay string             `json:"defaultStopDay"`
	CanMaintain    bool               `json:"canMaintain"`
}
type AddressTypeList struct {
	Items           []AddressTypeView `json:"items"`
	CanMaintain     bool              `json:"canMaintain"`
	DefaultFirstDay string            `json:"defaultFirstDay"`
}
type AddressTypeCommand struct {
	SubmissionID     string  `json:"submissionId"`
	ID               string  `json:"id,omitempty"`
	ExpectedRevision string  `json:"expectedRevision,omitempty"`
	Code             string  `json:"code,omitempty"`
	Name             string  `json:"name"`
	Order            int     `json:"sortOrder"`
	FirstDay         string  `json:"firstDay,omitempty"`
	Selectable       *bool   `json:"selectable,omitempty"`
	StopDay          *string `json:"stopDay,omitempty"`
}

type AddressTypeReceipt struct {
	Outcome string       `json:"outcome"`
	Item    *AddressType `json:"item,omitempty"`
}

type AddressTypeSource struct {
	AddressType
	Dated            bool      `json:"dated"`
	PreviousRevision string    `json:"previousRevision,omitempty"`
	ActorID          *string   `json:"actorId,omitempty"`
	RecordedAt       time.Time `json:"recordedAt"`
}

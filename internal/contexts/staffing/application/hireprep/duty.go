package hireprep

import f "chnhcm.local/chnhcm/internal/foundation"

type DutyRef struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}
type DutyObservation struct {
	Dispatch        DecisionRef `json:"dispatch"`
	Issuer          string      `json:"issuer"`
	Number          string      `json:"number"`
	From            string      `json:"from"`
	LastDay         string      `json:"lastDay,omitempty"`
	ObservedThrough string      `json:"observedThrough"`
	Previous        *DutyRef    `json:"previous,omitempty"`
}
type DutyEvidence struct {
	DutyRef
	f.Period
	Dispatch        DecisionRef `json:"dispatch"`
	DispatchNumber  string      `json:"dispatchNumber"`
	Number          string      `json:"number"`
	Issuer          string      `json:"issuer"`
	ObservedThrough string      `json:"observedThrough"`
	Position        string      `json:"position"`
	PositionName    string      `json:"positionName"`
}
type DutySources struct {
	Decisions    []DecisionOption `json:"decisions"`
	Observations []DutyEvidence   `json:"observations"`
}
type DutyCommand struct {
	Expected    string           `json:"expectedSourceVersion"`
	Action      *ActionRef       `json:"action,omitempty"`
	Observation *DutyRef         `json:"observation,omitempty"`
	Paper       *DutyObservation `json:"paper,omitempty"`
}

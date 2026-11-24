package hireprep

type OfferHireCommand struct {
	Expected   *Adoption `json:"expected,omitempty"`
	Version    string    `json:"version"`
	Person     string    `json:"person"`
	Employer   string    `json:"employer"`
	Arrival    string    `json:"arrival"`
	LastDay    string    `json:"lastDay,omitempty"`
	Department string    `json:"department"`
	Job        string    `json:"job"`
	Position   string    `json:"position,omitempty"`
}
type OfferHireState struct {
	Missing    []string          `json:"missing,omitempty"`
	CanHire    bool              `json:"canHire"`
	Binding    *OfferHireCommand `json:"binding,omitempty"`
	Result     *ActionRef        `json:"result,omitempty"`
	Assignment string            `json:"assignment,omitempty"`
}

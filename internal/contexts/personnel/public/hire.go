package public

// HireIdentityRequest is supplied by the authenticated consumer of the customer rule.
type HireIdentityRequest struct {
	PersonID, Purpose, Relationship, Arrival string
	AcceptedDocuments                        []string
	RequireKnownExpiry, RequireInspection    bool
	Expected                                 *HireIdentitySource
}

type HireIdentitySource struct {
	PersonID           string `json:"personId"`
	PersonVersion      string `json:"personVersion"`
	DocumentID         string `json:"documentId"`
	DocumentRevision   string `json:"documentRevision"`
	InspectionRevision string `json:"inspectionRevision,omitempty"`
}

type HireIdentityResult struct {
	Source  HireIdentitySource
	Missing []string
}

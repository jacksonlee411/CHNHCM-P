package dated

// ItemFault carries a stable protocol item identity, never an array index.
type ItemFault struct {
	ItemID string
	Err    error
}

func (e *ItemFault) Error() string { return e.Err.Error() }
func (e *ItemFault) Unwrap() error { return e.Err }
func CandidateID[P any](it Item[P]) string {
	if it.Action == "ADD" && it.RecordID != "" {
		return it.RecordID
	}
	return it.ID
}

func itemForRecord[P any](items []Item[P], id string) string {
	for _, it := range items {
		if it.RecordID == id || it.Action == "ADD" && CandidateID(it) == id {
			return it.ID
		}
	}
	return ""
}

package foundation

// CatalogSource identifies a dated source observed inside a consumer's transaction.
type CatalogSource struct {
	ID, Source, Code string
	Segments         []CatalogSegment
}

type CatalogSegment struct {
	Period
	RecordID, Revision, Name string
	Active                   bool
}

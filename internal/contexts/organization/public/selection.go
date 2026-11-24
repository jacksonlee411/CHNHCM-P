package public

type SelectionQuery struct {
	Q              string `json:"q,omitempty"`
	Status         string `json:"status,omitempty"`
	FilterField    string `json:"filterField,omitempty"`
	FilterOperator string `json:"filterOperator,omitempty"`
	FilterValue    string `json:"filterValue,omitempty"`
}
type Selection struct {
	Mode        string    `json:"mode"`
	IDs         *[]string `json:"ids,omitempty"`
	ExcludedIDs *[]string `json:"excludedIds,omitempty"`
}
type SelectionRequest struct {
	Query                    SelectionQuery `json:"query"`
	Selection                Selection      `json:"selection"`
	ExpectedSelectionVersion string         `json:"expectedSelectionVersion"`
}
type ExportRequest struct {
	Query                    SelectionQuery `json:"query"`
	Selection                Selection      `json:"selection"`
	ExpectedSelectionVersion string         `json:"expectedSelectionVersion"`
	Sort                     string         `json:"sort,omitempty"`
}
type SelectionResult struct {
	QueryKey         string `json:"queryKey"`
	SelectionVersion string `json:"selectionVersion"`
	TotalCount       int    `json:"totalCount"`
	SelectedCount    int    `json:"selectedCount"`
	CanExport        bool   `json:"canExport"`
}
type Export struct {
	Data       []byte
	Count      int
	BusinessAt string
}

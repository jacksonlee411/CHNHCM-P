package public

import "encoding/json"

// Tree input carries presentation position only; business day and rights belong
// to the authenticated Go request.
type DepartmentTreeInput struct {
	ParentID            *string
	Q, Cursor, AroundID string
	PageSize            int
}
type DepartmentTreeNode struct {
	ID          string  `json:"id"`
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	ParentID    *string `json:"parentId"`
	Kind        string  `json:"kind"`
	HasChildren *bool   `json:"hasChildren,omitempty"`
}
type DepartmentTreePage struct {
	Items          []DepartmentTreeNode `json:"items"`
	BusinessAt     string               `json:"businessAt"`
	HasMore        bool                 `json:"hasMore"`
	NextCursor     string               `json:"nextCursor,omitempty"`
	HasPrevious    bool                 `json:"hasPrevious"`
	PreviousCursor string               `json:"previousCursor,omitempty"`
}
type DepartmentTreeLocation struct {
	Items      []DepartmentTreeNode `json:"items"`
	BusinessAt string               `json:"businessAt"`
}

type DepartmentSelectionQuery struct {
	Q                  string `json:"q,omitempty"`
	Status             string `json:"status,omitempty"`
	FilterField        string `json:"filterField,omitempty"`
	FilterOperator     string `json:"filterOperator,omitempty"`
	FilterValue        string `json:"filterValue,omitempty"`
	ScopeDepartmentID  string `json:"scopeDepartmentId,omitempty"`
	IncludeDescendants *bool  `json:"includeDescendants,omitempty"`
}

// The absent current page is explicit; it is never serialized as an empty list.
func (p DepartmentSelectionPage) MarshalJSON() ([]byte, error) {
	if p.PageState == "scope_unavailable" {
		return json.Marshal(struct {
			PageState  string                    `json:"pageState"`
			BusinessAt string                    `json:"businessAt"`
			Summary    DepartmentSelectionResult `json:"summary"`
		}{p.PageState, p.BusinessAt, p.Summary})
	}
	type plain DepartmentSelectionPage
	return json.Marshal(plain(p))
}

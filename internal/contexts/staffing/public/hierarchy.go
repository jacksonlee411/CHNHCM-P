package public

import "time"

const Hierarchy Kind = "positionhierarchy"

type PublishHierarchy struct {
	ExpectedVersion string  `json:"expectedVersion"`
	EffectiveDate   string  `json:"effectiveDate"`
	ChildID         string  `json:"childId"`
	ParentID        *string `json:"parentId"`
}
type HierarchyVersion struct {
	SourceVersion string    `json:"sourceVersion"`
	Number        int       `json:"number"`
	RecordedAt    time.Time `json:"recordedAt"`
}
type HierarchyNode struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Code     string `json:"code,omitempty"`
	ParentID string `json:"parentId,omitempty"`
}
type HierarchyView struct {
	BusinessAt    string             `json:"businessAt"`
	HierarchyID   string             `json:"hierarchyId,omitempty"`
	SourceVersion string             `json:"sourceVersion"`
	Versions      []HierarchyVersion `json:"versions"`
	Nodes         []HierarchyNode    `json:"nodes"`
	CanPublish    bool               `json:"canPublish"`
}

// StructureQuery selects one bounded branch, search page, path, or relation.
type StructureQuery struct {
	Mode, BusinessAt, SourceVersion, NodeID, Q string
	Page, PageSize                             int
}
type StructureNode struct {
	SiblingPage    int    `json:"siblingPage,omitempty"`
	ID             string `json:"id"`
	Name           string `json:"name"`
	Code           string `json:"code,omitempty"`
	ParentID       string `json:"parentId,omitempty"`
	HasChildren    bool   `json:"hasChildren"`
	PositionSource string `json:"positionSource"`
	RecordID       string `json:"recordId"`
}
type StructureView struct {
	BusinessAt     string             `json:"businessAt"`
	SourceVersion  string             `json:"sourceVersion"`
	CanPublish     bool               `json:"canPublish"`
	CanReadHistory bool               `json:"canReadHistory"`
	Nodes          []StructureNode    `json:"nodes"`
	Versions       []HierarchyVersion `json:"versions,omitempty"`
	More           bool               `json:"more"`
	Selected       *StructureNode     `json:"selected,omitempty"`
	Parent         *StructureNode     `json:"parent,omitempty"`
	EndInclusive   string             `json:"endInclusive,omitempty"`
	NextStart      string             `json:"nextStart,omitempty"`
}

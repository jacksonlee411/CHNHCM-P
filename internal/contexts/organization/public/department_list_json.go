package public

import "encoding/json"

// Count remains exact for ordinary lists, including zero. Path pages expose
// only cursor continuation; an internal negative sentinel is never serialized.
func (v DepartmentList) MarshalJSON() ([]byte, error) {
	type plain DepartmentList
	var count *int
	var more *bool
	if v.Count < 0 {
		more = &v.HasMore
	}
	if v.Count >= 0 {
		count = &v.Count
	}
	return json.Marshal(struct {
		plain
		Count   *int  `json:"count,omitempty"`
		HasMore *bool `json:"hasMore,omitempty"`
	}{plain(v), count, more})
}

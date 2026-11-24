package public

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Arrays remain arrays at the transport boundary. Scalar changes keep their
// existing shape; each collection has explicit presence and replacement intent.
func (in *Input) UnmarshalJSON(data []byte) error {
	type scalar Input
	*in = Input{}
	wire := struct {
		*scalar
		JobFunctionID json.RawMessage            `json:"jobFunctionId"`
		GradeIDs      json.RawMessage            `json:"gradeIds"`
		Changes       map[string]json.RawMessage `json:"changes"`
	}{scalar: (*scalar)(in)}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&wire); err != nil {
		return err
	}
	if wire.JobFunctionID != nil {
		in.JobFunctionPresent = true
		if err := json.Unmarshal(wire.JobFunctionID, &in.JobFunctionID); err != nil {
			return err
		}
	}
	parse := func(raw json.RawMessage) (*[]string, error) {
		if raw == nil {
			return nil, nil
		}
		var ids []string
		if err := json.Unmarshal(raw, &ids); err != nil {
			return nil, err
		}
		if ids == nil {
			return nil, fmt.Errorf("gradeIds不能为null")
		}
		return &ids, nil
	}
	var err error
	if in.GradeIDs, err = parse(wire.GradeIDs); err != nil {
		return err
	}
	if wire.Changes != nil {
		in.Changes = map[string]*string{}
	}
	for field, raw := range wire.Changes {
		if field == "gradeIds" {
			if in.ChangedGradeIDs, err = parse(raw); err != nil {
				return err
			}
			continue
		}
		var value *string
		if err = json.Unmarshal(raw, &value); err != nil {
			return err
		}
		in.Changes[field] = value
	}
	return nil
}
func (in Input) MarshalJSON() ([]byte, error) {
	type scalar Input
	changes := map[string]any{}
	for k, v := range in.Changes {
		changes[k] = v
	}
	if in.ChangedGradeIDs != nil {
		changes["gradeIds"] = *in.ChangedGradeIDs
	}
	return json.Marshal(struct {
		*scalar
		Changes map[string]any `json:"changes,omitempty"`
	}{scalar: (*scalar)(&in), Changes: changes})
}

type GradeAdoptedRef struct {
	BusinessAt    string `json:"businessAt"`
	SourceVersion string `json:"sourceVersion"`
	RecordID      string `json:"recordId"`
	Revision      string `json:"revision"`
}
type GradeMemberView struct {
	GradeID     string           `json:"gradeId"`
	Name        string           `json:"name"`
	Code        *string          `json:"code,omitempty"`
	Position    *int             `json:"position,omitempty"`
	DisplayDate string           `json:"displayDate"`
	Adopted     *GradeAdoptedRef `json:"adopted,omitempty"`
}

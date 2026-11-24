package public

import (
	"bytes"
	"encoding/json"
	"slices"

	"chnhcm.local/chnhcm/internal/foundation"
)

// CorrectInput is one current single-record format with independently authorized facets.
func (v *CorrectInput) UnmarshalJSON(raw []byte) error {
	type plain CorrectInput
	var p plain
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&p); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	for _, value := range fields {
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return foundation.Fail(400, "M02_INVALID_INPUT", "修改内容不完整，请重新打开表单")
		}
	}
	if _, present := fields["newEffectiveDate"]; present && !foundation.ValidDate(p.NewEffectiveDate) {
		return &foundation.Fault{Status: 400, Code: "M02_INVALID_INPUT", Field: "newEffectiveDate", Message: "新日期须为有效日历日期"}
	}
	*v = CorrectInput(p)
	return nil
}

func (v CorrectInput) Validate() error {
	field, message := "", ""
	switch {
	case v.Changes != nil && len(v.Changes) == 0:
		field, message = "changes", "请选择名称修改内容"
	case v.NewEffectiveDate != "" && !foundation.ValidDate(v.NewEffectiveDate):
		field, message = "newEffectiveDate", "新日期须为有效日历日期"
	case v.Classification != nil && v.Classification.OperationType == "":
		field, message = "classification.operationType", "请选择操作类型"
	case v.Classification != nil && v.Classification.OperationReason == "":
		field, message = "classification.operationReason", "请选择操作原因"
	case v.Changes == nil && v.NewEffectiveDate == "" && v.Classification == nil:
		field, message = "changes", "没有修改内容"
	}
	if message != "" {
		return &foundation.Fault{Status: 400, Code: "M02_INVALID_INPUT", Field: field, Message: message}
	}
	return nil
}

func CorrectionFields(v CorrectInput) []string {
	fields := []string{}
	for f := range v.Changes {
		fields = append(fields, f)
	}
	if v.NewEffectiveDate != "" {
		fields = append(fields, "effectiveDate")
	}
	if v.Classification != nil {
		fields = append(fields, "operationType", "operationReason")
	}
	slices.Sort(fields)
	return slices.Compact(fields)
}

type CorrectionCapabilities struct {
	Facets              []string `json:"facets"`
	ClassificationTypes []string `json:"classificationTypes"`
	MinEffectiveDate    string   `json:"minEffectiveDate,omitempty"`
	MaxEffectiveDate    string   `json:"maxEffectiveDate,omitempty"`
}

func (v *EnterpriseCorrectInput) UnmarshalJSON(raw []byte) error {
	type plain EnterpriseCorrectInput
	var p plain
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&p); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	for _, value := range fields {
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return foundation.Fail(400, "M02_INVALID_INPUT", "修改内容不完整，请重新打开表单")
		}
	}
	if _, present := fields["newEffectiveDate"]; present && !foundation.ValidDate(p.NewEffectiveDate) {
		return &foundation.Fault{Status: 400, Code: "M02_INVALID_INPUT", Field: "newEffectiveDate", Message: "新日期须为有效日历日期"}
	}
	*v = EnterpriseCorrectInput(p)
	return nil
}

func (v *EnterprisePreviewInput) UnmarshalJSON(raw []byte) error {
	type plain EnterprisePreviewInput
	var p plain
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&p); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	for _, value := range fields {
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return foundation.Fail(400, "M02_INVALID_INPUT", "修改内容不完整，请重新打开表单")
		}
	}
	if _, present := fields["newEffectiveDate"]; present && !foundation.ValidDate(p.NewEffectiveDate) {
		return &foundation.Fault{Status: 400, Code: "M02_INVALID_INPUT", Field: "newEffectiveDate", Message: "新日期须为有效日历日期"}
	}
	*v = EnterprisePreviewInput(p)
	return nil
}

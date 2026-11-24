package location

import (
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"slices"
)

var dataFields = []string{"name", "nameEn", "countryRegionId", "geoAreaId", "addressLine", "postalCode"}
var dateFields = []string{"effectiveDate", "operationType", "operationReason"}

func expected(c org.LocationCommand) string {
	switch c.Action {
	case "CORRECT":
		return c.Correct.ExpectedSourceVersion
	case "DELETE":
		return c.Delete.ExpectedSourceVersion
	case "REDATE":
		return c.Redate.ExpectedSourceVersion
	case "INSERT_LIFECYCLE_PAIR":
		return c.InsertPair.ExpectedSourceVersion
	case "DELETE_LIFECYCLE_PAIR":
		return c.DeletePair.ExpectedSourceVersion
	case "MAINTAIN_LIFECYCLE_PAIR":
		return c.MaintainPair.ExpectedSourceVersion
	default:
		return c.Add.ExpectedSourceVersion
	}
}
func metadata(c org.LocationCommand) (Terminal, []string, error) {
	t := Terminal{ID: c.SubmissionID, ObjectID: c.LocationID, Action: c.Action, Digest: c.Digest}
	fields := []string{}
	revision := ""
	var patch org.LocationChangesInput
	if !foundation.ValidID(t.ID) || len(t.Digest) != 32 {
		return t, nil, fault("M02_INVALID_INPUT", "", "提交身份或摘要无效")
	}
	switch c.Action {
	case "CREATE":
		t.ObjectID = foundation.NewID()
		fields = []string{"code", "name", "countryRegionId", "addressLine", "effectiveDate", "operationType", "operationReason"}
		for _, v := range []struct {
			key   string
			value *string
		}{{"nameEn", c.Create.NameEn}, {"geoAreaId", c.Create.GeoAreaID}, {"postalCode", c.Create.PostalCode}} {
			if v.value != nil {
				fields = append(fields, v.key)
			}
		}
		if c.Create.OperationType != "CREATE" {
			return t, nil, fault("M02_INVALID_INPUT", "operationType", "新建操作类型无效")
		}
	case "ADD", "DEACTIVATE", "REACTIVATE":
		patch = c.Add.Changes
		fields = slices.Clone(dateFields)
		kind := c.Action
		if kind == "ADD" {
			kind = "CHANGE"
		}
		if c.Add.OperationType != kind || !foundation.ValidID(c.Add.AnchorRecordID) || c.Action == "DEACTIVATE" && patch != nil {
			return t, nil, fault("M02_INVALID_INPUT", "", "操作类型或位置无效")
		}
	case "CORRECT":
		revision = c.Correct.ExpectedRecordRevision
		patch = c.Correct.Changes
		if c.Correct.NewEffectiveDate != "" {
			fields = append(fields, "effectiveDate")
		}
		if c.Correct.Classification != nil {
			fields = append(fields, "operationType", "operationReason")
		}
		if patch == nil && len(fields) == 0 {
			return t, nil, fault("M02_INVALID_INPUT", "", "没有修改内容")
		}
	case "DELETE":
		revision = c.Delete.ExpectedRecordRevision
	case "REDATE":
		revision = c.Redate.ExpectedRecordRevision
		fields = []string{"effectiveDate"}
	case "INSERT_LIFECYCLE_PAIR":
		fields = slices.Clone(dateFields)
	case "DELETE_LIFECYCLE_PAIR":
	case "MAINTAIN_LIFECYCLE_PAIR":
		fields = slices.Clone(dateFields)
		patch = c.MaintainPair.Changes
	default:
		return t, nil, fault("M02_INVALID_INPUT", "action", "操作无效")
	}
	if c.Action != "CREATE" && (!foundation.ValidID(t.ObjectID) || !foundation.ValidID(expected(c))) {
		return t, nil, fault("M02_INVALID_INPUT", "", "地点或来源身份无效")
	}
	if slices.Contains([]string{"CORRECT", "DELETE", "REDATE"}, c.Action) && (!foundation.ValidID(c.RecordID) || !foundation.ValidID(revision)) {
		return t, nil, fault("M02_INVALID_INPUT", "", "记录或修订身份无效")
	}
	if patch != nil && len(patch) == 0 && c.Action != "ADD" {
		return t, nil, fault("M02_INVALID_INPUT", "changes", "没有修改内容")
	}
	for key, value := range patch {
		if !slices.Contains(dataFields, key) {
			return t, nil, fault("M02_FORBIDDEN_INPUT", "changes."+key, "字段不可修改")
		}
		if value == nil && slices.Contains([]string{"name", "countryRegionId", "addressLine"}, key) {
			return t, nil, fault("M02_INVALID_INPUT", "changes."+key, "必填字段不能为空")
		}
		fields = append(fields, key)
	}
	slices.Sort(fields)
	return t, slices.Compact(fields), nil
}

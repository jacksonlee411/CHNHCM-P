package country

import (
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/country"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"slices"
)

func fault(code, field, message string) error {
	status := 409
	if code == "M02_INVALID_INPUT" || code == "M02_FORBIDDEN_INPUT" {
		status = 400
	}
	return &foundation.Fault{Status: status, Code: code, Field: field, Message: message}
}
func origin(v org.CountryOrigin) d.Origin {
	return d.Origin{Scheme: v.Scheme, Key: v.Key, Code: v.Code, Name: v.Name, NameEn: v.NameEn, PackageVersion: v.PackageVersion, PackageDigest: v.PackageDigest, ArchiveDigest: v.ArchiveDigest}
}
func candidate(ctx context.Context, t d.Timeline, cmd org.CountryCommand) (d.Candidate, error) {
	if cmd.Action == "CREATE" {
		if e := d.ValidateOrigin(origin(cmd.Import.Origin)); e != nil {
			return d.Candidate{}, e
		}
		return d.Kernel.Create(ctx, t.First, cmd.Import.EffectiveDate, d.MinDate, "SOURCE_INITIALIZATION", d.Data{Name: cmd.Import.Name, NameEn: cmd.Import.NameEn, PhoneCode: cmd.Import.PhoneCode})
	}
	item := d.Item{ID: "direct", RecordID: cmd.RecordID}
	switch cmd.Action {
	case "CORRECT":
		patch := d.Patch(cmd.Correct.Changes)
		item.Action = "CORRECT"
		item.ExpectedRevision = cmd.Correct.ExpectedRecordRevision
		item.Changes = &patch
	case "ADD", "DEACTIVATE", "REACTIVATE":
		a := cmd.Add
		item.Action = "ADD"
		item.Date = a.EffectiveDate
		item.Type = a.OperationType
		item.Reason = a.OperationReason
		if len(t.Records) == 0 || t.Records[len(t.Records)-1].ID != a.AnchorRecordID {
			return d.Candidate{}, fault("M02_INVALID_INPUT", "anchorRecordId", "目录普通维护只允许在末条之后追加")
		}
		if a.Changes != nil {
			p := d.Patch(a.Changes)
			item.Changes = &p
		}
		if cmd.Action == "ADD" {
			if a.OperationType != "CHANGE" {
				return d.Candidate{}, fault("M02_INVALID_INPUT", "operationType", "资料变更类型无效")
			}
			if e := d.Kernel.ValidateAdditionPosition(t.Records, a.AnchorRecordID, d.MinDate, a.EffectiveDate); e != nil {
				return d.Candidate{}, e
			}
		} else {
			if a.OperationType != cmd.Action {
				return d.Candidate{}, fault("M02_INVALID_INPUT", "operationType", "选用状态操作类型无效")
			}
			if e := d.Kernel.ValidateLifecycleAppend(t.Records, a.AnchorRecordID, cmd.Action, d.MinDate, a.EffectiveDate); e != nil {
				return d.Candidate{}, e
			}
		}
	default:
		return d.Candidate{}, fault("M02_FORBIDDEN_INPUT", "action", "目录未开放此操作")
	}
	return d.Kernel.Calculate(ctx, t.Records, t.First, d.MinDate, []d.Item{item})
}
func metadata(cmd org.CountryCommand) (Terminal, []string, error) {
	t := Terminal{ID: cmd.SubmissionID, ObjectID: cmd.CountryRegionID, Action: cmd.Action, Digest: cmd.Digest}
	fields := []string{}
	if !foundation.ValidID(t.ID) || len(t.Digest) != 32 {
		return t, nil, fault("M02_INVALID_INPUT", "", "提交标识或摘要无效")
	}
	var patch org.CountryChangesInput
	switch cmd.Action {
	case "CREATE":
		t.ObjectID = cmd.Import.ID
		fields = []string{"code", "name", "nameEn", "phoneCode", "effectiveDate", "operationType", "operationReason"}
		if !foundation.ValidID(t.ObjectID) {
			return t, nil, fault("M02_INVALID_INPUT", "", "来源对象标识无效")
		}
	case "CORRECT":
		patch = cmd.Correct.Changes
		if !foundation.ValidID(cmd.RecordID) || !foundation.ValidID(cmd.Correct.ExpectedRecordRevision) || len(patch) == 0 {
			return t, nil, fault("M02_INVALID_INPUT", "changes", "请明确更正记录及资料")
		}
	case "ADD", "DEACTIVATE", "REACTIVATE":
		patch = cmd.Add.Changes
		fields = []string{"effectiveDate", "operationType", "operationReason"}
		if !foundation.ValidID(cmd.Add.AnchorRecordID) || !foundation.ValidDate(cmd.Add.EffectiveDate) {
			return t, nil, fault("M02_INVALID_INPUT", "effectiveDate", "请明确有效日期及末条记录")
		}
	default:
		return t, nil, fault("M02_FORBIDDEN_INPUT", "action", "目录未开放此操作")
	}
	if cmd.Action != "CREATE" && (!foundation.ValidID(t.ObjectID) || !foundation.ValidID(expected(cmd))) {
		return t, nil, fault("M02_INVALID_INPUT", "", "对象或来源标识无效")
	}
	for k, v := range patch {
		if !slices.Contains([]string{"name", "nameEn", "phoneCode"}, k) {
			return t, nil, fault("M02_FORBIDDEN_INPUT", "changes."+k, "字段不允许修改")
		}
		if k == "name" && v == nil {
			return t, nil, fault("M02_INVALID_INPUT", "changes.name", "名称不能为空")
		}
		fields = append(fields, k)
	}
	if cmd.Action == "DEACTIVATE" && patch != nil {
		return t, nil, fault("M02_FORBIDDEN_INPUT", "changes", "停止选用不接受资料修改")
	}
	slices.Sort(fields)
	return t, fields, nil
}
func expected(cmd org.CountryCommand) string {
	if cmd.Action == "CORRECT" {
		return cmd.Correct.ExpectedSourceVersion
	}
	return cmd.Add.ExpectedSourceVersion
}

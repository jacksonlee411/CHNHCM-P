package enterprise

import (
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/enterprise"
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
func candidate(ctx context.Context, t d.Timeline, cmd org.EnterpriseCommand) (d.Candidate, error) {
	item := d.Item{ID: "direct", RecordID: cmd.RecordID}
	patch := func(p org.EnterpriseChangesInput) *d.Patch {
		if p == nil {
			return nil
		}
		v := d.Patch(p)
		return &v
	}
	var e error
	switch cmd.Action {
	case "ADD", "DEACTIVATE", "REACTIVATE":
		a := cmd.Add
		item.Action = "ADD"
		item.Date = a.EffectiveDate
		item.Type = a.OperationType
		item.Reason = a.OperationReason
		item.Changes = patch(a.Changes)
		if cmd.Action == "ADD" {
			if a.OperationType != "CHANGE" {
				return d.Candidate{}, fault("M02_INVALID_INPUT", "operationType", "资料变更类型无效")
			}
			if item.Changes == nil {
				v := d.Patch{}
				item.Changes = &v
			}
			e = d.Kernel.ValidateAdditionPosition(t.Records, a.AnchorRecordID, d.MinDate, a.EffectiveDate)
		} else {
			if a.OperationType != cmd.Action {
				return d.Candidate{}, fault("M02_INVALID_INPUT", "operationType", "启停操作类型无效")
			}
			e = d.Kernel.ValidateLifecycleAppend(t.Records, a.AnchorRecordID, cmd.Action, d.MinDate, a.EffectiveDate)
		}
	case "CORRECT":
		item.Action = "UPDATE"
		item.ExpectedRevision = cmd.Correct.ExpectedRecordRevision
		item.Date = cmd.Correct.NewEffectiveDate
		item.Changes = patch(cmd.Correct.Changes)
		if c := cmd.Correct.Classification; c != nil {
			item.Type = c.OperationType
			item.Reason = c.OperationReason
		}
		if item.Date != "" {
			e = d.Kernel.ValidateCorrectionDate(t.Records, item.RecordID, d.MinDate, item.Date)
		}
	case "DELETE":
		item.Action = "DELETE"
		item.ExpectedRevision = cmd.Delete.ExpectedRecordRevision
	case "INSERT_LIFECYCLE_PAIR":
		p := cmd.InsertPair
		if e = d.Kernel.ValidateLifecyclePair(t.Records, p.AnchorRecordID, d.MinDate, p.DeactivationDate, p.ReactivationDate); e != nil {
			return d.Candidate{}, e
		}
		return d.Kernel.Calculate(ctx, t.Records, t.First, d.MinDate, []d.Item{{ID: "deactivation", Action: "ADD", Date: p.DeactivationDate, Type: "DEACTIVATE", Reason: p.DeactivationReason}, {ID: "reactivation", Action: "ADD", Date: p.ReactivationDate, Type: "REACTIVATE", Reason: p.ReactivationReason}})
	case "DELETE_LIFECYCLE_PAIR":
		p := cmd.DeletePair
		a, b, e := d.Kernel.LifecyclePair(t.Records, p.DeactivationRecordID)
		if e != nil {
			return d.Candidate{}, e
		}
		if a.ID != p.DeactivationRecordID || b.ID != p.ReactivationRecordID || a.Revision != p.DeactivationRevision || b.Revision != p.ReactivationRevision {
			return d.Candidate{}, fault("M02_REVISION_CONFLICT", "", "停用和重新启用记录已更新，请核对后再操作")
		}
		return d.Kernel.Calculate(ctx, t.Records, t.First, d.MinDate, []d.Item{{ID: "deactivation", Action: "DELETE", RecordID: a.ID, ExpectedRevision: a.Revision}, {ID: "reactivation", Action: "DELETE", RecordID: b.ID, ExpectedRevision: b.Revision}})
	default:
		return d.Candidate{}, fault("M02_FORBIDDEN_INPUT", "action", "此操作不可用")
	}
	if e != nil {
		return d.Candidate{}, e
	}
	return d.Kernel.Calculate(ctx, t.Records, t.First, d.MinDate, []d.Item{item})
}
func metadata(cmd org.EnterpriseCommand) (Terminal, []string, error) {
	t := Terminal{ID: cmd.SubmissionID, ObjectID: cmd.EnterpriseID, Action: cmd.Action, Digest: cmd.Digest}
	if !foundation.ValidID(t.ID) || !foundation.ValidID(t.ObjectID) || len(t.Digest) != 32 || !foundation.ValidID(expected(cmd)) {
		return t, nil, fault("M02_INVALID_INPUT", "", "提交、对象或来源标识无效")
	}
	fields := []string{}
	var patch org.EnterpriseChangesInput
	switch cmd.Action {
	case "CORRECT":
		c := cmd.Correct
		patch = c.Changes
		if !foundation.ValidID(cmd.RecordID) || !foundation.ValidID(c.ExpectedRecordRevision) {
			return t, nil, fault("M02_INVALID_INPUT", "", "请选择有效的记录")
		}
		if c.Changes != nil && len(c.Changes) == 0 || c.Changes == nil && c.NewEffectiveDate == "" && c.Classification == nil {
			return t, nil, fault("M02_INVALID_INPUT", "changes", "没有修改内容")
		}
		if c.NewEffectiveDate != "" && !foundation.ValidDate(c.NewEffectiveDate) {
			return t, nil, fault("M02_INVALID_INPUT", "newEffectiveDate", "新日期须为有效日历日期")
		}
		if c.Classification != nil && (c.Classification.OperationType == "" || c.Classification.OperationReason == "") {
			return t, nil, fault("M02_INVALID_INPUT", "classification", "请选择操作类型和原因")
		}
	case "ADD", "DEACTIVATE", "REACTIVATE":
		patch = cmd.Add.Changes
		fields = []string{"effectiveDate", "operationType", "operationReason"}
		if !foundation.ValidID(cmd.Add.AnchorRecordID) || !foundation.ValidDate(cmd.Add.EffectiveDate) {
			return t, nil, fault("M02_INVALID_INPUT", "effectiveDate", "请明确有效日期及所选记录")
		}
	case "DELETE":
		if !foundation.ValidID(cmd.RecordID) || !foundation.ValidID(cmd.Delete.ExpectedRecordRevision) {
			return t, nil, fault("M02_INVALID_INPUT", "", "请选择有效的记录")
		}
	case "INSERT_LIFECYCLE_PAIR":
		p := cmd.InsertPair
		if !foundation.ValidID(p.AnchorRecordID) || !foundation.ValidDate(p.DeactivationDate) || !foundation.ValidDate(p.ReactivationDate) {
			return t, nil, fault("M02_INVALID_INPUT", "", "停用期间的日期和记录依据不完整")
		}
	case "DELETE_LIFECYCLE_PAIR":
		p := cmd.DeletePair
		for _, id := range []string{p.DeactivationRecordID, p.ReactivationRecordID, p.DeactivationRevision, p.ReactivationRevision} {
			if !foundation.ValidID(id) {
				return t, nil, fault("M02_INVALID_INPUT", "", "停用期间的记录依据不完整")
			}
		}
	default:
		return t, nil, fault("M02_FORBIDDEN_INPUT", "action", "此操作不可用")
	}
	for k, v := range patch {
		if !slices.Contains([]string{"name", "nameEn"}, k) {
			return t, nil, fault("M02_FORBIDDEN_INPUT", "changes."+k, "字段不允许修改")
		}
		if k == "name" && v == nil {
			return t, nil, fault("M02_INVALID_INPUT", "changes.name", "名称不能为空")
		}
		fields = append(fields, k)
	}
	if cmd.Action == "DEACTIVATE" && patch != nil {
		return t, nil, fault("M02_FORBIDDEN_INPUT", "changes", "停用不接受资料修改")
	}
	slices.Sort(fields)
	return t, fields, nil
}
func expected(c org.EnterpriseCommand) string {
	switch c.Action {
	case "CORRECT":
		return c.Correct.ExpectedSourceVersion
	case "DELETE":
		return c.Delete.ExpectedSourceVersion
	case "INSERT_LIFECYCLE_PAIR":
		return c.InsertPair.ExpectedSourceVersion
	case "DELETE_LIFECYCLE_PAIR":
		return c.DeletePair.ExpectedSourceVersion
	default:
		return c.Add.ExpectedSourceVersion
	}
}

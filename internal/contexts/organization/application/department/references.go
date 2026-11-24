package department

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/application/submission"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

func referenceCommand(in org.DepartmentReferenceContext) org.DepartmentCommand {
	cmd := org.DepartmentCommand{Action: in.Action, DepartmentID: in.DepartmentID, RecordID: in.RecordID, Add: org.DepartmentAddInput{ExpectedSourceVersion: in.ExpectedSourceVersion, AnchorRecordID: in.AnchorRecordID, EffectiveDate: in.EffectiveDate, OperationType: in.OperationType, OperationReason: in.OperationReason, Changes: in.Changes}, Correct: org.DepartmentCorrectInput{ExpectedSourceVersion: in.ExpectedSourceVersion, ExpectedRecordRevision: in.ExpectedRecordRevision, NewEffectiveDate: in.NewEffectiveDate, Changes: in.Changes}}
	if in.Classification != nil {
		cmd.Correct.Classification = &org.Classification{OperationType: in.Classification.OperationType, OperationReason: in.Classification.OperationReason}
	}
	if in.Action == "REACTIVATE" {
		cmd.Add.OperationType = "REACTIVATE"
	}
	return cmd
}

func referenceMaintenance(ctx context.Context, u Unit, t d.Timeline, now time.Time, q org.DepartmentReferenceContext) error {
	fields, err := readable(ctx, u, t, now, false)
	if err != nil {
		return err
	}
	for _, field := range []string{"managementBuId", "parentDepartmentId", "effectiveDate", "operationType"} {
		if !slices.Contains(fields, field) {
			return denied()
		}
	}
	if q.Classification != nil && q.Classification.OperationReason != "" && !slices.Contains(fields, "operationReason") {
		return denied()
	}
	if t.Source != q.ExpectedSourceVersion {
		return foundation.Fail(409, "M02_SOURCE_CONFLICT", "资料已更新，请核对最新内容后保存")
	}
	return nil
}
func referenceRows(ctx context.Context, t d.Timeline, in org.DepartmentReferenceContext, kind, choice, start string) ([]d.Record, bool, error) {
	if in.Mode == "create" {
		bu, parent := in.ManagementBUID, ""
		if kind == "bu" {
			bu = choice
		} else {
			parent = choice
		}
		v, e := d.ReferenceKernel.CreateReference(ctx, t.First, in.EffectiveDate, start, d.Data{BU: bu, Parent: parent})
		return v.Records, true, e
	}
	cmd := referenceCommand(in)
	patch := d.Patch{}
	for k, v := range in.Changes {
		patch[k] = v
	}
	if kind == "bu" {
		patch["managementBuId"] = &choice
	} else {
		patch["parentDepartmentId"] = &choice
	}
	it := d.Item{ID: "reference", RecordID: in.RecordID, ExpectedRevision: in.ExpectedRecordRevision, Action: "UPDATE", Changes: &patch, Date: in.NewEffectiveDate}
	applicable := true
	if in.Action == "CORRECT" {
		var found *d.Record
		for i := range t.Records {
			if t.Records[i].ID == in.RecordID {
				found = &t.Records[i]
			}
		}
		if found == nil {
			return nil, false, foundation.Fail(409, "M02_RECORD_NOT_FOUND", "当前资料中没有该记录")
		}
		typ := found.Type
		if in.Classification != nil {
			typ = in.Classification.OperationType
			it.Type = typ
			it.Reason = in.Classification.OperationReason
		}
		if typ == "DEACTIVATE" {
			applicable = false
			it.Changes = nil
		}
		if it.Date != "" {
			if e := d.Kernel.ValidateCorrectionDate(t.Records, in.RecordID, start, it.Date); e != nil {
				return nil, false, e
			}
		}
		// A query with no edited field remains a read of the accurately targeted record.
		if !applicable && it.Type == "" && it.Date == "" {
			if found.Revision != in.ExpectedRecordRevision {
				return nil, false, foundation.Fail(409, "M02_REVISION_CONFLICT", "记录已更新，请核对最新内容后保存")
			}
			return d.Kernel.CloneRecords(t.Records), false, nil
		}
	} else {
		it.Action = "ADD"
		it.RecordID = "reference-candidate"
		it.Date = in.EffectiveDate
		it.Type = cmd.Add.OperationType
		it.Reason = in.OperationReason
		var e error
		if in.Action == "ADD" {
			e = d.Kernel.ValidateAdditionPosition(t.Records, in.AnchorRecordID, start, it.Date)
		} else {
			e = d.Kernel.ValidateLifecycleAppend(t.Records, in.AnchorRecordID, "REACTIVATE", start, it.Date)
		}
		if e != nil {
			return nil, false, e
		}
	}
	v, e := d.ReferenceKernel.PrepareReference(ctx, t.Records, t.First, start, []d.Item{it})
	return v.Records, applicable, e
}

// References serves management-BU labels and qualification. Department parents
// have their own cursor contract in Parents; there is no legacy paging branch.
func (s Service) References(ctx context.Context, c access.Credentials, kind string, in org.DepartmentReferenceQuery) (out org.DepartmentReferences, err error) {
	out = org.DepartmentReferences{Applicable: true, Items: []org.DepartmentReferenceCandidate{}, Page: in.Page, PageSize: in.PageSize}
	if kind != "bu" {
		return out, foundation.Fail(400, "M02_INVALID_INPUT", "引用查询种类无效")
	}
	if err = validateReferenceInput(in, kind); err != nil {
		return out, err
	}
	q := in.Context
	err = s.Store.WithinDepartment(ctx, c, func(u Unit) error {
		ent, e := u.Enterprise(ctx, false)
		if e != nil {
			return e
		}
		if !departmentReadEntry(u, ent.ID) {
			return submission.RejectRead(denied())
		}
		if q.Mode == "filter" {
			permitted := false
			for _, g := range u.Access().Grants() {
				if g.Action == "department.read" && (g.ScopeKind != "enterprise" || g.ScopeID == ent.ID) && slices.Contains(g.ReadFields, "managementBuId") {
					permitted = true
				}
			}
			if !permitted {
				return submission.RejectRead(denied())
			}
		}
		now, e := u.Now(ctx)
		if e != nil {
			return e
		}
		id := ""
		if q.Mode == "maintain" {
			id = q.DepartmentID
		}
		read, e := u.BUReferenceSnapshot(ctx, id)
		if e != nil {
			return e
		}
		t := d.Timeline{ID: foundation.NewID(), First: foundation.NewID(), Enterprise: ent.ID}
		if q.Mode == "maintain" {
			current := read.Snapshot.Department(q.DepartmentID)
			if current == nil {
				return submission.RejectRead(missing())
			}
			t = *current
			if e = referenceMaintenance(ctx, u, t, now, q); e != nil {
				return submission.RejectRead(e)
			}
		}
		bu := q.ManagementBUID
		if q.Mode == "maintain" {
			bu, _, _, _, _ = explicitTargets(referenceCommand(q), t)
		}
		projector := newProjector(ctx, u, read.Snapshot, ent, now)
		if in.SelectedID != "" {
			selected := projector.ref(in.SelectedID, "bu")
			if selected.err != nil {
				return selected.err
			}
			out.SelectedReference = selected.value
		}
		if q.Mode != "filter" {
			_, applicable, e := referenceRows(ctx, t, q, "bu", bu, ent.ValidFrom)
			if e != nil {
				return submission.RejectRead(e)
			}
			out.Applicable = applicable
			if !applicable {
				return nil
			}
		}
		projected, e := referenceAuthorization(ctx, u, d.Snapshot{BusinessUnits: read.Snapshot.BusinessUnits}, ent.ID, now, "bu")
		if e != nil {
			return e
		}
		refs := projector.refs
		projector = newProjector(ctx, projected, read.Snapshot, ent, now)
		projector.refs = refs
		all := []org.DepartmentReferenceCandidate{}
		for _, b := range read.Snapshot.BusinessUnits {
			if e = ctx.Err(); e != nil {
				return e
			}
			ref := projector.ref(b.ID, "bu")
			if ref.err != nil {
				return ref.err
			}
			if ref.value.Code == nil && ref.value.Name == nil {
				continue
			}
			text := ""
			if ref.value.Code != nil {
				text += *ref.value.Code
			}
			if ref.value.Name != nil {
				text += " " + *ref.value.Name
			}
			if in.Q != b.ID && !strings.Contains(strings.ToLower(text), strings.ToLower(strings.TrimSpace(in.Q))) {
				continue
			}
			if q.Mode == "create" {
				target := fact(t, now, b.ID)
				dec, e := u.Access().Decide(ctx, requirements(org.DepartmentCommand{Action: "CREATE", Fields: CreateFields}, t, target))
				if e != nil {
					return e
				}
				if !dec.Allowed {
					continue
				}
			}
			eligibility := "readable"
			if q.Mode != "filter" {
				rows, _, e := referenceRows(ctx, t, q, "bu", b.ID, ent.ValidFrom)
				if e != nil {
					return submission.RejectRead(e)
				}
				candidate := t
				candidate.Records = rows
				if e = d.ValidateBUCoverage(ctx, read.Snapshot, candidate); e != nil {
					var qualification *d.QualificationError
					if errors.As(e, &qualification) {
						continue
					}
					return e
				}
				eligibility = "eligible"
			}
			all = append(all, org.DepartmentReferenceCandidate{ID: b.ID, Code: ref.value.Code, Name: ref.value.Name, Eligibility: eligibility})
		}
		slices.SortFunc(all, func(a, b org.DepartmentReferenceCandidate) int {
			code := func(v org.DepartmentReferenceCandidate) string {
				if v.Code == nil {
					return ""
				}
				return *v.Code
			}
			if n := strings.Compare(code(a), code(b)); n != 0 {
				return n
			}
			return strings.Compare(a.ID, b.ID)
		})
		out.Count = len(all)
		start := min((in.Page-1)*in.PageSize, len(all))
		out.Items = all[start:min(start+in.PageSize, len(all))]
		return nil
	})
	return
}

func validateReferenceInput(in org.DepartmentReferenceQuery, kind string) error {
	if in.Page < 1 || in.Page > 1000000 || !slices.Contains([]int{20, 50, 100}, in.PageSize) || utf8.RuneCountInString(in.Q) > 400 || !slices.Contains([]string{"bu", "department"}, kind) || in.SelectedID != "" && !foundation.ValidID(in.SelectedID) {
		return foundation.Fail(400, "M02_INVALID_INPUT", "引用查询条件无效")
	}
	q := in.Context
	if q.Mode != "filter" && q.Mode != "create" && (q.Mode != "maintain" || !foundation.ValidID(q.DepartmentID) || !foundation.ValidID(q.ExpectedSourceVersion) || !slices.Contains([]string{"ADD", "CORRECT", "REACTIVATE"}, q.Action)) {
		return foundation.Fail(400, "M02_INVALID_INPUT", "无法读取候选资料，请重新打开表单")
	}
	if q.Mode == "maintain" && (q.Action == "CORRECT" && (!foundation.ValidID(q.RecordID) || !foundation.ValidID(q.ExpectedRecordRevision)) || q.Action != "CORRECT" && !foundation.ValidID(q.AnchorRecordID)) || q.ManagementBUID != "" && !foundation.ValidID(q.ManagementBUID) {
		return foundation.Fail(400, "M02_INVALID_INPUT", "无法读取候选资料，请重新打开表单")
	}
	if q.Mode == "filter" && kind != "bu" {
		return foundation.Fail(400, "M02_INVALID_INPUT", "筛选仅支持管理业务单元")
	}
	for key, v := range q.Changes {
		if !slices.Contains([]string{"managementBuId", "parentDepartmentId"}, key) || v == nil && key == "managementBuId" || v != nil && !foundation.ValidID(*v) {
			return foundation.Fail(400, "M02_INVALID_INPUT", "关系字段无效")
		}
	}

	return nil
}

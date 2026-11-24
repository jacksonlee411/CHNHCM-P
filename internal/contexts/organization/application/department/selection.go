package department

import (
	"context"
	"encoding/csv"
	"errors"
	"io"
	"slices"
	"time"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	shared "chnhcm.local/chnhcm/internal/contexts/organization/application/selection"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

// Each operation fixes its business day once, even when selection identity and
// the independently current page use different statements in the transaction.
type selectionUnit struct {
	Unit
	now time.Time
}

func (u selectionUnit) Now(context.Context) (time.Time, error) { return u.now, nil }

func selectionQuery(q org.DepartmentSelectionQuery, sort string, page, size int) (org.DepartmentListInput, error) {
	in := org.DepartmentListInput{ScopeDepartmentID: q.ScopeDepartmentID, IncludeDescendants: q.IncludeDescendants, Q: q.Q, Status: q.Status, Sort: sort, Page: page, PageSize: size}
	if q.FilterField != "" || q.FilterOperator != "" || q.FilterValue != "" {
		in.Filter = &org.ColumnFilter{Field: q.FilterField, Operator: q.FilterOperator, Value: q.FilterValue}
	}
	return in, normalizeDepartmentList(&in)
}

func selectionPlan(ctx context.Context, u Unit) (access.ReadPlan, error) {
	ent, err := u.Enterprise(ctx, false)
	if err != nil {
		return access.ReadPlan{}, err
	}
	now, err := u.Now(ctx)
	if err != nil {
		return access.ReadPlan{}, err
	}
	planner, ok := u.Access().(access.ReadPlanner)
	if !ok {
		return access.ReadPlan{}, foundation.Fail(503, "M02_DEPENDENCY_UNAVAILABLE", "部门资料暂时无法加载")
	}
	plan, err := planner.PlanReads(ctx, ent.ID, now, access.DepartmentRead)
	if err == nil && !plan.Allowed.All && len(plan.Allowed.BUs) == 0 && len(plan.Allowed.Departments) == 0 {
		return plan, denied()
	}
	return plan, err
}

func selectionIDs(ctx context.Context, u Unit, q org.DepartmentListInput, plan access.ReadPlan, ids []string) ([]string, error) {
	if q.Filter != nil && q.Filter.Field == "departmentPath" {
		ent, e := u.Enterprise(ctx, false)
		if e != nil {
			return nil, e
		}
		// Full path selection necessarily visits every candidate. Capture its
		// shared current/path material once in one SQL snapshot, instead of
		// transferring and decoding the same ancestors once per root frame.
		captured, e := u.CaptureQuery(ctx, QueryRequest{Plan: plan, Input: q, Extent: QuerySelection, OnlyIDs: ids})
		if e != nil {
			return nil, e
		}
		return pathMatchingIDs(ctx, u, q, plan, captured, ent)

	}
	captured, err := u.CaptureQuery(ctx, QueryRequest{Plan: plan, Input: q, Extent: QueryIdentities, OnlyIDs: ids})
	return captured.PageIDs, err
}

func validSelectionIDs(ids []string) error {
	if len(ids) > DraftEditLimit {
		return DraftBatchLimit()
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if !foundation.ValidID(id) || seen[id] {
			return shared.Invalid()
		}
		seen[id] = true
	}
	return nil
}

func readDraftPage(ctx context.Context, u Unit, q org.DepartmentListInput, enterprise string) (page org.DepartmentList, ids []string, can bool, err error) {
	page, rows, _, now, err := queryRows(ctx, u, q, QueryPage)
	if err != nil {
		return page, nil, false, err
	}
	page.Items = memberSummaries(rows)
	ids = make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.t.ID)
	}
	planner := u.Access().(access.ReadPlanner)
	export, err := planner.PlanReads(ctx, enterprise, now, access.DepartmentExport)
	if err != nil {
		return page, nil, false, err
	}
	can = export.Allowed.All || len(export.Allowed.BUs) > 0 || len(export.Allowed.Departments) > 0
	return
}

func bindDraftPage(ctx context.Context, u Unit, page org.DepartmentList, ids []string, can bool, draft Draft) (org.DepartmentSelectionPage, error) {
	selected, err := u.DraftPage(ctx, draft, ids)
	if err != nil {
		return org.DepartmentSelectionPage{}, err
	}
	return org.DepartmentSelectionPage{PageState: "available", BusinessAt: page.BusinessAt, Page: page, SelectedIDs: selected.Selected, MemberIDs: selected.Members, Summary: org.DepartmentSelectionResult{QueryKey: draft.QueryKey, DraftID: draft.ID, Revision: draft.Revision, SelectedCount: draft.Selected, CanExport: can}}, nil
}

func draftPage(ctx context.Context, u Unit, q org.DepartmentListInput, draft Draft) (org.DepartmentSelectionPage, error) {
	page, ids, can, err := readDraftPage(ctx, u, q, draft.Enterprise)
	if err != nil {
		return org.DepartmentSelectionPage{}, err
	}
	return bindDraftPage(ctx, u, page, ids, can, draft)
}

func (s Service) StartSelection(ctx context.Context, c access.Credentials, in org.DepartmentSelectionStartInput) (out org.DepartmentSelectionPage, err error) {
	if in.Page < 1 || in.PageSize != 20 && in.PageSize != 50 && in.PageSize != 100 {
		return out, shared.Invalid()
	}
	q, err := selectionQuery(in.Query, in.Sort, in.Page, in.PageSize)
	q.Cursor = in.Cursor
	if err != nil {
		return out, err
	}
	var ids []string
	switch in.Selection.Mode {
	case "explicit":
		if in.Selection.IDs == nil || in.Selection.ExcludedIDs != nil {
			return out, shared.Invalid()
		}
		ids = *in.Selection.IDs
		if err = validSelectionIDs(ids); err != nil {
			return out, err
		}
		if ids == nil {
			ids = []string{}
		}
	case "allMatching":
		if in.Selection.IDs != nil || in.Selection.ExcludedIDs == nil || len(*in.Selection.ExcludedIDs) != 0 {
			return out, shared.Invalid()
		}
	default:
		return out, shared.Invalid()
	}
	err = s.Store.WithinDepartment(ctx, c, func(raw Unit) error {
		now, e := raw.Now(ctx)
		if e != nil {
			return e
		}
		u := selectionUnit{raw, now}
		plan, e := selectionPlan(ctx, u)
		if e != nil {
			return e
		}
		captured, e := selectionIDs(ctx, u, q, plan, ids)
		if e != nil {
			return e
		}
		if ids != nil && len(captured) != len(ids) {
			return shared.Invalid()
		}
		// Read the current page before reserving a temporary admission slot.
		// The bounded slot reservation does not serialize other sessions.
		page, pageIDs, can, e := readDraftPage(ctx, u, q, plan.Enterprise)
		if e != nil {
			return e
		}
		draft, e := u.CreateDraft(ctx, Draft{Enterprise: plan.Enterprise, Day: plan.Day, QueryKey: queryKey(q, plan.Instance+":"+plan.Subject), Mode: in.Selection.Mode}, captured)
		if e != nil {
			return e
		}
		out, e = bindDraftPage(ctx, u, page, pageIDs, can, draft)
		return e
	})
	return
}

func (s Service) ResolveSelection(ctx context.Context, c access.Credentials, in org.DepartmentSelectionInput) (out org.DepartmentSelectionPage, err error) {
	if in.Page < 1 || in.PageSize != 20 && in.PageSize != 50 && in.PageSize != 100 {
		return out, shared.Invalid()
	}
	q, err := selectionQuery(in.Query, in.Sort, in.Page, in.PageSize)
	q.Cursor = in.Cursor
	if err != nil {
		return out, err
	}
	if err = validSelectionIDs(append(append(append([]string{}, in.AddIDs...), in.RemoveIDs...), in.RestoreIDs...)); err != nil {
		return out, err
	}
	err = s.Store.WithinDepartment(ctx, c, func(raw Unit) error {
		now, e := raw.Now(ctx)
		if e != nil {
			return e
		}
		u := selectionUnit{raw, now}
		plan, e := selectionPlan(ctx, u)
		if e != nil {
			return e
		}
		draft, e := u.ReadDraft(ctx, in.DraftID, in.ExpectedRevision, plan.Day, queryKey(q, plan.Instance+":"+plan.Subject))
		if e != nil {
			return e
		}
		if len(in.AddIDs) > 0 {
			ids, e := selectionIDs(ctx, u, q, plan, in.AddIDs)
			if e != nil {
				return e
			}
			if len(ids) != len(in.AddIDs) {
				return shared.Invalid()
			}
		}
		draft, e = u.EditDraft(ctx, draft, DraftEdit{Add: in.AddIDs, Remove: in.RemoveIDs, Restore: in.RestoreIDs})
		if e != nil {
			return e
		}
		out, e = draftPage(ctx, u, q, draft)
		var fault *foundation.Fault
		if e != nil && errors.As(e, &fault) && fault.Code == "M02_TREE_SCOPE_UNAVAILABLE" && len(in.AddIDs)+len(in.RemoveIDs)+len(in.RestoreIDs) == 0 {
			export, err := u.Access().(access.ReadPlanner).PlanReads(ctx, plan.Enterprise, now, access.DepartmentExport)
			if err != nil {
				return err
			}
			can := export.Allowed.All || len(export.Allowed.BUs) > 0 || len(export.Allowed.Departments) > 0
			out = org.DepartmentSelectionPage{PageState: "scope_unavailable", BusinessAt: plan.Day, Summary: org.DepartmentSelectionResult{QueryKey: draft.QueryKey, DraftID: draft.ID, Revision: draft.Revision, SelectedCount: draft.Selected, CanExport: can}}
			return nil
		}
		return e
	})
	return
}

func (s Service) CancelSelection(ctx context.Context, c access.Credentials, in org.DepartmentDraftInput) error {
	return s.Store.WithinDepartment(ctx, c, func(u Unit) error { return u.CancelDraft(ctx, in.DraftID, in.ExpectedRevision) })
}

func (s Service) ExportSelection(ctx context.Context, c access.Credentials, in org.DepartmentExportInput) (out org.DepartmentExport, err error) {
	q, err := selectionQuery(in.Query, in.Sort, 1, 20)
	if err != nil {
		return out, err
	}
	var file ExportFile
	defer func() {
		if err != nil && file != nil {
			_ = file.Close()
			out = org.DepartmentExport{}
		}
	}()
	err = s.Store.WithinDepartment(ctx, c, func(raw Unit) error {
		now, e := raw.Now(ctx)
		if e != nil {
			return e
		}
		u := selectionUnit{raw, now}
		plan, e := selectionPlan(ctx, u)
		if e != nil {
			return e
		}
		draft, e := u.ReadDraft(ctx, in.DraftID, in.ExpectedRevision, plan.Day, queryKey(q, plan.Instance+":"+plan.Subject))
		if e != nil {
			return e
		}
		ids, e := u.DraftMembers(ctx, draft)
		if e != nil {
			return e
		}
		if len(ids) == 0 {
			return foundation.Fail(400, "M02_EMPTY_SELECTION", "请先选择部门")
		}
		current, e := selectionQuery(org.DepartmentSelectionQuery{}, q.Sort, 1, 20)
		if e != nil {
			return e
		}
		list, rows, read, _, e := queryRowsIDs(ctx, u, current, QuerySelection, ids)
		if e != nil {
			return e
		}
		if len(rows) != len(ids) {
			return denied()
		}
		if q.ScopeDepartmentID != "" {
			currentFacts := read.ByDay[list.BusinessAt]
			reads, e := capturedAuthorization(ctx, u, currentFacts, plan, u.Access().(access.ReadPlanner))
			if e != nil {
				return e
			}
			// All captured departments are selected identities or necessary path
			// references. Check those references without reapplying live scope.
			for _, fact := range currentFacts.Departments {
				ref := reads.referenceDecision(fact.ID, "department")
				if !ref.Allowed || !slices.Contains(ref.ReadFields, "code") || !slices.Contains(ref.ReadFields, "name") {
					return denied()
				}
			}
		}
		for _, row := range rows {
			if q.ScopeDepartmentID != "" {
				for _, field := range treeFields {
					if !row.has(field) {
						return denied()
					}
				}
			}
			if q.Filter != nil && !row.has(q.Filter.Field) || q.Status != "all" && !row.has("status") {
				return denied()
			}
		}
		allowed, e := selectionExportAllowed(ctx, u, read, list.BusinessAt, now, rows)
		if e != nil {
			return e
		}
		if !allowed {
			return denied()
		}
		file, e = s.Store.CreateExport(ctx)
		if e != nil {
			return e
		}
		if e = departmentCSV(ctx, file, rows); e != nil {
			return e
		}
		if e = file.Seal(); e != nil {
			return e
		}
		out = org.DepartmentExport{File: file, Count: len(rows), BusinessAt: list.BusinessAt}
		return nil
	})
	return
}

// Export is a read action with its own plan under the same policy guard. The
// source scope always comes from the captured current day, never a path date.
func selectionExportAllowed(ctx context.Context, u Unit, read QueryCapture, day string, now time.Time, rows []listMember) (bool, error) {
	if len(rows) == 0 {
		return false, nil
	}
	planner, ok := u.Access().(access.ReadPlanner)
	if !ok {
		return false, foundation.Fail(503, "M02_DEPENDENCY_UNAVAILABLE", "部门资料暂时无法加载")
	}
	plan, err := planner.PlanReads(ctx, rows[0].t.Enterprise, now, access.DepartmentExport)
	if err != nil {
		return false, err
	}
	current := read.ByDay[day]
	allowed := true
	for _, row := range rows {
		t := current.Department(row.t.ID)
		if t == nil {
			return false, foundation.Fail(503, "M02_DEPENDENCY_UNAVAILABLE", "部门列表暂时无法加载")
		}
		target := factOnDay(*t, now, day, "")
		decision, e := planner.DecidePlan(ctx, plan, access.ReadIntent{Action: string(access.DepartmentExport), Kind: "object", EnterpriseID: plan.Enterprise, DepartmentTarget: &target})
		if e != nil {
			return false, e
		}
		allowed = allowed && decision.Allowed
	}
	return allowed, nil
}

func departmentCSV(ctx context.Context, output io.Writer, rows []listMember) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := io.WriteString(output, "\xef\xbb\xbf"); err != nil {
		return err
	}
	writer := csv.NewWriter(output)
	writer.UseCRLF = true
	if e := writer.Write([]string{"编码", "名称", "简称", "BU编码", "BU名称", "部门路径", "状态", "生效日期"}); e != nil {
		return e
	}
	text := func(p *string) string {
		if p == nil {
			return ""
		}
		return shared.CSVText(*p)
	}
	for _, member := range rows {
		r := member.summary()
		if e := ctx.Err(); e != nil {
			return e
		}
		code, name := "", ""
		if r.ManagementBU != nil {
			code = text(r.ManagementBU.Code)
			name = text(r.ManagementBU.Name)
		}
		status := map[string]string{"active": "启用", "inactive": "停用", "not_effective": "未生效"}[text(r.Status)]
		if e := writer.Write([]string{text(r.Code), text(r.Name), text(r.ShortName), code, name, text(r.Path.Path), status, text(r.EffectiveDate)}); e != nil {
			return e
		}
	}
	writer.Flush()
	if writer.Error() != nil {
		return writer.Error()
	}
	return ctx.Err()
}

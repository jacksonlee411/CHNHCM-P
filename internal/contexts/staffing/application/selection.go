package application

import (
	"bytes"
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	s "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"encoding/csv"
	"encoding/json"
	"slices"
	"strings"
	"time"
)

func identityQuery(q s.Query) s.Query { q.Sort = ""; q.Page = 0; q.PageSize = 0; return q }
func selectionUnavailable() error {
	return foundation.Fail(409, "M03_SELECTION_UNAVAILABLE", "固定选择已失效，请重新选择")
}
func validateDraft(u Unit, v Draft, in s.SelectionInput) error {
	if v.Cancelled || v.BusinessAt != today(u) || !u.Now().Before(v.Expiry) {
		return selectionUnavailable()
	}
	left, right := identityQuery(in.Query), identityQuery(v.Query)
	includeLeft := left.IncludeDescendants == nil || *left.IncludeDescendants
	includeRight := right.IncludeDescendants == nil || *right.IncludeDescendants
	left.IncludeDescendants, right.IncludeDescendants = nil, nil
	if left != right || includeLeft != includeRight {
		return foundation.Fail(409, "M03_SELECTION_QUERY_CHANGED", "筛选条件已变化，请重新选择")
	}
	return nil
}

func (svc Service) selectionTransaction(ctx context.Context, c foundation.Credentials, q s.Query, fn func(Unit) error) error {
	if q.ScopeDepartmentID != "" {
		return svc.Store.WithinSelection(ctx, c, fn)
	}
	return svc.Store.Within(ctx, c, true, fn)
}

// A fixed selection keeps its captured members when the department moves or
// stops being navigable. The anchor's present grants still constrain delivery.
func fixedScopeAccess(ctx context.Context, u Unit, q s.Query) error {
	if q.ScopeDepartmentID == "" {
		return nil
	}
	refs, err := u.PositionSources(ctx, "departmentId", []string{q.ScopeDepartmentID}, false)
	if err != nil {
		return err
	}
	denied := foundation.Fail(403, "M03_FILTER_FORBIDDEN", "无权使用原部门范围")
	if len(refs) != 1 {
		return denied
	}
	target := refs[0]
	for _, action := range []string{"department.read", "department.reference.read"} {
		fields := []string{"code", "name"}
		if action == "department.read" {
			fields = append(fields, "status", "effectiveDate", "parentDepartmentId", "departmentPath")
		}
		decision, err := u.Access().Decide(ctx, access.Intent{DepartmentTarget: target.DepartmentTarget, EnterpriseID: target.Enterprise, Kind: "object", Action: action})
		if err != nil {
			return err
		}
		if !decision.Allowed || !has(decision.ReadFields, fields...) {
			return denied
		}
	}
	return nil
}
func (svc Service) Select(ctx context.Context, c foundation.Credentials, k s.Kind, op string, in s.SelectionInput) (out s.Selection, err error) {
	ctx = readContext(ctx)
	if op != "start" && in.InitialIDs != nil {
		return out, foundation.Fail(400, "M03_INVALID_INPUT", "初始选择只适用于新建选择")
	}
	if !foundation.ValidID(in.DraftID) {
		return out, foundation.Fail(400, "M03_INVALID_INPUT", "选择标识无效")
	}
	err = svc.selectionTransaction(ctx, c, in.Query, func(u Unit) error {
		if e := u.LockSubmission(ctx, d.Kind(k), in.DraftID); e != nil {
			return e
		}
		v, found, e := u.Draft(ctx, d.Kind(k), in.DraftID, true)
		if e != nil {
			return e
		}
		if !found {
			v = Draft{Selection: s.Selection{State: "active", DraftID: in.DraftID, Revision: foundation.NewID(), BusinessAt: today(u), Members: []string{}, ExcludedIDs: []string{}}, Query: identityQuery(in.Query), Expiry: u.Now().Add(30 * time.Minute)}
			if op != "start" {
				v.Cancelled = true
				v.State = "cancelled"
				if e = u.SaveDraft(ctx, d.Kind(k), v); e != nil {
					return e
				}
				out = v.Selection
				return nil
			}
			items, _, _, e := listAll(ctx, u, d.Kind(k), in.Query)
			if e != nil {
				return e
			}
			for _, item := range items {
				v.Members = append(v.Members, item["id"].(string))
			}
			if in.InitialIDs != nil {
				seen := map[string]bool{}
				for _, id := range in.InitialIDs {
					if seen[id] || !slices.Contains(v.Members, id) {
						return foundation.Fail(400, "M03_INVALID_INPUT", "初始成员不属于查询结果")
					}
					seen[id] = true
				}
				for _, id := range v.Members {
					if !seen[id] {
						v.ExcludedIDs = append(v.ExcludedIDs, id)
					}
				}
			}
			v.Count = len(v.Members) - len(v.ExcludedIDs)
			if e = u.SaveDraft(ctx, d.Kind(k), v); e != nil {
				return e
			}
			out = v.Selection
			return nil
		}
		if op == "cancel" {
			v.Cancelled = true
			v.State = "cancelled"
			v.Revision = foundation.NewID()
			out = v.Selection
			return u.SaveDraft(ctx, d.Kind(k), v)
		}
		if e = validateDraft(u, v, in); e != nil {
			return e
		}
		if op != "start" && (in.ExpectedRevision != "" || in.ExcludedIDs != nil) && in.ExpectedRevision != v.Revision {
			return foundation.Fail(409, "M03_SELECTION_CONFLICT", "固定选择已变化，请重新确认")
		}
		ent, e := u.Enterprise(ctx, false)
		if e != nil {
			return e
		}
		read, e := decide(ctx, u, d.Kind(k), ent.ID, "read", nil)
		if e != nil {
			return e
		}
		if e = queryCheck(ctx, u, d.Kind(k), ent.ID, in.Query, read.ReadFields); e != nil {
			return e
		}
		if e = fixedScopeAccess(ctx, u, in.Query); e != nil {
			return e
		}
		if op == "resolve" && in.ExcludedIDs != nil {
			ids := slices.Clone(in.ExcludedIDs)
			slices.Sort(ids)
			ids = slices.Compact(ids)
			for _, id := range ids {
				if !slices.Contains(v.Members, id) {
					return foundation.Fail(400, "M03_INVALID_INPUT", "排除成员不属于固定选择")
				}
			}
			old := slices.Clone(v.ExcludedIDs)
			slices.Sort(old)
			if !slices.Equal(old, ids) {
				v.ExcludedIDs = ids
				v.Revision = foundation.NewID()
				if e = u.SaveDraft(ctx, d.Kind(k), v); e != nil {
					return e
				}
			}
		}
		v.Count = len(v.Members) - len(v.ExcludedIDs)
		out = v.Selection
		return nil
	})
	return
}
func csvFields(k d.Kind) ([]string, []string) {
	if k == d.Position {
		return []string{"code", "name", "nameEn", "departmentId", "jobId", "locationId", "gradeId", "status", "effectiveDate"}, []string{"编码", "名称", "英文名称", "所属部门", "职务", "工作地点", "职级", "状态", "生效日期"}
	}
	if k == d.Ladder {
		return []string{"code", "name", "nameEn", "description", "status", "effectiveDate", "gradeIds"}, []string{"编码", "名称", "英文名称", "说明", "状态", "生效日期", "职级成员"}
	}
	if k == d.JobFunction || k == d.Family || k == d.Grade {
		return []string{"code", "name", "nameEn", "description", "status", "effectiveDate"}, []string{"编码", "名称", "英文名称", "说明", "状态", "生效日期"}
	}
	return []string{"code", "name", "nameEn", "jobFamilyId", "responsibilities", "requirements", "status", "effectiveDate", "jobFunctionId"}, []string{"编码", "名称", "英文名称", "职务系列", "主要职责", "任职要求", "状态", "生效日期", "职类"}
}
func csvText(s string) string {
	v := strings.TrimLeft(s, " \t\r\n")
	if v != "" && strings.ContainsRune("=+-@", rune(v[0])) {
		return "'" + s
	}
	return s
}
func (svc Service) Export(ctx context.Context, c foundation.Credentials, k s.Kind, in s.SelectionInput) (out s.Export, err error) {
	ctx = readContext(ctx)
	if !foundation.ValidID(in.DraftID) {
		return out, selectionUnavailable()
	}
	err = svc.selectionTransaction(ctx, c, in.Query, func(u Unit) error {
		if e := u.LockSubmission(ctx, d.Kind(k), in.DraftID); e != nil {
			return e
		}
		v, found, e := u.Draft(ctx, d.Kind(k), in.DraftID, true)
		if e != nil {
			return e
		}
		if !found {
			return selectionUnavailable()
		}
		if e = validateDraft(u, v, in); e != nil {
			return e
		}
		if in.ExpectedRevision != v.Revision {
			return foundation.Fail(409, "M03_SELECTION_CONFLICT", "固定选择已变化，请重新确认")
		}
		ent, e := u.Enterprise(ctx, false)
		if e != nil {
			return e
		}
		read, e := decide(ctx, u, d.Kind(k), ent.ID, "read", nil)
		if e != nil {
			return e
		}
		ex, e := decide(ctx, u, d.Kind(k), ent.ID, "export", nil)
		if e != nil {
			return e
		}
		fields, headers := csvFields(d.Kind(k))
		if !has(read.ReadFields, fields...) || !has(ex.ReadFields, fields...) {
			return foundation.Fail(403, "M03_EXPORT_FORBIDDEN", "导出列权限已变化，无法生成完整文件")
		}
		if e = queryCheck(ctx, u, d.Kind(k), ent.ID, in.Query, read.ReadFields); e != nil {
			return e
		}
		if e = fixedScopeAccess(ctx, u, in.Query); e != nil {
			return e
		}
		// Export has a separate ordering input. Validate it against the same
		// object/field policy; a valid captured query cannot authorize another sort.
		if e = queryCheck(ctx, u, d.Kind(k), ent.ID, s.Query{Sort: in.Sort}, read.ReadFields); e != nil {
			return e
		}
		items := []s.View{}
		for _, id := range v.Members {
			if slices.Contains(v.ExcludedIDs, id) {
				continue
			}
			t, e := u.Load(ctx, d.Kind(k), id, "")
			if e != nil {
				return e
			}
			if t.ID == "" {
				return selectionUnavailable()
			}
			row, e := summary(ctx, u, t, fields)
			if e != nil {
				return e
			}
			if d.Kind(k) == d.Job {
				index, _ := selected(t, today(u))
				if t.Records[index].Data["jobFunctionId"] != "" && row["jobFunctionId"] == nil {
					return foundation.Fail(403, "M03_EXPORT_FORBIDDEN", "职类资料不可读，无法生成完整文件")
				}
				if row["jobFunctionId"] != nil {
					row["jobFunctionId"] = row["jobFunctionName"]
				}
			}
			if d.Kind(k) == d.Job && row["jobFamilyId"] == nil {
				return foundation.Fail(403, "M03_EXPORT_FORBIDDEN", "职务系列资料不可读，无法生成完整文件")
			}
			if d.Kind(k) == d.Position {
				index, state := selected(t, today(u))
				record := t.Records[index]
				date := today(u)
				if state == "not_effective" {
					date = record.Date
				} else if state == "inactive" {
					date = dayBefore(record.Date)
				}
				if err := projectPosition(ctx, u, record, row, fields, date, false); err != nil {
					return err
				}
				for _, field := range d.PositionFields() {
					if row[field] == nil {
						return foundation.Fail(403, "M03_EXPORT_FORBIDDEN", "引用资料不可读，无法生成完整文件")
					}
					row[field] = row[strings.TrimSuffix(field, "Id")+"Name"]
				}
			}
			if d.Kind(k) == d.Ladder {
				index, state := selected(t, today(u))
				date := today(u)
				if state == "not_effective" {
					date = t.Records[index].Date
				} else if state == "inactive" {
					date = dayBefore(t.Records[index].Date)
				}
				members, visible, err := memberViews(ctx, u, t, t.Records[index], date, false)
				if err != nil {
					return err
				}
				if !visible {
					return foundation.Fail(403, "M03_EXPORT_FORBIDDEN", "职级资料不可读，无法生成完整文件")
				}
				pairs := [][2]string{}
				for _, member := range members {
					if member.Code == nil {
						return foundation.Fail(403, "M03_EXPORT_FORBIDDEN", "职级编码不可读，无法生成完整文件")
					}
					pairs = append(pairs, [2]string{*member.Code, member.Name})
				}
				encoded, err := json.Marshal(pairs)
				if err != nil {
					return err
				}
				row["gradeIds"] = string(encoded)
			}
			items = append(items, row)
		}
		sortViews(items, in.Sort, read.ReadFields)
		var buf bytes.Buffer
		buf.WriteString("\xef\xbb\xbf")
		w := csv.NewWriter(&buf)
		if e = w.Write(headers); e != nil {
			return e
		}
		for _, item := range items {
			row := []string{}
			for _, field := range fields {
				if field == "jobFamilyId" {
					field = "jobFamilyName"
				}
				value, _ := item[field].(string)
				if field == "status" {
					value = map[string]string{"active": "启用", "inactive": "停用", "not_effective": "未生效"}[value]
				}
				row = append(row, csvText(value))
			}
			if e = w.Write(row); e != nil {
				return e
			}
		}
		w.Flush()
		if e = w.Error(); e != nil {
			return e
		}
		out = s.Export{Bytes: buf.Bytes(), Count: len(items), Date: today(u)}
		return nil
	})
	return
}

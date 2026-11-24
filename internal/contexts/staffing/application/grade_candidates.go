package application

import (
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	s "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"slices"
	"strings"
	"unicode/utf8"
)

func (svc Service) GradeReferenceContext(ctx context.Context, c foundation.Credentials, cmd s.Command) (out s.View, err error) {
	k := d.Kind(cmd.Kind)
	if !d.HasGrades(k) || action(cmd.Input.Action) == "" {
		return nil, foundation.Fail(400, "M03_INVALID_INPUT", "职级候选动作无效")
	}
	if utf8.RuneCountInString(cmd.Input.ReferenceQ) > 2000 {
		return nil, &foundation.Fault{Status: 400, Code: "M03_INVALID_INPUT", Message: "职级搜索内容不能超过2000个字符", Field: "referenceQ"}
	}
	for _, id := range []string{cmd.ObjectID, cmd.Input.RecordID, cmd.Input.AnchorRecordID} {
		if id != "" && !foundation.ValidID(id) {
			return nil, foundation.Fail(400, "M03_INVALID_INPUT", "候选定位标识无效")
		}
	}
	page, size := cmd.Input.ReferencePage, cmd.Input.ReferencePageSize
	if page == 0 {
		page = 1
	}
	if size == 0 {
		size = 20
	}
	if page < 1 || page > 1000000 || !slices.Contains([]int{20, 50, 100}, size) {
		return nil, foundation.Fail(400, "M03_INVALID_INPUT", "候选分页无效")
	}
	ctx = readContext(ctx)
	err = svc.Store.Within(ctx, c, false, func(u Unit) error {
		ent, e := u.Enterprise(ctx, false)
		if e != nil {
			return e
		}
		rd, e := decide(ctx, u, k, ent.ID, "read", nil)
		if e != nil {
			return e
		}
		wd, e := decide(ctx, u, k, ent.ID, action(cmd.Input.Action), nil)
		if e != nil {
			return e
		}
		if !has(rd.ReadFields, "gradeIds") || !has(wd.WriteFields, "gradeIds") {
			return foundation.Fail(403, "M03_FIELD_FORBIDDEN", "无权维护职级成员")
		}
		before := d.Timeline{}
		if cmd.ObjectID != "" {
			before, e = u.Load(ctx, k, cmd.ObjectID, "")
			if e != nil {
				return e
			}
		}
		probe := command(cmd)
		// A complete intended collection is mandatory even when empty; omission
		// cannot silently switch candidate context to a different saved collection.
		if !probe.Grades.Present {
			return foundation.Fail(400, "M03_INVALID_INPUT", "请提供完整拟职级成员")
		}
		p, e := d.ReferencePlan(k, ent.ID, before, probe)
		if e != nil {
			return e
		}
		at := d.Index(p.After, p.Selected)
		if at < 0 {
			return foundation.Fail(400, "M03_INVALID_INPUT", "候选定位无效")
		}
		selected := p.After.Records[at]
		if !selected.Active() {
			out = s.View{"applicable": false, "items": []s.View{}, "selected": []s.GradeMemberView{}, "page": page, "pageSize": size, "totalCount": 0, "businessAt": today(u)}
			return nil
		}
		if e = requireReplacementReadable(ctx, u, p, true); e != nil {
			return e
		}
		ok, e := gradeRead(ctx, u, ent.ID, selected.Date, false)
		if e != nil {
			return e
		}
		if !ok {
			return foundation.Fail(403, "M03_REFERENCE_FORBIDDEN", "无权读取该日期的职级候选")
		}
		grades, e := u.ReferenceGrades(ctx)
		if e != nil {
			return e
		}
		byID := make(map[string]d.Timeline, len(grades))
		for _, g := range grades {
			byID[g.ID] = g
		}
		// Redating the selected record can enlarge a predecessor's interval.
		// Validate every other final interval before advertising any candidate.
		for _, r := range p.After.Records {
			if r.ID == p.Selected || !r.Active() {
				continue
			}
			for _, m := range r.Grades {
				g := byID[m.GradeID]
				if g.Enterprise != ent.ID || !d.Covers(g, r.Date, r.End) {
					return foundation.Fail(409, "M03_REFERENCE_CONFLICT", "拟定日期与现有职级引用期间冲突，请先处理相关资料")
				}
			}
		}
		gr, e := allowed(ctx, u, d.Grade, ent.ID, "read")
		if e != nil {
			return e
		}
		codeAllowed := has(gr.ReadFields, "code")
		if selected.Date != today(u) {
			hd, e := allowed(ctx, u, d.Grade, ent.ID, "history.read")
			if e != nil {
				return e
			}
			codeAllowed = codeAllowed && has(hd.ReadFields, "code")
		}
		if !codeAllowed {
			// Hidden fields cannot determine observable candidate pagination.
			slices.SortFunc(grades, func(a, b d.Timeline) int { return strings.Compare(a.ID, b.ID) })
		}
		items := []s.View{}
		total, first := 0, (page-1)*size
		resolved := []s.GradeMemberView{}
		invalidSelected := false
		selectedIDs := selected.Grades.IDs()
		selectedSet := make(map[string]bool, len(selectedIDs))
		for _, id := range selectedIDs {
			selectedSet[id] = true
		}
		q := strings.ToLower(strings.TrimSpace(cmd.Input.ReferenceQ))
		for _, g := range grades {
			i := d.At(g, selected.Date)
			valid := i >= 0 && g.Enterprise == ent.ID && d.Covers(g, selected.Date, selected.End) && p.After.Records[0].Date >= ent.ValidFrom
			// A member inherited by following stops has no active obligation there;
			// independent successors and the remaining selected members stay untouched.
			if selectedSet[g.ID] && !valid {
				invalidSelected = true
			}
			if selectedSet[g.ID] && i >= 0 {
				v := s.GradeMemberView{GradeID: g.ID, Name: g.Records[i].Data["name"], DisplayDate: selected.Date}
				if codeAllowed {
					code := g.Code
					v.Code = &code
				}
				resolved = append(resolved, v)
			}
			if !valid {
				continue
			}
			name := g.Records[i].Data["name"]
			if !strings.Contains(strings.ToLower(name), q) && !(codeAllowed && strings.Contains(strings.ToLower(g.Code), q)) {
				continue
			}
			total++
			if total <= first || len(items) >= size {
				continue
			}
			v := s.View{"id": g.ID, "name": name, "date": selected.Date}
			if codeAllowed {
				v["code"] = g.Code
			}
			items = append(items, v)
		}
		// Preserve the proposed order, including for candidate refreshes after dates
		// change. Unknown/unavailable selected members are explicitly unresolved.
		ordered := []s.GradeMemberView{}
		unresolved := invalidSelected
		for _, id := range selectedIDs {
			i := slices.IndexFunc(resolved, func(v s.GradeMemberView) bool { return v.GradeID == id })
			if i < 0 {
				unresolved = true
				continue
			}
			v := resolved[i]
			if k == d.Ladder {
				position := len(ordered) + 1
				v.Position = &position
			}
			ordered = append(ordered, v)
		}
		out = s.View{"applicable": true, "items": items, "totalCount": total, "page": page, "pageSize": size, "businessAt": today(u), "selected": ordered, "selectedUnavailable": unresolved, "effectiveDate": selected.Date, "endExclusive": selected.End}
		return nil
	})
	return
}

package application

import (
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	s "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"slices"
)

func gradeRead(ctx context.Context, u Unit, enterprise, date string, exact bool) (bool, error) {
	required := []string{"name", "effectiveDate", "status"}
	rd, e := allowed(ctx, u, d.Grade, enterprise, "read")
	if e != nil {
		return false, e
	}
	if !rd.Allowed || !has(rd.ReadFields, required...) {
		return false, nil
	}
	if exact || date != today(u) {
		hd, e := allowed(ctx, u, d.Grade, enterprise, "history.read")
		if e != nil {
			return false, e
		}
		if !hd.Allowed || !has(hd.ReadFields, required...) {
			return false, nil
		}
	}
	return true, nil
}

// Only collections involved in the replacement require complete source reads.
// Unchanged independent records keep their adoption and authorization boundary.
func requireReplacementReadable(ctx context.Context, u Unit, p d.Plan, explicit bool) error {
	check := func(t, other d.Timeline, after bool) error {
		for _, r := range t.Records {
			i := d.Index(other, r.ID)
			affected := i < 0 || !r.Grades.EqualData(other.Records[i].Grades)
			if !affected && i == 0 && r.Date != other.Records[0].Date {
				// Moving the first date changes where a nonempty collection begins.
				affected = true
			}
			if !affected && (r.Date != other.Records[i].Date || r.End != other.Records[i].End) {
				// A moved boundary can replace another record's members without
				// changing either record's own collection.
				affected = slices.ContainsFunc(other.Records, func(peer d.Record) bool {
					overlap := (r.End == "" || peer.Date < r.End) && (peer.End == "" || r.Date < peer.End)
					return overlap && !r.Grades.EqualData(peer.Grades)
				})
			}
			if explicit && after && r.ID == p.Selected {
				affected = true
			}
			if explicit && !after {
				j := d.Index(p.After, p.Selected)
				if j >= 0 && d.At(t, p.After.Records[j].Date) == d.Index(t, r.ID) {
					affected = true
				}
			}
			if !affected || len(r.Grades) == 0 {
				continue
			}
			date := r.Date
			if !r.Active() {
				date = dayBefore(date)
			}
			ok, e := gradeRead(ctx, u, t.Enterprise, date, false)
			if e != nil {
				return e
			}
			if !ok {
				return foundation.Fail(403, "M03_REFERENCE_FORBIDDEN", "须能完整读取职级资料后才能替换成员")
			}
		}
		return nil
	}
	if e := check(p.Before, p.After, false); e != nil {
		return e
	}
	return check(p.After, p.Before, true)
}
func prepareGrades(ctx context.Context, u Unit, p *d.Plan, c d.Command) error {
	k := p.After.Kind
	if k == d.Grade && d.CoverageChanged(p.Before, p.After) {
		consumers, e := u.GradeConsumerHeads(ctx)
		if e != nil {
			return e
		}
		for _, consumer := range consumers {
			for _, r := range consumer.Records {
				if !r.Active() {
					continue
				}
				for _, m := range r.Grades {
					if m.GradeID == p.After.ID && !d.Covers(p.After, r.Date, r.End) {
						return foundation.Fail(409, "M03_REFERENCE_CONFLICT", "现有引用的启用期间超出候选职级范围，请先处理相关资料")
					}
				}
			}
		}
		return nil
	}
	if !d.HasGrades(k) {
		return nil
	}
	if c.Grades.Present || slices.Contains(p.Fields, "gradeIds") {
		rd, e := decide(ctx, u, k, p.After.Enterprise, "read", nil)
		if e != nil {
			return e
		}
		if !has(rd.ReadFields, "gradeIds") {
			return foundation.Fail(403, "M03_FIELD_FORBIDDEN", "须能完整读取职级成员后才能替换")
		}
		if e = requireReplacementReadable(ctx, u, *p, c.Grades.Present); e != nil {
			return e
		}
	}
	grades := map[string]d.Timeline{}
	ids := d.GradeIDs(p.After)
	if len(ids) > 0 {
		heads, e := u.Heads(ctx, d.Grade, ids)
		if e != nil {
			return e
		}
		for _, g := range heads {
			grades[g.ID] = g
		}
	}
	if e := d.ValidateGradeCoverage(p.After, grades); e != nil {
		return e
	}
	for i := range p.After.Records {
		r := &p.After.Records[i]
		if !r.Active() {
			r.Grades = p.After.Records[i-1].Grades.Clone()
			continue
		}
		for j := range r.Grades {
			m := &r.Grades[j]
			if m.Adopted.SourceVersion != "" {
				continue
			}
			ok, e := gradeRead(ctx, u, p.After.Enterprise, r.Date, false)
			if e != nil {
				return e
			}
			if !ok {
				return foundation.Fail(403, "M03_REFERENCE_FORBIDDEN", "无权采用该日期的职级资料")
			}
			grade := grades[m.GradeID]
			at := d.At(grade, r.Date)
			if at < 0 {
				return foundation.Fail(409, "M03_REFERENCE_CONFLICT", "职级在采用日期不可用")
			}
			gr := grade.Records[at]
			m.Adopted = d.ReferenceAdoption{BusinessAt: r.Date, SourceVersion: grade.Source, RecordID: gr.ID, Revision: gr.Revision}
		}
	}
	return nil
}

func grade(ctx context.Context, u Unit, id, source string) (d.Timeline, error) {
	cache, _ := ctx.Value(cacheKey{}).(map[string]d.Timeline)
	key := "grade:" + id + ":" + source
	if t, ok := cache[key]; ok {
		return t, nil
	}
	t, e := u.Load(ctx, d.Grade, id, source)
	if e == nil && cache != nil {
		cache[key] = t
	}
	return t, e
}
func memberViews(ctx context.Context, u Unit, t d.Timeline, r d.Record, date string, exact bool) ([]s.GradeMemberView, bool, error) {
	out := []s.GradeMemberView{}
	if len(r.Grades) == 0 {
		return out, true, nil
	}
	ok, e := gradeRead(ctx, u, t.Enterprise, date, exact)
	if e != nil || !ok {
		return nil, false, e
	}
	rd, e := allowed(ctx, u, d.Grade, t.Enterprise, "read")
	if e != nil {
		return nil, false, e
	}
	codeAllowed := has(rd.ReadFields, "code")
	if exact || date != today(u) {
		hd, e := allowed(ctx, u, d.Grade, t.Enterprise, "history.read")
		if e != nil {
			return nil, false, e
		}
		codeAllowed = codeAllowed && has(hd.ReadFields, "code")
	}
	// Batch current source facts once per request snapshot. Qualification and
	// exact adoption remain Go decisions; no cache survives a transaction.
	if !exact {
		cache, _ := ctx.Value(cacheKey{}).(map[string]d.Timeline)
		if cache == nil {
			ctx = readContext(ctx)
			cache, _ = ctx.Value(cacheKey{}).(map[string]d.Timeline)
		}
		missing := []string{}
		for _, m := range r.Grades {
			if _, ok := cache["grade:"+m.GradeID+":"]; !ok {
				missing = append(missing, m.GradeID)
			}
		}
		if len(missing) > 0 {
			heads, e := u.Heads(ctx, d.Grade, missing)
			if e != nil {
				return nil, false, e
			}
			for _, id := range missing {
				cache["grade:"+id+":"] = d.Timeline{}
			}
			for _, g := range heads {
				cache["grade:"+g.ID+":"] = g
			}
		}
	}
	for i, m := range r.Grades {
		source := ""
		if exact {
			source = m.Adopted.SourceVersion
			if source == "" {
				return nil, false, foundation.Fail(503, "M03_DEPENDENCY_UNAVAILABLE", "职级采用依据暂不可读")
			}
		}
		g, e := grade(ctx, u, m.GradeID, source)
		if e != nil {
			return nil, false, e
		}
		at := d.At(g, date)
		if exact {
			at = d.Index(g, m.Adopted.RecordID)
		}
		if at < 0 || g.Enterprise != t.Enterprise || exact && g.Records[at].Revision != m.Adopted.Revision {
			return nil, false, foundation.Fail(503, "M03_DEPENDENCY_UNAVAILABLE", "职级引用资料暂不可读")
		}
		v := s.GradeMemberView{GradeID: m.GradeID, Name: g.Records[at].Data["name"], DisplayDate: date}
		if codeAllowed {
			code := g.Code
			v.Code = &code
		}
		if t.Kind == d.Ladder {
			position := i + 1
			v.Position = &position
		}
		if exact {
			a := m.Adopted
			v.Adopted = &s.GradeAdoptedRef{BusinessAt: a.BusinessAt, SourceVersion: a.SourceVersion, RecordID: a.RecordID, Revision: a.Revision}
			v.DisplayDate = a.BusinessAt
		}
		out = append(out, v)
	}
	return out, true, nil
}
func projectMembers(ctx context.Context, u Unit, t d.Timeline, r d.Record, v s.View, fields []string, date string, exact bool) error {
	if !d.HasGrades(t.Kind) || !has(fields, "gradeIds") || r.Grades == nil {
		return nil
	}
	members, visible, e := memberViews(ctx, u, t, r, date, exact)
	if e != nil {
		return e
	}
	if !visible {
		v["gradesRestricted"] = true
		return nil
	}
	v["gradeIds"] = r.Grades.IDs()
	v["gradeMembers"] = members
	return nil
}

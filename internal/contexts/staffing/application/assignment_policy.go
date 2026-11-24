package application

import (
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	s "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	f "chnhcm.local/chnhcm/internal/foundation"
	"context"
	"slices"
	"strings"
	"unicode/utf8"
)

func ChangePolicy(rows []s.AssignmentPolicy, change f.RuleChange[s.AssignmentPolicy], enterpriseStart string) ([]s.AssignmentPolicy, bool, error) {
	row := change.Row

	row.Category = strings.TrimSpace(row.Category)
	row.Purpose = strings.TrimSpace(row.Purpose)
	if !row.Period.Valid() || !f.ValidID(row.ID) || row.Category == "" || row.Purpose == "" || utf8.RuneCountInString(row.Category) > 60 || utf8.RuneCountInString(row.Purpose) > 60 || !slices.Contains([]string{"JobOnly", "PositionExempt", "PositionCounted"}, row.Mode) {
		return nil, false, f.Fail(400, "M03_INVALID_INPUT", "请核对人员类别、用途、期间和任职模式")
	}
	if row.From < enterpriseStart {
		return nil, false, f.Fail(409, "M03_POLICY_PERIOD", "企业从 "+enterpriseStart+" 起生效")
	}
	if change.InsertAfter != "" {
		anchor := slices.IndexFunc(rows, func(r s.AssignmentPolicy) bool { return r.ID == change.InsertAfter })
		if anchor < 0 || row.Revision != "" || slices.ContainsFunc(rows, func(r s.AssignmentPolicy) bool { return r.ID == row.ID }) {
			return nil, false, f.Fail(409, "RULE_ANCHOR", "所选期间已变化，请重新选择")
		}
		old := rows[anchor]
		cut, ok := old.Period.Predecessor(row.Period)
		if !ok || old.Category != row.Category || old.Purpose != row.Purpose {
			return nil, false, f.Fail(409, "RULE_ANCHOR", "新段须在原段内生效并承接同一作用范围和结束边界")
		}
		rows[anchor].Period = cut
		rows[anchor].Revision = f.NewID()
	}
	index := -1
	for i, other := range rows {
		if other.ID == row.ID {
			if other.Revision != row.Revision {
				return nil, false, f.Fail(409, "M03_SOURCE_CHANGED", "所选规则已更新")
			}
			if other == row {
				return rows, false, nil
			}
			index = i
			continue
		}
		if other.Category == row.Category && other.Purpose == row.Purpose && other.Overlaps(row.Period) {
			return nil, false, f.Fail(409, "M03_POLICY_CONFLICT", max(other.From, row.From)+" 起同一类别和用途已有规则")
		}
	}
	if index < 0 && row.Revision != "" {
		return nil, false, f.Fail(409, "M03_SOURCE_CHANGED", "所选规则不存在")
	}
	row.Revision = f.NewID()
	if index < 0 {
		return append(rows, row), true, nil
	}
	rows[index] = row
	return rows, true, nil
}
func QuotaPolicies(enterprise string, rows []s.AssignmentPolicy) []d.QuotaPolicy {
	var rules []d.QuotaPolicy
	for _, r := range rows {
		mode := "PositionManaged"
		if r.Mode == "JobOnly" {
			mode = "JobOnly"
		}
		rules = append(rules, d.QuotaPolicy{UsageSpan: d.UsageSpan{From: r.From, Until: r.Until}, Enterprise: enterprise, Category: r.Category, Purpose: r.Purpose, Version: r.Revision, Mode: mode, Consumes: r.Mode == "PositionCounted"})
	}
	return rules
}
func CheckJobAssignment(ctx context.Context, u Unit, in s.AssignmentCheck, head string, rows []s.AssignmentPolicy, locked bool) ([]s.AssignmentSegment, error) {
	if !in.Period.Valid() || !f.ValidID(in.Job) || !f.ValidID(in.Department) || in.Category == "" || in.Purpose == "" {
		return nil, f.Fail(400, "M03_INVALID_INPUT", "请核对拟任期间、人员类别、用途、部门及职务")
	}
	rules := QuotaPolicies(in.Enterprise, rows)
	// Reject an unavailable starting policy before loading its targets. Later
	// boundaries are checked in date order with department and job coverage.
	var days []string
	for _, rule := range rules {
		if rule.Category == in.Category && rule.Purpose == in.Purpose {
			days = append(days, rule.From, rule.Until)
		}
	}
	checkMode := func(day string) (*d.QuotaPolicy, error) {
		rule, e := d.ApplicablePolicy(rules, in.Enterprise, in.Category, in.Purpose, day)
		if e != nil {
			return nil, e
		}
		if rule.Mode != "JobOnly" {
			return nil, f.Fail(409, "M03_POSITION_PATH_UNAVAILABLE", day+" 起适用按岗位规则，补员及实际用量尚未接入")
		}
		return rule, nil
	}
	if _, e := checkMode(in.From); e != nil {
		return nil, e
	}
	if in.Position != "" {
		return nil, f.Fail(409, "M03_POLICY_CONFLICT", "仅按职务不能指定岗位")
	}
	targets := make([]d.Timeline, 0, 2)
	for _, ref := range []struct{ field, id string }{{"departmentId", in.Department}, {"jobId", in.Job}} {
		found, e := u.PositionSources(ctx, ref.field, []string{ref.id}, locked)
		if e != nil {
			return nil, e
		}
		if len(found) != 1 {
			return nil, f.Fail(404, "M03_REFERENCE_UNAVAILABLE", "部门或职务不可用")
		}
		visible, e := positionRead(ctx, u, ref.field, found[0], in.From, true, false)
		if e != nil {
			return nil, e
		}
		if !visible {
			return nil, f.Fail(403, "ACCESS_DENIED", "当前权限无法核对所选部门或职务")
		}
		t := found[0].Timeline
		for _, r := range t.Records {
			days = append(days, r.Date, r.End)
		}
		targets = append(targets, t)
	}
	var out []s.AssignmentSegment
	for _, p := range in.Split(days) {
		rule, e := checkMode(p.From)
		if e != nil {
			return nil, e
		}
		dep, job := targets[0], targets[1]
		di, ji := d.At(dep, p.From), d.At(job, p.From)
		if di < 0 || !dep.Records[di].Active() {
			return nil, positionFault("departmentId", p.From)
		}
		if ji < 0 || !job.Records[ji].Active() {
			return nil, positionFault("jobId", p.From)
		}
		out = append(out, s.AssignmentSegment{Period: p, DepartmentName: dep.Records[di].Data["name"], JobName: job.Records[ji].Data["name"], PolicyVersion: head, Policy: rule.Version, DepartmentSource: dep.Source, DepartmentRevision: dep.Records[di].Revision, JobSource: job.Source, JobRevision: job.Records[ji].Revision})
	}
	return out, nil
}

// HireTargetLabel uses the same reference read permissions and dated timeline
// as submission. Return an empty label for a hidden or ineligible candidate.
func HireTargetLabel(ctx context.Context, u Unit, field string, source PositionSource, term f.Period) (string, error) {
	visible, err := positionRead(ctx, u, field, source, term.From, true, false)
	if err != nil || !visible {
		return "", err
	}
	if d.PositionConflict(source.Timeline, term.From, term.Until, "") != "" {
		return "", nil
	}
	return source.Records[d.At(source.Timeline, term.From)].Data["name"], nil
}

// JobGradeVisible checks the job membership fields required to adopt a grade.
func JobGradeVisible(ctx context.Context, u Unit, source PositionSource, day string) (bool, error) {
	return positionRead(ctx, u, "jobId", source, day, true, true)
}

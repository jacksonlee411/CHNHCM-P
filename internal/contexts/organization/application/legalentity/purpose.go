package legalentity

import (
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	f "chnhcm.local/chnhcm/internal/foundation"
	"slices"
	"strings"
	"unicode/utf8"
)

var roles = []string{"employer", "receiver", "signatory", "payroll", "withholding", "contribution", "payment", "cost"}

func ValidRole(role string) bool { return slices.Contains(roles, role) }
func ChangePurpose(rows []org.LegalPurpose, change f.RuleChange[org.LegalPurpose]) ([]org.LegalPurpose, bool, error) {
	row := change.Row

	row.Basis = strings.TrimSpace(row.Basis)
	if !row.Period.Valid() || !ValidRole(row.Role) || !f.ValidID(row.ID) || row.Basis == "" || utf8.RuneCountInString(row.Basis) > 500 {
		return nil, false, f.Fail(400, "M02_INVALID_INPUT", "请核对用途、有效期间及资格依据")
	}
	switch row.Kind {
	case "declaration":
		if row.Target != "" {
			return nil, false, f.Fail(400, "M02_INVALID_INPUT", "用途声明不能指定绑定目标")
		}
	case "bu", "location":
		if !f.ValidID(row.Target) || row.RequireBU || row.RequireLocation {
			return nil, false, f.Fail(400, "M02_INVALID_INPUT", "请选择有效的绑定目标")
		}
	default:
		return nil, false, f.Fail(400, "M02_INVALID_INPUT", "请选择用途声明或绑定类型")
	}
	if change.InsertAfter != "" {
		anchor := slices.IndexFunc(rows, func(r org.LegalPurpose) bool { return r.ID == change.InsertAfter })
		if anchor < 0 || row.Revision != "" || slices.ContainsFunc(rows, func(r org.LegalPurpose) bool { return r.ID == row.ID }) {
			return nil, false, f.Fail(409, "RULE_ANCHOR", "所选期间已变化，请重新选择")
		}
		old := rows[anchor]
		cut, ok := old.Period.Predecessor(row.Period)
		if !ok || old.Role != row.Role || old.Kind != row.Kind || old.Target != row.Target {
			return nil, false, f.Fail(409, "RULE_ANCHOR", "新段须在原段内生效并承接同一作用范围和结束边界")
		}
		rows[anchor].Period = cut
		rows[anchor].Revision = f.NewID()
	}
	index := -1
	for i, other := range rows {
		if row.ID == other.ID {
			if row.Revision != other.Revision {
				return nil, false, f.Fail(409, "M02_SOURCE_CHANGED", "所选期间已更新")
			}
			if row == other {
				return rows, false, nil
			}
			index = i
			continue
		}
		if other.Role == row.Role && other.Kind == row.Kind && other.Target == row.Target && other.Overlaps(row.Period) {
			return nil, false, f.Fail(409, "M02_PURPOSE_CONFLICT", max(other.From, row.From)+" 起已有相同用途或绑定")
		}
	}
	if index < 0 && row.Revision != "" {
		return nil, false, f.Fail(409, "M02_SOURCE_CHANGED", "所选记录不存在")
	}
	row.Revision = f.NewID()
	if index < 0 {
		return append(rows, row), true, nil
	}
	rows[index] = row
	return rows, true, nil
}

type ActiveRecord struct {
	f.Period
	Revision string
	Active   bool
}

func ActiveAt(records []ActiveRecord, day string) string {
	for _, r := range records {
		if r.Includes(day) && r.Active {
			return r.Revision
		}
	}
	return ""
}
func CoverActive(period f.Period, records []ActiveRecord, label string) error {
	var days []string
	for _, r := range records {
		days = append(days, r.From, r.Until)
	}
	for _, p := range period.Split(days) {
		if ActiveAt(records, p.From) == "" {
			return f.Fail(409, "M02_PURPOSE_GAP", p.From+" 起"+label+"未启用")
		}
	}
	return nil
}
func ResolvePurpose(in org.PurposeCheck, entitySource, version string, records []ActiveRecord, rows []org.LegalPurpose) ([]org.LegalAdoption, error) {
	if !ValidRole(in.Role) || !in.Period.Valid() {
		return nil, f.Fail(400, "M02_INVALID_INPUT", "用途或拟任期间无效")
	}
	if in.Expected != "" && in.Expected != version {
		return nil, f.Fail(409, "M02_SOURCE_CHANGED", "用途已更新，请重新核对")
	}
	var days []string
	for _, r := range records {
		days = append(days, r.From, r.Until)
	}
	for _, r := range rows {
		if r.Role == in.Role && (r.Kind == "declaration" || r.Kind == "bu" && r.Target == in.BU || r.Kind == "location" && r.Target == in.Location) {
			days = append(days, r.From, r.Until)
		}
	}
	var out []org.LegalAdoption
	for _, p := range in.Split(days) {
		a := org.LegalAdoption{Period: p, EntityVersion: entitySource, EntityRevision: ActiveAt(records, p.From), PurposeVersion: version, Bindings: []string{}}
		if a.EntityRevision == "" {
			return nil, f.Fail(409, "M02_PURPOSE_GAP", p.From+" 起法律实体未启用")
		}
		var requiredBU, requiredLocation bool
		var bu, location string
		for _, r := range rows {
			if r.Role != in.Role || !r.Includes(p.From) {
				continue
			}
			switch r.Kind {
			case "declaration":
				if a.Declaration != "" {
					return nil, f.Fail(409, "M02_PURPOSE_CONFLICT", p.From+" 起用途声明冲突")
				}
				a.Declaration = r.Revision
				requiredBU = r.RequireBU
				requiredLocation = r.RequireLocation
			case "bu":
				if r.Target == in.BU {
					bu = r.Revision
				}
			case "location":
				if r.Target == in.Location {
					location = r.Revision
				}
			}
		}
		if a.Declaration == "" {
			return nil, f.Fail(409, "M02_PURPOSE_GAP", p.From+" 起缺少所选用途资格")
		}
		if requiredBU {
			if bu == "" {
				return nil, f.Fail(409, "M02_PURPOSE_GAP", p.From+" 起缺少适用的业务单元绑定")
			}
			a.Bindings = append(a.Bindings, bu)
		}
		if requiredLocation {
			if location == "" {
				return nil, f.Fail(409, "M02_PURPOSE_GAP", p.From+" 起缺少适用的地点绑定")
			}
			a.Bindings = append(a.Bindings, location)
		}
		// An optional binding can change without affecting the adopted qualification.
		if n := len(out); n > 0 && out[n-1].EntityRevision == a.EntityRevision && out[n-1].Declaration == a.Declaration && slices.Equal(out[n-1].Bindings, a.Bindings) {
			out[n-1].Until = a.Until
		} else {
			out = append(out, a)
		}
	}
	return out, nil
}

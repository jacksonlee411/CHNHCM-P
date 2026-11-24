package domain

import (
	"slices"
	"strconv"

	"chnhcm.local/chnhcm/internal/foundation"
)

// UsageSpan is a business-date interval [From, Until). An empty Until is open.
type UsageSpan struct{ From, Until string }

func (s UsageSpan) Contains(day string) bool {
	return s.From != "" && s.From <= day && (s.Until == "" || day < s.Until)
}

type SourceCoverage struct {
	UsageSpan
	Owner, Version string
}

type PositionUse struct {
	UsageSpan
	Owner, Source, Policy, Effect string
	Department, Job, Assignment   string
}

// Managed identifies the explicitly taken-over part of this position's timeline.
// Each owner attests its own coverage separately, including a verified zero.
type QuotaUsage struct {
	Managed  []UsageSpan
	Coverage []SourceCoverage
	Uses     []PositionUse
}

type UsageCount struct {
	Managed, Known             bool
	Reserved, Occupied, Exempt int
}

func (u QuotaUsage) coveredBy(owner, day string) bool {
	var count int
	for _, s := range u.Coverage {
		if s.Owner == owner && s.Version != "" && s.Contains(day) {
			count++
		}
	}
	return count == 1
}

func (u QuotaUsage) At(day string) UsageCount {
	c := UsageCount{Managed: slices.ContainsFunc(u.Managed, func(s UsageSpan) bool { return s.Contains(day) })}
	if !c.Managed {
		return c
	}
	c.Known = u.coveredBy("M05", day) && u.coveredBy("M04", day)
	for _, use := range u.Uses {
		if !use.Contains(day) {
			continue
		}
		switch {
		case use.Source == "" || use.Policy == "":
			c.Known = false
		case use.Owner == "M05" && use.Effect == "reserved":
			c.Reserved++
		case use.Owner == "M05" && use.Effect == "none":
			// A recruitment segment can adopt an exempt rule without reserving a seat.
			continue
		case use.Owner == "M04" && use.Effect == "occupied":
			c.Occupied++
		case use.Owner == "M04" && use.Effect == "exempt":
			c.Exempt++
		default:
			c.Known = false
		}
	}
	return c
}

func (u QuotaUsage) boundaries() []string {
	var days []string
	for _, s := range u.Managed {
		days = append(days, s.From, s.Until)
	}
	for _, s := range u.Coverage {
		days = append(days, s.From, s.Until)
	}
	for _, s := range u.Uses {
		days = append(days, s.From, s.Until)
	}
	return days
}

func capacity(q StaffingQuota, day string) int {
	if i := q.At(day); i >= 0 && q.Records[i].Amount != nil {
		return *q.Records[i].Amount
	}
	return -1
}

func activeOn(p QuotaPeriods, day string) bool {
	i := p.At(day)
	return i >= 0 && p.Records[i].Active
}

func usageConflict(day, message string) error {
	return foundation.Fail(409, "M03_QUOTA_CONFLICT", day+" "+message)
}

// CheckTightening evaluates only intervals made stricter by this operation.
func (u QuotaUsage) CheckTightening(before, after StaffingQuota, previous, proposed QuotaPeriods) error {
	if len(u.Managed) == 0 {
		return nil
	}
	days := u.boundaries()
	for _, q := range []StaffingQuota{before, after} {
		for _, r := range q.Records {
			days = append(days, r.Date)
		}
	}
	for _, p := range []QuotaPeriods{previous, proposed} {
		for _, r := range p.Records {
			days = append(days, r.Date, r.End)
		}
	}
	slices.Sort(days)
	for _, day := range slices.Compact(days) {
		old, next := capacity(before, day), capacity(after, day)
		stop := activeOn(previous, day) && !activeOn(proposed, day)
		if !stop && next >= old {
			continue
		}
		c := u.At(day)
		if !c.Managed {
			continue
		}
		if !c.Known {
			return usageConflict(day, "岗位用量尚未核齐")
		}
		if stop && c.Reserved+c.Occupied+c.Exempt > 0 {
			return usageConflict(day, "仍有预留或任职，无法停用岗位")
		}
		if next < old && c.Reserved+c.Occupied > max(0, next) {
			return usageConflict(day, "核定数不足以覆盖已预留和已占名额")
		}
	}
	return nil
}

// QuotaPolicy is the exact dated rule supplied by the policy owner, scoped to
// enterprise, person category and purpose. Its version is adopted by each segment.
type QuotaPolicy struct {
	UsageSpan
	Enterprise, Category, Purpose, Version, Mode string
	Consumes                                     bool
}

type UsageIntent struct {
	UsageSpan
	Enterprise, Category, Purpose, Owner string
}

func (u QuotaUsage) PlanUse(intent UsageIntent, rules []QuotaPolicy, q StaffingQuota, position QuotaPeriods) ([]PositionUse, error) {
	if !foundation.ValidDate(intent.From) || intent.Until != "" && (!foundation.ValidDate(intent.Until) || intent.From >= intent.Until) ||
		intent.Enterprise == "" || intent.Category == "" || intent.Purpose == "" || (intent.Owner != "M04" && intent.Owner != "M05") {
		return nil, foundation.Fail(400, "M03_INVALID_INPUT", "用量来源、用途或期间无效")
	}
	days := append(u.boundaries(), intent.From)
	var applicable []QuotaPolicy
	for _, rule := range rules {
		if rule.Enterprise == intent.Enterprise && rule.Category == intent.Category && rule.Purpose == intent.Purpose {
			applicable = append(applicable, rule)
			days = append(days, rule.From, rule.Until)
		}
	}
	for _, r := range q.Records {
		days = append(days, r.Date)
	}
	for _, r := range position.Records {
		days = append(days, r.Date, r.End)
	}
	slices.Sort(days)
	var uses []PositionUse
	for _, day := range slices.Compact(days) {
		if !intent.Contains(day) {
			continue
		}
		chosen, err := ApplicablePolicy(applicable, intent.Enterprise, intent.Category, intent.Purpose, day)
		if err != nil {
			return nil, err
		}
		if len(uses) > 0 {
			uses[len(uses)-1].Until = day
		}
		use := PositionUse{UsageSpan: UsageSpan{day, intent.Until}, Owner: intent.Owner, Policy: chosen.Version}
		switch chosen.Mode {
		case "JobOnly":
			if chosen.Consumes || position.ID != "" {
				return nil, usageConflict(day, "仅按职务的安排不能绑定岗位名额")
			}
			use.Effect = "none"
		case "PositionManaged":
			if position.ID == "" || !activeOn(position, day) {
				return nil, usageConflict(day, "安排期间没有有效岗位")
			}
			if !u.At(day).Managed || !u.coveredBy(intent.Owner, day) {
				return nil, usageConflict(day, "该来源尚未接管本岗位的办理期间")
			}
			switch {
			case !chosen.Consumes && intent.Owner == "M04":
				use.Effect = "exempt"
			case !chosen.Consumes:
				use.Effect = "none"
			default:
				c := u.At(day)
				if !c.Known {
					return nil, usageConflict(day, "岗位用量尚未核齐")
				}
				limit := capacity(q, day)
				if limit < 0 {
					return nil, usageConflict(day, "该岗位尚未核定常规编制")
				}
				if shortage := c.Occupied + c.Reserved + 1 - limit; shortage > 0 {
					return nil, foundation.Fail(409, "M03_QUOTA_SHORTAGE", day+" 起缺 "+strconv.Itoa(shortage)+" 编")
				}
				if intent.Owner == "M04" {
					use.Effect = "occupied"
				} else {
					use.Effect = "reserved"
				}
			}
		default:
			return nil, usageConflict(day, "任职模式无效")
		}
		// Keep the adopted rule even for an interval that consumes no seat.
		if n := len(uses); n > 0 && uses[n-1].Policy == use.Policy && uses[n-1].Effect == use.Effect {
			uses[n-1].Until = intent.Until
		} else {
			uses = append(uses, use)
		}
	}
	return uses, nil
}

func ApplicablePolicy(rules []QuotaPolicy, enterprise, category, purpose, day string) (*QuotaPolicy, error) {
	var chosen *QuotaPolicy
	for i := range rules {
		if rules[i].Enterprise == enterprise && rules[i].Category == category && rules[i].Purpose == purpose && rules[i].Contains(day) {
			if chosen != nil {
				return nil, usageConflict(day, "同一用途有多份适用的占编规则")
			}
			chosen = &rules[i]
		}
	}
	if chosen == nil || chosen.Version == "" {
		return nil, usageConflict(day, "缺少适用的占编规则版本")
	}
	return chosen, nil
}

// CheckExchange compares both published assignments per position. Third-party
// reservations and appointments enter the final count at every dated boundary.
func (u QuotaUsage) CheckExchange(before QuotaUsage, q StaffingQuota, first, second string) error {
	days := append(before.boundaries(), u.boundaries()...)
	for _, r := range q.Records {
		days = append(days, r.Date)
	}
	occupied := func(e QuotaUsage, id, day string) bool {
		return slices.ContainsFunc(e.Uses, func(v PositionUse) bool {
			return v.Owner == "M04" && v.Effect == "occupied" && v.Assignment == id && v.Contains(day)
		})
	}
	slices.Sort(days)
	for _, day := range slices.Compact(days) {
		if day == "" || !(occupied(u, first, day) && !occupied(before, first, day) || occupied(u, second, day) && !occupied(before, second, day)) {
			continue
		}
		c := u.At(day)
		if !c.Known {
			return usageConflict(day, "岗位用量尚未核齐")
		}
		limit := capacity(q, day)
		if limit < 0 {
			return usageConflict(day, "没有有效岗位核定数")
		}
		if deficit := c.Reserved + c.Occupied - limit; deficit > 0 {
			return foundation.Fail(409, "M03_QUOTA_SHORTAGE", day+" 起缺 "+strconv.Itoa(deficit)+" 编")
		}
	}
	return nil
}

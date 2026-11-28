package postgres

import (
	"context"
	"slices"
	"strconv"

	app "chnhcm.local/chnhcm/internal/contexts/compensation/application"
	d "chnhcm.local/chnhcm/internal/contexts/compensation/domain"
	cfg "chnhcm.local/chnhcm/internal/contexts/configuration/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

// Complete coverage is checked at the actual source boundaries, including an
// inactive interval in the middle and the final interval of an open-ended plan.
func catalogCoverage(source foundation.CatalogSource, p foundation.Period, field, label string) (foundation.CatalogSegment, error) {
	var first foundation.CatalogSegment
	cursor := p.From
	for _, seg := range source.Segments {
		if !seg.Includes(cursor) {
			continue
		}
		if !seg.Active {
			break
		}
		if first.RecordID == "" {
			first = seg
		}
		if seg.Until == "" || p.Until != "" && seg.Until >= p.Until {
			return first, nil
		}
		cursor = seg.Until
	}
	return first, d.Invalid(field, cursor+" 起"+label+"未覆盖计划期间")
}
func rateCoverage(rate app.RateState, p foundation.Period, field string) (int, error) {
	start := rate.Records.At(p.From)
	if start < 0 {
		return -1, d.Invalid(field, p.From+" 起职级标准没有有效记录")
	}
	for i := start; i < len(rate.Records); i++ {
		record := rate.Records[i]
		if p.Until != "" && record.Start >= p.Until {
			break
		}
		if !record.Active {
			return -1, d.Invalid(field, max(record.Start, p.From)+" 起职级标准停用")
		}
	}
	return start, nil
}
func policyChoices(p cfg.Policy) cfg.Scope {
	s := cfg.Scope{Mode: p.Mode}
	for _, m := range p.Members {
		s.Choices = append(s.Choices, m.Choice)
	}
	return s
}
func sourcePermission(dimension string) string {
	switch dimension {
	case "legalEntity":
		return "legalentity.read"
	case "jobFamily":
		return "jobfamily.read"
	case "grade":
		return "grade.read"
	}
	return ""
}
func (s *Store) scopeMembers(ctx context.Context, u *unit, spec cfg.Scope, period foundation.Period) ([]cfg.Member, error) {
	members := make([]cfg.Member, 0, len(spec.Choices))
	for _, choice := range spec.Choices {
		grant, err := u.require(ctx, sourcePermission(choice.Dimension), nil)
		if err != nil {
			return nil, err
		}
		if !has(grant.ReadFields, "name") {
			return nil, denied()
		}
		var source foundation.CatalogSource
		switch choice.Dimension {
		case "legalEntity":
			source, err = s.LegalEntities.CompensationLegalEntity(ctx, u.enterprise.ID, choice.ID, true)
		case "jobFamily":
			source, err = s.Grades.CompensationJobFamily(ctx, u.enterprise.ID, choice.ID, true)
		case "grade":
			source, err = s.Grades.CompensationGrade(ctx, u.enterprise.ID, choice.ID, true)
		default:
			return nil, d.Invalid("scope", "适用范围维度无效")
		}
		if err != nil {
			return nil, err
		}
		first, err := catalogCoverage(source, period, "scope", "所选适用对象")
		if err != nil {
			return nil, err
		}
		members = append(members, cfg.Member{Choice: choice, Code: source.Code, Name: first.Name, Source: source.Source, Record: first.RecordID, Revision: first.Revision})
	}
	return members, nil
}

// Only a change to the contractual rate mapping (or the scope/period) makes a
// correction depend on the upstream standards. Reordering rows and correcting a
// different allowance retain the exact rate originally adopted by this plan.
type componentRate struct {
	row, grade, rate, currency, frequency, unit string
	precision                                   int
}

func componentRates(rows []d.Component) map[componentRate]bool {
	out := make(map[componentRate]bool)
	for _, row := range rows {
		for _, link := range row.Rates {
			out[componentRate{row.ID, link.GradeID, link.RateID, row.Currency, row.Frequency, row.Unit, row.Precision}] = true
		}
	}
	return out
}

func planRateIDs(records d.Timeline[d.PlanTerms]) []string {
	var ids []string
	for _, r := range records {
		for _, row := range r.Data.Components {
			for _, link := range row.Rates {
				ids = append(ids, link.RateID)
			}
		}
	}
	slices.Sort(ids)
	return slices.Compact(ids)
}
func cloneComponents(rows []d.Component) []d.Component {
	out := slices.Clone(rows)
	for i, r := range out {
		out[i].Rates = slices.Clone(r.Rates)
	}
	return out
}
func (u *unit) requirePlanRateReference(ctx context.Context) error {
	grant, err := u.require(ctx, "graderate.reference", nil)
	if err != nil {
		return err
	}
	if !planRateFields(grant.ReadFields) {
		return denied()
	}
	return nil
}

func planRateFields(fields []string) bool {
	for _, field := range []string{"gradeId", "currency", "frequency", "unit", "precision"} {
		if !has(fields, field) {
			return false
		}
	}
	return true
}

func (s *Store) verifyComponents(ctx context.Context, u *unit, spec cfg.Scope, rows []d.Component, period foundation.Period, rates map[string]app.RateState, adopt bool) error {
	for ix := range rows {
		row := &rows[ix]
		if row.Method != "GRADE_RATE" {
			continue
		}
		field := "components." + strconv.Itoa(ix) + ".rates"
		if err := u.requirePlanRateReference(ctx); err != nil {
			return err
		}
		chosen := map[string]bool{}
		for i := range row.Rates {
			link := &row.Rates[i]
			state := rates[link.RateID]
			if state.ID == "" || state.Key.GradeID != link.GradeID || state.Key.Currency != row.Currency || state.Key.Frequency != row.Frequency || state.Key.Unit != row.Unit || state.Places != row.Precision {
				return d.Invalid(field, "所选标准与职级、币种、频率、单位或精度不一致")
			}
			if !s.Scopes.PermitsDimension(spec, "grade", link.GradeID) {
				return d.Invalid(field, "所选职级不在计划的适用范围内")
			}
			chosen[link.GradeID] = true
			start, err := rateCoverage(state, period, field)
			if err != nil {
				return err
			}
			grade, err := s.Grades.CompensationGrade(ctx, u.enterprise.ID, link.GradeID, true)
			if err != nil {
				return err
			}
			if _, err = catalogCoverage(grade, period, field, "所选职级"); err != nil {
				return err
			}
			if adopt {
				r := state.Records[start]
				var version string
				err = u.tx.QueryRow(ctx, "SELECT id::text FROM compensation.grade_rate_version WHERE instance_id=$1 AND rate_id=$2 AND record_id=$3 AND revision=$4 AND action<>'DELETE'", s.Instance, state.ID, r.ID, r.Revision).Scan(&version)
				if err != nil {
					return err
				}
				link.Adopted = d.AdoptedRate{BusinessAt: period.From, Head: state.Version, Source: version, Record: r.ID, Revision: r.Revision}
			}
		}
		for _, choice := range spec.Choices {
			if choice.Dimension == "grade" && !choice.Exclude && s.Scopes.PermitsDimension(spec, "grade", choice.ID) && !chosen[choice.ID] {
				return d.Invalid(field, "请为适用范围中的每个职级选择标准")
			}
		}
	}
	return nil
}

// The caller holds the rate row lock. A plan changing its references takes
// the same rate locks, in stable order, before updating the live use periods.
func (s *Store) protectPlanUse(ctx context.Context, u *unit, state app.RateState, records d.Timeline[d.RateTerms]) error {
	result, err := u.tx.Query(ctx, `SELECT lower(valid_period)::text,coalesce(upper(valid_period)::text,'') FROM compensation.plan_rate_use WHERE instance_id=$1 AND rate_id=$2`, s.Instance, state.ID)
	if err != nil {
		return err
	}
	defer result.Close()
	state.Records = records
	for result.Next() {
		var p foundation.Period
		if err = result.Scan(&p.From, &p.Until); err != nil {
			return err
		}
		if _, e := rateCoverage(state, p, "effectiveDate"); e != nil {
			return d.Invalid("effectiveDate", "仍有薪酬计划在受影响期间采用这份标准，请先结束相应引用")
		}
	}
	return result.Err()
}

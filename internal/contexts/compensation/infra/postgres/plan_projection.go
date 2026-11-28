package postgres

import (
	"context"
	"encoding/json"
	"slices"

	d "chnhcm.local/chnhcm/internal/contexts/compensation/domain"
	c "chnhcm.local/chnhcm/internal/contexts/compensation/public"
	cfg "chnhcm.local/chnhcm/internal/contexts/configuration/public"
)

type planView struct {
	fields                                  []string
	rateFields                              []string
	rateHistoryFields                       []string
	rateReference, rateCurrent, rateHistory bool
}

func (u *unit) planView(ctx context.Context, historical bool) (v planView, err error) {
	base, err := u.requireObjectRead(ctx, planCommands)
	if err != nil {
		return v, err
	}
	v.fields = slices.Clone(base.ReadFields)
	if historical {
		history, e := u.require(ctx, "compplan.history.read", nil)
		if e != nil {
			return v, e
		}
		v.fields = slices.DeleteFunc(v.fields, func(k string) bool { return !has(history.ReadFields, k) })
	}
	if !has(v.fields, "components") || !has(v.fields, "rates") {
		return v, nil
	}
	ref, e := u.decision(ctx, "graderate.reference", nil)
	if e != nil {
		return v, e
	}
	v.rateReference = ref.Allowed && has(ref.ReadFields, "gradeId")
	read, e := u.decision(ctx, "graderate.read", nil)
	if e != nil {
		return v, e
	}
	v.rateCurrent = read.Allowed
	v.rateFields = read.ReadFields
	h, e := u.decision(ctx, "graderate.history.read", nil)
	if e != nil {
		return v, e
	}
	v.rateHistory = read.Allowed && h.Allowed
	v.rateHistoryFields = slices.DeleteFunc(slices.Clone(read.ReadFields), func(k string) bool { return !has(h.ReadFields, k) })
	return v, nil
}
func (s *Store) scopeView(ctx context.Context, u *unit, policy cfg.Policy, historical bool) (*c.PlanScope, error) {
	out := &c.PlanScope{Mode: policy.Mode, Version: policy.Version}
	for _, member := range policy.Members {
		grant, e := u.decision(ctx, sourcePermission(member.Dimension), nil)
		if e != nil {
			return nil, e
		}
		if !grant.Allowed || !has(grant.ReadFields, "name") {
			out.Restricted = true
			continue
		}
		m := c.ScopeMember{Dimension: member.Dimension, ID: member.ID, Exclude: member.Exclude, Name: member.Name}
		if has(grant.ReadFields, "code") {
			m.Code = member.Code
		}
		ha, e := u.decision(ctx, sourceHistoryPermission(member.Dimension), nil)
		if e != nil {
			return nil, e
		}
		if historical && ha.Allowed && has(ha.ReadFields, "name") {
			m.Source, m.RecordID, m.Revision = member.Source, member.Record, member.Revision
		}
		out.Members = append(out.Members, m)
	}
	return out, nil
}
func sourceHistoryPermission(dim string) string {
	switch dim {
	case "legalEntity":
		return "legalentity.history.read"
	case "jobFamily":
		return "jobfamily.history.read"
	case "grade":
		return "grade.history.read"
	}
	return ""
}
func (s *Store) projectPlan(ctx context.Context, u *unit, owner string, v planView, r d.Record[d.PlanTerms], until, day string, historical, withRows bool) (out c.PlanRecord, err error) {
	out.ID, out.Revision = r.ID, r.Revision
	f := v.fields
	if has(f, "name") {
		out.Name = r.Data.Name
	}
	if has(f, "description") {
		out.Description = r.Data.Description
	}
	if has(f, "effectiveDate") {
		out.Start, out.Until = r.Start, until
	}
	if has(f, "status") {
		if r.Active {
			out.Status = "ACTIVE"
		} else {
			out.Status = "INACTIVE"
		}
	}
	if has(f, "reason") {
		out.Operation, out.Reason = r.Operation, r.Reason
	}
	if has(f, "scope") {
		p, e := s.Scopes.ReadPlanScope(ctx, u.enterprise.ID, owner, r.Data.ScopeVersion)
		if e != nil {
			return out, e
		}
		out.Scope, err = s.scopeView(ctx, u, p, historical)
		if err != nil {
			return out, err
		}
	}
	if !withRows || !has(f, "components") {
		return out, nil
	}
	visibleFixed := map[string]string{}
	grades := map[string][]string{}
	for _, item := range r.Data.Components {
		row := c.PlanComponent{ID: item.ID, Code: item.Code, Name: item.Name, Nature: item.Nature, Method: item.Method, Currency: item.Currency, Frequency: item.Frequency, Unit: item.Unit, Precision: item.Precision}
		if has(f, "amount") {
			row.Amount = item.Amount
		}
		if has(f, "percentage") && has(f, "amount") {
			row.BaseID, row.Percent, row.Rounding = item.BaseID, item.Percent, item.Rounding
		}
		if v.rateReference {
			for _, link := range item.Rates {
				rate, e := s.projectRateLink(ctx, u, v, link, day, historical)
				if e != nil {
					return out, e
				}
				row.Rates = append(row.Rates, rate)
				grades[item.ID] = append(grades[item.ID], link.GradeID)
				if rate.Record != nil && rate.Record.Form == "FIXED" && rate.Record.Amount != "" {
					visibleFixed[link.RateID] = rate.Record.Amount
				}
			}
		}
		out.Components = append(out.Components, row)
	}
	if !has(f, "amount") || !has(f, "percentage") {
		return out, nil
	}
	byID := map[string]d.Component{}
	for _, item := range r.Data.Components {
		byID[item.ID] = item
	}
	for i := range out.Components {
		row := &out.Components[i]
		if row.Method != "PERCENT" {
			continue
		}
		base := byID[row.BaseID]
		for n := 0; base.Method == "PERCENT" && n < len(byID); n++ {
			base = byID[base.BaseID]
		}
		candidates := grades[base.ID]
		if base.Method == "FIXED" {
			candidates = []string{""}
		}
		for _, grade := range candidates {
			amount, ok := d.TargetQuote(r.Data.Components, row.ID, grade, func(link d.RateLink) (string, bool) { a, ok := visibleFixed[link.RateID]; return a, ok })
			if ok {
				row.Targets = append(row.Targets, c.Target{GradeID: grade, Amount: amount})
			}
		}
	}
	return out, nil
}
func (s *Store) projectRateLink(ctx context.Context, u *unit, v planView, link d.RateLink, day string, historical bool) (out c.PlanRate, err error) {
	out = c.PlanRate{RateID: link.RateID, GradeID: link.GradeID, BusinessAt: day, Open: v.rateCurrent}
	ad := link.Adopted
	if v.rateHistory {
		out.Adopted = &c.Reference{Open: true, BusinessAt: ad.BusinessAt, SourceVersion: ad.Source, RecordID: ad.Record, Revision: ad.Revision}
		out.AdoptedHead = ad.Head
	}
	var rec d.Record[d.RateTerms]
	var until string
	fields := v.rateFields
	if historical {
		var raw []byte
		err = u.tx.QueryRow(ctx, "SELECT fact FROM compensation.grade_rate_revision WHERE instance_id=$1 AND rate_id=$2 AND record_id=$3 AND revision=$4", s.Instance, link.RateID, ad.Record, ad.Revision).Scan(&raw)
		if err != nil {
			return out, err
		}
		if err = json.Unmarshal(raw, &rec); err != nil {
			return out, err
		}
		out.BusinessAt = ad.BusinessAt
		fields = v.rateHistoryFields
	} else {
		state, e := s.loadRate(ctx, u, link.RateID, false)
		if e != nil {
			return out, e
		}
		pos := state.Records.At(day)
		if pos < 0 {
			return out, nil
		}
		rec = state.Records[pos]
		until = state.Records.Period(pos).Until
	}
	out.Name, out.Code = rec.Data.Grade.Name, rec.Data.Grade.Code
	if !has(v.fields, "amount") || !v.rateCurrent || historical && !v.rateHistory {
		return out, nil
	}
	visible := visibility{fields: fields}
	record := visible.record(d.RateKey{GradeID: link.GradeID}, rec, until)
	out.Record = &record
	return out, nil
}

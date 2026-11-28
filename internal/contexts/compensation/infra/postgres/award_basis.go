package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"slices"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	d "chnhcm.local/chnhcm/internal/contexts/compensation/domain"
	c "chnhcm.local/chnhcm/internal/contexts/compensation/public"
	st "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

type awardAdoption struct {
	Currency  string                 `json:"currency"`
	Frequency string                 `json:"frequency"`
	Unit      string                 `json:"unit"`
	Precision int                    `json:"precision"`
	Day       string                 `json:"businessAt"`
	Term      st.PayAssignment       `json:"term"`
	Basis     c.PayBasis             `json:"basis"`
	Sources   map[string]c.Reference `json:"sources"`
}
type awardVisibility struct {
	plans, rates []string
	grade        bool
}

func (u *unit) awardVisibility(ctx context.Context, history bool) (v awardVisibility, err error) {
	fields := func(action string) ([]string, error) {
		g, e := u.decision(ctx, action+".read", nil)
		if e != nil || !g.Allowed {
			return nil, e
		}
		if !history {
			return g.ReadFields, nil
		}
		h, e := u.decision(ctx, action+".history.read", nil)
		if e != nil || !h.Allowed {
			return nil, e
		}
		return slices.DeleteFunc(slices.Clone(g.ReadFields), func(f string) bool { return !has(h.ReadFields, f) }), nil
	}
	if v.plans, err = fields("compplan"); err != nil {
		return
	}
	if v.rates, err = fields("graderate"); err != nil {
		return
	}
	grade, e := u.decision(ctx, "grade.read", nil)
	v.grade = grade.Allowed && has(grade.ReadFields, "name")
	if e == nil && history && v.grade {
		h, err := u.decision(ctx, "grade.history.read", nil)
		if err != nil {
			return v, err
		}
		v.grade = h.Allowed && has(h.ReadFields, "name")
	}
	return v, e
}
func (v awardVisibility) canChoose() bool {
	return has(v.plans, "components") && has(v.plans, "rates") && has(v.plans, "scope") && v.grade && has(v.rates, "minimum") && has(v.rates, "maximum")
}
func (v awardVisibility) project(b c.PayBasis) *c.PayBasis {
	if len(v.plans) == 0 && len(v.rates) == 0 {
		return nil
	}
	if !has(v.plans, "name") {
		b.PlanName = ""
	}
	if !has(v.plans, "code") {
		b.PlanCode = ""
	}
	if len(v.plans) == 0 {
		b.PlanID, b.PlanRecord, b.PlanRevision, b.PlanSource = "", "", "", ""
	}
	if !has(v.plans, "scope") {
		b.ScopeVersion = ""
	}
	if !has(v.plans, "components") {
		b.ComponentID, b.ComponentName = "", ""
	}
	if !has(v.plans, "amount") || !has(v.plans, "components") {
		b.Allowances = nil
	}
	if !has(v.plans, "rates") || len(v.rates) == 0 {
		b.RateID, b.RateRecord, b.RateRevision, b.RateSource, b.Minimum, b.Guidance, b.Maximum = "", "", "", "", "", "", ""
	}
	if !has(v.rates, "minimum") {
		b.Minimum = ""
	}
	if !has(v.rates, "maximum") {
		b.Maximum = ""
	}
	if !has(v.rates, "guidance") {
		b.Guidance = ""
	}
	if !v.grade {
		b.Grade = ""
	}
	b.Token = ""
	return &b
}

func (s *Store) resolveAward(ctx context.Context, u *unit, term st.PayAssignment, day, id string, write bool) (*awardAdoption, error) {
	if !term.Active || term.Grade == "" {
		return nil, nil
	}
	plan, e := s.loadPlan(ctx, u, id, write)
	if e != nil {
		return nil, e
	}
	pi := plan.Records.At(day)
	if pi < 0 || !plan.Records[pi].Active {
		return nil, nil
	}
	pr := plan.Records[pi]
	basic, rateID, ok := d.MonthlyBasic(pr.Data.Components, term.Grade)
	if !ok {
		return nil, nil
	}
	policy, e := s.Scopes.ReadPlanScope(ctx, u.enterprise.ID, id, pr.Data.ScopeVersion)
	if e != nil {
		return nil, e
	}
	values := map[string]string{"legalEntity": term.Employer, "jobFamily": term.Family, "grade": term.Grade}
	used := map[string]bool{"grade": true}
	for _, m := range policy.Members {
		if values[m.Dimension] == "" {
			return nil, nil
		}
		used[m.Dimension] = true
	}
	if !s.Scopes.Matches(policy, values) {
		return nil, nil
	}
	rate, e := s.loadRate(ctx, u, rateID, write)
	if e != nil {
		return nil, e
	}
	ri := rate.Records.At(day)
	if ri < 0 || !rate.Records[ri].Active || rate.Key.GradeID != term.Grade || rate.Key.Currency != "CNY" || rate.Key.Frequency != "MONTH" || rate.Key.Unit != "PER_MONTH" || rate.Places != 2 {
		return nil, nil
	}
	rr := rate.Records[ri]
	band := rr.Data.Amounts
	if band.Form != "RANGE" {
		return nil, nil
	}
	adoption := awardAdoption{Day: day, Term: term, Currency: rate.Key.Currency, Frequency: rate.Key.Frequency, Unit: rate.Key.Unit, Precision: 2, Sources: map[string]c.Reference{}}
	for _, dim := range []string{"legalEntity", "jobFamily", "grade"} {
		if !used[dim] {
			continue
		}
		g, err := u.decision(ctx, sourcePermission(dim), nil)
		if err != nil {
			return nil, err
		}
		if !g.Allowed || !has(g.ReadFields, "name") {
			return nil, nil
		}
		var src foundation.CatalogSource
		switch dim {
		case "legalEntity":
			src, e = s.LegalEntities.CompensationLegalEntity(ctx, u.enterprise.ID, values[dim], write)
		case "grade":
			src, e = s.Grades.CompensationGrade(ctx, u.enterprise.ID, values[dim], write)
		case "jobFamily":
			src, e = s.Grades.CompensationJobFamily(ctx, u.enterprise.ID, values[dim], write)
		}
		if e != nil {
			return nil, e
		}
		ix := slices.IndexFunc(src.Segments, func(seg foundation.CatalogSegment) bool { return seg.Active && seg.Includes(day) })
		if ix < 0 {
			return nil, nil
		}
		seg := src.Segments[ix]
		adoption.Sources[dim] = c.Reference{BusinessAt: day, SourceVersion: src.Source, RecordID: seg.RecordID, Revision: seg.Revision}
		if dim == "grade" {
			adoption.Basis.Grade = seg.Name
		}
	}
	b := &adoption.Basis
	b.PlanID, b.PlanCode, b.PlanName, b.PlanRecord, b.PlanRevision, b.ScopeVersion = id, plan.Code, pr.Data.Name, pr.ID, pr.Revision, policy.Version
	b.ComponentID, b.ComponentName, b.RateID, b.RateRecord, b.RateRevision = basic.ID, basic.Name, rateID, rr.ID, rr.Revision
	b.Minimum, b.Guidance, b.Maximum = band.Minimum, band.Guidance, band.Maximum
	e = u.tx.QueryRow(ctx, `SELECT p.id::text,r.id::text FROM compensation.plan_version p,compensation.grade_rate_version r
 WHERE p.instance_id=$1 AND p.plan_id=$2 AND p.record_id=$3 AND p.revision=$4 AND p.action<>'DELETE'
 AND r.instance_id=$1 AND r.rate_id=$5 AND r.record_id=$6 AND r.revision=$7 AND r.action<>'DELETE'`, s.Instance, id, pr.ID, pr.Revision, rateID, rr.ID, rr.Revision).Scan(&b.PlanSource, &b.RateSource)
	if e != nil {
		return nil, e
	}
	for _, row := range pr.Data.Components {
		if row.Nature == "ALLOWANCE" && row.Method == "FIXED" && row.Currency == "CNY" && row.Frequency == "MONTH" {
			b.Allowances = append(b.Allowances, c.PayAllowance{Name: row.Name, Amount: row.Amount})
		}
	}
	// Keep the complete source snapshots above. The editor token compares only the facts
	// used on this day, so changing a later source segment does not invalidate this decision.
	type chosenSource struct{ ID, Record, Revision string }
	selected := map[string]chosenSource{}
	for dim, src := range adoption.Sources {
		selected[dim] = chosenSource{values[dim], src.RecordID, src.Revision}
	}
	raw, e := json.Marshal(struct {
		Day, Assignment, Employment string
		Basis                       c.PayBasis
		Sources                     map[string]chosenSource
	}{day, term.Assignment, term.Employment, *b, selected})
	if e != nil {
		return nil, e
	}
	sum := sha256.Sum256(raw)
	b.Token = hex.EncodeToString(sum[:])
	return &adoption, nil
}

func (s *Store) awardCandidates(ctx context.Context, u *unit, term st.PayAssignment, day string, rights access.BasePayRights, v awardVisibility) ([]c.PayBasis, error) {
	out := []c.PayBasis{}
	if !term.Active || term.Grade == "" || !v.canChoose() || !rights.UseAll && len(rights.Plans) == 0 {
		return out, nil
	}
	rows, e := u.tx.Query(ctx, `SELECT p.plan_id::text FROM compensation.plan_period p JOIN compensation.plan h ON (h.instance_id,h.id)=(p.instance_id,p.plan_id) WHERE p.instance_id=$1 AND h.enterprise_id=$2 AND p.active AND p.valid_period @> $3::date ORDER BY h.code,h.id`, s.Instance, u.enterprise.ID, day)
	if e != nil {
		return nil, e
	}
	var ids []string
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return nil, e
		}
		if rights.UseAll || rights.Plans[id] {
			ids = append(ids, id)
		}
	}
	rows.Close()
	if e = rows.Err(); e != nil {
		return nil, e
	}
	for _, id := range ids {
		adopt, e := s.resolveAward(ctx, u, term, day, id, false)
		if e != nil {
			return nil, e
		}
		if adopt != nil {
			b := v.project(adopt.Basis)
			b.Token = adopt.Basis.Token
			out = append(out, *b)
		}
	}
	return out, nil
}

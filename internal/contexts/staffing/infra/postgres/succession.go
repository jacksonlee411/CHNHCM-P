package postgres

import (
	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	f "chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
	"cmp"
	"context"
	"slices"
)

type successionContext struct{}
type successionAdmission struct {
	source              h.Succession
	recipient, position string
}

// A succession credit follows the identified regular appointment only up to
// its continuous, already published departure. A person with no end on record
// does not release a regular place for the successor.
func incumbentCoverage(usage d.QuotaUsage, id, day string) (d.PositionUse, string, bool) {
	var pieces []d.PositionUse
	for _, v := range usage.Uses {
		if v.Owner == "M04" && v.Assignment == id && v.Effect == "occupied" {
			pieces = append(pieces, v)
		}
	}
	slices.SortFunc(pieces, func(a, b d.PositionUse) int { return cmp.Compare(a.From, b.From) })
	for i, v := range pieces {
		if !v.Contains(day) {
			continue
		}
		until := v.Until
		for _, n := range pieces[i+1:] {
			if n.From != until {
				break
			}
			until = n.Until
		}
		return v, until, until != ""
	}
	return d.PositionUse{}, "", false
}

func (u *unit) canArrangeSuccession(ctx context.Context, newcomer, incumbent h.Action, dept string) (bool, error) {
	for _, a := range []h.Action{newcomer, incumbent} {
		for _, r := range []string{"read", "position.handover"} {
			ok, e := u.employmentAllowed(ctx, r, a.Person, a.Employer, dept, false)
			if e != nil || !ok {
				return false, e
			}
		}
	}
	return u.employmentAllowed(ctx, "transfer", newcomer.Person, newcomer.Employer, dept, false)
}

func (st *Store) SuccessionOptions(ctx context.Context, c f.Credentials, id string, in h.SuccessionSearch) (out []h.SuccessionOption, err error) {
	out = []h.SuccessionOption{}
	if !f.ValidID(in.Position) || !f.ValidDate(in.BusinessDate) || in.Page < 1 || in.Page > 1000000 {
		return out, f.Fail(400, "SUCCESSION_INPUT", "请核对岗位、调入日与页码")
	}
	err = st.within(ctx, c, false, true, func(u *unit) error {
		a, e := u.assignmentAt(ctx, id, in.BusinessDate, "")
		if e != nil {
			return e
		}
		for _, r := range []string{"transfer", "position.handover"} {
			if e = u.requireEmployment(ctx, r, a.Person, a.Employer, a.Department); e != nil {
				return e
			}
		}
		ent, e := u.Enterprise(ctx, false)
		if e != nil {
			return e
		}
		ok, e := u.positionReadable(ctx, ent.ID)
		if e != nil || !ok {
			return e
		}
		usage, e := u.PositionUsage(ctx, in.Position)
		if e != nil {
			return e
		}
		if !usage.At(in.BusinessDate).Known {
			return nil
		}
		term, e := u.maintenancePeriod(ctx, a, in.BusinessDate, "Transfer")
		if e != nil {
			return e
		}
		type candidate struct {
			option h.SuccessionOption
			person string
		}
		var candidates []candidate
		var people []string
		for _, v := range usage.Uses {
			if v.Owner != "M04" || v.Effect != "occupied" || !v.Contains(in.BusinessDate) || v.Assignment == id {
				continue
			}
			_, end, eligible := incumbentCoverage(usage, v.Assignment, in.BusinessDate)
			if !eligible {
				continue
			}
			other, e := u.assignmentIdentity(ctx, v.Assignment)
			if e != nil {
				return e
			}
			if other.Person == a.Person {
				continue
			}
			ok, e := u.canArrangeSuccession(ctx, a, other, v.Department)
			if e != nil {
				return e
			}
			if !ok {
				continue
			}
			ok, e = u.employmentAllowed(ctx, "directory", other.Person, other.Employer, v.Department, false)
			if e != nil {
				return e
			}
			if !ok {
				continue
			}
			p, e := u.assignmentPlacement(ctx, other, in.BusinessDate)
			if e != nil {
				return e
			}
			continues := term.Until == "" || term.Until > end
			if !continues {
				end = term.Until
			}
			o := h.SuccessionOption{Continues: continues, IncumbentRef: h.IncumbentRef{Assignment: v.Assignment, Source: v.Source}, Number: other.RecordNumber, Until: end, Department: p.Adoption.DepartmentName, Job: p.Adoption.JobName}
			candidates = append(candidates, candidate{o, other.Person})
			people = append(people, other.Person)
		}
		names, _, e := st.HirePeople.RecognizedPeople(dbtx.With(ctx, u.tx), c, people, in.Ref)
		if e != nil {
			return e
		}
		labels := map[string]string{}
		for _, n := range names {
			labels[n.ID] = n.Name
		}
		for _, row := range candidates {
			if name, ok := labels[row.person]; ok {
				row.option.Name = name
				out = append(out, row.option)
			}
		}
		slices.SortFunc(out, func(a, b h.SuccessionOption) int {
			return cmp.Or(cmp.Compare(a.Name, b.Name), cmp.Compare(a.Number, b.Number), cmp.Compare(a.Assignment, b.Assignment))
		})
		start := min(len(out), (in.Page-1)*20)
		out = out[start:min(start+20, len(out))]
		return nil
	})
	return
}

// Caller already holds the locks for both transfer positions. The immutable
// adoption and both quota periods are saved by this same transfer transaction.
func (u *unit) adoptSuccession(ctx context.Context, a h.Action, action, position string, span f.Period, ref h.IncumbentRef) (*h.Succession, error) {
	if ref.Assignment == a.Assignment {
		return nil, f.Fail(409, "SUCCESSION_INPUT", "接任者与原任须为不同人员")
	}
	other, e := u.assignmentIdentity(ctx, ref.Assignment)
	if e != nil {
		return nil, e
	}
	usage, e := u.PositionUsage(ctx, position)
	if e != nil {
		return nil, e
	}
	member, end, ok := incumbentCoverage(usage, other.Assignment, span.From)
	if !ok || other.SourceVersion != ref.Source || other.Person == a.Person {
		return nil, f.Fail(409, "SUCCESSION_PERIOD", "原任职或其确定的离岗边界已变化，请重新选择")
	}
	term := f.Period{From: span.From, Until: end}
	if span.Until != "" && span.Until < end {
		term.Until = span.Until
	}
	permit, e := u.canArrangeSuccession(ctx, a, other, member.Department)
	if e != nil {
		return nil, e
	}
	if !permit {
		return nil, f.Fail(403, "SUCCESSION_DENIED", "当前权限不能安排该岗位交接")
	}
	for _, v := range usage.Uses {
		if v.Owner == "M04" && v.Assignment == other.Assignment && (f.Period{From: v.From, Until: v.Until}).Overlaps(term) {
			permit, e = u.canArrangeSuccession(ctx, a, other, v.Department)
			if e != nil {
				return nil, e
			}
			if !permit {
				return nil, f.Fail(403, "SUCCESSION_DENIED", "当前权限不能覆盖全部交接期间")
			}
		}
		if v.Incumbent == other.Assignment && v.Assignment != a.Assignment && (f.Period{From: v.From, Until: v.Until}).Overlaps(term) {
			return nil, f.Fail(409, "SUCCESSION_ASSIGNED", "原任在该期间已有接替安排")
		}
	}
	names, _, e := u.store.HirePeople.RecognizedPeople(dbtx.With(ctx, u.tx), u.credentials, []string{other.Person}, "")
	if e != nil {
		return nil, e
	}
	if len(names) != 1 {
		return nil, f.Fail(403, "SUCCESSION_DENIED", "当前权限不能读取被接替人员")
	}
	s := h.Succession{Continues: span.Until == "" || span.Until > end, IncumbentRef: ref, Period: term, ID: f.NewID(), Name: names[0].Name, Number: other.RecordNumber}
	_, e = u.tx.Exec(ctx, `INSERT INTO staffing.position_succession(instance_id,id,position_id,assignment_id,action_id,incumbent,incumbent_source,period) VALUES($1,$2,$3,$4,$5,$6,$7,daterange($8::date,$9::date,'[)'))`, u.store.Instance, s.ID, position, a.Assignment, action, s.Assignment, s.Source, s.From, s.Until)
	return &s, e
}

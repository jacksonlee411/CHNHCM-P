package postgres

import (
	"context"
	"errors"
	"slices"

	recruit "chnhcm.local/chnhcm/internal/contexts/recruiting/public"
	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	f "chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
	"github.com/jackc/pgx/v5"
)

type offerContext struct{}
type offerAdmission struct {
	offer recruit.Offer
	ref   recruit.HandoffRef
}

func offerBinding(v recruit.Offer) h.OfferHireCommand {
	cmd := h.OfferHireCommand{Version: v.Version, Person: v.Person, Employer: v.Employer, Arrival: v.Arrival, LastDay: v.LastDay, Department: v.Department, Job: v.Job, Position: v.Position}
	for _, seg := range v.Segments {
		if seg.Includes(v.Arrival) {
			cmd.Department, cmd.Job = seg.DepartmentID, seg.JobID
			break
		}
	}
	return cmd
}
func offerHireInput(in h.OfferHireCommand) h.Hire {
	return h.Hire{Person: in.Person, Employer: in.Employer, Arrival: in.Arrival, LastDay: in.LastDay, Position: in.Position, Department: in.Department, Job: in.Job, Notice: "已接受的招聘录用", Expected: in.Expected}
}
func offerWork(ctx context.Context, v recruit.Offer) context.Context {
	return context.WithValue(ctx, offerContext{}, offerAdmission{v, recruit.HandoffRef{Offer: v.ID, Version: v.Version, Number: v.Number}})
}
func checkOfferHire(v recruit.Offer, in h.Hire, p hirePlan) error {
	conflict := f.Fail(409, "OFFER_HANDOFF", "招聘录用的岗位、职务或预留期间与入职不一致，请从原录用核对")
	days := []string{}
	for _, seg := range v.Segments {
		days = append(days, seg.From, seg.Until)
	}
	if p.Adoption == nil {
		return conflict
	}
	for _, seg := range p.Adoption.Segments {
		for _, period := range seg.Split(days) {
			i := slices.IndexFunc(v.Segments, func(s recruit.AssignmentSegment) bool { return s.Includes(period.From) })
			if i < 0 {
				return conflict
			}
			before := v.Segments[i]
			dept, job := seg.DepartmentID, seg.JobID
			if dept == "" {
				dept = in.Department
			}
			if job == "" {
				job = in.Job
			}
			if before.Position != seg.Position || before.DepartmentID != dept || before.JobID != job || before.Seats != seg.Seats {
				return conflict
			}
		}
	}
	return nil
}
func (u *unit) visibleHandoff(ctx context.Context, v recruit.Offer) (out h.OfferHireState, err error) {
	cmd := offerBinding(v)
	known, _, err := u.store.HirePeople.RecognizedPeople(dbtx.With(ctx, u.tx), u.credentials, []string{v.Person}, "")
	if err != nil {
		return
	}
	if len(known) != 1 {
		return
	}
	if v.AcceptedVersion != "" {
		var r h.ActionRef
		var id string
		err = u.tx.QueryRow(ctx, `SELECT s.action_id,s.revision,a.assignment_id FROM staffing.hired_offer s JOIN staffing.employment_action a ON (a.instance_id,a.id)=(s.instance_id,s.action_id) WHERE s.instance_id=$1 AND s.offer_id=$2 AND s.accepted_version=$3`, u.store.Instance, v.ID, v.AcceptedVersion).Scan(&r.Action, &r.Revision, &id)
		if err != nil {
			return
		}
		_, err = u.assignmentAt(ctx, id, v.Arrival, "")
		var fault *f.Fault
		if errors.As(err, &fault) && (fault.Status == 403 || fault.Status == 404) {
			return out, nil
		}
		if err != nil {
			return
		}
		out.Result, out.Assignment = &r, id
		return
	}
	allowed, err := u.employmentAllowed(ctx, "hire", v.Person, v.Employer, cmd.Department, false)
	if err != nil || !allowed {
		return
	}
	out.CanHire = v.Current && v.Status == "accepted" && slices.ContainsFunc(v.Segments, func(s recruit.AssignmentSegment) bool { return s.Includes(v.Arrival) })
	if out.CanHire {
		out.Binding = &cmd
	}
	return
}
func (st *Store) OfferHireState(ctx context.Context, c f.Credentials, id, version string) (out h.OfferHireState, err error) {
	if !f.ValidID(id) || !f.ValidID(version) {
		return out, f.Fail(400, "OFFER_INPUT", "请核对原录用及版本")
	}
	err = st.within(ctx, c, false, true, func(u *unit) error {
		v, e := st.OfferHandoffs.OfferForHire(dbtx.With(ctx, u.tx), c, id, version, false)
		if e != nil {
			return e
		}
		out, e = u.visibleHandoff(ctx, v)
		if e != nil || !out.CanHire {
			return e
		}
		input := offerHireInput(*out.Binding)
		p, e := u.prepareHire(offerWork(ctx, v), input)
		if e != nil {
			return e
		}
		if len(p.Missing) > 0 {
			out.CanHire = false
			out.Missing = p.Missing
			out.Binding = nil
			return nil
		}
		if e = checkOfferHire(v, input, p); e != nil {
			return e
		}
		out.Binding.Expected = p.Adoption
		return nil
	})
	return
}
func (st *Store) HireFromOffer(ctx context.Context, c f.Credentials, id, key string, in h.OfferHireCommand) (h.Receipt, error) {
	return st.hireCommand(ctx, c, key, []any{"offer-hire", id, in}, func(u *unit) (string, error) {
		if !f.ValidID(id) || !f.ValidID(in.Version) || !f.ValidID(in.Person) {
			return "", f.Fail(400, "OFFER_INPUT", "请核对原录用及人员")
		}
		if e := lock(ctx, u.tx, false, "staffing.hire.person", st.Instance, in.Person); e != nil {
			return "", e
		}
		txctx := dbtx.With(ctx, u.tx)
		v, e := st.OfferHandoffs.OfferForHire(txctx, c, id, in.Version, true)
		if e != nil {
			return "", e
		}
		binding := in
		binding.Expected = nil
		if v.Status != "accepted" || binding != offerBinding(v) {
			return "", f.Fail(409, "OFFER_HANDOFF", "人员、用人单位、到岗日或任用安排与已接受录用不符")
		}
		if e = u.requireEmployment(ctx, "hire", v.Person, v.Employer, in.Department); e != nil {
			return "", e
		}
		state, e := u.visibleHandoff(ctx, v)
		if e != nil {
			return "", e
		}
		if state.Result != nil {
			return state.Result.Action, nil
		}
		if !state.CanHire {
			return "", f.Fail(409, "OFFER_HANDOFF", "原录用或预留已变化，请先核实当前交接")
		}
		hire := offerHireInput(in)
		result, e := u.createHire(offerWork(ctx, v), hire)
		if e != nil {
			return "", e
		}
		action, e := u.action(ctx, result)
		if e != nil {
			return "", e
		}
		converted, e := st.OfferHandoffs.CompleteHire(txctx, c, v, recruit.HireResult{Action: result, Revision: action.Revision, Assignment: action.Assignment})
		if e != nil {
			return "", e
		}
		_, e = u.tx.Exec(ctx, `INSERT INTO staffing.hired_offer VALUES($1,$2,$3,$4,$5,$6)`, st.Instance, id, v.Version, converted, result, action.Revision)
		return result, e
	})
}
func (u *unit) cancelAdoptedOffer(ctx context.Context, original, action, revision, day string) error {
	var ref recruit.HandoffRef
	err := u.tx.QueryRow(ctx, `SELECT offer_id,accepted_version FROM staffing.hired_offer WHERE instance_id=$1 AND action_id=$2`, u.store.Instance, original).Scan(&ref.Offer, &ref.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return u.store.OfferHandoffs.CancelHireConversion(dbtx.With(ctx, u.tx), u.credentials, ref, action, revision, day)
}

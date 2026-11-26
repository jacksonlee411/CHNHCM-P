package postgres

import (
	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	f "chnhcm.local/chnhcm/internal/foundation"
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"strings"
	"time"
	"unicode/utf8"
)

type dutyContext struct{}
type dutyAdmission struct {
	assignment string
	evidence   h.DutyEvidence
}

func (u *unit) dutyScope(ctx context.Context, a h.Action, position string, period f.Period, right string) error {
	ent, e := u.Enterprise(ctx, false)
	if e != nil {
		return e
	}
	visible, e := u.positionReadable(ctx, ent.ID)
	if e != nil {
		return e
	}
	if !visible {
		return f.Fail(403, "DUTY_DENIED", "当前权限不能读取该岗位的履职资料")
	}
	p, e := u.Load(ctx, d.Position, position, "")
	if e != nil {
		return e
	}
	var bounds []string
	for _, row := range p.Records {
		bounds = append(bounds, row.Date, row.End)
	}
	for _, span := range period.Split(bounds) {
		i := d.At(p, span.From)
		if i < 0 || !p.Records[i].Active() {
			return f.Fail(409, "DUTY_POSITION", "实际履职期间没有有效岗位")
		}
		for _, grant := range []string{"read", right} {
			if e = u.requireEmployment(ctx, grant, a.Person, a.Employer, p.Records[i].Data["departmentId"]); e != nil {
				return e
			}
		}
	}
	return nil
}
func (u *unit) dutyDecision(ctx context.Context, a h.Action, ref h.DecisionRef) (h.DecisionOption, error) {
	if !validDecisionRef(ref) {
		return h.DecisionOption{}, appointmentConflict()
	}
	if e := lock(ctx, u.tx, true, "staffing.appointment", u.store.Instance, ref.Source); e != nil {
		return h.DecisionOption{}, e
	}
	opt, e := scanDecision(u.tx.QueryRow(ctx, `WITH versions AS (SELECT * FROM staffing.appointment_version WHERE instance_id=$1 AND source=$3 ORDER BY edition DESC LIMIT 1) `+memberSelect, u.store.Instance, a.Person, ref.Source))
	if errors.Is(e, pgx.ErrNoRows) {
		return opt, appointmentConflict()
	}
	if e != nil {
		return opt, e
	}
	if opt.Ref != ref || opt.Position == nil || opt.Employer != nil && opt.Employer.ID != a.Employer {
		return opt, appointmentConflict()
	}
	return opt, nil
}
func (st *Store) DutySources(ctx context.Context, c f.Credentials, id string) (out h.DutySources, err error) {
	out = h.DutySources{Decisions: []h.DecisionOption{}, Observations: []h.DutyEvidence{}}
	err = st.within(ctx, c, false, true, func(u *unit) error {
		a, e := u.assignmentIdentity(ctx, id)
		if e != nil {
			return e
		}
		can, e := u.employmentAllowed(ctx, "read", a.Person, a.Employer, "", true)
		if e != nil {
			return e
		}
		if !can {
			return unknownAssignment()
		}
		permit := func(pos string, period f.Period, right string) (bool, error) {
			e := u.dutyScope(ctx, a, pos, period, right)
			var fault *f.Fault
			if errors.As(e, &fault) && (fault.Status == 403 || fault.Status == 409) {
				return false, nil
			}
			return e == nil, e
		}
		ent, e := u.Enterprise(ctx, false)
		if e != nil {
			return e
		}
		ok, e := u.positionReadable(ctx, ent.ID)
		if e != nil || !ok {
			return e
		}
		rows, e := u.tx.Query(ctx, latestDecisions+memberSelect+" ORDER BY v.number,v.source", u.store.Instance, a.Person)
		if e != nil {
			return e
		}
		choices, e := pgx.CollectRows(rows, func(r pgx.CollectableRow) (h.DecisionOption, error) { return scanDecision(r) })
		if e != nil {
			return e
		}
		for _, opt := range choices {
			if opt.Position == nil || opt.Arrival == "" || opt.Employer != nil && opt.Employer.ID != a.Employer || !a.Includes(opt.Arrival) {
				continue
			}
			at, e := u.assignmentAt(ctx, id, opt.Arrival, "")
			if e != nil {
				var fault *f.Fault
				if errors.As(e, &fault) && fault.Status == 403 {
					continue
				}
				return e
			}
			ok, e := u.employmentAllowed(ctx, "duty.verify", at.Person, at.Employer, at.Department, false)
			if e != nil {
				return e
			}
			if !ok {
				continue
			}
			period, e := hireDates(opt.Arrival, opt.LastDay)
			if e != nil {
				return e
			}
			if ok, e := permit(opt.Position.ID, period, "duty.verify"); e != nil {
				return e
			} else if ok {
				out.Decisions = append(out.Decisions, h.DecisionOption{Ref: opt.Ref, Number: opt.Number, DecisionRow: h.DecisionRow{Position: opt.Position, Person: a.Person, Arrival: opt.Arrival, LastDay: opt.LastDay}})
			}
		}
		rows, e = u.tx.Query(ctx, `SELECT r.fact FROM staffing.duty_observation o JOIN staffing.duty_observation_head x ON (o.instance_id,o.id)=(x.instance_id,x.observation_id) JOIN staffing.duty_observation_revision r ON (r.instance_id,r.observation_id,r.version)=(x.instance_id,x.observation_id,x.version)
       WHERE o.instance_id=$1 AND o.assignment_id=$2 ORDER BY o.issuer,o.number`, u.store.Instance, id)
		if e != nil {
			return e
		}
		proofs, e := pgx.CollectRows(rows, pgx.RowTo[h.DutyEvidence])
		if e != nil {
			return e
		}
		for _, v := range proofs {
			at, e := u.assignmentAt(ctx, id, v.From, "")
			if e != nil {
				var fault *f.Fault
				if errors.As(e, &fault) && fault.Status == 403 {
					continue
				}
				return e
			}
			can, e := u.employmentAllowed(ctx, "duty.correct", at.Person, at.Employer, at.Department, false)
			if e != nil {
				return e
			}
			if can {
				ok, e := permit(v.Position, v.Period, "duty.correct")
				if e != nil {
					return e
				}
				if ok {
					out.Observations = append(out.Observations, v)
				}
			}
		}
		return nil
	})
	return
}
func (st *Store) ReceiveDuty(ctx context.Context, c f.Credentials, id, key string, in h.DutyObservation) (out h.DutyEvidence, err error) {
	err = st.within(ctx, c, true, false, func(u *unit) error {
		var e error
		out, e = u.receiveDuty(ctx, id, key, in)
		return e
	})
	return
}
func (u *unit) lockDutyAssignment(ctx context.Context, id string) error {
	a, err := u.assignmentIdentity(ctx, id)
	if err != nil {
		return err
	}
	return lock(ctx, u.tx, false, "staffing.hire.person", u.store.Instance, a.Person)
}

func (u *unit) receiveDuty(ctx context.Context, id, key string, in h.DutyObservation) (out h.DutyEvidence, err error) {
	in.Issuer, in.Number = strings.TrimSpace(in.Issuer), strings.TrimSpace(in.Number)
	if !f.ValidID(key) || !f.ValidID(id) || in.Issuer == "" || in.Number == "" || utf8.RuneCountInString(in.Issuer) > 160 || utf8.RuneCountInString(in.Number) > 160 || in.Previous != nil && (!f.ValidID(in.Previous.ID) || !f.ValidID(in.Previous.Version)) {
		return out, f.Fail(400, "DUTY_EVIDENCE", "请核对出勤原件的出具方、编号与修订")
	}
	term, e := hireDates(in.From, in.LastDay)
	if e != nil {
		return out, e
	}
	today := u.now.In(f.Shanghai).Format(time.DateOnly)
	if !f.ValidDate(in.ObservedThrough) || in.ObservedThrough < in.From || in.ObservedThrough > today || in.LastDay != "" && in.LastDay > in.ObservedThrough {
		return out, f.Fail(409, "DUTY_NOT_OBSERVED", "只能依据已核实的实际到岗及出勤记录办理")
	}
	if e := u.lockDutyAssignment(ctx, id); e != nil {
		return out, e
	}
	a, e := u.assignmentAt(ctx, id, in.From, "")
	if e != nil {
		return out, e
	}
	verifiedEnd := in.LastDay
	if verifiedEnd == "" {
		verifiedEnd = in.ObservedThrough
	}
	if !a.Includes(in.From) || a.Cancelled || !a.Includes(verifiedEnd) {
		return out, f.Fail(409, "DUTY_TERM", "实际履职须在该任期内")
	}
	if e = u.requireEmployment(ctx, "duty.verify", a.Person, a.Employer, a.Department); e != nil {
		return out, e
	}
	if e = lock(ctx, u.tx, false, "staffing.duty.paper", u.store.Instance, id, in.Issuer, in.Number); e != nil {
		return out, e
	}
	digest, e := hireDigest([]any{id, in})
	if e != nil {
		return out, e
	}
	var priorDigest string
	e = u.tx.QueryRow(ctx, `SELECT digest,fact FROM staffing.duty_observation_revision WHERE instance_id=$1 AND version=$2 AND actor=$3`, u.store.Instance, key, u.guard.Principal().ID).Scan(&priorDigest, &out)
	if e == nil {
		if digest != priorDigest {
			return out, assignmentConflict()
		}
		return out, u.dutyScope(ctx, a, out.Position, out.Period, "duty.verify")
	}
	if !errors.Is(e, pgx.ErrNoRows) {
		return out, e
	}
	dispatch, e := u.dutyDecision(ctx, a, in.Dispatch)
	if e != nil {
		return out, e
	}
	if dispatch.Arrival == "" || dispatch.Arrival > in.ObservedThrough {
		return out, f.Fail(409, "DUTY_DISPATCH", "派岗原件须记载已到的派岗日期")
	}
	if e = u.dutyScope(ctx, a, dispatch.Position.ID, term, "duty.verify"); e != nil {
		return out, e
	}
	p, e := u.Load(ctx, d.Position, dispatch.Position.ID, "")
	if e != nil {
		return out, e
	}
	at := d.At(p, in.From)
	if at < 0 || !p.Records[at].Active() {
		return out, f.Fail(409, "DUTY_POSITION", "实际履职期间没有有效岗位")
	}
	var proof, previous string
	e = u.tx.QueryRow(ctx, `SELECT o.id,x.version FROM staffing.duty_observation o JOIN staffing.duty_observation_head x ON (o.instance_id,o.id)=(x.instance_id,x.observation_id) WHERE o.instance_id=$1 AND o.assignment_id=$2 AND o.issuer=$3 AND o.number=$4`, u.store.Instance, id, in.Issuer, in.Number).Scan(&proof, &previous)
	if !errors.Is(e, pgx.ErrNoRows) && e != nil {
		return out, e
	}
	if previous != "" && (in.Previous == nil || in.Previous.ID != proof || in.Previous.Version != previous) || previous == "" && in.Previous != nil {
		return out, assignmentConflict()
	}
	if previous != "" {
		var old h.DutyEvidence
		if e = u.tx.QueryRow(ctx, `SELECT fact FROM staffing.duty_observation_revision WHERE instance_id=$1 AND observation_id=$2 AND version=$3`, u.store.Instance, proof, previous).Scan(&old); e != nil {
			return out, e
		}
		was, e := u.assignmentAt(ctx, id, old.From, "")
		if e != nil {
			return out, e
		}
		if e = u.requireEmployment(ctx, "duty.verify", was.Person, was.Employer, was.Department); e != nil {
			return out, e
		}
		if e = u.dutyScope(ctx, was, old.Position, old.Period, "duty.verify"); e != nil {
			return out, e
		}
	}
	if proof == "" {
		proof = f.NewID()
		if _, e = u.tx.Exec(ctx, `INSERT INTO staffing.duty_observation VALUES($1,$2,$3,$4,$5)`, u.store.Instance, proof, id, in.Issuer, in.Number); e != nil {
			return out, e
		}
	}
	out = h.DutyEvidence{DutyRef: h.DutyRef{ID: proof, Version: key}, Period: term, Dispatch: dispatch.Ref, DispatchNumber: dispatch.Number, Number: in.Number, Issuer: in.Issuer, ObservedThrough: in.ObservedThrough, Position: dispatch.Position.ID, PositionName: p.Records[at].Data["name"]}
	_, e = u.tx.Exec(ctx, `INSERT INTO staffing.duty_observation_revision(instance_id,observation_id,version,previous,actor,digest,dispatch,dispatch_version,fact) VALUES($1,$2,$3,nullif($4,'')::uuid,$5,$6,$7,$8,$9)`, u.store.Instance, proof, key, previous, u.guard.Principal().ID, digest, dispatch.Ref.Source, dispatch.Ref.Version, out)
	if e == nil {
		_, e = u.tx.Exec(ctx, `INSERT INTO staffing.duty_observation_head VALUES($1,$2,$3) ON CONFLICT(instance_id,observation_id) DO UPDATE SET version=excluded.version`, u.store.Instance, proof, key)
	}
	return out, e
}
func (st *Store) RecordDuty(ctx context.Context, c f.Credentials, id, key string, in h.DutyCommand) (h.Receipt, error) {
	return st.employmentCommand(ctx, c, key, []any{"actual-duty", id, in}, func(u *unit) (out employmentResult, err error) {
		if (in.Paper == nil) == (in.Observation == nil) || !in.Action.Valid() {
			return out, f.Fail(400, "DUTY_EVIDENCE", "请选择已核实原件或登记本次出勤原件")
		}
		var proof h.DutyEvidence
		if in.Paper != nil {
			proof, err = u.receiveDuty(ctx, id, f.NewID(), *in.Paper)
			if err != nil {
				return out, err
			}
		} else {
			if !f.ValidID(in.Observation.ID) || !f.ValidID(in.Observation.Version) {
				return out, assignmentConflict()
			}
			if err = u.lockDutyAssignment(ctx, id); err != nil {
				return out, err
			}
			err = u.tx.QueryRow(ctx, `SELECT r.fact FROM staffing.duty_observation o JOIN staffing.duty_observation_head x ON (o.instance_id,o.id)=(x.instance_id,x.observation_id) JOIN staffing.duty_observation_revision r ON (r.instance_id,r.observation_id,r.version)=(x.instance_id,x.observation_id,x.version)
       WHERE o.instance_id=$1 AND o.assignment_id=$2 AND o.id=$3 AND x.version=$4 FOR SHARE OF x`, u.store.Instance, id, in.Observation.ID, in.Observation.Version).Scan(&proof)
			if errors.Is(err, pgx.ErrNoRows) {
				return out, assignmentConflict()
			}
			if err != nil {
				return out, err
			}
		}
		a, e := u.assignmentAt(ctx, id, proof.From, "")
		if e != nil {
			return out, e
		}
		if e = u.requireEmployment(ctx, "duty.correct", a.Person, a.Employer, a.Department); e != nil {
			return out, e
		}
		if e = u.dutyScope(ctx, a, proof.Position, proof.Period, "duty.correct"); e != nil {
			return out, e
		}
		if _, e = u.dutyDecision(ctx, a, proof.Dispatch); e != nil {
			return out, e
		}
		edit := recordEdit{kind: "Transfer", duty: &proof, RecordMaintenance: h.RecordMaintenance{ExpectedSource: in.Expected, BusinessDate: proof.From, Position: proof.Position, Notice: "派岗决定及核实的实际出勤原件"}}
		if in.Action != nil {
			edit.action = in.Action.Action
			edit.ExpectedRevision = in.Action.Revision
			edit.operation = "Correct"
		}
		return u.editRecord(ctx, id, edit)
	})
}

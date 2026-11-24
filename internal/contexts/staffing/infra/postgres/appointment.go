package postgres

import (
	"context"
	"errors"
	"time"

	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	f "chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
	"github.com/jackc/pgx/v5"
)

func appointmentConflict() error {
	return f.Fail(409, "APPOINTMENT_CHANGED", "任用决定已变化或与本次安排不符，请重新核对")
}

func (st *Store) PublishDecision(ctx context.Context, c f.Credentials, key string, in h.DecisionPublication) (out h.DecisionEdition, err error) {
	if !f.ValidID(key) {
		return out, f.Fail(400, "APPOINTMENT_VERSION", "发布提交号无效")
	}
	if err = in.Validate(); err != nil {
		return
	}
	err = st.within(ctx, c, true, false, func(u *unit) error {
		if e := u.requireEmployment(ctx, "appointment.issue", "", "", ""); e != nil {
			return e
		}
		if e := lock(ctx, u.tx, false, "staffing.appointment", st.Instance, in.Source); e != nil {
			return e
		}
		digest, e := hireDigest(in)
		if e != nil {
			return e
		}
		out = h.DecisionEdition{Source: in.Source, Version: key}
		var same bool
		e = u.tx.QueryRow(ctx, `SELECT actor=$4 AND digest=$5 FROM staffing.appointment_version WHERE instance_id=$1 AND source=$2 AND version=$3`, st.Instance, in.Source, key, u.guard.Principal().ID, digest).Scan(&same)
		if e == nil {
			if same {
				return nil
			}
			return appointmentConflict()
		}
		if !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		var edition int
		var head, number string
		var roster, withdrawn bool
		e = u.tx.QueryRow(ctx, `SELECT edition,version,number,roster,withdrawn FROM staffing.appointment_version
   WHERE instance_id=$1 AND source=$2 ORDER BY edition DESC LIMIT 1`, st.Instance, in.Source).Scan(&edition, &head, &number, &roster, &withdrawn)
		if e != nil && !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		if head != in.Expected || withdrawn || edition > 0 && (number != in.Number || roster != in.Roster) {
			return appointmentConflict()
		}
		for _, row := range in.Rows {
			if edition > 0 && roster {
				var moved bool
				e := u.tx.QueryRow(ctx, `SELECT EXISTS(SELECT FROM staffing.appointment_version v CROSS JOIN LATERAL jsonb_array_elements(v.members) m
     WHERE v.instance_id=$1 AND v.source=$2 AND m->>'person'=$3 AND m->>'line'<>$4)`, st.Instance, in.Source, row.Person, row.Line).Scan(&moved)
				if e != nil {
					return e
				}
				if moved {
					return appointmentConflict()
				}
			}
			if e := st.HirePeople.CheckHirePerson(dbtx.With(ctx, u.tx), c, row.Person); e != nil {
				return e
			}
		}
		if in.Rows == nil {
			in.Rows = []h.DecisionRow{}
		}
		if e := u.requireEmployment(ctx, "appointment.issue", "", "", ""); e != nil {
			return e
		}
		tag, e := u.tx.Exec(ctx, `INSERT INTO staffing.appointment_version(instance_id,source,version,edition,number,roster,withdrawn,members,digest,actor)
   VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT DO NOTHING`, st.Instance, in.Source, key, edition+1, in.Number, in.Roster, in.Withdrawn, in.Rows, digest, u.guard.Principal().ID)
		if e != nil {
			return e
		}
		if tag.RowsAffected() != 1 {
			return appointmentConflict()
		}
		_, e = u.tx.Exec(ctx, `INSERT INTO staffing.appointment_outbox VALUES($1,$2,$3)`, st.Instance, in.Source, key)
		return e
	})
	return
}

// Project only this person's original row. Each edition seals all its rows
// together, so a roster member's revision is its containing edition.
func scanDecision(row pgx.Row) (opt h.DecisionOption, err error) {
	err = row.Scan(&opt.Ref.Source, &opt.Ref.Version, &opt.Number, &opt.DecisionRow)
	if opt.Line != "" {
		opt.Ref.Line = opt.Line
		opt.Ref.LineRevision = opt.Ref.Version
	}
	return
}

const memberSelect = `SELECT v.source,v.version,v.number,m FROM versions v CROSS JOIN LATERAL jsonb_array_elements(v.members) m WHERE NOT v.withdrawn AND m->>'person'=$2`
const latestDecisions = `WITH versions AS (SELECT DISTINCT ON(source) * FROM staffing.appointment_version WHERE instance_id=$1 ORDER BY source,edition DESC) `

func targetID(v *org.EligibleEntity) string {
	if v != nil {
		return v.ID
	}
	return ""
}
func (st *Store) HireDecisions(ctx context.Context, c f.Credentials, person string, rehire *h.ActionRef) ([]h.DecisionOption, error) {
	if !f.ValidID(person) {
		return nil, f.Fail(400, "HIRE_INPUT", "人员无效")
	}
	out := []h.DecisionOption{}
	err := st.within(ctx, c, false, true, func(u *unit) error {
		if err := st.HirePeople.CheckHirePerson(dbtx.With(ctx, u.tx), c, person); err != nil {
			return err
		}
		if rehire != nil {
			if err := u.checkRehireOrigin(ctx, person, "9999-12-31", rehire); err != nil {
				return err
			}
		}
		permission := hirePermission(rehire)
		ok, err := u.employmentAllowed(ctx, permission, person, "", "", true)
		if err != nil {
			return err
		}
		if !ok {
			return f.Fail(403, "HIRE_DENIED", "没有本次办理权限")
		}
		rows, err := u.tx.Query(ctx, latestDecisions+memberSelect+" ORDER BY v.number", st.Instance, person)
		if err != nil {
			return err
		}
		opts, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (h.DecisionOption, error) { return scanDecision(r) })
		if err != nil {
			return err
		}
		for _, opt := range opts {
			ok, err := u.employmentAllowed(ctx, permission, person, targetID(opt.Employer), targetID(opt.Department), true)
			if err != nil {
				return err
			}
			if ok && opt.Position != nil {
				ent, e := u.Enterprise(ctx, false)
				if e != nil {
					return e
				}
				ok, e = u.positionReadable(ctx, ent.ID)
				if e != nil {
					return e
				}
				if ok {
					day := opt.Arrival
					if day == "" {
						day = u.now.In(f.Shanghai).Format(time.DateOnly)
					}
					term, e := hireDates(day, opt.LastDay)
					if e != nil {
						return e
					}
					ok, e = u.positionEmploymentScope(ctx, opt.Position.ID, term, person, targetID(opt.Employer), permission, "hire")
					if e != nil {
						return e
					}
				}
			}
			if ok {
				out = append(out, opt)
			}
		}
		return nil
	})
	return out, err
}

func (u *unit) CheckAppointment(ctx context.Context, _ f.Credentials, ref h.DecisionRef, a h.Appointment) error {
	if !validDecisionRef(ref) {
		return appointmentConflict()
	}
	if e := lock(ctx, u.tx, true, "staffing.appointment", u.store.Instance, ref.Source); e != nil {
		return e
	}
	opt, e := scanDecision(u.tx.QueryRow(ctx, `WITH versions AS (SELECT * FROM staffing.appointment_version WHERE instance_id=$1 AND source=$3 ORDER BY edition DESC LIMIT 1) `+memberSelect, u.store.Instance, a.Person, ref.Source))
	if errors.Is(e, pgx.ErrNoRows) {
		return appointmentConflict()
	}
	if e != nil {
		return e
	}
	for _, seg := range a.Segments {
		if opt.Department != nil && seg.DepartmentID != "" && opt.Department.ID != seg.DepartmentID || opt.Job != nil && seg.JobID != "" && opt.Job.ID != seg.JobID {
			return appointmentConflict()
		}
	}
	if opt.Position != nil && opt.Position.ID != a.Position {
		return appointmentConflict()
	}
	if opt.Ref != ref || opt.Employer != nil && opt.Employer.ID != a.Employer || opt.Department != nil && opt.Department.ID != a.Department || opt.Job != nil && opt.Job.ID != a.Job || opt.BU != nil && opt.BU.ID != a.BU || opt.Location != nil && opt.Location.ID != a.Location || opt.Arrival != "" && opt.Arrival != a.From {
		return appointmentConflict()
	}
	if opt.LastDay != "" {
		date, _ := time.Parse(time.DateOnly, opt.LastDay)
		if date.AddDate(0, 0, 1).Format(time.DateOnly) != a.Until {
			return appointmentConflict()
		}
	}
	return nil
}
func validDecisionRef(ref h.DecisionRef) bool {
	return f.ValidID(ref.Source) && f.ValidID(ref.Version) && (ref.Line == "" && ref.LineRevision == "" || f.ValidID(ref.Line) && ref.LineRevision == ref.Version)
}

func (u *unit) previousDecisionHire(ctx context.Context, in h.Hire, term f.Period) (string, error) {
	var id string
	var same bool
	err := u.tx.QueryRow(ctx, `SELECT d.action_id, e.person_id=$4 AND e.employer_id=$5 AND a.department_id=$6 AND a.job_id=$7
 AND e.period=daterange($8::date,NULLIF($9,'')::date,'[)') AND a.bu_id IS NOT DISTINCT FROM NULLIF($10,'')::uuid
 AND a.location_id IS NOT DISTINCT FROM NULLIF($11,'')::uuid
 AND coalesce(r.adoption#>>'{sources,grade,id}','')=$12 AND coalesce(r.adoption->>'trialStart','')=$13 AND (r.adoption->'rehireOf') IS NOT DISTINCT FROM $14::jsonb AND coalesce(r.adoption->>'plannedPosition','')=$15
 FROM staffing.hire_decision_use d JOIN staffing.employment_action h ON (h.instance_id,h.id)=(d.instance_id,d.action_id)
 JOIN staffing.employment_action_revision r ON (r.instance_id,r.action_id,r.revision)=(h.instance_id,h.id,h.revision)
 JOIN staffing.employment e ON (e.instance_id,e.id)=(h.instance_id,h.employment_id)
 JOIN staffing.assignment a ON (a.instance_id,a.id)=(h.instance_id,h.assignment_id)
 WHERE d.instance_id=$1 AND d.source=$2 AND d.line_id IS NOT DISTINCT FROM NULLIF($3,'')::uuid`,
		u.store.Instance, in.Decision.Source, in.Decision.Line, in.Person, in.Employer, in.Department, in.Job, term.From, term.Until, in.BU, in.Location, in.Grade, in.TrialStart, in.RehireOf, in.Position).Scan(&id, &same)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err == nil && !same {
		return "", appointmentConflict()
	}
	return id, err
}

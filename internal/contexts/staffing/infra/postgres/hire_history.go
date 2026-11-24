package postgres

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	f "chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
)

const hireActionSQL = `SELECT a.id,a.revision,a.kind,e.id AS employment,t.id AS assignment,e.person_id AS person,e.employer_id AS employer,t.department_id AS department,t.job_id AS job,
a.notice,to_char(a.recorded_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') AS recorded_at,coalesce(a.original_id::text,'') AS original,
lower(e.period)::text,coalesce(upper(e.period)::text,''),EXISTS(SELECT FROM staffing.employment_action c WHERE c.instance_id=e.instance_id AND c.employment_id=e.id AND c.kind='CancelHire')
FROM staffing.employment_action a JOIN staffing.employment e ON (e.instance_id,e.id)=(a.instance_id,a.employment_id)
JOIN staffing.assignment t ON (t.instance_id,t.id)=(a.instance_id,a.assignment_id) `

func scanHireAction(row pgx.Row) (a h.Action, err error) {
	err = row.Scan(&a.ID, &a.Revision, &a.Kind, &a.Employment, &a.Assignment, &a.Person, &a.Employer, &a.Department, &a.Job, &a.Notice, &a.RecordedAt, &a.Original, &a.From, &a.Until, &a.Cancelled)
	return
}
func (u *unit) action(ctx context.Context, id string) (h.Action, error) {
	a, err := scanHireAction(u.tx.QueryRow(ctx, hireActionSQL+" WHERE a.instance_id=$1 AND a.id=$2", u.store.Instance, id))
	if errors.Is(err, pgx.ErrNoRows) {
		err = f.Fail(404, "HIRE_UNAVAILABLE", "所选办理不可用")
	}
	return a, err
}
func (st *Store) PersonEmployment(ctx context.Context, c f.Credentials, person, day string) (out h.Overview, err error) {
	if !f.ValidID(person) || day != "" && !f.ValidDate(day) {
		return out, f.Fail(400, "HIRE_INPUT", "人员或业务日期无效")
	}
	out.Actions = []h.Action{}
	err = st.within(ctx, c, false, true, func(u *unit) error {
		today := u.now.In(f.Shanghai).Format("2006-01-02")
		if day == "" {
			day = today
		}
		var e error
		out.CanHire, e = u.employmentAllowed(ctx, "hire", person, "", "", true)
		if e != nil {
			return e
		}
		rows, e := u.tx.Query(ctx, hireActionSQL+" WHERE a.instance_id=$1 AND e.person_id=$2 ORDER BY a.recorded_at DESC,a.id", st.Instance, person)
		if e != nil {
			return e
		}
		actions, e := pgx.CollectRows(rows, func(row pgx.CollectableRow) (h.Action, error) { return scanHireAction(row) })
		if e != nil {
			return e
		}
		for _, a := range actions {
			allowed, e := u.employmentAllowed(ctx, "read", person, a.Employer, a.Department, false)
			if e != nil {
				return e
			}
			if !allowed {
				continue
			}
			if a.Kind == "Hire" && !a.Cancelled {
				if a.Includes(day) {
					var seats int
					if e := u.tx.QueryRow(ctx, `SELECT (adoption->>'seats')::int FROM staffing.assignment_segment
                    WHERE instance_id=$1 AND assignment_id=$2 AND period @> $3::date`, st.Instance, a.Assignment, day).Scan(&seats); e != nil {
						return e
					}
					out.Seats += seats
					out.People = 1
					out.Assignments++
				}
				if a.From > today {
					a.CanCancel, e = u.employmentAllowed(ctx, "cancel", person, a.Employer, a.Department, false)
					if e != nil {
						return e
					}
				}
			}
			a.CanHistory, e = u.employmentAllowed(ctx, "history", person, a.Employer, a.Department, false)
			if e != nil {
				return e
			}
			out.Actions = append(out.Actions, a)
		}
		return nil
	})
	return
}

func (st *Store) CancelHire(ctx context.Context, c f.Credentials, submission, original string, in h.Cancel) (h.Receipt, error) {
	return st.hireCommand(ctx, c, submission, []any{"CancelHire", original, in}, func(u *unit) (string, error) {
		if !f.ValidID(original) || !f.ValidID(in.Revision) || strings.TrimSpace(in.Reason) == "" || utf8.RuneCountInString(in.Reason) > 1000 {
			return "", f.Fail(400, "HIRE_INPUT", "请核对原办理并填写撤销依据")
		}
		a, err := u.action(ctx, original)
		if err != nil {
			return "", err
		}
		if err = u.requireEmployment(ctx, "cancel", a.Person, a.Employer, a.Department); err != nil {
			return "", err
		}
		if err = lock(ctx, u.tx, false, "staffing.hire.person", st.Instance, a.Person); err != nil {
			return "", err
		}
		a, err = u.action(ctx, original)
		if err != nil {
			return "", err
		}
		var today string
		err = u.tx.QueryRow(ctx, "SELECT (clock_timestamp() AT TIME ZONE 'Asia/Shanghai')::date::text").Scan(&today)
		if err != nil {
			return "", err
		}
		if a.Kind != "Hire" || a.Cancelled || a.From <= today || in.Revision != a.Revision {
			return "", f.Fail(409, "HIRE_CANCEL_UNAVAILABLE", "原办理已生效、已撤销或修订已变化")
		}
		if err = u.requireEmployment(ctx, "cancel", a.Person, a.Employer, a.Department); err != nil {
			return "", err
		}
		tag, err := u.tx.Exec(ctx, "DELETE FROM staffing.employment_primary WHERE instance_id=$1 AND action_id=$2", st.Instance, a.ID)
		if err != nil {
			return "", err
		}
		if tag.RowsAffected() != 2 {
			return "", f.Fail(409, "HIRE_DEPENDENCIES", "主选择已变化，请核对后继安排")
		}
		id := f.NewID()
		_, err = u.tx.Exec(ctx, `INSERT INTO staffing.employment_action(instance_id,id,revision,kind,employment_id,assignment_id,original_id,business_date,actor,notice,adoption)
		VALUES($1,$2,$3,'CancelHire',$4,$5,$6,$7,$8,$9,$10)`, st.Instance, id, f.NewID(), a.Employment, a.Assignment, a.ID, a.From, u.guard.Principal().ID, strings.TrimSpace(in.Reason), map[string]string{"originalRevision": a.Revision})
		return id, err
	})
}

func (u *unit) hireReceipt(ctx context.Context, key string) (h.Receipt, string, error) {
	r := h.Receipt{Submission: key}
	var digest, id string
	err := u.tx.QueryRow(ctx, `SELECT digest,outcome,coalesce(action_id::text,'') FROM staffing.hire_terminal WHERE instance_id=$1 AND actor=$2 AND submission_id=$3`, u.store.Instance, u.guard.Principal().ID, key).Scan(&digest, &r.Outcome, &id)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, "", nil
	}
	if err != nil || id == "" {
		return r, digest, err
	}
	a, err := u.action(ctx, id)
	if err != nil {
		return r, digest, err
	}
	ok, err := u.employmentAllowed(ctx, "read", a.Person, a.Employer, a.Department, false)
	if ok {
		r.Action, r.Employment, r.Assignment = id, a.Employment, a.Assignment
	}
	return r, digest, err
}
func (st *Store) ResolveHire(ctx context.Context, c f.Credentials, key string) (h.Receipt, error) {
	return st.hireCommand(ctx, c, key, nil, nil)
}
func (st *Store) hireCommand(ctx context.Context, c f.Credentials, key string, input any, work func(*unit) (string, error)) (out h.Receipt, err error) {
	if !f.ValidID(key) {
		return out, f.Fail(400, "HIRE_SUBMISSION", "提交号无效")
	}
	var failure error
	err = st.within(ctx, c, true, false, func(u *unit) error {
		actor := u.guard.Principal().ID
		if e := lock(ctx, u.tx, false, "staffing.hire.submission", st.Instance, actor, key); e != nil {
			return e
		}
		receipt, original, e := u.hireReceipt(ctx, key)
		if e != nil {
			return e
		}
		digest, e := hireDigest(input)
		if e != nil {
			return e
		}
		if receipt.Outcome != "" {
			out = receipt
			if work != nil {
				if original != digest || out.Outcome == "not_applied" {
					return f.Fail(409, "HIRE_SUBMISSION_CONFLICT", "原号已封闭或提交内容不同")
				}
				if out.Outcome == "rejected" {
					return f.Fail(409, "HIRE_REJECTED", "原提交已拒绝")
				}
			}
			return nil
		}
		var id string
		out = h.Receipt{Submission: key, Outcome: "not_applied"}
		if work != nil {
			child, e := u.tx.Begin(ctx)
			if e != nil {
				return e
			}
			business := *u
			business.tx = child
			id, e = work(&business)
			if e == nil {
				_, e = child.Exec(ctx, "INSERT INTO staffing.employment_outbox VALUES($1,$2,$3) ON CONFLICT(instance_id,action_id) DO NOTHING", st.Instance, f.NewID(), id)
			}
			if e != nil {
				if rb := child.Rollback(ctx); rb != nil {
					return rb
				}
				var fault *f.Fault
				if !errors.As(e, &fault) || fault.Status >= 500 {
					return e
				}
				id = ""
				out.Outcome = "rejected"
				failure = e
			} else {
				if e = child.Commit(ctx); e != nil {
					return e
				}
				out.Outcome = "accepted"
			}
		}
		_, e = u.tx.Exec(ctx, "INSERT INTO staffing.hire_terminal VALUES($1,$2,$3,$4,$5,NULLIF($6,'')::uuid)", st.Instance, actor, key, digest, out.Outcome, id)
		if e != nil {
			return e
		}
		out, _, e = u.hireReceipt(ctx, key)
		return e
	})
	if err == nil {
		err = failure
	}
	return
}

func (st *Store) OriginalAction(ctx context.Context, c f.Credentials, id string) (out h.Original, err error) {
	if !f.ValidID(id) {
		return out, f.Fail(404, "HIRE_UNAVAILABLE", "所选办理不可用")
	}
	err = st.within(ctx, c, false, true, func(u *unit) error {
		a, e := u.action(ctx, id)
		if e != nil {
			return e
		}
		if e = u.requireEmployment(ctx, "read", a.Person, a.Employer, a.Department); e != nil {
			return e
		}
		if e = u.requireEmployment(ctx, "history", a.Person, a.Employer, a.Department); e != nil {
			return e
		}
		e = u.tx.QueryRow(ctx, "SELECT adoption FROM staffing.employment_action WHERE instance_id=$1 AND id=$2", st.Instance, id).Scan(&out)
		if e == nil && out.Sources != nil && out.Sources.Decision != nil {
			ref := out.Sources.Decision
			opt, err := scanDecision(u.tx.QueryRow(ctx, `WITH versions AS (SELECT * FROM staffing.appointment_version WHERE instance_id=$1 AND source=$3 AND version=$4) `+memberSelect, st.Instance, a.Person, ref.Source, ref.Version))
			if err != nil {
				return err
			}
			if opt.Ref != *ref {
				return errors.New("adopted appointment row unavailable")
			}
			out.Appointment = &opt
		}
		a.CanHistory = true
		out.Action = a
		return e
	})
	return
}

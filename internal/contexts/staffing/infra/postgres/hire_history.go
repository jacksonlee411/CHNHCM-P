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

const assignmentActionSelect = `SELECT a.id,r.revision,a.kind,r.operation,e.id AS employment,t.id AS assignment,t.record_number,e.person_id AS person,e.employer_id AS employer,
coalesce(r.adoption#>>'{promotion,department}',origin.department_id::text,t.department_id::text) AS department,coalesce(origin.job_id,t.job_id) AS job,
r.business_date::text,h.version,r.notice,to_char(a.recorded_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') AS recorded_at,coalesce(a.original_id::text,'') AS original,
lower(t.period)::text,coalesce(upper(t.period)::text,''),EXISTS(SELECT FROM staffing.assignment_member cm JOIN staffing.employment_action c ON (c.instance_id,c.id)=(cm.instance_id,cm.action_id) WHERE (cm.instance_id,cm.assignment_id,cm.version)=(h.instance_id,h.assignment_id,h.version) AND c.kind='CancelHire'),r.adoption->'previous'
FROM staffing.employment_action a JOIN staffing.employment e ON (e.instance_id,e.id)=(a.instance_id,a.employment_id)
JOIN staffing.assignment t ON (t.instance_id,t.id)=(a.instance_id,a.assignment_id)
`
const hireActionSQL = assignmentActionSelect + ` JOIN staffing.assignment_head h ON (h.instance_id,h.assignment_id)=(a.instance_id,a.assignment_id) ` + assignmentRevisionJoin
const assignmentActionAtSourceSQL = assignmentActionSelect + ` JOIN staffing.assignment_publication h ON (h.instance_id,h.assignment_id)=(a.instance_id,a.assignment_id) AND h.version=$3 ` + assignmentRevisionJoin
const assignmentRevisionJoin = ` JOIN staffing.assignment_member m ON (m.instance_id,m.assignment_id,m.version,m.action_id)=(h.instance_id,h.assignment_id,h.version,a.id)
JOIN staffing.employment_action_revision r ON (r.instance_id,r.action_id,r.revision)=(m.instance_id,m.action_id,m.revision)
LEFT JOIN staffing.employment_revision_publication issued ON (issued.instance_id,issued.action_id,issued.revision)=(r.instance_id,r.action_id,r.revision)
LEFT JOIN staffing.assignment_segment origin ON (origin.instance_id,origin.assignment_id,origin.version)=(issued.instance_id,issued.assignment_id,issued.version) AND origin.period @> r.business_date `

func scanHireAction(row pgx.Row) (a h.Action, err error) {
	err = row.Scan(&a.ID, &a.Revision, &a.Kind, &a.Operation, &a.Employment, &a.Assignment, &a.RecordNumber, &a.Person, &a.Employer, &a.Department, &a.Job, &a.BusinessDate, &a.SourceVersion, &a.Notice, &a.RecordedAt, &a.Original, &a.From, &a.Until, &a.Cancelled, &a.Previous)
	a.PrimaryJob = a.RecordNumber == 0
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
	out.Terms = []h.TermLink{}
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
			if h.RelationAction(a.Kind) {
				scoped, visible, e := u.relationVisibleAssignment(ctx, a, "read")
				if e != nil {
					return e
				}
				if !visible {
					continue
				}
				_, scoped.CanHistory, e = u.relationVisibleAssignment(ctx, a, "history")
				if e != nil {
					return e
				}
				out.Actions = append(out.Actions, scoped)
				continue
			}

			if h.StartsAssignment(a.Kind) && !a.Cancelled {
				var seats int
				var department string
				e = u.tx.QueryRow(ctx, `SELECT department_id,(adoption->>'seats')::int FROM staffing.assignment_segment WHERE instance_id=$1 AND assignment_id=$2 AND period @> $3::date AND version=$4`, st.Instance, a.Assignment, day, a.SourceVersion).Scan(&department, &seats)
				active := e == nil
				if e != nil && !errors.Is(e, pgx.ErrNoRows) {
					return e
				}
				if !active {
					department = a.Department
				}
				visible, e := u.employmentAllowed(ctx, "read", person, a.Employer, department, false)
				if e != nil {
					return e
				}
				if visible {
					out.Terms = append(out.Terms, h.TermLink{Assignment: a.Assignment, RecordNumber: a.RecordNumber, From: a.From})
					if active {
						out.Seats += seats
						out.People = 1
						out.Assignments++
					}
				}
			}
			read, e := u.actionDisclosure(ctx, a, false)
			if e != nil {
				return e
			}
			if !read.from && !read.to {
				continue
			}
			history, e := u.actionDisclosure(ctx, a, true)
			if e != nil {
				return e
			}
			a.CanHistory = history.from || history.to
			if a.Kind == "Transfer" && !read.to {
				a.Department, a.Job = read.before, read.beforeJob
			}
			if a.Kind == "Hire" && !a.Cancelled && a.From > today {
				a.CanCancel, e = u.employmentAllowed(ctx, "cancel", person, a.Employer, a.Department, false)
				if e != nil {
					return e
				}
			}
			out.Actions = append(out.Actions, a)
		}
		return nil
	})
	return
}

func (st *Store) CancelHire(ctx context.Context, c f.Credentials, submission, original string, in h.Cancel) (h.Receipt, error) {
	if in.ExpectedSource != "" || in.Notice != "" || in.Basis != nil {
		return st.RecordPromotion(ctx, c, "", original, "withdraw", submission, h.PromotionCommand{ExpectedSource: in.ExpectedSource, ExpectedRevision: in.Revision, Reason: in.Reason, Notice: in.Notice, Basis: in.Basis})
	}
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
		var parallel bool
		err = u.tx.QueryRow(ctx, `SELECT EXISTS(SELECT FROM staffing.assignment x JOIN staffing.assignment_head head ON (head.instance_id,head.assignment_id)=(x.instance_id,x.id)
         JOIN staffing.assignment_segment s ON (s.instance_id,s.assignment_id,s.version)=(head.instance_id,head.assignment_id,head.version) WHERE x.instance_id=$1 AND x.person_id=$2 AND x.id<>$3 AND s.period && daterange($4::date,nullif($5,'')::date,'[)'))`, st.Instance, a.Person, a.Assignment, a.From, a.Until).Scan(&parallel)
		if err != nil {
			return "", err
		}
		if parallel {
			return "", f.Fail(409, "HIRE_DEPENDENCIES", "原期间已有关联任职，请先核对后续安排")
		}

		tag, err := u.tx.Exec(ctx, "DELETE FROM staffing.employment_primary WHERE instance_id=$1 AND action_id=$2", st.Instance, a.ID)
		if err != nil {
			return "", err
		}
		if tag.RowsAffected() != 2 {
			return "", f.Fail(409, "HIRE_DEPENDENCIES", "主选择已变化，请核对后继安排")
		}
		var hasDependent bool
		if err = u.tx.QueryRow(ctx, `SELECT EXISTS(SELECT FROM staffing.assignment_member m JOIN staffing.employment_action_revision r USING(instance_id,assignment_id,action_id,revision) WHERE m.instance_id=$1 AND m.assignment_id=$2 AND m.version=$3 AND m.action_id<>$4 AND r.operation<>'Delete')`, st.Instance, a.Assignment, a.SourceVersion, a.ID).Scan(&hasDependent); err != nil {
			return "", err
		}
		if hasDependent {
			return "", f.Fail(409, "HIRE_DEPENDENCIES", "该任职已有后续办理，请先核对")
		}
		id, revision, version := f.NewID(), f.NewID(), f.NewID()
		err = u.newEmploymentAction(ctx, employmentEntry{id: id, revision: revision, kind: "CancelHire", employment: a.Employment, assignment: a.Assignment, original: a.ID, date: a.From, notice: strings.TrimSpace(in.Reason), adoption: map[string]string{"originalRevision": a.Revision}})
		if err != nil {
			return "", err
		}
		if err = u.startAssignmentSource(ctx, a.Assignment, a.SourceVersion, version, id, revision); err != nil {
			return "", err
		}
		return id, u.publishAssignmentSource(ctx, a.Assignment, a.SourceVersion, version)
	})
}

func (u *unit) hireReceipt(ctx context.Context, key string) (h.Receipt, string, error) {
	r := h.Receipt{Submission: key}
	var digest, id, revision string
	err := u.tx.QueryRow(ctx, `SELECT digest,outcome,coalesce(action_id::text,''),coalesce(revision::text,'') FROM staffing.hire_terminal WHERE instance_id=$1 AND actor=$2 AND submission_id=$3`, u.store.Instance, u.guard.Principal().ID, key).Scan(&digest, &r.Outcome, &id, &revision)
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
	if h.RelationAction(a.Kind) {
		scoped, visible, e := u.relationVisibleAssignment(ctx, a, "read")
		if e != nil {
			return r, digest, e
		}
		if visible {
			r.Action, r.Revision, r.Assignment, r.Employment = id, revision, scoped.Assignment, a.Employment
			r.RecordNumber = &scoped.RecordNumber
		}
		return r, digest, nil
	}

	visible, err := u.actionDisclosure(ctx, a, false)
	if visible.from || visible.to {
		r.Action, r.Employment, r.Assignment = id, a.Employment, a.Assignment
		r.RecordNumber = &a.RecordNumber
		r.Revision = revision
		if a.Kind == "Promotion" {
			var other h.ActionRef
			e := u.tx.QueryRow(ctx, `SELECT transfer,transfer_revision FROM staffing.promotion_transfer WHERE instance_id=$1 AND promotion=$2 AND promotion_revision=$3`, u.store.Instance, id, revision).Scan(&other.Action, &other.Revision)
			if e == nil {
				peer, e := u.action(ctx, other.Action)
				if e != nil {
					return r, digest, e
				}
				sides, e := u.actionDisclosure(ctx, peer, false)
				if e != nil {
					return r, digest, e
				}
				if sides.from || sides.to {
					r.Related = []h.ActionRef{other}
				}
			} else if !errors.Is(e, pgx.ErrNoRows) {
				return r, digest, e
			}
		}
	}
	return r, digest, err
}
func (st *Store) ResolveHire(ctx context.Context, c f.Credentials, key string) (h.Receipt, error) {
	return st.employmentCommand(ctx, c, key, nil, nil)
}

type employmentResult struct{ action, revision string }

func (st *Store) hireCommand(ctx context.Context, c f.Credentials, key string, input any, work func(*unit) (string, error)) (h.Receipt, error) {
	return st.employmentCommand(ctx, c, key, input, func(u *unit) (employmentResult, error) {
		var result employmentResult
		id, err := work(u)
		if err != nil {
			return result, err
		}
		result.action = id
		err = u.tx.QueryRow(ctx, `SELECT revision FROM staffing.employment_action WHERE instance_id=$1 AND id=$2`, st.Instance, id).Scan(&result.revision)
		return result, err
	})
}
func (st *Store) employmentCommand(ctx context.Context, c f.Credentials, key string, input any, work func(*unit) (employmentResult, error)) (out h.Receipt, err error) {
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
		var result employmentResult
		out = h.Receipt{Submission: key, Outcome: "not_applied"}
		if work != nil {
			child, e := u.tx.Begin(ctx)
			if e != nil {
				return e
			}
			business := *u
			business.tx = child
			result, e = work(&business)
			if e == nil {
				_, e = child.Exec(ctx, "INSERT INTO staffing.employment_outbox(instance_id,event_id,action_id,revision) VALUES($1,$2,$3,$4) ON CONFLICT(instance_id,action_id,revision) DO NOTHING", st.Instance, f.NewID(), result.action, result.revision)
			}
			if e != nil {
				if rb := child.Rollback(ctx); rb != nil {
					return rb
				}
				var fault *f.Fault
				if !errors.As(e, &fault) || fault.Status >= 500 {
					return e
				}
				result = employmentResult{}
				out.Outcome = "rejected"
				failure = e
			} else {
				if e = child.Commit(ctx); e != nil {
					return e
				}
				out.Outcome = "accepted"
			}
		}
		_, e = u.tx.Exec(ctx, "INSERT INTO staffing.hire_terminal(instance_id,actor,submission_id,digest,outcome,action_id,revision) VALUES($1,$2,$3,$4,$5,NULLIF($6,'')::uuid,NULLIF($7,'')::uuid)", st.Instance, actor, key, digest, out.Outcome, result.action, result.revision)
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

func (st *Store) OriginalAction(ctx context.Context, c f.Credentials, id, revision string) (out h.Original, err error) {
	if !f.ValidID(id) || revision != "" && !f.ValidID(revision) {
		return out, f.Fail(404, "HIRE_UNAVAILABLE", "所选办理不可用")
	}
	err = st.within(ctx, c, false, true, func(u *unit) error {
		a, e := u.action(ctx, id)
		if e != nil {
			return e
		}
		if h.RelationAction(a.Kind) {
			out, e = u.relationOriginal(ctx, a, revision)
			return e
		}

		current, e := u.actionDisclosure(ctx, a, true)
		if e != nil {
			return e
		}
		if !current.from && !current.to {
			return f.Fail(403, "HIRE_DENIED", "当前权限不允许读取原件")
		}
		if revision == "" {
			revision = a.Revision
		}
		var version string
		e = u.tx.QueryRow(ctx, `SELECT version FROM staffing.employment_revision_publication WHERE instance_id=$1 AND assignment_id=$2 AND action_id=$3 AND revision=$4`, st.Instance, a.Assignment, id, revision).Scan(&version)
		if errors.Is(e, pgx.ErrNoRows) {
			return f.Fail(404, "ACTION_REVISION_UNAVAILABLE", "所选修订不可用")
		}
		if e != nil {
			return e
		}
		a, e = scanHireAction(u.tx.QueryRow(ctx, assignmentActionAtSourceSQL+" WHERE a.instance_id=$1 AND a.id=$2", st.Instance, id, version))
		if e != nil {
			return e
		}
		disclosure, e := u.actionDisclosure(ctx, a, true)
		if e != nil {
			return e
		}
		if !disclosure.from && !disclosure.to {
			return f.Fail(403, "HIRE_DENIED", "当前权限不允许读取本次修订")
		}
		rows, e := u.tx.Query(ctx, `SELECT coalesce(nullif(operator_snapshot->>'displayName',''),operator_snapshot->>'login',actor::text),revision,actor,to_char(recorded_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') FROM staffing.employment_action_revision WHERE instance_id=$1 AND action_id=$2 ORDER BY recorded_at DESC`, st.Instance, id)
		if e != nil {
			return e
		}
		revisions, e := pgx.CollectRows(rows, pgx.RowToStructByPos[h.Revision])
		if e != nil {
			return e
		}
		e = u.tx.QueryRow(ctx, "SELECT adoption FROM staffing.employment_action_revision WHERE instance_id=$1 AND action_id=$2 AND revision=$3", st.Instance, id, a.Revision).Scan(&out)
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
		if e != nil {
			return e
		}
		e = u.tx.QueryRow(ctx, `SELECT coalesce((SELECT adoption->>'departmentName' FROM staffing.assignment_segment
            WHERE instance_id=$1 AND assignment_id=$2 AND version=$3 AND period @> $4::date),'')`, st.Instance, a.Assignment, version, a.BusinessDate).Scan(&out.DepartmentName)
		if e != nil {
			return e
		}
		if out.Promotion != nil {
			out.DepartmentName = out.Promotion.DepartmentName
		}
		if out.Promotion != nil {
			out.DepartmentName = out.Promotion.DepartmentName
		}
		if out.Transfer != nil {
			transfer := out.Transfer
			if out.Promotion != nil {
				disclosure.from = true
				disclosure.to, e = u.employmentAllowed(ctx, "history", a.Person, a.Employer, transfer.After.Department, false)
				if e != nil {
					return e
				}
				read, e := u.employmentAllowed(ctx, "read", a.Person, a.Employer, transfer.After.Department, false)
				if e != nil {
					return e
				}
				disclosure.to = disclosure.to && read
			}
			if !disclosure.to {
				a.Department, a.Job = transfer.BeforeDepartment, transfer.BeforeJob
				out.DepartmentName = transfer.BeforeDepartmentName
				transfer.After = h.JobTransfer{}
				transfer.AfterGrade = nil
				if out.Promotion != nil {
					out.Promotion.After.Job = ""
					out.Promotion.After.JobRevision = ""
					out.Promotion.After.JobSource = ""
				}
			}
			if !disclosure.from {
				transfer.BeforeGrade = nil
				transfer.BeforeDepartment, transfer.BeforeJob, transfer.BeforeDepartmentName, transfer.BeforeJobName = "", "", "", ""
			}
		}
		a.CanHistory = true
		for _, rev := range revisions {
			visible, e := u.revisionVisible(ctx, a, rev.ID)
			if e != nil {
				return e
			}
			if visible {
				out.Revisions = append(out.Revisions, rev)
			}
		}
		if req := out.Requires; req != nil {
			prior, e := u.action(ctx, req.Action)
			if e != nil {
				return e
			}
			visible, e := u.revisionVisible(ctx, prior, req.Revision)
			if e != nil {
				return e
			}
			if !visible {
				out.Requires = nil
			}
		}
		if ref := out.NoticeRef; ref != nil && ref.Action != a.ID {
			prior, e := u.action(ctx, ref.Action)
			if e != nil {
				return e
			}
			ok, e := u.revisionVisible(ctx, prior, ref.Revision)
			if e != nil {
				return e
			}
			if !ok {
				out.NoticeRef = nil
			}
		}
		if out.Promotion != nil && out.Promotion.WithdrawalBasis != nil {
			ref := out.Promotion.WithdrawalBasis
			prior, e := u.action(ctx, ref.Action)
			if e != nil {
				return e
			}
			ok, e := u.revisionVisible(ctx, prior, ref.Revision)
			if e != nil {
				return e
			}
			if !ok {
				out.Promotion.WithdrawalBasis = nil
			}
		}
		if ref := out.Previous; ref != nil {
			prior, e := u.action(ctx, ref.Action)
			if e != nil {
				return e
			}
			var visible bool
			if h.RelationAction(prior.Kind) {
				_, visible, e = u.relationVisibleAssignment(ctx, prior, "history")
			} else {
				visible, e = u.revisionVisible(ctx, prior, ref.Revision)
			}
			if e != nil {
				return e
			}
			if !visible {
				out.Previous = nil
			}
		}
		out.Action = a
		return e
	})
	return
}

func (u *unit) revisionVisible(ctx context.Context, a h.Action, revision string) (bool, error) {
	item, e := scanHireAction(u.tx.QueryRow(ctx, assignmentActionSelect+`
     JOIN staffing.employment_revision_publication origin_version ON origin_version.instance_id=a.instance_id AND origin_version.action_id=a.id AND origin_version.revision=$3
     JOIN staffing.assignment_publication h ON (h.instance_id,h.assignment_id,h.version)=(origin_version.instance_id,origin_version.assignment_id,origin_version.version) `+assignmentRevisionJoin+" WHERE a.instance_id=$1 AND a.id=$2", u.store.Instance, a.ID, revision))
	if e != nil {
		return false, e
	}
	disclosure, e := u.actionDisclosure(ctx, item, true)
	return disclosure.from || disclosure.to, e
}

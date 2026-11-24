package postgres

import (
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	f "chnhcm.local/chnhcm/internal/foundation"
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"strings"
	"unicode/utf8"
)

// Counts are by relationship, while the selected assignment source controls
// which later relationship decisions were known in that edition.
func (u *unit) probationAt(ctx context.Context, a h.Action, version, day string) (state h.ProbationState, err error) {
	err = u.tx.QueryRow(ctx, `SELECT coalesce(
     (SELECT rr.adoption->>'trialStart' FROM staffing.employment_action origin JOIN staffing.employment_action_revision rr
      ON (rr.instance_id,rr.action_id,rr.revision)=(origin.instance_id,origin.id,origin.revision)
      WHERE origin.instance_id=$1 AND origin.employment_id=$2 AND origin.kind IN ('Hire','Rehire')),
     max(r.business_date::text) FILTER(WHERE evt.kind='ProbationStart'),''),
     coalesce(max(r.business_date::text) FILTER(WHERE evt.kind='ConfirmEmployment'),'')
     FROM staffing.assignment_member m JOIN staffing.employment_action evt ON (evt.instance_id,evt.id)=(m.instance_id,m.action_id)
     JOIN staffing.employment_action_revision r ON (r.instance_id,r.action_id,r.revision)=(m.instance_id,m.action_id,m.revision)
     WHERE m.instance_id=$1 AND m.assignment_id=$3 AND m.version=$4`, u.store.Instance, a.Employment, a.Assignment, version).Scan(&state.Start, &state.Confirmed)
	return state.At(day), err
}

type relationDepartment struct {
	ID, Name, Assignment string
	Primary              bool
}

func (u *unit) relationDepartments(ctx context.Context, employment, day string) ([]relationDepartment, error) {
	rows, e := u.tx.Query(ctx, `WITH daily AS (
      SELECT t.id,head.version,s.department_id,s.adoption->>'departmentName' AS name,
      EXISTS(SELECT FROM staffing.employment_primary p WHERE p.instance_id=t.instance_id AND p.layer='assignment' AND p.owner_id=t.employment_id AND p.target_id=t.id AND p.purpose='personnel_management' AND p.period @> $3::date) AS primary_job
      FROM staffing.assignment t JOIN staffing.assignment_head head ON (head.instance_id,head.assignment_id)=(t.instance_id,t.id)
      JOIN staffing.assignment_segment s ON (s.instance_id,s.assignment_id,s.version)=(head.instance_id,head.assignment_id,head.version)
      WHERE t.instance_id=$1 AND t.employment_id=$2 AND s.period @> $3::date)
      SELECT department_id,name,id,primary_job FROM daily
      UNION SELECT (r.adoption#>>'{transfer,beforeDepartment}')::uuid,r.adoption#>>'{transfer,beforeDepartmentName}',d.id,d.primary_job FROM daily d
      JOIN staffing.assignment_member m ON (m.instance_id,m.assignment_id,m.version)=($1,d.id,d.version)
      JOIN staffing.employment_action a ON (a.instance_id,a.id)=(m.instance_id,m.action_id)
      JOIN staffing.employment_action_revision r ON (r.instance_id,r.action_id,r.revision)=(m.instance_id,m.action_id,m.revision)
      WHERE a.kind='Transfer' AND r.operation<>'Delete' AND r.business_date=$3::date ORDER BY 2,1`, u.store.Instance, employment, day)
	if e != nil {
		return nil, e
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[relationDepartment])
}

func (st *Store) ProbationDepartments(ctx context.Context, c f.Credentials, id string, in h.LocationSearch) (out []org.EligibleEntity, err error) {
	if !f.ValidDate(in.BusinessDate) || in.Page < 1 || in.Page > 1000000 || utf8.RuneCountInString(in.Search) > 400 {
		return nil, f.Fail(400, "PROBATION_INPUT", "请核对日期与查询条件")
	}
	out = []org.EligibleEntity{}
	err = st.within(ctx, c, false, true, func(u *unit) error {
		a, e := u.assignmentAt(ctx, id, in.BusinessDate, "")
		if e != nil {
			return e
		}
		rows, e := u.relationDepartments(ctx, a.Employment, in.BusinessDate)
		if e != nil {
			return e
		}
		skip := (in.Page - 1) * 20
		seen := map[string]bool{}
		for _, v := range rows {
			if seen[v.ID] || !strings.Contains(strings.ToLower(v.Name), strings.ToLower(strings.TrimSpace(in.Search))) {
				continue
			}
			seen[v.ID] = true
			history, e := u.employmentAllowed(ctx, "read", a.Person, a.Employer, v.ID, false)
			if e != nil {
				return e
			}
			permitted, e := u.employmentAllowed(ctx, "probation.confirm", a.Person, a.Employer, v.ID, false)
			if e != nil {
				return e
			}
			if !history || !permitted {
				continue
			}
			if skip > 0 {
				skip--
				continue
			}
			out = append(out, org.EligibleEntity{ID: v.ID, Name: v.Name})
			if len(out) == 20 {
				break
			}
		}
		return nil
	})
	return
}

func (u *unit) probationAttribution(ctx context.Context, a h.Action, in h.ProbationCommand) (ev h.ProbationEvidence, err error) {
	departments, e := u.relationDepartments(ctx, a.Employment, in.BusinessDate)
	if e != nil {
		return ev, e
	}
	primary := map[string]string{}
	var main string
	for _, v := range departments {
		if v.Primary {
			primary[v.ID] = v.Name
			main = v.Assignment
		}
		if in.Department != "" && v.ID == in.Department {
			ev.Department, ev.DepartmentName, ev.Attribution = v.ID, v.Name, "notice"
		}
	}
	if in.Department != "" {
		if ev.Department == "" {
			return ev, f.Fail(409, "PROBATION_DEPARTMENT", "通知记载的发生部门与该关系当日事实不符")
		}
		if e = u.requireEmployment(ctx, "read", a.Person, a.Employer, ev.Department); e != nil {
			return ev, e
		}
		return ev, nil
	}
	if in.AfterAction != "" {
		act, e := u.action(ctx, in.AfterAction)
		if e != nil {
			return ev, e
		}
		if act.Assignment != main || act.Kind != "Transfer" || act.Revision != in.AfterRevision || act.Operation == "Delete" || act.BusinessDate != in.BusinessDate {
			return ev, f.Fail(409, "PROBATION_ORDER", "所引调动与关系主任职、日期或修订不符")
		}
		visible, e := u.actionDisclosure(ctx, act, true)
		if e != nil {
			return ev, e
		}
		if !visible.to {
			return ev, f.Fail(403, "HIRE_DENIED", "当前权限不可采用所引原件")
		}
		ev.Department, ev.DepartmentName, ev.Attribution = act.Department, primary[act.Department], "preceding-transfer"
		ev.AfterAction, ev.AfterRevision = act.ID, act.Revision
		return ev, nil
	}
	if len(primary) == 1 {
		for d, name := range primary {
			ev.Department, ev.DepartmentName, ev.Attribution = d, name, "relationship-primary"
		}
	} else if len(primary) == 0 {
		ev.Attribution = "primary-unavailable"
	} else {
		ev.Attribution = "same-day-order-undetermined"
	}
	return ev, nil
}

func (st *Store) RecordProbation(ctx context.Context, c f.Credentials, id, key, kind string, in h.ProbationCommand) (h.Receipt, error) {
	return st.hireCommand(ctx, c, key, []any{kind, id, in}, func(u *unit) (string, error) {
		if !f.ValidDate(in.BusinessDate) || !f.ValidID(in.ExpectedSource) || strings.TrimSpace(in.Notice) == "" || utf8.RuneCountInString(in.Notice) > 1000 || in.Department != "" && !f.ValidID(in.Department) ||
			(in.AfterAction == "") != (in.AfterRevision == "") || in.AfterAction != "" && (!f.ValidID(in.AfterAction) || !f.ValidID(in.AfterRevision) || in.Department != "") || kind == "ProbationStart" && (in.Department != "" || in.AfterAction != "") {
			return "", f.Fail(400, "PROBATION_INPUT", "请核对实际日期、所属关系和通知依据")
		}
		a, e := u.assignmentIdentity(ctx, id)
		if e != nil {
			return "", e
		}
		if e = lock(ctx, u.tx, false, "staffing.hire.person", st.Instance, a.Person); e != nil {
			return "", e
		}
		// A late-entered trial can predate the assignment through which its
		// relationship is reached. Authority is checked against a real arrangement.
		anchorDay := max(in.BusinessDate, a.From)
		a, e = u.assignmentAt(ctx, id, anchorDay, "")
		if e != nil {
			return "", e
		}
		if a.Cancelled || !a.Includes(anchorDay) || a.SourceVersion != in.ExpectedSource {
			return "", assignmentConflict()
		}
		var term f.Period
		if e = u.tx.QueryRow(ctx, `SELECT lower(e.period)::text,coalesce(x.first_inactive::text,upper(e.period)::text,'') FROM staffing.employment e LEFT JOIN staffing.employment_exit x ON (x.instance_id,x.employment_id)=(e.instance_id,e.id) WHERE e.instance_id=$1 AND e.id=$2`, st.Instance, a.Employment).Scan(&term.From, &term.Until); e != nil {
			return "", e
		}
		prior, e := u.probationAt(ctx, a, a.SourceVersion, in.BusinessDate)
		if e != nil {
			return "", e
		}
		permission := "probation.start"
		if kind == "ConfirmEmployment" {
			permission = "probation.confirm"
		}
		global, e := u.employmentAllowed(ctx, permission, a.Person, a.Employer, "", false)
		if e != nil {
			return "", e
		}
		hasGrant, e := u.employmentAllowed(ctx, permission, a.Person, a.Employer, "", true)
		if e != nil {
			return "", e
		}
		if !hasGrant {
			return "", f.Fail(403, "HIRE_DENIED", "没有该关系的办理权限")
		}
		ev := h.ProbationEvidence{Start: in.BusinessDate, Attribution: "relationship"}
		if kind == "ConfirmEmployment" {
			ev, e = u.probationAttribution(ctx, a, in)
			if e != nil {
				return "", e
			}
			ev.Start = prior.Start
			if !global && ev.Department == "" {
				return "", f.Fail(409, "PROBATION_SCOPE", "办理授权需要核定发生部门，请补足通知或实际先后")
			}
			if e = u.requireEmployment(ctx, permission, a.Person, a.Employer, ev.Department); e != nil {
				return "", e
			}
		} else if e = u.requireEmployment(ctx, permission, a.Person, a.Employer, a.Department); e != nil {
			return "", e
		}
		if e = prior.CanRecord(kind, in.BusinessDate, term); e != nil {
			return "", e
		}
		item := employmentEntry{id: f.NewID(), revision: f.NewID(), kind: kind, employment: a.Employment, assignment: id, date: in.BusinessDate, notice: strings.TrimSpace(in.Notice), adoption: map[string]any{"probation": ev}}
		if e = u.newEmploymentAction(ctx, item); e != nil {
			return "", e
		}
		rows, e := u.tx.Query(ctx, `SELECT t.id,h.version FROM staffing.assignment t JOIN staffing.assignment_head h ON (h.instance_id,h.assignment_id)=(t.instance_id,t.id) WHERE t.instance_id=$1 AND t.employment_id=$2`, st.Instance, a.Employment)
		if e != nil {
			return "", e
		}
		terms, e := pgx.CollectRows(rows, pgx.RowToStructByPos[struct{ ID, Version string }])
		if e != nil {
			return "", e
		}
		for _, t := range terms {
			version := f.NewID()
			if e = u.startAssignmentSource(ctx, t.ID, t.Version, version, item.id, item.revision); e != nil {
				return "", e
			}
			_, e = u.tx.Exec(ctx, `INSERT INTO staffing.assignment_segment(instance_id,assignment_id,version,period,location_id,location_ref,adoption,department_id,job_id,grade_ref)
              SELECT instance_id,assignment_id,$4,period,location_id,location_ref,adoption,department_id,job_id,grade_ref FROM staffing.assignment_segment WHERE instance_id=$1 AND assignment_id=$2 AND version=$3`, st.Instance, t.ID, t.Version, version)
			if e != nil {
				return "", e
			}
			if e = u.publishAssignmentSource(ctx, t.ID, t.Version, version); e != nil {
				return "", e
			}
		}
		return item.id, nil
	})
}

func (u *unit) relationVisibleAssignment(ctx context.Context, a h.Action, required string) (h.Action, bool, error) {
	// Prefer the original anchor only when that assignment is presently readable.
	scopeDay := a.BusinessDate
	if a.Kind == "Leave" {
		if e := u.tx.QueryRow(ctx, `SELECT (first_inactive-1)::text FROM staffing.employment_exit WHERE instance_id=$1 AND action_id=$2`, u.store.Instance, a.ID).Scan(&scopeDay); e != nil {
			return a, false, e
		}
	}
	rows, e := u.tx.Query(ctx, `SELECT t.id,t.record_number,coalesce(s.department_id,t.department_id),coalesce(s.job_id,t.job_id),h.version
      FROM staffing.assignment t JOIN staffing.assignment_head h ON (h.instance_id,h.assignment_id)=(t.instance_id,t.id)
      LEFT JOIN staffing.assignment_segment s ON (s.instance_id,s.assignment_id,s.version)=(h.instance_id,h.assignment_id,h.version) AND s.period @> greatest($3::date,lower(t.period))
      WHERE t.instance_id=$1 AND t.employment_id=$2 AND (t.id=$4 OR s.assignment_id IS NOT NULL) ORDER BY (t.id=$4) DESC,t.record_number`, u.store.Instance, a.Employment, scopeDay, a.Assignment)
	if e != nil {
		return a, false, e
	}
	type candidate struct {
		ID                       string
		Number                   int
		Department, Job, Version string
	}
	choices, e := pgx.CollectRows(rows, pgx.RowToStructByPos[candidate])
	if e != nil {
		return a, false, e
	}
	for _, v := range choices {
		visible, e := u.employmentAllowed(ctx, "read", a.Person, a.Employer, v.Department, false)
		if e != nil {
			return a, false, e
		}
		if !visible {
			continue
		}
		if required != "read" {
			visible, e = u.employmentAllowed(ctx, required, a.Person, a.Employer, v.Department, false)
			if e != nil {
				return a, false, e
			}
			if !visible {
				continue
			}
		}
		e = u.tx.QueryRow(ctx, `SELECT src.version FROM staffing.assignment_source src JOIN staffing.assignment_member m USING(instance_id,assignment_id,version)
           WHERE src.instance_id=$1 AND src.assignment_id=$2 AND m.action_id=$3 AND m.revision=$4 AND NOT EXISTS(SELECT FROM staffing.assignment_member prev WHERE (prev.instance_id,prev.assignment_id,prev.version,prev.action_id,prev.revision)=(m.instance_id,m.assignment_id,src.previous,m.action_id,m.revision))`, u.store.Instance, v.ID, a.ID, a.Revision).Scan(&v.Version)
		if errors.Is(e, pgx.ErrNoRows) {
			continue
		}
		if e != nil {
			return a, false, e
		}
		a.Assignment, a.RecordNumber, a.PrimaryJob, a.Department, a.Job, a.SourceVersion = v.ID, v.Number, v.Number == 0, v.Department, v.Job, v.Version
		return a, true, nil
	}
	return a, false, nil
}

func (u *unit) relationOriginal(ctx context.Context, a h.Action, revision string) (out h.Original, err error) {
	if revision == "" {
		revision = a.Revision
	}
	scoped, ok, e := u.relationVisibleAssignment(ctx, a, "history")
	if e != nil {
		return out, e
	}
	if !ok {
		return out, f.Fail(403, "HIRE_DENIED", "当前权限不允许读取关系原件")
	}
	var ev h.Original
	e = u.tx.QueryRow(ctx, `SELECT adoption FROM staffing.employment_action_revision WHERE instance_id=$1 AND action_id=$2 AND revision=$3`, u.store.Instance, a.ID, revision).Scan(&ev)
	if errors.Is(e, pgx.ErrNoRows) {
		return out, f.Fail(404, "ACTION_REVISION_UNAVAILABLE", "所选修订不可用")
	}
	if e != nil {
		return out, e
	}
	if ev.Probation != nil && ev.Probation.Department != "" {
		allowed, e := u.employmentAllowed(ctx, "read", a.Person, a.Employer, ev.Probation.Department, false)
		if e != nil {
			return out, e
		}
		if !allowed {
			ev.Probation.Department, ev.Probation.DepartmentName, ev.Probation.AfterAction, ev.Probation.AfterRevision = "", "", "", ""
		}
	}
	scoped.CanHistory = true
	ev.Action = scoped
	var rev h.Revision
	e = u.tx.QueryRow(ctx, `SELECT revision,actor,coalesce(nullif(operator_snapshot->>'displayName',''),operator_snapshot->>'login',actor::text),to_char(recorded_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') FROM staffing.employment_action_revision WHERE instance_id=$1 AND action_id=$2 AND revision=$3`, u.store.Instance, a.ID, revision).Scan(&rev.ID, &rev.Actor, &rev.Operator, &rev.RecordedAt)
	if e != nil {
		return ev, e
	}
	ev.Relation, e = u.relationshipSummary(ctx, a, scoped, true)
	if e == nil && ev.Exit != nil {
		ev.Relation.Until = ev.Exit.FirstInactive
	}

	ev.Revisions = []h.Revision{rev}
	return ev, e
}

// The full set of employment assignments stays inside M04. Only assignments
// identified by the current scope can appear as links on the relationship record.
func (u *unit) relationshipSummary(ctx context.Context, a, scoped h.Action, includeTerms bool) (*h.RelationSummary, error) {
	out := &h.RelationSummary{Terms: []h.TermLink{}}
	e := u.tx.QueryRow(ctx, `SELECT lower(e.period)::text,coalesce(upper(e.period)::text,''),coalesce(
      (SELECT x->>'entityName' FROM jsonb_array_elements(coalesce(r.adoption#>'{sources,legal}',r.adoption#>'{secondary,legal}')) x WHERE x->>'from'<=$3 AND coalesce(nullif(x->>'until',''),'9999-12-31')>$3 LIMIT 1),
      r.adoption#>>'{sources,legal,0,entityName}',r.adoption#>>'{secondary,legal,0,entityName}','')
      FROM staffing.employment e JOIN staffing.employment_action origin ON (origin.instance_id,origin.employment_id)=(e.instance_id,e.id)
      JOIN staffing.employment_action_revision r ON (r.instance_id,r.action_id,r.revision)=(origin.instance_id,origin.id,origin.revision)
      WHERE e.instance_id=$1 AND origin.assignment_id=$2 AND origin.kind IN ('Hire','AdditionalAssignment','ParallelEmployment','Rehire')`, u.store.Instance, scoped.Assignment, a.BusinessDate).Scan(&out.From, &out.Until, &out.EmployerName)
	if e != nil || !includeTerms {
		return out, e
	}
	rows, e := u.tx.Query(ctx, `SELECT t.id,t.record_number,lower(t.period)::text,s.department_id FROM staffing.assignment t JOIN staffing.assignment_head head ON (head.instance_id,head.assignment_id)=(t.instance_id,t.id)
       JOIN staffing.assignment_segment s ON (s.instance_id,s.assignment_id,s.version)=(head.instance_id,head.assignment_id,head.version)
       WHERE t.instance_id=$1 AND t.employment_id=$2 AND s.period @> $3::date ORDER BY t.record_number`, u.store.Instance, a.Employment, a.BusinessDate)
	if e != nil {
		return nil, e
	}
	type term struct {
		ID               string
		Number           int
		From, Department string
	}
	available, e := pgx.CollectRows(rows, pgx.RowToStructByPos[term])
	if e != nil {
		return nil, e
	}
	for _, v := range available {
		ok, e := u.employmentAllowed(ctx, "read", a.Person, a.Employer, v.Department, false)
		if e != nil {
			return nil, e
		}
		if ok {
			out.Terms = append(out.Terms, h.TermLink{Assignment: v.ID, RecordNumber: v.Number, From: v.From})
		}
	}
	return out, nil
}

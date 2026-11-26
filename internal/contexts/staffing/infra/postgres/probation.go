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
      WHERE r.adoption ? 'transfer' AND r.operation NOT IN ('Delete','Withdraw') AND r.business_date=$3::date ORDER BY 2,1`, u.store.Instance, employment, day)
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
	for _, v := range departments {
		if v.Primary {
			primary[v.ID] = v.Name
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
	}
	if in.AfterAction != "" {
		err = u.applyProbationPredecessor(ctx, a, in, &ev)
		return ev, err
	}
	if in.Department != "" {
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
	return st.employmentCommand(ctx, c, key, []any{kind, id, in}, func(u *unit) (result employmentResult, err error) {
		if !f.ValidDate(in.BusinessDate) || !f.ValidID(in.ExpectedSource) || strings.TrimSpace(in.Notice) == "" || utf8.RuneCountInString(in.Notice) > 1000 || in.Department != "" && !f.ValidID(in.Department) ||
			(in.AfterAction == "") != (in.AfterRevision == "") || in.AfterAction != "" && (!f.ValidID(in.AfterAction) || !f.ValidID(in.AfterRevision)) || !in.Precedes.Valid() || kind == "ProbationStart" && (in.Department != "" || in.AfterAction != "" || in.Precedes != nil) {
			return result, f.Fail(400, "PROBATION_INPUT", "请核对实际日期、所属关系和通知依据")
		}
		a, e := u.assignmentIdentity(ctx, id)
		if e != nil {
			return result, e
		}
		if e = lock(ctx, u.tx, false, "staffing.hire.person", st.Instance, a.Person); e != nil {
			return result, e
		}
		// A late-entered trial can predate the assignment through which its
		// relationship is reached. Authority is checked against a real arrangement.
		anchorDay := max(in.BusinessDate, a.From)
		a, e = u.assignmentAt(ctx, id, anchorDay, "")
		if e != nil {
			return result, e
		}
		if a.Cancelled || !a.Includes(anchorDay) || a.SourceVersion != in.ExpectedSource {
			return result, assignmentConflict()
		}
		var term f.Period
		if e = u.tx.QueryRow(ctx, `SELECT lower(e.period)::text,coalesce(x.first_inactive::text,upper(e.period)::text,'') FROM staffing.employment e LEFT JOIN staffing.employment_exit x ON (x.instance_id,x.employment_id)=(e.instance_id,e.id) WHERE e.instance_id=$1 AND e.id=$2`, st.Instance, a.Employment).Scan(&term.From, &term.Until); e != nil {
			return result, e
		}
		prior, e := u.probationAt(ctx, a, a.SourceVersion, in.BusinessDate)
		if e != nil {
			return result, e
		}
		permission := "probation.start"
		if kind == "ConfirmEmployment" {
			permission = "probation.confirm"
		}
		hasGrant, e := u.employmentAllowed(ctx, permission, a.Person, a.Employer, "", true)
		if e != nil {
			return result, e
		}
		if !hasGrant {
			return result, f.Fail(403, "HIRE_DENIED", "没有该关系的办理权限")
		}
		ev := h.ProbationEvidence{Start: in.BusinessDate, Attribution: "relationship"}
		if kind == "ConfirmEmployment" {
			ev, e = u.probationAttribution(ctx, a, in)
			if e != nil {
				return result, e
			}
			ev.Start = prior.Start
		} else if e = u.requireEmployment(ctx, permission, a.Person, a.Employer, a.Department); e != nil {
			return result, e
		}
		if kind == "ConfirmEmployment" && prior.Confirmed != "" {
			existing, err := u.currentConfirmation(ctx, a)
			if err != nil {
				return result, err
			}
			if existing.ID != "" && existing.Day == in.BusinessDate && existing.Notice == strings.TrimSpace(in.Notice) &&
				existing.Evidence.Probation.Start == prior.Start && existing.Evidence.Probation.AfterAction == in.AfterAction &&
				(existing.Evidence.Probation.Attribution == "notice") == (in.Department != "") && (in.Department == "" || existing.Evidence.Probation.Department == in.Department) {
				if err = u.confirmScope(ctx, a, existing.Evidence.Probation.Department); err != nil {
					return result, err
				}
				var namedSuccessor string
				err = u.tx.QueryRow(ctx, `SELECT coalesce((
					SELECT pr.action_id::text FROM staffing.employment_action act
					JOIN staffing.employment_action_revision first ON (first.instance_id,first.action_id,first.revision)=(act.instance_id,act.id,act.revision)
					JOIN staffing.employment_revision_publication issued ON (issued.instance_id,issued.action_id,issued.revision)=(act.instance_id,act.id,act.revision)
					JOIN staffing.assignment_member m ON (m.instance_id,m.assignment_id,m.version)=(issued.instance_id,issued.assignment_id,issued.version)
					JOIN staffing.employment_action_revision pr ON (pr.instance_id,pr.action_id,pr.revision)=(m.instance_id,m.action_id,m.revision)
					WHERE act.instance_id=$1 AND act.id=$2 AND first.operation='Insert' AND pr.adoption#>>'{previous,action}'=act.id::text), '')`, st.Instance, existing.ID).Scan(&namedSuccessor)
				if err != nil {
					return result, err
				}
				if (in.Precedes == nil) != (namedSuccessor == "") || in.Precedes != nil && in.Precedes.Action != namedSuccessor {
					return result, f.Fail(409, "ALREADY_CONFIRMED", "通知载明的后续办理与原转正安排不同")
				}
				if in.Precedes != nil {
					successor, err := u.action(ctx, in.Precedes.Action)
					if err != nil {
						return result, err
					}
					anchor := a
					anchor.Assignment = successor.Assignment
					if err = u.validatePreceding(ctx, anchor, in.BusinessDate, in.Precedes); err != nil {
						return result, err
					}
					if successor.Kind != "Promotion" || successor.Previous == nil || successor.Previous.Action != existing.ID || successor.Previous.Revision != existing.Revision {
						return result, assignmentConflict()
					}
				}
				return employmentResult{action: existing.ID, revision: existing.Revision, entry: id, reused: true}, nil
			}
			return result, f.Fail(409, "ALREADY_CONFIRMED", "转正通知或安排与已办记录不同，请从本关系的转正办理核对")
		}
		if e = prior.CanRecord(kind, in.BusinessDate, term); e != nil {
			return result, e
		}
		if in.Precedes != nil {
			if e = u.validatePreceding(ctx, a, in.BusinessDate, in.Precedes); e != nil {
				return result, e
			}
			result, e = u.insertConfirmation(ctx, a, in, ev)
			result.entry = id
			return result, e
		}
		if kind == "ConfirmEmployment" {
			if e = u.confirmScope(ctx, a, ev.Department); e != nil {
				return result, e
			}
		}
		item := employmentEntry{id: f.NewID(), revision: f.NewID(), kind: kind, employment: a.Employment, assignment: id, date: in.BusinessDate, notice: strings.TrimSpace(in.Notice), adoption: map[string]any{"probation": ev}}
		if e = u.newEmploymentAction(ctx, item); e != nil {
			return result, e
		}
		if e = u.publishRelationMembers(ctx, a, item.id, item.revision, ""); e != nil {
			return result, e
		}
		return employmentResult{action: item.id, revision: item.revision, entry: id}, nil
	})
}

func (u *unit) insertConfirmation(ctx context.Context, a h.Action, in h.ProbationCommand, ev h.ProbationEvidence) (out employmentResult, err error) {
	initial, rows, err := u.historyRows(ctx, a)
	if err != nil {
		return out, err
	}
	originals := map[string]savedRecord{}
	target := savedRecord{HistoryEntry: h.HistoryEntry{ID: f.NewID(), Revision: f.NewID(), Day: in.BusinessDate, Kind: "ConfirmEmployment"}, Owner: a.Assignment, Operation: "Insert", Notice: strings.TrimSpace(in.Notice), Evidence: h.Original{Probation: &ev}}
	if in.AfterAction != "" {
		target.Previous = &h.ActionRef{Action: in.AfterAction, Revision: in.AfterRevision}
	}
	found := false
	for i, row := range rows {
		originals[row.ID] = row
		if row.ID != in.Precedes.Action {
			continue
		}
		if row.Kind != "Promotion" || row.Removed || row.Day != target.Day || row.Revision != in.Precedes.Revision {
			return out, assignmentConflict()
		}
		if row.Previous != nil {
			if target.Previous != nil && *target.Previous != *row.Previous {
				return out, assignmentConflict()
			}
			target.Previous = row.Previous
		}
		rows[i].Previous = &h.ActionRef{Action: target.ID, Revision: target.Revision}
		found = true
	}
	if !found {
		return out, assignmentConflict()
	}
	rows = append(rows, target)
	return u.publishRecordEdit(ctx, a, initial, rows, originals, len(rows)-1, "probation.confirm", "")
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
	_, ok, e := u.relationVisibleAssignment(ctx, a, "history")
	if e != nil {
		return out, e
	}
	if !ok {
		return out, f.Fail(403, "HIRE_DENIED", "当前权限不允许读取关系原件")
	}
	var ev h.Original
	e = u.tx.QueryRow(ctx, `SELECT adoption,operation,notice FROM staffing.employment_action_revision WHERE instance_id=$1 AND action_id=$2 AND revision=$3`, u.store.Instance, a.ID, revision).Scan(&ev, &a.Operation, &a.Notice)
	if errors.Is(e, pgx.ErrNoRows) {
		return out, f.Fail(404, "ACTION_REVISION_UNAVAILABLE", "所选修订不可用")
	}
	if e != nil {
		return out, e
	}
	a.Revision = revision
	scoped, ok, e := u.relationVisibleAssignment(ctx, a, "history")
	if e != nil {
		return out, e
	}
	if !ok {
		return out, f.Fail(403, "HIRE_DENIED", "当前权限不允许读取这份关系修订")
	}
	if ev.Probation != nil && ev.Probation.AfterAction != "" {
		var evidence h.Original
		err := u.tx.QueryRow(ctx, `SELECT adoption FROM staffing.employment_action_revision
			WHERE instance_id=$1 AND action_id=$2 AND revision=$3`, u.store.Instance, ev.Probation.AfterAction, ev.Probation.AfterRevision).Scan(&evidence)
		if err != nil {
			return out, err
		}
		visible, err := u.canReadProbationPredecessor(ctx, a, evidence)
		if err != nil {
			return out, err
		}
		if !visible {
			ev.Probation.AfterAction, ev.Probation.AfterRevision = "", ""
		}
	}
	if ev.Previous != nil {
		prior, err := u.action(ctx, ev.Previous.Action)
		if err != nil {
			return out, err
		}
		visible, err := u.revisionVisible(ctx, prior, ev.Previous.Revision)
		if err != nil {
			return out, err
		}
		if !visible {
			ev.Previous = nil
		}
	}
	if ev.Probation != nil && ev.Probation.Department != "" {
		allowed, e := u.employmentAllowed(ctx, "read", a.Person, a.Employer, ev.Probation.Department, false)
		if e != nil {
			return out, e
		}
		if !allowed {
			ev.Probation.Department, ev.Probation.DepartmentName, ev.Probation.AfterAction, ev.Probation.AfterRevision = "", "", "", ""
			ev.Probation.DepartmentSource, ev.Probation.DepartmentRevision = "", ""
		}
	}
	scoped.CanHistory = true
	ev.Action = scoped
	if a.Kind == "Leave" {
		var department, name string
		e = u.tx.QueryRow(ctx, `SELECT coalesce(r.adoption->>'department',''),coalesce(exit_department.name,'')
			FROM staffing.employment_action a JOIN staffing.employment_action_revision r ON (r.instance_id,r.action_id)=(a.instance_id,a.id)
			JOIN staffing.employment_revision_publication issued ON (issued.instance_id,issued.action_id,issued.revision)=(r.instance_id,r.action_id,r.revision) `+exitDepartmentJoin+`
			WHERE a.instance_id=$1 AND a.id=$2 AND r.revision=$3`, u.store.Instance, a.ID, revision).Scan(&department, &name)
		if e != nil {
			return out, e
		}
		ev.Action.Department = ""
		if department != "" {
			visible, err := u.employmentAllowed(ctx, "history", a.Person, a.Employer, department, false)
			if err != nil {
				return out, err
			}
			read, err := u.employmentAllowed(ctx, "read", a.Person, a.Employer, department, false)
			if err != nil {
				return out, err
			}
			if read && visible {
				ev.Action.Department = department
				ev.DepartmentName = name
			}
		}
	}
	rows, e := u.tx.Query(ctx, `SELECT coalesce(nullif(operator_snapshot->>'displayName',''),operator_snapshot->>'login',actor::text),revision,actor,to_char(recorded_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') FROM staffing.employment_action_revision WHERE instance_id=$1 AND action_id=$2 ORDER BY recorded_at DESC`, u.store.Instance, a.ID)
	if e != nil {
		return out, e
	}
	revisions, e := pgx.CollectRows(rows, pgx.RowToStructByPos[h.Revision])
	if e != nil {
		return out, e
	}
	for _, rev := range revisions {
		if rev.ID != revision {
			other := a
			other.Revision = rev.ID
			_, visible, err := u.relationVisibleAssignment(ctx, other, "history")
			if err != nil {
				return out, err
			}
			if !visible {
				continue
			}
		}
		ev.Revisions = append(ev.Revisions, rev)
	}

	ev.Relation, e = u.relationshipSummary(ctx, a, scoped, true)
	if e == nil && ev.Exit != nil {
		ev.Relation.Until = ev.Exit.FirstInactive
	}

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
	out.Terms, e = u.relationshipTerms(ctx, a, false)
	return out, e
}

func (u *unit) relationshipTerms(ctx context.Context, a h.Action, onlyOpen bool) ([]h.TermLink, error) {
	rows, e := u.tx.Query(ctx, `SELECT t.id,t.record_number,lower(t.period)::text,s.department_id FROM staffing.assignment t JOIN staffing.assignment_head head ON (head.instance_id,head.assignment_id)=(t.instance_id,t.id)
       JOIN staffing.assignment_segment s ON (s.instance_id,s.assignment_id,s.version)=(head.instance_id,head.assignment_id,head.version)
       WHERE t.instance_id=$1 AND t.employment_id=$2 AND s.period @> $3::date
       AND (NOT $4 OR NOT EXISTS (SELECT FROM staffing.assignment_exit x WHERE (x.instance_id,x.assignment_id)=(t.instance_id,t.id)))
       ORDER BY t.record_number`, u.store.Instance, a.Employment, a.BusinessDate, onlyOpen)
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
	terms := []h.TermLink{}
	for _, v := range available {
		ok, e := u.employmentAllowed(ctx, "read", a.Person, a.Employer, v.Department, false)
		if e != nil {
			return nil, e
		}
		if ok {
			terms = append(terms, h.TermLink{Assignment: v.ID, RecordNumber: v.Number, From: v.From})
		}
	}
	return terms, nil
}

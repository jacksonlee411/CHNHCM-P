package postgres

import (
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	app "chnhcm.local/chnhcm/internal/contexts/staffing/application"
	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	staff "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	f "chnhcm.local/chnhcm/internal/foundation"
	"context"
	"github.com/jackc/pgx/v5"
	"strings"
	"unicode/utf8"
)

func secondaryPermission(parallel bool) string {
	if parallel {
		return "employment.parallel"
	}
	return "assignment.add"
}

// The person lock also protects primary appointment, withdrawal and all positive
// numbers ever issued to this person, including inactive or inaccessible records.
func (st *Store) AddSecondary(ctx context.Context, c f.Credentials, id, key string, in h.Secondary) (h.Receipt, error) {
	return st.employmentCommand(ctx, c, key, []any{"AddSecondary", id, in}, func(u *unit) (employmentResult, error) {
		if err := in.Validate(); err != nil {
			return employmentResult{}, err
		}
		term, e := hireDates(in.From, in.LastDay)
		if e != nil {
			return employmentResult{}, e
		}
		a, e := u.assignmentIdentity(ctx, id)
		if e != nil {
			return employmentResult{}, e
		}
		if e = lock(ctx, u.tx, false, "staffing.hire.person", st.Instance, a.Person); e != nil {
			return employmentResult{}, e
		}
		a, e = u.assignmentAt(ctx, id, in.From, "")
		if e != nil {
			return employmentResult{}, e
		}
		permit := secondaryPermission(in.Parallel)
		if e = u.requireEmployment(ctx, permit, a.Person, a.Employer, a.Department); e != nil {
			return employmentResult{}, e
		}
		if a.Cancelled || !a.Includes(in.From) || a.SourceVersion != in.Expected {
			return employmentResult{}, assignmentConflict()
		}
		employer, employment, kind := a.Employer, a.Employment, "AdditionalAssignment"
		if in.Parallel {
			employer, employment, kind = in.Employer, f.NewID(), "ParallelEmployment"
		}
		if in.Position != "" {
			ok, e := u.positionEmploymentScope(ctx, in.Position, term, a.Person, employer, permit, "secondary")
			if e != nil {
				return employmentResult{}, e
			}
			if !ok {
				return employmentResult{}, f.Fail(403, "SECONDARY_DENIED", "所选岗位不在兼岗授权范围内")
			}
		} else {
			if e = u.requireEmployment(ctx, permit, a.Person, employer, in.Department); e != nil {
				return employmentResult{}, e
			}
			if e = u.requireEmployment(ctx, "read", a.Person, employer, in.Department); e != nil {
				return employmentResult{}, e
			}

		}

		match, e := u.originalSecondary(ctx, a, in, employer, kind, term)
		if e != nil || match.reused {
			return match, e
		}

		var covered bool
		// Both the human-level primary and the source relationship are actual
		// facts. An independent relationship does not become the person's primary.
		e = u.tx.QueryRow(ctx, `SELECT coalesce((SELECT range_agg(s.period) @> daterange($4::date,nullif($5,'')::date,'[)')
            FROM staffing.assignment t JOIN staffing.assignment_head head ON (head.instance_id,head.assignment_id)=(t.instance_id,t.id)
            JOIN staffing.assignment_segment s ON (s.instance_id,s.assignment_id,s.version)=(head.instance_id,head.assignment_id,head.version)
            WHERE t.instance_id=$1 AND t.person_id=$2 AND t.record_number=0),false)
            AND ($6 OR EXISTS(SELECT FROM staffing.employment WHERE instance_id=$1 AND id=$3 AND period @> daterange($4::date,nullif($5,'')::date,'[)') AND NOT EXISTS(SELECT FROM staffing.employment_exit x WHERE x.instance_id=$1 AND x.employment_id=$3 AND ($5='' OR x.first_inactive<nullif($5,'')::date))))`,
			st.Instance, a.Person, a.Employment, term.From, term.Until, in.Parallel).Scan(&covered)
		if e != nil {
			return employmentResult{}, e
		}
		if !covered {
			return employmentResult{}, f.Fail(409, "SECONDARY_PERIOD", "兼岗期间缺少必要的主岗或所属工作关系")
		}
		ent, e := u.Enterprise(ctx, true)
		if e != nil {
			return employmentResult{}, e
		}
		sourceCtx := u.transactionContext(ctx)
		var adoptedLocation *h.LocationRef
		if in.Location != "" {
			adoptedLocation, e = u.adoptHistoryLocation(ctx, in.Location, term)
			if e != nil {
				return employmentResult{}, e
			}
		}
		if in.BU != "" {
			choices, e := u.PositionSources(ctx, "buId", []string{in.BU}, true)
			if e != nil {
				return employmentResult{}, e
			}
			if len(choices) != 1 {
				return employmentResult{}, f.Fail(409, "SECONDARY_BU", "业务单元不可用于拟任期间")
			}
			label, e := app.HireTargetLabel(ctx, u, "buId", choices[0], term)
			if e != nil {
				return employmentResult{}, e
			}
			if label == "" {
				return employmentResult{}, f.Fail(409, "SECONDARY_BU", "业务单元不可用于拟任期间")
			}
		}
		legal, e := st.HirePurposes.CheckPurpose(sourceCtx, c, employer, org.PurposeCheck{Role: "employer", Period: term, BU: in.BU, Location: in.Location})
		if e != nil {
			return employmentResult{}, e
		}
		segments, e := st.CheckAssignment(sourceCtx, c, staff.AssignmentCheck{Period: term, Enterprise: ent.ID, Category: "Employee", Purpose: "secondary", Department: in.Department, Job: in.Job, Position: in.Position})
		if e != nil {
			return employmentResult{}, e
		}
		if len(segments) == 0 {
			return employmentResult{}, f.Fail(409, "SECONDARY_PERIOD", "所选期间缺少可用任职安排")
		}
		var adoptedGrade *h.GradeRef
		for i, seg := range segments {
			dept, job := in.Department, in.Job
			if seg.DepartmentID != "" {
				dept, job = seg.DepartmentID, seg.JobID
			}
			if e = u.requireEmployment(ctx, permit, a.Person, employer, dept); e != nil {
				return employmentResult{}, e
			}
			if e = u.requireEmployment(ctx, "read", a.Person, employer, dept); e != nil {
				return employmentResult{}, e
			}
			if e = u.checkPlacementGrade(ctx, seg, in.Grade); e != nil {
				return employmentResult{}, e
			}
			if in.Grade != "" {
				grade, e := u.gradeForJob(ctx, job, in.Grade, seg.Period)
				if e != nil {
					return employmentResult{}, e
				}
				if i == 0 {
					adoptedGrade = grade
				}
			}
		}
		if segments[0].DepartmentID != "" {
			in.Department, in.Job = segments[0].DepartmentID, segments[0].JobID
		}
		if in.Parallel {
			_, e = u.tx.Exec(ctx, `INSERT INTO staffing.employment(instance_id,id,person_id,employer_id,relationship,period) VALUES($1,$2,$3,$4,'Employee',daterange($5::date,nullif($6,'')::date,'[)'))`, st.Instance, employment, a.Person, employer, term.From, term.Until)
			if e != nil {
				return employmentResult{}, e
			}
		}
		var number int
		e = u.tx.QueryRow(ctx, `INSERT INTO staffing.employment_record(instance_id,person_id,record_number)
          SELECT $1,$2,coalesce(max(record_number),0)+1 FROM staffing.employment_record WHERE instance_id=$1 AND person_id=$2 RETURNING record_number`, st.Instance, a.Person).Scan(&number)
		if e != nil {
			return employmentResult{}, e
		}
		item := employmentEntry{id: f.NewID(), revision: f.NewID(), kind: kind, employment: employment, assignment: f.NewID(), date: term.From, notice: strings.TrimSpace(in.Notice),
			adoption: map[string]any{"secondary": h.SecondaryEvidence{Grade: adoptedGrade, RelationshipPrimary: in.Parallel, Legal: legal, Segments: segments}}}
		input := h.Hire{Person: a.Person, Department: in.Department, Job: in.Job, BU: in.BU, Location: in.Location}
		if e = u.openAssignment(ctx, item, input, number, term, segments, adoptedLocation, adoptedGrade); e != nil {
			return employmentResult{}, e
		}
		if in.Parallel {
			result, e := u.tx.Exec(ctx, `INSERT INTO staffing.employment_primary(instance_id,action_id,layer,owner_id,target_id,purpose,period)
                SELECT $1,$2,'assignment',$3,$4,p.purpose,daterange($5::date,nullif($6,'')::date,'[)') FROM staffing.employment_primary p WHERE p.instance_id=$1 AND p.layer='employment' AND p.owner_id=$7 AND p.period @> $5::date`, st.Instance, item.id, employment, item.assignment, term.From, term.Until, a.Person)
			if e != nil {
				return employmentResult{}, e
			}
			if result.RowsAffected() == 0 {
				return employmentResult{}, f.Fail(409, "SECONDARY_PRIMARY", "没有适用的关系主任职用途")
			}
		}
		return employmentResult{action: item.id, revision: item.revision}, nil
	})
}

// Compare the original appointment and its immutable adoption while holding the
// person's numbering lock. A later transfer or end preserves this identity.
func (u *unit) originalSecondary(ctx context.Context, anchor h.Action, in h.Secondary, employer, kind string, term f.Period) (employmentResult, error) {
	rows, err := u.tx.Query(ctx, `SELECT act.id,act.revision,t.department_id,t.job_id,NOT EXISTS (
	    SELECT FROM jsonb_array_elements(r.adoption#>'{secondary,segments}') seg WHERE coalesce(seg->>'position','')='')
	FROM staffing.assignment t
	JOIN staffing.employment e ON (e.instance_id,e.id)=(t.instance_id,t.employment_id)
	JOIN staffing.employment_action act ON (act.instance_id,act.assignment_id)=(t.instance_id,t.id)
	JOIN staffing.employment_action_revision r ON (r.instance_id,r.action_id,r.revision)=(act.instance_id,act.id,act.revision)
	WHERE t.instance_id=$1 AND t.person_id=$2 AND act.kind=$3 AND e.employer_id=$4
	AND ($3='ParallelEmployment' OR t.employment_id=$5)
	AND t.period=daterange($6::date,nullif($7,'')::date,'[)')
	AND t.bu_id IS NOT DISTINCT FROM nullif($8,'')::uuid AND t.location_id IS NOT DISTINCT FROM nullif($9,'')::uuid
	AND coalesce(r.adoption#>>'{secondary,grade,id}','')=$10
	AND coalesce((SELECT max(nullif(seg->>'position','')) FROM jsonb_array_elements(r.adoption#>'{secondary,segments}') seg),'')=$11
	AND NOT EXISTS (SELECT FROM jsonb_array_elements(r.adoption#>'{secondary,segments}') seg
		WHERE coalesce(seg->>'position','')='' AND (coalesce(nullif(seg->>'departmentId',''),t.department_id::text)<>$12
		OR coalesce(nullif(seg->>'jobId',''),t.job_id::text)<>$13)) LIMIT 2`,
		u.store.Instance, anchor.Person, kind, employer, anchor.Employment, term.From, term.Until, in.BU, in.Location, in.Grade, in.Position, in.Department, in.Job)
	if err != nil {
		return employmentResult{}, err
	}
	type original struct {
		employmentResult
		department, job string
		allByPosition   bool
	}
	matches, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (r original, e error) {
		r.reused = true
		e = row.Scan(&r.action, &r.revision, &r.department, &r.job, &r.allByPosition)
		return
	})
	if err != nil || len(matches) == 0 {
		return employmentResult{}, err
	}
	// The match above requires this position to occur in the original adopted
	// segments, including an appointment that also contained job-only periods.
	if in.Position != "" {
		enterprise, err := u.Enterprise(ctx, false)
		if err != nil {
			return employmentResult{}, err
		}
		visible, err := u.positionReadable(ctx, enterprise.ID)
		if err != nil {
			return employmentResult{}, err
		}
		if !visible {
			return employmentResult{}, f.Fail(403, "ACCESS_DENIED", "当前权限无法采用岗位安排")
		}
	}
	if len(matches) == 1 {
		a, e := u.action(ctx, matches[0].action)
		if e != nil {
			return employmentResult{}, e
		}
		disclosure, e := u.actionDisclosure(ctx, a, false)
		if e != nil {
			return employmentResult{}, e
		}
		if disclosure.to {
			rows, e := u.tx.Query(ctx, `SELECT DISTINCT coalesce(nullif(seg->>'departmentId',''),t.department_id::text)
			  FROM staffing.employment_action_revision r JOIN staffing.assignment t ON (t.instance_id,t.id)=(r.instance_id,r.assignment_id)
			  CROSS JOIN LATERAL jsonb_array_elements(r.adoption#>'{secondary,segments}') seg
			  WHERE r.instance_id=$1 AND r.action_id=$2 AND r.revision=$3`, u.store.Instance, a.ID, matches[0].revision)
			if e != nil {
				return employmentResult{}, e
			}
			departments, e := pgx.CollectRows(rows, pgx.RowTo[string])
			if e != nil {
				return employmentResult{}, e
			}
			for _, dept := range departments {
				for _, perm := range []string{"read", secondaryPermission(in.Parallel)} {
					if e = u.requireEmployment(ctx, perm, a.Person, a.Employer, dept); e != nil {
						return employmentResult{}, e
					}
				}
			}
			match := matches[0]
			if match.allByPosition && positionTargetsConflict(in.Department, in.Job, match.department, match.job) {
				return employmentResult{}, f.Fail(409, "POSITION_TARGET_CHANGED", "岗位归属与拟任安排不符")
			}
			return match.employmentResult, nil
		}
	}
	return employmentResult{}, f.Fail(409, "SECONDARY_EXISTING", "请由获权经办人核对既有兼任安排")
}

func (st *Store) SecondaryOptions(ctx context.Context, c f.Credentials, id string, in h.SecondaryOption) (out []org.EligibleEntity, err error) {
	term, e := hireDates(in.From, in.LastDay)
	if e != nil {
		return nil, e
	}
	field := map[string]string{"position": "positionId", "grade": "gradeId", "employer": "employer", "department": "departmentId", "job": "jobId", "location": "locationId", "bu": "buId"}[in.Kind]
	if field == "" || field == "gradeId" && !f.ValidID(in.Job) || (field == "employer" && !in.Parallel) || in.Page < 1 || in.Page > 1000000 || utf8.RuneCountInString(in.Search) > 400 || in.Department != "" && !f.ValidID(in.Department) || in.Employer != "" && !f.ValidID(in.Employer) {
		return nil, f.Fail(400, "SECONDARY_INPUT", "请核对候选查询")
	}
	out = []org.EligibleEntity{}
	err = st.within(ctx, c, false, true, func(u *unit) error {
		a, e := u.assignmentAt(ctx, id, term.From, "")
		if e != nil {
			return e
		}
		if a.Cancelled || !a.Includes(term.From) {
			return f.Fail(409, "SECONDARY_PERIOD", "所属任职在开始日无效")
		}
		permission := secondaryPermission(in.Parallel)
		if e = u.requireEmployment(ctx, permission, a.Person, a.Employer, a.Department); e != nil {
			return e
		}
		if field == "employer" {
			return u.employerTargets(ctx, term, h.EmployerSearch{BU: in.BU, Location: in.Location, Person: a.Person, Department: in.Department, Search: in.Search, Page: in.Page}, permission, &out)
		}
		employer := a.Employer
		if in.Parallel {
			employer = in.Employer
		}
		allowed, e := u.employmentAllowed(ctx, permission, a.Person, employer, in.Department, true)
		if e != nil {
			return e
		}
		if !allowed {
			return f.Fail(403, "SECONDARY_DENIED", "当前安排不在授权范围内")
		}
		query := h.TargetSearch{Person: a.Person, Department: in.Department, Job: in.Job, Employer: employer, Page: in.Page, Search: in.Search}
		if field == "positionId" {
			return u.positionTargets(ctx, term, query, permission, "secondary", nil, &out)
		}
		return u.arrangementTargets(ctx, term, query, field, permission, &out)
	})
	return
}

func (st *Store) SecondaryArrangement(ctx context.Context, c f.Credentials, id string, in h.SecondaryArrangementQuery) (out h.ArrangementMode, err error) {
	term, e := hireDates(in.From, in.LastDay)
	if e != nil {
		return out, e
	}
	err = st.within(ctx, c, false, true, func(u *unit) error {
		a, e := u.assignmentAt(ctx, id, term.From, "")
		if e != nil {
			return e
		}
		if a.Cancelled || !a.Includes(term.From) {
			return f.Fail(409, "SECONDARY_PERIOD", "所属任职在开始日无效")
		}
		permission := secondaryPermission(in.Parallel)
		if e = u.requireEmployment(ctx, permission, a.Person, a.Employer, a.Department); e != nil {
			return e
		}
		employer := a.Employer
		if in.Parallel {
			employer = in.Employer
		}
		return u.arrangementMode(ctx, term, a.Person, employer, in.Position, permission, "secondary", nil, &out)
	})
	return
}

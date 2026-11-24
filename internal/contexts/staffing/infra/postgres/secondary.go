package postgres

import (
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	app "chnhcm.local/chnhcm/internal/contexts/staffing/application"
	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	staff "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	f "chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
	"context"
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
	return st.hireCommand(ctx, c, key, []any{"AddSecondary", id, in}, func(u *unit) (string, error) {
		if err := in.Validate(); err != nil {
			return "", err
		}
		term, e := hireDates(in.From, in.LastDay)
		if e != nil {
			return "", e
		}
		a, e := u.assignmentIdentity(ctx, id)
		if e != nil {
			return "", e
		}
		if e = lock(ctx, u.tx, false, "staffing.hire.person", st.Instance, a.Person); e != nil {
			return "", e
		}
		a, e = u.assignmentAt(ctx, id, in.From, "")
		if e != nil {
			return "", e
		}
		permit := secondaryPermission(in.Parallel)
		if e = u.requireEmployment(ctx, permit, a.Person, a.Employer, a.Department); e != nil {
			return "", e
		}
		if a.Cancelled || !a.Includes(in.From) || a.SourceVersion != in.Expected {
			return "", assignmentConflict()
		}
		employer, employment, kind := a.Employer, a.Employment, "AdditionalAssignment"
		if in.Parallel {
			employer, employment, kind = in.Employer, f.NewID(), "ParallelEmployment"
		}
		if e = u.requireEmployment(ctx, permit, a.Person, employer, in.Department); e != nil {
			return "", e
		}
		if e = u.requireEmployment(ctx, "read", a.Person, employer, in.Department); e != nil {
			return "", e
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
			return "", e
		}
		if !covered {
			return "", f.Fail(409, "SECONDARY_PERIOD", "兼岗期间缺少必要的主岗或所属工作关系")
		}
		ent, e := u.Enterprise(ctx, true)
		if e != nil {
			return "", e
		}
		sourceCtx := dbtx.With(ctx, u.tx)
		var adoptedLocation *h.LocationRef
		if in.Location != "" {
			adoptedLocation, e = u.adoptHistoryLocation(ctx, in.Location, term)
			if e != nil {
				return "", e
			}
		}
		if in.BU != "" {
			choices, e := u.PositionSources(ctx, "buId", []string{in.BU}, true)
			if e != nil {
				return "", e
			}
			if len(choices) != 1 {
				return "", f.Fail(409, "SECONDARY_BU", "业务单元不可用于拟任期间")
			}
			label, e := app.HireTargetLabel(ctx, u, "buId", choices[0], term)
			if e != nil {
				return "", e
			}
			if label == "" {
				return "", f.Fail(409, "SECONDARY_BU", "业务单元不可用于拟任期间")
			}
		}
		legal, e := st.HirePurposes.CheckPurpose(sourceCtx, c, employer, org.PurposeCheck{Role: "employer", Period: term, BU: in.BU, Location: in.Location})
		if e != nil {
			return "", e
		}
		segments, e := st.CheckAssignment(sourceCtx, c, staff.AssignmentCheck{Period: term, Enterprise: ent.ID, Category: "Employee", Purpose: "secondary", Department: in.Department, Job: in.Job})
		if e != nil {
			return "", e
		}
		if in.Parallel {
			_, e = u.tx.Exec(ctx, `INSERT INTO staffing.employment(instance_id,id,person_id,employer_id,relationship,period) VALUES($1,$2,$3,$4,'Employee',daterange($5::date,nullif($6,'')::date,'[)'))`, st.Instance, employment, a.Person, employer, term.From, term.Until)
			if e != nil {
				return "", e
			}
		}
		var number int
		e = u.tx.QueryRow(ctx, `INSERT INTO staffing.employment_record(instance_id,person_id,record_number)
          SELECT $1,$2,coalesce(max(record_number),0)+1 FROM staffing.employment_record WHERE instance_id=$1 AND person_id=$2 RETURNING record_number`, st.Instance, a.Person).Scan(&number)
		if e != nil {
			return "", e
		}
		item := employmentEntry{id: f.NewID(), revision: f.NewID(), kind: kind, employment: employment, assignment: f.NewID(), date: term.From, notice: strings.TrimSpace(in.Notice),
			adoption: map[string]any{"secondary": h.SecondaryEvidence{RelationshipPrimary: in.Parallel, Legal: legal, Segments: segments}}}
		input := h.Hire{Person: a.Person, Department: in.Department, Job: in.Job, BU: in.BU, Location: in.Location}
		if e = u.openAssignment(ctx, item, input, number, term, segments, adoptedLocation); e != nil {
			return "", e
		}
		if in.Parallel {
			result, e := u.tx.Exec(ctx, `INSERT INTO staffing.employment_primary(instance_id,action_id,layer,owner_id,target_id,purpose,period)
                SELECT $1,$2,'assignment',$3,$4,p.purpose,daterange($5::date,nullif($6,'')::date,'[)') FROM staffing.employment_primary p WHERE p.instance_id=$1 AND p.layer='employment' AND p.owner_id=$7 AND p.period @> $5::date`, st.Instance, item.id, employment, item.assignment, term.From, term.Until, a.Person)
			if e != nil {
				return "", e
			}
			if result.RowsAffected() == 0 {
				return "", f.Fail(409, "SECONDARY_PRIMARY", "没有适用的关系主任职用途")
			}
		}
		return item.id, nil
	})
}

func (st *Store) SecondaryOptions(ctx context.Context, c f.Credentials, id string, in h.SecondaryOption) (out []org.EligibleEntity, err error) {
	term, e := hireDates(in.From, in.LastDay)
	if e != nil {
		return nil, e
	}
	field := map[string]string{"employer": "employer", "department": "departmentId", "job": "jobId", "location": "locationId", "bu": "buId"}[in.Kind]
	if field == "" || (field == "employer" && !in.Parallel) || in.Page < 1 || in.Page > 1000000 || utf8.RuneCountInString(in.Search) > 400 || in.Department != "" && !f.ValidID(in.Department) || in.Employer != "" && !f.ValidID(in.Employer) {
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
		return u.arrangementTargets(ctx, term, h.TargetSearch{Person: a.Person, Department: in.Department, Employer: employer, Page: in.Page, Search: in.Search}, field, permission, &out)
	})
	return
}

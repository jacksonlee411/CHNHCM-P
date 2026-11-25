package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"

	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	app "chnhcm.local/chnhcm/internal/contexts/staffing/application"
	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	staff "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	f "chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
	"github.com/jackc/pgx/v5"
)

func (u *unit) employmentAllowed(ctx context.Context, action, person, employer, department string, anyTarget bool, fields ...string) (bool, error) {
	if right, ok := strings.CutPrefix(action, "offer."); ok {
		return u.store.OfferAllowed(ctx, u.tx, u.guard.Principal().ID, right, person, employer, department, anyTarget)
	}
	return u.store.EmploymentAllowed(ctx, u.tx, u.guard.Principal().ID, action, person, employer, department, anyTarget, fields...)
}
func (u *unit) requireEmployment(ctx context.Context, action, person, employer, department string) error {
	ok, err := u.employmentAllowed(ctx, action, person, employer, department, false)
	if err == nil && !ok {
		return f.Fail(403, "HIRE_DENIED", "当前权限不允许此办理")
	}
	return err
}

func hireDates(arrival, last string) (f.Period, error) {
	term := f.Period{From: arrival}
	if last != "" {
		date, err := time.Parse(time.DateOnly, last)
		if err != nil {
			return term, f.Fail(400, "HIRE_DATE", "任期至无效")
		}
		term.Until = date.AddDate(0, 0, 1).Format(time.DateOnly)
	}
	if !term.Valid() {
		return term, f.Fail(400, "HIRE_DATE", "请核对到岗日及任期至")
	}
	return term, nil
}
func hireTerm(in h.Hire) (f.Period, error) {
	term, err := hireDates(in.Arrival, in.LastDay)
	if err != nil {
		return term, err
	}
	if in.Grade != "" && !f.ValidID(in.Grade) || !f.ValidID(in.Person) || !f.ValidID(in.Employer) || in.Position == "" && (!f.ValidID(in.Department) || !f.ValidID(in.Job)) || in.Position != "" && (!f.ValidID(in.Position) || in.Department != "" && !f.ValidID(in.Department) || in.Job != "" && !f.ValidID(in.Job)) || in.BU != "" && !f.ValidID(in.BU) || in.Location != "" && !f.ValidID(in.Location) || utf8.RuneCountInString(in.Notice) > 1000 {
		return term, f.Fail(400, "HIRE_INPUT", "请核对人员、任用说明、日期和拟任安排")
	}
	return term, nil
}

type hirePlan struct {
	h.Preview
	prepared               h.Preparation
	term                   f.Period
	employment, assignment string
	department, job        string
	original               employmentResult
	location               *h.LocationRef
}

func (u *unit) prepareHire(ctx context.Context, in h.Hire, showOriginalSources bool) (hirePlan, error) {
	var plan hirePlan
	term, err := hireTerm(in)
	if err != nil {
		return plan, err
	}
	plan.term = term
	if in.TrialStart != "" && (!f.ValidDate(in.TrialStart) || !term.Includes(in.TrialStart)) {
		return plan, f.Fail(400, "PROBATION_START", "实际试用开始日须在工作关系期间内")
	}
	allowed, e := u.employmentAllowed(ctx, hirePermission(in.RehireOf), in.Person, in.Employer, in.Department, in.Position != "")
	if e != nil {
		return plan, e
	}
	if !allowed {
		return plan, f.Fail(403, "HIRE_DENIED", "当前权限不允许此办理")
	}
	if err = lock(ctx, u.tx, false, "staffing.hire.person", u.store.Instance, in.Person); err != nil {
		return plan, err
	}
	ctx = dbtx.With(ctx, u.tx)
	if in.Decision != nil {
		if err = u.findDecisionHire(ctx, in, &plan, showOriginalSources); err != nil || plan.original.reused {
			return plan, err
		}
	}
	if err = u.checkRehireOrigin(ctx, in.Person, term.From, in.RehireOf); err != nil {
		return plan, err
	}
	plan.employment, plan.assignment = f.NewID(), f.NewID()
	request := h.FirstRequest{Appointment: h.Appointment{Person: in.Person, Employer: in.Employer, Department: in.Department, Job: in.Job, Position: in.Position, BU: in.BU, Location: in.Location, Period: term},
		Basis: h.Basis{Decision: in.Decision, Notice: strings.TrimSpace(in.Notice)}, Relationship: h.Choice{Complete: true, Eligible: []string{plan.employment}},
		Assignments: map[string]h.Choice{plan.employment: {Complete: true, Eligible: []string{plan.assignment}}}}
	legal := org.PurposeCheck{Role: "employer", Period: term, BU: in.BU, Location: in.Location}
	ent, err := u.Enterprise(ctx, true)
	if err != nil {
		return plan, err
	}
	job := staff.AssignmentCheck{Period: term, Category: "Employee", Purpose: "hire", Enterprise: ent.ID, Department: in.Department, Job: in.Job, Position: in.Position}
	if in.Expected != nil {
		if !reflect.DeepEqual(in.Decision, in.Expected.Decision) {
			return plan, appointmentConflict()
		}
		if in.Expected.Rule == "" || len(in.Expected.Legal) == 0 || len(in.Expected.Segments) == 0 {
			return plan, f.Fail(400, "HIRE_EXPECTED", "期望采用不完整")
		}
		request.ExpectedRuleVersion = in.Expected.Rule
		request.ExpectedIdentity = &in.Expected.Identity
		legal.Adopted = in.Expected.Legal
		job.Adopted = in.Expected.Segments
	}
	var positionSegments []staff.AssignmentSegment
	if in.Position != "" {
		ok, e := u.positionEmploymentScope(ctx, in.Position, term, in.Person, in.Employer, hirePermission(in.RehireOf), "hire")
		if e != nil {
			return plan, e
		}
		if !ok {
			return plan, f.Fail(403, "HIRE_DENIED", "所选安排超出获权范围")
		}
		positionSegments, err = u.store.CheckAssignment(ctx, u.credentials, job)
		if err != nil {
			return plan, err
		}
		for _, seg := range positionSegments {
			if err = u.requireEmployment(ctx, hirePermission(in.RehireOf), in.Person, in.Employer, seg.DepartmentID); err != nil {
				return plan, err
			}
		}
		in.Department, in.Job = positionSegments[0].DepartmentID, positionSegments[0].JobID
		request.Appointment.Department, request.Appointment.Job = in.Department, in.Job
		request.Appointment.Segments = positionSegments
	}
	plan.department, plan.job = in.Department, in.Job
	plan.prepared, err = h.PrepareFirst(ctx, h.Sources{Rules: u.store, People: u.store.HirePeople, Decisions: u}, u.credentials, request)
	if err != nil {
		return plan, err
	}
	if len(plan.prepared.Identity.Missing) > 0 {
		plan.Missing = plan.prepared.Identity.Missing
		return plan, nil
	}
	var existing bool
	err = u.tx.QueryRow(ctx, `SELECT EXISTS(SELECT FROM staffing.employment e WHERE e.instance_id=$1 AND e.person_id=$2
	AND (($5 AND EXISTS(SELECT FROM staffing.assignment t WHERE (t.instance_id,t.employment_id)=(e.instance_id,e.id) AND t.record_number=0)) OR (e.period * daterange(NULL,(SELECT x.first_inactive FROM staffing.employment_exit x WHERE (x.instance_id,x.employment_id)=(e.instance_id,e.id)),'[)')) && daterange($3::date,NULLIF($4,'')::date,'[)')) AND NOT EXISTS(SELECT FROM staffing.employment_action c
	WHERE c.instance_id=e.instance_id AND c.employment_id=e.id AND c.kind='CancelHire'))`, u.store.Instance, in.Person, term.From, term.Until, in.RehireOf == nil).Scan(&existing)
	if err != nil {
		return plan, err
	}
	if existing {
		return plan, f.Fail(409, "HIRE_EXISTING", "请核对既有工作关系。已任职人员再次到岗时，应从原离职办理再聘")
	}
	adopt := h.Adoption{Decision: in.Decision, Rule: plan.prepared.RuleVersion, Identity: plan.prepared.Identity.Source}
	adopt.Legal, err = u.store.HirePurposes.CheckPurpose(ctx, u.credentials, in.Employer, legal)
	if err != nil {
		return plan, err
	}
	for _, ref := range []struct{ field, id string }{{"buId", in.BU}, {"locationId", in.Location}} {
		if ref.id == "" {
			continue
		}
		sources, e := u.PositionSources(ctx, ref.field, []string{ref.id}, true)
		if e != nil {
			return plan, e
		}
		var label string
		if len(sources) == 1 {
			label, e = app.HireTargetLabel(ctx, u, ref.field, sources[0], term)
			if e != nil {
				return plan, e
			}
		}
		if label != "" && ref.field == "locationId" {
			source := sources[0]
			plan.location = &h.LocationRef{ID: ref.id, Name: label, Source: source.Source, Revision: source.Records[d.At(source.Timeline, term.From)].Revision}
		}
		if label == "" {
			return plan, f.Fail(409, "HIRE_CONTEXT_UNAVAILABLE", "所选业务单元或地点不可用于拟任期间")
		}
	}
	adopt.Segments = positionSegments
	if adopt.Segments == nil {
		adopt.Segments, err = u.store.CheckAssignment(ctx, u.credentials, job)
	}
	if err != nil {
		return plan, err
	}
	for _, seg := range adopt.Segments {
		if err = u.checkPlacementGrade(ctx, seg, in.Grade); err != nil {
			return plan, err
		}
	}
	if in.Grade != "" {
		for i, seg := range adopt.Segments {
			id := seg.JobID
			if id == "" {
				id = in.Job
			}
			grade, e := u.gradeForJob(ctx, id, in.Grade, seg.Period)
			if e != nil {
				return plan, e
			}
			if i == 0 {
				adopt.Grade = grade
			}
		}
		if err != nil {
			return plan, err
		}
	}
	if in.Expected != nil && !reflect.DeepEqual(in.Expected.Grade, adopt.Grade) {
		return plan, assignmentConflict()
	}
	plan.Adoption = &adopt
	return plan, nil
}

func (st *Store) PreviewHire(ctx context.Context, c f.Credentials, in h.Hire) (out h.Preview, err error) {
	err = st.within(ctx, c, false, false, func(u *unit) error { p, e := u.prepareHire(ctx, in, true); out = p.Preview; return e })
	return
}

func (st *Store) Hire(ctx context.Context, c f.Credentials, submission string, in h.Hire) (h.Receipt, error) {
	return st.employmentCommand(ctx, c, submission, []any{"Hire", in}, func(u *unit) (employmentResult, error) {
		return u.createHire(ctx, in)
	})
}

func (u *unit) createHire(ctx context.Context, in h.Hire) (employmentResult, error) {
	p, err := u.prepareHire(ctx, in, false)
	if err != nil || p.original.reused {
		return p.original, err
	}
	in.Department, in.Job = p.department, p.job
	if proof, ok := ctx.Value(offerContext{}).(offerAdmission); ok && len(p.Missing) == 0 {
		if err = checkOfferHire(proof.offer, in, p); err != nil {
			return employmentResult{}, err
		}
		p.Adoption.Offer = &proof.ref
	}
	if len(p.Missing) > 0 {
		return employmentResult{}, f.Fail(409, "HIRE_IDENTITY", "请先补齐入职所需资料")
	}
	if err = u.requireEmployment(ctx, hirePermission(in.RehireOf), in.Person, in.Employer, in.Department); err != nil {
		return employmentResult{}, err
	}
	id := f.NewID()
	instance := u.store.Instance
	_, err = u.tx.Exec(ctx, `INSERT INTO staffing.employment VALUES($1,$2,$3,$4,'Employee',daterange($5::date,NULLIF($6,'')::date,'[)'))`, instance, p.employment, in.Person, in.Employer, p.term.From, p.term.Until)
	if err != nil {
		return employmentResult{}, err
	}
	_, err = u.tx.Exec(ctx, `INSERT INTO staffing.employment_record(instance_id,person_id,record_number) VALUES($1,$2,0)
            ON CONFLICT(instance_id,person_id,record_number) DO NOTHING`, instance, in.Person)
	if err != nil {
		return employmentResult{}, err
	}
	revision := f.NewID()
	adoption := map[string]any{"sources": p.Adoption, "reason": p.prepared.Reason, "state": p.prepared.InitialState, "primaryPurpose": p.prepared.PrimaryPurpose, "primaryEmployment": p.employment, "primaryAssignment": p.assignment}
	if in.Position != "" {
		adoption["plannedPosition"] = in.Position
	}
	if in.TrialStart != "" {
		adoption["trialStart"] = in.TrialStart
	}
	kind := "Hire"
	if in.RehireOf != nil {
		kind = "Rehire"
		adoption["rehireOf"] = in.RehireOf
		adoption["reason"] = "rehire"
	}
	item := employmentEntry{id: id, revision: revision, employment: p.employment, assignment: p.assignment, kind: kind, date: in.Arrival, notice: p.prepared.Basis.Notice, adoption: adoption}
	if err = u.openAssignment(ctx, item, in, 0, p.term, p.Adoption.Segments, p.location, p.Adoption.Grade); err != nil {
		return employmentResult{}, err
	}
	if in.Decision != nil {
		_, err = u.tx.Exec(ctx, `INSERT INTO staffing.hire_decision_use(instance_id,source,version,line_id,action_id) VALUES($1,$2,$3,NULLIF($4,'')::uuid,$5)`, instance, in.Decision.Source, in.Decision.Version, in.Decision.Line, id)
		if err != nil {
			return employmentResult{}, err
		}
	}
	_, err = u.tx.Exec(ctx, `INSERT INTO staffing.employment_primary
		SELECT $1::uuid,$2::uuid,v.layer,v.owner,v.target,$6,daterange($7::date,NULLIF($8,'')::date,'[)')
		FROM (VALUES ('employment',$3::uuid,$4::uuid),('assignment',$4::uuid,$5::uuid)) v(layer,owner,target)`, instance, id, in.Person, p.employment, p.assignment, p.prepared.PrimaryPurpose, p.term.From, p.term.Until)
	return employmentResult{action: id, revision: revision}, err
}

func (u *unit) openAssignment(ctx context.Context, item employmentEntry, in h.Hire, number int, term f.Period, segments []staff.AssignmentSegment, location *h.LocationRef, grade *h.GradeRef) error {
	instance := u.store.Instance
	version := f.NewID()
	_, err := u.tx.Exec(ctx, `INSERT INTO staffing.assignment(instance_id,id,employment_id,department_id,job_id,bu_id,location_id,person_id,record_number,period)
        VALUES($1,$2,$3,$4,$5,NULLIF($6,'')::uuid,NULLIF($7,'')::uuid,$8,$9,daterange($10::date,nullif($11,'')::date,'[)'))`, instance, item.assignment, item.employment, in.Department, in.Job, in.BU, in.Location, in.Person, number, term.From, term.Until)
	if err != nil {
		return err
	}
	if err = u.newEmploymentAction(ctx, item); err != nil {
		return err
	}
	if err = u.startAssignmentSource(ctx, item.assignment, "", version, item.id, item.revision); err != nil {
		return err
	}
	for _, segment := range segments {
		dept, job := in.Department, in.Job
		if segment.DepartmentID != "" {
			dept, job = segment.DepartmentID, segment.JobID
		}
		_, err = u.tx.Exec(ctx, `INSERT INTO staffing.assignment_segment(instance_id,assignment_id,version,period,location_id,adoption,location_ref,department_id,job_id,grade_ref)
                VALUES($1,$2,$3,daterange($4::date,NULLIF($5,'')::date,'[)'),NULLIF($6,'')::uuid,$7,$8,$9,$10,$11)`, instance, item.assignment, version, segment.From, segment.Until, in.Location, segment, location, dept, job, grade)
		if err != nil {
			return err
		}
	}
	// A later assignment adopts the relationship revisions current on admission.
	_, err = u.tx.Exec(ctx, `INSERT INTO staffing.assignment_member(instance_id,assignment_id,version,action_id,revision)
        SELECT a.instance_id,$2,$3,a.id,m.revision FROM staffing.employment_action a
        JOIN staffing.assignment_head hd ON (hd.instance_id,hd.assignment_id)=(a.instance_id,a.assignment_id)
        JOIN staffing.assignment_member m ON (m.instance_id,m.assignment_id,m.version,m.action_id)=(hd.instance_id,hd.assignment_id,hd.version,a.id)
        WHERE a.instance_id=$1 AND a.employment_id=$4 AND a.kind IN ('ProbationStart','ConfirmEmployment')`, instance, item.assignment, version, item.employment)
	if err != nil {
		return err
	}
	if err = u.publishAssignmentSource(ctx, item.assignment, "", version); err != nil {
		return err
	}
	return nil
}

func (st *Store) EmploymentPurposeUses(ctx context.Context, instance, employer string) ([]staff.PurposeUse, error) {
	tx, ok := dbtx.From(ctx)
	if !ok || instance != st.Instance {
		return nil, errors.New("employment coverage requires the employer transaction")
	}
	rows, err := tx.Query(ctx, `SELECT greatest(lower(s.period),(clock_timestamp() AT TIME ZONE 'Asia/Shanghai')::date)::text,coalesce(upper(s.period)::text,''),coalesce(a.bu_id::text,''),coalesce(s.location_id::text,'')
	FROM staffing.employment e JOIN staffing.assignment a ON (a.instance_id,a.employment_id)=(e.instance_id,e.id)
	JOIN staffing.assignment_head h ON (h.instance_id,h.assignment_id)=(a.instance_id,a.id)
	JOIN staffing.assignment_segment s ON (s.instance_id,s.assignment_id,s.version)=(h.instance_id,h.assignment_id,h.version)
	WHERE e.instance_id=$1 AND e.employer_id=$2 AND (upper_inf(s.period) OR upper(s.period)>(clock_timestamp() AT TIME ZONE 'Asia/Shanghai')::date)`, instance, employer)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[staff.PurposeUse])
}

func hireDigest(v any) (string, error) {
	b, e := json.Marshal(v)
	if e != nil {
		return "", e
	}
	hash := sha256.Sum256(b)
	return hex.EncodeToString(hash[:]), nil
}

func (st *Store) HireEmployers(ctx context.Context, c f.Credentials, in h.EmployerSearch) ([]org.EligibleEntity, error) {
	term, err := hireDates(in.Arrival, in.LastDay)
	if err != nil {
		return nil, err
	}
	if !f.ValidID(in.Person) || in.Department != "" && !f.ValidID(in.Department) || in.Page < 1 || in.Page > 1000000 {
		return nil, f.Fail(400, "HIRE_INPUT", "人员或候选查询无效")
	}
	candidates := []org.EligibleEntity{}
	err = st.within(ctx, c, false, false, func(u *unit) error {
		allowed, e := u.employmentAllowed(ctx, hirePermission(in.RehireOf), in.Person, "", "", true)
		if e != nil {
			return e
		}
		if !allowed {
			return f.Fail(403, "HIRE_DENIED", "没有本次办理权限")
		}
		if in.RehireOf != nil {
			if e := u.checkRehireOrigin(ctx, in.Person, in.Arrival, in.RehireOf); e != nil {
				return e
			}
		}
		return u.employerTargets(ctx, term, in, hirePermission(in.RehireOf), &candidates)
	})
	return candidates, err
}

func (u *unit) employerTargets(ctx context.Context, term f.Period, in h.EmployerSearch, permission string, candidates *[]org.EligibleEntity) error {
	skip := (in.Page - 1) * 20
	for page := 1; ; page++ {
		chunk, e := u.store.HirePurposes.Candidates(dbtx.With(ctx, u.tx), u.credentials, org.PurposeCandidates{PurposeCheck: org.PurposeCheck{Role: "employer", Period: term, BU: in.BU, Location: in.Location}, Search: in.Search, Page: page})
		if e != nil {
			return e
		}
		for _, v := range chunk {
			ok, e := u.employmentAllowed(ctx, permission, in.Person, v.ID, in.Department, true)
			if e != nil {
				return e
			}
			if ok {
				if skip > 0 {
					skip--
					continue
				}
				*candidates = append(*candidates, v)
				if len(*candidates) == 20 {
					return nil
				}
			}
		}
		if len(chunk) < 20 {
			return nil
		}
	}
}

// HireTargets pages after both the dated source check and the operator's M04
// scope. Department selection may precede employer selection.
func (st *Store) HireTargets(ctx context.Context, c f.Credentials, in h.TargetSearch) ([]org.EligibleEntity, error) {
	term, err := hireDates(in.Arrival, in.LastDay)
	if err != nil {
		return nil, err
	}
	field := map[string]string{"bu": "buId", "location": "locationId", "department": "departmentId", "job": "jobId", "grade": "gradeId", "position": "positionId"}[in.Kind]
	if field == "" || field == "gradeId" && !f.ValidID(in.Job) || !f.ValidID(in.Person) || in.Employer != "" && !f.ValidID(in.Employer) || in.Department != "" && !f.ValidID(in.Department) || in.Page < 1 || in.Page > 1000000 || utf8.RuneCountInString(in.Search) > 400 {
		return nil, f.Fail(400, "HIRE_INPUT", "拟任目标查询无效")
	}
	items := []org.EligibleEntity{}
	err = st.within(ctx, c, false, true, func(u *unit) error {
		dept := in.Department
		if field == "departmentId" {
			dept = ""
		}
		allowed, err := u.employmentAllowed(ctx, hirePermission(in.RehireOf), in.Person, in.Employer, dept, true)
		if err != nil {
			return err
		}
		if !allowed {
			return f.Fail(403, "HIRE_DENIED", "没有本次办理权限")
		}
		if in.RehireOf != nil {
			if e := u.checkRehireOrigin(ctx, in.Person, in.Arrival, in.RehireOf); e != nil {
				return e
			}
		}
		if field == "positionId" {
			return u.positionTargets(ctx, term, in, hirePermission(in.RehireOf), "hire", nil, &items)
		}
		return u.arrangementTargets(ctx, term, in, field, hirePermission(in.RehireOf), &items)
	})
	return items, err
}

func (u *unit) arrangementTargets(ctx context.Context, term f.Period, in h.TargetSearch, field, permission string, items *[]org.EligibleEntity) error {
	needle := strings.ToLower(strings.TrimSpace(in.Search))
	skip := (in.Page - 1) * 20
	const batch = 64
	var members []string
	var job app.PositionSource
	if field == "gradeId" {
		var e error
		job, e = u.gradeJob(ctx, in.Job, term.From, false)
		if e != nil {
			return e
		}
		members = job.Records[d.At(job.Timeline, term.From)].Grades.IDs()
		if len(members) == 0 {
			return nil
		}
	}
	for offset := 0; ; offset += batch {
		ids, err := u.PositionCandidateIDs(ctx, field, term.From, needle, members, offset, batch)
		if err != nil {
			return err
		}
		sources, err := u.PositionSources(ctx, field, ids, false)
		if err != nil {
			return err
		}
		byID := make(map[string]app.PositionSource, len(sources))
		for _, source := range sources {
			byID[source.ID] = source
		}
		for _, id := range ids {
			if field == "gradeId" && d.PositionConflict(job.Timeline, term.From, term.Until, id) != "" {
				continue
			}
			if field == "departmentId" {
				ok, e := u.employmentAllowed(ctx, permission, in.Person, in.Employer, id, true)
				if e != nil {
					return e
				}
				if ok && permission == "transfer" {
					ok, e = u.employmentAllowed(ctx, "read", in.Person, in.Employer, id, false)
					if e != nil {
						return e
					}
				}
				if !ok {
					continue
				}
			}
			label, e := app.HireTargetLabel(ctx, u, field, byID[id], term)
			if e != nil {
				return e
			}
			if label == "" || !strings.Contains(strings.ToLower(label), needle) {
				continue
			}
			if skip > 0 {
				skip--
				continue
			}
			*items = append(*items, org.EligibleEntity{ID: id, Name: label})
			if len(*items) == 20 {
				return nil
			}
		}
		if len(ids) < batch {
			return nil
		}
	}
}

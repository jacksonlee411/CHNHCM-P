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
	if !f.ValidID(in.Person) || !f.ValidID(in.Employer) || !f.ValidID(in.Department) || !f.ValidID(in.Job) || in.BU != "" && !f.ValidID(in.BU) || in.Location != "" && !f.ValidID(in.Location) || utf8.RuneCountInString(in.Notice) > 1000 {
		return term, f.Fail(400, "HIRE_INPUT", "请核对人员、任用说明、日期和拟任安排")
	}
	return term, nil
}

type hirePlan struct {
	h.Preview
	prepared               h.Preparation
	term                   f.Period
	employment, assignment string
	previousAction         string
	location               *h.LocationRef
}

func (u *unit) prepareHire(ctx context.Context, in h.Hire) (hirePlan, error) {
	var plan hirePlan
	term, err := hireTerm(in)
	if err != nil {
		return plan, err
	}
	plan.term = term
	if err = u.requireEmployment(ctx, "hire", in.Person, in.Employer, in.Department); err != nil {
		return plan, err
	}
	if err = lock(ctx, u.tx, false, "staffing.hire.person", u.store.Instance, in.Person); err != nil {
		return plan, err
	}
	plan.employment, plan.assignment = f.NewID(), f.NewID()
	request := h.FirstRequest{Appointment: h.Appointment{Person: in.Person, Employer: in.Employer, Department: in.Department, Job: in.Job, BU: in.BU, Location: in.Location, Period: term},
		Basis: h.Basis{Decision: in.Decision, Notice: strings.TrimSpace(in.Notice)}, Relationship: h.Choice{Complete: true, Eligible: []string{plan.employment}},
		Assignments: map[string]h.Choice{plan.employment: {Complete: true, Eligible: []string{plan.assignment}}}}
	legal := org.PurposeCheck{Role: "employer", Period: term, BU: in.BU, Location: in.Location}
	ent, err := u.Enterprise(ctx, true)
	if err != nil {
		return plan, err
	}
	job := staff.AssignmentCheck{Period: term, Category: "Employee", Purpose: "hire", Enterprise: ent.ID, Department: in.Department, Job: in.Job}
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
	ctx = dbtx.With(ctx, u.tx)
	plan.prepared, err = h.PrepareFirst(ctx, h.Sources{Rules: u.store, People: u.store.HirePeople, Decisions: u}, u.credentials, request)
	if err != nil {
		return plan, err
	}
	if len(plan.prepared.Identity.Missing) > 0 {
		plan.Missing = plan.prepared.Identity.Missing
		return plan, nil
	}
	if in.Decision != nil {
		plan.previousAction, err = u.previousDecisionHire(ctx, in, term)
		if err != nil {
			return plan, err
		}
	}
	var existing bool
	err = u.tx.QueryRow(ctx, `SELECT EXISTS(SELECT FROM staffing.employment e WHERE e.instance_id=$1 AND e.person_id=$2
	AND e.period && daterange($3::date,NULLIF($4,'')::date,'[)') AND NOT EXISTS(SELECT FROM staffing.employment_action c
	WHERE c.instance_id=e.instance_id AND c.employment_id=e.id AND c.kind='CancelHire'))`, u.store.Instance, in.Person, term.From, term.Until).Scan(&existing)
	if err != nil {
		return plan, err
	}
	if existing && plan.previousAction == "" {
		return plan, f.Fail(409, "HIRE_EXISTING", "当前无法按首笔入职办理，请由获权人核对工作安排")
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
	adopt.Segments, err = u.store.CheckAssignment(ctx, u.credentials, job)
	if err != nil {
		return plan, err
	}
	plan.Adoption = &adopt
	return plan, nil
}

func (st *Store) PreviewHire(ctx context.Context, c f.Credentials, in h.Hire) (out h.Preview, err error) {
	err = st.within(ctx, c, false, false, func(u *unit) error { p, e := u.prepareHire(ctx, in); out = p.Preview; return e })
	return
}

func (st *Store) Hire(ctx context.Context, c f.Credentials, submission string, in h.Hire) (h.Receipt, error) {
	return st.hireCommand(ctx, c, submission, []any{"Hire", in}, func(u *unit) (string, error) {
		p, err := u.prepareHire(ctx, in)
		if err != nil {
			return "", err
		}
		if len(p.Missing) > 0 {
			return "", f.Fail(409, "HIRE_IDENTITY", "请先补齐入职所需资料")
		}
		if err = u.requireEmployment(ctx, "hire", in.Person, in.Employer, in.Department); err != nil {
			return "", err
		}
		if p.previousAction != "" {
			return p.previousAction, nil
		}
		id := f.NewID()
		instance := st.Instance
		_, err = u.tx.Exec(ctx, `INSERT INTO staffing.employment VALUES($1,$2,$3,$4,'Employee',daterange($5::date,NULLIF($6,'')::date,'[)'))`, instance, p.employment, in.Person, in.Employer, p.term.From, p.term.Until)
		if err != nil {
			return "", err
		}
		_, err = u.tx.Exec(ctx, `INSERT INTO staffing.employment_record(instance_id,person_id,record_number) VALUES($1,$2,0)
            ON CONFLICT(instance_id,person_id,record_number) DO NOTHING`, instance, in.Person)
		if err != nil {
			return "", err
		}
		revision := f.NewID()
		adoption := map[string]any{"sources": p.Adoption, "reason": p.prepared.Reason, "state": p.prepared.InitialState, "primaryPurpose": p.prepared.PrimaryPurpose, "primaryEmployment": p.employment, "primaryAssignment": p.assignment}
		item := employmentEntry{id: id, revision: revision, employment: p.employment, assignment: p.assignment, kind: "Hire", date: in.Arrival, notice: p.prepared.Basis.Notice, adoption: adoption}
		if err = u.openAssignment(ctx, item, in, 0, p.term, p.Adoption.Segments, p.location); err != nil {
			return "", err
		}
		if in.Decision != nil {
			_, err = u.tx.Exec(ctx, `INSERT INTO staffing.hire_decision_use(instance_id,source,version,line_id,action_id) VALUES($1,$2,$3,NULLIF($4,'')::uuid,$5)`, instance, in.Decision.Source, in.Decision.Version, in.Decision.Line, id)
			if err != nil {
				return "", err
			}
		}
		_, err = u.tx.Exec(ctx, `INSERT INTO staffing.employment_primary
		SELECT $1::uuid,$2::uuid,v.layer,v.owner,v.target,$6,daterange($7::date,NULLIF($8,'')::date,'[)')
		FROM (VALUES ('employment',$3::uuid,$4::uuid),('assignment',$4::uuid,$5::uuid)) v(layer,owner,target)`, instance, id, in.Person, p.employment, p.assignment, p.prepared.PrimaryPurpose, p.term.From, p.term.Until)
		return id, err
	})
}

func (u *unit) openAssignment(ctx context.Context, item employmentEntry, in h.Hire, number int, term f.Period, segments []staff.AssignmentSegment, location *h.LocationRef) error {
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
		_, err = u.tx.Exec(ctx, `INSERT INTO staffing.assignment_segment(instance_id,assignment_id,version,period,location_id,adoption,location_ref,department_id,job_id)
                VALUES($1,$2,$3,daterange($4::date,NULLIF($5,'')::date,'[)'),NULLIF($6,'')::uuid,$7,$8,$9,$10)`, instance, item.assignment, version, segment.From, segment.Until, in.Location, segment, location, in.Department, in.Job)
		if err != nil {
			return err
		}
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
		allowed, e := u.employmentAllowed(ctx, "hire", in.Person, "", "", true)
		if e != nil {
			return e
		}
		if !allowed {
			return f.Fail(403, "HIRE_DENIED", "没有本次办理权限")
		}
		return u.employerTargets(ctx, term, in, "hire", &candidates)
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
	field := map[string]string{"bu": "buId", "location": "locationId", "department": "departmentId", "job": "jobId"}[in.Kind]
	if field == "" || !f.ValidID(in.Person) || in.Employer != "" && !f.ValidID(in.Employer) || in.Department != "" && !f.ValidID(in.Department) || in.Page < 1 || in.Page > 1000000 || utf8.RuneCountInString(in.Search) > 400 {
		return nil, f.Fail(400, "HIRE_INPUT", "拟任目标查询无效")
	}
	items := []org.EligibleEntity{}
	err = st.within(ctx, c, false, true, func(u *unit) error {
		dept := in.Department
		if field == "departmentId" {
			dept = ""
		}
		allowed, err := u.employmentAllowed(ctx, "hire", in.Person, in.Employer, dept, true)
		if err != nil {
			return err
		}
		if !allowed {
			return f.Fail(403, "HIRE_DENIED", "没有本次办理权限")
		}
		return u.arrangementTargets(ctx, term, in, field, "hire", &items)
	})
	return items, err
}

func (u *unit) arrangementTargets(ctx context.Context, term f.Period, in h.TargetSearch, field, permission string, items *[]org.EligibleEntity) error {
	needle := strings.ToLower(strings.TrimSpace(in.Search))
	skip := (in.Page - 1) * 20
	const batch = 64
	for offset := 0; ; offset += batch {
		ids, err := u.PositionCandidateIDs(ctx, field, term.From, needle, nil, offset, batch)
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

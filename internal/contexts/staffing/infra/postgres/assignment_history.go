package postgres

import (
	"context"
	"errors"

	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	f "chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
)

type savedRecord struct {
	h.HistoryEntry
	Operation string
	Notice    string
	Evidence  h.Original
}

func (u *unit) historyRows(ctx context.Context, a h.Action) ([]h.Placement, []savedRecord, error) {
	rows, err := u.tx.Query(ctx, `SELECT lower(s.period)::text,coalesce(upper(s.period)::text,''),s.department_id,s.job_id,s.adoption,
        coalesce(s.location_id::text,''),s.location_ref,s.grade_ref FROM staffing.assignment_segment s
        JOIN staffing.employment_revision_publication p USING(instance_id,assignment_id,version)
        WHERE p.instance_id=$1 AND p.assignment_id=$2 AND p.action_id=$3 AND p.revision=$4 ORDER BY lower(s.period)`, u.store.Instance, a.Assignment, a.ID, a.Revision)
	if err != nil {
		return nil, nil, err
	}
	initial, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (p h.Placement, e error) {
		var id string
		e = r.Scan(&p.From, &p.Until, &p.Department, &p.Job, &p.Adoption, &id, &p.Location, &p.Grade)
		if p.Location == nil && id != "" {
			p.Location = &h.LocationRef{ID: id}
		}
		return
	})
	if err != nil {
		return nil, nil, err
	}
	rows, err = u.tx.Query(ctx, `SELECT a.id,r.revision,r.business_date::text,a.kind,r.operation,r.notice,r.adoption
        FROM staffing.assignment_member m JOIN staffing.employment_action a ON (a.instance_id,a.id)=(m.instance_id,m.action_id)
        JOIN staffing.employment_action_revision r ON (r.instance_id,r.assignment_id,r.action_id,r.revision)=(m.instance_id,m.assignment_id,m.action_id,m.revision)
        WHERE m.instance_id=$1 AND m.assignment_id=$2 AND m.version=$3 AND a.kind IN ('DataChange','Transfer','GradeRecord','Promotion')`, u.store.Instance, a.Assignment, a.SourceVersion)
	if err != nil {
		return nil, nil, err
	}
	records, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (r savedRecord, e error) {
		e = row.Scan(&r.ID, &r.Revision, &r.Day, &r.Kind, &r.Operation, &r.Notice, &r.Evidence)
		r.Removed = r.Operation == "Delete" || r.Operation == "Withdraw"
		r.Previous = r.Evidence.Previous
		if r.Evidence.Promotion != nil {
			r.Grade = &r.Evidence.Promotion.After
		}
		r.Requires = r.Evidence.Requires
		if r.Evidence.LocationChange != nil {
			r.Location = &r.Evidence.LocationChange.After
		}
		if r.Evidence.Transfer != nil {
			r.Transfer = &r.Evidence.Transfer.After
		}
		return
	})
	return initial, records, err
}

func projectRecords(a h.Action, initial []h.Placement, records []savedRecord) (h.HistoryProjection, error) {
	entries := make([]h.HistoryEntry, len(records))
	for i, r := range records {
		entries[i] = r.HistoryEntry
	}
	return h.ProjectHistory(a.Period, initial, entries)
}

// A correction or re-date can change what an independent successor saw on
// entry. Its stable action and original notice survive the new revision.
func (u *unit) appendRecordRevision(ctx context.Context, assignment string, r savedRecord, previous string) error {
	actor, e := u.guard.OperatorSnapshot(ctx)
	if e != nil {
		return e
	}
	_, e = u.tx.Exec(ctx, `INSERT INTO staffing.employment_action_revision(instance_id,assignment_id,action_id,revision,previous_revision,business_date,operation,notice,actor,operator_snapshot,adoption)
        VALUES($1,$2,$3,$4,$5,$6::date,$7,$8,$9,$10,$11)`, u.store.Instance, assignment, r.ID, r.Revision, previous, r.Day, r.Operation, r.Notice, u.guard.Principal().ID, actor, recordAdoption(r))
	return e
}

func recordAdoption(r savedRecord) map[string]any {
	fields := map[string]any{}
	if r.Evidence.NoticeRef != nil {
		fields["noticeRef"] = r.Evidence.NoticeRef
	}
	if r.Evidence.Promotion != nil {
		fields["promotion"] = r.Evidence.Promotion
	}
	if r.Previous != nil {
		fields["previous"] = r.Previous
	}
	if r.Evidence.LocationChange != nil {
		fields["locationChange"] = r.Evidence.LocationChange
	}
	if r.Evidence.Transfer != nil {
		fields["transfer"] = r.Evidence.Transfer
	}
	if r.Requires != nil {
		fields["requires"] = r.Requires
	}
	return fields
}

func (u *unit) saveReplayedSource(ctx context.Context, a h.Action, version string, target savedRecord, records []savedRecord, segments []h.Placement) error {
	instance := u.store.Instance
	_, e := u.tx.Exec(ctx, `INSERT INTO staffing.assignment_source(instance_id,assignment_id,version,previous,action_id,revision) VALUES($1,$2,$3,$4,$5,$6)`,
		instance, a.Assignment, version, a.SourceVersion, target.ID, target.Revision)
	if e != nil {
		return e
	}
	_, e = u.tx.Exec(ctx, `INSERT INTO staffing.assignment_member SELECT m.instance_id,m.assignment_id,$4::uuid,m.action_id,m.revision FROM staffing.assignment_member m
        JOIN staffing.employment_action a ON (a.instance_id,a.id)=(m.instance_id,m.action_id)
        WHERE m.instance_id=$1 AND m.assignment_id=$2 AND m.version=$3 AND a.kind NOT IN ('DataChange','Transfer','GradeRecord','Promotion')`, instance, a.Assignment, a.SourceVersion, version)
	if e != nil {
		return e
	}
	for _, r := range records {
		_, e = u.tx.Exec(ctx, `INSERT INTO staffing.assignment_member(instance_id,assignment_id,version,action_id,revision) VALUES($1,$2,$3,$4,$5)`,
			instance, a.Assignment, version, r.ID, r.Revision)
		if e != nil {
			return e
		}
	}
	for _, s := range segments {
		id := ""
		if s.Location != nil {
			id = s.Location.ID
		}
		_, e = u.tx.Exec(ctx, `INSERT INTO staffing.assignment_segment(instance_id,assignment_id,version,period,department_id,job_id,adoption,location_id,location_ref,grade_ref)
            VALUES($1,$2,$3,daterange($4::date,nullif($5,'')::date,'[)'),$6,$7,$8,nullif($9,'')::uuid,$10,$11)`,
			instance, a.Assignment, version, s.From, s.Until, s.Department, s.Job, s.Adoption, id, s.Location, s.Grade)
		if e != nil {
			return e
		}
	}
	return u.publishAssignmentSource(ctx, a.Assignment, a.SourceVersion, version)
}

func (u *unit) assignmentPlacement(ctx context.Context, a h.Action, day string) (p h.Placement, err error) {
	var location string
	err = u.tx.QueryRow(ctx, `SELECT lower(period)::text,coalesce(upper(period)::text,''),department_id,job_id,adoption,coalesce(location_id::text,''),location_ref,grade_ref
        FROM staffing.assignment_segment WHERE instance_id=$1 AND assignment_id=$2 AND version=$3 AND period @> $4::date`,
		u.store.Instance, a.Assignment, a.SourceVersion, day).Scan(&p.From, &p.Until, &p.Department, &p.Job, &p.Adoption, &location, &p.Location, &p.Grade)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, f.Fail(409, "ASSIGNMENT_PERIOD", "所选日期没有有效任职")
	}
	if err == nil && p.Location == nil && location != "" {
		p.Location = &h.LocationRef{ID: location}
	}
	return
}

type actionDisclosure struct {
	from, to          bool
	before, beforeJob string
}

func (u *unit) actionDisclosure(ctx context.Context, a h.Action, history bool) (v actionDisclosure, err error) {
	allowed := func(dept string) (bool, error) {
		if dept == "" && a.Kind != "Promotion" {
			return false, nil
		}
		ok, e := u.employmentAllowed(ctx, "read", a.Person, a.Employer, dept, false)
		if !ok || e != nil || !history {
			return ok, e
		}
		return u.employmentAllowed(ctx, "history", a.Person, a.Employer, dept, false)
	}
	if a.Kind != "Transfer" {
		v.to, err = allowed(a.Department)
		return
	}
	var before, after string
	err = u.tx.QueryRow(ctx, `SELECT adoption#>>'{transfer,beforeDepartment}',adoption#>>'{transfer,after,department}',adoption#>>'{transfer,beforeJob}'
        FROM staffing.employment_action_revision WHERE instance_id=$1 AND action_id=$2 AND revision=$3`,
		u.store.Instance, a.ID, a.Revision).Scan(&before, &after, &v.beforeJob)
	v.before = before
	if err != nil {
		return
	}
	v.from, err = allowed(before)
	if err != nil {
		return
	}
	v.to, err = allowed(after)
	return
}

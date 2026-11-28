package postgres

import (
	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	st "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
)

// Uses the caller's transaction and the same person and job guards as staffing maintenance.
func (s *Store) PayAssignment(ctx context.Context, enterprise, subject, id, day string, writing bool) (out st.PayAssignment, err error) {
	if !foundation.ValidID(id) {
		return out, unknownAssignment()
	}
	tx, ok := dbtx.From(ctx)
	if !ok {
		return out, errors.New("assignment source requires transaction")
	}
	u := unit{store: s, tx: tx}
	a, e := u.assignmentIdentity(ctx, id)
	if e != nil {
		return out, e
	}
	if writing {
		if e = lock(ctx, tx, true, "staffing.hire.person", s.Instance, a.Person); e != nil {
			return out, e
		}
		a, e = u.assignmentIdentity(ctx, id)
		if e != nil {
			return out, e
		}
	}
	out = st.PayAssignment{Person: a.Person, Assignment: id, Employment: a.Employment, Employer: a.Employer, Source: a.SourceVersion, Department: a.Department, Job: a.Job, From: a.From, Until: a.Until}
	var grade *h.GradeRef
	e = tx.QueryRow(ctx, `SELECT department_id,job_id,grade_ref,lower(period)::text,coalesce(upper(period)::text,'') FROM staffing.assignment_segment WHERE instance_id=$1 AND assignment_id=$2 AND version=$3 AND period @> $4::date`, s.Instance, id, a.SourceVersion, day).Scan(&out.Department, &out.Job, &grade, &out.SegmentFrom, &out.SegmentUntil)
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		return out, e
	}
	out.Active = e == nil && !a.Cancelled && a.Period.Includes(day)
	allowed, e := s.EmploymentAllowed(ctx, tx, subject, "read", a.Person, a.Employer, out.Department, false)
	if e != nil {
		return out, e
	}
	if !allowed {
		return st.PayAssignment{}, unknownAssignment()
	}
	if !out.Active {
		return out, nil
	}
	if grade != nil {
		out.Grade, out.GradeName, out.GradeSource, out.GradeAdoption = grade.ID, grade.Name, grade.Source, grade.Revision
	}
	if writing {
		if e = lock(ctx, tx, true, objectGuard(d.Job), s.Instance, out.Job); e != nil {
			return out, e
		}
	}
	var raw []byte
	e = tx.QueryRow(ctx, "SELECT head FROM staffing.job WHERE instance_id=$1 AND enterprise_id=$2 AND id=$3", s.Instance, enterprise, out.Job).Scan(&raw)
	if errors.Is(e, pgx.ErrNoRows) {
		return out, nil
	}
	if e != nil {
		return out, e
	}
	job, e := decode(raw, d.Job)
	if e != nil {
		return out, e
	}
	if ix := d.At(job, day); ix >= 0 && job.Records[ix].Active() {
		r := job.Records[ix]
		out.Family, out.JobSource, out.JobRevision = r.Data["jobFamilyId"], job.Source, r.Revision
		out.JobRecord = r.ID
	}
	return out, nil
}

package postgres

import (
	"context"
	"errors"

	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	f "chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
)

func exitBoundary(exit *h.ExitEvidence) string {
	if exit != nil {
		return exit.FirstInactive
	}
	return ""
}

// The original anchor owns the occurrence department. Other affected assignments
// adopt its revision while retaining their own placements and exit boundaries.
func (u *unit) reconcileExitDepartment(ctx context.Context, assignment string, proposed []h.Placement, apply bool) (*h.ActionRef, error) {
	var action, revision, kind, last, department, oldSource, oldRevision string
	err := u.tx.QueryRow(ctx, `SELECT a.id,m.revision,a.kind,(x.first_inactive-1)::text,r.adoption->>'department',
	  coalesce(r.adoption->>'departmentSource',exit_department.source,''),
	  coalesce(r.adoption->>'departmentRevision',exit_department.revision,'')
	FROM staffing.assignment_exit x
	JOIN staffing.employment_action a ON (a.instance_id,a.id,a.assignment_id)=(x.instance_id,x.action_id,x.assignment_id)
	JOIN staffing.assignment_head hd ON (hd.instance_id,hd.assignment_id)=(x.instance_id,x.assignment_id)
	JOIN staffing.assignment_member m ON (m.instance_id,m.assignment_id,m.version,m.action_id)=(hd.instance_id,hd.assignment_id,hd.version,a.id)
	JOIN staffing.employment_action_revision r ON (r.instance_id,r.action_id,r.revision)=(m.instance_id,m.action_id,m.revision)
	JOIN staffing.employment_revision_publication issued ON (issued.instance_id,issued.action_id,issued.revision)=(r.instance_id,r.action_id,r.revision)
	`+exitDepartmentJoin+`
	WHERE x.instance_id=$1 AND x.assignment_id=$2`, u.store.Instance, assignment).Scan(&action, &revision, &kind, &last, &department, &oldSource, &oldRevision)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	anchor, err := u.assignmentIdentity(ctx, assignment)
	if err != nil {
		return nil, err
	}
	var at h.Placement
	if proposed == nil {
		at, err = u.assignmentPlacement(ctx, anchor, last)
		if err != nil {
			return nil, err
		}
	} else {
		for _, span := range proposed {
			if span.Includes(last) {
				at = span
				break
			}
		}
	}
	if at.Department == "" {
		return nil, f.Fail(409, "EXIT_DEPARTMENT", "请核对退出前最后在任日的部门依据")
	}
	if department == at.Department && (oldRevision == "" || oldSource == at.Adoption.DepartmentSource && oldRevision == at.Adoption.DepartmentRevision) {
		return nil, nil
	}
	permission := "assignment.end"
	if kind == "Leave" {
		permission = "employment.leave"
	}
	for _, dept := range []string{department, at.Department} {
		if err := u.requireEmployment(ctx, permission, anchor.Person, anchor.Employer, dept); err != nil {
			return nil, err
		}
		if err := u.requireEmployment(ctx, "read", anchor.Person, anchor.Employer, dept); err != nil {
			return nil, err
		}
	}
	ref := &h.ActionRef{Action: action, Revision: revision}
	if !apply {
		return ref, nil
	}
	operator, err := u.guard.OperatorSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	ref.Revision = f.NewID()
	_, err = u.tx.Exec(ctx, `INSERT INTO staffing.employment_action_revision(instance_id,assignment_id,action_id,revision,previous_revision,business_date,operation,notice,actor,operator_snapshot,adoption)
	SELECT instance_id,assignment_id,action_id,$4,revision,business_date,'Rebase',notice,$5,$6,adoption ||
	  jsonb_build_object('department',$7::text,'departmentSource',$8::text,'departmentRevision',$9::text)
	FROM staffing.employment_action_revision WHERE instance_id=$1 AND action_id=$2 AND revision=$3`, u.store.Instance, action, revision, ref.Revision, u.guard.Principal().ID, operator, at.Department, at.Adoption.DepartmentSource, at.Adoption.DepartmentRevision)
	if err != nil {
		return nil, err
	}
	rows, err := u.tx.Query(ctx, `SELECT x.assignment_id,hd.version FROM staffing.assignment_exit x JOIN staffing.assignment_head hd USING(instance_id,assignment_id)
	    WHERE x.instance_id=$1 AND x.action_id=$2 AND ($3 OR x.assignment_id<>$4) ORDER BY x.assignment_id`, u.store.Instance, action, proposed == nil, assignment)
	if err != nil {
		return nil, err
	}
	peers, err := pgx.CollectRows(rows, pgx.RowToStructByPos[struct{ Assignment, Version string }])
	if err != nil {
		return nil, err
	}
	for _, peer := range peers {
		if err = u.lifecycleSource(ctx, peer.Assignment, peer.Version, action, ref.Revision, ""); err != nil {
			return nil, err
		}
	}
	_, err = u.tx.Exec(ctx, `INSERT INTO staffing.employment_outbox(instance_id,event_id,action_id,revision) VALUES($1,$2,$3,$4)`, u.store.Instance, f.NewID(), action, ref.Revision)
	return ref, err
}

// A verified paper can be on file before the assignment adopts it. Keep both the
// latest paper and any still adopted edition until the correction is applied.
func (u *unit) exitConflictsWithDuty(ctx context.Context, assignment, boundary string) (bool, error) {
	var conflicts bool
	err := u.tx.QueryRow(ctx, `WITH evidence AS (
	SELECT r.fact FROM staffing.duty_observation o
	JOIN staffing.duty_observation_head hd ON (hd.instance_id,hd.observation_id)=(o.instance_id,o.id)
	JOIN staffing.duty_observation_revision r ON (r.instance_id,r.observation_id,r.version)=(hd.instance_id,hd.observation_id,hd.version)
	WHERE o.instance_id=$1 AND o.assignment_id=$2
	UNION ALL
	SELECT r.adoption#>'{transfer,after,duty}' FROM staffing.assignment_head hd JOIN staffing.assignment_member m USING(instance_id,assignment_id,version)
	JOIN staffing.employment_action_revision r USING(instance_id,action_id,revision)
	WHERE hd.instance_id=$1 AND hd.assignment_id=$2 AND r.operation NOT IN ('Delete','Withdraw'))
	SELECT EXISTS(SELECT FROM evidence WHERE coalesce((fact->>'until')::date-1,(fact->>'observedThrough')::date)>=$3::date)`,
		u.store.Instance, assignment, boundary).Scan(&conflicts)
	return conflicts, err
}

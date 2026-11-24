package postgres

import (
	"context"

	f "chnhcm.local/chnhcm/internal/foundation"
)

type employmentEntry struct {
	id, revision, employment, assignment, kind, original, date, notice string
	adoption                                                           any
}

func (u *unit) newEmploymentAction(ctx context.Context, e employmentEntry) error {
	operator, err := u.guard.OperatorSnapshot(ctx)
	if err != nil {
		return err
	}
	_, err = u.tx.Exec(ctx, `WITH new_action AS (
		INSERT INTO staffing.employment_action(instance_id,id,revision,kind,employment_id,assignment_id,original_id,actor)
		VALUES($1,$2,$3,$4,$5,$6,NULLIF($7,'')::uuid,$8) RETURNING *)
		INSERT INTO staffing.employment_action_revision(instance_id,assignment_id,action_id,revision,business_date,notice,actor,recorded_at,adoption,operator_snapshot)
		SELECT instance_id,assignment_id,id,revision,$9::date,$10,actor,recorded_at,$11,$12 FROM new_action`,
		u.store.Instance, e.id, e.revision, e.kind, e.employment, e.assignment, e.original, u.guard.Principal().ID, e.date, e.notice, e.adoption, operator)
	return err
}

func (u *unit) startAssignmentSource(ctx context.Context, assignment, previous, version, action, revision string) error {
	instance := u.store.Instance
	_, err := u.tx.Exec(ctx, `INSERT INTO staffing.assignment_source(instance_id,assignment_id,version,previous,action_id,revision)
		VALUES($1,$2,$3,NULLIF($4,'')::uuid,$5,$6)`, instance, assignment, version, previous, action, revision)
	if err != nil {
		return err
	}
	_, err = u.tx.Exec(ctx, `INSERT INTO staffing.assignment_member(instance_id,assignment_id,version,action_id,revision)
		SELECT instance_id,assignment_id,$3::uuid,action_id,revision FROM staffing.assignment_member
		WHERE instance_id=$1 AND assignment_id=$2 AND version=NULLIF($4,'')::uuid AND action_id<>$5
		UNION ALL SELECT $1::uuid,$2::uuid,$3::uuid,$5::uuid,$6::uuid`, instance, assignment, version, previous, action, revision)
	return err
}

func (u *unit) publishAssignmentSource(ctx context.Context, assignment, previous, version string) error {
	_, err := u.tx.Exec(ctx, `INSERT INTO staffing.assignment_publication VALUES($1,$2,$3)`, u.store.Instance, assignment, version)
	if err != nil {
		return err
	}
	if previous == "" {
		_, err = u.tx.Exec(ctx, `INSERT INTO staffing.assignment_head VALUES($1,$2,$3)`, u.store.Instance, assignment, version)
		return err
	}
	result, err := u.tx.Exec(ctx, `UPDATE staffing.assignment_head SET version=$4 WHERE instance_id=$1 AND assignment_id=$2 AND version=$3`, u.store.Instance, assignment, previous, version)
	if err == nil && result.RowsAffected() != 1 {
		return assignmentConflict()
	}
	return err
}
func assignmentConflict() error {
	return f.Fail(409, "ASSIGNMENT_CHANGED", "任职或原办理已更新，请重读后核对")
}

package postgres

import (
	"context"
	"github.com/jackc/pgx/v5"

	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	f "chnhcm.local/chnhcm/internal/foundation"
)

type employmentEntry struct {
	id, revision, employment, assignment, kind, original, date, notice, operation string
	adoption                                                                      any
}

// A relation decision may have been published to an assignment that had
// already ended. Use only the effective nodes belonging to the chosen term.
const activeRecordMembers = ` FROM staffing.assignment_member m
	JOIN staffing.employment_action a ON (a.instance_id,a.id)=(m.instance_id,m.action_id)
	JOIN staffing.employment_action_revision r ON (r.instance_id,r.action_id,r.revision)=(m.instance_id,m.action_id,m.revision)
	WHERE r.operation NOT IN ('Delete','Withdraw') AND a.kind<>'CancelHire'
	AND (a.kind NOT IN ('ProbationStart','ConfirmEmployment') OR EXISTS(SELECT FROM staffing.assignment_segment seg
		WHERE (seg.instance_id,seg.assignment_id,seg.version)=(m.instance_id,m.assignment_id,m.version) AND seg.period @> r.business_date)) `

func (u *unit) newRecordOperation(ctx context.Context, assignment, day string) (string, error) {
	var later bool
	err := u.tx.QueryRow(ctx, `SELECT EXISTS(SELECT`+activeRecordMembers+`
		AND m.instance_id=$1 AND m.assignment_id=$2 AND r.business_date>$3::date
		AND m.version=(SELECT version FROM staffing.assignment_head WHERE instance_id=$1 AND assignment_id=$2))`,
		u.store.Instance, assignment, day).Scan(&later)
	if later {
		return "Insert", err
	}
	return "Add", err
}

func (u *unit) newEmploymentAction(ctx context.Context, e employmentEntry) error {
	if e.operation == "" && !h.StartsAssignment(e.kind) && e.kind != "CancelHire" {
		op, err := u.newRecordOperation(ctx, e.assignment, e.date)
		if err != nil {
			return err
		}
		e.operation = op
	}
	operator, err := u.guard.OperatorSnapshot(ctx)
	if err != nil {
		return err
	}
	_, err = u.tx.Exec(ctx, `WITH new_action AS (
		INSERT INTO staffing.employment_action(instance_id,id,revision,kind,employment_id,assignment_id,original_id,actor)
		VALUES($1,$2,$3,$4,$5,$6,NULLIF($7,'')::uuid,$8) RETURNING *)
		INSERT INTO staffing.employment_action_revision(instance_id,assignment_id,action_id,revision,business_date,notice,actor,recorded_at,adoption,operator_snapshot,operation)
		SELECT instance_id,assignment_id,id,revision,$9::date,$10,actor,recorded_at,$11,$12,coalesce(nullif($13,''),'Add') FROM new_action`,
		u.store.Instance, e.id, e.revision, e.kind, e.employment, e.assignment, e.original, u.guard.Principal().ID, e.date, e.notice, e.adoption, operator, e.operation)
	return err
}

func (u *unit) startAssignmentSource(ctx context.Context, assignment, previous, version, action, revision string) error {
	return u.startSourceReplacing(ctx, assignment, previous, version, action, revision, []h.ActionRef{{Action: action, Revision: revision}})
}

func (u *unit) startSourceReplacing(ctx context.Context, assignment, previous, version, action, revision string, refs []h.ActionRef) error {
	if previous != "" {
		rows, e := u.tx.Query(ctx, `SELECT DISTINCT adoption->>'position' AS id FROM staffing.assignment_segment WHERE instance_id=$1 AND assignment_id=$2 AND version=$3 AND coalesce(adoption->>'position','')<>'' ORDER BY id`, u.store.Instance, assignment, previous)
		if e != nil {
			return e
		}
		ids, e := pgx.CollectRows(rows, pgx.RowTo[string])
		if e != nil {
			return e
		}
		for _, id := range ids {
			if e = u.LockQuotaTarget(ctx, id); e != nil {
				return e
			}
		}
	}
	instance := u.store.Instance
	_, err := u.tx.Exec(ctx, `INSERT INTO staffing.assignment_source(instance_id,assignment_id,version,previous,action_id,revision)
		VALUES($1,$2,$3,NULLIF($4,'')::uuid,$5,$6)`, instance, assignment, version, previous, action, revision)
	if err != nil {
		return err
	}
	ids, revisions := make([]string, len(refs)), make([]string, len(refs))
	for i, ref := range refs {
		ids[i], revisions[i] = ref.Action, ref.Revision
	}
	_, err = u.tx.Exec(ctx, `INSERT INTO staffing.assignment_member(instance_id,assignment_id,version,action_id,revision)
		SELECT instance_id,assignment_id,$3::uuid,action_id,revision FROM staffing.assignment_member
		WHERE instance_id=$1 AND assignment_id=$2 AND version=NULLIF($4,'')::uuid AND NOT (action_id=ANY($5::uuid[]))
		UNION ALL SELECT $1::uuid,$2::uuid,$3::uuid,id,rev FROM unnest($5::uuid[],$6::uuid[]) AS refs(id,rev)`, instance, assignment, version, previous, ids, revisions)
	return err
}

func (u *unit) publishAssignmentSource(ctx context.Context, assignment, previous, version string) error {
	// The caller holds the person's arrangement lock. Check the newly published
	// periods against other current terms of the same working relationship.
	var overlapping bool
	if err := u.tx.QueryRow(ctx, `SELECT EXISTS(SELECT FROM staffing.assignment a
      JOIN staffing.assignment_segment proposed ON proposed.instance_id=a.instance_id AND proposed.assignment_id=a.id
      JOIN staffing.assignment other ON other.instance_id=a.instance_id AND other.employment_id=a.employment_id AND other.id<>a.id
      JOIN staffing.assignment_head current ON current.instance_id=other.instance_id AND current.assignment_id=other.id
      JOIN staffing.assignment_segment active ON (active.instance_id,active.assignment_id,active.version)=(current.instance_id,current.assignment_id,current.version)
      WHERE a.instance_id=$1 AND a.id=$2 AND proposed.version=$3 AND proposed.period && active.period
      AND proposed.adoption->>'position'<>'' AND proposed.adoption->>'position'=active.adoption->>'position')`,
		u.store.Instance, assignment, version).Scan(&overlapping); err != nil {
		return err
	}
	if overlapping {
		return f.Fail(409, "ASSIGNMENT_OVERLAP", "同一工作关系在该期间已有此岗位任职")
	}
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

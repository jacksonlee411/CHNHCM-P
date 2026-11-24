package postgres

import (
	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	f "chnhcm.local/chnhcm/internal/foundation"
	"context"
	"github.com/jackc/pgx/v5"
)

func hirePermission(ref *h.ActionRef) string {
	if ref != nil {
		return "employment.rehire"
	}
	return "hire"
}

func (u *unit) rehireAnchor(ctx context.Context, person, day string) (*h.ActionRef, error) {
	var ref h.ActionRef
	e := u.tx.QueryRow(ctx, `SELECT x.action_id,x.revision FROM staffing.employment_exit x
 JOIN staffing.assignment t ON t.instance_id=x.instance_id AND t.employment_id=x.employment_id AND t.record_number=0
 WHERE x.instance_id=$1 AND t.person_id=$2 AND x.first_inactive<=$3::date AND NOT EXISTS(
 SELECT FROM staffing.employment_action a JOIN staffing.employment_action_revision r ON (r.instance_id,r.action_id,r.revision)=(a.instance_id,a.id,a.revision)
 WHERE a.instance_id=x.instance_id AND a.kind='Rehire' AND r.adoption#>>'{rehireOf,action}'=x.action_id::text
 AND NOT EXISTS(SELECT FROM staffing.employment_action c WHERE c.instance_id=a.instance_id AND c.original_id=a.id))
 ORDER BY x.first_inactive DESC LIMIT 1`, u.store.Instance, person, day).Scan(&ref.Action, &ref.Revision)
	if e == pgx.ErrNoRows {
		return nil, nil
	}
	return &ref, e
}
func (u *unit) checkRehireOrigin(ctx context.Context, person, day string, ref *h.ActionRef) error {
	if !ref.Valid() {
		return f.Fail(400, "REHIRE_SOURCE", "请从本人的原离职办理进入")
	}
	expected, e := u.rehireAnchor(ctx, person, day)
	if e != nil {
		return e
	}
	if ref == nil {
		if expected != nil {
			return f.Fail(409, "REHIRE_REQUIRED", "本人已有离职记录，请从原离职办理再聘")
		}
		return nil
	}
	if expected == nil || *expected != *ref {
		return f.Fail(409, "REHIRE_SOURCE", "请核对原离职办理和拟到岗日")
	}
	original, e := u.action(ctx, ref.Action)
	if e != nil {
		return e
	}
	_, ok, e := u.relationVisibleAssignment(ctx, original, "history")
	if e != nil {
		return e
	}
	if !ok {
		return f.Fail(403, "REHIRE_SCOPE", "当前权限不允许从该段关系办理再聘")
	}
	return nil
}

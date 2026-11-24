package postgres

import (
	"context"
	"errors"

	r "chnhcm.local/chnhcm/internal/contexts/recruiting/public"
	f "chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
)

func (s *Store) handoffUnit(ctx context.Context, c f.Credentials, offer string, forUpdate bool) (*unit, error) {
	tx, ok := dbtx.From(ctx)
	if !ok {
		return nil, errors.New("offer handoff requires the staffing transaction")
	}
	guard, e := s.Guard(ctx, tx, c)
	if e != nil {
		return nil, e
	}
	if forUpdate {
		if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,0))", "recruiting.offer:"+s.Instance+offer); e != nil {
			return nil, e
		}
	}
	return &unit{s, tx, guard.Principal().ID, c}, nil
}
func (s *Store) OfferForHire(ctx context.Context, c f.Credentials, id, expected string, forUpdate bool) (v r.Offer, err error) {
	u, e := s.handoffUnit(ctx, c, id, forUpdate)
	if e != nil {
		return v, e
	}
	v, e = u.load(ctx, id, expected)
	if e != nil {
		return v, e
	}
	ok, e := u.visible(ctx, v, "read")
	if e != nil {
		return v, e
	}
	if !ok {
		return v, f.Fail(403, "OFFER_DENIED", "当前权限无法核对原录用")
	}
	var accepted *string
	e = u.tx.QueryRow(ctx, `SELECT h.version=$3,(SELECT x.accepted_version FROM recruiting.hire_handoff x LEFT JOIN recruiting.handoff_cancellation c USING(instance_id,offer_id,accepted_version) WHERE x.instance_id=h.instance_id AND x.offer_id=h.offer_id AND $3 IN (x.accepted_version,x.converted_version,c.cancelled_version))::text FROM recruiting.offer_head h WHERE h.instance_id=$1 AND h.offer_id=$2`, s.Instance, id, v.Version).Scan(&v.Current, &accepted)
	if e != nil {
		return v, e
	}
	if accepted != nil {
		v.AcceptedVersion = *accepted
	}
	return v, nil
}
func (s *Store) CompleteHire(ctx context.Context, c f.Credentials, v r.Offer, result r.HireResult) (string, error) {
	u, e := s.handoffUnit(ctx, c, v.ID, true)
	if e != nil {
		return "", e
	}
	current, e := u.load(ctx, v.ID, "")
	if e != nil {
		return "", e
	}
	if current.Version != v.Version || current.Status != "accepted" {
		return "", f.Fail(409, "OFFER_CHANGED", "录用状态或预留已变化，请从原录用核对")
	}
	current.Segments = beforeDate(current.Segments, v.Arrival)
	saved, e := u.append(ctx, current, "convert", "converted", v.Arrival, "按已接受录用办理入职")
	if e != nil {
		return "", e
	}
	_, e = u.tx.Exec(ctx, `INSERT INTO recruiting.hire_handoff VALUES($1,$2,$3,$4,$5,$6,$7)`, s.Instance, v.ID, v.Version, saved.Version, result.Action, result.Revision, result.Assignment)
	return saved.Version, e
}
func (s *Store) CancelHireConversion(ctx context.Context, c f.Credentials, ref r.HandoffRef, action, revision, day string) error {
	u, e := s.handoffUnit(ctx, c, ref.Offer, true)
	if e != nil {
		return e
	}
	v, e := u.load(ctx, ref.Offer, "")
	if e != nil {
		return e
	}
	var current bool
	if e = u.tx.QueryRow(ctx, `SELECT EXISTS(SELECT FROM recruiting.hire_handoff WHERE instance_id=$1 AND offer_id=$2 AND accepted_version=$3 AND converted_version=$4)`, s.Instance, ref.Offer, ref.Version, v.Version).Scan(&current); e != nil {
		return e
	}
	if !current || v.Status != "converted" {
		return f.Fail(409, "OFFER_CHANGED", "该入职所采用的录用转换已变化")
	}
	// Conversion already retained exactly the pre-arrival commitments.
	// Undoing the hire must preserve that history.
	saved, e := u.append(ctx, v, "cancel_conversion", "issued", day, "未来入职已撤销，原转换结束")
	if e != nil {
		return e
	}
	_, e = u.tx.Exec(ctx, `INSERT INTO recruiting.handoff_cancellation VALUES($1,$2,$3,$4,$5,$6)`, s.Instance, ref.Offer, ref.Version, saved.Version, action, revision)
	return e
}

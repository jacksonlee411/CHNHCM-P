package postgres

import (
	"context"
	"slices"
	"strings"
	"unicode/utf8"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	s "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	f "chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
)

func (st *Store) Replenishment(ctx context.Context, c f.Credentials, id, day string) (out s.ReplenishmentView, err error) {
	if !f.ValidID(id) || day != "" && !f.ValidDate(day) {
		return out, f.Fail(400, "POSITION_INPUT", "岗位或日期无效")
	}
	err = st.within(ctx, c, false, true, func(u *unit) error {
		if day == "" {
			day = u.now.In(f.Shanghai).Format("2006-01-02")
		}
		var e error
		out, e = u.replenishmentView(ctx, id, day)
		return e
	})
	return
}
func (u *unit) replenishmentView(ctx context.Context, id, day string) (out s.ReplenishmentView, err error) {
	ent, e := u.Enterprise(ctx, false)
	if e != nil {
		return out, e
	}
	ok, e := u.positionReadable(ctx, ent.ID)
	if e != nil {
		return out, e
	}
	if !ok {
		return out, f.Fail(403, "ACCESS_DENIED", "当前权限无法读取岗位补员资料")
	}
	pos, e := u.Load(ctx, d.Position, id, "")
	if e != nil {
		return out, e
	}
	if pos.ID == "" {
		return out, f.Fail(404, "POSITION_UNAVAILABLE", "所选岗位不可用")
	}
	out.CanChange, e = access.RuleAllowed(ctx, u.guard, ent.ID, "replenishment.maintain")
	if e != nil {
		return out, e
	}
	out.Records, e = u.replenishmentRows(ctx, id)
	if e != nil {
		return out, e
	}
	out.Version, e = hireDigest([]any{pos.Source, out.Records})
	if e != nil {
		return out, e
	}
	evidence, e := u.PositionUsage(ctx, id)
	if e != nil {
		return out, e
	}
	for _, c := range evidence.Coverage {
		if c.Owner == "M04" && c.Contains(day) {
			out.Covered = true
		}
	}
	if r := replenishmentAt(out.Records, day); r != nil {
		out.Paused = r.Paused
	}
	quota, e := u.guard.Decide(ctx, access.Intent{EnterpriseID: ent.ID, PositionID: id, Kind: "quota-target", Action: "staffingquota.read"})
	if e != nil {
		return out, e
	}
	if out.Covered && quota.Allowed && slices.Contains(quota.ReadFields, "quota") {
		limit, err := u.LoadPositionQuota(ctx, id, "")
		if err != nil {
			return out, err
		}
		if i := limit.At(day); i >= 0 {
			out.Quota = limit.Records[i].Amount
		}
		n := 0
		for _, v := range evidence.Uses {
			if v.Owner == "M04" && v.Effect == "exempt" && v.Contains(day) {
				n++
			}
		}
		out.Exempt = &n
		if use := evidence.At(day); use.Known {
			out.Reserved, out.Occupied = &use.Reserved, &use.Occupied
			if out.Quota != nil {
				available := max(0, *out.Quota-use.Occupied-use.Reserved)
				excess := max(0, use.Occupied-*out.Quota)
				out.Available, out.Excess = &available, &excess
			}
		}
	}
	return out, nil
}
func (st *Store) ChangeReplenishment(ctx context.Context, c f.Credentials, id, key string, in s.ReplenishmentChange) (out s.ReplenishmentView, err error) {
	in.Reason = strings.TrimSpace(in.Reason)
	if !f.ValidID(id) || !f.ValidID(key) || !f.ValidDate(in.Date) || in.Expected == "" || in.Reason == "" || utf8.RuneCountInString(in.Reason) > 1000 {
		return out, f.Fail(400, "REPLENISHMENT_INPUT", "请核对岗位、日期及补员原因")
	}
	err = st.within(ctx, c, true, false, func(u *unit) error {
		if e := lock(ctx, u.tx, false, "staffing.replenishment", st.Instance, u.guard.Principal().ID, key); e != nil {
			return e
		}
		if e := u.LockQuotaTarget(ctx, id); e != nil {
			return e
		}
		v, e := u.replenishmentView(ctx, id, in.Date)
		if e != nil {
			return e
		}
		if !v.CanChange {
			return f.Fail(403, "ACCESS_DENIED", "当前权限无法维护岗位补员")
		}
		digest, e := hireDigest([]any{id, in})
		if e != nil {
			return e
		}
		var prev string
		e = u.tx.QueryRow(ctx, "SELECT digest FROM staffing.position_replenishment WHERE instance_id=$1 AND actor=$2 AND revision=$3", st.Instance, u.guard.Principal().ID, key).Scan(&prev)
		if e == nil {
			if prev != digest {
				return f.Fail(409, "REPLENISHMENT_KEY", "该提交已经用于另一项补员安排")
			}
			out = v
			return nil
		}
		if e != pgx.ErrNoRows {
			return e
		}
		var closed bool
		if e = u.tx.QueryRow(ctx, "SELECT EXISTS(SELECT FROM staffing.position_replenishment_closed WHERE instance_id=$1 AND actor=$2 AND submission_id=$3)", st.Instance, u.guard.Principal().ID, key).Scan(&closed); e != nil {
			return e
		}
		if closed {
			return f.Fail(409, "REPLENISHMENT_CLOSED", "该提交已查回为未办理，请重新发起")
		}
		if v.Version != in.Expected {
			return f.Fail(409, "M03_SOURCE_CHANGED", "补员安排已变化，请重新核对")
		}
		pos, e := u.Load(ctx, d.Position, id, "")
		if e != nil {
			return e
		}
		i := d.At(pos, in.Date)
		if i < 0 || !pos.Records[i].Active() {
			return f.Fail(409, "POSITION_PERIOD", "所选日期没有有效岗位")
		}
		for _, r := range v.Records {
			if r.Date == in.Date {
				return f.Fail(409, "REPLENISHMENT_DATE", "当日已有补员决定")
			}
		}
		if in.Paused == v.Paused {
			return f.Fail(409, "REPLENISHMENT_STATE", "请核对该日原补员状态")
		}
		_, e = u.tx.Exec(ctx, "INSERT INTO staffing.position_replenishment(instance_id,position_id,revision,actor,business_date,paused,reason,digest) VALUES($1,$2,$3,$4,$5::date,$6,$7,$8)", st.Instance, id, key, u.guard.Principal().ID, in.Date, in.Paused, in.Reason, digest)
		if e != nil {
			return e
		}
		out, e = u.replenishmentView(ctx, id, in.Date)
		return e
	})
	return
}

func (st *Store) ResolveReplenishment(ctx context.Context, c f.Credentials, id, key string) (out f.RuleReceipt, err error) {
	if !f.ValidID(id) || !f.ValidID(key) {
		return out, f.Fail(400, "REPLENISHMENT_INPUT", "岗位或提交号无效")
	}
	err = st.within(ctx, c, true, false, func(u *unit) error {
		if e := lock(ctx, u.tx, false, "staffing.replenishment", st.Instance, u.guard.Principal().ID, key); e != nil {
			return e
		}
		var position string
		e := u.tx.QueryRow(ctx, "SELECT position_id FROM staffing.position_replenishment WHERE instance_id=$1 AND actor=$2 AND revision=$3", st.Instance, u.guard.Principal().ID, key).Scan(&position)
		if e == nil {
			if position != id {
				return f.Fail(409, "REPLENISHMENT_KEY", "该提交已用于另一岗位")
			}
			out = f.RuleReceipt{SubmissionID: key, Outcome: "accepted", Version: key}
			return nil
		}
		if e != pgx.ErrNoRows {
			return e
		}
		_, e = u.tx.Exec(ctx, "INSERT INTO staffing.position_replenishment_closed(instance_id,actor,submission_id) VALUES($1,$2,$3) ON CONFLICT DO NOTHING", st.Instance, u.guard.Principal().ID, key)
		out = f.RuleReceipt{SubmissionID: key, Outcome: "not_applied"}
		return e
	})
	return
}

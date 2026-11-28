package postgres

import (
	"context"
	"errors"
	"time"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	d "chnhcm.local/chnhcm/internal/contexts/compensation/domain"
	c "chnhcm.local/chnhcm/internal/contexts/compensation/public"
	st "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
)

type awardRow struct {
	c.PayAward
	adoption awardAdoption
}

const awardSelect = `SELECT r.award_id::text,r.revision::text,r.assignment_id::text,r.effective_date::text,r.action,r.amount::text,
 r.reason,r.note,r.recorded_at,r.operator_id::text,coalesce(r.previous::text,''),r.adoption,
 EXISTS(SELECT FROM compensation.award_period p WHERE (p.instance_id,p.award_id,p.revision)=(r.instance_id,r.award_id,r.revision))
 FROM compensation.award_revision r `

func scanAward(row pgx.Row) (v awardRow, err error) {
	var stamp time.Time
	err = row.Scan(&v.ID, &v.Revision, &v.Assignment, &v.Day, &v.Action, &v.Amount, &v.Reason, &v.Note, &stamp, &v.Operator, &v.Previous, &v.adoption, &v.Current)
	v.RecordedAt = stamp.UTC().Format(time.RFC3339Nano)
	return
}
func (s *Store) lookupAward(ctx context.Context, u *unit, id, rev string) (awardRow, error) {
	if !foundation.ValidID(id) || !foundation.ValidID(rev) {
		return awardRow{}, unavailable()
	}
	v, e := scanAward(u.tx.QueryRow(ctx, awardSelect+"WHERE r.instance_id=$1 AND r.award_id=$2 AND r.revision=$3", s.Instance, id, rev))
	if errors.Is(e, pgx.ErrNoRows) {
		return v, unavailable()
	}
	return v, e
}
func (s *Store) awardTerm(ctx context.Context, u *unit, id, day string, write bool) (term st.PayAssignment, rights access.BasePayRights, err error) {
	if !foundation.ValidDate(day) {
		err = d.Invalid("effectiveDate", "请输入有效日期")
		return
	}
	rights, err = s.PayAccessControl.BasePayRights(ctx, u.guard.Principal().ID, id)
	if err != nil {
		return
	}
	if !rights.Read {
		err = unavailable()
		return
	}
	term, err = s.Assignments.PayAssignment(ctx, u.enterprise.ID, u.guard.Principal().ID, id, day, write)
	return
}
func (s *Store) PayAccess(ctx context.Context, cred foundation.Credentials, id, day string) (ok bool, err error) {
	if !foundation.ValidID(id) {
		return false, unavailable()
	}
	err = s.within(ctx, cred, false, func(ctx context.Context, u *unit) error {
		if day == "" {
			day = u.today
		}
		_, _, e := s.awardTerm(ctx, u, id, day, false)
		if f, yes := fault(e); yes && (f.Status == 404 || f.Status == 403) {
			return nil
		}
		ok = e == nil
		return e
	})
	return
}
func (s *Store) payHead(ctx context.Context, u *unit, id string) (version string, err error) {
	err = u.tx.QueryRow(ctx, "SELECT coalesce((SELECT head::text FROM compensation.award_timeline WHERE instance_id=$1 AND assignment_id=$2),'')", s.Instance, id).Scan(&version)
	return
}
func (s *Store) awardDetails(ctx context.Context, u *unit, r awardRow, history bool) (c.PayAward, error) {
	out := r.PayAward
	if history {
		v, e := u.awardVisibility(ctx, true)
		if e != nil {
			return out, e
		}
		out.Basis = v.project(r.adoption.Basis)
	} else {
		out.Operator, out.Note, out.Previous, out.RecordedAt, out.Reason = "", "", "", "", ""
	}
	return out, nil
}
func (s *Store) readOriginal(ctx context.Context, u *unit, id, rev string, receipt bool) (c.PayAward, error) {
	r, e := s.lookupAward(ctx, u, id, rev)
	if e != nil {
		return c.PayAward{}, e
	}
	_, rights, e := s.awardTerm(ctx, u, r.Assignment, r.Day, false)
	if e != nil {
		return c.PayAward{}, e
	}
	if !receipt && !rights.History {
		return c.PayAward{}, unavailable()
	}
	if receipt && !rights.History && !r.Current {
		return c.PayAward{}, unavailable()
	}
	return s.awardDetails(ctx, u, r, rights.History)
}
func (s *Store) AwardOriginal(ctx context.Context, cred foundation.Credentials, id, rev string) (v c.PayAward, err error) {
	err = s.within(ctx, cred, false, func(ctx context.Context, u *unit) error {
		var e error
		v, e = s.readOriginal(ctx, u, id, rev, false)
		return e
	})
	return
}
func (s *Store) AwardReceipt(ctx context.Context, cred foundation.Credentials, key string) (v c.AwardReceipt, err error) {
	if !foundation.ValidID(key) {
		return v, unavailable()
	}
	err = s.within(ctx, cred, false, func(ctx context.Context, u *unit) error {
		var e error
		v, e = s.readAwardReceipt(ctx, u, key)
		return e
	})
	return
}
func (s *Store) BasePay(ctx context.Context, cred foundation.Credentials, id string, query c.PayQuery) (out c.PayDetail, err error) {
	if !foundation.ValidID(id) {
		return out, unavailable()
	}
	day := query.Day
	if query.Page < 1 || query.Page > 1000000 || query.Plan != "" && !foundation.ValidID(query.Plan) {
		return out, d.Invalid("candidatePage", "请选择有效的计划或页码")
	}
	out = c.PayDetail{CandidatePageSize: 20, CandidatePage: 1, Assignment: id, Actions: []string{}, Candidates: []c.PayBasis{}, Items: []c.PayAward{}}
	err = s.within(ctx, cred, false, func(ctx context.Context, u *unit) error {
		if day == "" {
			day = u.today
		}
		out.BusinessAt = day
		term, rights, e := s.awardTerm(ctx, u, id, day, false)
		if e != nil {
			return e
		}
		out.Active = term.Active
		if out.Version, e = s.payHead(ctx, u, id); e != nil {
			return e
		}
		var live int
		if e = u.tx.QueryRow(ctx, "SELECT count(*) FROM compensation.award_period WHERE instance_id=$1 AND assignment_id=$2", s.Instance, id).Scan(&live); e != nil {
			return e
		}
		if rights.First && live == 0 && term.Active {
			out.Actions = append(out.Actions, "FIRST")
		}
		if rights.Adjust && live > 0 && term.Active {
			out.Actions = append(out.Actions, "ADJUST")
		}
		if rights.History && rights.Correct {
			out.Actions = append(out.Actions, "CORRECT")
		}
		if rights.History && rights.Withdraw {
			out.Actions = append(out.Actions, "WITHDRAW")
		}
		v, e := u.awardVisibility(ctx, false)
		if e != nil {
			return e
		}
		var policyID string
		if term.Active {
			r, e := scanAward(u.tx.QueryRow(ctx, awardSelect+`JOIN compensation.award_period p ON (r.instance_id,r.award_id,r.revision)=(p.instance_id,p.award_id,p.revision) WHERE p.instance_id=$1 AND p.assignment_id=$2 AND p.valid_period @> $3::date`, s.Instance, id, day))
			if e == nil {
				current, e := s.awardDetails(ctx, u, r, false)
				if e != nil {
					return e
				}
				out.Current = &current
				policyID = r.adoption.Basis.PlanID
			} else if !errors.Is(e, pgx.ErrNoRows) {
				return e
			}
		}
		if query.Prepare || policyID == "" {
			if out.Candidates, e = s.awardCandidates(ctx, u, term, day, rights, v); e != nil {
				return e
			}
			out.CandidateCount = len(out.Candidates)
			for i := range out.Candidates {
				if query.Plan == out.Candidates[i].PlanID {
					out.SelectedBasis = &out.Candidates[i]
					break
				}
			}
			out.CandidatePage = min(query.Page, max(1, (out.CandidateCount+out.CandidatePageSize-1)/out.CandidatePageSize))
			start := (out.CandidatePage - 1) * out.CandidatePageSize
			out.Candidates = out.Candidates[start:min(start+out.CandidatePageSize, out.CandidateCount)]
		}
		switch {
		case !term.Active:
			out.Message = "该日无有效任职"
		case term.Grade == "":
			out.Message = "该任期缺少实际职级，请按任用依据补记后重读"
		case policyID != "" && len(v.plans) > 0:
			current, e := s.resolveAward(ctx, u, term, day, policyID, false)
			if e != nil {
				return e
			}
			if current != nil {
				out.Policy = v.project(current.Basis)
			} else {
				out.Message = "原计划或薪带在查询日已不适用"
			}
		case len(out.Candidates) == 0 && (rights.UseAll || len(rights.Plans) > 0):
			out.Message = "该日没有可采用的基本月薪计划"
		}
		if query.Prepare || !rights.History {
			return nil
		}
		rows, e := u.tx.Query(ctx, awardSelect+"WHERE r.instance_id=$1 AND r.assignment_id=$2 ORDER BY r.effective_date DESC,r.recorded_at DESC,r.revision", s.Instance, id)
		if e != nil {
			return e
		}
		var records []awardRow
		for rows.Next() {
			r, e := scanAward(rows)
			if e != nil {
				rows.Close()
				return e
			}
			records = append(records, r)
		}
		rows.Close()
		if e = rows.Err(); e != nil {
			return e
		}
		visible := map[string]bool{day: true}
		for _, r := range records {
			if _, known := visible[r.Day]; !known {
				_, e := s.Assignments.PayAssignment(ctx, u.enterprise.ID, u.guard.Principal().ID, id, r.Day, false)
				f, yes := fault(e)
				if e != nil && (!yes || f.Status != 404 && f.Status != 403) {
					return e
				}
				visible[r.Day] = e == nil
			}
			if visible[r.Day] {
				item := r.PayAward
				item.Basis = nil
				if rights.Withdraw && item.Current {
					var amount string
					if e := u.tx.QueryRow(ctx, `SELECT coalesce((SELECT prev.amount::text FROM compensation.award_period p JOIN compensation.award_revision prev
                      ON (prev.instance_id,prev.award_id,prev.revision)=(p.instance_id,p.award_id,p.revision)
                      WHERE p.instance_id=$1 AND p.assignment_id=$2 AND p.start_date<$3 ORDER BY p.start_date DESC LIMIT 1),'')`, s.Instance, id, item.Day).Scan(&amount); e != nil {
						return e
					}
					item.Restoration = &c.AwardRestoration{Amount: amount}
				}
				out.Items = append(out.Items, item)
			}
		}
		return nil
	})
	return
}

func (s *Store) readAwardReceipt(ctx context.Context, u *unit, key string) (v c.AwardReceipt, err error) {
	var id, rev string
	e := u.tx.QueryRow(ctx, "SELECT award_id,revision FROM compensation.award_receipt WHERE instance_id=$1 AND subject_id=$2 AND submission_id=$3", s.Instance, u.guard.Principal().ID, key).Scan(&id, &rev)
	if errors.Is(e, pgx.ErrNoRows) {
		return v, unavailable()
	}
	if e != nil {
		return v, e
	}
	v.Submission = key
	v.Award, e = s.readOriginal(ctx, u, id, rev, true)
	return v, e
}

func (s *Store) ResolveAward(ctx context.Context, cred foundation.Credentials, key string, input c.RecoveryInput) (out c.AwardResolution, err error) {
	if !foundation.ValidID(key) {
		return out, unavailable()
	}
	if (input.Command == nil) != (input.Assignment == "") || input.Assignment != "" && !foundation.ValidID(input.Assignment) {
		return out, d.Invalid("assignmentId", "请选择原申请的任职")
	}
	err = s.within(ctx, cred, true, func(ctx context.Context, u *unit) error {
		subject := u.guard.Principal().ID
		if e := u.lock(ctx, "base-pay/receipt/"+s.Instance+"/"+subject+"/"+key); e != nil {
			return e
		}
		var exists bool
		if e := u.tx.QueryRow(ctx, "SELECT EXISTS(SELECT FROM compensation.award_receipt WHERE instance_id=$1 AND subject_id=$2 AND submission_id=$3)", s.Instance, subject, key).Scan(&exists); e != nil {
			return e
		}
		out.Submission = key
		if exists {
			r, e := s.readAwardReceipt(ctx, u, key)
			if e != nil {
				return e
			}
			out.Outcome = "accepted"
			out.Award = &r.Award
			return nil
		}
		var digest []byte
		if input.Command != nil {
			if _, _, e := s.awardTerm(ctx, u, input.Assignment, input.Command.EffectiveDate, false); e != nil {
				return e
			}
			var e error
			digest, e = normalizedAward(input.Assignment, input.Command)
			if e != nil {
				return e
			}
		}
		_, e := u.tx.Exec(ctx, "INSERT INTO compensation.award_recovery VALUES($1,$2,$3,$4,$5) ON CONFLICT DO NOTHING", s.Instance, subject, key, foundation.NewID(), digest)
		if e != nil {
			return e
		}
		out.Outcome = "not_applied"
		return u.tx.QueryRow(ctx, "SELECT confirmation::text FROM compensation.award_recovery WHERE instance_id=$1 AND subject_id=$2 AND submission_id=$3", s.Instance, subject, key).Scan(&out.Confirmation)
	})
	return
}

package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"

	d "chnhcm.local/chnhcm/internal/contexts/compensation/domain"
	c "chnhcm.local/chnhcm/internal/contexts/compensation/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
)

func (s *Store) ConfirmAward(ctx context.Context, cred foundation.Credentials, assignment, key, confirmation string, in c.ConfirmAward) (out c.AwardReceipt, err error) {
	if !foundation.ValidID(key) || !foundation.ValidID(assignment) {
		return out, d.Invalid("submissionId", "提交号或任职身份无效")
	}
	original := in.Action == "CORRECT" || in.Action == "WITHDRAW"
	if (!original && in.Action != "FIRST" && in.Action != "ADJUST") || !foundation.ValidDate(in.EffectiveDate) {
		return out, d.Invalid("effectiveDate", "请选择办理动作和合法生效日")
	}
	if original && (!foundation.ValidID(in.ID) || !foundation.ValidID(in.Revision)) || !original && (in.ID != "" || in.Revision != "") {
		return out, d.Invalid("awardId", "请选定准确的核定原笔")
	}
	digest, e := normalizedAward(assignment, &in)
	if e != nil {
		return out, e
	}
	err = s.within(ctx, cred, true, func(ctx context.Context, u *unit) error {
		subject := u.guard.Principal().ID
		if e := u.lock(ctx, "base-pay/receipt/"+s.Instance+"/"+subject+"/"+key); e != nil {
			return e
		}
		var oldHash []byte
		var id, rev string
		e := u.tx.QueryRow(ctx, "SELECT digest,award_id,revision FROM compensation.award_receipt WHERE instance_id=$1 AND subject_id=$2 AND submission_id=$3", s.Instance, subject, key).Scan(&oldHash, &id, &rev)
		if e == nil {
			award, e := s.readOriginal(ctx, u, id, rev, true)
			if e != nil {
				return e
			}
			if !bytes.Equal(digest[:], oldHash) {
				return d.Conflict()
			}
			out = c.AwardReceipt{Submission: key, Award: award}
			return nil
		}
		if !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		var token string
		var confirmedHash []byte
		e = u.tx.QueryRow(ctx, "SELECT confirmation::text,digest FROM compensation.award_recovery WHERE instance_id=$1 AND subject_id=$2 AND submission_id=$3", s.Instance, subject, key).Scan(&token, &confirmedHash)
		if e == nil {
			if confirmation != token || len(confirmedHash) == 0 || !bytes.Equal(confirmedHash, digest) {
				return foundation.Fail(409, "M09_AWARD_CONFIRMATION", "请按原申请号重送已经核对的申请")
			}
		} else if !errors.Is(e, pgx.ErrNoRows) {
			return e
		} else if confirmation != "" {
			return d.Conflict()
		}
		if e = u.lock(ctx, "base-pay/term/"+s.Instance+"/"+assignment); e != nil {
			return e
		}
		term, rights, e := s.awardTerm(ctx, u, assignment, in.EffectiveDate, true)
		if e != nil {
			return e
		}
		switch in.Action {
		case "FIRST":
			if !rights.First {
				return denied()
			}
		case "ADJUST":
			if !rights.Adjust {
				return denied()
			}
		case "CORRECT":
			if !rights.History || !rights.Correct {
				return denied()
			}
		case "WITHDRAW":
			if !rights.History || !rights.Withdraw {
				return denied()
			}
		}
		if !term.Active && in.Action != "WITHDRAW" && !(in.Action == "CORRECT" && in.Amount == nil && in.Plan == "") {
			return d.Invalid("effectiveDate", "所选日无有效任职")
		}
		head, e := s.payHead(ctx, u, assignment)
		if e != nil {
			return e
		}
		if head != in.ExpectedVersion {
			return d.Conflict()
		}
		r := awardRow{PayAward: c.PayAward{ID: foundation.NewID(), Assignment: assignment, Day: in.EffectiveDate}}
		if original {
			r, e = s.lookupAward(ctx, u, in.ID, in.Revision)
			if e != nil {
				return e
			}
			if r.Assignment != assignment || !r.Current || r.Day != in.EffectiveDate {
				return d.Conflict()
			}
			if in.Note == "" {
				return d.Invalid("note", "请说明本次更正或撤去的缘由")
			}
			r.Previous = r.Revision
		} else {
			var count int
			var occupied bool
			if e = u.tx.QueryRow(ctx, "SELECT count(*),coalesce(bool_or(start_date=$3::date),false) FROM compensation.award_period WHERE instance_id=$1 AND assignment_id=$2", s.Instance, assignment, in.EffectiveDate).Scan(&count, &occupied); e != nil {
				return e
			}
			if occupied {
				return d.Invalid("effectiveDate", "该日已有核定，请定位原笔核对")
			}
			if (count == 0) != (in.Action == "FIRST") {
				return d.Conflict()
			}
		}
		old := r
		r.Revision, r.Action = foundation.NewID(), in.Action
		if in.Reason != nil {
			r.Reason = *in.Reason
		}
		if in.Action == "ADJUST" && r.Reason == "" {
			return d.Invalid("reason", "请填写调薪原因")
		}
		r.Note = in.Note
		if in.Action == "WITHDRAW" {
			if in.Amount != nil || in.Plan != "" || in.Component != "" || in.BasisToken != "" || in.Reason != nil {
				return d.Invalid("action", "撤去使用选定原笔和本次缘由")
			}
		} else if !original || in.Amount != nil || in.Plan != "" || in.Component != "" || in.BasisToken != "" {
			if !term.Active {
				return d.Invalid("effectiveDate", "所选日无有效任职")
			}
			plan := in.Plan
			if plan == "" {
				plan = r.adoption.Basis.PlanID
			}
			if !foundation.ValidID(plan) {
				return d.Invalid("planId", "请选择计划")
			}
			if !rights.UseAll && !rights.Plans[plan] {
				return denied()
			}
			v, e := u.awardVisibility(ctx, false)
			if e != nil {
				return e
			}
			if !v.canChoose() {
				return denied()
			}
			adopted, e := s.resolveAward(ctx, u, term, in.EffectiveDate, plan, true)
			if e != nil {
				return e
			}
			if adopted == nil {
				return d.Invalid("planId", "该计划在生效日无法用于本任期的基本月薪")
			}
			if adopted.Basis.Token != in.BasisToken || in.Component != adopted.Basis.ComponentID {
				return d.Conflict()
			}
			if in.Amount != nil {
				r.Amount = *in.Amount
			}
			if r.Amount == "" {
				return d.Invalid("amount", "请填写个人基本工资")
			}
			r.Amount, e = d.MonthlyAwardAmount(r.Amount, d.Amounts{Form: "RANGE", Minimum: adopted.Basis.Minimum, Maximum: adopted.Basis.Maximum})
			if e != nil {
				return e
			}
			r.adoption = *adopted
		}
		if in.Action == "CORRECT" && old.Amount == r.Amount && old.Reason == r.Reason && old.adoption.Basis.PlanID == r.adoption.Basis.PlanID && old.adoption.Basis.ComponentID == r.adoption.Basis.ComponentID {
			return d.Invalid("amount", "核定内容没有变化")
		}
		if in.Action == "ADJUST" {
			prior, e := scanAward(u.tx.QueryRow(ctx, awardSelect+`JOIN compensation.award_period p ON (p.instance_id,p.award_id,p.revision)=(r.instance_id,r.award_id,r.revision) WHERE p.instance_id=$1 AND p.assignment_id=$2 AND p.start_date<$3 ORDER BY p.start_date DESC LIMIT 1`, s.Instance, assignment, r.Day))
			if e != nil && !errors.Is(e, pgx.ErrNoRows) {
				return e
			}
			if e == nil && prior.Amount == r.Amount && prior.adoption.Basis.PlanID == r.adoption.Basis.PlanID && prior.adoption.Basis.ComponentID == r.adoption.Basis.ComponentID {
				return d.Invalid("amount", "与前一份核定相比没有变化")
			}
		}
		_, e = u.tx.Exec(ctx, `INSERT INTO compensation.award_revision(instance_id,assignment_id,award_id,revision,effective_date,amount,previous,action,reason,note,adoption,operator_id)
 VALUES($1,$2,$3,$4,$5::date,$6::numeric,nullif($7,'')::uuid,$8,$9,$10,$11,$12)`, s.Instance, assignment, r.ID, r.Revision, r.Day, r.Amount, r.Previous, r.Action, r.Reason, r.Note, r.adoption, subject)
		if e != nil {
			return e
		}
		if e = s.applyAwardPeriod(ctx, u, r); e != nil {
			return e
		}
		if _, e = u.tx.Exec(ctx, "INSERT INTO compensation.award_timeline VALUES($1,$2,$3) ON CONFLICT(instance_id,assignment_id) DO UPDATE SET head=EXCLUDED.head", s.Instance, assignment, r.Revision); e != nil {
			return e
		}
		if _, e = u.tx.Exec(ctx, "INSERT INTO compensation.award_receipt VALUES($1,$2,$3,$4,$5,$6)", s.Instance, subject, key, digest[:], r.ID, r.Revision); e != nil {
			return e
		}
		if _, e = u.tx.Exec(ctx, "INSERT INTO compensation.award_outbox VALUES($1,$2,$3,$4)", s.Instance, assignment, r.ID, r.Revision); e != nil {
			return e
		}
		out.Submission = key
		out.Award, e = s.readOriginal(ctx, u, r.ID, r.Revision, true)
		return e
	})
	return
}

func (s *Store) applyAwardPeriod(ctx context.Context, u *unit, r awardRow) error {
	args := []any{s.Instance, r.Assignment, r.Day}
	if r.Action == "CORRECT" {
		_, e := u.tx.Exec(ctx, "UPDATE compensation.award_period SET revision=$4 WHERE instance_id=$1 AND assignment_id=$2 AND start_date=$3", append(args, r.Revision)...)
		return e
	}
	var until string
	e := u.tx.QueryRow(ctx, "SELECT coalesce(min(start_date)::text,'') FROM compensation.award_period WHERE instance_id=$1 AND assignment_id=$2 AND start_date>$3", args...).Scan(&until)
	if e != nil {
		return e
	}
	end := r.Day
	if r.Action == "WITHDRAW" {
		if _, e = u.tx.Exec(ctx, "DELETE FROM compensation.award_period WHERE instance_id=$1 AND assignment_id=$2 AND start_date=$3", args...); e != nil {
			return e
		}
		end = until
	}
	_, e = u.tx.Exec(ctx, `UPDATE compensation.award_period SET until_date=nullif($4,'')::date WHERE instance_id=$1 AND assignment_id=$2 AND start_date=(SELECT max(start_date) FROM compensation.award_period WHERE instance_id=$1 AND assignment_id=$2 AND start_date<$3)`, append(args, end)...)
	if e != nil || r.Action == "WITHDRAW" {
		return e
	}
	_, e = u.tx.Exec(ctx, "INSERT INTO compensation.award_period(instance_id,assignment_id,start_date,award_id,revision,until_date) VALUES($1,$2,$3,$4,$5,nullif($6,'')::date)", append(args, r.ID, r.Revision, until)...)
	return e
}

func normalizedAward(assignment string, in *c.ConfirmAward) ([]byte, error) {
	if in.Reason != nil {
		v := strings.TrimSpace(*in.Reason)
		in.Reason = &v
	}
	in.Note = strings.TrimSpace(in.Note)
	if utf8.RuneCountInString(in.Note) > 1000 || in.Reason != nil && utf8.RuneCountInString(*in.Reason) > 1000 {
		return nil, d.Invalid("reason", "原因说明限一千字")
	}
	if in.Amount != nil {
		if len(*in.Amount) > 100 {
			return nil, d.Invalid("amount", "金额超出支持范围")
		}
		amount, _, e := d.Decimal("amount", *in.Amount, 2)
		if e != nil {
			return nil, e
		}
		in.Amount = &amount
	}
	payload, e := json.Marshal(struct {
		Assignment string
		Input      c.ConfirmAward
	}{assignment, *in})
	if e != nil {
		return nil, e
	}
	digest := sha256.Sum256(payload)
	return digest[:], nil

}

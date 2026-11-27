package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"slices"

	app "chnhcm.local/chnhcm/internal/contexts/compensation/application"
	d "chnhcm.local/chnhcm/internal/contexts/compensation/domain"
	c "chnhcm.local/chnhcm/internal/contexts/compensation/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type storedReceipt struct {
	c.Receipt
	Failure *foundation.Fault `json:"failure,omitempty"`
	Status  int               `json:"httpStatus,omitempty"`
	Action  string            `json:"action"`
	Fields  []string          `json:"fields,omitempty"`
}

func (s *Store) receipt(ctx context.Context, u *unit, id string) (storedReceipt, []byte, error) {
	var receipt storedReceipt
	var raw, digest []byte
	err := u.tx.QueryRow(ctx, "SELECT digest,result FROM compensation.grade_rate_receipt WHERE instance_id=$1 AND subject_id=$2 AND submission_id=$3", s.Instance, u.guard.Principal().ID, id).Scan(&digest, &raw)
	if err != nil {
		return receipt, nil, err
	}
	err = json.Unmarshal(raw, &receipt)
	if receipt.Failure != nil {
		receipt.Failure.Status = receipt.Status
	}
	return receipt, digest, err
}
func readReceipt(ctx context.Context, u *unit, r storedReceipt) (c.Receipt, error) {
	if _, err := u.requireRateRead(ctx); err != nil {
		return c.Receipt{}, err
	}
	if r.Failure != nil {
		if _, err := u.rateWriteGrant(ctx, r.Action, r.Fields); err != nil {
			return c.Receipt{}, err
		}
		return c.Receipt{}, r.Failure
	}
	return r.Receipt, nil
}
func (s *Store) RateReceipt(ctx context.Context, cred foundation.Credentials, id string) (out c.Receipt, err error) {
	if !foundation.ValidID(id) {
		return out, unavailable()
	}
	err = s.within(ctx, cred, false, func(ctx context.Context, u *unit) error {
		if _, e := u.requireRateRead(ctx); e != nil {
			return e
		}
		r, _, e := s.receipt(ctx, u, id)
		if errors.Is(e, pgx.ErrNoRows) {
			return unavailable()
		}
		if e != nil {
			return e
		}
		out, e = readReceipt(ctx, u, r)
		return e
	})
	return
}

func (s *Store) SaveRate(ctx context.Context, cred foundation.Credentials, id, submission string, input c.WriteRate) (out c.Receipt, err error) {
	if !foundation.ValidID(submission) || id != "" && !foundation.ValidID(id) || (id == "") != (input.Action == "CREATE") {
		return out, d.Invalid("submissionId", "提交或目标身份无效")
	}
	fields, err := app.Fields(input)
	if err != nil {
		return out, err
	}
	b, err := json.Marshal(struct {
		ID    string
		Input c.WriteRate
	}{id, input})
	if err != nil {
		return out, err
	}
	sum := sha256.Sum256(b)
	var outcomeErr error
	err = s.within(ctx, cred, true, func(ctx context.Context, u *unit) error {
		if _, e := u.requireRateRead(ctx); e != nil {
			return e
		}
		if e := u.lock(ctx, "compensation.receipt/"+s.Instance+"/"+u.guard.Principal().ID+"/"+submission); e != nil {
			return e
		}
		saved, digest, e := s.receipt(ctx, u, submission)
		if e == nil {
			if saved.Outcome == "not_applied" {
				return foundation.Fail(409, "M09_SUBMISSION_CLOSED", "本次提交已确认未执行，请重新办理")
			}
			if !bytes.Equal(digest, sum[:]) {
				return foundation.Fail(409, "M09_SUBMISSION_CONFLICT", "此提交号已用于其他内容")
			}
			out, outcomeErr = readReceipt(ctx, u, saved)
			return nil
		}
		if !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		if _, e = u.rateWriteGrant(ctx, input.Action, fields); e != nil {
			return e
		}
		// Reject a business failure atomically without retaining its partial
		// writes. Infrastructure errors remain retryable under this submission.
		child, e := u.tx.Begin(ctx)
		if e != nil {
			return e
		}
		work := *u
		work.tx = child
		out, e = s.applyRate(ctx, &work, id, input, fields)
		out.SubmissionID = submission
		receipt := storedReceipt{Receipt: out, Action: input.Action, Fields: fields}
		if e != nil {
			if rollbackErr := child.Rollback(ctx); rollbackErr != nil {
				return rollbackErr
			}
			f, ok := fault(e)
			if !ok || f.Status >= 500 || f.Status == 403 || f.Status == 404 {
				return e
			}
			receipt.Failure, receipt.Status = f, f.Status
			receipt.Outcome = "rejected"
			outcomeErr = e
		} else if e = child.Commit(ctx); e != nil {
			return e
		}
		payload, e := json.Marshal(receipt)
		if e != nil {
			return e
		}
		_, e = u.tx.Exec(ctx, "INSERT INTO compensation.grade_rate_receipt(instance_id,subject_id,submission_id,digest,result) VALUES($1,$2,$3,$4,$5)", s.Instance, u.guard.Principal().ID, submission, sum[:], payload)
		return e
	})
	if err == nil {
		err = outcomeErr
	}
	return
}

func (s *Store) applyRate(ctx context.Context, u *unit, id string, in c.WriteRate, explicit []string) (c.Receipt, error) {
	var out c.Receipt
	var state app.RateState
	var err error
	if id == "" {
		state, err = app.NewRate(in)
		state.Enterprise = u.enterprise.ID
	} else {
		state, err = s.loadRate(ctx, u, id, true)
	}
	if err != nil {
		return out, err
	}
	if state.Version != in.ExpectedVersion {
		return out, d.Conflict()
	}
	if in.EffectiveDate != "" && in.EffectiveDate < u.enterprise.ValidFrom {
		return out, d.Invalid("effectiveDate", "生效日期早于企业起始日期")
	}
	if next, changing := in.Changes["form"]; changing && next != nil && id != "" {
		for _, r := range state.Records {
			if r.ID != in.RecordID || r.Data.Amounts.Form == *next {
				continue
			}
			removed := []string{"amount"}
			if r.Data.Amounts.Form == "RANGE" {
				removed = []string{"minimum", "guidance", "maximum"}
			}
			if _, e := u.rateWriteGrant(ctx, in.Action, append(slices.Clone(explicit), removed...)); e != nil {
				return out, e
			}
			break
		}
	}
	gd, err := u.require(ctx, "grade.read", nil)
	if err != nil {
		return out, err
	}
	if !slices.Contains(gd.ReadFields, "name") {
		return out, denied()
	}
	grade, err := s.Grades.CompensationGrade(ctx, u.enterprise.ID, state.Key.GradeID, true)
	if err != nil {
		return out, err
	}
	change, err := app.PrepareRate(state, in, grade)
	if err != nil {
		return out, err
	}
	out = c.Receipt{ObjectID: state.ID, RecordID: change.Target.ID, Revision: change.Target.Revision, Version: state.Version, Outcome: "no_change"}
	if !change.Changed {
		return out, nil
	}
	op, err := u.guard.OperatorSnapshot(ctx)
	if err != nil {
		return out, err
	}
	operator, err := json.Marshal(op)
	if err != nil {
		return out, err
	}
	out.Outcome, out.Version = "accepted", foundation.NewID()
	if id == "" {
		_, err = u.tx.Exec(ctx, `INSERT INTO compensation.grade_rate(instance_id,enterprise_id,id,grade_id,currency,frequency,unit,places,head) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, s.Instance, state.Enterprise, state.ID, state.Key.GradeID, state.Key.Currency, state.Key.Frequency, state.Key.Unit, state.Places, out.Version)
		var pg *pgconn.PgError
		if errors.As(err, &pg) && pg.Code == "23505" {
			return c.Receipt{}, foundation.Fail(409, "M09_DUPLICATE_RATE", "此职级及计量口径已有标准，请选用已有标准")
		}
		if err != nil {
			return out, err
		}
	}
	indices := make(map[string]int, len(state.Records))
	for i, record := range state.Records {
		indices[record.ID] = i
	}
	for i, r := range change.Records {
		old, exists := indices[r.ID]
		if !exists || state.Records[old].Revision != r.Revision {
			raw, e := json.Marshal(r)
			if e != nil {
				return out, e
			}
			if _, e = u.tx.Exec(ctx, `INSERT INTO compensation.grade_rate_revision(instance_id,rate_id,record_id,revision,fact) VALUES($1,$2,$3,$4,$5)`, s.Instance, state.ID, r.ID, r.Revision, raw); e != nil {
				return out, e
			}
		}
		p := change.Records.Period(i)
		if exists && state.Records[old].Revision == r.Revision && state.Records.Period(old) == p {
			continue
		}
		_, err = u.tx.Exec(ctx, `INSERT INTO compensation.grade_rate_period(instance_id,rate_id,record_id,revision,start_date,until_date,active) VALUES($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT(instance_id,rate_id,record_id) DO UPDATE SET revision=excluded.revision,start_date=excluded.start_date,until_date=excluded.until_date,active=excluded.active`, s.Instance, state.ID, r.ID, r.Revision, r.Start, optional(p.Until), r.Active)
		if err != nil {
			return out, err
		}
	}
	if in.Action == "DELETE" {
		if _, err = u.tx.Exec(ctx, "DELETE FROM compensation.grade_rate_period WHERE instance_id=$1 AND rate_id=$2 AND record_id=$3", s.Instance, state.ID, change.Target.ID); err != nil {
			return out, err
		}
	}
	var previousRecord, previousRevision any
	if change.Previous != nil {
		previousRecord, previousRevision = change.Previous.ID, change.Previous.Revision
	}
	_, err = u.tx.Exec(ctx, `INSERT INTO compensation.grade_rate_version(instance_id,rate_id,id,predecessor,action,record_id,revision,previous_record_id,previous_revision,until_date,previous_until,operator) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, s.Instance, state.ID, out.Version, optional(state.Version), in.Action, change.Target.ID, change.Target.Revision, previousRecord, previousRevision, optional(change.Until), optional(change.PreviousUntil), operator)
	if err != nil {
		return out, err
	}
	if id != "" {
		if _, err = u.tx.Exec(ctx, "UPDATE compensation.grade_rate SET head=$3 WHERE instance_id=$1 AND id=$2", s.Instance, state.ID, out.Version); err != nil {
			return out, err
		}
	}
	_, err = u.tx.Exec(ctx, `INSERT INTO compensation.grade_rate_outbox VALUES($1,$2,$3,jsonb_build_object('type','GradeRateChanged','sourceVersion',$3::uuid,'recordId',$4::uuid))`, s.Instance, state.ID, out.Version, change.Target.ID)
	return out, err
}

// Resolve serializes with the original write. Closing an absent submission is
// safe only while holding that same subject and submission lock.
func (s *Store) ResolveRate(ctx context.Context, cred foundation.Credentials, id string) (out c.Receipt, err error) {
	if !foundation.ValidID(id) {
		return out, unavailable()
	}
	err = s.within(ctx, cred, true, func(ctx context.Context, u *unit) error {
		if _, e := u.visibility(ctx, false); e != nil {
			return e
		}
		if e := u.lock(ctx, "compensation.receipt/"+s.Instance+"/"+u.guard.Principal().ID+"/"+id); e != nil {
			return e
		}
		r, _, e := s.receipt(ctx, u, id)
		if errors.Is(e, pgx.ErrNoRows) {
			out = c.Receipt{SubmissionID: id, Outcome: "not_applied"}
			raw, e := json.Marshal(storedReceipt{Receipt: out})
			if e != nil {
				return e
			}
			_, e = u.tx.Exec(ctx, "INSERT INTO compensation.grade_rate_receipt(instance_id,subject_id,submission_id,result) VALUES($1,$2,$3,$4)", s.Instance, u.guard.Principal().ID, id, raw)
			return e
		}
		if e != nil {
			return e
		}
		out = r.Receipt
		if r.Failure != nil {
			if _, e = u.rateWriteGrant(ctx, r.Action, r.Fields); e != nil {
				return e
			}
			out.Message = r.Failure.Message
		}
		return nil
	})
	return
}

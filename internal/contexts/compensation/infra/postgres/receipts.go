package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"

	d "chnhcm.local/chnhcm/internal/contexts/compensation/domain"
	c "chnhcm.local/chnhcm/internal/contexts/compensation/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
	"github.com/jackc/pgx/v5"
)

type commandOwner struct{ table, prefix, permission string }

var rateCommands = commandOwner{"compensation.grade_rate_receipt", "compensation.receipt/", "graderate"}
var planCommands = commandOwner{"compensation.plan_receipt", "compensation.plan.receipt/", "compplan"}

type storedReceipt struct {
	c.Receipt
	Failure *foundation.Fault `json:"failure,omitempty"`
	Status  int               `json:"httpStatus,omitempty"`
	Action  string            `json:"action"`
	Fields  []string          `json:"fields,omitempty"`
}

func (s *Store) receipt(ctx context.Context, u *unit, entity commandOwner, id string) (r storedReceipt, digest []byte, err error) {
	var raw []byte
	err = u.tx.QueryRow(ctx, "SELECT digest,result FROM "+entity.table+" WHERE instance_id=$1 AND subject_id=$2 AND submission_id=$3", s.Instance, u.guard.Principal().ID, id).Scan(&digest, &raw)
	if err != nil {
		return
	}
	err = json.Unmarshal(raw, &r)
	if r.Failure != nil {
		r.Failure.Status = r.Status
	}
	return
}
func receiptResult(ctx context.Context, u *unit, entity commandOwner, r storedReceipt) (c.Receipt, error) {
	if _, err := u.requireObjectRead(ctx, entity); err != nil {
		return c.Receipt{}, err
	}
	if r.Failure != nil {
		if _, err := u.objectWriteGrant(ctx, entity, r.Action, r.Fields); err != nil {
			return c.Receipt{}, err
		}
		return c.Receipt{}, r.Failure
	}
	return r.Receipt, nil
}
func (s *Store) lookupReceipt(ctx context.Context, cred foundation.Credentials, entity commandOwner, id string, resolve bool) (out c.Receipt, err error) {
	if !foundation.ValidID(id) {
		return out, unavailable()
	}
	err = s.within(ctx, cred, resolve, func(ctx context.Context, u *unit) error {
		if _, e := u.requireObjectRead(ctx, entity); e != nil {
			return e
		}
		if resolve {
			if e := u.lock(ctx, entity.prefix+s.Instance+"/"+u.guard.Principal().ID+"/"+id); e != nil {
				return e
			}
		}
		r, _, e := s.receipt(ctx, u, entity, id)
		if errors.Is(e, pgx.ErrNoRows) {
			if !resolve {
				return unavailable()
			}
			out = c.Receipt{SubmissionID: id, Outcome: "not_applied"}
			raw, e := json.Marshal(storedReceipt{Receipt: out})
			if e != nil {
				return e
			}
			_, e = u.tx.Exec(ctx, "INSERT INTO "+entity.table+"(instance_id,subject_id,submission_id,result) VALUES($1,$2,$3,$4)", s.Instance, u.guard.Principal().ID, id, raw)
			return e
		}
		if e != nil {
			return e
		}
		if resolve && r.Failure != nil {
			if _, e := u.objectWriteGrant(ctx, entity, r.Action, r.Fields); e != nil {
				return e
			}
			out = r.Receipt
			out.Message = r.Failure.Message
			return nil
		}
		out, e = receiptResult(ctx, u, entity, r)
		return e
	})
	return
}

func (s *Store) RateReceipt(ctx context.Context, cred foundation.Credentials, id string) (c.Receipt, error) {
	return s.lookupReceipt(ctx, cred, rateCommands, id, false)
}
func (s *Store) ResolveRate(ctx context.Context, cred foundation.Credentials, id string) (c.Receipt, error) {
	return s.lookupReceipt(ctx, cred, rateCommands, id, true)
}
func (s *Store) PlanReceipt(ctx context.Context, cred foundation.Credentials, id string) (c.Receipt, error) {
	return s.lookupReceipt(ctx, cred, planCommands, id, false)
}
func (s *Store) ResolvePlan(ctx context.Context, cred foundation.Credentials, id string) (c.Receipt, error) {
	return s.lookupReceipt(ctx, cred, planCommands, id, true)
}

func saveCommand[T any](ctx context.Context, s *Store, cred foundation.Credentials, entity commandOwner, id, submission, action string, fields []string, input T, apply func(context.Context, *unit, string, T, []string) (c.Receipt, error)) (out c.Receipt, err error) {
	if !foundation.ValidID(submission) || id != "" && !foundation.ValidID(id) || (id == "") != (action == "CREATE") {
		return out, d.Invalid("submissionId", "提交或目标身份无效")
	}
	payload, err := json.Marshal(struct {
		ID    string
		Input T
	}{id, input})
	if err != nil {
		return out, err
	}
	digest := sha256.Sum256(payload)
	var outcomeErr error
	err = s.within(ctx, cred, true, func(ctx context.Context, u *unit) error {
		if _, e := u.requireObjectRead(ctx, entity); e != nil {
			return e
		}
		if e := u.lock(ctx, entity.prefix+s.Instance+"/"+u.guard.Principal().ID+"/"+submission); e != nil {
			return e
		}
		saved, previous, e := s.receipt(ctx, u, entity, submission)
		if e == nil {
			if saved.Outcome == "not_applied" {
				return foundation.Fail(409, "M09_SUBMISSION_CLOSED", "本次提交已确认未执行，请重新办理")
			}
			if !bytes.Equal(previous, digest[:]) {
				return foundation.Fail(409, "M09_SUBMISSION_CONFLICT", "此提交号已用于其他内容")
			}
			out, outcomeErr = receiptResult(ctx, u, entity, saved)
			return nil
		}
		if !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		if _, e = u.objectWriteGrant(ctx, entity, action, fields); e != nil {
			return e
		}
		child, e := u.tx.Begin(ctx)
		if e != nil {
			return e
		}
		work := *u
		work.tx = child
		out, e = apply(dbtx.With(ctx, child), &work, id, input, fields)
		out.SubmissionID = submission
		r := storedReceipt{Receipt: out, Action: action, Fields: fields}
		if e != nil {
			if rollbackErr := child.Rollback(ctx); rollbackErr != nil {
				return rollbackErr
			}
			f, ok := fault(e)
			if !ok || f.Status >= 500 || f.Status == 403 || f.Status == 404 {
				return e
			}
			r.Failure, r.Status, r.Outcome = f, f.Status, "rejected"
			outcomeErr = e
		} else if e = child.Commit(ctx); e != nil {
			return e
		}
		raw, e := json.Marshal(r)
		if e != nil {
			return e
		}
		_, e = u.tx.Exec(ctx, "INSERT INTO "+entity.table+"(instance_id,subject_id,submission_id,digest,result) VALUES($1,$2,$3,$4,$5)", s.Instance, u.guard.Principal().ID, submission, digest[:], raw)
		return e
	})
	if err == nil {
		err = outcomeErr
	}
	return
}

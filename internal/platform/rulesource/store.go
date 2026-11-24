// Package rulesource stores small, immutable sets of dated configuration records.
// The owner supplies the statements, validates records and serializes writers.
package rulesource

import (
	"bytes"
	f "chnhcm.local/chnhcm/internal/foundation"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"time"
)

// Statements belong to the context that owns the tables and their migration.
type Statements struct {
	Head, SourceExists, Members, History                  string
	Replay, Terminal, InsertSource, InsertMember, Publish string
}

type Store[T any] struct {
	SQL             Statements
	Tx              pgx.Tx
	Instance, Owner string
}

func (s Store[T]) Head(ctx context.Context) (string, error) {
	var version string
	err := s.Tx.QueryRow(ctx, s.SQL.Head, s.Instance, s.Owner).Scan(&version)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return version, err
}
func (s Store[T]) Load(ctx context.Context, version string) ([]T, error) {
	if version == "" {
		return []T{}, nil
	}
	var exists bool
	if err := s.Tx.QueryRow(ctx, s.SQL.SourceExists, s.Instance, s.Owner, version).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, f.Fail(404, "SOURCE_UNAVAILABLE", "所选版本不存在或不可读")
	}
	rows, err := s.Tx.Query(ctx, s.SQL.Members, s.Instance, s.Owner, version)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (v T, err error) { err = row.Scan(&v); return })
}
func (s Store[T]) History(ctx context.Context) ([]f.DatedSource, error) {
	rows, err := s.Tx.Query(ctx, s.SQL.History, s.Instance, s.Owner)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (v f.DatedSource, err error) {
		var at time.Time
		err = row.Scan(&v.Version, &at)
		v.RecordedAt = at.UTC().Format(time.RFC3339Nano)
		return
	})
}

// lookup serializes a late save and recovery before looking at either source
// kind. A submission belongs to its subject across all owners of the rule.
func (s Store[T]) lookup(ctx context.Context, subject, id string, authorizeOwner func(string) error) (f.RuleReceipt, []byte, error) {
	r := f.RuleReceipt{SubmissionID: id}
	if !f.ValidID(id) {
		return r, nil, f.Fail(400, "INVALID_INPUT", "提交号无效")
	}
	_, err := s.Tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,0))", "rule-submission:"+s.Instance+":"+subject+":"+id)
	if err != nil {
		return r, nil, err
	}
	var owner, code, message string
	var digest []byte
	var status int
	err = s.Tx.QueryRow(ctx, s.SQL.Replay, s.Instance, subject, id).Scan(&owner, &r.Outcome, &r.Version, &digest, &status, &code, &message)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, nil, nil
	}
	if err != nil {
		return r, nil, err
	}
	if owner != "" && s.Owner != "" && owner != s.Owner {
		return r, nil, f.Fail(409, "SUBMISSION_CONFLICT", "提交号与当前规则对象不符")
	}
	if owner != "" && authorizeOwner != nil {
		if err = authorizeOwner(owner); err != nil {
			return f.RuleReceipt{}, nil, err
		}
	}
	if status != 0 {
		r.Failure = &f.Fault{Status: status, Code: code, Message: message}
	}
	return r, digest, nil
}
func (s Store[T]) terminal(ctx context.Context, subject string, r f.RuleReceipt, digest []byte) (f.RuleReceipt, error) {
	var status, code, message any
	if r.Failure != nil {
		status = r.Failure.Status
		code = r.Failure.Code
		message = r.Failure.Message
	}
	_, err := s.Tx.Exec(ctx, s.SQL.Terminal, s.Instance, subject, r.SubmissionID, s.Owner, r.Outcome, r.Version, digest, status, code, message)
	return r, err
}
func (s Store[T]) Resolve(ctx context.Context, subject, id string, authorizeOwner func(string) error) (f.RuleReceipt, error) {
	r, _, err := s.lookup(ctx, subject, id, authorizeOwner)
	if err != nil || r.Outcome != "" {
		return r, err
	}
	r.Outcome = "not_applied"
	return s.terminal(ctx, subject, r, nil)
}
func (s Store[T]) Save(ctx context.Context, subject, submission string, input f.RuleChange[T], prepare func([]T) ([]T, bool, error), key func(T) (string, string, f.Period)) (f.RuleReceipt, error) {
	r, stored, err := s.lookup(ctx, subject, submission, nil)
	if err != nil {
		return r, err
	}
	raw, err := json.Marshal(input)
	if err != nil {
		return r, err
	}
	digest := sha256.Sum256(raw)
	if r.Outcome == "not_applied" {
		return r, f.Fail(409, "RULE_SUBMISSION_CLOSED", "原提交已确认未生效，请重新提交")
	}
	if r.Outcome != "" {
		if !bytes.Equal(stored, digest[:]) {
			return r, f.Fail(409, "SUBMISSION_CONFLICT", "该提交号已用于另一项修改")
		}
		return r, nil
	}
	head, err := s.Head(ctx)
	if err != nil {
		return r, err
	}
	var rows []T
	var changed bool
	if head != input.Expected {
		err = f.Fail(409, "SOURCE_CHANGED", "规则已更新，请重新核对当前资料")
	} else {
		rows, err = s.Load(ctx, head)
		if err != nil {
			return r, err
		}
		rows, changed, err = prepare(rows)
	}
	if err != nil {
		var fault *f.Fault
		if !errors.As(err, &fault) || (fault.Status != 400 && fault.Status != 409) {
			return r, err
		}
		r.Outcome, r.Failure = "rejected", fault
		return s.terminal(ctx, subject, r, digest[:])
	}
	if !changed {
		r.Outcome, r.Version = "no_change", head
		return s.terminal(ctx, subject, r, digest[:])
	}
	r.Outcome, r.Version = "accepted", f.NewID()
	_, err = s.Tx.Exec(ctx, s.SQL.InsertSource, s.Instance, s.Owner, r.Version, subject, submission, digest[:])
	if err != nil {
		return r, err
	}
	batch := &pgx.Batch{}
	for _, row := range rows {
		id, rev, span := key(row)
		batch.Queue(s.SQL.InsertMember, s.Instance, s.Owner, r.Version, id, rev, span.From, span.Until, row)
	}
	if err = s.Tx.SendBatch(ctx, batch).Close(); err != nil {
		return r, err
	}
	_, err = s.Tx.Exec(ctx, s.SQL.Publish, s.Instance, s.Owner, r.Version)
	return r, err
}

func (s Store[T]) View(ctx context.Context, name, version string, history, write bool) (v f.DatedRules[T], err error) {
	v = f.DatedRules[T]{Owner: s.Owner, Name: name, Version: version, CanWrite: write && version == "", History: []f.DatedSource{}}
	if version == "" {
		v.Version, err = s.Head(ctx)
	} else if !f.ValidID(version) {
		return v, f.Fail(400, "INVALID_INPUT", "来源版本无效")
	}
	if err == nil {
		v.Rows, err = s.Load(ctx, v.Version)
	}
	if err == nil && history {
		v.History, err = s.History(ctx)
	}
	return
}

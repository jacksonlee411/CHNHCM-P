package postgres

import (
	"bytes"
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/enterprise"
	"chnhcm.local/chnhcm/internal/contexts/organization/application/submission"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history/model"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/compact"
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"slices"
	"time"
)

type enterpriseUnit struct{ *unit }

func (s *Store) WithinEnterprise(ctx context.Context, c access.Credentials, fn func(app.Unit) error) error {
	return s.withUnit(ctx, c, false, func(u *unit) error { return fn(&enterpriseUnit{u}) })
}
func (s *Store) WithinEnterpriseWrite(ctx context.Context, c access.Credentials, fn func(app.Unit) error) error {
	return s.withUnit(ctx, c, true, func(u *unit) error { return fn(&enterpriseUnit{u}) })
}
func (u *enterpriseUnit) LockSubmission(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	key, e := (submission.Identity{Kind: submission.Enterprise, InstanceID: u.instance, SubjectID: u.guard.Principal().ID, SubmissionID: id}).LockKey()
	if e != nil {
		return e
	}
	_, e = u.tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", key)
	return e
}
func (u *enterpriseUnit) Lock(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	var actual string
	if e := u.tx.QueryRow(ctx, `SELECT enterprise_id::text FROM organization.enterprise WHERE instance_id=$1 AND enterprise_id=$2 FOR UPDATE`, u.instance, id).Scan(&actual); e != nil {
		if errors.Is(e, pgx.ErrNoRows) {
			return foundation.Fail(404, "M02_OBJECT_NOT_FOUND", "企业不存在或当前不可见")
		}
		return dependency(e)
	}
	return nil
}
func (u *enterpriseUnit) Load(ctx context.Context, id, source string) (app.State, error) {
	if source == "" {
		states, e := u.enterpriseCurrent(ctx, []string{id})
		if e != nil {
			return app.State{}, e
		}
		if len(states) == 0 {
			return app.State{}, nil
		}
		return states[0], nil
	}
	number, e := history.Locate(ctx, u.tx, history.Enterprise, u.instance, id, source)
	if errors.Is(e, pgx.ErrNoRows) {
		return app.State{}, nil
	}
	if e != nil {
		return app.State{}, e
	}
	chunk, e := history.LoadChunk(ctx, u.tx, history.Enterprise, u.instance, id, number)
	if e != nil {
		return app.State{}, e
	}
	slot := chunk.Index(source)
	if slot < 0 {
		return app.State{}, compact.ErrCorrupt
	}
	b, e := history.LoadShape(ctx, u.tx, history.Enterprise, u.instance, u.instance, chunk.States[slot])
	if e != nil {
		return app.State{}, e
	}
	v, e := history.DecodeEnterpriseState(ctx, chunk.Values, b)
	if e != nil {
		return app.State{}, e
	}
	if v.State.Timeline.ID != id || v.State.Timeline.Source != source || v.State.Timeline.SourceNo != chunk.First+int64(slot) {
		return app.State{}, compact.ErrCorrupt
	}
	return v.State, validateEnterpriseState(v.State)
}
func validateEnterpriseState(v app.State) error {
	t := v.Timeline
	if !foundation.ValidID(t.ID) || !foundation.ValidID(t.First) || !foundation.ValidID(t.Source) || t.SourceNo < 1 || len(t.Records) == 0 || t.Records[0].ID != t.First {
		return compact.ErrCorrupt
	}
	seen := map[string]bool{}
	summaries := map[string]org.OperationSummary{}
	for _, s := range v.Summaries {
		if summaries[s.RecordID].RecordID != "" || !foundation.ValidID(s.OperationSourceVersion) {
			return compact.ErrCorrupt
		}
		summaries[s.RecordID] = s
	}
	for i, r := range t.Records {
		if !foundation.ValidID(r.ID) || !foundation.ValidID(r.Revision) || seen[r.ID] || !foundation.ValidDate(r.Date) || r.RevisionNo < 1 {
			return compact.ErrCorrupt
		}
		seen[r.ID] = true
		end := ""
		if i+1 < len(t.Records) {
			end = t.Records[i+1].Date
			if end <= r.Date {
				return compact.ErrCorrupt
			}
		}
		if r.End != end {
			return compact.ErrCorrupt
		}
		if summaries[r.ID].Revision != r.Revision {
			return compact.ErrCorrupt
		}
	}
	return nil
}
func (u *enterpriseUnit) enterpriseCurrent(ctx context.Context, ids []string) ([]app.State, error) {
	rows, e := u.tx.Query(ctx, `SELECT b.enterprise_id::text,b.catalog_id::text,b.code,b.first_record_id::text,b.head_source_id::text,b.head_no,b.head_chunk,h.dictionary_id,b.current_shape,organization.project_ids(h.ids,b.current_refs)::text[],b.current_envelope,b.initialization_key FROM organization.enterprise_profile b JOIN organization.enterprise_history h ON (h.instance_id,h.object_id,h.chunk_no)=(b.instance_id,b.enterprise_id,b.head_chunk) WHERE b.instance_id=$1 AND ($2 OR b.enterprise_id=ANY($3::text[]::uuid[])) ORDER BY b.enterprise_id`, u.instance, ids == nil, ids)
	if e != nil {
		return nil, e
	}
	projections := []projectionRow{}
	sourceKeys := map[string]string{}
	dicts, shapes := []int32{}, []int32{}
	for rows.Next() {
		var r projectionRow
		var sourceKey string
		if e = rows.Scan(&r.head.ID, &r.head.Enterprise, &r.head.Code, &r.head.First, &r.head.Source, &r.head.Number, &r.head.Chunk, &r.dict, &r.shape, &r.ids, &r.envelope, &sourceKey); e != nil {
			rows.Close()
			return nil, e
		}
		sourceKeys[r.head.ID] = sourceKey
		projections = append(projections, r)
		dicts = append(dicts, r.dict)
		shapes = append(shapes, r.shape)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	if e = u.catalog.Prepare(ctx, u.tx, history.Enterprise, u.instance, u.instance, dicts, shapes); e != nil {
		return nil, e
	}
	out := make([]app.State, 0, len(projections))
	for _, r := range projections {
		m, e := u.catalog.Get(history.Enterprise, u.instance, u.instance, r.dict, r.shape)
		if e != nil {
			return nil, e
		}
		seal, e := history.ProjectionDigest(history.Enterprise, u.instance, u.instance, r.head.ID, r.head.Source, r.ids, m.DictionaryDigest, m.ShapeDigest)
		if e != nil || !bytes.Equal(seal, r.envelope) {
			return nil, compact.ErrCorrupt
		}
		m.Values.IDs = r.ids
		state, e := history.DecodeEnterpriseState(ctx, m.Values, m.Data)
		if e != nil {
			return nil, e
		}
		t := state.State.Timeline
		if r.head.Enterprise != u.instance || t.ID != r.head.ID || t.First != r.head.First || t.Source != r.head.Source || t.SourceNo != r.head.Number || t.Code != r.head.Code || t.InitializationDigest != sourceKeys[t.ID] {
			return nil, compact.ErrCorrupt
		}
		if e = validateEnterpriseState(state.State); e != nil {
			return nil, e
		}
		out = append(out, state.State)
	}
	return out, nil
}
func (u *enterpriseUnit) Persist(ctx context.Context, w app.Write) (org.EnterpriseResult, error) {
	if !w.Validated() || w.Before.ID != w.After.Timeline.ID || w.After.Timeline.SourceNo != w.Before.SourceNo+1 {
		return org.EnterpriseResult{}, fmt.Errorf("unvalidated enterprise write")
	}
	t, op := w.After.Timeline, w.Operation
	last := int32(0)
	if w.Before.Source == "" {
		_, e := u.tx.Exec(ctx, `INSERT INTO organization.enterprise_profile(instance_id,catalog_id,enterprise_id,code,normalized_code,initialization_key,first_record_id,head_source_id,head_no,head_chunk) VALUES($1,$1,$2,$3,lower($3),$4,$5,$6,1,1)`, u.instance, t.ID, t.Code, t.InitializationDigest, t.First, t.Source)
		if e != nil {
			var pg *pgconn.PgError
			if errors.As(e, &pg) && pg.Code == "23505" {
				return org.EnterpriseResult{}, foundation.Fail(409, "M02_CODE_CONFLICT", "来源代码已存在")
			}
			return org.EnterpriseResult{}, e
		}
	} else {
		if e := u.tx.QueryRow(ctx, `UPDATE organization.enterprise_profile SET head_source_id=$4,head_no=$5 WHERE instance_id=$1 AND enterprise_id=$2 AND head_source_id=$3 RETURNING head_chunk`, u.instance, t.ID, w.Before.Source, t.Source, t.SourceNo).Scan(&last); e != nil {
			return org.EnterpriseResult{}, e
		}
	}
	previous := history.Chunk{Kind: history.Enterprise, Instance: u.instance, Enterprise: u.instance, Object: t.ID, Number: 1, First: 1}
	if last != 0 {
		var e error
		previous, e = history.LoadChunk(ctx, u.tx, history.Enterprise, u.instance, t.ID, last)
		if e != nil {
			return org.EnterpriseResult{}, e
		}
	}
	chunk, e := history.Append(ctx, u.tx, previous, t.SourceNo, t.Source, op.Subject, op.Submission, func(p *compact.Pool) (state, operation, evidence []byte, err error) {
		state, err = history.EncodeEnterpriseState(ctx, p, model.EnterpriseState{State: w.After})
		if err != nil {
			return
		}
		operation, err = history.EncodeEnterpriseOperation(ctx, p, model.EnterpriseOperation{Operation: op})
		if err != nil {
			return
		}
		evidence, err = history.EncodeEnterpriseEvidence(ctx, p, model.EnterpriseEvidence{InitializationDigest: t.InitializationDigest})
		return
	})
	if e != nil {
		return org.EnterpriseResult{}, e
	}
	if _, e = u.tx.Exec(ctx, `UPDATE organization.enterprise_profile SET head_chunk=$3 WHERE instance_id=$1 AND enterprise_id=$2`, u.instance, t.ID, chunk.Number); e != nil {
		return org.EnterpriseResult{}, e
	}
	e = history.SaveProjection(ctx, u.tx, chunk, t.Source, func(p *compact.Pool) ([]byte, error) {
		return history.EncodeEnterpriseState(ctx, p, model.EnterpriseState{State: w.After})
	})
	return op.Terminal.Result, e
}
func (u *enterpriseUnit) StoreTerminal(ctx context.Context, t app.Terminal) error {
	return history.SaveTerminal(ctx, u.tx, history.Enterprise, u.instance, u.instance, u.guard.Principal().ID, t.ID, history.TerminalEnvelope{Object: t.ObjectID, Outcome: t.Outcome}, func(p *compact.Pool) ([]byte, error) {
		return history.EncodeEnterpriseTerminal(ctx, p, model.EnterpriseTerminal{Terminal: t})
	})
}
func (u *enterpriseUnit) operation(ctx context.Context, chunk history.Chunk, slot int) (app.Operation, error) {
	if slot < 0 || slot >= chunk.Count() {
		return app.Operation{}, compact.ErrCorrupt
	}
	b, e := history.LoadShape(ctx, u.tx, history.Enterprise, u.instance, u.instance, chunk.Operations[slot])
	if e != nil {
		return app.Operation{}, e
	}
	v, e := history.DecodeEnterpriseOperation(ctx, chunk.Values, b)
	if e != nil {
		return app.Operation{}, e
	}
	op := v.Operation
	if op.ID != chunk.ID(chunk.Sources[slot]) || op.Subject != chunk.ID(chunk.Subjects[slot]) || op.Submission != chunk.ID(chunk.Submissions[slot]) || op.Number != chunk.First+int64(slot) || op.Terminal.ID != op.Submission || op.Terminal.ObjectID != chunk.Object {
		return app.Operation{}, compact.ErrCorrupt
	}
	return op, nil
}
func (u *enterpriseUnit) FindResult(ctx context.Context, id string) (*app.Saved, error) {
	subject := u.guard.Principal().ID
	meta, e := history.TerminalIdentity(ctx, u.tx, history.Enterprise, u.instance, subject, id)
	if e == nil {
		out := &app.Saved{RequestObjectID: meta.Object, Result: org.EnterpriseResult{SubmissionID: id, Outcome: meta.Outcome}}
		values, b, e := history.LoadTerminal(ctx, u.tx, history.Enterprise, u.instance, subject, id)
		if e != nil {
			out.DecodeError = e
			return out, nil
		}
		v, e := history.DecodeEnterpriseTerminal(ctx, values, b)
		if e != nil {
			out.DecodeError = e
			return out, nil
		}
		t := v.Terminal
		if t.ID != id || t.ObjectID != meta.Object || t.Outcome != meta.Outcome {
			out.DecodeError = compact.ErrCorrupt
			return out, nil
		}
		out.Digest = t.Digest
		out.Result = t.Result
		return out, nil
	}
	if !errors.Is(e, pgx.ErrNoRows) {
		return nil, e
	}
	found, e := history.FindSubmission(ctx, u.tx, history.Enterprise, u.instance, subject, id)
	if e != nil || found == nil {
		return nil, e
	}
	out := &app.Saved{RequestObjectID: found.Object, Result: org.EnterpriseResult{SubmissionID: id, Outcome: "accepted", EnterpriseID: &found.Object}}
	chunk, e := history.LoadChunk(ctx, u.tx, history.Enterprise, u.instance, found.Object, found.Chunk)
	if e != nil {
		out.DecodeError = e
		return out, nil
	}
	op, e := u.operation(ctx, chunk, found.Slot)
	if e != nil {
		out.DecodeError = e
		return out, nil
	}
	out.Digest = op.Terminal.Digest
	out.Result = op.Terminal.Result
	return out, nil
}
func (u *enterpriseUnit) History(ctx context.Context, id, through string) ([]app.Operation, error) {
	state, e := u.Load(ctx, id, through)
	if e != nil {
		return nil, e
	}
	if state.Timeline.ID == "" {
		return nil, pgx.ErrNoRows
	}
	rows, e := u.tx.Query(ctx, `SELECT chunk_no FROM organization.enterprise_history WHERE instance_id=$1 AND object_id=$2 AND first_no<=$3 ORDER BY chunk_no`, u.instance, id, state.Timeline.SourceNo)
	if e != nil {
		return nil, e
	}
	numbers := []int32{}
	for rows.Next() {
		var n int32
		if e = rows.Scan(&n); e != nil {
			rows.Close()
			return nil, e
		}
		numbers = append(numbers, n)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	out := []app.Operation{}
	for _, n := range numbers {
		chunk, e := history.LoadChunk(ctx, u.tx, history.Enterprise, u.instance, id, n)
		if e != nil {
			return nil, e
		}
		for i := 0; i < chunk.Count() && chunk.First+int64(i) <= state.Timeline.SourceNo; i++ {
			op, e := u.operation(ctx, chunk, i)
			if e != nil {
				return nil, e
			}
			out = append(out, op)
		}
	}
	if int64(len(out)) != state.Timeline.SourceNo {
		return nil, compact.ErrCorrupt
	}
	slices.Reverse(out)
	return out, nil
}

func (u *enterpriseUnit) Identity(ctx context.Context) (string, error) {
	var id string
	e := u.tx.QueryRow(ctx, `SELECT enterprise_id::text FROM organization.enterprise WHERE instance_id=$1`, u.instance).Scan(&id)
	return id, dependency(e)
}

// InitializeEnterpriseMaintenance runs inside the upgrader's initialization transaction.
func InitializeEnterpriseMaintenance(ctx context.Context, tx pgx.Tx, instance string, operator access.OperatorSnapshot) error {
	var exists bool
	if e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT FROM organization.enterprise_profile WHERE instance_id=$1)`, instance).Scan(&exists); e != nil {
		return e
	}
	if exists {
		return nil
	}
	var id, code, name, start, digest string
	var now time.Time
	if e := tx.QueryRow(ctx, `SELECT enterprise_id::text,code,name,lower(valid_range)::text,encode(initialization_digest,'hex'),clock_timestamp() FROM organization.enterprise WHERE instance_id=$1 FOR UPDATE`, instance).Scan(&id, &code, &name, &start, &digest, &now); e != nil {
		return e
	}
	w, e := app.Initialize(ctx, id, code, name, start, digest, now, operator)
	if e != nil {
		return e
	}
	key, e := (submission.Identity{Kind: submission.Enterprise, InstanceID: instance, SubjectID: operator.SubjectID, SubmissionID: w.Operation.Submission}).LockKey()
	if e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, key); e != nil {
		return e
	}
	u := &enterpriseUnit{&unit{tx: tx, instance: instance, catalog: &history.Catalog{}}}
	_, e = u.Persist(ctx, w)
	return e
}

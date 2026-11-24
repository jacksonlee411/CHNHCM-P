package postgres

import (
	"bytes"
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	geo "chnhcm.local/chnhcm/internal/contexts/organization/application/geoarea"
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/location"
	"chnhcm.local/chnhcm/internal/contexts/organization/application/submission"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/location"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history/model"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/compact"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"slices"
	"time"
)

type locationUnit struct {
	*unit
	enterprise  org.Enterprise
	exportHeads map[string]projectionRow
}

func (s *Store) WithinLocation(ctx context.Context, c access.Credentials, fn func(app.Unit) error) error {
	return s.withUnit(ctx, c, false, func(u *unit) error {
		ent, e := u.Enterprise(ctx, false)
		if e != nil {
			return e
		}
		return fn(&locationUnit{unit: u, enterprise: ent})
	})
}
func (s *Store) WithinLocationWrite(ctx context.Context, c access.Credentials, fn func(app.Unit) error) error {
	return s.withUnit(ctx, c, true, func(u *unit) error {
		ent, e := u.Enterprise(ctx, false)
		if e != nil {
			return e
		}
		return fn(&locationUnit{unit: u, enterprise: ent})
	})
}
func (u *locationUnit) LockSubmission(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	key, e := (submission.Identity{Kind: submission.Location, InstanceID: u.instance, SubjectID: u.guard.Principal().ID, SubmissionID: id}).LockKey()
	if e != nil {
		return e
	}
	_, e = u.tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", key)
	return e
}
func (u *locationUnit) Lock(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	// A first-date change of the enterprise and a location write serialize on
	// the same parent row; refresh the floor only after obtaining that lock.
	ent, e := u.Enterprise(ctx, true)
	if e != nil {
		return e
	}
	u.enterprise = ent
	b, _ := json.Marshal([]string{"organization.location.object", u.instance, id})
	h := sha256.Sum256(b)
	if _, e := u.tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", int64(binary.BigEndian.Uint64(h[:8]))); e != nil {
		return e
	}
	_, e = u.tx.Exec(ctx, `SELECT location_id FROM organization.location WHERE instance_id=$1 AND location_id=$2 FOR UPDATE`, u.instance, id)
	return e
}
func (u *locationUnit) Load(ctx context.Context, id, source string) (app.State, error) {
	if source == "" {
		states, e := u.locationCurrent(ctx, []string{id})
		if e != nil {
			return app.State{}, e
		}
		if len(states) == 0 {
			return app.State{}, nil
		}
		return states[0], nil
	}
	number, e := history.Locate(ctx, u.tx, history.Location, u.instance, id, source)
	if errors.Is(e, pgx.ErrNoRows) {
		return app.State{}, nil
	}
	if e != nil {
		return app.State{}, e
	}
	chunk, e := history.LoadChunk(ctx, u.tx, history.Location, u.instance, id, number)
	if e != nil {
		return app.State{}, e
	}
	slot := chunk.Index(source)
	if slot < 0 {
		return app.State{}, compact.ErrCorrupt
	}
	b, e := history.LoadShape(ctx, u.tx, history.Location, u.instance, u.enterprise.ID, chunk.States[slot])
	if e != nil {
		return app.State{}, e
	}
	v, e := history.DecodeLocationState(ctx, chunk.Values, b)
	if e != nil {
		return app.State{}, e
	}
	if v.State.Timeline.ID != id || v.State.Timeline.Source != source || v.State.Timeline.SourceNo != chunk.First+int64(slot) {
		return app.State{}, compact.ErrCorrupt
	}
	if v.State.Timeline.Enterprise != u.enterprise.ID {
		return app.State{}, nil
	}
	return v.State, validateLocationState(v.State)
}
func validateLocationState(v app.State) error {
	t := v.Timeline
	if !foundation.ValidID(t.Enterprise) || !foundation.ValidID(t.ID) || !foundation.ValidID(t.First) || !foundation.ValidID(t.Source) || t.SourceNo < 1 || len(t.Records) == 0 || t.Records[0].ID != t.First {
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
		if d.ValidateAdoption(r.Data) != nil {
			return compact.ErrCorrupt
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
func (u *locationUnit) locationCurrent(ctx context.Context, ids []string) ([]app.State, error) {
	rows, e := u.tx.Query(ctx, `SELECT b.location_id::text,b.enterprise_id::text,b.code,b.first_record_id::text,b.head_source_id::text,b.head_no,b.head_chunk,h.dictionary_id,b.current_shape,organization.project_ids(h.ids,b.current_refs)::text[],b.current_envelope FROM organization.location b JOIN organization.location_history h ON (h.instance_id,h.object_id,h.chunk_no)=(b.instance_id,b.location_id,b.head_chunk) WHERE b.instance_id=$1 AND b.enterprise_id=$4 AND ($2 OR b.location_id=ANY($3::text[]::uuid[])) ORDER BY b.location_id`, u.instance, ids == nil, ids, u.enterprise.ID)
	if e != nil {
		return nil, e
	}
	projections := []projectionRow{}

	for rows.Next() {
		var r projectionRow

		if e = rows.Scan(&r.head.ID, &r.head.Enterprise, &r.head.Code, &r.head.First, &r.head.Source, &r.head.Number, &r.head.Chunk, &r.dict, &r.shape, &r.ids, &r.envelope); e != nil {
			rows.Close()
			return nil, e
		}

		projections = append(projections, r)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	return u.decodeLocationProjections(ctx, projections)
}
func (u *locationUnit) decodeLocationProjections(ctx context.Context, projections []projectionRow) ([]app.State, error) {
	dicts, shapes := []int32{}, []int32{}
	for _, r := range projections {
		dicts = append(dicts, r.dict)
		shapes = append(shapes, r.shape)
	}
	var e error
	if e = u.catalog.Prepare(ctx, u.tx, history.Location, u.instance, u.enterprise.ID, dicts, shapes); e != nil {
		return nil, e
	}
	out := make([]app.State, 0, len(projections))
	for _, r := range projections {
		m, e := u.catalog.Get(history.Location, u.instance, u.enterprise.ID, r.dict, r.shape)
		if e != nil {
			return nil, e
		}
		seal, e := history.ProjectionDigest(history.Location, u.instance, u.enterprise.ID, r.head.ID, r.head.Source, r.ids, m.DictionaryDigest, m.ShapeDigest)
		if e != nil || !bytes.Equal(seal, r.envelope) {
			return nil, compact.ErrCorrupt
		}
		m.Values.IDs = r.ids
		state, e := history.DecodeLocationState(ctx, m.Values, m.Data)
		if e != nil {
			return nil, e
		}
		t := state.State.Timeline
		if r.head.Enterprise != u.enterprise.ID || t.Enterprise != u.enterprise.ID || t.ID != r.head.ID || t.First != r.head.First || t.Source != r.head.Source || t.SourceNo != r.head.Number || t.Code != r.head.Code {
			return nil, compact.ErrCorrupt
		}
		if e = validateLocationState(state.State); e != nil {
			return nil, e
		}
		out = append(out, state.State)
	}
	return out, nil
}
func (u *locationUnit) Persist(ctx context.Context, w app.Write) (org.LocationResult, error) {
	if !w.Validated() || w.Before.ID != w.After.Timeline.ID || w.After.Timeline.SourceNo != w.Before.SourceNo+1 {
		return org.LocationResult{}, fmt.Errorf("unvalidated location write")
	}
	t, op := w.After.Timeline, w.Operation
	if e := protectLegalPurposes(ctx, u.unit, "location", t.ID, w.Before.Records, t.Records); e != nil {
		return org.LocationResult{}, e
	}
	if err := checkPositionUses(ctx, u.unit, t.Enterprise, "locationId", t.ID, w.Before.Records, t.Records); err != nil {
		return org.LocationResult{}, err
	}
	last := int32(0)
	if w.Before.Source == "" {
		_, e := u.tx.Exec(ctx, `INSERT INTO organization.location(instance_id,enterprise_id,location_id,code,normalized_code,first_record_id,head_source_id,head_no,head_chunk) VALUES($1,$4,$2,$3,lower($3),$5,$6,1,1)`, u.instance, t.ID, t.Code, t.Enterprise, t.First, t.Source)
		if e != nil {
			var pg *pgconn.PgError
			if errors.As(e, &pg) && pg.Code == "23505" {
				return org.LocationResult{}, foundation.Fail(409, "M02_CODE_CONFLICT", "来源代码已存在")
			}
			return org.LocationResult{}, e
		}
	} else {
		if e := u.tx.QueryRow(ctx, `UPDATE organization.location SET head_source_id=$4,head_no=$5 WHERE instance_id=$1 AND location_id=$2 AND head_source_id=$3 RETURNING head_chunk`, u.instance, t.ID, w.Before.Source, t.Source, t.SourceNo).Scan(&last); e != nil {
			return org.LocationResult{}, e
		}
	}
	previous := history.Chunk{Kind: history.Location, Instance: u.instance, Enterprise: u.enterprise.ID, Object: t.ID, Number: 1, First: 1}
	if last != 0 {
		var e error
		previous, e = history.LoadChunk(ctx, u.tx, history.Location, u.instance, t.ID, last)
		if e != nil {
			return org.LocationResult{}, e
		}
	}
	chunk, e := history.Append(ctx, u.tx, previous, t.SourceNo, t.Source, op.Subject, op.Submission, func(p *compact.Pool) (state, operation, evidence []byte, err error) {
		state, err = history.EncodeLocationState(ctx, p, model.LocationState{State: w.After})
		if err != nil {
			return
		}
		operation, err = history.EncodeLocationOperation(ctx, p, model.LocationOperation{Operation: op})
		if err != nil {
			return
		}
		evidence, err = history.EncodeLocationEvidence(ctx, p, model.LocationEvidence{ObjectID: t.ID})
		return
	})
	if e != nil {
		return org.LocationResult{}, e
	}
	if _, e = u.tx.Exec(ctx, `UPDATE organization.location SET head_chunk=$3 WHERE instance_id=$1 AND location_id=$2`, u.instance, t.ID, chunk.Number); e != nil {
		return org.LocationResult{}, e
	}
	e = history.SaveProjection(ctx, u.tx, chunk, t.Source, func(p *compact.Pool) ([]byte, error) {
		return history.EncodeLocationState(ctx, p, model.LocationState{State: w.After})
	})
	if e != nil {
		return org.LocationResult{}, e
	}
	e = u.saveRead(ctx, t)
	return op.Terminal.Result, e
}
func (u *locationUnit) StoreTerminal(ctx context.Context, t app.Terminal) error {
	return history.SaveTerminal(ctx, u.tx, history.Location, u.instance, u.enterprise.ID, u.guard.Principal().ID, t.ID, history.TerminalEnvelope{Object: t.ObjectID, Outcome: t.Outcome}, func(p *compact.Pool) ([]byte, error) {
		return history.EncodeLocationTerminal(ctx, p, model.LocationTerminal{Terminal: t})
	})
}
func (u *locationUnit) operation(ctx context.Context, chunk history.Chunk, slot int) (app.Operation, error) {
	if slot < 0 || slot >= chunk.Count() {
		return app.Operation{}, compact.ErrCorrupt
	}
	b, e := history.LoadShape(ctx, u.tx, history.Location, u.instance, u.enterprise.ID, chunk.Operations[slot])
	if e != nil {
		return app.Operation{}, e
	}
	v, e := history.DecodeLocationOperation(ctx, chunk.Values, b)
	if e != nil {
		return app.Operation{}, e
	}
	op := v.Operation
	if op.ID != chunk.ID(chunk.Sources[slot]) || op.Subject != chunk.ID(chunk.Subjects[slot]) || op.Submission != chunk.ID(chunk.Submissions[slot]) || op.Number != chunk.First+int64(slot) || op.Terminal.ID != op.Submission || op.Terminal.ObjectID != chunk.Object {
		return app.Operation{}, compact.ErrCorrupt
	}
	return op, nil
}
func (u *locationUnit) FindResult(ctx context.Context, id string) (*app.Saved, error) {
	subject := u.guard.Principal().ID
	meta, e := history.TerminalIdentity(ctx, u.tx, history.Location, u.instance, subject, id)
	if e == nil {
		out := &app.Saved{RequestObjectID: meta.Object, Result: org.LocationResult{SubmissionID: id, Outcome: meta.Outcome}}
		values, b, e := history.LoadTerminal(ctx, u.tx, history.Location, u.instance, subject, id)
		if e != nil {
			out.DecodeError = e
			return out, nil
		}
		v, e := history.DecodeLocationTerminal(ctx, values, b)
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
	found, e := history.FindSubmission(ctx, u.tx, history.Location, u.instance, subject, id)
	if e != nil || found == nil {
		return nil, e
	}
	out := &app.Saved{RequestObjectID: found.Object, Result: org.LocationResult{SubmissionID: id, Outcome: "accepted", LocationID: &found.Object}}
	chunk, e := history.LoadChunk(ctx, u.tx, history.Location, u.instance, found.Object, found.Chunk)
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
func (u *locationUnit) History(ctx context.Context, id, through string) ([]app.Operation, error) {
	state, e := u.Load(ctx, id, through)
	if e != nil {
		return nil, e
	}
	if state.Timeline.ID == "" {
		return nil, pgx.ErrNoRows
	}
	rows, e := u.tx.Query(ctx, `SELECT chunk_no FROM organization.location_history WHERE instance_id=$1 AND object_id=$2 AND first_no<=$3 ORDER BY chunk_no`, u.instance, id, state.Timeline.SourceNo)
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
		chunk, e := history.LoadChunk(ctx, u.tx, history.Location, u.instance, id, n)
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

func (u *locationUnit) EnterpriseScope() org.Enterprise { return u.enterprise }
func (u *locationUnit) References() geo.ReferenceUnit   { return &geoareaUnit{unit: u.unit} }

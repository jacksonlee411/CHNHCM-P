package postgres

import (
	"bytes"
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/geoarea"
	"chnhcm.local/chnhcm/internal/contexts/organization/application/submission"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/geoarea"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history/model"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/compact"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
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

type geoareaUnit struct {
	*unit
	candidateCursor bool
}

func (s *Store) WithinGeoArea(ctx context.Context, c access.Credentials, fn func(app.Unit) error) error {
	return s.withUnit(ctx, c, false, func(u *unit) error { return fn(&geoareaUnit{unit: u}) })
}
func (s *Store) WithinGeoAreaWrite(ctx context.Context, c access.Credentials, fn func(app.Unit) error) error {
	return s.withUnit(ctx, c, true, func(u *unit) error { return fn(&geoareaUnit{unit: u}) })
}
func (u *geoareaUnit) LockSubmission(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	key, e := (submission.Identity{Kind: submission.GeoArea, InstanceID: u.instance, SubjectID: u.guard.Principal().ID, SubmissionID: id}).LockKey()
	if e != nil {
		return e
	}
	_, e = u.tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", key)
	return e
}
func (u *geoareaUnit) Lock(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	b, _ := json.Marshal([]string{"organization.geoarea.object", u.instance, id})
	h := sha256.Sum256(b)
	if _, e := u.tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", int64(binary.BigEndian.Uint64(h[:8]))); e != nil {
		return e
	}
	_, e := u.tx.Exec(ctx, `SELECT geoarea_id FROM organization.geoarea WHERE instance_id=$1 AND geoarea_id=$2 FOR UPDATE`, u.instance, id)
	return e
}
func (u *geoareaUnit) Load(ctx context.Context, id, source string) (app.State, error) {
	if source == "" {
		states, e := u.geoareaCurrent(ctx, []string{id})
		if e != nil {
			return app.State{}, e
		}
		if len(states) == 0 {
			return app.State{}, nil
		}
		return states[0], nil
	}
	current, e := u.geoareaCurrent(ctx, []string{id})
	if e != nil {
		return app.State{}, e
	}
	if len(current) > 0 && current[0].Timeline.Source == source {
		return current[0], nil
	}
	number, e := history.Locate(ctx, u.tx, history.GeoArea, u.instance, id, source)
	if errors.Is(e, pgx.ErrNoRows) {
		return app.State{}, nil
	}
	if e != nil {
		return app.State{}, e
	}
	chunk, e := history.LoadChunk(ctx, u.tx, history.GeoArea, u.instance, id, number)
	if e != nil {
		return app.State{}, e
	}
	slot := chunk.Index(source)
	if slot < 0 {
		return app.State{}, compact.ErrCorrupt
	}
	b, e := history.LoadShape(ctx, u.tx, history.GeoArea, u.instance, u.instance, chunk.States[slot])
	if e != nil {
		return app.State{}, e
	}
	v, e := history.DecodeGeoAreaState(ctx, chunk.Values, b)
	if e != nil {
		return app.State{}, e
	}
	if v.State.Timeline.ID != id || v.State.Timeline.Source != source || v.State.Timeline.SourceNo != chunk.First+int64(slot) {
		return app.State{}, compact.ErrCorrupt
	}
	return v.State, validateGeoAreaState(v.State)
}
func validateGeoAreaState(v app.State) error {
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
func (u *geoareaUnit) geoareaCurrent(ctx context.Context, ids []string) ([]app.State, error) {
	rows, e := u.tx.Query(ctx, `SELECT b.geoarea_id::text,b.catalog_id::text,b.code,b.first_record_id::text,b.head_source_id::text,b.head_no,b.head_chunk,h.dictionary_id,b.current_shape,organization.project_ids(h.ids,b.current_refs)::text[],b.current_envelope,b.source_key FROM organization.geoarea b JOIN organization.geoarea_history h ON (h.instance_id,h.object_id,h.chunk_no)=(b.instance_id,b.geoarea_id,b.head_chunk) WHERE b.instance_id=$1 AND ($2 OR b.geoarea_id=ANY($3::text[]::uuid[])) ORDER BY b.geoarea_id`, u.instance, ids == nil, ids)
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
	if e = u.catalog.Prepare(ctx, u.tx, history.GeoArea, u.instance, u.instance, dicts, shapes); e != nil {
		return nil, e
	}
	out := make([]app.State, 0, len(projections))
	for _, r := range projections {
		m, e := u.catalog.Get(history.GeoArea, u.instance, u.instance, r.dict, r.shape)
		if e != nil {
			return nil, e
		}
		seal, e := history.ProjectionDigest(history.GeoArea, u.instance, u.instance, r.head.ID, r.head.Source, r.ids, m.DictionaryDigest, m.ShapeDigest)
		if e != nil || !bytes.Equal(seal, r.envelope) {
			return nil, compact.ErrCorrupt
		}
		m.Values.IDs = r.ids
		state, e := history.DecodeGeoAreaState(ctx, m.Values, m.Data)
		if e != nil {
			return nil, e
		}
		t := state.State.Timeline
		if r.head.Enterprise != u.instance || t.ID != r.head.ID || t.First != r.head.First || t.Source != r.head.Source || t.SourceNo != r.head.Number || t.Origin.Code != r.head.Code || t.Origin.Key != sourceKeys[t.ID] {
			return nil, compact.ErrCorrupt
		}
		if e = validateGeoAreaState(state.State); e != nil {
			return nil, e
		}
		out = append(out, state.State)
	}
	return out, nil
}
func (u *geoareaUnit) Persist(ctx context.Context, w app.Write) (org.GeoAreaResult, error) {
	if !w.Validated() || w.Before.ID != w.After.Timeline.ID || w.After.Timeline.SourceNo != w.Before.SourceNo+1 {
		return org.GeoAreaResult{}, fmt.Errorf("unvalidated geoarea write")
	}
	t, op := w.After.Timeline, w.Operation
	last := int32(0)
	if w.Before.Source == "" {
		_, e := u.tx.Exec(ctx, `INSERT INTO organization.geoarea(instance_id,catalog_id,geoarea_id,code,normalized_code,source_key,first_record_id,head_source_id,head_no,head_chunk,country_id,country_code,scheme,nature,level,parent_id,ancestors,initial_date,source_version,package_version,package_digest,archive_digest) VALUES($1,$1,$2,$3,lower($3),$4,$5,$6,1,1,$7,$8,$9,$10,$11,NULLIF($12,'')::uuid,$13::text[]::uuid[],$14,$15,$16,$17,$18)`, u.instance, t.ID, t.Origin.Code, t.Origin.Key, t.First, t.Source, t.Origin.CountryID, t.Origin.CountryCode, t.Origin.Scheme, t.Origin.Nature, t.Origin.Level, t.Origin.ParentID, ancestorIDs(t.Records[0].Data.Adoption), t.Records[0].Date, t.Origin.SourceVersion, t.Origin.PackageVersion, t.Origin.PackageDigest, t.Origin.ArchiveDigest)
		if e != nil {
			var pg *pgconn.PgError
			if errors.As(e, &pg) && pg.Code == "23505" {
				return org.GeoAreaResult{}, foundation.Fail(409, "M02_CODE_CONFLICT", "来源代码已存在")
			}
			return org.GeoAreaResult{}, e
		}
	} else {
		if e := u.tx.QueryRow(ctx, `UPDATE organization.geoarea SET head_source_id=$4,head_no=$5 WHERE instance_id=$1 AND geoarea_id=$2 AND head_source_id=$3 RETURNING head_chunk`, u.instance, t.ID, w.Before.Source, t.Source, t.SourceNo).Scan(&last); e != nil {
			return org.GeoAreaResult{}, e
		}
	}
	previous := history.Chunk{Kind: history.GeoArea, Instance: u.instance, Enterprise: u.instance, Object: t.ID, Number: 1, First: 1}
	if last != 0 {
		var e error
		previous, e = history.LoadChunk(ctx, u.tx, history.GeoArea, u.instance, t.ID, last)
		if e != nil {
			return org.GeoAreaResult{}, e
		}
	}
	chunk, e := history.Append(ctx, u.tx, previous, t.SourceNo, t.Source, op.Subject, op.Submission, func(p *compact.Pool) (state, operation, evidence []byte, err error) {
		state, err = history.EncodeGeoAreaState(ctx, p, model.GeoAreaState{State: w.After})
		if err != nil {
			return
		}
		operation, err = history.EncodeGeoAreaOperation(ctx, p, model.GeoAreaOperation{Operation: op})
		if err != nil {
			return
		}
		evidence, err = history.EncodeGeoAreaEvidence(ctx, p, model.GeoAreaEvidence{PackageDigest: t.Origin.PackageDigest})
		return
	})
	if e != nil {
		return org.GeoAreaResult{}, e
	}
	if _, e = u.tx.Exec(ctx, `UPDATE organization.geoarea SET head_chunk=$3 WHERE instance_id=$1 AND geoarea_id=$2`, u.instance, t.ID, chunk.Number); e != nil {
		return org.GeoAreaResult{}, e
	}
	e = history.SaveProjection(ctx, u.tx, chunk, t.Source, func(p *compact.Pool) ([]byte, error) {
		return history.EncodeGeoAreaState(ctx, p, model.GeoAreaState{State: w.After})
	})
	if e != nil {
		return org.GeoAreaResult{}, e
	}
	if _, e = u.tx.Exec(ctx, `DELETE FROM organization.geoarea_read WHERE instance_id=$1 AND object_id=$2`, u.instance, t.ID); e != nil {
		return org.GeoAreaResult{}, e
	}
	for _, r := range t.Records {
		_, e = u.tx.Exec(ctx, `INSERT INTO organization.geoarea_read(instance_id,object_id,record_id,valid_period,name,name_en,state,effective_date) VALUES($1,$2,$3,daterange($4::date,NULLIF($5,'')::date,'[)'),$6,$7,$8,$4::date)`, u.instance, t.ID, r.ID, r.Date, r.End, r.Data.Name, r.Data.NameEn, r.State)
		if e != nil {
			return org.GeoAreaResult{}, e
		}
	}
	return op.Terminal.Result, nil
}
func (u *geoareaUnit) StoreTerminal(ctx context.Context, t app.Terminal) error {
	return history.SaveTerminal(ctx, u.tx, history.GeoArea, u.instance, u.instance, u.guard.Principal().ID, t.ID, history.TerminalEnvelope{Object: t.ObjectID, Outcome: t.Outcome}, func(p *compact.Pool) ([]byte, error) {
		return history.EncodeGeoAreaTerminal(ctx, p, model.GeoAreaTerminal{Terminal: t})
	})
}
func (u *geoareaUnit) operation(ctx context.Context, chunk history.Chunk, slot int) (app.Operation, error) {
	if slot < 0 || slot >= chunk.Count() {
		return app.Operation{}, compact.ErrCorrupt
	}
	b, e := history.LoadShape(ctx, u.tx, history.GeoArea, u.instance, u.instance, chunk.Operations[slot])
	if e != nil {
		return app.Operation{}, e
	}
	v, e := history.DecodeGeoAreaOperation(ctx, chunk.Values, b)
	if e != nil {
		return app.Operation{}, e
	}
	op := v.Operation
	if op.ID != chunk.ID(chunk.Sources[slot]) || op.Subject != chunk.ID(chunk.Subjects[slot]) || op.Submission != chunk.ID(chunk.Submissions[slot]) || op.Number != chunk.First+int64(slot) || op.Terminal.ID != op.Submission || op.Terminal.ObjectID != chunk.Object {
		return app.Operation{}, compact.ErrCorrupt
	}
	return op, nil
}
func (u *geoareaUnit) FindResult(ctx context.Context, id string) (*app.Saved, error) {
	subject := u.guard.Principal().ID
	meta, e := history.TerminalIdentity(ctx, u.tx, history.GeoArea, u.instance, subject, id)
	if e == nil {
		out := &app.Saved{RequestObjectID: meta.Object, Result: org.GeoAreaResult{SubmissionID: id, Outcome: meta.Outcome}}
		values, b, e := history.LoadTerminal(ctx, u.tx, history.GeoArea, u.instance, subject, id)
		if e != nil {
			out.DecodeError = e
			return out, nil
		}
		v, e := history.DecodeGeoAreaTerminal(ctx, values, b)
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
	found, e := history.FindSubmission(ctx, u.tx, history.GeoArea, u.instance, subject, id)
	if e != nil || found == nil {
		return nil, e
	}
	out := &app.Saved{RequestObjectID: found.Object, Result: org.GeoAreaResult{SubmissionID: id, Outcome: "accepted", GeoAreaID: &found.Object}}
	chunk, e := history.LoadChunk(ctx, u.tx, history.GeoArea, u.instance, found.Object, found.Chunk)
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
func (u *geoareaUnit) History(ctx context.Context, id, through string) ([]app.Operation, error) {
	state, e := u.Load(ctx, id, through)
	if e != nil {
		return nil, e
	}
	if state.Timeline.ID == "" {
		return nil, pgx.ErrNoRows
	}
	rows, e := u.tx.Query(ctx, `SELECT chunk_no FROM organization.geoarea_history WHERE instance_id=$1 AND object_id=$2 AND first_no<=$3 ORDER BY chunk_no`, u.instance, id, state.Timeline.SourceNo)
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
		chunk, e := history.LoadChunk(ctx, u.tx, history.GeoArea, u.instance, id, n)
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

func ancestorIDs(a d.Adoption) []string {
	out := []string{}
	for _, r := range a.Parents {
		out = append(out, r.ObjectID)
	}
	return out
}

// Batch the bounded page/chain projections; a concurrently changed head is
// resolved by the immutable source captured by the request cursor.
func (u *geoareaUnit) LoadSources(ctx context.Context, want map[string]string) (map[string]app.State, error) {
	ids := make([]string, 0, len(want))
	for id := range want {
		ids = append(ids, id)
	}
	out := map[string]app.State{}
	if len(ids) == 0 {
		return out, nil
	}
	states, e := u.geoareaCurrent(ctx, ids)
	if e != nil {
		return nil, e
	}
	for _, v := range states {
		if v.Timeline.Source == want[v.Timeline.ID] {
			out[v.Timeline.ID] = v
		}
	}
	for id, source := range want {
		if _, ok := out[id]; ok {
			continue
		}
		v, e := u.Load(ctx, id, source)
		if e != nil {
			return nil, e
		}
		out[id] = v
	}
	return out, nil
}

func (s *Store) profileCatalogUnit(ctx context.Context, c foundation.Credentials) (*geoareaUnit, error) {
	tx, ok := dbtx.From(ctx)
	if !ok {
		return nil, errors.New("profile catalog adoption requires the caller's transaction")
	}
	g, err := s.Guard(ctx, tx, c)
	if err != nil {
		return nil, err
	}
	return &geoareaUnit{unit: &unit{tx: tx, instance: s.Instance, guard: g, catalog: &s.historyCatalog, clock: s.Clock}}, nil
}

func (s *Store) AdoptProfileArea(ctx context.Context, c foundation.Credentials, id, date string) ([]org.GeoAreaPathNode, error) {
	s.historyCatalog.Enter()
	defer s.historyCatalog.Leave()
	u, err := s.profileCatalogUnit(ctx, c)
	if err != nil {
		return nil, err
	}
	return app.ResolveProfileArea(ctx, u, id, date)
}

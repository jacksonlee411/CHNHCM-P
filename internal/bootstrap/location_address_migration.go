package bootstrap

import (
	"bytes"
	migrationfiles "chnhcm.local/chnhcm/db/migrations"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history/model"
	"chnhcm.local/chnhcm/internal/platform/compact"
	"github.com/pressly/goose/v3"
)

// LocationAddress is an explicit, transactional development-data conversion.
// The API never invokes the prior-format reader. Goose records 16 in
// this same transaction; any corrupt input or failed write rolls everything back.
func locationAddressMigration() *goose.Migration {
	return goose.NewGoMigration(16, &goose.GoFunc{RunTx: mergeLocationAddresses}, &goose.GoFunc{RunTx: func(context.Context, *sql.Tx) error {
		return errors.New("merged location addresses cannot be split automatically")
	}})
}

type addressMigration struct {
	ctx context.Context
	tx  *sql.Tx
}
type addressObject struct {
	Instance, Enterprise, ID, Source string
	Number                           int64
	Chunk                            int32
}

func mergeLocationAddresses(ctx context.Context, tx *sql.Tx) error {
	m := addressMigration{ctx, tx}
	// Lock only the affected tables, and keep trigger changes transaction-local.
	if _, e := tx.ExecContext(ctx, `LOCK TABLE organization.location,organization.location_history,organization.location_read,organization.location_selection_draft,organization.location_selection_block,access.subject_grant,access.policy_guard IN ACCESS EXCLUSIVE MODE;
 ALTER TABLE organization.location_history DISABLE TRIGGER USER;
 ALTER TABLE organization.location DISABLE TRIGGER protect_head;`); e != nil {
		return e
	}
	rows, e := tx.QueryContext(ctx, `SELECT instance_id::text,enterprise_id::text,location_id::text,head_source_id::text,head_no,head_chunk FROM organization.location ORDER BY instance_id,location_id`)
	if e != nil {
		return e
	}
	var objects []addressObject
	for rows.Next() {
		var o addressObject
		if e = rows.Scan(&o.Instance, &o.Enterprise, &o.ID, &o.Source, &o.Number, &o.Chunk); e != nil {
			rows.Close()
			return e
		}
		objects = append(objects, o)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, o := range objects {
		if e = m.object(o); e != nil {
			return fmt.Errorf("location address conversion: %w", e)
		}
	}
	ddl, e := migrationfiles.Files.ReadFile("00016_location_address.sql")
	if e != nil {
		return e
	}
	_, e = tx.ExecContext(ctx, strings.Split(string(ddl), "-- +goose Down")[0])
	return e
}

// JSON is only a database/sql array transport here, never a persisted history format.
const addressValuesSQL = `jsonb_build_object('IDs',h.ids,'Strings',d.words,'Binary',ARRAY(SELECT encode(b,'base64') FROM unnest(d.binaries) b),'Ranges',ARRAY(SELECT jsonb_build_object('Start',coalesce(lower(r)::text,''),'End',coalesce(upper(r)::text,'')) FROM unnest(d.ranges) r))`

func (m addressMigration) load(o addressObject, number int32) (history.Chunk, error) {
	c := history.Chunk{Kind: history.Location, Instance: o.Instance, Enterprise: o.Enterprise, Object: o.ID, Number: number}
	var values, sources, subjects, submissions, states, operations, evidence []byte
	e := m.tx.QueryRowContext(m.ctx, `SELECT h.first_no,h.dictionary_id,to_jsonb(h.source_refs),to_jsonb(h.subject_refs),to_jsonb(h.submission_refs),to_jsonb(h.states),to_jsonb(h.operations),to_jsonb(h.evidence),h.envelope,h.sealed,h.encoded_bytes,`+addressValuesSQL+` FROM organization.location_history h JOIN organization.location_dictionary d ON (d.instance_id,d.enterprise_id,d.id)=(h.instance_id,h.enterprise_id,h.dictionary_id) WHERE h.instance_id=$1 AND h.object_id=$2 AND h.chunk_no=$3`, o.Instance, o.ID, number).Scan(&c.First, &c.Dictionary, &sources, &subjects, &submissions, &states, &operations, &evidence, &c.Envelope, &c.Sealed, &c.EncodedBytes, &values)
	if e != nil {
		return c, e
	}
	for _, pair := range []struct {
		b []byte
		v any
	}{{values, &c.Values}, {sources, &c.Sources}, {subjects, &c.Subjects}, {submissions, &c.Submissions}, {states, &c.States}, {operations, &c.Operations}, {evidence, &c.Evidence}} {
		if e = json.Unmarshal(pair.b, pair.v); e != nil {
			return c, e
		}
	}
	return c, c.Validate()
}
func (m addressMigration) shape(o addressObject, id int32) ([]byte, error) {
	var b []byte
	var valid bool
	e := m.tx.QueryRowContext(m.ctx, `SELECT to_jsonb(ARRAY(SELECT encode(p,'base64') FROM unnest(parts) p)),digest=organization.history_parts_hash(parts) FROM organization.location_shape WHERE instance_id=$1 AND enterprise_id=$2 AND id=$3`, o.Instance, o.Enterprise, id).Scan(&b, &valid)
	if e != nil {
		return nil, e
	}
	if !valid {
		return nil, compact.ErrCorrupt
	}
	var parts [][]byte
	if e = json.Unmarshal(b, &parts); e != nil {
		return nil, e
	}
	if len(parts) == 0 || len(parts) > 256 {
		return nil, compact.ErrCorrupt
	}
	for _, p := range parts {
		if len(p) == 0 || len(p) > compact.MaxPart {
			return nil, compact.ErrCorrupt
		}
	}
	return bytes.Join(parts, nil), nil
}
func (m addressMigration) putShape(o addressObject, b []byte) (int32, error) {
	parts := [][]byte{}
	for len(b) > 0 {
		n := min(len(b), compact.MaxPart)
		parts = append(parts, b[:n])
		b = b[n:]
	}
	var id int32
	e := m.tx.QueryRowContext(m.ctx, `INSERT INTO organization.location_shape(instance_id,enterprise_id,parts) VALUES($1,$2,$3) ON CONFLICT(instance_id,enterprise_id,digest) DO NOTHING RETURNING id`, o.Instance, o.Enterprise, parts).Scan(&id)
	if errors.Is(e, sql.ErrNoRows) {
		e = m.tx.QueryRowContext(m.ctx, `SELECT id FROM organization.location_shape WHERE instance_id=$1 AND enterprise_id=$2 AND digest=organization.history_parts_hash($3)`, o.Instance, o.Enterprise, parts).Scan(&id)
	}
	return id, e
}
func (m addressMigration) putDictionary(o addressObject, v compact.Values) (int32, error) {
	var id int32
	args := []any{o.Instance, o.Enterprise, v.Strings, v.Binary, history.RangesSQL(v)}
	e := m.tx.QueryRowContext(m.ctx, `INSERT INTO organization.location_dictionary(instance_id,enterprise_id,words,binaries,ranges) VALUES($1,$2,$3,$4,$5::text[]::daterange[]) ON CONFLICT(instance_id,enterprise_id,digest) DO NOTHING RETURNING id`, args...).Scan(&id)
	if errors.Is(e, sql.ErrNoRows) {
		e = m.tx.QueryRowContext(m.ctx, `SELECT id FROM organization.location_dictionary WHERE instance_id=$1 AND enterprise_id=$2 AND digest=organization.history_dictionary_hash($3,$4,$5::text[]::daterange[])`, args...).Scan(&id)
	}
	return id, e
}

func (m addressMigration) object(o addressObject) error {
	// Verify the old current projection independently before replacing it.
	var projection, refs []byte
	var shape int32
	var envelope, dh, sh []byte
	e := m.tx.QueryRowContext(m.ctx, `SELECT b.current_shape,to_jsonb(b.current_refs),b.current_envelope,d.digest,s.digest FROM organization.location b JOIN organization.location_history h ON (h.instance_id,h.object_id,h.chunk_no)=(b.instance_id,b.location_id,b.head_chunk) JOIN organization.location_dictionary d ON d.id=h.dictionary_id JOIN organization.location_shape s ON s.id=b.current_shape WHERE b.instance_id=$1 AND b.location_id=$2`, o.Instance, o.ID).Scan(&shape, &refs, &envelope, &dh, &sh)
	if e != nil {
		return e
	}
	last, e := m.load(o, o.Chunk)
	if e != nil {
		return e
	}
	var indices []int32
	if e = json.Unmarshal(refs, &indices); e != nil {
		return e
	}
	ids := []string{}
	for _, n := range indices {
		v := last.ID(n)
		if v == "" {
			return compact.ErrCorrupt
		}
		ids = append(ids, v)
	}
	digest, e := history.ProjectionDigest(history.Location, o.Instance, o.Enterprise, o.ID, o.Source, ids, dh, sh)
	if e != nil || !bytes.Equal(envelope, digest) {
		return compact.ErrCorrupt
	}
	projection, e = m.shape(o, shape)
	if e != nil {
		return e
	}
	values := last.Values
	values.IDs = ids
	head, e := decodePriorLocationState(m.ctx, values, projection)
	if e != nil {
		return e
	}
	if head.State.Timeline.Source != o.Source || head.State.Timeline.SourceNo != o.Number {
		return compact.ErrCorrupt
	}
	rows, e := m.tx.QueryContext(m.ctx, `SELECT chunk_no FROM organization.location_history WHERE instance_id=$1 AND object_id=$2 ORDER BY chunk_no`, o.Instance, o.ID)
	if e != nil {
		return e
	}
	var numbers []int32
	for rows.Next() {
		var n int32
		if e = rows.Scan(&n); e != nil {
			rows.Close()
			return e
		}
		numbers = append(numbers, n)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	var output []history.Chunk
	c := history.Chunk{Kind: history.Location, Instance: o.Instance, Enterprise: o.Enterprise, Object: o.ID, Number: 1, First: 1}
	var final model.LocationState
	for _, n := range numbers {
		old, e := m.load(o, n)
		if e != nil {
			return e
		}
		for slot := 0; slot < old.Count(); slot++ {
			stateBytes, e := m.shape(o, old.States[slot])
			if e != nil {
				return e
			}
			state, e := decodePriorLocationState(m.ctx, old.Values, stateBytes)
			if e != nil {
				return e
			}
			opBytes, e := m.shape(o, old.Operations[slot])
			if e != nil {
				return e
			}
			op, e := decodePriorLocationOperation(m.ctx, old.Values, opBytes)
			if e != nil {
				return e
			}
			evidenceBytes, e := m.shape(o, old.Evidence[slot])
			if e != nil {
				return e
			}
			evidence, e := history.DecodeLocationEvidence(m.ctx, old.Values, evidenceBytes)
			if e != nil {
				return e
			}
			number := old.First + int64(slot)
			source, subject, submission := old.ID(old.Sources[slot]), old.ID(old.Subjects[slot]), old.ID(old.Submissions[slot])
			t := state.State.Timeline
			if number != c.First+int64(c.Count()) || t.ID != o.ID || t.Enterprise != o.Enterprise || t.Source != source || t.SourceNo != number || op.Operation.ID != source || op.Operation.Number != number || op.Operation.Subject != subject || op.Operation.Submission != submission || evidence.ObjectID != o.ID {
				return compact.ErrCorrupt
			}
			for {
				p := compact.NewPool(c.Values)
				a, e := p.ID(source)
				if e != nil {
					return e
				}
				b, e := p.ID(subject)
				if e != nil {
					return e
				}
				d, e := p.ID(submission)
				if e != nil {
					return e
				}
				sb, e := history.EncodeLocationState(m.ctx, p, state)
				if e != nil {
					return e
				}
				ob, e := history.EncodeLocationOperation(m.ctx, p, op)
				if e != nil {
					return e
				}
				eb, e := history.EncodeLocationEvidence(m.ctx, p, evidence)
				if e != nil {
					return e
				}
				added := len(sb) + len(ob) + len(eb) + (len(p.IDs)-len(c.Values.IDs))*16 + (len(p.Ranges)-len(c.Values.Ranges))*24
				for _, v := range p.Strings[len(c.Values.Strings):] {
					added += len(v)
				}
				for _, v := range p.Binary[len(c.Values.Binary):] {
					added += len(v)
				}
				if c.Count() > 0 && (c.Count() == 64 || int(c.EncodedBytes)+added > compact.MaxPart) {
					c.Sealed = true
					output = append(output, c)
					c = history.Chunk{Kind: history.Location, Instance: o.Instance, Enterprise: o.Enterprise, Object: o.ID, Number: c.Number + 1, First: number}
					continue
				}
				si, e := m.putShape(o, sb)
				if e != nil {
					return e
				}
				oi, e := m.putShape(o, ob)
				if e != nil {
					return e
				}
				ei, e := m.putShape(o, eb)
				if e != nil {
					return e
				}
				c.Values = p.Values
				c.Sources = append(c.Sources, int32(a))
				c.Subjects = append(c.Subjects, int32(b))
				c.Submissions = append(c.Submissions, int32(d))
				c.States = append(c.States, si)
				c.Operations = append(c.Operations, oi)
				c.Evidence = append(c.Evidence, ei)
				c.EncodedBytes += int32(added)
				break
			}
			final = state
		}
	}
	if final.State.Timeline.Source != o.Source || final.State.Timeline.SourceNo != o.Number {
		return compact.ErrCorrupt
	}
	// The independent current projection must describe exactly the final source.
	a, _ := json.Marshal(head)
	b, _ := json.Marshal(final)
	if !bytes.Equal(a, b) {
		return compact.ErrCorrupt
	}
	output = append(output, c)
	if _, e = m.tx.ExecContext(m.ctx, `DELETE FROM organization.location_history WHERE instance_id=$1 AND object_id=$2`, o.Instance, o.ID); e != nil {
		return e
	}
	return m.publish(o, output, final)
}

func (m addressMigration) publish(o addressObject, chunks []history.Chunk, final model.LocationState) error {
	for i := range chunks {
		c := &chunks[i]
		var e error
		c.Dictionary, e = m.putDictionary(o, c.Values)
		if e != nil {
			return e
		}
		if e = c.Seal(); e != nil {
			return e
		}
		_, e = m.tx.ExecContext(m.ctx, `INSERT INTO organization.location_history(instance_id,enterprise_id,object_id,chunk_no,first_no,format,ids,dictionary_id,source_refs,subject_refs,submission_refs,states,operations,evidence,envelope,sealed,encoded_bytes) VALUES($1,$2,$3,$4,$5,1,$6::text[]::uuid[],$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`, o.Instance, o.Enterprise, o.ID, c.Number, c.First, c.Values.IDs, c.Dictionary, c.Sources, c.Subjects, c.Submissions, c.States, c.Operations, c.Evidence, c.Envelope, c.Sealed, c.EncodedBytes)
		if e != nil {
			return e
		}
	}
	last := chunks[len(chunks)-1]
	values := last.Values
	values.IDs = nil
	p := compact.NewPool(values)
	b, e := history.EncodeLocationState(m.ctx, p, final)
	if e != nil {
		return e
	}
	if len(p.Strings) != len(values.Strings) || len(p.Binary) != len(values.Binary) || len(p.Ranges) != len(values.Ranges) {
		return compact.ErrCorrupt
	}
	shape, e := m.putShape(o, b)
	if e != nil {
		return e
	}
	refs := []int32{}
	for _, id := range p.IDs {
		n := slices.Index(last.Values.IDs, id)
		if n < 0 {
			return compact.ErrCorrupt
		}
		refs = append(refs, int32(n+1))
	}
	var dh, sh []byte
	e = m.tx.QueryRowContext(m.ctx, `SELECT d.digest,s.digest FROM organization.location_dictionary d,organization.location_shape s WHERE d.id=$1 AND s.id=$2`, last.Dictionary, shape).Scan(&dh, &sh)
	if e != nil {
		return e
	}
	envelope, e := history.ProjectionDigest(history.Location, o.Instance, o.Enterprise, o.ID, o.Source, p.IDs, dh, sh)
	if e != nil {
		return e
	}
	_, e = m.tx.ExecContext(m.ctx, `UPDATE organization.location SET head_chunk=$3,current_shape=$4,current_refs=$5,current_envelope=$6 WHERE instance_id=$1 AND location_id=$2`, o.Instance, o.ID, last.Number, shape, refs, envelope)
	return e
}

package history

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/compact"
	"github.com/jackc/pgx/v5"
)

// Kind is chosen by the owning adapter, never by a request parameter.
type Kind uint8

const (
	BusinessUnit Kind = iota + 1
	Department
	Country
	GeoArea
	Location
	Enterprise
	LegalEntity
)

func (k Kind) prefix() string {
	switch k {
	case BusinessUnit:
		return "bu"
	case Department:
		return "department"
	case GeoArea:
		return "geoarea"
	case LegalEntity:
		return "legal_entity"
	case Location:
		return "location"
	case Enterprise:
		return "enterprise"
	case Country:
		return "country"
	default:
		panic("unknown history owner")
	}
}

// Country material belongs to the authenticated instance catalog, never to an
// enterprise. All SQL templates and owner kinds are fixed by the adapter.
func (kind Kind) format(query string, args ...any) string {
	query = fmt.Sprintf(query, args...)
	if kind == Country || kind == GeoArea || kind == Enterprise {
		query = strings.ReplaceAll(query, "enterprise_id", "catalog_id")
	}
	return query
}

type Chunk struct {
	Kind                                                         Kind
	Instance, Enterprise, Object                                 string
	Number                                                       int32
	First                                                        int64
	Values                                                       compact.Values
	Dictionary                                                   int32
	Sources, Subjects, Submissions, States, Operations, Evidence []int32
	Envelope                                                     []byte
	Sealed                                                       bool
	EncodedBytes                                                 int32
}

func (c Chunk) Count() int { return len(c.Sources) }
func (c Chunk) ID(ref int32) string {
	if ref < 1 || int(ref) > len(c.Values.IDs) {
		return ""
	}
	return c.Values.IDs[ref-1]
}
func (c Chunk) Index(id string) int {
	for i, r := range c.Sources {
		if c.ID(r) == id {
			return i
		}
	}
	return -1
}
func (c Chunk) digest() ([]byte, error) {
	buf := []byte{}
	for _, xs := range [][]int32{c.Sources, c.Subjects, c.Submissions, c.States, c.Operations, c.Evidence} {
		buf = binary.BigEndian.AppendUint32(buf, uint32(len(xs)))
		for _, x := range xs {
			buf = binary.BigEndian.AppendUint32(buf, uint32(x))
		}
	}
	return compact.Digest([]string{c.Kind.prefix(), c.Instance, c.Enterprise, c.Object, strconv.Itoa(int(c.Number)), strconv.FormatInt(c.First, 10), "1"}, c.Values, [][]byte{buf})
}

// Seal binds the owned directory and its complete value pools before storage.
func (c *Chunk) Seal() error {
	var e error
	c.Envelope, e = c.digest()
	if e != nil {
		return e
	}
	return c.Validate()
}
func (c Chunk) Validate() error {
	if c.Number < 1 || c.First < 1 || c.Count() < 1 || c.Count() > 64 || c.EncodedBytes < 1 || (c.EncodedBytes > compact.MaxPart && c.Count() != 1) {
		return compact.ErrCorrupt
	}
	if !compact.IsID(c.Instance) || !compact.IsID(c.Enterprise) || !compact.IsID(c.Object) {
		return compact.ErrCorrupt
	}
	if err := compact.ValidateValues(c.Values); err != nil {
		return err
	}
	n := c.Count()
	for _, xs := range [][]int32{c.Subjects, c.Submissions, c.States, c.Operations, c.Evidence} {
		if len(xs) != n {
			return compact.ErrCorrupt
		}
	}
	seenSources, seenSubmissions := map[string]bool{}, map[string]bool{}
	for i := range n {
		source, subject, sid := c.ID(c.Sources[i]), c.ID(c.Subjects[i]), c.ID(c.Submissions[i])
		if source == "" || subject == "" || sid == "" || seenSources[source] || seenSubmissions[subject+sid] || c.States[i] < 1 || c.Operations[i] < 1 || c.Evidence[i] < 1 {
			return compact.ErrCorrupt
		}
		seenSources[source] = true
		seenSubmissions[subject+sid] = true
	}
	h, e := c.digest()
	if e != nil || !bytes.Equal(h, c.Envelope) {
		return compact.ErrCorrupt
	}
	return nil
}

func LoadChunk(ctx context.Context, tx pgx.Tx, kind Kind, instance, object string, number int32) (Chunk, error) {
	defer foundation.BeginStage(ctx, "history.load_chunk")()
	c := Chunk{Kind: kind, Instance: instance, Object: object, Number: number}
	var starts, ends []string
	q := kind.format(`SELECT h.enterprise_id::text,h.first_no,h.ids::text[],h.dictionary_id,h.source_refs,h.subject_refs,h.submission_refs,h.states,h.operations,h.evidence,h.envelope,h.sealed,h.encoded_bytes,d.words,d.binaries,ARRAY(SELECT coalesce(lower(r)::text,'') FROM unnest(d.ranges) r),ARRAY(SELECT coalesce(upper(r)::text,'') FROM unnest(d.ranges) r)
 FROM organization.%[1]s_history h JOIN organization.%[1]s_dictionary d ON (d.instance_id,d.enterprise_id,d.id)=(h.instance_id,h.enterprise_id,h.dictionary_id) WHERE h.instance_id=$1 AND h.object_id=$2 AND h.chunk_no=$3`, kind.prefix())
	e := tx.QueryRow(ctx, q, instance, object, number).Scan(&c.Enterprise, &c.First, &c.Values.IDs, &c.Dictionary, &c.Sources, &c.Subjects, &c.Submissions, &c.States, &c.Operations, &c.Evidence, &c.Envelope, &c.Sealed, &c.EncodedBytes, &c.Values.Strings, &c.Values.Binary, &starts, &ends)
	if e != nil {
		return c, e
	}
	if len(starts) != len(ends) {
		return c, compact.ErrCorrupt
	}
	for i := range starts {
		c.Values.Ranges = append(c.Values.Ranges, compact.Range{Start: starts[i], End: ends[i]})
	}
	return c, c.Validate()
}

func Locate(ctx context.Context, tx pgx.Tx, kind Kind, instance, object, source string) (int32, error) {
	var n int32
	q := kind.format(`SELECT chunk_no FROM organization.%s_history h WHERE instance_id=$1 AND object_id=$2 AND EXISTS(SELECT FROM unnest(source_refs) r WHERE ids[r]=$3::uuid)`, kind.prefix())
	e := tx.QueryRow(ctx, q, instance, object, source).Scan(&n)
	return n, e
}

func hashParts(parts [][]byte) []byte {
	h := sha256.New()
	_, _ = h.Write(binary.BigEndian.AppendUint32(nil, uint32(len(parts))))
	for _, p := range parts {
		_, _ = h.Write(binary.BigEndian.AppendUint32(nil, uint32(len(p))))
		_, _ = h.Write(p)
	}
	return h.Sum(nil)
}

func putShape(ctx context.Context, tx pgx.Tx, kind Kind, instance, enterprise string, data []byte) (int32, error) {
	defer foundation.BeginStage(ctx, "history.put_shape")()
	parts := [][]byte{}
	for len(data) > 0 {
		n := min(len(data), compact.MaxPart)
		parts = append(parts, data[:n])
		data = data[n:]
	}
	if len(parts) == 0 || len(parts) > 256 {
		return 0, compact.ErrCorrupt
	}
	h := hashParts(parts)
	var id int32
	q := kind.format(`INSERT INTO organization.%s_shape(instance_id,enterprise_id,parts) VALUES($1,$2,$3) ON CONFLICT(instance_id,enterprise_id,digest) DO NOTHING RETURNING id`, kind.prefix())
	e := tx.QueryRow(ctx, q, instance, enterprise, parts).Scan(&id)
	if errors.Is(e, pgx.ErrNoRows) {
		e = tx.QueryRow(ctx, kind.format(`SELECT id FROM organization.%s_shape WHERE instance_id=$1 AND enterprise_id=$2 AND digest=$3`, kind.prefix()), instance, enterprise, h).Scan(&id)
	}
	return id, e
}
func LoadShape(ctx context.Context, tx pgx.Tx, kind Kind, instance, enterprise string, id int32) ([]byte, error) {
	var parts [][]byte
	var digest []byte
	e := tx.QueryRow(ctx, kind.format(`SELECT parts,digest FROM organization.%s_shape WHERE instance_id=$1 AND enterprise_id=$2 AND id=$3`, kind.prefix()), instance, enterprise, id).Scan(&parts, &digest)
	if e != nil {
		return nil, e
	}
	if len(parts) == 0 || len(parts) > 256 || !bytes.Equal(hashParts(parts), digest) {
		return nil, compact.ErrCorrupt
	}
	var data []byte
	for _, p := range parts {
		if len(p) == 0 || len(p) > compact.MaxPart {
			return nil, compact.ErrCorrupt
		}
		data = append(data, p...)
	}
	return data, nil
}
func dictionaryParts(v compact.Values) [][]byte {
	p := [][]byte{binary.BigEndian.AppendUint32(nil, uint32(len(v.Strings)))}
	for _, s := range v.Strings {
		p = append(p, []byte(s))
	}
	p = append(p, binary.BigEndian.AppendUint32(nil, uint32(len(v.Binary))))
	p = append(p, v.Binary...)
	p = append(p, binary.BigEndian.AppendUint32(nil, uint32(len(v.Ranges))))
	for _, r := range v.Ranges {
		p = append(p, []byte(r.Start), []byte(r.End))
	}
	return p
}
func putDictionary(ctx context.Context, tx pgx.Tx, kind Kind, instance, enterprise string, v compact.Values) (int32, error) {
	defer foundation.BeginStage(ctx, "history.put_dictionary")()
	ranges := make([]string, len(v.Ranges))
	for i, r := range v.Ranges {
		ranges[i] = "[" + r.Start + "," + r.End + ")"
		if r.Start == "" && r.End == "" {
			ranges[i] = "empty"
		}
	}
	q := kind.format(`INSERT INTO organization.%s_dictionary(instance_id,enterprise_id,words,binaries,ranges) VALUES($1,$2,$3,$4,$5::text[]::daterange[]) ON CONFLICT(instance_id,enterprise_id,digest) DO NOTHING RETURNING id`, kind.prefix())
	var id int32
	e := tx.QueryRow(ctx, q, instance, enterprise, v.Strings, v.Binary, ranges).Scan(&id)
	if errors.Is(e, pgx.ErrNoRows) {
		e = tx.QueryRow(ctx, kind.format(`SELECT id FROM organization.%s_dictionary WHERE instance_id=$1 AND enterprise_id=$2 AND digest=$3 FOR KEY SHARE`, kind.prefix()), instance, enterprise, hashParts(dictionaryParts(v))).Scan(&id)
	}
	return id, e
}

// Encoder is implemented by the BU or Department adapter using its typed
// codec. Re-encoding on a new chunk only remaps local references.
type Encoder func(*compact.Pool) (state, operation, evidence []byte, err error)

// Append consumes the exact last chunk already loaded under the object's write
// guard, or the previous append in this same transaction. A new object supplies
// its empty first chunk. It never accepts a cross-request cached mutable chunk;
// the database still checks every unchanged prefix and the atomic head closure.
func Append(ctx context.Context, tx pgx.Tx, c Chunk, sourceNo int64, source, subject, submission string, encode Encoder) (Chunk, error) {
	defer foundation.BeginStage(ctx, "history.append")()
	kind, instance, enterprise, object := c.Kind, c.Instance, c.Enterprise, c.Object
	var err error
	if sourceNo != c.First+int64(c.Count()) {
		return c, fmt.Errorf("non-contiguous source: %w", compact.ErrCorrupt)
	}
	nextChunk := func() error {
		if !c.Sealed {
			if _, e := tx.Exec(ctx, kind.format(`UPDATE organization.%s_history SET sealed=true WHERE instance_id=$1 AND object_id=$2 AND chunk_no=$3`, kind.prefix()), instance, object, c.Number); e != nil {
				return e
			}
		}
		c = Chunk{Kind: kind, Instance: instance, Enterprise: enterprise, Object: object, Number: c.Number + 1, First: sourceNo}
		return nil
	}
	// These bounds are known before encoding. Do not clone a closed chunk's
	// complete pools and encode a large source just to discard that result.
	if c.Count() > 0 && (c.Sealed || c.Count() == 64 || c.EncodedBytes >= compact.MaxPart) {
		if err = nextChunk(); err != nil {
			return c, err
		}
	}
	for {
		p := compact.NewPool(c.Values)
		a, e := p.ID(source)
		if e != nil {
			return c, e
		}
		b, e := p.ID(subject)
		if e != nil {
			return c, e
		}
		d, e := p.ID(submission)
		if e != nil {
			return c, e
		}
		encoded := foundation.BeginStage(ctx, "history.encode")
		state, operation, evidence, e := encode(p)
		encoded()
		if e != nil {
			return c, e
		}
		// The physical directory is bounded. Oversize typed values are split into
		// independently checksummed parts; they never extend a previous source.
		added := len(state) + len(operation) + len(evidence) + (len(p.IDs)-len(c.Values.IDs))*16
		for _, s := range p.Strings[len(c.Values.Strings):] {
			added += len(s)
		}
		for _, b := range p.Binary[len(c.Values.Binary):] {
			added += len(b)
		}
		added += (len(p.Ranges) - len(c.Values.Ranges)) * 24
		if c.Count() > 0 && int(c.EncodedBytes)+added > compact.MaxPart {
			if err = nextChunk(); err != nil {
				return c, err
			}
			continue
		}
		// A single large timeline/path uses individually bounded shape parts.
		// It owns a chunk of its own; the next append seals it before proceeding.
		isNew := c.Count() == 0
		// Finish the object's dictionary before acquiring an uncommitted shared
		// shape. All material stays inside the same business transaction.
		c.Values = p.Values
		c.Dictionary, e = putDictionary(ctx, tx, kind, instance, enterprise, c.Values)
		if e != nil {
			return c, e
		}
		stateID, e := putShape(ctx, tx, kind, instance, enterprise, state)
		if e != nil {
			return c, e
		}
		opID, e := putShape(ctx, tx, kind, instance, enterprise, operation)
		if e != nil {
			return c, e
		}
		evidenceID, e := putShape(ctx, tx, kind, instance, enterprise, evidence)
		if e != nil {
			return c, e
		}
		c.Sources = append(c.Sources, int32(a))
		c.Subjects = append(c.Subjects, int32(b))
		c.Submissions = append(c.Submissions, int32(d))
		c.States = append(c.States, stateID)
		c.Operations = append(c.Operations, opID)
		c.Evidence = append(c.Evidence, evidenceID)
		c.EncodedBytes += int32(added)
		c.Envelope, e = c.digest()
		if e != nil {
			return c, e
		}
		if e = c.Validate(); e != nil {
			return c, e
		}
		columns := `instance_id,enterprise_id,object_id,chunk_no,first_no,format,ids,dictionary_id,source_refs,subject_refs,submission_refs,states,operations,evidence,envelope,sealed,encoded_bytes`
		args := []any{instance, enterprise, object, c.Number, c.First, 1, c.Values.IDs, c.Dictionary, c.Sources, c.Subjects, c.Submissions, c.States, c.Operations, c.Evidence, c.Envelope, c.Sealed, c.EncodedBytes}
		q := kind.format(`INSERT INTO organization.%s_history(%s) VALUES($1,$2,$3,$4,$5,$6,$7::text[]::uuid[],$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`, kind.prefix(), columns)
		if !isNew {
			q = kind.format(`UPDATE organization.%s_history SET enterprise_id=$2,first_no=$5,format=$6,ids=$7::text[]::uuid[],dictionary_id=$8,source_refs=$9,subject_refs=$10,submission_refs=$11,states=$12,operations=$13,evidence=$14,envelope=$15,sealed=$16,encoded_bytes=$17 WHERE instance_id=$1 AND object_id=$3 AND chunk_no=$4`, kind.prefix())
		}
		written := foundation.BeginStage(ctx, "history.sql_write")
		_, err = tx.Exec(ctx, q, args...)
		written()
		return c, err
	}
}

type Found struct {
	Object string
	Chunk  int32
	Slot   int
}

func FindSubmission(ctx context.Context, tx pgx.Tx, kind Kind, instance, subject, submission string) (*Found, error) {
	tag, e := compact.Tag(submission)
	if e != nil {
		return nil, e
	}
	rows, e := tx.Query(ctx, kind.format(`SELECT object_id::text,chunk_no,organization.submission_slot(ids,submission_refs,subject_refs,$3,$4)-1 FROM organization.%s_history h WHERE instance_id=$1 AND submission_tags @> ARRAY[$2::integer] AND organization.submission_slot(ids,submission_refs,subject_refs,$3,$4) IS NOT NULL`, kind.prefix()), instance, tag, subject, submission)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var found *Found
	for rows.Next() {
		if found != nil {
			return nil, compact.ErrCorrupt
		}
		v := new(Found)
		if e = rows.Scan(&v.Object, &v.Chunk, &v.Slot); e != nil {
			return nil, e
		}
		found = v
	}
	return found, rows.Err()
}

type TerminalEnvelope struct{ Object, Outcome string }

// TerminalIdentity deliberately reads only the native authorization envelope;
// corrupt optional details must not precede current object read authorization.
func TerminalIdentity(ctx context.Context, tx pgx.Tx, kind Kind, instance, subject, submission string) (TerminalEnvelope, error) {
	var v TerminalEnvelope
	e := tx.QueryRow(ctx, kind.format(`SELECT coalesce(request_object_id::text,''),outcome FROM organization.%s_terminal WHERE instance_id=$1 AND subject_id=$2 AND submission_id=$3`, kind.prefix()), instance, subject, submission).Scan(&v.Object, &v.Outcome)
	return v, e
}
func SaveTerminal(ctx context.Context, tx pgx.Tx, kind Kind, instance, enterprise, subject, submission string, meta TerminalEnvelope, encode func(*compact.Pool) ([]byte, error)) error {
	p := compact.NewPool(compact.Values{})
	data, e := encode(p)
	if e != nil {
		return e
	}
	dict, e := putDictionary(ctx, tx, kind, instance, enterprise, p.Values)
	if e != nil {
		return e
	}
	shape, e := putShape(ctx, tx, kind, instance, enterprise, data)
	if e != nil {
		return e
	}
	h, e := compact.Digest([]string{kind.prefix(), instance, enterprise, subject, submission, meta.Object, meta.Outcome, "terminal"}, p.Values, [][]byte{binary.BigEndian.AppendUint32(nil, uint32(shape))})
	if e != nil {
		return e
	}
	_, e = tx.Exec(ctx, kind.format(`INSERT INTO organization.%s_terminal(instance_id,enterprise_id,subject_id,submission_id,ids,dictionary_id,operation,envelope,request_object_id,outcome) VALUES($1,$2,$3,$4,$5::text[]::uuid[],$6,$7,$8,NULLIF($9,'')::uuid,$10)`, kind.prefix()), instance, enterprise, subject, submission, p.IDs, dict, shape, h, meta.Object, meta.Outcome)
	return e
}
func LoadTerminal(ctx context.Context, tx pgx.Tx, kind Kind, instance, subject, submission string) (compact.Values, []byte, error) {
	var v compact.Values
	var enterprise string
	var meta TerminalEnvelope
	var operation int32
	var envelope []byte
	var starts, ends []string
	q := kind.format(`SELECT coalesce(t.request_object_id::text,''),t.outcome,t.enterprise_id::text,t.ids::text[],t.operation,t.envelope,d.words,d.binaries,ARRAY(SELECT coalesce(lower(r)::text,'') FROM unnest(d.ranges) r),ARRAY(SELECT coalesce(upper(r)::text,'') FROM unnest(d.ranges) r) FROM organization.%[1]s_terminal t JOIN organization.%[1]s_dictionary d ON (d.instance_id,d.enterprise_id,d.id)=(t.instance_id,t.enterprise_id,t.dictionary_id) WHERE t.instance_id=$1 AND t.subject_id=$2 AND t.submission_id=$3`, kind.prefix())
	e := tx.QueryRow(ctx, q, instance, subject, submission).Scan(&meta.Object, &meta.Outcome, &enterprise, &v.IDs, &operation, &envelope, &v.Strings, &v.Binary, &starts, &ends)
	if e != nil {
		return v, nil, e
	}
	if len(starts) != len(ends) {
		return v, nil, compact.ErrCorrupt
	}
	for i := range starts {
		v.Ranges = append(v.Ranges, compact.Range{Start: starts[i], End: ends[i]})
	}
	if e = compact.ValidateValues(v); e != nil {
		return v, nil, e
	}
	h, e := compact.Digest([]string{kind.prefix(), instance, enterprise, subject, submission, meta.Object, meta.Outcome, "terminal"}, v, [][]byte{binary.BigEndian.AppendUint32(nil, uint32(operation))})
	if e != nil || !bytes.Equal(envelope, h) {
		return v, nil, compact.ErrCorrupt
	}
	b, e := LoadShape(ctx, tx, kind, instance, enterprise, operation)
	return v, b, e
}

// PruneDictionaries only removes content no committed source or terminal uses.
// It belongs to maintenance of a task-owned database, never history retention.
func PruneDictionaries(ctx context.Context, tx pgx.Tx, kind Kind, instance string) error {
	if !compact.IsID(instance) {
		return compact.ErrCorrupt
	}
	p := kind.prefix()
	if kind == Country || kind == GeoArea || kind == Location || kind == Enterprise || kind == LegalEntity {
		_, e := tx.Exec(ctx, kind.format(`DELETE FROM organization.%[1]s_dictionary d WHERE d.instance_id=$1 AND NOT EXISTS(SELECT FROM organization.%[1]s_history h WHERE h.dictionary_id=d.id) AND NOT EXISTS(SELECT FROM organization.%[1]s_terminal t WHERE t.dictionary_id=d.id)`, p), instance)
		return e
	}
	_, e := tx.Exec(ctx, kind.format(`DELETE FROM organization.%[1]s_dictionary d WHERE d.instance_id=$1 AND NOT EXISTS(SELECT FROM organization.%[1]s_history h WHERE h.dictionary_id=d.id) AND NOT EXISTS(SELECT FROM organization.%[1]s_terminal t WHERE t.dictionary_id=d.id) AND NOT EXISTS(SELECT FROM organization.%[1]s_read_segment r WHERE r.dictionary_id=d.id)`, p), instance)
	return e
}

func RangesSQL(v compact.Values) []string {
	out := make([]string, len(v.Ranges))
	for i, r := range v.Ranges {
		out[i] = "[" + strings.Join([]string{r.Start, r.End}, ",") + ")"
	}
	return out
}

package history

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history/model"
	"chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/compact"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const ReadRule = 4

// RebuildCurrentReadIndex is exclusively for an explicitly owned, offline
// candidate. It rewrites derivatives and leaves sources and results unchanged.
func RebuildCurrentReadIndex(ctx context.Context, tx pgx.Tx, kind Kind, instance, object, source string, number int32) error {
	c, err := LoadChunk(ctx, tx, kind, instance, object, number)
	if err != nil {
		return err
	}
	i := c.Index(source)
	if i < 0 {
		return compact.ErrCorrupt
	}
	raw, err := LoadShape(ctx, tx, kind, instance, c.Enterprise, c.States[i])
	if err != nil {
		return err
	}
	return saveReadSegments(ctx, tx, c, source, c.Values, raw)
}

type readSegment struct {
	start, end string
	department d.Timeline
	bu         d.BusinessUnit
}

type encodedReadSegment struct {
	start, end string
	data       []byte
	ids        []string
	query      departmentQueryFacts
}

// Each call is confined to one validated source, dictionary and rule. Compare
// the complete structure and ordered UUID directory, never visible labels or
// a shortened digest. The real first member and its date remain in the data.
func appendReadSegment(out []encodedReadSegment, next encodedReadSegment) []encodedReadSegment {
	if n := len(out); n > 0 {
		previous := &out[n-1]
		// Preserve invalid/empty ranges for the existing native rejection. An
		// empty interval must never disappear into its identical neighbour.
		previousNonempty := previous.start == "" || previous.end == "" || previous.start < previous.end
		nextNonempty := next.start == "" || next.end == "" || next.start < next.end
		if previousNonempty && nextNonempty && previous.end != "" && previous.end == next.start && bytes.Equal(previous.data, next.data) && slices.Equal(previous.ids, next.ids) && previous.query == next.query {
			previous.end = next.end
			return out
		}
	}
	// ResetIDs reuses the encoder's directory for the next span.
	next.ids = slices.Clone(next.ids)
	return append(out, next)
}

// The complete source was validated before its atomic commit. The index keeps
// the real first member and the member/edge covering each Go-generated span.
// Keeping the first member preserves future-first scope semantics without
// synthesizing a second business fact or copying operation summaries.
func departmentSegments(t d.Timeline) ([]readSegment, error) {
	if len(t.Records) == 0 || t.Records[0].ID != t.First {
		return nil, compact.ErrCorrupt
	}
	boundaries := []string{}
	for _, r := range t.Records {
		boundaries = append(boundaries, r.Date)
	}
	for i, e := range t.Edges {
		if e.End != "" && e.Start >= e.End || i > 0 && (t.Edges[i-1].End == "" || t.Edges[i-1].End > e.Start) {
			return nil, compact.ErrCorrupt
		}
		boundaries = append(boundaries, e.Start)
		if e.End != "" {
			boundaries = append(boundaries, e.End)
		}
	}
	slices.Sort(boundaries)
	boundaries = slices.Compact(boundaries)
	out := []readSegment{}
	edgeIndex := 0
	for i := -1; i < len(boundaries); i++ {
		start, end := "", ""
		if i >= 0 {
			start = boundaries[i]
		}
		if i+1 < len(boundaries) {
			end = boundaries[i+1]
		}
		date := start
		if date == "" {
			date = t.Records[0].Date
		}
		r := d.ScopeRecord(t, date)
		if r == nil {
			return nil, compact.ErrCorrupt
		}
		v := t
		v.Records = []d.Record{t.Records[0]}
		v.Edges = nil
		if r.ID != t.First {
			v.Records = append(v.Records, *r)
		}
		// Validated edges are ordered and non-overlapping. Advance once across
		// all spans instead of scanning every edge at every boundary.
		for edgeIndex < len(t.Edges) && t.Edges[edgeIndex].End != "" && t.Edges[edgeIndex].End <= date {
			edgeIndex++
		}
		if edgeIndex < len(t.Edges) && t.Edges[edgeIndex].Start <= date {
			v.Edges = []d.Edge{t.Edges[edgeIndex]}
		}
		if i == -1 {
			// The synthetic before-first span uses the real first record's
			// date; actual relation boundaries may precede that date.
			edgeIndex = 0
		}
		out = append(out, readSegment{start: start, end: end, department: v})
	}
	return out, nil
}
func buSegments(t d.BusinessUnit) ([]readSegment, error) {
	if len(t.Records) == 0 || t.Records[0].ID != t.First {
		return nil, compact.ErrCorrupt
	}
	out := []readSegment{}
	for i := -1; i < len(t.Records); i++ {
		start, end := "", ""
		if i >= 0 {
			start = t.Records[i].Date
		}
		if i+1 < len(t.Records) {
			end = t.Records[i+1].Date
		}
		v := t
		v.Records = []d.BURecord{t.Records[0]}
		if i > 0 {
			v.Records = append(v.Records, t.Records[i])
		}
		out = append(out, readSegment{start: start, end: end, bu: v})
	}
	return out, nil
}

func ReadSegmentDigest(kind Kind, instance, enterprise, object, source, start, end string, ids []string, dictHash, shapeHash []byte) ([]byte, error) {
	return compact.Digest([]string{kind.prefix(), instance, enterprise, object, source, "read-segment-4", start, end}, compact.Values{IDs: ids}, [][]byte{dictHash, shapeHash})
}

func saveReadSegments(ctx context.Context, tx pgx.Tx, c Chunk, source string, full compact.Values, raw []byte) error {
	defer foundation.BeginStage(ctx, "segments.publish")()
	buildDone := foundation.BeginStage(ctx, "segments.build_encode")
	var segments []readSegment
	var err error
	var code, first string
	if c.Kind == Department {
		v, e := DecodeDepartmentState(ctx, full, raw)
		if e != nil {
			return e
		}
		if v.Timeline.ID != c.Object || v.Timeline.Enterprise != c.Enterprise || v.Timeline.Source != source {
			return compact.ErrCorrupt
		}
		if err = tx.QueryRow(ctx, `SELECT code FROM organization.department WHERE instance_id=$1 AND department_id=$2 AND head_source_id=$3`, c.Instance, c.Object, source).Scan(&code); err != nil {
			return err
		}
		v.Timeline.Code = code
		first = v.Timeline.First
		segments, err = departmentSegments(v.Timeline)
	} else {
		v, e := DecodeBUState(ctx, full, raw)
		if e != nil {
			return e
		}
		t := d.BusinessUnit{ID: v.Timeline.BUID, Code: v.Timeline.Code, Source: v.Timeline.Source, First: v.Timeline.First}
		for _, r := range v.Timeline.Records {
			t.Records = append(t.Records, d.BURecord{ID: r.ID, Revision: r.Revision, Date: r.Date, End: r.End, State: r.State, Name: r.Data.Name})
		}
		segments, err = buSegments(t)
	}
	if err != nil {
		return err
	}
	table := c.Kind.prefix() + "_read_segment"
	byID := make(map[string]int32, len(c.Values.IDs))
	for i, id := range c.Values.IDs {
		byID[id] = int32(i + 1)
	}
	rows := make([][]any, 0, len(segments))
	structures := make([][]byte, 0, len(segments))
	structureHashes := make([][]byte, 0, len(segments))
	dictionaryHash := hashParts(dictionaryParts(c.Values))
	values := c.Values
	values.IDs = nil
	p := compact.NewPool(values)
	encoded := make([]encodedReadSegment, 0, len(segments))
	for _, segment := range segments {
		if err = ctx.Err(); err != nil {
			return err
		}
		p.ResetIDs()
		var data []byte
		if c.Kind == Department {
			data, err = EncodeDepartmentRead(ctx, p, model.ReadDepartment(segment.department))
		} else {
			data, err = EncodeBURead(ctx, p, model.BURead{Timeline: segment.bu})
		}
		if err != nil {
			return err
		}
		if len(p.Strings) != len(c.Values.Strings) || len(p.Ranges) != len(c.Values.Ranges) || len(p.Binary) != len(c.Values.Binary) {
			return compact.ErrCorrupt
		}
		// A read segment has at most two records and one applicable edge. Its
		// generated structure contains references, not raw names or evidence.
		if len(data) == 0 || len(data) > compact.MaxPart {
			return compact.ErrCorrupt
		}
		next := encodedReadSegment{start: segment.start, end: segment.end, data: data, ids: p.IDs}
		if c.Kind == Department {
			next.query, err = departmentQuery(segment.department, segment.start)
			if err != nil {
				return err
			}
		}
		encoded = appendReadSegment(encoded, next)
	}
	for slot, segment := range encoded {
		if err = ctx.Err(); err != nil {
			return err
		}
		ids := make([]pgtype.UUID, len(segment.ids))
		for i, id := range segment.ids {
			if byID[id] == 0 {
				return compact.ErrCorrupt
			}
			if err = ids[i].Scan(id); err != nil {
				return err
			}
		}
		parts := [][]byte{}
		for rest := segment.data; len(rest) > 0; {
			n := min(len(rest), compact.MaxPart)
			parts = append(parts, rest[:n])
			rest = rest[n:]
		}
		hash := hashParts(parts)
		envelope, err := ReadSegmentDigest(c.Kind, c.Instance, c.Enterprise, c.Object, source, segment.start, segment.end, segment.ids, dictionaryHash, hash)
		if err != nil {
			return err
		}
		span := pgtype.Range[pgtype.Date]{LowerType: pgtype.Unbounded, UpperType: pgtype.Unbounded, Valid: true}
		if segment.start != "" {
			value, e := time.Parse("2006-01-02", segment.start)
			if e != nil {
				return e
			}
			span.Lower = pgtype.Date{Time: value, Valid: true}
			span.LowerType = pgtype.Inclusive
		}
		if segment.end != "" {
			value, e := time.Parse("2006-01-02", segment.end)
			if e != nil {
				return e
			}
			span.Upper = pgtype.Date{Time: value, Valid: true}
			span.UpperType = pgtype.Exclusive
		}
		rows = append(rows, []any{c.Instance, c.Object, int32(slot), source, int32(ReadRule), span, c.Dictionary, int32(0), ids, envelope})
		structures = append(structures, segment.data)
		structureHashes = append(structureHashes, hash)
	}
	buildDone()
	shapeDone := foundation.BeginStage(ctx, "segments.shapes")
	byHash, err := readShapeIDs(ctx, tx, c, structures, structureHashes)
	shapeDone()
	if err != nil {
		return err
	}
	for i, hash := range structureHashes {
		id := byHash[string(hash)]
		if id == 0 {
			return compact.ErrCorrupt
		}
		rows[i][7] = id
		if c.Kind == Department {
			seal := departmentQueryDigest(c, source, code, first, int32(i), id, encoded[i], rows[i][9].([]byte), dictionaryHash, hash)
			values, e := encoded[i].query.values(seal)
			if e != nil {
				return e
			}
			rows[i] = append(rows[i], values...)
		}
	}
	if c.Kind == BusinessUnit {
		updateDone := foundation.BeginStage(ctx, "segments.update")
		rows, err = updateBUReadSegments(ctx, tx, c, source, rows, encoded)
		updateDone()
		if err != nil {
			return err
		}
	} else {
		if _, err = tx.Exec(ctx, fmt.Sprintf(`DELETE FROM organization.%s WHERE instance_id=$1 AND object_id=$2`, table), c.Instance, c.Object); err != nil {
			return err
		}
	}
	columns := []string{"instance_id", "object_id", "slot", "source_id", "rule", "span", "dictionary_id", "shape_id", "ids", "envelope"}
	if c.Kind == Department {
		columns = append(columns, departmentQueryColumns...)
	}
	if len(rows) == 0 {
		return nil
	}
	copyDone := foundation.BeginStage(ctx, "segments.copy")
	_, err = tx.CopyFrom(ctx, pgx.Identifier{"organization", table}, columns, pgx.CopyFromRows(rows))
	copyDone()
	return err
}

// Keep the Go-derived slot and range when both still match. Updating payload
// bindings avoids deleting and rebuilding unchanged keys and range entries.
// Remove changed ranges first, so immediate exclusion constraints never see a
// transient overlap during a redate, insertion, deletion or segment merge.
// The caller publishes every row and the new head in the same transaction.
func updateBUReadSegments(ctx context.Context, tx pgx.Tx, c Chunk, source string, rows [][]any, encoded []encodedReadSegment) ([][]any, error) {
	slots := make([]int32, len(rows))
	spans := make([]pgtype.Range[pgtype.Date], len(rows))
	shapes := make([]int32, len(rows))
	ids := make([]string, len(rows))
	envelopes := make([][]byte, len(rows))
	for i, row := range rows {
		slots[i] = row[2].(int32)
		spans[i] = row[5].(pgtype.Range[pgtype.Date])
		shapes[i] = row[7].(int32)
		ids[i] = "{" + strings.Join(encoded[i].ids, ",") + "}"
		envelopes[i] = row[9].([]byte)
	}
	// The array size varies from one to thousands of spans. Reuse the parameter
	// description, but plan these two bulk statements with the actual values:
	// a cached generic plan can estimate one target row and repeatedly rescan
	// the whole input array. Other statements keep their normal prepared plans.
	_, err := tx.Exec(ctx, `DELETE FROM organization.bu_read_segment r WHERE r.instance_id=$1 AND r.object_id=$2
 AND NOT EXISTS(SELECT FROM unnest($3::integer[],$4::daterange[]) v(slot,span) WHERE v.slot=r.slot AND v.span=r.span)`, pgx.QueryExecModeCacheDescribe, c.Instance, c.Object, slots, spans)
	if err != nil {
		return nil, err
	}
	updated, err := tx.Query(ctx, `UPDATE organization.bu_read_segment r SET source_id=$8,rule=$9,dictionary_id=$10,shape_id=v.shape,ids=v.ids::uuid[],envelope=v.envelope
 FROM unnest($3::integer[],$4::daterange[],$5::integer[],$6::text[],$7::bytea[]) v(slot,span,shape,ids,envelope)
 WHERE r.instance_id=$1 AND r.object_id=$2 AND r.slot=v.slot AND r.span=v.span RETURNING r.slot`, pgx.QueryExecModeCacheDescribe, c.Instance, c.Object, slots, spans, shapes, ids, envelopes, source, int32(ReadRule), c.Dictionary)
	if err != nil {
		return nil, err
	}
	defer updated.Close()
	kept := make([]bool, len(rows))
	for updated.Next() {
		var slot int32
		if err = updated.Scan(&slot); err != nil {
			return nil, err
		}
		if slot < 0 || int(slot) >= len(kept) || kept[slot] {
			return nil, compact.ErrCorrupt
		}
		kept[slot] = true
	}
	if err = updated.Err(); err != nil {
		return nil, err
	}
	remaining := rows[:0]
	for i, row := range rows {
		if !kept[i] {
			remaining = append(remaining, row)
		}
	}
	return remaining, nil
}

// Shapes are immutable and shared. A changed UUID or dictionary value often
// leaves the encoded reference structure unchanged. Look up those structures
// before writing, so a long correction does not attempt thousands of inserts
// whose generated hashes and constraints run only to discover a conflict.
func readShapeIDs(ctx context.Context, tx pgx.Tx, c Chunk, structures, hashes [][]byte) (map[string]int32, error) {
	byHash := make(map[string]int32, len(hashes))
	lookup := func(wanted [][]byte) error {
		rows, err := tx.Query(ctx, fmt.Sprintf(`SELECT id,digest FROM organization.%s_shape WHERE instance_id=$1 AND enterprise_id=$2 AND digest=ANY($3::bytea[])`, c.Kind.prefix()), c.Instance, c.Enterprise, wanted)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id int32
			var hash []byte
			if err = rows.Scan(&id, &hash); err != nil {
				return err
			}
			byHash[string(hash)] = id
		}
		return rows.Err()
	}
	if err := lookup(hashes); err != nil {
		return nil, err
	}
	missing, missingHashes := [][]byte{}, [][]byte{}
	seen := make(map[string]bool)
	for i, hash := range hashes {
		key := string(hash)
		if byHash[key] == 0 && !seen[key] {
			seen[key] = true
			missing = append(missing, structures[i])
			missingHashes = append(missingHashes, hash)
		}
	}
	if len(missing) == 0 {
		return byHash, nil
	}
	_, err := tx.Exec(ctx, fmt.Sprintf(`INSERT INTO organization.%s_shape(instance_id,enterprise_id,parts) SELECT $1,$2,ARRAY[data] FROM unnest($3::bytea[]) data ON CONFLICT(instance_id,enterprise_id,digest) DO NOTHING`, c.Kind.prefix()), c.Instance, c.Enterprise, missing)
	if err != nil {
		return nil, err
	}
	// Another object's writer can insert the same immutable structure. Resolve
	// the actual identity after ON CONFLICT, using the next statement snapshot.
	if err = lookup(missingHashes); err != nil {
		return nil, err
	}
	return byHash, nil
}

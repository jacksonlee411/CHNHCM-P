package history

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"chnhcm.local/chnhcm/internal/platform/compact"
	"github.com/jackc/pgx/v5"
)

type catalogKey struct {
	kind                 Kind
	instance, enterprise string
	id                   int32
}
type dictionary struct {
	values compact.Values
	digest []byte
}
type shape struct{ data, digest []byte }

// Catalog caches immutable encoding material, never authorization or a mutable
// head. Every request obtains its own head identities from PostgreSQL.
type Catalog struct {
	mu           sync.RWMutex
	dictionaries map[catalogKey]dictionary
	shapes       map[catalogKey]shape
	bytes        int
	active       int
}
type Material struct {
	Values                              compact.Values
	Data, DictionaryDigest, ShapeDigest []byte
}

func (c *Catalog) Prepare(ctx context.Context, tx pgx.Tx, kind Kind, instance, enterprise string, dicts, shapes []int32) error {
	wantD, wantS := []int32{}, []int32{}
	seenD, seenS := map[int32]bool{}, map[int32]bool{}
	c.mu.RLock()
	for _, id := range dicts {
		k := catalogKey{kind, instance, enterprise, id}
		if _, ok := c.dictionaries[k]; !ok && !seenD[id] {
			seenD[id] = true
			wantD = append(wantD, id)
		}
	}
	for _, id := range shapes {
		k := catalogKey{kind, instance, enterprise, id}
		if _, ok := c.shapes[k]; !ok && !seenS[id] {
			seenS[id] = true
			wantS = append(wantS, id)
		}
	}
	c.mu.RUnlock()
	ds := map[catalogKey]dictionary{}
	ss := map[catalogKey]shape{}
	size := 0
	if len(wantD) > 0 {
		rows, e := tx.Query(ctx, kind.format(`SELECT id,words,binaries,ARRAY(SELECT coalesce(lower(r)::text,'') FROM unnest(ranges) r),ARRAY(SELECT coalesce(upper(r)::text,'') FROM unnest(ranges) r),digest FROM organization.%s_dictionary WHERE instance_id=$1 AND enterprise_id=$2 AND id=ANY($3)`, kind.prefix()), instance, enterprise, wantD)
		if e != nil {
			return e
		}
		for rows.Next() {
			var id int32
			var d dictionary
			var starts, ends []string
			if e = rows.Scan(&id, &d.values.Strings, &d.values.Binary, &starts, &ends, &d.digest); e != nil {
				rows.Close()
				return e
			}
			if len(starts) != len(ends) {
				rows.Close()
				return compact.ErrCorrupt
			}
			for i := range starts {
				d.values.Ranges = append(d.values.Ranges, compact.Range{Start: starts[i], End: ends[i]})
			}
			if !bytes.Equal(hashParts(dictionaryParts(d.values)), d.digest) {
				rows.Close()
				return compact.ErrCorrupt
			}
			ds[catalogKey{kind, instance, enterprise, id}] = d
			for _, v := range dictionaryParts(d.values) {
				size += len(v)
			}
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		if len(ds) != len(wantD) {
			return compact.ErrCorrupt
		}
	}
	if len(wantS) > 0 {
		rows, e := tx.Query(ctx, kind.format(`SELECT id,parts,digest FROM organization.%s_shape WHERE instance_id=$1 AND enterprise_id=$2 AND id=ANY($3)`, kind.prefix()), instance, enterprise, wantS)
		if e != nil {
			return e
		}
		for rows.Next() {
			var id int32
			var s shape
			var parts [][]byte
			if e = rows.Scan(&id, &parts, &s.digest); e != nil {
				rows.Close()
				return e
			}
			if len(parts) == 0 || len(parts) > 256 || !bytes.Equal(hashParts(parts), s.digest) {
				rows.Close()
				return compact.ErrCorrupt
			}
			for _, p := range parts {
				if len(p) == 0 || len(p) > compact.MaxPart {
					rows.Close()
					return compact.ErrCorrupt
				}
				s.data = append(s.data, p...)
			}
			ss[catalogKey{kind, instance, enterprise, id}] = s
			size += len(s.data)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		if len(ss) != len(wantS) {
			return compact.ErrCorrupt
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.dictionaries == nil {
		c.dictionaries = map[catalogKey]dictionary{}
		c.shapes = map[catalogKey]shape{}
	}
	for k, v := range ds {
		c.dictionaries[k] = v
	}
	for k, v := range ss {
		c.shapes[k] = v
	}
	c.bytes += size
	return nil
}
func (c *Catalog) Get(kind Kind, instance, enterprise string, dict, root int32) (Material, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	d, ok := c.dictionaries[catalogKey{kind, instance, enterprise, dict}]
	s, ok2 := c.shapes[catalogKey{kind, instance, enterprise, root}]
	if !ok || !ok2 {
		return Material{}, compact.ErrCorrupt
	}
	return Material{d.values, s.data, d.digest, s.digest}, nil
}
func (c *Catalog) Enter() { c.mu.Lock(); c.active++; c.mu.Unlock() }
func (c *Catalog) Leave() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.active--
	if c.active == 0 && c.bytes > 64*1024*1024 {
		c.dictionaries = nil
		c.shapes = nil
		c.bytes = 0
	}
}

func ProjectionDigest(kind Kind, instance, enterprise, object, source string, ids []string, dictHash, shapeHash []byte) ([]byte, error) {
	return compact.Digest([]string{kind.prefix(), instance, enterprise, object, source, "current-1"}, compact.Values{IDs: ids}, [][]byte{dictHash, shapeHash})
}

// SaveProjection references UUIDs in the same history chunk. Only the local
// numbers and an immutable shape are additional storage.
func SaveProjection(ctx context.Context, tx pgx.Tx, c Chunk, source string, encode func(*compact.Pool) ([]byte, error)) error {
	return PublishProjection(ctx, tx, c, source, encode, func(p Projection) error {
		table, key := "business_unit", "bu_id"
		if c.Kind == Department {
			table, key = "department", "department_id"
		}
		if c.Kind == LegalEntity {
			table, key = "legal_entity", "legal_entity_id"
		}
		if c.Kind == Enterprise {
			table, key = "enterprise_profile", "enterprise_id"
		}
		if c.Kind == Location {
			table, key = "location", "location_id"
		}
		if c.Kind == Country {
			table, key = "country_region", "country_region_id"
		}
		if c.Kind == GeoArea {
			table, key = "geoarea", "geoarea_id"
		}
		_, err := tx.Exec(ctx, fmt.Sprintf(`UPDATE organization.%s SET current_shape=$3,current_refs=$4,current_envelope=$5 WHERE instance_id=$1 AND %s=$2`, table, key), c.Instance, c.Object, p.Shape, p.References, p.Envelope)
		return err
	})
}

type Projection struct {
	Shape      int32
	References []int32
	Envelope   []byte
}

// PublishProjection lets the object adapter publish the source head and its
// complete projection in one UPDATE. Shape ownership, UUID membership, seals
// and every read segment are still written in the caller's atomic transaction.
func PublishProjection(ctx context.Context, tx pgx.Tx, c Chunk, source string, encode func(*compact.Pool) ([]byte, error), publish func(Projection) error) error {
	values := c.Values
	values.IDs = nil
	p := compact.NewPool(values)
	b, e := encode(p)
	if e != nil {
		return e
	}
	readBytes := b
	if len(p.Strings) != len(c.Values.Strings) || len(p.Binary) != len(c.Values.Binary) || len(p.Ranges) != len(c.Values.Ranges) {
		return compact.ErrCorrupt
	}
	root, e := putShape(ctx, tx, c.Kind, c.Instance, c.Enterprise, b)
	if e != nil {
		return e
	}
	byID := map[string]int32{}
	for i, id := range c.Values.IDs {
		byID[id] = int32(i + 1)
	}
	refs := make([]int32, len(p.IDs))
	for i, id := range p.IDs {
		refs[i] = byID[id]
		if refs[i] == 0 {
			return compact.ErrCorrupt
		}
	}
	parts := [][]byte{}
	for len(b) > 0 {
		n := min(len(b), compact.MaxPart)
		parts = append(parts, b[:n])
		b = b[n:]
	}
	digest, e := ProjectionDigest(c.Kind, c.Instance, c.Enterprise, c.Object, source, p.IDs, hashParts(dictionaryParts(c.Values)), hashParts(parts))
	if e != nil {
		return e
	}
	if e = publish(Projection{root, refs, digest}); e != nil {
		return e
	}
	if c.Kind == Country || c.Kind == GeoArea || c.Kind == Location || c.Kind == Enterprise || c.Kind == LegalEntity {
		return nil
	}
	return saveReadSegments(ctx, tx, c, source, p.Values, readBytes)
}

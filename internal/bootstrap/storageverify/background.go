package storageverify

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/sync/errgroup"
)

func mapped(prefix, id string) string {
	h := md5.Sum([]byte(prefix + id))
	h[6] = (h[6] & 15) | 0x30
	h[8] = (h[8] & 15) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", h[:4], h[4:6], h[6:8], h[8:10], h[10:])
}
func uuid(s string) pgtype.UUID {
	var v pgtype.UUID
	if e := v.Scan(s); e != nil {
		panic(e)
	}
	return v
}
func uuids(ss []string) []pgtype.UUID {
	out := make([]pgtype.UUID, len(ss))
	for i, s := range ss {
		out[i] = uuid(s)
	}
	return out
}

type cloneTemplate struct {
	bundle              bundle
	chunk               history.Chunk
	currentShape        int32
	currentRefs         []int32
	dictHash, shapeHash []byte
	relation            string
	stable              map[string]bool
}

func loadTemplate(ctx context.Context, tx pgx.Tx, b bundle) (cloneTemplate, error) {
	t := cloneTemplate{bundle: b, stable: map[string]bool{b.Owner.Instance: true, b.Owner.Enterprise: true}}
	kind, prefix, table, key, object := history.BusinessUnit, "bu", "business_unit", "bu_id", b.Owner.BUID
	if b.Kind == "department" {
		kind, prefix, table, key, object = history.Department, "department", "department", "department_id", b.Owner.DepartmentID
	}
	var e error
	t.chunk, e = history.LoadChunk(ctx, tx, kind, b.Owner.Instance, object, 1)
	if e != nil {
		return t, e
	}
	if len(b.Frames) != 50 || t.chunk.Count() != 50 {
		return t, fmt.Errorf("background template must have exact 50-source chunk")
	}
	e = tx.QueryRow(ctx, fmt.Sprintf(`SELECT o.current_shape,o.current_refs,d.digest,s.digest FROM organization.%s o JOIN organization.%s_dictionary d ON d.id=$3 JOIN organization.%s_shape s ON s.id=o.current_shape WHERE o.instance_id=$1 AND o.%s=$2`, table, prefix, prefix, key), b.Owner.Instance, object, t.chunk.Dictionary).Scan(&t.currentShape, &t.currentRefs, &t.dictHash, &t.shapeHash)
	if e != nil {
		return t, e
	}
	frames, e := decodeFrames(b)
	if e != nil {
		return t, e
	}
	for _, f := range frames {
		if f.bu != nil {
			s := f.bu.Operation.Source
			t.stable[s.Subject] = true
		} else {
			s := f.dep.Operation.Source
			t.stable[s.Subject] = true
			t.relation = f.dep.State.Timeline.RelationSource
		}
	}
	return t, nil
}
func (t cloneTemplate) copy(o owner, protoBU string, index int) (history.Chunk, []byte, string, error) {
	c := t.chunk
	c.Object = o.BUID
	if c.Kind == history.Department {
		c.Object = o.DepartmentID
	}
	prefix := o.Code + ":"
	newBU := mapped(fmt.Sprintf("R6_B_%05d:", index%2000), protoBU)
	remap := func(id string) string {
		if t.stable[id] {
			return id
		}
		if id == protoBU {
			return newBU
		}
		return mapped(prefix, id)
	}
	if o.Instance != t.bundle.Owner.Instance || o.Enterprise != t.bundle.Owner.Enterprise || o.Head != remap(t.bundle.Owner.Head) || o.First != remap(t.bundle.Owner.First) {
		return c, nil, "", fmt.Errorf("original %s identity differs from seed mapping", o.Code)
	}
	originalObject := t.bundle.Owner.BUID
	if c.Kind == history.Department {
		originalObject = t.bundle.Owner.DepartmentID
	}
	if c.Object != remap(originalObject) {
		return c, nil, "", fmt.Errorf("original object mapping differs")
	}
	c.Values.IDs = make([]string, len(t.chunk.Values.IDs))
	for i, id := range t.chunk.Values.IDs {
		c.Values.IDs[i] = remap(id)
	}
	if e := c.Seal(); e != nil {
		return c, nil, "", e
	}
	currentIDs := make([]string, len(t.currentRefs))
	for i, r := range t.currentRefs {
		currentIDs[i] = c.ID(r)
	}
	digest, e := history.ProjectionDigest(c.Kind, c.Instance, c.Enterprise, c.Object, o.Head, currentIDs, t.dictHash, t.shapeHash)
	return c, digest, remap(t.relation), e
}
func importBackground(ctx context.Context, dir string) (any, error) {
	_, pool, e := openDatabase(ctx, dir)
	if e != nil {
		return nil, e
	}
	defer pool.Close()
	raw, e := os.ReadFile(filepath.Join(dir, "owners.json"))
	if e != nil {
		return nil, e
	}
	var owners map[string][]owner
	if e = json.Unmarshal(raw, &owners); e != nil {
		return nil, e
	}
	protoBU, e := readBundle(filepath.Join(dir, "bu-prototype.json"))
	if e != nil {
		return nil, e
	}
	out := map[string]any{}
	for _, kind := range []string{"bu", "dep"} {
		started := time.Now()
		b, e := readBundle(filepath.Join(dir, kind+"-prototype.json"))
		if e != nil {
			return nil, e
		}
		tx, e := pool.Begin(ctx)
		if e != nil {
			return nil, e
		}
		t, e := loadTemplate(ctx, tx, b)
		_ = tx.Rollback(ctx)
		if e != nil {
			return nil, e
		}
		prefix, table, key, tag := "bu", "business_unit", "bu_id", "R6_B_"
		if kind == "dep" {
			prefix, table, key, tag = "department", "department", "department_id", "R6_D_"
		}
		targets := []owner{}
		for _, o := range owners[kind] {
			if strings.HasPrefix(o.Code, tag) && len(o.Code) == 10 {
				targets = append(targets, o)
			}
		}
		expected := 2000
		if kind == "dep" {
			expected = 20000
		}
		if len(targets) != expected {
			return nil, fmt.Errorf("background owners %d want %d", len(targets), expected)
		}
		group, workContext := errgroup.WithContext(ctx)
		group.SetLimit(2)
		for start := 0; start < len(targets); start += 16 {
			start := start
			group.Go(func() error {
				ctx := workContext
				batch := targets[start:min(start+16, len(targets))]
				tx, e := pool.Begin(ctx)
				if e != nil {
					return e
				}
				work := func() error {
					rowsOwner, rowsHistory, rowsRelation := [][]any{}, [][]any{}, [][]any{}
					actors, submissions := []string{}, []string{}
					chunks := []history.Chunk{}
					for _, o := range batch {
						var index int
						if _, e := fmt.Sscanf(o.Code, tag+"%05d", &index); e != nil {
							return e
						}
						c, digest, relation, e := t.copy(o, protoBU.Owner.BUID, index)
						if e != nil {
							return e
						}
						var existing bool
						if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT FROM organization."+table+" WHERE instance_id=$1 AND "+key+"=$2)", o.Instance, c.Object).Scan(&existing); e != nil {
							return e
						}
						if existing {
							got, e := history.LoadChunk(ctx, tx, c.Kind, c.Instance, c.Object, 1)
							if e != nil {
								return e
							}
							if !reflect.DeepEqual(c, got) {
								return fmt.Errorf("existing task object %s differs", o.Code)
							}
							continue
						}
						chunks = append(chunks, c)
						row := []any{uuid(o.Instance), uuid(o.Enterprise), uuid(c.Object), o.Code, o.NormalizedCode, uuid(o.First), uuid(o.Head), int64(50), int32(1), t.currentShape, t.currentRefs, digest}
						if kind == "dep" {
							row = append(row, uuids([]string{mapped(fmt.Sprintf("R6_B_%05d:", index%2000), protoBU.Owner.BUID)}), []pgtype.UUID{})
							rowsRelation = append(rowsRelation, []any{uuid(o.Instance), uuid(o.Enterprise), uuid(c.Object), uuid(relation)})
						}
						rowsOwner = append(rowsOwner, row)
						rowsHistory = append(rowsHistory, []any{uuid(c.Instance), uuid(c.Enterprise), uuid(c.Object), c.Number, c.First, int16(1), uuids(c.Values.IDs), c.Dictionary, c.Sources, c.Subjects, c.Submissions, c.States, c.Operations, c.Evidence, c.Envelope, c.Sealed, c.EncodedBytes})
						for i := range c.Sources {
							actors = append(actors, c.ID(c.Subjects[i]))
							submissions = append(submissions, c.ID(c.Submissions[i]))
						}
					}
					if len(chunks) == 0 {
						return nil
					}
					if _, e = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(organization.submission_key($1,$2,a::uuid,s::uuid)) FROM unnest($3::text[],$4::text[]) x(a,s)`, prefix, b.Owner.Instance, actors, submissions); e != nil {
						return e
					}
					cols := []string{"instance_id", "enterprise_id", key, "code", "normalized_code", "first_record_id", "head_source_id", "head_no", "head_chunk", "current_shape", "current_refs", "current_envelope"}
					if kind == "dep" {
						cols = append(cols, "management_bus", "parents")
					}
					if _, e = tx.CopyFrom(ctx, pgx.Identifier{"organization", table}, cols, pgx.CopyFromRows(rowsOwner)); e != nil {
						return e
					}
					cols = []string{"instance_id", "enterprise_id", "object_id", "chunk_no", "first_no", "format", "ids", "dictionary_id", "source_refs", "subject_refs", "submission_refs", "states", "operations", "evidence", "envelope", "sealed", "encoded_bytes"}
					if _, e = tx.CopyFrom(ctx, pgx.Identifier{"organization", prefix + "_history"}, cols, pgx.CopyFromRows(rowsHistory)); e != nil {
						return e
					}
					if kind == "dep" {
						if _, e = tx.CopyFrom(ctx, pgx.Identifier{"organization", "department_hierarchy_relation"}, []string{"instance_id", "enterprise_id", "department_id", "head_source_id"}, pgx.CopyFromRows(rowsRelation)); e != nil {
							return e
						}
					}
					for _, c := range chunks {
						got, e := history.LoadChunk(ctx, tx, c.Kind, c.Instance, c.Object, 1)
						if e != nil {
							return e
						}
						if !reflect.DeepEqual(c, got) {
							return fmt.Errorf("stored full background identity directory differs")
						}
					}
					return nil
				}
				if e = work(); e != nil {
					_ = tx.Rollback(ctx)
					return e
				}
				if e = tx.Commit(ctx); e != nil {
					return e
				}
				if start%256 == 0 {
					fmt.Fprintln(os.Stderr, kind, "background", min(start+16, len(targets)), "of", len(targets), "elapsed", time.Since(started).Round(time.Second))
				}
				return nil
			})
		}
		if e := group.Wait(); e != nil {
			return nil, e
		}
		out[kind] = map[string]any{"objects": len(targets), "sources": len(targets) * 50, "allNativeIdentityPoolsRoundtripped": true, "seconds": time.Since(started).Seconds()}
	}
	return out, nil
}

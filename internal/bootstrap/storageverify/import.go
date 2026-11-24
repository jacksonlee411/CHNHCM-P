package storageverify

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"

	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	orgdb "chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres"
	"chnhcm.local/chnhcm/internal/platform/compact"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type frame struct {
	bu  *bootstrap.BUHistoryFact
	dep *bootstrap.DepartmentHistoryFact
}

func (f frame) source() (id, subject, submission string, no int64) {
	if f.bu != nil {
		s := f.bu.Operation.Source
		return s.ID, s.Subject, s.Submission, s.Number
	}
	s := f.dep.Operation.Source
	return s.ID, s.Subject, s.Submission, s.Number
}
func (f frame) encode(ctx context.Context, p *compact.Pool) (a, b, c []byte, e error) {
	if f.bu != nil {
		a, e = history.EncodeBUState(ctx, p, f.bu.State)
		if e == nil {
			b, e = history.EncodeBUOperation(ctx, p, f.bu.Operation)
		}
		return a, b, []byte{0}, e
	}
	a, e = history.EncodeDepartmentState(ctx, p, f.dep.State)
	if e != nil {
		return
	}
	b, e = history.EncodeDepartmentOperation(ctx, p, f.dep.Operation)
	if e != nil {
		return
	}
	c, e = history.EncodeDepartmentEvidence(ctx, p, f.dep.Evidence)
	return
}
func (f frame) projection(ctx context.Context, p *compact.Pool) ([]byte, error) {
	if f.bu != nil {
		return history.EncodeBUState(ctx, p, f.bu.State)
	}
	return history.EncodeDepartmentState(ctx, p, f.dep.State)
}
func decodeFrames(b bundle) ([]frame, error) {
	out := make([]frame, len(b.Frames))
	previous := ""
	for i, raw := range b.Frames {
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.DisallowUnknownFields()
		if b.Kind == "bu" {
			v := new(bootstrap.BUHistoryFact)
			if e := dec.Decode(v); e != nil {
				return nil, e
			}
			if e := orgdb.VerifyHistoryBU(v.State, v.Operation); e != nil {
				return nil, fmt.Errorf("%s source %d BU closure: %w", b.Owner.Code, i+1, e)
			}
			if v.Object != b.Owner.BUID || v.Instance != b.Owner.Instance || v.Enterprise != b.Owner.Enterprise || v.Operation.Source.Previous != previous || v.Operation.Source.Number != int64(i+1) {
				return nil, compact.ErrCorrupt
			}
			out[i].bu = v
			previous = v.Operation.Source.ID
		} else {
			v := new(bootstrap.DepartmentHistoryFact)
			if e := dec.Decode(v); e != nil {
				return nil, e
			}
			if e := orgdb.VerifyHistoryDepartment(v.State, v.Operation, v.Evidence); e != nil {
				return nil, fmt.Errorf("%s source %d department closure: %w", b.Owner.Code, i+1, e)
			}
			if v.Object != b.Owner.DepartmentID || v.Instance != b.Owner.Instance || v.Enterprise != b.Owner.Enterprise || v.Operation.Source.Previous != previous || v.Operation.Source.Number != int64(i+1) {
				return nil, compact.ErrCorrupt
			}
			out[i].dep = v
			previous = v.Operation.Source.ID
		}
	}
	if len(out) == 0 || previous != b.Owner.Head {
		return nil, compact.ErrCorrupt
	}
	return out, nil
}
func verifyStored(ctx context.Context, tx pgx.Tx, kind history.Kind, o owner, frames []frame) error {
	prefix, object := "bu", o.BUID
	if kind == history.Department {
		prefix, object = "department", o.DepartmentID
	}
	var count int
	if e := tx.QueryRow(ctx, "SELECT count(*) FROM organization."+prefix+"_history WHERE instance_id=$1 AND object_id=$2", o.Instance, object).Scan(&count); e != nil {
		return e
	}
	offset := 0
	for n := 1; n <= count; n++ {
		c, e := history.LoadChunk(ctx, tx, kind, o.Instance, object, int32(n))
		if e != nil {
			return e
		}
		for i := range c.Sources {
			a, e := history.LoadShape(ctx, tx, kind, o.Instance, o.Enterprise, c.States[i])
			if e != nil {
				return e
			}
			b, e := history.LoadShape(ctx, tx, kind, o.Instance, o.Enterprise, c.Operations[i])
			if e != nil {
				return e
			}
			if offset >= len(frames) {
				return compact.ErrCorrupt
			}
			expected := frames[offset]
			id, actor, submission, no := expected.source()
			if c.ID(c.Sources[i]) != id || c.ID(c.Subjects[i]) != actor || c.ID(c.Submissions[i]) != submission || c.First+int64(i) != no {
				return compact.ErrCorrupt
			}
			if kind == history.BusinessUnit {
				state, e := history.DecodeBUState(ctx, c.Values, a)
				if e != nil {
					return e
				}
				op, e := history.DecodeBUOperation(ctx, c.Values, b)
				if e != nil {
					return e
				}
				if !reflect.DeepEqual(state, expected.bu.State) || !reflect.DeepEqual(op, expected.bu.Operation) {
					return fmt.Errorf("%s source %d typed BU roundtrip differs", o.Code, no)
				}
			} else {
				state, e := history.DecodeDepartmentState(ctx, c.Values, a)
				if e != nil {
					return e
				}
				op, e := history.DecodeDepartmentOperation(ctx, c.Values, b)
				if e != nil {
					return e
				}
				raw, e := history.LoadShape(ctx, tx, kind, o.Instance, o.Enterprise, c.Evidence[i])
				if e != nil {
					return e
				}
				proof, e := history.DecodeDepartmentEvidence(ctx, c.Values, raw)
				if e != nil {
					return e
				}
				if !reflect.DeepEqual(state, expected.dep.State) || !reflect.DeepEqual(op, expected.dep.Operation) || !reflect.DeepEqual(proof, expected.dep.Evidence) {
					return fmt.Errorf("%s source %d typed department roundtrip differs", o.Code, no)
				}
			}
			offset++
		}
	}
	if offset != len(frames) {
		return compact.ErrCorrupt
	}
	return nil
}
func writeBundle(ctx context.Context, pool *pgxpool.Pool, b bundle) error {
	frames, e := decodeFrames(b)
	if e != nil {
		return e
	}
	kind, table, key, object := history.BusinessUnit, "business_unit", "bu_id", b.Owner.BUID
	if b.Kind == "department" {
		kind, table, key, object = history.Department, "department", "department_id", b.Owner.DepartmentID
	}
	if !compact.IsID(object) || b.Owner.NormalizedCode == "" {
		return compact.ErrCorrupt
	}
	tx, e := pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	var existing bool
	if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT FROM organization."+table+" WHERE instance_id=$1 AND "+key+"=$2)", b.Owner.Instance, object).Scan(&existing); e != nil {
		return e
	}
	if existing {
		var matches bool
		if e = tx.QueryRow(ctx, "SELECT enterprise_id=$3 AND code=$4 AND normalized_code=$5 AND first_record_id=$6 AND head_source_id=$7 FROM organization."+table+" WHERE instance_id=$1 AND "+key+"=$2", b.Owner.Instance, object, b.Owner.Enterprise, b.Owner.Code, b.Owner.NormalizedCode, b.Owner.First, b.Owner.Head).Scan(&matches); e != nil {
			return e
		}
		if !matches {
			return fmt.Errorf("existing verification object differs")
		}
		if e = verifyStored(ctx, tx, kind, b.Owner, frames); e != nil {
			return e
		}
		return tx.Commit(ctx)
	}
	first, _, _, _ := frames[0].source()
	_, e = tx.Exec(ctx, fmt.Sprintf(`INSERT INTO organization.%s(instance_id,enterprise_id,%s,code,normalized_code,first_record_id,head_source_id,head_no,head_chunk) VALUES($1,$2,$3,$4,$5,$6,$7,1,1)`, table, key), b.Owner.Instance, b.Owner.Enterprise, object, b.Owner.Code, b.Owner.NormalizedCode, b.Owner.First, first)
	if e != nil {
		return e
	}
	last := history.Chunk{Kind: kind, Instance: b.Owner.Instance, Enterprise: b.Owner.Enterprise, Object: object, Number: 1, First: 1}
	for _, f := range frames {
		source, subject, submission, no := f.source()
		namespace := "bu"
		if kind == history.Department {
			namespace = "department"
		}
		if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(organization.submission_key($1,$2,$3,$4))", namespace, b.Owner.Instance, subject, submission); e != nil {
			return e
		}

		last, e = history.Append(ctx, tx, last, no, source, subject, submission, func(p *compact.Pool) ([]byte, []byte, []byte, error) { return f.encode(ctx, p) })
		if e != nil {
			return fmt.Errorf("%s source %d append: %w", b.Owner.Code, no, e)
		}
	}
	_, e = tx.Exec(ctx, fmt.Sprintf(`UPDATE organization.%s SET head_source_id=$3,head_no=$4,head_chunk=$5 WHERE instance_id=$1 AND %s=$2`, table, key), b.Owner.Instance, object, b.Owner.Head, len(frames), last.Number)
	if e != nil {
		return e
	}
	final := frames[len(frames)-1]
	if kind == history.Department {
		t := final.dep.State.Timeline
		if _, e = tx.Exec(ctx, "INSERT INTO organization.department_hierarchy_relation VALUES($1,$2,$3,$4)", b.Owner.Instance, b.Owner.Enterprise, object, t.RelationSource); e != nil {
			return e
		}
		bus, parents := []string{}, []string{}
		for _, r := range t.Records {
			bus = append(bus, r.Data.BU)
			if r.Data.Parent != "" {
				parents = append(parents, r.Data.Parent)
			}
		}
		for _, r := range t.Edges {
			bus = append(bus, r.BU)
			if r.Parent != "" {
				parents = append(parents, r.Parent)
			}
		}
		slices.Sort(bus)
		bus = slices.Compact(bus)
		slices.Sort(parents)
		parents = slices.Compact(parents)
		if _, e = tx.Exec(ctx, "UPDATE organization.department SET management_bus=$3::text[]::uuid[],parents=$4::text[]::uuid[] WHERE instance_id=$1 AND department_id=$2", b.Owner.Instance, object, bus, parents); e != nil {
			return e
		}
	}
	if e = history.SaveProjection(ctx, tx, last, b.Owner.Head, func(p *compact.Pool) ([]byte, error) { return final.projection(ctx, p) }); e != nil {
		return e
	}
	if e = verifyStored(ctx, tx, kind, b.Owner, frames); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func importObjects(ctx context.Context, dir string, prototypeOnly bool) (any, error) {
	_, pool, e := openDatabase(ctx, dir)
	if e != nil {
		return nil, e
	}
	defer pool.Close()
	summary := map[string]any{}
	for _, kind := range []string{"bu", "dep"} {
		count, sources := 0, 0
		digest := sha256.New()
		consume := func(b bundle) error {
			if e := writeBundle(ctx, pool, b); e != nil {
				return e
			}
			count++
			sources += len(b.Frames)
			digest.Write([]byte(b.OriginalSHA256))
			if count%100 == 0 {
				fmt.Fprintln(os.Stderr, kind, "objects", count, "sources", sources)
			}
			return nil
		}
		if prototypeOnly {
			b, e := readBundle(filepath.Join(dir, kind+"-prototype.json"))
			if e != nil {
				return nil, e
			}
			if e = consume(b); e != nil {
				return nil, e
			}
		} else {
			f, e := os.Open(filepath.Join(dir, kind+"-typed.jsonl.gz"))
			if e != nil {
				return nil, e
			}
			gz, e := gzip.NewReader(f)
			if e != nil {
				f.Close()
				return nil, e
			}
			r := bufio.NewReader(gz)
			for {
				line, e := r.ReadBytes('\n')
				if len(line) > 0 {
					var b bundle
					if e := json.Unmarshal(line, &b); e != nil {
						return nil, e
					}
					if b.Owner.Code != "R6_PROTO_BU" && b.Owner.Code != "R6_PROTO_DEPT" {
						if e = consume(b); e != nil {
							return nil, e
						}
					}
				}
				if e == io.EOF {
					break
				}
				if e != nil {
					return nil, e
				}
			}
			gz.Close()
			f.Close()
		}
		summary[kind] = map[string]any{"objects": count, "sources": sources, "allTypedValuesRoundtripped": true, "originalGraphHashChain": hex.EncodeToString(digest.Sum(nil))}
	}
	return summary, nil
}

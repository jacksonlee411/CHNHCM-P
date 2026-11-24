// Package storageverify wires privileged offline case verification.
// It is not imported or registered by the product server.
// It is an offline verification tool, not a production migration or fallback.
package storageverify

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	orgdb "chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres"
	"chnhcm.local/chnhcm/internal/platform/compact"
)

type owner struct {
	Instance       string `json:"instance_id"`
	Enterprise     string `json:"enterprise_id"`
	BUID           string `json:"bu_id"`
	DepartmentID   string `json:"department_id"`
	Code           string `json:"code"`
	NormalizedCode string `json:"normalized_code"`
	First          string `json:"first_record_id"`
	Head           string `json:"head_source_id"`
}
type bundle struct {
	Kind              string
	Owner             owner
	Frames            []json.RawMessage
	OriginalRowCounts map[string]int
	OriginalSHA256    string
}

func readBundle(path string) (bundle, error) {
	f, e := os.Open(path)
	if e != nil {
		return bundle{}, e
	}
	defer f.Close()
	var b bundle
	d := json.NewDecoder(f)
	d.DisallowUnknownFields()
	e = d.Decode(&b)
	return b, e
}
func measureEncoding(ctx context.Context, b bundle) (any, error) {
	pool := compact.NewPool(compact.Values{})
	total := 0
	for i, raw := range b.Frames {
		var a, op, proof []byte
		var e error
		if b.Kind == "bu" {
			var v bootstrap.BUHistoryFact
			if e = json.Unmarshal(raw, &v); e != nil {
				return nil, e
			}
			if e = orgdb.VerifyHistoryBU(v.State, v.Operation); e != nil {
				return nil, fmt.Errorf("source %d BU closure: %w", i+1, e)
			}
			for _, id := range []string{v.Operation.Source.ID, v.Operation.Source.Subject, v.Operation.Source.Submission} {
				if _, e = pool.ID(id); e != nil {
					return nil, e
				}
			}
			a, e = history.EncodeBUState(ctx, pool, v.State)
			if e != nil {
				return nil, e
			}
			op, e = history.EncodeBUOperation(ctx, pool, v.Operation)
			if e != nil {
				return nil, e
			}
			proof = []byte{0}
		} else {
			var v bootstrap.DepartmentHistoryFact
			if e = json.Unmarshal(raw, &v); e != nil {
				return nil, e
			}
			if e = orgdb.VerifyHistoryDepartment(v.State, v.Operation, v.Evidence); e != nil {
				return nil, fmt.Errorf("source %d department closure: %w", i+1, e)
			}
			for _, id := range []string{v.Operation.Source.ID, v.Operation.Source.Subject, v.Operation.Source.Submission} {
				if _, e = pool.ID(id); e != nil {
					return nil, e
				}
			}
			a, e = history.EncodeDepartmentState(ctx, pool, v.State)
			if e != nil {
				return nil, e
			}
			op, e = history.EncodeDepartmentOperation(ctx, pool, v.Operation)
			if e != nil {
				return nil, e
			}
			proof, e = history.EncodeDepartmentEvidence(ctx, pool, v.Evidence)
			if e != nil {
				return nil, e
			}
		}
		total += len(a) + len(op) + len(proof)
	}
	return map[string]any{"kind": b.Kind, "sources": len(b.Frames), "uuidCount": len(pool.IDs), "uuidBytes": len(pool.IDs) * 16, "stringCount": len(pool.Strings), "binaryCount": len(pool.Binary), "rangeCount": len(pool.Ranges), "encodedStructureBytesBeforePGSharing": total, "scope": "typed prototype encoding only; not database size"}, nil
}
func Run(mode, input string) (any, error) {
	var out any
	var e error
	switch mode {
	case "frozen-tree":
		out, e = importFrozenTree(context.Background(), input)
	case "read-index":
		var db privateDatabase
		db, pool, err := openDatabase(context.Background(), input, true)
		if err != nil {
			return nil, err
		}
		defer pool.Close()
		return bootstrap.RebuildCurrentReadIndexes(context.Background(), pool, db.Instance)
	case "plans":
		out, e = explainReads(context.Background(), input)
	case "serve", "serve-web":
		out, e = serveDatabase(context.Background(), input, mode == "serve-web")
	case "size":
		out, e = databaseSize(context.Background(), input)
	case "maintain", "compact", "maintain-dry-run":
		out, e = maintainDatabase(context.Background(), input, mode == "compact", mode == "maintain-dry-run")
	case "background":
		out, e = importBackground(context.Background(), input)
	case "prototypes":
		out, e = importObjects(context.Background(), input, true)
	case "extras":
		out, e = importObjects(context.Background(), input, false)
	case "prepare":
		out, e = prepareDatabase(context.Background(), input)
	case "auxiliary":
		out, e = importAuxiliary(context.Background(), input)
	case "encode":
		var b bundle
		b, e = readBundle(input)
		if e == nil {
			out, e = measureEncoding(context.Background(), b)
		}
	default:
		e = fmt.Errorf("unknown operation")
	}
	return out, e
}

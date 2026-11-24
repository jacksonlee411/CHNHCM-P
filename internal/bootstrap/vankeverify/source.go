package vankeverify

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	organization "chnhcm.local/chnhcm/internal/contexts/organization/domain"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5/pgxpool"
)

func auditSource(ctx context.Context, pool *pgxpool.Pool, dir string) (any, error) {
	rows, e := pool.Query(ctx, "SELECT phase,row_no,payload,result FROM verification.source_row ORDER BY phase,row_no")
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	file, e := os.Open(filepath.Join(dir, "source-rows.jsonl"))
	if e != nil {
		return nil, e
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 65536), 1<<20)
	hash := sha256.New()
	counts := map[string]int{}
	outcomes := map[string]int{}
	invalid := map[string]int{}
	normalized := map[string]int{}
	examples := map[string][]int{}
	for rows.Next() {
		var phase string
		var no int
		var raw, result []byte
		if e = rows.Scan(&phase, &no, &raw, &result); e != nil {
			return nil, e
		}
		var row map[string]string
		if e = json.Unmarshal(raw, &row); e != nil {
			return nil, e
		}
		if !scanner.Scan() {
			return nil, errors.New("source archive row count mismatch")
		}
		var expected struct {
			Phase   string
			Row     int
			Payload map[string]string
			Result  any
		}
		if e = json.Unmarshal(scanner.Bytes(), &expected); e != nil {
			return nil, e
		}
		a, _ := json.Marshal(row)
		b, _ := json.Marshal(expected.Payload)
		if phase != expected.Phase || no != expected.Row || string(a) != string(b) {
			return nil, fmt.Errorf("source archive mismatch at %s/%d", phase, no)
		}
		var actualResult map[string]any
		if len(result) > 0 {
			if e = json.Unmarshal(result, &actualResult); e != nil {
				return nil, e
			}
		}
		aResult, _ := json.Marshal(actualResult)
		bResult, _ := json.Marshal(expected.Result)
		if string(aResult) != string(bResult) {
			return nil, fmt.Errorf("source result mapping mismatch at %s/%d", phase, no)
		}
		if outcome, ok := actualResult["outcome"].(string); ok {
			outcomes[phase+"."+outcome]++
		}
		fmt.Fprintf(hash, "%s/%d/%s\n", phase, no, a)
		counts[phase]++
		if _, e = organization.NormalizeCode(row["org_code"]); e != nil {
			invalid[phase+".code"]++
		}
		name := row["name"]
		if phase == "version" {
			name = row["core.name"]
		}
		if name != "" {
			n, e := organization.NormalizeName(name, true)
			if e != nil {
				key := phase + ".name"
				invalid[key]++
				examples[key] = append(examples[key], no)
			} else if n != name {
				normalized[phase+".name"]++
			}
		}
		day := row["effective_date"]
		if row["operation"] == "correct_parent" {
			day = row["target_effective_date"]
		}
		if !foundation.ValidDate(day) {
			invalid[phase+".date"]++
		}
	}
	if e = rows.Err(); e != nil {
		return nil, e
	}
	if scanner.Scan() {
		return nil, errors.New("source archive has extra rows")
	}
	if e = scanner.Err(); e != nil {
		return nil, e
	}
	return map[string]any{"roundtripAllSourceCells": true, "roundtripStoredResultMappings": true, "canonicalSHA256": hex.EncodeToString(hash.Sum(nil)), "counts": counts, "storedOutcomes": outcomes, "invalidExplicitInputs": invalid, "normalizedExplicitInputs": normalized, "invalidRowNumbers": examples, "scope": "original source cells and available import results compared; historical product facts require separate verification"}, nil
}

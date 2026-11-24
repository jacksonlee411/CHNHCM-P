//go:build integration

package bootstrap_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"testing"
	"time"
)

// Independent framing oracle: identical digest bytes are required for retained
// dictionaries/shapes, including empty values, binary bytes and Unicode text.
func hash133Parts(parts [][]byte) []byte {
	raw := binary.BigEndian.AppendUint32(nil, uint32(len(parts)))
	for _, p := range parts {
		raw = binary.BigEndian.AppendUint32(raw, uint32(len(p)))
		raw = append(raw, p...)
	}
	h := sha256.Sum256(raw)
	return h[:]
}

func TestHistory133SQLHashesPreserveExactFraming(t *testing.T) {
	f := fresh(t)
	for _, n := range []int{0, 100, 1000, 10000} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			words, binaryParts, ranges := []string{}, [][]byte{}, []string{}
			wordParts, rangeParts := [][]byte{}, [][]byte{}
			for i := 0; i < n; i++ {
				word := fmt.Sprintf("部门 É %d", i)
				if i%7 == 0 {
					word = ""
				}
				words = append(words, word)
				wordParts = append(wordParts, []byte(word))
				binaryParts = append(binaryParts, []byte{0, byte(i), 255})
				if i%3 == 0 {
					ranges = append(ranges, "empty")
					rangeParts = append(rangeParts, []byte{}, []byte{})
				} else {
					ranges = append(ranges, "[2020-01-01,2021-01-01)")
					rangeParts = append(rangeParts, []byte("2020-01-01"), []byte("2021-01-01"))
				}
			}
			parts := [][]byte{binary.BigEndian.AppendUint32(nil, uint32(n))}
			parts = append(parts, wordParts...)
			parts = append(parts, binary.BigEndian.AppendUint32(nil, uint32(n)))
			parts = append(parts, binaryParts...)
			parts = append(parts, binary.BigEndian.AppendUint32(nil, uint32(n)))
			parts = append(parts, rangeParts...)
			var got, direct []byte
			start := time.Now()
			must(t, f.admin.QueryRow(f.ctx, `SELECT organization.history_dictionary_hash($1::text[],$2::bytea[],$3::text[]::daterange[])`, words, binaryParts, ranges).Scan(&got))
			t.Logf("dictionary items per kind=%d query=%s", n, time.Since(start))
			must(t, f.admin.QueryRow(f.ctx, `SELECT organization.history_parts_hash($1::bytea[])`, parts).Scan(&direct))
			if !bytes.Equal(got, hash133Parts(parts)) || !bytes.Equal(got, direct) {
				t.Fatal("native digest changed framing")
			}
		})
	}
	var v []byte
	must(t, f.admin.QueryRow(f.ctx, `SELECT organization.history_parts_hash(ARRAY['a'::bytea,NULL::bytea])`).Scan(&v))
	if v != nil {
		t.Fatal("null part no longer poisons the original framing")
	}
}

func TestHistory133SQLPartBoundsAndSinglePartFraming(t *testing.T) {
	f := fresh(t)
	for _, tc := range []struct {
		name, expression string
		parts            [][]byte
		valid, nullHash  bool
	}{
		{"empty", "'{}'::bytea[]", [][]byte{}, false, false},
		{"single", "ARRAY[decode('00ff','hex')]", [][]byte{{0, 255}}, true, false},
		{"empty_part", "ARRAY[''::bytea]", [][]byte{{}}, false, false},
		{"null_part", "ARRAY[NULL::bytea]", nil, false, true},
		{"null_in_many", "ARRAY['a'::bytea,NULL::bytea]", nil, false, true},
		{"lower_bound", "array_fill('a'::bytea,ARRAY[1],ARRAY[0])", [][]byte{[]byte("a")}, false, false},
		{"single_multidimensional", "array_fill('a'::bytea,ARRAY[1,1])", [][]byte{[]byte("a")}, false, false},
		{"multidimensional", "ARRAY[ARRAY['a'::bytea,'b'::bytea],ARRAY['c'::bytea,'d'::bytea]]", [][]byte{[]byte("a"), []byte("b"), []byte("c"), []byte("d")}, false, false},
		{"maximum_part", "ARRAY[convert_to(repeat('x',262144),'UTF8')]", [][]byte{bytes.Repeat([]byte("x"), 262144)}, true, false},
		{"oversize_part", "ARRAY[convert_to(repeat('x',262145),'UTF8')]", [][]byte{bytes.Repeat([]byte("x"), 262145)}, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var digest []byte
			var valid bool
			must(t, f.admin.QueryRow(f.ctx, `SELECT organization.history_parts_hash(p),organization.history_parts_valid(p) FROM (SELECT `+tc.expression+` p) input`).Scan(&digest, &valid))
			if valid != tc.valid || tc.nullHash && digest != nil || !tc.nullHash && !bytes.Equal(digest, hash133Parts(tc.parts)) {
				t.Fatal("changed part framing or admission", valid, digest)
			}
		})
	}
	for _, count := range []int{256, 257} {
		var valid bool
		must(t, f.admin.QueryRow(f.ctx, `SELECT organization.history_parts_valid(array_fill('a'::bytea,ARRAY[$1::integer]))`, count).Scan(&valid))
		if valid != (count == 256) {
			t.Fatal("part count boundary changed", count, valid)
		}
	}
	var nullHash, nullValid bool
	must(t, f.admin.QueryRow(f.ctx, `SELECT organization.history_parts_hash(NULL::bytea[]) IS NULL,organization.history_parts_valid(NULL::bytea[]) IS NULL`).Scan(&nullHash, &nullValid))
	if !nullHash || !nullValid {
		t.Fatal("strict null behavior changed")
	}
}

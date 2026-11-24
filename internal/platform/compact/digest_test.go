package compact

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"strings"
	"testing"
)

// The original envelope framing is a persisted contract. Reusing varint
// buffers must retain exact bytes, including multi-byte lengths and counts.
func TestDigestKeepsOriginalEnvelopeFraming(t *testing.T) {
	for _, n := range []int{0, 1, 127, 128, 255, 1024} {
		v := Values{Strings: make([]string, n), Binary: [][]byte{nil, {}, bytes.Repeat([]byte{0, 255}, 200)}, Ranges: []Range{{Start: "2020-01-01", End: ""}}}
		for i := range v.Strings {
			v.Strings[i] = strings.Repeat("名称", 1+i%100)
		}
		owner := []string{"department", "instance", "object", "source"}
		roots := [][]byte{{0, 1, 2}, bytes.Repeat([]byte{3}, 128)}
		h := sha256.New()
		put := func(b []byte) { h.Write(binary.AppendUvarint(nil, uint64(len(b)))); h.Write(b) }
		put([]byte("CHNHCM compact envelope 1"))
		for _, g := range [][]string{owner, v.IDs, v.Strings} {
			put(binary.AppendUvarint(nil, uint64(len(g))))
			for _, s := range g {
				put([]byte(s))
			}
		}
		put(binary.AppendUvarint(nil, uint64(len(v.Binary))))
		for _, b := range v.Binary {
			put(b)
		}
		put(binary.AppendUvarint(nil, uint64(len(v.Ranges))))
		for _, r := range v.Ranges {
			put([]byte(r.Start))
			put([]byte(r.End))
		}
		put(binary.AppendUvarint(nil, uint64(len(roots))))
		for _, b := range roots {
			put(b)
		}
		got, err := Digest(owner, v, roots)
		if err != nil || !bytes.Equal(got, h.Sum(nil)) {
			t.Fatal("stored envelope changed", n, err)
		}
	}
}

// Package compact provides the physical primitives used by generated, typed
// storage codecs. It knows no objects, permissions, commands or business dates.
package compact

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

const MaxPart = 256 * 1024
const MaxItems = 1 << 20

var ErrCorrupt = errors.New("invalid compact storage content")

// Values are immutable within a committed prefix. UUIDs are stored as their
// complete 128 bits; reference numbers are local, never public identities.
type Values struct {
	IDs     []string
	Strings []string
	Binary  [][]byte
	Ranges  []Range
}
type Range struct{ Start, End string }

type Pool struct {
	Values
	ids, strings, binary map[string]uint64
	ranges               map[Range]uint64
}

// ResetIDs begins another local UUID directory while retaining the same value
// dictionary. Callers must finish consuming the previous directory first.
func (p *Pool) ResetIDs() {
	p.IDs = p.IDs[:0]
	clear(p.ids)
}

func NewPool(v Values) *Pool {
	p := &Pool{Values: Values{append([]string{}, v.IDs...), append([]string{}, v.Strings...), append([][]byte{}, v.Binary...), append([]Range{}, v.Ranges...)}, ids: map[string]uint64{}, strings: map[string]uint64{}, binary: map[string]uint64{}, ranges: map[Range]uint64{}}
	for i, x := range p.IDs {
		p.ids[x] = uint64(i + 1)
	}
	for i, x := range p.Strings {
		p.strings[x] = uint64(i + 1)
	}
	for i, x := range p.Binary {
		p.binary[string(x)] = uint64(i + 1)
	}
	for i, x := range p.Ranges {
		p.ranges[x] = uint64(i + 1)
	}
	return p
}

func IsID(s string) bool {
	if len(s) != 36 || s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue
		}
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func (p *Pool) ID(s string) (uint32, error) {
	if !IsID(s) {
		return 0, ErrCorrupt
	}
	if n, ok := p.ids[s]; ok {
		return uint32(n), nil
	}
	if len(p.IDs) >= MaxItems {
		return 0, ErrCorrupt
	}
	p.IDs = append(p.IDs, s)
	n := uint64(len(p.IDs))
	p.ids[s] = n
	return uint32(n), nil
}

type Writer struct {
	Pool    *Pool
	Data    []byte
	Err     error
	Context context.Context
	work    int
}

func NewWriter(ctx context.Context, p *Pool, format uint64) *Writer {
	w := &Writer{Pool: p, Context: ctx}
	w.Uint(format)
	return w
}
func (w *Writer) tick() {
	w.work++
	if w.work&255 == 0 && w.Err == nil {
		w.Err = w.Context.Err()
	}
}
func (w *Writer) Uint(n uint64) { w.tick(); w.Data = binary.AppendUvarint(w.Data, n) }
func (w *Writer) Int(n int64)   { w.tick(); w.Data = binary.AppendVarint(w.Data, n) }
func (w *Writer) Bool(b bool) {
	if b {
		w.Uint(1)
	} else {
		w.Uint(0)
	}
}
func (w *Writer) String(s string) {
	if s == "" {
		w.Uint(0)
		return
	}
	if IsID(s) {
		n, e := w.Pool.ID(s)
		if e != nil {
			w.Err = e
			return
		}
		w.Uint(uint64(n)<<1 | 1)
		return
	}
	n, ok := w.Pool.strings[s]
	if !ok {
		if len(w.Pool.Strings) >= MaxItems {
			w.Err = ErrCorrupt
			return
		}
		w.Pool.Strings = append(w.Pool.Strings, s)
		n = uint64(len(w.Pool.Strings))
		w.Pool.strings[s] = n
	}
	w.Uint(n << 1)
}
func (w *Writer) Bytes(b []byte) {
	if b == nil {
		w.Uint(0)
		return
	}
	n, ok := w.Pool.binary[string(b)]
	if !ok {
		if len(w.Pool.Binary) >= MaxItems {
			w.Err = ErrCorrupt
			return
		}
		w.Pool.Binary = append(w.Pool.Binary, bytes.Clone(b))
		n = uint64(len(w.Pool.Binary))
		w.Pool.binary[string(b)] = n
	}
	w.Uint(n)
}
func (w *Writer) Range(start, end string) {
	v := Range{start, end}
	n, ok := w.Pool.ranges[v]
	if !ok {
		if len(w.Pool.Ranges) >= MaxItems {
			w.Err = ErrCorrupt
			return
		}
		w.Pool.Ranges = append(w.Pool.Ranges, v)
		n = uint64(len(w.Pool.Ranges))
		w.Pool.ranges[v] = n
	}
	w.Uint(n)
}
func (w *Writer) Finish() ([]byte, error) {
	if w.Err != nil {
		return nil, w.Err
	}
	if e := w.Context.Err(); e != nil {
		return nil, e
	}
	return w.Data, nil
}

type Reader struct {
	Values  Values
	Data    []byte
	Err     error
	Context context.Context
	work    int
}

func NewReader(ctx context.Context, v Values, data []byte, format uint64) *Reader {
	r := &Reader{Values: v, Data: data, Context: ctx}
	if r.Uint() != format {
		r.Err = ErrCorrupt
	}
	return r
}
func (r *Reader) tick() {
	r.work++
	if r.work&255 == 0 && r.Err == nil {
		r.Err = r.Context.Err()
	}
}
func (r *Reader) Uint() uint64 {
	r.tick()
	if r.Err != nil {
		return 0
	}
	n, size := binary.Uvarint(r.Data)
	if size <= 0 {
		r.Err = ErrCorrupt
		return 0
	}
	// Reject alternate encodings: a source has exactly one canonical encoding.
	if size > 1 && r.Data[size-1] == 0 {
		r.Err = ErrCorrupt
		return 0
	}
	r.Data = r.Data[size:]
	return n
}
func (r *Reader) Int() int64 { n := r.Uint(); return int64(n>>1) ^ -int64(n&1) }
func (r *Reader) Bool() bool {
	n := r.Uint()
	if n > 1 {
		r.Err = ErrCorrupt
	}
	return n == 1
}
func (r *Reader) Len() int {
	n := r.Uint()
	if n > MaxItems || n > uint64(len(r.Data))+1 {
		r.Err = ErrCorrupt
		return 0
	}
	return int(n)
}
func (r *Reader) String() string {
	n := r.Uint()
	if n == 0 {
		return ""
	}
	i := n >> 1
	if i == 0 {
		r.Err = ErrCorrupt
		return ""
	}
	if n&1 != 0 {
		if i > uint64(len(r.Values.IDs)) {
			r.Err = ErrCorrupt
			return ""
		}
		return r.Values.IDs[i-1]
	}
	if i > uint64(len(r.Values.Strings)) {
		r.Err = ErrCorrupt
		return ""
	}
	return r.Values.Strings[i-1]
}
func (r *Reader) Bytes() []byte {
	n := r.Uint()
	if n == 0 {
		return nil
	}
	if n > uint64(len(r.Values.Binary)) {
		r.Err = ErrCorrupt
		return nil
	}
	return bytes.Clone(r.Values.Binary[n-1])
}
func (r *Reader) Range() (string, string) {
	n := r.Uint()
	if n == 0 || n > uint64(len(r.Values.Ranges)) {
		r.Err = ErrCorrupt
		return "", ""
	}
	v := r.Values.Ranges[n-1]
	return v.Start, v.End
}
func (r *Reader) Finish() error {
	if r.Err != nil {
		return r.Err
	}
	if len(r.Data) != 0 {
		return ErrCorrupt
	}
	return r.Context.Err()
}

// Digest binds a frame to its owner and complete dictionary. Content hashes
// alone cannot distinguish swapping two valid pools or two valid owners.
func Digest(owner []string, v Values, roots [][]byte) ([]byte, error) {
	h := sha256.New()
	var length, count [10]byte
	put := func(b []byte) {
		n := binary.PutUvarint(length[:], uint64(len(b)))
		_, _ = h.Write(length[:n])
		_, _ = h.Write(b)
	}
	putCount := func(n int) { size := binary.PutUvarint(count[:], uint64(n)); put(count[:size]) }
	put([]byte("CHNHCM compact envelope 1"))
	for _, group := range [][]string{owner, v.IDs, v.Strings} {
		putCount(len(group))
		for _, s := range group {
			put([]byte(s))
		}
	}
	putCount(len(v.Binary))
	for _, b := range v.Binary {
		put(b)
	}
	putCount(len(v.Ranges))
	for _, v := range v.Ranges {
		put([]byte(v.Start))
		put([]byte(v.End))
	}
	putCount(len(roots))
	for _, b := range roots {
		put(b)
	}
	return h.Sum(nil), nil
}

func ValidateValues(v Values) error {
	if len(v.IDs) > MaxItems || len(v.Strings) > MaxItems || len(v.Binary) > MaxItems || len(v.Ranges) > MaxItems {
		return ErrCorrupt
	}
	seen := map[string]bool{}
	for _, s := range v.IDs {
		if !IsID(s) || seen[s] {
			return ErrCorrupt
		}
		seen[s] = true
	}
	seen = map[string]bool{}
	for _, s := range v.Strings {
		if s == "" || IsID(s) || seen[s] {
			return ErrCorrupt
		}
		seen[s] = true
	}
	seen = map[string]bool{}
	for _, b := range v.Binary {
		if b == nil || seen[string(b)] {
			return ErrCorrupt
		}
		seen[string(b)] = true
	}
	seen = map[string]bool{}
	for _, v := range v.Ranges {
		key := v.Start + "/" + v.End
		if seen[key] {
			return ErrCorrupt
		}
		seen[key] = true
	}
	return nil
}

func Tag(id string) (int32, error) {
	if !IsID(id) {
		return 0, fmt.Errorf("submission identity: %w", ErrCorrupt)
	}
	b, e := hex.DecodeString(strings.ReplaceAll(id, "-", ""))
	if e != nil {
		return 0, e
	}
	h := sha256.Sum256(b)
	return int32(binary.BigEndian.Uint16(h[:2])), nil
}

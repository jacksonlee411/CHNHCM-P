package department

import (
	"context"
	"crypto/sha256"
	"slices"
	"strings"

	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
)

type nodeDigestKey struct{ department, source, record, edge string }

// ListFacts indexes unqualified immutable inputs only. It contains no date,
// composed path, status, field mask, principal or authorization result.
type ListFacts struct {
	nodes     map[nodeDigestKey][32]byte
	codeOrder []int
}

func nodeKey(t *d.Timeline, r *d.Record, e *d.Edge) nodeDigestKey {
	return nodeDigestKey{t.ID, t.Source, r.ID, e.ID}
}
func nodeDigest(t *d.Timeline, r *d.Record, e *d.Edge) [32]byte {
	var b digestBytes
	for _, s := range []string{"department-node-input/1", "department", t.ID, r.Data.Name, t.Source, r.ID, r.Revision, e.Ref.DepartmentID, e.Ref.Source, e.Ref.MemberID, e.Ref.BU} {
		b.text(s)
	}
	return sha256.Sum256(b)
}
func NewListFacts(ctx context.Context, s d.Snapshot) (*ListFacts, error) {
	v := &ListFacts{nodes: make(map[nodeDigestKey][32]byte, len(s.Departments)), codeOrder: make([]int, len(s.Departments))}
	for i := range s.Departments {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		t := &s.Departments[i]
		v.codeOrder[i] = i
		for j := range t.Records {
			r := &t.Records[j]
			for k := range t.Edges {
				e := &t.Edges[k]
				if (r.End == "" || e.Start < r.End) && (e.End == "" || r.Date < e.End) {
					v.nodes[nodeKey(t, r, e)] = nodeDigest(t, r, e)
				}
			}
		}
	}
	slices.SortFunc(v.codeOrder, func(a, b int) int {
		x, y := &s.Departments[a], &s.Departments[b]
		if n := strings.Compare(x.Code, y.Code); n != 0 {
			return n
		}
		return strings.Compare(x.ID, y.ID)
	})
	return v, nil
}
func (v *ListFacts) RetainedBytes() int {
	if v == nil {
		return 0
	}
	return len(v.nodes)*192 + len(v.codeOrder)*8
}

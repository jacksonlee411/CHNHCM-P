package department

import (
	"context"
	"fmt"
	"reflect"
	"slices"

	"chnhcm.local/chnhcm/internal/contexts/organization/domain/dated"
)

type PathNode struct {
	Kind, DepartmentID, Source, RecordID, Revision, Name string
	Edge                                                 MemberRef
}
type PathSegment struct {
	RecordID, Revision, Start, End, Mode, BusinessDate string
	Nodes                                              []PathNode
}

func PathAt(ctx context.Context, s Snapshot, id, date, enterpriseName string) ([]PathNode, error) {
	return NewPathResolver(s, enterpriseName).At(ctx, id, date)
}

type pathKey struct{ id, date string }

// PathLink shares a validated immutable ancestor prefix within one request.
// Consumers that need only an aggregate need not allocate every full node list.
type PathLink struct {
	Node   PathNode
	Parent *PathLink
	Length int
}

// PathStep borrows one validated immutable node. Aggregate list consumers do
// not need a second complete PathNode/MemberRef allocation for every row.
type PathStep struct {
	Timeline *Timeline
	Record   *Record
	Edge     *Edge
}

func ReadPathStep(s Snapshot, id, date string) (PathStep, error) {
	t := s.Department(id)
	if t == nil {
		return PathStep{}, fmt.Errorf("missing department path node")
	}
	r := recordAt(t.Records, date)
	if r == nil || r.State != "active" {
		return PathStep{}, fmt.Errorf("missing active department path fact")
	}
	for i := range t.Edges {
		e := &t.Edges[i]
		if e.Start <= date && (e.End == "" || date < e.End) {
			if e.BU != r.Data.BU || e.Parent != r.Data.Parent {
				return PathStep{}, fmt.Errorf("department path relation differs from facts")
			}
			return PathStep{t, r, e}, nil
		}
	}
	return PathStep{}, fmt.Errorf("department path relation differs from facts")
}

// PathResolver belongs to one immutable request snapshot. It caches only raw
// path facts for that request; each caller separately applies current rights.
// Returned paths are immutable, including the shared ancestor prefix.
type PathResolver struct {
	s        Snapshot
	paths    map[pathKey][]PathNode
	links    map[pathKey]*PathLink
	root     *PathLink
	visiting map[pathKey]bool
}

func NewPathResolver(s Snapshot, enterprise string) *PathResolver {
	return &PathResolver{s: s, paths: map[pathKey][]PathNode{}, links: map[pathKey]*PathLink{}, root: &PathLink{Node: PathNode{Kind: "enterprise", Name: enterprise}, Length: 1}, visiting: map[pathKey]bool{}}
}

func (p *PathResolver) At(ctx context.Context, id, date string) ([]PathNode, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	k := pathKey{id, date}
	if nodes, ok := p.paths[k]; ok {
		return nodes, nil
	}
	link, err := p.Link(ctx, id, date)
	if err != nil {
		return nil, err
	}
	nodes := make([]PathNode, link.Length)
	for at := link; at != nil; at = at.Parent {
		nodes[at.Length-1] = at.Node
	}
	p.paths[k] = nodes
	return nodes, nil
}

func (p *PathResolver) Link(ctx context.Context, id, date string) (*PathLink, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if id == "" {
		return p.root, nil
	}
	k := pathKey{id, date}
	if link, ok := p.links[k]; ok {
		return link, nil
	}
	if p.visiting[k] {
		return nil, fmt.Errorf("cyclic stored department path")
	}
	p.visiting[k] = true
	defer delete(p.visiting, k)
	step, err := ReadPathStep(p.s, id, date)
	if err != nil {
		return nil, err
	}
	t, r, edge := step.Timeline, step.Record, step.Edge
	parent, err := p.Link(ctx, edge.Parent, date)
	if err != nil {
		return nil, err
	}
	link := &PathLink{Node: PathNode{Kind: "department", DepartmentID: id, Source: t.Source, RecordID: r.ID, Revision: r.Revision, Name: r.Data.Name, Edge: edge.Ref}, Parent: parent, Length: parent.Length + 1}
	p.links[k] = link
	return link, nil
}

// Evidence is computed from a single locked snapshot with the final target
// overlaid. Every source member, including reused revisions, gets exact evidence.
func Evidence(ctx context.Context, s Snapshot, t Timeline, enterpriseName string) ([]PathSegment, error) {
	out := []PathSegment{}
	paths := NewPathResolver(s, enterpriseName)
	deps := Discover(s, t.ID, nil, nil)
	boundaries := []string{}
	for _, id := range deps.Departments {
		if x := s.Department(id); x != nil {
			for _, r := range x.Records {
				boundaries = append(boundaries, r.Date)
			}
			for _, e := range x.Edges {
				boundaries = append(boundaries, e.Start)
				if e.End != "" {
					boundaries = append(boundaries, e.End)
				}
			}
		}
	}
	slices.Sort(boundaries)
	boundaries = slices.Compact(boundaries)
	for _, r := range t.Records {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if r.State == "inactive" {
			day := dated.NextDate(r.Date, -1)
			nodes, e := paths.At(ctx, t.ID, day)
			if e != nil {
				return nil, e
			}
			out = append(out, PathSegment{RecordID: r.ID, Revision: r.Revision, Start: r.Date, End: r.End, Mode: "inactive_before_stop", BusinessDate: day, Nodes: nodes})
			continue
		}
		dates := []string{r.Date}
		for _, d := range boundaries {
			if d > r.Date && (r.End == "" || d < r.End) {
				dates = append(dates, d)
			}
		}
		for i, date := range dates {
			end := r.End
			if i+1 < len(dates) {
				end = dates[i+1]
			}
			nodes, e := paths.At(ctx, t.ID, date)
			if e != nil {
				return nil, e
			}
			n := len(out)
			if n > 0 && out[n-1].RecordID == r.ID && out[n-1].End == date && reflect.DeepEqual(out[n-1].Nodes, nodes) {
				out[n-1].End = end
				continue
			}
			out = append(out, PathSegment{RecordID: r.ID, Revision: r.Revision, Start: date, End: end, Mode: "active", Nodes: nodes})
		}
	}
	return out, nil
}

// ValidateEvidence checks completeness and exact links without deriving an old
// path from today's ancestor heads. FK enforcement alone cannot detect missing
// nodes or missing segments after storage corruption.
func ValidateEvidence(t Timeline, r Record, segments []PathSegment) error {
	bad := func() error { return fmt.Errorf("incomplete or inconsistent department path evidence") }
	if len(segments) == 0 {
		return bad()
	}
	next := r.Date
	for _, seg := range segments {
		if seg.RecordID != r.ID || seg.Revision != r.Revision || seg.Start != next || (seg.End != "" && seg.End <= seg.Start) || len(seg.Nodes) < 2 {
			return bad()
		}
		next = seg.End
		day := seg.Start
		if r.State == "inactive" {
			day = dated.NextDate(r.Date, -1)
			if seg.Mode != "inactive_before_stop" || seg.BusinessDate != day || len(segments) != 1 {
				return bad()
			}
		} else if seg.Mode != "active" || seg.BusinessDate != "" {
			return bad()
		}
		if seg.Nodes[0].Kind != "enterprise" || seg.Nodes[0].DepartmentID != "" {
			return bad()
		}
		visited := map[string]bool{}
		parent := ""
		for _, node := range seg.Nodes[1:] {
			ref := node.Edge
			if node.Kind != "department" || node.DepartmentID == "" || node.Source == "" || node.RecordID == "" || node.Revision == "" || node.Name == "" || visited[node.DepartmentID] || ref.DepartmentID != node.DepartmentID || ref.Source == "" || ref.MemberID == "" || ref.BU != r.Data.BU || ref.Parent != parent || ref.Start > day || (ref.End != "" && ref.End <= day) {
				return bad()
			}
			if r.State == "active" && ref.End != "" && (seg.End == "" || ref.End < seg.End) {
				return bad()
			}
			visited[node.DepartmentID] = true
			parent = node.DepartmentID
		}
		leaf := seg.Nodes[len(seg.Nodes)-1]
		if leaf.DepartmentID != t.ID || leaf.Source != t.Source {
			return bad()
		}
		if r.State == "active" && (leaf.RecordID != r.ID || leaf.Revision != r.Revision) {
			return bad()
		}
		if r.State == "inactive" && (leaf.RecordID != r.InheritedRecord || leaf.Revision != r.InheritedRevision) {
			return bad()
		}
	}
	if next != r.End {
		return bad()
	}
	return nil
}

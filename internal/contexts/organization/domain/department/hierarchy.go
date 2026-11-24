package department

import (
	"context"
	"fmt"
	"slices"
)

// Dependencies include former/future parents and reverse children. These stable
// identities are discovered before taking locks, then checked again under guards.
type Dependencies struct{ BUs, Departments []string }

// HasIncomingReferences examines every dated record in the discovered snapshot,
// including inactive and former/future parent links. Callers must lock the target
// and recheck the complete dependency set before relying on an empty result.
func HasIncomingReferences(s Snapshot, target string) bool {
	for _, t := range s.Departments {
		for _, r := range t.Records {
			if r.Data.Parent == target {
				return true
			}
		}
	}
	return false
}

func Discover(s Snapshot, target string, buSeeds, parentSeeds []string) Dependencies {
	ds, bs := map[string]bool{}, map[string]bool{}
	if target != "" {
		ds[target] = true
	}
	for _, id := range buSeeds {
		if id != "" {
			bs[id] = true
		}
	}
	// A BU writer seeds all actual/future users; a department writer does not
	// unnecessarily lock every peer merely because it shares a BU.
	if target == "" {
		for _, t := range s.Departments {
			for _, r := range t.Records {
				if bs[r.Data.BU] && r.State == "active" {
					ds[t.ID] = true
				}
			}
		}
	}
	// First include descendants of the changed identities. Merely being a
	// sibling of a proposed parent does not make a department a reverse user.
	for changed := true; changed; {
		changed = false
		for _, t := range s.Departments {
			if ds[t.ID] {
				continue
			}
			for _, r := range t.Records {
				if r.Data.Parent != "" && ds[r.Data.Parent] {
					ds[t.ID] = true
					changed = true
					break
				}
			}
		}
	}
	// Ancestors supply qualification, paths and stable lock identities. Traversing
	// an ancestor must not restart a reverse traversal into its other children.
	for _, id := range parentSeeds {
		if id != "" {
			ds[id] = true
		}
	}
	for changed := true; changed; {
		changed = false
		for _, t := range s.Departments {
			if !ds[t.ID] {
				continue
			}
			for _, r := range t.Records {
				bs[r.Data.BU] = true
				if r.Data.Parent != "" && !ds[r.Data.Parent] {
					ds[r.Data.Parent] = true
					changed = true
				}
			}
		}
	}

	out := Dependencies{BUs: []string{}, Departments: []string{}}
	for id := range bs {
		if id != "" {
			out.BUs = append(out.BUs, id)
		}
	}
	for id := range ds {
		out.Departments = append(out.Departments, id)
	}
	slices.Sort(out.BUs)
	slices.Sort(out.Departments)
	return out
}

// QualificationError is internal. Application projects only authorized facts.
type QualificationError struct{ Code, DepartmentID, DependencyID, Start string }

func (e *QualificationError) Error() string { return e.Code }
func invalidRelation(code, id, dependency, date string) error {
	return &QualificationError{code, id, dependency, date}
}

// Validate checks all boundary dates in the affected component. The database
// supplies facts and locks; it does not decide dates, state, coverage or cycles.
func Validate(ctx context.Context, s Snapshot, deps Dependencies) error {
	// Resolve immutable identities once. All dated boundaries are still checked,
	// in their original order, against this request's locked snapshot. Dense
	// positions avoid rebuilding string maps for every date in a large closure.
	type linkedRecord struct {
		record     *Record
		parent, bu int
	}
	positions := make(map[string]int, len(deps.Departments))
	for i, id := range deps.Departments {
		positions[id] = i
	}
	rows := make([][]linkedRecord, len(deps.Departments))
	bus := []*BusinessUnit{}
	buPositions := map[string]int{}
	dates := []string{}
	for i, id := range deps.Departments {
		if t := s.Department(id); t != nil {
			rows[i] = make([]linkedRecord, len(t.Records))
			for j := range t.Records {
				r := &t.Records[j]
				dates = append(dates, r.Date)
				parent, ok := positions[r.Data.Parent]
				if !ok || r.Data.Parent == "" {
					parent = -1
				}
				bu, ok := buPositions[r.Data.BU]
				if !ok {
					bu = len(bus)
					buPositions[r.Data.BU] = bu
					bus = append(bus, s.BU(r.Data.BU))
				}
				rows[i][j] = linkedRecord{r, parent, bu}
			}
		}
	}
	for _, id := range deps.BUs {
		if b := s.BU(id); b != nil {
			for _, r := range b.Records {
				dates = append(dates, r.Date)
			}
		}
	}
	slices.Sort(dates)
	dates = slices.Compact(dates)
	active := make([]*linkedRecord, len(rows))
	cursors := make([]int, len(rows))
	for i := range cursors {
		cursors[i] = -1
	}
	covered := make([]bool, len(bus))
	visited := make([]uint8, len(rows))
	for _, date := range dates {
		if err := ctx.Err(); err != nil {
			return err
		}
		for i, records := range rows {
			for cursors[i]+1 < len(records) && records[cursors[i]+1].record.Date <= date {
				cursors[i]++
			}
			active[i] = nil
			if j := cursors[i]; j >= 0 && records[j].record.State == "active" {
				active[i] = &records[j]
			}
		}
		for i, b := range bus {
			covered[i] = b != nil && activeBURecord(BUAt(b.Records, date))
		}
		for i, id := range deps.Departments {
			link := active[i]
			if link == nil {
				continue
			}
			r := link.record
			if !covered[link.bu] {
				return invalidRelation("M02_DEPARTMENT_BU_CONFLICT", id, r.Data.BU, date)
			}
			if r.Data.Parent != "" {
				if link.parent < 0 || active[link.parent] == nil || active[link.parent].record.Data.BU != r.Data.BU {
					return invalidRelation("M02_DEPARTMENT_PARENT_INVALID", id, r.Data.Parent, date)
				}
			}
		}
		clear(visited)
		for i, id := range deps.Departments {
			if active[i] == nil || visited[i] == 2 {
				continue
			}
			next := i
			for next >= 0 && visited[next] != 2 {
				if visited[next] == 1 {
					return invalidRelation("M02_DEPARTMENT_HIERARCHY_CYCLE", id, deps.Departments[next], date)
				}
				visited[next] = 1
				next = active[next].parent
			}
			for next = i; next >= 0 && visited[next] == 1; next = active[next].parent {
				visited[next] = 2
			}
		}
	}
	return nil
}

func parentInBU(parent *Record, bu string) bool {
	return parent != nil && parent.State == "active" && parent.Data.BU == bu
}

func activeBURecord(record *BURecord) bool { return record != nil && record.State == "active" }

// ParentQualifiesAt is only a necessary boundary check. A candidate that passes
// it still needs complete dated qualification, descendant and cycle validation.
func ParentQualifiesAt(s Snapshot, parent, date, bu string) bool {
	t := s.Department(parent)
	return t != nil && parentInBU(recordAt(t.Records, date), bu)
}

type RelationPlan struct {
	Changed        bool
	Edges, Deleted []Edge
}

func sameEdge(a, b Edge) bool {
	return a.Start == b.Start && a.End == b.End && a.BU == b.BU && a.Parent == b.Parent
}

// Plan builds only the target child's relation timeline. Planned references are
// deterministic keys until the authorized final write allocates real identities.
func Plan(t Timeline, rows []Record) RelationPlan {
	edges := []Edge{}
	for _, r := range rows {
		if r.State != "active" {
			continue
		}
		n := len(edges)
		if n > 0 && edges[n-1].End == r.Date && edges[n-1].BU == r.Data.BU && edges[n-1].Parent == r.Data.Parent {
			edges[n-1].End = r.End
			continue
		}
		edges = append(edges, Edge{Start: r.Date, End: r.End, BU: r.Data.BU, Parent: r.Data.Parent})
	}
	used := map[string]bool{}
	for i := range edges {
		for _, old := range t.Edges {
			if sameEdge(old, edges[i]) {
				edges[i] = old
				used[old.ID] = true
				break
			}
		}
	}
	changed := len(edges) != len(t.Edges)
	for i := range edges {
		if edges[i].ID != "" {
			continue
		}
		changed = true
		for _, old := range t.Edges {
			if !used[old.ID] && old.Start == edges[i].Start {
				edges[i].ID = old.ID
				edges[i].Revision = old.Revision
				edges[i].RevisionNo = old.RevisionNo
				used[old.ID] = true
				break
			}
		}
		key := fmt.Sprintf("planned:%d", i)
		edges[i].Ref = MemberRef{DepartmentID: t.ID, Source: "planned", MemberID: key, BU: edges[i].BU, Start: edges[i].Start, End: edges[i].End, Parent: edges[i].Parent}
	}
	deleted := []Edge{}
	for _, old := range t.Edges {
		if !used[old.ID] {
			deleted = append(deleted, old)
		}
	}
	return RelationPlan{Changed: changed, Edges: edges, Deleted: deleted}
}
func (p RelationPlan) Bind(rows []Record) ([]Record, error) {
	out := Kernel.CloneRecords(rows)
	for i := range out {
		r := &out[i]
		if r.State != "active" {
			continue
		}
		found := false
		for _, edge := range p.Edges {
			if edge.Start <= r.Date && (edge.End == "" || r.End != "" && edge.End >= r.End) && edge.BU == r.Data.BU && edge.Parent == r.Data.Parent {
				old := r.Data.Member
				if old.MemberID == "" || old.Start != edge.Start || old.End != edge.End || old.BU != edge.BU || old.Parent != edge.Parent {
					r.Data.Member = edge.Ref
				}
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("department relation binding incomplete")
		}
	}
	return out, nil
}

type RelationDelta struct {
	OldBU, NewBU             string
	BUChanged, ParentChanged bool
	Start, End               string
}

// Deltas compare stable values across the entire candidate, including implicit
// deletion/redate reversions. Closing/reopening an identical relation adds no right.
func Deltas(before, after []Record) []RelationDelta {
	dates := []string{}
	for _, rows := range [][]Record{before, after} {
		for _, r := range rows {
			dates = append(dates, r.Date)
		}
	}
	slices.Sort(dates)
	dates = slices.Compact(dates)
	out := []RelationDelta{}
	for i, start := range dates {
		b, a := At(before, start), At(after, start)
		// A moved first date is compared with the original fixed first's data.
		if b == nil && len(before) > 0 {
			v := before[0]
			b = &v
		}
		if a == nil || b == nil {
			continue
		}
		if b.Data.BU == a.Data.BU && b.Data.Parent == a.Data.Parent {
			continue
		}
		end := ""
		if i+1 < len(dates) {
			end = dates[i+1]
		}
		out = append(out, RelationDelta{OldBU: b.Data.BU, NewBU: a.Data.BU, BUChanged: b.Data.BU != a.Data.BU, ParentChanged: b.Data.Parent != a.Data.Parent, Start: start, End: end})
	}
	return out
}

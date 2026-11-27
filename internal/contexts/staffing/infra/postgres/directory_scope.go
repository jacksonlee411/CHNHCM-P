package postgres

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"maps"
	"slices"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	app "chnhcm.local/chnhcm/internal/contexts/staffing/application"
	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	f "chnhcm.local/chnhcm/internal/foundation"
)

type directoryDepartment struct {
	app.PositionSource
	name, parent [2]bool
}

// Start at disclosed assignments. Only cross an edge when both its source
// relationship and its destination name can be read.
func directoryDepartments(ctx context.Context, u *unit, rows []h.EmployeeRow, day string, includeAncestors bool) ([]h.EmployeeDepartment, map[string][]string, error) {
	type step struct{ row, id, child, at string }
	nodes := map[string]h.EmployeeDepartment{}
	paths := map[string][]string{}
	var frontier []step
	for _, r := range rows {
		nodes[r.Department] = h.EmployeeDepartment{ID: r.Department, Name: r.DepartmentName}
		frontier = append(frontier, step{row: r.ID, id: r.Department, at: max(day, r.From)})
	}
	loaded := map[string]directoryDepartment{}
	edgeDay := map[string]string{}
	today := u.now.In(f.Shanghai).Format("2006-01-02")
	for len(frontier) > 0 {
		var missing []string
		for _, s := range frontier {
			if _, ok := loaded[s.id]; !ok {
				missing = append(missing, s.id)
				loaded[s.id] = directoryDepartment{}
			}
		}
		if len(missing) > 0 {
			sources, err := u.PositionSources(ctx, "departmentId", missing, false)
			if err != nil {
				return nil, nil, err
			}
			for _, src := range sources {
				node := directoryDepartment{PositionSource: src}
				for i, action := range []string{"department.read", "department.history.read"} {
					grant, err := u.Access().Decide(ctx, access.Intent{EnterpriseID: src.Enterprise, DepartmentTarget: src.DepartmentTarget, Action: action, Kind: "object"})
					if err != nil {
						return nil, nil, err
					}
					node.name[i] = grant.Allowed && slices.Contains(grant.ReadFields, "name")
					node.parent[i] = grant.Allowed && slices.Contains(grant.ReadFields, "parentDepartmentId")
				}
				loaded[src.ID] = node
			}
		}
		var next []step
		for _, s := range frontier {
			node := loaded[s.id]
			i := d.At(node.Timeline, s.at)
			if i < 0 || slices.Contains(paths[s.row], s.id) {
				continue
			}
			record := node.Records[i]
			current := s.at == today
			if s.child != "" {
				if includeAncestors && (!(node.name[0] && (current || node.name[1])) || record.Data["name"] == "") {
					continue
				}
				if _, ok := nodes[s.id]; !ok {
					nodes[s.id] = h.EmployeeDepartment{ID: s.id, Name: record.Data["name"]}
				}
				if prior := edgeDay[s.child]; prior == "" || s.at < prior {
					child := nodes[s.child]
					child.Parent = s.id
					nodes[s.child] = child
					edgeDay[s.child] = s.at
				}
			}
			paths[s.row] = append(paths[s.row], s.id)
			if parent := record.Data["parent"]; parent != "" && node.parent[0] && (includeAncestors && current || node.parent[1]) {
				if _, direct := nodes[parent]; !includeAncestors && !direct {
					continue
				}
				next = append(next, step{row: s.row, id: parent, child: s.id, at: s.at})
			}
		}
		frontier = next
	}
	tree := slices.Collect(maps.Values(nodes))
	if tree == nil {
		tree = []h.EmployeeDepartment{}
	}
	slices.SortFunc(tree, func(a, b h.EmployeeDepartment) int {
		return cmp.Or(cmp.Compare(a.Name, b.Name), cmp.Compare(a.ID, b.ID))
	})
	// Separate effective dates can represent opposite moves. Keep the displayed
	// tree acyclic while retaining each assignment's actual dated scope above.
	visited := map[string]bool{}
	for _, node := range tree {
		path := map[string]bool{}
		for id := node.ID; id != "" && !visited[id]; id = nodes[id].Parent {
			if path[id] {
				root := nodes[id]
				root.Parent = ""
				nodes[id] = root
				break
			}
			path[id] = true
		}
		for id := range path {
			visited[id] = true
		}
	}
	for i, node := range tree {
		tree[i] = nodes[node.ID]
	}
	return tree, paths, nil
}

func directorySelection(q h.PersonnelQuery, rows []h.EmployeeRow, session, day string, canList bool, base string, matched, total int) *h.DirectorySelectionResult {
	digest := sha256.New()
	encoder := json.NewEncoder(digest)
	_ = encoder.Encode([]any{session, day, q.Ref, q.Department, q.DirectOnly, q.Primary, canList, base})
	sorted := slices.Clone(rows)
	slices.SortFunc(sorted, func(a, b h.EmployeeRow) int { return cmp.Compare(a.ID, b.ID) })
	for _, r := range sorted {
		_ = encoder.Encode([]string{r.ID, r.SourceVersion, r.PersonVersion, r.Until})
	}
	version := hex.EncodeToString(digest.Sum(nil))
	result := &h.DirectorySelectionResult{Version: version}
	result.Changed = (q.Selection.Version != "" && q.Selection.Version != version) || matched != len(q.Selection.IDs)
	if !result.Changed {
		result.Count = matched
		if q.Selection.All {
			result.Count = total - matched
		}
	}
	return result
}

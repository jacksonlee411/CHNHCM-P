package department

import (
	"context"
	"testing"
	"time"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
)

type pathDigestGuard struct {
	access.Guard
	hide, closeParent bool
}

func (g pathDigestGuard) Decide(_ context.Context, in access.Intent) (access.Decision, error) {
	v := access.Decision{Allowed: true, ReadFields: []string{"code", "name"}}
	if in.DepartmentTarget != nil && in.DepartmentTarget.DepartmentID == "parent" {
		if g.hide && in.Action == "department.reference.read" {
			v.ReadFields = []string{"code"}
		}
		if g.closeParent && in.Action == "department.read" {
			v.Allowed = false
		}
	}
	return v, nil
}

func TestListPathDigestKeepsBusinessDependenciesWithoutNavigation(t *testing.T) {
	// Expected equivalence classes are independent of the digest representation.
	type scenario struct {
		name, parentName, source, edge, sibling, end string
		enterpriseName                               string
		hide, closeParent, noPath                    bool
	}
	cases := []scenario{
		{name: "original"}, {name: "ancestor label", parentName: "新上级"},
		{name: "ancestor source", source: "new-source"}, {name: "relation member", edge: "new-member"},
		{name: "unrelated sibling", sibling: "新兄弟"}, {name: "record boundary", end: "2027-01-01"},
		{name: "ancestor navigation", closeParent: true}, {name: "hidden ancestor", hide: true},
		{name: "hidden ancestor changed", hide: true, parentName: "隐藏更名", source: "hidden-source"},
		{name: "path field hidden", noPath: true},
		{name: "enterprise rename", enterpriseName: "更名企业"},
	}
	groups := []int{0, 1, 2, 3, 0, 5, 0, 7, 7, 7, 0}
	versions := []string{}
	referenceVersions := []string{}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			name, source, edge := "上级", "parent-source", "parent-member"
			enterpriseName := "企业"
			if c.enterpriseName != "" {
				enterpriseName = c.enterpriseName
			}
			if c.parentName != "" {
				name = c.parentName
			}
			if c.source != "" {
				source = c.source
			}
			if c.edge != "" {
				edge = c.edge
			}
			makeTimeline := func(id, parent, name, source, member string) d.Timeline {
				return d.Timeline{ID: id, Code: id, Enterprise: "enterprise", First: id + "-record", Source: source, RelationSource: id + "-relation", Records: []d.Record{{ID: id + "-record", Revision: id + "-revision", Date: "2020-01-01", State: "active", Data: d.Data{Name: name, BU: "bu", Parent: parent}}}, Edges: []d.Edge{{ID: id + "-edge", BU: "bu", Parent: parent, Start: "2020-01-01", Ref: d.MemberRef{DepartmentID: id, Source: id + "-relation", MemberID: member, BU: "bu"}}}}
			}
			parent := makeTimeline("parent", "", name, source, edge)
			child := makeTimeline("child", "parent", "子部门", "child-source", "child-member")
			child.Records[0].End = c.end
			sibling := makeTimeline("sibling", "parent", c.sibling, "sibling-source", "sibling-member")
			s := d.Snapshot{Departments: []d.Timeline{parent, child, sibling}}.Indexed()
			u := projectedReadUnit{guard: pathDigestGuard{hide: c.hide, closeParent: c.closeParent}}
			p := newProjector(context.Background(), u, s, org.Enterprise{Name: enterpriseName}, time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC))
			fields := []string{"departmentPath", "effectiveDate", "status"}
			if c.noPath {
				fields = []string{"effectiveDate", "status"}
			}
			value, err := p.listPathValue(&child, &child.Records[0], readMask(fields))
			if err != nil {
				t.Fatal(err)
			}
			got := value.summary()
			facts, err := NewListFacts(context.Background(), s)
			if err != nil {
				t.Fatal(err)
			}
			cached := newProjector(context.Background(), u, s, org.Enterprise{Name: enterpriseName}, time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC))
			cached.listFacts = facts
			cachedValue, err := cached.listPathValue(&child, &child.Records[0], readMask(fields))
			if err != nil || cachedValue.summary().PathVersion != got.PathVersion {
				t.Fatal("unqualified node cache changed current projection", err)
			}
			want, err := referencePath(p, child, child.Records[0], fields)
			if err != nil || got.PathMode != want.PathMode || valueString(got.Path) != valueString(want.Path) || valueString(got.PathBusinessAt) != valueString(want.PathBusinessAt) {
				t.Fatal("visible path changed", got, want, err)
			}
			referenceVersions = append(referenceVersions, want.PathVersion)
			if c.hide || c.noPath {
				if got.PathMode != "restricted" || got.Path != nil {
					t.Fatal("hidden dependency leaked", got)
				}
			} else {
				label := name + " / 子部门"
				if got.Path == nil || *got.Path != label || got.PathMode != "active" || *got.PathBusinessAt != "2026-11-06" || len(got.PathSegments) != 0 {
					t.Fatal("incorrect visible path", got)
				}

			}
			// The same prefix may be reused in this request; it must not change it.
			again, err := p.listPathValue(&child, &child.Records[0], readMask(fields))
			if err != nil || again.summary().PathVersion != got.PathVersion {
				t.Fatal("prefix reuse changed path")
			}
			versions = append(versions, got.PathVersion)
		})
	}
	for i := range versions {
		for j := range versions {
			if (versions[i] == versions[j]) != (groups[i] == groups[j]) || (versions[i] == versions[j]) != (referenceVersions[i] == referenceVersions[j]) {
				t.Fatalf("changed original equivalence: %s / %s", cases[i].name, cases[j].name)
			}
		}
	}
}

func valueString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

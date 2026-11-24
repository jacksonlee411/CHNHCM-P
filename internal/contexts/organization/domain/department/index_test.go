package department

import (
	"slices"
	"testing"
)

func TestIndexedSnapshotCandidateDoesNotReplaceOriginalIdentity(t *testing.T) {
	original := Snapshot{Departments: []Timeline{{ID: "a", Code: "original"}}, BusinessUnits: []BusinessUnit{{ID: "bu"}}}.Indexed()
	replaced := original.With(Timeline{ID: "a", Code: "candidate"})
	added := replaced.With(Timeline{ID: "b", Code: "new"})
	if original.Department("a").Code != "original" || replaced.Department("a").Code != "candidate" || added.Department("b").Code != "new" || original.Department("b") != nil || replaced.Department("b") != nil || added.BU("bu") == nil || added.BU("missing") != nil {
		t.Fatal("candidate identity index modified its base or lost a lookup")
	}
}

func TestChangedDescendantsAndAncestorQualificationsExcludeUnaffectedSibling(t *testing.T) {
	rec := func(parent string) []Record { return []Record{{State: "active", Data: Data{BU: "bu", Parent: parent}}} }
	s := Snapshot{Departments: []Timeline{
		{ID: "p", Records: rec("")}, {ID: "q", Records: rec("")},
		{ID: "c", Records: append(rec("p"), rec("q")...)},
		{ID: "d", Records: rec("c")}, {ID: "sibling", Records: rec("p")},
	}}.Indexed()
	for _, tc := range []struct {
		target  string
		parents []string
		want    []string
	}{
		{"c", nil, []string{"c", "d", "p", "q"}},
		{"new", []string{"p"}, []string{"new", "p"}},
		{"p", nil, []string{"c", "d", "p", "q", "sibling"}},
	} {
		got := Discover(s, tc.target, nil, tc.parents)
		if !slices.Equal(got.Departments, tc.want) || !slices.Equal(got.BUs, []string{"bu"}) {
			t.Fatalf("target %s: %+v", tc.target, got)
		}
	}
}

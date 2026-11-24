package history

import (
	"testing"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
)

func TestTreeReadSetCoverageDoesNotInferMembership(t *testing.T) {
	for _, tc := range []struct {
		name          string
		cover, wanted access.ReadSet
		ok            bool
	}{
		{"enterprise", access.ReadSet{All: true}, access.ReadSet{Departments: []string{"d"}}, true},
		{"cannot_cover_enterprise", access.ReadSet{BUs: []string{"b"}}, access.ReadSet{All: true}, false},
		{"union", access.ReadSet{BUs: []string{"b"}, Departments: []string{"d", "e"}}, access.ReadSet{BUs: []string{"b"}, Departments: []string{"d"}}, true},
		{"missing_department", access.ReadSet{Departments: []string{"e"}}, access.ReadSet{Departments: []string{"d"}}, false},
		{"missing_bu", access.ReadSet{BUs: []string{"b"}}, access.ReadSet{BUs: []string{"c"}}, false},
		{"bu_does_not_prove_department", access.ReadSet{BUs: []string{"same-id"}}, access.ReadSet{Departments: []string{"same-id"}}, false},
		{"department_does_not_prove_bu", access.ReadSet{Departments: []string{"same-id"}}, access.ReadSet{BUs: []string{"same-id"}}, false},
		{"empty", access.ReadSet{}, access.ReadSet{}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := treeSetCovers(tc.cover, tc.wanted); got != tc.ok {
				t.Fatalf("coverage=%v, want %v", got, tc.ok)
			}
		})
	}
}

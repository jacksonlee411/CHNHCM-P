package postgres

import (
	"testing"

	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
)

func sparseTimeline() d.Timeline {
	return d.Timeline{ID: "d", First: "first", Source: "s", RelationSource: "rel", Records: []d.Record{{ID: "first", Date: "2020-01-01", End: "2020-02-01", State: "active"}, {ID: "current", Date: "2020-04-01", State: "active"}}, Edges: []d.Edge{{ID: "e", Start: "2020-01-01", BU: "bu"}}}
}
func TestMergingSparseFactsDoesNotMutateEarlierReaders(t *testing.T) {
	old := sparseTimeline()
	v := old
	v.Records = []d.Record{{ID: "middle", Date: "2020-02-01", End: "2020-04-01", State: "active"}}
	merged := mergeDepartment(old, v)
	if len(old.Records) != 2 || len(merged.Records) != 3 || d.ScopeRecord(merged, "2020-03-01").ID != "middle" {
		t.Fatal("merge modified old reader or lost range")
	}
	merged.Records[0].Date = "changed"
	if old.Records[0].Date != "2020-01-01" {
		t.Fatal("merged slice aliases prior reader")
	}
}
func TestSparseBUCannotTreatUnloadedPastAsFutureFirst(t *testing.T) {
	b := d.BusinessUnit{First: "first", Records: []d.BURecord{{ID: "later", Date: "2035-01-01", State: "active"}}}
	if buCovers(b, "2026-01-01") {
		t.Fatal("unloaded past treated as future first")
	}
	b.Records[0].ID = "first"
	if !buCovers(b, "2026-01-01") {
		t.Fatal("real future first was not covered")
	}
}

package department

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	"chnhcm.local/chnhcm/internal/foundation"
)

func treeProjectionFixture() (d.Snapshot, projectedReads) {
	rows := []d.Timeline{}
	for i, id := range []string{"a", "b", "c"} {
		parent := ""
		if i > 0 {
			parent = []string{"a", "b"}[i-1]
		}
		rows = append(rows, d.Timeline{ID: id, Code: id, Source: id + "-source", Records: []d.Record{{ID: id + "-record", Revision: id + "-revision", Date: "2020-01-01", State: "active", Data: d.Data{Name: id, BU: "bu", Parent: parent}}}, Edges: []d.Edge{{Start: "2020-01-01", BU: "bu", Parent: parent, Ref: d.MemberRef{DepartmentID: id, Source: "relation", MemberID: id, Parent: parent}}}})
	}
	snapshot := d.Snapshot{Departments: rows}.ReadIndexed()
	reads := projectedReads{snapshot: snapshot}
	for range rows {
		n := len(reads.decisions)
		reads.departments = append(reads.departments, projectedDepartmentReads{target: &access.DepartmentTarget{}, object: n, reference: n + 1})
		reads.decisions = append(reads.decisions, access.Decision{Allowed: true, ReadFields: append([]string{}, treeFields...)}, access.Decision{Allowed: true, ReadFields: []string{"code", "name"}})
	}
	return snapshot, reads
}

func TestTreePathReferenceAncestorDoesNotAcquireObjectRights(t *testing.T) {
	snapshot, reads := treeProjectionFixture()
	reads.decisions[2] = access.Decision{}
	got, err := projectTreePath(context.Background(), snapshot, reads, "c", "2026-01-01")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0].ID != "a" || got[1].ID != "b" || got[2].ID != "c" || got[1].Kind != "context" || got[2].Kind != "node" || got[0].ParentID != nil || got[2].ParentID == nil || *got[2].ParentID != "b" {
		t.Fatalf("incorrect authorized path: %#v", got)
	}
	// Hiding any necessary reference rejects the entire chain, never a prefix.
	reads.decisions[3] = access.Decision{}
	got, err = projectTreePath(context.Background(), snapshot, reads, "c", "2026-01-01")
	var fault *foundation.Fault
	if got != nil || !errors.As(err, &fault) || fault.Code != "M02_TREE_SCOPE_UNAVAILABLE" {
		t.Fatal("restricted ancestry exposed", got, err)
	}
}

func TestTreePathRequiredFieldsAndCurrentDay(t *testing.T) {
	for _, field := range treeFields {
		t.Run(field, func(t *testing.T) {
			snapshot, reads := treeProjectionFixture()
			fields := []string{}
			for _, f := range treeFields {
				if f != field {
					fields = append(fields, f)
				}
			}
			reads.decisions[4].ReadFields = fields
			if got, err := projectTreePath(context.Background(), snapshot, reads, "c", "2026-01-01"); got != nil || err == nil {
				t.Fatal("missing field allowed", field, got, err)
			}
		})
	}
	for _, day := range []string{"2019-12-31", "2026-01-01"} {
		snapshot, reads := treeProjectionFixture()
		if day == "2026-01-01" {
			snapshot.Departments[2].Records[0].State = "inactive"
		}
		if got, err := projectTreePath(context.Background(), snapshot, reads, "c", day); got != nil || err == nil {
			t.Fatal("noncurrent node returned", got, err)
		}
	}
}

func TestTreePathCancellationAndBrokenEdgeHaveNoPartialResult(t *testing.T) {
	snapshot, reads := treeProjectionFixture()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got, err := projectTreePath(ctx, snapshot, reads, "c", "2026-01-01"); got != nil || !errors.Is(err, context.Canceled) {
		t.Fatal(got, err)
	}
	before := append([]d.Edge{}, snapshot.Departments[2].Edges...)
	snapshot.Departments[2].Edges[0].Parent = "missing"
	got, err := projectTreePath(context.Background(), snapshot, reads, "c", "2026-01-01")
	var fault *foundation.Fault
	if got != nil || !errors.As(err, &fault) || fault.Status != 503 {
		t.Fatal("broken facts not rejected", got, err)
	}
	if reflect.DeepEqual(before, snapshot.Departments[2].Edges) {
		t.Fatal("fixture was not corrupted")
	}
}

func TestTreePathDeepChainIsCompleteBeyondBrowserCacheBudget(t *testing.T) {
	const count = 2501
	rows := make([]d.Timeline, 0, count)
	for i := 0; i < count; i++ {
		id := fmt.Sprintf("N%06d", i)
		parent := ""
		if i > 0 {
			parent = fmt.Sprintf("N%06d", i-1)
		}
		rows = append(rows, d.Timeline{ID: id, Code: id, Source: id + "-source", Records: []d.Record{{ID: id + "-record", Revision: id + "-revision", Date: "2020-01-01", State: "active", Data: d.Data{Name: id, BU: "bu", Parent: parent}}}, Edges: []d.Edge{{Start: "2020-01-01", BU: "bu", Parent: parent, Ref: d.MemberRef{DepartmentID: id, Source: "relation", MemberID: id, Parent: parent}}}})
	}
	snapshot := d.Snapshot{Departments: rows}.ReadIndexed()
	reads := projectedReads{snapshot: snapshot}
	for range rows {
		n := len(reads.decisions)
		reads.departments = append(reads.departments, projectedDepartmentReads{target: &access.DepartmentTarget{}, object: n, reference: n + 1})
		reads.decisions = append(reads.decisions, access.Decision{Allowed: true, ReadFields: treeFields}, access.Decision{Allowed: true, ReadFields: []string{"code", "name"}})
	}
	chain, err := projectTreePath(context.Background(), snapshot, reads, rows[count-1].ID, "2026-01-01")
	if err != nil || len(chain) != count {
		t.Fatal("path truncated at client cache budget", len(chain), err)
	}
	for i, n := range chain {
		if n.ID != rows[i].ID {
			t.Fatal("path order changed", i)
		}
	}
}

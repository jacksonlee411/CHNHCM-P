package dated

import (
	"context"
	"reflect"
	"testing"
)

func TestReferenceRevisionOnlyHasStructuralImpact(t *testing.T) {
	k := fixtureKernel()
	base := fixtureBase()
	for _, action := range []string{"CORRECT", "ADD"} {
		t.Run(action, func(t *testing.T) {
			it := Item[fixturePatch]{ID: "reference", Action: action, RecordID: "first", ExpectedRevision: "first-v1", Changes: &fixturePatch{Ref: &relation{"root-a", "edge-v2"}}}
			if action == "ADD" {
				it.RecordID = ""
				it.ExpectedRevision = ""
				it.Date = "2026-03-01"
				it.Type = "CHANGE"
				it.Reason = "valid"
			}
			c, e := k.Calculate(context.Background(), base, "first", "2020-01-01", []Item[fixturePatch]{it})
			if e != nil || c.NoChange || len(c.Impact.DataAffectedRanges) != 0 || len(c.Impact.StructuralChanges) == 0 {
				t.Fatalf("reference update lost: %+v, %v", c, e)
			}
			if action == "CORRECT" {
				want := []StructuralChange{{RecordID: "first", ItemID: "reference", BeforeRevision: val("first-v1"), Facets: []string{"REFERENCE"}}}
				if !reflect.DeepEqual(c.Impact.StructuralChanges, want) {
					t.Fatalf("got %+v", c.Impact.StructuralChanges)
				}
			}
		})
	}
}

func TestReferenceChangePropagatesThroughStopButNotIndependentResume(t *testing.T) {
	k := fixtureKernel()
	base := fixtureBase()
	base[0].End = "2026-04-01"
	stop := k.CloneRecords(base)[0]
	stop.ID = "stop"
	stop.Revision = "stop-v1"
	stop.Date = "2026-04-01"
	stop.End = "2026-06-01"
	stop.Type = "DEACTIVATE"
	stop.State = "inactive"
	stop.InheritedRecord = "first"
	stop.InheritedRevision = "first-v1"
	resume := k.CloneRecords(base)[0]
	resume.ID = "resume"
	resume.Revision = "resume-v1"
	resume.Date = "2026-06-01"
	resume.End = ""
	resume.Type = "REACTIVATE"
	base = append(base, stop, resume)
	c, e := k.Calculate(context.Background(), base, "first", "2020-01-01", []Item[fixturePatch]{{ID: "reference", Action: "CORRECT", RecordID: "first", ExpectedRevision: "first-v1", Changes: &fixturePatch{Ref: &relation{"root-a", "edge-v2"}}}})
	if e != nil {
		t.Fatal(e)
	}
	if c.Records[1].Data.Ref.Revision != "edge-v2" || c.Records[1].InheritedRevision != "" || !reflect.DeepEqual(c.Records[2], resume) || len(c.Impact.DataAffectedRanges) != 0 {
		t.Fatalf("wrong inheritance: %+v", c)
	}
	if !reflect.DeepEqual(c.Impact.AffectedRecordIDs, []string{"first", "stop"}) {
		t.Fatal(c.Impact)
	}
}

func TestDataChangesIncludesRelationshipIdentityAndDetachesReports(t *testing.T) {
	k := fixtureKernel()
	base := fixtureBase()
	after := k.CloneRecords(base)
	after[0].Data.Ref.Target = "root-b"
	rows, e := k.DataChanges(context.Background(), base, after)
	if e != nil || len(rows) != 1 || rows[0].Start != "2026-01-01" || rows[0].End != "" {
		t.Fatalf("relationship omitted: %+v %v", rows, e)
	}
	rows[0].Before.Data.Tags[0] = "changed"
	rows[0].After.Data.Labels["key"] = "changed"
	if base[0].Data.Tags[0] != "tag" || after[0].Data.Labels["key"] != "value" {
		t.Fatal("report aliases snapshots")
	}
	after = k.CloneRecords(base)
	after[0].Data.Ref.Revision = "edge-v2"
	rows, e = k.DataChanges(context.Background(), base, after)
	if e != nil || len(rows) != 0 {
		t.Fatal("version-only change treated as business data")
	}
}

func TestAdoptedReferenceChangePeriods(t *testing.T) {
	k := fixtureKernel()
	base := fixtureBase()
	before := k.CloneRecords(base)
	for _, step := range []struct{ date, revision string }{
		{"2026-03-01", "v2"},
		{"2026-05-01", "v2"},
		{"2026-07-01", "v3"},
		{"2026-09-01", base[0].Data.Ref.Revision},
	} {
		row := k.CloneRecords(base)[0]
		row.ID = "record-" + step.date
		row.Type = "CHANGE"
		row.Date = step.date
		row.Data.Ref.Revision = step.revision
		before = append(before, row)
	}
	changes, err := k.DataChangesWithAdoptions(context.Background(), before, base)
	if err != nil || len(changes) != 2 {
		t.Fatalf("adopted reference intervals: %+v, %v", changes, err)
	}
	if changes[0].Start != "2026-03-01" || changes[0].End != "2026-07-01" || changes[0].Before.Data.Ref.Revision != "v2" ||
		changes[1].Start != "2026-07-01" || changes[1].End != "2026-09-01" || changes[1].Before.Data.Ref.Revision != "v3" {
		t.Fatal("distinct adopted versions must retain their own before image", changes)
	}
}

func TestBusinessAndReferenceImpactsAreIndependent(t *testing.T) {
	k := fixtureKernel()
	referenceChange := []StructuralChange{{RecordID: "first", ItemID: "correct", BeforeRevision: val("first-v1"), Facets: []string{"REFERENCE"}}}
	for _, tc := range []struct {
		name          string
		patch         fixturePatch
		wantName      string
		wantReference string
		wantData      []Range
		wantStructure []StructuralChange
	}{
		{
			name: "business-only", patch: fixturePatch{Name: val("new")},
			wantName: "new", wantReference: "edge-v1",
			wantData: []Range{{Start: "2026-01-01"}}, wantStructure: []StructuralChange{},
		},
		{
			name: "reference-only", patch: fixturePatch{Ref: &relation{"root-a", "edge-v2"}},
			wantName: "same", wantReference: "edge-v2",
			wantData: []Range{}, wantStructure: referenceChange,
		},
		{
			name: "business-and-reference", patch: fixturePatch{Name: val("new"), Ref: &relation{"root-a", "edge-v2"}},
			wantName: "new", wantReference: "edge-v2",
			wantData: []Range{{Start: "2026-01-01"}}, wantStructure: referenceChange,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := fixtureBase()
			c, e := k.Calculate(context.Background(), base, "first", "2020-01-01", []Item[fixturePatch]{{ID: "correct", Action: "CORRECT", RecordID: "first", ExpectedRevision: "first-v1", Changes: &tc.patch}})
			if e != nil {
				t.Fatal(e)
			}
			if c.NoChange || len(c.Records) != 1 || c.Records[0].Data.Name != tc.wantName || !reflect.DeepEqual(c.Records[0].Data.Ref, &relation{"root-a", tc.wantReference}) {
				t.Fatalf("correction lost: %+v", c)
			}
			want := Impact{DataAffectedRanges: tc.wantData, StructuralChanges: tc.wantStructure, AffectedRecordIDs: []string{"first"}, PeriodChanges: []PeriodChange{}}
			if !reflect.DeepEqual(c.Impact, want) {
				t.Fatalf("got impact %+v; want %+v", c.Impact, want)
			}
		})
	}
}

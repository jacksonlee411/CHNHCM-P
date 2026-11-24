package domain

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"slices"
	"testing"
)

const gradeA = "00000000-0000-4000-8000-000000000001"
const gradeB = "00000000-0000-4000-8000-000000000002"

func TestMembershipInputAndClone(t *testing.T) {
	original := GradeMembers{{GradeID: gradeA, Adopted: ReferenceAdoption{BusinessAt: "2026-01-01", SourceVersion: foundation.NewID()}}, {GradeID: gradeB}}
	for _, k := range []Kind{Job, Ladder} {
		kept, e := (MembershipPatch{}).Apply(k, original)
		if e != nil || !kept.EqualRevision(original) {
			t.Fatalf("omitted %s: %v", k, e)
		}
		kept[0].Adopted.BusinessAt = "2030-01-01"
		if original[0].Adopted.BusinessAt != "2026-01-01" {
			t.Fatal("aliased adoption")
		}
		for _, ids := range [][]string{nil, {gradeA, gradeA}, {"bad"}} {
			if _, e := (MembershipPatch{Present: true, IDs: ids}).Apply(k, original); e == nil {
				t.Fatalf("accepted invalid %v", ids)
			}
		}
		empty, e := (MembershipPatch{Present: true, IDs: []string{}}).Apply(k, original)
		if k == Job && (e != nil || len(empty) != 0) || k == Ladder && e == nil {
			t.Fatalf("empty semantics %s: %v", k, e)
		}
		reordered, e := (MembershipPatch{Present: true, IDs: []string{gradeB, gradeA}}).Apply(k, original)
		if e != nil {
			t.Fatal(e)
		}
		if reordered.EqualData(original) != (k == Job) {
			t.Fatalf("order semantics %s", k)
		}
	}
}
func memberCreate(t *testing.T, k Kind) Timeline {
	t.Helper()
	name := "资料"
	changes := map[string]*string{"name": &name}
	reason := "NEW_GRADE_LADDER"
	if k == Job {
		f := foundation.NewID()
		changes["jobFunctionId"] = str(foundation.NewID())
		changes["jobFamilyId"] = &f
		reason = "NEW_JOB"
	}
	p, e := Build(k, foundation.NewID(), Timeline{}, Command{Action: "CREATE", Type: "CREATE", Reason: reason, Code: "G2", Date: "2026-01-01", Changes: changes, Grades: MembershipPatch{true, []string{gradeA, gradeB}}})
	if e != nil {
		t.Fatal(e)
	}
	for i := range p.After.Records[0].Grades {
		p.After.Records[0].Grades[i].Adopted = ReferenceAdoption{BusinessAt: "2026-01-01", SourceVersion: foundation.NewID(), RecordID: foundation.NewID(), Revision: foundation.NewID()}
	}
	return p.After
}
func TestMembershipDatedDecisions(t *testing.T) {
	for _, k := range []Kind{Job, Ladder} {
		t.Run(string(k), func(t *testing.T) {
			before := memberCreate(t, k)
			r := before.Records[0]
			c := Command{Action: "CORRECT", RecordID: r.ID, ExpectedSource: before.Source, ExpectedRevision: r.Revision, Grades: MembershipPatch{true, []string{gradeB, gradeA}}}
			p, e := Build(k, before.Enterprise, before, c)
			if e != nil {
				t.Fatal(e)
			}
			if p.Changed != (k == Ladder) {
				t.Fatal("wrong order revision")
			}
			if k == Ladder && (p.After.Source == before.Source || p.After.Records[0].Revision == r.Revision) {
				t.Fatal("missing revision")
			}
			if !p.After.Records[0].Grades[0].Adopted.EqualsForTest(r.Grades[map[bool]int{true: 1, false: 0}[k == Ladder]].Adopted) {
				t.Fatal("reorder replaced evidence")
			}
			c.Grades = MembershipPatch{}
			c.NewDate = "2025-01-01"
			p, e = Build(k, before.Enterprise, before, c)
			if e != nil {
				t.Fatal(e)
			}
			for _, m := range p.After.Records[0].Grades {
				if m.Adopted != (ReferenceAdoption{}) {
					t.Fatal("redate did not request readoption")
				}
			}
			p.After.Records[0].Grades[0].GradeID = gradeB
			if before.Records[0].Grades[0].GradeID != gradeA {
				t.Fatal("aliased plan")
			}
			c = Command{Action: "ADD", AnchorID: r.ID, ExpectedSource: before.Source, Date: "2027-01-01", Type: "CHANGE", Reason: "OTHER_DATA_ADJUSTMENT", Grades: MembershipPatch{true, []string{gradeB}}}
			future, e := Build(k, before.Enterprise, before, c)
			if e != nil {
				t.Fatal(e)
			}
			successor := future.After.Records[1]
			for _, newDate := range []string{"2026-12-01", "2027-02-01"} {
				moved, err := Build(k, before.Enterprise, future.After, Command{Action: "REDATE", RecordID: successor.ID, ExpectedRevision: successor.Revision, ExpectedSource: future.After.Source, NewDate: newDate})
				if err != nil || !slices.Contains(moved.Fields, "gradeIds") {
					t.Fatalf("moving a membership boundary to %s: fields=%v err=%v", newDate, moved.Fields, err)
				}
			}
			c.ExpectedSource = future.After.Source
			c.Date = "2026-07-01"
			c.Grades = MembershipPatch{true, []string{gradeA}}
			inserted, e := Build(k, before.Enterprise, future.After, c)
			if e != nil {
				t.Fatal(e)
			}
			if !slices.Equal(inserted.After.Records[2].Grades.IDs(), []string{gradeB}) {
				t.Fatal("propagated to independent successor")
			}
			if !inserted.After.Records[1].Grades[0].Adopted.EqualsForTest(r.Grades[0].Adopted) {
				t.Fatal("CHANGE did not inherit evidence")
			}
			del := Command{Action: "DELETE", RecordID: future.After.Records[1].ID, ExpectedRevision: future.After.Records[1].Revision, ExpectedSource: future.After.Source}
			deleted, e := Build(k, before.Enterprise, future.After, del)
			if e != nil {
				t.Fatal(e)
			}
			if !slices.Contains(deleted.Fields, "gradeIds") {
				t.Fatal("implicit deletion missing write field")
			}
		})
	}
}
func (a ReferenceAdoption) EqualsForTest(b ReferenceAdoption) bool { return a == b }
func TestGradeCoverageWholePeriods(t *testing.T) {
	c := memberCreate(t, Ladder)
	g := map[string]Timeline{}
	for _, id := range []string{gradeA, gradeB} {
		g[id] = Timeline{ID: id, Enterprise: c.Enterprise, Records: []Record{{Date: "2026-01-01", Type: "CREATE", End: "2027-06-01"}, {Date: "2027-06-01", Type: "DEACTIVATE", End: "2027-07-01"}, {Date: "2027-07-01", Type: "REACTIVATE"}}}
	}
	if ValidateGradeCoverage(c, g) == nil {
		t.Fatal("missed future inactive gap")
	}
	c.Records[0].End = "2027-06-01"
	if e := ValidateGradeCoverage(c, g); e != nil {
		t.Fatal("half open boundary", e)
	}
	c.Records[0].End = "2027-06-02"
	if ValidateGradeCoverage(c, g) == nil {
		t.Fatal("accepted one invalid day")
	}
}

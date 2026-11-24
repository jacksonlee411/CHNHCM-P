package domain

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"slices"
	"strings"
	"testing"
)

func str(v string) *string { return &v }
func create(t *testing.T, k Kind, date string) Timeline {
	t.Helper()
	c := Command{Action: "CREATE", Code: " test-01 ", Date: date, Type: "CREATE", Reason: "NEW_JOB_FAMILY", Changes: map[string]*string{"name": str("测试资料")}}
	if k == JobFunction {
		c.Reason = "NEW_JOB_FUNCTION"
	}
	if k == Job {
		c.Reason = "NEW_JOB"
		c.Changes["jobFunctionId"] = str(foundation.NewID())
		c.Changes["jobFamilyId"] = str(foundation.NewID())
	}
	p, e := Build(k, foundation.NewID(), Timeline{}, c)
	if e != nil {
		t.Fatal(e)
	}
	return p.After
}
func apply(t *testing.T, b Timeline, c Command) Plan {
	t.Helper()
	c.ExpectedSource = b.Source
	p, e := Build(b.Kind, b.Enterprise, b, c)
	if e != nil {
		t.Fatalf("%s: %v", c.Action, e)
	}
	return p
}
func TestStaffing307Admission(t *testing.T) {
	for _, k := range []Kind{Family, JobFunction, Job} {
		v := create(t, k, "")
		if v.Code != "TEST-01" || v.Records[0].Date != "1900-01-01" {
			t.Fatal(v)
		}
	}
	for _, code := range []string{"", "_START", "ＡBC", "ı", "A.B", strings.Repeat("A", 31)} {
		if _, e := NormalizeCode(code); e == nil {
			t.Errorf("accepted code %q", code)
		}
	}
	for _, name := range []string{"", "😀", "x\n", "x\t", "\u200d"} {
		_, e := patch(Family, nil, map[string]*string{"name": str(name)})
		if e == nil {
			t.Errorf("accepted name %q", name)
		}
	}
	v, e := patch(Family, nil, map[string]*string{"name": str(" Cafe\u0301 "), "description": str("  一\r\n二  ")})
	if e != nil || v["name"] != "Café" || v["description"] != "一\n二" {
		t.Fatalf("normalization %v %v", v, e)
	}
	for _, test := range []struct {
		k string
		v *string
	}{{"name", nil}, {"unknown", str("x")}, {"description", str(strings.Repeat("中", 1001))}, {"description", str("A\tB")}} {
		if _, e = patch(Family, v, map[string]*string{test.k: test.v}); e == nil {
			t.Errorf("accepted %s", test.k)
		}
	}
}
func TestStaffing307IndependentSuccessors(t *testing.T) {
	v := create(t, Family, "2000-01-01")
	first := v.Records[0]
	p := apply(t, v, Command{Action: "ADD", AnchorID: first.ID, Date: "2010-01-01", Type: "CHANGE", Reason: "NAME_ADJUSTMENT", Changes: map[string]*string{"name": str("独立后项")}})
	v = p.After
	last := v.Records[1]
	v = apply(t, v, Command{Action: "CORRECT", RecordID: first.ID, ExpectedRevision: v.Records[0].Revision, Changes: map[string]*string{"name": str("更正首项")}}).After
	if v.Records[1].Revision != last.Revision || v.Records[1].Data["name"] != "独立后项" {
		t.Fatal("rewrote independent successor")
	}
	v = apply(t, v, Command{Action: "ADD", AnchorID: first.ID, Date: "2005-01-01", Type: "CHANGE", Reason: "DESCRIPTION_ADJUSTMENT", Changes: map[string]*string{"description": str("中间插入")}}).After
	r := v.Records[1]
	v = apply(t, v, Command{Action: "DELETE", RecordID: r.ID, ExpectedRevision: r.Revision}).After
	if v.Records[1].Revision != last.Revision {
		t.Fatal("deletion rewrote independent successor")
	}
	for _, c := range []Command{{Action: "DELETE", RecordID: first.ID, ExpectedRevision: v.Records[0].Revision}, {Action: "REDATE", RecordID: first.ID, ExpectedRevision: v.Records[0].Revision, NewDate: "2010-01-01"}} {
		c.ExpectedSource = v.Source
		if _, e := Build(Family, v.Enterprise, v, c); e == nil {
			t.Fatal("invalid first decision accepted")
		}
	}
}
func TestStaffing307PairAndAdoption(t *testing.T) {
	v := create(t, Job, "2000-01-01")
	a := Adoption{FamilyID: v.Records[0].Data["jobFamilyId"], Date: "2000-01-01", Source: foundation.NewID(), RecordID: foundation.NewID(), Revision: foundation.NewID()}
	v.Records[0].Adoption = a
	p := apply(t, v, Command{Action: "ADD", AnchorID: v.Records[0].ID, Date: "2010-01-01", Type: "CHANGE", Reason: "NAME_ADJUSTMENT", Changes: map[string]*string{"name": str("同系列后项")}})
	if len(p.Adopt) != 0 || p.After.Records[1].Adoption != a {
		t.Fatal("same-family addition must inherit")
	}
	v = p.After
	p = apply(t, v, Command{Action: "INSERT_LIFECYCLE_PAIR", AnchorID: v.Records[0].ID, StopDate: "2002-01-01", StartDate: "2004-01-01", StopReason: "JOB_CLOSURE", StartReason: "JOB_RESUMPTION"})
	if len(p.Adopt) != 1 || p.After.Records[1].Adoption != a {
		t.Fatal("pair recovery must newly adopt")
	}
	v = p.After
	v.Records[2].Adoption = Adoption{FamilyID: a.FamilyID, Date: "2004-01-01", Source: foundation.NewID(), RecordID: foundation.NewID(), Revision: foundation.NewID()}
	original := v.Records[2].Adoption
	cmd := Command{Action: "MAINTAIN_LIFECYCLE_PAIR", StopID: v.Records[1].ID, StopRevision: v.Records[1].Revision, StartID: v.Records[2].ID, StartRevision: v.Records[2].Revision, StopDate: "2003-01-01", StartDate: "2004-01-01", StopReason: "JOB_CLOSURE", StartReason: "JOB_RESUMPTION"}
	p = apply(t, v, cmd)
	if len(p.Adopt) != 0 || p.After.Records[2].Adoption != original {
		t.Fatal("stop-only change replaced recovery evidence")
	}
	v = p.After
	cmd.Action = "DELETE_LIFECYCLE_PAIR"
	cmd.StopRevision = v.Records[1].Revision
	cmd.StartRevision = v.Records[2].Revision
	v = apply(t, v, cmd).After
	if len(v.Records) != 2 || v.Records[1].Adoption != a {
		t.Fatal("pair delete did not retain independent evidence")
	}
}
func TestStaffing307LifecycleInheritanceAndHiddenFields(t *testing.T) {
	v := create(t, Job, "2000-01-01")
	v = apply(t, v, Command{Action: "DEACTIVATE", AnchorID: v.Records[0].ID, Date: "2010-01-01", Type: "DEACTIVATE", Reason: "JOB_CLOSURE"}).After
	oldStop := v.Records[1]
	next := foundation.NewID()
	p := apply(t, v, Command{Action: "CORRECT", RecordID: v.Records[0].ID, ExpectedRevision: v.Records[0].Revision, Changes: map[string]*string{"jobFamilyId": &next}})
	if !slices.Contains(p.Fields, "jobFamilyId") || p.After.Records[1].Data["jobFamilyId"] != next || p.After.Records[1].Revision == oldStop.Revision {
		t.Fatal("implicit inherited change missing")
	}
	v = p.After
	p = apply(t, v, Command{Action: "REACTIVATE", AnchorID: v.Records[1].ID, Date: "2020-01-01", Type: "REACTIVATE", Reason: "JOB_RESUMPTION"})
	if len(p.Adopt) != 1 {
		t.Fatal("new recovery must adopt")
	}
	v = p.After
	c := Command{Action: "DELETE", ExpectedSource: v.Source, RecordID: v.Records[1].ID, ExpectedRevision: v.Records[1].Revision}
	if _, e := Build(Job, v.Enterprise, v, c); e == nil {
		t.Fatal("broken recovery sequence accepted")
	}
}
func TestStaffing307FullCoverage(t *testing.T) {
	f := create(t, Family, "2000-01-01")
	j := create(t, Job, "2001-01-01")
	j.Enterprise = f.Enterprise
	j.Records[0].Data["jobFamilyId"] = f.ID
	if e := ValidateCoverage(j, map[string]Timeline{f.ID: f}); e != nil {
		t.Fatal(e)
	}
	f = apply(t, f, Command{Action: "DEACTIVATE", AnchorID: f.Records[0].ID, Date: "2030-01-01", Type: "DEACTIVATE", Reason: "FAMILY_CLOSURE"}).After
	if e := ValidateCoverage(j, map[string]Timeline{f.ID: f}); e == nil {
		t.Fatal("finite family accepted open tail")
	}
	j = apply(t, j, Command{Action: "DEACTIVATE", AnchorID: j.Records[0].ID, Date: "2029-01-01", Type: "DEACTIVATE", Reason: "JOB_CLOSURE"}).After
	if e := ValidateCoverage(j, map[string]Timeline{f.ID: f}); e != nil {
		t.Fatal(e)
	}
	j = apply(t, j, Command{Action: "REACTIVATE", AnchorID: j.Records[1].ID, Date: "2031-01-01", Type: "REACTIVATE", Reason: "JOB_RESUMPTION"}).After
	if e := ValidateCoverage(j, map[string]Timeline{f.ID: f}); e == nil {
		t.Fatal("future restoration was ignored")
	}
}
func TestStaffing307ExplicitAndImplicitFieldSet(t *testing.T) {
	v := create(t, Job, "2000-01-01")
	p := apply(t, v, Command{Action: "ADD", AnchorID: v.Records[0].ID, Date: "2001-01-01", Type: "CHANGE", Reason: "NAME_ADJUSTMENT", Changes: map[string]*string{"name": str("新名称")}})
	if slices.Contains(p.Fields, "jobFamilyId") {
		t.Fatal("unchanged inherited field was treated as an implicit write")
	}
	if !slices.Contains(p.Fields, "name") {
		t.Fatal("explicit field omitted")
	}
}

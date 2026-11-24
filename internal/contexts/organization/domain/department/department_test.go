package department

import (
	"context"
	"errors"
	"testing"
)

const testBU = "10000000-0000-4000-8000-000000000001"
const otherBU = "10000000-0000-4000-8000-000000000002"

func row(id, date, end, name, parent string) Record {
	return Record{ID: id, Revision: id + "-v1", RevisionNo: 1, Date: date, End: end, Type: "CHANGE", Reason: "NAME_ADJUSTMENT", TypeLabel: "数据更改", ReasonLabel: "名称调整", State: "active", Data: Data{Name: name, BU: testBU, Parent: parent}}
}
func first(id, date, end, name, parent string) Record {
	r := row(id, date, end, name, parent)
	r.Type = "CREATE"
	r.Reason = "NEW_ORGANIZATION"
	r.TypeLabel = "新建"
	r.ReasonLabel = "新设组织"
	return r
}
func relation(t Timeline) Timeline {
	for i, r := range t.Records {
		if r.State != "active" {
			continue
		}
		ref := MemberRef{DepartmentID: t.ID, Source: t.ID + "-hs", MemberID: r.ID + "-member", BU: r.Data.BU, Start: r.Date, End: r.End, Parent: r.Data.Parent, EdgeID: r.ID + "-edge", Revision: r.ID + "-edge-v1"}
		t.Records[i].Data.Member = ref
		t.Edges = append(t.Edges, Edge{ID: ref.EdgeID, Revision: ref.Revision, Start: r.Date, End: r.End, BU: r.Data.BU, Parent: r.Data.Parent, Ref: ref})
	}
	return t
}
func TestRedundantAdditionDoesNotCreateRelationshipSource(t *testing.T) {
	ctx := context.Background()
	base := relation(Timeline{ID: "C", First: "first", Records: []Record{first("first", "2026-01-01", "", "组织发展部", "")}})
	name := "组织发展部"
	patch := Patch{"name": &name}
	it := Item{ID: "add", RecordID: "new", Action: "ADD", Date: "2026-06-01", Type: "CHANGE", Reason: "NAME_ADJUSTMENT", Changes: &patch}
	work, e := Kernel.Prepare(ctx, base.Records, base.First, "2020-01-01", []Item{it})
	if e != nil {
		t.Fatal(e)
	}
	plan := Plan(base, work.Records)
	if plan.Changed || len(plan.Edges) != 1 {
		t.Fatal("name-only same data split relationship")
	}
	work.Records, e = plan.Bind(work.Records)
	if e != nil {
		t.Fatal(e)
	}
	result, e := Kernel.Finalize(ctx, work)
	if e != nil {
		t.Fatal(e)
	}
	if !result.NoChange || len(result.Records) != 1 || result.Records[0].Data.Member.MemberID != "first-member" {
		t.Fatalf("redundant addition changed adoption: %+v", result)
	}
}
func TestFutureCycleAndHalfOpenCoverage(t *testing.T) {
	ctx := context.Background()
	a := Timeline{ID: "A", Records: []Record{first("a1", "2026-01-01", "2026-08-01", "甲", ""), row("a2", "2026-08-01", "", "甲", "B")}}
	b := Timeline{ID: "B", Records: []Record{first("b1", "2026-01-01", "2026-07-01", "乙", ""), row("b2", "2026-07-01", "", "乙", "A")}}
	s := Snapshot{Departments: []Timeline{a, b}, BusinessUnits: []BusinessUnit{{ID: testBU, Records: []BURecord{{Date: "2026-01-01", State: "active"}}}}}
	var q *QualificationError
	e := Validate(ctx, s, Discover(s, "A", nil, nil))
	if !errors.As(e, &q) || q.Code != "M02_DEPARTMENT_HIERARCHY_CYCLE" || q.Start != "2026-08-01" {
		t.Fatalf("future cycle not localized: %v", e)
	}
	a.Records = []Record{first("a1", "2026-01-01", "2026-06-01", "甲", "")}
	stop := row("as", "2026-06-01", "", "甲", "")
	stop.Type = "DEACTIVATE"
	stop.State = "inactive"
	a.Records = append(a.Records, stop)
	s.Departments = []Timeline{a}
	s.BusinessUnits[0].Records = []BURecord{{Date: "2026-01-01", End: "2026-06-01", State: "active"}, {Date: "2026-06-01", State: "inactive"}}
	if e = Validate(ctx, s, Discover(s, "A", nil, nil)); e != nil {
		t.Fatal("same day stop must not overlap", e)
	}
	s.Departments[0].Records[0].End = "2026-06-02"
	s.Departments[0].Records[1].Date = "2026-06-02"
	e = Validate(ctx, s, Discover(s, "A", nil, nil))
	if !errors.As(e, &q) || q.Start != "2026-06-01" {
		t.Fatal("one-day uncovered interval accepted")
	}
}
func TestReferenceProjectionDoesNotInventReason(t *testing.T) {
	ctx := context.Background()
	base := []Record{first("first", "2026-01-01", "", "名称", "")}
	patch := Patch{"managementBuId": func() *string { v := otherBU; return &v }()}
	item := Item{ID: "partial", RecordID: "new", Action: "ADD", Date: "2026-06-01", Type: "CHANGE", Changes: &patch}
	p, e := ReferenceKernel.PrepareReference(ctx, base, "first", "2020-01-01", []Item{item})
	if e != nil {
		t.Fatal(e)
	}
	if len(p.Records) != 2 || p.Records[0].End != "2026-06-01" || p.Records[1].Reason != "" || p.Records[1].ReasonLabel != "" || p.Records[1].Data.BU != otherBU {
		t.Fatalf("unexpected read projection: %+v", p)
	}
	if _, e = Kernel.Prepare(ctx, base, "first", "2020-01-01", []Item{item}); e == nil {
		t.Fatal("incomplete reason accepted by save")
	}
	item.Reason = "ORGANIZATION_REASSIGNMENT"
	w, e := Kernel.Prepare(ctx, base, "first", "2020-01-01", []Item{item})
	if e != nil {
		t.Fatal(e)
	}
	if w.Records[0].End != p.Records[0].End || w.Records[1].Date != p.Records[1].Date {
		t.Fatal("assistance duplicated a different date algorithm")
	}
}
func TestRelationDeltaSeparatesReassignmentFromLifecycle(t *testing.T) {
	a := first("first", "2026-01-01", "", "部门", "")
	stop := row("stop", "2026-06-01", "", "部门", "")
	stop.Type = "DEACTIVATE"
	stop.State = "inactive"
	b := a
	b.End = stop.Date
	if got := Deltas([]Record{a}, []Record{b, stop}); len(got) != 0 {
		t.Fatal("same relation lifecycle requires reassignment")
	}
	move := row("move", "2026-07-01", "", "部门", "")
	move.Data.BU = otherBU
	b.End = move.Date
	got := Deltas([]Record{b, move}, []Record{a})
	if len(got) != 1 || got[0].Start != "2026-07-01" || got[0].OldBU != otherBU || got[0].NewBU != testBU {
		t.Fatalf("implicit rollback not represented: %+v", got)
	}
}

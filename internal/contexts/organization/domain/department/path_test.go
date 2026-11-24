package department

import (
	"context"
	"reflect"
	"testing"
)

func pathFixture(id, parent string) Timeline {
	return Timeline{ID: id, Source: id + "-source", Records: []Record{{ID: id + "-record", Revision: id + "-revision", Date: "2020-01-01", State: "active", Data: Data{Name: id, BU: "bu", Parent: parent}}}, Edges: []Edge{{Start: "2020-01-01", BU: "bu", Parent: parent, Ref: MemberRef{DepartmentID: id, Source: "relation", MemberID: id}}}}
}
func TestRequestPathReuseKeepsDatesAndSnapshotsDistinct(t *testing.T) {
	p := pathFixture("p", "")
	c := pathFixture("c", "p")
	p.Records[0].End = "2021-01-01"
	next := p.Records[0]
	next.Date = "2021-01-01"
	next.End = ""
	next.ID = "p-new"
	next.Revision = "p-new-revision"
	next.Data.Name = "新上级"
	p.Records = append(p.Records, next)
	s := Snapshot{Departments: []Timeline{p, c, pathFixture("sibling", "p")}}.Indexed()
	r := NewPathResolver(s, "企业")
	old, e := r.At(context.Background(), "c", "2020-06-01")
	if e != nil {
		t.Fatal(e)
	}
	current, e := r.At(context.Background(), "c", "2022-01-01")
	if e != nil {
		t.Fatal(e)
	}
	if old[1].Name != "p" || current[1].Name != "新上级" || len(old) != 3 || old[0].Name != "企业" {
		t.Fatal("date-specific ancestry corrupted", old, current)
	}
	again, e := r.At(context.Background(), "c", "2020-06-01")
	if e != nil || !reflect.DeepEqual(old, again) {
		t.Fatal("cached date changed", again, e)
	}
	p.Source = "later-source"
	p.Records = append([]Record{}, p.Records...)
	p.Records[1].Data.Name = "再次更名"
	later := NewPathResolver(s.With(p), "企业")
	v, e := later.At(context.Background(), "c", "2022-01-01")
	if e != nil || v[1].Name != "再次更名" || current[1].Name != "新上级" {
		t.Fatal("request snapshots aliased", v, current, e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e = r.At(ctx, "c", "2020-06-01"); e == nil {
		t.Fatal("cancelled cache hit ignored cancellation")
	}
}
func TestRequestPathReuseRejectsCycleAndInconsistentEdge(t *testing.T) {
	a, b := pathFixture("a", "b"), pathFixture("b", "a")
	if _, e := NewPathResolver(Snapshot{Departments: []Timeline{a, b}}.Indexed(), "企业").At(context.Background(), "a", "2022-01-01"); e == nil {
		t.Fatal("cycle accepted")
	}
	a = pathFixture("a", "")
	a.Edges[0].Parent = "different"
	if _, e := PathAt(context.Background(), Snapshot{Departments: []Timeline{a}}.Indexed(), "a", "2022-01-01", "企业"); e == nil {
		t.Fatal("inconsistent relation accepted")
	}
}

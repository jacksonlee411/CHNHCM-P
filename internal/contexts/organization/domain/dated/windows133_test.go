package dated

import (
	"reflect"
	"slices"
	"testing"
)

func TestWindows133UnsortedInputAndDetachedResults(t *testing.T) {
	k := fixtureKernel()
	rows := fixtureBase()
	for _, v := range []struct{ id, date, typ, state string }{{"stop", "2026-03-01", "DEACTIVATE", "inactive"}, {"resume", "2026-04-01", "REACTIVATE", "active"}} {
		r := k.CloneRecords(rows[:1])[0]
		r.ID, r.Date, r.Type, r.State = v.id, v.date, v.typ, v.state
		rows = append(rows, r)
	}
	slices.Reverse(rows)
	original := k.CloneRecords(rows)
	addition, err := k.PrepareAddition(rows, "first", "2020-01-01", "2026-02-01")
	if err != nil || addition.Mode != "INSERT" || addition.Next != "stop" || addition.Max != "2026-02-28" {
		t.Fatalf("wrong adjacent insertion: %+v %v", addition, err)
	}
	appendWindow, err := k.PrepareLifecycleAppend(rows, "resume", "DEACTIVATE", "2020-01-01", "2026-04-02")
	if err != nil || !appendWindow.Available || appendWindow.Min != "2026-04-02" {
		t.Fatalf("wrong append: %+v %v", appendWindow, err)
	}
	stop, resume, err := k.LifecyclePair(rows, "resume")
	if err != nil || stop.ID != "stop" || resume.ID != "resume" {
		t.Fatal("pair selection changed", err)
	}
	lo, hi, err := k.CorrectionBounds(rows, "stop", "2020-01-01")
	if err != nil || lo != "2026-01-02" || hi != "2026-03-31" {
		t.Fatal("correction bounds changed", lo, hi, err)
	}
	for _, r := range []*Record[fixtureData]{&addition.Anchor, &appendWindow.Anchor, &stop, &resume} {
		r.Data.Ref.Revision = "changed"
		r.Data.Tags[0] = "changed"
		r.Data.Labels["key"] = "changed"
	}
	if !reflect.DeepEqual(rows, original) {
		t.Fatal("read-only capability preparation mutated caller data or order")
	}
	// Repeat using an ordered source: the borrowed directory must still not
	// escape through the returned mutable record data.
	slices.Reverse(rows)
	original = k.CloneRecords(rows)
	w, err := k.PrepareAddition(rows, "first", "2020-01-01", "")
	if err != nil {
		t.Fatal(err)
	}
	w.Anchor.Data.Tags[0] = "changed"
	if !reflect.DeepEqual(rows, original) {
		t.Fatal("ordered source was aliased")
	}
}

package dated

import (
	"context"
	"testing"

	"chnhcm.local/chnhcm/internal/foundation"
)

func TestObjectDateBoundary202(t *testing.T) {
	k := fixtureKernel()
	id := foundation.NewID()
	ctx := context.Background()
	// A directory can admit dates before the default; an enterprise-scoped
	// adapter supplies its own later floor to the very same kernel.
	for _, floor := range []string{"0001-01-01", "2020-01-01"} {
		wantOK := floor == "0001-01-01"
		_, e := k.Create(ctx, id, "1890-01-01", floor, "valid", fixtureData{Name: "original"})
		if (e == nil) != wantOK {
			t.Fatalf("create floor %s: %v", floor, e)
		}
		_, e = k.CreateReference(ctx, id, "1890-01-01", floor, fixtureData{Name: "original"})
		if (e == nil) != wantOK {
			t.Fatalf("reference floor %s: %v", floor, e)
		}
		base := []Record[fixtureData]{{ID: id, Revision: "v1", Date: "2021-01-01", Type: "CREATE", Reason: "valid", State: "active", Data: fixtureData{Name: "original"}}}
		_, e = k.Calculate(ctx, base, id, floor, []Item[fixturePatch]{{ID: "move", Action: "REDATE", RecordID: id, ExpectedRevision: "v1", Date: "1890-01-01"}})
		if (e == nil) != wantOK {
			t.Fatalf("redate floor %s: %v", floor, e)
		}
		min, _, e := k.CorrectionBounds(base, id, floor)
		if e != nil || min != floor {
			t.Fatalf("correction floor: %s %v", min, e)
		}
	}
	base := []Record[fixtureData]{{ID: id, Date: "1880-01-01", Type: "CREATE", State: "active", Data: fixtureData{Name: "original"}}}
	for _, floor := range []string{"0001-01-01", "1885-01-01"} {
		wantMin := "1880-01-02"
		if floor == "1885-01-01" {
			wantMin = floor
		}
		w, e := k.PrepareAddition(base, id, floor, "1890-01-01")
		if e != nil || w.Min != wantMin || w.Default != "1890-01-01" {
			t.Fatalf("addition: %+v %v", w, e)
		}
		w, e = k.PrepareLifecycleAppend(base, id, "DEACTIVATE", floor, "1890-01-01")
		if e != nil || w.Min != wantMin || w.Default != "1890-01-01" {
			t.Fatalf("lifecycle: %+v %v", w, e)
		}
		w, e = k.PrepareLifecyclePair(base, id, floor)
		if e != nil || w.Min != wantMin || !w.Available {
			t.Fatalf("pair: %+v %v", w, e)
		}
	}
	for _, date := range []string{"", "1900-02-29", "1900-1-1"} {
		if _, e := k.Create(ctx, id, date, "0001-01-01", "valid", fixtureData{}); e == nil {
			t.Fatalf("invalid date admitted: %q", date)
		}
	}
}

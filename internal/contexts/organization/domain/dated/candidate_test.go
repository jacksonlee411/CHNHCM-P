package dated

import (
	"context"
	"errors"
	"maps"
	"reflect"
	"slices"
	"testing"
)

// Synthetic relation data verifies extension points only, not Department runtime.
type relation struct{ Target, Revision string }
type fixtureData struct {
	Name, Group string
	Ref         *relation
	Tags        []string
	Labels      map[string]string
}
type fixturePatch struct {
	Name, Group *string
	Ref         *relation
}
type fixturePolicy struct{}

func (fixturePolicy) EmptyPatch(p fixturePatch) bool {
	return p.Name == nil && p.Group == nil && p.Ref == nil
}

func (p fixturePolicy) Normalize(d fixtureData) (fixtureData, error) { return p.Clone(d), nil }
func (fixturePolicy) Clone(d fixtureData) fixtureData {
	if d.Ref != nil {
		r := *d.Ref
		d.Ref = &r
	}
	d.Tags = slices.Clone(d.Tags)
	d.Labels = maps.Clone(d.Labels)
	return d
}
func (fixturePolicy) Apply(d fixtureData, p *fixturePatch) (fixtureData, error) {
	if p == nil || p.Name == nil && p.Group == nil && p.Ref == nil {
		return d, invalid("M02_INVALID_INPUT", "changes", "empty")
	}
	if p.Name != nil {
		d.Name = *p.Name
	}
	if p.Group != nil {
		d.Group = *p.Group
	}
	if p.Ref != nil {
		d.Ref = p.Ref
	}
	return d, nil
}
func (p fixturePolicy) EqualData(a, b fixtureData) bool {
	a = p.Clone(a)
	b = p.Clone(b)
	if a.Ref != nil {
		a.Ref.Revision = ""
	}
	if b.Ref != nil {
		b.Ref.Revision = ""
	}
	return reflect.DeepEqual(a, b)
}

// Only the exact relation adoption belongs to object-owned revision structure.
// Names, groups, tags and labels are business values compared by EqualData.
func (fixturePolicy) EqualRevision(a, b fixtureData) bool { return reflect.DeepEqual(a.Ref, b.Ref) }
func (fixturePolicy) OperationReasons(t string) map[string]string {
	return map[string]string{"valid": "frozen reason"}
}
func (fixturePolicy) OperationLabel(t string) string { return t }
func fixtureKernel() Kernel[fixtureData, fixturePatch] {
	return Kernel[fixtureData, fixturePatch]{Policy: fixturePolicy{}}
}
func fixtureBase() []Record[fixtureData] {
	return []Record[fixtureData]{{ID: "first", Revision: "first-v1", Date: "2026-01-01", Type: "CREATE", Reason: "retired", TypeLabel: "original type", ReasonLabel: "original reason", State: "active", Data: fixtureData{Name: "same", Group: "bu-a", Ref: &relation{"root-a", "edge-v1"}, Tags: []string{"tag"}, Labels: map[string]string{"key": "value"}}}}
}
func val(s string) *string { return &s }

func TestCandidateDetachesBaselinePatchAndSiblingSnapshots(t *testing.T) {
	k := fixtureKernel()
	base := fixtureBase()
	original := k.CloneRecords(base)
	patch := fixturePatch{Group: val("bu-b"), Ref: &relation{"root-b", "edge-v2"}}
	items := []Item[fixturePatch]{{ID: "change", Action: "ADD", Date: "2026-03-01", Type: "CHANGE", Reason: "valid", Changes: &patch}, {ID: "stop", Action: "ADD", Date: "2026-04-01", Type: "DEACTIVATE", Reason: "valid"}, {ID: "resume", Action: "ADD", Date: "2026-06-01", Type: "REACTIVATE", Reason: "valid"}}
	c, e := k.Calculate(context.Background(), base, "first", "2020-01-01", items)
	if e != nil || len(c.Records) != 4 {
		t.Fatalf("candidate %+v: %v", c, e)
	}
	if !reflect.DeepEqual(base, original) || patch.Ref.Revision != "edge-v2" {
		t.Fatal("calculation mutated inputs")
	}
	for i := range c.Records {
		c.Records[i].Data.Ref.Revision = "mutated"
		c.Records[i].Data.Tags[0] = "mutated"
		c.Records[i].Data.Labels["key"] = "mutated"
		if i+1 < len(c.Records) && c.Records[i+1].Data.Tags[0] != "tag" {
			t.Fatal("siblings share data")
		}
	}
	if !reflect.DeepEqual(base, original) || patch.Ref.Revision != "edge-v2" {
		t.Fatal("candidate aliases input")
	}
	if c.Records[1].Data.Group != "bu-b" || c.Records[2].Data.Group != "bu-b" || c.Records[3].Data.Group != "bu-b" {
		t.Fatal("relation fields not copied")
	}
}

func TestDeletedAndFactSnapshotsAreDetached(t *testing.T) {
	k := fixtureKernel()
	base := fixtureBase()
	base[0].End = "2026-03-01"
	next := k.CloneRecords(base)[0]
	next.ID = "next"
	next.Revision = "next-v1"
	next.Date = "2026-03-01"
	next.End = ""
	next.Type = "CHANGE"
	base = append(base, next)
	c, e := k.Calculate(context.Background(), base, "first", "2020-01-01", []Item[fixturePatch]{{ID: "delete", Action: "DELETE", RecordID: "next", ExpectedRevision: "next-v1"}})
	if e != nil {
		t.Fatal(e)
	}
	c.Deleted[0].Data.Labels["key"] = "deleted"
	c.Records[0].Data.Tags[0] = "current"
	r := k.FactAt(base, "2026-04-01")
	r.Data.Ref.Revision = "read"
	if base[0].Data.Tags[0] != "tag" || base[1].Data.Labels["key"] != "value" || base[1].Data.Ref.Revision != "edge-v1" {
		t.Fatal("snapshot alias")
	}
	if k.FactAt(base, "2025-12-31") != nil {
		t.Fatal("fact before first")
	}
}

func TestCandidateOrderValidationAndStableErrorIdentity(t *testing.T) {
	k := fixtureKernel()
	base := fixtureBase()
	items := []Item[fixturePatch]{{ID: "z-stop", Action: "ADD", Date: "2026-04-01", Type: "DEACTIVATE", Reason: "valid"}, {ID: "a-resume", Action: "ADD", Date: "2026-06-01", Type: "REACTIVATE", Reason: "valid"}}
	a, e := k.Calculate(context.Background(), base, "first", "2020-01-01", items)
	if e != nil {
		t.Fatal(e)
	}
	slices.Reverse(items)
	b, e := k.Calculate(context.Background(), base, "first", "2020-01-01", items)
	if e != nil || !reflect.DeepEqual(a, b) {
		t.Fatalf("input order matters: %v", e)
	}
	for _, patch := range []*fixturePatch{nil} {
		_, e = k.Calculate(context.Background(), base, "first", "2020-01-01", []Item[fixturePatch]{{ID: "empty", Action: "ADD", Date: "2026-03-01", Type: "CHANGE", Reason: "valid", Changes: patch}})
		var item *ItemFault
		if !errors.As(e, &item) || item.ItemID != "empty" {
			t.Fatalf("missing stable item identity: %v", e)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e = k.Calculate(ctx, base, "first", "2020-01-01", items); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
}

func TestCorrectionPreservesClassificationSnapshotsAndIndependentSuccessor(t *testing.T) {
	k := fixtureKernel()
	base := fixtureBase()
	base[0].End = "2026-06-01"
	next := k.CloneRecords(base)[0]
	next.ID = "next"
	next.Revision = "next-v1"
	next.Date = "2026-06-01"
	next.End = ""
	next.Type = "CHANGE"
	base = append(base, next)
	c, e := k.Calculate(context.Background(), base, "first", "2020-01-01", []Item[fixturePatch]{{ID: "correct", Action: "CORRECT", RecordID: "first", ExpectedRevision: "first-v1", Changes: &fixturePatch{Group: val("bu-b")}}})
	if e != nil {
		t.Fatal(e)
	}
	if c.Records[0].Reason != "retired" || c.Records[0].ReasonLabel != "original reason" || !reflect.DeepEqual(c.Records[1], base[1]) {
		t.Fatal("classification or independent successor rewritten")
	}
	want := []Range{{Start: "2026-01-01", End: val("2026-06-01")}}
	if !reflect.DeepEqual(c.Impact.DataAffectedRanges, want) {
		t.Fatalf("relationship ignored: %+v", c.Impact)
	}
}

func TestEmptyAdditionHasNoEffectButDoesNotBypassGuards(t *testing.T) {
	k := fixtureKernel()
	base := fixtureBase()
	item := Item[fixturePatch]{ID: "empty", Action: "ADD", Date: "2026-03-01", Type: "CHANGE", Reason: "valid", Changes: &fixturePatch{}}
	c, e := k.Calculate(context.Background(), base, "first", "2020-01-01", []Item[fixturePatch]{item})
	if e != nil || !c.NoChange || !reflect.DeepEqual(c.Records, base) || len(c.Impact.StructuralChanges) != 0 {
		t.Fatalf("copied defaults must be no_change: %+v %v", c, e)
	}
	item.Date = "2026-01-01"
	if _, e = k.Calculate(context.Background(), base, "first", "2020-01-01", []Item[fixturePatch]{item}); e == nil {
		t.Fatal("empty patch bypassed date conflict")
	}
	for _, action := range []string{"CORRECT", "UPDATE"} {
		it := Item[fixturePatch]{ID: "invalid-empty", Action: action, RecordID: "first", ExpectedRevision: "first-v1", Changes: &fixturePatch{}}
		if _, e = k.Calculate(context.Background(), base, "first", "2020-01-01", []Item[fixturePatch]{it}); e == nil {
			t.Fatal("empty correction accepted", action)
		}
	}
	stop, e := k.Calculate(context.Background(), base, "first", "2020-01-01", []Item[fixturePatch]{{ID: "stop", Action: "ADD", Date: "2026-04-01", Type: "DEACTIVATE", Reason: "valid"}})
	if e != nil {
		t.Fatal(e)
	}
	for _, kind := range []string{"CHANGE", "REACTIVATE"} {
		item.Date = "2026-06-01"
		item.Type = kind
		if _, e = k.Calculate(context.Background(), stop.Records, "first", "2020-01-01", []Item[fixturePatch]{item}); e == nil {
			t.Fatal("empty patch bypassed lifecycle or reactivation validation", kind)
		}
	}
}

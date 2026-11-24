package domain

import (
	"context"
	"reflect"
	"testing"
)

func TestLifecyclePairWindowAndCurrentPair(t *testing.T) {
	for _, tc := range []struct {
		left, right string
		available   bool
		min, max    string
	}{{"2026-06-01", "2026-06-04", true, "2026-06-02", "2026-06-03"}, {"2026-06-01", "2026-06-03", false, "", ""}, {"2026-06-01", "2026-06-02", false, "", ""}, {"2028-02-27", "2028-03-01", true, "2028-02-28", "2028-02-29"}, {"9999-12-29", "", true, "9999-12-30", "9999-12-31"}, {"9999-12-30", "", false, "", ""}} {
		base := []Record{{ID: "left", Date: tc.left, State: "active"}}
		if tc.right != "" {
			base = append(base, Record{ID: "right", Date: tc.right, State: "active"})
		}
		w, e := PrepareLifecyclePair(base, "left", "2025-01-01")
		if e != nil || w.Available != tc.available || tc.available && (w.Min != tc.min || w.Max != tc.max) {
			t.Fatalf("%+v: %+v %v", tc, w, e)
		}
	}
	for _, state := range []string{"active", "inactive"} {
		w, e := PrepareLifecyclePair([]Record{{ID: "future", Date: "2028-02-29", State: state}}, "future", "2025-01-01")
		if e != nil || w.Available != (state == "active") || state == "active" && w.Min != "2028-03-01" {
			t.Fatal("future first / inactive gap", w, e)
		}
	}
	base := []Record{{ID: "first", Type: "CREATE", Date: "2026-01-01"}, {ID: "s1", Type: "DEACTIVATE", Date: "2026-03-01"}, {ID: "r1", Type: "REACTIVATE", Date: "2026-04-01"}, {ID: "s2", Type: "DEACTIVATE", Date: "2026-07-01"}, {ID: "r2", Type: "REACTIVATE", Date: "2026-08-01"}}
	for _, id := range []string{"s1", "r1"} {
		s, r, e := LifecyclePair(base, id)
		if e != nil || s.ID != "s1" || r.ID != "r1" {
			t.Fatal(id, s, r, e)
		}
	}
	for _, id := range []string{"first", "missing"} {
		if _, _, e := LifecyclePair(base, id); e == nil {
			t.Fatal("accepted non-pair", id)
		}
	}
	if _, _, e := LifecyclePair(base[:4], "s2"); e == nil {
		t.Fatal("paired with previous cycle")
	}
}
func TestLifecyclePairInsertionAndInheritedDeletionChanges(t *testing.T) {
	ctx := context.Background()
	base := []Record{{ID: "f", Revision: "fr", Date: "2026-01-01", Type: "CREATE", Reason: "NEW_ORGANIZATION", TypeLabel: "新建", ReasonLabel: "新设组织", State: "active", Data: BUData{Name: "甲"}}}
	c, e := Calculate(ctx, base, "f", "2025-01-01", []Item{{ID: "s", Action: "ADD", Date: "2026-03-01", Type: "DEACTIVATE", Reason: "BUSINESS_CLOSURE"}, {ID: "r", Action: "ADD", Date: "2026-04-01", Type: "REACTIVATE", Reason: "BUSINESS_RESUMPTION"}})
	if e != nil || len(c.Records) != 3 || c.NoChange || c.Records[1].State != "inactive" || c.Records[2].Data.Name != "甲" || c.Records[2].InheritedRecord != "" {
		t.Fatal(c, e)
	}
	c.Records[1].Revision = "sr"
	c.Records[2].Revision = "rr"
	same, e := Calculate(ctx, c.Records, "f", "2025-01-01", []Item{{ID: "ds", Action: "DELETE", RecordID: "s", ExpectedRevision: "sr"}, {ID: "dr", Action: "DELETE", RecordID: "r", ExpectedRevision: "rr"}})
	if e != nil {
		t.Fatal(e)
	}
	noNameChanges, e := PairNameChanges(ctx, c.Records, same.Records)
	if e != nil || len(noNameChanges) != 0 {
		t.Fatal("same names generated an unnecessary change notice", noNameChanges, e)
	}
	c.Records[2].Data.Name = "乙"
	c2, e := Calculate(ctx, c.Records, "f", "2025-01-01", []Item{{ID: "s2", Action: "ADD", Date: "2026-07-01", Type: "DEACTIVATE", Reason: "BUSINESS_CLOSURE"}, {ID: "r2", Action: "ADD", Date: "2026-08-01", Type: "REACTIVATE", Reason: "BUSINESS_RESUMPTION", Changes: PatchFromFields(map[string]*string{"name": textValue("丙")})}})
	if e != nil {
		t.Fatal(e)
	}
	original := append([]Record(nil), c2.Records...)
	deleted, e := Calculate(ctx, c2.Records, "f", "2025-01-01", []Item{{ID: "ds", Action: "DELETE", RecordID: "s", ExpectedRevision: "sr"}, {ID: "dr", Action: "DELETE", RecordID: "r", ExpectedRevision: "rr"}})
	if e != nil {
		t.Fatal(e)
	}
	changes, e := PairNameChanges(ctx, original, deleted.Records)
	if e != nil || len(changes) != 1 || changes[0].Start != "2026-04-01" || changes[0].End != "2026-08-01" || changes[0].Before.Data.Name != "乙" || changes[0].After.Data.Name != "甲" {
		t.Fatal(changes, e)
	}
	if deleted.Records[1].ID != "s2" || deleted.Records[1].Data.Name != "甲" || deleted.Records[2].Data.Name != "丙" || !reflect.DeepEqual(original, c2.Records) {
		t.Fatal("snapshot or inherited state changed incorrectly")
	}
	for _, dates := range [][2]string{{"2026-03-01", "2026-03-01"}, {"2026-04-01", "2026-03-01"}, {"2026-01-01", "2026-02-01"}} {
		if e := ValidateLifecyclePair(base, "f", "2025-01-01", dates[0], dates[1]); e == nil {
			t.Fatal("invalid pair accepted", dates)
		}
	}
}

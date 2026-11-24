package domain

import (
	"context"
	"reflect"
	"slices"
	"testing"
)

func TestT07PartialFieldsNoChangeAndRetiredReason(t *testing.T) {
	for _, tc := range []struct {
		name     string
		changes  map[string]*string
		wantName string
		wantEn   *string
		noChange bool
	}{
		{"TL053_clear", map[string]*string{"nameEn": nil}, "乙", nil, false},
		{"TL053_omit", map[string]*string{"name": textValue("新")}, "新", textValue("Beta"), false},
		{"TL054_same", map[string]*string{"name": textValue("乙"), "nameEn": textValue("Beta")}, "乙", textValue("Beta"), true},
		{"TL054_normalized_same", map[string]*string{"name": textValue("　乙 "), "nameEn": textValue(" Beta　")}, "乙", textValue("Beta"), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := baseline()
			original := baseline()
			got, e := Calculate(context.Background(), base, "first", "2020-01-01", []Item{{ID: "fix", Action: "CORRECT", RecordID: "june", ExpectedRevision: "r2", Changes: PatchFromFields(tc.changes)}})
			if e != nil {
				t.Fatal(e)
			}
			r := got.Records[1]
			if got.NoChange != tc.noChange || r.Data.Name != tc.wantName || !reflect.DeepEqual(r.Data.NameEn, tc.wantEn) || r.ID != "june" || r.Date != "2026-06-01" || r.End != "2026-10-01" || r.Type != "CHANGE" || !reflect.DeepEqual(base, original) || !reflect.DeepEqual(got.Records[2], base[2]) {
				t.Fatal(got)
			}
			if tc.noChange && (len(got.Impact.DataAffectedRanges) != 0 || len(got.Impact.StructuralChanges) != 0) {
				t.Fatal("same values changed facts")
			}
		})
	}
	base := baseline()
	n, e := Calculate(context.Background(), base, "first", "2020-01-01", []Item{lifeItem("same", "2026-03-01", "CHANGE", "NAME_ADJUSTMENT", map[string]*string{"name": textValue("甲")})})
	if e != nil || !n.NoChange || !reflect.DeepEqual(n.Records, base) {
		t.Fatal("TL040 same addition reserved a day", n, e)
	}
	w, e := Calculate(context.Background(), n.Records, "first", "2020-01-01", []Item{lifeItem("real", "2026-03-01", "CHANGE", "NAME_ADJUSTMENT", map[string]*string{"name": textValue("新")})})
	if e != nil || w.NoChange || len(w.Records) != 4 || w.Records[1].Data.Name != "新" || *w.Records[1].Data.NameEn != "Alpha" {
		t.Fatal(w, e)
	}
	// Current directory deliberately has no retired code. A correction retains the
	// exact old classification rather than consulting that directory.
	base[1].Reason = "RETIRED_REASON"
	base[1].TypeLabel = "历史资料变更"
	base[1].ReasonLabel = "旧原因名称"
	got, e := Calculate(context.Background(), base, "first", "2020-01-01", []Item{{ID: "fix", Action: "CORRECT", RecordID: "june", ExpectedRevision: "r2", Changes: PatchFromFields(map[string]*string{"name": textValue("新")})}})
	if e != nil || got.Records[1].Reason != "RETIRED_REASON" || got.Records[1].ReasonLabel != "旧原因名称" || got.Records[1].TypeLabel != "历史资料变更" {
		t.Fatal("TL058 historical classification changed", got, e)
	}
}

func TestT07ReactivationPartialFieldsAndF10(t *testing.T) {
	f := Record{ID: "f", Revision: "f-v1", Date: "2026-01-01", End: "2026-04-01", Type: "CREATE", State: "active", Data: BUData{Name: "甲", NameEn: textValue("Alpha")}}
	s := Record{ID: "s", Revision: "s-v1", Date: "2026-04-01", Type: "DEACTIVATE", State: "inactive", InheritedRecord: "f", InheritedRevision: "f-v1", Data: BUData{Name: "甲", NameEn: textValue("Alpha")}}
	for _, tc := range []struct {
		name     string
		changes  map[string]*string
		wantName string
		wantEn   *string
	}{
		{"TL062_Chinese_only", map[string]*string{"name": textValue("新")}, "新", textValue("Alpha")},
		{"TL062_English_null", map[string]*string{"nameEn": nil}, "甲", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, e := Calculate(context.Background(), []Record{f, s}, "f", "2020-01-01", []Item{lifeItem("r", "2026-05-01", "REACTIVATE", "BUSINESS_RESUMPTION", tc.changes)})
			if e != nil {
				t.Fatal(e)
			}
			r := got.Records[2]
			if r.Data.Name != tc.wantName || !reflect.DeepEqual(r.Data.NameEn, tc.wantEn) || r.InheritedRecord != "" || r.State != "active" || r.Type != "REACTIVATE" || got.Records[1].End != "2026-05-01" {
				t.Fatal(got)
			}
		})
	}
	// Exact F10 values, independently written expected facts and ranges.
	base := deletionFixture(
		f,
		Record{ID: "s1", Date: "2026-04-01", End: "2026-05-01", Type: "DEACTIVATE", State: "inactive", InheritedRecord: "f", InheritedRevision: "f-v1", Data: BUData{Name: "甲", NameEn: textValue("Alpha")}},
		Record{ID: "r1", Date: "2026-05-01", End: "2026-08-01", Type: "REACTIVATE", State: "active", Data: BUData{Name: "乙", NameEn: textValue("Beta")}},
		Record{ID: "s2", Date: "2026-08-01", End: "2026-09-01", Type: "DEACTIVATE", State: "inactive", InheritedRecord: "r1", InheritedRevision: "r1-v1", Data: BUData{Name: "乙", NameEn: textValue("Beta")}},
		Record{ID: "r2", Date: "2026-09-01", End: "2026-11-01", Type: "REACTIVATE", State: "active", Data: BUData{Name: "丙", NameEn: textValue("Gamma")}},
		Record{ID: "c", Date: "2026-11-01", Type: "CHANGE", State: "active", Data: BUData{Name: "新", NameEn: textValue("Zulu")}},
	)
	got, e := Calculate(context.Background(), base, "f", "2020-01-01", []Item{{ID: "fix", Action: "CORRECT", RecordID: "r1", ExpectedRevision: "r1-v1", Changes: PatchFromFields(map[string]*string{"name": textValue("新")})}})
	if e != nil {
		t.Fatal(e)
	}
	if got.Records[2].Data.Name != "新" || *got.Records[2].Data.NameEn != "Beta" || got.Records[3].Data.Name != "新" || *got.Records[3].Data.NameEn != "Beta" || got.Records[3].InheritedRevision != "" || !reflect.DeepEqual(got.Records[4:], base[4:]) {
		t.Fatal("TL048 F10", got)
	}
	found := false
	for _, x := range got.Impact.StructuralChanges {
		if x.RecordID == "s2" {
			found = slices.Contains(x.Facets, "INHERITANCE") && x.AfterRevision == nil
		}
	}
	if !found || !reflect.DeepEqual(got.Impact.DataAffectedRanges, []Range{{Start: "2026-05-01", End: textValue("2026-09-01")}}) {
		t.Fatal(got.Impact)
	}
	// TL073 each incremental step must produce exactly the hand-written F10 prefix.
	current := []Record{base[0]}
	current[0].End = ""
	for i, r := range base[1:] {
		var changes map[string]*string
		if r.Type != "DEACTIVATE" {
			changes = map[string]*string{"name": textValue(r.Data.Name), "nameEn": r.Data.NameEn}
		}
		reason := "NAME_ADJUSTMENT"
		if r.Type == "DEACTIVATE" {
			reason = "BUSINESS_CLOSURE"
		}
		if r.Type == "REACTIVATE" {
			reason = "BUSINESS_RESUMPTION"
		}
		next, e := Calculate(context.Background(), current, "f", "2020-01-01", []Item{lifeItem(r.ID, r.Date, r.Type, reason, changes)})
		if e != nil || next.NoChange || len(next.Records) != i+2 {
			t.Fatal(i, e)
		}
		for j, a := range next.Records {
			want := base[j]
			end := want.End
			if j == len(next.Records)-1 {
				end = ""
			}
			if a.Data.Name != want.Data.Name || !reflect.DeepEqual(a.Data.NameEn, want.Data.NameEn) || a.Date != want.Date || a.End != end || a.Type != want.Type || a.State != want.State {
				t.Fatal(i, j, a, want)
			}
		}
		current = next.Records
	}
}

func TestT07EqualSnapshotsSwappingDatesStillChangesStructure(t *testing.T) {
	base := baseline()
	base[2].Data.Name = "乙"
	base[2].Data.NameEn = textValue("Beta")
	original := append([]Record(nil), base...)
	got, e := Calculate(context.Background(), base, "first", "2020-01-01", []Item{
		{ID: "left", Action: "UPDATE", RecordID: "june", ExpectedRevision: "r2", Date: "2026-10-01"},
		{ID: "right", Action: "UPDATE", RecordID: "oct", ExpectedRevision: "r3", Date: "2026-06-01"},
	})
	if e != nil {
		t.Fatal(e)
	}
	if got.NoChange || len(got.Impact.DataAffectedRanges) != 0 || got.Records[1].ID != "oct" || got.Records[2].ID != "june" || !reflect.DeepEqual(base, original) {
		t.Fatal(got)
	}
	for _, id := range []string{"june", "oct"} {
		found := false
		for _, s := range got.Impact.StructuralChanges {
			if s.RecordID == id && slices.Contains(s.Facets, "REDATE") {
				found = true
			}
		}
		if !found {
			t.Fatal("missing date structure", id, got.Impact)
		}
	}
}

func TestT07SingleFirstRedateDoesNotCompleteOldDate(t *testing.T) {
	first := Record{ID: "f", Revision: "v1", Date: "2026-06-01", Type: "CREATE", State: "active", Data: BUData{Name: "乙", NameEn: textValue("Beta")}}
	got, e := Calculate(context.Background(), []Record{first}, "f", "2020-01-01", []Item{{ID: "move", Action: "REDATE", RecordID: "f", ExpectedRevision: "v1", Date: "2026-01-01"}})
	if e != nil {
		t.Fatal(e)
	}
	if got.NoChange || len(got.Records) != 1 || got.Records[0].ID != "f" || got.Records[0].Date != "2026-01-01" || got.Records[0].End != "" || got.Records[0].Data.Name != "乙" || *got.Records[0].Data.NameEn != "Beta" {
		t.Fatal(got)
	}
	if !reflect.DeepEqual(got.Impact.DataAffectedRanges, []Range{{Start: "2026-01-01", End: textValue("2026-06-01")}}) {
		t.Fatal(got.Impact)
	}
}

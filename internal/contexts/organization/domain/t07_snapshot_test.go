package domain

import (
	"context"
	"reflect"
	"testing"
)

func TestT07F11ExplicitEnglishCorrectionDoesNotPropagate(t *testing.T) {
	base := deletionFixture(
		Record{ID: "f", Date: "2026-01-01", End: "2026-06-01", Type: "CREATE", State: "active", Data: BUData{Name: "甲", NameEn: textValue("Nroth")}},
		Record{ID: "c1", Date: "2026-06-01", End: "2026-07-01", Type: "CHANGE", State: "active", Data: BUData{Name: "乙", NameEn: textValue("Nroth")}},
		Record{ID: "s1", Date: "2026-07-01", End: "2026-08-01", Type: "DEACTIVATE", State: "inactive", InheritedRecord: "c1", InheritedRevision: "c1-v1", Data: BUData{Name: "乙", NameEn: textValue("Nroth")}},
		Record{ID: "r1", Date: "2026-08-01", Type: "REACTIVATE", State: "active", Data: BUData{Name: "乙", NameEn: textValue("Nroth")}},
	)
	original := append([]Record(nil), base...)
	for _, tc := range []struct {
		name    string
		ids     []string
		english []string
		end     string
	}{
		{"TL050_first_only", []string{"f"}, []string{"North", "Nroth", "Nroth", "Nroth"}, "2026-06-01"},
		{"TL051_first_and_c1", []string{"f", "c1"}, []string{"North", "North", "North", "Nroth"}, "2026-08-01"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			items := []Item{}
			for _, id := range tc.ids {
				items = append(items, Item{ID: id + "-item", Action: "UPDATE", RecordID: id, ExpectedRevision: id + "-v1", Changes: PatchFromFields(map[string]*string{"nameEn": textValue("North")})})
			}
			got, e := Calculate(context.Background(), base, "f", "2020-01-01", items)
			if e != nil {
				t.Fatal(e)
			}
			if len(got.Records) != 4 || got.NoChange {
				t.Fatal("wrong membership/outcome")
			}
			for i, r := range got.Records {
				if *r.Data.NameEn != tc.english[i] || r.Data.Name != base[i].Data.Name || r.Date != base[i].Date || r.End != base[i].End || r.Type != base[i].Type || r.State != base[i].State {
					t.Fatalf("record %d %+v", i, r)
				}
			}
			if !reflect.DeepEqual(got.Impact.DataAffectedRanges, []Range{{Start: "2026-01-01", End: textValue(tc.end)}}) {
				t.Fatal("wrong affected periods", got.Impact)
			}
			if !reflect.DeepEqual(base, original) || !reflect.DeepEqual(got.Records[3], base[3]) {
				t.Fatal("old evidence or independent reactivation changed")
			}
			if len(tc.ids) == 2 && (got.Records[2].InheritedRecord != "c1" || got.Records[2].InheritedRevision != "") {
				t.Fatal("stop did not inherit new exact c1 revision placeholder")
			}
		})
	}
}

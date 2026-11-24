package domain

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

func deletionFixture(rows ...Record) []Record {
	// Fixture metadata, not an expected-result calculation.
	for i := range rows {
		r := &rows[i]
		r.Revision = r.ID + "-v1"
		r.RevisionNo = 1
		r.TypeLabel = OperationLabel(r.Type)
		r.Reason = map[string]string{"CREATE": "NEW_ORGANIZATION", "CHANGE": "NAME_ADJUSTMENT", "DEACTIVATE": "BUSINESS_CLOSURE", "REACTIVATE": "BUSINESS_RESUMPTION"}[r.Type]
		r.ReasonLabel = OperationReasons(r.Type)[r.Reason]
	}
	return rows
}
func deleteItem(r Record) Item {
	return Item{ID: "delete", Action: "DELETE", RecordID: r.ID, ExpectedRevision: r.Revision}
}
func deleteCycle() []Record {
	return deletionFixture(
		Record{ID: "f", Date: "2026-01-01", End: "2026-04-01", Type: "CREATE", State: "active", Data: BUData{Name: "甲", NameEn: textValue("Alpha")}},
		Record{ID: "s", Date: "2026-04-01", End: "2026-08-01", Type: "DEACTIVATE", State: "inactive", InheritedRecord: "f", InheritedRevision: "f-v1", Data: BUData{Name: "甲", NameEn: textValue("Alpha")}},
		Record{ID: "r", Date: "2026-08-01", End: "2026-10-01", Type: "REACTIVATE", State: "active", Data: BUData{Name: "乙", NameEn: textValue("Beta")}},
		Record{ID: "c", Date: "2026-10-01", End: "2026-12-01", Type: "CHANGE", State: "active", Data: BUData{Name: "丙", NameEn: textValue("Gamma")}},
		Record{ID: "s2", Date: "2026-12-01", Type: "DEACTIVATE", State: "inactive", InheritedRecord: "c", InheritedRevision: "c-v1", Data: BUData{Name: "丙", NameEn: textValue("Gamma")}},
	)
}
func TestDeleteIndependentPeriodsAndLifecycle(t *testing.T) {
	ordinary := baseline()
	stopped := deleteCycle()[:2]
	stopped[1].End = ""
	resumed := deleteCycle()[:3]
	resumed[1].End = "2026-05-01"
	resumed[2].Date = "2026-05-01"
	resumed[2].End = ""
	for _, tc := range []struct {
		name   string
		base   []Record
		target int
		want   []struct{ id, date, end, name, en, state string }
		ranges []Range
	}{
		{"middle", ordinary, 1, []struct{ id, date, end, name, en, state string }{{"first", "2026-01-01", "2026-10-01", "甲", "Alpha", "active"}, {"oct", "2026-10-01", "", "丙", "Gamma", "active"}}, []Range{{Start: "2026-06-01", End: textValue("2026-10-01")}}},
		{"tail", ordinary, 2, []struct{ id, date, end, name, en, state string }{{"first", "2026-01-01", "2026-06-01", "甲", "Alpha", "active"}, {"june", "2026-06-01", "", "乙", "Beta", "active"}}, []Range{{Start: "2026-10-01", End: nil}}},
		{"tail_stop", stopped, 1, []struct{ id, date, end, name, en, state string }{{"f", "2026-01-01", "", "甲", "Alpha", "active"}}, []Range{{Start: "2026-04-01", End: nil}}},
		{"tail_reactivation", resumed, 2, []struct{ id, date, end, name, en, state string }{{"f", "2026-01-01", "2026-04-01", "甲", "Alpha", "active"}, {"s", "2026-04-01", "", "甲", "Alpha", "inactive"}}, []Range{{Start: "2026-05-01", End: nil}}},
		{"ordinary_before_stop", deleteCycle(), 3, []struct{ id, date, end, name, en, state string }{{"f", "2026-01-01", "2026-04-01", "甲", "Alpha", "active"}, {"s", "2026-04-01", "2026-08-01", "甲", "Alpha", "inactive"}, {"r", "2026-08-01", "2026-12-01", "乙", "Beta", "active"}, {"s2", "2026-12-01", "", "乙", "Beta", "inactive"}}, []Range{{Start: "2026-10-01", End: nil}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := append([]Record(nil), tc.base...)
			target := tc.base[tc.target]
			c, e := Calculate(context.Background(), tc.base, tc.base[0].ID, "2020-01-01", []Item{deleteItem(target)})
			if e != nil {
				t.Fatal(e)
			}
			if c.NoChange || len(c.Records) != len(tc.want) || !reflect.DeepEqual(c.Deleted, []Record{target}) || !reflect.DeepEqual(c.Impact.DataAffectedRanges, tc.ranges) {
				t.Fatal("candidate/evidence/impact", c)
			}
			for i, w := range tc.want {
				r := c.Records[i]
				if r.ID != w.id || r.Date != w.date || r.End != w.end || r.Data.Name != w.name || *r.Data.NameEn != w.en || r.State != w.state {
					t.Fatal(i, r, w)
				}
			}
			if tc.name == "ordinary_before_stop" && (c.Records[3].InheritedRecord != "r" || c.Records[3].InheritedRevision != "") {
				t.Fatal("old or unbound predecessor", c.Records[3])
			}
			if !reflect.DeepEqual(tc.base, before) {
				t.Fatal("mutated original")
			}
			// D before/on/after the target makes the same record future/today/past.
			day, _ := time.Parse("2006-01-02", target.Date)
			wantFact := tc.want[tc.target-1]
			for _, offset := range []int{-1, 0, 1} {
				d := day.AddDate(0, 0, offset).Format("2006-01-02")
				r := FactAt(c.Records, d)
				if r == nil || r.Data.Name != wantFact.name || *r.Data.NameEn != wantFact.en || r.State != wantFact.state {
					t.Fatal("P/T/F resulting fact", d, r, wantFact)
				}
			}
		})
	}
}
func TestDeleteFullSuccessorAndTargetRejections(t *testing.T) {
	for _, tc := range []struct {
		name string
		base []Record
		it   Item
		code string
	}{
		{"stop_with_reactivation", deleteCycle(), deleteItem(deleteCycle()[1]), "M02_TIMELINE_INVALID"},
		{"reactivation_with_change", deleteCycle(), deleteItem(deleteCycle()[2]), "M02_TIMELINE_INVALID"},
		{"first_only", baseline()[:1], deleteItem(baseline()[0]), "M02_FIRST_RECORD_PROTECTED"},
		{"first_with_successor", baseline(), deleteItem(baseline()[0]), "M02_FIRST_RECORD_PROTECTED"},
		{"expired", baseline(), Item{ID: "delete", Action: "DELETE", RecordID: "june", ExpectedRevision: "stale"}, "M02_REVISION_CONFLICT"},
		{"missing", baseline(), Item{ID: "delete", Action: "DELETE", RecordID: "other"}, "M02_RECORD_NOT_FOUND"},
		{"extra_field", baseline(), Item{ID: "delete", Action: "DELETE", RecordID: "june", ExpectedRevision: "r2", Changes: PatchFromFields(map[string]*string{})}, "M02_FORBIDDEN_INPUT"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, e := Calculate(context.Background(), tc.base, tc.base[0].ID, "2020-01-01", []Item{tc.it})
			if deletionFault(e) != tc.code {
				t.Fatal(e)
			}
		})
	}
	// A reactivation before a second stop cannot be removed, even without CHANGE.
	b := deleteCycle()
	b = append(b[:3], b[4])
	b[2].End = "2026-12-01"
	b[3].Data.Name = "乙"
	b[3].Data.NameEn = textValue("Beta")
	b[3].InheritedRecord = "r"
	b[3].InheritedRevision = "r-v1"
	if _, e := Calculate(context.Background(), b, "f", "2020-01-01", []Item{deleteItem(b[2])}); deletionFault(e) != "M02_TIMELINE_INVALID" {
		t.Fatal(e)
	}
	future := deletionFixture(Record{ID: "future", Date: "2026-07-01", Type: "CREATE", State: "active", Data: BUData{Name: "乙"}})
	if _, e := Calculate(context.Background(), future, "future", "2020-01-01", []Item{deleteItem(future[0])}); deletionFault(e) != "M02_FIRST_RECORD_PROTECTED" {
		t.Fatal(e)
	}
}
func TestDeleteExactF5AndF10SuccessorDependencies(t *testing.T) {
	f5 := deletionFixture(
		Record{ID: "f", Date: "2026-01-01", End: "2026-04-01", Type: "CREATE", State: "active", Data: BUData{Name: "甲", NameEn: textValue("Alpha")}},
		Record{ID: "s1", Date: "2026-04-01", End: "2026-05-01", Type: "DEACTIVATE", State: "inactive", InheritedRecord: "f", InheritedRevision: "f-v1", Data: BUData{Name: "甲", NameEn: textValue("Alpha")}},
		Record{ID: "r1", Date: "2026-05-01", End: "2026-06-01", Type: "REACTIVATE", State: "active", Data: BUData{Name: "乙", NameEn: textValue("Beta")}},
		Record{ID: "c1", Date: "2026-06-01", Type: "CHANGE", State: "active", Data: BUData{Name: "丙", NameEn: textValue("Gamma")}},
	)
	f10 := deletionFixture(
		Record{ID: "f", Date: "2026-01-01", End: "2026-04-01", Type: "CREATE", State: "active", Data: BUData{Name: "甲", NameEn: textValue("Alpha")}},
		Record{ID: "s1", Date: "2026-04-01", End: "2026-05-01", Type: "DEACTIVATE", State: "inactive", InheritedRecord: "f", InheritedRevision: "f-v1", Data: BUData{Name: "甲", NameEn: textValue("Alpha")}},
		Record{ID: "r1", Date: "2026-05-01", End: "2026-08-01", Type: "REACTIVATE", State: "active", Data: BUData{Name: "乙", NameEn: textValue("Beta")}},
		Record{ID: "s2", Date: "2026-08-01", End: "2026-09-01", Type: "DEACTIVATE", State: "inactive", InheritedRecord: "r1", InheritedRevision: "r1-v1", Data: BUData{Name: "乙", NameEn: textValue("Beta")}},
		Record{ID: "r2", Date: "2026-09-01", End: "2026-11-01", Type: "REACTIVATE", State: "active", Data: BUData{Name: "丙", NameEn: textValue("Gamma")}},
		Record{ID: "c1", Date: "2026-11-01", Type: "CHANGE", State: "active", Data: BUData{Name: "新", NameEn: textValue("New")}},
	)
	for _, fixture := range []struct {
		name string
		base []Record
		date string
	}{{"TL-082/F5-with-CHANGE", f5, "2026-06-01"}, {"TL-083/F10", f10, "2026-08-01"}} {
		t.Run(fixture.name, func(t *testing.T) {
			before := append([]Record(nil), fixture.base...)
			_, err := Calculate(context.Background(), fixture.base, "f", "2025-01-01", []Item{deleteItem(fixture.base[2])})
			var fault *foundation.Fault
			if !errors.As(err, &fault) || fault.Code != "M02_TIMELINE_INVALID" || !strings.Contains(fault.Message, fixture.date) || !reflect.DeepEqual(before, fixture.base) {
				t.Fatal("exact dependent successor/original source", err)
			}
		})
	}
}

func TestDeleteSameValuePredecessorIsStructural(t *testing.T) {
	b := deletionFixture(
		Record{ID: "f", Date: "2026-01-01", End: "2026-03-01", Type: "CREATE", State: "active", Data: BUData{Name: "乙", NameEn: textValue("Beta")}},
		Record{ID: "c", Date: "2026-03-01", End: "2026-05-01", Type: "CHANGE", State: "active", Data: BUData{Name: "乙", NameEn: textValue("Beta")}},
		Record{ID: "s", Date: "2026-05-01", End: "2026-07-01", Type: "DEACTIVATE", State: "inactive", InheritedRecord: "c", InheritedRevision: "c-v1", Data: BUData{Name: "乙", NameEn: textValue("Beta")}},
		Record{ID: "r", Date: "2026-07-01", Type: "REACTIVATE", State: "active", Data: BUData{Name: "乙", NameEn: textValue("Beta")}},
	)
	c, e := Calculate(context.Background(), b, "f", "2020-01-01", []Item{deleteItem(b[1])})
	if e != nil {
		t.Fatal(e)
	}
	want := Impact{DataAffectedRanges: []Range{}, StructuralChanges: []StructuralChange{{RecordID: "f", BeforeRevision: textValue("f-v1"), Facets: []string{"PERIOD"}}, {RecordID: "s", BeforeRevision: textValue("s-v1"), Facets: []string{"INHERITANCE"}}, {RecordID: "c", ItemID: "delete", BeforeRevision: textValue("c-v1"), Facets: []string{"DELETE"}}}, AffectedRecordIDs: []string{"f", "s", "c"}, PeriodChanges: []PeriodChange{{RecordID: "f", Before: &Range{Start: "2026-01-01", End: textValue("2026-03-01")}, After: &Range{Start: "2026-01-01", End: textValue("2026-05-01")}}, {RecordID: "c", ItemID: "delete", Before: &Range{Start: "2026-03-01", End: textValue("2026-05-01")}}}}
	if c.NoChange || !reflect.DeepEqual(c.Impact, want) || c.Records[1].InheritedRecord != "f" || c.Records[1].InheritedRevision != "" || !reflect.DeepEqual(c.Records[2], b[3]) {
		t.Fatal("same data is still a structural fact", c)
	}
}

func deletionFault(err error) string {
	var f *foundation.Fault
	if errors.As(err, &f) {
		return f.Code
	}
	return ""
}

func TestDeleteLongTimelineCancellationBudget(t *testing.T) {
	base := make([]Record, 120000)
	start, _ := time.Parse("2006-01-02", "2020-01-01")
	for i := range base {
		date := start.AddDate(0, 0, i).Format("2006-01-02")
		end := ""
		if i+1 < len(base) {
			end = start.AddDate(0, 0, i+1).Format("2006-01-02")
		}
		kind := "CHANGE"
		if i == 0 {
			kind = "CREATE"
		}
		base[i] = Record{ID: fmt.Sprint(i), Revision: "v1", Date: date, End: end, Type: kind, State: "active", Data: BUData{Name: "甲"}}
	}
	var maximum time.Duration
	for i := 0; i < 5; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() {
			_, err := Calculate(ctx, base, "0", "2020-01-01", []Item{deleteItem(base[60000])})
			done <- err
		}()
		time.Sleep(time.Millisecond)
		at := time.Now()
		cancel()
		err := <-done
		elapsed := time.Since(at)
		if !errors.Is(err, context.Canceled) || elapsed > time.Second {
			t.Fatalf("delete cancellation: %v after %s", err, elapsed)
		}
		if elapsed > maximum {
			maximum = elapsed
		}
	}
	t.Logf("120000 records; 5 DELETE cancellations; maximum local stop=%s (<1s)", maximum)
}

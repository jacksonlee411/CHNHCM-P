package domain

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"
)

func redateItem(r Record, date string) Item {
	return Item{ID: "redate", Action: "REDATE", RecordID: r.ID, ExpectedRevision: r.Revision, Date: date}
}

func TestRedateFourKindsNineDateCells(t *testing.T) {
	// Independent expected transitions at D=June 15. Each target remains the
	// same complete snapshot; only S inherits its accurate predecessor.
	for _, kind := range []string{"CREATE", "CHANGE", "DEACTIVATE", "REACTIVATE"} {
		for _, old := range []string{"2026-06-05", "2026-06-15", "2026-06-25"} {
			for _, next := range []string{"2026-06-10", "2026-06-15", "2026-06-20"} {
				t.Run(kind+"/"+old+"/"+next, func(t *testing.T) {
					rows := []Record{{ID: "f", Date: "2026-01-01", Type: "CREATE", State: "active", Data: BUData{Name: "甲", NameEn: textValue("Alpha")}}}
					if kind == "CREATE" {
						rows[0].Date = old
					} else {
						if kind == "REACTIVATE" {
							rows[0].End = "2026-04-01"
							rows = append(rows, Record{ID: "s", Date: "2026-04-01", Type: "DEACTIVATE", State: "inactive", InheritedRecord: "f", InheritedRevision: "f-v1", Data: BUData{Name: "甲", NameEn: textValue("Alpha")}})
						}
						rows[len(rows)-1].End = old
						target := Record{ID: "target", Date: old, Type: kind, State: "active", Data: BUData{Name: "乙", NameEn: textValue("Beta")}}
						if kind == "DEACTIVATE" {
							target.Data.Name, target.Data.NameEn, target.State = "甲", textValue("Alpha"), "inactive"
							target.InheritedRecord, target.InheritedRevision = "f", "f-v1"
						}
						rows = append(rows, target)
					}
					rows = deletionFixture(rows...)
					before := append([]Record(nil), rows...)
					target := rows[len(rows)-1]
					c, err := Calculate(context.Background(), rows, "f", "2025-01-01", []Item{redateItem(target, next)})
					if err != nil || c.NoChange != (old == next) || len(c.Records) != len(rows) {
						t.Fatal("candidate", err, c)
					}
					got := c.Records[len(c.Records)-1]
					if got.ID != target.ID || got.Date != next || got.End != "" || got.Data.Name != target.Data.Name || !reflect.DeepEqual(got.Data.NameEn, target.Data.NameEn) || got.Type != target.Type || got.Reason != target.Reason || got.TypeLabel != target.TypeLabel || got.ReasonLabel != target.ReasonLabel {
						t.Fatal("identity, snapshot or classification changed", got)
					}
					if len(c.Records) > 1 && c.Records[len(c.Records)-2].End != next {
						t.Fatal("predecessor period", c.Records)
					}
					wantName, wantState := target.Data.Name, target.State
					if next > "2026-06-15" {
						wantName, wantState = "甲", "active"
						if kind == "CREATE" {
							wantName, wantState = "", "not_effective"
						}
						if kind == "REACTIVATE" {
							wantState = "inactive"
						}
					}
					fact := FactAt(c.Records, "2026-06-15")
					if StatusAt(c.Records, "2026-06-15") != wantState || (fact != nil && fact.Data.Name != wantName) || (wantName == "" && fact != nil) {
						t.Fatal("independent current-day expectation", fact, wantName, wantState)
					}
					if old != next {
						start, end := old, next
						if next < old {
							start, end = next, old
						}
						if !reflect.DeepEqual(c.Impact.DataAffectedRanges, []Range{{Start: start, End: textValue(end)}}) {
							t.Fatal("exact data impact", c.Impact)
						}
					}
					if !reflect.DeepEqual(rows, before) {
						t.Fatal("mutated original source")
					}
					for _, boundary := range []string{old, next} {
						day, _ := time.Parse("2006-01-02", boundary)
						for _, offset := range []int{-1, 0, 1} {
							at := day.AddDate(0, 0, offset).Format("2006-01-02")
							wantName, wantState, wantEn := target.Data.Name, target.State, *target.Data.NameEn
							if at < next {
								wantName, wantState, wantEn = "甲", "active", "Alpha"
								if kind == "CREATE" {
									wantName, wantState = "", "not_effective"
								}
								if kind == "REACTIVATE" {
									wantState = "inactive"
								}
							}
							fact := FactAt(c.Records, at)
							if wantName == "" {
								if fact != nil {
									t.Fatal("first boundary must have no fact", at)
								}
							} else if fact == nil || fact.Data.Name != wantName || fact.Data.NameEn == nil || *fact.Data.NameEn != wantEn || fact.State != wantState {
								t.Fatal("boundary day fact", at, fact, wantName, wantState)
							}
						}
					}
				})
			}
		}
	}
}

func TestRedateIndependentReorderingAndFirstBoundary(t *testing.T) {
	for _, tc := range []struct {
		name   string
		target int
		date   string
		want   [][4]string
	}{
		{"first_earlier", 0, "2025-12-01", [][4]string{{"first", "2025-12-01", "2026-06-01", "甲"}, {"june", "2026-06-01", "2026-10-01", "乙"}, {"oct", "2026-10-01", "", "丙"}}},
		{"first_later", 0, "2026-02-01", [][4]string{{"first", "2026-02-01", "2026-06-01", "甲"}, {"june", "2026-06-01", "2026-10-01", "乙"}, {"oct", "2026-10-01", "", "丙"}}},
		{"cross_backward", 2, "2026-03-01", [][4]string{{"first", "2026-01-01", "2026-03-01", "甲"}, {"oct", "2026-03-01", "2026-06-01", "丙"}, {"june", "2026-06-01", "", "乙"}}},
		{"cross_forward", 1, "2026-11-01", [][4]string{{"first", "2026-01-01", "2026-10-01", "甲"}, {"oct", "2026-10-01", "2026-11-01", "丙"}, {"june", "2026-11-01", "", "乙"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := baseline()
			c, err := Calculate(context.Background(), base, "first", "2025-01-01", []Item{redateItem(base[tc.target], tc.date)})
			if err != nil {
				t.Fatal(err)
			}
			for i, w := range tc.want {
				r := c.Records[i]
				if r.ID != w[0] || r.Date != w[1] || r.End != w[2] || r.Data.Name != w[3] {
					t.Fatal(r, w)
				}
			}
			if tc.name == "first_later" && FactAt(c.Records, "2026-01-15") != nil {
				t.Fatal("first gap is not NO_FACT")
			}
		})
	}
}

func TestRedateExactF9FutureFirstAndStopDependencies(t *testing.T) {
	f9 := deletionFixture(Record{ID: "f", Date: "2026-01-01", End: "2026-03-01", Type: "CREATE", State: "active", Data: BUData{Name: "甲", NameEn: textValue("Alpha")}}, Record{ID: "c1", Date: "2026-03-01", End: "2026-05-01", Type: "CHANGE", State: "active", Data: BUData{Name: "乙", NameEn: textValue("Beta")}}, Record{ID: "c2", Date: "2026-05-01", End: "2026-07-01", Type: "CHANGE", State: "active", Data: BUData{Name: "丙", NameEn: textValue("Gamma")}}, Record{ID: "c3", Date: "2026-07-01", Type: "CHANGE", State: "active", Data: BUData{Name: "新", NameEn: textValue("New")}})
	c, err := Calculate(context.Background(), f9, "f", "2025-01-01", []Item{redateItem(f9[2], "2026-02-01")})
	if err != nil {
		t.Fatal(err)
	}
	want := [][4]string{{"f", "2026-01-01", "2026-02-01", "甲"}, {"c2", "2026-02-01", "2026-03-01", "丙"}, {"c1", "2026-03-01", "2026-07-01", "乙"}, {"c3", "2026-07-01", "", "新"}}
	for i, w := range want {
		r := c.Records[i]
		if r.ID != w[0] || r.Date != w[1] || r.End != w[2] || r.Data.Name != w[3] {
			t.Fatal(r, w)
		}
		wantEnglish := []string{"Alpha", "Gamma", "Beta", "New"}[i]
		if r.Data.NameEn == nil || *r.Data.NameEn != wantEnglish {
			t.Fatal("independent English snapshot", r)
		}
	}
	first := baseline()[:1]
	first[0].End = ""
	future, err := Calculate(context.Background(), first, "first", "2025-01-01", []Item{redateItem(first[0], "2026-07-01")})
	if err != nil || FactAt(future.Records, "2026-06-15") != nil || FactAt(first, "2026-06-15").Data.Name != "甲" || future.Records[0].ID != "first" {
		t.Fatal(err, future)
	}
	cycle := deleteCycle()
	same, err := Calculate(context.Background(), cycle, "f", "2025-01-01", []Item{redateItem(cycle[2], "2026-08-01")})
	if err != nil || !same.NoChange {
		t.Fatal(err, same)
	}
	base := deletionFixture(Record{ID: "f", Date: "2026-01-01", End: "2026-03-01", Type: "CREATE", State: "active", Data: BUData{Name: "甲"}}, Record{ID: "c", Date: "2026-03-01", End: "2026-04-01", Type: "CHANGE", State: "active", Data: BUData{Name: "新"}}, Record{ID: "s", Date: "2026-04-01", End: "2026-08-01", Type: "DEACTIVATE", State: "inactive", InheritedRecord: "c", InheritedRevision: "c-v1", Data: BUData{Name: "新"}}, Record{ID: "r", Date: "2026-08-01", Type: "REACTIVATE", State: "active", Data: BUData{Name: "乙"}})
	_, err = Calculate(context.Background(), base, "f", "2025-01-01", []Item{redateItem(base[2], "2026-02-01")})
	if deletionFault(err) != "M02_TIMELINE_INVALID" {
		t.Fatal(err)
	}
}

func TestRedateFullSuccessorRejections(t *testing.T) {
	for _, tc := range []struct {
		name       string
		base       []Record
		target     int
		date, code string
	}{
		{"first_occupied", baseline(), 0, "2026-06-01", "M02_DATE_CONFLICT"},
		{"first_crosses", baseline(), 0, "2026-06-02", "M02_FIRST_RECORD_PROTECTED"},
		{"enterprise", baseline(), 0, "2024-12-31", "M02_INVALID_INPUT"},
		{"before_first", baseline(), 1, "2025-12-31", "M02_FIRST_RECORD_PROTECTED"},
		{"at_first", baseline(), 1, "2026-01-01", "M02_DATE_CONFLICT"},
		{"occupied", baseline(), 1, "2026-10-01", "M02_DATE_CONFLICT"},
		{"change_inactive", deleteCycle(), 3, "2026-06-01", "M02_TIMELINE_INVALID"},
		{"stop_after_resume", deleteCycle(), 1, "2026-09-01", "M02_TIMELINE_INVALID"},
		{"resume_before_stop", deleteCycle(), 2, "2026-03-01", "M02_TIMELINE_INVALID"},
		{"resume_after_change", deleteCycle(), 2, "2026-11-01", "M02_TIMELINE_INVALID"},
		{"invalid_date", baseline(), 1, "2026-02-30", "M02_INVALID_INPUT"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Calculate(context.Background(), tc.base, tc.base[0].ID, "2025-01-01", []Item{redateItem(tc.base[tc.target], tc.date)})
			if deletionFault(err) != tc.code {
				t.Fatal(err)
			}
		})
	}
	base := baseline()
	it := redateItem(base[1], base[1].Date)
	it.ExpectedRevision = "stale"
	if _, err := Calculate(context.Background(), base, "first", "2025-01-01", []Item{it}); deletionFault(err) != "M02_REVISION_CONFLICT" {
		t.Fatal("same-date revision bypass", err)
	}
	it = redateItem(base[1], "2026-07-01")
	it.Changes = PatchFromFields(map[string]*string{})
	if _, err := Calculate(context.Background(), base, "first", "2025-01-01", []Item{it}); deletionFault(err) != "M02_FORBIDDEN_INPUT" {
		t.Fatal(err)
	}
}

func TestRedateLifecycleAndStructuralInheritance(t *testing.T) {
	for _, tc := range []struct {
		target int
		date   string
		index  int
		state  string
	}{{1, "2026-03-01", 1, "inactive"}, {1, "2026-05-01", 1, "inactive"}, {2, "2026-09-01", 2, "active"}} {
		base := deleteCycle()
		c, err := Calculate(context.Background(), base, "f", "2025-01-01", []Item{redateItem(base[tc.target], tc.date)})
		if err != nil {
			t.Fatal(err)
		}
		if c.Records[tc.index].Date != tc.date || c.Records[tc.index].State != tc.state || c.Records[tc.index].Data.Name != base[tc.target].Data.Name {
			t.Fatal(c)
		}
	}
	base := deletionFixture(Record{ID: "f", Date: "2026-01-01", End: "2026-03-01", Type: "CREATE", State: "active", Data: BUData{Name: "乙", NameEn: textValue("Beta")}}, Record{ID: "c", Date: "2026-03-01", End: "2026-05-01", Type: "CHANGE", State: "active", Data: BUData{Name: "乙", NameEn: textValue("Beta")}}, Record{ID: "s", Date: "2026-05-01", End: "2026-07-01", Type: "DEACTIVATE", State: "inactive", InheritedRecord: "c", InheritedRevision: "c-v1", Data: BUData{Name: "乙", NameEn: textValue("Beta")}}, Record{ID: "r", Date: "2026-07-01", Type: "REACTIVATE", State: "active", Data: BUData{Name: "乙", NameEn: textValue("Beta")}})
	c, err := Calculate(context.Background(), base, "f", "2025-01-01", []Item{redateItem(base[1], "2026-04-01")})
	if err != nil || c.NoChange || len(c.Impact.DataAffectedRanges) != 0 || c.Records[2].InheritedRecord != "c" || c.Records[2].InheritedRevision != "" || len(c.Impact.StructuralChanges) != 3 {
		t.Fatal("same-value movement must change periods and accurate inheritance", err, c)
	}
	if !reflect.DeepEqual(c.Records[3], base[3]) {
		t.Fatal("existing reactivation snapshot changed")
	}

}

func TestRedateLongTimelineCancellationBudget(t *testing.T) {
	base := make([]Record, 120000)
	start, _ := time.Parse("2006-01-02", "2020-01-01")
	for i := range base {
		date := start.AddDate(0, 0, i*2).Format("2006-01-02")
		end := ""
		if i+1 < len(base) {
			end = start.AddDate(0, 0, (i+1)*2).Format("2006-01-02")
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
			_, err := Calculate(ctx, base, "0", "2020-01-01", []Item{redateItem(base[60000], start.AddDate(0, 0, 120001).Format("2006-01-02"))})
			done <- err
		}()
		time.Sleep(time.Millisecond)
		at := time.Now()
		cancel()
		err := <-done
		elapsed := time.Since(at)
		if !errors.Is(err, context.Canceled) || elapsed > time.Second {
			t.Fatal(err, elapsed)
		}
		if elapsed > maximum {
			maximum = elapsed
		}
	}
	t.Logf("120000 records; 5 REDATE cancellations; maximum local stop=%s (<1s)", maximum)
}

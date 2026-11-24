package domain

import (
	"reflect"
	"testing"
)

func TestBUCoverageUsesAllPeriodsAndRejectsMalformedFacts(t *testing.T) {
	base := []Record{{Date: "2026-01-01", State: "active"}}
	for _, tc := range []struct {
		name string
		rows []Record
		same bool
	}{
		{"rename", []Record{{Date: "2026-01-01", State: "active", Data: BUData{Name: "changed"}}}, true},
		{"subdivision", []Record{{Date: "2026-01-01", End: "2035-01-01", State: "active"}, {Date: "2035-01-01", State: "active"}}, true},
		{"first moved", []Record{{Date: "2026-01-02", State: "active"}}, false},
		{"future gap", []Record{{Date: "2026-01-01", End: "2035-01-01", State: "active"}, {Date: "2035-01-01", End: "2036-01-01", State: "inactive"}, {Date: "2036-01-01", State: "active"}}, false},
		{"overlap", []Record{{Date: "2026-01-01", End: "2035-01-01", State: "active"}, {Date: "2034-01-01", State: "active"}}, false},
		{"unknown state", []Record{{Date: "2026-01-01", State: "unknown"}}, false},
		{"missing", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if SameBUCoverage(base, tc.rows) != tc.same {
				t.Fatal("incorrect coverage classification")
			}
		})
	}
}

func TestRemovedBUCoveragePreservesEveryHalfOpenBoundary(t *testing.T) {
	rows := func(values ...string) []Record {
		out := []Record{}
		for i := 0; i < len(values); i += 2 {
			r := Record{Date: values[i], State: values[i+1]}
			if i+2 < len(values) {
				r.End = values[i+2]
			}
			out = append(out, r)
		}
		return out
	}
	base := rows("2026-01-01", "active")
	for _, tc := range []struct {
		name          string
		before, after []Record
		want          []BUCoverageRange
	}{
		{"future stop", base, rows("2026-01-01", "active", "2035-01-01", "inactive"), []BUCoverageRange{{Start: "2035-01-01", End: ""}}},
		{"first postponed", base, rows("2027-01-01", "active"), []BUCoverageRange{{Start: "2026-01-01", End: "2027-01-01"}}},
		{"future gap", base, rows("2026-01-01", "active", "2035-01-01", "inactive", "2036-01-01", "active"), []BUCoverageRange{{Start: "2035-01-01", End: "2036-01-01"}}},
		{"subdivided active period", base, rows("2026-01-01", "active", "2027-01-01", "active"), []BUCoverageRange{}},
		{"expansion", rows("2027-01-01", "active"), base, []BUCoverageRange{}},
		{"gap shifted", rows("2026-01-01", "active", "2027-01-01", "inactive", "2029-01-01", "active"), rows("2026-01-01", "active", "2028-01-01", "inactive", "2030-01-01", "active"), []BUCoverageRange{{Start: "2029-01-01", End: "2030-01-01"}}},
		{"one new period bridges old gaps", rows("2026-01-01", "active", "2027-01-01", "inactive", "2028-01-01", "active", "2030-01-01", "inactive"), rows("2026-06-01", "active", "2029-01-01", "inactive"), []BUCoverageRange{{Start: "2026-01-01", End: "2026-06-01"}, {Start: "2029-01-01", End: "2030-01-01"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := RemovedBUCoverage(tc.before, tc.after)
			if err != nil || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("lost=%v err=%v want=%v", got, err, tc.want)
			}
		})
	}
	for _, invalid := range [][]Record{nil, {{Date: "invalid", State: "active"}}, {{Date: "2026-01-01", End: "2026-01-01", State: "active"}}, {{Date: "2026-01-01", State: "unknown"}}} {
		if _, err := RemovedBUCoverage(base, invalid); err == nil {
			t.Fatal("invalid after accepted")
		}
		if _, err := RemovedBUCoverage(invalid, base); err == nil {
			t.Fatal("invalid before accepted")
		}
	}
	if (BUCoverageRange{Start: "2027-01-01", End: "2028-01-01"}).Intersects("2026-01-01", "2027-01-01") || (BUCoverageRange{Start: "2027-01-01", End: "2028-01-01"}).Intersects("2028-01-01", "") {
		t.Fatal("touching boundaries overlap")
	}
}

func TestRemovedBUCoverageMatchesIndependentDailyMembership(t *testing.T) {
	// Every four-period active/inactive combination, with distinct first dates.
	// Expected membership is read directly from the input periods, independently
	// of interval subtraction or the production coverage helpers.
	dates := []string{"2026-01-01", "2026-01-03", "2026-01-05", "2026-01-07"}
	makeRows := func(mask, offset int) []Record {
		out := []Record{}
		for i := offset; i < len(dates); i++ {
			r := Record{Date: dates[i], State: "inactive"}
			if mask&(1<<i) != 0 {
				r.State = "active"
			}
			if i+1 < len(dates) {
				r.End = dates[i+1]
			}
			out = append(out, r)
		}
		return out
	}
	active := func(rows []Record, day string) bool {
		for _, r := range rows {
			if r.Date <= day && (r.End == "" || day < r.End) {
				return r.State == "active"
			}
		}
		return false
	}
	for a := 0; a < 16; a++ {
		for b := 0; b < 16; b++ {
			for offset := 0; offset < 4; offset++ {
				before, after := makeRows(a, 0), makeRows(b, offset)
				lost, err := RemovedBUCoverage(before, after)
				if err != nil {
					t.Fatal(err)
				}
				for _, day := range []string{"2025-12-31", "2026-01-01", "2026-01-02", "2026-01-03", "2026-01-04", "2026-01-05", "2026-01-06", "2026-01-07", "2035-01-01"} {
					got := false
					for _, w := range lost {
						if w.Start <= day && (w.End == "" || day < w.End) {
							got = true
						}
					}
					if got != (active(before, day) && !active(after, day)) {
						t.Fatalf("masks=%d/%d offset=%d day=%s lost=%v", a, b, offset, day, lost)
					}
				}
			}
		}
	}
}

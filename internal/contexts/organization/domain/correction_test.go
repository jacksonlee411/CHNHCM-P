package domain

import "testing"

func TestCorrectionBoundsIncludeOriginalAndPreserveNeighbours(t *testing.T) {
	for _, tc := range []struct {
		name             string
		rows             []Record
		target, min, max string
	}{
		{"middle", []Record{{ID: "p", Date: "2026-06-01"}, {ID: "t", Date: "2026-06-15"}, {ID: "n", Date: "2026-07-01"}}, "t", "2026-06-02", "2026-06-30"},
		{"only-original", []Record{{ID: "p", Date: "2026-06-01"}, {ID: "t", Date: "2026-06-02"}, {ID: "n", Date: "2026-06-03"}}, "t", "2026-06-02", "2026-06-02"},
		{"first", []Record{{ID: "t", Date: "2026-06-01"}, {ID: "n", Date: "2026-10-01"}}, "t", "2025-01-01", "2026-09-30"},
		{"single", []Record{{ID: "t", Date: "2028-02-29"}}, "t", "2025-01-01", ""},
		{"leap", []Record{{ID: "p", Date: "2028-02-28"}, {ID: "t", Date: "2028-02-29"}, {ID: "n", Date: "2028-03-01"}}, "t", "2028-02-29", "2028-02-29"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			min, max, e := CorrectionBounds(tc.rows, tc.target, "2025-01-01")
			if e != nil || min != tc.min || max != tc.max {
				t.Fatalf("bounds %q..%q %v", min, max, e)
			}
			for _, r := range tc.rows {
				if r.ID == tc.target {
					if e := ValidateCorrectionDate(tc.rows, tc.target, "2025-01-01", r.Date); e != nil {
						t.Fatal("original date rejected", e)
					}
				}
			}
			if len(tc.rows) > 1 && tc.rows[0].ID != tc.target {
				if ValidateCorrectionDate(tc.rows, tc.target, "2025-01-01", tc.rows[0].Date) == nil {
					t.Fatal("previous record date accepted")
				}
			}
		})
	}
}

// Literal calendar cases exercise the allowed dates themselves, rather than
// treating the range returned by CorrectionBounds as the validation oracle.
func TestCorrectionDateInclusiveEdgesAndCalendarTransitions(t *testing.T) {
	middle := []Record{{ID: "p", Date: "2026-01-01"}, {ID: "t", Date: "2026-06-01"}, {ID: "n", Date: "2026-10-01"}}
	year := []Record{{ID: "p", Date: "2025-12-31"}, {ID: "t", Date: "2026-01-10"}, {ID: "n", Date: "2026-02-01"}}
	leap := []Record{{ID: "p", Date: "2028-02-28"}, {ID: "t", Date: "2028-03-05"}, {ID: "n", Date: "2028-03-10"}}
	first := []Record{{ID: "t", Date: "2026-01-01"}, {ID: "n", Date: "2026-06-01"}}
	for _, tc := range []struct {
		name    string
		rows    []Record
		date    string
		allowed bool
	}{
		{"previous-plus-one", middle, "2026-01-02", true},
		{"next-minus-one", middle, "2026-09-30", true},
		{"previous-occupied", middle, "2026-01-01", false},
		{"next-occupied", middle, "2026-10-01", false},
		{"before-previous", middle, "2025-12-31", false},
		{"after-next", middle, "2026-10-02", false},
		{"year-first-day", year, "2026-01-01", true},
		{"month-last-day", year, "2026-01-31", true},
		{"leap-day", leap, "2028-02-29", true},
		{"non-leap-day", middle, "2026-02-29", false},
		{"enterprise-first-day", first, "2020-01-01", true},
		{"before-enterprise", first, "2019-12-31", false},
		{"single-open-end", []Record{{ID: "t", Date: "2028-02-29"}}, "2050-01-01", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCorrectionDate(tc.rows, "t", "2020-01-01", tc.date)
			if (err == nil) != tc.allowed {
				t.Fatalf("date %s allowed=%v, got %v", tc.date, tc.allowed, err)
			}
		})
	}
}

package domain

import "testing"

func TestAdditionWindowPositionDefaultsAndCalendar(t *testing.T) {
	const a = "11111111-1111-4111-8111-111111111111"
	const b = "22222222-2222-4222-8222-222222222222"
	for _, tc := range []struct {
		name, left, right, today, state, min, max, def, mode string
		available                                            bool
	}{
		{"past-latest", "2026-01-01", "", "2026-11-02", "active", "2026-01-02", "", "2026-11-02", "APPEND", true},
		{"today-latest", "2026-11-02", "", "2026-11-02", "active", "2026-11-03", "", "2026-11-03", "APPEND", true},
		{"future-latest", "2027-01-01", "", "2026-11-02", "active", "2027-01-02", "", "2027-01-02", "APPEND", true},
		{"middle", "2026-06-01", "2026-10-01", "2026-11-02", "active", "2026-06-02", "2026-09-30", "2026-06-02", "INSERT", true},
		{"no-gap", "2026-06-01", "2026-06-02", "2026-11-02", "active", "", "", "", "INSERT", false},
		{"one-day", "2026-06-01", "2026-06-03", "2026-11-02", "active", "2026-06-02", "2026-06-02", "2026-06-02", "INSERT", true},
		{"january-end-insert", "2026-01-31", "2026-02-03", "2026-01-01", "active", "2026-02-01", "2026-02-02", "2026-02-01", "INSERT", true},
		{"leap-end-insert", "2028-02-29", "2028-03-03", "2026-01-01", "active", "2028-03-01", "2028-03-02", "2028-03-01", "INSERT", true},
		{"month-end", "2026-04-30", "", "2026-04-01", "active", "2026-05-01", "", "2026-05-01", "APPEND", true},
		{"year-end", "2026-12-31", "", "2026-11-02", "active", "2027-01-01", "", "2027-01-01", "APPEND", true},
		{"leap-day", "2028-02-28", "2028-03-01", "2026-11-02", "active", "2028-02-29", "2028-02-29", "2028-02-29", "INSERT", true},
		{"non-leap", "2027-02-28", "", "2026-11-02", "active", "2027-03-01", "", "2027-03-01", "APPEND", true},
		{"stopped-gap", "2026-04-01", "2026-08-01", "2026-11-02", "inactive", "", "", "", "INSERT", false},
		{"stopped-latest", "2026-04-01", "", "2026-11-02", "inactive", "", "", "", "APPEND", false},
		{"calendar-end", "9999-12-31", "", "2026-11-02", "active", "", "", "", "APPEND", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := []Record{{ID: a, Date: tc.left, State: tc.state}}
			if tc.right != "" {
				base = append(base, Record{ID: b, Date: tc.right, State: "active"})
			}
			w, e := PrepareAddition(base, a, "2020-01-01", tc.today)
			if e != nil || w.Available != tc.available || w.Min != tc.min || w.Max != tc.max || w.Default != tc.def || w.Mode != tc.mode {
				t.Fatalf("unexpected prepared gap: %+v %v", w, e)
			}
		})
	}
}

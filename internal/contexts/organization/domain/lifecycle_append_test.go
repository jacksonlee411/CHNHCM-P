package domain

import (
	"errors"
	"reflect"
	"testing"

	"chnhcm.local/chnhcm/internal/foundation"
)

func TestSingleLifecycleAppendWindow(t *testing.T) {
	for _, tc := range []struct {
		name, date, state, kind, today, min, def string
		available                                bool
	}{
		{"past-stop", "2026-01-01", "active", "DEACTIVATE", "2026-11-03", "2026-01-02", "2026-11-03", true},
		{"future-resume", "2028-12-31", "inactive", "REACTIVATE", "2026-11-03", "2029-01-01", "2029-01-01", true},
		{"today-stop", "2026-11-03", "active", "DEACTIVATE", "2026-11-03", "2026-11-04", "2026-11-04", true},
		{"leap-resume", "2028-02-28", "inactive", "REACTIVATE", "2026-11-03", "2028-02-29", "2028-02-29", true},
		{"end-stop", "9999-12-30", "active", "DEACTIVATE", "2026-11-03", "9999-12-31", "9999-12-31", true},
		{"calendar-end", "9999-12-31", "active", "DEACTIVATE", "2026-11-03", "", "", false},
		{"already-active", "2026-01-01", "active", "REACTIVATE", "2026-11-03", "", "", false},
		{"already-stopped", "2026-01-01", "inactive", "DEACTIVATE", "2026-11-03", "", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, e := PrepareLifecycleAppend([]Record{{ID: "tail", Date: tc.date, State: tc.state}}, "tail", tc.kind, "2020-01-01", tc.today)
			if e != nil || w.Available != tc.available || w.Min != tc.min || w.Default != tc.def || tc.available && w.Max != "9999-12-31" {
				t.Fatalf("window %+v: %v", w, e)
			}
		})
	}
}

func TestSingleLifecycleAppendBindsLastEffectivePosition(t *testing.T) {
	const first = "11111111-1111-4111-8111-111111111111"
	const tail = "22222222-2222-4222-8222-222222222222"
	// Order by effective date, independently of input order or record identity.
	rows := []Record{{ID: tail, Date: "2028-08-01", State: "active"}, {ID: first, Date: "2026-01-01", State: "active"}}
	original := append([]Record(nil), rows...)
	for _, kind := range []string{"DEACTIVATE", "REACTIVATE"} {
		w, e := PrepareLifecycleAppend(rows, first, kind, "2020-01-01", "2026-11-03")
		if e != nil || w.Available {
			t.Fatal(w, e)
		}
		if e := ValidateLifecycleAppend(rows, first, kind, "2020-01-01", "2028-09-01"); e == nil {
			t.Fatal("old selection silently appended")
		}
	}
	for _, date := range []string{"2028-08-01", "2028-07-15", "2028-02-30"} {
		var fault *foundation.Fault
		if e := ValidateLifecycleAppend(rows, tail, "DEACTIVATE", "2020-01-01", date); !errors.As(e, &fault) || fault.Field != "effectiveDate" {
			t.Fatal(date, e)
		}
	}
	if e := ValidateLifecycleAppend(rows, tail, "DEACTIVATE", "2020-01-01", "2028-08-02"); e != nil {
		t.Fatal(e)
	}
	if _, e := PrepareLifecycleAppend(rows, "missing", "DEACTIVATE", "2020-01-01", "2026-11-03"); e == nil {
		t.Fatal("unknown anchor accepted")
	}
	if !reflect.DeepEqual(rows, original) {
		t.Fatal("preparation mutated source")
	}
}

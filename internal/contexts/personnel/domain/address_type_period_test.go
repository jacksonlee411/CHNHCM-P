package domain

import (
	"slices"
	"testing"
)

func TestAddressTypePeriodBoundaries(t *testing.T) {
	const start, stop = "2027-02-01", "2027-03-01"
	p := AddressTypePeriod{FirstDay: start}
	scheduled, err := p.WithStop(stop)
	if err != nil {
		t.Fatal(err)
	}
	for day, want := range map[string]string{"2027-01-31": "not_effective", start: "active", "2027-02-28": "active", stop: "inactive", "2027-03-02": "inactive"} {
		if got := scheduled.State(day); got != want {
			t.Errorf("%s: %s, want %s", day, got, want)
		}
	}
	if !slices.Equal(scheduled.Dates(), []string{start, stop}) {
		t.Fatal(scheduled.Dates())
	}
	zero, err := p.WithStop(start)
	if err != nil || zero.State(start) != "inactive" || !slices.Equal(zero.Dates(), []string{start}) {
		t.Fatal(zero, err)
	}
	corrected, err := scheduled.WithStop("")
	if err != nil || corrected.State(stop) != "active" || scheduled.State(stop) != "inactive" {
		t.Fatal("correction changed prior period")
	}
	migrated := AddressTypePeriod{FirstDay: "1900-01-01", ConfirmedFrom: start, UndatedStop: true}
	if migrated.State("2027-01-31") != "unknown" || migrated.State(start) != "inactive" || !slices.Equal(migrated.Dates(), []string{"1900-01-01", start}) {
		t.Fatal(migrated)
	}
	for _, day := range []string{"2027-02-30", "2027-01-31"} {
		if _, err := migrated.WithStop(day); err == nil {
			t.Fatal("accepted", day)
		}
	}
}

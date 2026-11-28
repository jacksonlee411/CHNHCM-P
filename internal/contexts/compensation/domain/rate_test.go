package domain_test

import (
	"testing"

	d "chnhcm.local/chnhcm/internal/contexts/compensation/domain"
)

func TestGradeRateDecimalAndBands(t *testing.T) {
	for _, tc := range []struct {
		input  string
		places int
		want   string
	}{
		{"0", 2, "0.00"}, {"00.01", 2, "0.01"}, {"12345678901234567890.1234", 4, "12345678901234567890.1234"},
		{"0.001", 2, ""}, {"-1", 2, ""}, {"1e3", 2, ""}, {"", 2, ""}, {"1.20", 0, ""},
	} {
		got, _, err := d.Decimal("amount", tc.input, tc.places)
		if tc.want == "" && err == nil || tc.want != "" && (err != nil || got != tc.want) {
			t.Errorf("%q (%d places) returned %q, %v", tc.input, tc.places, got, err)
		}
	}
	form, zero, upper, over := "RANGE", "0", "100", "101"
	start, err := (d.Amounts{}).Apply(map[string]*string{"form": &form, "minimum": &zero, "guidance": &zero, "maximum": &upper}, 2)
	if err != nil || start.Guidance != "0.00" {
		t.Fatalf("zero guidance: %+v, %v", start, err)
	}
	cleared, err := start.Apply(map[string]*string{"guidance": nil}, 2)
	if err != nil || cleared.Guidance != "" || cleared.Minimum != "0.00" {
		t.Fatal(cleared, err)
	}
	for _, invalid := range []map[string]*string{{"maximum": nil}, {"guidance": &over}, {"minimum": &over}} {
		if _, err := cleared.Apply(invalid, 2); err == nil {
			t.Errorf("accepted invalid band %v", invalid)
		}
	}
	fixed := "FIXED"
	converted, err := start.Apply(map[string]*string{"form": &fixed, "amount": &upper}, 2)
	if err != nil || converted.Amount != "100.00" || converted.Minimum != "" || converted.Guidance != "" || converted.Maximum != "" {
		t.Fatal(converted, err)
	}
}

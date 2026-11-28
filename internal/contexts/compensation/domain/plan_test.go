package domain

import (
	"slices"
	"testing"
)

func TestPlanTargetAndStableBase(t *testing.T) {
	a := Component{ID: "1ee0952e-8887-412d-98ba-9fd977526f8a", Code: "BASIC", Name: "年基本工资", Nature: "BASIC", Currency: "CNY", Frequency: "YEAR", Unit: "PER_YEAR", Precision: 2, Method: "FIXED", Amount: "180000.04"}
	b := Component{ID: "10dc84ca-a380-4af9-94b7-f0dee5849803", Code: "BONUS", Name: "目标绩效", Nature: "BONUS", Currency: "CNY", Frequency: "YEAR", Unit: "PER_YEAR", Precision: 2, Method: "PERCENT", BaseID: a.ID, Percent: "12.50", Rounding: "HALF_UP"}
	rows := []Component{b, a}
	if err := ValidateComponents(rows); err != nil {
		t.Fatal(err)
	}
	if amount, ok := TargetQuote(rows, b.ID, "", nil); !ok || amount != "22500.01" {
		t.Fatalf("half-cent target %q", amount)
	}
	a.Amount = "180000.00"
	rows[1] = a
	if amount, ok := TargetQuote(rows, b.ID, "", nil); !ok || amount != "22500.00" {
		t.Fatalf("annual target %q", amount)
	}
	for _, test := range []struct {
		name   string
		change func([]Component)
	}{
		{"self", func(rs []Component) { rs[0].BaseID = rs[0].ID }},
		{"missing", func(rs []Component) { rs[0].BaseID = "404" }},
		{"cycle", func(rs []Component) {
			rs[1].Method = "PERCENT"
			rs[1].Amount = ""
			rs[1].Percent = "8"
			rs[1].Rounding = "HALF_UP"
			rs[1].BaseID = rs[0].ID
		}},
		{"cross currency", func(rs []Component) { rs[1].Currency = "USD" }},
		{"cross frequency", func(rs []Component) { rs[1].Frequency = "MONTH"; rs[1].Unit = "PER_MONTH" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			rs := slices.Clone(rows)
			test.change(rs)
			if ValidateComponents(rs) == nil {
				t.Fatal("invalid basis accepted")
			}
		})
	}
	a.Method = "GRADE_RATE"
	a.Amount = ""
	a.Rates = []RateLink{{GradeID: "5252f7fa-867f-4995-a949-db8d49b6075a", RateID: "b4f48b17-04cd-4437-a4dc-1f5755ef510b"}}
	rows = []Component{a, b}
	if err := ValidateComponents(rows); err != nil {
		t.Fatal(err)
	}
	if _, ok := TargetQuote(rows, b.ID, a.Rates[0].GradeID, func(RateLink) (string, bool) { return "", false }); ok {
		t.Fatal("range used as a personal amount")
	}
}

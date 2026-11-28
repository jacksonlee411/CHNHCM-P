package domain

// A monthly award uses the exact contract amount on its effective day.
func MonthlyAwardAmount(input string, band Amounts) (string, error) {
	amount, value, err := Decimal("amount", input, 2)
	if err != nil {
		return "", err
	}
	if band.Form != "RANGE" {
		return "", Invalid("planId", "请选择人民币月额连续薪带")
	}
	_, low, err := Decimal("minimum", band.Minimum, 2)
	if err != nil {
		return "", err
	}
	_, high, err := Decimal("maximum", band.Maximum, 2)
	if err != nil {
		return "", err
	}
	if value.Cmp(low) < 0 || value.Cmp(high) > 0 {
		return "", Invalid("amount", "基本月薪须在生效日薪带上下限内")
	}
	return amount, nil
}
func MonthlyBasic(rows []Component, grade string) (Component, string, bool) {
	var basic *Component
	for i := range rows {
		if rows[i].Nature == "BASIC" {
			if basic != nil {
				return Component{}, "", false
			}
			basic = &rows[i]
		}
	}
	if basic == nil || basic.Method != "GRADE_RATE" || basic.Currency != "CNY" || basic.Frequency != "MONTH" || basic.Unit != "PER_MONTH" || basic.Precision != 2 {
		return Component{}, "", false
	}
	for _, r := range basic.Rates {
		if r.GradeID == grade {
			return *basic, r.RateID, true
		}
	}
	return Component{}, "", false
}

// Package domain owns the meaning of compensation configuration and its business dates.
package domain

import (
	"math/big"
	"strings"

	"chnhcm.local/chnhcm/internal/foundation"
)

func Invalid(field, message string) error {
	return &foundation.Fault{Status: 422, Code: "M09_INVALID_CONFIGURATION", Field: field, Message: message}
}

type RateKey struct {
	GradeID   string `json:"gradeId"`
	Currency  string `json:"currency"`
	Frequency string `json:"frequency"`
	Unit      string `json:"unit"`
}

func (k RateKey) Validate() error {
	if !foundation.ValidID(k.GradeID) {
		return Invalid("gradeId", "请选择职级")
	}
	if !CurrencyCode(k.Currency) {
		return Invalid("currency", "请输入三位大写币种代码")
	}
	unit, ok := FrequencyUnit(k.Frequency)
	if !ok {
		return Invalid("frequency", "请选择月、年、小时或一次性")
	}
	if k.Unit != unit {
		return Invalid("unit", "计量单位与约定频率不一致")
	}
	return nil
}

func CurrencyCode(value string) bool {
	if len(value) != 3 {
		return false
	}
	for _, c := range value {
		if c < 'A' || c > 'Z' {
			return false
		}
	}
	return true
}

func FrequencyUnit(frequency string) (string, bool) {
	switch frequency {
	case "MONTH":
		return "PER_MONTH", true
	case "YEAR":
		return "PER_YEAR", true
	case "HOUR":
		return "PER_HOUR", true
	case "ONCE":
		return "PER_OCCURRENCE", true
	}
	return "", false
}

// RateReasons describes the original operation. A correction retains that operation.
func RateReasons(operation string) []string {
	switch operation {
	case "CREATE":
		return []string{"建立标准"}
	case "ADD":
		return []string{"薪酬制度调整", "其他资料调整"}
	case "DEACTIVATE":
		return []string{"停止使用", "薪酬制度调整"}
	case "REACTIVATE":
		return []string{"恢复使用", "薪酬制度调整"}
	}
	return []string{}
}

// Decimal accepts major-unit decimal strings. No binary float or early rounding
// is involved, including when an amount is zero. Values are padded for display.
func Decimal(field, input string, places int) (string, *big.Int, error) {
	whole, fraction, point := strings.Cut(input, ".")
	if places < 0 || places > 18 || whole == "" || point && fraction == "" || len(fraction) > places {
		return "", nil, Invalid(field, "请输入符合本制度精度的非负数")
	}
	for _, part := range []string{whole, fraction} {
		for _, char := range part {
			if char < '0' || char > '9' {
				return "", nil, Invalid(field, "请输入非负十进制数")
			}
		}
	}
	whole = strings.TrimLeft(whole, "0")
	if whole == "" {
		whole = "0"
	}
	fraction += strings.Repeat("0", places-len(fraction))
	minor, _ := new(big.Int).SetString(whole+fraction, 10)
	if places == 0 {
		return whole, minor, nil
	}
	return whole + "." + fraction, minor, nil
}

type Amounts struct {
	Form     string `json:"form"`
	Amount   string `json:"amount,omitempty"`
	Minimum  string `json:"minimum,omitempty"`
	Guidance string `json:"guidance,omitempty"`
	Maximum  string `json:"maximum,omitempty"`
}

// Apply preserves omitted fields. A form change discards only values which
// cease to apply. An explicit null clears the optional guidance.
func (a Amounts) Apply(changes map[string]*string, places int) (Amounts, error) {
	if form, exists := changes["form"]; exists {
		if form == nil || (*form != "FIXED" && *form != "RANGE") {
			return Amounts{}, Invalid("form", "请选择定额或连续薪带")
		}
		if a.Form != *form {
			a = Amounts{Form: *form}
		}
	}
	if a.Form != "FIXED" && a.Form != "RANGE" {
		return Amounts{}, Invalid("form", "请选择定额或连续薪带")
	}
	for field, value := range changes {
		if field == "form" {
			continue
		}
		var target *string
		switch field {
		case "amount":
			if a.Form == "FIXED" {
				target = &a.Amount
			}
		case "minimum":
			if a.Form == "RANGE" {
				target = &a.Minimum
			}
		case "guidance":
			if a.Form == "RANGE" {
				target = &a.Guidance
			}
		case "maximum":
			if a.Form == "RANGE" {
				target = &a.Maximum
			}
		}
		if target == nil {
			return Amounts{}, Invalid(field, "此取值方式不使用该字段")
		}
		if value == nil && field == "guidance" {
			*target = ""
			continue
		}
		if value == nil {
			return Amounts{}, Invalid(field, "请填写金额")
		}
		formatted, _, err := Decimal(field, *value, places)
		if err != nil {
			return Amounts{}, err
		}
		*target = formatted
	}
	if a.Form == "FIXED" {
		if a.Amount == "" {
			return Amounts{}, Invalid("amount", "请填写定额")
		}
		return a, nil
	}
	_, low, err := Decimal("minimum", a.Minimum, places)
	if err != nil {
		return Amounts{}, err
	}
	_, high, err := Decimal("maximum", a.Maximum, places)
	if err != nil {
		return Amounts{}, err
	}
	if low.Cmp(high) > 0 {
		return Amounts{}, Invalid("maximum", "上限应大于或等于下限")
	}
	if a.Guidance != "" {
		_, guide, err := Decimal("guidance", a.Guidance, places)
		if err != nil {
			return Amounts{}, err
		}
		if guide.Cmp(low) < 0 || guide.Cmp(high) > 0 {
			return Amounts{}, Invalid("guidance", "指导值应在下限和上限之间")
		}
	}
	return a, nil
}

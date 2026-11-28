package domain

import (
	"math/big"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"chnhcm.local/chnhcm/internal/foundation"
)

// The rate identity follows subsequent business dates. Adopted identifies the
// precise effective record and the aggregate head observed when the plan was saved.
type AdoptedRate struct {
	BusinessAt string `json:"businessAt"`
	Head       string `json:"head"`
	Source     string `json:"sourceVersion"`
	Record     string `json:"recordId"`
	Revision   string `json:"revision"`
}
type RateLink struct {
	GradeID string      `json:"gradeId"`
	RateID  string      `json:"rateId"`
	Adopted AdoptedRate `json:"adopted"`
}
type Component struct {
	ID        string     `json:"id"`
	Code      string     `json:"code"`
	Name      string     `json:"name"`
	Nature    string     `json:"nature"`
	Currency  string     `json:"currency"`
	Frequency string     `json:"frequency"`
	Unit      string     `json:"unit"`
	Precision int        `json:"precision"`
	Method    string     `json:"method"`
	Amount    string     `json:"amount,omitempty"`
	Rates     []RateLink `json:"rates,omitempty"`
	BaseID    string     `json:"baseId,omitempty"`
	Percent   string     `json:"percent,omitempty"`
	Rounding  string     `json:"rounding,omitempty"`
}
type PlanTerms struct {
	Name         string      `json:"name"`
	Description  string      `json:"description,omitempty"`
	ScopeVersion string      `json:"scopeVersion"`
	Components   []Component `json:"components"`
}

func PlanReasons(action string) []string {
	if action == "CREATE" {
		return []string{"建立计划"}
	}
	return RateReasons(action)
}
func NormalizeCode(value string) (string, bool) {
	code := strings.ToUpper(strings.TrimSpace(value))
	if code == "" || utf8.RuneCountInString(code) > 64 {
		return "", false
	}
	for _, ch := range code {
		if !unicode.IsLetter(ch) && !unicode.IsDigit(ch) && ch != '-' && ch != '_' && ch != '.' {
			return "", false
		}
	}
	return code, true
}
func ValidateComponents(items []Component) error {
	if len(items) == 0 {
		return Invalid("components", "计划至少需要一个组成项")
	}
	byID := make(map[string]int, len(items))
	codes := make(map[string]bool, len(items))
	for i := range items {
		row := &items[i]
		field := "components." + strconv.Itoa(i) + "."
		if !foundation.ValidID(row.ID) {
			return Invalid(field+"id", "组成项身份无效")
		}
		if _, duplicate := byID[row.ID]; duplicate {
			return Invalid(field+"id", "计划内组成项身份重复")
		}
		byID[row.ID] = i
		code, ok := NormalizeCode(row.Code)
		if !ok || codes[code] {
			return Invalid(field+"code", "请填写本计划内唯一的项编码")
		}
		row.Code = code
		codes[code] = true
		row.Name = strings.TrimSpace(row.Name)
		if row.Name == "" {
			return Invalid(field+"name", "请填写组成项名称")
		}
		if !slices.Contains([]string{"BASIC", "ALLOWANCE", "PERFORMANCE", "BONUS", "OTHER"}, row.Nature) {
			return Invalid(field+"nature", "请选择组成项性质")
		}
		if !CurrencyCode(row.Currency) {
			return Invalid(field+"currency", "请输入三位大写币种代码")
		}
		unit, ok := FrequencyUnit(row.Frequency)
		if !ok || unit != row.Unit {
			return Invalid(field+"unit", "约定频率与计量单位不匹配")
		}
		if row.Precision < 0 || row.Precision > 18 || row.Currency == "CNY" && row.Frequency != "HOUR" && row.Precision != 2 {
			return Invalid(field+"precision", "请核对金额小数位数")
		}
		switch row.Method {
		case "FIXED":
			if len(row.Rates) != 0 || row.BaseID != "" || row.Percent != "" || row.Rounding != "" {
				return Invalid(field+"method", "固定额只填写本计划约定金额")
			}
			amount, _, err := Decimal(field+"amount", row.Amount, row.Precision)
			if err != nil {
				return err
			}
			row.Amount = amount
		case "GRADE_RATE":
			if row.Amount != "" || row.BaseID != "" || row.Percent != "" || row.Rounding != "" {
				return Invalid(field+"method", "职级取值须选用共用标准")
			}
			if len(row.Rates) == 0 {
				return Invalid(field+"rates", "请逐项选择职级及对应的共用标准")
			}
			seen := map[string]bool{}
			for _, link := range row.Rates {
				if !foundation.ValidID(link.GradeID) || !foundation.ValidID(link.RateID) || seen[link.GradeID] {
					return Invalid(field+"rates", "同一职级须对应一份共用标准")
				}
				seen[link.GradeID] = true
			}
		case "PERCENT":
			if row.Amount != "" || len(row.Rates) > 0 {
				return Invalid(field+"method", "目标比例只填写基准项、比例和舍入规则")
			}
			if row.Rounding != "HALF_UP" {
				return Invalid(field+"rounding", "请选择最终金额四舍五入")
			}
			if _, _, err := Decimal(field+"percent", row.Percent, 4); err != nil {
				return err
			}
		default:
			return Invalid(field+"method", "请选择组成项的取值方式")
		}
	}
	done, visiting := map[string]bool{}, map[string]bool{}
	var visit func(int) error
	visit = func(i int) error {
		row := items[i]
		if done[row.ID] || row.Method != "PERCENT" {
			return nil
		}
		field := "components." + strconv.Itoa(i) + ".baseId"
		base, exists := byID[row.BaseID]
		if !exists {
			return Invalid(field, "请从本计划选择仍存在的基准项")
		}
		if visiting[row.ID] {
			return Invalid(field, "目标比例不能自引或循环引用")
		}
		if items[base].Currency != row.Currency || items[base].Frequency != row.Frequency || items[base].Unit != row.Unit {
			return Invalid(field, "基准项的币种、频率和单位须一致")
		}
		visiting[row.ID] = true
		if err := visit(base); err != nil {
			return err
		}
		delete(visiting, row.ID)
		done[row.ID] = true
		return nil
	}
	for i := range items {
		if err := visit(i); err != nil {
			return err
		}
	}
	return nil
}

// TargetQuote evaluates a validated chain using exact rational numbers. A
// range or a restricted base returns no reference amount; its formula remains.
func TargetQuote(items []Component, target, grade string, rateAmount func(RateLink) (string, bool)) (string, bool) {
	byID := make(map[string]Component, len(items))
	for _, v := range items {
		byID[v.ID] = v
	}
	item, ok := byID[target]
	if !ok {
		return "", false
	}
	factor := big.NewRat(1, 1)
	for steps := 0; steps < len(items); steps++ {
		switch item.Method {
		case "FIXED", "GRADE_RATE":
			amount := item.Amount
			if item.Method == "GRADE_RATE" {
				pos := slices.IndexFunc(item.Rates, func(r RateLink) bool { return r.GradeID == grade })
				if pos < 0 {
					return "", false
				}
				amount, ok = rateAmount(item.Rates[pos])
				if !ok {
					return "", false
				}
			}
			n, ok := new(big.Rat).SetString(amount)
			if !ok {
				return "", false
			}
			return roundReference(n.Mul(n, factor), byID[target].Precision), true
		case "PERCENT":
			n, ok := new(big.Rat).SetString(item.Percent)
			if !ok {
				return "", false
			}
			factor.Mul(factor, n).Quo(factor, big.NewRat(100, 1))
			item, ok = byID[item.BaseID]
			if !ok {
				return "", false
			}
		default:
			return "", false
		}
	}
	return "", false
}

func roundReference(value *big.Rat, places int) string {
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(places)), nil)
	numer := new(big.Int).Mul(value.Num(), scale)
	amount, rem := new(big.Int), new(big.Int)
	amount.QuoRem(numer, value.Denom(), rem)
	if rem.Lsh(rem, 1).Cmp(value.Denom()) >= 0 {
		amount.Add(amount, big.NewInt(1))
	}
	digits := amount.String()
	if places == 0 {
		return digits
	}
	if missing := places + 1 - len(digits); missing > 0 {
		digits = strings.Repeat("0", missing) + digits
	}
	return digits[:len(digits)-places] + "." + digits[len(digits)-places:]
}

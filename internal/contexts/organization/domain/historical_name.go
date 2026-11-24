package domain

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"unicode"
	"unicode/utf8"
)

// HistoricalName preserves an externally evidenced spelling. It is not a new
// name admission rule: callers must establish origin before selecting it.
func HistoricalName(value string) (string, error) {
	if !utf8.ValidString(value) || value == "" || utf8.RuneCountInString(value) > NameMaxLength {
		return "", foundation.Fail(400, "M02_INVALID_INPUT", "历史名称须为有效UTF-8且不超过200个字符")
	}
	meaningful := false
	for _, r := range value {
		if unicode.IsControl(r) || unicode.Is(unicode.Zl, r) || unicode.Is(unicode.Zp, r) || unicode.Is(unicode.Cf, r) {
			return "", foundation.Fail(400, "M02_INVALID_INPUT", "历史名称不能包含控制或不可见格式字符")
		}
		meaningful = meaningful || unicode.IsLetter(r) || unicode.IsNumber(r)
	}
	if !meaningful {
		return "", foundation.Fail(400, "M02_INVALID_INPUT", "历史名称缺少文字或数字")
	}
	return value, nil
}

package foundation

import (
	"errors"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

const NameMaxLength = 200
const NameGuidance = "名称最多200个字符"
const namePunctuation = ".．·・-‐‑–—_()（）[]［］【】'’‘\"“”/／&＆+＋,，、:：#＃%％"
const nameInvalid = "名称请使用文字、数字和名称标点，不可包含表情、装饰图形或不合法的不可见字符"

func nameIn(r rune, ranges []nameRuneSpan) bool {
	i := sort.Search(len(ranges), func(i int) bool { return ranges[i].hi >= r })
	return i < len(ranges) && ranges[i].lo <= r
}

// NormalizeName validates only newly supplied values. Stored snapshots, inherited
// fields and historical results are never cleaned or normalized while reading.
func NormalizeName(s string, required bool) (string, error) {
	fail := func(message string) (string, error) { return "", errors.New(message) }
	if !utf8.ValidString(s) {
		return fail("名称不是有效的 UTF-8")
	}
	// Reject line/control characters before trimming, including at either edge.
	for _, r := range s {
		if unicode.IsControl(r) || unicode.Is(unicode.Zl, r) || unicode.Is(unicode.Zp, r) {
			return fail("名称不能包含控制字符、换行或制表符")
		}
	}
	s = NFC(strings.TrimFunc(s, func(r rune) bool { return unicode.Is(unicode.Zs, r) }))
	if (required && s == "") || utf8.RuneCountInString(s) > NameMaxLength {
		return fail("名称应为 1 至 200 个字符")
	}
	if s == "" {
		return s, nil
	}
	runes := []rune(s)
	meaningful, markBase := false, false
	for i, r := range runes {
		if nameIn(r, nameUnicodeExtended_Pictographic) {
			return fail(nameInvalid)
		}
		switch {
		case unicode.Is(unicode.Variation_Selector, r):
			// Textual CJK/Mongolian variation only; FE0E/FE0F emoji styles
			// and selectors without a preceding letter are not name text.
			if i == 0 || !((unicode.Is(unicode.Han, runes[i-1]) && ((r >= 0xFE00 && r <= 0xFE0D) || (r >= 0xE0100 && r <= 0xE01EF))) ||
				(unicode.Is(unicode.Mongolian, runes[i-1]) && unicode.IsLetter(runes[i-1]) && (r >= 0x180B && r <= 0x180D || r == 0x180F))) {
				return fail(nameInvalid)
			}
		case r == 0x200C || r == 0x200D:
			if !nameJoiningContext(runes, i) {
				return fail(nameInvalid)
			}
			markBase = true
		case unicode.Is(unicode.Cf, r) || unicode.Is(unicode.Other_Default_Ignorable_Code_Point, r):
			return fail(nameInvalid)
		case unicode.IsLetter(r) || unicode.IsNumber(r):
			meaningful = true
			markBase = unicode.IsLetter(r)
		case unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Mc, r):
			if !markBase {
				return fail(nameInvalid)
			}
		case unicode.Is(unicode.Zs, r) || strings.ContainsRune(namePunctuation, r):
			markBase = false
		default:
			return fail(nameInvalid)
		}
	}
	if !meaningful {
		return fail("名称至少包含一个文字或数字")
	}
	return s, nil
}

// The two finite joining contexts follow UTS #39: virama joining, and ZWNJ
// between joining letters. Require letters from the same script on both sides.
func nameJoiningContext(s []rune, i int) bool {
	if i == 0 || i+1 == len(s) {
		return false
	}
	left, right := i-1, i+1
	for left >= 0 && nameIn(s[left], nameUnicodeT) {
		left--
	}
	for right < len(s) && nameIn(s[right], nameUnicodeT) {
		right++
	}
	if left < 0 || right == len(s) || !unicode.IsLetter(s[left]) || !unicode.IsLetter(s[right]) {
		return false
	}
	sameScript := false
	for script, table := range unicode.Scripts {
		if script != "Common" && script != "Inherited" && unicode.Is(table, s[left]) && unicode.Is(table, s[right]) {
			sameScript = true
			break
		}
	}
	if !sameScript {
		return false
	}
	if CanonicalCombiningClass(s[i-1]) == 9 && right == i+1 {
		return true
	}
	return s[i] == 0x200C && (nameIn(s[left], nameUnicodeL) || nameIn(s[left], nameUnicodeD)) &&
		(nameIn(s[right], nameUnicodeR) || nameIn(s[right], nameUnicodeD))
}

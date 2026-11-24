package domain

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

func Invalid(field, msg string) error {
	return &foundation.Fault{Status: 400, Code: "PERSON_INPUT", Field: field, Message: msg}
}
func Denied() error  { return foundation.Fail(403, "PERSON_DENIED", "没有本次办理权限") }
func Missing() error { return foundation.Fail(404, "PERSON_UNAVAILABLE", "所选资料不可用") }
func Conflict() error {
	return foundation.Fail(409, "PERSON_CONFLICT", "资料已变化或依据已被使用，请重新核对")
}
func Name(s string) (string, error) {
	if strings.IndexFunc(s, func(r rune) bool { return unicode.IsControl(r) || r == '\u2028' || r == '\u2029' }) >= 0 {
		return "", Invalid("name", "姓名不能含控制符或换行")
	}
	s = foundation.NFC(strings.TrimSpace(s))
	if utf8.RuneCountInString(s) > 200 || strings.IndexFunc(s, unicode.IsLetter) < 0 {
		return "", Invalid("name", "请输入有效姓名（最多 200 个字符）")
	}
	return s, nil
}

var numberRE = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9-]{0,30}[A-Za-z0-9])?$`)
var residentNumber = regexp.MustCompile(`^[0-9]{17}[0-9X]$`)
var phoneRE = regexp.MustCompile(`^\+?[0-9]([ -]?[0-9]){4,14}$`)

func Number(s string) (string, error) {
	s = strings.Trim(s, " ")
	if !numberRE.MatchString(s) {
		return "", Invalid("newNumber", "编号限 1 至 32 位字母、数字及中间连字符")
	}
	return strings.ToUpper(s), nil
}
func Material(kind, ref string) (string, error) {
	if kind != "personal" && kind != "form" {
		return "", Invalid("evidenceType", "请选择登记来源类型")
	}
	return reference(ref, "reference", "请输入登记单号")
}
func reference(s, field, message string) (string, error) {
	if strings.IndexFunc(s, func(r rune) bool { return unicode.IsControl(r) || r == '\u2028' || r == '\u2029' }) >= 0 {
		return "", Invalid(field, message)
	}
	s = strings.TrimSpace(s)
	if s == "" || utf8.RuneCountInString(s) > 200 {
		return "", Invalid(field, message)
	}
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' {
			return r - 'a' + 'A'
		}
		return r
	}, s), nil
}
func Document(kind, number string) (string, error) {
	switch kind {
	case "resident_id", "passport", "other":
	default:
		return "", Invalid("documentType", "请选择主要证件类型")
	}
	n, err := reference(number, "documentNumber", "请输入有效的主要证件号码")
	if err != nil {
		return "", err
	}
	if kind == "resident_id" && !residentNumber.MatchString(n) {
		return "", Invalid("documentNumber", "居民身份证号码应为18位，末位可以是 X")
	}
	return n, nil
}
func Contact(kind, value string) (string, error) {
	if kind == "phone" && phoneRE.MatchString(value) {
		return strings.NewReplacer(" ", "", "-", "").Replace(value), nil
	}
	if kind == "email" && len(value) <= 254 && strings.Count(value, "@") == 1 && strings.IndexFunc(value, func(r rune) bool { return r < 33 || r > 126 || r == ',' || r == ';' || r == '<' || r == '>' }) < 0 {
		l, d, _ := strings.Cut(value, "@")
		if l != "" && strings.Contains(d, ".") && !strings.HasPrefix(d, ".") && !strings.HasSuffix(d, ".") {
			return l + "@" + strings.ToLower(d), nil
		}
	}
	return "", Invalid("contacts", "请核对联系方式的类型和值")
}
func Mask(kind, v string) string {
	if kind == "phone" && len(v) >= 4 {
		return "****" + v[len(v)-4:]
	}
	if kind == "email" && len(v) > 0 {
		return v[:1] + "****"
	}
	return ""
}
func Cutoff(now time.Time, days int) time.Time {
	d := now.In(foundation.Shanghai)
	return time.Date(d.Year(), d.Month(), d.Day()+days, 0, 0, 0, 0, foundation.Shanghai)
}
func Decision(now time.Time, reason, ref, until string, permanent bool) error {
	if reason != "correction" && reason != "migration" {
		return Invalid("reason", "请选择改号原因")
	}
	if strings.TrimSpace(ref) == "" || utf8.RuneCountInString(ref) > 128 || strings.IndexFunc(ref, unicode.IsControl) >= 0 {
		return Invalid("decisionRef", "请输入客户决定文号")
	}
	if until == "" && permanent {
		return nil
	}
	t, e := time.ParseInLocation("2006-01-02", until, foundation.Shanghai)
	if e != nil || !now.Before(t) {
		return Invalid("validUntil", "请输入仍有效的决定截止日")
	}
	return nil
}

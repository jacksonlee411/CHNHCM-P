package domain

import (
	f "chnhcm.local/chnhcm/internal/foundation"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

var BasicFields = []string{"formerName", "englishName", "birthDate", "gender", "maritalStatus", "nationality", "ethnicity", "nativePlace", "birthPlace", "householdCategory", "householdNature", "politicalStatus"}
var DocumentFields = []string{"documentKind", "documentNumber", "documentName", "issuingCountry", "issuer", "issueDate", "expiryDate"}

func ProfileText(field, value string) (string, error) {
	value = strings.TrimSpace(value)
	if utf8.RuneCountInString(value) > 200 || strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return "", Invalid(field, "请填写最多 200 字的单行资料")
	}
	if slices.Contains([]string{"birthDate", "issueDate", "expiryDate"}, field) && value != "" {
		if !f.ValidDate(value) {
			return "", Invalid(field, "请输入有效日期 YYYY-MM-DD")
		}
	}
	return value, nil
}
func ValidateDocument(fields map[string]string, longTerm bool) error {
	number, err := Document(fields["documentKind"], fields["documentNumber"])
	if err != nil {
		return err
	}
	fields["documentNumber"] = number
	if end := fields["expiryDate"]; end != "" && (longTerm || end < fields["issueDate"]) {
		return Invalid("expiryDate", "到期日期须不早于签发日期，长期有效时请清空到期日期")
	}
	return nil
}

package domain

import (
	f "chnhcm.local/chnhcm/internal/foundation"
	"slices"
)

type ItemSection struct {
	Name, Action     string
	Fields, Required []string
}

var Addresses = ItemSection{"addresses", "profile", []string{"addressType", "addressCountry", "province", "area", "street", "postalCode", "addressPrimary"}, []string{"addressType", "addressCountry", "street"}}
var Family = ItemSection{"family", "family", []string{"familyName", "relationship", "familyBirthDate", "employer", "familyJob", "familyPhone", "familyEmail", "emergency"}, []string{"familyName", "relationship"}}

func (s ItemSection) Validate(fields map[string]string) error {
	for _, key := range s.Required {
		if fields[key] == "" {
			return Invalid(key, "请填写必填业务字段")
		}
	}
	switch s.Name {
	case BankAccounts.Name:
		if BankNumberKey(fields["accountNumber"]) == "********" {
			return Invalid("accountNumber", "请填写实际银行账号")
		}
		return nil
	case Education.Name:
		return validateEducation(fields)
	case Credentials.Name:
		return validateCredential(fields)
	case Addresses.Name:
		return nil
	}
	if !slices.Contains([]string{"配偶", "父母", "父亲", "母亲", "子女", "子", "女", "兄弟姐妹", "祖父母", "其他亲属", "朋友", "其他"}, fields["relationship"]) {
		return Invalid("relationship", "请选择与员工的关系")
	}
	if date := fields["familyBirthDate"]; date != "" && !f.ValidDate(date) {
		return Invalid("familyBirthDate", "请输入有效日期 YYYY-MM-DD")
	}
	for _, c := range []struct{ field, kind string }{{"familyPhone", "phone"}, {"familyEmail", "email"}} {
		if value := fields[c.field]; value != "" {
			normal, err := Contact(c.kind, value)
			if err != nil {
				return Invalid(c.field, "请核对联系方式")
			}
			fields[c.field] = normal
		}
	}
	if fields["emergency"] == "yes" && fields["familyPhone"] == "" && fields["familyEmail"] == "" {
		return Invalid("emergency", "紧急联系人须有电话或电子邮箱，请补填或撤下紧急联系人标记")
	}
	return nil
}

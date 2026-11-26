package domain

import (
	"slices"
	"time"

	f "chnhcm.local/chnhcm/internal/foundation"
)

var Education = ItemSection{"education", "qualification",
	[]string{"educationType", "school", "major", "educationLevel", "degree", "studyMode", "educationPath", "startMonth", "endMonth", "completion", "awardDate", "highestEducation", "highestDegree"},
	[]string{"school", "educationType"}}
var Credentials = ItemSection{"credentials", "qualification",
	[]string{"credentialKind", "credentialName", "credentialLevel", "specialty", "certificateNumber", "awardingBody", "issuedOn", "validFrom", "validUntil", "indefinite"},
	[]string{"credentialName", "credentialKind"}}

var ItemSections = []ItemSection{Addresses, Family, Education, Credentials, BankAccounts}

func EducationFieldApplies(kind, key string) bool {
	switch key {
	case "awardDate":
		return kind == "学位授予"
	case "startMonth", "completion":
		return kind == "学习经历"
	case "educationLevel", "studyMode", "educationPath", "endMonth", "highestEducation":
		return kind == "学习经历" || kind == "已取得学历"
	case "degree", "highestDegree":
		return kind == "学习经历" || kind == "学位授予"
	default:
		return true
	}
}

func validateEducation(v map[string]string) error {
	kind := v["educationType"]
	if !slices.Contains([]string{"学习经历", "已取得学历", "学位授予"}, kind) {
		return Invalid("educationType", "请选择教育记录类型")
	}
	for _, key := range Education.Fields {
		if v[key] != "" && !EducationFieldApplies(kind, key) {
			return Invalid(key, "该字段不适用于所选教育记录类型")
		}
	}
	if kind == "已取得学历" && v["educationLevel"] == "" {
		return Invalid("educationLevel", "请填写已取得的学历")
	}
	if kind == "学位授予" && v["degree"] == "" {
		return Invalid("degree", "请填写已授予的学位")
	}
	if value := v["completion"]; value != "" && !slices.Contains([]string{"在读", "已毕业", "结业", "肄业"}, value) {
		return Invalid("completion", "请选择教育完成状态")
	}
	for _, key := range []string{"startMonth", "endMonth"} {
		if month := v[key]; month != "" {
			if _, err := time.Parse("2006-01", month); err != nil || month[:4] == "0000" {
				return Invalid(key, "请输入有效年月 YYYY-MM")
			}
		}
	}
	if v["endMonth"] != "" && v["endMonth"] < v["startMonth"] {
		return Invalid("endMonth", "毕业或结束年月须不早于入学年月")
	}
	if day := v["awardDate"]; day != "" && !f.ValidDate(day) {
		return Invalid("awardDate", "请输入有效日期 YYYY-MM-DD")
	}
	if v["highestEducation"] != "" && (v["educationLevel"] == "" || kind == "学习经历" && v["completion"] != "已毕业") {
		return Invalid("highestEducation", "最高学历须有已取得学历的记录")
	}
	if v["highestDegree"] != "" && (v["degree"] == "" || kind == "学习经历" && v["completion"] == "在读") {
		return Invalid("highestDegree", "最高学位须有已授予学位的记录")
	}
	return nil
}

func validateCredential(v map[string]string) error {
	if !slices.Contains([]string{"职称", "职业资格", "职业技能等级", "执业许可", "专业认证"}, v["credentialKind"]) {
		return Invalid("credentialKind", "请选择凭证类别")
	}
	for _, key := range []string{"issuedOn", "validFrom", "validUntil"} {
		if day := v[key]; day != "" && !f.ValidDate(day) {
			return Invalid(key, "请输入有效日期 YYYY-MM-DD")
		}
	}
	if end := v["validUntil"]; end != "" && (v["indefinite"] != "" || end < v["validFrom"]) {
		return Invalid("validUntil", "截止日期须不早于有效起始日期，长期有效时请清空截止日期")
	}
	return nil
}

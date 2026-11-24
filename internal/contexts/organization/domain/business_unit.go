package domain

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"regexp"
	"strings"
)

type Create struct {
	Code, Name, EffectiveDate, OperationType, OperationReason string
	NameEn                                                    *string
}

var codePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,39}$`)

func NormalizeCode(code string) (string, error) {
	code = strings.TrimSpace(code)
	if !codePattern.MatchString(code) {
		return "", &foundation.Fault{Status: 400, Code: "M02_INVALID_INPUT", Field: "code", Message: "编码须为 1 至 40 位字母、数字、下划线或连字符，并以字母或数字开头"}
	}
	return code, nil
}

func ValidateCreate(in Create, enterpriseStart string) (Create, error) {
	var err error
	in.Code, err = NormalizeCode(in.Code)
	if err != nil {
		return in, err
	}
	data, err := (buPolicy{}).Normalize(BUData{in.Name, in.NameEn})
	if err != nil {
		return in, err
	}
	in.Name, in.NameEn = data.Name, data.NameEn
	if !foundation.ValidDate(in.EffectiveDate) || in.EffectiveDate < enterpriseStart {
		return in, &foundation.Fault{Status: 400, Code: "M02_INVALID_INPUT", Field: "effectiveDate", Message: "生效日期须在所属企业有效范围内"}
	}
	if in.OperationType != "CREATE" || OperationReasons("CREATE")[in.OperationReason] == "" {
		return in, &foundation.Fault{Status: 400, Code: "M02_INVALID_INPUT", Field: "operationReason", Message: "请选择适用的新建操作原因"}
	}
	return in, nil
}

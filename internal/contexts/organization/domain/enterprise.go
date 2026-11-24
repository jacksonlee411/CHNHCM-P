package domain

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"strings"
)

// EnterpriseInitialization has its own internal establishment contract, without
// public BU action/reason fields. Re-initialization identity is checked in storage.
type EnterpriseInitialization struct{ Code, Name, Start string }

func ValidateEnterpriseInitialization(in EnterpriseInitialization) (EnterpriseInitialization, error) {
	in.Code = strings.TrimSpace(in.Code)
	if !codePattern.MatchString(in.Code) {
		return in, invalid("M02_INVALID_INPUT", "code", "企业编码须为1至40位字母、数字、下划线或连字符")
	}
	name, err := NormalizeName(in.Name, true)
	if err != nil {
		return in, err
	}
	in.Name = name
	if !foundation.ValidDate(in.Start) {
		return in, invalid("M02_INVALID_INPUT", "start", "企业开始日期无效")
	}
	return in, nil
}

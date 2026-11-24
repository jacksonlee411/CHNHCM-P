// Package legalentity owns declared organization identity data, not legal uses
// or personnel relationships. The dated kernel owns periods and lifecycle.
package legalentity

import (
	org "chnhcm.local/chnhcm/internal/contexts/organization/domain"
	"chnhcm.local/chnhcm/internal/contexts/organization/domain/country"
	"chnhcm.local/chnhcm/internal/contexts/organization/domain/dated"
	"chnhcm.local/chnhcm/internal/foundation"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Data struct {
	Name                                                           string
	NameEn                                                         *string
	CountryID, SubjectType                                         string
	RegistrationType, RegistrationNumber, EstablishingOrganization *string
	Adoption                                                       country.Adoption
}
type Patch map[string]*string
type Policy struct{}
type Record = dated.Record[Data]
type Candidate = dated.Candidate[Data]
type Item = dated.Item[Patch]
type Timeline struct {
	ID, Enterprise, Code, First, Source string
	SourceNo                            int64
	Records                             []Record
}

var Kernel = dated.Kernel[Data, Patch]{Policy: Policy{}}

func invalid(field, message string) error {
	return &foundation.Fault{Status: 400, Code: "M02_INVALID_INPUT", Field: field, Message: message}
}
func clone(v *string) *string {
	if v == nil {
		return nil
	}
	s := *v
	return &s
}
func equal(a, b *string) bool { return a == nil && b == nil || a != nil && b != nil && *a == *b }
func (Policy) Clone(v Data) Data {
	v.NameEn = clone(v.NameEn)
	v.RegistrationType = clone(v.RegistrationType)
	v.RegistrationNumber = clone(v.RegistrationNumber)
	v.EstablishingOrganization = clone(v.EstablishingOrganization)
	return v
}
func optional(v *string, field string, limit int) (*string, error) {
	if v == nil {
		return nil, nil
	}
	if strings.ContainsFunc(*v, unicode.IsControl) {
		return nil, invalid(field, "不能包含控制字符")
	}
	s := strings.TrimSpace(*v)
	if utf8.RuneCountInString(s) > limit {
		return nil, invalid(field, "内容超过允许长度")
	}
	if s == "" {
		return nil, nil
	}
	return &s, nil
}
func (p Policy) Normalize(v Data) (Data, error) {
	fields := Patch{"name": &v.Name, "nameEn": v.NameEn, "countryRegionId": &v.CountryID, "subjectType": &v.SubjectType, "registrationType": v.RegistrationType, "registrationNumber": v.RegistrationNumber, "establishingOrganization": v.EstablishingOrganization}
	return p.Apply(v, &fields)
}
func (p Policy) Apply(v Data, patch *Patch) (Data, error) {
	if patch == nil || len(*patch) == 0 {
		return v, nil
	}
	v = p.Clone(v)
	keys := make([]string, 0, len(*patch))
	for k := range *patch {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		raw := (*patch)[k]
		field := "changes." + k
		if raw == nil && slices.Contains([]string{"name", "countryRegionId", "subjectType"}, k) {
			return v, invalid(field, "必填字段不能为空")
		}
		var e error
		switch k {
		case "name":
			v.Name, e = org.NormalizeName(*raw, true)
		case "nameEn":
			v.NameEn = nil
			if raw != nil {
				var s string
				s, e = org.NormalizeName(*raw, false)
				if s != "" {
					v.NameEn = &s
				}
			}
		case "countryRegionId":
			if !foundation.ValidID(*raw) {
				return v, invalid(field, "国家身份无效")
			}
			v.CountryID = *raw
		case "subjectType":
			if !slices.Contains([]string{"legal_person", "branch", "unincorporated", "other"}, *raw) {
				return v, invalid(field, "请选择主体类别")
			}
			v.SubjectType = *raw
		case "registrationType":
			v.RegistrationType, e = optional(raw, field, 80)
		case "registrationNumber":
			v.RegistrationNumber, e = optional(raw, field, 100)
		case "establishingOrganization":
			v.EstablishingOrganization, e = optional(raw, field, 200)
		default:
			return v, &foundation.Fault{Status: 400, Code: "M02_FORBIDDEN_INPUT", Field: field, Message: "字段不允许通过资料补丁修改"}
		}
		if e != nil {
			return v, invalid(field, "字段不符合文字或长度规则")
		}
	}
	// Check the complete patch, never an intermediate assignment order.
	if (v.RegistrationType == nil) != (v.RegistrationNumber == nil) {
		field := "registrationNumber"
		if v.RegistrationType == nil {
			field = "registrationType"
		}
		return v, invalid("changes."+field, "登记类型和登记号码须同时填写或同时清空")
	}
	return v, nil
}
func (Policy) EmptyPatch(p Patch) bool { return len(p) == 0 }
func SameCountry(a, b Data) bool       { return a.CountryID == b.CountryID }
func (Policy) EqualData(a, b Data) bool {
	return a.Name == b.Name && equal(a.NameEn, b.NameEn) && SameCountry(a, b) && a.SubjectType == b.SubjectType && equal(a.RegistrationType, b.RegistrationType) && equal(a.RegistrationNumber, b.RegistrationNumber) && equal(a.EstablishingOrganization, b.EstablishingOrganization)
}
func (Policy) EqualRevision(a, b Data) bool { return a.Adoption == b.Adoption }
func (Policy) OperationReasons(kind string) map[string]string {
	switch kind {
	case "CREATE":
		return map[string]string{"NEW_LEGAL_ENTITY": "新设法律实体"}
	case "CHANGE":
		return map[string]string{"NAME_ADJUSTMENT": "名称调整", "REGISTRATION_ADJUSTMENT": "登记资料调整", "OTHER_DATA_ADJUSTMENT": "其他资料调整"}
	case "DEACTIVATE":
		return map[string]string{"ENTITY_CLOSURE": "法律实体停用"}
	case "REACTIVATE":
		return map[string]string{"ENTITY_RESUMPTION": "法律实体恢复"}
	default:
		return map[string]string{}
	}
}
func (Policy) OperationLabel(kind string) string { return dated.OperationLabel(kind) }
func SubjectTypeLabel(kind string) string {
	return map[string]string{"legal_person": "法人组织", "branch": "分支机构", "unincorporated": "非法人组织", "other": "其他组织"}[kind]
}
func NeedsAdoption(before []Record, r Record, predecessor *Record) bool {
	if r.Type == "DEACTIVATE" {
		return false
	}
	for _, old := range before {
		if old.ID == r.ID {
			return old.Date != r.Date || old.Type != r.Type || !SameCountry(old.Data, r.Data)
		}
	}
	return r.Type == "CREATE" || r.Type == "REACTIVATE" || predecessor == nil || !SameCountry(predecessor.Data, r.Data)
}
func ValidateAdoption(v Data) error { return country.ValidateAdoption(v.CountryID, v.Adoption) }

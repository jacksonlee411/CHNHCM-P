// Package location owns enterprise location data and exact address adoption.
// Ordering, periods and lifecycle remain owned by the shared dated kernel.
package location

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	org "chnhcm.local/chnhcm/internal/contexts/organization/domain"
	"chnhcm.local/chnhcm/internal/contexts/organization/domain/dated"
	geo "chnhcm.local/chnhcm/internal/contexts/organization/domain/geoarea"
	"chnhcm.local/chnhcm/internal/foundation"
)

type Data struct {
	Name        string
	NameEn      *string
	CountryID   string
	AreaID      *string
	AddressLine string
	PostalCode  *string
	Adoption    geo.Adoption
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
	v.AreaID = clone(v.AreaID)
	v.PostalCode = clone(v.PostalCode)
	v.Adoption.Parents = slices.Clone(v.Adoption.Parents)
	return v
}
func text(v, field string, limit int, required bool) (string, error) {
	// Check controls before trimming, so a line break cannot silently disappear.
	if strings.ContainsFunc(v, unicode.IsControl) {
		return "", invalid(field, "不能包含控制字符")
	}
	v = strings.TrimSpace(v)
	if required && v == "" || utf8.RuneCountInString(v) > limit {
		return "", invalid(field, "内容为空或超过允许长度")
	}
	return v, nil
}
func optional(v *string, field string, limit int) (*string, error) {
	if v == nil {
		return nil, nil
	}
	s, e := text(*v, field, limit, false)
	if e != nil || s == "" {
		return nil, e
	}
	return &s, nil
}
func (p Policy) Normalize(v Data) (Data, error) {
	fields := Patch{"name": &v.Name, "nameEn": v.NameEn, "countryRegionId": &v.CountryID, "geoAreaId": v.AreaID, "addressLine": &v.AddressLine, "postalCode": v.PostalCode}
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
		if raw == nil && slices.Contains([]string{"name", "countryRegionId", "addressLine"}, k) {
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
		case "geoAreaId":
			v.AreaID = nil
			if raw != nil && *raw != "" {
				if !foundation.ValidID(*raw) {
					return v, invalid(field, "区域身份无效")
				}
				v.AreaID = clone(raw)
			}
		case "addressLine":
			v.AddressLine, e = text(*raw, field, 1001, true)
		case "postalCode":
			v.PostalCode, e = optional(raw, field, 20)
		default:
			return v, &foundation.Fault{Status: 400, Code: "M02_FORBIDDEN_INPUT", Field: field, Message: "字段不允许通过资料补丁修改"}
		}
		if e != nil {
			return v, invalid(field, "字段不符合文字或长度规则")
		}
	}
	return v, nil
}
func (Policy) EmptyPatch(p Patch) bool { return len(p) == 0 }
func SameAddress(a, b Data) bool       { return a.CountryID == b.CountryID && equal(a.AreaID, b.AreaID) }
func (Policy) EqualData(a, b Data) bool {
	return a.Name == b.Name && equal(a.NameEn, b.NameEn) && SameAddress(a, b) && a.AddressLine == b.AddressLine && equal(a.PostalCode, b.PostalCode)
}
func (Policy) EqualRevision(a, b Data) bool {
	return a.Adoption.Date == b.Adoption.Date && a.Adoption.Country == b.Adoption.Country && slices.Equal(a.Adoption.Parents, b.Adoption.Parents)
}
func (Policy) OperationReasons(kind string) map[string]string {
	switch kind {
	case "CREATE":
		return map[string]string{"NEW_LOCATION": "新设地点"}
	case "CHANGE":
		return map[string]string{"NAME_ADJUSTMENT": "名称调整", "ADDRESS_ADJUSTMENT": "地址调整", "OTHER_DATA_ADJUSTMENT": "其他资料调整"}
	case "DEACTIVATE":
		return map[string]string{"LOCATION_CLOSURE": "地点停用"}
	case "REACTIVATE":
		return map[string]string{"LOCATION_RESUMPTION": "地点恢复"}
	default:
		return map[string]string{}
	}
}
func (Policy) OperationLabel(kind string) string { return dated.OperationLabel(kind) }

// NeedsAdoption examines final records after dated.Prepare. A newly inserted
// CHANGE inherits its final predecessor; an independent successor keeps its own
// evidence. DEACTIVATE never captures an independent address.
func NeedsAdoption(before []Record, r Record, predecessor *Record) bool {
	if r.Type == "DEACTIVATE" {
		return false
	}
	for _, old := range before {
		if old.ID == r.ID {
			return old.Date != r.Date || old.Type != r.Type || !SameAddress(old.Data, r.Data)
		}
	}
	return r.Type == "CREATE" || r.Type == "REACTIVATE" || predecessor == nil || !SameAddress(predecessor.Data, r.Data)
}
func ValidateAdoption(v Data) error {
	parent := ""
	if v.AreaID != nil {
		parent = *v.AreaID
	}
	return geo.ValidateAdoption(geo.Structure{CountryID: v.CountryID, ParentID: parent}, v.Adoption)
}

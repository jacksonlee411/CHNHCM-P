// Package geoarea owns fixed geographic structure and display data. Timeline
// calculation belongs exclusively to the shared dated kernel.
package geoarea

import (
	org "chnhcm.local/chnhcm/internal/contexts/organization/domain"
	"chnhcm.local/chnhcm/internal/contexts/organization/domain/dated"
	"chnhcm.local/chnhcm/internal/foundation"
	"slices"
	"strings"
)

const MinDate = "0001-01-01"

type Structure struct{ CountryID, CountryCode, Scheme, Code, Nature, Level, ParentID string }
type Reference struct{ ObjectID, SourceVersion, RecordID, Revision string }
type Adoption struct {
	Date    string
	Country Reference
	Parents []Reference
}
type Data struct {
	Name      string
	NameEn    *string
	Structure Structure
	Adoption  Adoption
}
type Patch map[string]*string
type Policy struct{}
type Record = dated.Record[Data]
type Candidate = dated.Candidate[Data]
type Item = dated.Item[Patch]

var Kernel = dated.Kernel[Data, Patch]{Policy: Policy{}}

func invalid(field, message string) error {
	return &foundation.Fault{Status: 400, Code: "M02_INVALID_INPUT", Field: field, Message: message}
}
func (Policy) Clone(d Data) Data {
	if d.NameEn != nil {
		s := *d.NameEn
		d.NameEn = &s
	}
	d.Adoption.Parents = slices.Clone(d.Adoption.Parents)
	return d
}
func normalizeDisplay(d Data, name, english bool) (Data, error) {
	var e error
	if name {
		d.Name, e = org.NormalizeName(d.Name, true)
		if e != nil {
			return d, invalid("name", "名称不符合文字规则")
		}
	}
	if english && d.NameEn != nil {
		s, e := org.NormalizeName(*d.NameEn, false)
		if e != nil {
			return d, invalid("nameEn", "英文名称不符合文字规则")
		}
		d.NameEn = nil
		if s != "" {
			d.NameEn = &s
		}
	}
	return (Policy{}).Clone(d), nil
}
func (Policy) Normalize(d Data) (Data, error) {
	if e := ValidateStructure(d.Structure); e != nil {
		return d, e
	}
	if e := ValidateAdoption(d.Structure, d.Adoption); e != nil {
		return d, e
	}
	return normalizeDisplay(d, true, true)
}
func (Policy) Apply(d Data, p *Patch) (Data, error) {
	if p == nil || len(*p) == 0 {
		return d, invalid("changes", "至少明确一个资料字段")
	}
	keys := make([]string, 0, len(*p))
	for k := range *p {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		v := (*p)[k]
		switch k {
		case "name":
			if v == nil {
				return d, invalid("changes.name", "名称不能为空")
			}
			d.Name = *v
		case "nameEn":
			d.NameEn = v
		default:
			return d, &foundation.Fault{Status: 400, Code: "M02_FORBIDDEN_INPUT", Field: "changes." + k, Message: "区划结构和记录日期不允许通过显示资料更正修改"}
		}
	}
	_, n := (*p)["name"]
	_, en := (*p)["nameEn"]
	return normalizeDisplay(d, n, en)
}
func (Policy) EmptyPatch(p Patch) bool { return len(p) == 0 }
func (Policy) EqualData(a, b Data) bool {
	return a.Name == b.Name && (a.NameEn == nil && b.NameEn == nil || a.NameEn != nil && b.NameEn != nil && *a.NameEn == *b.NameEn) && a.Structure == b.Structure
}
func (Policy) EqualRevision(a, b Data) bool {
	return a.Adoption.Date == b.Adoption.Date && a.Adoption.Country == b.Adoption.Country && slices.Equal(a.Adoption.Parents, b.Adoption.Parents)
}
func (Policy) OperationReasons(kind string) map[string]string {
	switch kind {
	case "CREATE":
		return map[string]string{"SOURCE_INITIALIZATION": "来源初始化"}
	case "CHANGE":
		return map[string]string{"DISPLAY_DATA_UPDATE": "显示资料更新"}
	case "DEACTIVATE":
		return map[string]string{"STOP_SELECTION": "停止选用"}
	case "REACTIVATE":
		return map[string]string{"RESUME_SELECTION": "恢复选用"}
	default:
		return map[string]string{}
	}
}
func (Policy) OperationLabel(kind string) string {
	switch kind {
	case "DEACTIVATE":
		return "停止选用"
	case "REACTIVATE":
		return "恢复选用"
	default:
		return dated.OperationLabel(kind)
	}
}
func digits(s string, n int) bool {
	return len(s) == n && strings.IndexFunc(s, func(r rune) bool { return r < '0' || r > '9' }) < 0
}
func ValidateStructure(s Structure) error {
	if !foundation.ValidID(s.CountryID) || s.ParentID != "" && !foundation.ValidID(s.ParentID) {
		return invalid("countryRegionId", "国家或上级区域身份无效")
	}
	valid := false
	switch s.CountryCode {
	case "CN":
		valid = s.Scheme == "CN-MCA" && s.Nature == "administrative" && slices.Contains([]string{"province", "prefecture", "county"}, s.Level) && digits(s.Code, 6)
	case "HK":
		valid = s.Level == "region" && s.Scheme == "HK-IF-REGION" && s.Nature == "address_region" && slices.Contains([]string{"HK", "KLN", "NT"}, s.Code) || s.Level == "district" && s.Scheme == "HK-IF-DISTRICT" && s.Nature == "administrative" && len(s.Code) >= 2 && len(s.Code) <= 3 && strings.IndexFunc(s.Code, func(r rune) bool { return r < 'A' || r > 'Z' }) < 0
	case "MO":
		valid = s.ParentID == "" && s.Code == "" && s.Nature == "address_region" &&
			(s.Scheme == "MO-CTT" && s.Level == "postal-region" || s.Scheme == "MO-CADASTRE" && s.Level == "parish")
	case "TW":
		valid = s.Scheme == "TW-NLSC" && s.Nature == "administrative" && (s.Level == "county-city" && digits(s.Code, 5) || s.Level == "township" && digits(s.Code, 8))
	}
	if !valid {
		return invalid("structure", "区划代码、性质或层级不属于本期准入范围")
	}
	return nil
}
func ValidateAdoption(s Structure, a Adoption) error {
	if !foundation.ValidDate(a.Date) || a.Country.ObjectID != s.CountryID {
		return invalid("adoption", "采用日期或国家引用不一致")
	}
	refs := append([]Reference{a.Country}, a.Parents...)
	seen := map[string]bool{}
	for _, r := range refs {
		if !foundation.ValidID(r.ObjectID) || !foundation.ValidID(r.SourceVersion) || !foundation.ValidID(r.RecordID) || !foundation.ValidID(r.Revision) || seen[r.ObjectID] {
			return invalid("adoption", "准确引用身份不完整或重复")
		}
		seen[r.ObjectID] = true
	}
	if s.ParentID == "" && len(a.Parents) != 0 || s.ParentID != "" && (len(a.Parents) == 0 || a.Parents[len(a.Parents)-1].ObjectID != s.ParentID) {
		return invalid("adoption", "采用链与直接上级不一致")
	}
	return nil
}

type Origin struct {
	Structure
	Key, SourceVersion, PackageVersion, PackageDigest, ArchiveDigest string
}
type Timeline struct {
	ID, First, Source string
	SourceNo          int64
	Origin            Origin
	Records           []Record
}

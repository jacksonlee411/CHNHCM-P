// Package country owns instance-scoped reference data. The shared dated kernel
// owns date ordering, inheritance, lifecycle and revision calculation.
package country

import (
	"slices"
	"strings"

	org "chnhcm.local/chnhcm/internal/contexts/organization/domain"
	"chnhcm.local/chnhcm/internal/contexts/organization/domain/dated"
	"chnhcm.local/chnhcm/internal/foundation"
)

const MinDate = "0001-01-01"
const Scheme = "odoo-country"

type Data struct {
	Name              string
	NameEn, PhoneCode *string
}
type Patch map[string]*string
type Policy struct{}
type Record = dated.Record[Data]
type Candidate = dated.Candidate[Data]
type Item = dated.Item[Patch]

var Kernel = dated.Kernel[Data, Patch]{Policy: Policy{}}

type Origin struct {
	Scheme, Key, Code, Name, NameEn, PackageVersion, PackageDigest, ArchiveDigest string
}
type Timeline struct {
	ID, First, Source string
	SourceNo          int64
	Origin            Origin
	Records           []Record
}

func invalid(field, message string) error {
	return &foundation.Fault{Status: 400, Code: "M02_INVALID_INPUT", Field: field, Message: message}
}
func clone(p *string) *string {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}
func (Policy) Clone(d Data) Data {
	d.NameEn = clone(d.NameEn)
	d.PhoneCode = clone(d.PhoneCode)
	return d
}
func (Policy) EmptyPatch(p Patch) bool          { return len(p) == 0 }
func (p Policy) Normalize(d Data) (Data, error) { return p.normalize(d, true, true, true) }
func (p Policy) normalize(d Data, name, english, phone bool) (Data, error) {
	if name {
		v, e := org.NormalizeName(d.Name, true)
		if e != nil {
			return d, invalid("name", "名称不符合文字规则")
		}
		d.Name = v
	}
	if english && d.NameEn != nil {
		v, e := org.NormalizeName(*d.NameEn, false)
		if e != nil {
			return d, invalid("nameEn", "英文名称不符合文字规则")
		}
		d.NameEn = nil
		if v != "" {
			d.NameEn = &v
		}
	}
	if phone && d.PhoneCode != nil {
		v := strings.TrimSpace(*d.PhoneCode)
		d.PhoneCode = nil
		if v != "" {
			if len(v) > 6 {
				return d, invalid("phoneCode", "电话区号最多6位数字")
			}
			for _, r := range v {
				if r < '0' || r > '9' {
					return d, invalid("phoneCode", "电话区号须为数字")
				}
			}
			d.PhoneCode = &v
		}
	}
	return p.Clone(d), nil
}
func (p Policy) Apply(d Data, patch *Patch) (Data, error) {
	if patch == nil || len(*patch) == 0 {
		return d, invalid("changes", "至少明确一个资料字段")
	}
	keys := make([]string, 0, len(*patch))
	for k := range *patch {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		v := (*patch)[k]
		switch k {
		case "name":
			if v == nil {
				return d, invalid("changes.name", "名称不能为空")
			}
			d.Name = *v
		case "nameEn":
			d.NameEn = v
		case "phoneCode":
			d.PhoneCode = v
		default:
			return d, &foundation.Fault{Status: 400, Code: "M02_FORBIDDEN_INPUT", Field: "changes." + k, Message: "来源身份不允许通过资料维护修改"}
		}
	}
	_, n := (*patch)["name"]
	_, e := (*patch)["nameEn"]
	_, f := (*patch)["phoneCode"]
	return p.normalize(d, n, e, f)
}
func equal(a, b *string) bool { return a == nil && b == nil || a != nil && b != nil && *a == *b }
func (Policy) EqualData(a, b Data) bool {
	return a.Name == b.Name && equal(a.NameEn, b.NameEn) && equal(a.PhoneCode, b.PhoneCode)
}
func (Policy) EqualRevision(a, b Data) bool { return true }
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
func ValidateOrigin(o Origin) error {
	if o.Scheme != Scheme || len(o.Code) != 2 || o.Code[0] < 'A' || o.Code[0] > 'Z' || o.Code[1] < 'A' || o.Code[1] > 'Z' || len(o.Key) != 7 || !strings.HasPrefix(o.Key, "base.") || o.Key[5] < 'a' || o.Key[5] > 'z' || o.Key[6] < 'a' || o.Key[6] > 'z' {
		return invalid("sourceKey", "来源代码或命名空间无效")
	}
	if o.Name == "" || o.PackageVersion == "" || len(o.PackageDigest) != 64 || len(o.ArchiveDigest) != 64 {
		return invalid("source", "来源版次及摘要不完整")
	}
	return nil
}

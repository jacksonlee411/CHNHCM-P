// Package enterprise owns the single enterprise timeline. The shared dated kernel
// owns date ordering, inheritance, lifecycle and revision calculation.
package enterprise

import (
	"slices"

	org "chnhcm.local/chnhcm/internal/contexts/organization/domain"
	"chnhcm.local/chnhcm/internal/contexts/organization/domain/dated"
	"chnhcm.local/chnhcm/internal/foundation"
)

const MinDate = "0001-01-01"

type Data struct {
	Name   string
	NameEn *string
}
type Patch map[string]*string
type Policy struct{}
type Record = dated.Record[Data]
type Candidate = dated.Candidate[Data]
type Item = dated.Item[Patch]

var Kernel = dated.Kernel[Data, Patch]{Policy: Policy{}}

type Timeline struct {
	ID, Code, First, Source, InitializationDigest string
	SourceNo                                      int64
	Records                                       []Record
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
	return d
}
func (Policy) EmptyPatch(p Patch) bool          { return len(p) == 0 }
func (p Policy) Normalize(d Data) (Data, error) { return p.normalize(d, true, true) }
func (p Policy) normalize(d Data, name, english bool) (Data, error) {
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
		default:
			return d, &foundation.Fault{Status: 400, Code: "M02_FORBIDDEN_INPUT", Field: "changes." + k, Message: "字段不允许通过资料维护修改"}
		}
	}
	_, n := (*patch)["name"]
	_, e := (*patch)["nameEn"]
	return p.normalize(d, n, e)
}
func equal(a, b *string) bool { return a == nil && b == nil || a != nil && b != nil && *a == *b }
func (Policy) EqualData(a, b Data) bool {
	return a.Name == b.Name && equal(a.NameEn, b.NameEn)
}
func (Policy) EqualRevision(a, b Data) bool                   { return true }
func (Policy) OperationReasons(kind string) map[string]string { return org.OperationReasons(kind) }
func (Policy) OperationLabel(kind string) string              { return org.OperationLabel(kind) }

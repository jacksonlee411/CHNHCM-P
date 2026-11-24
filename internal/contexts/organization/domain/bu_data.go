package domain

import (
	"reflect"

	"chnhcm.local/chnhcm/internal/contexts/organization/domain/dated"
	"chnhcm.local/chnhcm/internal/foundation"
)

type BUData struct {
	Name   string
	NameEn *string
}
type BUField struct {
	Present bool
	Value   *string
}
type BUPatch struct {
	Name, NameEn BUField
	Forbidden    bool
}
type buPolicy struct{}

func (buPolicy) EmptyPatch(p BUPatch) bool {
	return p.Empty()
}

var buKernel = dated.Kernel[BUData, BUPatch]{Policy: buPolicy{}}

func (buPolicy) Clone(d BUData) BUData {
	if d.NameEn != nil {
		v := *d.NameEn
		d.NameEn = &v
	}
	return d
}
func (buPolicy) Normalize(d BUData) (BUData, error) {
	n, err := NormalizeName(d.Name, true)
	if err != nil {
		err.(*foundation.Fault).Field = "name"
		return d, err
	}
	d.Name = n
	if d.NameEn != nil {
		n, err = NormalizeName(*d.NameEn, false)
		if err != nil {
			err.(*foundation.Fault).Field = "nameEn"
			return d, err
		}
		d.NameEn = ptr(n)
	}
	return d, nil
}
func (buPolicy) Apply(d BUData, p *BUPatch) (BUData, error) {
	if p == nil || !p.Name.Present && !p.NameEn.Present && !p.Forbidden {
		return d, invalid("M02_INVALID_INPUT", "changes", "没有修改内容")
	}
	if p.Forbidden {
		return d, invalid("M02_FORBIDDEN_INPUT", "changes", "字段不允许修改")
	}
	for _, field := range []struct {
		name  string
		value BUField
	}{{"name", p.Name}, {"nameEn", p.NameEn}} {
		if !field.value.Present {
			continue
		}
		if field.name == "name" && field.value.Value == nil {
			return d, invalid("M02_INVALID_INPUT", "changes.name", "中文名称不能为空")
		}
		v := ""
		if field.value.Value != nil {
			v = *field.value.Value
		}
		n, err := NormalizeName(v, field.name == "name")
		if err != nil {
			err.(*foundation.Fault).Field = "changes." + field.name
			return d, err
		}
		if field.name == "name" {
			if !sameName(d.Name, n) {
				d.Name = n
			}
		} else if d.NameEn == nil || !sameName(*d.NameEn, n) {
			d.NameEn = ptr(n)
		}
	}
	return d, nil
}
func (buPolicy) EqualData(a, b BUData) bool {
	return a.Name == b.Name && reflect.DeepEqual(a.NameEn, b.NameEn)
}

// BU has no object-owned adoption metadata beyond the common record fields.
func (buPolicy) EqualRevision(a, b BUData) bool              { return true }
func (buPolicy) OperationReasons(t string) map[string]string { return OperationReasons(t) }
func (buPolicy) OperationLabel(t string) string              { return OperationLabel(t) }

// PatchFromFields is the one-way API input adapter. Presence and explicit nil
// remain distinct; the typed kernel never carries a mutable request map.
func PatchFromFields(fields map[string]*string) *BUPatch {
	if fields == nil {
		return nil
	}
	p := &BUPatch{}
	for name, value := range fields {
		if value != nil {
			copy := *value
			value = &copy
		}
		field := BUField{Present: true, Value: value}
		switch name {
		case "name":
			p.Name = field
		case "nameEn":
			p.NameEn = field
		default:
			p.Forbidden = true
		}
	}
	return p
}
func (p BUPatch) Empty() bool { return !p.Name.Present && !p.NameEn.Present && !p.Forbidden }

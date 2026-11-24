package department

import (
	"strings"

	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
)

// Collection members borrow source facts and retain only filtering, ordering
// and path material. Public response DTOs are assembled by page.
type listMember struct {
	position int
	t        *d.Timeline
	r        *d.Record
	fields   uint16
	bu       *org.DepartmentReference
	status   *string
	path     listPathValue
}

const (
	readCode uint16 = 1 << iota
	readName
	readShort
	readDate
	readStatus
	readBU
	readPath
	readParent
)

func fieldBit(field string) uint16 {
	switch field {
	case "code":
		return readCode
	case "name":
		return readName
	case "shortName":
		return readShort
	case "effectiveDate":
		return readDate
	case "status":
		return readStatus
	case "managementBuId":
		return readBU
	case "parentDepartmentId":
		return readParent
	case "departmentPath":
		return readPath
	}
	return 0
}

func readMask(fields []string) uint16 {
	var mask uint16
	for _, field := range fields {
		mask |= fieldBit(field)
	}
	return mask
}
func (m *listMember) has(field string) bool {
	return m.fields&fieldBit(field) != 0
}
func (m *listMember) code() *string {
	if m.fields&readCode == 0 {
		return nil
	}
	return borrowedString(&m.t.Code)
}
func (m *listMember) name() *string {
	if m.fields&readName == 0 {
		return nil
	}
	return borrowedString(&m.r.Data.Name)
}
func (m *listMember) field(name string) *string {
	if !m.has(name) {
		return nil
	}
	switch name {
	case "code":
		return borrowedString(&m.t.Code)
	case "name":
		return borrowedString(&m.r.Data.Name)
	case "shortName":
		return m.r.Data.ShortName
	case "effectiveDate":
		return borrowedString(&m.r.Date)
	case "status":
		return m.status
	case "departmentPath":
		if m.path.prefix != nil && m.path.prefix.visible {
			return borrowedString(&m.path.prefix.text)
		}
	}
	return nil
}
func (m *listMember) summary() org.DepartmentSummary {
	return org.DepartmentSummary{ID: m.t.ID, Code: m.field("code"), Name: m.field("name"), ShortName: m.field("shortName"), EffectiveDate: m.field("effectiveDate"), Status: m.field("status"), ManagementBU: m.bu, Path: m.path.summary()}
}
func memberSummaries(rows []listMember) []org.DepartmentSummary {
	out := make([]org.DepartmentSummary, len(rows))
	for i := range rows {
		out[i] = rows[i].summary()
	}
	return out
}
func (m *listMember) matches(in org.DepartmentListInput) bool {
	text := func(p *string) string {
		if p == nil {
			return ""
		}
		return strings.ToLower(*p)
	}
	if in.Q != "" && !strings.Contains(text(m.field("code")), in.Q) && !strings.Contains(text(m.field("name")), in.Q) {
		return false
	}
	if in.Status != "all" && text(m.status) != in.Status {
		return false
	}
	if in.Filter == nil {
		return true
	}
	f := in.Filter
	value := text(m.field(f.Field))
	if f.Field == "managementBuId" && m.bu != nil {
		value = m.bu.ID
	}
	switch f.Operator {
	case "contains":
		return strings.Contains(value, f.Value)
	case "equals", "is":
		return value == f.Value
	case "isAnyOf":
		for _, v := range strings.Split(f.Value, ",") {
			if value == v {
				return true
			}
		}
		return false
	case "before":
		return value < f.Value
	case "after":
		return value > f.Value
	case "onOrBefore":
		return value <= f.Value
	case "onOrAfter":
		return value >= f.Value
	}
	return false
}

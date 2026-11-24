// Package department owns Department data and administrative relationships.
// Ordering, lifecycle, inheritance and date windows belong to dated.
package department

import (
	"reflect"
	"slices"

	org "chnhcm.local/chnhcm/internal/contexts/organization/domain"
	"chnhcm.local/chnhcm/internal/contexts/organization/domain/dated"
	"chnhcm.local/chnhcm/internal/foundation"
)

type MemberRef struct {
	DepartmentID, Source, MemberID, BU string
	// Exact edge semantics loaded with the reference; never a second persisted parent.
	Start, End, Parent, EdgeID, Revision string
}
type Data struct {
	Name       string
	ShortName  *string
	BU, Parent string
	Member     MemberRef
}
type Patch map[string]*string
type Policy struct{ historicalName bool }

var Kernel = dated.Kernel[Data, Patch]{Policy: Policy{}}

// HistoricalNameKernel is only for an independently verified historical input.
// Normal commands always use Kernel. It preserves the exact source spelling.
func HistoricalNameKernel() dated.Kernel[Data, Patch] {
	return dated.Kernel[Data, Patch]{Policy: Policy{historicalName: true}}
}

type Record = dated.Record[Data]
type Working = dated.WorkingCandidate[Data, Patch]
type Candidate = dated.Candidate[Data]
type Item = dated.Item[Patch]

func Fault(code, field, message string) error {
	status := 409
	if code == "M02_INVALID_INPUT" || code == "M02_FORBIDDEN_INPUT" {
		status = 400
	}
	return &foundation.Fault{Status: status, Code: code, Field: field, Message: message}
}
func (Policy) Clone(d Data) Data {
	if d.ShortName != nil {
		v := *d.ShortName
		d.ShortName = &v
	}
	return d
}
func (Policy) EmptyPatch(p Patch) bool          { return len(p) == 0 }
func (p Policy) Normalize(d Data) (Data, error) { return p.normalize(d, true, true) }

func (p Policy) normalize(d Data, nameInput, shortInput bool) (Data, error) {
	if nameInput {
		name, err := org.NormalizeName(d.Name, true)
		if p.historicalName {
			name, err = org.HistoricalName(d.Name)
		}
		if err != nil {
			err.(*foundation.Fault).Field = "name"
			return d, err
		}
		d.Name = name
	}
	if shortInput && d.ShortName != nil {
		v, e := org.NormalizeName(*d.ShortName, false)
		if e != nil {
			e.(*foundation.Fault).Field = "shortName"
			return d, e
		}
		d.ShortName = nil
		if v != "" {
			d.ShortName = &v
		}
	}
	if !foundation.ValidID(d.BU) {
		return d, Fault("M02_INVALID_INPUT", "managementBuId", "请选择管理业务单元")
	}
	if d.Parent != "" && !foundation.ValidID(d.Parent) {
		return d, Fault("M02_INVALID_INPUT", "parentDepartmentId", "上级部门标识无效")
	}
	return d, nil
}
func (p Policy) Apply(d Data, patch *Patch) (Data, error) {
	if patch == nil || len(*patch) == 0 {
		return d, Fault("M02_INVALID_INPUT", "changes", "没有修改内容")
	}
	keys := make([]string, 0, len(*patch))
	for k := range *patch {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, key := range keys {
		v := (*patch)[key]
		s := ""
		if v != nil {
			s = *v
		}
		switch key {
		case "name":
			if v == nil {
				return d, Fault("M02_INVALID_INPUT", "changes.name", "名称不能为空")
			}
			d.Name = s
		case "shortName":
			d.ShortName = v
		case "managementBuId":
			if v == nil {
				return d, Fault("M02_INVALID_INPUT", "changes.managementBuId", "管理业务单元不能为空")
			}
			d.BU = s
		case "parentDepartmentId":
			d.Parent = s
		default:
			return d, Fault("M02_FORBIDDEN_INPUT", "changes", "字段不允许修改")
		}
	}
	// Independent stored fields are not resubmitted by an unrelated patch.
	// In particular, an imported historical spelling must remain untouched.
	_, suppliedName := (*patch)["name"]
	_, suppliedShort := (*patch)["shortName"]
	return p.normalize(p.Clone(d), suppliedName, suppliedShort)
}
func (Policy) EqualData(a, b Data) bool {
	return a.Name == b.Name && reflect.DeepEqual(a.ShortName, b.ShortName) && a.BU == b.BU && a.Parent == b.Parent
}
func (Policy) EqualRevision(a, b Data) bool {
	return a.Member.DepartmentID == b.Member.DepartmentID && a.Member.Source == b.Member.Source && a.Member.MemberID == b.Member.MemberID && a.Member.BU == b.Member.BU
}
func (Policy) OperationReasons(t string) map[string]string {
	return dated.OperationReasons(dated.Department, t)
}
func (Policy) OperationLabel(t string) string { return dated.OperationLabel(t) }

type Edge struct {
	ID, Revision, Source   string
	RevisionNo             int64
	Start, End, BU, Parent string
	Ref                    MemberRef
}
type Timeline struct {
	ID, Code, Enterprise, First, Source string
	SourceNo                            int64
	RecordedAt                          string
	Records                             []Record
	RelationSource                      string
	RelationNo                          int64
	Edges                               []Edge
}
type BURecord struct{ ID, Revision, Date, End, State, Name string }
type BusinessUnit struct {
	ID, Code, Source, First string
	Records                 []BURecord
}
type Snapshot struct {
	Departments     []Timeline
	BusinessUnits   []BusinessUnit
	departmentIndex map[string]int
	buIndex         map[string]int
	children        map[string][]string
	// A reference probe changes one timeline without copying the frozen forest.
	replacement *Timeline
}

// Indexed builds request-owned identity lookups; the facts remain immutable.
func (s Snapshot) Indexed() Snapshot {
	s = s.ReadIndexed()
	s.children = map[string][]string{}
	for i := range s.Departments {
		t := &s.Departments[i]
		seen := map[string]bool{}
		for j := range t.Records {
			parent := t.Records[j].Data.Parent
			if parent != "" && !seen[parent] {
				s.children[parent] = append(s.children[parent], t.ID)
				seen[parent] = true
			}
		}
	}
	return s
}

func (s Snapshot) ReadIndexed() Snapshot {
	if len(s.departmentIndex) != len(s.Departments) || s.departmentIndex == nil {
		s.departmentIndex = make(map[string]int, len(s.Departments))
		for i := range s.Departments {
			s.departmentIndex[s.Departments[i].ID] = i
		}
	}
	if len(s.buIndex) != len(s.BusinessUnits) || s.buIndex == nil {
		s.buIndex = make(map[string]int, len(s.BusinessUnits))
		for i := range s.BusinessUnits {
			s.buIndex[s.BusinessUnits[i].ID] = i
		}
	}
	return s
}

func (s Snapshot) Department(id string) *Timeline {
	if s.replacement != nil && s.replacement.ID == id {
		return s.replacement
	}
	if s.departmentIndex != nil {
		if i, ok := s.departmentIndex[id]; ok {
			return &s.Departments[i]
		}
		return nil
	}
	for i := range s.Departments {
		if s.Departments[i].ID == id {
			return &s.Departments[i]
		}
	}
	return nil
}

// DepartmentPosition lets read projections share the immutable identity index
// instead of constructing a second identity map in every request.
func (s Snapshot) DepartmentPosition(id string) (int, bool) {
	if s.departmentIndex != nil {
		i, ok := s.departmentIndex[id]
		return i, ok
	}
	for i := range s.Departments {
		if s.Departments[i].ID == id {
			return i, true
		}
	}
	return 0, false
}
func (s Snapshot) BU(id string) *BusinessUnit {
	if s.buIndex != nil {
		if i, ok := s.buIndex[id]; ok {
			return &s.BusinessUnits[i]
		}
		return nil
	}
	for i := range s.BusinessUnits {
		if s.BusinessUnits[i].ID == id {
			return &s.BusinessUnits[i]
		}
	}
	return nil
}
func At(rows []Record, date string) *Record { return Kernel.FactAt(rows, date) }
func recordAt(rows []Record, date string) *Record {
	i := dated.IndexAt(rows, date, func(r *Record) string { return r.Date })
	if i < 0 {
		return nil
	}
	return &rows[i]
}
func BUAt(rows []BURecord, date string) *BURecord {
	i := dated.IndexAt(rows, date, func(r *BURecord) string { return r.Date })
	if i < 0 {
		return nil
	}
	r := rows[i]
	return &r
}

// ScopeRecord borrows the selected immutable source member. Callers must not
// mutate it; candidate editing continues to use the detached Kernel.FactAt.
func ScopeRecord(t Timeline, date string) *Record {
	i := dated.IndexAt(t.Records, date, func(r *Record) string { return r.Date })
	if i < 0 && len(t.Records) > 0 {
		i = 0
	}
	if i < 0 {
		return nil
	}
	return &t.Records[i]
}
func Status(t Timeline, date string) string {
	i := dated.IndexAt(t.Records, date, func(r *Record) string { return r.Date })
	if i < 0 {
		return "not_effective"
	}
	return t.Records[i].State
}
func (s Snapshot) With(t Timeline) Snapshot {
	s.Departments = slices.Clone(s.Departments)
	s.children = nil
	for i := range s.Departments {
		if s.Departments[i].ID == t.ID {
			s.Departments[i] = t
			return s
		}
	}
	s.Departments = append(s.Departments, t)
	return s.Indexed()
}

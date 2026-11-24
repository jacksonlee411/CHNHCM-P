package domain

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"slices"
	"strings"
)

// ReferenceAdoption is evidence belonging to the consuming record. Qualification
// against a current source timeline never changes this evidence implicitly.
type ReferenceAdoption struct {
	BusinessAt    string `json:"businessAt"`
	SourceVersion string `json:"sourceVersion"`
	RecordID      string `json:"recordId"`
	Revision      string `json:"revision"`
}

type GradeMember struct {
	GradeID string            `json:"gradeId"`
	Adopted ReferenceAdoption `json:"adopted"`
}

type GradeMembers []GradeMember

func (m GradeMembers) Clone() GradeMembers { return append(GradeMembers{}, m...) }
func (m GradeMembers) IDs() []string {
	out := make([]string, len(m))
	for i := range m {
		out[i] = m[i].GradeID
	}
	return out
}
func (m GradeMembers) EqualData(other GradeMembers) bool {
	return slices.EqualFunc(m, other, func(a, b GradeMember) bool { return a.GradeID == b.GradeID })
}
func (m GradeMembers) EqualRevision(other GradeMembers) bool { return slices.Equal(m, other) }

// Present distinguishes an omitted collection from an explicit empty array.
// A present nil slice is JSON null and is never a valid collection replacement.
type MembershipPatch struct {
	Present bool
	IDs     []string
}

func NormalizeGradeID(raw string) (string, bool) {
	id := strings.ToLower(raw)
	return id, foundation.ValidID(id)
}

type GradeLadderData struct {
	GradeData
	Members GradeMembers
}
type GradeLadderPatch struct {
	GradePatch
	Members MembershipPatch
}

func (p GradeLadderPatch) Apply(base GradeLadderData) (GradeLadderData, error) {
	data, err := p.GradePatch.Apply(base.GradeData)
	if err != nil {
		return base, err
	}
	members, err := p.Members.Apply(Ladder, base.Members)
	return GradeLadderData{GradeData: data, Members: members}, err
}

func (p MembershipPatch) Apply(k Kind, base GradeMembers) (GradeMembers, error) {
	return p.apply(k, base, false)
}
func (p MembershipPatch) apply(k Kind, base GradeMembers, allowEmpty bool) (GradeMembers, error) {
	if !HasGrades(k) {
		if p.Present {
			return nil, fieldInvalid("gradeIds", "此对象不接受职级成员")
		}
		return nil, nil
	}
	out := base.Clone()
	if p.Present {
		if p.IDs == nil {
			return nil, fieldInvalid("gradeIds", "职级成员不能为null")
		}
		out = GradeMembers{}
		seen := map[string]bool{}
		for _, raw := range p.IDs {
			id, valid := NormalizeGradeID(raw)
			if !valid || seen[id] {
				return nil, fieldInvalid("gradeIds", "请选择不重复的有效职级")
			}
			seen[id] = true
			member := GradeMember{GradeID: id}
			for _, old := range base {
				if old.GradeID == id {
					member = old
					break
				}
			}
			out = append(out, member)
		}
	}
	if k == Ladder && len(out) == 0 && !allowEmpty {
		return nil, fieldInvalid("gradeIds", "职级阶梯至少需要一个职级成员")
	}
	if k == Job {
		slices.SortFunc(out, func(a, b GradeMember) int { return strings.Compare(a.GradeID, b.GradeID) })
	}
	return out, nil
}

func HasGrades(k Kind) bool { return k == Job || k == Ladder }

// gradeMembersAt returns the set in force at a boundary, including an empty
// set before the first record. Callers compare both timelines' boundaries.
func gradeMembersAt(t Timeline, date string) GradeMembers {
	if i := At(t, date); i >= 0 {
		return t.Records[i].Grades
	}
	return nil
}
func GradeIDs(t Timeline) []string {
	out := []string{}
	for _, r := range t.Records {
		out = append(out, r.Grades.IDs()...)
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// applyMembership completes the already validated final dated topology. Each
// independent successor keeps its own value; only stops inherit predecessors.
func applyMembership(p *Plan, c Command, referenceOnly bool) error {
	if !HasGrades(p.After.Kind) {
		_, err := c.Grades.Apply(p.After.Kind, nil)
		return err
	}
	for i := range p.After.Records {
		r := &p.After.Records[i]
		oldIndex := Index(p.Before, r.ID)
		if !r.Active() {
			if !referenceOnly && c.Grades.Present && r.ID == p.Selected && c.Action != "DELETE" && c.Action != "DELETE_LIFECYCLE_PAIR" {
				return fieldInvalid("gradeIds", "停用记录继承前项职级成员")
			}
			r.Grades = p.After.Records[i-1].Grades.Clone()
			continue
		}
		base := r.Grades
		if oldIndex < 0 && i > 0 {
			base = p.After.Records[i-1].Grades
		}
		patch := MembershipPatch{}
		if r.ID == p.Selected {
			patch = c.Grades
		}
		var members GradeMembers
		var err error
		if p.After.Kind == Ladder && !referenceOnly {
			// The dated topology carries the already patched scalar value;
			// the Ladder value applies its typed membership as one full value.
			value, e := (GradeLadderPatch{Members: patch}).Apply(GradeLadderData{GradeData: GradeData{Name: r.Data["name"], NameEn: r.Data["nameEn"], Description: r.Data["description"]}, Members: base})
			members, err = value.Members, e
		} else {
			members, err = patch.apply(p.After.Kind, base, referenceOnly)
		}
		if err != nil {
			return err
		}
		readopt := oldIndex < 0 && r.Type != "CHANGE"
		if oldIndex >= 0 {
			old := p.Before.Records[oldIndex]
			readopt = old.Date != r.Date || old.Type != r.Type
		}
		if readopt {
			for j := range members {
				members[j].Adopted = ReferenceAdoption{}
			}
		}
		r.Grades = members
	}
	return nil
}

func ValidateGradeCoverage(consumer Timeline, grades map[string]Timeline) error {
	for _, r := range consumer.Records {
		if !r.Active() {
			continue
		}
		for _, member := range r.Grades {
			g := grades[member.GradeID]
			if g.ID == "" || g.Enterprise != consumer.Enterprise || !Covers(g, r.Date, r.End) {
				return &foundation.Fault{Status: 409, Code: "M03_REFERENCE_CONFLICT", Message: "职级不能覆盖资料的完整启用期间，请先处理相关期间", Field: "gradeIds"}
			}
		}
	}
	return nil
}

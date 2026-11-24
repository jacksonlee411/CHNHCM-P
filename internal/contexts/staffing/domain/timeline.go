// Package domain owns Staffing's dated decisions. It does not use Organization's
// state machine: a Job's classification evidence is part of its own decision.
package domain

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"errors"
	"maps"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Kind string

const (
	Family      Kind = "jobfamily"
	JobFunction Kind = "jobfunction"
	Job         Kind = "job"
	Grade       Kind = "grade"
	Ladder      Kind = "gradeladder"
	Position    Kind = "position"
)

func (k Kind) Valid() bool {
	return k == JobFunction || k == Family || k == Job || k == Grade || k == Ladder || k == Position
}
func Fields(k Kind) []string {
	v := []string{"code", "name", "nameEn", "status", "effectiveDate", "operationType", "operationReason"}
	if k == Position {
		return append(v, PositionFields()...)
	}
	if k == Ladder {
		v = append(v, "gradeIds")
	}
	if k == JobFunction || k == Family || k == Grade || k == Ladder {
		return append(v, "description")
	}
	return append(v, "jobFunctionId", "jobFamilyId", "responsibilities", "requirements", "gradeIds")
}

type Adoption struct {
	FamilyID string `json:"familyId"`
	Date     string `json:"date"`
	Source   string `json:"source"`
	RecordID string `json:"recordId"`
	Revision string `json:"revision"`
}

// FunctionEvidence identifies either an adopted JobFunction record or an
// immutable pre-cutover Job record (empty FunctionID and BusinessAt).
type FunctionEvidence struct {
	FunctionID string `json:"functionId,omitempty"`
	BusinessAt string `json:"businessAt,omitempty"`
	Source     string `json:"sourceVersion"`
	RecordID   string `json:"recordId"`
	Revision   string `json:"revision"`
}
type Audit struct {
	SourceVersion string `json:"sourceVersion"`
	Action        string `json:"action"`
	RecordedAt    string `json:"recordedAt"`
	SubjectID     string `json:"subjectId"`
	DisplayName   string `json:"displayName"`
	Login         string `json:"login"`
	IdentityKind  string `json:"identityKind"`
}
type Record struct {
	Function     FunctionEvidence            `json:"functionEvidence,omitzero"`
	PositionRefs map[string]PositionAdoption `json:"positionRefs,omitempty"`
	Grades       GradeMembers                `json:"-"`
	Summary      Audit                       `json:"summary"`
	ID           string                      `json:"id"`
	Revision     string                      `json:"revision"`
	Date         string                      `json:"date"`
	End          string                      `json:"end"`
	Type         string                      `json:"type"`
	Reason       string                      `json:"reason"`
	ReasonLabel  string                      `json:"reasonLabel"`
	Data         map[string]string           `json:"data"`
	Adoption     Adoption                    `json:"adoption"`
}

func (r Record) Active() bool { return r.Type != "DEACTIVATE" }

type Timeline struct {
	Kind       Kind     `json:"kind"`
	ID         string   `json:"id"`
	Enterprise string   `json:"enterprise"`
	Code       string   `json:"code"`
	Source     string   `json:"source"`
	Number     int64    `json:"number"`
	Records    []Record `json:"records"`
}
type Classification struct {
	Type   string
	Reason string
}
type Command struct {
	Action, ID, RecordID, ExpectedSource, ExpectedRevision, AnchorID                           string
	Code, Date, NewDate, Type, Reason                                                          string
	Grades                                                                                     MembershipPatch
	Changes                                                                                    map[string]*string
	Classification                                                                             *Classification
	StopID, StopRevision, StartID, StartRevision, StopDate, StartDate, StopReason, StartReason string
}
type Plan struct {
	Before, After  Timeline
	Adopt          []string
	AdoptFunctions []string
	Fields         []string
	Selected       string
	Changed        bool
}

func invalid(message string) error { return foundation.Fail(400, "M03_INVALID_INPUT", message) }
func fieldInvalid(field, message string) error {
	return &foundation.Fault{Status: 400, Code: "M03_INVALID_INPUT", Message: message, Field: field}
}
func conflict(message string) error { return foundation.Fail(409, "M03_CONFLICT", message) }
func Reasons(k Kind, op string) map[string]string {
	if k == Position {
		return map[string]map[string]string{
			"CREATE": {"NEW_POSITION": "新设岗位"}, "CHANGE": {"NAME_ADJUSTMENT": "名称调整", "POSITION_REASSIGNMENT": "岗位归属调整", "OTHER_DATA_ADJUSTMENT": "其他资料调整"},
			"DEACTIVATE": {"POSITION_CLOSURE": "岗位停用"}, "REACTIVATE": {"POSITION_RESUMPTION": "岗位恢复"},
		}[op]
	}
	switch op {
	case "CREATE":
		if k == Ladder {
			return map[string]string{"NEW_GRADE_LADDER": "新设职级阶梯"}
		}
		if k == Grade {
			return map[string]string{"NEW_GRADE": "新设职级"}
		}
		if k == Family {
			return map[string]string{"NEW_JOB_FAMILY": "新设职务系列"}
		}
		if k == JobFunction {
			return map[string]string{"NEW_JOB_FUNCTION": "新设职类"}
		}
		return map[string]string{"NEW_JOB": "新设职务"}
	case "CHANGE":
		v := map[string]string{"NAME_ADJUSTMENT": "名称调整", "OTHER_DATA_ADJUSTMENT": "其他数据调整"}
		if k == JobFunction || k == Family || k == Grade || k == Ladder {
			v["DESCRIPTION_ADJUSTMENT"] = "说明调整"
		} else {
			v["RESPONSIBILITY_ADJUSTMENT"] = "职责调整"
			v["REQUIREMENT_ADJUSTMENT"] = "任职要求调整"
			v["FUNCTION_REASSIGNMENT"] = "职类调整"
			v["FAMILY_REASSIGNMENT"] = "职务系列调整"
			v["GRADE_SCOPE_ADJUSTMENT"] = "可用职级调整"
		}
		if k == Ladder {
			v["GRADE_MEMBERSHIP_ADJUSTMENT"] = "职级成员调整"
			v["GRADE_ORDER_ADJUSTMENT"] = "职级顺序调整"
		}
		return v
	case "DEACTIVATE":
		if k == Ladder {
			return map[string]string{"GRADE_LADDER_CLOSURE": "职级阶梯停用"}
		}
		if k == Grade {
			return map[string]string{"GRADE_CLOSURE": "职级停用"}
		}
		if k == Family {
			return map[string]string{"FAMILY_CLOSURE": "职务系列停用"}
		}
		if k == JobFunction {
			return map[string]string{"FUNCTION_CLOSURE": "职类停用"}
		}
		return map[string]string{"JOB_CLOSURE": "职务停用"}
	case "REACTIVATE":
		if k == Ladder {
			return map[string]string{"GRADE_LADDER_RESUMPTION": "职级阶梯恢复"}
		}
		if k == Grade {
			return map[string]string{"GRADE_RESUMPTION": "职级恢复"}
		}
		if k == Family {
			return map[string]string{"FAMILY_RESUMPTION": "职务系列恢复"}
		}
		if k == JobFunction {
			return map[string]string{"FUNCTION_RESUMPTION": "职类恢复"}
		}
		return map[string]string{"JOB_RESUMPTION": "职务恢复"}
	}
	return map[string]string{}
}
func NormalizeCode(s string) (string, error) {
	s = strings.TrimSpace(s)
	if len(s) < 1 || len(s) > 30 {
		return "", fieldInvalid("code", "编码应为1至30个字符")
	}
	for i, c := range s {
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || i > 0 && (c == '_' || c == '-')) {
			return "", fieldInvalid("code", "编码请使用字母、数字、下划线或短横线，并以字母或数字开头")
		}
	}
	return strings.ToUpper(s), nil
}
func Text(s string, max int) (string, error) {
	if !utf8.ValidString(s) {
		return "", invalid("文本不是有效的UTF-8")
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	for _, r := range s {
		if (unicode.IsControl(r) && r != '\n') || r == '\u2028' || r == '\u2029' {
			return "", invalid("文本不能包含控制字符")
		}
	}
	s = foundation.NFC(strings.TrimSpace(s))
	if utf8.RuneCountInString(s) > max {
		return "", invalid("文本超过允许长度")
	}
	return s, nil
}
func patch(k Kind, base map[string]string, changes map[string]*string) (map[string]string, error) {
	if k == Grade || k == Ladder {
		return applyGrade(base, changes)
	}
	v := maps.Clone(base)
	if v == nil {
		v = map[string]string{}
	}
	for key, p := range changes {
		raw := ""
		if p != nil {
			raw = *p
		}
		var err error
		switch key {
		case "name", "nameEn":
			raw, err = foundation.NormalizeName(raw, key == "name")
		case "description":
			if k != Family && k != JobFunction {
				return nil, invalid("字段不适用于职务")
			}
			raw, err = Text(raw, 1000)
		case "responsibilities", "requirements":
			if k != Job {
				return nil, invalid("字段不适用于此对象")
			}
			raw, err = Text(raw, 2000)
		case "jobFunctionId":
			raw = strings.ToLower(raw)
			if k != Job || !foundation.ValidID(raw) {
				return nil, fieldInvalid(key, "请选择有效职类")
			}
		case "jobFamilyId":
			if k != Job || !foundation.ValidID(raw) {
				return nil, invalid("请选择有效职务系列")
			}
		case "departmentId", "jobId", "locationId", "gradeId":
			if k != Position {
				return nil, invalid("字段不适用于此对象")
			}
			raw = strings.ToLower(raw)
			if !foundation.ValidID(raw) && !(key == "gradeId" && raw == "") {
				return nil, fieldInvalid(key, "请选择有效的"+PositionLabel(key))
			}
			if raw == "" {
				delete(v, key)
				continue
			}
		default:
			return nil, invalid("未知或不可更改字段")
		}
		if err != nil {
			message := err.Error()
			var fault *foundation.Fault
			if errors.As(err, &fault) {
				message = fault.Message
			}
			return nil, &foundation.Fault{Status: 400, Code: "M03_INVALID_INPUT", Message: message, Field: key}
		}
		v[key] = raw
	}
	if v["name"] == "" {
		return nil, fieldInvalid("name", "请输入名称")
	}
	if k == Job && !foundation.ValidID(v["jobFamilyId"]) {
		return nil, invalid("请选择职务系列")
	}
	if k == Position {
		for _, field := range PositionFields()[:3] {
			if v[field] == "" {
				return nil, fieldInvalid(field, "请选择"+PositionLabel(field))
			}
		}
	}
	return v, nil
}
func clone(t Timeline) Timeline {
	t.Records = slices.Clone(t.Records)
	for i := range t.Records {
		t.Records[i].Data = maps.Clone(t.Records[i].Data)
		t.Records[i].PositionRefs = maps.Clone(t.Records[i].PositionRefs)
		t.Records[i].Grades = t.Records[i].Grades.Clone()
	}
	return t
}
func Index(t Timeline, id string) int {
	return slices.IndexFunc(t.Records, func(r Record) bool { return r.ID == id })
}
func At(t Timeline, date string) int {
	return slices.IndexFunc(t.Records, func(r Record) bool { return r.Date <= date && (r.End == "" || date < r.End) })
}
func pair(t Timeline, stop, start string) (int, error) {
	i := Index(t, stop)
	if i <= 0 || i+1 >= len(t.Records) || t.Records[i].Type != "DEACTIVATE" || t.Records[i+1].ID != start || t.Records[i+1].Type != "REACTIVATE" {
		return -1, conflict("停用恢复组合已变化，请重新查询")
	}
	return i, nil
}
func newRecord(k Kind, date, typ, reason string, data map[string]string) (Record, error) {
	label := Reasons(k, typ)[reason]
	if label == "" {
		return Record{}, fieldInvalid("operationReason", "请选择与操作类型对应的业务原因")
	}
	return Record{ID: foundation.NewID(), Revision: foundation.NewID(), Date: date, Type: typ, Reason: reason, ReasonLabel: label, Data: maps.Clone(data)}, nil
}

// Build computes the whole resulting head. Immutable history is not rewritten.
// Qualification/adoption are completed under guards by the application layer.
func Build(k Kind, enterprise string, before Timeline, c Command) (Plan, error) {
	return build(k, enterprise, before, c, false)
}

// ReferencePlan validates the dated topology and typed collection without
// requiring scalar form fields or a reason the user has not entered yet.
// It is read-only; only Build may produce a persistable business decision.
func ReferencePlan(k Kind, enterprise string, before Timeline, c Command) (Plan, error) {
	return build(k, enterprise, before, c, true)
}
func build(k Kind, enterprise string, before Timeline, c Command, referenceOnly bool) (Plan, error) {
	applyData := patch
	makeRecord := newRecord
	if referenceOnly {
		applyData = func(kind Kind, base map[string]string, changes map[string]*string) (map[string]string, error) {
			out := maps.Clone(base)
			if out == nil {
				out = map[string]string{}
			}
			for key, v := range changes {
				if kind == Position && slices.Contains(PositionFields(), key) {
					if v == nil || *v == "" {
						delete(out, key)
					} else {
						out[key] = strings.ToLower(*v)
					}
				} else if v != nil {
					out[key] = *v
				}
			}
			return out, nil
		}
		makeRecord = func(_ Kind, date, typ, reason string, data map[string]string) (Record, error) {
			return Record{ID: foundation.NewID(), Revision: foundation.NewID(), Date: date, Type: typ, Reason: reason, Data: maps.Clone(data)}, nil
		}
	}

	p := Plan{Before: before, After: clone(before)}
	t := &p.After
	if !k.Valid() {
		return p, invalid("对象类型无效")
	}
	if c.Action == "CREATE" {
		if before.ID != "" {
			return p, conflict("对象已存在")
		}
		code := c.Code
		var e error
		if !referenceOnly {
			code, e = NormalizeCode(c.Code)
		}
		if e != nil {
			return p, e
		}
		data, e := applyData(k, nil, c.Changes)
		if e != nil {
			return p, e
		}
		date := c.Date
		if date == "" {
			date = foundation.DefaultEffectiveDate
		}
		if c.Type != "CREATE" {
			return p, invalid("新建操作类型无效")
		}
		r, e := makeRecord(k, date, "CREATE", c.Reason, data)
		if e != nil {
			return p, e
		}
		*t = Timeline{Kind: k, ID: foundation.NewID(), Enterprise: enterprise, Code: code, Records: []Record{r}}
		p.Selected = r.ID
		if k == Job {
			p.Adopt = append(p.Adopt, r.ID)
		}
	} else {
		if before.ID == "" {
			return p, foundation.Fail(404, "M03_OBJECT_NOT_FOUND", "资料不存在或当前不可见")
		}
		if c.ExpectedSource != before.Source {
			return p, conflict("资料已变化，请重新查询后重试")
		}
		index := Index(before, c.RecordID)
		anchor := Index(before, c.AnchorID)
		switch c.Action {
		case "ADD", "DEACTIVATE", "REACTIVATE":
			if anchor < 0 {
				return p, conflict("定位记录已变化")
			}
			prev := t.Records[anchor]
			typ := "CHANGE"
			if c.Action != "ADD" {
				typ = c.Action
				if anchor != len(t.Records)-1 {
					return p, invalid("启停操作应定位最后一条记录")
				}
			}
			if c.Type != typ {
				return p, invalid("操作类型与动作不一致")
			}
			if typ == "REACTIVATE" == prev.Active() {
				return p, invalid("当前状态不允许该操作")
			}
			if c.Date <= prev.Date || prev.End != "" && c.Date >= prev.End {
				return p, fieldInvalid("effectiveDate", "日期应在所选记录之后及下一记录之前")
			}
			data, e := applyData(k, prev.Data, c.Changes)
			if e != nil {
				return p, e
			}
			if typ == "DEACTIVATE" && len(c.Changes) > 0 {
				return p, invalid("停用继承前项资料")
			}
			r, e := makeRecord(k, c.Date, typ, c.Reason, data)
			if e != nil {
				return p, e
			}
			r.Adoption = prev.Adoption
			t.Records = slices.Insert(t.Records, anchor+1, r)
			p.Selected = r.ID
			if k == Job && typ != "DEACTIVATE" && (typ == "REACTIVATE" || data["jobFamilyId"] != prev.Data["jobFamilyId"]) {
				p.Adopt = append(p.Adopt, r.ID)
			}
		case "CORRECT", "REDATE", "DELETE":
			if index < 0 || c.ExpectedRevision != before.Records[index].Revision {
				return p, conflict("记录已变化，请重新查询")
			}
			old := before.Records[index]
			r := old
			r.Data = maps.Clone(old.Data)
			if c.Action == "DELETE" {
				if index == 0 {
					return p, invalid("首条记录不能删除")
				}
				t.Records = slices.Delete(t.Records, index, index+1)
				p.Selected = t.Records[index-1].ID
				break
			}
			if c.NewDate != "" {
				r.Date = c.NewDate
			}
			if c.Action == "REDATE" && c.NewDate == "" {
				return p, invalid("请提供新日期")
			}
			if (index > 0 && r.Date <= t.Records[index-1].Date) || (index+1 < len(t.Records) && r.Date >= t.Records[index+1].Date) {
				return p, fieldInvalid("effectiveDate", "记录日期只能在原位置内调整")
			}
			if c.Classification != nil {
				r.Type = c.Classification.Type
				r.Reason = c.Classification.Reason
				r.ReasonLabel = Reasons(k, r.Type)[r.Reason]
				if !referenceOnly && r.ReasonLabel == "" || (index == 0 && r.Type != "CREATE") || (index > 0 && r.Type == "CREATE") {
					return p, invalid("操作分类无效")
				}
			}
			data, e := applyData(k, r.Data, c.Changes)
			if e != nil {
				return p, e
			}
			r.Data = data
			if !r.Active() && len(c.Changes) > 0 {
				return p, invalid("停用记录继承前项资料")
			}
			t.Records[index] = r
			p.Selected = r.ID
			if k == Job && r.Active() && (r.Date != old.Date || r.Type != old.Type || r.Data["jobFamilyId"] != old.Data["jobFamilyId"]) {
				p.Adopt = append(p.Adopt, r.ID)
			}
		case "INSERT_LIFECYCLE_PAIR":
			if anchor < 0 || !t.Records[anchor].Active() {
				return p, conflict("组合定位记录无效")
			}
			prev := t.Records[anchor]
			if c.StopDate <= prev.Date || c.StartDate <= c.StopDate || (prev.End != "" && c.StartDate >= prev.End) {
				return p, invalid("组合日期应位于同一启用区间内")
			}
			stop, e := makeRecord(k, c.StopDate, "DEACTIVATE", c.StopReason, prev.Data)
			if e != nil {
				return p, e
			}
			data, e := applyData(k, prev.Data, c.Changes)
			if e != nil {
				return p, e
			}
			start, e := makeRecord(k, c.StartDate, "REACTIVATE", c.StartReason, data)
			if e != nil {
				return p, e
			}
			stop.Adoption = prev.Adoption
			t.Records = slices.Insert(t.Records, anchor+1, stop, start)
			p.Selected = start.ID
			if k == Job {
				p.Adopt = append(p.Adopt, start.ID)
			}
		case "MAINTAIN_LIFECYCLE_PAIR", "DELETE_LIFECYCLE_PAIR":
			i, e := pair(before, c.StopID, c.StartID)
			if e != nil {
				return p, e
			}
			if before.Records[i].Revision != c.StopRevision || before.Records[i+1].Revision != c.StartRevision {
				return p, conflict("组合记录已变化")
			}
			if c.Action == "DELETE_LIFECYCLE_PAIR" {
				t.Records = slices.Delete(t.Records, i, i+2)
				p.Selected = t.Records[i-1].ID
				break
			}
			stop, start := t.Records[i], t.Records[i+1]
			stop.Date = c.StopDate
			start.Date = c.StartDate
			if stop.Date <= t.Records[i-1].Date || start.Date <= stop.Date || (i+2 < len(t.Records) && start.Date >= t.Records[i+2].Date) {
				return p, invalid("组合日期只能在原位置内调整")
			}
			stop.Reason = c.StopReason
			start.Reason = c.StartReason
			stop.ReasonLabel = Reasons(k, "DEACTIVATE")[stop.Reason]
			start.ReasonLabel = Reasons(k, "REACTIVATE")[start.Reason]
			if !referenceOnly && (stop.ReasonLabel == "" || start.ReasonLabel == "") {
				return p, invalid("组合原因无效")
			}
			data, e := applyData(k, start.Data, c.Changes)
			if e != nil {
				return p, e
			}
			start.Data = data
			if k == Job && (start.Date != before.Records[i+1].Date || start.Data["jobFamilyId"] != before.Records[i+1].Data["jobFamilyId"]) {
				p.Adopt = append(p.Adopt, start.ID)
			}
			t.Records[i] = stop
			t.Records[i+1] = start
			p.Selected = start.ID
		default:
			return p, invalid("操作无效")
		}
	}
	// Validate transitions without silently converting independent successors.
	for i := range t.Records {
		r := &t.Records[i]
		if !foundation.ValidDate(r.Date) {
			return p, fieldInvalid("effectiveDate", "生效日期无效")
		}
		if i == 0 && r.Type != "CREATE" {
			return p, invalid("首条记录必须为新建")
		}
		if i > 0 {
			prev := t.Records[i-1]
			if r.Date <= prev.Date {
				return p, invalid("同日只能有一条记录")
			}
			if r.Type == "CREATE" || r.Type == "CHANGE" && !prev.Active() || r.Type == "DEACTIVATE" && !prev.Active() || r.Type == "REACTIVATE" && prev.Active() {
				return p, invalid("操作将破坏启停顺序，请使用组合维护")
			}
			if !r.Active() {
				r.Data = maps.Clone(prev.Data)
				r.PositionRefs = maps.Clone(prev.PositionRefs)
				r.Adoption = prev.Adoption
			}
		}
		r.End = ""
		if i+1 < len(t.Records) {
			r.End = t.Records[i+1].Date
		}
	}
	if k == Job && !referenceOnly {
		if err := planFunctions(&p); err != nil {
			return p, err
		}
	}
	if err := applyMembership(&p, c, referenceOnly); err != nil {
		return p, err
	}
	p.Fields = ChangedFields(before, *t, c)
	p.Changed = !Equal(before, *t)
	for i := range t.Records {
		r := &t.Records[i]
		j := Index(before, r.ID)
		if j >= 0 && !sameRecord(before.Records[j], *r) {
			r.Revision = foundation.NewID()
		}
	}
	if p.Changed {
		t.Source = foundation.NewID()
		t.Number = before.Number + 1
	}
	return p, nil
}
func sameRecord(a, b Record) bool {
	return a.ID == b.ID && a.Date == b.Date && a.End == b.End && a.Type == b.Type && a.Reason == b.Reason && a.ReasonLabel == b.ReasonLabel && a.Function == b.Function && a.Adoption == b.Adoption && maps.Equal(a.Data, b.Data) && maps.Equal(a.PositionRefs, b.PositionRefs) && a.Grades.EqualRevision(b.Grades)
}
func Equal(a, b Timeline) bool {
	if a.ID != b.ID || len(a.Records) != len(b.Records) {
		return false
	}
	for i, r := range a.Records {
		if !sameRecord(r, b.Records[i]) {
			return false
		}
	}
	return true
}
func ChangedFields(a, b Timeline, c Command) []string {
	set := map[string]bool{}
	if b.Kind == Job && functionChanged(a, b) {
		set["jobFunctionId"] = true
	}
	if b.Kind == Position && a.ID != "" {
		for _, use := range ChangedPositionUses(a, b) {
			set[use.Field] = true
		}
	}
	if c.Grades.Present {
		set["gradeIds"] = true
	}
	if HasGrades(b.Kind) && !c.Grades.Present {
		for _, old := range a.Records {
			if !old.Grades.EqualData(gradeMembersAt(b, old.Date)) {
				set["gradeIds"] = true
				break
			}
		}
	}
	for k := range c.Changes {
		set[k] = true
	}
	if c.Action == "CREATE" {
		set["code"] = true
	}
	for _, r := range b.Records {
		i := Index(a, r.ID)
		if HasGrades(b.Kind) && !set["gradeIds"] && (i < 0 || r.Date != a.Records[i].Date) && !r.Grades.EqualData(gradeMembersAt(a, r.Date)) {
			set["gradeIds"] = true
		}
		if i < 0 {
			predecessor := map[string]string{}
			if i := Index(b, r.ID); i > 0 {
				predecessor = b.Records[i-1].Data
			}
			for k, v := range r.Data {
				if c.Action == "CREATE" || predecessor[k] != v {
					set[k] = true
				}
			}
			set["effectiveDate"] = true
			set["operationType"] = true
			set["operationReason"] = true
			continue
		}
		old := a.Records[i]
		if !old.Grades.EqualData(r.Grades) {
			set["gradeIds"] = true
		}
		for k, v := range r.Data {
			if old.Data[k] != v {
				set[k] = true
			}
		}
		if old.Date != r.Date {
			set["effectiveDate"] = true
		}
		if old.Type != r.Type {
			set["operationType"] = true
		}
		if old.Reason != r.Reason {
			set["operationReason"] = true
		}
	}
	out := []string{}
	for k := range set {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}
func FamilyIDs(t Timeline) []string {
	out := []string{}
	for _, r := range t.Records {
		if id := r.Data["jobFamilyId"]; id != "" {
			out = append(out, id)
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}
func Covers(f Timeline, start, end string) bool {
	cursor := start
	for _, r := range f.Records {
		if !r.Active() || r.Date > cursor || r.End != "" && r.End <= cursor {
			continue
		}
		if r.End == "" || end != "" && r.End >= end {
			return true
		}
		cursor = r.End
	}
	return false
}
func ValidateCoverage(job Timeline, families map[string]Timeline) error {
	for _, r := range job.Records {
		if !r.Active() {
			continue
		}
		f := families[r.Data["jobFamilyId"]]
		if f.ID == "" || f.Enterprise != job.Enterprise || !Covers(f, r.Date, r.End) {
			return &foundation.Fault{Status: 409, Code: "M03_REFERENCE_CONFLICT", Message: "职务系列不能覆盖职务的完整启用期间，请先处理相关期间", Field: "jobFamilyId"}
		}
	}
	return nil
}
func CoverageChanged(a, b Timeline) bool {
	for _, r := range a.Records {
		if r.Active() && !Covers(b, r.Date, r.End) {
			return true
		}
	}
	return false
}

// Finalize applies revisions after adoption resolution, including inherited stops.
func Finalize(p *Plan) {
	for i := range p.After.Records {
		r := &p.After.Records[i]
		j := Index(p.Before, r.ID)
		if j >= 0 && !sameRecord(p.Before.Records[j], *r) && r.Revision == p.Before.Records[j].Revision {
			r.Revision = foundation.NewID()
		}
	}
}

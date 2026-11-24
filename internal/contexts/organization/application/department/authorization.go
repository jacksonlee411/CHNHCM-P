package department

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

var ReadFields = []string{"code", "name", "shortName", "managementBuId", "parentDepartmentId", "departmentPath", "status", "effectiveDate", "operationType", "operationReason"}
var CreateFields = []string{"code", "name", "managementBuId", "effectiveDate", "operationType", "operationReason"}

func ptr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
func denied() error { return foundation.Fail(403, "ACCESS_DENIED", "当前权限不允许此操作") }
func missing() error {
	return foundation.Fail(404, "M02_OBJECT_NOT_FOUND", "部门不存在或当前不可见")
}
func missingReference() error {
	return foundation.Fail(404, "M02_OBJECT_NOT_FOUND", "引用对象不存在或当前不可见")
}
func genericRelation() error {
	return foundation.Fail(409, "M02_DEPARTMENT_REFERENCE_CONFLICT", "当前关系不满足保存条件")
}
func fact(t d.Timeline, now time.Time, createBU string) access.DepartmentTarget {
	return factOnDay(t, now, now.In(foundation.Shanghai).Format("2006-01-02"), createBU)
}

func PositionScope(t d.Timeline, now time.Time) org.PositionDepartmentScope {
	scope := fact(t, now, "").ScopeFact
	return org.PositionDepartmentScope{BusinessDate: scope.BusinessDate, Mode: scope.Mode, ManagementBUID: scope.ManagementBUID, RecordID: scope.RecordID, Revision: scope.Revision}
}
func factOnDay(t d.Timeline, now time.Time, day, createBU string) access.DepartmentTarget {
	f := access.ScopeFact{EvaluatedAt: now, BusinessDate: day, Mode: "create_candidate", ManagementBUID: createBU}
	id := t.ID
	if len(t.Records) > 0 {
		r := d.ScopeRecord(t, f.BusinessDate)
		f.Mode = "current_record"
		if f.BusinessDate < t.Records[0].Date {
			f.Mode = "future_first"
		}
		f.ManagementBUID = r.Data.BU
		f.CurrentSourceVersion = t.Source
		f.RecordID = r.ID
		f.Revision = r.Revision
	}
	return access.DepartmentTarget{EnterpriseID: t.Enterprise, DepartmentID: id, ScopeFact: f}
}
func intent(target access.DepartmentTarget, action, kind string, fields []string) access.Intent {
	if fields == nil {
		fields = []string{}
	}
	return access.Intent{DepartmentTarget: &target, EnterpriseID: target.EnterpriseID, Action: action, Kind: kind, Fields: fields}
}
func require(ctx context.Context, u Unit, in access.Intent) (access.Decision, error) {
	defer foundation.BeginStage(ctx, "department.authorize")()
	result, e := u.Access().Decide(ctx, in)
	if e != nil {
		return result, e
	}
	if !result.Allowed {
		return result, denied()
	}
	return result, nil
}
func readable(ctx context.Context, u Unit, t d.Timeline, now time.Time, history bool) ([]string, error) {
	if t.Source == "" {
		return nil, missing()
	}
	target := fact(t, now, "")
	r, e := require(ctx, u, intent(target, "department.read", "object", nil))
	if e != nil {
		var f *foundation.Fault
		if errors.As(e, &f) && f.Code == "ACCESS_DENIED" {
			return nil, missing()
		}
		return nil, e
	}
	fields := r.ReadFields
	if history {
		h, e := require(ctx, u, intent(target, "department.history.read", "history", nil))
		if e != nil {
			return nil, e
		}
		fields = intersection(fields, h.ReadFields)
	}
	return fields, nil
}
func intersection(a, b []string) []string {
	out := []string{}
	for _, v := range a {
		if slices.Contains(b, v) {
			out = append(out, v)
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}
func patchFields(p org.DepartmentChangesInput) []string {
	out := []string{}
	for f := range p {
		out = append(out, f)
	}
	slices.Sort(out)
	return out
}
func categoryAction(typ, suffix string) string {
	prefix := map[string]string{"CREATE": "first", "CHANGE": "record", "DEACTIVATE": "deactivation", "REACTIVATE": "reactivation"}[typ]
	if prefix == "" || typ == "CREATE" && suffix == "delete" {
		return ""
	}
	return "department." + prefix + "." + suffix
}
func requirements(cmd org.DepartmentCommand, t d.Timeline, target access.DepartmentTarget) access.Intent {
	checks := []access.Intent{}
	add := func(action string, fields []string) {
		if action != "" {
			checks = append(checks, intent(target, action, "command-requirement", fields))
		}
	}
	original := ""
	for _, r := range t.Records {
		if r.ID == cmd.RecordID {
			original = r.Type
		}
	}
	switch cmd.Action {
	case "CREATE":
		add("department.create", cmd.Fields)
		add("department.read", nil)
		add("department.history.read", nil)
	case "ADD":
		add("department.record.add", append(patchFields(cmd.Add.Changes), "effectiveDate", "operationType", "operationReason"))
	case "DEACTIVATE", "REACTIVATE":
		add("department."+strings.ToLower(cmd.Action), append(patchFields(cmd.Add.Changes), "effectiveDate", "operationType", "operationReason"))
	case "CORRECT":
		final := original
		if cmd.Correct.Classification != nil {
			final = cmd.Correct.Classification.OperationType
		}
		if cmd.Correct.Changes != nil {
			add("department.record.correct", patchFields(cmd.Correct.Changes))
		}
		if cmd.Correct.NewEffectiveDate != "" {
			for _, typ := range slices.Compact([]string{original, final}) {
				add(categoryAction(typ, "redate"), []string{"effectiveDate"})
			}
		}
		if cmd.Correct.Classification != nil {
			for _, typ := range slices.Compact([]string{original, final}) {
				add(categoryAction(typ, "reclassify"), []string{"operationType", "operationReason"})
			}
		}
	case "DELETE":
		add(categoryAction(original, "delete"), nil)
	case "REDATE":
		add(categoryAction(original, "redate"), []string{"effectiveDate"})
	case "INSERT_LIFECYCLE_PAIR":
		add("department.deactivate", []string{"effectiveDate", "operationType", "operationReason"})
		add("department.reactivate", []string{"effectiveDate", "operationType", "operationReason"})
	case "DELETE_LIFECYCLE_PAIR":
		add("department.deactivation.delete", nil)
		add("department.reactivation.delete", nil)
	}
	if len(checks) == 0 {
		add("department.record.correct", nil)
	}
	outer := checks[0]
	outer.Kind = "department-command"
	outer.Requirements = checks
	return outer
}
func authorizeDelta(ctx context.Context, u Unit, target access.DepartmentTarget, deltas []d.RelationDelta) error {
	for _, delta := range deltas {
		fields := []string{}
		if delta.BUChanged {
			fields = append(fields, "managementBuId")
		}
		if delta.ParentChanged {
			fields = append(fields, "parentDepartmentId")
		}
		seen := map[string]bool{}
		for _, bu := range []string{delta.OldBU, delta.NewBU} {
			if seen[bu] {
				continue
			}
			seen[bu] = true
			in := intent(target, "department.reassign", "affected-scope", fields)
			in.AffectedScope = &access.AffectedScope{ManagementBUID: bu}
			if _, e := require(ctx, u, in); e != nil {
				return e
			}
		}
	}
	return nil
}
func reference(ctx context.Context, u Unit, s d.Snapshot, enterprise, id, kind string, now time.Time) (access.Decision, error) {
	var in access.Intent
	exists := false
	if kind == "bu" {
		exists = s.BU(id) != nil
		in = access.Intent{Action: "bu.reference.read", EnterpriseID: enterprise, BUID: id, Kind: "reference", Fields: []string{}}
	} else {
		t := s.Department(id)
		if t == nil {
			return access.Decision{}, missingReference()
		}
		exists = true
		in = intent(fact(*t, now, ""), "department.reference.read", "reference", nil)
	}
	result, e := u.Access().Decide(ctx, in)
	if e != nil {
		return result, e
	}
	if !exists || !result.Allowed {
		return result, missingReference()
	}
	return result, nil
}
func explicitTargets(cmd org.DepartmentCommand, t d.Timeline) (bu, parent string, buDeclared, parentDeclared bool, base *d.Record) {
	patch := cmd.Add.Changes
	if cmd.Action == "CREATE" {
		bu, parent = cmd.Create.ManagementBUID, ""
		if cmd.Create.ParentDepartmentID != nil {
			parent = *cmd.Create.ParentDepartmentID
		}
		return bu, parent, true, parent != "", nil
	}
	if cmd.Action == "CORRECT" {
		patch = cmd.Correct.Changes
	}
	id := cmd.RecordID
	if id == "" {
		id = cmd.Add.AnchorRecordID
	}
	for _, r := range t.Records {
		if r.ID == id {
			v := r
			base = &v
			bu, parent = r.Data.BU, r.Data.Parent
			break
		}
	}
	if v, ok := patch["managementBuId"]; ok {
		buDeclared = true
		bu = ""
		if v != nil {
			bu = *v
		}
	}
	if v, ok := patch["parentDepartmentId"]; ok {
		parentDeclared = true
		parent = ""
		if v != nil {
			parent = *v
		}
	}
	return
}
func preauthorizeRelations(ctx context.Context, u Unit, s d.Snapshot, t d.Timeline, cmd org.DepartmentCommand, target access.DepartmentTarget, now time.Time) error {
	bu, parent, buDeclared, parentDeclared, base := explicitTargets(cmd, t)
	if base != nil && (buDeclared || parentDeclared) && (bu != base.Data.BU || parent != base.Data.Parent) {
		if e := authorizeDelta(ctx, u, target, []d.RelationDelta{{OldBU: base.Data.BU, NewBU: bu, BUChanged: base.Data.BU != bu, ParentChanged: base.Data.Parent != parent}}); e != nil {
			return e
		}
	}
	if buDeclared {
		if _, e := reference(ctx, u, s, t.Enterprise, bu, "bu", now); e != nil {
			return e
		}
	}
	if parentDeclared && parent != "" {
		if _, e := reference(ctx, u, s, t.Enterprise, parent, "department", now); e != nil {
			return e
		}
	}
	return nil
}
func authorizeIntroduced(ctx context.Context, u Unit, s d.Snapshot, t d.Timeline, rows []d.Record, target access.DepartmentTarget, now time.Time) error {
	deltas := d.Deltas(t.Records, rows)
	if e := authorizeDelta(ctx, u, target, deltas); e != nil {
		return e
	}
	// Only introduced/replaced relations require new reference rights. Keeping an
	// old, now-hidden ancestor never becomes an extra maintenance prerequisite.
	for _, delta := range deltas {
		r := d.At(rows, delta.Start)
		if r == nil {
			continue
		}
		if delta.BUChanged {
			if _, e := reference(ctx, u, s, t.Enterprise, r.Data.BU, "bu", now); e != nil {
				return e
			}
		}
		if (delta.BUChanged || delta.ParentChanged) && r.Data.Parent != "" {
			if _, e := reference(ctx, u, s, t.Enterprise, r.Data.Parent, "department", now); e != nil {
				return e
			}
		}
	}
	return nil
}
func safeQualification(ctx context.Context, u Unit, s d.Snapshot, enterprise, targetID string, now time.Time, err error) error {
	var q *d.QualificationError
	if !errors.As(err, &q) {
		return err
	}
	kind := "department"
	field := "parentDepartmentId"
	if q.Code == "M02_DEPARTMENT_BU_CONFLICT" {
		kind = "bu"
		field = "managementBuId"
	}
	for _, ref := range []struct{ id, kind string }{{q.DepartmentID, "department"}, {q.DependencyID, kind}} {
		if ref.kind == "department" && ref.id == targetID {
			continue
		}
		dec, e := reference(ctx, u, s, enterprise, ref.id, ref.kind, now)
		if e != nil {
			var f *foundation.Fault
			if errors.As(e, &f) && f.Status < 500 {
				return genericRelation()
			}
			return e
		}
		if !slices.Contains(dec.ReadFields, "name") {
			return genericRelation()
		}
	}
	message := map[string]string{"M02_DEPARTMENT_BU_CONFLICT": "管理业务单元不能覆盖部门的全部启用期间", "M02_DEPARTMENT_PARENT_INVALID": "上级部门须在全部启用期间属于同一业务单元并保持启用", "M02_DEPARTMENT_HIERARCHY_CYCLE": "所选上级会形成部门循环"}[q.Code]
	return &foundation.Fault{Status: 409, Code: q.Code, Field: field, Message: message}
}

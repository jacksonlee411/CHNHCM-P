package application

import (
	"bytes"
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	s "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"errors"
	"maps"
	"slices"
	"strings"
	"time"
)

func command(c s.Command) d.Command {
	in := c.Input
	changes := maps.Clone(in.Changes)
	if in.Action == "CREATE" {
		changes = map[string]*string{}
		for k, v := range map[string]*string{"name": in.Name, "nameEn": in.NameEn, "description": in.Description, "jobFamilyId": in.JobFamilyID, "responsibilities": in.Responsibilities, "requirements": in.Requirements, "departmentId": in.DepartmentID, "jobId": in.JobID, "locationId": in.LocationID, "gradeId": in.GradeID} {
			if v != nil {
				changes[k] = v
			}
		}
	}
	if in.Action == "CREATE" && (in.JobFunctionPresent || in.JobFunctionID != nil) {
		changes["jobFunctionId"] = in.JobFunctionID
	}
	v := d.Command{Action: in.Action, ID: c.ObjectID, RecordID: in.RecordID, ExpectedSource: in.ExpectedSourceVersion, ExpectedRevision: in.ExpectedRecordRevision, AnchorID: in.AnchorRecordID, Code: in.Code, Date: in.EffectiveDate, NewDate: in.NewEffectiveDate, Type: in.OperationType, Reason: in.OperationReason, Changes: changes, StopID: in.DeactivationRecordID, StopRevision: in.DeactivationRevision, StartID: in.ReactivationRecordID, StartRevision: in.ReactivationRevision, StopDate: in.DeactivationDate, StartDate: in.ReactivationDate, StopReason: in.DeactivationReason, StartReason: in.ReactivationReason}
	ids := in.ChangedGradeIDs
	if in.Action == "CREATE" {
		ids = in.GradeIDs
	}
	if ids != nil {
		v.Grades = d.MembershipPatch{Present: true, IDs: *ids}
	}
	if in.Classification != nil {
		v.Classification = &d.Classification{Type: in.Classification.OperationType, Reason: in.Classification.OperationReason}
	}
	return v
}
func action(a string) string {
	return map[string]string{"CREATE": "create", "ADD": "record.add", "CORRECT": "record.correct", "REDATE": "record.redate", "DELETE": "record.delete", "DEACTIVATE": "deactivate", "REACTIVATE": "reactivate", "INSERT_LIFECYCLE_PAIR": "lifecycle.pair", "MAINTAIN_LIFECYCLE_PAIR": "lifecycle.pair", "DELETE_LIFECYCLE_PAIR": "lifecycle.pair"}[a]
}
func decide(ctx context.Context, u Unit, k d.Kind, enterprise, purpose string, fields []string) (access.Decision, error) {
	dec, e := u.Access().Decide(ctx, access.Intent{Action: string(k) + "." + purpose, Kind: string(k), EnterpriseID: enterprise, Fields: fields})
	if e != nil {
		return dec, e
	}
	if !dec.Allowed {
		return dec, foundation.Fail(403, "M03_FORBIDDEN", "当前没有执行此操作的权限")
	}
	return dec, nil
}
func authorizeFields(dec access.Decision, fields []string) error {
	for _, field := range fields {
		if !slices.Contains(dec.WriteFields, field) {
			return foundation.Fail(403, "M03_FIELD_FORBIDDEN", "当前没有修改相关字段的权限")
		}
	}
	return nil
}
func prepare(ctx context.Context, u Unit, c s.Command, locked bool) (d.Plan, error) {
	k := d.Kind(c.Kind)
	cmd := command(c)
	var before d.Timeline
	var e error
	if !k.Valid() || action(cmd.Action) == "" {
		return d.Plan{}, foundation.Fail(400, "M03_INVALID_INPUT", "操作无效")
	}
	ent, e := u.Enterprise(ctx, locked)
	if e != nil {
		return d.Plan{}, e
	}
	if _, e = decide(ctx, u, k, ent.ID, action(cmd.Action), nil); e != nil {
		return d.Plan{}, e
	}
	if _, present := cmd.Changes["jobFunctionId"]; k == d.Job && present {
		dec, err := decide(ctx, u, k, ent.ID, action(cmd.Action), []string{"jobFunctionId"})
		if err != nil {
			return d.Plan{}, err
		}
		if err = authorizeFields(dec, []string{"jobFunctionId"}); err != nil {
			return d.Plan{}, err
		}
	}
	if cmd.Action != "CREATE" {
		if !foundation.ValidID(c.ObjectID) {
			return d.Plan{}, foundation.Fail(400, "M03_INVALID_INPUT", "资料标识无效")
		}
		if !locked {
			before, e = u.Load(ctx, k, c.ObjectID, "")
			if e != nil {
				return d.Plan{}, e
			}
		}
	}
	if locked {
		familyIDs := d.FamilyIDs(before)
		if id := cmd.Changes["jobFamilyId"]; id != nil && foundation.ValidID(*id) {
			familyIDs = append(familyIDs, *id)
		}
		lockID := c.ObjectID
		if lockID == "" {
			lockID = foundation.NewID()
		}
		gradeIDs := d.GradeIDs(before)
		if cmd.Grades.Present {
			for _, raw := range cmd.Grades.IDs {
				if id, valid := d.NormalizeGradeID(raw); valid {
					gradeIDs = append(gradeIDs, id)
				}
			}
		}
		var functions []string
		if id := cmd.Changes["jobFunctionId"]; id != nil && foundation.ValidID(*id) {
			functions = append(functions, strings.ToLower(*id))
		}
		var quotaDepartments []string
		if k == d.Position {
			def, err := u.Load(ctx, k, lockID, "")
			if err != nil {
				return d.Plan{}, err
			}
			quotaDepartments = d.QuotaEligibility(def).Departments()
			if id := cmd.Changes["departmentId"]; id != nil && foundation.ValidID(*id) {
				quotaDepartments = append(quotaDepartments, *id)
			}
			if err = u.LockQuotaDepartments(ctx, quotaDepartments); err != nil {
				return d.Plan{}, err
			}
		}
		before, e = u.LockObject(ctx, k, lockID, familyIDs, functions, gradeIDs)
		if e == nil && k == d.Position && !departmentsLocked(d.QuotaEligibility(before).Departments(), quotaDepartments) {
			return d.Plan{}, foundation.Fail(409, "M03_QUOTA_CONFLICT", "岗位归属已变化，请刷新后重试")
		}
		if e != nil {
			return d.Plan{}, e
		}
	}
	// Explicit collection authority precedes candidate diagnostics.
	if cmd.Grades.Present {
		dec, err := decide(ctx, u, k, ent.ID, action(cmd.Action), []string{"gradeIds"})
		if err != nil {
			return d.Plan{}, err
		}
		if err = authorizeFields(dec, []string{"gradeIds"}); err != nil {
			return d.Plan{}, err
		}
	}
	p, e := d.Build(k, ent.ID, before, cmd)
	if e != nil {
		return p, e
	}
	if p.After.Records[0].Date < ent.ValidFrom {
		return p, &foundation.Fault{Status: 409, Code: "M03_ENTERPRISE_DATE_CONFLICT", Message: "首条日期不能早于企业首条日期", Field: "effectiveDate"}
	}
	dec, e := decide(ctx, u, k, ent.ID, action(cmd.Action), p.Fields)
	if e != nil {
		return p, e
	}
	if e = authorizeFields(dec, p.Fields); e != nil {
		return p, e
	}
	if k == d.Position {
		read, err := decide(ctx, u, k, ent.ID, "read", nil)
		if err != nil {
			return p, err
		}
		for _, field := range d.PositionFields() {
			if has(p.Fields, field) && !has(read.ReadFields, field) {
				return p, foundation.Fail(403, "M03_FIELD_FORBIDDEN", "无权维护该岗位引用")
			}
		}
	}
	if k == d.Job {
		families := map[string]d.Timeline{}
		for _, id := range d.FamilyIDs(p.After) {
			f, e := u.Load(ctx, d.Family, id, "")
			if e != nil {
				return p, e
			}
			families[id] = f
		}
		if e = d.ValidateCoverage(p.After, families); e != nil {
			return p, e
		}
		for _, id := range p.Adopt {
			i := d.Index(p.After, id)
			r := &p.After.Records[i]
			family := families[r.Data["jobFamilyId"]]
			at := d.At(family, r.Date)
			if at < 0 {
				return p, foundation.Fail(409, "M03_REFERENCE_CONFLICT", "职务系列在采用日期不可用")
			}
			required := []string{"name", "effectiveDate", "status"}
			dec, e := decide(ctx, u, d.Family, ent.ID, "read", nil)
			if e != nil {
				return p, e
			}
			if !has(dec.ReadFields, required...) {
				return p, foundation.Fail(403, "M03_REFERENCE_FORBIDDEN", "无权采用该职务系列")
			}
			if r.Date != today(u) {
				hd, e := decide(ctx, u, d.Family, ent.ID, "history.read", nil)
				if e != nil {
					return p, e
				}
				if !has(hd.ReadFields, required...) {
					return p, foundation.Fail(403, "M03_REFERENCE_FORBIDDEN", "无权读取该日期的职务系列")
				}
			}
			fr := family.Records[at]
			r.Adoption = d.Adoption{FamilyID: family.ID, Date: r.Date, Source: family.Source, RecordID: fr.ID, Revision: fr.Revision}
		}
		// A changed predecessor adoption also propagates to inherited stop records.
		for i := range p.After.Records {
			if i > 0 && !p.After.Records[i].Active() {
				p.After.Records[i].Adoption = p.After.Records[i-1].Adoption
			}
		}
	} else if k == d.Family && d.CoverageChanged(before, p.After) {
		jobs, e := u.Dependents(ctx, d.Family, p.After.ID)
		if e != nil {
			return p, e
		}
		for _, job := range jobs {
			families := map[string]d.Timeline{p.After.ID: p.After}
			for _, id := range d.FamilyIDs(job) {
				if id != p.After.ID {
					f, e := u.Load(ctx, d.Family, id, "")
					if e != nil {
						return p, e
					}
					families[id] = f
				}
			}
			if e = d.ValidateCoverage(job, families); e != nil {
				return p, foundation.Fail(409, "M03_REFERENCE_CONFLICT", "现有职务的启用期间超出候选系列范围，请先处理相关职务")
			}
		}
	}
	if e = prepareFunctions(ctx, u, &p); e != nil {
		return p, e
	}
	if e = checkHierarchyUses(ctx, u, p); e != nil {
		return p, e
	}
	if e = preparePositions(ctx, u, &p, locked); e != nil {
		return p, e
	}
	if e = prepareGrades(ctx, u, &p, cmd); e != nil {
		return p, e
	}
	d.Finalize(&p)
	return p, nil
}
func has(fields []string, required ...string) bool {
	for _, f := range required {
		if !slices.Contains(fields, f) {
			return false
		}
	}
	return true
}
func today(u interface{ Now() time.Time }) string {
	return u.Now().In(foundation.Shanghai).Format("2006-01-02")
}
func (svc Service) Execute(ctx context.Context, c foundation.Credentials, cmd s.Command) (out s.Result, err error) {
	if !foundation.ValidID(cmd.SubmissionID) || len(cmd.Digest) != 32 {
		return out, foundation.Fail(400, "M03_INVALID_INPUT", "提交标识无效")
	}
	err = svc.Store.Within(ctx, c, true, func(u Unit) error {
		k := d.Kind(cmd.Kind)
		if !k.Valid() {
			return foundation.Fail(400, "M03_INVALID_INPUT", "对象类型无效")
		}
		if e := u.LockSubmission(ctx, k, cmd.SubmissionID); e != nil {
			return e
		}
		old, e := u.Terminal(ctx, k, cmd.SubmissionID)
		if e != nil {
			return e
		}
		if old.Exists {
			if old.Result.Outcome != "not_applied" && !bytes.Equal(old.Digest, cmd.Digest) {
				return foundation.Fail(409, "M03_SUBMISSION_MISMATCH", "原提交标识已用于不同内容")
			}
			out = old.Result
			out.Replayed = true
			return redactResult(ctx, u, k, &out)
		}
		out = s.Result{Kind: cmd.Kind, SubmissionID: cmd.SubmissionID, Action: cmd.Input.Action}
		e = u.Business(ctx, func() error {
			p, e := prepare(ctx, u, cmd, true)
			if e != nil {
				return e
			}
			out.ObjectID = p.After.ID
			out.SourceVersion = p.After.Source
			out.SelectedRecordID = p.Selected
			if i := d.Index(p.After, p.Selected); i >= 0 {
				out.SelectedRecordRevision = p.After.Records[i].Revision
			}
			out.Outcome = "no_change"
			if p.Changed {
				operator, e := u.Access().OperatorSnapshot(ctx)
				if e != nil {
					return e
				}
				for i := range p.After.Records {
					r := &p.After.Records[i]
					j := d.Index(p.Before, r.ID)
					if j < 0 || p.Before.Records[j].Revision != r.Revision {
						r.Summary = d.Audit{SourceVersion: p.After.Source, Action: cmd.Input.Action, RecordedAt: u.Now().Format(time.RFC3339Nano), SubjectID: operator.SubjectID, IdentityKind: operator.Kind}
						if operator.DisplayName != nil {
							r.Summary.DisplayName = *operator.DisplayName
						}
						if operator.Login != nil {
							r.Summary.Login = *operator.Login
						}
					}
				}
				if e = u.Save(ctx, p, cmd, operator); e != nil {
					return e
				}
				out.Outcome = "accepted"
			}
			return nil
		})
		if e != nil {
			var fault *foundation.Fault
			if !errors.As(e, &fault) || fault.Status >= 500 {
				return e
			}
			out = s.Result{Kind: cmd.Kind, SubmissionID: cmd.SubmissionID, Action: cmd.Input.Action, Outcome: "rejected", Failure: fault, FailureStatus: fault.Status}
		}
		if e = u.Close(ctx, k, cmd.SubmissionID, cmd.Digest, out); e != nil {
			return e
		}
		return redactResult(ctx, u, k, &out)
	})
	return
}
func redactResult(ctx context.Context, u Unit, k d.Kind, out *s.Result) error {
	out.Kind = s.Kind(k)
	if out.ObjectID == "" {
		return nil
	}
	ent, e := u.Enterprise(ctx, false)
	if e != nil {
		return e
	}
	if _, e = decide(ctx, u, k, ent.ID, "read", nil); e != nil {
		var f *foundation.Fault
		if !errors.As(e, &f) || f.Status != 403 {
			return e
		}
		out.ObjectID = ""
		out.SourceVersion = ""
		out.SelectedRecordID = ""
		out.SelectedRecordRevision = ""
	}
	return nil
}
func (svc Service) Resolve(ctx context.Context, c foundation.Credentials, k s.Kind, id string) (out s.Result, err error) {
	if !foundation.ValidID(id) || !d.Kind(k).Valid() && k != s.Hierarchy {
		return out, foundation.Fail(400, "M03_INVALID_INPUT", "提交标识无效")
	}
	err = svc.Store.Within(ctx, c, true, func(u Unit) error {
		if e := u.LockSubmission(ctx, d.Kind(k), id); e != nil {
			return e
		}
		old, e := u.Terminal(ctx, d.Kind(k), id)
		if e != nil {
			return e
		}
		if old.Exists {
			out = old.Result
			out.Replayed = true
			return redactResult(ctx, u, d.Kind(k), &out)
		}
		out = s.Result{Kind: k, SubmissionID: id, Outcome: "not_applied"}
		return u.Close(ctx, d.Kind(k), id, nil, out)
	})
	return
}
func (svc Service) Preview(ctx context.Context, c foundation.Credentials, cmd s.Command) (out s.View, err error) {
	err = svc.Store.Within(ctx, c, false, func(u Unit) error {
		p, e := prepare(ctx, u, cmd, false)
		if e != nil {
			return e
		}
		dec, e := decide(ctx, u, d.Kind(cmd.Kind), p.After.Enterprise, "read", nil)
		if e != nil {
			return e
		}
		before, e := detailView(ctx, u, p.Before, dec, "", false)
		if e != nil {
			return e
		}
		after, e := detailView(ctx, u, p.After, dec, p.Selected, false)
		out = s.View{"before": before, "after": after, "changed": p.Changed}
		return e
	})
	return
}

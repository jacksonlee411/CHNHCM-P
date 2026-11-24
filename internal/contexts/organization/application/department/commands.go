package department

import (
	"chnhcm.local/chnhcm/internal/contexts/organization/application/enterprisecheck"
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/application/submission"
	base "chnhcm.local/chnhcm/internal/contexts/organization/domain"
	"chnhcm.local/chnhcm/internal/contexts/organization/domain/dated"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

func commandMetadata(cmd org.DepartmentCommand) (Terminal, error) {
	t := Terminal{ID: cmd.SubmissionID, DepartmentID: cmd.DepartmentID, RecordID: cmd.RecordID, Action: cmd.Action, Digest: cmd.Digest, Fields: []string{}}
	switch cmd.Action {
	case "CREATE":
		if !foundation.ValidID(cmd.Create.ManagementBUID) || cmd.Create.ParentDepartmentID != nil && !foundation.ValidID(*cmd.Create.ParentDepartmentID) {
			return t, d.Fault("M02_INVALID_INPUT", "", "引用标识无效")
		}
		if cmd.Create.OperationType != "CREATE" {
			return t, d.Fault("M02_INVALID_INPUT", "operationType", "新建类型无效")
		}
		t.Fields = slices.Clone(CreateFields)
		if cmd.Create.ShortName != nil || slices.Contains(cmd.Fields, "shortName") {
			t.Fields = append(t.Fields, "shortName")
		}
		if cmd.Create.ParentDepartmentID != nil || slices.Contains(cmd.Fields, "parentDepartmentId") {
			t.Fields = append(t.Fields, "parentDepartmentId")
		}
	case "ADD", "DEACTIVATE", "REACTIVATE":
		t.ExpectedSource = cmd.Add.ExpectedSourceVersion
		t.Fields = append(patchFields(cmd.Add.Changes), "effectiveDate", "operationType", "operationReason")
		expected := cmd.Action
		if expected == "ADD" {
			expected = "CHANGE"
		}
		if cmd.Add.OperationType != expected || !foundation.ValidID(cmd.Add.AnchorRecordID) || cmd.Action == "DEACTIVATE" && cmd.Add.Changes != nil {
			return t, d.Fault("M02_INVALID_INPUT", "", "动作及位置输入无效")
		}
	case "CORRECT":
		t.ExpectedSource, t.Revision = cmd.Correct.ExpectedSourceVersion, cmd.Correct.ExpectedRecordRevision
		if cmd.Correct.Changes == nil && cmd.Correct.Classification == nil && cmd.Correct.NewEffectiveDate == "" {
			return t, d.Fault("M02_INVALID_INPUT", "", "请选择更正内容")
		}
		t.Fields = patchFields(cmd.Correct.Changes)
		if cmd.Correct.NewEffectiveDate != "" {
			t.Fields = append(t.Fields, "effectiveDate")
		}
		if cmd.Correct.Classification != nil {
			t.Fields = append(t.Fields, "operationType", "operationReason")
		}
	case "DELETE":
		t.ExpectedSource, t.Revision = cmd.Delete.ExpectedSourceVersion, cmd.Delete.ExpectedRecordRevision
	case "REDATE":
		t.ExpectedSource, t.Revision = cmd.Redate.ExpectedSourceVersion, cmd.Redate.ExpectedRecordRevision
		t.Fields = []string{"effectiveDate"}
	case "INSERT_LIFECYCLE_PAIR":
		t.Action = "REVISE_TIMELINE"
		t.ExpectedSource = cmd.InsertPair.ExpectedSourceVersion
		t.Fields = []string{"effectiveDate", "operationType", "operationReason"}
	case "DELETE_LIFECYCLE_PAIR":
		t.Action = "REVISE_TIMELINE"
		t.ExpectedSource = cmd.DeletePair.ExpectedSourceVersion
	default:
		return t, d.Fault("M02_INVALID_INPUT", "action", "动作无效")
	}
	if !foundation.ValidID(t.ID) || len(t.Digest) != 32 {
		return t, d.Fault("M02_INVALID_INPUT", "", "保存请求无效")
	}
	if cmd.Action != "CREATE" && (!foundation.ValidID(t.DepartmentID) || !foundation.ValidID(t.ExpectedSource)) {
		return t, d.Fault("M02_INVALID_INPUT", "", "部门或来源标识无效")
	}
	if slices.Contains([]string{"CORRECT", "DELETE", "REDATE"}, cmd.Action) && (!foundation.ValidID(t.RecordID) || !foundation.ValidID(t.Revision)) {
		return t, d.Fault("M02_INVALID_INPUT", "", "目标记录或修订标识无效")
	}
	for _, patch := range []org.DepartmentChangesInput{cmd.Add.Changes, cmd.Correct.Changes} {
		if patch != nil && len(patch) == 0 && cmd.Action != "ADD" {
			return t, d.Fault("M02_INVALID_INPUT", "changes", "没有修改内容")
		}
		for key, v := range patch {
			if !slices.Contains([]string{"name", "shortName", "managementBuId", "parentDepartmentId"}, key) {
				return t, d.Fault("M02_FORBIDDEN_INPUT", "changes", "字段不可修改")
			}
			if (key == "name" || key == "managementBuId") && v == nil {
				return t, d.Fault("M02_INVALID_INPUT", "changes."+key, "必填资料不能为空")
			}
			if (key == "managementBuId" || key == "parentDepartmentId") && v != nil && !foundation.ValidID(*v) {
				return t, d.Fault("M02_INVALID_INPUT", "changes."+key, "引用标识无效")
			}
		}
	}
	slices.Sort(t.Fields)
	t.Fields = slices.Compact(t.Fields)
	return t, nil
}

func localCandidate(ctx context.Context, cmd org.DepartmentCommand, t d.Timeline, enterpriseStart string, kernel dated.Kernel[d.Data, d.Patch]) (d.Working, []d.Item, error) {
	defer foundation.BeginStage(ctx, "department.candidate")()
	var work d.Working
	if cmd.Action == "CREATE" {
		parent := ""
		if cmd.Create.ParentDepartmentID != nil {
			parent = *cmd.Create.ParentDepartmentID
		}
		w, e := kernel.PrepareCreate(ctx, t.First, cmd.Create.EffectiveDate, enterpriseStart, cmd.Create.OperationReason, d.Data{Name: cmd.Create.Name, ShortName: cmd.Create.ShortName, BU: cmd.Create.ManagementBUID, Parent: parent})
		return w, nil, e
	}
	item := d.Item{ID: "candidate", RecordID: cmd.RecordID}
	patch := func(v org.DepartmentChangesInput) *d.Patch {
		if v == nil {
			return nil
		}
		p := d.Patch(v)
		return &p
	}
	var err error
	switch cmd.Action {
	case "ADD", "DEACTIVATE", "REACTIVATE":
		item.Action = "ADD"
		item.RecordID = foundation.NewID()
		item.Date = cmd.Add.EffectiveDate
		item.Type = cmd.Add.OperationType
		item.Reason = cmd.Add.OperationReason
		item.Changes = patch(cmd.Add.Changes)
		if cmd.Action == "ADD" {
			// Missing CHANGE fields mean copied defaults; the kernel still validates
			// the position and lifecycle before eliminating an unchanged addition.
			if item.Changes == nil {
				empty := d.Patch{}
				item.Changes = &empty
			}
			err = kernel.ValidateAdditionPosition(t.Records, cmd.Add.AnchorRecordID, enterpriseStart, item.Date)
		} else {
			err = kernel.ValidateLifecycleAppend(t.Records, cmd.Add.AnchorRecordID, cmd.Action, enterpriseStart, item.Date)
		}
	case "CORRECT":
		item.Action = "UPDATE"
		item.ExpectedRevision = cmd.Correct.ExpectedRecordRevision
		item.Date = cmd.Correct.NewEffectiveDate
		item.Changes = patch(cmd.Correct.Changes)
		if cmd.Correct.Classification != nil {
			item.Type = cmd.Correct.Classification.OperationType
			item.Reason = cmd.Correct.Classification.OperationReason
		}
		if item.Date != "" {
			err = kernel.ValidateCorrectionDate(t.Records, item.RecordID, enterpriseStart, item.Date)
		}
	case "REDATE":
		item.Action = "REDATE"
		item.ExpectedRevision = cmd.Redate.ExpectedRecordRevision
		item.Date = cmd.Redate.NewEffectiveDate
		err = kernel.ValidateCorrectionDate(t.Records, item.RecordID, enterpriseStart, item.Date)
	case "DELETE":
		item.Action = "DELETE"
		item.ExpectedRevision = cmd.Delete.ExpectedRecordRevision
	case "INSERT_LIFECYCLE_PAIR":
		p := cmd.InsertPair
		err = kernel.ValidateLifecyclePair(t.Records, p.AnchorRecordID, enterpriseStart, p.DeactivationDate, p.ReactivationDate)
		if err != nil {
			return work, nil, err
		}
		items := []d.Item{{ID: "deactivation", RecordID: foundation.NewID(), Action: "ADD", Date: p.DeactivationDate, Type: "DEACTIVATE", Reason: p.DeactivationReason}, {ID: "reactivation", RecordID: foundation.NewID(), Action: "ADD", Date: p.ReactivationDate, Type: "REACTIVATE", Reason: p.ReactivationReason}}
		w, e := kernel.Prepare(ctx, t.Records, t.First, enterpriseStart, items)
		return w, items, e
	case "DELETE_LIFECYCLE_PAIR":
		p := cmd.DeletePair
		stop, resume, e := kernel.LifecyclePair(t.Records, p.DeactivationRecordID)
		if e != nil {
			return work, nil, e
		}
		if stop.ID != p.DeactivationRecordID || resume.ID != p.ReactivationRecordID || stop.Revision != p.DeactivationRevision || resume.Revision != p.ReactivationRevision {
			return work, nil, d.Fault("M02_REVISION_CONFLICT", "", "停用和重新启用记录已更新，请核对后再操作")
		}
		items := []d.Item{{ID: "deactivation", RecordID: stop.ID, ExpectedRevision: stop.Revision, Action: "DELETE"}, {ID: "reactivation", RecordID: resume.ID, ExpectedRevision: resume.Revision, Action: "DELETE"}}
		w, e := kernel.Prepare(ctx, t.Records, t.First, enterpriseStart, items)
		return w, items, e
	}
	if err != nil {
		return work, nil, err
	}
	items := []d.Item{item}
	work, err = kernel.Prepare(ctx, t.Records, t.First, enterpriseStart, items)
	return work, items, err
}

func (s Service) Execute(ctx context.Context, c access.Credentials, cmd org.DepartmentCommand) (org.DepartmentResult, error) {
	return s.execute(ctx, c, cmd, d.Kernel)
}

// ExecuteHistoricalName is an offline-import entry point, absent from the public
// service interface and HTTP DTO. The composition root verifies source origin.
func (s Service) ExecuteHistoricalName(ctx context.Context, c access.Credentials, cmd org.DepartmentCommand, original string) (org.DepartmentResult, error) {
	var supplied *string
	if cmd.Action == "CREATE" {
		supplied = &cmd.Create.Name
	} else if cmd.Action == "ADD" {
		supplied = cmd.Add.Changes["name"]
	}
	if supplied == nil || *supplied != original {
		return org.DepartmentResult{}, d.Fault("M02_FORBIDDEN_INPUT", "name", "历史名称与已核对来源不符")
	}
	if _, e := base.HistoricalName(original); e != nil {
		return org.DepartmentResult{}, e
	}
	return s.execute(ctx, c, cmd, d.HistoricalNameKernel())
}

func (s Service) execute(ctx context.Context, c access.Credentials, cmd org.DepartmentCommand, kernel dated.Kernel[d.Data, d.Patch]) (org.DepartmentResult, error) {
	terminal, e := commandMetadata(cmd)
	if e != nil {
		return org.DepartmentResult{}, e
	}
	cmd.Fields = terminal.Fields
	reply, e := submission.Run(ctx, func(ctx context.Context, fn func(Unit) error) error { return s.Store.WithinDepartmentWrite(ctx, c, fn) }, func(u Unit, reply *submission.Reply[org.DepartmentResult]) error {
		p := u.Access().Principal()
		terminal.Digest = submission.BindDigest(p.InstanceID, p.ID, terminal.Digest)
		saved, e := submission.Lookup[org.DepartmentResult, Terminal](ctx, u, terminal.ID)
		if e != nil {
			return e
		}
		if saved != nil {
			*reply = submission.Replay(ctx, u, saved, terminal.Digest, submission.ReplayPolicy[org.DepartmentResult]{Outcome: func(r org.DepartmentResult) string { return r.Outcome }, Read: func(ctx context.Context, v *Saved) error { return authorizeSaved(ctx, u, v) }, MarkReplayed: func(r *org.DepartmentResult) { r.Replayed = true }})
			return nil
		}
		ent, e := u.Enterprise(ctx, true)
		if e != nil {
			return e
		}
		id := cmd.DepartmentID
		if cmd.Action == "CREATE" {
			id = foundation.NewID()
		}
		bu, parent, _, _, _ := explicitTargets(cmd, d.Timeline{})
		lock := LockRequest{DepartmentID: id, Code: cmd.Create.Code, BUs: []string{bu}, Parents: []string{parent}, Creating: cmd.Action == "CREATE"}
		locked, e := lockCommand(ctx, u, lock, cmd, terminal.ExpectedSource, ent.ValidFrom, kernel)
		if e != nil {
			return e
		}
		now, e := u.Now(ctx)
		if e != nil {
			return e
		}
		now = now.Truncate(time.Microsecond)
		ent = enterprisecheck.Display(ent, now)
		terminal.EvaluatedAt = &now
		before := d.Timeline{ID: id, Enterprise: ent.ID, First: foundation.NewID()}
		fields := []string{}
		deny := func(err error) error {
			var f *foundation.Fault
			if !errors.As(err, &f) || f.Status >= 500 {
				return err
			}
			reply.Rejection = err
			u.Confirm()
			return nil
		}
		if cmd.Action != "CREATE" {
			t := locked.Snapshot.Department(id)
			if t == nil {
				return deny(missing())
			}
			before = *t
			fields, e = readable(ctx, u, before, now, false)
			if e != nil {
				return deny(e)
			}
		}
		target := fact(before, now, cmd.Create.ManagementBUID)
		_, e = require(ctx, u, requirements(cmd, before, target))
		if e != nil {
			return deny(e)
		}
		if cmd.Action == "CREATE" {
			r, e := require(ctx, u, intent(target, "department.read", "command-result", nil))
			if e != nil {
				return deny(e)
			}
			fields = r.ReadFields
		}
		if e = preauthorizeRelations(ctx, u, locked.Snapshot, before, cmd, target, now); e != nil {
			return deny(e)
		}
		if cmd.Action != "CREATE" {
			terminal.EnterpriseID = ent.ID
		}
		reject := func(err error) error {
			var fault *foundation.Fault
			if !errors.As(err, &fault) || fault.Status >= 500 {
				return err
			}
			fault = safeLocalFault(fault, fields)
			terminal.Outcome = "rejected"
			terminal.ErrorCode = fault.Code
			locations := []org.Location{}
			if fault.Field != "" {
				locations = append(locations, org.Location{FieldPath: fault.Field})
			}
			terminal.Failure = &org.FailureDetails{FormatVersion: 1, Issues: []org.Issue{{Code: fault.Code, Locations: locations}}}
			if terminal.Action == "REVISE_TIMELINE" && terminal.Items == nil {
				terminal.Items = []org.RequestItem{}
			}
			terminal.Result = org.DepartmentResult{SubmissionID: terminal.ID, Outcome: "rejected", ErrorCode: &terminal.ErrorCode, FailureDetails: terminal.Failure}
			if terminal.Action == "REVISE_TIMELINE" {
				terminal.Result.Items = []org.DepartmentItemResult{}
			}
			if e := u.StoreTerminal(ctx, terminal); e != nil {
				return e
			}
			u.Confirm()
			reply.Result = terminal.Result
			reply.Rejection = fault
			return nil
		}
		if cmd.Action == "CREATE" {
			before.Code, e = base.NormalizeCode(cmd.Create.Code)
			if e != nil {
				return reject(e)
			}
		} else if before.Source != terminal.ExpectedSource {
			return reject(d.Fault("M02_SOURCE_CONFLICT", "", "资料已更新，请核对最新内容后保存"))
		}
		candidate, plan, items, e := qualifiedCandidate(ctx, u, locked, before, cmd, target, now, ent, kernel)
		if e != nil {
			var f *foundation.Fault
			if errors.As(e, &f) && (f.Code == "ACCESS_DENIED" || f.Code == "M02_OBJECT_NOT_FOUND") {
				return deny(e)
			}
			return reject(e)
		}
		if e = enterprisecheck.Records(ent, candidate.Records); e != nil {
			return reject(e)
		}
		terminal.Items = metadata(items, cmd, before)
		if candidate.NoChange {
			terminal.Outcome = "no_change"
			terminal.Result = org.DepartmentResult{SubmissionID: terminal.ID, Outcome: "no_change", DepartmentID: &id, ExpectedSourceVersion: &terminal.ExpectedSource, ObservedSourceVersion: &before.Source, Action: terminal.Action, SelectedRecordID: ptr(cmd.RecordID)}
			if cmd.Action == "ADD" {
				terminal.Result.SelectedRecordID = ptr(cmd.Add.AnchorRecordID)
			}
			setEvaluation(&terminal.Result, now)
			terminal.Result.CurrentStatus = d.Status(before, *terminal.Result.EvaluatedBusinessDate)
			if e = u.StoreTerminal(ctx, terminal); e != nil {
				return e
			}
			u.Confirm()
			reply.Result = terminal.Result
			trimResult(&reply.Result, fields)
			return nil
		}
		operator, e := u.Access().OperatorSnapshot(ctx)
		if e != nil {
			return e
		}
		write, e := assemble(ctx, locked.Snapshot, before, candidate, plan, items, cmd, terminal, operator, ent.Name)
		if e != nil {
			return e
		}
		e = u.Business(ctx, func() error { r, e := u.Persist(ctx, write); reply.Result = r; return e })
		if e != nil {
			return reject(e)
		}
		if e = populateResultItems(ctx, u, &reply.Result, fields, now); e != nil {
			return e
		}
		u.Confirm()
		trimResult(&reply.Result, fields)
		return nil
	})
	if e != nil {
		return org.DepartmentResult{}, e
	}
	return reply.Result, reply.Rejection
}

// Only the complete Go candidate can prove that descendants are unaffected.
// This preliminary read grants no right and does not authorize persistence:
// the locked source, all command rights and the finalized candidate are checked
// below, and the store binds the narrower guard to this exact original source.
func lockCommand(ctx context.Context, u Unit, in LockRequest, cmd org.DepartmentCommand, expected, enterpriseStart string, kernel dated.Kernel[d.Data, d.Patch]) (ReadSnapshot, error) {
	// Preliminary discovery and the locked reread share one preparation budget.
	// Lock acquisition has its own cumulative bound inside the store; necessary
	// fact reads must not consume that shorter contention allowance.
	ctx, cancel := context.WithTimeout(ctx, foundation.DepartmentPreparationBudget(ctx))
	defer cancel()
	// These lifecycle operations use the conservative full dependency plan.
	// An extra unlocked candidate can grant no narrower plan for a valid
	// coverage change; all qualification still runs on the locked snapshot.
	switch cmd.Action {
	case "INSERT_LIFECYCLE_PAIR", "DELETE_LIFECYCLE_PAIR", "DEACTIVATE", "REACTIVATE":
		return u.Lock(ctx, in)
	}
	if !in.Creating {
		read, err := u.PreviewSnapshot(ctx, in.DepartmentID, in.BUs, in.Parents, false)
		if err != nil {
			return ReadSnapshot{}, err
		}
		if before := read.Snapshot.Department(in.DepartmentID); before != nil && before.Source == expected {
			work, _, err := localCandidate(ctx, cmd, *before, enterpriseStart, kernel)
			if err == nil && !d.Plan(*before, work.Records).Changed {
				return u.LockUnchangedRelations(ctx, in, *before, work.Records)
			}
		}
	}
	return u.Lock(ctx, in)
}

func setEvaluation(r *org.DepartmentResult, now time.Time) {
	t := now.UTC().Format(time.RFC3339Nano)
	date := now.In(foundation.Shanghai).Format("2006-01-02")
	r.EvaluatedAt = &t
	r.EvaluatedBusinessDate = &date
}
func safeLocalFault(f *foundation.Fault, fields []string) *foundation.Fault {
	v := *f
	field := strings.TrimPrefix(v.Field, "changes.")
	field = strings.TrimPrefix(field, "classification.")
	if field == "newEffectiveDate" {
		field = "effectiveDate"
	}
	if field != "" && !slices.Contains(fields, field) || !slices.Contains(fields, "effectiveDate") {
		v.Field = ""
		v.Message = "当前资料不满足本次操作条件"
	}
	return &v
}
func trimResult(r *org.DepartmentResult, fields []string) {
	if !slices.Contains(fields, "effectiveDate") {
		r.EffectiveDate = nil
		r.PreviousEffectiveDate = nil
		r.EvaluatedBusinessDate = nil
		if r.Impact != nil {
			r.Impact.DataAffectedRanges = nil
			r.Impact.PeriodChanges = nil
		}
		for i := range r.Items {
			r.Items[i].EffectiveDate = ""
		}
	}
	if !slices.Contains(fields, "status") {
		r.CurrentStatus = ""
	}
	if r.FailureDetails != nil {
		for i := range r.FailureDetails.Issues {
			locations := []org.Location{}
			for _, loc := range r.FailureDetails.Issues[i].Locations {
				if slices.Contains(fields, strings.TrimPrefix(loc.FieldPath, "changes.")) {
					locations = append(locations, loc)
				}
			}
			r.FailureDetails.Issues[i].Locations = locations
		}
	}
}

func qualifiedCandidate(ctx context.Context, u Unit, locked ReadSnapshot, before d.Timeline, cmd org.DepartmentCommand, target access.DepartmentTarget, now time.Time, ent org.Enterprise, kernel dated.Kernel[d.Data, d.Patch]) (d.Candidate, d.RelationPlan, []d.Item, error) {
	work, items, e := localCandidate(ctx, cmd, before, ent.ValidFrom, kernel)
	if e != nil {
		return d.Candidate{}, d.RelationPlan{}, nil, e
	}
	return qualifyPreparedCandidate(ctx, u, locked, before, cmd, target, now, ent, work, items)
}

// Preview and saving share the same qualification/finalization. The prepared
// local candidate is never a persistence candidate and never bypasses rights.
func qualifyPreparedCandidate(ctx context.Context, u Unit, locked ReadSnapshot, before d.Timeline, cmd org.DepartmentCommand, target access.DepartmentTarget, now time.Time, ent org.Enterprise, work dated.WorkingCandidate[d.Data, d.Patch], items []d.Item) (d.Candidate, d.RelationPlan, []d.Item, error) {
	defer foundation.BeginStage(ctx, "department.qualify")()
	var e error
	if e = authorizeIntroduced(ctx, u, locked.Snapshot, before, work.Records, target, now); e != nil {
		return d.Candidate{}, d.RelationPlan{}, nil, e
	}
	if cmd.Action == "ADD" && len(d.Deltas(before.Records, work.Records)) > 0 && cmd.Add.OperationReason != "ORGANIZATION_REASSIGNMENT" {
		return d.Candidate{}, d.RelationPlan{}, nil, d.Fault("M02_INVALID_INPUT", "operationReason", "变更归属组织须选择对应原因")
	}
	over := before
	over.Records = work.Records
	facts := locked.Snapshot.With(over)
	deps := d.Discover(facts, before.ID, nil, nil)
	if e = d.Validate(ctx, facts, deps); e != nil {
		return d.Candidate{}, d.RelationPlan{}, nil, safeQualification(ctx, u, facts, ent.ID, before.ID, now, e)
	}
	qualifiedRows := d.Kernel.CloneRecords(work.Records)
	var plan d.RelationPlan
	var candidate d.Candidate
	for remaining := len(work.Records) + 1; remaining > 0; remaining-- {
		plan = d.Plan(before, work.Records)
		bound, e := plan.Bind(work.Records)
		if e != nil {
			return d.Candidate{}, d.RelationPlan{}, nil, e
		}
		work.Records = bound
		candidate, e = d.Kernel.Finalize(ctx, work)
		if e != nil {
			return d.Candidate{}, d.RelationPlan{}, nil, e
		}
		if reflect.DeepEqual(candidate.Records, work.Records) {
			break
		}
		work.Records = candidate.Records
		if remaining == 1 {
			return d.Candidate{}, d.RelationPlan{}, nil, fmt.Errorf("department candidate did not converge")
		}
	}
	if len(d.Deltas(qualifiedRows, candidate.Records)) != 0 {
		return d.Candidate{}, d.RelationPlan{}, nil, fmt.Errorf("finalization expanded authorized relation envelope")
	}
	if e = authorizeIntroduced(ctx, u, locked.Snapshot, before, candidate.Records, target, now); e != nil {
		return d.Candidate{}, d.RelationPlan{}, nil, e
	}
	if _, e = require(ctx, u, requirements(cmd, before, target)); e != nil {
		return d.Candidate{}, d.RelationPlan{}, nil, e
	}

	return candidate, plan, items, nil
}

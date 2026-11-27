package application

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/application/buauth"
	"chnhcm.local/chnhcm/internal/contexts/organization/application/enterprisecheck"
	"chnhcm.local/chnhcm/internal/contexts/organization/application/submission"
	"chnhcm.local/chnhcm/internal/contexts/organization/domain"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"
)

type Timeline struct {
	BUID, Code, First, Source string
	SourceNo                  int64
	Records                   []domain.Record
}
type Terminal struct {
	ID, Outcome, Action, BUID, RecordID, Revision, ExpectedSource, ErrorCode string
	Digest                                                                   []byte
	Items                                                                    []org.RequestItem
	ItemOutcomes                                                             []org.ItemResult
	Fields                                                                   []string
	Failure                                                                  *org.FailureDetails
	EvaluatedAt                                                              *time.Time
	Result                                                                   org.Result
}
type Write struct {
	Timeline  Timeline
	Candidate domain.Candidate
	Terminal  Terminal
	Items     []domain.Item
	Create    *domain.Create
}

func (s Service) Create(ctx context.Context, c access.Credentials, cmd org.CreateCommand) (org.Result, error) {
	return s.execute(ctx, c, cmd, nil)
}
func (s Service) Maintain(ctx context.Context, c access.Credentials, cmd org.MaintainCommand) (org.Result, error) {
	fixed, e := fixedCommand(cmd)
	if e != nil {
		return org.Result{}, e
	}
	return s.execute(ctx, c, org.CreateCommand{}, &fixed)
}
func maintenance(cmd maintenanceCommand) (domain.Item, string, string) {
	it := domain.Item{ID: "candidate", Action: cmd.Action, RecordID: cmd.RecordID}
	expected := ""
	action := ""
	if cmd.Action == "REVISE_TIMELINE" && cmd.PairAction != "" {
		return it, cmd.PairExpectedSource, "bu.deactivate"
	}
	if slices.Contains([]string{"ADD", "DEACTIVATE", "REACTIVATE"}, cmd.Action) {
		it.Date = cmd.Add.EffectiveDate
		it.Type = cmd.Add.OperationType
		it.Reason = cmd.Add.OperationReason
		it.Changes = domain.PatchFromFields(cmd.Add.Changes)
		expected = cmd.Add.ExpectedSourceVersion
		action = "bu.record.add"
		if cmd.Action == "ADD" && it.Type != "CHANGE" {
			return it, "", ""
		}
		if cmd.Action != "ADD" {
			it.Action = "ADD"
			if it.Type != cmd.Action {
				return it, "", ""
			}
			action = map[string]string{"DEACTIVATE": "bu.deactivate", "REACTIVATE": "bu.reactivate"}[cmd.Action]
		}
	}
	if cmd.Action == "CORRECT" {
		it.Action = "UPDATE"
		it.Date = cmd.Correct.NewEffectiveDate
		if cmd.Correct.Classification != nil {
			it.Type = cmd.Correct.Classification.OperationType
			it.Reason = cmd.Correct.Classification.OperationReason
		}
		it.ExpectedRevision = cmd.Correct.ExpectedRecordRevision
		it.Changes = domain.PatchFromFields(cmd.Correct.Changes)
		expected = cmd.Correct.ExpectedSourceVersion
		action = "bu.record.correct"
	}
	if cmd.Action == "DELETE" {
		it.ExpectedRevision = cmd.Delete.ExpectedRecordRevision
		expected = cmd.Delete.ExpectedSourceVersion
		// The actual category is resolved under the BU lock before authorization.
		action = "bu.record.delete"
	}
	if cmd.Action == "REDATE" {
		it.Date = cmd.Redate.NewEffectiveDate
		it.ExpectedRevision = cmd.Redate.ExpectedRecordRevision
		expected = cmd.Redate.ExpectedSourceVersion
		action = "bu.record.redate"
	}
	return it, expected, action
}
func commandRead(ctx context.Context, u Unit, enterprise, bu string) (access.Decision, error) {
	d, e := u.Access().Decide(ctx, access.Intent{Action: "bu.read", EnterpriseID: enterprise, BUID: bu, Kind: "command-object", Fields: []string{}})
	if e != nil {
		return d, e
	}
	if !d.Allowed {
		return d, notFound()
	}
	return d, nil
}
func (s Service) execute(ctx context.Context, c access.Credentials, create org.CreateCommand, maintain *maintenanceCommand) (out org.Result, err error) {
	terminal := Terminal{ID: create.SubmissionID, Action: "CREATE", Digest: create.Digest, Fields: create.Fields}
	action := "bu.create"
	var item domain.Item
	if maintain != nil {
		terminal = Terminal{ID: maintain.SubmissionID, Action: maintain.Action, BUID: maintain.BUID, RecordID: maintain.RecordID, Digest: maintain.Digest, Fields: maintain.Fields}
		item, terminal.ExpectedSource, action = maintenance(*maintain)
		if slices.Contains([]string{"ADD", "DEACTIVATE", "REACTIVATE"}, maintain.Action) {
			if !foundation.ValidID(maintain.Add.AnchorRecordID) {
				return out, &foundation.Fault{Status: 400, Code: "M02_INVALID_INPUT", Field: "anchorRecordId", Message: "请选择有效的新增位置"}
			}
			if maintain.Action == "ADD" && maintain.Add.Changes == nil {
				return out, &foundation.Fault{Status: 400, Code: "M02_INVALID_INPUT", Field: "changes", Message: "请提供名称修改对象"}
			}
		}
		if maintain.Action == "CORRECT" {
			if e := maintain.Correct.Validate(); e != nil {
				return out, e
			}
			terminal.Fields = org.CorrectionFields(maintain.Correct)
		}
		terminal.Revision = item.ExpectedRevision
		if action == "" || !foundation.ValidID(terminal.BUID) || !foundation.ValidID(terminal.ExpectedSource) || (slices.Contains([]string{"CORRECT", "DELETE", "REDATE"}, terminal.Action) && (!foundation.ValidID(terminal.RecordID) || !foundation.ValidID(terminal.Revision))) {
			return out, foundation.Fail(400, "M02_INVALID_INPUT", "操作内容不完整，请重新打开表单")
		}
	}
	if maintain != nil && terminal.Action == "REVISE_TIMELINE" {
		if e := validatePairIdentity(maintain.PairItems); e != nil {
			return out, e
		}
		terminal.Items = pairMetadata(*maintain, nil)
	}
	slices.Sort(terminal.Fields)
	terminal.Fields = slices.Compact(terminal.Fields)
	if !foundation.ValidID(terminal.ID) {
		return out, foundation.Fail(400, "M02_INVALID_INPUT", "保存请求无效")
	}
	var outputFields []string
	reply, err := submission.Run(ctx, func(ctx context.Context, fn func(Unit) error) error {
		return s.Store.WithinWrite(ctx, c, fn)
	}, func(u Unit, reply *submission.Reply[org.Result]) error {
		p := u.Access().Principal()
		terminal.Digest = submission.BindDigest(p.InstanceID, p.ID, terminal.Digest)
		old, e := submission.Lookup[org.Result, Terminal](ctx, u, terminal.ID)
		if e != nil {
			return e
		}
		ent, e := u.Enterprise(ctx, false)
		if e != nil {
			return e
		}
		if old != nil {
			*reply = submission.Replay(ctx, u, old, terminal.Digest, submission.ReplayPolicy[org.Result]{
				Outcome:      func(r org.Result) string { return r.Outcome },
				Read:         func(ctx context.Context, saved *Saved) error { return authorizeSaved(ctx, u, saved, ent.ID) },
				MarkReplayed: func(r *org.Result) { r.Replayed = true },
			})
			return nil
		}
		// The subject guard and submission lock already exist. Read denial does not
		// establish a safe business target and does not persist a terminal.
		if maintain != nil {
			var read access.Decision
			if read, e = commandRead(ctx, u, ent.ID, terminal.BUID); e != nil {
				reply.Rejection = e
				u.Confirm()
				return nil
			}
			outputFields = read.ReadFields
		}
		var lockedTimeline *Timeline
		if maintain != nil {
			if _, e = u.Enterprise(ctx, true); e != nil {
				return e
			}
			base, loadErr := u.LoadTimeline(ctx, terminal.BUID, false)
			if loadErr != nil {
				return loadErr
			}
			_, proposed, preparationErr := prepareMaintenance(ctx, base, ent.ValidFrom, *maintain, item)
			var value Timeline
			if preparationErr == nil && base.Source == terminal.ExpectedSource && domain.SameBUCoverage(base.Records, proposed.Records) {
				value, loadErr = u.LockUnchangedBUCoverage(ctx, base, proposed)
			} else {
				value, loadErr = u.LoadTimeline(ctx, terminal.BUID, true)
			}
			if loadErr != nil {
				return loadErr
			}
			lockedTimeline = &value
			if terminal.Action == "DELETE" || terminal.Action == "REDATE" {
				action = targetAction(value, terminal.RecordID, terminal.Action)
			}
		}
		if terminal.Action == "REVISE_TIMELINE" && lockedTimeline != nil && lockedTimeline.Source == terminal.ExpectedSource {
			terminal.Items = pairMetadata(*maintain, lockedTimeline.Records)
		}
		intent := access.Intent{Action: action, EnterpriseID: ent.ID, BUID: terminal.BUID, Fields: terminal.Fields, Kind: "command"}
		if terminal.Action == "REVISE_TIMELINE" {
			intent = pairIntent(*lockedTimeline, *maintain, ent.ID)
		}
		if terminal.Action == "CORRECT" {
			intent = correctionIntent(*lockedTimeline, terminal.RecordID, ent.ID, maintain.Correct)
		}
		intent.PermissionOnly = true
		decision, e := u.Access().Decide(ctx, intent)
		if e != nil {
			return e
		}
		if !decision.Allowed {
			reply.Rejection = permission()
			u.Confirm()
			return nil
		}
		if maintain == nil {
			read, e := u.Access().Decide(ctx, access.Intent{Action: "bu.read", EnterpriseID: ent.ID, Kind: "command-result", Fields: []string{}})
			if e != nil {
				return e
			}
			if !read.Allowed {
				reply.Rejection = permission()
				u.Confirm()
				return nil
			}
			outputFields = read.ReadFields
		}
		ent, e = u.Enterprise(ctx, true)
		if e != nil {
			return e
		}
		if e = u.Access().ValidateUse(); e != nil {
			return e
		}
		w := Write{Terminal: terminal}
		reject := func(failure error) error {
			failure = targetError(terminal.Action, failure)
			var f *foundation.Fault
			if !errors.As(failure, &f) || f.Status >= 500 {
				return failure
			}
			terminal.Outcome = "rejected"
			terminal.ItemOutcomes = nil
			terminal.Result = org.Result{}
			terminal.ErrorCode = f.Code
			loc := []org.Location{}
			if f.Field != "" {
				loc = append(loc, org.Location{FieldPath: f.Field})
			}
			terminal.Failure = &org.FailureDetails{FormatVersion: 1, Issues: []org.Issue{{Code: f.Code, Locations: loc}}}
			if terminal.Action == "REVISE_TIMELINE" {
				terminal.Failure = jointFailure(failure, terminal.Items)
			}
			if e := u.StoreTerminal(ctx, terminal); e != nil {
				return e
			}
			u.Confirm()
			reply.Result = org.Result{SubmissionID: terminal.ID, Outcome: "rejected", ErrorCode: &terminal.ErrorCode, FailureDetails: terminal.Failure}
			reply.Rejection = failure
			return nil
		}
		if maintain == nil {
			in, candidate, e := domain.CreateTimeline(ctx, domain.Create{Code: create.Input.Code, Name: create.Input.Name, NameEn: create.Input.NameEn, EffectiveDate: create.Input.EffectiveDate, OperationType: create.Input.OperationType, OperationReason: create.Input.OperationReason}, ent.ValidFrom)
			if e != nil {
				return reject(e)
			}
			w.Create = &in
			w.Candidate = candidate
			w.Timeline = Timeline{Code: in.Code}
		} else {
			if lockedTimeline != nil {
				w.Timeline = *lockedTimeline
			} else {
				w.Timeline, e = u.LoadTimeline(ctx, terminal.BUID, true)
			}
			if e != nil {
				return reject(e)
			}
			if w.Timeline.Source != terminal.ExpectedSource {
				return reject(domainFault("M02_SOURCE_CONFLICT", "", "资料已更新，请核对最新内容后保存"))
			}
			w.Items, w.Candidate, e = prepareMaintenance(ctx, w.Timeline, ent.ValidFrom, *maintain, item)
			if e != nil {
				return reject(e)
			}
		}
		now, e := u.Now(ctx)
		if e != nil {
			return e
		}
		terminal.EvaluatedAt = &now
		if terminal.Action == "REVISE_TIMELINE" {
			terminal.ItemOutcomes = jointOutcomes(w.Timeline.Records, w.Candidate, w.Items)
		}
		w.Terminal = terminal
		if maintain != nil && w.Candidate.NoChange {
			terminal.Outcome = "no_change"
			terminal.Result = org.Result{SubmissionID: terminal.ID, Outcome: "no_change", BUID: &terminal.BUID, ObservedSourceVersion: &terminal.ExpectedSource, ExpectedSourceVersion: &terminal.ExpectedSource, Action: terminal.Action}
			terminal.Result.Items = terminal.ItemOutcomes
			if terminal.RecordID != "" {
				terminal.Result.SelectedRecordID = &terminal.RecordID
			} else if terminal.Action == "ADD" {
				terminal.Result.SelectedRecordID = &maintain.Add.AnchorRecordID
			}
			if e = u.StoreTerminal(ctx, terminal); e != nil {
				return e
			}
			u.Confirm()
			reply.Result = terminal.Result
			if terminal.Action == "REVISE_TIMELINE" {
				stored, e := u.FindResult(ctx, terminal.ID)
				if e != nil {
					return e
				}
				if stored == nil || stored.DecodeError != nil {
					return foundation.Fail(503, "M02_RESULT_UNAVAILABLE", "结果暂不可读")
				}
				reply.Result = stored.Result
			}
			setEvaluation(&reply.Result, now)
			reply.Result.CurrentStatus = domain.StatusAt(w.Timeline.Records, *reply.Result.EvaluatedBusinessDate)
			return nil
		}
		if e = enterprisecheck.Records(ent, w.Candidate.Records); e != nil {
			return reject(e)
		}
		e = u.Business(ctx, func() error { var e error; reply.Result, e = u.Persist(ctx, w); return e })
		if e != nil {
			return reject(e)
		}
		u.Confirm()
		return nil
	})
	if err != nil {
		// Callback results are provisional until the outer transaction commits.
		// In particular, a rolled-back rejection must not close client recovery.
		return org.Result{}, err
	}
	out, err = reply.Result, reply.Rejection
	if outputFields != nil {
		out.CurrentReadFields = outputFields
		TrimResult(&out, outputFields)
		err = safeTimelineError(err, outputFields)
	}
	return
}

func targetAction(tl Timeline, recordID, command string) string {
	for _, r := range tl.Records {
		if r.ID == recordID {
			action := buauth.DeleteAction(r.Type)
			if command == "REDATE" {
				action = buauth.RedateAction(r.Type)
			}
			if action != "" {
				return action
			}
		}
	}
	// No valid category exists for a missing target or the protected first record.
	// This does not grant execution; domain validation still rejects the request.
	if command == "REDATE" {
		return "bu.record.redate"
	}
	return "bu.record.delete"
}

func targetError(action string, err error) error {
	var fault *foundation.Fault
	if (action == "REDATE" || action == "CORRECT") && errors.As(err, &fault) && fault.Field == "effectiveDate" {
		copy := *fault
		copy.Field = "newEffectiveDate"
		return &copy
	}
	if action == "DELETE" && errors.As(err, &fault) && fault.Field != "" {
		copy := *fault
		copy.Field = "" // A deletion has no editable business fields to focus.
		return &copy
	}
	return err
}
func domainFault(code, field, message string) error {
	return &foundation.Fault{Status: 409, Code: code, Field: field, Message: message}
}
func setEvaluation(out *org.Result, now time.Time) {
	a := now.UTC().Format(time.RFC3339Nano)
	d := now.In(foundation.Shanghai).Format("2006-01-02")
	out.EvaluatedAt = &a
	out.EvaluatedBusinessDate = &d
}
func authorizeSaved(ctx context.Context, u Unit, saved *Saved, enterprise string) error {
	r := &saved.Result
	bu := saved.RequestObjectID
	if r.BUID != nil {
		bu = *r.BUID
	}
	if bu == "" {
		d, e := u.Access().Decide(ctx, access.Intent{Action: "submission.resolve", Kind: "resolve-own", Fields: []string{}})
		if e != nil {
			return e
		}
		if !d.Allowed {
			return permission()
		}
		return nil
	}
	// Rejections retain original request location, without claiming it exists.
	if r.BUID == nil {
		r.BUID = &bu
		defer func() { r.BUID = nil }()
	}
	return authorizeResult(ctx, u, r, enterprise)
}
func PublicImpact(v domain.Impact) org.TimelineImpact {
	b, _ := json.Marshal(v)
	var out org.TimelineImpact
	_ = json.Unmarshal(b, &out)
	return out
}
func PublicRecord(r domain.Record, fields []string) org.Record {
	out := org.Record{ID: r.ID, Revision: r.Revision, State: r.State}
	if slices.Contains(fields, "effectiveDate") {
		out.EffectiveDate = r.Date
		if r.End != "" {
			out.EndExclusive = &r.End
			// Display the last applicable day without changing the stored [start,end) range.
			end := domain.NextDate(r.End, -1)
			out.EndInclusive = &end
		}
	}
	if slices.Contains(fields, "name") {
		out.Name = &r.Data.Name
	}
	if slices.Contains(fields, "nameEn") {
		out.NameEn = r.Data.NameEn
	}
	if slices.Contains(fields, "operationType") {
		out.OperationType = r.Type
		out.TypeLabel = r.TypeLabel
	}
	if slices.Contains(fields, "operationReason") {
		out.OperationReason = r.Reason
		out.ReasonLabel = r.ReasonLabel
	}
	return out
}
func (s Service) Preview(ctx context.Context, c access.Credentials, id string, in org.PreviewInput) (out org.Preview, err error) {
	if !foundation.ValidID(id) || !foundation.ValidID(in.ExpectedSourceVersion) {
		return out, foundation.Fail(400, "M02_INVALID_INPUT", "所选资料无效，请重新打开")
	}
	cmd := org.MaintainCommand{Action: in.Action, BUID: id, RecordID: in.RecordID, Add: org.AddInput{AnchorRecordID: in.AnchorRecordID, ExpectedSourceVersion: in.ExpectedSourceVersion, EffectiveDate: in.EffectiveDate, OperationType: in.OperationType, OperationReason: in.OperationReason, Changes: in.Changes}, Correct: org.CorrectInput{ExpectedSourceVersion: in.ExpectedSourceVersion, ExpectedRecordRevision: in.ExpectedRecordRevision, Changes: in.Changes, NewEffectiveDate: in.NewEffectiveDate, Classification: in.Classification}, Delete: org.DeleteInput{ExpectedSourceVersion: in.ExpectedSourceVersion, ExpectedRecordRevision: in.ExpectedRecordRevision}, Redate: org.RedateInput{ExpectedSourceVersion: in.ExpectedSourceVersion, ExpectedRecordRevision: in.ExpectedRecordRevision, NewEffectiveDate: in.NewEffectiveDate}}
	it, _, action := maintenance(maintenanceCommand{MaintainCommand: cmd})
	if action == "" {
		return out, foundation.Fail(400, "M02_INVALID_INPUT", "未开放的预览动作")
	}
	isTarget := in.Action == "CORRECT" || in.Action == "DELETE" || in.Action == "REDATE"
	if isTarget && (in.EffectiveDate != "" || in.OperationType != "" || in.OperationReason != "" || !foundation.ValidID(in.RecordID) || !foundation.ValidID(in.ExpectedRecordRevision)) {
		return out, foundation.Fail(400, "M02_FORBIDDEN_INPUT", "记录处理不接受日期或分类")
	}
	if (in.Action == "DELETE" || in.Action == "REDATE") && in.Changes != nil {
		return out, foundation.Fail(400, "M02_FORBIDDEN_INPUT", "此记录处理不接受名称修改")
	}
	if !isTarget && (in.RecordID != "" || in.ExpectedRecordRevision != "") {
		return out, foundation.Fail(400, "M02_FORBIDDEN_INPUT", "新增内容无效，请重新打开表单")
	}
	if in.Action == "DEACTIVATE" && in.Changes != nil {
		return out, foundation.Fail(400, "M02_FORBIDDEN_INPUT", "停用不接受名称修改")
	}
	if in.Action == "REACTIVATE" && in.Changes != nil && len(in.Changes) == 0 {
		return out, foundation.Fail(400, "M02_INVALID_INPUT", "明确修改时至少提供一个名称字段")
	}
	if in.Action != "REDATE" && in.Action != "CORRECT" && in.NewEffectiveDate != "" {
		return out, foundation.Fail(400, "M02_FORBIDDEN_INPUT", "此动作不接受新日期")
	}
	if slices.Contains([]string{"ADD", "DEACTIVATE", "REACTIVATE"}, in.Action) && !foundation.ValidID(in.AnchorRecordID) {
		return out, &foundation.Fault{Status: 400, Code: "M02_INVALID_INPUT", Field: "anchorRecordId", Message: "请选择有效的新增位置"}
	}
	if !slices.Contains([]string{"ADD", "DEACTIVATE", "REACTIVATE"}, in.Action) && in.AnchorRecordID != "" {
		return out, foundation.Fail(400, "M02_FORBIDDEN_INPUT", "本动作不接受新增位置")
	}
	if in.Action != "CORRECT" && in.Classification != nil {
		return out, foundation.Fail(400, "M02_FORBIDDEN_INPUT", "本动作不接受分类更正")
	}
	fields := []string{}
	if in.Action == "REDATE" {
		fields = append(fields, "effectiveDate")
	}
	for f := range in.Changes {
		fields = append(fields, f)
	}
	if !isTarget {
		fields = append(fields, "effectiveDate", "operationType", "operationReason")
	}
	err = s.Store.Within(ctx, c, func(u Unit) error {
		ent, e := u.Enterprise(ctx, false)
		if e != nil {
			return e
		}
		read, e := commandRead(ctx, u, ent.ID, id)
		if e != nil {
			var fault *foundation.Fault
			if errors.As(e, &fault) {
				return RejectRead(e)
			}
			return e
		}
		out.ReadFields = read.ReadFields
		tl, e := u.LoadTimeline(ctx, id, false)
		if e != nil {
			return e
		}
		if in.Action == "DELETE" || in.Action == "REDATE" {
			action = targetAction(tl, in.RecordID, in.Action)
		}
		intent := access.Intent{Action: action, EnterpriseID: ent.ID, BUID: id, Kind: "preview", Fields: fields}
		if in.Action == "CORRECT" {
			if e := cmd.Correct.Validate(); e != nil {
				return RejectRead(e)
			}
			intent = correctionIntent(tl, in.RecordID, ent.ID, cmd.Correct)
		}
		d, e := u.Access().Decide(ctx, intent)
		if e != nil {
			return e
		}
		if !d.Allowed {
			return RejectRead(permission())
		}
		if tl.Source != in.ExpectedSourceVersion {
			return RejectRead(domainFault("M02_SOURCE_CONFLICT", "", "资料已更新，请核对后保存"))
		}
		out.ExpectedSourceVersion = tl.Source
		// Incomplete date assistance validates its position without constructing a
		// candidate. Complete previews share exactly the save preparation below.
		incomplete := foundation.ValidDate(in.EffectiveDate) && ((in.Action == "ADD" && (in.Changes == nil || in.OperationReason == "")) || (slices.Contains([]string{"DEACTIVATE", "REACTIVATE"}, in.Action) && in.OperationReason == ""))
		if incomplete {
			var positionErr error
			if in.Action == "ADD" {
				positionErr = domain.ValidateAdditionPosition(tl.Records, in.AnchorRecordID, ent.ValidFrom, in.EffectiveDate)
			} else {
				positionErr = domain.ValidateLifecycleAppend(tl.Records, in.AnchorRecordID, in.Action, ent.ValidFrom, in.EffectiveDate)
			}
			if positionErr != nil {
				return RejectRead(safeTimelineError(positionErr, read.ReadFields))
			}
			out.Outcome = "incomplete"
			return nil
		}
		_, candidate, e := prepareMaintenance(ctx, tl, ent.ValidFrom, maintenanceCommand{MaintainCommand: cmd}, it)
		if e != nil {
			var fault *foundation.Fault
			if errors.As(e, &fault) {
				return RejectRead(safeTimelineError(targetError(in.Action, e), read.ReadFields))
			}
			return e
		}
		if e = enterprisecheck.Records(ent, candidate.Records); e != nil {
			return submission.RejectRead(e)
		}
		if e = u.ValidateReferences(ctx, tl, candidate); e != nil {
			return RejectRead(e)
		}
		out.Outcome = "accepted"
		if candidate.NoChange {
			out.Outcome = "no_change"
		}

		if in.Action == "REDATE" || in.Action == "CORRECT" && in.NewEffectiveDate != "" {
			for _, r := range tl.Records {
				if r.ID == in.RecordID {
					if r.ID == tl.First && it.Date > r.Date && slices.Contains(read.ReadFields, "effectiveDate") {
						out.NewNoFactRange = &org.PreviewDateRange{Start: r.Date, EndExclusive: it.Date, EndInclusive: domain.NextDate(it.Date, -1)}
					}
				}
			}

		}
		out.Impact = PublicImpact(candidate.Impact)
		out.Impact.AffectedRecordIDs = slices.DeleteFunc(out.Impact.AffectedRecordIDs, func(id string) bool { return id == "candidate" })
		for i := range out.Impact.StructuralChanges {
			if isTarget {
				out.Impact.StructuralChanges[i].ItemID = ""
			}
			if out.Impact.StructuralChanges[i].RecordID == "candidate" {
				out.Impact.StructuralChanges[i].RecordID = ""
			}
		}
		for i := range out.Impact.PeriodChanges {
			if isTarget {
				out.Impact.PeriodChanges[i].ItemID = ""
			}
			if out.Impact.PeriodChanges[i].RecordID == "candidate" {
				out.Impact.PeriodChanges[i].RecordID = ""
			}
		}
		if !slices.Contains(read.ReadFields, "effectiveDate") || !slices.Contains(read.ReadFields, "name") || !slices.Contains(read.ReadFields, "nameEn") {
			out.Impact = org.TimelineImpact{}
		}

		now, e := u.Now(ctx)
		if e != nil {
			return e
		}
		out.EvaluatedAt = now.UTC().Format(time.RFC3339Nano)
		out.EvaluatedBusinessDate = now.In(foundation.Shanghai).Format("2006-01-02")
		out.CurrentStatus = domain.StatusAt(candidate.Records, out.EvaluatedBusinessDate)
		if in.Action == "DELETE" || in.Action == "REDATE" || in.Action == "CORRECT" && in.NewEffectiveDate != "" {
			out.PreviousCurrentStatus = domain.StatusAt(tl.Records, out.EvaluatedBusinessDate)
			date := in.NewEffectiveDate
			if in.Action == "DELETE" {
				date = candidate.Deleted[0].Date
			}
			if r := domain.FactAt(candidate.Records, date); r != nil {
				v := PublicRecord(*r, read.ReadFields)
				out.AfterRecord = &v
			}
		}
		return nil
	})
	return
}

// SafeDiagnostics removes locations whose corresponding business field is no longer readable.
func SafeDiagnostics(details *org.FailureDetails, fields []string) {
	if details == nil {
		return
	}
	for i := range details.Issues {
		details.Issues[i].Locations = slices.DeleteFunc(details.Issues[i].Locations, func(l org.Location) bool {
			field := strings.TrimPrefix(strings.TrimPrefix(l.FieldPath, "changes."), "classification.")
			if field == "newEffectiveDate" || field == "anchorRecordId" {
				field = "effectiveDate"
			}
			return field != "" && !slices.Contains(fields, field)
		})
	}
}

func TrimResult(result *org.Result, fields []string) {
	trimJointResult(result, fields)
	if !slices.Contains(fields, "effectiveDate") {
		result.EffectiveDate = nil
		result.PreviousEffectiveDate = nil
	}
	if !slices.Contains(fields, "effectiveDate") || !slices.Contains(fields, "name") || !slices.Contains(fields, "nameEn") {
		result.Impact = nil
	}
	SafeDiagnostics(result.FailureDetails, fields)
}

func safeTimelineError(err error, fields []string) error {
	var fault *foundation.Fault
	if errors.As(err, &fault) && fault.Code == "M02_TIMELINE_INVALID" && !slices.Contains(fields, "effectiveDate") {
		copy := *fault
		copy.Message = "当前修改无法保存"
		copy.Field = ""
		return &copy
	}
	return err
}

func prepareMaintenance(ctx context.Context, tl Timeline, enterpriseStart string, cmd maintenanceCommand, item domain.Item) ([]domain.Item, domain.Candidate, error) {
	defer foundation.BeginStage(ctx, "bu.candidate")()
	var e error
	if cmd.Action == "ADD" {
		if e := domain.ValidateAdditionPosition(tl.Records, cmd.Add.AnchorRecordID, enterpriseStart, item.Date); e != nil {
			return nil, domain.Candidate{}, e
		}
	}
	if cmd.Action == "DEACTIVATE" || cmd.Action == "REACTIVATE" {
		if cmd.Action == "REACTIVATE" && item.Changes != nil && item.Changes.Empty() {
			return nil, domain.Candidate{}, &foundation.Fault{Status: 400, Code: "M02_INVALID_INPUT", Field: "changes", Message: "明确修改时至少提供一个名称字段"}
		}
		if e := domain.ValidateLifecycleAppend(tl.Records, cmd.Add.AnchorRecordID, cmd.Action, enterpriseStart, item.Date); e != nil {
			return nil, domain.Candidate{}, e
		}
	}
	if (cmd.Action == "CORRECT" || cmd.Action == "REDATE") && item.Date != "" {
		if e := domain.ValidateCorrectionDate(tl.Records, cmd.RecordID, enterpriseStart, item.Date); e != nil {
			return nil, domain.Candidate{}, e
		}
	}
	items := []domain.Item{item}
	if cmd.Action == "REVISE_TIMELINE" {
		items, e = preparePair(ctx, tl, enterpriseStart, cmd)
		if e != nil {
			return nil, domain.Candidate{}, e
		}
	}
	candidate, e := domain.Calculate(ctx, tl.Records, tl.First, enterpriseStart, items)
	if e != nil {
		return nil, domain.Candidate{}, e
	}
	return items, candidate, nil
}

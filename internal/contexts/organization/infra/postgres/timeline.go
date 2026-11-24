package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"strings"
	"time"

	"chnhcm.local/chnhcm/internal/contexts/organization/application"
	departmentapp "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	"chnhcm.local/chnhcm/internal/contexts/organization/domain"
	dept "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history/model"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/compact"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func optional(s string) any {
	if s == "" {
		return nil
	}
	return s
}
func resultUnavailable() error {
	return foundation.Fail(503, "M02_RESULT_UNAVAILABLE", "暂时无法查看保存结果")
}
func (u *unit) LoadTimeline(ctx context.Context, id string, lock bool) (application.Timeline, error) {
	defer foundation.BeginStage(ctx, "bu.read_timeline")()
	if lock {
		if _, e := u.lockDepartmentDependencies(ctx, departmentapp.LockRequest{}, id, false); e != nil {
			return application.Timeline{}, e
		}
		var locked string
		if e := u.tx.QueryRow(ctx, `SELECT bu_id::text FROM organization.business_unit WHERE instance_id=$1 AND bu_id=$2 FOR UPDATE`, u.instance, id).Scan(&locked); e != nil {
			if errors.Is(e, pgx.ErrNoRows) {
				return application.Timeline{}, foundation.Fail(404, "M02_OBJECT_NOT_FOUND", "业务单元不存在或当前不可见")
			}
			return application.Timeline{}, e
		}
	}
	v, e := u.readBU(ctx, id, "")
	if errors.Is(e, pgx.ErrNoRows) {
		return application.Timeline{}, foundation.Fail(404, "M02_OBJECT_NOT_FOUND", "业务单元不存在或当前不可见")
	}
	if e != nil {
		return application.Timeline{}, e
	}
	return v.State.Timeline, nil
}
func (u *unit) StoreTerminal(ctx context.Context, t application.Terminal) error {
	if t.Outcome == "accepted" {
		return errors.New("accepted terminal belongs to its atomic source")
	}
	ent, e := u.Enterprise(ctx, false)
	if e != nil {
		return e
	}
	if t.Action == "REVISE_TIMELINE" && t.ItemOutcomes == nil {
		t.ItemOutcomes = []org.ItemResult{}
	}
	if t.Fields == nil && t.Outcome != "not_applied" {
		t.Fields = []string{}
	}
	if t.Outcome == "rejected" {
		t.Result = org.Result{SubmissionID: t.ID, Outcome: t.Outcome, ErrorCode: depPtr(t.ErrorCode), FailureDetails: t.Failure}
	}
	if t.Outcome == "not_applied" {
		t.Result = org.Result{SubmissionID: t.ID, Outcome: t.Outcome, ErrorCode: depPtr(t.ErrorCode)}
	}
	if e = validateTerminalHeader(t.ID, t.Outcome, t.Action, t.Digest, t.Fields, t.EvaluatedAt, t.Failure, t.ErrorCode, false); e != nil {
		return e
	}
	t.Result.Action = t.Action
	t.Result.ExpectedSourceVersion = depPtr(t.ExpectedSource)
	setEvaluated(&t.Result, t.EvaluatedAt)
	return history.SaveTerminal(ctx, u.tx, history.BusinessUnit, u.instance, ent.ID, u.guard.Principal().ID, t.ID, history.TerminalEnvelope{Object: t.BUID, Outcome: t.Outcome}, func(p *compact.Pool) ([]byte, error) {
		return history.EncodeBUOperation(ctx, p, model.BUOperation{Terminal: t})
	})
}
func (u *unit) Persist(ctx context.Context, w application.Write) (org.Result, error) {
	defer foundation.BeginStage(ctx, "bu.persist")()
	if w.Create == nil && u.stableBUCoverage != nil {
		locked := u.stableBUCoverage
		if locked.BUID != w.Timeline.BUID || locked.Source != w.Timeline.Source || !domain.SameBUCoverage(locked.Records, w.Candidate.Records) {
			return org.Result{}, errors.New("BU coverage changed outside locked dependency plan")
		}
	}
	if w.Create == nil && u.stableBUCoverage == nil {
		if e := protectLegalPurposes(ctx, u, "bu", w.Timeline.BUID, w.Timeline.Records, w.Candidate.Records); e != nil {
			return org.Result{}, e
		}
		rows := make([]dept.BURecord, 0, len(w.Candidate.Records))
		for _, r := range w.Candidate.Records {
			rows = append(rows, dept.BURecord{ID: r.ID, Revision: r.Revision, Date: r.Date, End: r.End, State: r.State, Name: r.Data.Name})
		}
		if e := u.validateBUReferences(ctx, w.Timeline.BUID, rows); e != nil {
			return org.Result{}, e
		}
	}
	operator, identityErr := u.guard.OperatorSnapshot(ctx)
	if identityErr != nil {
		return org.Result{}, identityErr
	}
	if operator.SubjectID != u.guard.Principal().ID || operator.Login == nil || operator.IdentityEvidence != "captured" || (operator.Kind != "human" && operator.Kind != "service") {
		return org.Result{}, foundation.Fail(503, "M02_DEPENDENCY_UNAVAILABLE", "操作身份暂不可用")
	}
	tl := w.Timeline
	t := w.Terminal
	if t.Fields == nil {
		t.Fields = []string{}
	}
	source := foundation.NewID()
	if w.Create != nil {
		in := w.Create
		tl.BUID = foundation.NewID()
		tl.First = foundation.NewID()
		ent, e := u.Enterprise(ctx, false)
		if e != nil {
			return org.Result{}, e
		}
		_, e = u.tx.Exec(ctx, `INSERT INTO organization.business_unit(instance_id,bu_id,enterprise_id,code,normalized_code,first_record_id,head_source_id,head_no,head_chunk) VALUES($1,$2,$3,$4,$5,$6,$7,1,1)`, u.instance, tl.BUID, ent.ID, in.Code, strings.ToLower(in.Code), tl.First, source)
		if e != nil {
			var p *pgconn.PgError
			if errors.As(e, &p) && p.Code == "23505" {
				return org.Result{}, foundation.Fail(409, "M02_CODE_CONFLICT", "编码已被使用")
			}
			return org.Result{}, e
		}
		w.Candidate.Records[0].ID = tl.First
	}
	// Allocate identities for accepted ADDs only, retaining itemId as protocol location.
	newIDs := map[string]string{}
	old := map[string]domain.Record{}
	for _, r := range tl.Records {
		old[r.ID] = r
	}
	for i := range w.Candidate.Records {
		r := &w.Candidate.Records[i]
		if _, exists := old[r.ID]; !exists && w.Create == nil {
			original := r.ID
			r.ID = foundation.NewID()
			newIDs[original] = r.ID
		}
	}
	for i := range w.Items {
		if id, ok := newIDs[domain.CandidateID(w.Items[i])]; ok {
			w.Items[i].RecordID = id
		}
	}
	for i := range t.ItemOutcomes {
		if id, ok := newIDs[t.ItemOutcomes[i].RecordID]; ok {
			t.ItemOutcomes[i].RecordID = id
		}
	}
	recorded, err := u.recordedNow(ctx)
	if err != nil {
		return org.Result{}, err
	}
	var e error
	target := ""
	revisions := []revisionInsert{}
	for i := range w.Candidate.Records {
		if e = ctx.Err(); e != nil {
			return org.Result{}, e
		}
		r := &w.Candidate.Records[i]
		b, exists := old[r.ID]
		if r.Type == "DEACTIVATE" && i > 0 {
			r.InheritedRecord = w.Candidate.Records[i-1].ID
			r.InheritedRevision = w.Candidate.Records[i-1].Revision
		}
		aCopy, bCopy := *r, b
		aCopy.Revision = ""
		bCopy.Revision = ""
		aCopy.RevisionNo = 0
		bCopy.RevisionNo = 0
		if !exists || !reflect.DeepEqual(aCopy, bCopy) {
			r.Revision = foundation.NewID()
			r.RevisionNo = b.RevisionNo + 1
			var previous *string
			if b.Revision != "" {
				v := b.Revision
				previous = &v
			}
			revisions = append(revisions, revisionInsert{Record: *r, PreviousRevision: previous})
		}
		if t.Action == "CREATE" || slices.Contains([]string{"ADD", "DEACTIVATE", "REACTIVATE"}, t.Action) && !exists || slices.Contains([]string{"CORRECT", "REDATE"}, t.Action) && r.ID == t.RecordID {
			target = r.ID
		}
	}
	for i := range w.Candidate.Deleted {
		r := &w.Candidate.Deleted[i]
		b := old[r.ID]
		r.Revision = foundation.NewID()
		r.RevisionNo = b.RevisionNo + 1
		previous := b.Revision
		revisions = append(revisions, revisionInsert{Record: *r, PreviousRevision: &previous, Deleted: true})
		target = r.ID
	}
	var impactItems []domain.Item
	if t.Action == "REVISE_TIMELINE" {
		impactItems = w.Items
	}
	impact := domain.Compare(ctx, tl.Records, w.Candidate.Records, impactItems)
	if e = ctx.Err(); e != nil {
		return org.Result{}, e
	}
	resultRecords := make(map[string]domain.Record, len(w.Candidate.Records)+len(w.Candidate.Deleted))
	for _, records := range [][]domain.Record{w.Candidate.Records, w.Candidate.Deleted} {
		for _, record := range records {
			resultRecords[record.ID] = record
		}
	}
	for i := range impact.StructuralChanges {
		if record, ok := resultRecords[impact.StructuralChanges[i].RecordID]; ok {
			impact.StructuralChanges[i].AfterRevision = &record.Revision
		}
	}
	r := org.Result{SubmissionID: t.ID, Outcome: "accepted", Action: t.Action, BUID: &tl.BUID, SourceVersion: &source, SelectedRecordID: &target, TargetRecordID: &target, Impact: nil}
	if record, ok := resultRecords[target]; ok {
		r.TargetRevision = &record.Revision
		r.EffectiveDate = &record.Date
	}
	if t.Action == "REDATE" || t.Action == "CORRECT" && old[target].Date != *r.EffectiveDate {
		previousDate := old[target].Date
		r.PreviousEffectiveDate = &previousDate
	}
	if t.Action == "DELETE" && r.EffectiveDate != nil {
		if selected := domain.FactAt(w.Candidate.Records, *r.EffectiveDate); selected != nil {
			r.SelectedRecordID = &selected.ID
		}
	}
	if t.Action == "REVISE_TIMELINE" {
		r.TargetRecordID = nil
		r.TargetRevision = nil
		r.SelectedRecordID = nil
		r.EffectiveDate = nil
		if e = u.persistJointItems(ctx, tl, source, &t, w.Candidate); e != nil {
			return r, e
		}
		r.Items = t.ItemOutcomes
	}
	if r.TargetRevision == nil && t.Action != "REVISE_TIMELINE" {
		return r, errors.New("direct result revision missing")
	}
	setEvaluated(&r, t.EvaluatedAt)
	stamp := recorded.UTC().Format(time.RFC3339Nano)
	r.RecordedAt = &stamp
	r.CurrentStatus = domain.StatusAt(w.Candidate.Records, *r.EvaluatedBusinessDate)
	publicImpact := application.PublicImpact(impact)
	r.Impact = &publicImpact
	t.Outcome = "accepted"
	t.Result = r
	t.Result.ExpectedSourceVersion = depPtr(t.ExpectedSource)
	ent, e := u.Enterprise(ctx, false)
	if e != nil {
		return r, e
	}
	next := model.BUState{Timeline: application.Timeline{BUID: tl.BUID, First: tl.First, Source: source, SourceNo: tl.SourceNo + 1, Records: w.Candidate.Records}, Summaries: []org.OperationSummary{}}
	summaries := map[string]org.OperationSummary{}
	last := history.Chunk{Kind: history.BusinessUnit, Instance: u.instance, Enterprise: ent.ID, Object: tl.BUID, Number: 1, First: 1}
	if tl.Source != "" {
		prior, e := u.readBU(ctx, tl.BUID, tl.Source)
		if e != nil {
			return r, e
		}
		last = prior.Chunk
		for _, s := range prior.State.Summaries {
			summaries[s.Revision] = s
		}
	}
	identity := org.OperatorSnapshot{SubjectID: operator.SubjectID, DisplayName: operator.DisplayName, Login: operator.Login, Kind: operator.Kind, IdentityEvidence: operator.IdentityEvidence}
	for _, v := range w.Candidate.Records {
		summary, exists := summaries[v.Revision]
		if !exists {
			summary = org.OperationSummary{RecordID: v.ID, Revision: v.Revision, OperationSourceVersion: source, RecordedAt: stamp, Operator: identity}
		}
		next.Summaries = append(next.Summaries, summary)
	}
	op := model.BUOperation{Source: model.Source{ID: source, Previous: tl.Source, Number: tl.SourceNo + 1, Subject: operator.SubjectID, Submission: t.ID, Action: t.Action, RecordedAt: recorded, Operator: identity}, Terminal: t, EventID: foundation.NewID(), Impact: publicImpact}
	for _, v := range revisions {
		rev := model.BURevision{Record: v.Record, Deleted: v.Deleted}
		if v.PreviousRevision != nil {
			rev.Previous = *v.PreviousRevision
			before := old[v.ID]
			rev.Before = &before
		}
		op.Revisions = append(op.Revisions, rev)
	}
	if e = validateTerminalHeader(t.ID, t.Outcome, t.Action, t.Digest, t.Fields, t.EvaluatedAt, t.Failure, t.ErrorCode, false); e != nil {
		return r, e
	}
	if e = u.tx.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&op.TerminalClosedAt); e != nil {
		return r, e
	}
	if e = validateBUFacts(next, op); e != nil {
		return r, e
	}
	chunk, e := history.Append(ctx, u.tx, last, tl.SourceNo+1, source, operator.SubjectID, t.ID, func(p *compact.Pool) ([]byte, []byte, []byte, error) {
		a, e := history.EncodeBUState(ctx, p, next)
		if e != nil {
			return nil, nil, nil, e
		}
		b, e := history.EncodeBUOperation(ctx, p, op)
		return a, b, []byte{0}, e
	})
	if e != nil {
		return r, e
	}
	if w.Create == nil {
		tag, e := u.tx.Exec(ctx, `UPDATE organization.business_unit SET head_source_id=$3,head_no=$5,head_chunk=$6 WHERE instance_id=$1 AND bu_id=$2 AND head_source_id=$4`, u.instance, tl.BUID, source, tl.Source, tl.SourceNo+1, chunk.Number)
		if e != nil {
			return r, e
		}
		if tag.RowsAffected() != 1 {
			return r, foundation.Fail(409, "M02_SOURCE_CONFLICT", "资料已更新，请核对后保存")
		}
	}
	if e = history.SaveProjection(ctx, u.tx, chunk, source, func(p *compact.Pool) ([]byte, error) { return history.EncodeBUState(ctx, p, next) }); e != nil {
		return r, e
	}
	if t.Action == "REVISE_TIMELINE" {
		raw, _ := json.Marshal(t.Items)
		if e = u.hydrateJointResult(ctx, &r, raw, &tl.Source); e != nil {
			return r, e
		}
	}
	setEvaluated(&r, t.EvaluatedAt)
	r.CurrentStatus = domain.StatusAt(w.Candidate.Records, *r.EvaluatedBusinessDate)
	v := application.PublicImpact(impact)
	r.Impact = &v
	return r, nil
}

func setEvaluated(r *org.Result, t *time.Time) {
	if t == nil {
		return
	}
	a := t.UTC().Format(time.RFC3339Nano)
	d := t.In(foundation.Shanghai).Format("2006-01-02")
	r.EvaluatedAt = &a
	r.EvaluatedBusinessDate = &d
}
func (u *unit) FindResult(ctx context.Context, id string) (*application.Saved, error) {
	subject := u.guard.Principal().ID
	found, e := history.FindSubmission(ctx, u.tx, history.BusinessUnit, u.instance, subject, id)
	if e != nil {
		return nil, e
	}
	var t application.Terminal
	if found != nil {
		c, e := history.LoadChunk(ctx, u.tx, history.BusinessUnit, u.instance, found.Object, found.Chunk)
		if e != nil {
			return &application.Saved{RequestObjectID: found.Object, Result: org.Result{SubmissionID: id, Outcome: "accepted"}, DecodeError: resultUnavailable()}, nil
		}
		v, e := u.readBU(ctx, found.Object, c.ID(c.Sources[found.Slot]))
		if e != nil {
			return &application.Saved{RequestObjectID: found.Object, Result: org.Result{SubmissionID: id, Outcome: "accepted"}, DecodeError: resultUnavailable()}, nil
		}
		t = v.Operation.Terminal
	} else {
		meta, e := history.TerminalIdentity(ctx, u.tx, history.BusinessUnit, u.instance, subject, id)
		if errors.Is(e, pgx.ErrNoRows) {
			return nil, nil
		}
		if e != nil {
			return nil, e
		}
		damaged := func() (*application.Saved, error) {
			return &application.Saved{RequestObjectID: meta.Object, Result: org.Result{SubmissionID: id, Outcome: meta.Outcome}, DecodeError: resultUnavailable()}, nil
		}
		pool, raw, e := history.LoadTerminal(ctx, u.tx, history.BusinessUnit, u.instance, subject, id)
		if e != nil {
			return damaged()
		}
		op, e := history.DecodeBUOperation(ctx, pool, raw)
		if e != nil {
			return damaged()
		}
		if op.Terminal.BUID != meta.Object || op.Terminal.Outcome != meta.Outcome {
			return damaged()
		}
		t = op.Terminal
	}
	out := &application.Saved{Result: t.Result, Digest: t.Digest, RequestObjectID: t.BUID}
	unavailable := func() (*application.Saved, error) { out.DecodeError = resultUnavailable(); return out, nil }
	if t.ID != id || t.Outcome != out.Result.Outcome || t.Result.SubmissionID != id {
		return unavailable()
	}
	if validateTerminalHeader(t.ID, t.Outcome, t.Action, t.Digest, t.Fields, t.EvaluatedAt, t.Failure, t.ErrorCode, false) != nil {
		return unavailable()
	}
	if t.Outcome == "no_change" {
		out.Result.BUID = depPtr(t.BUID)
		if t.Action == "ADD" {
			if t.Result.SelectedRecordID == nil || !foundation.ValidID(*t.Result.SelectedRecordID) || t.Result.ObservedSourceVersion == nil {
				return unavailable()
			}
			v, err := u.readBU(ctx, t.BUID, *t.Result.ObservedSourceVersion)
			if err != nil {
				return unavailable()
			}
			found := false
			for _, r := range v.State.Timeline.Records {
				found = found || r.ID == *t.Result.SelectedRecordID
			}
			if !found {
				return unavailable()
			}
		} else {
			out.Result.SelectedRecordID = depPtr(t.RecordID)
		}
	}
	if t.Outcome == "rejected" {
		for _, issue := range t.Failure.Issues {
			for _, loc := range issue.Locations {
				allowed := []string{"", "code", "name", "nameEn", "effectiveDate", "operationType", "operationReason"}
				if t.Action != "CREATE" {
					allowed = []string{"", "changes", "changes.name", "changes.nameEn", "effectiveDate", "operationType", "operationReason"}
				}
				if slices.Contains([]string{"ADD", "DEACTIVATE", "REACTIVATE"}, t.Action) {
					allowed = append(allowed, "anchorRecordId")
				}
				if t.Action == "REVISE_TIMELINE" || t.Action == "CORRECT" {
					allowed = append(allowed, "newEffectiveDate", "classification.operationType", "classification.operationReason")
				}
				if t.Action == "REDATE" {
					allowed = append(allowed, "newEffectiveDate")
				}
				if !slices.Contains(allowed, loc.FieldPath) || t.Action != "REVISE_TIMELINE" && loc.ItemID != "" {
					return unavailable()
				}
			}
		}
		if t.Action == "REVISE_TIMELINE" && !validJointDiagnostics(*t.Failure, t.Items) {
			return unavailable()
		}
	}

	if t.Action == "REVISE_TIMELINE" {
		req, _ := json.Marshal(t.Items)
		outcomes, _ := json.Marshal(t.ItemOutcomes)
		items, e := decodeJointTerminal(req, outcomes, t.Outcome)
		if e != nil {
			return unavailable()
		}
		out.Result.Items = items
		if t.Outcome == "accepted" || t.Outcome == "no_change" {
			if e = u.hydrateJointResult(ctx, &out.Result, req, depPtr(t.ExpectedSource)); e != nil {
				return unavailable()
			}
		}
	}
	if out.Result.BUID != nil {
		source := out.Result.SourceVersion
		if source == nil {
			source = out.Result.ObservedSourceVersion
		}
		if source != nil {
			h, e := u.head(ctx, history.BusinessUnit, *out.Result.BUID)
			if e != nil {
				return nil, e
			}
			out.Result.CurrentChanged = h.Source != *source
			if t.EvaluatedAt != nil {
				v, e := u.readBU(ctx, *out.Result.BUID, *source)
				if e != nil {
					return unavailable()
				}
				out.Result.CurrentStatus = domain.StatusAt(v.State.Timeline.Records, t.EvaluatedAt.In(foundation.Shanghai).Format("2006-01-02"))
			}
		}
	}
	return out, nil
}

package postgres

import (
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history/model"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/compact"
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"reflect"
	"slices"
	"strings"
	"time"
)

func depPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
func value(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
func departmentRange(start, end string) string { return "[" + start + "," + end + ")" }
func (u *departmentUnit) Persist(ctx context.Context, w app.Write) (org.DepartmentResult, error) {
	defer foundation.BeginStage(ctx, "department.persist")()
	u.publication = nil
	if !w.Validated() {
		return org.DepartmentResult{}, errors.New("department write is not authorized and finalized")
	}
	if original := u.stableDepartmentRelations; original != nil && (original.ID != w.Before.ID || original.Source != w.Before.Source || w.RelationChanged || d.Plan(*original, w.Candidate.Records).Changed) {
		return org.DepartmentResult{}, errors.New("department candidate expanded its locked relation envelope")
	}
	if w.Operator.SubjectID != u.guard.Principal().ID || w.After.Enterprise == "" || w.After.SourceNo != w.Before.SourceNo+1 || w.Terminal.ID != w.Terminal.Result.SubmissionID || w.Terminal.Result.SourceVersion == nil || *w.Terminal.Result.SourceVersion != w.After.Source {
		return org.DepartmentResult{}, compact.ErrCorrupt
	}
	if err := checkPositionUses(ctx, u.unit, w.After.Enterprise, "departmentId", w.After.ID, w.Before.Records, w.After.Records); err != nil {
		return org.DepartmentResult{}, err
	}
	if w.Before.Source == "" {
		_, e := u.tx.Exec(ctx, `INSERT INTO organization.department(instance_id,enterprise_id,department_id,code,normalized_code,first_record_id,head_source_id,head_no,head_chunk) VALUES($1,$2,$3,$4,$5,$6,$7,1,1)`, u.instance, w.After.Enterprise, w.After.ID, w.After.Code, strings.ToLower(w.After.Code), w.After.First, w.After.Source)
		if e != nil {
			var pg *pgconn.PgError
			if errors.As(e, &pg) && pg.Code == "23505" {
				return org.DepartmentResult{}, foundation.Fail(409, "M02_CODE_CONFLICT", "编码已被使用")
			}
			return org.DepartmentResult{}, e
		}
	}
	previousSummaries := map[string]org.OperationSummary{}
	last := history.Chunk{Kind: history.Department, Instance: u.instance, Enterprise: w.After.Enterprise, Object: w.After.ID, Number: 1, First: 1}
	if w.Before.Source != "" {
		old, e := u.readDepartment(ctx, w.After.ID, w.Before.Source, false)
		if e != nil {
			return org.DepartmentResult{}, e
		}
		last = old.Chunk
		for _, s := range old.State.Summaries {
			previousSummaries[s.Revision] = s
		}
	}
	operator := org.OperatorSnapshot{SubjectID: w.Operator.SubjectID, DisplayName: w.Operator.DisplayName, Login: w.Operator.Login, Kind: w.Operator.Kind, IdentityEvidence: w.Operator.IdentityEvidence}
	summary := func(r d.Record) org.OperationSummary {
		if s, ok := previousSummaries[r.Revision]; ok {
			return s
		}
		return org.OperationSummary{RecordID: r.ID, Revision: r.Revision, OperationSourceVersion: w.After.Source, RecordedAt: w.After.RecordedAt, Operator: operator}
	}
	next := model.DepartmentState{Timeline: w.After, Summaries: []org.OperationSummary{}}
	next.Timeline.Code = ""
	for _, r := range w.After.Records {
		next.Summaries = append(next.Summaries, summary(r))
	}
	at, e := time.Parse(time.RFC3339Nano, w.After.RecordedAt)
	if e != nil {
		return org.DepartmentResult{}, e
	}
	hs := app.HistorySource{ID: w.After.Source, Previous: w.Before.Source, At: w.After.RecordedAt, Action: w.Terminal.Action, No: w.After.SourceNo, Operator: operator, Items: []app.HistoryItem{}}
	for _, c := range w.Changes {
		item := app.HistoryItem{RecordID: c.RecordID, Role: c.Role, Action: c.Action}
		if c.BeforeRevision != "" {
			item.BeforeSource = w.Before.Source
			for _, r := range w.Before.Records {
				if r.ID == c.RecordID && r.Revision == c.BeforeRevision {
					item.Before = &app.HistoryRevision{Record: r, Summary: summary(r)}
				}
			}
			if item.Before == nil {
				return org.DepartmentResult{}, compact.ErrCorrupt
			}
		}
		for _, r := range w.NewRecords {
			if r.ID == c.RecordID && r.Revision == c.AfterRevision {
				item.After = &app.HistoryRevision{Record: r, Summary: summary(r)}
			}
		}
		for _, r := range w.DeletedRecords {
			if r.ID == c.RecordID && r.Revision == c.AfterRevision {
				item.After = &app.HistoryRevision{Record: r, Deleted: true, Summary: summary(r)}
			}
		}
		if item.After == nil {
			return org.DepartmentResult{}, compact.ErrCorrupt
		}
		hs.Items = append(hs.Items, item)
	}
	slices.SortFunc(hs.Items, func(a, b app.HistoryItem) int { return strings.Compare(a.RecordID, b.RecordID) })
	op := model.DepartmentOperation{Source: model.Source{ID: w.After.Source, Previous: w.Before.Source, Subject: w.Operator.SubjectID, Submission: w.Terminal.ID, Action: w.Terminal.Action, Number: w.After.SourceNo, RecordedAt: at, Operator: operator}, History: hs, Terminal: w.Terminal, EventID: foundation.NewID(), RelationPrevious: w.Before.RelationSource, Changes: w.Changes, EdgeChanges: w.EdgeChanges, NewEdges: w.NewEdges, DeletedEdges: w.DeletedEdges, DeletedRecords: w.DeletedRecords}
	// Persist finalized member references, never the planner's temporary token.
	for i := range op.NewEdges {
		for _, edge := range w.After.Edges {
			if edge.ID == op.NewEdges[i].ID && edge.Revision == op.NewEdges[i].Revision {
				op.NewEdges[i] = edge
				break
			}
		}
	}
	if w.RelationChanged {
		op.RelationEventID = foundation.NewID()
	}
	evidence := model.DepartmentEvidence{Segments: w.Evidence}
	t := w.Terminal
	if e = validateTerminalHeader(t.ID, t.Outcome, t.Action, t.Digest, t.Fields, t.EvaluatedAt, t.Failure, t.ErrorCode, true); e != nil {
		return org.DepartmentResult{}, e
	}
	if e = u.tx.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&op.TerminalClosedAt); e != nil {
		return org.DepartmentResult{}, e
	}
	if e = validateDepartmentFacts(next, op, evidence); e != nil {
		return org.DepartmentResult{}, e
	}
	chunk, e := history.Append(ctx, u.tx, last, w.After.SourceNo, w.After.Source, w.Operator.SubjectID, w.Terminal.ID, func(p *compact.Pool) ([]byte, []byte, []byte, error) {
		a, e := history.EncodeDepartmentState(ctx, p, next)
		if e != nil {
			return nil, nil, nil, e
		}
		b, e := history.EncodeDepartmentOperation(ctx, p, op)
		if e != nil {
			return nil, nil, nil, e
		}
		c, e := history.EncodeDepartmentEvidence(ctx, p, evidence)
		return a, b, c, e
	})
	if e != nil {
		return org.DepartmentResult{}, e
	}
	if w.RelationChanged {
		if w.Before.RelationSource == "" {
			_, e = u.tx.Exec(ctx, `INSERT INTO organization.department_hierarchy_relation VALUES($1,$2,$3,$4)`, u.instance, w.After.Enterprise, w.After.ID, w.After.RelationSource)
		} else {
			_, e = u.tx.Exec(ctx, `UPDATE organization.department_hierarchy_relation SET head_source_id=$4 WHERE instance_id=$1 AND enterprise_id=$2 AND department_id=$3 AND head_source_id=$5`, u.instance, w.After.Enterprise, w.After.ID, w.After.RelationSource, w.Before.RelationSource)
		}
		if e != nil {
			return org.DepartmentResult{}, e
		}
	}
	bus, parents := []string{}, []string{}
	for _, r := range w.After.Records {
		bus = append(bus, r.Data.BU)
		if r.Data.Parent != "" {
			parents = append(parents, r.Data.Parent)
		}
	}
	for _, edge := range w.After.Edges {
		bus = append(bus, edge.BU)
		if edge.Parent != "" {
			parents = append(parents, edge.Parent)
		}
	}
	slices.Sort(bus)
	bus = slices.Compact(bus)
	slices.Sort(parents)
	parents = slices.Compact(parents)
	expected := w.Before.Source
	if expected == "" {
		expected = w.After.Source // The new identity was inserted in this transaction.
	}
	if e = history.PublishProjection(ctx, u.tx, chunk, w.After.Source, func(p *compact.Pool) ([]byte, error) { return history.EncodeDepartmentState(ctx, p, next) }, func(p history.Projection) error {
		tag, err := u.tx.Exec(ctx, `UPDATE organization.department SET head_source_id=$4,head_no=$5,head_chunk=$6,management_bus=$8::text[]::uuid[],parents=$9::text[]::uuid[],current_shape=$10,current_refs=$11,current_envelope=$12 WHERE instance_id=$1 AND enterprise_id=$2 AND department_id=$3 AND head_source_id=$7`, u.instance, w.After.Enterprise, w.After.ID, w.After.Source, w.After.SourceNo, chunk.Number, expected, bus, parents, p.Shape, p.References, p.Envelope)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return foundation.Fail(409, "M02_SOURCE_CONFLICT", "资料已更新，请核对后保存")
		}
		return nil
	}); e != nil {
		return org.DepartmentResult{}, e
	}
	// Only this exact newly published source can reuse the already validated
	// write facts and locked dependencies for response projection. The policy
	// guard and dependency locks remain held through COMMIT; a failed savepoint
	// discards this value. Independent reads/recovery always load the database.
	if u.departmentFacts != nil && len(w.Terminal.Result.Items) > 0 {
		after := w.After
		read := app.ReadSnapshot{Snapshot: u.departmentFacts.Snapshot.With(after), Selected: &after, Evidence: w.Evidence, Summaries: map[string]org.OperationSummary{}}
		for _, summary := range next.Summaries {
			read.Summaries[summary.Revision] = summary
		}
		u.publication = &departmentPublication{id: after.ID, source: after.Source, read: read, history: hs}
	}
	return w.Terminal.Result, nil
}
func validateDepartmentState(s model.DepartmentState) error {
	t := s.Timeline
	if len(t.Records) == 0 || t.Records[0].ID != t.First || t.Records[0].Type != "CREATE" || len(s.Summaries) != len(t.Records) || !foundation.ValidID(t.RelationSource) || t.RelationNo < 1 {
		return compact.ErrCorrupt
	}
	summaries := summaryIndex(s.Summaries)
	seen := map[string]bool{}
	for i, r := range t.Records {
		m := r.Data.Member
		if seen[r.ID] || !foundation.ValidID(r.ID) || !foundation.ValidID(r.Revision) || r.RevisionNo < 1 || !validSpan(r.Date, r.End) || !summaries[[2]string{r.ID, r.Revision}] || !foundation.ValidID(r.Data.BU) || r.Data.Parent == t.ID || m.DepartmentID != t.ID || m.BU != r.Data.BU || m.Parent != r.Data.Parent || !foundation.ValidID(m.Source) || !foundation.ValidID(m.MemberID) || !foundation.ValidID(m.EdgeID) || !foundation.ValidID(m.Revision) || !validSpan(m.Start, m.End) {
			return compact.ErrCorrupt
		}
		seen[r.ID] = true
		if i+1 < len(t.Records) && r.End != t.Records[i+1].Date || i+1 == len(t.Records) && r.End != "" {
			return compact.ErrCorrupt
		}
		if r.State == "inactive" {
			if i == 0 || r.Type != "DEACTIVATE" {
				return compact.ErrCorrupt
			}
			prev := t.Records[i-1]
			if prev.State != "active" || r.InheritedRecord != prev.ID || r.InheritedRevision != prev.Revision || !reflect.DeepEqual(r.Data, prev.Data) {
				return compact.ErrCorrupt
			}
		} else if r.State != "active" || r.Type == "DEACTIVATE" || r.InheritedRecord != "" || r.InheritedRevision != "" {
			return compact.ErrCorrupt
		}
	}
	edges := map[string]bool{}
	for i, e := range t.Edges {
		if edges[e.ID] || !foundation.ValidID(e.ID) || !foundation.ValidID(e.Revision) || !foundation.ValidID(e.Source) || e.RevisionNo < 1 || !validSpan(e.Start, e.End) || e.Ref.DepartmentID != t.ID || e.Ref.Source != t.RelationSource || e.Ref.EdgeID != e.ID || e.Ref.Revision != e.Revision || e.Ref.BU != e.BU || e.Ref.Parent != e.Parent || e.Ref.Start != e.Start || e.Ref.End != e.End || !foundation.ValidID(e.Ref.MemberID) {
			return compact.ErrCorrupt
		}
		edges[e.ID] = true
		if i > 0 && (t.Edges[i-1].End == "" || t.Edges[i-1].End > e.Start) {
			return compact.ErrCorrupt
		}
	}
	return nil
}
func validateDepartmentOperation(s model.DepartmentState, op model.DepartmentOperation) error {
	if err := validateDepartmentState(s); err != nil {
		return err
	}
	if !foundation.ValidID(op.EventID) || op.Source.Subject != op.Source.Operator.SubjectID || op.Source.Action != op.Terminal.Action || op.Source.Number != s.Timeline.SourceNo || op.Source.RecordedAt.IsZero() || op.TerminalClosedAt.IsZero() || op.History.ID != op.Source.ID || op.History.No != op.Source.Number || op.History.Previous != op.Source.Previous || op.History.Action != op.Source.Action || op.Terminal.ID != op.Source.Submission {
		return compact.ErrCorrupt
	}
	if (op.Source.Number == 1) != (op.Source.Previous == "") || op.Source.Previous == op.Source.ID {
		return compact.ErrCorrupt
	}
	if op.RelationEventID != "" && !foundation.ValidID(op.RelationEventID) {
		return compact.ErrCorrupt
	}

	t := s.Timeline
	if len(t.Records) == 0 || t.Records[0].ID != t.First || t.Source != op.Source.ID || op.Terminal.Outcome != "accepted" || value(op.Terminal.Result.DepartmentID) != t.ID || value(op.Terminal.Result.SourceVersion) != t.Source {
		return compact.ErrCorrupt
	}
	if !reflect.DeepEqual(op.Source.Operator, op.History.Operator) || op.History.At != t.RecordedAt || op.History.At != op.Source.RecordedAt.UTC().Format(time.RFC3339Nano) || len(op.History.Items) == 0 || len(op.Changes) != len(op.History.Items) {
		return compact.ErrCorrupt
	}
	if err := validateTerminalHeader(op.Terminal.ID, op.Terminal.Outcome, op.Terminal.Action, op.Terminal.Digest, op.Terminal.Fields, op.Terminal.EvaluatedAt, op.Terminal.Failure, op.Terminal.ErrorCode, true); err != nil {
		return err
	}
	if op.Terminal.Result.SubmissionID != op.Source.Submission || op.Terminal.Result.Action != op.Source.Action || value(op.Terminal.Result.RecordedAt) != t.RecordedAt {
		return compact.ErrCorrupt
	}
	members := map[string]d.Record{}
	for _, r := range t.Records {
		members[r.ID] = r
	}
	changes := map[string]app.Change{}
	for _, c := range op.Changes {
		if _, ok := changes[c.RecordID]; ok {
			return compact.ErrCorrupt
		}
		if !foundation.ValidID(c.RecordID) || !foundation.ValidID(c.AfterRevision) || (c.Role != "direct" && c.Role != "derived") || len(c.Facets) == 0 || !slices.IsSorted(c.Facets) {
			return compact.ErrCorrupt
		}
		for i, f := range c.Facets {
			if (i > 0 && c.Facets[i-1] == f) || !slices.Contains([]string{"ADD", "DELETE", "CORRECT", "REDATE", "RECLASSIFY", "PERIOD", "INHERITANCE", "REFERENCE"}, f) {
				return compact.ErrCorrupt
			}
		}
		if c.Action == "ADD" {
			if c.BeforeRevision != "" || !slices.Equal(c.Facets, []string{"ADD"}) {
				return compact.ErrCorrupt
			}
		} else if c.Action == "DELETE" {
			if !foundation.ValidID(c.BeforeRevision) || !slices.Equal(c.Facets, []string{"DELETE"}) {
				return compact.ErrCorrupt
			}
		} else if c.Action != "UPDATE" || !foundation.ValidID(c.BeforeRevision) || slices.Contains(c.Facets, "ADD") || slices.Contains(c.Facets, "DELETE") {
			return compact.ErrCorrupt
		}
		changes[c.RecordID] = c
	}
	direct := false
	seen := map[string]bool{}
	for _, it := range op.History.Items {
		c, ok := changes[it.RecordID]
		if !ok || seen[it.RecordID] || it.After == nil || c.Role != it.Role || c.Action != it.Action || it.After.Record.ID != it.RecordID || it.After.Record.Revision != c.AfterRevision {
			return compact.ErrCorrupt
		}
		seen[it.RecordID] = true
		after := it.After.Record
		if it.Before == nil {
			if c.BeforeRevision != "" || it.BeforeSource != "" || after.RevisionNo != 1 || it.Action != "ADD" {
				return compact.ErrCorrupt
			}
		} else {
			before := it.Before.Record
			if it.BeforeSource != op.Source.Previous || before.ID != it.RecordID || before.Revision != c.BeforeRevision || before.Revision == after.Revision || after.RevisionNo != before.RevisionNo+1 || it.Before.Deleted {
				return compact.ErrCorrupt
			}
		}
		member, live := members[it.RecordID]
		if live == it.After.Deleted || it.After.Deleted != (it.Action == "DELETE") || live && !reflect.DeepEqual(member, after) {
			return compact.ErrCorrupt
		}
		if !validSpan(after.Date, after.End) || it.After.Summary.RecordID != after.ID || it.After.Summary.Revision != after.Revision || it.After.Summary.OperationSourceVersion != op.Source.ID {
			return compact.ErrCorrupt
		}
		if it.Role == "direct" && value(op.Terminal.Result.TargetRecordID) == after.ID && value(op.Terminal.Result.TargetRevision) == after.Revision {
			direct = true
		}
	}
	if op.Source.Action != "REVISE_TIMELINE" && !direct {
		return compact.ErrCorrupt
	}
	return nil
}
func validateDepartmentFacts(s model.DepartmentState, op model.DepartmentOperation, e model.DepartmentEvidence) error {
	if err := validateDepartmentOperation(s, op); err != nil {
		return err
	}
	t := s.Timeline
	records := map[string]d.Record{}
	for i, r := range t.Records {
		if !foundation.ValidID(r.ID) || !foundation.ValidID(r.Revision) || r.RevisionNo < 1 || r.Data.Member.DepartmentID != t.ID || r.Data.Member.BU != r.Data.BU || !foundation.ValidID(r.Data.Member.Source) || !foundation.ValidID(r.Data.Member.MemberID) || r.Reason == "" || r.TypeLabel == "" || r.ReasonLabel == "" {
			return compact.ErrCorrupt
		}
		if _, exists := records[r.ID]; exists {
			return compact.ErrCorrupt
		}
		records[r.ID] = r
		if i+1 < len(t.Records) && r.End != t.Records[i+1].Date || i+1 == len(t.Records) && r.End != "" || r.End != "" && r.Date >= r.End {
			return compact.ErrCorrupt
		}
	}
	covered := map[string]bool{}
	for _, p := range e.Segments {
		r, ok := records[p.RecordID]
		if !ok || r.Revision != p.Revision || len(p.Nodes) == 0 || p.Nodes[0].Kind != "enterprise" {
			return compact.ErrCorrupt
		}
		covered[p.RecordID] = true
		for _, n := range p.Nodes[1:] {
			if n.Kind != "department" || !foundation.ValidID(n.DepartmentID) || !foundation.ValidID(n.Source) || !foundation.ValidID(n.RecordID) || !foundation.ValidID(n.Revision) || n.Edge.DepartmentID != n.DepartmentID || !foundation.ValidID(n.Edge.Source) || !foundation.ValidID(n.Edge.MemberID) {
				return compact.ErrCorrupt
			}
		}
	}
	if len(covered) != len(records) {
		return compact.ErrCorrupt
	}
	for _, r := range t.Records {
		segments := []d.PathSegment{}
		for _, p := range e.Segments {
			if p.RecordID == r.ID {
				segments = append(segments, p)
			}
		}
		if err := d.ValidateEvidence(t, r, segments); err != nil {
			return compact.ErrCorrupt
		}
	}
	return nil
}
func (u *departmentUnit) StoreTerminal(ctx context.Context, t app.Terminal) error {
	if t.Outcome == "accepted" {
		return errors.New("accepted terminal belongs to its atomic department source")
	}
	ent, e := u.Enterprise(ctx, false)
	if e != nil {
		return e
	}
	if t.Fields == nil && t.Outcome != "not_applied" {
		t.Fields = []string{}
	}
	if t.Outcome == "rejected" || t.Outcome == "not_applied" {
		t.Result = org.DepartmentResult{SubmissionID: t.ID, Outcome: t.Outcome, Action: t.Action, ErrorCode: depPtr(t.ErrorCode), FailureDetails: t.Failure}
	}
	t.Result.ExpectedSourceVersion = depPtr(t.ExpectedSource)
	if t.EvaluatedAt != nil {
		t.Result.EvaluatedAt = depPtr(t.EvaluatedAt.UTC().Format(time.RFC3339Nano))
		t.Result.EvaluatedBusinessDate = depPtr(t.EvaluatedAt.In(foundation.Shanghai).Format("2006-01-02"))
	}
	if e = validateTerminalHeader(t.ID, t.Outcome, t.Action, t.Digest, t.Fields, t.EvaluatedAt, t.Failure, t.ErrorCode, true); e != nil {
		return e
	}
	return history.SaveTerminal(ctx, u.tx, history.Department, u.instance, ent.ID, u.guard.Principal().ID, t.ID, history.TerminalEnvelope{Object: t.DepartmentID, Outcome: t.Outcome}, func(p *compact.Pool) ([]byte, error) {
		return history.EncodeDepartmentOperation(ctx, p, model.DepartmentOperation{Terminal: t})
	})
}
func (u *departmentUnit) FindResult(ctx context.Context, id string) (*app.Saved, error) {
	return u.departmentResult(ctx, id, false)
}
func (u *departmentUnit) CompleteResult(ctx context.Context, saved *app.Saved) error {
	full, e := u.departmentResult(ctx, saved.Result.SubmissionID, true)
	if e != nil {
		return e
	}
	if full == nil {
		return resultUnavailable()
	}
	saved.Result = full.Result
	saved.DecodeError = full.DecodeError
	return nil
}
func (u *departmentUnit) departmentResult(ctx context.Context, id string, complete bool) (*app.Saved, error) {
	subject := u.guard.Principal().ID
	found, e := history.FindSubmission(ctx, u.tx, history.Department, u.instance, subject, id)
	if e != nil {
		return nil, e
	}
	var t app.Terminal
	if found != nil {
		c, e := history.LoadChunk(ctx, u.tx, history.Department, u.instance, found.Object, found.Chunk)
		if e != nil {
			return &app.Saved{RequestObjectID: found.Object, Result: org.DepartmentResult{SubmissionID: id, Outcome: "accepted"}, DecodeError: resultUnavailable()}, nil
		}
		v, e := u.readDepartment(ctx, found.Object, c.ID(c.Sources[found.Slot]), false)
		if e != nil {
			return &app.Saved{RequestObjectID: found.Object, Result: org.DepartmentResult{SubmissionID: id, Outcome: "accepted"}, DecodeError: resultUnavailable()}, nil
		}
		t = v.Operation.Terminal
	} else {
		meta, e := history.TerminalIdentity(ctx, u.tx, history.Department, u.instance, subject, id)
		if errors.Is(e, pgx.ErrNoRows) {
			return nil, nil
		}
		if e != nil {
			return nil, e
		}
		damaged := func() (*app.Saved, error) {
			return &app.Saved{RequestObjectID: meta.Object, Result: org.DepartmentResult{SubmissionID: id, Outcome: meta.Outcome}, DecodeError: resultUnavailable()}, nil
		}
		pool, raw, e := history.LoadTerminal(ctx, u.tx, history.Department, u.instance, subject, id)
		if e != nil {
			return damaged()
		}
		op, e := history.DecodeDepartmentOperation(ctx, pool, raw)
		if e != nil {
			return damaged()
		}
		if op.Terminal.DepartmentID != meta.Object || op.Terminal.Outcome != meta.Outcome {
			return damaged()
		}
		t = op.Terminal
	}
	saved := &app.Saved{Result: t.Result, Digest: t.Digest, RequestObjectID: t.DepartmentID}
	if t.ID != id || t.Result.SubmissionID != id || t.Outcome != t.Result.Outcome || t.Outcome != "not_applied" && (len(t.Digest) != 32 || t.Fields == nil || !slices.IsSorted(t.Fields)) {
		saved.DecodeError = resultUnavailable()
	}
	if validateTerminalHeader(t.ID, t.Outcome, t.Action, t.Digest, t.Fields, t.EvaluatedAt, t.Failure, t.ErrorCode, true) != nil {
		saved.DecodeError = resultUnavailable()
	}

	if t.Action == "REVISE_TIMELINE" && t.Outcome == "accepted" {
		if len(t.Items) != 2 || len(t.Result.Items) != 2 {
			saved.DecodeError = resultUnavailable()
		}
		seen := map[string]bool{}
		for _, it := range t.Result.Items {
			if it.ItemID == "" || seen[it.ItemID] || !foundation.ValidID(it.RecordID) || !foundation.ValidID(value(it.AfterRevisionID)) || it.Outcome != "accepted" || it.BeforeRecord != nil || it.AfterRecord != nil {
				saved.DecodeError = resultUnavailable()
			}
			seen[it.ItemID] = true
		}
	}
	if complete && t.Outcome == "no_change" {
		if t.EvaluatedAt == nil || t.Result.ObservedSourceVersion == nil {
			return nil, resultUnavailable()
		}
		v, e := u.readDepartment(ctx, t.DepartmentID, *t.Result.ObservedSourceVersion, false)
		if e != nil {
			return nil, e
		}
		saved.Result.CurrentStatus = d.Status(v.State.Timeline, t.EvaluatedAt.In(foundation.Shanghai).Format("2006-01-02"))
		saved.Result.DepartmentID = depPtr(t.DepartmentID)
		if t.Action == "ADD" {
			if t.Result.SelectedRecordID == nil || !foundation.ValidID(*t.Result.SelectedRecordID) {
				return nil, resultUnavailable()
			}
			found := false
			for _, r := range v.State.Timeline.Records {
				found = found || r.ID == *t.Result.SelectedRecordID
			}
			if !found {
				return nil, resultUnavailable()
			}
		}
	}
	return saved, nil
}

package postgres

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history/model"
	"chnhcm.local/chnhcm/internal/platform/compact"
	"github.com/jackc/pgx/v5"
)

type buSource struct {
	State     model.BUState
	Operation model.BUOperation
	Chunk     history.Chunk
	Slot      int
}
type departmentSource struct {
	State     model.DepartmentState
	Operation model.DepartmentOperation
	Evidence  model.DepartmentEvidence
	Chunk     history.Chunk
	Slot      int
}
type ownerHead struct {
	ID, Enterprise, Code, First, Source string
	Number                              int64
	Chunk                               int32
}

func (u *unit) head(ctx context.Context, kind history.Kind, id string) (h ownerHead, err error) {
	table, key := "business_unit", "bu_id"
	if kind == history.Department {
		table, key = "department", "department_id"
	}
	err = u.tx.QueryRow(ctx, fmt.Sprintf(`SELECT %s::text,enterprise_id::text,code,first_record_id::text,head_source_id::text,head_no,head_chunk FROM organization.%s WHERE instance_id=$1 AND %s=$2`, key, table, key), u.instance, id).Scan(&h.ID, &h.Enterprise, &h.Code, &h.First, &h.Source, &h.Number, &h.Chunk)
	return
}
func (u *unit) sourceChunk(ctx context.Context, kind history.Kind, h ownerHead, source string) (history.Chunk, int, error) {
	n := h.Chunk
	if source == "" {
		source = h.Source
	}
	if source != h.Source {
		var e error
		n, e = history.Locate(ctx, u.tx, kind, u.instance, h.ID, source)
		if e != nil {
			return history.Chunk{}, 0, e
		}
	}
	c, e := history.LoadChunk(ctx, u.tx, kind, u.instance, h.ID, n)
	if e != nil {
		return c, 0, e
	}
	i := c.Index(source)
	if i < 0 {
		return c, 0, pgx.ErrNoRows
	}
	return c, i, nil
}
func (u *unit) readBU(ctx context.Context, id, source string) (*buSource, error) {
	if source != "" && u.buHistory != nil {
		if cached := u.buHistory[id+source]; cached != nil {
			return cached, nil
		}
	}
	h, e := u.head(ctx, history.BusinessUnit, id)
	if e != nil {
		return nil, e
	}
	c, i, e := u.sourceChunk(ctx, history.BusinessUnit, h, source)
	if e != nil {
		return nil, e
	}
	a, e := history.LoadShape(ctx, u.tx, history.BusinessUnit, u.instance, h.Enterprise, c.States[i])
	if e != nil {
		return nil, e
	}
	b, e := history.LoadShape(ctx, u.tx, history.BusinessUnit, u.instance, h.Enterprise, c.Operations[i])
	if e != nil {
		return nil, e
	}
	decoded := foundation.BeginStage(ctx, "bu.decode_state")
	state, e := history.DecodeBUState(ctx, c.Values, a)
	decoded()
	if e != nil {
		return nil, e
	}
	decoded = foundation.BeginStage(ctx, "bu.decode_operation")
	op, e := history.DecodeBUOperation(ctx, c.Values, b)
	decoded()
	if e != nil {
		return nil, e
	}
	state.Timeline.Code = h.Code
	if state.Timeline.BUID != id || state.Timeline.First != h.First || state.Timeline.Source != c.ID(c.Sources[i]) || state.Timeline.SourceNo != c.First+int64(i) || op.Source.ID != state.Timeline.Source || op.Source.Number != state.Timeline.SourceNo || op.Source.Subject != c.ID(c.Subjects[i]) || op.Source.Submission != c.ID(c.Submissions[i]) {
		return nil, compact.ErrCorrupt
	}
	if e = validateBUFacts(state, op); e != nil {
		return nil, e
	}
	v := &buSource{state, op, c, i}
	if u.buHistory == nil {
		u.buHistory = map[string]*buSource{}
	}
	u.buHistory[id+state.Timeline.Source] = v
	return v, nil
}
func (u *unit) readDepartment(ctx context.Context, id, source string, evidence bool) (*departmentSource, error) {
	cacheKey := id + source
	if source != "" && u.departmentHistory != nil {
		if cached := u.departmentHistory[cacheKey]; cached != nil && (!evidence || cached.Evidence.Segments != nil) {
			return cached, nil
		}
	}
	h, e := u.head(ctx, history.Department, id)
	if e != nil {
		return nil, e
	}
	c, i, e := u.sourceChunk(ctx, history.Department, h, source)
	if e != nil {
		return nil, e
	}
	a, e := history.LoadShape(ctx, u.tx, history.Department, u.instance, h.Enterprise, c.States[i])
	if e != nil {
		return nil, e
	}
	b, e := history.LoadShape(ctx, u.tx, history.Department, u.instance, h.Enterprise, c.Operations[i])
	if e != nil {
		return nil, e
	}
	state, e := history.DecodeDepartmentState(ctx, c.Values, a)
	if e != nil {
		return nil, e
	}
	op, e := history.DecodeDepartmentOperation(ctx, c.Values, b)
	if e != nil {
		return nil, e
	}
	state.Timeline.Code = h.Code
	if state.Timeline.ID != id || state.Timeline.Enterprise != h.Enterprise || state.Timeline.First != h.First || state.Timeline.Source != c.ID(c.Sources[i]) || state.Timeline.SourceNo != c.First+int64(i) || op.Source.ID != state.Timeline.Source || op.Source.Number != state.Timeline.SourceNo || op.Source.Subject != c.ID(c.Subjects[i]) || op.Source.Submission != c.ID(c.Submissions[i]) {
		return nil, compact.ErrCorrupt
	}
	if e = validateDepartmentOperation(state, op); e != nil {
		return nil, e
	}
	v := &departmentSource{State: state, Operation: op, Chunk: c, Slot: i}
	if evidence {
		b, e = history.LoadShape(ctx, u.tx, history.Department, u.instance, h.Enterprise, c.Evidence[i])
		if e != nil {
			return nil, e
		}
		v.Evidence, e = history.DecodeDepartmentEvidence(ctx, c.Values, b)
		if e != nil {
			return nil, e
		}
		if e = validateDepartmentFacts(state, op, v.Evidence); e != nil {
			return nil, foundation.Fail(503, "M02_DEPARTMENT_PATH_UNAVAILABLE", "部门路径暂时无法加载")
		}
	}
	if u.departmentHistory == nil {
		u.departmentHistory = map[string]*departmentSource{}
	}
	u.departmentHistory[id+state.Timeline.Source] = v
	return v, nil
}

// Root identity is checked before these facts leave the storage adapter. The
// full write candidate still belongs to the existing Go domain validator.
func validateBUState(state model.BUState) error {
	rows := state.Timeline.Records
	if len(rows) == 0 || rows[0].ID != state.Timeline.First || rows[0].Type != "CREATE" {
		return compact.ErrCorrupt
	}
	seen := map[string]bool{}
	for i, r := range rows {
		if !compact.IsID(r.ID) || !compact.IsID(r.Revision) || seen[r.ID] || r.RevisionNo < 1 || (r.State != "active" && r.State != "inactive") {
			return compact.ErrCorrupt
		}
		if r.Reason == "" || r.TypeLabel == "" || r.ReasonLabel == "" {
			return foundation.Fail(503, "M02_CLASSIFICATION_UNAVAILABLE", "暂时无法显示操作详情")
		}
		seen[r.ID] = true
		if i+1 < len(rows) && r.End != rows[i+1].Date || i+1 == len(rows) && r.End != "" || r.End != "" && r.Date >= r.End {
			return compact.ErrCorrupt
		}
	}
	if len(state.Summaries) != len(rows) {
		return compact.ErrCorrupt
	}
	summaries := summaryIndex(state.Summaries)
	for i, r := range rows {
		if !validSpan(r.Date, r.End) || !summaries[[2]string{r.ID, r.Revision}] {
			return compact.ErrCorrupt
		}
		if r.State == "inactive" {
			if i == 0 || r.Type != "DEACTIVATE" {
				return compact.ErrCorrupt
			}
			before := rows[i-1]
			if before.State != "active" || r.InheritedRecord != before.ID || r.InheritedRevision != before.Revision || r.Data.Name != before.Data.Name || !reflect.DeepEqual(r.Data.NameEn, before.Data.NameEn) {
				return compact.ErrCorrupt
			}
		} else if r.InheritedRecord != "" || r.InheritedRevision != "" || r.Type == "DEACTIVATE" {
			return compact.ErrCorrupt
		}
	}
	return nil
}
func validateBUFacts(state model.BUState, op model.BUOperation) error {
	if e := validateBUState(state); e != nil {
		return e
	}
	if !foundation.ValidID(op.EventID) || !foundation.ValidID(op.Source.ID) || !foundation.ValidID(op.Source.Subject) || op.Source.Operator.SubjectID != op.Source.Subject || op.Source.Action != op.Terminal.Action || op.Source.RecordedAt.IsZero() || op.TerminalClosedAt.IsZero() || op.Source.Number != state.Timeline.SourceNo || op.Source.ID != state.Timeline.Source || (op.Source.Number == 1) != (op.Source.Previous == "") {
		return compact.ErrCorrupt
	}
	if op.Source.Previous != "" && (!foundation.ValidID(op.Source.Previous) || op.Source.Previous == op.Source.ID) {
		return compact.ErrCorrupt
	}
	if len(op.Revisions) == 0 || op.Terminal.Result.Impact == nil || !reflect.DeepEqual(op.Impact, *op.Terminal.Result.Impact) {
		return compact.ErrCorrupt
	}
	members := make(map[string]int, len(state.Timeline.Records))
	for i, r := range state.Timeline.Records {
		members[r.ID] = i
	}
	seen := map[string]bool{}
	for _, rev := range op.Revisions {
		r := rev.Record
		if seen[r.ID] || !foundation.ValidID(r.ID) || !foundation.ValidID(r.Revision) || !validSpan(r.Date, r.End) {
			return compact.ErrCorrupt
		}
		seen[r.ID] = true
		if rev.Before == nil {
			if rev.Previous != "" || r.RevisionNo != 1 || rev.Deleted {
				return compact.ErrCorrupt
			}
		} else if rev.Before.ID != r.ID || rev.Previous != rev.Before.Revision || r.RevisionNo != rev.Before.RevisionNo+1 || r.Revision == rev.Previous {
			return compact.ErrCorrupt
		}
		index, found := members[r.ID]
		if found && !reflect.DeepEqual(state.Timeline.Records[index], r) {
			return compact.ErrCorrupt
		}
		if found == rev.Deleted {
			return compact.ErrCorrupt
		}
	}
	result := op.Terminal.Result
	if result.SubmissionID != op.Source.Submission || result.Action != op.Source.Action || value(result.RecordedAt) != op.Source.RecordedAt.UTC().Format(time.RFC3339Nano) || len(op.Impact.DataAffectedRanges)+len(op.Impact.StructuralChanges) == 0 {
		return compact.ErrCorrupt
	}
	if result.Action != "REVISE_TIMELINE" {
		matched := false
		for _, r := range op.Revisions {
			if value(result.TargetRecordID) == r.Record.ID && value(result.TargetRevision) == r.Record.Revision && value(result.EffectiveDate) == r.Record.Date {
				matched = true
			}
		}
		if !matched {
			return compact.ErrCorrupt
		}
	}
	if op.Terminal.ID != op.Source.Submission || op.Terminal.Outcome != "accepted" || op.Terminal.Result.SourceVersion == nil || *op.Terminal.Result.SourceVersion != op.Source.ID || op.Terminal.Result.BUID == nil || *op.Terminal.Result.BUID != state.Timeline.BUID {
		return compact.ErrCorrupt
	}
	return nil
}

func missingHistory(err error) bool {
	return errors.Is(err, pgx.ErrNoRows) || errors.Is(err, compact.ErrCorrupt)
}

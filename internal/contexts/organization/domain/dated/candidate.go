package dated

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"fmt"
	"slices"
	"strings"
)

// Record[T] is a complete independent snapshot, except DEACTIVATE which inherits its
// immediate predecessor. IDs are allocated by persistence after validation.
type Record[T any] struct {
	ID, Revision                                string
	RevisionNo                                  int64
	Date, End                                   string
	Data                                        T
	Type, Reason, TypeLabel, ReasonLabel, State string
	InheritedRecord, InheritedRevision          string
}
type Item[P any] struct {
	ID, Action, RecordID, ExpectedRevision, Date, Type, Reason string
	Changes                                                    *P
}
type Range struct {
	Start string  `json:"start"`
	End   *string `json:"endExclusive"`
}
type StructuralChange struct {
	RecordID       string   `json:"recordId,omitempty"`
	ItemID         string   `json:"itemId,omitempty"`
	BeforeRevision *string  `json:"beforeRevisionId"`
	AfterRevision  *string  `json:"afterRevisionId"`
	Facets         []string `json:"facets"`
}
type PeriodChange struct {
	RecordID string `json:"recordId,omitempty"`
	ItemID   string `json:"itemId,omitempty"`
	Before   *Range `json:"before"`
	After    *Range `json:"after"`
}
type Impact struct {
	DataAffectedRanges []Range            `json:"dataAffectedRanges"`
	StructuralChanges  []StructuralChange `json:"structuralChanges"`
	AffectedRecordIDs  []string           `json:"affectedRecordIds"`
	PeriodChanges      []PeriodChange     `json:"periodChanges"`
}
type Candidate[T any] struct {
	Records  []Record[T]
	Deleted  []Record[T] // Original snapshots retained as deletion evidence, never live members.
	Impact   Impact
	NoChange bool
}

func ptr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
func span[T any](r Record[T]) Range { return Range{Start: r.Date, End: ptr(r.End)} }
func invalid(code, field, message string) error {
	status := 409
	if code == "M02_INVALID_INPUT" || code == "M02_FORBIDDEN_INPUT" {
		status = 400
	}
	return &foundation.Fault{Status: status, Code: code, Field: field, Message: message}
}

// WorkingCandidate is a complete local timeline before relation binding and
// redundant-addition elimination. Only Finalize can produce the persistence candidate.
// Adapters may bind object-owned references in Records; base/items remain private.
type WorkingCandidate[T, P any] struct {
	Records []Record[T]
	Deleted []Record[T]
	base    []Record[T]
	items   []Item[P]
}

// Calculate is the no-relation adapter, using the same two phases as Department.
func (k Kernel[T, P]) Calculate(ctx context.Context, base []Record[T], first, minDate string, items []Item[P]) (Candidate[T], error) {
	w, err := k.Prepare(ctx, base, first, minDate, items)
	if err != nil {
		return Candidate[T]{}, err
	}
	return k.Finalize(ctx, w)
}

// Prepare applies the complete set without eliminating pending CHANGE records.
// Item order never means execution order; external relation rules run afterwards.

func (k Kernel[T, P]) Prepare(ctx context.Context, base []Record[T], first, minDate string, items []Item[P]) (WorkingCandidate[T, P], error) {
	return k.prepare(ctx, base, first, minDate, items, false)
}

// ReferenceProjection is read-only assistance. It cannot be passed to Finalize.
// An unselected reason stays empty; it never receives a fabricated default.
type ReferenceProjection[T any] struct{ Records []Record[T] }

func (k Kernel[T, P]) PrepareReference(ctx context.Context, base []Record[T], first, minDate string, items []Item[P]) (ReferenceProjection[T], error) {
	w, e := k.prepare(ctx, base, first, minDate, items, true)
	return ReferenceProjection[T]{Records: w.Records}, e
}
func (k Kernel[T, P]) prepare(ctx context.Context, base []Record[T], first, minDate string, items []Item[P], assistance bool) (out WorkingCandidate[T, P], err error) {
	if err := ctx.Err(); err != nil {
		return out, err
	}
	out = WorkingCandidate[T, P]{Records: k.CloneRecords(base), base: k.CloneRecords(base), items: slices.Clone(items)}
	items = slices.Clone(items)
	slices.SortFunc(items, func(a, b Item[P]) int { return strings.Compare(a.ID, b.ID) })
	currentItem := ""
	defer func() {
		if err != nil && currentItem != "" {
			err = &ItemFault{ItemID: currentItem, Err: err}
		}
	}()
	if len(base) == 0 || len(items) == 0 {
		return out, invalid("M02_INVALID_INPUT", "", "缺少当前时间线或变更项")
	}
	seenItems, targets := map[string]bool{}, map[string]bool{}
	pending := []Item[P]{}
	for _, it := range items {
		currentItem = it.ID
		if err := ctx.Err(); err != nil {
			return out, err
		}
		if it.ID == "" || seenItems[it.ID] {
			return out, invalid("M02_REVISION_ITEM_CONFLICT", "", "变更项标识重复")
		}
		seenItems[it.ID] = true
		switch it.Action {
		case "ADD":
			if !slices.Contains([]string{"CHANGE", "DEACTIVATE", "REACTIVATE"}, it.Type) || !k.validReason(it.Type, it.Reason, assistance) {
				return out, invalid("M02_INVALID_INPUT", "operationReason", "请选择适用的操作类型及原因")
			}
			if it.Type == "DEACTIVATE" && it.Changes != nil {
				return out, invalid("M02_FORBIDDEN_INPUT", "changes", "停用只继承前序资料，不能夹带资料修改")
			}
			if it.Type == "REACTIVATE" && it.Changes != nil && k.Policy.EmptyPatch(*it.Changes) {
				return out, invalid("M02_INVALID_INPUT", "changes", "没有修改内容")
			}
			if !foundation.ValidDate(it.Date) || it.Date < minDate {
				return out, invalid("M02_INVALID_INPUT", "effectiveDate", "日期须在对象允许的有效日期范围内")
			}
			pending = append(pending, it)
			out.Records = append(out.Records, Record[T]{ID: CandidateID(it), Date: it.Date, Type: it.Type, Reason: it.Reason, TypeLabel: k.Policy.OperationLabel(it.Type), ReasonLabel: k.Policy.OperationReasons(it.Type)[it.Reason]})
		case "CORRECT", "DELETE", "REDATE", "UPDATE":
			if targets[it.RecordID] {
				return out, invalid("M02_REVISION_ITEM_CONFLICT", "", "同一记录只能修改一次")
			}
			targets[it.RecordID] = true
			if !slices.ContainsFunc(base, func(r Record[T]) bool { return r.ID == it.RecordID }) {
				return out, invalid("M02_RECORD_NOT_FOUND", "", "当前资料中没有该记录")
			}
			n := slices.IndexFunc(out.Records, func(r Record[T]) bool { return r.ID == it.RecordID })
			if n < 0 {
				return out, invalid("M02_RECORD_NOT_FOUND", "", "当前资料中没有该记录")
			}
			r := out.Records[n]
			if r.Revision != it.ExpectedRevision {
				return out, invalid("M02_REVISION_CONFLICT", "", "记录已更新，请核对最新内容后保存")
			}
			if it.Action == "UPDATE" {
				if it.Changes == nil && it.Date == "" && it.Type == "" {
					return out, invalid("M02_INVALID_INPUT", "", "没有修改内容")
				}
				if it.Type != "" {
					if (r.ID == first && it.Type != "CREATE") || (r.ID != first && it.Type == "CREATE") {
						return out, invalid("M02_FIRST_RECORD_PROTECTED", "classification.operationType", "首条类型必须保持新建")
					}
					if r.Type != it.Type || r.Reason != it.Reason {
						if !k.validReason(it.Type, it.Reason, assistance) {
							return out, invalid("M02_INVALID_INPUT", "classification.operationReason", "请选择适用分类")
						}
						r.Type, r.Reason = it.Type, it.Reason
						r.TypeLabel, r.ReasonLabel = k.Policy.OperationLabel(it.Type), k.Policy.OperationReasons(it.Type)[it.Reason]
					}
				}
				if it.Date != "" {
					if !foundation.ValidDate(it.Date) || it.Date < minDate {
						return out, invalid("M02_INVALID_INPUT", "newEffectiveDate", "日期须在对象允许的有效日期范围内")
					}
					r.Date = it.Date
				}
				if r.Type == "DEACTIVATE" && it.Changes != nil {
					return out, invalid("M02_FORBIDDEN_INPUT", "changes", "停用只继承前序资料，不能夹带资料修改")
				}
				if r.Type != "DEACTIVATE" {
					r.InheritedRecord, r.InheritedRevision = "", ""
				}
				if it.Changes != nil {
					r, err = k.apply(r, it.Changes)
					if err != nil {
						return out, err
					}
				}
				out.Records[n] = r
				continue
			}
			if it.Action == "REDATE" {
				if it.Changes != nil || it.Type != "" || it.Reason != "" {
					return out, invalid("M02_FORBIDDEN_INPUT", "", "改日只接受新日期和目标修订")
				}
				if !foundation.ValidDate(it.Date) || it.Date < minDate {
					return out, invalid("M02_INVALID_INPUT", "effectiveDate", "新日期须在对象允许的有效日期范围内")
				}
				out.Records[n].Date = it.Date
				continue
			}
			if it.Action == "DELETE" {
				if it.Changes != nil || it.Date != "" || it.Type != "" || it.Reason != "" {
					return out, invalid("M02_FORBIDDEN_INPUT", "", "删除只接受来源和目标修订")
				}
				if r.ID == first {
					return out, invalid("M02_FIRST_RECORD_PROTECTED", "", "首条记录不能删除")
				}
				out.Deleted = append(out.Deleted, r)
				out.Records = slices.Delete(out.Records, n, n+1)
				continue
			}
			if r.Type == "DEACTIVATE" {
				return out, invalid("M02_ACTION_NOT_ALLOWED", "", "停用资料由前序继承，不能独立更正")
			}
			var err error
			out.Records[n], err = k.apply(r, it.Changes)
			if err != nil {
				return out, err
			}
		default:
			return out, invalid("M02_CAPABILITY_UNSUPPORTED", "", "此操作不可用")
		}
	}
	currentItem = ""
	if err := ctx.Err(); err != nil {
		return out, err
	}
	slices.SortFunc(out.Records, func(a, b Record[T]) int {
		if a.Date < b.Date {
			return -1
		}
		if a.Date > b.Date {
			return 1
		}
		return 0
	})
	if err := ctx.Err(); err != nil {
		return out, err
	}
	// Uniqueness and fixed-first checks precede no-change elimination.
	if out.Records[0].ID != first || out.Records[0].Type != "CREATE" {
		return out, invalid("M02_FIRST_RECORD_PROTECTED", "effectiveDate", "固定首条必须仍为最早记录")
	}
	for i, r := range out.Records {
		currentItem = itemForRecord(items, r.ID)
		if i%128 == 0 {
			if err := ctx.Err(); err != nil {
				return out, err
			}
		}
		if !foundation.ValidDate(r.Date) || r.Date < minDate {
			return out, invalid("M02_INVALID_INPUT", "effectiveDate", "日期不在对象允许范围")
		}
		if i > 0 && r.Date == out.Records[i-1].Date {
			return out, invalid("M02_DATE_CONFLICT", "effectiveDate", "同一日期已有生效记录")
		}
	}
	resolved := make([]Record[T], 0, len(out.Records))
	state := "no_fact"
	for _, r := range out.Records {
		currentItem = itemForRecord(items, r.ID)
		if err := ctx.Err(); err != nil {
			return out, err
		}
		var prev Record[T]
		if len(resolved) > 0 {
			prev = resolved[len(resolved)-1]
		}
		switch r.Type {
		case "CREATE":
			if state != "no_fact" || r.ID != first {
				return out, invalid("M02_TIMELINE_INVALID", "", "新建必须是固定首条")
			}
			r.State = "active"
		case "CHANGE":
			if state != "active" {
				return out, invalid("M02_TIMELINE_INVALID", "effectiveDate", fmt.Sprintf("%s 的资料变更缺少启用前提，请核对前后启停记录", r.Date))
			}
			r.State = "active"
		case "DEACTIVATE":
			if state != "active" {
				return out, invalid("M02_TIMELINE_INVALID", "effectiveDate", fmt.Sprintf("%s 的停用缺少启用前提，请核对前后启停记录", r.Date))
			}
			r.State = "inactive"
			r.Data = k.Policy.Clone(prev.Data)
			r.InheritedRecord = prev.ID
			r.InheritedRevision = prev.Revision
		case "REACTIVATE":
			if state != "inactive" {
				return out, invalid("M02_TIMELINE_INVALID", "effectiveDate", fmt.Sprintf("%s 的重新启用缺少停用前提，请核对前后启停记录", r.Date))
			}
			r.State = "active"
		default:
			return out, invalid("M02_TIMELINE_INVALID", "", "未知业务类型")
		}
		p := slices.IndexFunc(pending, func(it Item[P]) bool { return CandidateID(it) == r.ID })
		if p >= 0 && r.Type != "DEACTIVATE" {
			it := pending[p]
			r.Data = k.Policy.Clone(prev.Data)
			var err error
			// An explicit empty CHANGE patch represents unchanged copied defaults.
			// Date, first-record and lifecycle checks still precede elimination.
			if r.Type == "CHANGE" && it.Changes == nil && !assistance || it.Changes != nil && !k.Policy.EmptyPatch(*it.Changes) {
				r, err = k.apply(r, it.Changes)
			}
			if err != nil {
				return out, err
			}
		}
		state = r.State
		resolved = append(resolved, r)
	}
	currentItem = ""
	for i := range resolved {
		resolved[i].End = ""
		if i+1 < len(resolved) {
			resolved[i].End = resolved[i+1].Date
		}
	}
	out.Records = resolved
	return out, ctx.Err()
}

// Finalize binds inheritance after adapter-owned relations are final. It never
// changes stable business identities or invents lifecycle events.
func (k Kernel[T, P]) Finalize(ctx context.Context, work WorkingCandidate[T, P]) (out Candidate[T], err error) {
	base, items := work.base, work.items
	pending := map[string]bool{}
	for _, it := range items {
		if it.Action == "ADD" {
			pending[CandidateID(it)] = true
		}
	}
	resolved := make([]Record[T], 0, len(work.Records))
	for _, r := range k.CloneRecords(work.Records) {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		if len(resolved) > 0 {
			prev := resolved[len(resolved)-1]
			if r.Type == "DEACTIVATE" {
				r.Data = k.Policy.Clone(prev.Data)
				r.InheritedRecord = prev.ID
				r.InheritedRevision = prev.Revision
			}
			if pending[r.ID] && r.Type == "CHANGE" && k.sameData(r, prev) && k.Policy.EqualRevision(r.Data, prev.Data) {
				continue
			}
		}
		resolved = append(resolved, r)
	}
	for i := range resolved {
		resolved[i].End = ""
		if i+1 < len(resolved) {
			resolved[i].End = resolved[i+1].Date
		}
	}
	out.Deleted = k.CloneRecords(work.Deleted)
	// Periods must be final before resolving revision dependencies. Record[T].Revision
	// still identifies the baseline; a pending successor must not inherit that
	// old revision. Its exact reference is bound only after persistence allocates it.
	old := make(map[string]Record[T], len(base))
	for i, r := range base {
		if i%128 == 0 && ctx.Err() != nil {
			return out, ctx.Err()
		}
		old[r.ID] = r
	}
	pendingRevisions := make(map[string]bool)
	for i := range resolved {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		r := &resolved[i]
		if r.Type == "DEACTIVATE" && i > 0 {
			r.InheritedRevision = resolved[i-1].Revision
			if pendingRevisions[resolved[i-1].ID] {
				r.InheritedRevision = ""
			}
		}
		if b, exists := old[r.ID]; !exists || !k.sameRevision(b, *r) {
			pendingRevisions[r.ID] = true
		}
	}
	out.Records = resolved
	out.Impact = k.Compare(ctx, base, resolved, items)
	out.NoChange = len(out.Impact.AffectedRecordIDs) == 0 && len(out.Impact.StructuralChanges) == 0
	return out, ctx.Err()
}

// Compare uses boundary sweep, keeping unchanged gaps out of data impact.
func (k Kernel[T, P]) Compare(ctx context.Context, before, after []Record[T], items []Item[P]) Impact {
	impact := Impact{DataAffectedRanges: []Range{}, StructuralChanges: []StructuralChange{}, AffectedRecordIDs: []string{}, PeriodChanges: []PeriodChange{}}
	boundaries := []string{}
	for _, rows := range [][]Record[T]{before, after} {
		for _, r := range rows {
			boundaries = append(boundaries, r.Date)
		}
	}
	slices.Sort(boundaries)
	boundaries = slices.Compact(boundaries)
	bi, ai := -1, -1
	for i, start := range boundaries {
		if i%128 == 0 && ctx.Err() != nil {
			return impact
		}
		for bi+1 < len(before) && before[bi+1].Date <= start {
			bi++
		}
		for ai+1 < len(after) && after[ai+1].Date <= start {
			ai++
		}
		different := bi < 0 || ai < 0
		if bi >= 0 && ai >= 0 {
			different = !k.sameData(before[bi], after[ai])
		}
		if !different {
			continue
		}
		end := ""
		if i+1 < len(boundaries) {
			end = boundaries[i+1]
		}
		n := len(impact.DataAffectedRanges)
		if n > 0 && impact.DataAffectedRanges[n-1].End != nil && *impact.DataAffectedRanges[n-1].End == start {
			impact.DataAffectedRanges[n-1].End = ptr(end)
		} else {
			impact.DataAffectedRanges = append(impact.DataAffectedRanges, Range{Start: start, End: ptr(end)})
		}
	}
	old := map[string]Record[T]{}
	for _, r := range before {
		old[r.ID] = r
	}
	for i, r := range after {
		if i%128 == 0 && ctx.Err() != nil {
			return impact
		}
		b, exists := old[r.ID]
		delete(old, r.ID)
		if exists && k.sameRevision(b, r) {
			continue
		}
		impact.AffectedRecordIDs = append(impact.AffectedRecordIDs, r.ID)
		facets := []string{}
		if !exists {
			facets = append(facets, "ADD")
		} else if b.Date != r.Date || b.End != r.End {
			facets = append(facets, "PERIOD")
		}
		if exists && (b.InheritedRecord != r.InheritedRecord || b.InheritedRevision != r.InheritedRevision) {
			facets = append(facets, "INHERITANCE")
		}
		if exists && b.Date != r.Date {
			facets = append(facets, "REDATE")
		}
		if exists && (b.Type != r.Type || b.Reason != r.Reason || b.TypeLabel != r.TypeLabel || b.ReasonLabel != r.ReasonLabel) {
			facets = append(facets, "RECLASSIFY")
		}
		if exists && !k.Policy.EqualRevision(b.Data, r.Data) {
			facets = append(facets, "REFERENCE")
		}
		itemID := ""
		for _, it := range items {
			if it.RecordID == r.ID || it.Action == "ADD" && CandidateID(it) == r.ID {
				itemID = it.ID
				break
			}
		}
		if len(facets) > 0 {
			impact.StructuralChanges = append(impact.StructuralChanges, StructuralChange{RecordID: r.ID, ItemID: itemID, BeforeRevision: ptr(b.Revision), Facets: facets})
		}
		if !exists || b.Date != r.Date || b.End != r.End {
			var bp *Range
			if exists {
				v := span(b)
				bp = &v
			}
			ap := span(r)
			impact.PeriodChanges = append(impact.PeriodChanges, PeriodChange{RecordID: r.ID, ItemID: itemID, Before: bp, After: &ap})
		}
	}
	// Removed records still have a structural effect even if business values are
	// identical. Iterate the original order for stable, independent impact output.
	for i, r := range before {
		if i%128 == 0 && ctx.Err() != nil {
			return impact
		}
		if _, removed := old[r.ID]; !removed {
			continue
		}
		itemID := ""
		for _, it := range items {
			if it.RecordID == r.ID {
				itemID = it.ID
			}
		}
		impact.AffectedRecordIDs = append(impact.AffectedRecordIDs, r.ID)
		impact.StructuralChanges = append(impact.StructuralChanges, StructuralChange{RecordID: r.ID, ItemID: itemID, BeforeRevision: ptr(r.Revision), Facets: []string{"DELETE"}})
		before := span(r)
		impact.PeriodChanges = append(impact.PeriodChanges, PeriodChange{RecordID: r.ID, ItemID: itemID, Before: &before})
	}
	return impact
}

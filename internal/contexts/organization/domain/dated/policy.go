package dated

import (
	"context"
	"slices"
	"strings"

	"chnhcm.local/chnhcm/internal/foundation"
)

// Policy owns object data, never ordering, lifecycle or first-record rules.
// Clone must detach every mutable child. Apply must not mutate its patch.
// Normalize is used for new data only; stored snapshots are not renormalized.
type Policy[T, P any] interface {
	Normalize(T) (T, error)
	Clone(T) T
	Apply(T, *P) (T, error)
	EmptyPatch(P) bool
	EqualData(T, T) bool
	// EqualRevision compares object-owned revision structure independently of business values.
	EqualRevision(T, T) bool
	OperationReasons(string) map[string]string
	OperationLabel(string) string
}

// Date-taking entry points receive the object adapter's minimum date.
// They never use the initial default as a lower bound.
type Kernel[T, P any] struct{ Policy Policy[T, P] }

// orderedView is read-only. Current sources already have ordered records; an
// unsorted caller gets a shallow sorted directory without cloning every datum.
// Any record returned to a caller must detach its data separately.
func orderedView[T any](rows []Record[T]) []Record[T] {
	compare := func(a, b Record[T]) int { return strings.Compare(a.Date, b.Date) }
	if slices.IsSortedFunc(rows, compare) {
		return rows
	}
	out := slices.Clone(rows)
	slices.SortFunc(out, compare)
	return out
}

func (k Kernel[T, P]) CloneRecords(rows []Record[T]) []Record[T] {
	out := slices.Clone(rows)
	for i := range out {
		out[i].Data = k.Policy.Clone(out[i].Data)
	}
	return out
}

func (k Kernel[T, P]) apply(r Record[T], patch *P) (Record[T], error) {
	if patch == nil || k.Policy.EmptyPatch(*patch) {
		return r, invalid("M02_INVALID_INPUT", "changes", "没有修改内容")
	}
	data, err := k.Policy.Apply(k.Policy.Clone(r.Data), patch)
	if err != nil {
		return r, err
	}
	r.Data = k.Policy.Clone(data)
	return r, nil
}

func (k Kernel[T, P]) sameData(a, b Record[T]) bool {
	return a.State == b.State && k.Policy.EqualData(a.Data, b.Data)
}

func (k Kernel[T, P]) sameRevision(a, b Record[T]) bool {
	return a.ID == b.ID && a.Date == b.Date && a.End == b.End &&
		a.Type == b.Type && a.Reason == b.Reason && a.TypeLabel == b.TypeLabel &&
		a.ReasonLabel == b.ReasonLabel && a.State == b.State &&
		a.InheritedRecord == b.InheritedRecord && a.InheritedRevision == b.InheritedRevision &&
		k.Policy.EqualData(a.Data, b.Data) && k.Policy.EqualRevision(a.Data, b.Data)
}

// FactAt returns a detached snapshot from a complete, sorted source.
func (k Kernel[T, P]) FactAt(rows []Record[T], date string) *Record[T] {
	i := IndexAt(rows, date, func(r *Record[T]) string { return r.Date })
	if i < 0 {
		return nil
	}
	r := rows[i]
	r.Data = k.Policy.Clone(r.Data)
	return &r
}

func (k Kernel[T, P]) Create(ctx context.Context, id, date, minDate, reason string, data T) (Candidate[T], error) {
	if err := ctx.Err(); err != nil {
		return Candidate[T]{}, err
	}
	if !foundation.ValidDate(date) || date < minDate {
		return Candidate[T]{}, invalid("M02_INVALID_INPUT", "effectiveDate", "生效日期须在对象允许的有效日期范围内")
	}
	label := k.Policy.OperationReasons("CREATE")[reason]
	if label == "" {
		return Candidate[T]{}, invalid("M02_INVALID_INPUT", "operationReason", "请选择适用的新建操作原因")
	}
	normalized, err := k.Policy.Normalize(k.Policy.Clone(data))
	if err != nil {
		return Candidate[T]{}, err
	}
	rows := []Record[T]{{ID: id, Date: date, Type: "CREATE", Reason: reason, TypeLabel: k.Policy.OperationLabel("CREATE"), ReasonLabel: label, State: "active", Data: k.Policy.Clone(normalized)}}
	return Candidate[T]{Records: rows, Impact: k.Compare(ctx, nil, rows, nil)}, ctx.Err()
}

// PrepareCreate shares first-record admission with the no-relation adapter.
func (k Kernel[T, P]) PrepareCreate(ctx context.Context, id, date, minDate, reason string, data T) (WorkingCandidate[T, P], error) {
	c, err := k.Create(ctx, id, date, minDate, reason, data)
	return WorkingCandidate[T, P]{Records: c.Records}, err
}

// IndexAt preserves logarithmic lookup for adapters with flat object snapshots.
func IndexAt[R any](rows []R, date string, dateOf func(*R) string) int {
	if len(rows) == 0 {
		return -1
	}
	first := dateOf(&rows[0])
	if date < first {
		return -1
	}
	if date == first {
		return 0
	}
	last := dateOf(&rows[len(rows)-1])
	if date > last || date == last && (len(rows) == 1 || dateOf(&rows[len(rows)-2]) < date) {
		return len(rows) - 1
	}
	// Inspect dates in place. Passing a complete dated record by value to each
	// comparator copied hundreds of bytes at every binary-search step.
	lo, hi := 0, len(rows)
	for lo < hi {
		mid := int(uint(lo+hi) >> 1)
		if dateOf(&rows[mid]) < date {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo == len(rows) || dateOf(&rows[lo]) != date {
		return lo - 1
	}
	return lo
}

func (k Kernel[T, P]) validReason(kind, reason string, assistance bool) bool {
	reasons := k.Policy.OperationReasons(kind)
	return len(reasons) > 0 && (reasons[reason] != "" || assistance && reason == "")
}

// CreateReference validates the same first-date boundary without inventing data or a reason.
func (k Kernel[T, P]) CreateReference(ctx context.Context, id, date, minDate string, data T) (ReferenceProjection[T], error) {
	if err := ctx.Err(); err != nil {
		return ReferenceProjection[T]{}, err
	}
	if !foundation.ValidDate(date) || date < minDate {
		return ReferenceProjection[T]{}, invalid("M02_INVALID_INPUT", "effectiveDate", "生效日期须在对象允许的有效日期范围内")
	}
	normalized, e := k.Policy.Normalize(k.Policy.Clone(data))
	if e != nil {
		return ReferenceProjection[T]{}, e
	}
	return ReferenceProjection[T]{Records: []Record[T]{{ID: id, Date: date, Type: "CREATE", TypeLabel: k.Policy.OperationLabel("CREATE"), State: "active", Data: normalized}}}, nil
}

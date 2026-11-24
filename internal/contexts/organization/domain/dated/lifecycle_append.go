package dated

import (
	"slices"

	"chnhcm.local/chnhcm/internal/foundation"
)

// A single state transition can only follow the last effective record.
// Availability depends on that record's resulting state, including future facts.
func (k Kernel[T, P]) PrepareLifecycleAppend(base []Record[T], anchor, kind, minDate, today string) (AdditionWindow[T], error) {
	rows := orderedView(base)
	n := slices.IndexFunc(rows, func(r Record[T]) bool { return r.ID == anchor })
	if n < 0 {
		return AdditionWindow[T]{}, invalid("M02_INVALID_INPUT", "anchorRecordId", "请选择当前来源中的末条记录")
	}
	w := AdditionWindow[T]{Anchor: rows[n], Mode: "APPEND"}
	w.Anchor.Data = k.Policy.Clone(w.Anchor.Data)
	state := map[string]string{"DEACTIVATE": "active", "REACTIVATE": "inactive"}[kind]
	if n != len(rows)-1 || state == "" || w.Anchor.State != state {
		return w, nil
	}
	w.Min = NextDate(w.Anchor.Date, 1)
	if w.Min == "" {
		return w, nil
	}
	if w.Min < minDate {
		w.Min = minDate
	}
	w.Max = "9999-12-31"
	w.Available = w.Min <= w.Max
	w.Default = max(w.Min, today)
	return w, nil
}

func (k Kernel[T, P]) ValidateLifecycleAppend(base []Record[T], anchor, kind, minDate, date string) error {
	w, e := k.PrepareLifecycleAppend(base, anchor, kind, minDate, date)
	if e != nil {
		return e
	}
	if !w.Available {
		return invalid("M02_ACTION_NOT_ALLOWED", "anchorRecordId", "请选中状态适合本次操作的末条记录")
	}
	if !foundation.ValidDate(date) {
		return invalid("M02_INVALID_INPUT", "effectiveDate", "请选择有效的生效日期")
	}
	if date < w.Min || date > w.Max {
		return invalid("M02_TIMELINE_INVALID", "effectiveDate", "生效日期须晚于末条记录日期并在允许范围内")
	}
	return nil
}

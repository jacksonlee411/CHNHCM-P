package dated

import (
	"slices"

	"chnhcm.local/chnhcm/internal/foundation"
)

// AdditionWindow[T] describes the one selected gap in the current source.
// Empty defaults never mean that a different gap may be chosen implicitly.
type AdditionWindow[T any] struct {
	Anchor            Record[T]
	Next              string
	Mode              string
	Available         bool
	Min, Max, Default string
}

func (k Kernel[T, P]) PrepareAddition(base []Record[T], anchor, minDate, today string) (AdditionWindow[T], error) {
	rows := orderedView(base)
	n := slices.IndexFunc(rows, func(r Record[T]) bool { return r.ID == anchor })
	if n < 0 {
		return AdditionWindow[T]{}, invalid("M02_INVALID_INPUT", "anchorRecordId", "请选择当前来源中的记录作为新增位置")
	}
	out := AdditionWindow[T]{Anchor: rows[n], Mode: "APPEND"}
	out.Anchor.Data = k.Policy.Clone(out.Anchor.Data)
	if n+1 < len(rows) {
		out.Mode = "INSERT"
		out.Next = rows[n+1].ID
	}
	min := NextDate(rows[n].Date, 1)
	if min == "" || rows[n].State != "active" {
		return out, nil
	}
	if min < minDate {
		min = minDate
	}
	max := ""
	if out.Next != "" {
		max = NextDate(rows[n+1].Date, -1)
		if max == "" || min > max {
			return out, nil
		}
	}
	out.Available = true
	out.Min = min
	out.Max = max
	date := NextDate(rows[n].Date, 1)
	if out.Mode == "APPEND" && date < today {
		date = today
	}
	if foundation.ValidDate(date) && date >= min && (max == "" || date <= max) {
		out.Default = date
	}
	return out, nil
}

func (k Kernel[T, P]) ValidateAdditionPosition(base []Record[T], anchor, minDate, date string) error {
	if !foundation.ValidID(anchor) {
		return invalid("M02_INVALID_INPUT", "anchorRecordId", "请选择有效的新增位置")
	}
	w, e := k.PrepareAddition(base, anchor, minDate, date)
	if e != nil {
		return e
	}
	if !foundation.ValidDate(date) {
		return invalid("M02_INVALID_INPUT", "effectiveDate", "请选择有效的生效日期")
	}
	for _, r := range base {
		if r.Date == date {
			return invalid("M02_DATE_CONFLICT", "effectiveDate", "同一日期已有生效记录")
		}
	}
	if !w.Available {
		if w.Anchor.State != "active" {
			return invalid("M02_TIMELINE_INVALID", "effectiveDate", "停用期间不能新增资料变更")
		}
		return invalid("M02_INVALID_INPUT", "effectiveDate", "所选位置没有可新增日期")
	}
	if date < w.Min || w.Max != "" && date > w.Max {
		return invalid("M02_INVALID_INPUT", "effectiveDate", "生效日期须在所选记录与紧邻后一条之间")
	}
	return nil
}

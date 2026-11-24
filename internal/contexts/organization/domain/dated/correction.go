package dated

import (
	"slices"

	"chnhcm.local/chnhcm/internal/foundation"
)

// CorrectionBounds preserves the target's position while allowing its original date.
func (k Kernel[T, P]) CorrectionBounds(base []Record[T], target, minDate string) (min, max string, err error) {
	rows := orderedView(base)
	n := slices.IndexFunc(rows, func(r Record[T]) bool { return r.ID == target })
	if n < 0 {
		return "", "", invalid("M02_RECORD_NOT_FOUND", "", "当前资料中没有该记录")
	}
	min = minDate
	if n > 0 {
		min = NextDate(rows[n-1].Date, 1)
	}
	if n+1 < len(rows) {
		max = NextDate(rows[n+1].Date, -1)
	}
	return
}

func (k Kernel[T, P]) ValidateCorrectionDate(base []Record[T], target, minDate, date string) error {
	for _, r := range base {
		if r.ID != target && r.Date == date {
			return invalid("M02_DATE_CONFLICT", "newEffectiveDate", "同一日期已有记录")
		}
	}
	min, max, e := k.CorrectionBounds(base, target, minDate)
	if e != nil {
		return e
	}
	if !foundation.ValidDate(date) || date < min || max != "" && date > max {
		return invalid("M02_INVALID_INPUT", "newEffectiveDate", "日期须在原相邻记录之间")
	}
	return nil
}

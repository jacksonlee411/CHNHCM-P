package dated

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"slices"
	"time"
)

func NextDate(date string, days int) string {
	d, e := time.Parse("2006-01-02", date)
	if e != nil {
		return ""
	}
	v := d.AddDate(0, 0, days).Format("2006-01-02")
	if !foundation.ValidDate(v) {
		return ""
	}
	return v
}
func (k Kernel[T, P]) PrepareLifecyclePair(base []Record[T], anchor, minDate string) (AdditionWindow[T], error) {
	w, e := k.PrepareAddition(base, anchor, minDate, "")
	if e != nil || !w.Available {
		return w, e
	}
	if w.Max == "" {
		w.Max = "9999-12-31"
	}
	second := NextDate(w.Min, 1)
	w.Available = second != "" && second <= w.Max
	if !w.Available {
		w.Default = ""
	}
	return w, nil
}
func (k Kernel[T, P]) ValidateLifecyclePair(base []Record[T], anchor, minDate, stop, resume string) error {
	if e := k.ValidateAdditionPosition(base, anchor, minDate, stop); e != nil {
		return e
	}
	if e := k.ValidateAdditionPosition(base, anchor, minDate, resume); e != nil {
		return e
	}
	if stop >= resume {
		return invalid("M02_INVALID_INPUT", "reactivationDate", "重新启用日期须晚于停用日期")
	}
	return nil
}
func (k Kernel[T, P]) LifecyclePair(base []Record[T], selected string) (stop, resume Record[T], err error) {
	rows := orderedView(base)
	n := slices.IndexFunc(rows, func(r Record[T]) bool { return r.ID == selected })
	if n >= 0 && rows[n].Type == "REACTIVATE" {
		n--
	}
	if n < 0 || n+1 >= len(rows) || rows[n].Type != "DEACTIVATE" || rows[n+1].Type != "REACTIVATE" {
		return stop, resume, invalid("M02_PAIR_NOT_FOUND", "", "所选记录没有完整的停用与重新启用配对")
	}
	stop, resume = rows[n], rows[n+1]
	stop.Data, resume.Data = k.Policy.Clone(stop.Data), k.Policy.Clone(resume.Data)
	return stop, resume, nil
}

type DataChange[T any] struct {
	Start, End    string
	Before, After Record[T]
}

// Changes follow actual source periods, including later inherited DEACTIVATE data.
func (k Kernel[T, P]) DataChanges(ctx context.Context, before, after []Record[T]) ([]DataChange[T], error) {
	return k.dataChanges(ctx, before, after, k.Policy.EqualData)
}

// DataChangesWithAdoptions retains intervals whose reference versions differ.
// The caller projects each side under current read permissions to decide
// whether the adopted labels actually differ for this reader.
func (k Kernel[T, P]) DataChangesWithAdoptions(ctx context.Context, before, after []Record[T]) ([]DataChange[T], error) {
	return k.dataChanges(ctx, before, after, func(a, b T) bool {
		return k.Policy.EqualData(a, b) && k.Policy.EqualRevision(a, b)
	})
}

func (k Kernel[T, P]) dataChanges(ctx context.Context, before, after []Record[T], equal func(T, T) bool) ([]DataChange[T], error) {
	dates := []string{}
	for _, r := range before {
		dates = append(dates, r.Date)
	}
	for _, r := range after {
		dates = append(dates, r.Date)
	}
	slices.Sort(dates)
	dates = slices.Compact(dates)
	out := []DataChange[T]{}
	bi, ai := -1, -1
	for i, start := range dates {
		if e := ctx.Err(); e != nil {
			return nil, e
		}
		for bi+1 < len(before) && before[bi+1].Date <= start {
			bi++
		}
		for ai+1 < len(after) && after[ai+1].Date <= start {
			ai++
		}
		if bi < 0 || ai < 0 {
			continue
		}
		b, a := before[bi], after[ai]
		if equal(b.Data, a.Data) {
			continue
		}
		end := ""
		if i+1 < len(dates) {
			end = dates[i+1]
		}
		n := len(out)
		if n > 0 && out[n-1].End == start && equal(out[n-1].Before.Data, b.Data) && equal(out[n-1].After.Data, a.Data) {
			out[n-1].End = end
		} else {
			out = append(out, DataChange[T]{Start: start, End: end, Before: k.CloneRecords([]Record[T]{b})[0], After: k.CloneRecords([]Record[T]{a})[0]})
		}
	}
	return out, nil
}

package dated

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"errors"
)

// CoverageRange is a half-open active period; an empty End is unbounded.
type CoverageRange struct{ Start, End string }

func ActiveCoverage[T any](rows []Record[T]) ([]CoverageRange, bool) {
	if len(rows) == 0 {
		return nil, false
	}
	out := []CoverageRange{}
	for i, r := range rows {
		if !foundation.ValidDate(r.Date) || (r.State != "active" && r.State != "inactive") ||
			(r.End != "" && (!foundation.ValidDate(r.End) || r.End <= r.Date)) ||
			(i+1 < len(rows) && r.End != rows[i+1].Date) || (i+1 == len(rows) && r.End != "") {
			return nil, false
		}
		if r.State != "active" {
			continue
		}
		if len(out) > 0 && out[len(out)-1].End == r.Date {
			out[len(out)-1].End = r.End
		} else {
			out = append(out, CoverageRange{r.Date, r.End})
		}
	}
	return out, true
}

func RemovedCoverage[T any](before, after []Record[T]) ([]CoverageRange, error) {
	a, av := ActiveCoverage(before)
	b, bv := ActiveCoverage(after)
	if !av || !bv {
		return nil, errors.New("object coverage requires complete valid timelines")
	}
	lost := []CoverageRange{}
	j := 0
	for _, old := range a {
		start, covered := old.Start, false
		for j < len(b) && b[j].End != "" && b[j].End <= start {
			j++
		}
		for j < len(b) && (old.End == "" || b[j].Start < old.End) {
			if b[j].Start > start {
				lost = append(lost, CoverageRange{start, b[j].Start})
			}
			if b[j].End == "" || old.End != "" && b[j].End >= old.End {
				covered = true
				break
			}
			start = b[j].End
			j++
		}
		if !covered {
			lost = append(lost, CoverageRange{start, old.End})
		}
	}
	return lost, nil
}

// Intersects uses business boundaries, never a query adapter's approximation.
func (w CoverageRange) Intersects(start, end string) bool {
	return (w.End == "" || start < w.End) && (end == "" || w.Start < end)
}

// Covered validates a complete use interval against merged active periods.
func Covered(periods []CoverageRange, start, end string) bool {
	for _, p := range periods {
		if p.Start <= start && (p.End == "" || end != "" && p.End >= end) {
			return true
		}
	}
	return false
}

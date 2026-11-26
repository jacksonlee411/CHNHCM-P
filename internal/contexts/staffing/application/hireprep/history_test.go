package hireprep

import (
	"reflect"
	"testing"

	staff "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	f "chnhcm.local/chnhcm/internal/foundation"
)

func TestInsertedLocationPreservesIndependentTransfer(t *testing.T) {
	term := f.Period{From: "2027-04-01"}
	base := []Placement{{Period: term, Department: "研发部", Job: "工程师", Location: &LocationRef{ID: "甲"}}}
	may := HistoryEntry{ID: "first", Revision: "may", Kind: "DataChange", Day: "2027-05-01", Location: &LocationRef{ID: "乙"}}
	june := HistoryEntry{ID: "insert", Revision: "june", Kind: "DataChange", Day: "2027-06-01", Location: &LocationRef{ID: "丙"}}
	july := HistoryEntry{ID: "transfer", Revision: "july", Kind: "Transfer", Day: "2027-07-01", Transfer: &JobTransfer{Department: "产品部", Job: "产品专员", Segments: []staff.AssignmentSegment{{Period: f.Period{From: "2027-07-01"}}}}}
	calc := func(rows ...HistoryEntry) HistoryProjection {
		t.Helper()
		out, e := ProjectHistory(term, base, rows)
		if e != nil {
			t.Fatal(e)
		}
		return out
	}
	current := calc(july, may, june)
	if !reflect.DeepEqual(current, calc(may, june, july)) {
		t.Fatal("录入顺序改变了日状态")
	}
	tail := current.Segments[len(current.Segments)-1]
	if len(current.Segments) != 4 || tail.From != july.Day || tail.Department != "产品部" || tail.Location.ID != "丙" || current.Changes[june.ID].Before.Location.ID != "乙" {
		t.Fatal(current)
	}
	june.Removed = true
	tail = calc(may, june, july).Segments[2]
	if tail.Location.ID != "乙" || tail.Job != "产品专员" {
		t.Fatal(tail)
	}
	july.Requires = &RequiredFact{Action: june.ID, Revision: june.Revision, Location: "丙"}
	if _, e := ProjectHistory(term, base, []HistoryEntry{july, may, june}); e == nil {
		t.Fatal("误放行依赖缺失的后继")
	}
	june.Removed = false
	june.Day = "2027-08-01"
	if _, e := ProjectHistory(term, base, []HistoryEntry{july, may, june}); e == nil {
		t.Fatal("改日后依赖先后颠倒")
	}
}

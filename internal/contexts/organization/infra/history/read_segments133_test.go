package history

import (
	"fmt"
	"testing"
	"time"

	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
)

func segment133Fixture(n int) d.Timeline {
	t := d.Timeline{First: "0"}
	start := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < n; i++ {
		date := start.AddDate(0, 0, i*2).Format("2006-01-02")
		end := start.AddDate(0, 0, i*2+1).Format("2006-01-02")
		t.Records = append(t.Records, d.Record{ID: fmt.Sprint(i), Date: date, State: "active"})
		t.Edges = append(t.Edges, d.Edge{ID: fmt.Sprint(i), Start: date, End: end})
	}
	return t
}

func TestSegments133GapsAndInvalidOverlap(t *testing.T) {
	timeline := segment133Fixture(100)
	segments, err := departmentSegments(timeline)
	if err != nil {
		t.Fatal(err)
	}
	if len(segments) != 201 {
		t.Fatal("lost boundaries", len(segments))
	}
	for i, s := range segments[1:] {
		if i%2 == 0 {
			if len(s.department.Edges) != 1 || s.department.Edges[0].ID != fmt.Sprint(i/2) {
				t.Fatal("wrong active edge", i)
			}
		} else if len(s.department.Edges) != 0 {
			t.Fatal("edge leaked into gap", i)
		}
	}
	timeline.Edges[1].Start = timeline.Edges[0].Start
	if _, err := departmentSegments(timeline); err == nil {
		t.Fatal("overlap accepted")
	}
}

func BenchmarkSegments133(b *testing.B) {
	for _, n := range []int{100, 1000, 10000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			t := segment133Fixture(n)
			b.ReportAllocs()
			for b.Loop() {
				if _, err := departmentSegments(t); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

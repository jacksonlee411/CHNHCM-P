package history

import (
	"context"
	"reflect"
	"testing"

	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history/model"
	"chnhcm.local/chnhcm/internal/platform/compact"
)

func TestReadSegmentCoalescingPreservesFutureFirstAndBusinessDate(t *testing.T) {
	timeline := d.Timeline{ID: "00000000-0000-4000-8000-000000000001", Enterprise: "00000000-0000-4000-8000-000000000002", First: "00000000-0000-4000-8000-000000000003", Source: "00000000-0000-4000-8000-000000000004", RelationSource: "00000000-0000-4000-8000-000000000005", Records: []d.Record{
		{ID: "00000000-0000-4000-8000-000000000003", Revision: "00000000-0000-4000-8000-000000000006", Date: "2030-01-01", State: "active", Data: d.Data{Name: "未来首条", BU: "00000000-0000-4000-8000-000000000007"}},
	}}
	segments, err := departmentSegments(timeline)
	if err != nil {
		t.Fatal(err)
	}
	p := compact.NewPool(compact.Values{})
	var encoded []encodedReadSegment
	for _, segment := range segments {
		p.ResetIDs()
		data, err := EncodeDepartmentRead(context.Background(), p, model.ReadDepartment(segment.department))
		if err != nil {
			t.Fatal(err)
		}
		query, err := departmentQuery(segment.department, segment.start)
		if err != nil {
			t.Fatal(err)
		}
		encoded = appendReadSegment(encoded, encodedReadSegment{start: segment.start, end: segment.end, data: data, ids: p.IDs, query: query})
	}
	if len(encoded) != 2 || encoded[0].start != "" || encoded[0].end != "2030-01-01" || encoded[1].start != "2030-01-01" || encoded[1].end != "" || encoded[0].query.state != "not_effective" || encoded[1].query.state != "active" {
		t.Fatal("query modes lost the true first-date boundary", encoded)
	}
	values := p.Values
	values.IDs = encoded[0].ids
	read, err := DecodeDepartmentRead(context.Background(), values, encoded[0].data)
	if err != nil {
		t.Fatal(err)
	}
	decoded := read.Domain()
	for _, date := range []string{"1900-01-01", "2029-12-31", "2030-01-01", "2030-01-02", "9999-12-31"} {
		if d.Status(timeline, date) != d.Status(decoded, date) || !reflect.DeepEqual(d.ScopeRecord(timeline, date), d.ScopeRecord(decoded, date)) {
			t.Fatalf("merging a lookup span changed business facts on %s", date)
		}
	}
	if decoded.Records[0].Date != "2030-01-01" {
		t.Fatal("lookup coverage replaced the real first date")
	}
}

func TestReadSegmentCoalescingRequiresExactContentAndAdjacentCoverage(t *testing.T) {
	first := encodedReadSegment{end: "2030-01-01", data: []byte{1, 2, 3}, ids: []string{"first", "second"}}
	for _, tc := range []struct {
		name string
		next encodedReadSegment
	}{
		{"different_structure", encodedReadSegment{start: "2030-01-01", data: []byte{1, 2, 4}, ids: []string{"first", "second"}}},
		{"different_identity", encodedReadSegment{start: "2030-01-01", data: []byte{1, 2, 3}, ids: []string{"first", "third"}}},
		{"different_order", encodedReadSegment{start: "2030-01-01", data: []byte{1, 2, 3}, ids: []string{"second", "first"}}},
		{"gap", encodedReadSegment{start: "2030-01-02", data: []byte{1, 2, 3}, ids: []string{"first", "second"}}},
		{"overlap", encodedReadSegment{start: "2029-12-31", data: []byte{1, 2, 3}, ids: []string{"first", "second"}}},
		{"empty", encodedReadSegment{start: "2030-01-01", end: "2030-01-01", data: []byte{1, 2, 3}, ids: []string{"first", "second"}}},
		{"reversed", encodedReadSegment{start: "2030-01-01", end: "2029-12-31", data: []byte{1, 2, 3}, ids: []string{"first", "second"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := appendReadSegment([]encodedReadSegment{first}, tc.next); len(got) != 2 {
				t.Fatal("merged different facts or non-adjacent coverage")
			}
		})
	}
	input := []string{"first", "second"}
	got := appendReadSegment(nil, encodedReadSegment{data: []byte{1}, ids: input})
	input[0] = "overwritten by next encoding"
	if got[0].ids[0] != "first" {
		t.Fatal("stored segment borrowed the encoder's reusable ID directory")
	}
}

func TestReadSegmentsPreserveFutureFirstStoppedAndRelationBoundaries(t *testing.T) {
	timeline := d.Timeline{ID: "department", First: "first", Source: "source", RelationSource: "relation", Records: []d.Record{
		{ID: "first", Date: "2020-01-01", End: "2021-01-01", State: "active", Data: d.Data{Name: "甲", BU: "bu", Parent: "p"}},
		{ID: "stop", Date: "2021-01-01", End: "2022-01-01", State: "inactive", Data: d.Data{Name: "甲", BU: "bu", Parent: "p"}},
		{ID: "restart", Date: "2022-01-01", State: "active", Data: d.Data{Name: "乙", BU: "bu", Parent: "q"}},
	}, Edges: []d.Edge{{ID: "e1", Start: "2020-01-01", End: "2020-07-01", BU: "bu", Parent: "p"}, {ID: "e2", Start: "2020-07-01", End: "2021-01-01", BU: "bu", Parent: "p"}, {ID: "e3", Start: "2022-01-01", BU: "bu", Parent: "q"}}}
	segments, err := departmentSegments(timeline)
	if err != nil {
		t.Fatal(err)
	}
	for _, date := range []string{"1900-01-01", "2019-12-31", "2020-01-01", "2020-06-30", "2020-07-01", "2020-12-31", "2021-01-01", "2021-12-31", "2022-01-01", "9999-12-31"} {
		matches := 0
		for _, s := range segments {
			if (s.start == "" || s.start <= date) && (s.end == "" || date < s.end) {
				matches++
				if !reflect.DeepEqual(d.ScopeRecord(timeline, date), d.ScopeRecord(s.department, date)) {
					t.Fatalf("member differs on %s", date)
				}
				if s.department.Records[0].ID != timeline.First || len(s.department.Records) > 2 {
					t.Fatal("lost first member or retained whole timeline")
				}
				if d.Status(timeline, date) != d.Status(s.department, date) {
					t.Fatalf("status differs on %s", date)
				}
				if date >= timeline.Records[0].Date && d.ScopeRecord(timeline, date).State == "active" {
					expected := []d.Edge{}
					for _, e := range timeline.Edges {
						if e.Start <= date && (e.End == "" || date < e.End) {
							expected = append(expected, e)
						}
					}
					if !reflect.DeepEqual(expected, s.department.Edges) {
						t.Fatalf("edge differs on %s", date)
					}
				}
			}
		}
		if matches != 1 {
			t.Fatalf("%s covered %d times", date, matches)
		}
	}
}

package department

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestReferenceProbePreservesFutureDescendantAndCoverageRules(t *testing.T) {
	root := Timeline{ID: "root", Records: []Record{first("r", "2020-01-01", "", "根", "")}}
	target := Timeline{ID: "target", Records: []Record{first("t", "2020-01-01", "", "目标", "")}}
	child := Timeline{ID: "child", Records: []Record{first("c", "2020-01-01", "2027-01-01", "子", ""), row("cf", "2027-01-01", "", "子", "target")}}
	stopped := Timeline{ID: "stopped", Records: []Record{first("p", "2020-01-01", "2028-01-01", "上级", ""), {ID: "ps", Date: "2028-01-01", State: "inactive"}}}
	s := Snapshot{Departments: []Timeline{root, target, child, stopped}, BusinessUnits: []BusinessUnit{{ID: testBU, Records: []BURecord{{Date: "2020-01-01", State: "active"}}}, {ID: otherBU, Records: []BURecord{{Date: "2020-01-01", State: "active"}}}}}.Indexed()
	original := Kernel.CloneRecords(target.Records)
	validator := NewReferenceValidator(s, target.ID)
	for _, tc := range []struct{ name, parent, bu, code string }{
		{"root", "root", testBU, ""},
		{"future cycle", "child", testBU, "M02_DEPARTMENT_HIERARCHY_CYCLE"},
		{"future stopped parent", "stopped", testBU, "M02_DEPARTMENT_PARENT_INVALID"},
		{"future child across BU", "", otherBU, "M02_DEPARTMENT_PARENT_INVALID"},
		{"missing parent", "absent", testBU, "M02_DEPARTMENT_PARENT_INVALID"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			candidate := target
			candidate.Records = Kernel.CloneRecords(target.Records)
			candidate.Records[0].Data.Parent, candidate.Records[0].Data.BU = tc.parent, tc.bu
			err := validator.Validate(context.Background(), candidate)
			var q *QualificationError
			if tc.code == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if !errors.As(err, &q) || q.Code != tc.code {
				t.Fatalf("qualification: %v", err)
			}
			// The original complete-forest rule is also the command-side oracle.
			full := s.With(candidate)
			want := Validate(context.Background(), full, Discover(full, target.ID, nil, nil))
			if !reflect.DeepEqual(err, want) {
				t.Fatalf("probe and command differ: %v / %v", err, want)
			}
		})
	}
	if !reflect.DeepEqual(s.Department(target.ID).Records, original) {
		t.Fatal("candidate modified frozen facts")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if !errors.Is(validator.Validate(ctx, target), context.Canceled) {
		t.Fatal("canceled probe continued")
	}
}

func TestReferenceProbeNewTargetAndFutureBUStop(t *testing.T) {
	s := Snapshot{Departments: []Timeline{{ID: "p", Records: []Record{first("p", "2020-01-01", "", "上级", "")}}}, BusinessUnits: []BusinessUnit{{ID: testBU, Records: []BURecord{{Date: "2020-01-01", End: "2028-01-01", State: "active"}, {Date: "2028-01-01", State: "inactive"}}}}}.Indexed()
	candidate := Timeline{ID: "new", Records: []Record{first("n", "2026-01-01", "", "新建", "p")}}
	var q *QualificationError
	if err := NewReferenceValidator(s, candidate.ID).Validate(context.Background(), candidate); !errors.As(err, &q) || q.Code != "M02_DEPARTMENT_BU_CONFLICT" || q.Start != "2028-01-01" {
		t.Fatalf("future BU coverage lost: %v", err)
	}
	if s.Department("new") != nil {
		t.Fatal("new probe leaked into original snapshot")
	}
}

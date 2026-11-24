package department

import (
	"context"
	"errors"
	"testing"
)

func TestPreparedParentChoicesMatchCompleteRuleAcrossDatedGraphs(t *testing.T) {
	ctx := context.Background()
	// Exhaustive two-period graphs include changing BUs, stopped parents,
	// simultaneous cycles, safe union-graph cycles and future descendants.
	for shape := 0; shape < 256; shape++ {
		base := Timeline{ID: "target", First: "t0", Records: []Record{first("t0", "2020-01-01", "2027-01-01", "目标", ""), row("t1", "2027-01-01", "", "目标", "")}}
		a := Timeline{ID: "a", Records: []Record{first("a0", "2020-01-01", "2027-01-01", "甲", ""), row("a1", "2027-01-01", "", "甲", "")}}
		b := Timeline{ID: "b", Records: []Record{first("b0", "2020-01-01", "2027-01-01", "乙", ""), row("b1", "2027-01-01", "", "乙", "")}}
		if shape&1 != 0 {
			a.Records[0].Data.Parent = "b"
		}
		if shape&2 != 0 {
			a.Records[1].Data.Parent = "target"
		}
		if shape&4 != 0 {
			b.Records[0].Data.Parent = "a"
		}
		if shape&8 != 0 {
			b.Records[1].Data.Parent = "a"
		}
		if shape&16 != 0 {
			a.Records[1].State = "inactive"
		}
		if shape&32 != 0 {
			b.Records[0].Data.BU = otherBU
		}
		if shape&64 != 0 {
			a.Records[0].Data.Parent = "missing"
			a.Records[0].State = "inactive"
		}
		child := Timeline{ID: "child", Records: []Record{first("c0", "2028-01-01", "", "子", "target")}}
		if shape&128 != 0 {
			child.Records[0].Data.BU = otherBU
		}
		s := Snapshot{Departments: []Timeline{base, a, b, child}, BusinessUnits: []BusinessUnit{{ID: testBU, Records: []BURecord{{Date: "2020-01-01", State: "active"}}}, {ID: otherBU, Records: []BURecord{{Date: "2020-01-01", State: "active"}}}}}.Indexed()
		for _, variable := range []map[string]bool{{"t0": true}, {"t1": true}, {"t0": true, "t1": true}} {
			prepared, err := NewParentChoices(ctx, s, base, variable)
			if err != nil {
				var q *QualificationError
				if !errors.As(err, &q) {
					t.Fatal(err)
				}
			}
			for _, parent := range []string{"a", "b", "child", "target", "missing"} {
				got := false
				if prepared != nil {
					got, err = prepared.Eligible(ctx, parent)
					if err != nil {
						t.Fatal(err)
					}
				}
				candidate := base
				candidate.Records = Kernel.CloneRecords(base.Records)
				for i := range candidate.Records {
					if variable[candidate.Records[i].ID] {
						candidate.Records[i].Data.Parent = parent
					}
				}
				full := s.With(candidate)
				want := Validate(ctx, full, Discover(full, base.ID, nil, nil)) == nil
				if got != want {
					t.Fatalf("shape=%d variable=%v parent=%s got=%v want=%v", shape, variable, parent, got, want)
				}
			}
		}
	}
}

func TestPreparedParentChecksAncestorsOutsideChildOverlapAndBUStops(t *testing.T) {
	stop := row("ts", "2027-01-01", "", "目标", "")
	stop.State = "inactive"
	base := Timeline{ID: "target", Records: []Record{first("t", "2026-01-01", "2027-01-01", "目标", ""), stop}}
	ancestor := Timeline{ID: "ancestor", Records: []Record{first("z0", "2020-01-01", "2025-01-01", "祖先", ""), row("z1", "2025-01-01", "", "祖先", "")}}
	ancestor.Records[0].Data.BU = otherBU
	parent := Timeline{ID: "parent", Records: []Record{first("p", "2026-01-01", "", "上级", "ancestor")}}
	s := Snapshot{Departments: []Timeline{base, ancestor, parent}, BusinessUnits: []BusinessUnit{{ID: testBU, Records: []BURecord{{Date: "2020-01-01", State: "active"}}}, {ID: otherBU, Records: []BURecord{{Date: "2021-01-01", State: "active"}}}}}.Indexed()
	// End the target before BU stop; the unrelated ancestor's earlier conflict
	// must still disqualify a parent whose own overlapping interval is valid.
	prepared, err := NewParentChoices(context.Background(), s, base, map[string]bool{"t": true})
	if err != nil {
		t.Fatal(err)
	}
	if got, err := prepared.Eligible(context.Background(), "parent"); err != nil || got {
		t.Fatal("earlier ancestor conflict ignored", got, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := prepared.Eligible(ctx, "ancestor"); !errors.Is(err, context.Canceled) {
		t.Fatal("canceled qualification continued")
	}
}

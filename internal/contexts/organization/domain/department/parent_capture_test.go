package department

import (
	"context"
	"testing"
)

func TestCapturedParentWindowsMatchFullCandidate(t *testing.T) {
	ctx := context.Background()
	comparisons := 0
	for shape := 0; shape < 256; shape++ {
		base := Timeline{ID: "target", First: "t0", Records: []Record{first("t0", "2020-01-01", "2027-01-01", "target", ""), row("t1", "2027-01-01", "", "target", "")}}
		a := Timeline{ID: "a", Records: []Record{first("a0", "2020-01-01", "2027-01-01", "a", ""), row("a1", "2027-01-01", "", "a", "")}}
		b := Timeline{ID: "b", Records: []Record{first("b0", "2020-01-01", "2027-01-01", "b", ""), row("b1", "2027-01-01", "", "b", "")}}
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
		child := Timeline{ID: "child", Records: []Record{first("c0", "2028-01-01", "", "child", "target")}}
		if shape&128 != 0 {
			child.Records[0].Data.BU = otherBU
		}
		s := Snapshot{Departments: []Timeline{base, a, b, child}, BusinessUnits: []BusinessUnit{{ID: testBU, Records: []BURecord{{Date: "2020-01-01", State: "active"}}}, {ID: otherBU, Records: []BURecord{{Date: "2020-01-01", State: "active"}}}}}.Indexed()
		if Validate(ctx, s, Dependencies{Departments: []string{"target", "a", "b", "child"}, BUs: []string{testBU, otherBU}}) != nil {
			continue
		}
		common := map[string]bool{}
		for _, id := range Discover(s, "target", nil, nil).Departments {
			common[id] = true
		}
		for _, variable := range []map[string]bool{{"t0": true}, {"t1": true}, {"t0": true, "t1": true}} {
			windows := ParentWindows(base, variable)
			narrow := Snapshot{BusinessUnits: s.BusinessUnits}
			for _, original := range s.Departments {
				next := original
				next.Records = nil
				for _, r := range original.Records {
					keep := common[original.ID]
					for _, w := range windows {
						keep = keep || (r.End == "" || w.Start < r.End) && (w.End == "" || r.Date < w.End)
					}
					if keep {
						next.Records = append(next.Records, r)
					}
				}
				narrow.Departments = append(narrow.Departments, next)
			}
			p, e := NewCapturedParentChoices(ctx, narrow.Indexed(), base, variable)
			if e != nil {
				t.Fatal(shape, e)
			}
			for _, parent := range []string{"a", "b", "child", "target", "missing"} {
				got, e := p.Eligible(ctx, parent)
				if e != nil {
					t.Fatal(e)
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
					t.Fatalf("shape=%d windows=%v parent=%s got=%v want=%v", shape, windows, parent, got, want)
				}
				comparisons++
			}
		}
	}
	if comparisons != 720 {
		t.Fatal("legal graph matrix changed", comparisons)
	}
	t.Logf("%d same-fact narrow-vs-complete comparisons", comparisons)
}

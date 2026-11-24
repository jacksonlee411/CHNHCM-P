//go:build integration && query131

package department

import (
	"context"
	"testing"
)

func TestQuery131WindowProofAgainstCompleteCandidate(t *testing.T) {
	ctx := context.Background()
	comparisons, valid := 0, 0
	// Only valid immutable stored graphs may supply the exclusion premise.
	// Invalid old/future components must be rejected before any interval pruning.
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
		storedValid := Validate(ctx, s, Dependencies{Departments: []string{"target", "a", "b", "child"}, BUs: []string{testBU, otherBU}}) == nil
		for _, variable := range []map[string]bool{{"t0": true}, {"t1": true}, {"t0": true, "t1": true}} {
			p, _, err := WindowChoices131(ctx, s, base, variable)
			if !storedValid {
				if err == nil {
					t.Fatal("invalid stored graph supplied exclusion proof", shape)
				}
				continue
			}
			if err != nil {
				t.Fatal(shape, err)
			}
			valid++
			for _, parent := range []string{"a", "b", "child", "target", "missing"} {
				got, err := p.Eligible(ctx, parent)
				if err != nil {
					t.Fatal(err)
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
				comparisons++
			}
		}
	}
	if comparisons == 0 {
		t.Fatal("no valid graphs checked")
	}
	t.Logf("%d valid window preparations, %d candidate comparisons; invalid stored graphs rejected before pruning", valid, comparisons)
}

func TestQuery131WindowProofKeepsIncomingAndFarFutureBoundaries(t *testing.T) {
	ctx := context.Background()
	base := Timeline{ID: "target", Records: []Record{first("t0", "2026-01-01", "2027-01-01", "目标", ""), row("t1", "2027-01-01", "", "目标", "")}}
	parent := Timeline{ID: "parent", Records: []Record{first("p0", "2020-01-01", "2025-01-01", "old", ""), row("p1", "2025-01-01", "2099-01-01", "current", ""), row("p2", "2099-01-01", "", "future", "")}}
	parent.Records[2].State = "inactive"
	s := Snapshot{Departments: []Timeline{base, parent}, BusinessUnits: []BusinessUnit{{ID: testBU, Records: []BURecord{{Date: "2020-01-01", State: "active"}}}}}.Indexed()
	p, stats, err := WindowChoices131(ctx, s, base, map[string]bool{"t0": true})
	if err != nil {
		t.Fatal(err)
	}
	got, err := p.Eligible(ctx, "parent")
	if err != nil || !got || stats.WindowRecords >= stats.OriginalRecords {
		t.Fatal("record starting before W lost or irrelevant history retained", got, stats, err)
	}
	p, _, err = WindowChoices131(ctx, s, base, map[string]bool{"t1": true})
	if err != nil {
		t.Fatal(err)
	}
	got, err = p.Eligible(ctx, "parent")
	if err != nil || got {
		t.Fatal("open end lost 2099 stop", got, err)
	}
	// A new edge can reach the target through a descendant even when that
	// descendant becomes active only in the distant future.
	child := Timeline{ID: "child", Records: []Record{first("c", "2026-01-01", "2090-01-01", "子", ""), row("cf", "2090-01-01", "", "子", "target")}}
	s.Departments = append(s.Departments, child)
	s = s.Indexed()
	p, _, err = WindowChoices131(ctx, s, base, map[string]bool{"t1": true})
	if err != nil {
		t.Fatal(err)
	}
	got, err = p.Eligible(ctx, "child")
	if err != nil || got {
		t.Fatal("future reverse cycle escaped", got, err)
	}
	// A one-day inactive interval is not removable as adjacent equal state.
	parent.Records = []Record{first("p", "2020-01-01", "2026-06-01", "", ""), row("stop", "2026-06-01", "2026-06-02", "", ""), row("start", "2026-06-02", "", "", "")}
	parent.Records[1].State = "inactive"
	s = s.With(parent)
	p, _, err = WindowChoices131(ctx, s, base, map[string]bool{"t0": true})
	if err != nil {
		t.Fatal(err)
	}
	got, err = p.Eligible(ctx, "parent")
	if err != nil || got {
		t.Fatal("one day hole escaped", got, err)
	}
}

func TestQuery131PreparedOperationPeriods(t *testing.T) {
	ctx := context.Background()
	original := Timeline{ID: "target", First: "t0", Records: []Record{first("t0", "2020-01-01", "2026-01-01", "目标", ""), row("stop", "2026-01-01", "2027-01-01", "目标", ""), row("resume", "2027-01-01", "", "目标", "")}}
	original.Records[1].State = "inactive"
	parent := Timeline{ID: "parent", Records: []Record{first("p0", "2020-01-01", "2026-08-01", "父", ""), row("p1", "2026-08-01", "", "父", "")}}
	parent.Records[1].State = "inactive"
	s := Snapshot{Departments: []Timeline{original, parent}, BusinessUnits: []BusinessUnit{{ID: testBU, Records: []BURecord{{Date: "2020-01-01", State: "active"}}}}}.Indexed()
	// Literal prepared final records stand for the common Go command output.
	// This tests the exclusion proof's boundary, not an alternate command engine.
	for _, tc := range []struct {
		name       string
		records    []Record
		variable   string
		start, end string
		eligible   bool
	}{
		{"create_open", []Record{first("new", "2026-01-01", "", "目标", "")}, "new", "2026-01-01", "", false},
		{"add_finite", []Record{first("t0", "2020-01-01", "2025-01-01", "目标", ""), row("added", "2025-01-01", "2026-01-01", "目标", ""), original.Records[1], original.Records[2]}, "added", "2025-01-01", "2026-01-01", true},
		{"correct_history", original.Records, "t0", "2020-01-01", "2026-01-01", true},
		{"redate_resume", []Record{original.Records[0], {ID: "stop", Date: "2026-01-01", End: "2026-06-01", State: "inactive", Data: Data{BU: testBU}}, row("resume", "2026-06-01", "", "目标", "")}, "resume", "2026-06-01", "", false},
		{"delete_stop", []Record{first("t0", "2020-01-01", "2027-01-01", "目标", ""), original.Records[2]}, "t0", "2020-01-01", "2027-01-01", false},
		{"reactivate_open", original.Records, "resume", "2027-01-01", "", false},
		{"fixed_period", []Record{first("t0", "2020-01-01", "2024-01-01", "目标", ""), {ID: "stop2", Date: "2024-01-01", End: "2025-01-01", State: "inactive", Data: Data{BU: testBU}}, row("resume2", "2025-01-01", "2026-01-01", "目标", ""), original.Records[1], original.Records[2]}, "resume2", "2025-01-01", "2026-01-01", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := original
			base.Records = Kernel.CloneRecords(tc.records)
			base.First = base.Records[0].ID
			p, _, e := WindowChoices131(ctx, s, base, map[string]bool{tc.variable: true})
			if e != nil {
				t.Fatal(e)
			}
			if len(p.windows) != 1 || p.windows[0].start != tc.start || p.windows[0].end != tc.end {
				t.Fatal("prepared window lost", p.windows)
			}
			got, e := p.Eligible(ctx, "parent")
			if e != nil || got != tc.eligible {
				t.Fatal(got, e)
			}
			for i := range base.Records {
				if base.Records[i].ID == tc.variable {
					base.Records[i].Data.Parent = "parent"
				}
			}
			full := s.With(base)
			want := Validate(ctx, full, Discover(full, base.ID, nil, nil)) == nil
			if want != tc.eligible {
				t.Fatal("literal expectation differs from full save", want)
			}
		})
	}
}

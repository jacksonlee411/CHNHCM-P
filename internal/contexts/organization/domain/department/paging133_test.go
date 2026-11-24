package department

import (
	"context"
	"fmt"
	"reflect"
	"testing"
)

// G1 early-stop prototype. Production capture/cursor wiring belongs to T03.
// The lookahead row is never used as the returned continuation anchor.
func page133(ids []string, after string, size int, qualify func(string) (bool, error)) (page []string, more bool, scanned int, err error) {
	for _, id := range ids {
		if id <= after {
			continue
		}
		scanned++
		ok, e := qualify(id)
		if e != nil {
			return nil, false, scanned, e
		}
		if !ok {
			continue
		}
		if len(page) == size {
			return page, true, scanned, nil
		}
		page = append(page, id)
	}
	return
}
func TestRemediation133QualifiedLookahead(t *testing.T) {
	ctx := context.Background()
	for _, eligibleCount := range []int{0, 20, 21, 137} {
		base := Timeline{ID: "target", First: "t", Records: []Record{first("t", "2020-01-01", "", "目标", "")}}
		s := Snapshot{Departments: []Timeline{base}, BusinessUnits: []BusinessUnit{{ID: testBU, Records: []BURecord{{Date: "2020-01-01", State: "active"}}}}}
		ids := []string{}
		want := []string{}
		for i := 0; i < 200; i++ {
			id := fmt.Sprintf("p%03d", i)
			ids = append(ids, id)
			r := first(id, "2020-01-01", "2099-01-01", "上级", "")
			later := row(id+"later", "2099-01-01", "", "上级", "")
			if i < 200-eligibleCount {
				later.State = "inactive"
			} // sparse prefix and far-future rejection
			s.Departments = append(s.Departments, Timeline{ID: id, First: id, Records: []Record{r, later}})
		}
		s = s.Indexed()
		// Independent full final-candidate validation, not the paging predicate.
		for _, id := range ids {
			candidate := base
			candidate.Records = Kernel.CloneRecords(base.Records)
			candidate.Records[0].Data.Parent = id
			full := s.With(candidate)
			if Validate(ctx, full, Discover(full, base.ID, nil, nil)) == nil {
				want = append(want, id)
			}
		}
		if len(want) != eligibleCount {
			t.Fatalf("fixture %d got %d", eligibleCount, len(want))
		}
		prepared, e := NewParentChoices(ctx, s, base, map[string]bool{"t": true})
		if e != nil {
			t.Fatal(e)
		}
		actual := []string{}
		anchor := ""
		for {
			page, more, _, e := page133(ids, anchor, 20, func(id string) (bool, error) { return prepared.Eligible(ctx, id) })
			if e != nil {
				t.Fatal(e)
			}
			actual = append(actual, page...)
			if !more {
				break
			}
			if len(page) != 20 {
				t.Fatal("false full page")
			}
			anchor = page[len(page)-1]
		}
		if !reflect.DeepEqual(want, actual) {
			t.Fatalf("omission/duplicate for %d", eligibleCount)
		}
	}
	ids := make([]string, 87343)
	for i := range ids {
		ids[i] = fmt.Sprintf("%08d", i)
	}
	p, more, n, e := page133(ids, "", 20, func(string) (bool, error) { return true, nil })
	if e != nil || !more || n != 21 || len(p) != 20 {
		t.Fatal("dense scan did not stop", n)
	}
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	_, _, _, e = page133(ids, "", 20, func(string) (bool, error) { return false, ctx.Err() })
	if e != context.Canceled {
		t.Fatal("cancel became empty success")
	}
	t.Log("87343 dense identities: 21 qualification calls; 0/20/21/137 sparse qualified results match full final-candidate validation, including year 2099")
}

package department

import (
	"context"
	"fmt"
	"math/rand"
	"reflect"
	"slices"
	"testing"
)

// Retain the exhaustive date/map oracle independently of the dense traversal.
// Dated moves, missing references, inactive intervals and cycles must report the
// same earliest conflict, including its department and dependency identities.
func TestDatedQualificationMatchesExhaustiveGraph(t *testing.T) {
	random := rand.New(rand.NewSource(125300))
	for sample := 0; sample < 400; sample++ {
		s := Snapshot{BusinessUnits: []BusinessUnit{{ID: testBU, Records: []BURecord{{Date: "1900-01-01", State: "active"}}}, {ID: otherBU, Records: []BURecord{{Date: "1900-01-01", State: "active"}}}}}
		for i := 0; i < 40; i++ {
			id := fmt.Sprintf("D%02d", i)
			parent := ""
			if i > 0 {
				parent = fmt.Sprintf("D%02d", random.Intn(i))
			}
			rows := []Record{first(id+"first", "2000-01-01", "", id, parent)}
			for j := 1; j <= 8; j++ {
				r := row(fmt.Sprintf("%s-%d", id, j), fmt.Sprintf("%04d-01-01", 2000+j), "", id+fmt.Sprint(j), parent)
				if sample%5 != 0 && i == sample%40 && j == 1+sample%8 {
					switch sample % 5 {
					case 1:
						r.Data.Parent = fmt.Sprintf("D%02d", random.Intn(41)) // cycles or missing
					case 2:
						r.State = "inactive"
					case 3:
						r.Data.BU = otherBU
					case 4:
						r.Data.BU = "missing-bu"
					}
				}
				rows[len(rows)-1].End = r.Date
				rows = append(rows, r)
			}
			s.Departments = append(s.Departments, Timeline{ID: id, Records: rows})
		}
		if sample%7 == 0 {
			s.BusinessUnits[0].Records = append(s.BusinessUnits[0].Records, BURecord{Date: "2007-06-01", State: "inactive"})
		}
		s = s.Indexed()
		deps := Dependencies{BUs: []string{testBU, otherBU}}
		for _, d := range s.Departments {
			deps.Departments = append(deps.Departments, d.ID)
		}
		// Snapshot.With must preserve candidate replacement, not validate the old head.
		if sample%3 == 0 {
			changed := s.Departments[0]
			changed.Records = slices.Clone(changed.Records)
			changed.Records[len(changed.Records)-1].Data.Parent = "D39"
			s = s.With(changed)
		}
		got, want := Validate(context.Background(), s, deps), exhaustiveMapQualification(context.Background(), s, deps)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("sample %d: got %#v, want %#v", sample, got, want)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s := Snapshot{BusinessUnits: []BusinessUnit{{ID: testBU, Records: []BURecord{{Date: "2000-01-01", State: "active"}}}}}
	if got := Validate(ctx, s, Dependencies{BUs: []string{testBU}}); got != context.Canceled {
		t.Fatalf("cancellation lost: %v", got)
	}
}

func exhaustiveMapQualification(ctx context.Context, s Snapshot, deps Dependencies) error {
	dates := []string{}
	for _, id := range deps.Departments {
		if t := s.Department(id); t != nil {
			for _, r := range t.Records {
				dates = append(dates, r.Date)
			}
		}
	}
	for _, id := range deps.BUs {
		if b := s.BU(id); b != nil {
			for _, r := range b.Records {
				dates = append(dates, r.Date)
			}
		}
	}
	slices.Sort(dates)
	dates = slices.Compact(dates)
	for _, date := range dates {
		if err := ctx.Err(); err != nil {
			return err
		}
		active := map[string]*Record{}
		for _, id := range deps.Departments {
			if t := s.Department(id); t != nil {
				if r := recordAt(t.Records, date); r != nil && r.State == "active" {
					active[id] = r
				}
			}
		}
		for _, id := range deps.Departments {
			r := active[id]
			if r == nil {
				continue
			}
			b := s.BU(r.Data.BU)
			if b == nil {
				return invalidRelation("M02_DEPARTMENT_BU_CONFLICT", id, r.Data.BU, date)
			}
			br := BUAt(b.Records, date)
			if !activeBURecord(br) {
				return invalidRelation("M02_DEPARTMENT_BU_CONFLICT", id, b.ID, date)
			}
			if r.Data.Parent != "" {
				p := active[r.Data.Parent]
				if !parentInBU(p, r.Data.BU) {
					return invalidRelation("M02_DEPARTMENT_PARENT_INVALID", id, r.Data.Parent, date)
				}
			}
		}
		done := map[string]bool{}
		for _, id := range deps.Departments {
			if active[id] == nil || done[id] {
				continue
			}
			path := map[string]bool{}
			next := id
			for next != "" && !done[next] {
				if path[next] {
					return invalidRelation("M02_DEPARTMENT_HIERARCHY_CYCLE", id, next, date)
				}
				path[next] = true
				r := active[next]
				if r == nil {
					break
				}
				next = r.Data.Parent
			}
			for n := range path {
				done[n] = true
			}
		}
	}
	return nil
}

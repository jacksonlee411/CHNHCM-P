//go:build integration && query131

package department

import "context"

type WindowStats131 struct{ OriginalRecords, WindowRecords, Windows int }

// WindowChoices131 is an executable exclusion proof, not a production loading
// path. Validate establishes the immutable-source invariant once; T03 must get
// that invariant from the verified same-source storage capture. A partial or
// corrupt snapshot cannot be passed off as that proof. The existing final
// candidate validator remains unchanged and still executes under write guards.
func WindowChoices131(ctx context.Context, s Snapshot, base Timeline, variable map[string]bool) (*ParentChoices, WindowStats131, error) {
	stats := WindowStats131{}
	deps := Dependencies{}
	for _, t := range s.Departments {
		deps.Departments = append(deps.Departments, t.ID)
		stats.OriginalRecords += len(t.Records)
	}
	for _, b := range s.BusinessUnits {
		deps.BUs = append(deps.BUs, b.ID)
		stats.OriginalRecords += len(b.Records)
	}
	if err := Validate(ctx, s.Indexed(), deps); err != nil {
		return nil, stats, err
	}
	p, err := NewParentChoices(ctx, s, base, variable)
	if err != nil {
		return nil, stats, err
	}
	stats.Windows = len(p.windows)
	if stats.Windows == 0 {
		stats.WindowRecords = stats.OriginalRecords
		return p, stats, nil
	}
	intersects := func(start, end string) bool {
		for _, w := range p.windows {
			if (end == "" || w.start < end) && (w.end == "" || start < w.end) {
				return true
			}
		}
		return false
	}
	narrow := Snapshot{}
	for _, t := range s.Departments {
		v := Timeline{ID: t.ID, Enterprise: t.Enterprise, First: t.First, Source: t.Source, RelationSource: t.RelationSource}
		for _, r := range t.Records {
			if intersects(r.Date, r.End) {
				// Names, log metadata, revisions and old source versions do not
				// participate in the existing at/coverage/cycle predicates.
				v.Records = append(v.Records, Record{ID: r.ID, Date: r.Date, End: r.End, State: r.State, Data: Data{BU: r.Data.BU, Parent: r.Data.Parent}})
			}
		}
		stats.WindowRecords += len(v.Records)
		narrow.Departments = append(narrow.Departments, v)
		p.history[t.ID] = 1 // The complete immutable original snapshot was proved above.
	}
	for _, b := range s.BusinessUnits {
		v := BusinessUnit{ID: b.ID, First: b.First, Source: b.Source}
		for _, r := range b.Records {
			if intersects(r.Date, r.End) {
				v.Records = append(v.Records, BURecord{ID: r.ID, Date: r.Date, End: r.End, State: r.State})
			}
		}
		stats.WindowRecords += len(v.Records)
		narrow.BusinessUnits = append(narrow.BusinessUnits, v)
	}
	narrow = narrow.ReadIndexed()
	narrow.replacement = &base
	p.s = narrow
	return p, stats, nil
}

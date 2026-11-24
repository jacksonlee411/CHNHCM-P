package department

import (
	"context"
	"sort"

	"chnhcm.local/chnhcm/internal/contexts/organization/domain/dated"
)

type parentWindow struct{ start, end, bu string }

// ParentWindow includes the covering predecessor and every registered future
// boundary. An empty End is unbounded; storage only retrieves facts.
type ParentWindow struct{ Start, End, BU string }

func ParentWindows(base Timeline, variable map[string]bool) []ParentWindow {
	out := []ParentWindow{}
	for _, r := range base.Records {
		if variable[r.ID] && r.State == "active" {
			out = append(out, ParentWindow{r.Date, r.End, r.Data.BU})
		}
	}
	return out
}

// NewCapturedParentChoices consumes same-source verified storage facts. The
// common target/descendant component is complete; other candidates carry only
// the requested windows. Successful immutable source writes already validated
// their unchanged relations, so unrelated candidate history is not rechecked.
func NewCapturedParentChoices(ctx context.Context, s Snapshot, base Timeline, variable map[string]bool) (*ParentChoices, error) {
	p, err := NewParentChoices(ctx, s, base, variable)
	if err != nil {
		return nil, err
	}
	for _, t := range s.Departments {
		p.history[t.ID] = 1
	}
	return p, nil
}

type qualificationSpan struct {
	start, end           string
	record               *Record
	valid, reachesTarget bool
}

// ParentChoices prepares the common candidate once, then reuses immutable
// qualification intervals within this query only. Commands still run Validate
// under their guards. The active-BU/parent predicates and half-open boundaries
// are the same rules; no permission or dated result survives the request.
type ParentChoices struct {
	s        Snapshot
	target   string
	windows  []parentWindow
	spans    map[string][]qualificationSpan
	history  map[string]uint8
	visiting map[string]bool
}

// AddCaptured extends a single cursor snapshot with a candidate's complete
// window/ancestor closure. Existing immutable facts and eligibility intervals
// are shared; the target's invariant component was validated exactly once.
func (p *ParentChoices) AddCaptured(s Snapshot) error {
	for _, t := range s.Departments {
		if old := p.s.Department(t.ID); old != nil {
			if old.Source != t.Source {
				return Fault("M02_DEPENDENCY_UNAVAILABLE", "", "选项已变化，请刷新")
			}
			continue
		}
		p.s.departmentIndex[t.ID] = len(p.s.Departments)
		p.s.Departments = append(p.s.Departments, t)
		p.history[t.ID] = 1
	}
	for _, b := range s.BusinessUnits {
		if old := p.s.BU(b.ID); old != nil {
			if old.Source != b.Source {
				return Fault("M02_DEPENDENCY_UNAVAILABLE", "", "选项已变化，请刷新")
			}
			continue
		}
		p.s.buIndex[b.ID] = len(p.s.BusinessUnits)
		p.s.BusinessUnits = append(p.s.BusinessUnits, b)
	}
	return nil
}

func NewParentChoices(ctx context.Context, s Snapshot, base Timeline, variable map[string]bool) (*ParentChoices, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	validator := NewReferenceValidator(s, base.ID)
	if err := validator.Validate(ctx, base); err != nil {
		return nil, err
	}
	s = validator.snapshot
	s.replacement = &base
	p := &ParentChoices{s: s, target: base.ID, spans: map[string][]qualificationSpan{}, history: map[string]uint8{}, visiting: map[string]bool{}}
	for i := range base.Records {
		r := &base.Records[i]
		if variable[r.ID] && r.State == "active" {
			p.windows = append(p.windows, parentWindow{r.Date, r.End, r.Data.BU})
		}
	}
	return p, nil
}

func later(a, b string) string {
	if a > b {
		return a
	}
	return b
}
func earlierEnd(a, b string) string {
	if a == "" || b != "" && b < a {
		return b
	}
	return a
}
func (p *ParentChoices) remember(id string, span qualificationSpan) qualificationSpan {
	values := p.spans[id]
	i := sort.Search(len(values), func(i int) bool { return values[i].start >= span.start })
	if i < len(values) && values[i].start == span.start {
		values[i] = span
	} else {
		values = append(values, qualificationSpan{})
		copy(values[i+1:], values[i:])
		values[i] = span
	}
	p.spans[id] = values
	return span
}

func (p *ParentChoices) at(ctx context.Context, id, date string) (qualificationSpan, error) {
	if err := ctx.Err(); err != nil {
		return qualificationSpan{}, err
	}
	values := p.spans[id]
	i := sort.Search(len(values), func(i int) bool { return values[i].start > date }) - 1
	if i >= 0 && (values[i].end == "" || date < values[i].end) {
		return values[i], nil
	}
	t := p.s.Department(id)
	span := qualificationSpan{valid: true}
	if t == nil {
		return p.remember(id, span), nil
	}
	r := recordAt(t.Records, date)
	if r == nil {
		if len(t.Records) > 0 {
			span.end = t.Records[0].Date
		}
		return p.remember(id, span), nil
	}
	span.start, span.end, span.record = r.Date, r.End, r
	if r.State != "active" {
		return p.remember(id, span), nil
	}
	if p.visiting[id] {
		return qualificationSpan{start: date, record: r, valid: false}, nil
	}
	p.visiting[id] = true
	defer delete(p.visiting, id)
	span.reachesTarget = id == p.target
	b := p.s.BU(r.Data.BU)
	var br *BURecord
	if b != nil {
		at := dated.IndexAt(b.Records, date, func(r *BURecord) string { return r.Date })
		if at >= 0 {
			br = &b.Records[at]
			span.start = later(span.start, br.Date)
			span.end = earlierEnd(span.end, br.End)
		} else if len(b.Records) > 0 {
			span.end = earlierEnd(span.end, b.Records[0].Date)
		}
	}
	if !activeBURecord(br) {
		span.valid = false
		return p.remember(id, span), nil
	}
	if r.Data.Parent != "" {
		parent, err := p.at(ctx, r.Data.Parent, date)
		if err != nil {
			return span, err
		}
		span.start = later(span.start, parent.start)
		span.end = earlierEnd(span.end, parent.end)
		span.valid = parent.valid && parentInBU(parent.record, r.Data.BU)
		span.reachesTarget = span.reachesTarget || parent.reachesTarget
	}
	return p.remember(id, span), nil
}

// Validate every active interval of every historical/future ancestor identity,
// including ancestors referenced only by inactive records. This preserves the
// complete closure checked by Discover+Validate, not just today's ancestry.
func (p *ParentChoices) completeHistory(ctx context.Context, id string) (bool, error) {
	if known := p.history[id]; known != 0 {
		return known == 1, nil
	}
	queue := []string{id}
	seen := map[string]bool{}
	for i := 0; i < len(queue); i++ {
		current := queue[i]
		if seen[current] || p.history[current] == 1 {
			continue
		}
		if p.history[current] == 2 {
			p.history[id] = 2
			return false, nil
		}
		seen[current] = true
		t := p.s.Department(current)
		if t == nil {
			continue
		}
		for j := range t.Records {
			r := &t.Records[j]
			if r.Data.Parent != "" {
				queue = append(queue, r.Data.Parent)
			}
			if r.State != "active" {
				continue
			}
			for date := r.Date; ; {
				span, err := p.at(ctx, current, date)
				if err != nil {
					return false, err
				}
				if !span.valid {
					p.history[current] = 2
					p.history[id] = 2
					return false, nil
				}
				if span.end == "" || r.End != "" && span.end >= r.End {
					break
				}
				if span.end <= date {
					return false, Fault("M02_DEPENDENCY_UNAVAILABLE", "", "关系期间不完整")
				}
				date = span.end
			}
		}
	}
	for current := range seen {
		p.history[current] = 1
	}
	return true, nil
}

func (p *ParentChoices) Eligible(ctx context.Context, parent string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	valid, err := p.completeHistory(ctx, parent)
	if err != nil || !valid {
		return valid, err
	}
	for _, window := range p.windows {
		for date := window.start; ; {
			span, err := p.at(ctx, parent, date)
			if err != nil {
				return false, err
			}
			if !span.valid || span.reachesTarget || !parentInBU(span.record, window.bu) {
				return false, nil
			}
			if span.end == "" || window.end != "" && span.end >= window.end {
				break
			}
			if span.end <= date {
				return false, Fault("M02_DEPENDENCY_UNAVAILABLE", "", "关系期间不完整")
			}
			date = span.end
		}
	}
	return true, nil
}

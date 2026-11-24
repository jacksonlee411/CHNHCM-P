package foundation

import "slices"

// Period represents business dates [From, Until), with an optional open end.
type Period struct {
	From  string `json:"from"`
	Until string `json:"until,omitempty"`
}

func (p Period) Valid() bool {
	return ValidDate(p.From) && (p.Until == "" || ValidDate(p.Until) && p.Until > p.From)
}
func (p Period) Includes(day string) bool   { return day >= p.From && (p.Until == "" || day < p.Until) }
func (p Period) Overlaps(other Period) bool { return p.Includes(other.From) || other.Includes(p.From) }

// Split returns a gapless partition, including the open tail when applicable.
func (p Period) Split(boundaries []string) []Period {
	days := []string{p.From}
	for _, day := range boundaries {
		if p.Includes(day) {
			days = append(days, day)
		}
	}
	slices.Sort(days)
	days = slices.Compact(days)
	out := make([]Period, len(days))
	for i, day := range days {
		end := p.Until
		if i+1 < len(days) {
			end = days[i+1]
		}
		out[i] = Period{day, end}
	}
	return out
}

type DatedSource struct {
	Version    string `json:"version"`
	RecordedAt string `json:"recordedAt"`
}
type DatedRules[T any] struct {
	Owner    string        `json:"owner"`
	Name     string        `json:"name"`
	Version  string        `json:"version"`
	Rows     []T           `json:"rows"`
	History  []DatedSource `json:"history"`
	CanWrite bool          `json:"canWrite"`
}
type RuleReceipt struct {
	SubmissionID string `json:"submissionId"`
	Outcome      string `json:"outcome"`
	Version      string `json:"version"`
	Failure      *Fault `json:"-"`
}
type RuleChange[T any] struct {
	InsertAfter string `json:"insertAfter,omitempty"`
	Expected    string `json:"expectedVersion"`
	Row         T      `json:"row"`
}

// Predecessor cuts only the selected interval and preserves its original end on the new segment.
func (p Period) Predecessor(next Period) (Period, bool) {
	if !next.Valid() || !p.Includes(next.From) || next.From == p.From || next.Until != p.Until {
		return Period{}, false
	}
	return Period{From: p.From, Until: next.From}, true
}

package domain

import (
	"chnhcm.local/chnhcm/internal/contexts/organization/domain/dated"
	"context"
)

func NextDate(date string, days int) string { return dated.NextDate(date, days) }
func PrepareLifecyclePair(base []Record, anchor, enterpriseStart string) (AdditionWindow, error) {
	w, e := buKernel.PrepareLifecyclePair(base, anchor, enterpriseStart)
	return w, e
}
func ValidateLifecyclePair(base []Record, anchor, enterpriseStart, stop, resume string) error {
	return buKernel.ValidateLifecyclePair(base, anchor, enterpriseStart, stop, resume)
}
func LifecyclePair(base []Record, selected string) (Record, Record, error) {
	a, b, e := buKernel.LifecyclePair(base, selected)
	return a, b, e
}

type NameChange struct {
	Start, End    string
	Before, After Record
}

func PairNameChanges(ctx context.Context, before, after []Record) ([]NameChange, error) {
	rows, e := buKernel.DataChanges(ctx, before, after)
	out := make([]NameChange, len(rows))
	for i, r := range rows {
		out[i] = NameChange{r.Start, r.End, r.Before, r.After}
	}
	return out, e
}

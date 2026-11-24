package domain

import (
	"chnhcm.local/chnhcm/internal/contexts/organization/domain/dated"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
)

// Record is a complete independent snapshot, except DEACTIVATE which inherits its
// immediate predecessor. IDs are allocated by persistence after validation.
type Record = dated.Record[BUData]
type Item = dated.Item[BUPatch]
type Range = dated.Range
type StructuralChange = dated.StructuralChange
type PeriodChange = dated.PeriodChange
type Impact = dated.Impact
type Candidate = dated.Candidate[BUData]

func ptr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
func invalid(code, field, message string) error {
	status := 409
	if code == "M02_INVALID_INPUT" || code == "M02_FORBIDDEN_INPUT" {
		status = 400
	}
	return &foundation.Fault{Status: status, Code: code, Field: field, Message: message}
}

func Calculate(ctx context.Context, base []Record, first, enterpriseStart string, items []Item) (Candidate, error) {
	c, err := buKernel.Calculate(ctx, base, first, enterpriseStart, items)
	return c, err
}
func Compare(ctx context.Context, before, after []Record, items []Item) Impact {
	return buKernel.Compare(ctx, before, after, items)
}
func FactAt(records []Record, date string) *Record {
	i := dated.IndexAt(records, date, func(r *Record) string { return r.Date })
	if i < 0 {
		return nil
	}
	v := records[i]
	v.Data.NameEn = (buPolicy{}).Clone(BUData{v.Data.Name, v.Data.NameEn}).NameEn
	return &v
}
func StatusAt(records []Record, date string) string {
	r := FactAt(records, date)
	if r == nil {
		return "not_effective"
	}
	return r.State
}

// CreateTimeline constructs the same complete snapshot and precise impact used by
// maintenance. The initial identity is allocated only after domain validation.
func CreateTimeline(ctx context.Context, input Create, enterpriseStart string) (Create, Candidate, error) {
	in, err := ValidateCreate(input, enterpriseStart)
	if err != nil {
		return in, Candidate{}, err
	}
	if err = ctx.Err(); err != nil {
		return in, Candidate{}, err
	}
	c, err := buKernel.Create(ctx, "candidate", in.EffectiveDate, enterpriseStart, in.OperationReason, BUData{in.Name, in.NameEn})
	return in, c, err
}

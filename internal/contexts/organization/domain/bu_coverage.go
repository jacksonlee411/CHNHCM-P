package domain

import (
	"chnhcm.local/chnhcm/internal/contexts/organization/domain/dated"
	"slices"
)

type BUCoverageRange = dated.CoverageRange

func buCoverage(rows []Record) ([]BUCoverageRange, bool) { return dated.ActiveCoverage(rows) }
func SameBUCoverage(before, after []Record) bool {
	a, av := buCoverage(before)
	b, bv := buCoverage(after)
	return av && bv && slices.Equal(a, b)
}
func RemovedBUCoverage(before, after []Record) ([]BUCoverageRange, error) {
	return dated.RemovedCoverage(before, after)
}

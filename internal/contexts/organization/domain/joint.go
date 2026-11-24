package domain

import "chnhcm.local/chnhcm/internal/contexts/organization/domain/dated"

type ItemFault = dated.ItemFault

func CandidateID(it Item) string { return dated.CandidateID(it) }
func DirectFacets(it Item, base, after []Record) []string {
	return buKernel.DirectFacets(it, base, after)
}

package domain

import "chnhcm.local/chnhcm/internal/contexts/organization/domain/dated"

func OperationReasons(kind string) map[string]string {
	return dated.OperationReasons(dated.BusinessUnit, kind)
}
func OperationLabel(kind string) string { return dated.OperationLabel(kind) }

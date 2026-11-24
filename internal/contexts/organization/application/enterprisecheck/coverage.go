package enterprisecheck

import (
	"chnhcm.local/chnhcm/internal/contexts/organization/domain/dated"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

func Records[T any](e org.Enterprise, rows []dated.Record[T]) error {
	periods := make([]dated.CoverageRange, len(e.ActivePeriods))
	for i, p := range e.ActivePeriods {
		periods[i] = dated.CoverageRange{Start: p.Start, End: p.End}
	}
	if len(rows) == 0 {
		return foundation.Fail(503, "M02_DEPENDENCY_UNAVAILABLE", "组织资料暂不可读")
	}
	if rows[0].Date < e.ValidFrom {
		return &foundation.Fault{Status: 409, Code: "M02_REFERENCE_CONFLICT", Field: "effectiveDate", Message: "生效日期不能早于企业开始日期"}
	}
	for _, r := range rows {
		if r.State == "active" && !dated.Covered(periods, r.Date, r.End) {
			return &foundation.Fault{Status: 409, Code: "M02_REFERENCE_CONFLICT", Field: "effectiveDate", Message: "企业启用期间不能完整覆盖本次组织资料"}
		}
	}
	return nil
}

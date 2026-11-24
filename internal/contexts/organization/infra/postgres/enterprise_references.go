package postgres

import (
	"chnhcm.local/chnhcm/internal/contexts/organization/application/enterprisecheck"
	"chnhcm.local/chnhcm/internal/contexts/organization/domain/dated"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/enterprise"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
	"context"
	"fmt"
)

func enterpriseProjection(t d.Timeline) (org.Enterprise, error) {
	periods, ok := dated.ActiveCoverage(t.Records)
	if !ok {
		return org.Enterprise{}, foundation.Fail(503, "M02_DEPENDENCY_UNAVAILABLE", "企业资料暂不可读")
	}
	e := org.Enterprise{ID: t.ID, Code: t.Code, ValidFrom: t.Records[0].Date, ActivePeriods: []org.EnterpriseActivePeriod{}}
	for _, r := range t.Records {
		e.Names = append(e.Names, org.EnterpriseName{Date: r.Date, Name: r.Data.Name})
	}
	for _, p := range periods {
		e.ActivePeriods = append(e.ActivePeriods, org.EnterpriseActivePeriod{Start: p.Start, End: p.End})
	}
	return e, nil
}
func (u *enterpriseUnit) ValidateReferences(ctx context.Context, before, after d.Timeline) error {
	lost, e := dated.RemovedCoverage(before.Records, after.Records)
	if e != nil {
		return e
	}
	if len(lost) == 0 && after.Records[0].Date <= before.Records[0].Date {
		return nil
	}
	ent, e := enterpriseProjection(after)
	if e != nil {
		return e
	}
	// Locations and legal entities use the enterprise first-date floor, not its active coverage.
	// Their writer holds the same enterprise row lock before validating dates.
	if after.Records[0].Date > before.Records[0].Date {
		if u.staffing == nil {
			return foundation.Fail(503, "M02_DEPENDENCY_UNAVAILABLE", "暂时无法核对职务配置的最早起日")
		}
		staffingFirst, err := u.staffing.EarliestStaffingDate(dbtx.With(ctx, u.tx), after.ID)
		if err != nil {
			return foundation.Fail(503, "M02_DEPENDENCY_UNAVAILABLE", "暂时无法核对职务配置的最早起日")
		}
		if staffingFirst != "" && staffingFirst < ent.ValidFrom {
			return &foundation.Fault{Status: 409, Code: "M02_REFERENCE_CONFLICT", Field: "effectiveDate", Message: "现有职务配置的首条日期早于企业候选首条日期，请先处理相关资料"}
		}
		var first *string
		if e = u.tx.QueryRow(ctx, `SELECT min(effective_date)::text FROM organization.location_read WHERE instance_id=$1 AND enterprise_id=$2`, u.instance, after.ID).Scan(&first); e != nil {
			return e
		}
		if first != nil && *first < ent.ValidFrom {
			return &foundation.Fault{Status: 409, Code: "M02_REFERENCE_CONFLICT", Field: "effectiveDate", Message: "现有地点的首条日期早于企业候选首条日期，请先处理相关地点"}
		}
		if e = u.tx.QueryRow(ctx, `SELECT min(effective_date)::text FROM organization.legal_entity_read WHERE instance_id=$1 AND enterprise_id=$2`, u.instance, after.ID).Scan(&first); e != nil {
			return e
		}
		if first != nil && *first < ent.ValidFrom {
			return &foundation.Fault{Status: 409, Code: "M02_REFERENCE_CONFLICT", Field: "effectiveDate", Message: "现有法律实体的首条日期早于企业候选首条日期，请先处理相关法律实体"}
		}
	}
	// Only qualification changes enumerate real dependencies. Head pages are
	// bounded; immutable projections are decoded and checked in Go.
	cursor := "00000000-0000-0000-0000-000000000000"
	for {
		rows, e := u.tx.Query(ctx, `SELECT bu_id::text,head_source_id::text FROM organization.business_unit WHERE instance_id=$1 AND bu_id>$2::uuid ORDER BY bu_id LIMIT 100`, u.instance, cursor)
		if e != nil {
			return e
		}
		heads := map[string]string{}
		for rows.Next() {
			var id, source string
			if e = rows.Scan(&id, &source); e != nil {
				rows.Close()
				return e
			}
			heads[id] = source
			cursor = id
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		if len(heads) == 0 {
			break
		}
		facts, e := u.currentFacts(ctx, history.BusinessUnit, heads)
		if e != nil {
			return e
		}
		for _, v := range facts.BUs {
			if e = enterprisecheck.Records(ent, v.Timeline.Records); e != nil {
				return &foundation.Fault{Status: 409, Code: "M02_REFERENCE_CONFLICT", Field: "effectiveDate", Message: "现有业务单元的有效期间超出企业候选范围，请先处理相关业务单元"}
			}
		}
	}
	windows := []string{}
	for _, w := range lost {
		windows = append(windows, fmt.Sprintf("[%s,%s)", w.Start, w.End))
	}
	cursor = "00000000-0000-0000-0000-000000000000"
	for {
		rows, e := u.tx.Query(ctx, `SELECT DISTINCT d.department_id::text,d.head_source_id::text FROM organization.department d JOIN organization.department_read_segment r ON r.instance_id=d.instance_id AND r.object_id=d.department_id AND r.source_id=d.head_source_id WHERE d.instance_id=$1 AND d.department_id>$2::uuid AND (r.q_first_date<$3::date OR (r.q_state='active' AND r.span && ANY($4::daterange[]))) ORDER BY d.department_id::text LIMIT 100`, u.instance, cursor, ent.ValidFrom, windows)
		if e != nil {
			return e
		}
		heads := map[string]string{}
		for rows.Next() {
			var id, source string
			if e = rows.Scan(&id, &source); e != nil {
				rows.Close()
				return e
			}
			heads[id] = source
			cursor = id
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		if len(heads) == 0 {
			break
		}
		facts, e := u.currentFacts(ctx, history.Department, heads)
		if e != nil {
			return e
		}
		for _, v := range facts.Departments {
			if e = enterprisecheck.Records(ent, v.Timeline.Records); e != nil {
				return &foundation.Fault{Status: 409, Code: "M02_REFERENCE_CONFLICT", Field: "effectiveDate", Message: "现有部门的有效期间超出企业候选范围，请先处理相关部门"}
			}
		}
	}
	return nil
}

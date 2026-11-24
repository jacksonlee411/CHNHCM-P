package application

import (
	"bytes"
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	sharedselection "chnhcm.local/chnhcm/internal/contexts/organization/application/selection"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"encoding/csv"
	"encoding/hex"
	"strings"
)

func selectionInput(q org.SelectionQuery, version, sort string) (org.ListInput, error) {
	in := org.ListInput{Q: q.Q, Status: q.Status, Sort: sort, Page: 1, PageSize: 20}
	if q.FilterField != "" || q.FilterOperator != "" || q.FilterValue != "" {
		in.Filter = &org.ColumnFilter{Field: q.FilterField, Operator: q.FilterOperator, Value: q.FilterValue}
	}
	if len(version) != 64 || version != strings.ToLower(version) {
		return in, invalidSelection()
	}
	if _, err := hex.DecodeString(version); err != nil {
		return in, invalidSelection()
	}
	err := normalizeList(&in)
	return in, err
}
func invalidSelection() error {
	return foundation.Fail(400, "M02_INVALID_SELECTION", "选择内容无效，请重新选择")
}
func selectRows(list org.List, choice org.Selection) ([]org.Summary, error) {
	return sharedselection.Rows(list.Items, choice, func(v org.Summary) string { return v.ID })
}
func (s Service) ResolveSelection(ctx context.Context, c access.Credentials, in org.SelectionRequest) (out org.SelectionResult, err error) {
	query, err := selectionInput(in.Query, in.ExpectedSelectionVersion, "code")
	if err != nil {
		return out, err
	}
	err = s.Store.Within(ctx, c, func(u Unit) error {
		ent, e := u.Enterprise(ctx, false)
		if e != nil {
			return e
		}
		now, e := u.Now(ctx)
		if e != nil {
			return e
		}
		query.BusinessAt = now.In(foundation.Shanghai).Format("2006-01-02")
		list, e := u.SelectionList(ctx, query, ent, query.BusinessAt)
		if e != nil {
			return e
		}
		if list.SelectionVersion != in.ExpectedSelectionVersion {
			return RejectRead(foundation.Fail(409, "M02_SELECTION_CHANGED", "列表已变化，请重新选择"))
		}
		rows, e := selectRows(list, in.Selection)
		if e != nil {
			return RejectRead(e)
		}
		can := len(rows) > 0
		for _, row := range rows {
			can = can && allowed(u.Access().Grants(), "bu.export", ent.ID, row.ID)
		}
		out = org.SelectionResult{QueryKey: list.QueryKey, SelectionVersion: list.SelectionVersion, TotalCount: list.Count, SelectedCount: len(rows), CanExport: can}
		return nil
	})
	if err != nil {
		return org.SelectionResult{}, err
	}
	return
}
func (s Service) ExportSelection(ctx context.Context, c access.Credentials, in org.ExportRequest) (out org.Export, err error) {
	query, err := selectionInput(in.Query, in.ExpectedSelectionVersion, in.Sort)
	if err != nil {
		return out, err
	}
	err = s.Store.Within(ctx, c, func(u Unit) error {
		ent, e := u.Enterprise(ctx, false)
		if e != nil {
			return e
		}
		now, e := u.Now(ctx)
		if e != nil {
			return e
		}
		query.BusinessAt = now.In(foundation.Shanghai).Format("2006-01-02")
		list, e := u.SelectionList(ctx, query, ent, query.BusinessAt)
		if e != nil {
			return e
		}
		if list.SelectionVersion != in.ExpectedSelectionVersion {
			return RejectRead(foundation.Fail(409, "M02_SELECTION_CHANGED", "列表已变化，请重新选择"))
		}
		rows, e := selectRows(list, in.Selection)
		if e != nil {
			return RejectRead(e)
		}
		if len(rows) == 0 {
			return RejectRead(foundation.Fail(400, "M02_EMPTY_SELECTION", "请先选择业务单元"))
		}
		for _, row := range rows {
			d, e := u.Access().Decide(ctx, access.Intent{Action: "bu.export", EnterpriseID: ent.ID, BUID: row.ID, Kind: "export", Fields: []string{}})
			if e != nil {
				return e
			}
			if !d.Allowed {
				return RejectRead(permission())
			}
		}
		data, e := selectionCSV(ctx, rows)
		if e != nil {
			return e
		}
		out = org.Export{Data: data, Count: len(rows), BusinessAt: query.BusinessAt}
		return nil
	})
	if err != nil {
		return org.Export{}, err
	}
	return
}
func csvText(value string) string { return sharedselection.CSVText(value) }
func selectionCSV(ctx context.Context, rows []org.Summary) ([]byte, error) {
	var buffer bytes.Buffer
	buffer.WriteString("\xef\xbb\xbf")
	writer := csv.NewWriter(&buffer)
	writer.UseCRLF = true
	if err := writer.Write([]string{"编码", "名称", "状态", "生效日期"}); err != nil {
		return nil, err
	}
	for _, row := range rows {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		name := ""
		if row.Name != nil {
			name = *row.Name
		}
		status := map[string]string{"active": "启用", "inactive": "停用", "not_effective": "未生效"}[row.Status]
		if err := writer.Write([]string{csvText(row.Code), csvText(name), status, row.EffectiveDate}); err != nil {
			return nil, err
		}
		if buffer.Len() > 32<<20 {
			return nil, foundation.Fail(413, "M02_EXPORT_SIZE_LIMIT", "导出内容过多，请减少所选项或分批导出")
		}
	}
	writer.Flush()
	if buffer.Len() > 32<<20 {
		return nil, foundation.Fail(413, "M02_EXPORT_SIZE_LIMIT", "导出内容过多，请减少所选项或分批导出")
	}
	if err := writer.Error(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

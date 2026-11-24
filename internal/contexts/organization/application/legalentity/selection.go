package legalentity

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	shared "chnhcm.local/chnhcm/internal/contexts/organization/application/selection"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/legalentity"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"io"
)

func selectionQuery(q org.LegalEntitySelectionQuery, sort string, page, size int) (org.LegalEntityQuery, error) {
	v := queryInput(q, sort)
	v.Page = page
	v.PageSize = size
	return v, normalizeList(&v)
}
func queryKey(u Unit, q org.LegalEntityQuery) string {
	p := u.Access().Principal()
	raw, _ := json.Marshal(struct {
		Instance, Enterprise, Subject, Q, Status, Country string
		Filter                                            *org.ColumnFilter
		Grants                                            []access.Grant
	}{p.InstanceID, u.EnterpriseScope().ID, p.ID, q.Q, q.Status, q.CountryRegionID, q.Filter, u.Access().Grants()})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
func validIDs(ids []string) error {
	if len(ids) > DraftEditLimit {
		return DraftBatchLimit()
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if !foundation.ValidID(id) || seen[id] {
			return shared.Invalid()
		}
		seen[id] = true
	}
	return nil
}
func selectionScope(ctx context.Context, u Unit, q org.LegalEntityQuery) ([]string, string, error) {
	fields, e := readable(ctx, u, false)
	if e != nil {
		return nil, "", e
	}
	if e = authorizeQuery(ctx, u, q, fields); e != nil {
		return nil, "", e
	}
	now, e := u.Now(ctx)
	if e != nil {
		return nil, "", e
	}
	return fields, now.In(foundation.Shanghai).Format("2006-01-02"), nil
}
func draftPage(ctx context.Context, u Unit, q org.LegalEntityQuery, fields []string, day string, draft Draft) (org.LegalEntitySelectionPage, error) {
	page, e := u.List(ctx, q, fields, day)
	page.QueryKey = queryKey(u, q)
	page.BusinessAt = day
	if e != nil {
		return org.LegalEntitySelectionPage{}, e
	}
	ids := []string{}
	for _, v := range page.Items {
		ids = append(ids, v.ID)
	}
	selected, e := u.DraftPage(ctx, draft, ids)
	if e != nil {
		return org.LegalEntitySelectionPage{}, e
	}
	export, e := u.Access().Decide(ctx, intent(u, "export", nil))
	if e != nil {
		return org.LegalEntitySelectionPage{}, e
	}
	return org.LegalEntitySelectionPage{PageState: "available", BusinessAt: day, Page: page, MemberIDs: selected.Members, SelectedIDs: selected.Selected, Summary: org.LegalEntitySelectionResult{QueryKey: draft.QueryKey, DraftID: draft.ID, Revision: draft.Revision, SelectedCount: draft.Selected, CanExport: export.Allowed}}, nil
}
func (s Service) StartSelection(ctx context.Context, c access.Credentials, in org.LegalEntitySelectionStartInput) (out org.LegalEntitySelectionPage, err error) {
	q, e := selectionQuery(in.Query, in.Sort, in.Page, in.PageSize)
	if e != nil {
		return out, e
	}
	var only []string
	switch in.Selection.Mode {
	case "explicit":
		if in.Selection.IDs == nil || in.Selection.ExcludedIDs != nil {
			return out, shared.Invalid()
		}
		only = *in.Selection.IDs
		if only == nil {
			only = []string{}
		}
		if e = validIDs(only); e != nil {
			return out, e
		}
	case "allMatching":
		if in.Selection.IDs != nil || in.Selection.ExcludedIDs == nil || len(*in.Selection.ExcludedIDs) != 0 {
			return out, shared.Invalid()
		}
	default:
		return out, shared.Invalid()
	}
	err = s.read(ctx, c, func(u Unit) error {
		fields, day, e := selectionScope(ctx, u, q)
		if e != nil {
			return e
		}
		ids, e := u.QueryIDs(ctx, q, fields, day, only)
		if e != nil {
			return e
		}
		if only != nil && len(ids) != len(only) {
			return shared.Invalid()
		}
		draft, e := u.CreateDraft(ctx, Draft{Enterprise: u.EnterpriseScope().ID, Day: day, QueryKey: queryKey(u, q), Mode: in.Selection.Mode}, ids)
		if e != nil {
			return e
		}
		out, e = draftPage(ctx, u, q, fields, day, draft)
		return e
	})
	return
}
func (s Service) ResolveSelection(ctx context.Context, c access.Credentials, in org.LegalEntitySelectionInput) (out org.LegalEntitySelectionPage, err error) {
	q, e := selectionQuery(in.Query, in.Sort, in.Page, in.PageSize)
	if e != nil {
		return out, e
	}
	ids := append(append(append([]string{}, in.AddIDs...), in.RemoveIDs...), in.RestoreIDs...)
	if e = validIDs(ids); e != nil {
		return out, e
	}
	err = s.read(ctx, c, func(u Unit) error {
		fields, day, e := selectionScope(ctx, u, q)
		if e != nil {
			return e
		}
		draft, e := u.ReadDraft(ctx, in.DraftID, in.ExpectedRevision, day, queryKey(u, q))
		if e != nil {
			return e
		}
		if draft.Enterprise != u.EnterpriseScope().ID {
			return denied()
		}
		if len(in.AddIDs) > 0 {
			captured, e := u.QueryIDs(ctx, q, fields, day, in.AddIDs)
			if e != nil {
				return e
			}
			if len(captured) != len(in.AddIDs) {
				return shared.Invalid()
			}
		}
		draft, e = u.EditDraft(ctx, draft, DraftEdit{Add: in.AddIDs, Remove: in.RemoveIDs, Restore: in.RestoreIDs})
		if e != nil {
			return e
		}
		out, e = draftPage(ctx, u, q, fields, day, draft)
		return e
	})
	return
}
func (s Service) CancelSelection(ctx context.Context, c access.Credentials, in org.LegalEntityDraftInput) error {
	return s.read(ctx, c, func(u Unit) error { return u.CancelDraft(ctx, in.DraftID, in.ExpectedRevision) })
}

var exportFields = []string{"code", "name", "nameEn", "countryRegionId", "subjectType", "registrationType", "registrationNumber", "establishingOrganization", "status", "effectiveDate"}

func (s Service) ExportSelection(ctx context.Context, c access.Credentials, in org.LegalEntityExportInput) (out org.LegalEntityExport, err error) {
	q, e := selectionQuery(in.Query, in.Sort, 1, 20)
	if e != nil {
		return out, e
	}
	var file ExportFile
	defer func() {
		if err != nil && file != nil {
			_ = file.Close()
			out = org.LegalEntityExport{}
		}
	}()
	err = s.read(ctx, c, func(u Unit) error {
		fields, day, e := selectionScope(ctx, u, q)
		if e != nil {
			return e
		}
		ex, e := require(ctx, u, "export", nil)
		if e != nil {
			return e
		}
		fields = intersection(fields, ex.ReadFields)
		if len(intersection(fields, exportFields)) != len(exportFields) {
			return denied()
		}
		draft, e := u.ReadDraft(ctx, in.DraftID, in.ExpectedRevision, day, queryKey(u, q))
		if e != nil {
			return e
		}
		if draft.Enterprise != u.EnterpriseScope().ID {
			return denied()
		}
		ids, e := u.DraftMembers(ctx, draft)
		if e != nil {
			return e
		}
		if len(ids) == 0 {
			return foundation.Fail(400, "M02_EMPTY_SELECTION", "请先选择法律实体")
		}
		// Captured membership is independent of subsequent live predicate changes.
		current := org.LegalEntityQuery{ListInput: org.ListInput{Sort: in.Sort}}
		ordered, e := u.CaptureExport(ctx, current, fields, day, ids)
		if e != nil {
			return e
		}
		if len(ordered) != len(ids) {
			return denied()
		}
		file, e = s.Store.CreateLegalEntityExport(ctx)
		if e != nil {
			return e
		}
		if _, e = io.WriteString(file, "\xef\xbb\xbf"); e != nil {
			return e
		}
		writer := csv.NewWriter(file)
		writer.UseCRLF = true
		if e = writer.Write([]string{"编码", "名称", "英文名", "国家/地区", "主体类别", "登记类型", "登记号码", "设立单位说明", "状态", "生效日期"}); e != nil {
			return e
		}
		text := func(p *string) string {
			if p == nil {
				return ""
			}
			return shared.CSVText(*p)
		}
		for start := 0; start < len(ordered); start += 100 {
			batch := ordered[start:min(start+100, len(ordered))]
			states, e := u.LoadMany(ctx, batch)
			if e != nil {
				return e
			}
			byID := map[string]d.Timeline{}
			for _, v := range states {
				byID[v.Timeline.ID] = v.Timeline
			}
			for _, id := range batch {
				if e = ctx.Err(); e != nil {
					return e
				}
				t, ok := byID[id]
				if !ok {
					return denied()
				}
				r := d.Kernel.FactAt(t.Records, day)
				state := "not_effective"
				if r == nil {
					r = &t.Records[0]
				} else {
					state = r.State
				}
				v := project(*r, fields, nil)
				if e = projectCountry(ctx, u, *r, fields, day, &v); e != nil {
					return e
				}
				if v.CountryRestricted || v.CountryName == nil {
					return denied()
				}
				if e = writer.Write([]string{shared.CSVText(t.Code), shared.CSVText(r.Data.Name), text(r.Data.NameEn), text(v.CountryName), d.SubjectTypeLabel(r.Data.SubjectType), text(r.Data.RegistrationType), text(r.Data.RegistrationNumber), text(r.Data.EstablishingOrganization), map[string]string{"active": "启用", "inactive": "停用", "not_effective": "未生效"}[state], r.Date}); e != nil {
					return e
				}
			}
		}
		writer.Flush()
		if e = writer.Error(); e != nil {
			return e
		}
		if e = file.Seal(); e != nil {
			return e
		}
		out = org.LegalEntityExport{File: file, Count: len(ordered), BusinessAt: day}
		return nil
	})
	return
}

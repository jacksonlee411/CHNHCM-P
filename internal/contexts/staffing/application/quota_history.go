package application

import (
	"context"
	"slices"
	"strconv"
	"time"

	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	s "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

func (svc Service) QuotaChanges(ctx context.Context, creds foundation.Credentials, id string, query s.LogQuery) (out s.ChangeLog, err error) {
	if !foundation.ValidID(id) {
		return out, quotaNotFound()
	}
	err = svc.QuotaStore.WithinQuota(ctx, creds, false, func(u QuotaTx) error {
		ent, e := u.Enterprise(ctx, false)
		if e != nil {
			return e
		}
		target, e := u.QuotaIdentity(ctx, id)
		if e != nil {
			return e
		}
		if target.ID == "" {
			return quotaNotFound()
		}
		rights, e := allocationRights(ctx, u, ent.ID, target, true)
		if e != nil {
			return e
		}
		if !rights.Allowed || !has(rights.ReadFields, "quota", "sourceVersion") {
			return quotaNotFound()
		}
		if target.Kind == "department" {
			if !has(rights.ReadFields, "departmentId", "status") {
				return quotaNotFound()
			}
			if _, e = allocationTimeline(ctx, u, ent.ID, target, true, false); e != nil {
				return e
			}
		}
		if e = checkLogQuery(query, rights.ReadFields, d.Kind(s.Quota)); e != nil {
			return e
		}
		entries, e := u.QuotaHistory(ctx, id)
		if e != nil {
			return e
		}
		out, e = changePage(query, rights.ReadFields, rights.ReadFields, len(entries), func(i int) string { return entries[i].Source }, func(i int) (s.ChangeGroup, error) {
			var before d.StaffingQuota
			if i+1 < len(entries) {
				before = entries[i+1].Snapshot
			}
			return quotaChangeGroup(entries[i], before, rights.ReadFields), nil
		})
		return e
	})
	return
}

func quotaChangeGroup(entry QuotaHistoryEntry, before d.StaffingQuota, fields []string) s.ChangeGroup {
	after := entry.Snapshot
	group := s.ChangeGroup{SourceVersion: entry.Source, Action: entry.Action, Items: []s.ChangeItem{}}
	if before.Source != "" {
		group.PreviousSourceVersion = &before.Source
	}
	if has(fields, "audit") {
		group.RecordedAt = entry.RecordedAt.UTC().Format(time.RFC3339Nano)
		group.Operator = s.ChangeOperator(entry.Operator)
	}
	ids := []string{entry.RecordID}
	// A boundary change can also change the preceding record's period.
	if has(fields, "effectiveDate") {
		for _, q := range []d.StaffingQuota{before, after} {
			i := slices.IndexFunc(q.Records, func(r d.QuotaRecord) bool { return r.ID == entry.RecordID })
			if i > 0 && !slices.Contains(ids, q.Records[i-1].ID) {
				ids = append(ids, q.Records[i-1].ID)
			}
		}
	}
	for _, id := range ids {
		item := s.ChangeItem{RecordID: id, ChangeKind: "direct", Diffs: []s.ChangeDiff{}}
		bv, br := quotaLogSide(before, id, fields)
		av, ar := quotaLogSide(after, id, fields)
		item.BeforeRef, item.AfterRef = br, ar
		item.BeforeContext, item.AfterContext = logContext(bv), logContext(av)
		for _, field := range []string{"effectiveDate", "quota", "reason", "period"} {
			if !logFieldAllowed(field, fields) {
				continue
			}
			b, a := logValue(bv, field), logValue(av, field)
			if b.Kind != a.Kind || b.Text != a.Text || b.Start != a.Start || b.EndInclusive != a.EndInclusive {
				item.Diffs = append(item.Diffs, s.ChangeDiff{Field: field, Before: b, After: a})
			}
		}
		if id != entry.RecordID {
			if len(item.Diffs) == 0 {
				continue
			}
			item.ChangeKind = "derived"
		}
		group.Items = append(group.Items, item)
	}
	return group
}
func quotaLogSide(q d.StaffingQuota, id string, fields []string) (s.View, *s.ChangeRef) {
	i := slices.IndexFunc(q.Records, func(r d.QuotaRecord) bool { return r.ID == id })
	if i < 0 {
		return nil, nil
	}
	r := q.Records[i]
	v := s.View{}
	if has(fields, "effectiveDate") {
		v["effectiveDate"] = r.Date
		if i+1 < len(q.Records) {
			v["endExclusive"] = q.Records[i+1].Date
		}
	}
	if has(fields, "quota") {
		v["quota"] = "未核编"
		if r.Amount != nil {
			v["quota"] = strconv.Itoa(*r.Amount)
		}
	}
	if has(fields, "reason") {
		v["reason"] = r.Reason
		v["reasonLabel"] = r.Reason
	}
	return v, &s.ChangeRef{SourceVersion: q.Source, RecordID: id, Revision: r.Revision}
}

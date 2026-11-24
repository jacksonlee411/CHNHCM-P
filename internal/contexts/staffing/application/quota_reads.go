package application

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	s "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"math/big"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

func quotaRecord(q d.StaffingQuota, i int, fields []string) s.View {
	r := q.Records[i]
	v := s.View{}
	if has(fields, "quota") {
		v["quota"] = r.Amount
		v["quotaState"] = "unset"
		if r.Amount != nil {
			v["quotaState"] = "set"
		}
	}
	if has(fields, "effectiveDate") {
		v["effectiveDate"] = r.Date
		if i+1 < len(q.Records) {
			v["endInclusive"] = dayBefore(q.Records[i+1].Date)
		}
	}
	if has(fields, "reason") {
		v["reason"] = r.Reason
	}
	if has(fields, "sourceVersion", "quota") {
		v["recordId"] = r.ID
		v["revision"] = r.Revision
	}
	if has(fields, "audit") {
		v["recordedAt"] = r.Audit.RecordedAt
		label := r.Audit.DisplayName
		if label != "" && r.Audit.Login != "" {
			label += "（" + r.Audit.Login + "）"
		} else if label == "" {
			label = r.Audit.Login
		}
		if label == "" {
			label = r.Audit.SubjectID
		}
		v["operator"] = label
	}
	return v
}
func quotaRights(ctx context.Context, u QuotaTx, ent, pos string, historical bool) (access.Decision, error) {
	dec, e := quotaDecision(ctx, u, ent, pos, "read", nil)
	if e != nil || !dec.Allowed || !historical {
		return dec, e
	}
	hist, e := quotaDecision(ctx, u, ent, pos, "history.read", nil)
	dec.Allowed = hist.Allowed
	dec.ReadFields = intersection(dec.ReadFields, hist.ReadFields)
	return dec, e
}

type departmentReader interface {
	PositionSources(context.Context, string, []string, bool) ([]PositionSource, error)
	Access() access.Guard
	Now() time.Time
}

func departmentQuotaSource(ctx context.Context, u departmentReader, ent, id string, historical, lock bool) (d.Timeline, []string, error) {
	sources, err := u.PositionSources(ctx, "departmentId", []string{id}, lock)
	if err != nil || len(sources) == 0 {
		return d.Timeline{}, nil, err
	}
	ref := sources[0]
	actions := []string{"department.read"}
	if historical {
		actions = append(actions, "department.history.read")
	}
	var fields []string
	for i, action := range actions {
		dec, err := u.Access().Decide(ctx, access.Intent{Kind: "object", EnterpriseID: ent, DepartmentTarget: ref.DepartmentTarget, Action: action})
		if err != nil || !dec.Allowed || !has(dec.ReadFields, "name") {
			return d.Timeline{}, nil, err
		}
		if i == 0 {
			fields = dec.ReadFields
		} else {
			fields = intersection(fields, dec.ReadFields)
		}
	}
	ref.Timeline.Kind = d.Kind("department")
	return ref.Timeline, fields, nil
}
func quotaDepartment(ctx context.Context, u departmentReader, ent, day, id string) (string, error) {
	def, _, err := departmentQuotaSource(ctx, u, ent, id, day != today(u), false)
	if err != nil {
		return "", err
	}
	if i := d.At(def, day); i >= 0 {
		return def.Records[i].Data["name"], nil
	}
	return "", nil
}
func quotaIdentity(pos, code string, r d.QuotaPeriod, fields []string) s.View {
	v := s.View{"positionId": pos, "readFields": fields}
	if has(fields, "code") {
		v["code"] = code
	}
	if has(fields, "name") {
		v["name"] = r.Name
	}
	if has(fields, "status") {
		v["active"] = r.Active
	}
	if has(fields, "quota") {
		v["quotaState"] = "unset"
		v["quota"] = nil
	}
	if has(fields, "usageState") {
		v["usageState"] = "unknown"
	}
	return v
}
func projectQuotaUsage(ctx context.Context, u QuotaTx, view s.View, position, day string) error {
	if _, visible := view["usageState"]; !visible {
		return nil
	}
	usage, err := u.PositionUsage(ctx, position)
	if err == nil && usage.At(day).Known {
		view["usageState"] = "known"
	}
	return err
}
func quotaDetail(ctx context.Context, u QuotaTx, ent string, q s.QuotaQuery) (s.View, error) {
	history := q.Day != today(u) || q.Source != ""
	dec, e := quotaRights(ctx, u, ent, q.PositionID, history)
	if e != nil {
		return nil, e
	}
	if !dec.Allowed || q.Source != "" && !has(dec.ReadFields, "sourceVersion", "quota") {
		return nil, quotaNotFound()
	}
	pos, e := u.PositionPeriods(ctx, q.PositionID)
	if e != nil {
		return nil, e
	}
	if len(pos.Records) == 0 {
		return nil, quotaNotFound()
	}
	quota, e := u.LoadPositionQuota(ctx, pos.ID, q.Source)
	if e != nil {
		return nil, e
	}
	if q.Source != "" && quota.ID == "" {
		return nil, quotaNotFound()
	}
	if q.Source != "" && quota.PositionSource != "" {
		original, err := u.Load(ctx, d.Position, pos.ID, quota.PositionSource)
		if err != nil {
			return nil, err
		}
		if len(original.Records) == 0 {
			return nil, quotaNotFound()
		}
		pos = d.QuotaEligibility(original)
	}
	hd, e := quotaRights(ctx, u, ent, pos.ID, true)
	if e != nil {
		return nil, e
	}
	timelineAllowed := hd.Allowed && has(hd.ReadFields, "sourceVersion", "quota")
	index := quota.At(q.Day)
	selectedFields := dec.ReadFields
	missing := false
	if q.RecordID != "" {
		if index < 0 || quota.Records[index].ID != q.RecordID {
			index = -1
			if timelineAllowed {
				index = slices.IndexFunc(quota.Records, func(r d.QuotaRecord) bool { return r.ID == q.RecordID })
				selectedFields = hd.ReadFields
			}
		}
		missing = index < 0 || !has(selectedFields, "sourceVersion", "quota")
	} else if !q.Statistics && index < 0 && timelineAllowed && len(quota.Records) > 0 {
		index = 0
		selectedFields = hd.ReadFields
	}
	day := q.Day
	if !q.Statistics && index >= 0 && (q.RecordID != "" || quota.Records[index].Date > day) {
		day = quota.Records[index].Date
	}
	i := pos.At(day)
	if i < 0 {
		if q.Statistics {
			return nil, quotaNotFound()
		}
		i = 0
		day = pos.Records[0].Date
	}
	r := pos.Records[i]
	v := quotaIdentity(pos.ID, pos.Code, r, selectedFields)
	v["businessAt"] = q.Day
	if q.Source == "" {
		if err := projectQuotaUsage(ctx, u, v, pos.ID, day); err != nil {
			return nil, err
		}
	}
	if has(selectedFields, "departmentId") {
		name, e := quotaDepartment(ctx, u, ent, day, r.DepartmentID)
		if e != nil {
			return nil, e
		}
		if name != "" {
			v["departmentId"] = r.DepartmentID
			v["departmentName"] = name
		}
	}
	if quota.ID != "" && has(selectedFields, "quota", "sourceVersion") {
		v["quotaId"] = quota.ID
		v["sourceVersion"] = quota.Source
	}
	if !missing && index >= 0 && has(selectedFields, "quota") {
		for k, val := range quotaRecord(quota, index, selectedFields) {
			v[k] = val
		}
		if !q.Statistics && has(selectedFields, "effectiveDate") {
			v["selectedRecordIsFuture"] = quota.Records[index].Date > q.Day
		}
	}
	if missing {
		v["selectionMissing"] = true
		delete(v, "quotaState")
		delete(v, "quota")
	}
	if !q.Statistics && q.Source == "" && !missing {
		if e := quotaActions(ctx, u, ent, d.QuotaTarget{Kind: "position", ID: pos.ID}, quota, access.Decision{ReadFields: selectedFields}, []string{"create", "adjust", "withdraw", "correct", "redate", "delete"}, v); e != nil {
			return nil, e
		}
		basic, e := u.Access().Decide(ctx, access.Intent{Kind: "position", EnterpriseID: ent, Action: "position.read"})
		if e != nil {
			return nil, e
		}
		v["canReadDefinition"] = basic.Allowed
	}
	if timelineAllowed && quota.ID != "" {
		records := []s.View{}
		for i := range quota.Records {
			records = append(records, quotaRecord(quota, i, hd.ReadFields))
		}
		v["records"] = records
		v["historyReadFields"] = hd.ReadFields
	}
	return v, nil
}
func quotaActions(ctx context.Context, u QuotaTx, ent string, target d.QuotaTarget, quota d.StaffingQuota, dec access.Decision, operations []string, v s.View) error {
	actions := []string{}
	if has(dec.ReadFields, "quota") {
		for _, op := range operations {
			if (quota.ID == "") != (op == "create") || quota.ID != "" && !has(dec.ReadFields, "sourceVersion") {
				continue
			}
			fields := []string{"quota", "reason", "effectiveDate"}
			if op == "redate" || op == "delete" {
				fields = []string{"effectiveDate"}
			}
			if op == "correct" {
				fields = nil
			}
			a, e := allocationDecision(ctx, u, ent, target, op, fields)
			if e != nil {
				return e
			}
			if a.Allowed && has(dec.ReadFields, fields...) {
				if op == "correct" {
					editable := intersection(a.WriteFields, dec.ReadFields)
					v["correctFields"] = editable
					if !slices.Contains(editable, "quota") && !slices.Contains(editable, "reason") {
						continue
					}
				}
				actions = append(actions, op)
			}
		}
	}
	v["actions"] = actions
	return nil
}
func completeQuotaScope(ctx context.Context, u QuotaTx, ent, day string) (bool, error) {
	broad := access.Intent{Kind: "quota-enterprise", Action: "staffingquota.read", EnterpriseID: ent}
	all, e := u.Access().Decide(ctx, broad)
	if e != nil {
		return false, e
	}
	complete := all.Allowed && has(all.ReadFields, "quota", "departmentId", "status")
	if day != today(u) {
		broad.Action = "staffingquota.history.read"
		h, e := u.Access().Decide(ctx, broad)
		if e != nil {
			return false, e
		}
		complete = complete && h.Allowed && has(h.ReadFields, "quota", "departmentId", "status")
	}
	return complete, nil
}
func quotaList(ctx context.Context, u QuotaTx, ent string, q s.QuotaQuery) (s.View, error) {
	rows, e := u.QuotaCandidates(ctx, q.Day)
	if e != nil {
		return nil, e
	}
	items := []s.View{}
	departments := []s.View{}
	groups := map[string]s.View{}
	sums := map[string]*big.Int{}
	filteredAmounts := map[string]bool{}
	names := map[string]string{}
	// A position-scoped grant always labels its subtotal as partial, independent
	// of how many other positions actually exist in that department.
	complete, e := completeQuotaScope(ctx, u, ent, q.Day)
	if e != nil {
		return nil, e
	}
	needle := strings.ToLower(strings.TrimSpace(q.Search))
	for _, row := range rows {
		if q.DepartmentDetail && row.Position.DepartmentID != q.DepartmentID {
			continue
		}
		dec, e := quotaRights(ctx, u, ent, row.PositionID, q.Day != today(u))
		if e != nil {
			return nil, e
		}
		if !dec.Allowed {
			continue
		}
		v := quotaIdentity(row.PositionID, row.Code, row.Position, dec.ReadFields)
		if err := projectQuotaUsage(ctx, u, v, row.PositionID, q.Day); err != nil {
			return nil, err
		}
		if has(dec.ReadFields, "quota") && row.Amount >= 0 {
			v["quotaState"] = "set"
			v["quota"] = row.Amount
		}
		if row.RecordID != "" && has(dec.ReadFields, "quota", "sourceVersion") {
			v["recordId"] = row.RecordID
		}

		dep := row.Position.DepartmentID
		if has(dec.ReadFields, "departmentId") {
			name, ok := names[dep]
			if !ok {
				name, e = quotaDepartment(ctx, u, ent, q.Day, dep)
				if e != nil {
					return nil, e
				}
				names[dep] = name
			}
			if name != "" {
				v["departmentId"] = dep
				v["departmentName"] = name
				group := groups[dep]
				if group == nil {
					group = s.View{"departmentId": dep, "name": name, "scope": "visible"}
					if complete {
						group["scope"] = "complete"
					}
					groups[dep] = group
					departments = append(departments, group)
				}
				if has(dec.ReadFields, "quota", "status") {
					n := sums[dep]
					if n == nil {
						n = new(big.Int)
						sums[dep] = n
					}
					if _, ok := group["unsetCount"]; !ok {
						group["unsetCount"] = 0
					}
					if row.Position.Active && row.Amount < 0 {
						group["unsetCount"] = group["unsetCount"].(int) + 1
					}
					if row.Amount >= 0 && row.Position.Active {
						n.Add(n, big.NewInt(row.Amount))
					}
					group["total"] = n
				} else {
					filteredAmounts[dep] = true
				}
			}
		}
		if q.DepartmentID != "" && v["departmentId"] != q.DepartmentID {
			continue
		}
		if needle != "" && !(has(dec.ReadFields, "code") && strings.Contains(strings.ToLower(row.Code), needle) || has(dec.ReadFields, "name") && strings.Contains(strings.ToLower(row.Position.Name), needle)) {
			continue
		}
		items = append(items, v)
	}
	if complete {
		ids := []string{q.DepartmentID}
		if !q.DepartmentDetail {
			var err error
			ids, err = u.DepartmentQuotaIDs(ctx)
			if err != nil {
				return nil, err
			}
			if q.DepartmentID != "" {
				ids = append(ids, q.DepartmentID)
			}
		}
		for _, id := range ids {
			if groups[id] != nil {
				continue
			}
			if q.DepartmentDetail && id != q.DepartmentID {
				continue
			}
			name, e := quotaDepartment(ctx, u, ent, q.Day, id)
			if e != nil {
				return nil, e
			}
			if name == "" {
				continue
			}
			group := s.View{"departmentId": id, "name": name, "scope": "complete", "total": int64(0), "unsetCount": 0}
			groups[id] = group
			departments = append(departments, group)
		}
		for id, group := range groups {
			if q.DepartmentDetail && id != q.DepartmentID {
				continue
			}
			cap, err := u.LoadTargetQuota(ctx, d.QuotaTarget{Kind: "department", ID: id}, "")
			if err != nil {
				return nil, err
			}
			group["quotaState"] = "unset"
			group["quota"] = nil
			if i := cap.At(q.Day); i >= 0 && cap.Records[i].Amount != nil {
				group["quotaState"] = "set"
				group["quota"] = *cap.Records[i].Amount
				remaining := big.NewInt(int64(*cap.Records[i].Amount))
				if n := sums[id]; n != nil {
					remaining.Sub(remaining, n)
				}
				group["remaining"] = remaining
			}
		}
	}
	if q.DepartmentDetail {
		return allocationDetail(ctx, u, ent, q, groups[q.DepartmentID], items)
	}
	for id := range filteredAmounts {
		delete(groups[id], "total")
		delete(groups[id], "unsetCount")
		delete(groups[id], "remaining")
	}

	count := len(items)
	start := min((q.Page-1)*q.PageSize, count)
	out := s.View{"businessAt": q.Day, "items": items[start:min(start+q.PageSize, count)], "canBrowseDepartments": complete, "totalCount": count, "departments": departments}
	if q.Anchor != "" {
		i := slices.IndexFunc(items, func(v s.View) bool { return v["positionId"] == q.Anchor })
		nav := s.View{"position": i + 1, "totalCount": count, "businessAt": q.Day, "previousId": nil, "nextId": nil}
		if i > 0 {
			nav["previousId"] = items[i-1]["positionId"]
			if record, ok := items[i-1]["recordId"]; ok {
				nav["previousRecordId"] = record
			}
		}
		if i >= 0 && i+1 < count {
			nav["nextId"] = items[i+1]["positionId"]
			if record, ok := items[i+1]["recordId"]; ok {
				nav["nextRecordId"] = record
			}
		}
		out["navigation"] = nav
	}
	return out, nil
}
func (svc Service) Quotas(ctx context.Context, c foundation.Credentials, q s.QuotaQuery) (out s.View, err error) {
	if q.RecordID != "" && !foundation.ValidID(q.RecordID) || q.Anchor != "" && !foundation.ValidID(q.Anchor) || q.DepartmentID != "" && !foundation.ValidID(q.DepartmentID) || q.DepartmentOptions && utf8.RuneCountInString(q.Search) > 400 || q.Day != "" && !foundation.ValidDate(q.Day) || q.Source != "" && !foundation.ValidID(q.Source) || q.PositionID != "" && !foundation.ValidID(q.PositionID) {
		return nil, foundation.Fail(400, "M03_INVALID_INPUT", "查询日期或标识无效")
	}
	if q.Page == 0 {
		q.Page = 1
	}
	if q.PageSize == 0 {
		q.PageSize = 20
	}
	if q.Page < 1 || q.Page > 100000000 || q.PageSize < 1 || q.PageSize > 100 {
		return nil, foundation.Fail(400, "M03_INVALID_INPUT", "分页无效")
	}
	err = svc.QuotaStore.WithinQuota(ctx, c, false, func(u QuotaTx) error {
		if q.Day == "" {
			q.Day = today(u)
		}
		ent, e := u.Enterprise(ctx, false)
		if e != nil {
			return e
		}
		dec, e := quotaRights(ctx, u, ent.ID, "", q.Day != today(u) && q.Source == "")
		if e != nil {
			return e
		}
		if !dec.Allowed {
			return quotaDenied()
		}
		if q.DepartmentOptions {
			complete, err := completeQuotaScope(ctx, u, ent.ID, q.Day)
			if err != nil {
				return err
			}
			if !complete {
				return quotaDenied()
			}
			options, err := u.SearchQuotaDepartments(ctx, c, ent.ID, q.Day, strings.ToLower(strings.TrimSpace(q.Search)), (q.Page-1)*q.PageSize, q.PageSize+1)
			if err != nil {
				return err
			}
			out = s.View{"items": options[:min(len(options), q.PageSize)], "hasMore": len(options) > q.PageSize}
		} else if q.PositionID != "" {
			out, e = quotaDetail(ctx, u, ent.ID, q)
		} else {
			if q.DepartmentDetail && q.RecordID != "" {
				q.Day, e = allocationRecordDay(ctx, u, ent.ID, q)
				if e != nil {
					return e
				}
			}
			out, e = quotaList(ctx, u, ent.ID, q)
		}
		return e
	})
	return
}

package application

import (
	"context"
	"slices"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	s "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

func allocationDecision(ctx context.Context, u QuotaTx, enterprise string, t d.QuotaTarget, action string, fields []string) (access.Decision, error) {
	if t.Kind == "position" {
		return quotaDecision(ctx, u, enterprise, t.ID, action, fields)
	}
	return u.Access().Decide(ctx, access.Intent{Kind: "quota-enterprise", EnterpriseID: enterprise, Action: "staffingquota." + action, Fields: fields})
}
func allocationRights(ctx context.Context, u QuotaTx, enterprise string, t d.QuotaTarget, historical bool) (access.Decision, error) {
	read, err := allocationDecision(ctx, u, enterprise, t, "read", nil)
	if err != nil || !read.Allowed || !historical {
		return read, err
	}
	h, err := allocationDecision(ctx, u, enterprise, t, "history.read", nil)
	read.Allowed = h.Allowed
	read.ReadFields = intersection(read.ReadFields, h.ReadFields)
	return read, err
}
func allocationTimeline(ctx context.Context, u QuotaTx, enterprise string, t d.QuotaTarget, historical, lock bool) (d.QuotaPeriods, error) {
	if t.Kind == "position" {
		return u.PositionPeriods(ctx, t.ID)
	}
	def, _, err := departmentQuotaSource(ctx, u, enterprise, t.ID, historical, lock)
	if err == nil && def.ID == "" {
		return d.QuotaPeriods{}, quotaNotFound()
	}
	return d.QuotaEligibility(def), err
}

// Discover the full department envelope before acquiring any position guard.
// An intervening move into an unlocked department invalidates this attempt.
func lockAllocationTargets(ctx context.Context, u QuotaTx, targets []d.QuotaTarget) error {
	var deps, positions []string
	for _, t := range targets {
		if t.Kind == "department" {
			deps = append(deps, t.ID)
			continue
		}
		def, err := u.PositionPeriods(ctx, t.ID)
		if err != nil {
			return err
		}
		deps = append(deps, def.Departments()...)
		positions = append(positions, t.ID)
	}
	slices.Sort(deps)
	deps = slices.Compact(deps)
	if err := u.LockQuotaDepartments(ctx, deps); err != nil {
		return err
	}
	slices.Sort(positions)
	for _, id := range slices.Compact(positions) {
		if err := u.LockQuotaTarget(ctx, id); err != nil {
			return err
		}
		def, err := u.PositionPeriods(ctx, id)
		if err != nil {
			return err
		}
		if !departmentsLocked(def.Departments(), deps) {
			return foundation.Fail(409, "M03_QUOTA_CONFLICT", "岗位归属已变化，请刷新后重试")
		}
	}
	return nil
}
func departmentsLocked(departments, ids []string) bool {
	for _, id := range departments {
		if !slices.Contains(ids, id) {
			return false
		}
	}
	return true
}

type allocationValidation interface {
	departmentReader
	AllocationTx
	Access() access.Guard
}

func validateAllocation(ctx context.Context, u allocationValidation, enterprise string, ids []string, plans []d.QuotaPlan, definition *d.QuotaPeriods) error {
	quota := map[d.QuotaTarget]d.StaffingQuota{}
	for _, p := range plans {
		quota[p.After.Target()] = p.After
	}
	var first *d.AllocationConflict
	slices.Sort(ids)
	for _, id := range slices.Compact(ids) {
		target := d.QuotaTarget{Kind: "department", ID: id}
		cap, ok := quota[target]
		if !ok {
			var err error
			cap, err = u.LoadTargetQuota(ctx, target, "")
			if err != nil {
				return err
			}
		}
		if len(cap.Records) == 0 {
			continue
		}
		members, err := u.DepartmentMembers(ctx, id)
		if err != nil {
			return err
		}
		found := false
		for i := range members {
			pos := &members[i]
			if definition != nil && pos.Position.ID == definition.ID {
				pos.Position = *definition
				found = true
			}
			if q, ok := quota[d.QuotaTarget{Kind: "position", ID: pos.Position.ID}]; ok {
				pos.Quota = q
			}
		}
		if definition != nil && !found {
			q, err := u.LoadTargetQuota(ctx, d.QuotaTarget{Kind: "position", ID: definition.ID}, "")
			if err != nil {
				return err
			}
			members = append(members, d.AllocatedPosition{Position: *definition, Quota: q})
		}
		conflict := d.DepartmentCapacity(id, cap, members)
		if conflict != nil && (first == nil || conflict.Day < first.Day) {
			first = conflict
		}
	}
	if first == nil {
		return nil
	}
	// The existence of a hidden position or its contribution must stay private.
	for _, action := range []string{"staffingquota.read", "staffingquota.history.read"} {
		dec, err := u.Access().Decide(ctx, access.Intent{Kind: "quota-enterprise", EnterpriseID: enterprise, Action: action})
		if err != nil {
			return err
		}
		if !dec.Allowed || !has(dec.ReadFields, "departmentId", "quota", "effectiveDate", "status") {
			return foundation.Fail(409, "M03_DEPARTMENT_QUOTA", "本次调整超过部门编制上限")
		}
	}
	name, err := quotaDepartment(ctx, u, enterprise, first.Day, first.DepartmentID)
	if err != nil {
		return err
	}
	if name == "" {
		return foundation.Fail(409, "M03_DEPARTMENT_QUOTA", "本次调整超过部门编制上限")
	}
	return foundation.Fail(409, "M03_DEPARTMENT_QUOTA", "自 "+first.Day+" 起，"+name+"的岗位核定合计超出部门总额 "+first.Shortfall+" 编")
}

func checkAllocationDate(ctx context.Context, u QuotaTx, enterprise, id, day string, locked bool) error {
	def, fields, err := departmentQuotaSource(ctx, u, enterprise, id, day != today(u), locked)
	if err != nil {
		return err
	}
	if def.ID == "" {
		return quotaNotFound()
	}
	index := d.At(def, day)
	if index < 0 && len(def.Records) > 0 && has(fields, "effectiveDate", "status") {
		return foundation.Fail(409, "M03_QUOTA_CONFLICT", "生效日不能早于部门首日 "+def.Records[0].Date)
	}
	if index < 0 || !def.Records[index].Active() {
		return foundation.Fail(409, "M03_QUOTA_CONFLICT", "部门在生效日未启用")
	}
	return nil
}

func allocationPlans(ctx context.Context, u QuotaTx, cmd s.QuotaCommand, lock bool) ([]d.QuotaPlan, []string, error) {
	ent, err := u.Enterprise(ctx, lock)
	if err != nil {
		return nil, nil, err
	}
	if cmd.Allocation == nil {
		p, err := quotaPrepare(ctx, u, cmd, lock)
		if err != nil {
			return nil, nil, err
		}
		t := p.After.Target()
		ids := []string{t.ID}
		if t.Kind == "position" {
			pos, err := u.PositionPeriods(ctx, t.ID)
			if err != nil {
				return nil, nil, err
			}
			ids = pos.Departments()
		}
		if p.Changed {
			if err := validateAllocation(ctx, u, ent.ID, ids, []d.QuotaPlan{p}, nil); err != nil {
				return nil, nil, err
			}
		}
		return []d.QuotaPlan{p}, []string{cmd.Action}, nil
	}
	allowed := false
	for _, action := range []string{"create", "adjust", "correct", "withdraw"} {
		gate, err := quotaDecision(ctx, u, ent.ID, "", action, nil)
		if err != nil {
			return nil, nil, err
		}
		if gate.Allowed {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, nil, quotaDenied()
	}
	in := cmd.Allocation
	dep := d.QuotaTarget{Kind: "department", ID: cmd.DepartmentID}
	invalid := func() error {
		return foundation.Fail(400, "M03_INVALID_INPUT", "请输入生效日、调整原因和各目标的非负整数或撤去意图")
	}
	if !foundation.ValidID(dep.ID) || !foundation.ValidDate(in.EffectiveDate) || in.Reason == "" || len(in.Positions) == 0 && in.Total == nil {
		return nil, nil, invalid()
	}
	if err := checkAllocationDate(ctx, u, ent.ID, dep.ID, in.EffectiveDate, false); err != nil {
		return nil, nil, err
	}
	targets := []d.QuotaTarget{dep}
	seen := map[string]bool{}
	rows := in.Positions
	for _, row := range rows {
		if !foundation.ValidID(row.PositionID) || seen[row.PositionID] {
			return nil, nil, invalid()
		}
		seen[row.PositionID] = true
		targets = append(targets, d.QuotaTarget{Kind: "position", ID: row.PositionID})
	}
	if in.Total != nil {
		if in.Total.PositionID != "" {
			return nil, nil, invalid()
		}
		rows = append([]s.AllocationRow{*in.Total}, rows...)
	}
	if lock {
		if err = lockAllocationTargets(ctx, u, targets); err != nil {
			return nil, nil, err
		}
	}
	if lock {
		if err := checkAllocationDate(ctx, u, ent.ID, dep.ID, in.EffectiveDate, true); err != nil {
			return nil, nil, err
		}
	}
	ids := []string{dep.ID}
	var plans []d.QuotaPlan
	var actions []string
	for _, row := range rows {
		if (row.Amount == nil) != row.Withdraw || row.Amount != nil && *row.Amount < 0 {
			return nil, nil, invalid()
		}
		t := dep
		if row.PositionID != "" {
			t = d.QuotaTarget{Kind: "position", ID: row.PositionID}
		}
		read, err := allocationRights(ctx, u, ent.ID, t, in.EffectiveDate != today(u))
		if err != nil {
			return nil, nil, err
		}
		if !read.Allowed || !has(read.ReadFields, "quota", "sourceVersion") {
			return nil, nil, quotaNotFound()
		}
		if row.Withdraw {
			permit, err := allocationDecision(ctx, u, ent.ID, t, "withdraw", []string{"quota", "reason", "effectiveDate"})
			if err != nil {
				return nil, nil, err
			}
			if !permit.Allowed {
				return nil, nil, quotaDenied()
			}
		}
		current, err := u.LoadTargetQuota(ctx, t, "")
		if err != nil {
			return nil, nil, err
		}
		change := s.QuotaCommand{Action: "adjust", QuotaID: current.ID, Input: s.QuotaInput{EffectiveDate: in.EffectiveDate, Amount: row.Amount, Reason: in.Reason, ExpectedSource: row.ExpectedSource}}
		if t.Kind == "position" {
			position, err := u.PositionPeriods(ctx, t.ID)
			if err != nil {
				return nil, nil, err
			}
			i := position.At(in.EffectiveDate)
			if i < 0 || !position.Records[i].Active || position.Records[i].DepartmentID != dep.ID {
				return nil, nil, foundation.Fail(409, "M03_QUOTA_CONFLICT", "岗位在生效日不属于该部门的启用岗位")
			}
			change.Input.PositionID = t.ID
			ids = append(ids, position.Departments()...)
		} else {
			change.DepartmentID = t.ID
		}
		if current.ID == "" {
			if row.Withdraw {
				return nil, nil, foundation.Fail(409, "M03_QUOTA_CONFLICT", "此目标尚未核定")
			}
			change.Action = "create"
		} else if j := current.At(in.EffectiveDate); j >= 0 && current.Records[j].Date == in.EffectiveDate {
			change.Action = "allocate"
			change.RecordID = current.Records[j].ID
			change.Input.ExpectedRevision = current.Records[j].Revision
		} else if row.Withdraw {
			change.Action = "withdraw"
		}
		p, err := quotaPrepare(ctx, u, change, false)
		if err != nil {
			return nil, nil, err
		}
		if current.Source != row.ExpectedSource {
			return nil, nil, foundation.Fail(409, "M03_QUOTA_CONFLICT", "额度已变化，请刷新后重新办理")
		}
		// A common reason belongs only to the targets whose allocation changes.
		if index := current.At(in.EffectiveDate); index >= 0 {
			old := current.Records[index].Amount
			if old == nil && row.Amount == nil || old != nil && row.Amount != nil && *old == *row.Amount {
				p = d.QuotaPlan{After: current, Selected: current.Records[index].ID}
			}
		}
		plans = append(plans, p)
		actions = append(actions, change.Action)
	}
	if err := validateAllocation(ctx, u, ent.ID, ids, plans, nil); err != nil {
		return nil, nil, err
	}
	return plans, actions, nil
}

func allocationEditorAction(q d.StaffingQuota, day string) string {
	if q.ID == "" {
		return "create"
	}
	if i := q.At(day); i >= 0 && q.Records[i].Date == day {
		return "correct"
	}
	return "adjust"
}

// An explicit total record from a receipt or log can predate, or postdate,
// the originating statistics query. Resolve it inside the same read snapshot.
func allocationRecordDay(ctx context.Context, u QuotaTx, enterprise string, query s.QuotaQuery) (string, error) {
	target := d.QuotaTarget{Kind: "department", ID: query.DepartmentID}
	history, err := allocationRights(ctx, u, enterprise, target, true)
	if err != nil || !history.Allowed || !has(history.ReadFields, "quota", "sourceVersion", "effectiveDate", "departmentId", "status") {
		return query.Day, err
	}
	_, fields, err := departmentQuotaSource(ctx, u, enterprise, target.ID, true, false)
	if err != nil || !has(fields, "name") {
		return query.Day, err
	}
	q, err := u.LoadTargetQuota(ctx, target, query.Source)
	if err != nil {
		return "", err
	}
	for _, r := range q.Records {
		if r.ID == query.RecordID {
			return r.Date, nil
		}
	}
	return query.Day, nil
}

// Date disclosure follows the same stable-position grants as the daily rows.
// Membership can be visible even when the quota amount is masked.
func visibleMemberDates(ctx context.Context, u QuotaTx, enterprise, department, day string) (dates []string, canSubtotal bool, err error) {
	positions, err := u.DepartmentMembers(ctx, department)
	if err != nil {
		return nil, false, err
	}
	for _, p := range positions {
		rights, err := quotaRights(ctx, u, enterprise, p.Position.ID, true)
		if err != nil {
			return nil, false, err
		}
		if !rights.Allowed || !has(rights.ReadFields, "effectiveDate", "departmentId", "status") {
			continue
		}
		amount := has(rights.ReadFields, "quota")
		boundaries := p.AllocationDates(department, amount)
		if amount && len(boundaries) > 0 && boundaries[0] <= day {
			canSubtotal = true
		}
		dates = append(dates, boundaries...)
	}
	return dates, canSubtotal, nil
}

func allocationDetail(ctx context.Context, u QuotaTx, enterprise string, query s.QuotaQuery, group s.View, items []s.View) (s.View, error) {
	target := d.QuotaTarget{Kind: "department", ID: query.DepartmentID}
	def, orgFields, err := departmentQuotaSource(ctx, u, enterprise, target.ID, query.Source != "" || query.Day != today(u), false)
	if err == nil && def.ID == "" {
		return nil, quotaNotFound()
	}
	if err != nil {
		return nil, err
	}
	index := d.At(def, query.Day)
	if index < 0 {
		return nil, quotaNotFound()
	}
	exact := query.Source != ""
	historyFields := orgFields
	if !exact && query.Day == today(u) {
		_, historyFields, err = departmentQuotaSource(ctx, u, enterprise, target.ID, true, false)
		if err != nil {
			return nil, err
		}
	}
	var dates []string
	var canSubtotal bool
	if !exact && has(historyFields, "name") {
		dates, canSubtotal, err = visibleMemberDates(ctx, u, enterprise, target.ID, query.Day)
		if err != nil {
			return nil, err
		}
	}
	if group == nil {
		// A scoped reader can follow their last visible position transferring
		// out. An unrelated department still has no disclosable allocation.
		if !slices.ContainsFunc(dates, func(date string) bool { return date <= query.Day }) {
			return nil, quotaNotFound()
		}
		group = s.View{"name": def.Records[index].Data["name"], "scope": "visible"}
		if canSubtotal {
			group["total"], group["unsetCount"] = 0, 0
		}
	}
	detail := s.View{"departmentId": target.ID, "name": group["name"], "businessAt": query.Day, "scope": group["scope"]}
	for _, k := range []string{"total", "unsetCount", "quota", "quotaState", "remaining"} {
		if v, ok := group[k]; ok {
			detail[k] = v
		}
	}
	active := def.Records[index].Active()
	if has(orgFields, "status") {
		detail["active"] = active
	}
	if has(historyFields, "effectiveDate", "status") && len(def.Records) > 0 {
		detail["firstEffectiveDate"] = def.Records[0].Date
	}
	if exact && group["scope"] != "complete" {
		return nil, quotaNotFound()
	}
	read, err := allocationRights(ctx, u, enterprise, target, query.Day != today(u) || exact)
	if err != nil {
		return nil, err
	}
	if exact && (!read.Allowed || !has(read.ReadFields, "sourceVersion", "quota")) {
		return nil, quotaNotFound()
	}
	if group["scope"] == "complete" && read.Allowed {
		q, err := u.LoadTargetQuota(ctx, target, query.Source)
		if err != nil {
			return nil, err
		}
		if exact && q.ID == "" {
			return nil, quotaNotFound()
		}
		if has(read.ReadFields, "sourceVersion", "quota") && q.ID != "" {
			detail["quotaId"] = q.ID
			detail["sourceVersion"] = q.Source
		}
		if exact {
			detail["quotaState"] = "unset"
			detail["quota"] = nil
		}
		i := q.At(query.Day)
		if exact && query.RecordID != "" {
			i = slices.IndexFunc(q.Records, func(r d.QuotaRecord) bool { return r.ID == query.RecordID })
			if i < 0 {
				return nil, quotaNotFound()
			}
		}
		if i >= 0 {
			for k, v := range quotaRecord(q, i, read.ReadFields) {
				detail[k] = v
			}
			if active && !exact && q.Records[i].Date == query.Day && has(read.ReadFields, "effectiveDate") {
				ops := []string{"correct", "redate"}
				if i > 0 {
					ops = append(ops, "delete")
				}
				if err := quotaActions(ctx, u, enterprise, target, q, read, ops, detail); err != nil {
					return nil, err
				}
			}
		}
		op := allocationEditorAction(q, query.Day)
		auth, err := allocationDecision(ctx, u, enterprise, target, op, []string{"quota", "effectiveDate", "reason"})
		if err != nil {
			return nil, err
		}
		detail["canSetTotal"] = active && !exact && auth.Allowed && has(read.ReadFields, "quota", "reason", "effectiveDate", "sourceVersion")
		withdraw, err := allocationDecision(ctx, u, enterprise, target, "withdraw", []string{"quota", "reason", "effectiveDate"})
		if err != nil {
			return nil, err
		}
		detail["canWithdraw"] = detail["canSetTotal"] == true && withdraw.Allowed
		hist, err := allocationRights(ctx, u, enterprise, target, true)
		if err != nil {
			return nil, err
		}
		if has(historyFields, "name") && hist.Allowed && has(hist.ReadFields, "sourceVersion", "quota") && q.ID != "" {
			detail["historyReadFields"] = hist.ReadFields
		}
		if !exact && has(historyFields, "name") && hist.Allowed && has(hist.ReadFields, "effectiveDate", "departmentId", "status", "quota") {
			for _, r := range q.Records {
				dates = append(dates, r.Date)
			}
		}
	}
	if query.RecordID != "" && detail["recordId"] != query.RecordID {
		return s.View{"departmentId": target.ID, "name": group["name"], "businessAt": query.Day, "scope": group["scope"],
			"selectionMissing": true, "members": []s.View{}}, nil
	}
	members := []s.View{}
	if !exact {
		for _, row := range items {
			if row["active"] != true {
				continue
			}
			id := row["positionId"].(string)
			member := d.QuotaTarget{Kind: "position", ID: id}
			dec, err := allocationRights(ctx, u, enterprise, member, query.Day != today(u))
			if err != nil {
				return nil, err
			}
			if dec.Allowed && has(dec.ReadFields, "quota", "sourceVersion", "effectiveDate", "reason") {
				q, err := u.LoadTargetQuota(ctx, member, "")
				if err != nil {
					return nil, err
				}
				auth, err := allocationDecision(ctx, u, enterprise, member, allocationEditorAction(q, query.Day), []string{"quota", "reason", "effectiveDate"})
				if err != nil {
					return nil, err
				}
				row["canAllocate"] = active && auth.Allowed
				withdraw, err := allocationDecision(ctx, u, enterprise, member, "withdraw", []string{"quota", "reason", "effectiveDate"})
				if err != nil {
					return nil, err
				}
				row["canWithdraw"] = active && auth.Allowed && withdraw.Allowed
				if q.ID != "" {
					row["sourceVersion"] = q.Source
				}
			}
			members = append(members, row)
		}
	}
	if exact {
		delete(detail, "total")
		delete(detail, "remaining")
		delete(detail, "unsetCount")
	} else {
		slices.Sort(dates)
		if len(dates) > 0 {
			detail["allocationDates"] = slices.Compact(dates)
		}
		existing := detail["quotaId"] != nil || slices.ContainsFunc(members, func(row s.View) bool {
			return row["sourceVersion"] != nil || row["recordId"] != nil
		})
		detail["defaultEffectiveDate"] = query.Day
		if !existing {
			detail["defaultEffectiveDate"] = foundation.DefaultEffectiveDate
		}
	}
	detail["members"] = members
	return detail, nil
}

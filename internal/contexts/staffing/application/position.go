package application

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strings"
	"unicode/utf8"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	s "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

type PositionSource struct {
	d.Timeline
	DepartmentTarget *access.DepartmentTarget
}
type positionCatalog map[string]map[string]PositionSource

func positionRead(ctx context.Context, u Unit, field string, source PositionSource, date string, exact, requireMembership bool) (bool, error) {
	visible, _, err := positionDisplay(ctx, u, field, source, date, exact, requireMembership)
	return visible, err
}

func positionDisplay(ctx context.Context, u Unit, field string, source PositionSource, date string, exact, requireMembership bool) (visible, showCode bool, err error) {
	if source.ID == "" {
		return false, false, nil
	}
	kind := strings.TrimSuffix(field, "Id")
	required := []string{"name", "effectiveDate"}
	// BU lifecycle is carried by its effective records; status is not a BU grant field.
	if field != "buId" {
		required = append(required, "status")
	}
	if requireMembership && field == "jobId" {
		required = append(required, "gradeIds")
	}
	purposes := []string{"read"}
	if exact || date != today(u) {
		purposes = append(purposes, "history.read")
	}
	showCode = true
	for _, purpose := range purposes {
		intent := access.Intent{EnterpriseID: source.Enterprise, DepartmentTarget: source.DepartmentTarget, Action: kind + "." + purpose, Kind: "object"}
		if field == "buId" {
			intent.BUID = source.ID
		}
		dec, err := u.Access().Decide(ctx, intent)
		if err != nil {
			return false, false, err
		}
		if !dec.Allowed || !has(dec.ReadFields, required...) {
			return false, false, nil
		}
		showCode = showCode && has(dec.ReadFields, "code")
	}
	return true, showCode, nil
}

func positionCatalogFor(ctx context.Context, u Unit, locked bool, timelines ...d.Timeline) (positionCatalog, error) {
	result := positionCatalog{}
	for _, field := range []string{"gradeId", "jobId", "departmentId", "locationId"} {
		ids := []string{}
		for _, timeline := range timelines {
			for _, record := range timeline.Records {
				if id := record.Data[field]; id != "" {
					ids = append(ids, id)
				}
			}
		}
		slices.Sort(ids)
		sources, err := u.PositionSources(ctx, field, slices.Compact(ids), locked)
		if err != nil {
			return nil, err
		}
		result[field] = map[string]PositionSource{}
		for _, source := range sources {
			result[field][source.ID] = source
		}
	}
	return result, nil
}

func positionFault(field, date string) error {
	return &foundation.Fault{Status: 409, Code: "M03_REFERENCE_CONFLICT", Field: field, Message: d.PositionLabel(field) + "在 " + date + " 起不满足拟定期间，请调整资料或生效日期"}
}

func positionConflict(catalog positionCatalog, field string, record d.Record) string {
	id := record.Data[field]
	if field == "gradeId" && id == "" {
		return ""
	}
	first := d.PositionConflict(catalog[field][id].Timeline, record.Date, record.End, "")
	if field == "gradeId" {
		other := d.PositionConflict(catalog["jobId"][record.Data["jobId"]].Timeline, record.Date, record.End, id)
		if other != "" && (first == "" || other < first) {
			first = other
		}
	}
	return first
}

func losesPositionEligibility(before, after d.Timeline) bool {
	if d.CoverageChanged(before, after) {
		return true
	}
	if after.Kind != d.Job {
		return false
	}
	for _, old := range before.Records {
		if !old.Active() {
			continue
		}
		index := d.At(after, old.Date)
		if index >= 0 {
			next := after.Records[index]
			if next.End == old.End && next.Grades.EqualData(old.Grades) {
				continue
			}
		}
		for _, member := range old.Grades {
			if d.PositionConflict(after, old.Date, old.End, member.GradeID) != "" {
				return true
			}
		}
	}
	return false
}

func preparePositions(ctx context.Context, u Unit, plan *d.Plan, locked bool) error {
	before, after := plan.Before, plan.After
	if after.Kind == d.Position && len(before.Records) > 0 && after.Records[0].Date > before.Records[0].Date {
		quota, err := u.LoadPositionQuota(ctx, after.ID, "")
		if err != nil {
			return err
		}
		if len(quota.Records) > 0 && quota.Records[0].Date < after.Records[0].Date {
			return foundation.Fail(409, "M03_QUOTA_CONFLICT", "岗位首日不能晚于已有核编，请先维护额度")
		}
	}
	if after.Kind == d.Position && len(before.Records) > 0 {
		proposed := d.QuotaEligibility(after)
		previous := d.QuotaEligibility(before)
		usage, err := u.PositionUsage(ctx, after.ID)
		if err != nil {
			return err
		}
		for _, use := range usage.Uses {
			if use.Owner != "M04" || use.Department == "" {
				continue
			}
			span := foundation.Period{From: use.From, Until: use.Until}
			var boundaries []string
			for _, r := range after.Records {
				boundaries = append(boundaries, r.Date, r.End)
			}
			for _, p := range span.Split(boundaries) {
				i := d.At(after, p.From)
				if i < 0 || !after.Records[i].Active() || after.Records[i].Data["departmentId"] != use.Department || after.Records[i].Data["jobId"] != use.Job {
					return foundation.Fail(409, "POSITION_ASSIGNMENT_USE", p.From+" 起已有岗位任职，请先核对实际安排")
				}
			}
		}
		if losesPositionEligibility(before, after) {
			if err = usage.CheckTightening(d.StaffingQuota{}, d.StaffingQuota{}, previous, proposed); err != nil {
				return err
			}
		}
		ids := append(previous.Departments(), proposed.Departments()...)
		if err := validateAllocation(ctx, u, after.Enterprise, ids, nil, &proposed); err != nil {
			return err
		}
	}
	if after.Kind != d.Position {
		if (after.Kind != d.Job && after.Kind != d.Grade) || !losesPositionEligibility(before, after) {
			return nil
		}
		field := string(after.Kind) + "Id"
		uses, err := u.PositionUses(ctx, after.Enterprise, field, after.ID)
		if err != nil {
			return err
		}
		first := ""
		for _, use := range uses {
			grade := ""
			if after.Kind == d.Job {
				grade = use.GradeID
			}
			date := d.PositionConflict(after, use.Start, use.End, grade)
			if date != "" && (first == "" || date < first) {
				first = date
			}
		}
		if first != "" {
			return foundation.Fail(409, "M03_REFERENCE_CONFLICT", "自 "+first+" 起仍有岗位采用此职务或职级，请先调整岗位")
		}
		return nil
	}
	catalog, err := positionCatalogFor(ctx, u, locked, before, after)
	if err != nil {
		return err
	}
	checked := map[d.PositionUseBoundary]bool{}
	authorize := func(field, id, date string, membership bool) error {
		use := d.PositionUseBoundary{Field: field, ID: id, Date: date, WithGrade: membership}
		if checked[use] {
			return nil
		}
		source := catalog[field][id]
		if source.ID == "" {
			return positionFault(field, date)
		}
		ok, err := positionRead(ctx, u, field, source, date, false, membership)
		if err != nil {
			return err
		}
		if !ok {
			return foundation.Fail(403, "M03_REFERENCE_FORBIDDEN", "没有读取岗位引用资料的权限")
		}
		checked[use] = true
		return nil
	}
	if before.ID != "" {
		for _, use := range d.ChangedPositionUses(before, after) {
			if err = authorize(use.Field, use.ID, use.Date, use.WithGrade); err != nil {
				return err
			}
		}
	}
	for _, old := range before.Records {
		index := d.Index(after, old.ID)
		for _, field := range d.PositionFields() {
			if id := old.Data[field]; id != "" && (index < 0 || after.Records[index].Data[field] != id) {
				if err = authorize(field, id, old.Date, field == "jobId" && old.Data["gradeId"] != ""); err != nil {
					return err
				}
			}
		}
	}
	for index := range after.Records {
		record := &after.Records[index]
		if !record.Active() {
			record.PositionRefs = maps.Clone(after.Records[index-1].PositionRefs)
			continue
		}
		oldIndex := d.Index(before, record.ID)
		var old d.Record
		if oldIndex >= 0 {
			old = before.Records[oldIndex]
		}
		record.PositionRefs = maps.Clone(old.PositionRefs)
		if record.PositionRefs == nil {
			record.PositionRefs = map[string]d.PositionAdoption{}
		}
		first, badField := "", ""
		for _, field := range d.PositionFields() {
			id := record.Data[field]
			if id == "" {
				delete(record.PositionRefs, field)
				continue
			}
			source := catalog[field][id]
			renew := old.ID == "" || old.Date != record.Date || !old.Active() || id != old.Data[field] || (field == "jobId" && old.Data["gradeId"] != record.Data["gradeId"]) || (field == "gradeId" && old.Data["jobId"] != record.Data["jobId"])
			if renew {
				if err = authorize(field, id, record.Date, field == "jobId" && record.Data["gradeId"] != ""); err != nil {
					return err
				}
			}
			if date := positionConflict(catalog, field, *record); date != "" {
				if first == "" || date < first {
					first, badField = date, field
				}
				continue
			}
			if renew {
				fact := source.Records[d.At(source.Timeline, record.Date)]
				record.PositionRefs[field] = d.PositionAdoption{ReferenceAdoption: d.ReferenceAdoption{BusinessAt: record.Date, SourceVersion: source.Source, RecordID: fact.ID, Revision: fact.Revision}, Name: fact.Data["name"]}
			}
		}
		if first != "" {
			return positionFault(badField, first)
		}
	}
	return nil
}

type positionCacheKey struct{}

func projectPosition(ctx context.Context, u Unit, record d.Record, view s.View, fields []string, date string, exact bool) error {
	adopted := map[string]d.PositionAdoption{}
	for _, field := range d.PositionFields() {
		if !has(fields, field) {
			continue
		}
		nameKey := strings.TrimSuffix(field, "Id") + "Name"
		delete(view, field)
		delete(view, nameKey)
		id := record.Data[field]
		if id == "" {
			view[field] = ""
			continue
		}
		cache, _ := ctx.Value(positionCacheKey{}).(map[string]PositionSource)
		key := field + ":" + id
		source, ok := cache[key]
		if !ok {
			refs, err := u.PositionSources(ctx, field, []string{id}, false)
			if err != nil {
				return err
			}
			if len(refs) != 1 {
				return errors.New("referenced position source unavailable")
			}
			source = refs[0]
			if cache != nil {
				cache[key] = source
			}
		}
		visible, err := positionRead(ctx, u, field, source, date, exact, false)
		if err != nil {
			return err
		}
		if !visible {
			view[nameKey] = "受限"
			continue
		}
		view[field] = id
		if exact {
			ref, found := record.PositionRefs[field]
			if !found || !foundation.ValidID(ref.SourceVersion) || !foundation.ValidID(ref.RecordID) || !foundation.ValidID(ref.Revision) {
				return errors.New("missing position adoption")
			}
			view[nameKey] = ref.Name
			adopted[field] = ref
		} else {
			index := d.At(source.Timeline, date)
			if index < 0 {
				return errors.New("position source does not cover display date")
			}
			view[nameKey] = source.Records[index].Data["name"]
		}
	}
	if exact && len(adopted) > 0 {
		view["positionAdoptions"] = adopted
	}
	return nil
}

func (svc Service) PositionReferences(ctx context.Context, cred foundation.Credentials, field string, cmd s.Command) (out s.View, err error) {
	if !slices.Contains(d.PositionFields(), field) || action(cmd.Input.Action) == "" || cmd.ObjectID != "" && !foundation.ValidID(cmd.ObjectID) {
		return nil, foundation.Fail(400, "M03_INVALID_INPUT", "候选请求无效")
	}
	page, size := cmd.Input.ReferencePage, cmd.Input.ReferencePageSize
	if page == 0 {
		page = 1
	}
	if size == 0 {
		size = 50
	}
	if page < 1 || page > 1000000 || size < 1 || size > 100 || utf8.RuneCountInString(cmd.Input.ReferenceQ) > 2000 {
		return nil, foundation.Fail(400, "M03_INVALID_INPUT", "候选查询范围无效")
	}
	err = svc.Store.Within(ctx, cred, false, func(u Unit) error {
		ent, err := u.Enterprise(ctx, false)
		if err != nil {
			return err
		}
		read, err := decide(ctx, u, d.Position, ent.ID, "read", nil)
		if err != nil {
			return err
		}
		write, err := decide(ctx, u, d.Position, ent.ID, action(cmd.Input.Action), nil)
		if err != nil {
			return err
		}
		if !has(read.ReadFields, field) || !has(write.WriteFields, field) || field == "gradeId" && !has(read.ReadFields, "jobId") {
			return foundation.Fail(403, "M03_FIELD_FORBIDDEN", "无权维护该岗位引用")
		}
		var before d.Timeline
		if cmd.ObjectID != "" {
			before, err = u.Load(ctx, d.Position, cmd.ObjectID, "")
			if err != nil {
				return err
			}
		}
		probe := command(cmd)
		plan, err := d.ReferencePlan(d.Position, ent.ID, before, probe)
		if err != nil {
			return err
		}
		idx := d.Index(plan.After, plan.Selected)
		if idx < 0 {
			return foundation.Fail(400, "M03_INVALID_INPUT", "未定位到记录")
		}
		record := plan.After.Records[idx]
		items := []s.View{}
		out = s.View{"items": items, "hasMore": false, "page": page, "effectiveDate": record.Date, "endExclusive": record.End}
		if !record.Active() || plan.After.Records[0].Date < ent.ValidFrom {
			return nil
		}
		others := d.Timeline{}
		for index, other := range plan.After.Records {
			if index != idx && other.Active() {
				others.Records = append(others.Records, other)
			}
		}
		if len(others.Records) > 0 {
			previous, err := positionCatalogFor(ctx, u, false, others)
			if err != nil {
				return err
			}
			for _, other := range others.Records {
				date, badField := "", ""
				for _, key := range d.PositionFields() {
					if conflict := positionConflict(previous, key, other); conflict != "" && (date == "" || conflict < date) {
						date, badField = conflict, key
					}
				}
				if date != "" {
					if !has(read.ReadFields, badField) {
						return foundation.Fail(403, "M03_FIELD_FORBIDDEN", "无权维护受影响的岗位引用")
					}
					return positionFault(badField, date)
				}
			}
		}
		catalog := positionCatalog{field: map[string]PositionSource{}}
		var gradeIDs []string
		checkGrade := field == "jobId" && record.Data["gradeId"] != ""
		if checkGrade {
			if !has(read.ReadFields, "gradeId") {
				return foundation.Fail(403, "M03_FIELD_FORBIDDEN", "无权维护该岗位职级")
			}
			id := record.Data["gradeId"]
			if !foundation.ValidID(id) {
				return foundation.Fail(400, "M03_INVALID_INPUT", "职级标识无效")
			}
			grades, err := u.PositionSources(ctx, "gradeId", []string{id}, false)
			if err != nil || len(grades) == 0 {
				return err
			}
			visible, err := positionRead(ctx, u, "gradeId", grades[0], record.Date, false, false)
			if err != nil {
				return err
			}
			if !visible {
				return foundation.Fail(403, "M03_REFERENCE_FORBIDDEN", "无权读取该职级")
			}
			catalog["gradeId"] = map[string]PositionSource{id: grades[0]}
		}
		if field == "gradeId" {
			id := record.Data["jobId"]
			if id == "" {
				return nil
			}
			if !foundation.ValidID(id) {
				return foundation.Fail(400, "M03_INVALID_INPUT", "职务标识无效")
			}
			jobs, err := u.PositionSources(ctx, "jobId", []string{id}, false)
			if err != nil {
				return err
			}
			if len(jobs) == 0 {
				return nil
			}
			ok, err := positionRead(ctx, u, "jobId", jobs[0], record.Date, false, true)
			if err != nil {
				return err
			}
			if !ok {
				return foundation.Fail(403, "M03_REFERENCE_FORBIDDEN", "无权读取该职务的可用职级")
			}
			catalog["jobId"] = map[string]PositionSource{id: jobs[0]}
			start := d.At(jobs[0].Timeline, record.Date)
			if start < 0 || !jobs[0].Records[start].Active() || len(jobs[0].Records[start].Grades) == 0 {
				return nil
			}
			for _, member := range jobs[0].Records[start].Grades {
				gradeIDs = append(gradeIDs, member.GradeID)
			}
		}
		needle := strings.ToLower(strings.TrimSpace(cmd.Input.ReferenceQ))
		skipped, skip := 0, (page-1)*size
		batchSize := max(32, size+1)
		for offset := 0; ; offset += batchSize {
			ids, err := u.PositionCandidateIDs(ctx, field, record.Date, needle, gradeIDs, offset, batchSize)
			if err != nil {
				return err
			}
			sources, err := u.PositionSources(ctx, field, ids, false)
			if err != nil {
				return err
			}
			if len(sources) != len(ids) {
				return errors.New("position candidate source unavailable")
			}
			catalog[field] = map[string]PositionSource{}
			for _, source := range sources {
				catalog[field][source.ID] = source
			}
			for _, id := range ids {
				source := catalog[field][id]
				record.Data[field] = source.ID
				if positionConflict(catalog, field, record) != "" || checkGrade && positionConflict(catalog, "gradeId", record) != "" {
					continue
				}
				ok, showCode, err := positionDisplay(ctx, u, field, source, record.Date, false, checkGrade)
				if err != nil {
					return err
				}
				if !ok {
					continue
				}
				name := source.Records[d.At(source.Timeline, record.Date)].Data["name"]
				if strings.Contains(strings.ToLower(name), needle) || showCode && strings.Contains(strings.ToLower(source.Code), needle) {
					if skipped < skip {
						skipped++
						continue
					}
					if len(items) == size {
						out["hasMore"] = true
						out["items"] = items
						return nil
					}
					item := s.View{"id": source.ID, "name": name}
					if showCode {
						item["code"] = source.Code
					}
					items = append(items, item)
				}
			}
			if len(ids) < batchSize {
				break
			}
		}
		out["items"] = items
		return nil
	})
	return
}

func positionBoundaries(before, after d.Timeline) []string {
	var out []string
	for _, t := range []d.Timeline{before, after} {
		for _, r := range t.Records {
			out = append(out, r.Date)
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

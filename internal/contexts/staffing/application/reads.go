package application

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	s "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"cmp"
	"context"
	"errors"
	"maps"
	"slices"
	"strings"
	"time"
)

type cacheKey struct{}
type decisionKey struct{}

func readContext(ctx context.Context) context.Context {
	ctx = context.WithValue(ctx, positionCacheKey{}, map[string]PositionSource{})
	return context.WithValue(context.WithValue(ctx, decisionKey{}, map[string]access.Decision{}), cacheKey{}, map[string]d.Timeline{})
}
func classificationTimeline(ctx context.Context, u Unit, kind d.Kind, id, source string) (d.Timeline, error) {
	cache, _ := ctx.Value(cacheKey{}).(map[string]d.Timeline)
	key := string(kind) + ":" + id + ":" + source
	if t, ok := cache[key]; ok {
		return t, nil
	}
	t, e := u.Load(ctx, kind, id, source)
	if e == nil && cache != nil {
		cache[key] = t
	}
	return t, e
}
func allowed(ctx context.Context, u Unit, k d.Kind, ent, purpose string) (access.Decision, error) {
	cache, _ := ctx.Value(decisionKey{}).(map[string]access.Decision)
	key := string(k) + ":" + ent + ":" + purpose
	if v, ok := cache[key]; ok {
		return v, nil
	}
	dec, e := u.Access().Decide(ctx, access.Intent{EnterpriseID: ent, Kind: string(k), Action: string(k) + "." + purpose})
	if e == nil && cache != nil {
		cache[key] = dec
	}
	return dec, e
}
func capabilities(ctx context.Context, u Unit, k d.Kind, ent string, read access.Decision) (s.View, error) {
	v := s.View{"readFields": read.ReadFields, "writeFields": []string{}, "actions": map[string]bool{}, "actionFields": map[string][]string{}}
	actions := v["actions"].(map[string]bool)
	fields := v["actionFields"].(map[string][]string)
	for _, op := range []string{"CREATE", "ADD", "CORRECT", "REDATE", "DELETE", "DEACTIVATE", "REACTIVATE", "INSERT_LIFECYCLE_PAIR", "MAINTAIN_LIFECYCLE_PAIR", "DELETE_LIFECYCLE_PAIR"} {
		dec, e := allowed(ctx, u, k, ent, action(op))
		if e != nil {
			return nil, e
		}
		actions[op] = dec.Allowed
		fields[op] = []string{}
		if dec.Allowed {
			for _, f := range dec.WriteFields {
				if slices.Contains(read.ReadFields, f) {
					fields[op] = append(fields[op], f)
				}
			}
		}
		if op == "CREATE" {
			required := []string{"code", "name", "effectiveDate", "operationType", "operationReason"}
			if k == d.Job {
				required = append(required, "jobFamilyId", "jobFunctionId")
			}
			if k == d.Ladder {
				required = append(required, "gradeIds")
			}
			if k == d.Position {
				required = append(required, "departmentId", "jobId", "locationId")
			}
			actions[op] = actions[op] && has(fields[op], required...)
			v["writeFields"] = fields[op]
		}
		if op == "REDATE" {
			actions[op] = actions[op] && has(fields[op], "effectiveDate")
		}
		if op == "ADD" || op == "DEACTIVATE" || op == "REACTIVATE" || strings.Contains(op, "LIFECYCLE_PAIR") && op != "DELETE_LIFECYCLE_PAIR" {
			actions[op] = actions[op] && has(fields[op], "effectiveDate", "operationType", "operationReason")
		}
		if op == "CORRECT" {
			actions[op] = actions[op] && len(fields[op]) > 0
		}
	}
	for _, purpose := range []string{"history.read", "export"} {
		dec, e := allowed(ctx, u, k, ent, purpose)
		if e != nil {
			return nil, e
		}
		actions[purpose] = dec.Allowed
	}
	return v, nil
}
func (svc Service) Context(ctx context.Context, c foundation.Credentials, k s.Kind) (out s.View, err error) {
	err = svc.Store.Within(ctx, c, false, func(u Unit) error {
		ent, e := u.Enterprise(ctx, false)
		if e != nil {
			return e
		}
		read, e := decide(ctx, u, d.Kind(k), ent.ID, "read", nil)
		if e != nil {
			return e
		}
		caps, e := capabilities(ctx, u, d.Kind(k), ent.ID, read)
		if e != nil {
			return e
		}
		if k == s.Position && has(read.ReadFields, "name", "effectiveDate", "status") {
			rd, err := allowed(ctx, u, hierarchyKind, ent.ID, "read")
			if err != nil {
				return err
			}
			if rd.Allowed {
				caps["actions"].(map[string]bool)["positionRelations"] = true
				wr, err := allowed(ctx, u, hierarchyKind, ent.ID, "publish")
				if err != nil {
					return err
				}
				caps["actions"].(map[string]bool)["adjustSuperior"] = wr.Allowed
			}
		}
		if k == s.Position && has(read.ReadFields, "departmentId") {
			planner, ok := u.Access().(access.ReadPlanner)
			if !ok {
				return foundation.Fail(503, "M03_DEPENDENCY_UNAVAILABLE", "部门范围权限暂不可读")
			}
			plan, err := planner.PlanReads(ctx, ent.ID, u.Now(), access.DepartmentRead)
			if err != nil {
				return err
			}
			caps["actions"].(map[string]bool)["departmentRange"] = plan.Allowed.All || len(plan.Allowed.BUs) > 0 || len(plan.Allowed.Departments) > 0
		}
		out = s.View{"currentDate": today(u), "defaultEffectiveDate": foundation.DefaultEffectiveDate, "enterpriseFirstDate": ent.ValidFrom, "timezone": "Asia/Shanghai", "nameMaxLength": 200, "capabilities": caps}
		return e
	})
	return
}
func (svc Service) Reasons(ctx context.Context, c foundation.Credentials, k s.Kind, op string) (s.View, error) {
	if _, e := svc.Context(ctx, c, k); e != nil {
		return nil, e
	}
	reasons := d.Reasons(d.Kind(k), op)
	if len(reasons) == 0 {
		return nil, foundation.Fail(400, "M03_INVALID_INPUT", "操作类型无效")
	}
	keys := []string{}
	for key := range reasons {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	items := []s.View{}
	for _, key := range keys {
		items = append(items, s.View{"code": key, "label": reasons[key]})
	}
	return s.View{"items": items}, nil
}
func dayBefore(date string) string {
	t, e := time.Parse("2006-01-02", date)
	if e != nil {
		return ""
	}
	return t.AddDate(0, 0, -1).Format("2006-01-02")
}
func dayAfter(date string) string {
	t, e := time.Parse("2006-01-02", date)
	if e != nil {
		return ""
	}
	return t.AddDate(0, 0, 1).Format("2006-01-02")
}
func selected(t d.Timeline, date string) (int, string) {
	i := d.At(t, date)
	if i < 0 {
		return 0, "not_effective"
	}
	if t.Records[i].Active() {
		return i, "active"
	}
	return i, "inactive"
}
func recordView(ctx context.Context, u Unit, t d.Timeline, r d.Record, fields []string, date string, exact bool) (s.View, error) {
	v := s.View{"recordId": r.ID, "revision": r.Revision}
	if r.Summary.SourceVersion != "" {
		v["operationSummary"] = r.Summary
	}
	if has(fields, "effectiveDate") {
		v["effectiveDate"] = r.Date
		v["endExclusive"] = nil
		if r.End != "" {
			v["endExclusive"] = r.End
			v["endInclusive"] = dayBefore(r.End)
		}
	}
	if has(fields, "status") {
		state := "active"
		if !r.Active() {
			state = "inactive"
		}
		v["state"] = state
	}
	if has(fields, "operationType") {
		v["operationType"] = r.Type
	}
	if has(fields, "operationReason") {
		v["operationReason"] = r.Reason
		v["reasonLabel"] = r.ReasonLabel
	}
	for key, value := range r.Data {
		if has(fields, key) && key != "jobFamilyId" && key != "jobFunctionId" && !(t.Kind == d.Position && slices.Contains(d.PositionFields(), key)) {
			v[key] = value
		}
	}
	if e := projectMembers(ctx, u, t, r, v, fields, date, exact); e != nil {
		return nil, e
	}
	if t.Kind == d.Position {
		return v, projectPosition(ctx, u, r, v, fields, date, exact)
	}
	if t.Kind == d.Job && has(fields, "jobFunctionId") {
		if err := projectFunction(ctx, u, t, r, v, date, exact); err != nil {
			return nil, err
		}
	}
	if t.Kind != d.Job || !has(fields, "jobFamilyId") {
		return v, nil
	}
	v["jobFamilyName"] = "受限"
	rd, e := allowed(ctx, u, d.Family, t.Enterprise, "read")
	if e != nil {
		return nil, e
	}
	if !rd.Allowed || !has(rd.ReadFields, "name", "effectiveDate") {
		return v, nil
	}
	history := exact || date != today(u)
	if history {
		hd, e := allowed(ctx, u, d.Family, t.Enterprise, "history.read")
		if e != nil {
			return nil, e
		}
		if !hd.Allowed || !has(hd.ReadFields, "name", "effectiveDate") {
			return v, nil
		}
	}
	source := ""
	if exact {
		source = r.Adoption.Source
	}
	f, e := classificationTimeline(ctx, u, d.Family, r.Data["jobFamilyId"], source)
	if e != nil {
		return nil, e
	}
	i := d.At(f, date)
	if exact {
		i = d.Index(f, r.Adoption.RecordID)
	}
	if i < 0 || f.Enterprise != t.Enterprise {
		return v, foundation.Fail(503, "M03_DEPENDENCY_UNAVAILABLE", "职务系列引用资料暂不可读")
	}
	fr := f.Records[i]
	if exact && fr.Revision != r.Adoption.Revision {
		return v, foundation.Fail(503, "M03_DEPENDENCY_UNAVAILABLE", "职务系列采用依据暂不可读")
	}
	v["jobFamilyId"] = f.ID
	v["jobFamilyName"] = fr.Data["name"]
	v["jobFamilyDisplayDate"] = date
	if exact {
		v["adoption"] = s.View{"date": r.Adoption.Date, "sourceVersion": r.Adoption.Source, "recordId": r.Adoption.RecordID, "revision": r.Adoption.Revision}
	}
	return v, nil
}
func detailView(ctx context.Context, u Unit, t d.Timeline, read access.Decision, recordID string, exact bool) (s.View, error) {
	if t.ID == "" {
		return s.View{}, nil
	}
	index, status := selected(t, today(u))
	missing := false
	if recordID != "" {
		j := d.Index(t, recordID)
		if j >= 0 {
			index = j
		} else {
			missing = true
		}
	}
	timeline := []s.View{}
	for _, r := range t.Records {
		date := r.Date
		if !r.Active() {
			date = dayBefore(r.Date)
		}
		rv, e := recordView(ctx, u, t, r, read.ReadFields, date, exact)
		if e != nil {
			return nil, e
		}
		timeline = append(timeline, rv)
	}
	caps, e := capabilities(ctx, u, t.Kind, t.Enterprise, read)
	if e != nil {
		return nil, e
	}
	v := s.View{"id": t.ID, "currentDate": today(u), "resolvedSourceVersion": t.Source, "headSourceVersion": t.Source, "timeline": timeline, "selectedRecord": timeline[index], "selectionMissing": missing, "readOnly": exact, "capabilities": caps, "selectedRecordIsFuture": has(read.ReadFields, "effectiveDate") && t.Records[index].Date > today(u)}
	if t.Kind == d.Position && !exact {
		quota, e := u.Access().Decide(ctx, access.Intent{Kind: "quota-target", EnterpriseID: t.Enterprise, PositionID: t.ID, Action: "staffingquota.read"})
		if e != nil {
			return nil, e
		}
		v["canReadQuota"] = quota.Allowed
	}
	if has(read.ReadFields, "code") {
		v["code"] = t.Code
	}
	if has(read.ReadFields, "status") {
		v["status"] = status
	}
	if !exact && recordID == "" {
		r := t.Records[index]
		date := today(u)
		if status == "not_effective" {
			date = r.Date
		} else if !r.Active() {
			date = dayBefore(r.Date)
		}
		current := maps.Clone(timeline[index])
		delete(current, "gradeIds")
		delete(current, "gradeMembers")
		delete(current, "gradesRestricted")
		if t.Kind == d.Job && has(read.ReadFields, "jobFunctionId") {
			if err := projectFunction(ctx, u, t, r, current, date, false); err != nil {
				return nil, err
			}
		}
		if e := projectMembers(ctx, u, t, r, current, read.ReadFields, date, false); e != nil {
			return nil, e
		}
		if t.Kind == d.Position {
			if err := projectPosition(ctx, u, r, current, read.ReadFields, date, false); err != nil {
				return nil, err
			}
		}
		v["selectedRecord"] = current
	}
	if !exact {
		actions := caps["actions"].(map[string]bool)
		actions["DELETE"] = actions["DELETE"] && index > 0
		r := t.Records[index]
		if r.Active() {
			v["additionContext"] = s.View{"anchorRecordId": r.ID, "available": r.End == "" || dayAfter(r.Date) < r.End, "defaultEffectiveDate": dayAfter(r.Date), "minEffectiveDate": dayAfter(r.Date), "maxEffectiveDate": dayBefore(r.End), "mode": map[bool]string{true: "APPEND", false: "INSERT"}[index == len(t.Records)-1]}
			if r.End == "" || dayAfter(dayAfter(r.Date)) < r.End {
				v["pairInsertion"] = s.View{"anchorRecordId": r.ID, "defaultStopDate": dayAfter(r.Date), "defaultResumeDate": dayAfter(dayAfter(r.Date))}
			}
		}
		if index == len(t.Records)-1 {
			op := "DEACTIVATE"
			if !r.Active() {
				op = "REACTIVATE"
			}
			v["lifecycleContext"] = s.View{"anchorRecordId": r.ID, "operationType": op, "defaultEffectiveDate": dayAfter(r.Date)}
		}
		pair := index
		if r.Type == "REACTIVATE" {
			pair--
		}
		if pair > 0 && pair+1 < len(t.Records) && t.Records[pair].Type == "DEACTIVATE" && t.Records[pair+1].Type == "REACTIVATE" {
			v["pairContext"] = s.View{"deactivation": timeline[pair], "reactivation": timeline[pair+1]}
		}
	}
	return v, nil
}
func (svc Service) Detail(ctx context.Context, c foundation.Credentials, k s.Kind, id, source, record string) (out s.View, err error) {
	ctx = readContext(ctx)
	if !foundation.ValidID(id) || (source != "" && !foundation.ValidID(source)) || (record != "" && !foundation.ValidID(record)) {
		return nil, foundation.Fail(400, "M03_INVALID_INPUT", "资料标识无效")
	}
	err = svc.Store.Within(ctx, c, false, func(u Unit) error {
		ent, e := u.Enterprise(ctx, false)
		if e != nil {
			return e
		}
		read, e := decide(ctx, u, d.Kind(k), ent.ID, "read", nil)
		if e != nil {
			return e
		}
		if source != "" {
			hd, e := decide(ctx, u, d.Kind(k), ent.ID, "history.read", nil)
			if e != nil {
				return e
			}
			read.ReadFields = intersection(read.ReadFields, hd.ReadFields)
		}
		head, e := u.Load(ctx, d.Kind(k), id, "")
		if e != nil {
			return e
		}
		t := head
		if source != "" {
			t, e = u.Load(ctx, d.Kind(k), id, source)
			if e != nil {
				return e
			}
		}
		if t.ID == "" {
			return foundation.Fail(404, "M03_OBJECT_NOT_FOUND", "资料不存在或当前不可见")
		}
		out, e = detailView(ctx, u, t, read, record, source != "")
		if e == nil {
			out["headSourceVersion"] = head.Source
		}
		return e
	})
	return
}
func intersection(a, b []string) []string {
	v := []string{}
	for _, s := range a {
		if slices.Contains(b, s) {
			v = append(v, s)
		}
	}
	return v
}
func summary(ctx context.Context, u Unit, t d.Timeline, fields []string) (s.View, error) {
	i, state := selected(t, today(u))
	r := t.Records[i]
	date := today(u)
	if state == "not_effective" {
		date = r.Date
	} else if state == "inactive" {
		date = dayBefore(r.Date)
	}
	// List summaries have no member column. Keep indexed and filtered queries
	// identical; details and the Ladder CSV resolve complete members separately.
	summaryFields := slices.DeleteFunc(slices.Clone(fields), func(field string) bool {
		return field == "gradeIds" || t.Kind == d.Position && slices.Contains(d.PositionFields(), field)
	})
	v, e := recordView(ctx, u, t, r, summaryFields, date, false)
	if e != nil {
		return nil, e
	}
	v["id"] = t.ID
	if has(fields, "code") {
		v["code"] = t.Code
	}
	if has(fields, "status") {
		v["status"] = state
	}
	return v, nil
}

type classificationDimension struct {
	kind             d.Kind
	field, id, label string
}

func classificationDimensions(q s.Query) [2]classificationDimension {
	return [2]classificationDimension{
		{d.Family, "jobFamilyId", q.JobFamilyID, "职务系列"},
		{d.JobFunction, "jobFunctionId", q.JobFunctionID, "职类"},
	}
}
func queryCheck(ctx context.Context, u Unit, k d.Kind, ent string, q s.Query, fields []string) error {
	if q.ScopeDepartmentID != "" || q.IncludeDescendants != nil {
		if k != d.Position || !foundation.ValidID(q.ScopeDepartmentID) {
			return foundation.Fail(400, "M03_INVALID_INPUT", "岗位部门范围无效")
		}
		if !has(fields, "departmentId") {
			return foundation.Fail(403, "M03_FILTER_FORBIDDEN", "无权按所属部门筛选")
		}
	}

	if strings.TrimSpace(q.Q) != "" && !has(fields, "code") && !has(fields, "name") && !has(fields, "nameEn") {
		return foundation.Fail(403, "M03_FILTER_FORBIDDEN", "关键字筛选权限已变化，请清除筛选后重试")
	}
	if q.Status != "" && q.Status != "all" && q.Status != "active" && q.Status != "inactive" && q.Status != "not_effective" {
		return foundation.Fail(400, "M03_INVALID_INPUT", "状态筛选无效")
	}
	if q.Status != "" && q.Status != "all" && !has(fields, "status") {
		return foundation.Fail(403, "M03_FILTER_FORBIDDEN", "状态筛选权限已变化，请清除筛选后重试")
	}
	classification := q.JobFamilyID != "" || q.JobFunctionID != "" || q.JobFunctionMissing
	if classification && (k != d.Job || q.JobFunctionMissing && q.JobFunctionID != "") {
		return foundation.Fail(400, "M03_INVALID_INPUT", "分类筛选组合无效")
	}
	if q.JobFunctionMissing && !has(fields, "jobFunctionId") {
		return foundation.Fail(403, "M03_FILTER_FORBIDDEN", "职类筛选权限已变化，请清除筛选后重试")
	}
	for _, target := range classificationDimensions(q) {
		if target.id == "" {
			continue
		}
		if !foundation.ValidID(target.id) {
			return foundation.Fail(400, "M03_INVALID_INPUT", target.label+"筛选无效")
		}
		denied := foundation.Fail(403, "M03_FILTER_FORBIDDEN", target.label+"筛选权限已变化，请清除筛选后重试")
		rd, err := allowed(ctx, u, target.kind, ent, "read")
		if err != nil {
			return err
		}
		if !has(fields, target.field) || !rd.Allowed || !has(rd.ReadFields, "name", "effectiveDate") {
			return denied
		}
		targetView, err := classificationTimeline(ctx, u, target.kind, target.id, "")
		if err != nil {
			return err
		}
		if targetView.ID == "" || targetView.Enterprise != ent {
			return denied
		}
	}
	sortable := []string{"code", "name", "nameEn", "description", "responsibilities", "requirements", "status", "effectiveDate"}
	filterable := slices.Clone(sortable)
	if k == d.Grade || k == d.Ladder {
		sortable = []string{"code", "name", "nameEn"}
		if q.Sort == "-code" {
			return foundation.Fail(400, "M03_INVALID_INPUT", "职级编码仅支持升序")
		}
	}
	sort := strings.TrimPrefix(q.Sort, "-")
	if sort != "" && (!slices.Contains(sortable, sort) || !has(fields, sort)) {
		return foundation.Fail(403, "M03_FILTER_FORBIDDEN", "排序字段不可用，请重新选择")
	}
	if q.FilterField != "" {
		if !has(fields, q.FilterField) || !slices.Contains(filterable, q.FilterField) {
			return foundation.Fail(403, "M03_FILTER_FORBIDDEN", "筛选字段不可用，请清除筛选后重试")
		}
		if !slices.Contains([]string{"contains", "equals", "startsWith", "endsWith", "isEmpty", "isNotEmpty", "after", "before"}, q.FilterOperator) {
			return foundation.Fail(400, "M03_INVALID_INPUT", "筛选运算符无效")
		}
	}
	return nil
}
func matches(v s.View, q s.Query) bool {
	value := func(k string) string { s, _ := v[k].(string); return s }
	if q.Q != "" {
		needle := strings.ToLower(strings.TrimSpace(q.Q))
		found := false
		for _, field := range []string{"code", "name", "nameEn"} {
			found = found || strings.Contains(strings.ToLower(value(field)), needle)
		}
		if !found {
			return false
		}
	}
	if q.Status != "" && q.Status != "all" && value("status") != q.Status {
		return false
	}
	if q.FilterField != "" {
		a, b := strings.ToLower(value(q.FilterField)), strings.ToLower(q.FilterValue)
		switch q.FilterOperator {
		case "contains":
			return strings.Contains(a, b)
		case "equals":
			return a == b
		case "startsWith":
			return strings.HasPrefix(a, b)
		case "endsWith":
			return strings.HasSuffix(a, b)
		case "isEmpty":
			return a == ""
		case "isNotEmpty":
			return a != ""
		case "after":
			return a > b
		case "before":
			return a < b
		}
	}
	return true
}

// Future first records and inactive records display dated reference labels.
func canMatchHistoricalClassifications(ctx context.Context, u Unit, ent string, q s.Query) (bool, error) {
	for _, ref := range classificationDimensions(q) {
		if ref.id == "" {
			continue
		}
		dec, err := allowed(ctx, u, ref.kind, ent, "history.read")
		if err != nil {
			return false, err
		}
		if !dec.Allowed || !has(dec.ReadFields, "name", "effectiveDate") {
			return false, nil
		}
	}
	return true, nil
}
func listAll(ctx context.Context, u Unit, k d.Kind, q s.Query) ([]s.View, access.Decision, map[string]d.Timeline, error) {
	ent, e := u.Enterprise(ctx, false)
	if e != nil {
		return nil, access.Decision{}, nil, e
	}
	read, e := decide(ctx, u, k, ent.ID, "read", nil)
	if e != nil {
		return nil, read, nil, e
	}
	if e = queryCheck(ctx, u, k, ent.ID, q, read.ReadFields); e != nil {
		return nil, read, nil, e
	}
	ts, e := u.Current(ctx, k)
	if e != nil {
		return nil, read, nil, e
	}
	departments := map[string]bool{}
	if q.ScopeDepartmentID != "" {
		for _, t := range ts {
			i, _ := selected(t, today(u))
			departments[t.Records[i].Data["departmentId"]] = false
		}
		ids := slices.Sorted(maps.Keys(departments))
		matches, err := u.PositionDepartments(ctx, q.ScopeDepartmentID, q.IncludeDescendants == nil || *q.IncludeDescendants, ids)
		if err != nil {
			return nil, read, nil, err
		}
		for _, id := range matches {
			departments[id] = true
		}
	}
	// Filtering/navigation/capture never need reference labels. Resolve labels only
	// for the visible page, retaining the full authorized query and stable IDs.
	narrow := slices.DeleteFunc(slices.Clone(read.ReadFields), func(f string) bool { return f == "jobFamilyId" || f == "jobFunctionId" })
	historical, e := canMatchHistoricalClassifications(ctx, u, ent.ID, q)
	if e != nil {
		return nil, read, nil, e
	}
	out := []s.View{}
	current := map[string]d.Timeline{}
	for _, t := range ts {
		if q.ScopeDepartmentID != "" {
			i, _ := selected(t, today(u))
			if !departments[t.Records[i].Data["departmentId"]] {
				continue
			}
		}
		v, e := summary(ctx, u, t, narrow)
		if e != nil {
			return nil, read, nil, e
		}
		i, state := selected(t, today(u))
		data := t.Records[i].Data
		if q.JobFamilyID != "" && data["jobFamilyId"] != q.JobFamilyID ||
			q.JobFunctionID != "" && data["jobFunctionId"] != q.JobFunctionID ||
			q.JobFunctionMissing && data["jobFunctionId"] != "" {
			continue
		}
		if state != "active" && !historical {
			continue
		}
		if matches(v, q) {
			out = append(out, v)
			current[t.ID] = t
		}
	}
	sortViews(out, q.Sort, read.ReadFields)
	return out, read, current, nil
}
func sortViews(v []s.View, sort string, fields []string) {
	field := strings.TrimPrefix(sort, "-")
	if field == "" {
		field = "code"
	}
	if !has(fields, field) {
		field = "id"
	}
	slices.SortFunc(v, func(a, b s.View) int {
		av, _ := a[field].(string)
		bv, _ := b[field].(string)
		n := cmp.Compare(av, bv)
		if strings.HasPrefix(sort, "-") {
			n = -n
		}
		if n == 0 {
			n = cmp.Compare(a["id"].(string), b["id"].(string))
		}
		return n
	})
}
func (svc Service) List(ctx context.Context, c foundation.Credentials, k s.Kind, q s.Query) (out s.View, err error) {
	ctx = readContext(ctx)
	err = svc.Store.Within(ctx, c, false, func(u Unit) error {
		page, size := q.Page, q.PageSize
		if page == 0 {
			page = 1
		}
		if size == 0 {
			size = 20
		}
		if page < 1 || !slices.Contains([]int{20, 50, 100}, size) {
			return foundation.Fail(400, "M03_INVALID_INPUT", "分页参数无效")
		}
		ent, e := u.Enterprise(ctx, false)
		if e != nil {
			return e
		}
		var v []s.View
		var read access.Decision
		var current map[string]d.Timeline
		var total int
		paged := q.ScopeDepartmentID == "" && q.Q == "" && (q.Status == "" || q.Status == "all") && q.JobFamilyID == "" && q.JobFunctionID == "" && !q.JobFunctionMissing && q.FilterField == "" && (q.Sort == "" || q.Sort == "code" || q.Sort == "-code")
		if paged {
			read, e = decide(ctx, u, d.Kind(k), ent.ID, "read", nil)
			if e != nil {
				return e
			}
			if e = queryCheck(ctx, u, d.Kind(k), ent.ID, q, read.ReadFields); e != nil {
				return e
			}
			var ts []d.Timeline
			ts, total, e = u.CurrentPage(ctx, d.Kind(k), (page-1)*size, size, has(read.ReadFields, "code"), q.Sort == "-code")
			if e != nil {
				return e
			}
			narrow := slices.DeleteFunc(slices.Clone(read.ReadFields), func(f string) bool { return f == "jobFamilyId" || f == "jobFunctionId" })
			v = []s.View{}
			current = map[string]d.Timeline{}
			for _, t := range ts {
				item, err := summary(ctx, u, t, narrow)
				if err != nil {
					return err
				}
				v = append(v, item)
				current[t.ID] = t
			}
		} else {
			v, read, current, e = listAll(ctx, u, d.Kind(k), q)
			if e != nil {
				return e
			}
			total = len(v)
			start := min((page-1)*size, total)
			v = v[start:min(start+size, total)]
		}
		caps, e := capabilities(ctx, u, d.Kind(k), ent.ID, read)
		if e != nil {
			return e
		}
		pageItems := v
		if d.Kind(k) == d.Job && (has(read.ReadFields, "jobFamilyId") || has(read.ReadFields, "jobFunctionId")) {
			for _, ref := range classificationDimensions(q) {
				if !has(read.ReadFields, ref.field) {
					continue
				}
				ids := []string{}
				for _, item := range pageItems {
					t := current[item["id"].(string)]
					i, _ := selected(t, today(u))
					if id := t.Records[i].Data[ref.field]; id != "" {
						ids = append(ids, id)
					}
				}
				if len(ids) == 0 {
					continue
				}
				rd, e := allowed(ctx, u, ref.kind, ent.ID, "read")
				if e != nil {
					return e
				}
				if !rd.Allowed || !has(rd.ReadFields, "name", "effectiveDate") {
					continue
				}
				slices.Sort(ids)
				heads, e := u.Heads(ctx, ref.kind, slices.Compact(ids))
				if e != nil {
					return e
				}
				cache := ctx.Value(cacheKey{}).(map[string]d.Timeline)
				for _, h := range heads {
					cache[string(ref.kind)+":"+h.ID+":"] = h
				}
			}
			for i, item := range pageItems {
				pageItems[i], e = summary(ctx, u, current[item["id"].(string)], read.ReadFields)
				if e != nil {
					return e
				}
			}
		}

		out = s.View{"items": pageItems, "totalCount": total, "page": page, "pageSize": size, "businessAt": today(u), "capabilities": caps}
		for _, ref := range classificationDimensions(q) {
			if ref.id == "" {
				continue
			}
			// queryCheck already authorized and cached this directory head in this snapshot.
			target, err := classificationTimeline(ctx, u, ref.kind, ref.id, "")
			if err != nil {
				return err
			}
			index, state := selected(target, today(u))
			if state != "active" {
				dec, err := allowed(ctx, u, ref.kind, ent.ID, "history.read")
				if err != nil {
					return err
				}
				if !dec.Allowed || !has(dec.ReadFields, "name", "effectiveDate") {
					continue
				}
			}
			out[strings.TrimSuffix(ref.field, "Id")+"FilterName"] = target.Records[index].Data["name"]
		}
		return e
	})
	return
}
func (svc Service) Navigation(ctx context.Context, c foundation.Credentials, k s.Kind, id string, q s.Query) (out s.View, err error) {
	if !foundation.ValidID(id) {
		return nil, foundation.Fail(400, "M03_INVALID_INPUT", "资料标识无效")
	}
	ctx = readContext(ctx)
	err = svc.Store.Within(ctx, c, false, func(u Unit) error {
		v, _, _, e := listAll(ctx, u, d.Kind(k), q)
		if e != nil {
			return e
		}
		i := slices.IndexFunc(v, func(v s.View) bool { return v["id"] == id })
		out = s.View{"position": 0, "totalCount": len(v), "previousId": nil, "nextId": nil, "businessAt": today(u)}
		if i >= 0 {
			out["position"] = i + 1
			if i > 0 {
				out["previousId"] = v[i-1]["id"]
			}
			if i+1 < len(v) {
				out["nextId"] = v[i+1]["id"]
			}
		}
		return nil
	})
	return
}
func (svc Service) ClassificationReferenceContext(ctx context.Context, c foundation.Credentials, target s.Kind, cmd s.Command) (out s.View, err error) {
	kind := d.Kind(target)
	field := "jobFamilyId"
	selectedID := cmd.Input.JobFamilyID
	switch kind {
	case d.Family:
	case d.JobFunction:
		field = "jobFunctionId"
		selectedID = cmd.Input.JobFunctionID
	default:
		return nil, foundation.Fail(400, "M03_INVALID_INPUT", "分类候选类型无效")
	}
	if cmd.Input.ReferencePage < 0 || cmd.Input.ReferencePage > 1000000 {
		return nil, foundation.Fail(400, "M03_INVALID_INPUT", "候选页码无效")
	}
	ctx = readContext(ctx)
	err = svc.Store.Within(ctx, c, false, func(u Unit) error {
		ent, e := u.Enterprise(ctx, false)
		if e != nil {
			return e
		}
		jobRead, e := decide(ctx, u, d.Job, ent.ID, "read", nil)
		if e != nil {
			return e
		}
		if !has(jobRead.ReadFields, field) {
			return foundation.Fail(403, "M03_FIELD_FORBIDDEN", "无权读取该职务分类字段")
		}
		rd, e := decide(ctx, u, kind, ent.ID, "read", nil)
		if e != nil {
			return e
		}
		if !has(rd.ReadFields, "name", "effectiveDate", "status") {
			return foundation.Fail(403, "M03_REFERENCE_FORBIDDEN", "无权读取该分类候选")
		}
		candidates, e := u.ReferenceClassifications(ctx, kind)
		if e != nil {
			return e
		}
		before := d.Timeline{}
		if cmd.ObjectID != "" {
			before, e = u.Load(ctx, d.Job, cmd.ObjectID, "")
			if e != nil {
				return e
			}
		}
		items := []s.View{}
		var selectedName string
		byID := make(map[string]d.Timeline, len(candidates))
		for _, candidate := range candidates {
			byID[candidate.ID] = candidate
		}
		probe := command(cmd)
		probe.Grades = d.MembershipPatch{}
		if probe.Action == "CREATE" {
			probe.Type = "CREATE"
		}
		for _, f := range candidates {
			id := f.ID
			probe.Changes = map[string]*string{field: &id}
			p, e := d.ReferencePlan(d.Job, ent.ID, before, probe)
			if e != nil {
				var fault *foundation.Fault
				if errors.As(e, &fault) && fault.Status < 500 {
					continue
				}
				return e
			}
			selected := d.Index(p.After, p.Selected)
			if selected < 0 || p.After.Records[0].Date < ent.ValidFrom {
				continue
			}
			record := p.After.Records[selected]
			if !record.Active() || record.Data[field] != id {
				continue
			}
			if kind == d.Family && d.ValidateCoverage(p.After, byID) != nil ||
				kind == d.JobFunction && d.FunctionCoverage(p.After, f, id) != nil {
				continue
			}
			date := record.Date
			if date != today(u) {
				hd, e := allowed(ctx, u, kind, ent.ID, "history.read")
				if e != nil {
					return e
				}
				if !hd.Allowed || !has(hd.ReadFields, "name", "effectiveDate", "status") {
					continue
				}
			}
			i := d.At(f, date)
			if i < 0 {
				continue
			}
			if selectedID != nil && *selectedID == id {
				selectedName = f.Records[i].Data["name"]
			}
			if !strings.Contains(strings.ToLower(f.Records[i].Data["name"]), strings.ToLower(strings.TrimSpace(cmd.Input.ReferenceQ))) {
				continue
			}
			items = append(items, s.View{"id": id, "name": f.Records[i].Data["name"], "date": date, "sourceVersion": f.Source})
		}
		page := max(1, cmd.Input.ReferencePage)
		start := min((page-1)*20, len(items))
		out = s.View{"items": items[start:min(start+20, len(items))], "totalCount": len(items), "page": page, "pageSize": 20, "businessAt": today(u)}
		if selectedID != nil && *selectedID != "" {
			out["selectedAvailable"] = selectedName != ""
			if selectedName != "" {
				out["selectedName"] = selectedName
			}
		}
		return nil
	})
	return
}

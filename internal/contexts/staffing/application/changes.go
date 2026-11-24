package application

import (
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	s "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"maps"
	"reflect"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

var logActions = []string{"CREATE", "ADD", "CORRECT", "REDATE", "DELETE", "DEACTIVATE", "REACTIVATE", "INSERT_LIFECYCLE_PAIR", "MAINTAIN_LIFECYCLE_PAIR", "DELETE_LIFECYCLE_PAIR"}

func logInvalid(message string) error { return foundation.Fail(400, "M03_INVALID_INPUT", message) }
func logFieldAllowed(field string, fields []string) bool {
	if field == "period" {
		field = "effectiveDate"
	}
	return slices.Contains(fields, field)
}
func checkLogQuery(q s.LogQuery, fields []string, kind d.Kind) error {
	if q.Page < 1 || !slices.Contains([]int{20, 50, 100}, q.PageSize) {
		return logInvalid("日志分页参数无效")
	}
	for _, id := range []string{q.ContextSourceVersion, q.ThroughSourceVersion} {
		if id != "" && !foundation.ValidID(id) {
			return logInvalid("日志来源标识无效")
		}
	}
	if q.FilterField == "" && q.FilterOperator == "" && q.FilterValue == "" {
		return nil
	}
	actions, filterFields := logActions, append(d.Fields(kind), "period")
	if kind == d.Kind(s.Quota) {
		actions = []string{"create", "adjust", "withdraw", "correct", "allocate", "redate", "delete"}
		filterFields = []string{"effectiveDate", "period", "quota", "reason"}
		if (q.FilterField == "actor" || q.FilterField == "recordedAt") && !has(fields, "audit") {
			return foundation.Fail(403, "M03_FILTER_FORBIDDEN", "无权筛选操作审计")
		}
	}
	value := strings.TrimSpace(q.FilterValue)
	if q.FilterField == "" || q.FilterOperator == "" || value == "" || utf8.RuneCountInString(value) > 400 {
		return logInvalid("日志筛选条件不完整或无效")
	}
	switch q.FilterField {
	case "recordedAt", "effectiveDate":
		if !slices.Contains([]string{"is", "before", "after", "onOrBefore", "onOrAfter"}, q.FilterOperator) {
			return logInvalid("日志日期条件无效")
		}
		if !foundation.ValidDate(value) {
			return logInvalid("日志筛选日期无效")
		}
		if q.FilterField == "effectiveDate" && !has(fields, "effectiveDate") {
			return foundation.Fail(403, "M03_FILTER_FORBIDDEN", "无权筛选生效日期")
		}
	case "actor":
		if !slices.Contains([]string{"contains", "equals"}, q.FilterOperator) {
			return logInvalid("操作者条件无效")
		}
	case "action", "changedField":
		if !slices.Contains([]string{"is", "isAnyOf"}, q.FilterOperator) {
			return logInvalid("日志枚举条件无效")
		}
		values := strings.Split(value, ",")
		seen := map[string]bool{}
		if q.FilterOperator == "is" && len(values) != 1 {
			return logInvalid("日志条件需要单个值")
		}
		for _, v := range values {
			if seen[v] {
				return logInvalid("日志条件包含重复值")
			}
			seen[v] = true
			if q.FilterField == "action" {
				if !slices.Contains(actions, v) {
					return logInvalid("记录处理无效")
				}
				continue
			}
			if !slices.Contains(filterFields, v) {
				return logInvalid("变更字段无效")
			}
			if !logFieldAllowed(v, fields) {
				return foundation.Fail(403, "M03_FILTER_FORBIDDEN", "无权筛选变更字段")
			}
		}
	default:
		return logInvalid("日志筛选字段无效")
	}
	return nil
}
func logDateMatches(day string, q s.LogQuery) bool {
	if day == "" {
		return false
	}
	v := strings.TrimSpace(q.FilterValue)
	switch q.FilterOperator {
	case "is":
		return day == v
	case "before":
		return day < v
	case "after":
		return day > v
	case "onOrBefore":
		return day <= v
	case "onOrAfter":
		return day >= v
	}
	return false
}
func logMatches(g s.ChangeGroup, q s.LogQuery) bool {
	switch q.FilterField {
	case "":
		return true
	case "recordedAt":
		t, e := time.Parse(time.RFC3339Nano, g.RecordedAt)
		return e == nil && logDateMatches(t.In(foundation.Shanghai).Format("2006-01-02"), q)
	case "actor":
		values := []string{}
		if g.Operator.IdentityEvidence == "subject_id_only" {
			values = append(values, g.Operator.SubjectID)
		} else {
			for _, v := range []*string{g.Operator.DisplayName, g.Operator.Login} {
				if v != nil {
					values = append(values, *v)
				}
			}
		}
		for _, v := range values {
			a, b := strings.ToLower(v), strings.ToLower(strings.TrimSpace(q.FilterValue))
			if q.FilterOperator == "equals" && a == b || q.FilterOperator == "contains" && strings.Contains(a, b) {
				return true
			}
		}
	case "action":
		return slices.Contains(strings.Split(strings.TrimSpace(q.FilterValue), ","), g.Action)
	case "effectiveDate":
		for _, item := range g.Items {
			for _, v := range []s.View{item.BeforeContext, item.AfterContext} {
				day, _ := v["effectiveDate"].(string)
				if logDateMatches(day, q) {
					return true
				}
			}
		}
	case "changedField":
		for _, item := range g.Items {
			for _, diff := range item.Diffs {
				if slices.Contains(strings.Split(strings.TrimSpace(q.FilterValue), ","), diff.Field) {
					return true
				}
			}
		}
	}
	return false
}
func logValue(v s.View, field string) s.ChangeValue {
	if v == nil {
		return s.ChangeValue{Kind: "missing"}
	}
	str := func(key string) string { x, _ := v[key].(string); return x }
	switch field {
	case "gradeIds":
		members, _ := v["gradeMembers"].([]s.GradeMemberView)
		return s.ChangeValue{Kind: "members", Members: members}
	case "period":
		return s.ChangeValue{Kind: "period", Start: str("effectiveDate"), EndInclusive: dayBefore(str("endExclusive"))}
	case "status":
		return s.ChangeValue{Kind: "state", Code: str("state")}
	case "operationType":
		code := str(field)
		label := map[string]string{"CREATE": "新建", "CHANGE": "数据更改", "DEACTIVATE": "停用", "REACTIVATE": "重新启用"}[code]
		return s.ChangeValue{Kind: "classification", Code: code, Label: label}
	case "operationReason":
		return s.ChangeValue{Kind: "classification", Code: str(field), Label: str("reasonLabel")}
	case "jobFunctionId":
		return s.ChangeValue{Kind: "reference", Text: str(field), Label: str("jobFunctionName")}
	case "jobFamilyId":
		return s.ChangeValue{Kind: "reference", Text: str(field), Label: str("jobFamilyName")}
	case "departmentId", "jobId", "locationId", "gradeId":
		if str(field) == "" {
			return s.ChangeValue{Kind: "empty"}
		}
		return s.ChangeValue{Kind: "reference", Text: str(field), Label: str(strings.TrimSuffix(field, "Id") + "Name")}
	}
	if str(field) == "" {
		return s.ChangeValue{Kind: "empty"}
	}
	return s.ChangeValue{Kind: "text", Text: str(field)}
}
func logContext(v s.View) s.View {
	if v == nil {
		return nil
	}
	result := s.View{}
	for _, key := range []string{"effectiveDate", "name", "reasonLabel"} {
		if value, ok := v[key]; ok {
			result[key] = value
		}
	}
	if value, ok := v["operationType"]; ok {
		result["typeLabel"] = map[string]string{"CREATE": "新建", "CHANGE": "数据更改", "DEACTIVATE": "停用", "REACTIVATE": "重新启用"}[value.(string)]
	}
	return result
}
func changeGroup(ctx context.Context, u Unit, source Source, before d.Timeline, fields []string) (s.ChangeGroup, error) {
	t := source.Timeline
	op := source.Operator
	g := s.ChangeGroup{SourceVersion: t.Source, RecordedAt: source.RecordedAt.UTC().Format(time.RFC3339Nano), Action: source.Command.Input.Action, Operator: s.ChangeOperator{SubjectID: op.SubjectID, DisplayName: op.DisplayName, Login: op.Login, Kind: op.Kind, IdentityEvidence: op.IdentityEvidence}, Items: []s.ChangeItem{}}
	if before.Source != "" {
		g.PreviousSourceVersion = &before.Source
	}
	ids := []string{}
	for _, timeline := range []d.Timeline{before, t} {
		for _, r := range timeline.Records {
			ids = append(ids, r.ID)
		}
	}
	slices.Sort(ids)
	ids = slices.Compact(ids)
	for _, id := range ids {
		bi, ai := d.Index(before, id), d.Index(t, id)
		if bi >= 0 && ai >= 0 && before.Records[bi].Revision == t.Records[ai].Revision {
			continue
		}
		item := s.ChangeItem{RecordID: id, ChangeKind: "direct", Diffs: []s.ChangeDiff{}}
		var bv, av s.View
		for _, side := range []struct {
			index    int
			timeline d.Timeline
			view     *s.View
			ref      **s.ChangeRef
		}{{bi, before, &bv, &item.BeforeRef}, {ai, t, &av, &item.AfterRef}} {
			if side.index < 0 {
				continue
			}
			r := side.timeline.Records[side.index]
			date := r.Date
			if !r.Active() {
				date = dayBefore(date)
			}
			v, e := recordView(ctx, u, side.timeline, r, fields, date, true)
			if e != nil {
				return g, e
			}
			if has(fields, "code") {
				v["code"] = side.timeline.Code
			}
			*side.view = v
			*side.ref = &s.ChangeRef{SourceVersion: side.timeline.Source, RecordID: r.ID, Revision: r.Revision}
		}
		if bi >= 0 && ai >= 0 {
			b, a := before.Records[bi], t.Records[ai]
			if b.Date == a.Date && b.Type == a.Type && b.Reason == a.Reason && maps.Equal(b.Data, a.Data) && b.Function == a.Function && b.Adoption == a.Adoption && b.Grades.EqualRevision(a.Grades) {
				item.ChangeKind = "derived"
			}
		}
		item.BeforeContext, item.AfterContext = logContext(bv), logContext(av)
		for _, field := range append(d.Fields(t.Kind), "period") {
			if !logFieldAllowed(field, fields) {
				continue
			}
			// Reference identities and labels require the referenced family's current
			// read/history permission too, not just the Job field grant.
			if field == "jobFunctionId" && (bv != nil && bv[field] == nil && bv["jobFunctionName"] == "受限" || av != nil && av[field] == nil && av["jobFunctionName"] == "受限") {
				continue
			}
			if (field == "jobFamilyId" || field == "gradeIds" || slices.Contains(d.PositionFields(), field)) && (bv != nil && bv[field] == nil || av != nil && av[field] == nil) {
				continue
			}
			b, a := logValue(bv, field), logValue(av, field)
			if !reflect.DeepEqual(b, a) {
				item.Diffs = append(item.Diffs, s.ChangeDiff{Field: field, Before: b, After: a})
			}
		}
		g.Items = append(g.Items, item)
	}
	return g, nil
}
func (svc Service) Changes(ctx context.Context, c foundation.Credentials, k s.Kind, id string, q s.LogQuery) (out s.ChangeLog, err error) {
	if !foundation.ValidID(id) {
		return out, logInvalid("资料标识无效")
	}
	ctx = readContext(ctx)
	if q.Page == 0 {
		q.Page = 1
	}
	if q.PageSize == 0 {
		q.PageSize = 20
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
		history, e := decide(ctx, u, d.Kind(k), ent.ID, "history.read", nil)
		if e != nil {
			return e
		}
		fields := intersection(read.ReadFields, history.ReadFields)
		if e = checkLogQuery(q, fields, d.Kind(k)); e != nil {
			return e
		}
		head, e := u.Load(ctx, d.Kind(k), id, "")
		if e != nil {
			return e
		}
		if head.ID == "" {
			return foundation.Fail(404, "M03_OBJECT_NOT_FOUND", "资料不存在或当前不可见")
		}
		sources, e := u.Sources(ctx, d.Kind(k), id)
		if e != nil {
			return e
		}
		out, e = changePage(q, fields, read.ReadFields, len(sources), func(i int) string { return sources[i].Timeline.Source }, func(i int) (s.ChangeGroup, error) {
			before := d.Timeline{}
			if i+1 < len(sources) {
				before = sources[i+1].Timeline
			}
			return changeGroup(ctx, u, sources[i], before, fields)
		})
		return e
	})
	return
}

func changePage(q s.LogQuery, fields, detailFields []string, count int, sourceAt func(int) string, groupAt func(int) (s.ChangeGroup, error)) (s.ChangeLog, error) {
	var out s.ChangeLog
	if count == 0 {
		return out, foundation.Fail(503, "M03_DEPENDENCY_UNAVAILABLE", "日志来源暂不可读")
	}
	find := func(source string) (int, error) {
		for i := 0; i < count; i++ {
			if sourceAt(i) == source {
				return i, nil
			}
		}
		return 0, foundation.Fail(404, "M03_OBJECT_NOT_FOUND", "来源不存在或当前不可见")
	}
	upper := 0
	if q.ContextSourceVersion != "" {
		index, e := find(q.ContextSourceVersion)
		if e != nil {
			return out, e
		}
		upper = index
		out.ContextSourceVersion = &q.ContextSourceVersion
		detailFields = fields
	}
	if q.ThroughSourceVersion != "" {
		index, e := find(q.ThroughSourceVersion)
		if e != nil {
			return out, e
		}
		if index < upper {
			return out, logInvalid("日志上界超出历史上下文")
		}
		upper = index
	}
	groups := []s.ChangeGroup{}
	for i := upper; i < count; i++ {
		g, e := groupAt(i)
		if e != nil {
			return out, e
		}
		if logMatches(g, q) {
			groups = append(groups, g)
		}
	}
	out.Page = min(q.Page, max(1, (len(groups)+q.PageSize-1)/q.PageSize))
	start := (out.Page - 1) * q.PageSize
	out.Items = groups[start:min(start+q.PageSize, len(groups))]
	out.Total = len(groups)
	out.PageSize = q.PageSize
	out.ThroughSourceVersion = sourceAt(upper)
	out.ReadFields = fields
	out.DetailReadFields = detailFields
	return out, nil
}

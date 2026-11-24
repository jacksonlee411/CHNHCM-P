package postgres

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	d "chnhcm.local/chnhcm/internal/contexts/personnel/domain"
	p "chnhcm.local/chnhcm/internal/contexts/personnel/public"
	f "chnhcm.local/chnhcm/internal/foundation"
)

// Read both sides from one PostgreSQL snapshot. The parent link, including
// when two saves share a timestamp, is the authoritative order of a person's history.
const logSnapshots = `with recursive chain(version, depth) as (
 select coalesce(nullif($3,'')::uuid,head),0 from personnel.person
 where instance_id=$1 and person_id=$2 and ordinary_until>$5
 union all
 select s.previous,c.depth+1 from chain c join personnel.source s
 on (s.instance_id,s.person_id,s.version)=($1,$2,c.version)
 where s.previous is not null and (not $4 or c.depth=0)
)
select s.version,coalesce(s.previous::text,''),s.recorded_at,s.operator_name,s.action,s.number,
 n.name,case when $6 then doc.kind end,case when $7 then doc.number end,
 coalesce(items.contacts,'[]'),array(select item from personnel.disposition x
 where (x.instance_id,x.person_id,x.version)=(s.instance_id,s.person_id,s.version) and x.category like 'contact.%')
from chain c join personnel.source s on (s.instance_id,s.person_id,s.version)=($1,$2,c.version)
left join personnel.identity_payload n on (n.instance_id,n.person_id,n.version)=(s.instance_id,s.person_id,s.version)
left join personnel.document_payload doc on (doc.instance_id,doc.person_id,doc.version)=(s.instance_id,s.person_id,s.version) and doc.is_primary
left join lateral (select jsonb_agg(jsonb_build_object('id',item_id,'kind',kind,
 'value',case when ordinary_until>$5 and kind=any($8::text[]) then value end) order by ordinal) contacts
 from personnel.contact_payload i where (i.instance_id,i.person_id,i.version)=(s.instance_id,s.person_id,s.version)) items on true
order by c.depth`

type logContact struct {
	ID    string
	Kind  string
	Value *string
}

type logImage struct {
	p.Source
	previous           string
	number             string
	name               *string
	docKind, docNumber *string
	contacts           []logContact
	disposed           []string
}

var logFields = []string{"name", "number", "document", "phone", "email"}
var logLabels = map[string]string{"name": "姓名", "number": "人员编号", "document": "主要证件", "phone": "电话", "email": "电子邮箱"}

type logAccess struct {
	fields []string
	full   map[string]bool
}

func (r *request) logAccess(id string) (logAccess, error) {
	a := logAccess{full: make(map[string]bool)}
	ok, err := r.readable(id)
	if err != nil {
		return a, err
	}
	if !ok || !r.allowed("history.read", id, "name", "number") {
		return a, d.Missing()
	}
	for _, field := range logFields {
		full := r.allowed("read", id, "name", "number", field) && r.allowed("history.read", id, "name", "number", field)
		a.full[field] = full
		if full || r.allowed("read", id, "name", "number", field+".masked") && r.allowed("history.read", id, "name", "number", field+".masked") {
			a.fields = append(a.fields, field)
		}
	}
	return a, nil
}

func changePaging(q p.ChangeQuery, fields []string) (int, int, error) {
	page, size := 1, 20
	if q.Page != nil {
		page = *q.Page
	}
	if q.PageSize != nil {
		size = *q.PageSize
	}
	if page < 1 || !slices.Contains([]int{20, 50, 100}, size) {
		return 0, 0, d.Invalid("page", "日志分页条件无效")
	}
	if q.Through != "" && !f.ValidID(q.Through) {
		return 0, 0, d.Missing()
	}
	if q.FilterField == "" && q.FilterOperator == "" && q.FilterValue == "" {
		return page, size, nil
	}
	if strings.TrimSpace(q.FilterValue) == "" || utf8.RuneCountInString(q.FilterValue) > 200 {
		return 0, 0, d.Invalid("filterValue", "筛选值无效")
	}
	switch q.FilterField {
	case "action", "changedField":
		if q.FilterOperator != "is" && q.FilterOperator != "isAnyOf" {
			break
		}
		values := strings.Split(q.FilterValue, ",")
		if q.FilterOperator == "is" && len(values) != 1 {
			break
		}
		allowed := fields
		if q.FilterField == "action" {
			allowed = []string{"register", "correct", "renumber"}
		}
		for _, v := range values {
			if !slices.Contains(allowed, v) {
				return 0, 0, d.Invalid("filterValue", "筛选值不可用")
			}
		}
		return page, size, nil
	case "actor":
		if q.FilterOperator == "contains" || q.FilterOperator == "equals" {
			return page, size, nil
		}
	case "recordedAt":
		if _, err := time.Parse(time.DateOnly, q.FilterValue); err != nil {
			return 0, 0, d.Invalid("filterValue", "请输入有效日期")
		}
		if slices.Contains([]string{"is", "before", "after", "onOrBefore", "onOrAfter"}, q.FilterOperator) {
			return page, size, nil
		}
	}
	return 0, 0, d.Invalid("filterField", "日志筛选条件无效")
}
func logMatches(q p.ChangeQuery, g p.ChangeGroup) bool {
	switch q.FilterField {
	case "recordedAt":
		date := g.At.In(f.Shanghai).Format(time.DateOnly)
		switch q.FilterOperator {
		case "before":
			return date < q.FilterValue
		case "onOrBefore":
			return date <= q.FilterValue
		case "after":
			return date > q.FilterValue
		case "onOrAfter":
			return date >= q.FilterValue
		default:
			return date == q.FilterValue
		}
	case "actor":
		v := strings.ToLower(strings.TrimSpace(q.FilterValue))
		actor := strings.ToLower(g.Operator)
		if q.FilterOperator == "equals" {
			return actor == v
		}
		return strings.Contains(actor, v)
	case "action":
		return slices.Contains(strings.Split(q.FilterValue, ","), g.Action)
	case "changedField":
		values := strings.Split(q.FilterValue, ",")
		for _, diff := range g.Diffs {
			if slices.Contains(values, diff.FieldKey) {
				return true
			}
		}
		return false
	default:
		return true
	}
}

func valueOf(v *string) string {
	if v != nil {
		return *v
	}
	return ""
}
func (a logAccess) differences(before, after logImage, reveal bool) []p.Difference {
	diffs := []p.Difference{}
	add := func(field, item, left, right string, canReveal bool) {
		diffs = append(diffs, p.Difference{FieldKey: field, Field: logLabels[field], ItemID: item, Before: left, After: right, CanReveal: canReveal})
	}
	identityAvailable := after.name != nil && (before.Version == "" || before.name != nil)
	if identityAvailable && valueOf(before.name) != valueOf(after.name) {
		add("name", "", valueOf(before.name), valueOf(after.name), false)
	}
	if before.number != after.number {
		add("number", "", before.number, after.number, false)
	}
	if identityAvailable && (valueOf(before.docKind) != valueOf(after.docKind) || valueOf(before.docNumber) != valueOf(after.docNumber)) {
		switch {
		case before.docKind == nil:
			add("document", "", "", "已登记", false)
		case after.docKind == nil:
			add("document", "", "已登记", "已撤回", false)
		default:
			add("document", "", "已登记", "已更换", false)
		}
	}
	old := make(map[string]logContact, len(before.contacts))
	for _, x := range before.contacts {
		old[x.ID] = x
	}
	contact := func(kind, id, left, right string) {
		if left == right {
			return
		}
		full := a.full[kind]
		ml, mr := d.Mask(kind, left), d.Mask(kind, right)
		if !full && ml == mr {
			return
		}
		if !full || !reveal {
			left, right = ml, mr
		}
		add(kind, id, left, right, full)
	}
	for _, cur := range after.contacts {
		prev, had := old[cur.ID]
		delete(old, cur.ID)
		if cur.Value == nil || had && prev.Value == nil || slices.Contains(before.disposed, cur.ID) {
			continue
		}
		var left string
		if had {
			left = *prev.Value
			if prev.Kind != cur.Kind {
				contact(prev.Kind, cur.ID, left, "")
				left = ""
			}
		}
		contact(cur.Kind, cur.ID, left, *cur.Value)
	}
	for _, x := range before.contacts {
		if _, ok := old[x.ID]; ok && x.Value != nil && !slices.Contains(after.disposed, x.ID) {
			contact(x.Kind, x.ID, *x.Value, "")
		}
	}
	return diffs
}

// A page retains only the requested groups, or the last legal page when a filter
// or grant change shrinks the result. Sensitive values are never cached.
func (r *request) changePage(id string, q p.ChangeQuery, exact bool) (p.ChangePage, error) {
	out := p.ChangePage{Items: []p.ChangeGroup{}}
	a, err := r.logAccess(id)
	if err != nil {
		return out, err
	}
	page, size, err := changePaging(q, a.fields)
	if err != nil {
		return out, err
	}
	out.Page, out.PageSize, out.ReadFields = 1, size, a.fields
	rows, err := r.tx.Query(r.ctx, logSnapshots, r.instance, id, q.Through, exact, r.a.At, slices.Contains(a.fields, "document"), a.full["document"], a.fields)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	var newer *logImage
	accept := func(before logImage) {
		group := p.ChangeGroup{Source: newer.Source, Previous: newer.previous, Diffs: a.differences(before, *newer, exact)}
		if !logMatches(q, group) {
			return
		}
		current := out.Total/size + 1
		if current <= page {
			if current != out.Page {
				out.Items = out.Items[:0]
				out.Page = current
			}
			out.Items = append(out.Items, group)
		}
		out.Total++
	}
	for rows.Next() {
		var img logImage
		var contacts []byte
		if err = rows.Scan(&img.Version, &img.previous, &img.At, &img.Operator, &img.Action, &img.number, &img.name, &img.docKind, &img.docNumber, &contacts, &img.disposed); err != nil {
			return out, err
		}
		if err = json.Unmarshal(contacts, &img.contacts); err != nil {
			return out, err
		}
		if newer != nil {
			accept(img)
		} else {
			out.Through = img.Version
		}
		newer = &img
	}
	if err = rows.Err(); err != nil {
		return out, err
	}
	if newer == nil {
		return out, d.Missing()
	}
	if newer.previous == "" && (!exact || newer.Version == out.Through) {
		accept(logImage{})
	}
	return out, nil
}
func (s *Store) Changes(ctx context.Context, c f.Credentials, id string, q p.ChangeQuery) (p.ChangePage, error) {
	return run(ctx, s, c, func(r *request) (p.ChangePage, error) { return r.changePage(id, q, false) })
}
func (s *Store) Differences(ctx context.Context, c f.Credentials, id, version string) ([]p.Difference, error) {
	return run(ctx, s, c, func(r *request) ([]p.Difference, error) {
		page, err := r.changePage(id, p.ChangeQuery{Through: version}, true)
		if err != nil {
			return nil, err
		}
		return page.Items[0].Diffs, nil
	})
}

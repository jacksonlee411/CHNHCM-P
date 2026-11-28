package postgres

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
)

func window(q access.ManagementQuery, total int) (int, int, int) {
	size := q.PageSize
	if size != 50 && size != 100 {
		size = 20
	}
	page := min(max(0, q.Page), max(total-1, 0)/size)
	return page * size, min((page+1)*size, total), page
}

func (m *manager) mandateState(d access.Mandate) string {
	r, s, f := m.binding(d.Limit.Role, d.Limit.Scope, d.Limit.Fields)
	switch {
	case d.Revoked:
		return "revoked"
	case d.Limit.Until != nil && !time.Now().Before(*d.Limit.Until):
		return "expired"
	case r.Disabled || s.Disabled || f.Disabled:
		return "disabled"
	default:
		return "active"
	}
}

// Count only this page's account IDs and evaluate each complete grant with the same ceiling as the detail.
func (m *manager) countVisibleGrants(accounts []access.ManagedAccount) error {
	byID := make(map[string]*access.ManagedAccount)
	ids := []string{}
	for i := range accounts {
		a := &accounts[i]
		if m.ownerScope("assign", a.Owner) {
			a.ActiveGrants = new(int)
			byID[a.ID] = a
			ids = append(ids, a.ID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	grants, err := m.grantRows(ids, "", "")
	if err != nil {
		return err
	}
	for _, g := range grants {
		a := byID[g.Subject]
		if g.State == "active" && m.allowedTerm("assign", m.terms(g, a.Owner)) {
			*a.ActiveGrants += 1
		}
	}
	return nil
}

func (m *manager) targetLabel(id string) string {
	for _, t := range m.targets {
		if t.ID == id {
			return strings.TrimSpace(strings.TrimSpace(t.Code) + " " + t.Name)
		}
	}
	if id != "" {
		return "原范围 " + id
	}
	return "原范围未登记"
}
func (m *manager) bindingLabel(role, scope, fields string) string {
	r, s, f := m.binding(role, scope, fields)
	return fmt.Sprintf("%s · V%d / %s · V%d（%s） / %s · V%d", r.Name, r.Number, s.Name, s.Number, m.targetLabel(s.TargetID), f.Name, f.Number)
}
func grantSource(source string) string {
	switch source {
	case "duty":
		return "本职"
	case "proxy":
		return "限期代理"
	case "migration":
		return "存量迁入"
	case "development":
		return "开发样本"
	}
	return source
}
func period(from, until *time.Time) string {
	format := func(t *time.Time, fallback string) string {
		if t == nil {
			return fallback
		}
		return t.In(time.FixedZone("北京时间", 8*60*60)).Format("2006-01-02 15:04:05")
	}
	return format(from, "原始起点未登记") + " 至 " + format(until, "长期")
}
func (m *manager) grantLabel(g access.ManagedGrant) string {
	return m.bindingLabel(g.Role, g.Scope, g.Fields) + " · " + grantSource(g.Source) + " · " + period(g.From, g.Until)
}

func (m *manager) mandateLabel(l access.ManagementLimit) string {
	names := map[string]string{"account": "账号管理", "credential": "凭据重置", "role": "角色发布", "policy": "策略管理", "assign": "业务授权", "delegate": "管理职责委派"}
	rights := make([]string, 0, len(l.Rights))
	for _, r := range l.Rights {
		rights = append(rights, names[r])
	}
	days := "无限制"
	if l.MaxProxyDays > 0 {
		days = fmt.Sprintf("%d 天", l.MaxProxyDays)
	}
	until := "长期"
	if l.Until != nil {
		until = l.Until.In(time.FixedZone("北京时间", 8*60*60)).Format("2006-01-02 15:04:05")
	}
	return strings.Join(rights, "、") + " · " + m.targetLabel(l.Owner) + " · " + m.bindingLabel(l.Role, l.Scope, l.Fields) + " · 代理期限上限 " + days + " · 职责到期 " + until
}

// Return one usable changed binding. The command rechecks the complete selection and original definitions.
func (m *manager) adoption(g access.ManagedGrant, owner string) *access.GrantAdoption {
	candidates := func(kind, id string) []string {
		original := m.versions[definitionKey{kind, id}]
		var versions []access.Definition
		for _, d := range m.versions {
			if d.Kind == kind && d.DefinitionID == original.DefinitionID && !d.Disabled && m.definitionVisible(d, "assign") {
				versions = append(versions, d)
			}
		}
		sort.Slice(versions, func(i, j int) bool { return versions[i].Number > versions[j].Number })
		ids := make([]string, 0, len(versions))
		for _, d := range versions {
			ids = append(ids, d.ID)
		}
		return ids
	}
	roles, scopes, fields := candidates("role", g.Role), candidates("scope", g.Scope), candidates("fields", g.Fields)
	for _, r := range roles {
		for _, s := range scopes {
			for _, f := range fields {
				if r == g.Role && s == g.Scope && f == g.Fields {
					continue
				}
				next := g
				next.Role, next.Scope, next.Fields = r, s, f
				def := m.versions[definitionKey{"role", r}]
				if slices.ContainsFunc(def.Actions, func(a string) bool { return purposeFor(a) == g.Purpose }) && m.grantBinding(next, owner, "assign") == nil {
					return &access.GrantAdoption{Role: r, Scope: s, Fields: f}
				}
			}
		}
	}
	return nil
}

func (m *manager) grantWorkspace(q access.ManagementQuery, a access.ManagedAccount, v *access.ManagementView) error {
	canAssign, canDelegate := m.ownerScope("assign", a.Owner), m.ownerScope("delegate", a.Owner)
	if q.Choices && canAssign {
		v.ProxyDays = m.proxyDays
		if !m.root {
			maxDays := 0
			for _, c := range m.caps {
				if c.Owner == a.Owner && slices.Contains(c.Rights, "assign") {
					if c.MaxProxyDays == 0 {
						maxDays = 0
						break
					}
					maxDays = max(maxDays, c.MaxProxyDays)
				}
			}
			if maxDays > 0 && (v.ProxyDays == 0 || maxDays < v.ProxyDays) {
				v.ProxyDays = maxDays
			}
		}
	}
	if canAssign {
		v.Panels = append(v.Panels, "grants")
	}
	if canDelegate {
		v.Panels = append(v.Panels, "mandates")
	}
	v.Panels = append(v.Panels, "events")
	v.Panel = q.Panel
	if v.Panel == "" {
		v.Panel = v.Panels[0]
	}
	if !slices.Contains(v.Panels, v.Panel) {
		return managementDenied()
	}
	if !q.Choices {
		v.Versions = nil
	}
	needed := map[definitionKey]bool{}
	addBinding := func(r, s, f string) {
		needed[definitionKey{"role", r}] = true
		needed[definitionKey{"scope", s}] = true
		needed[definitionKey{"fields", f}] = true
	}
	grants := []access.ManagedGrant{}
	mandates := []access.Mandate{}
	if canAssign && v.Panel != "mandates" {
		all, err := m.grants(a.ID, "", "")
		if err != nil {
			return err
		}
		for _, g := range all {
			if !m.allowedTerm("assign", m.terms(g, a.Owner)) {
				continue
			}
			if q.Filters {
				addBinding(g.Role, g.Scope, g.Fields)
				continue
			}
			if v.Panel == "grants" {
				if q.Source != "" && q.Source != g.Source || q.State != "" && q.State != g.State || q.Version != "" && q.Version != g.Role && q.Version != g.Scope && q.Version != g.Fields {
					continue
				}
				if q.Search != "" && !strings.Contains(strings.ToLower(m.grantLabel(g)+" "+g.ID), strings.ToLower(q.Search)) {
					continue
				}
			}
			grants = append(grants, g)
		}
	}
	if q.Filters {
		for k := range needed {
			d := m.versions[k]
			v.Versions = append(v.Versions, access.Definition{ID: d.ID, Kind: d.Kind, Number: d.Number, Name: d.Name})
		}
		v.Owners, v.Targets, v.Actions = nil, nil, nil
		return nil
	}
	if canDelegate && v.Panel != "grants" {
		all, err := m.mandates(a.ID)
		if err != nil {
			return err
		}
		for _, d := range all {
			if m.canDelegate(d.Limit, a.Owner) {
				d.CanRevoke = !d.Revoked && slices.Contains(v.Accounts[0].Options, "delegate")
				mandates = append(mandates, d)
			}
		}
	}
	switch v.Panel {
	case "events":
		events, err := m.events(q, access.ManagementView{Grants: grants, Mandates: mandates})
		if err != nil {
			return err
		}
		if q.Record != "" {
			events = slices.DeleteFunc(events, func(e access.ManagementEvent) bool { return e.Target != q.Record })
		}
		start, end, page := window(q, len(events))
		v.Total, v.Page, v.Events = len(events), page, events[start:end]
		if err := m.describeEvents(v.Events, grants, mandates, a.Owner); err != nil {
			return err
		}
	case "mandates":
		q = locate(q, mandates, func(d access.Mandate) string { return d.ID })
		start, end, page := window(q, len(mandates))
		v.Total, v.Page, v.Mandates = len(mandates), page, mandates[start:end]
		for _, d := range v.Mandates {
			l := d.Limit
			addBinding(l.Role, l.Scope, l.Fields)
		}
	case "grants":
		labels := map[string]int{}
		for _, g := range grants {
			labels[m.grantLabel(g)+g.State]++
		}
		for i := range grants {
			g := &grants[i]
			g.Distinguish = labels[m.grantLabel(*g)+g.State] > 1
		}
		rank := map[string]int{"active": 0, "future": 1, "expired": 2, "revoked": 3, "disabled": 4}
		sort.SliceStable(grants, func(i, j int) bool { return rank[grants[i].State] < rank[grants[j].State] })
		q = locate(q, grants, func(g access.ManagedGrant) string { return g.ID })
		start, end, page := window(q, len(grants))
		v.Total, v.Page, v.Grants = len(grants), page, grants[start:end]
		for i := range v.Grants {
			g := &v.Grants[i]
			addBinding(g.Role, g.Scope, g.Fields)
			if slices.Contains(v.Accounts[0].Options, "assign") && g.State != "expired" && g.State != "revoked" {
				g.CanRevoke = true
				g.Adoption = m.adoption(*g, a.Owner)
			}
		}
	}
	for k := range needed {
		if !slices.ContainsFunc(v.Versions, func(d access.Definition) bool { return d.ID == k.id && d.Kind == k.kind }) {
			v.Versions = append(v.Versions, m.versions[k])
		}
	}
	if !q.Choices {
		used := map[string]bool{}
		for _, d := range v.Versions {
			if d.TargetID != "" {
				used[d.TargetID] = true
			}
		}
		for _, d := range v.Mandates {
			used[d.Limit.Owner] = true
		}
		v.Targets = slices.DeleteFunc(v.Targets, func(t access.Target) bool { return !used[t.ID] })
		v.Owners = slices.DeleteFunc(v.Owners, func(t access.Target) bool { return t.ID != a.Owner && !used[t.ID] })
	}
	return nil
}
func locate[T any](q access.ManagementQuery, rows []T, id func(T) string) access.ManagementQuery {
	if q.Record != "" {
		if index := slices.IndexFunc(rows, func(r T) bool { return id(r) == q.Record }); index >= 0 {
			size := q.PageSize
			if size != 50 && size != 100 {
				size = 20
			}
			q.Page = index / size
		}
	}
	return q
}

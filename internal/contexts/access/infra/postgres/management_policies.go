package postgres

import (
	app "chnhcm.local/chnhcm/internal/contexts/access/application"
	"slices"
	"strings"
	"time"

	"chnhcm.local/chnhcm/internal/contexts/access/domain"
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

func policyRight(kind string) string {
	if kind == "role" {
		return "role"
	}
	return "policy"
}
func (m *manager) publish(d access.Definition, request string) (string, error) {
	if !slices.Contains([]string{"role", "scope", "fields"}, d.Kind) || strings.TrimSpace(d.Name) == "" || len(d.Name) > 180 {
		return "", invalidManagement("请输入版本名称并选择版本类型")
	}
	right := policyRight(d.Kind)
	if !m.permits(right, "", nil) {
		return "", managementDenied()
	}
	if d.DefinitionID != "" {
		previous := access.Definition{}
		for _, v := range m.versions {
			if v.Kind == d.Kind && v.DefinitionID == d.DefinitionID && v.Number > previous.Number {
				previous = v
			}
		}
		if !m.definitionVisible(previous, right) {
			return "", managementDenied()
		}
		if d.Number != previous.Number {
			return "", changedManagement()
		}
	} else if d.Number != 0 {
		return "", changedManagement()
	}
	id := foundation.NewID()
	if d.DefinitionID == "" {
		d.DefinitionID = id
	}
	d.ID = id
	d.Number++
	d.Disabled = false
	key := definitionKey{d.Kind, id}
	v := access.PolicyVersion{Kind: d.Kind, ID: id, DefinitionID: d.DefinitionID, Number: d.Number}
	switch d.Kind {
	case "scope":
		if !m.hasTarget(d.TargetKind, d.TargetID) || d.TargetKind == "instance" {
			return "", managementDenied()
		}
		v.ScopeKind, v.ScopeID = d.TargetKind, d.TargetID
	case "fields":
		allRead := append(slices.Clone(app.DepartmentFields), app.QuotaFields...)
		allWrite := append(slices.Clone(app.DepartmentWrite), app.QuotaWrite...)
		if !domain.ContainsAll(allRead, d.Read) || !domain.ContainsAll(allWrite, d.Write) {
			return "", invalidManagement("所选字段未开放委派")
		}
		v.ReadFields = append([]string{}, d.Read...)
		v.WriteFields = append([]string{}, d.Write...)
	case "role":
		_, scope, fields := m.binding(id, d.Scope, d.Fields)
		if !m.definitionVisible(scope, right) || !m.definitionVisible(fields, right) || scope.Disabled || fields.Disabled {
			return "", managementDenied()
		}
		if !app.ValidPublication(d, scope, fields) {
			return "", invalidManagement("动作、目标和字段不适用于同一职责")
		}
		v.Actions = d.Actions
	}
	m.published[key] = true
	if !m.definitionVisible(d, right) {
		return "", managementDenied()
	}
	if err := PublishVersion(m.ctx, m.tx, m.instance, m.actor, request, v); err != nil {
		return "", err
	}
	_, err := m.tx.Exec(m.ctx, `INSERT INTO access.management_publication(instance_id,kind,version_id,name,scope_id,fields_id) VALUES($1,$2,$3,$4,$5,$6)`, m.instance, d.Kind, id, d.Name, nullable(d.Scope), nullable(d.Fields))
	if err != nil {
		return "", err
	}
	m.versions[key] = d
	return id, nil
}
func (m *manager) changeVersion(in access.ManagementCommand) (string, error) {
	if in.Version == nil {
		return "", invalidManagement("请选择角色或策略")
	}
	d := *in.Version
	switch in.Operation {
	case "version.publish":
		if in.NewScope != nil || in.NewFields != nil {
			if d.Kind != "role" {
				return "", invalidManagement("仅角色可同时发布配套策略")
			}
			for _, p := range []struct {
				value *access.Definition
				kind  string
				ref   *string
			}{{in.NewScope, "scope", &d.Scope}, {in.NewFields, "fields", &d.Fields}} {
				if p.value == nil {
					continue
				}
				if p.value.Kind != p.kind {
					return "", invalidManagement("策略类型无效")
				}
				id, err := m.publish(*p.value, in.Request)
				if err != nil {
					return "", err
				}
				*p.ref = id
			}
		}
		return m.publish(d, in.Request)
	case "version.disable":
		v := m.versions[definitionKey{d.Kind, in.ID}]
		if !m.definitionVisible(v, policyRight(v.Kind)) {
			return "", managementDenied()
		}
		if v.Disabled || in.Revision != v.Number {
			return "", changedManagement()
		}
		impact, _, _, e := m.versionImpact(v, access.ManagementQuery{State: "impact"})
		if e != nil {
			return "", e
		}
		if impact.Token == "" {
			return "", managementDenied()
		}
		if in.ImpactToken != impact.Token {
			return "", foundation.Fail(409, "ACCESS_CONFLICT", "版本引用已变化，请重新核对停用影响")
		}
		return v.ID, DisableVersion(m.ctx, m.tx, m.instance, m.actor, in.Request, v.Kind, v.ID)
	}
	return "", invalidManagement("未知版本操作")
}

func (m *manager) grantBinding(g access.ManagedGrant, owner, right string) error {
	r, s, f := m.binding(g.Role, g.Scope, g.Fields)
	if !m.definitionVisible(r, right) || !m.definitionVisible(s, right) || !m.definitionVisible(f, right) || r.Disabled || s.Disabled || f.Disabled || !m.hasTarget(s.TargetKind, s.TargetID) {
		return managementDenied()
	}
	if !app.ValidPublication(r, s, f) {
		return invalidManagement("所选版本不能组合授予")
	}
	if !app.ProxyWithin(m.proxyDays, g.Source, g.From, g.Until) || !m.allowedTerm(right, m.terms(g, owner)) {
		return managementDenied()
	}
	return nil
}
func (m *manager) changeGrant(in access.ManagementCommand) (string, error) {
	if !foundation.ValidID(in.Subject) {
		return "", managementDenied()
	}
	a, e := m.account(in.Subject)
	if e != nil {
		return "", e
	}
	if a.ID == m.actor || a.ID == m.protected {
		return "", managementDenied()
	}
	if strings.HasPrefix(in.Operation, "mandate.") {
		return m.changeMandate(in, a)
	}
	if !m.ownerScope("assign", a.Owner) {
		return "", managementDenied()
	}
	op := strings.TrimPrefix(in.Operation, "grant.")
	g := access.ManagedGrant{ID: foundation.NewID(), Subject: a.ID, Role: in.Role, Scope: in.Scope, Fields: in.Fields, From: in.From, Until: in.Until, Source: in.Source, Purpose: in.Purpose}
	now := time.Now().UTC().Truncate(time.Microsecond)
	switch op {
	case "issue":
		if !slices.Contains([]string{"duty", "proxy"}, g.Source) || in.Revision != 0 {
			return "", invalidManagement("请选择本职或限期代理")
		}
		if g.From == nil {
			g.From = &now
		}
		if g.Until != nil && (!g.From.Before(*g.Until) || !now.Before(*g.Until)) || g.Source == "proxy" && g.Until == nil {
			return "", invalidManagement("到期时刻须晚于开始时刻和当前时刻")
		}
	case "adopt", "revoke":
		gs, e := m.grants(a.ID, "", "")
		if e != nil {
			return "", e
		}
		i := slices.IndexFunc(gs, func(v access.ManagedGrant) bool { return v.ID == in.ID })
		if i < 0 || !m.allowedTerm("assign", m.terms(gs[i], a.Owner)) {
			return "", managementDenied()
		}
		g = gs[i]
		if g.Revision != in.Revision || g.State == "expired" || g.State == "revoked" {
			return "", changedManagement()
		}
		if op == "adopt" {
			oldR, oldS, oldF := m.binding(g.Role, g.Scope, g.Fields)
			r, s, f := m.binding(in.Role, in.Scope, in.Fields)
			if oldR.DefinitionID != r.DefinitionID || oldS.DefinitionID != s.DefinitionID || oldF.DefinitionID != f.DefinitionID {
				return "", invalidManagement("换版须选择同一职责和策略的版本")
			}
			if g.Role == in.Role && g.Scope == in.Scope && g.Fields == in.Fields {
				return "", changedManagement()
			}
			g.Role, g.Scope, g.Fields = in.Role, in.Scope, in.Fields
		}
	default:
		return "", invalidManagement("未知授权操作")
	}
	b := access.VersionBinding{}
	if op != "revoke" {
		r, _, _ := m.binding(g.Role, g.Scope, g.Fields)
		purposes := []string{}
		for _, action := range r.Actions {
			p := purposeFor(action)
			if !slices.Contains(purposes, p) {
				purposes = append(purposes, p)
			}
		}
		if len(purposes) == 1 && g.Purpose == "" {
			g.Purpose = purposes[0]
		}
		if !slices.Contains(purposes, g.Purpose) {
			return "", invalidManagement("请选择职责可用的用途")
		}
		if e = m.grantBinding(g, a.Owner, "assign"); e != nil {
			return "", e
		}
		b = access.VersionBinding{Role: g.Role, Scope: g.Scope, Fields: g.Fields}
	}
	change := access.AssignmentChange{Actor: m.actor, RequestID: in.Request, Operation: op, Subject: a.ID, ID: g.ID, ExpectedRevision: in.Revision, Binding: b}
	if op == "issue" {
		change.Source, change.ValidFrom, change.ValidUntil, change.Purpose = g.Source, g.From, g.Until, g.Purpose
	}
	_, e = ChangeAssignment(m.ctx, m.tx, m.instance, change)
	return g.ID, e
}
func (m *manager) changeMandate(in access.ManagementCommand, a access.ManagedAccount) (string, error) {
	if !m.ownerScope("delegate", a.Owner) {
		return "", managementDenied()
	}
	var id string
	switch in.Operation {
	case "mandate.issue":
		if in.Limit == nil {
			return "", invalidManagement("请填写管理职责")
		}
		l := *in.Limit
		if len(l.Rights) == 0 || !domain.ContainsAll(app.ManagementRights, l.Rights) || l.MaxProxyDays < 0 || l.MaxProxyDays > 3650 || l.Until != nil && !time.Now().Before(*l.Until) {
			return "", invalidManagement("请核对委派职责和期限")
		}
		if !m.ownerAllowed("delegate", l.Owner) || !m.canDelegate(l, a.Owner) || m.proxyDays > 0 && (l.MaxProxyDays == 0 || l.MaxProxyDays > m.proxyDays) {
			return "", managementDenied()
		}
		g := access.ManagedGrant{Role: l.Role, Scope: l.Scope, Fields: l.Fields, Until: l.Until}
		if e := m.grantBinding(g, l.Owner, "delegate"); e != nil {
			return "", e
		}
		id = foundation.NewID()
		_, e := m.tx.Exec(m.ctx, `INSERT INTO access.management_mandate(instance_id,id,principal_id,owner_id,rights,role_id,scope_id,fields_id,max_proxy_days,valid_until) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, m.instance, id, a.ID, l.Owner, l.Rights, l.Role, l.Scope, l.Fields, l.MaxProxyDays, l.Until)
		if e != nil {
			return "", e
		}
	case "mandate.revoke":
		items, e := m.mandates(a.ID)
		if e != nil {
			return "", e
		}
		i := slices.IndexFunc(items, func(v access.Mandate) bool { return v.ID == in.ID })
		if i < 0 || !m.canDelegate(items[i].Limit, a.Owner) {
			return "", managementDenied()
		}
		if items[i].Revoked {
			return "", changedManagement()
		}
		id = in.ID
		if _, e = m.tx.Exec(m.ctx, `UPDATE access.management_mandate SET revoked_at=clock_timestamp() WHERE instance_id=$1 AND id=$2`, m.instance, id); e != nil {
			return "", e
		}
	default:
		return "", invalidManagement("未知管理职责操作")
	}
	if e := bumpSubject(m.ctx, m.tx, m.instance, a.ID); e != nil {
		return "", e
	}
	return id, recordManagement(m.ctx, m.tx, m.instance, m.actor, a.ID, in.Request, in.Operation, managementChangeReason)
}

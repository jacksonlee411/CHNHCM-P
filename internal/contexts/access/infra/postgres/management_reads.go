package postgres

import (
	app "chnhcm.local/chnhcm/internal/contexts/access/application"
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"chnhcm.local/chnhcm/internal/contexts/access/domain"
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
)

var areaRights = map[string][]string{"accounts": {"account", "credential"}, "roles": {"role", "policy"}, "grants": {"assign", "delegate"}}

func (m *manager) areaOK(area string) bool {
	return slices.ContainsFunc(areaRights[area], func(r string) bool { return m.permits(r, "", nil) })
}

const accountsSQL = `SELECT p.principal_id::text,p.login,coalesce(p.display_name,''),coalesce(p.management_owner::text,''),p.state,g.policy_no
 FROM access.principal p JOIN access.policy_guard g USING(instance_id,principal_id) WHERE p.instance_id=$1 `

func accountScan(r pgx.Row) (access.ManagedAccount, error) {
	var a access.ManagedAccount
	e := r.Scan(&a.ID, &a.Login, &a.Display, &a.Owner, &a.State, &a.Revision)
	return a, e
}
func (m *manager) account(id string) (access.ManagedAccount, error) {
	a, e := accountScan(m.tx.QueryRow(m.ctx, accountsSQL+"AND p.principal_id=$2", m.instance, id))
	if errors.Is(e, pgx.ErrNoRows) {
		return a, managementDenied()
	}
	return a, e
}
func (m *manager) ownerScope(right, owner string) bool {
	return m.root || owner != "" && m.permits(right, owner, nil)
}
func (m *manager) accounts(q access.ManagementQuery) ([]access.ManagedAccount, error) {
	var owners []string
	if !m.root {
		owners = []string{}
		for _, c := range m.caps {
			if slices.ContainsFunc(areaRights[q.Area], func(r string) bool { return m.permits(r, c.Owner, nil) }) {
				owners = append(owners, c.Owner)
			}
		}
	}
	rows, e := m.tx.Query(m.ctx, accountsSQL+`AND ($2::uuid[] IS NULL OR p.management_owner=ANY($2)) AND ($3='' OR strpos(p.normalized_login,lower($3))>0 OR strpos(lower(p.display_name),lower($3))>0)
 AND ($4='' OR p.management_owner::text=$4) AND ($5='' OR p.state=$5) AND ($6='' OR p.principal_id::text=$6)
 ORDER BY p.normalized_login,p.principal_id`, m.instance, owners, q.Search, q.Owner, q.State, q.Subject)
	if e != nil {
		return nil, e
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (access.ManagedAccount, error) { return accountScan(row) })
}
func (m *manager) grants(subject, kind, version string) ([]access.ManagedGrant, error) {
	var subjects []string
	if subject != "" {
		subjects = []string{subject}
	}
	return m.grantRows(subjects, kind, version)
}
func (m *manager) grantRows(subjects []string, kind, version string) ([]access.ManagedGrant, error) {
	rows, e := m.tx.Query(m.ctx, `SELECT a.grant_id::text,a.principal_id::text,b.role_id::text,b.scope_id::text,b.fields_id::text,a.revision,a.source,a.purpose,a.valid_from,a.valid_until,b.operation
 FROM access.grant_assignment a JOIN access.grant_revision b USING(instance_id,grant_id,revision)
 WHERE a.instance_id=$1 AND ($2::uuid[] IS NULL OR a.principal_id=ANY($2)) AND ($3='' OR (CASE $3 WHEN 'role' THEN b.role_id WHEN 'scope' THEN b.scope_id WHEN 'fields' THEN b.fields_id END)=$4::uuid)
 ORDER BY a.grant_id`, m.instance, subjects, kind, nullable(version))
	if e != nil {
		return nil, e
	}
	now := time.Now()
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (access.ManagedGrant, error) {
		var g access.ManagedGrant
		var op string
		e := row.Scan(&g.ID, &g.Subject, &g.Role, &g.Scope, &g.Fields, &g.Revision, &g.Source, &g.Purpose, &g.From, &g.Until, &op)
		r, s, f := m.binding(g.Role, g.Scope, g.Fields)
		switch {
		case op == "revoke":
			g.State = "revoked"
		case g.Until != nil && !now.Before(*g.Until):
			g.State = "expired"
		case r.Disabled || s.Disabled || f.Disabled:
			g.State = "disabled"
		case g.From != nil && now.Before(*g.From):
			g.State = "future"
		default:
			g.State = "active"
		}
		return g, e
	})
}
func (m *manager) mandates(subject string) ([]access.Mandate, error) {
	rows, e := m.tx.Query(m.ctx, `SELECT m.id::text,m.owner_id::text,m.rights,m.role_id::text,m.scope_id::text,m.fields_id::text,m.max_proxy_days,m.valid_until,m.revoked_at IS NOT NULL
 FROM access.management_mandate m WHERE m.instance_id=$1 AND m.principal_id=$2 ORDER BY m.id`, m.instance, subject)
	if e != nil {
		return nil, e
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (access.Mandate, error) {
		var v access.Mandate
		l := &v.Limit
		e := r.Scan(&v.ID, &l.Owner, &l.Rights, &l.Role, &l.Scope, &l.Fields, &l.MaxProxyDays, &l.Until, &v.Revoked)
		v.State = m.mandateState(v)
		return v, e
	})
}
func (m *manager) canDelegate(l access.ManagementLimit, accountOwner string) bool {
	if m.root {
		return true
	}
	if accountOwner != l.Owner {
		return false
	}
	t := m.terms(access.ManagedGrant{Role: l.Role, Scope: l.Scope, Fields: l.Fields, Until: l.Until}, l.Owner)
	return m.permits("delegate", accountOwner, func(c app.ManagementCeiling) bool {
		return c.Covers(t) && domain.ContainsAll(c.Rights, l.Rights) && (c.MaxProxyDays == 0 || l.MaxProxyDays > 0 && l.MaxProxyDays <= c.MaxProxyDays)
	})
}
func (m *manager) controls(a access.ManagedAccount, right string) (bool, error) {
	if a.ID == m.actor || m.isAdministrator(a.ID) || !m.ownerScope(right, a.Owner) {
		return false, nil
	}
	if m.root {
		return true, nil
	}
	var outside bool
	if e := m.tx.QueryRow(m.ctx, `SELECT EXISTS(SELECT FROM access.person_grant WHERE instance_id=$1 AND principal_id=$2 AND NOT denied AND (expires_at IS NULL OR expires_at>clock_timestamp())
 UNION ALL SELECT FROM access.employment_grant WHERE instance_id=$1 AND principal_id=$2 AND (expires_at IS NULL OR expires_at>clock_timestamp())
 UNION ALL SELECT FROM access.offer_grant WHERE instance_id=$1 AND principal_id=$2 AND (expires_at IS NULL OR expires_at>clock_timestamp()))`, m.instance, a.ID).Scan(&outside); e != nil || outside {
		return false, e
	}
	gs, e := m.grants(a.ID, "", "")
	if e != nil {
		return false, e
	}
	for _, g := range gs {
		if (g.State == "active" || g.State == "future") && !m.allowedTerm(right, m.terms(g, a.Owner)) {
			return false, nil
		}
	}
	caps, e := m.ceilings(a.ID)
	if e != nil {
		return false, e
	}
	for _, c := range caps {
		if !m.canDelegate(c.ManagementLimit, a.Owner) {
			return false, nil
		}
	}
	return true, nil
}
func (m *manager) options(a access.ManagedAccount) ([]string, error) {
	out := []string{}
	for _, right := range []string{"account", "credential"} {
		ok, e := m.controls(a, right)
		if e != nil {
			return nil, e
		}
		if ok {
			out = append(out, right)
		}
	}
	if a.ID != m.actor && a.ID != m.protected {
		for _, right := range areaRights["grants"] {
			if (right != "delegate" || a.ID != m.builtin) && m.ownerScope(right, a.Owner) {
				out = append(out, right)
			}
		}
	}
	return out, nil
}
func (m *manager) view(q access.ManagementQuery) (v access.ManagementView, err error) {
	for _, r := range app.ManagementRights {
		if m.permits(r, "", nil) {
			v.Rights = append(v.Rights, r)
		}
	}
	if q.Area == "context" {
		return v, nil
	}
	if !m.areaOK(q.Area) {
		return v, managementDenied()
	}
	if strings.HasPrefix(q.Object, "new-") {
		switch q.Area + "/" + q.Object {
		case "accounts/new-account":
			if !m.permits("account", "", nil) {
				return v, managementDenied()
			}
		case "roles/new-role", "roles/new-scope", "roles/new-fields":
			if !m.permits(policyRight(strings.TrimPrefix(q.Object, "new-")), "", nil) {
				return v, managementDenied()
			}
		default:
			return v, managementDenied()
		}
	} else if q.Object != "" && (q.Area != "roles" || !foundation.ValidID(q.Object)) {
		return v, managementDenied()
	}
	if err = m.prepare(); err != nil {
		return v, err
	}
	if err = m.directory(); err != nil {
		return v, err
	}
	// Use the selected subject's ownership for version and target choices.
	if q.Area == "grants" && q.Subject != "" && !m.root {
		if !foundation.ValidID(q.Subject) {
			return v, managementDenied()
		}
		target, e := m.account(q.Subject)
		if e != nil {
			return v, e
		}
		if !m.ownerScope("assign", target.Owner) && !m.ownerScope("delegate", target.Owner) {
			return v, managementDenied()
		}
		m.caps = slices.DeleteFunc(slices.Clone(m.caps), func(c app.ManagementCeiling) bool { return c.Owner != target.Owner })
	}
	if q.Area == "grants" && m.permits("delegate", "", nil) {
		v.SuggestedProxyDays = m.proxyDays
		for _, c := range m.caps {
			if slices.Contains(c.Rights, "delegate") && c.MaxProxyDays > 0 && (v.SuggestedProxyDays == 0 || c.MaxProxyDays < v.SuggestedProxyDays) {
				v.SuggestedProxyDays = c.MaxProxyDays
			}
		}
	}
	for _, t := range m.targets {
		if (t.Kind == "bu" || t.Kind == "instance") && slices.ContainsFunc(areaRights[q.Area], func(r string) bool { return m.ownerScope(r, t.ID) }) {
			for _, r := range app.ManagementRights {
				if m.ownerScope(r, t.ID) {
					t.Options = append(t.Options, r)
				}
			}
			v.Owners = append(v.Owners, t)
		}
		if q.Area != "accounts" && slices.ContainsFunc(areaRights[q.Area], func(r string) bool {
			return m.permits(r, "", func(c app.ManagementCeiling) bool { return c.TargetID == t.ID && c.TargetKind == t.Kind })
		}) && t.Kind != "instance" {
			v.Targets = append(v.Targets, t)
		}
	}
	if q.Area == "roles" || q.Area == "grants" && q.Choices && q.Subject != "" {
		for _, d := range m.versions {
			if slices.ContainsFunc(areaRights[q.Area], func(r string) bool { return m.definitionVisible(d, r) }) {
				for _, r := range app.ManagementRights {
					if m.definitionVisible(d, r) {
						d.Options = append(d.Options, r)
					}
				}
				v.Versions = append(v.Versions, d)
			}
		}
		for _, a := range app.ManagementCatalog() {
			if m.permits("role", "", func(c app.ManagementCeiling) bool { return slices.Contains(c.Actions, a.Key) }) {
				v.Actions = append(v.Actions, a)
			}
		}
		for _, fields := range []struct {
			all   []string
			dst   *[]string
			write bool
		}{{append(slices.Clone(app.DepartmentFields), app.QuotaFields...), &v.Read, false}, {append(slices.Clone(app.DepartmentWrite), app.QuotaWrite...), &v.Write, true}} {
			for _, f := range fields.all {
				if !slices.Contains(*fields.dst, f) && m.permits("policy", "", func(c app.ManagementCeiling) bool {
					if fields.write {
						return slices.Contains(c.Write, f)
					}
					return slices.Contains(c.Read, f)
				}) {
					*fields.dst = append(*fields.dst, f)
				}
			}
		}
	}
	if q.Object != "" && !strings.HasPrefix(q.Object, "new-") && !slices.ContainsFunc(v.Versions, func(d access.Definition) bool { return d.ID == q.Object }) {
		return v, managementDenied()
	}
	if q.Area != "roles" && q.Object == "" {
		identityQuery := q
		if q.Subject != "" {
			identityQuery.Search, identityQuery.State, identityQuery.Owner = "", "", ""
		}
		v.Accounts, err = m.accounts(identityQuery)
		if err != nil {
			return v, err
		}
		for i, a := range v.Accounts {
			// Expose required-target controls only in the selected account. The candidate list contains identity and status.
			if a.ID == q.Subject {
				v.Accounts[i].BuiltinSecurityAdmin = a.ID == m.builtin
				v.Accounts[i].GrantReadable = m.ownerScope("assign", a.Owner) || m.ownerScope("delegate", a.Owner)
				v.Accounts[i].Options, err = m.options(a)
				if err != nil {
					return v, err
				}
			}
		}
	}
	if q.Subject != "" {
		if !foundation.ValidID(q.Subject) || !slices.ContainsFunc(v.Accounts, func(a access.ManagedAccount) bool { return a.ID == q.Subject }) {
			return v, managementDenied()
		}
		a, err := m.account(q.Subject)
		if err != nil {
			return v, err
		}
		if q.Area == "grants" {
			if err = m.grantWorkspace(q, a, &v); err != nil {
				return v, err
			}
		}

	}
	sort.Slice(v.Versions, func(i, j int) bool {
		a, b := v.Versions[i], v.Versions[j]
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		if a.Number != b.Number {
			return a.Number > b.Number
		}
		return a.ID < b.ID
	})
	if q.Subject == "" && q.Object == "" {
		if q.Area == "roles" {
			v.Versions = slices.DeleteFunc(v.Versions, func(d access.Definition) bool {
				return (q.VersionKind != "" && d.Kind != q.VersionKind) || !strings.Contains(strings.ToLower(d.Name), strings.ToLower(q.Search))
			})
			v.Total = len(v.Versions)
		} else {
			v.Total = len(v.Accounts)
		}
		start, end, page := window(q, v.Total)
		v.Page = page
		if q.Area == "roles" {
			v.Versions = v.Versions[start:end]
		} else {
			v.Accounts = v.Accounts[start:end]
			if q.Area == "grants" {
				if err = m.countVisibleGrants(v.Accounts); err != nil {
					return v, err
				}
				v.Versions, v.Targets = nil, nil
			}
		}
	} else if q.Subject != "" {
		v.Accounts = slices.DeleteFunc(v.Accounts, func(a access.ManagedAccount) bool { return a.ID != q.Subject })
	}
	if q.Area != "grants" && (q.Subject != "" || q.Object != "") {
		v.Events, err = m.events(q, v)
		if err == nil && q.Panel == "events" {
			err = pageManagementEvents(q, v.Events, &v)
		}
	}
	if err == nil && q.Area == "roles" && q.Panel == "references" {
		var d access.Definition
		for _, def := range v.Versions {
			if def.ID == q.Object {
				d = def
				break
			}
		}
		if d.ID == "" {
			return v, managementDenied()
		}
		v.Impact, v.Total, v.Page, err = m.versionImpact(d, q)
	}
	return v, err
}
func (m *manager) events(q access.ManagementQuery, v access.ManagementView) ([]access.ManagementEvent, error) {
	ids := []string{}
	var selectedVersion *access.Definition
	switch q.Area {
	case "roles":
		for i := range v.Versions {
			d := &v.Versions[i]
			if d.ID == q.Object {
				selectedVersion = d
				ids = append(ids, d.ID)
				break
			}
		}
	case "accounts":
		if q.Subject != "" {
			ids = append(ids, q.Subject)
		}
	case "grants":
		for _, g := range v.Grants {
			ids = append(ids, g.ID)
		}
		for _, d := range v.Mandates {
			ids = append(ids, d.ID)
		}
	}
	rows, e := m.tx.Query(m.ctx, `SELECT coalesce(h.previous_owner::text,''),coalesce(h.assigned_owner::text,''),e.event_id::text,coalesce(p.login,h.operator_name,''),obj.id,
 CASE WHEN e.action='admin.handover.end' THEN e.action ELSE h.operation END,h.reason,e.occurred_at,h.request_id::text
 FROM access.management_receipt h JOIN access.security_event e USING(instance_id,request_id)
 LEFT JOIN access.principal p ON (h.instance_id,h.actor_id)=(p.instance_id,p.principal_id)
 CROSS JOIN LATERAL (SELECT CASE WHEN e.action LIKE 'version.%' OR e.action LIKE 'assignment.%' THEN e.reason_code
 WHEN e.action='admin.handover.end' THEN e.target_subject_id::text ELSE h.target_id::text END AS id) obj
 WHERE h.instance_id=$1 AND obj.id=ANY($2::text[]) AND ($3='' OR h.request_id::text=$3) ORDER BY h.receipt_order DESC,e.occurred_at DESC,e.event_id`, m.instance, ids, q.Request)
	if e != nil {
		return nil, e
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (access.ManagementEvent, error) {
		var event access.ManagementEvent
		err := row.Scan(&event.PreviousOwner, &event.AssignedOwner, &event.ID, &event.Actor, &event.Target, &event.Operation, &event.Reason, &event.At, &event.Request)
		for _, owner := range []*string{&event.PreviousOwner, &event.AssignedOwner} {
			if *owner != "" && !m.ownerScope("account", *owner) && !m.ownerScope("credential", *owner) {
				*owner = ""
			}
		}
		// Version identities are immutable. Use the definition named by this event,
		// even when a newer version is now available or this one has been disabled.
		switch event.Operation {
		case "version.publish":
			if selectedVersion != nil && selectedVersion.ID == event.Target {
				event.Summary = "版本"
				event.After = fmt.Sprintf("%s · V%d", selectedVersion.Name, selectedVersion.Number)
			}
		case "version.disable":
			event.Summary, event.Before, event.After = "版本状态", "可用", "已停用"
		case "account.disable":
			event.Summary, event.Before, event.After = "账号状态", "启用", "停用"
		case "account.enable":
			event.Summary, event.Before, event.After = "账号状态", "停用", "启用"
		}
		return event, err
	})
}
func (s *Store) ManagementView(ctx context.Context, c access.Credentials, q access.ManagementQuery) (v access.ManagementView, err error) {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return v, e
	}
	defer tx.Rollback(ctx)
	m, g, e := s.manager(ctx, tx, c, nil)
	if e != nil {
		return v, e
	}
	m.checking = true
	v, e = m.view(q)
	if e != nil {
		return v, e
	}
	return v, m.finish(g)
}
func areaFor(op string) string {
	switch strings.Split(op, ".")[0] {
	case "version":
		return "roles"
	case "grant", "mandate":
		return "grants"
	default:
		return "accounts"
	}
}

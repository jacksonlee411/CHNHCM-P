package postgres

import (
	app "chnhcm.local/chnhcm/internal/contexts/access/application"
	"context"
	"errors"
	"slices"
	"sort"
	"strings"
	"time"

	"chnhcm.local/chnhcm/internal/contexts/access/domain"
	"chnhcm.local/chnhcm/internal/contexts/access/infra/postgres/generated"
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
	"github.com/jackc/pgx/v5"
)

func managementDenied() error {
	return foundation.Fail(403, "ACCESS_DENIED", "无权办理此项管理操作")
}
func invalidManagement(message string) error { return foundation.Fail(400, "ACCESS_INPUT", message) }
func changedManagement() error {
	return foundation.Fail(409, "ACCESS_CONFLICT", "资料已更新，请核对当前版本后再保存")
}

type definitionKey struct{ kind, id string }
type manager struct {
	store                               *Store
	ctx                                 context.Context
	tx                                  pgx.Tx
	instance, actor, protected, builtin string
	root                                bool
	proxyDays                           int
	checking                            bool
	authorityUntil                      time.Time
	caps                                []app.ManagementCeiling
	versions                            map[definitionKey]access.Definition
	published                           map[definitionKey]bool
	targets                             []access.Target
}

func lockManager(ctx context.Context, tx pgx.Tx, instance string, write bool) error {
	// Only low-frequency administration shares this lock. Business transactions keep the per-subject/version locks.
	fn := "pg_advisory_xact_lock_shared"
	if write {
		fn = "pg_advisory_xact_lock"
	}
	_, err := tx.Exec(ctx, "SELECT "+fn+"(914027,hashtext($1))", instance)
	return err
}
func (s *Store) manager(ctx context.Context, tx pgx.Tx, c access.Credentials, cmd *access.ManagementCommand) (*manager, access.Guard, error) {
	var actor string
	if cmd != nil {
		session, e := generated.New(tx).ReadSession(ctx, generated.ReadSessionParams{InstanceID: s.Instance, TokenDigest: digest(c.Token)})
		if errors.Is(e, pgx.ErrNoRows) {
			return nil, nil, denied()
		}
		if e != nil {
			return nil, nil, e
		}
		if _, e = validateSession(session, c, time.Now()); e != nil {
			return nil, nil, e
		}
		actor = session.PrincipalID
	}
	if err := lockManager(ctx, tx, s.Instance, cmd != nil); err != nil {
		return nil, nil, err
	}
	if cmd != nil {
		target := cmd.Subject
		var retireKind, retireID string
		if cmd.Operation == "version.disable" && cmd.Version != nil {
			retireKind, retireID = cmd.Version.Kind, cmd.ID
		}
		if target != "" && !foundation.ValidID(target) || retireID != "" && !foundation.ValidID(retireID) {
			return nil, nil, managementDenied()
		}
		// Order actor and target before any session/version locks to avoid lock upgrades in opposite order.
		binding := access.VersionBinding{}
		switch cmd.Operation {
		case "grant.issue", "grant.adopt":
			binding = access.VersionBinding{Role: cmd.Role, Scope: cmd.Scope, Fields: cmd.Fields}
		case "mandate.issue":
			if l := cmd.Limit; l != nil {
				binding = access.VersionBinding{Role: l.Role, Scope: l.Scope, Fields: l.Fields}
			}
		case "version.publish":
			if d := cmd.Version; d != nil && d.Kind == "role" {
				binding.Scope, binding.Fields = d.Scope, d.Fields
			}
		}
		for _, id := range []string{binding.Role, binding.Scope, binding.Fields} {
			if id != "" && !foundation.ValidID(id) {
				return nil, nil, managementDenied()
			}
		}
		if e := lockManagement(ctx, tx, s.Instance, actor, target, binding, retireKind, retireID, true); e != nil {
			if errors.Is(e, pgx.ErrNoRows) || errors.Is(e, errMissingPolicyVersion) {
				return nil, nil, managementDenied()
			}
			return nil, nil, e
		}
	}
	g, e := s.authenticatedGuard(ctx, tx, c, false)
	if e != nil {
		return nil, nil, e
	}
	m := &manager{store: s, ctx: dbtx.With(ctx, tx), tx: tx, instance: s.Instance, actor: g.Principal().ID}
	e = tx.QueryRow(ctx, "SELECT principal_id::text,max_proxy_days FROM access.management_root WHERE instance_id=$1", s.Instance).Scan(&m.protected, &m.proxyDays)
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		return nil, nil, e
	}
	if m.protected != "" {
		enterprise, e := s.EnterpriseID(m.ctx)
		if e != nil {
			return nil, nil, e
		}
		m.builtin, e = initializedAdminID(ctx, tx, s.Instance, enterprise)
		if e != nil {
			return nil, nil, e
		}
	}
	m.root = m.isAdministrator(m.actor)
	if !m.root {
		caps, err := m.ceilings(m.actor)
		if err != nil {
			return nil, nil, err
		}
		m.caps = caps
	}
	return m, g, nil
}

func (m *manager) isAdministrator(subject string) bool {
	return subject != "" && (subject == m.protected || subject == m.builtin)
}

const mandateFrom = ` FROM access.management_mandate m
 JOIN access.policy_version r ON (r.instance_id,r.kind,r.version_id)=(m.instance_id,'role',m.role_id)
 JOIN access.policy_version s ON (s.instance_id,s.kind,s.version_id)=(m.instance_id,'scope',m.scope_id)
 JOIN access.policy_version f ON (f.instance_id,f.kind,f.version_id)=(m.instance_id,'fields',m.fields_id) `

func (m *manager) ceilings(subject string) ([]app.ManagementCeiling, error) {
	rows, e := m.tx.Query(m.ctx, `SELECT m.owner_id::text,m.rights,m.role_id::text,m.scope_id::text,m.fields_id::text,m.max_proxy_days,m.valid_until,r.actions,s.scope_kind,s.scope_id::text,f.read_fields,f.write_fields`+mandateFrom+`WHERE m.instance_id=$1 AND m.principal_id=$2 AND m.revoked_at IS NULL AND (m.valid_until IS NULL OR m.valid_until>clock_timestamp()) AND r.disabled_at IS NULL AND s.disabled_at IS NULL AND f.disabled_at IS NULL`, m.instance, subject)
	if e != nil {
		return nil, e
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (app.ManagementCeiling, error) {
		var c app.ManagementCeiling
		e := r.Scan(&c.Owner, &c.Rights, &c.Role, &c.Scope, &c.Fields, &c.MaxProxyDays, &c.Until, &c.Actions, &c.TargetKind, &c.TargetID, &c.Read, &c.Write)
		return c, e
	})
}
func (m *manager) permits(right, owner string, check func(app.ManagementCeiling) bool) bool {
	if m.root {
		return true
	}
	now := time.Now()
	var last time.Time
	for _, c := range m.caps {
		if c.Until != nil && !now.Before(*c.Until) || owner != "" && c.Owner != owner || !slices.Contains(c.Rights, right) || check != nil && !check(c) {
			continue
		}
		if c.Until == nil {
			return true
		}
		if c.Until.After(last) {
			last = *c.Until
		}
	}
	if last.IsZero() {
		return false
	}
	if m.checking && (m.authorityUntil.IsZero() || last.Before(m.authorityUntil)) {
		m.authorityUntil = last
	}
	return true
}

// finish checks the authority actually used by this request after its final lock wait.
func (m *manager) finish(g access.Guard) error {
	if err := g.RecordSessionActivity(m.ctx); err != nil {
		return err
	}
	if !m.authorityUntil.IsZero() {
		var now time.Time
		if err := m.tx.QueryRow(m.ctx, "SELECT clock_timestamp()").Scan(&now); err != nil {
			return err
		}
		if !now.Before(m.authorityUntil) || !time.Now().Before(m.authorityUntil) {
			return managementDenied()
		}
	}
	return m.tx.Commit(m.ctx)
}
func (m *manager) allowedTerm(right string, t app.PermissionTerms) bool {
	return m.permits(right, t.Owner, func(c app.ManagementCeiling) bool { return c.Covers(t) })
}
func (m *manager) prepare() error {
	rows, e := m.tx.Query(m.ctx, `SELECT v.kind,v.version_id::text,v.definition_id::text,v.version_no,coalesce(p.name,''),v.actions,
 coalesce(v.scope_kind,''),coalesce(v.scope_id::text,''),v.read_fields,v.write_fields, v.disabled_at IS NOT NULL,
 coalesce(p.scope_id::text,''),coalesce(p.fields_id::text,'') FROM access.policy_version v LEFT JOIN access.management_publication p USING(instance_id,kind,version_id) WHERE v.instance_id=$1`, m.instance)
	if e != nil {
		return e
	}
	defer rows.Close()
	m.versions = map[definitionKey]access.Definition{}
	m.published = map[definitionKey]bool{}
	names := map[string]string{}
	for _, a := range app.ManagementCatalog() {
		names[a.Key] = a.Name
	}
	for rows.Next() {
		var v access.Definition
		if e = rows.Scan(&v.Kind, &v.ID, &v.DefinitionID, &v.Number, &v.Name, &v.Actions, &v.TargetKind, &v.TargetID, &v.Read, &v.Write, &v.Disabled, &v.Scope, &v.Fields); e != nil {
			return e
		}
		key := definitionKey{v.Kind, v.ID}
		m.published[key] = v.Name != ""
		for _, action := range v.Actions {
			name := names[action]
			if name == "" {
				name = app.ManagementActionName(action)
			}
			v.ActionNames = append(v.ActionNames, name)
		}
		if v.Name == "" {
			switch v.Kind {
			case "role":
				v.Name = strings.Join(v.ActionNames, "、")
			case "scope":
				v.Name = map[string]string{"bu": "业务单元范围", "enterprise": "企业范围", "position": "岗位范围", "department": "部门范围", "instance": "实例范围"}[v.TargetKind]
				if v.Name == "" {
					v.Name = "原范围策略"
				}
			case "fields":
				v.Name = "原字段策略"
			}
		}
		m.versions[key] = v
	}
	return rows.Err()
}
func (m *manager) directory() error {
	var ids []string
	if !m.root {
		ids = []string{}
		for _, c := range m.caps {
			if c.Until == nil || time.Now().Before(*c.Until) {
				ids = append(ids, c.Owner, c.TargetID)
			}
		}
	}
	targets, e := m.store.Targets(m.ctx, ids)
	if e != nil {
		return e
	}
	targets = append(targets, access.Target{Kind: "instance", ID: m.instance, Name: "实例直管"})
	sort.Slice(targets, func(i, j int) bool {
		if targets[i].Kind != targets[j].Kind {
			return targets[i].Kind < targets[j].Kind
		}
		return targets[i].Code < targets[j].Code
	})
	m.targets = targets
	return nil
}
func (m *manager) hasTarget(kind, id string) bool {
	return slices.ContainsFunc(m.targets, func(t access.Target) bool { return t.Kind == kind && t.ID == id })
}
func (m *manager) ownerAllowed(right, owner string) bool {
	return (m.hasTarget("bu", owner) || m.hasTarget("instance", owner)) && m.permits(right, owner, nil)
}
func (m *manager) binding(role, scope, fields string) (r, s, f access.Definition) {
	return m.versions[definitionKey{"role", role}], m.versions[definitionKey{"scope", scope}], m.versions[definitionKey{"fields", fields}]
}
func (m *manager) terms(g access.ManagedGrant, owner string) app.PermissionTerms {
	r, s, f := m.binding(g.Role, g.Scope, g.Fields)
	return app.PermissionTerms{Owner: owner, Actions: r.Actions, Read: f.Read, Write: f.Write, TargetKind: s.TargetKind, TargetID: s.TargetID, Purpose: g.Purpose, Source: g.Source, From: g.From, Until: g.Until}
}
func (m *manager) definitionVisible(v access.Definition, right string) bool {
	if !m.published[definitionKey{v.Kind, v.ID}] {
		return false
	}
	return m.permits(right, "", func(c app.ManagementCeiling) bool {
		switch v.Kind {
		case "role":
			_, s, f := m.binding(v.ID, v.Scope, v.Fields)
			return domain.ContainsAll(c.Actions, v.Actions) && c.TargetKind == s.TargetKind && c.TargetID == s.TargetID && domain.ContainsAll(c.Read, f.Read) && domain.ContainsAll(c.Write, f.Write)
		case "scope":
			return c.TargetKind == v.TargetKind && c.TargetID == v.TargetID
		case "fields":
			return domain.ContainsAll(c.Read, v.Read) && domain.ContainsAll(c.Write, v.Write)
		}
		return false
	})
}

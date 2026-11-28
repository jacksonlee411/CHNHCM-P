package postgres

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"sort"
	"strings"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"github.com/jackc/pgx/v5"
)

// Resolve each event against the immutable revision written by its request, including its predecessor.
func (m *manager) describeEvents(events []access.ManagementEvent, grants []access.ManagedGrant, mandates []access.Mandate, owner string) error {
	if len(events) == 0 {
		return nil
	}
	byRequest := map[string]*access.ManagementEvent{}
	ids := make([]string, 0, len(events))
	for i := range events {
		e := &events[i]
		byRequest[e.Request] = e
		ids = append(ids, e.Request)
	}
	terms := map[string]access.ManagedGrant{}
	for _, g := range grants {
		terms[g.ID] = g
	}
	rows, err := m.tx.Query(m.ctx, `SELECT r.request_id::text,r.grant_id::text,r.role_id::text,r.scope_id::text,r.fields_id::text,
 coalesce(p.role_id::text,''),coalesce(p.scope_id::text,''),coalesce(p.fields_id::text,'')
 FROM access.grant_revision r LEFT JOIN access.grant_revision p ON p.instance_id=r.instance_id AND p.grant_id=r.grant_id AND p.revision=r.revision-1
 WHERE r.instance_id=$1 AND r.request_id=ANY($2::uuid[])`, m.instance, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var req, id, r, s, f, pr, ps, pf string
		if err := rows.Scan(&req, &id, &r, &s, &f, &pr, &ps, &pf); err != nil {
			return err
		}
		e := byRequest[req]
		g, ok := terms[id]
		if !ok || e.Target != id {
			continue
		}
		describe := func(r, s, f string) string {
			if r == "" {
				return ""
			}
			g.Role, g.Scope, g.Fields = r, s, f
			if !m.allowedTerm("assign", m.terms(g, owner)) {
				return ""
			}
			role, scope, _ := m.binding(g.Role, g.Scope, g.Fields)
			e.Summary = fmt.Sprintf("%s · V%d / %s / %s", role.Name, role.Number, m.targetLabel(scope.TargetID), grantSource(g.Source))
			return m.grantLabel(g)
		}
		e.Before = describe(pr, ps, pf)
		e.After = describe(r, s, f)
		if e.Operation == "grant.revoke" && e.After != "" {
			e.After = "已撤销"
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for i := range events {
		e := &events[i]
		if !strings.HasPrefix(e.Operation, "mandate.") {
			continue
		}
		for _, d := range mandates {
			if d.ID == e.Target {
				l := d.Limit
				summary := m.mandateLabel(l)
				e.Summary = m.targetLabel(l.Owner) + " / " + m.versions[definitionKey{"role", l.Role}].Name
				if e.Operation == "mandate.revoke" {
					e.Before = summary
					e.After = "已撤销"
				} else {
					e.After = summary
				}
				break
			}
		}
	}
	return nil
}

// The read and the locked disable command compute the same effective references.
// A restricted reference prevents issuing a token. Only permitted identities and counts enter the view.
func (m *manager) versionImpact(d access.Definition, q access.ManagementQuery) (*access.VersionImpact, int, int, error) {
	impact := &access.VersionImpact{}
	complete := !d.Disabled
	accounts, err := m.accounts(access.ManagementQuery{Area: "grants"})
	if err != nil {
		return nil, 0, 0, err
	}
	owners := map[string]access.ManagedAccount{}
	for _, a := range accounts {
		owners[a.ID] = a
	}
	grants, err := m.grants("", d.Kind, d.ID)
	if err != nil {
		return nil, 0, 0, err
	}
	affected := map[string]bool{}
	var parts []string
	for _, g := range grants {
		a, ok := owners[g.Subject]
		visible := ok && m.ownerScope("assign", a.Owner) && m.allowedTerm("assign", m.terms(g, a.Owner))
		live := g.State == "active" || g.State == "future"
		if live {
			if !visible || a.ID == m.actor || a.ID == m.protected {
				complete = false
			}
			if visible {
				impact.Grants++
				affected[a.ID] = true
				parts = append(parts, fmt.Sprintf("g/%s/%d/%s", g.ID, g.Revision, g.State))
			}
		}
		if visible && (q.State != "impact" || live) {
			impact.References = append(impact.References, access.VersionReference{ID: g.ID, Subject: a.ID, Account: a.Login, Kind: "grant", State: g.State, Summary: m.grantLabel(g)})
		}
	}
	rows, err := m.tx.Query(m.ctx, `SELECT m.id::text,m.principal_id::text,m.owner_id::text,m.rights,m.role_id::text,m.scope_id::text,m.fields_id::text,m.max_proxy_days,m.valid_until`+mandateFrom+`WHERE m.instance_id=$1 AND m.revoked_at IS NULL AND (m.valid_until IS NULL OR m.valid_until>clock_timestamp())
 AND r.disabled_at IS NULL AND s.disabled_at IS NULL AND f.disabled_at IS NULL AND (CASE $2 WHEN 'role' THEN m.role_id WHEN 'scope' THEN m.scope_id WHEN 'fields' THEN m.fields_id END)=$3::uuid`, m.instance, d.Kind, d.ID)
	if err != nil {
		return nil, 0, 0, err
	}
	type ref struct {
		id, subject string
		limit       access.ManagementLimit
	}
	refs, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (ref, error) {
		var x ref
		l := &x.limit
		err := row.Scan(&x.id, &x.subject, &l.Owner, &l.Rights, &l.Role, &l.Scope, &l.Fields, &l.MaxProxyDays, &l.Until)
		return x, err
	})
	if err != nil {
		return nil, 0, 0, err
	}
	for _, x := range refs {
		a, ok := owners[x.subject]
		visible := ok && m.ownerScope("delegate", a.Owner) && m.canDelegate(x.limit, a.Owner)
		if !visible || x.subject == m.actor {
			complete = false
		}
		if visible {
			l := x.limit
			impact.Mandates++
			affected[a.ID] = true
			parts = append(parts, "m/"+x.id)
			impact.References = append(impact.References, access.VersionReference{ID: x.id, Subject: x.subject, Account: a.Login, Kind: "mandate", State: "active", Summary: m.mandateLabel(l)})
		}
	}
	impact.Accounts = len(affected)
	if complete {
		sort.Strings(parts)
		sum := sha256.Sum256([]byte(d.Kind + "/" + d.ID + "\n" + strings.Join(parts, "\n")))
		impact.Token = hex.EncodeToString(sum[:])
	}
	sort.Slice(impact.References, func(i, j int) bool {
		a, b := impact.References[i], impact.References[j]
		if a.Account != b.Account {
			return a.Account < b.Account
		}
		return a.ID < b.ID
	})
	if q.Search != "" {
		impact.References = slices.DeleteFunc(impact.References, func(r access.VersionReference) bool {
			return !strings.Contains(strings.ToLower(r.Account+" "+r.Summary), strings.ToLower(q.Search))
		})
	}
	total := len(impact.References)
	q = locate(q, impact.References, func(r access.VersionReference) string { return r.ID })
	start, end, page := window(q, total)
	impact.References = impact.References[start:end]
	return impact, total, page, nil
}

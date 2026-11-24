package postgres

import (
	"time"

	"chnhcm.local/chnhcm/internal/contexts/access/domain"
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

type timedGrant struct {
	permission  access.Grant
	from, until time.Time
}

func (g *guard) selectAt(now time.Time) {
	g.grants = nil
	g.readActions = nil
	g.boundary = time.Time{}
	for _, term := range g.windows {
		next := term.until
		if now.Before(term.from) {
			next = term.from
		} else if term.until.IsZero() || now.Before(term.until) {
			g.grants = append(g.grants, term.permission)
		}
		if next.After(now) && (g.boundary.IsZero() || next.Before(g.boundary)) {
			g.boundary = next
		}
	}
}

func (g *guard) refresh() {
	if !g.boundary.IsZero() {
		if now := time.Now(); !now.Before(g.boundary) {
			g.selectAt(now)
			g.changedWindow = true
		}
	}
}

func coversDecision(now, used access.Decision) bool {
	return now.Allowed && domain.ContainsAll(now.ReadFields, used.ReadFields) && domain.ContainsAll(now.WriteFields, used.WriteFields)
}

// ValidateUse runs after all waits in the original transaction. If a window
// closed while a caller used a decision, the caller must roll back its writes.
func (g *guard) ValidateUse() error {
	for _, nested := range g.dependents {
		if err := nested.ValidateUse(); err != nil {
			return err
		}
	}
	g.refresh()
	if !g.changedWindow {
		return nil
	}
	for _, check := range g.checks {
		if !check(g) {
			return foundation.Fail(403, "ACCESS_DENIED", "当前授权已到期")
		}
	}
	g.changedWindow = false
	return nil
}

func (g *guard) Grants() []access.Grant {
	g.refresh()
	used := append([]access.Grant{}, g.grants...)
	g.retainProjection(used)
	return used
}

// A caller that compiles predicates from an entire projection may have used any
// field in it. An equivalent surviving grant continues to cover the projection.
func (g *guard) retainProjection(used []access.Grant) {
	if g.boundary.IsZero() {
		return
	}
	g.checks = append(g.checks, func(current *guard) bool {
		for _, old := range used {
			covered := false
			for _, live := range current.grants {
				if old.Action == live.Action && old.ScopeKind == live.ScopeKind && old.ScopeID == live.ScopeID && domain.ContainsAll(live.ReadFields, old.ReadFields) && domain.ContainsAll(live.WriteFields, old.WriteFields) {
					covered = true
					break
				}
			}
			if !covered {
				return false
			}
		}
		return true
	})
}

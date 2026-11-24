package geoarea

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	country "chnhcm.local/chnhcm/internal/contexts/organization/application/country"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/geoarea"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"slices"
	"strings"
	"time"
)

// ReferenceUnit is implemented by the consumer's existing transaction. It has
// no submission, catalog publication or geographic write capability.
type ReferenceUnit interface {
	Access() access.Guard
	Now(context.Context) (time.Time, error)
	Country(context.Context, string, string) (country.State, error)
	Load(context.Context, string, string) (State, error)
	LockReference(context.Context, string, string) error
	Published(context.Context, string) (bool, error)
	IsTerminal(context.Context, string) (bool, error)
}
type ReferenceRequest struct{ CountryID, AreaID, Date string }

// ResolveAddresses locks a finite union before resolving any record. Country
// locks precede area locks; area ordering is country, depth, then stable UUID.
func ResolveAddresses(ctx context.Context, u ReferenceUnit, requests []ReferenceRequest) ([]d.Adoption, error) {
	ctx, cancel := context.WithTimeout(ctx, foundation.ObjectGuardBudget(ctx))
	defer cancel()
	countries := map[string]bool{}
	type node struct {
		country, id string
		depth       int
	}
	nodes := map[string]node{}
	structures := make([]d.Structure, len(requests))
	for i, r := range requests {
		if !foundation.ValidID(r.CountryID) || !foundation.ValidDate(r.Date) || r.AreaID != "" && !foundation.ValidID(r.AreaID) {
			return nil, fault("M02_INVALID_INPUT", "countryRegionId", "地址引用或采用日期无效")
		}
		c, e := u.Country(ctx, r.CountryID, "")
		if e != nil {
			return nil, e
		}
		code := c.Timeline.Origin.Code
		if !slices.Contains([]string{"CN", "HK", "MO", "TW"}, code) || code == "MO" && r.AreaID != "" || code != "MO" && r.AreaID == "" {
			return nil, refUnavailable()
		}
		countries[r.CountryID] = true
		structures[i] = d.Structure{CountryID: r.CountryID, CountryCode: code, ParentID: r.AreaID}
		chain := []string{}
		id := r.AreaID
		for id != "" {
			if len(chain) >= 3 || slices.Contains(chain, id) {
				return nil, refUnavailable()
			}
			a, e := u.Load(ctx, id, "")
			if e != nil {
				return nil, e
			}
			if a.Timeline.ID == "" || a.Timeline.Origin.CountryID != r.CountryID {
				return nil, refUnavailable()
			}
			chain = append(chain, id)
			id = a.Timeline.Origin.ParentID
		}
		slices.Reverse(chain)
		for depth, id := range chain {
			nodes[id] = node{r.CountryID, id, depth}
		}
		if r.AreaID != "" {
			published, e := u.Published(ctx, r.AreaID)
			if e != nil {
				return nil, e
			}
			if !published {
				return nil, refUnavailable()
			}
			leaf, e := u.IsTerminal(ctx, r.AreaID)
			if e != nil {
				return nil, e
			}
			if !leaf {
				return nil, refUnavailable()
			}
		}
	}
	ids := make([]string, 0, len(countries))
	for id := range countries {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, id := range ids {
		if e := u.LockReference(ctx, "country", id); e != nil {
			return nil, e
		}
	}
	ordered := make([]node, 0, len(nodes))
	for _, n := range nodes {
		ordered = append(ordered, n)
	}
	slices.SortFunc(ordered, func(a, b node) int {
		if a.country != b.country {
			return strings.Compare(a.country, b.country)
		}
		if a.depth != b.depth {
			return a.depth - b.depth
		}
		return strings.Compare(a.id, b.id)
	})
	for _, n := range ordered {
		if e := u.LockReference(ctx, "geoarea", n.id); e != nil {
			return nil, e
		}
	}
	out := make([]d.Adoption, len(requests))
	for i, r := range requests {
		a, e := adopt(ctx, u, structures[i], r.Date, false)
		if e != nil {
			return nil, e
		}
		out[i] = a
	}
	return out, nil
}

// ProjectAdoption always uses exact source members and current read/history
// field intersections, even when an adopted source still happens to be head.
func ProjectAdoption(ctx context.Context, u ReferenceUnit, a d.Adoption, today string) ([]org.GeoAreaPathNode, error) {
	path, e := projectPath(ctx, u, a, true, today)
	if e != nil || len(path) > 0 || len(a.Parents) == 0 {
		return path, e
	}
	a.Parents = nil
	return projectPath(ctx, u, a, true, today)
}

// ResolveProfileArea admits an intermediate or terminal address component and
// returns its active, authorized chain under the caller's transaction locks.
func ResolveProfileArea(ctx context.Context, u ReferenceUnit, id, day string) ([]org.GeoAreaPathNode, error) {
	ok, err := u.Published(ctx, id)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, refUnavailable()
	}
	v, err := u.Load(ctx, id, "")
	if err != nil {
		return nil, err
	}
	structure := v.Timeline.Origin.Structure
	structure.ParentID = id
	a, err := adopt(ctx, u, structure, day, true)
	if err != nil {
		return nil, err
	}
	path, err := projectPath(ctx, u, a, false, day)
	if err != nil {
		return nil, err
	}
	if len(path) != len(a.Parents)+1 {
		return nil, refUnavailable()
	}
	path[0].SourceVersion = a.Country.SourceVersion
	for i, ref := range a.Parents {
		path[i+1].SourceVersion = ref.SourceVersion
		path[i+1].RecordID = ref.RecordID
		path[i+1].Revision = ref.Revision
	}
	return path, nil
}

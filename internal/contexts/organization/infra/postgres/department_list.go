package postgres

import (
	"context"
	"slices"
	"strings"
	"sync"

	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store immutable source members and Go-produced ranges, never dates, statuses,
// paths or grants. Pin all heads once, then load the required exact intervals.
const listCacheBytes = 768 << 20

var departmentCacheInitialization sync.Mutex

func (s *Store) departmentReadCache() *departmentListCache {
	departmentCacheInitialization.Lock()
	defer departmentCacheInitialization.Unlock()
	if s.listCache == nil || s.listCache.instance != s.Instance || s.listCache.pool != s.Pool {
		s.listCache = &departmentListCache{instance: s.Instance, pool: s.Pool}
	}
	return s.listCache
}

type cachedBU struct {
	value d.BusinessUnit
	bytes int
}
type departmentListCache struct {
	instance   string
	pool       *pgxpool.Pool
	mu         sync.RWMutex
	bus        map[string]cachedBU
	bytes      int
	loading    chan struct{}
	generation uint64
}

func (u *departmentUnit) CreationBUs(ctx context.Context) ([]string, error) {
	defer foundation.BeginStage(ctx, "context.bu_identities")()
	return u.q.ReadDepartmentCreationBUs(ctx, u.instance)
}

func buCovers(t d.BusinessUnit, date string) bool {
	if len(t.Records) == 0 {
		return false
	}
	if date < t.Records[0].Date {
		return t.Records[0].ID == t.First
	}
	r := d.BUAt(t.Records, date)
	return r != nil && r.Date <= date && (r.End == "" || date < r.End)
}
func mergeDepartment(old, next d.Timeline) d.Timeline {
	if old.Source != next.Source || old.RelationSource != next.RelationSource {
		return next
	}
	records := map[string]d.Record{}
	for _, r := range old.Records {
		records[r.ID] = r
	}
	for _, r := range next.Records {
		records[r.ID] = r
	}
	next.Records = make([]d.Record, 0, len(records))
	for _, r := range records {
		next.Records = append(next.Records, r)
	}
	slices.SortFunc(next.Records, func(a, b d.Record) int { return strings.Compare(a.Date, b.Date) })
	edges := map[string]d.Edge{}
	for _, e := range old.Edges {
		edges[e.ID] = e
	}
	for _, e := range next.Edges {
		edges[e.ID] = e
	}
	next.Edges = make([]d.Edge, 0, len(edges))
	for _, e := range edges {
		next.Edges = append(next.Edges, e)
	}
	slices.SortFunc(next.Edges, func(a, b d.Edge) int { return strings.Compare(a.Start, b.Start) })
	return next
}
func mergeBU(old, next d.BusinessUnit) d.BusinessUnit {
	if old.Source != next.Source {
		return next
	}
	records := map[string]d.BURecord{}
	for _, r := range old.Records {
		records[r.ID] = r
	}
	for _, r := range next.Records {
		records[r.ID] = r
	}
	next.Records = make([]d.BURecord, 0, len(records))
	for _, r := range records {
		next.Records = append(next.Records, r)
	}
	slices.SortFunc(next.Records, func(a, b d.BURecord) int { return strings.Compare(a.Date, b.Date) })
	return next
}

// Only the active BU consumer uses this loader. Department query capture owns
// its own consistent facts and no longer uses the former forest cache.
func (u *departmentUnit) loadBUListFacts(ctx context.Context, bs []map[string]any) (out d.Snapshot, err error) {
	defer foundation.BeginStage(ctx, "list.load_facts")()
	const batch = 512
	for start := 0; start < len(bs); start += batch {
		v, e := u.readListSegments(ctx, nil, bs[start:min(start+batch, len(bs))])
		if e != nil {
			return out, e
		}
		out.BusinessUnits = append(out.BusinessUnits, v.BusinessUnits...)
	}
	return out, nil
}

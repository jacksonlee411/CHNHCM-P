package history

import access "chnhcm.local/chnhcm/internal/contexts/access/public"

// SQL only tests membership in X03's already compiled sets. JSON object keys
// preserve that union while avoiding a linear JSON array scan for every fact.
// The originating Go plan, its validation and the current transaction remain
// authoritative; this representation is neither a new policy nor a cache.
type queryReadSet struct {
	All         bool            `json:"all"`
	BUs         map[string]bool `json:"bus"`
	Departments map[string]bool `json:"departments"`
}

func querySet(s access.ReadSet) queryReadSet {
	keys := func(ids []string) map[string]bool {
		out := make(map[string]bool, len(ids))
		for _, id := range ids {
			out[id] = true
		}
		return out
	}
	return queryReadSet{All: s.All, BUs: keys(s.BUs), Departments: keys(s.Departments)}
}

type queryReadPlan struct {
	Allowed queryReadSet            `json:"allowed"`
	Fields  map[string]queryReadSet `json:"fields"`
}

func queryPlan(p access.ReadPlan) queryReadPlan {
	out := queryReadPlan{Allowed: querySet(p.Allowed), Fields: make(map[string]queryReadSet, len(p.Fields))}
	for name, set := range p.Fields {
		out.Fields[name] = querySet(set)
	}
	return out
}

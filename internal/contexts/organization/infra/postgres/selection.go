package postgres

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"slices"
)

// Both the page and this entire-set basis come from the same SQL snapshot.
func selectionVersion(in org.ListInput, ent org.Enterprise, guard access.Guard, raw []byte) (string, string, error) {
	p := guard.Principal()
	predicate := []any{p.InstanceID, ent.ID, in.BusinessAt, in.Q, in.Status, in.Filter}
	key := hashJSON(predicate)
	var members [][2]string
	if err := json.Unmarshal(raw, &members); err != nil {
		return "", "", err
	}
	rows := make([]any, 0, len(members))
	for _, m := range members {
		fields := map[string]bool{}
		for _, g := range guard.Grants() {
			if g.Action == "bu.read" && (g.ScopeKind == "enterprise" && g.ScopeID == ent.ID || g.ScopeKind == "bu" && g.ScopeID == m[0]) {
				for _, f := range g.ReadFields {
					fields[f] = true
				}
			}
		}
		names := make([]string, 0, len(fields))
		for f := range fields {
			names = append(names, f)
		}
		slices.Sort(names)
		rows = append(rows, []any{m[0], m[1], names})
	}
	return key, hashJSON([]any{p.InstanceID, ent.ID, p.ID, p.SessionID, key, rows}), nil
}
func hashJSON(v any) string {
	b, _ := json.Marshal(v)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

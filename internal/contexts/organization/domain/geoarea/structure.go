package geoarea

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"slices"
)

type Node struct {
	ID, EffectiveDate string
	Structure         Structure
}

// ValidateTree validates one complete input, before any publication. The returned
// order is deterministic and parent first. It does not grant adoption permission.
func ValidateTree(nodes []Node) ([]string, error) {
	byID := make(map[string]Node, len(nodes))
	codes := map[[3]string]bool{}
	for _, n := range nodes {
		if !foundation.ValidID(n.ID) || !foundation.ValidDate(n.EffectiveDate) {
			return nil, invalid("id", "区域身份或日期无效")
		}
		if _, ok := byID[n.ID]; ok {
			return nil, invalid("id", "区域身份重复")
		}
		if e := ValidateStructure(n.Structure); e != nil {
			return nil, e
		}
		k := [3]string{n.Structure.CountryID, n.Structure.Scheme, n.Structure.Code}
		if n.Structure.Code != "" && codes[k] {
			return nil, invalid("code", "同地区及方案的代码重复")
		}
		codes[k] = true
		byID[n.ID] = n
	}
	ids := make([]string, 0, len(nodes))
	for id := range byID {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	state := map[string]int{}
	out := make([]string, 0, len(nodes))
	var visit func(string) error
	visit = func(id string) error {
		if state[id] == 2 {
			return nil
		}
		if state[id] == 1 {
			return invalid("parentAreaId", "区域父关系成环")
		}
		state[id] = 1
		n := byID[id]
		s := n.Structure
		if s.ParentID == "" {
			if !slices.Contains([]string{"province", "region", "county-city", "postal-region", "parish"}, s.Level) {
				return invalid("parentAreaId", "此层级不能作为根区域")
			}
		} else {
			p, ok := byID[s.ParentID]
			if !ok {
				return invalid("parentAreaId", "父区域缺失")
			}
			if p.Structure.CountryID != s.CountryID || p.Structure.CountryCode != s.CountryCode || p.EffectiveDate > n.EffectiveDate {
				return invalid("parentAreaId", "父区域归属或日期不合法")
			}
			legal := s.CountryCode == "CN" && (p.Structure.Level == "province" && (s.Level == "prefecture" || s.Level == "county") || p.Structure.Level == "prefecture" && s.Level == "county") || s.CountryCode == "HK" && p.Structure.Level == "region" && s.Level == "district" || s.CountryCode == "TW" && p.Structure.Level == "county-city" && s.Level == "township"
			if !legal {
				return invalid("level", "父子层级组合不合法")
			}
			if e := visit(s.ParentID); e != nil {
				return e
			}
		}
		state[id] = 2
		out = append(out, id)
		return nil
	}
	for _, id := range ids {
		if e := visit(id); e != nil {
			return nil, e
		}
	}
	return out, nil
}

// TerminalIDs consumes the complete admitted structure, never filtered candidates.
func TerminalIDs(nodes []Node) (map[string]bool, error) {
	if _, e := ValidateTree(nodes); e != nil {
		return nil, e
	}
	out := make(map[string]bool, len(nodes))
	for _, n := range nodes {
		out[n.ID] = true
	}
	for _, n := range nodes {
		delete(out, n.Structure.ParentID)
	}
	return out, nil
}

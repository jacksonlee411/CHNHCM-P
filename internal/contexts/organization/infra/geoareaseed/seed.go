// Package geoareaseed prepares versioned offline input for the fixed Go initializer.
package geoareaseed

import (
	"bytes"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/geoarea"
	"chnhcm.local/chnhcm/internal/foundation"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
)

//go:embed areas.json
var fixed []byte

const FileDigest = "f5fe9227f4d5d9c0bb7a7a90e21619bb9116967448320e3708a767d4fdcb5fb0"

type Row struct {
	Code          string `json:"code"`
	CountryKey    string `json:"countryKey"`
	EffectiveDate string `json:"effectiveDate"`
	Level         string `json:"level"`
	Name          string `json:"name"`
	Nature        string `json:"nature"`
	ParentKey     string `json:"parentKey"`
	Scheme        string `json:"scheme"`
	SourceKey     string `json:"sourceKey"`
	SourceVersion string `json:"sourceVersion"`
}
type Package struct {
	Version       string `json:"version"`
	Digest        string `json:"packageDigest"`
	ArchiveDigest string `json:"archiveDigest"`
	Areas         []Row  `json:"areas"`
}
type Prepared struct {
	Node                           d.Node
	Name, SourceKey, SourceVersion string
}

func Fixed() (Package, error) {
	var p Package
	sum := sha256.Sum256(fixed)
	if hex.EncodeToString(sum[:]) != FileDigest {
		return p, fmt.Errorf("fixed area bytes differ")
	}
	dec := json.NewDecoder(bytes.NewReader(fixed))
	dec.DisallowUnknownFields()
	if e := dec.Decode(&p); e != nil {
		return p, e
	}
	if p.Version != "chnhcm-geography-g0-v1" || p.Digest != "dc7eac11535bb2da2f3f72431d0767dc510c250a3e11596ffd76428e3fafc123" || p.ArchiveDigest != "8ffa0c5f633d6f5706dcff2bef7d86bf3af44cdbb7bcd76e638e5a2733553ad7" || len(p.Areas) != 3620 {
		return p, fmt.Errorf("fixed area manifest differs")
	}
	return p, nil
}
func objectID(instance, scheme, key string) string {
	b, _ := json.Marshal([]string{"organization.geoarea", instance, scheme, key})
	return uuid.NewSHA1(uuid.NameSpaceDNS, b).String()
}

// Prepare binds source keys to the authenticated instance's already resolved
// country identities. The caller must verify those countries and authorization.
// No adoption history is invented here; the write transaction must capture it.
func (p Package) Prepare(instance string, countries map[string]string) ([]Prepared, error) {
	if !foundation.ValidID(instance) {
		return nil, fmt.Errorf("invalid instance")
	}
	codes := map[string]string{"base.cn": "CN", "base.hk": "HK", "base.tw": "TW"}
	expected := map[string]int{"base.cn": 3209, "base.hk": 21, "base.tw": 390}
	load := Fixed
	if p.Version == macauVersion {
		load = Macau
		codes = map[string]string{"base.mo": "MO"}
		expected = map[string]int{"base.mo": 10}
	}
	canonical, err := load()
	if err != nil {
		return nil, err
	}
	if p.Version != canonical.Version || p.Digest != canonical.Digest || p.ArchiveDigest != canonical.ArchiveDigest {
		return nil, fmt.Errorf("source manifest changed")
	}
	original := make(map[string]Row, len(canonical.Areas))
	for _, row := range canonical.Areas {
		original[row.SourceKey] = row
	}
	countryIDs := map[string]bool{}
	for key := range codes {
		id := countries[key]
		if !foundation.ValidID(id) || countryIDs[id] {
			return nil, fmt.Errorf("missing or ambiguous country identity")
		}
		countryIDs[id] = true
	}
	ids := map[string]string{}
	counts := map[string]int{}
	for _, r := range p.Areas {
		if row, ok := original[r.SourceKey]; !ok || row != r {
			return nil, fmt.Errorf("source row changed: %s", r.SourceKey)
		}
		if codes[r.CountryKey] == "" || !foundation.ValidID(countries[r.CountryKey]) || r.SourceKey == "" || r.SourceVersion == "" || r.Name == "" {
			return nil, fmt.Errorf("invalid source row %s", r.SourceKey)
		}
		if _, ok := ids[r.SourceKey]; ok {
			return nil, fmt.Errorf("duplicate source key %s", r.SourceKey)
		}
		ids[r.SourceKey] = objectID(instance, r.Scheme, r.SourceKey)
		counts[r.CountryKey]++
	}
	for key, required := range expected {
		if counts[key] != required {
			return nil, fmt.Errorf("incomplete fixed area region %s", key)
		}
	}
	nodes := make([]d.Node, 0, len(p.Areas))
	prepared := map[string]Prepared{}
	for _, r := range p.Areas {
		parent := ""
		if r.ParentKey != "" {
			var ok bool
			parent, ok = ids[r.ParentKey]
			if !ok {
				return nil, fmt.Errorf("missing parent %s", r.ParentKey)
			}
		}
		n := d.Node{ID: ids[r.SourceKey], EffectiveDate: r.EffectiveDate, Structure: d.Structure{CountryID: countries[r.CountryKey], CountryCode: codes[r.CountryKey], Scheme: r.Scheme, Code: r.Code, Nature: r.Nature, Level: r.Level, ParentID: parent}}
		nodes = append(nodes, n)
		prepared[n.ID] = Prepared{Node: n, Name: r.Name, SourceKey: r.SourceKey, SourceVersion: r.SourceVersion}
	}
	order, e := d.ValidateTree(nodes)
	if e != nil {
		return nil, e
	}
	out := make([]Prepared, 0, len(order))
	for _, id := range order {
		out = append(out, prepared[id])
	}
	return out, nil
}

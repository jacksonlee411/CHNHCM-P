package geoareaseed

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"chnhcm.local/chnhcm/internal/foundation"
)

//go:embed macau.json
var macauFacts []byte

const macauVersion = "chnhcm-macau-20261119-v1"
const macauDigest = "45ed3eb29cda00a3f8699e0faab5adb40fc570b5e86d4b066f929180a75a7373"

// Macau binds the published names to local identities. Neither provider assigns
// these keys as official codes, or supplies a postal-region-to-parish hierarchy.
func Macau() (Package, error) {
	var facts struct {
		Version string
		Sources []struct {
			Scheme, Level, ResponseSHA256 string
			Entries                       []struct{ Key, Name string }
		}
	}
	sum := sha256.Sum256(macauFacts)
	if hex.EncodeToString(sum[:]) != macauDigest {
		return Package{}, fmt.Errorf("Macau source snapshot differs")
	}
	if err := json.Unmarshal(macauFacts, &facts); err != nil {
		return Package{}, err
	}
	p := Package{Version: facts.Version, ArchiveDigest: macauDigest}
	for _, source := range facts.Sources {
		for _, entry := range source.Entries {
			p.Areas = append(p.Areas, Row{CountryKey: "base.mo", EffectiveDate: foundation.DefaultEffectiveDate,
				Name: entry.Name, Scheme: source.Scheme, Level: source.Level, Nature: "address_region",
				SourceKey: "mo:" + source.Level + ":" + entry.Key, SourceVersion: source.ResponseSHA256})
		}
	}
	rows, err := json.Marshal(p.Areas)
	if err != nil {
		return Package{}, err
	}
	payload := sha256.Sum256(rows)
	p.Digest = hex.EncodeToString(payload[:])
	return p, nil
}

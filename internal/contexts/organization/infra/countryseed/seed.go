// Package countryseed is the fixed offline G0 country input. It has no runtime
// dependency on CubeBox, Odoo, adjacent repositories, or a network service.
package countryseed

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"

	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

//go:embed countries.json
var fixed []byte

type Row struct {
	Code      string `json:"code"`
	Date      string `json:"effectiveDate"`
	Name      string `json:"name"`
	NameEn    string `json:"nameEn"`
	PhoneCode int    `json:"phoneCode"`
	Scheme    string `json:"scheme"`
	SourceKey string `json:"sourceKey"`
}
type Package struct {
	Version       string `json:"version"`
	Digest        string `json:"packageDigest"`
	ArchiveDigest string `json:"archiveDigest"`
	Countries     []Row  `json:"countries"`
}

func Fixed() (Package, error) {
	var p Package
	decoder := json.NewDecoder(bytes.NewReader(fixed))
	decoder.DisallowUnknownFields()
	if e := decoder.Decode(&p); e != nil {
		return p, e
	}
	if len(p.Countries) != 251 || p.Version != "chnhcm-geography-g0-v1" || p.Digest != "dc7eac11535bb2da2f3f72431d0767dc510c250a3e11596ffd76428e3fafc123" || p.ArchiveDigest != "8ffa0c5f633d6f5706dcff2bef7d86bf3af44cdbb7bcd76e638e5a2733553ad7" {
		return p, fmt.Errorf("fixed country source manifest differs")
	}
	seen := map[string]bool{}
	for _, r := range p.Countries {
		if seen[r.SourceKey] || r.Scheme != "odoo-country" || r.SourceKey == "" || !foundation.ValidDate(r.Date) {
			return p, fmt.Errorf("invalid fixed country source: %s", r.SourceKey)
		}
		seen[r.SourceKey] = true
	}
	return p, nil
}

// UUIDs are namespaced by instance and provider key, never by display name,
// list position or a source database sequence. Reopening keeps object identity.
func stableID(parts ...string) string {
	b, _ := json.Marshal(parts)
	namespace, _ := hex.DecodeString("6ba7b8109dad11d180b400c04fd430c8")
	h := sha1.Sum(append(namespace, b...)) // UUID v5, not an integrity digest.
	h[6] = (h[6] & 15) | 0x50
	h[8] = (h[8] & 63) | 0x80
	s := hex.EncodeToString(h[:16])
	return s[:8] + "-" + s[8:12] + "-" + s[12:16] + "-" + s[16:20] + "-" + s[20:]
}
func (p Package) Command(instance, subject string, r Row) (org.CountryCommand, error) {
	if !foundation.ValidID(instance) || !foundation.ValidID(subject) {
		return org.CountryCommand{}, fmt.Errorf("instance and subject must be complete IDs")
	}
	var english, phone *string
	if r.NameEn != "" {
		v := r.NameEn
		english = &v
	}
	if r.PhoneCode != 0 {
		v := strconv.Itoa(r.PhoneCode)
		phone = &v
	}
	input := org.CountryImport{ID: stableID("organization.country", instance, r.Scheme, r.SourceKey), Origin: org.CountryOrigin{Scheme: r.Scheme, Key: r.SourceKey, Code: r.Code, Name: r.Name, NameEn: r.NameEn, PackageVersion: p.Version, PackageDigest: p.Digest, ArchiveDigest: p.ArchiveDigest}, EffectiveDate: r.Date, Name: r.Name, NameEn: english, PhoneCode: phone}
	b, _ := json.Marshal(input)
	hash := sha256.Sum256(b)
	return org.CountryCommand{Action: "CREATE", SubmissionID: stableID("organization.country.import", instance, subject, p.Digest, r.SourceKey), Digest: hash[:], Import: input}, nil
}

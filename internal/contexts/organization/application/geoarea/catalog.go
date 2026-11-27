package geoarea

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/geoarea"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"slices"
)

type CatalogPublication struct {
	CountryID, CountryCode, Version, Digest, ArchiveDigest string
	Count                                                  int
}
type Mapping struct {
	ID, Date string
	Origin   d.Origin
}

func (s Service) InitializeCatalog(ctx context.Context, c access.Credentials, p CatalogPublication, commands []org.GeoAreaCommand) error {
	if p.Count != len(commands) || p.Count != map[string]int{"CN": 3209, "HK": 21, "TW": 390, "MO": 10}[p.CountryCode] || p.Count == 0 {
		return fault("M02_INVALID_INPUT", "", "区域目录包不完整")
	}
	expected := make([]Mapping, 0, len(commands))
	nodes := []d.Node{}
	byID := map[string]org.GeoAreaCommand{}
	for _, cmd := range commands {
		o := origin(cmd.Import.Origin)
		if cmd.Action != "CREATE" || o.CountryID != p.CountryID || o.CountryCode != p.CountryCode || o.PackageDigest != p.Digest || o.ArchiveDigest != p.ArchiveDigest || o.PackageVersion != p.Version || cmd.Import.EffectiveDate != foundation.DefaultEffectiveDate {
			return fault("M02_INVALID_INPUT", "", "区域目录来源不一致")
		}
		nodes = append(nodes, d.Node{ID: cmd.Import.ID, EffectiveDate: cmd.Import.EffectiveDate, Structure: o.Structure})
		byID[cmd.Import.ID] = cmd
		expected = append(expected, Mapping{ID: cmd.Import.ID, Date: cmd.Import.EffectiveDate, Origin: o})
	}
	order, e := d.ValidateTree(nodes)
	if e != nil {
		return e
	}
	check := func(u Unit) error {
		// This deployment command targets a known package. Report missing
		// authority separately from a missing business object or source.
		for _, action := range []string{"read", "history.read"} {
			if _, e := require(ctx, u, action, nil); e != nil {
				return e
			}
		}
		_, e := require(ctx, u, "create", []string{"code", "scheme", "nature", "level", "countryRegionId", "parentAreaId", "name", "nameEn", "effectiveDate", "operationType", "operationReason"})
		return e
	}
	complete := false
	existing := []Mapping{}
	if e = s.read(ctx, c, func(u Unit) error {
		if e := check(u); e != nil {
			return e
		}
		var e error
		complete, e = u.CatalogComplete(ctx, p)
		if e != nil || complete {
			return e
		}
		existing, e = u.Mappings(ctx, p.CountryID)
		return e
	}); e != nil || complete {
		return e
	}
	want := map[string]Mapping{}
	for _, m := range expected {
		want[m.ID] = m
	}
	seen := map[string]bool{}
	for _, m := range existing {
		if want[m.ID] != m {
			return fault("M02_SOURCE_CONFLICT", "", "已有区域初始化映射与固定包不一致")
		}
		seen[m.ID] = true
	}
	for _, id := range order {
		if seen[id] {
			continue
		}
		cmd := byID[id]
		// A closed not_applied submission is immutable. Its next deterministic ID
		// is the same on every process restart and concurrent initializer.
		for {
			closed := false
			if e = s.read(ctx, c, func(u Unit) error {
				if e := check(u); e != nil {
					return e
				}
				v, e := u.FindResult(ctx, cmd.SubmissionID)
				if e != nil {
					return e
				}
				if v != nil {
					if v.DecodeError != nil {
						return v.DecodeError
					}
					closed = v.Result.Outcome == "not_applied"
				}
				return nil
			}); e != nil {
				return e
			}
			if !closed {
				break
			}
			cmd.SubmissionID = stableSubmissionID("geoarea.resume:" + cmd.SubmissionID)
		}
		if _, e = s.maintain(ctx, c, cmd); e != nil {
			return e
		}
	}
	return s.Store.WithinGeoAreaWrite(ctx, c, func(u Unit) error {
		if e := check(u); e != nil {
			return e
		}
		if e := u.PublishCatalog(ctx, p, expected); e != nil {
			return e
		}
		u.Confirm()
		return nil
	})
}

func ImportCommand(instance string, m Mapping, name string) org.GeoAreaCommand {
	in := org.GeoAreaImport{ID: m.ID, EffectiveDate: m.Date, Name: name, Origin: org.GeoAreaOrigin{GeoAreaStructure: org.GeoAreaStructure(m.Origin.Structure), Key: m.Origin.Key, SourceVersion: m.Origin.SourceVersion, PackageVersion: m.Origin.PackageVersion, PackageDigest: m.Origin.PackageDigest, ArchiveDigest: m.Origin.ArchiveDigest}}
	b, _ := json.Marshal(in)
	hash := sha256.Sum256(b)
	return org.GeoAreaCommand{Action: "CREATE", Import: in, SubmissionID: stableSubmissionID("geoarea.initialize:" + instance + ":" + m.ID + ":" + m.Origin.PackageDigest), Digest: slices.Clone(hash[:])}
}

// UUIDv5 DNS namespace; deterministic submission identity has no external effect.
func stableSubmissionID(name string) string {
	namespace := []byte{0x6b, 0xa7, 0xb8, 0x10, 0x9d, 0xad, 0x11, 0xd1, 0x80, 0xb4, 0x00, 0xc0, 0x4f, 0xd4, 0x30, 0xc8}
	hash := sha1.New()
	_, _ = hash.Write(namespace)
	_, _ = hash.Write([]byte(name))
	b := hash.Sum(nil)[:16]
	b[6] = (b[6] & 0x0f) | 0x50
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:])
}

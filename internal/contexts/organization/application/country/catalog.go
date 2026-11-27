package country

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
)

// CatalogPublication records completion of the fixed source package. It is not
// an import job framework; each country is committed by the normal service.
type CatalogPublication struct {
	Version, Digest, ArchiveDigest string
	Count                          int
}

func (s Service) InitializeCatalog(ctx context.Context, c access.Credentials, publication CatalogPublication, commands []org.CountryCommand) error {
	if publication.Count != 251 || len(commands) != publication.Count {
		return fault("M02_INVALID_INPUT", "", "国家目录包不完整")
	}
	check := func(u Unit) error {
		// This deployment command targets a known package. Report missing
		// authority separately from a missing business object or source.
		for _, action := range []string{"read", "history.read"} {
			if _, e := require(ctx, u, action, nil); e != nil {
				return e
			}
		}
		_, e := require(ctx, u, "create", []string{"code", "name", "nameEn", "phoneCode", "effectiveDate", "operationType", "operationReason"})
		return e
	}
	complete := false
	if e := s.read(ctx, c, func(u Unit) error {
		if e := check(u); e != nil {
			return e
		}
		var e error
		complete, e = u.CatalogComplete(ctx, publication)
		return e
	}); e != nil {
		return e
	}
	if complete {
		return nil
	}
	for _, cmd := range commands {
		if cmd.Action != "CREATE" || cmd.Import.Origin.PackageDigest != publication.Digest || cmd.Import.Origin.PackageVersion != publication.Version || cmd.Import.Origin.ArchiveDigest != publication.ArchiveDigest {
			return fault("M02_INVALID_INPUT", "", "国家目录来源不一致")
		}
		exists := false
		if e := s.read(ctx, c, func(u Unit) error {
			if e := check(u); e != nil {
				return e
			}
			state, e := u.Load(ctx, cmd.Import.ID, "")
			if e != nil {
				return e
			}
			exists = state.Timeline.ID != ""
			if exists && state.Timeline.Origin != origin(cmd.Import.Origin) {
				return fault("M02_SOURCE_CONFLICT", "", "已有国家来源与固定包不一致")
			}

			if !exists {
				saved, e := u.FindResult(ctx, cmd.SubmissionID)
				if e != nil {
					return e
				}
				if saved != nil && saved.DecodeError != nil {
					return saved.DecodeError
				}
				if saved != nil && saved.Result.Outcome == "not_applied" {
					cmd.SubmissionID = foundation.NewID()
				}
			}
			return nil
		}); e != nil {
			return e
		}
		if exists {
			continue
		} // Preserve all manual display maintenance on restart.
		result, e := s.Maintain(ctx, c, cmd)
		if e != nil {
			return e
		}
		if result.Outcome != "accepted" {
			return fault("M02_SOURCE_CONFLICT", "", "目录来源未完成，请核对原提交")
		}
	}
	return s.Store.WithinCountryWrite(ctx, c, func(u Unit) error {
		if e := check(u); e != nil {
			return e
		}
		for _, cmd := range commands {
			state, e := u.Load(ctx, cmd.Import.ID, "")
			if e != nil {
				return e
			}
			if state.Timeline.ID != cmd.Import.ID || state.Timeline.Origin != origin(cmd.Import.Origin) {
				return fault("M02_SOURCE_CONFLICT", "", "国家目录来源核对失败")
			}
		}
		if e := u.PublishCatalog(ctx, publication); e != nil {
			return e
		}
		u.Confirm()
		return nil
	})
}

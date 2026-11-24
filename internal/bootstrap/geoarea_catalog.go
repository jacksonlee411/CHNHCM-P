package bootstrap

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/geoarea"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/geoarea"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/countryseed"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/geoareaseed"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
)

func (r *Runtime) InitializeGeoAreaCatalog(ctx context.Context, login access.LoginInput) error {
	p, e := geoareaseed.Fixed()
	if e != nil {
		return e
	}
	session, e := r.Access.Login(ctx, login, foundation.NewID())
	if e != nil {
		return e
	}
	c := access.Credentials{Token: session.Token, CSRF: session.Principal.CSRF, Write: true}
	defer r.Access.Logout(context.WithoutCancel(ctx), c)
	countries, e := countryseed.Fixed()
	if e != nil {
		return e
	}
	ids := map[string]string{}
	for _, row := range countries.Countries {
		cmd, e := countries.Command(r.Instance, session.Principal.ID, row)
		if e != nil {
			return e
		}
		ids[row.SourceKey] = cmd.Import.ID
	}
	mo, e := geoareaseed.Macau()
	if e != nil {
		return e
	}
	if e = r.publishAreaPackage(ctx, c, p, ids, []string{"HK", "CN", "TW"}); e != nil {
		return e
	}
	return r.publishAreaPackage(ctx, c, mo, ids, []string{"MO"})
}

func (r *Runtime) publishAreaPackage(ctx context.Context, c access.Credentials, p geoareaseed.Package, ids map[string]string, codes []string) error {
	prepared, e := p.Prepare(r.Instance, ids)
	if e != nil {
		return e
	}
	for _, code := range codes {
		commands := []org.GeoAreaCommand{}
		country := ""
		for _, row := range prepared {
			if row.Node.Structure.CountryCode != code {
				continue
			}
			country = row.Node.Structure.CountryID
			m := app.Mapping{ID: row.Node.ID, Date: row.Node.EffectiveDate, Origin: d.Origin{Structure: row.Node.Structure, Key: row.SourceKey, SourceVersion: row.SourceVersion, PackageVersion: p.Version, PackageDigest: p.Digest, ArchiveDigest: p.ArchiveDigest}}
			commands = append(commands, app.ImportCommand(r.Instance, m, row.Name))
		}
		if e = r.GeoArea.(app.Service).InitializeCatalog(ctx, c, app.CatalogPublication{CountryID: country, CountryCode: code, Version: p.Version, Digest: p.Digest, ArchiveDigest: p.ArchiveDigest, Count: len(commands)}, commands); e != nil {
			return e
		}
	}
	return nil
}

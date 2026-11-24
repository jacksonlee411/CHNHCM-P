package bootstrap

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/country"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/countryseed"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
)

// InitializeCountryCatalog logs in with ordinary credentials and uses the same
// authorization, write service and submission recovery as interactive commands.
func (r *Runtime) InitializeCountryCatalog(ctx context.Context, login access.LoginInput) error {
	p, e := countryseed.Fixed()
	if e != nil {
		return e
	}
	session, e := r.Access.Login(ctx, login, foundation.NewID())
	if e != nil {
		return e
	}
	c := access.Credentials{Token: session.Token, CSRF: session.Principal.CSRF, Write: true}
	defer r.Access.Logout(context.WithoutCancel(ctx), c)
	commands := make([]org.CountryCommand, 0, len(p.Countries))
	for _, row := range p.Countries {
		cmd, e := p.Command(r.Instance, session.Principal.ID, row)
		if e != nil {
			return e
		}
		commands = append(commands, cmd)
	}
	return r.Country.(app.Service).InitializeCatalog(ctx, c, app.CatalogPublication{Version: p.Version, Digest: p.Digest, ArchiveDigest: p.ArchiveDigest, Count: len(commands)}, commands)
}

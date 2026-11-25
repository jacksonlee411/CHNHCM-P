package bootstrap

import (
	"context"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	f "chnhcm.local/chnhcm/internal/foundation"
)

// The upgrader uses the runtime role and an ordinary operator session.
func (r *Runtime) ReconcileProbation(ctx context.Context, login access.LoginInput, apply bool) (h.ProbationRepairReport, error) {
	session, err := r.Access.Login(ctx, login, f.NewID())
	if err != nil {
		return h.ProbationRepairReport{}, err
	}
	c := f.Credentials{Token: session.Token, CSRF: session.Principal.CSRF, Write: true}
	defer r.Access.Logout(context.WithoutCancel(ctx), c)
	return r.Hires.ReconcileProbation(ctx, c, apply)
}

//go:build integration

package bootstrap

import (
	accessdb "chnhcm.local/chnhcm/internal/contexts/access/infra/postgres"
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	orgdb "chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
)

// ImportIntegrationDepartmentName exercises the non-HTTP import application port.
// Only integration builds contain this fixture bridge.
func ImportIntegrationDepartmentName(ctx context.Context, r *Runtime, c foundation.Credentials, cmd org.DepartmentCommand, original string) (org.DepartmentResult, error) {
	a := &accessdb.Store{Pool: r.Pool, Instance: r.Instance}
	s := &orgdb.Store{Pool: r.Pool, Instance: r.Instance, Guard: a.Guard}
	return (app.Service{Store: s}).ExecuteHistoricalName(ctx, c, cmd, original)
}

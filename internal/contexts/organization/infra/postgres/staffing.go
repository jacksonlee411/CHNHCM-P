package postgres

import (
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
	"context"
)

func (s *Store) StaffingEnterprise(ctx context.Context, lock bool) (org.Enterprise, error) {
	tx, ok := dbtx.From(ctx)
	if !ok {
		return org.Enterprise{}, foundation.Fail(503, "M02_DEPENDENCY_UNAVAILABLE", "企业共同事务不可用")
	}
	s.historyCatalog.Enter()
	defer s.historyCatalog.Leave()
	u := &unit{tx: tx, instance: s.Instance, catalog: &s.historyCatalog}
	return u.Enterprise(ctx, lock)
}

package postgres

import (
	"context"
	"errors"
	"time"

	"chnhcm.local/chnhcm/internal/contexts/organization/application"
	"chnhcm.local/chnhcm/internal/contexts/organization/domain"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres/generated"
	"chnhcm.local/chnhcm/internal/foundation"
)

func (u *unit) LockUnchangedBUCoverage(ctx context.Context, base application.Timeline, candidate domain.Candidate) (application.Timeline, error) {
	if !domain.SameBUCoverage(base.Records, candidate.Records) {
		return application.Timeline{}, errors.New("BU unchanged coverage proof missing")
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	ent, err := u.Enterprise(ctx, true)
	if err != nil {
		return application.Timeline{}, err
	}
	locked := foundation.BeginStage(ctx, "guard.bu_wait")
	_, err = u.q.LockDepartmentBUWrite(ctx, generated.LockDepartmentBUWriteParams{InstanceID: u.instance, Column2: []string{base.BUID}})
	locked()
	if err != nil {
		return application.Timeline{}, err
	}
	locked = foundation.BeginStage(ctx, "guard.relation_wait")
	_, err = u.tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", administrativeGuardKey(u.instance, ent.ID, base.BUID))
	locked()
	if err != nil {
		return application.Timeline{}, err
	}
	// The source is reread after both existing guards. The application rechecks
	// its expected source and recomputes the candidate; it never extends this
	// lock set after discovering a changed source.
	current, err := u.LoadTimeline(ctx, base.BUID, false)
	if err != nil {
		return current, err
	}
	u.stableBUCoverage = &current
	return current, nil
}

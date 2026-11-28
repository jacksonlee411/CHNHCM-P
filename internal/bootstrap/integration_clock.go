//go:build integration

package bootstrap

import (
	compdb "chnhcm.local/chnhcm/internal/contexts/compensation/infra/postgres"
	"context"
	"time"

	"chnhcm.local/chnhcm/internal/contexts/organization/application"
	orgdb "chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres"
	"github.com/jackc/pgx/v5"
)

// SetIntegrationClock installs a fixture-owned clock through the composition root.
// It is absent from production builds and never exposed through an HTTP route.
// The caller must own the runtime and restore it after in-flight requests finish.
func SetIntegrationClock(runtime *Runtime, clock func() time.Time) (restore func()) {
	store := runtime.Organization.(application.Service).Store.(*orgdb.Store)
	original := store.Clock
	store.Clock = func(context.Context, pgx.Tx) (time.Time, error) { return clock(), nil }
	return func() { store.Clock = original }
}

// SetCompensationIntegrationClock drives the business date independently of audit time.
func SetCompensationIntegrationClock(runtime *Runtime, clock func() time.Time) func() {
	store := runtime.Compensation.(*compdb.Store)
	original := store.Clock
	store.Clock = clock
	return func() { store.Clock = original }
}

// SetIntegrationRecordedClock controls only persisted audit instants. Business
// dates continue to use their separate request clock.
func SetIntegrationRecordedClock(runtime *Runtime, clock func() time.Time) func() {
	store := runtime.Organization.(application.Service).Store.(*orgdb.Store)
	original := store.RecordedClock
	store.RecordedClock = func(context.Context, pgx.Tx) (time.Time, error) { return clock(), nil }
	return func() { store.RecordedClock = original }
}

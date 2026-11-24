package vankeverify

import (
	"context"
	"errors"
	"os"
	"strconv"

	"chnhcm.local/chnhcm/internal/bootstrap"
	accessapp "chnhcm.local/chnhcm/internal/contexts/access/application"
	accessdb "chnhcm.local/chnhcm/internal/contexts/access/infra/postgres"
	orgapp "chnhcm.local/chnhcm/internal/contexts/organization/application"
	departmentapp "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	orgdb "chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

// importPool is restricted to the ownership-checked temporary or retained case. Normal
// servers and all capacity measurements keep bootstrap.APIConnectionLimit.
// It changes connection parallelism only, retaining the same application ports,
// authentication, transaction guards, deadlines and durable database settings.
func importPool(ctx context.Context, current *bootstrap.Runtime, profile string) (*bootstrap.Runtime, error) {
	value := os.Getenv("VANKE127_IMPORT_CONNECTIONS")
	if value == "" {
		return current, nil
	}
	limit, err := strconv.Atoi(value)
	if err != nil || limit < int(bootstrap.APIConnectionLimit) || limit > 64 || (profile != "" && !persistentImportProfile(profile)) {
		current.Close()
		return nil, errors.New("import pool requires an owned Vanke case and 12..64 connections")
	}
	config := current.Pool.Config()
	instance := current.Instance
	current.Close()
	config.MaxConns = int32(limit)
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, errors.New("import connection pool unavailable")
	}
	access := &accessdb.Store{Pool: pool, Instance: instance}
	organization := &orgdb.Store{Pool: pool, Instance: instance, Guard: access.Guard}
	runtime := &bootstrap.Runtime{Pool: pool, Instance: instance, Access: accessapp.Service{Port: access}, Organization: orgapp.Service{Store: organization}, Department: departmentapp.Service{Store: organization}}
	if err = runtime.Ready(ctx); err != nil {
		runtime.Close()
		return nil, err
	}
	return runtime, nil
}

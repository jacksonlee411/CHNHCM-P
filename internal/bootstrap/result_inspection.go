package bootstrap

import (
	"context"

	app "chnhcm.local/chnhcm/internal/contexts/organization/application"
	department "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	"chnhcm.local/chnhcm/internal/platform/compact"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func inspectOperation(ctx context.Context, pool *pgxpool.Pool, kind history.Kind, instance, subject, submission string) (compact.Values, []byte, error) {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return compact.Values{}, nil, err
	}
	defer tx.Rollback(context.Background())
	found, err := history.FindSubmission(ctx, tx, kind, instance, subject, submission)
	if err != nil {
		return compact.Values{}, nil, err
	}
	var values compact.Values
	var raw []byte
	if found == nil {
		values, raw, err = history.LoadTerminal(ctx, tx, kind, instance, subject, submission)
	} else {
		var chunk history.Chunk
		chunk, err = history.LoadChunk(ctx, tx, kind, instance, found.Object, found.Chunk)
		if err == nil {
			values = chunk.Values
			raw, err = history.LoadShape(ctx, tx, kind, instance, chunk.Enterprise, chunk.Operations[found.Slot])
		}
	}
	if err != nil {
		return values, nil, err
	}
	return values, raw, tx.Commit(ctx)
}

// InspectBUResult exports stored facts for privileged verification; it never
// substitutes for the currently authorized business submission/resolve route.
func InspectBUResult(ctx context.Context, pool *pgxpool.Pool, instance, subject, submission string) (app.Terminal, error) {
	values, raw, err := inspectOperation(ctx, pool, history.BusinessUnit, instance, subject, submission)
	if err != nil {
		return app.Terminal{}, err
	}
	op, err := history.DecodeBUOperation(ctx, values, raw)
	return op.Terminal, err
}

func InspectDepartmentResult(ctx context.Context, pool *pgxpool.Pool, instance, subject, submission string) (department.Terminal, error) {
	values, raw, err := inspectOperation(ctx, pool, history.Department, instance, subject, submission)
	if err != nil {
		return department.Terminal{}, err
	}
	op, err := history.DecodeDepartmentOperation(ctx, values, raw)
	return op.Terminal, err
}

//go:build integration

package bootstrap

import (
	app "chnhcm.local/chnhcm/internal/contexts/organization/application"
	dapp "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	"chnhcm.local/chnhcm/internal/contexts/organization/domain"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	orgdb "chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres"
	"chnhcm.local/chnhcm/internal/platform/compact"
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
)

type IntegrationBURevision = domain.Record
type IntegrationDepartmentRevision = d.Record
type IntegrationBUTerminal = app.Terminal
type IntegrationDepartmentTerminal = dapp.Terminal

// MutateIntegrationBU and MutateIntegrationDepartment are deliberate corruption
// fixtures, not migration APIs. Their admin pool must belong to a disposable DB.
func MutateIntegrationBU(ctx context.Context, pool *pgxpool.Pool, runtime *Runtime, object, source string, edit func(*BUHistoryFact)) error {
	tx, e := pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	e = history.MutateIntegrationSource(ctx, tx, history.BusinessUnit, runtime.Instance, object, source, func(v compact.Values, state, op, proof []byte) (history.Encoder, error) {
		a, e := history.DecodeBUState(ctx, v, state)
		if e != nil {
			return nil, e
		}
		b, e := history.DecodeBUOperation(ctx, v, op)
		if e != nil {
			return nil, e
		}
		fact := BUHistoryFact{Instance: runtime.Instance, Object: object, State: a, Operation: b}
		edit(&fact)
		fact.State.Timeline.Code = ""
		return func(p *compact.Pool) ([]byte, []byte, []byte, error) {
			a, e := history.EncodeBUState(ctx, p, fact.State)
			if e != nil {
				return nil, nil, nil, e
			}
			b, e := history.EncodeBUOperation(ctx, p, fact.Operation)
			return a, b, proof, e
		}, nil
	})
	if e != nil {
		return e
	}
	if e = tx.Commit(ctx); e != nil {
		return e
	}
	runtime.Organization.(app.Service).Store.(*orgdb.Store).ResetIntegrationCaches()
	return nil
}
func MutateIntegrationDepartment(ctx context.Context, pool *pgxpool.Pool, runtime *Runtime, object, source string, edit func(*DepartmentHistoryFact)) error {
	tx, e := pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	e = history.MutateIntegrationSource(ctx, tx, history.Department, runtime.Instance, object, source, func(v compact.Values, state, op, proof []byte) (history.Encoder, error) {
		a, e := history.DecodeDepartmentState(ctx, v, state)
		if e != nil {
			return nil, e
		}
		b, e := history.DecodeDepartmentOperation(ctx, v, op)
		if e != nil {
			return nil, e
		}
		c, e := history.DecodeDepartmentEvidence(ctx, v, proof)
		if e != nil {
			return nil, e
		}
		fact := DepartmentHistoryFact{Instance: runtime.Instance, Object: object, State: a, Operation: b, Evidence: c}
		edit(&fact)
		fact.State.Timeline.Code = ""
		return func(p *compact.Pool) ([]byte, []byte, []byte, error) {
			a, e := history.EncodeDepartmentState(ctx, p, fact.State)
			if e != nil {
				return nil, nil, nil, e
			}
			b, e := history.EncodeDepartmentOperation(ctx, p, fact.Operation)
			if e != nil {
				return nil, nil, nil, e
			}
			c, e := history.EncodeDepartmentEvidence(ctx, p, fact.Evidence)
			return a, b, c, e
		}, nil
	})
	if e != nil {
		return e
	}
	if e = tx.Commit(ctx); e != nil {
		return e
	}
	runtime.Organization.(app.Service).Store.(*orgdb.Store).ResetIntegrationCaches()
	return nil
}
func MutateIntegrationBUResult(ctx context.Context, pool *pgxpool.Pool, runtime *Runtime, subject, submission string, edit func(*app.Terminal)) error {
	tx, e := pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	e = history.MutateIntegrationTerminal(ctx, tx, history.BusinessUnit, runtime.Instance, subject, submission, func(v compact.Values, raw []byte) ([]byte, compact.Values, error) {
		op, e := history.DecodeBUOperation(ctx, v, raw)
		if e != nil {
			return nil, v, e
		}
		edit(&op.Terminal)
		p := compact.NewPool(v)
		raw, e = history.EncodeBUOperation(ctx, p, op)
		return raw, p.Values, e
	})
	if e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func MutateIntegrationDepartmentResult(ctx context.Context, pool *pgxpool.Pool, runtime *Runtime, subject, submission string, edit func(*dapp.Terminal)) error {
	tx, e := pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	e = history.MutateIntegrationTerminal(ctx, tx, history.Department, runtime.Instance, subject, submission, func(v compact.Values, raw []byte) ([]byte, compact.Values, error) {
		op, e := history.DecodeDepartmentOperation(ctx, v, raw)
		if e != nil {
			return nil, v, e
		}
		edit(&op.Terminal)
		p := compact.NewPool(v)
		raw, e = history.EncodeDepartmentOperation(ctx, p, op)
		return raw, p.Values, e
	})
	if e != nil {
		return e
	}
	return tx.Commit(ctx)
}

//go:build integration

package bootstrap_test

import (
	"context"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"chnhcm.local/chnhcm/internal/bootstrap"
	accessapp "chnhcm.local/chnhcm/internal/contexts/access/application"
	accessdb "chnhcm.local/chnhcm/internal/contexts/access/infra/postgres"
	orgapp "chnhcm.local/chnhcm/internal/contexts/organization/application"
	depapp "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	orgdb "chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres"
	"github.com/jackc/pgx/v5"
)

type diagnostic133Tracer struct{ calls atomic.Int64 }

func (o *diagnostic133Tracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryStartData) context.Context {
	o.calls.Add(1)
	return ctx
}
func (*diagnostic133Tracer) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

// The observer must share the existing budget with access, both object adapters
// and cleanup; replacing their pool after cleanup starts is not safe.
func TestDiagnostic133SinglePoolWithConcurrentCleanup(t *testing.T) {
	f := fresh(t)
	dsn := f.runtime.Pool.Config().ConnString()
	f.server.Close()
	f.runtime.Close()
	observer := &diagnostic133Tracer{}
	runtime, err := bootstrap.OpenForDiagnostics(f.ctx, dsn, f.instance, observer)
	must(t, err)
	t.Cleanup(runtime.Close)
	f.runtime = runtime
	api := &bootstrap.API{Runtime: runtime, InsecureLocal: true}
	f.server = httptest.NewUnstartedServer(nil)
	api.Origin = "http://" + f.server.Listener.Addr().String()
	f.server.Config.Handler = api.Handler()
	f.server.Start()
	t.Cleanup(f.server.Close)
	organization := runtime.Organization.(orgapp.Service).Store.(*orgdb.Store)
	department := runtime.Department.(depapp.Service).Store.(*orgdb.Store)
	access := runtime.Access.(accessapp.Service).Port.(*accessdb.Store)
	if runtime.Pool.Config().MaxConns != 12 || organization.Pool != runtime.Pool || department.Pool != runtime.Pool || access.Pool != runtime.Pool {
		t.Fatal("diagnostics must not create an additional API connection budget")
	}
	b := f.login("admin", f.password)
	before := observer.calls.Load()
	done := make(chan error, 1)
	go func() {
		for range 20 {
			if err := organization.SweepDrafts(f.ctx); err != nil {
				done <- err
				return
			}
		}
		done <- nil
	}()
	for range 10 {
		b.call("GET", "/api/v1/business-units", nil, "", 200)
		b.call("GET", "/api/v1/departments", nil, "", 200)
	}
	must(t, <-done)
	if observer.calls.Load() <= before {
		t.Fatal("real reads and cleanup bypassed the installed observer")
	}
	if runtime.Pool.Stat().TotalConns() > 12 {
		t.Fatal("diagnostics exceeded the original pool budget")
	}
}

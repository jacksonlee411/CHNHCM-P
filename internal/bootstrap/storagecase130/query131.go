package storagecase130

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"chnhcm.local/chnhcm/internal/bootstrap"
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	dep "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5/pgconn"
)

// Verification-only observation of the real capture port. No facts, policies,
// identifiers, labels or credentials are logged; the business server does not
// import this package. The opt-in file must be new and outside the checkout.
type query131Route struct{}
type query131Request struct{}
type query131Store struct {
	dep.Store
	write func(any)
}
type query131Unit struct {
	dep.Unit
	write func(any)
	reads *query131Reads
}

type query131Reads struct {
	planned atomic.Int64
	batched atomic.Int64
	direct  atomic.Int64
}

type query131Guard struct {
	access.Guard
	reads *query131Reads
}

func (g query131Guard) Decide(ctx context.Context, in access.Intent) (access.Decision, error) {
	g.reads.direct.Add(1)
	return g.Guard.Decide(ctx, in)
}

func (g query131Guard) DecideReads(ctx context.Context, in []access.ReadIntent) (access.ReadDecisions, error) {
	g.reads.batched.Add(int64(len(in)))
	return g.Guard.DecideReads(ctx, in)
}

type query131Planner struct {
	query131Guard
	access.ReadPlanner
}

func (g query131Planner) DecidePlan(ctx context.Context, plan access.ReadPlan, in access.ReadIntent) (access.Decision, error) {
	g.reads.planned.Add(1)
	return g.ReadPlanner.DecidePlan(ctx, plan, in)
}

func (u query131Unit) Access() access.Guard {
	guard := query131Guard{Guard: u.Unit.Access(), reads: u.reads}
	if planner, ok := guard.Guard.(access.ReadPlanner); ok {
		// PlanReads remains the original method: its origin is the guarded
		// transaction's real planner, as required by CaptureQuery/Parents.
		return query131Planner{query131Guard: guard, ReadPlanner: planner}
	}
	return guard
}

func (s query131Store) WithinDepartment(ctx context.Context, c access.Credentials, fn func(dep.Unit) error) error {
	reads := &query131Reads{}
	records := []map[string]any{}
	err := s.Store.WithinDepartment(ctx, c, func(u dep.Unit) error {
		return fn(query131Unit{Unit: u, write: func(value any) {
			records = append(records, value.(map[string]any))
		}, reads: reads})
	})
	if len(records) == 0 {
		records = append(records, map[string]any{"Route": ctx.Value(query131Route{}), "Observation": "authorization-inputs", "Failed": err != nil})
	}
	for _, value := range records {
		// Offline diagnosis only: preserve the first cause without SQL, values or identifiers.
		if err != nil {
			types := []string{}
			for cause := err; cause != nil && len(types) < 8; cause = errors.Unwrap(cause) {
				types = append(types, fmt.Sprintf("%T", cause))
			}
			value["ErrorTypes"] = types
			var fault *foundation.Fault
			if errors.As(err, &fault) {
				value["FaultCode"] = fault.Code
			}
		}
		var pg *pgconn.PgError
		switch {
		case errors.As(err, &pg):
			value["SQLSTATE"] = pg.Code
		case errors.Is(err, context.Canceled):
			value["ErrorKind"] = "canceled"
		case errors.Is(err, context.DeadlineExceeded):
			value["ErrorKind"] = "deadline"
		case err != nil:
			value["ErrorKind"] = "other"
		}
		value["Request"] = ctx.Value(query131Request{})
		value["Authorization"] = map[string]int64{"PlannedInputs": reads.planned.Load(), "BatchedInputs": reads.batched.Load(), "DirectDecisions": reads.direct.Load()}
		s.write(value)
	}
	return err
}
func (u query131Unit) CaptureQuery(ctx context.Context, q dep.QueryRequest) (dep.QueryCapture, error) {
	v, err := u.Unit.CaptureQuery(ctx, q)
	u.write(map[string]any{"Route": ctx.Value(query131Route{}), "Stats": v.Stats, "Failed": err != nil, "CapturedProjectionIDs": len(v.PageIDs), "SelectionExtent": q.Extent == dep.QuerySelection})
	return v, err
}

func (u query131Unit) WalkParents(ctx context.Context, q dep.ParentRequest, visit func(dep.ParentFrame) (bool, error)) error {
	candidates := 0
	err := u.Unit.WalkParents(ctx, q, func(frame dep.ParentFrame) (bool, error) {
		if !frame.Common {
			candidates++
		}
		u.write(map[string]any{"Route": ctx.Value(query131Route{}), "Stats": frame.Capture.Stats, "Common": frame.Common, "Frame": candidates})
		return visit(frame)
	})
	u.write(map[string]any{"Route": ctx.Value(query131Route{}), "Candidates": candidates, "Windows": len(q.Windows), "Failed": err != nil})
	return err
}
func (u query131Unit) WalkQuery(ctx context.Context, q dep.QueryRequest, visit func(dep.QueryFrame) (bool, error)) error {
	frames := 0
	err := u.Unit.WalkQuery(ctx, q, func(frame dep.QueryFrame) (bool, error) {
		frames++
		u.write(map[string]any{"Route": ctx.Value(query131Route{}), "Stats": frame.Capture.Stats, "Frame": frames})
		return visit(frame)
	})
	u.write(map[string]any{"Route": ctx.Value(query131Route{}), "Frames": frames, "Failed": err != nil})
	return err
}

func observeQuery131(runtime *bootstrap.Runtime, handler http.Handler) (http.Handler, func() error, error) {
	return ObserveDepartmentQueries(runtime, handler)
}

// ObserveDepartmentQueries reuses the opt-in, private verification observer for
// other offline fixture servers. It is never enabled by the product executable.
func ObserveDepartmentQueries(runtime *bootstrap.Runtime, handler http.Handler) (http.Handler, func() error, error) {
	path := os.Getenv("CHNHCM_QUERY131_STATS")
	if path == "" {
		return handler, func() error { return nil }, nil
	}
	if err := privateDirectory(filepath.Dir(path)); err != nil {
		return nil, nil, err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, nil, err
	}
	var mu sync.Mutex
	var writeError error
	encoder := json.NewEncoder(file)
	service := runtime.Department.(dep.Service)
	service.Store = query131Store{service.Store, func(value any) {
		mu.Lock()
		defer mu.Unlock()
		if writeError == nil {
			writeError = encoder.Encode(value)
		}
	}}
	runtime.Department = service
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only controlled static routes; dynamic IDs are never written.
			route := "list"
			switch r.URL.Path {
			case "/api/v1/departments/selection/start":
				route = "start"
			case "/api/v1/departments/selection/resolve":
				route = "resolve"
			case "/api/v1/departments/export":
				route = "export"
			case "/api/v1/departments/references/parents":
				route = "parents"
			case "/api/v1/departments/navigation":
				route = "navigation"
			}
			ctx := context.WithValue(r.Context(), query131Route{}, route)
			ctx = context.WithValue(ctx, query131Request{}, w.Header().Get("X-Capacity-Request"))
			handler.ServeHTTP(w, r.WithContext(ctx))
		}), func() error {
			mu.Lock()
			defer mu.Unlock()
			err := file.Close()
			if writeError != nil {
				return writeError
			}
			return err
		}, nil
}

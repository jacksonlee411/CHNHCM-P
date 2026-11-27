package bootstrap

import (
	accessdb "chnhcm.local/chnhcm/internal/contexts/access/infra/postgres"
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"strconv"
)

func (a *API) accessRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/access/{area}", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		page, _ := strconv.Atoi(q.Get("page"))
		size, _ := strconv.Atoi(q.Get("pageSize"))
		v, e := a.Runtime.Management.ManagementView(r.Context(), credentials(r), access.ManagementQuery{Area: r.PathValue("area"), Page: page, PageSize: size, Subject: q.Get("subject"), Search: q.Get("q"), Owner: q.Get("owner"), State: q.Get("state"), Object: q.Get("object")})
		respond(w, 200, v, e)
	})
	mux.HandleFunc("POST /api/v1/access/commands", func(w http.ResponseWriter, r *http.Request) {
		var in access.ManagementCommand
		if _, e := decode(r, &in); e != nil {
			writeError(w, e)
			return
		}
		v, e := a.Runtime.Management.Manage(r.Context(), credentials(r), in)
		respond(w, 200, v, e)
	})
	mux.HandleFunc("POST /api/v1/access/submissions/{id}/resolve", func(w http.ResponseWriter, r *http.Request) {
		v, e := a.Runtime.Management.ResolveManagement(r.Context(), credentials(r), r.PathValue("id"))
		respond(w, 200, v, e)
	})
	mux.HandleFunc("GET /api/v1/access/submissions/{id}", func(w http.ResponseWriter, r *http.Request) {
		v, e := a.Runtime.Management.ManagementResult(r.Context(), credentials(r), r.PathValue("id"))
		respond(w, 200, v, e)
	})
}

func RecoverManagement(ctx context.Context, p *pgxpool.Pool, instance string, in access.RecoveryCommand) (access.ManagementResult, error) {
	var exists bool
	if e := p.QueryRow(ctx, "SELECT EXISTS(SELECT FROM access.principal WHERE instance_id=$1)", instance).Scan(&exists); e != nil {
		return access.ManagementResult{}, e
	}
	if !exists {
		return access.ManagementResult{}, errors.New("客户实例尚未初始化")
	}
	return accessdb.RecoverManagement(ctx, p, instance, in)
}

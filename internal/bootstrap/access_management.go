package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	accessdb "chnhcm.local/chnhcm/internal/contexts/access/infra/postgres"
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	orgdb "chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func (a *API) accessRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/access/{area}", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		page, _ := strconv.Atoi(q.Get("page"))
		size, _ := strconv.Atoi(q.Get("pageSize"))
		v, e := a.Runtime.Management.ManagementView(r.Context(), credentials(r), access.ManagementQuery{Area: r.PathValue("area"), Page: page, PageSize: size, Subject: q.Get("subject"), Search: q.Get("q"), Owner: q.Get("owner"), State: q.Get("state"), Object: q.Get("object"), Panel: q.Get("panel"), VersionKind: q.Get("versionKind"), LogField: q.Get("logField"), Source: q.Get("source"), Version: q.Get("version"), Record: q.Get("record"), Choices: q.Get("choices") == "1", Filters: q.Get("choices") == "filters"})
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
	org := &orgdb.Store{Instance: instance}
	return accessdb.RecoverManagement(ctx, p, instance, in, org.ManagementEnterpriseID)
}

// Upgraded databases may already contain a deployment-initialized admin but
// have never appointed a security administrator or run initialize again.
func initializeExistingManagement(ctx context.Context, pool *pgxpool.Pool) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `SELECT e.instance_id::text, e.enterprise_id::text FROM organization.enterprise e
 WHERE EXISTS(SELECT FROM access.principal p WHERE p.instance_id=e.instance_id AND p.normalized_login='admin')
 AND NOT EXISTS(SELECT FROM access.management_root m WHERE m.instance_id=e.instance_id) ORDER BY e.instance_id`)
	if err != nil {
		return err
	}
	type original struct{ Instance, Enterprise string }
	entries, err := pgx.CollectRows(rows, pgx.RowToStructByPos[original])
	if err != nil {
		return err
	}
	for _, e := range entries {
		if err := accessdb.InitializeManagement(ctx, tx, e.Instance, e.Enterprise); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

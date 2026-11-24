package bootstrap

import (
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"net/http"
)

func (a *API) departmentTreeRoutes(mux *http.ServeMux) {
	for _, mode := range []string{"nodes", "search"} {
		mux.HandleFunc("GET /api/v1/departments/tree/"+mode, func(w http.ResponseWriter, r *http.Request) {
			allowed := []string{"pageSize", "cursor"}
			if mode == "nodes" {
				allowed = append(allowed, "parentId", "aroundId")
			} else {
				allowed = append(allowed, "q")
			}
			q, e := queryValues(r, allowed...)
			if e != nil {
				writeError(w, e)
				return
			}
			size, e := number(q.Get("pageSize"), 0)
			if e != nil {
				writeError(w, e)
				return
			}
			if q.Has("pageSize") && size != 20 && size != 50 && size != 100 || q.Has("aroundId") && q.Get("aroundId") == "" {
				writeError(w, foundation.Fail(400, "M02_INVALID_INPUT", "部门树查询参数无效"))
				return
			}
			in := org.DepartmentTreeInput{Q: q.Get("q"), Cursor: q.Get("cursor"), AroundID: q.Get("aroundId"), PageSize: size}
			if q.Has("parentId") {
				parent := q.Get("parentId")
				in.ParentID = &parent
			}
			if mode == "nodes" {
				v, e := a.Runtime.Department.TreeNodes(r.Context(), credentials(r), in)
				respond(w, 200, v, e)
			} else {
				v, e := a.Runtime.Department.TreeSearch(r.Context(), credentials(r), in)
				respond(w, 200, v, e)
			}
		})
	}
	mux.HandleFunc("GET /api/v1/departments/{id}/tree-location", func(w http.ResponseWriter, r *http.Request) {
		if _, e := queryValues(r); e != nil {
			writeError(w, e)
			return
		}
		v, e := a.Runtime.Department.TreeLocation(r.Context(), credentials(r), r.PathValue("id"))
		respond(w, 200, v, e)
	})
}

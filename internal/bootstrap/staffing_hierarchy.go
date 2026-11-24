package bootstrap

import (
	"encoding/json"
	"io"
	"net/http"

	s "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

func (a *API) hierarchyRoutes(mux *http.ServeMux) {
	const base = "/api/v1/position-hierarchies"
	const reporting = base + "/reporting"
	mux.HandleFunc("GET "+reporting, func(w http.ResponseWriter, r *http.Request) {
		query, err := queryValues(r, "businessAt", "sourceVersion")
		if err != nil {
			writeError(w, staffingError(err))
			return
		}
		out, err := a.Runtime.Staffing.HierarchyAt(r.Context(), credentials(r), query.Get("businessAt"), query.Get("sourceVersion"))
		respond(w, 200, out, staffingError(err))
	})
	mux.HandleFunc("GET "+reporting+"/structure", func(w http.ResponseWriter, r *http.Request) {
		params, err := queryValues(r, "mode", "businessAt", "sourceVersion", "nodeId", "q", "page", "pageSize")
		if err != nil {
			writeError(w, staffingError(err))
			return
		}
		page, err := number(params.Get("page"), 1)
		if err != nil {
			writeError(w, staffingError(err))
			return
		}
		size, err := number(params.Get("pageSize"), 30)
		if err != nil {
			writeError(w, staffingError(err))
			return
		}
		q := s.StructureQuery{Mode: params.Get("mode"), BusinessAt: params.Get("businessAt"),
			SourceVersion: params.Get("sourceVersion"), NodeID: params.Get("nodeId"), Q: params.Get("q"), Page: page, PageSize: size}
		out, err := a.Runtime.Staffing.Structure(r.Context(), credentials(r), q)
		respond(w, 200, out, staffingError(err))
	})
	mux.HandleFunc("POST "+reporting+"/publish", func(w http.ResponseWriter, r *http.Request) {
		if _, err := queryValues(r); err != nil {
			writeError(w, staffingError(err))
			return
		}
		var input s.PublishHierarchy
		raw, err := decode(r, &input)
		if err == nil {
			var keys map[string]json.RawMessage
			_ = json.Unmarshal(raw, &keys)
			if keys["parentId"] == nil || keys["expectedVersion"] == nil || string(keys["expectedVersion"]) == "null" {
				err = foundation.Fail(400, "M03_INVALID_INPUT", "须明确上级岗位（撤边传null）及期望版本")
			}
		}
		if err != nil {
			writeError(w, staffingError(err))
			return
		}
		out, err := a.Runtime.Staffing.PublishHierarchy(r.Context(), credentials(r), r.Header.Get("Idempotency-Key"), digest(r, raw), input)
		if err != nil {
			writeError(w, staffingError(err))
			return
		}
		if out.Failure != nil {
			writeError(w, out.Failure)
			return
		}
		writeJSON(w, 200, out)
	})
	mux.HandleFunc("POST "+base+"/submissions/{id}/resolve", func(w http.ResponseWriter, r *http.Request) {
		if _, err := queryValues(r); err != nil {
			writeError(w, staffingError(err))
			return
		}
		if body, err := io.ReadAll(io.LimitReader(r.Body, 1)); err != nil || len(body) != 0 {
			writeError(w, foundation.Fail(400, "M03_INVALID_INPUT", "查回岗位关系提交时只需提交号"))
			return
		}
		out, err := a.Runtime.Staffing.Resolve(r.Context(), credentials(r), s.Hierarchy, r.PathValue("id"))
		respond(w, 200, out, staffingError(err))
	})
}

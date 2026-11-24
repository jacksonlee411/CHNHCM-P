package bootstrap

import (
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/geoarea"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"net/http"
	"slices"
)

func (a *API) geoareaRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/geo-areas/candidates", func(w http.ResponseWriter, r *http.Request) {
		q, e := queryValues(r, "countryRegionId", "parentAreaId", "adoptedDate", "q", "pageSize", "cursor")
		if e != nil {
			writeError(w, e)
			return
		}
		size, e := number(q.Get("pageSize"), 20)
		if e != nil {
			writeError(w, e)
			return
		}
		v, e := a.Runtime.GeoArea.Candidates(r.Context(), credentials(r), org.GeoAreaCandidateInput{CountryRegionID: q.Get("countryRegionId"), ParentAreaID: q.Get("parentAreaId"), AdoptedDate: q.Get("adoptedDate"), Q: q.Get("q"), PageSize: size, Cursor: q.Get("cursor")})
		respond(w, 200, v, e)
	})
	mux.HandleFunc("GET /api/v1/geo-areas/context", func(w http.ResponseWriter, r *http.Request) {
		if _, e := queryValues(r); e != nil {
			writeError(w, e)
			return
		}
		v, e := a.Runtime.GeoArea.Context(r.Context(), credentials(r))
		respond(w, 200, v, e)
	})
	mux.HandleFunc("GET /api/v1/geo-areas/operation-reasons", func(w http.ResponseWriter, r *http.Request) {
		q, e := queryValues(r, "operationType")
		if e != nil {
			writeError(w, e)
			return
		}
		if _, e = a.Runtime.GeoArea.Context(r.Context(), credentials(r)); e != nil {
			writeError(w, e)
			return
		}
		choices := (d.Policy{}).OperationReasons(q.Get("operationType"))
		if len(choices) == 0 {
			writeError(w, foundation.Fail(400, "M02_INVALID_INPUT", "操作类型无效"))
			return
		}
		keys := []string{}
		for k := range choices {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		out := []org.Reason{}
		for _, k := range keys {
			out = append(out, org.Reason{Code: k, Name: choices[k], Default: len(keys) == 1})
		}
		writeJSON(w, 200, out)
	})
	mux.HandleFunc("GET /api/v1/geo-areas", a.geoareaList)
	mux.HandleFunc("GET /api/v1/geo-areas/{id}", func(w http.ResponseWriter, r *http.Request) {
		q, e := queryValues(r, "selectedRecordId", "sourceVersion")
		if e != nil {
			writeError(w, e)
			return
		}
		for _, k := range []string{"selectedRecordId", "sourceVersion"} {
			if q.Has(k) && !foundation.ValidID(q.Get(k)) {
				writeError(w, foundation.Fail(400, "M02_INVALID_INPUT", "来源版本或所选记录无效"))
				return
			}
		}
		v, e := a.Runtime.GeoArea.Detail(r.Context(), credentials(r), org.DetailInput{ID: r.PathValue("id"), SelectedRecordID: q.Get("selectedRecordId"), SourceVersion: q.Get("sourceVersion")})
		respond(w, 200, v, e)
	})
	mux.HandleFunc("GET /api/v1/geo-areas/{id}/changes", a.geoareaChanges)
	mux.HandleFunc("POST /api/v1/geo-areas/navigation", func(w http.ResponseWriter, r *http.Request) {
		if _, e := queryValues(r); e != nil {
			writeError(w, e)
			return
		}
		var in org.GeoAreaNavigationInput
		_, e := decode(r, &in)
		if e != nil {
			writeError(w, e)
			return
		}
		v, e := a.Runtime.GeoArea.Navigate(r.Context(), credentials(r), in)
		respond(w, 200, v, e)
	})
	mux.HandleFunc("POST /api/v1/geo-areas/{id}/preview", func(w http.ResponseWriter, r *http.Request) {
		if _, e := queryValues(r); e != nil {
			writeError(w, e)
			return
		}
		var in org.GeoAreaPreviewInput
		raw, e := decode(r, &in)
		if e == nil {
			e = geoareaPreviewShape(raw, in)
		}
		if e != nil {
			writeError(w, e)
			return
		}
		v, e := a.Runtime.GeoArea.Preview(r.Context(), credentials(r), r.PathValue("id"), in)
		respond(w, 200, v, previewExecutionError(e))
	})
	for _, route := range []struct{ path, action string }{{"/{id}/records", "ADD"}, {"/{id}/records/{recordId}/correct", "CORRECT"}, {"/{id}/deactivate", "DEACTIVATE"}, {"/{id}/reactivate", "REACTIVATE"}} {
		mux.HandleFunc("POST /api/v1/geo-areas"+route.path, func(w http.ResponseWriter, r *http.Request) { a.geoareaExecute(w, r, route.action) })
	}
	mux.HandleFunc("POST /api/v1/geo-areas/submissions/{id}/resolve", func(w http.ResponseWriter, r *http.Request) {
		if _, e := queryValues(r); e != nil {
			writeError(w, e)
			return
		}
		v, e := a.Runtime.GeoArea.Resolve(r.Context(), credentials(r), r.PathValue("id"))
		var pg *pgconn.PgError
		if errors.Is(e, context.DeadlineExceeded) || errors.Is(e, context.Canceled) || (errors.As(e, &pg) && (pg.Code == "55P03" || pg.Code == "57014")) {
			writeJSON(w, 202, org.GeoAreaResult{SubmissionID: r.PathValue("id"), Outcome: "pending"})
			return
		}
		respond(w, 200, v, e)
	})
}
func geoareaPreviewShape(raw []byte, in org.GeoAreaPreviewInput) error {
	allowed := []string{"action", "expectedSourceVersion"}
	required := slices.Clone(allowed)
	switch in.Action {
	case "CORRECT":
		allowed = append(allowed, "recordId", "expectedRecordRevision", "changes")
		required = slices.Clone(allowed)
	case "ADD", "DEACTIVATE", "REACTIVATE":
		allowed = append(allowed, "anchorRecordId", "effectiveDate", "operationType", "operationReason")
		required = slices.Clone(allowed)
		if in.Action != "DEACTIVATE" {
			allowed = append(allowed, "changes")
		}
		if in.Action == "ADD" {
			required = append(required, "changes")
		}
	default:
		return foundation.Fail(400, "M02_INVALID_INPUT", "预览动作无效")
	}
	return commandShape(raw, allowed, required)
}
func (a *API) geoareaExecute(w http.ResponseWriter, r *http.Request, action string) {
	if _, e := queryValues(r); e != nil {
		writeError(w, e)
		return
	}
	cmd := org.GeoAreaCommand{Action: action, GeoAreaID: r.PathValue("id"), RecordID: r.PathValue("recordId"), SubmissionID: r.Header.Get("Idempotency-Key")}
	var input any = &cmd.Add
	if action == "CORRECT" {
		input = &cmd.Correct
	}
	raw, e := decode(r, input)
	if e != nil {
		writeError(w, e)
		return
	}
	cmd.Digest = digest(r, raw)
	v, e := a.Runtime.GeoArea.Maintain(r.Context(), credentials(r), cmd)
	if v.Outcome == "rejected" {
		code := "M02_INVALID_INPUT"
		if v.ErrorCode != nil {
			code = *v.ErrorCode
		}
		status := 409
		message := "原请求已拒绝，请核对输入和当前资料后重新办理"
		field := ""
		var f *foundation.Fault
		if errors.As(e, &f) {
			status = f.Status
			message = f.Message
			field = f.Field
		}
		writeJSON(w, status, map[string]any{"code": code, "message": message, "field": field, "submissionId": v.SubmissionID, "outcome": v.Outcome})
		return
	}
	status := 201
	if action == "CORRECT" || v.Outcome == "no_change" {
		status = 200
	}
	respond(w, status, v, e)
}
func (a *API) geoareaList(w http.ResponseWriter, r *http.Request) {
	q, e := queryValues(r, "q", "status", "sort", "page", "pageSize", "filterField", "filterOperator", "filterValue", "countryRegionId", "parentAreaId", "level")
	if e != nil {
		writeError(w, e)
		return
	}
	page, e := number(q.Get("page"), 1)
	if e != nil {
		writeError(w, e)
		return
	}
	size, e := number(q.Get("pageSize"), 20)
	if e != nil {
		writeError(w, e)
		return
	}
	in := org.GeoAreaQuery{Q: q.Get("q"), Status: q.Get("status"), Sort: q.Get("sort"), Page: page, PageSize: size, CountryRegionID: q.Get("countryRegionId"), ParentAreaID: q.Get("parentAreaId"), Level: q.Get("level")}
	if q.Has("filterField") || q.Has("filterOperator") || q.Has("filterValue") {
		in.Filter = &org.GeoAreaFilter{Field: q.Get("filterField"), Operator: q.Get("filterOperator"), Value: q.Get("filterValue")}
	}
	v, e := a.Runtime.GeoArea.List(r.Context(), credentials(r), in)
	respond(w, 200, v, e)
}

func (a *API) geoareaChanges(w http.ResponseWriter, r *http.Request) {
	q, err := queryValues(r, "page", "pageSize", "contextSourceVersion", "throughSourceVersion", "filterField", "filterOperator", "filterValue")
	if err != nil {
		writeError(w, err)
		return
	}
	for _, key := range []string{"page", "pageSize", "contextSourceVersion", "throughSourceVersion"} {
		if q.Has(key) && q.Get(key) == "" {
			writeError(w, foundation.Fail(400, "M02_INVALID_INPUT", "查询条件不能为空"))
			return
		}
	}
	page, err := number(q.Get("page"), 1)
	if err != nil {
		writeError(w, err)
		return
	}
	size, err := number(q.Get("pageSize"), 20)
	if err != nil {
		writeError(w, err)
		return
	}
	in := org.ChangeQuery{ID: r.PathValue("id"), Page: page, PageSize: size, ContextSourceVersion: q.Get("contextSourceVersion"), ThroughSourceVersion: q.Get("throughSourceVersion")}
	if q.Has("filterField") || q.Has("filterOperator") || q.Has("filterValue") {
		in.Filter = &org.ColumnFilter{Field: q.Get("filterField"), Operator: q.Get("filterOperator"), Value: q.Get("filterValue")}
	}
	value, err := a.Runtime.GeoArea.Changes(r.Context(), credentials(r), in)
	respond(w, 200, value, err)
}

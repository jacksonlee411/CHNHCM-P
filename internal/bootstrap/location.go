package bootstrap

import (
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/location"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"io"
	"net/http"
	"slices"
	"strconv"
)

func (a *API) locationRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/locations/reference-context", func(w http.ResponseWriter, r *http.Request) {
		if _, e := queryValues(r); e != nil {
			writeError(w, e)
			return
		}
		var in org.LocationReferenceInput
		if _, e := decode(r, &in); e != nil {
			writeError(w, e)
			return
		}
		v, e := a.Runtime.Location.ReferenceContext(r.Context(), credentials(r), in)
		respond(w, 200, v, e)
	})
	mux.HandleFunc("POST /api/v1/locations/selection/start", func(w http.ResponseWriter, r *http.Request) {
		var in org.LocationSelectionStartInput
		if !decodeIdentityDraft(w, r, &in, true) {
			return
		}
		v, e := a.Runtime.Location.StartSelection(r.Context(), credentials(r), in)
		if e != nil {
			writeSelectionError(w, e)
			return
		}
		writeJSON(w, 200, v)
	})
	mux.HandleFunc("POST /api/v1/locations/selection/resolve", func(w http.ResponseWriter, r *http.Request) {
		var in org.LocationSelectionInput
		if !decodeIdentityDraft(w, r, &in, true) {
			return
		}
		v, e := a.Runtime.Location.ResolveSelection(r.Context(), credentials(r), in)
		if e != nil {
			writeSelectionError(w, e)
			return
		}
		writeJSON(w, 200, v)
	})
	mux.HandleFunc("POST /api/v1/locations/selection/cancel", func(w http.ResponseWriter, r *http.Request) {
		var in org.LocationDraftInput
		if !decodeIdentityDraft(w, r, &in, false) {
			return
		}
		if err := a.Runtime.Location.CancelSelection(r.Context(), credentials(r), in); err != nil {
			writeSelectionError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /api/v1/locations/export", func(w http.ResponseWriter, r *http.Request) {
		var in org.LocationExportInput
		if !decodeIdentityDraft(w, r, &in, true) {
			return
		}
		v, e := a.Runtime.Location.ExportSelection(r.Context(), credentials(r), in)
		if e != nil {
			writeSelectionError(w, e)
			return
		}
		defer v.File.Close()
		if r.Context().Err() != nil {
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="chnhcm-locations-`+v.BusinessAt+`.csv"`)
		w.Header().Set("X-Selected-Count", strconv.Itoa(v.Count))
		w.WriteHeader(200)
		_, _ = io.Copy(w, v.File)
	})

	mux.HandleFunc("GET /api/v1/locations/context", func(w http.ResponseWriter, r *http.Request) {
		if _, e := queryValues(r); e != nil {
			writeError(w, e)
			return
		}
		v, e := a.Runtime.Location.Context(r.Context(), credentials(r))
		respond(w, 200, v, e)
	})
	mux.HandleFunc("GET /api/v1/locations/operation-reasons", func(w http.ResponseWriter, r *http.Request) {
		q, e := queryValues(r, "operationType")
		if e != nil {
			writeError(w, e)
			return
		}
		if _, e = a.Runtime.Location.Context(r.Context(), credentials(r)); e != nil {
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
	mux.HandleFunc("GET /api/v1/locations", a.locationList)
	mux.HandleFunc("GET /api/v1/locations/{id}", func(w http.ResponseWriter, r *http.Request) {
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
		v, e := a.Runtime.Location.Detail(r.Context(), credentials(r), org.DetailInput{ID: r.PathValue("id"), SelectedRecordID: q.Get("selectedRecordId"), SourceVersion: q.Get("sourceVersion")})
		respond(w, 200, v, e)
	})
	mux.HandleFunc("GET /api/v1/locations/{id}/changes", a.locationChanges)
	mux.HandleFunc("POST /api/v1/locations/navigation", func(w http.ResponseWriter, r *http.Request) {
		if _, e := queryValues(r); e != nil {
			writeError(w, e)
			return
		}
		var in org.LocationNavigationInput
		_, e := decode(r, &in)
		if e != nil {
			writeError(w, e)
			return
		}
		v, e := a.Runtime.Location.Navigate(r.Context(), credentials(r), in)
		respond(w, 200, v, e)
	})
	mux.HandleFunc("POST /api/v1/locations/{id}/preview", func(w http.ResponseWriter, r *http.Request) {
		if _, e := queryValues(r); e != nil {
			writeError(w, e)
			return
		}
		var in org.LocationPreviewInput
		raw, e := decode(r, &in)
		if e == nil {
			e = locationPreviewShape(raw, in)
		}
		if e != nil {
			writeError(w, e)
			return
		}
		v, e := a.Runtime.Location.Preview(r.Context(), credentials(r), r.PathValue("id"), in)
		respond(w, 200, v, previewExecutionError(e))
	})
	for _, route := range []struct{ path, action string }{{"", "CREATE"}, {"/{id}/records/{recordId}/delete", "DELETE"}, {"/{id}/records/{recordId}/redate", "REDATE"}, {"/{id}/lifecycle-pairs/insert", "INSERT_LIFECYCLE_PAIR"}, {"/{id}/lifecycle-pairs/delete", "DELETE_LIFECYCLE_PAIR"}, {"/{id}/lifecycle-pairs/maintain", "MAINTAIN_LIFECYCLE_PAIR"}, {"/{id}/records", "ADD"}, {"/{id}/records/{recordId}/correct", "CORRECT"}, {"/{id}/deactivate", "DEACTIVATE"}, {"/{id}/reactivate", "REACTIVATE"}} {
		mux.HandleFunc("POST /api/v1/locations"+route.path, func(w http.ResponseWriter, r *http.Request) { a.locationExecute(w, r, route.action) })
	}
	mux.HandleFunc("POST /api/v1/locations/submissions/{id}/resolve", func(w http.ResponseWriter, r *http.Request) {
		if _, e := queryValues(r); e != nil {
			writeError(w, e)
			return
		}
		v, e := a.Runtime.Location.Resolve(r.Context(), credentials(r), r.PathValue("id"))
		var pg *pgconn.PgError
		if errors.Is(e, context.DeadlineExceeded) || errors.Is(e, context.Canceled) || (errors.As(e, &pg) && (pg.Code == "55P03" || pg.Code == "57014")) {
			writeJSON(w, 202, org.LocationResult{SubmissionID: r.PathValue("id"), Outcome: "pending"})
			return
		}
		respond(w, 200, v, e)
	})
}
func locationPreviewShape(raw []byte, in org.LocationPreviewInput) error {
	allowed := []string{"action", "expectedSourceVersion"}
	required := slices.Clone(allowed)
	switch in.Action {
	case "CORRECT":
		allowed = append(allowed, "recordId", "expectedRecordRevision", "changes", "newEffectiveDate", "classification")
		required = append(required, "recordId", "expectedRecordRevision")
	case "DELETE", "REDATE":
		allowed = append(allowed, "recordId", "expectedRecordRevision")
		required = slices.Clone(allowed)
		if in.Action == "REDATE" {
			allowed = append(allowed, "newEffectiveDate")
			required = slices.Clone(allowed)
		}
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
func (a *API) locationExecute(w http.ResponseWriter, r *http.Request, action string) {
	if _, e := queryValues(r); e != nil {
		writeError(w, e)
		return
	}
	cmd := org.LocationCommand{Action: action, LocationID: r.PathValue("id"), RecordID: r.PathValue("recordId"), SubmissionID: r.Header.Get("Idempotency-Key")}
	var input any = &cmd.Add
	switch action {
	case "CREATE":
		input = &cmd.Create
	case "CORRECT":
		input = &cmd.Correct
	case "DELETE":
		input = &cmd.Delete
	case "REDATE":
		input = &cmd.Redate
	case "INSERT_LIFECYCLE_PAIR":
		input = &cmd.InsertPair
	case "DELETE_LIFECYCLE_PAIR":
		input = &cmd.DeletePair
	case "MAINTAIN_LIFECYCLE_PAIR":
		input = &cmd.MaintainPair
	}
	raw, e := decode(r, input)
	if e != nil {
		writeError(w, e)
		return
	}
	cmd.Digest = digest(r, raw)
	v, e := a.Runtime.Location.Maintain(r.Context(), credentials(r), cmd)
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
	status := 200
	if slices.Contains([]string{"CREATE", "ADD", "DEACTIVATE", "REACTIVATE"}, action) && v.Outcome != "no_change" {
		status = 201
	}
	respond(w, status, v, e)
}
func (a *API) locationList(w http.ResponseWriter, r *http.Request) {
	q, e := queryValues(r, "q", "status", "sort", "page", "pageSize", "filterField", "filterOperator", "filterValue", "countryRegionId")
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
	in := org.LocationQuery{ListInput: org.ListInput{Q: q.Get("q"), Status: q.Get("status"), Sort: q.Get("sort"), Page: page, PageSize: size}, CountryRegionID: q.Get("countryRegionId")}
	if q.Has("filterField") || q.Has("filterOperator") || q.Has("filterValue") {
		in.Filter = &org.ColumnFilter{Field: q.Get("filterField"), Operator: q.Get("filterOperator"), Value: q.Get("filterValue")}
	}
	v, e := a.Runtime.Location.List(r.Context(), credentials(r), in)
	respond(w, 200, v, e)
}

func (a *API) locationChanges(w http.ResponseWriter, r *http.Request) {
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
	value, err := a.Runtime.Location.Changes(r.Context(), credentials(r), in)
	respond(w, 200, value, err)
}

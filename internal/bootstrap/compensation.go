package bootstrap

import (
	c "chnhcm.local/chnhcm/internal/contexts/compensation/public"
	"net/http"
)

func (a *API) compensationRoutes(mux *http.ServeMux) {
	a.compensationPlanRoutes(mux)
	a.compensationAwardRoutes(mux)
	const base = "/api/v1/compensation/grade-rates"
	mux.HandleFunc("GET /api/v1/compensation/access", func(w http.ResponseWriter, r *http.Request) {
		v, e := a.Runtime.Compensation.Menu(r.Context(), credentials(r))
		respond(w, 200, v, e)
	})
	mux.HandleFunc("GET "+base+"/create-context", func(w http.ResponseWriter, r *http.Request) {
		v, e := a.Runtime.Compensation.RateContext(r.Context(), credentials(r))
		respond(w, 200, v, e)
	})
	mux.HandleFunc("GET "+base, func(w http.ResponseWriter, r *http.Request) {
		q, e := queryValues(r, "q", "gradeId", "currency", "frequency", "form", "status", "page", "pageSize")
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
		v, e := a.Runtime.Compensation.ListRates(r.Context(), credentials(r), c.RateQuery{Q: q.Get("q"), GradeID: q.Get("gradeId"), Currency: q.Get("currency"), Frequency: q.Get("frequency"), Form: q.Get("form"), Status: q.Get("status"), Page: page, Size: size})
		respond(w, 200, v, e)
	})
	mux.HandleFunc("GET "+base+"/{id}", func(w http.ResponseWriter, r *http.Request) {
		q, e := queryValues(r, "recordId", "businessAt")
		if e != nil {
			writeError(w, e)
			return
		}
		v, e := a.Runtime.Compensation.ReadRate(r.Context(), credentials(r), r.PathValue("id"), q.Get("recordId"), q.Get("businessAt"))
		respond(w, 200, v, e)
	})
	mux.HandleFunc("GET "+base+"/{id}/history", func(w http.ResponseWriter, r *http.Request) {
		q, e := queryValues(r, "sourceVersion")
		if e != nil {
			writeError(w, e)
			return
		}
		v, e := a.Runtime.Compensation.RateHistory(r.Context(), credentials(r), r.PathValue("id"), q.Get("sourceVersion"))
		respond(w, 200, map[string]any{"items": v}, e)
	})
	save := func(w http.ResponseWriter, r *http.Request) {
		var input c.WriteRate
		if _, e := decode(r, &input); e != nil {
			writeError(w, e)
			return
		}
		v, e := a.Runtime.Compensation.SaveRate(r.Context(), credentials(r), r.PathValue("id"), r.Header.Get("Idempotency-Key"), input)
		respond(w, 200, v, e)
	}
	mux.HandleFunc("POST "+base, save)
	mux.HandleFunc("POST "+base+"/{id}", save)
	mux.HandleFunc("POST /api/v1/compensation/grade-rate-receipts/{submissionId}/resolve", func(w http.ResponseWriter, r *http.Request) {
		v, e := a.Runtime.Compensation.ResolveRate(r.Context(), credentials(r), r.PathValue("submissionId"))
		respond(w, 200, v, e)
	})
	mux.HandleFunc("GET /api/v1/compensation/grade-rate-receipts/{submissionId}", func(w http.ResponseWriter, r *http.Request) {
		v, e := a.Runtime.Compensation.RateReceipt(r.Context(), credentials(r), r.PathValue("submissionId"))
		respond(w, 200, v, e)
	})
}

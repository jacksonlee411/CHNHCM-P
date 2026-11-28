package bootstrap

import (
	"net/http"
	"reflect"

	c "chnhcm.local/chnhcm/internal/contexts/compensation/public"
)

var planChangesType = reflect.TypeFor[c.PlanChanges]()

// Validate nested protocol names and types before keeping their raw bytes so
// omitted protected fields and explicitly cleared values remain distinguishable.
type planChangesWire struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Scope       *struct {
		Mode    string `json:"mode,omitempty"`
		Choices []struct {
			Dimension string `json:"dimension"`
			ID        string `json:"id"`
			Exclude   bool   `json:"exclude,omitempty"`
		} `json:"choices,omitempty"`
	} `json:"scope,omitempty"`
	Components []struct {
		ID        string  `json:"id,omitempty"`
		Code      string  `json:"code,omitempty"`
		Name      string  `json:"name,omitempty"`
		Nature    string  `json:"nature,omitempty"`
		Currency  string  `json:"currency,omitempty"`
		Frequency string  `json:"frequency,omitempty"`
		Unit      string  `json:"unit,omitempty"`
		Precision int     `json:"precision,omitempty"`
		Method    string  `json:"method,omitempty"`
		Amount    *string `json:"amount,omitempty"`
		BaseID    *string `json:"baseId,omitempty"`
		Percent   *string `json:"percent,omitempty"`
		Rounding  *string `json:"rounding,omitempty"`
		Rates     []struct {
			GradeID string `json:"gradeId"`
			RateID  string `json:"rateId"`
		} `json:"rates,omitempty"`
	} `json:"components,omitempty"`
}

func (a *API) compensationPlanRoutes(mux *http.ServeMux) {
	const base = "/api/v1/compensation/plans"
	mux.HandleFunc("GET "+base+"/create-context", func(w http.ResponseWriter, r *http.Request) {
		v, e := a.Runtime.Compensation.PlanContext(r.Context(), credentials(r))
		respond(w, 200, v, e)
	})
	mux.HandleFunc("GET "+base, func(w http.ResponseWriter, r *http.Request) {
		q, e := queryValues(r, "q", "status", "page", "pageSize")
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
		v, e := a.Runtime.Compensation.ListPlans(r.Context(), credentials(r), c.RateQuery{Q: q.Get("q"), Status: q.Get("status"), Page: page, Size: size})
		respond(w, 200, v, e)
	})
	mux.HandleFunc("GET "+base+"/rate-candidates", func(w http.ResponseWriter, r *http.Request) {
		q, e := queryValues(r, "businessAt", "q", "currency", "frequency", "limit")
		if e != nil {
			writeError(w, e)
			return
		}
		limit, e := number(q.Get("limit"), 50)
		if e != nil {
			writeError(w, e)
			return
		}
		v, e := a.Runtime.Compensation.RateCandidates(r.Context(), credentials(r), q.Get("businessAt"), q.Get("q"), q.Get("currency"), q.Get("frequency"), limit)
		respond(w, 200, map[string]any{"items": v}, e)
	})
	mux.HandleFunc("GET "+base+"/{id}", func(w http.ResponseWriter, r *http.Request) {
		q, e := queryValues(r, "recordId", "businessAt")
		if e != nil {
			writeError(w, e)
			return
		}
		v, e := a.Runtime.Compensation.ReadPlan(r.Context(), credentials(r), r.PathValue("id"), q.Get("recordId"), q.Get("businessAt"))
		respond(w, 200, v, e)
	})
	mux.HandleFunc("GET "+base+"/{id}/history", func(w http.ResponseWriter, r *http.Request) {
		q, e := queryValues(r, "sourceVersion")
		if e != nil {
			writeError(w, e)
			return
		}
		v, e := a.Runtime.Compensation.PlanHistory(r.Context(), credentials(r), r.PathValue("id"), q.Get("sourceVersion"))
		respond(w, 200, map[string]any{"items": v}, e)
	})
	mux.HandleFunc("GET "+base+"/{id}/applicability", func(w http.ResponseWriter, r *http.Request) {
		q, e := queryValues(r, "businessAt", "legalEntityId", "jobFamilyId", "gradeId")
		if e != nil {
			writeError(w, e)
			return
		}
		v, e := a.Runtime.Compensation.PlanApplicable(r.Context(), credentials(r), r.PathValue("id"), q.Get("businessAt"), map[string]string{"legalEntity": q.Get("legalEntityId"), "jobFamily": q.Get("jobFamilyId"), "grade": q.Get("gradeId")})
		respond(w, 200, map[string]bool{"applicable": v}, e)
	})
	save := func(w http.ResponseWriter, r *http.Request) {
		var input c.WritePlan
		if _, e := decode(r, &input); e != nil {
			writeError(w, e)
			return
		}
		v, e := a.Runtime.Compensation.SavePlan(r.Context(), credentials(r), r.PathValue("id"), r.Header.Get("Idempotency-Key"), input)
		respond(w, 200, v, e)
	}
	mux.HandleFunc("POST "+base, save)
	mux.HandleFunc("POST "+base+"/{id}", save)
	mux.HandleFunc("GET /api/v1/compensation/plan-receipts/{submissionId}", func(w http.ResponseWriter, r *http.Request) {
		v, e := a.Runtime.Compensation.PlanReceipt(r.Context(), credentials(r), r.PathValue("submissionId"))
		respond(w, 200, v, e)
	})
	mux.HandleFunc("POST /api/v1/compensation/plan-receipts/{submissionId}/resolve", func(w http.ResponseWriter, r *http.Request) {
		v, e := a.Runtime.Compensation.ResolvePlan(r.Context(), credentials(r), r.PathValue("submissionId"))
		respond(w, 200, v, e)
	})
}

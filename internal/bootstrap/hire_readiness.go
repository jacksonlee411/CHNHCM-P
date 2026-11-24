package bootstrap

import (
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	staff "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	f "chnhcm.local/chnhcm/internal/foundation"
	"net/http"
)

func ruleBody[T any](run func(*http.Request, T) (any, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, e := queryValues(r); e != nil {
			writeError(w, e)
			return
		}
		var input T
		if _, e := decode(r, &input); e != nil {
			writeError(w, e)
			return
		}
		result, e := run(r, input)
		respond(w, 200, result, e)
	}
}
func ruleGet(run func(*http.Request, string) (any, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q, e := queryValues(r, "sourceVersion")
		if e != nil {
			writeError(w, e)
			return
		}
		v, e := run(r, q.Get("sourceVersion"))
		respond(w, 200, v, e)
	}
}
func (a *API) readinessRoutes(mux *http.ServeMux) {
	hires := a.Runtime.Hires
	mux.HandleFunc("GET /api/v1/employment-actions/{id}", func(w http.ResponseWriter, r *http.Request) {
		if _, e := queryValues(r); e != nil {
			writeError(w, e)
			return
		}
		v, e := hires.OriginalAction(r.Context(), credentials(r), r.PathValue("id"))
		respond(w, 200, v, e)
	})
	mux.HandleFunc("POST /api/v1/appointment-decisions", ruleBody(func(r *http.Request, in h.DecisionPublication) (any, error) {
		return hires.PublishDecision(r.Context(), credentials(r), r.Header.Get("Idempotency-Key"), in)
	}))
	mux.HandleFunc("GET /api/v1/persons/{id}/appointment-decisions", func(w http.ResponseWriter, r *http.Request) {
		if _, e := queryValues(r); e != nil {
			writeError(w, e)
			return
		}
		result, e := hires.HireDecisions(r.Context(), credentials(r), r.PathValue("id"))
		respond(w, 200, result, e)
	})
	mux.HandleFunc("POST /api/v1/hires/targets", ruleBody(func(r *http.Request, in h.TargetSearch) (any, error) {
		return hires.HireTargets(r.Context(), credentials(r), in)
	}))
	mux.HandleFunc("POST /api/v1/hires/employers", ruleBody(func(r *http.Request, in h.EmployerSearch) (any, error) {
		return hires.HireEmployers(r.Context(), credentials(r), in)
	}))
	mux.HandleFunc("POST /api/v1/hires/preview", ruleBody(func(r *http.Request, in h.Hire) (any, error) {
		return hires.PreviewHire(r.Context(), credentials(r), in)
	}))
	mux.HandleFunc("POST /api/v1/hires", ruleBody(func(r *http.Request, in h.Hire) (any, error) {
		return hires.Hire(r.Context(), credentials(r), r.Header.Get("Idempotency-Key"), in)
	}))
	mux.HandleFunc("POST /api/v1/employment-actions/{id}/cancel", ruleBody(func(r *http.Request, in h.Cancel) (any, error) {
		return hires.CancelHire(r.Context(), credentials(r), r.Header.Get("Idempotency-Key"), r.PathValue("id"), in)
	}))
	mux.HandleFunc("POST /api/v1/hires/submissions/{sub}/resolve", func(w http.ResponseWriter, r *http.Request) {
		if _, err := queryValues(r); err != nil {
			writeError(w, err)
			return
		}
		v, e := hires.ResolveHire(r.Context(), credentials(r), r.PathValue("sub"))
		respond(w, 200, v, e)
	})
	mux.HandleFunc("GET /api/v1/persons/{id}/employment", func(w http.ResponseWriter, r *http.Request) {
		q, e := queryValues(r, "businessAt")
		if e != nil {
			writeError(w, e)
			return
		}
		v, e := hires.PersonEmployment(r.Context(), credentials(r), r.PathValue("id"), q.Get("businessAt"))
		respond(w, 200, v, e)
	})
	const purposes = "/api/v1/legal-entities/{id}/purposes"
	const policies = "/api/v1/assignment-policies"
	resolve := func(run func(*http.Request) (f.RuleReceipt, error)) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if _, e := queryValues(r); e != nil {
				writeError(w, e)
				return
			}
			v, e := run(r)
			respond(w, 200, v, e)
		}
	}
	mux.HandleFunc("POST /api/v1/legal-entities/purpose-submissions/{sub}/resolve", resolve(func(r *http.Request) (f.RuleReceipt, error) {
		return a.Runtime.LegalPurposes.ResolvePurpose(r.Context(), credentials(r), r.PathValue("sub"))
	}))
	mux.HandleFunc("POST "+policies+"/submissions/{sub}/resolve", resolve(func(r *http.Request) (f.RuleReceipt, error) {
		return a.Runtime.AssignmentPolicies.ResolvePolicy(r.Context(), credentials(r), r.PathValue("sub"))
	}))
	mux.HandleFunc("GET "+purposes, ruleGet(func(r *http.Request, v string) (any, error) {
		return a.Runtime.LegalPurposes.Purposes(r.Context(), credentials(r), r.PathValue("id"), v)
	}))
	mux.HandleFunc("POST "+purposes, ruleBody(func(r *http.Request, in f.RuleChange[org.LegalPurpose]) (any, error) {
		return a.Runtime.LegalPurposes.DeclarePurpose(r.Context(), credentials(r), r.PathValue("id"), r.Header.Get("Idempotency-Key"), in)
	}))
	mux.HandleFunc("POST "+purposes+"/check", ruleBody(func(r *http.Request, in org.PurposeCheck) (any, error) {
		return a.Runtime.LegalPurposes.CheckPurpose(r.Context(), credentials(r), r.PathValue("id"), in)
	}))
	mux.HandleFunc("POST /api/v1/legal-entities/purpose-candidates", ruleBody(func(r *http.Request, in org.PurposeCandidates) (any, error) {
		return a.Runtime.LegalPurposes.Candidates(r.Context(), credentials(r), in)
	}))
	mux.HandleFunc("GET "+policies, ruleGet(func(r *http.Request, v string) (any, error) {
		return a.Runtime.AssignmentPolicies.Policies(r.Context(), credentials(r), v)
	}))
	mux.HandleFunc("POST "+policies, ruleBody(func(r *http.Request, in f.RuleChange[staff.AssignmentPolicy]) (any, error) {
		return a.Runtime.AssignmentPolicies.PublishPolicy(r.Context(), credentials(r), r.Header.Get("Idempotency-Key"), in)
	}))
	mux.HandleFunc("POST "+policies+"/check", ruleBody(func(r *http.Request, in staff.AssignmentCheck) (any, error) {
		return a.Runtime.AssignmentPolicies.CheckAssignment(r.Context(), credentials(r), in)
	}))
}

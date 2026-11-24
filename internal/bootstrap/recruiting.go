package bootstrap

import (
	r "chnhcm.local/chnhcm/internal/contexts/recruiting/public"
	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	"net/http"
)

func (a *API) recruitingRoutes(mux *http.ServeMux) {
	offers := a.Runtime.Offers
	mux.HandleFunc("GET /api/v1/recruiting/context", func(w http.ResponseWriter, req *http.Request) {
		q, e := queryValues(req, "person")
		if e != nil {
			writeError(w, e)
			return
		}
		out, e := offers.Entry(req.Context(), credentials(req), q.Get("person"))
		respond(w, 200, out, e)
	})
	mux.HandleFunc("GET /api/v1/recruiting/offers", func(w http.ResponseWriter, req *http.Request) {
		q, e := queryValues(req, "person")
		if e != nil {
			writeError(w, e)
			return
		}
		out, e := offers.List(req.Context(), credentials(req), q.Get("person"))
		respond(w, 200, out, e)
	})
	mux.HandleFunc("GET /api/v1/recruiting/offers/{id}", ruleGet(func(req *http.Request, source string) (any, error) {
		return offers.Get(req.Context(), credentials(req), req.PathValue("id"), source)
	}))
	change := ruleBody(func(req *http.Request, in r.Change) (any, error) {
		return offers.Submit(req.Context(), credentials(req), req.PathValue("id"), req.Header.Get("Idempotency-Key"), in)
	})
	mux.HandleFunc("POST /api/v1/recruiting/offers", change)
	mux.HandleFunc("POST /api/v1/recruiting/offers/{id}/changes", change)
	mux.HandleFunc("POST /api/v1/recruiting/submissions/{sub}/resolve", ruleBody(func(req *http.Request, _ struct{}) (any, error) {
		return offers.Resolve(req.Context(), credentials(req), req.PathValue("sub"))
	}))
	mux.HandleFunc("POST /api/v1/recruiting/targets", ruleBody(func(req *http.Request, in h.TargetSearch) (any, error) {
		return a.Runtime.OfferTargets(req.Context(), credentials(req), in)
	}))
	mux.HandleFunc("POST /api/v1/recruiting/arrangement", ruleBody(func(req *http.Request, in h.ArrangementQuery) (any, error) {
		return a.Runtime.OfferArrangement(req.Context(), credentials(req), in)
	}))
}

package bootstrap

import (
	p "chnhcm.local/chnhcm/internal/contexts/personnel/public"
	f "chnhcm.local/chnhcm/internal/foundation"
	"context"
	"errors"
	"net/http"
)

func personPost[I, O any](fn func(*http.Request, I) (O, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, e := queryValues(r); e != nil {
			writeError(w, e)
			return
		}
		if id := r.PathValue("id"); id != "" && !f.ValidID(id) {
			writeError(w, f.Fail(404, "PERSON_UNAVAILABLE", "所选资料不可用"))
			return
		}
		var in I
		if _, e := decode(r, &in); e != nil {
			writeError(w, e)
			return
		}
		v, e := fn(r, in)
		var conflict *p.DocumentConflict
		if errors.As(e, &conflict) {
			writeJSON(w, conflict.Status, conflict)
			return
		}
		respond(w, 200, v, e)
	}
}
func (a *API) personnelRoutes(m *http.ServeMux) {
	const base = "/api/v1/persons"
	s := a.Runtime.Personnel
	get := func(path string, params []string, fn func(*http.Request) (any, error)) {
		m.HandleFunc("GET "+base+path, func(w http.ResponseWriter, r *http.Request) {
			if _, e := queryValues(r, params...); e != nil {
				writeError(w, e)
				return
			}
			for _, key := range []string{"id", "version"} {
				if v := r.PathValue(key); v != "" && !f.ValidID(v) {
					writeError(w, f.Fail(404, "PERSON_UNAVAILABLE", "所选资料不可用"))
					return
				}
			}
			v, e := fn(r)
			respond(w, 200, v, e)
		})
	}
	const types = "/api/v1/person-address-types"
	m.HandleFunc("GET "+types, func(w http.ResponseWriter, r *http.Request) {
		if _, e := queryValues(r); e != nil {
			writeError(w, e)
			return
		}
		v, e := s.AddressTypes(r.Context(), credentials(r))
		respond(w, 200, v, e)
	})
	m.HandleFunc("GET "+types+"/{typeId}", func(w http.ResponseWriter, r *http.Request) {
		if _, e := queryValues(r); e != nil {
			writeError(w, e)
			return
		}
		v, e := s.AddressTypeDetail(r.Context(), credentials(r), r.PathValue("typeId"))
		respond(w, 200, v, e)
	})
	m.HandleFunc("GET "+types+"/{typeId}/revisions/{revision}", func(w http.ResponseWriter, r *http.Request) {
		if _, e := queryValues(r); e != nil {
			writeError(w, e)
			return
		}
		v, e := s.AddressTypeSource(r.Context(), credentials(r), r.PathValue("typeId"), r.PathValue("revision"))
		respond(w, 200, v, e)
	})
	m.HandleFunc("POST "+types+"/submissions/{submission}/resolve", func(w http.ResponseWriter, r *http.Request) {
		if _, e := queryValues(r); e != nil {
			writeError(w, e)
			return
		}
		v, e := s.AddressTypeReceipt(r.Context(), credentials(r), r.PathValue("submission"))
		respond(w, 200, v, e)
	})
	m.HandleFunc("POST "+types, personPost(func(r *http.Request, in p.AddressTypeCommand) (p.AddressType, error) {
		return s.SaveAddressType(r.Context(), credentials(r), in)
	}))
	get("/{id}/address-type-candidates", []string{"itemId"}, func(r *http.Request) (any, error) {
		items, err := s.AddressTypeCandidates(r.Context(), credentials(r), r.PathValue("id"), r.URL.Query().Get("itemId"))
		return struct {
			Items []p.AddressTypeOption `json:"items"`
		}{items}, err
	})
	get("/context", nil, func(r *http.Request) (any, error) { return s.Entry(r.Context(), credentials(r)) })
	query := func(r *http.Request) (p.Query, error) {
		q := r.URL.Query()
		page, e := number(q.Get("page"), 1)
		if e != nil {
			return p.Query{}, e
		}
		size, e := number(q.Get("pageSize"), 20)
		return p.Query{Ref: q.Get("ref"), Sort: q.Get("sort"), Page: page, Size: size}, e
	}
	params := []string{"ref", "sort", "page", "pageSize"}
	get("", params, func(r *http.Request) (any, error) {
		q, e := query(r)
		if e != nil {
			return nil, e
		}
		return s.List(r.Context(), credentials(r), q)
	})
	get("/{id}/navigation", params, func(r *http.Request) (any, error) {
		q, e := query(r)
		if e != nil {
			return nil, e
		}
		return s.Navigation(r.Context(), credentials(r), r.PathValue("id"), q)
	})
	get("/{id}", []string{"view"}, func(r *http.Request) (any, error) {
		return s.Detail(r.Context(), credentials(r), r.PathValue("id"), "", r.URL.Query().Get("view"))
	})
	get("/{id}/sources/{version}", []string{"view"}, func(r *http.Request) (any, error) {
		return s.Detail(r.Context(), credentials(r), r.PathValue("id"), r.PathValue("version"), r.URL.Query().Get("view"))
	})
	get("/{id}/changes/{version}", nil, func(r *http.Request) (any, error) {
		return s.Differences(r.Context(), credentials(r), r.PathValue("id"), r.PathValue("version"))
	})
	m.HandleFunc("POST "+base+"/{id}/changes", personPost(func(r *http.Request, in p.ChangeQuery) (p.ChangePage, error) {
		return s.Changes(r.Context(), credentials(r), r.PathValue("id"), in)
	}))
	type search struct {
		Keyword string `json:"keyword"`
	}
	m.HandleFunc("POST "+base+"/queries", personPost(func(r *http.Request, in search) (map[string]string, error) {
		ref, e := s.CreateQuery(r.Context(), credentials(r), in.Keyword)
		return map[string]string{"ref": ref}, e
	}))
	m.HandleFunc("POST "+base, personPost(func(r *http.Request, in p.Register) (p.Result, error) {
		return s.Register(r.Context(), credentials(r), in)
	}))
	m.HandleFunc("POST "+base+"/{id}/corrections", personPost(func(r *http.Request, in p.Correct) (p.Result, error) {
		return s.Correct(r.Context(), credentials(r), r.PathValue("id"), in)
	}))
	m.HandleFunc("POST "+base+"/{id}/renumber", personPost(func(r *http.Request, in p.Renumber) (p.Result, error) {
		return s.Renumber(r.Context(), credentials(r), r.PathValue("id"), in)
	}))
	m.HandleFunc("POST "+base+"/submissions/{submission}/resolve", func(w http.ResponseWriter, r *http.Request) {
		v, e := s.Resolve(r.Context(), credentials(r), r.PathValue("submission"))
		if errors.Is(e, context.DeadlineExceeded) {
			writeJSON(w, 202, p.Result{Outcome: "pending", SubmissionID: r.PathValue("submission")})
			return
		}
		respond(w, 200, v, e)
	})
}

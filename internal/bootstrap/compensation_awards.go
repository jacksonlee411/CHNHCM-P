package bootstrap

import (
	"bytes"
	c "chnhcm.local/chnhcm/internal/contexts/compensation/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"encoding/json"
	"net/http"
)

func (a *API) compensationAwardRoutes(mux *http.ServeMux) {
	const base = "/api/v1/compensation/assignments/{assignmentId}/base-pay"
	mux.HandleFunc("GET "+base+"/access", func(w http.ResponseWriter, r *http.Request) {
		q, e := queryValues(r, "businessAt")
		if e != nil {
			writeError(w, e)
			return
		}
		ok, e := a.Runtime.Compensation.PayAccess(r.Context(), credentials(r), r.PathValue("assignmentId"), q.Get("businessAt"))
		respond(w, 200, map[string]bool{"available": ok}, e)
	})
	read := func(prepare bool) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			q, e := queryValues(r, "businessAt", "candidatePage", "planId")
			if e != nil {
				writeError(w, e)
				return
			}
			page, e := number(q.Get("candidatePage"), 1)
			if e != nil {
				writeError(w, e)
				return
			}
			v, e := a.Runtime.Compensation.BasePay(r.Context(), credentials(r), r.PathValue("assignmentId"), c.PayQuery{Day: q.Get("businessAt"), Page: page, Plan: q.Get("planId"), Prepare: prepare})
			respond(w, 200, v, e)
		}
	}
	mux.HandleFunc("GET "+base, read(false))
	mux.HandleFunc("GET "+base+"/context", read(true))
	mux.HandleFunc("POST "+base, func(w http.ResponseWriter, r *http.Request) {
		var in c.ConfirmAward
		if e := decodeAward(r, &in); e != nil {
			writeError(w, e)
			return
		}
		v, e := a.Runtime.Compensation.ConfirmAward(r.Context(), credentials(r), r.PathValue("assignmentId"), r.Header.Get("Idempotency-Key"), r.Header.Get("X-Award-Confirmation"), in)
		respond(w, 200, v, e)
	})
	mux.HandleFunc("GET /api/v1/compensation/awards/{awardId}/revisions/{revision}", func(w http.ResponseWriter, r *http.Request) {
		v, e := a.Runtime.Compensation.AwardOriginal(r.Context(), credentials(r), r.PathValue("awardId"), r.PathValue("revision"))
		respond(w, 200, v, e)
	})
	mux.HandleFunc("GET /api/v1/compensation/award-receipts/{submissionId}", func(w http.ResponseWriter, r *http.Request) {
		v, e := a.Runtime.Compensation.AwardReceipt(r.Context(), credentials(r), r.PathValue("submissionId"))
		respond(w, 200, v, e)
	})
	mux.HandleFunc("POST /api/v1/compensation/award-receipts/{submissionId}/resolve", func(w http.ResponseWriter, r *http.Request) {
		var input c.RecoveryInput
		if e := decodeAward(r, &input); e != nil {
			writeError(w, e)
			return
		}
		v, e := a.Runtime.Compensation.ResolveAward(r.Context(), credentials(r), r.PathValue("submissionId"), input)
		respond(w, 200, v, e)
	})
}

func decodeAward(r *http.Request, out any) error {
	raw, e := decode(r, out)
	if e != nil {
		return e
	}
	var values map[string]json.RawMessage
	if e = json.Unmarshal(raw, &values); e != nil {
		return e
	}
	if command, ok := values["command"]; ok {
		if bytes.Equal(command, []byte("null")) {
			return foundation.Fail(400, "M09_INVALID_INPUT", "原申请格式无效")
		}
		values = nil
		if e = json.Unmarshal(command, &values); e != nil {
			return e
		}
	}
	for key, value := range values {
		if bytes.Equal(value, []byte("null")) {
			return &foundation.Fault{Status: 400, Code: "M09_INVALID_INPUT", Field: key, Message: "已提交的字段须填写有效值"}
		}
	}
	return nil
}

package bootstrap

import (
	s "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"encoding/json"
	"net/http"
	"slices"
)

func (a *API) quotaRoutes(mux *http.ServeMux) {
	const base = "/api/v1/staffing-quotas"
	list := func(w http.ResponseWriter, r *http.Request) {
		options := r.URL.Path == base+"/department-options"
		stats := r.URL.Path == base+"/statistics"
		fields := []string{"q", "page", "pageSize"}
		if options || stats {
			fields = append(fields, "businessAt")
		}
		if stats {
			fields = append(fields, "anchor")
		}
		if !options {
			fields = append(fields, "departmentId")
		}
		q, e := queryValues(r, fields...)
		if e != nil {
			writeError(w, staffingError(e))
			return
		}
		page, e := number(q.Get("page"), 1)
		if e != nil {
			writeError(w, staffingError(e))
			return
		}
		size, e := number(q.Get("pageSize"), 20)
		if e != nil {
			writeError(w, staffingError(e))
			return
		}
		out, e := a.Runtime.Staffing.Quotas(r.Context(), credentials(r), s.QuotaQuery{DepartmentOptions: options, Statistics: stats, Anchor: q.Get("anchor"), Day: q.Get("businessAt"), Search: q.Get("q"), DepartmentID: q.Get("departmentId"), Page: page, PageSize: size})
		respond(w, 200, out, staffingError(e))
	}
	mux.HandleFunc("GET "+base+"/statistics", list)
	mux.HandleFunc("GET "+base+"/department-options", list)
	position := func(w http.ResponseWriter, r *http.Request) {
		stats := r.Pattern == "GET "+base+"/statistics/positions/{positionId}"
		fields := []string{"sourceVersion", "selectedRecordId"}
		if stats {
			fields = []string{"businessAt", "sourceVersion"}
		}
		q, e := queryValues(r, fields...)
		if e != nil {
			writeError(w, staffingError(e))
			return
		}
		out, e := a.Runtime.Staffing.Quotas(r.Context(), credentials(r), s.QuotaQuery{PositionID: r.PathValue("positionId"), Statistics: stats, RecordID: q.Get("selectedRecordId"), Day: q.Get("businessAt"), Source: q.Get("sourceVersion")})
		respond(w, 200, out, staffingError(e))
	}
	mux.HandleFunc("GET "+base+"/positions/{positionId}", position)
	mux.HandleFunc("GET "+base+"/statistics/positions/{positionId}", position)
	mux.HandleFunc("GET "+base+"/history/{id}/changes", func(w http.ResponseWriter, r *http.Request) {
		q, e := staffingLogQuery(r)
		if e != nil {
			writeError(w, staffingError(e))
			return
		}
		out, e := a.Runtime.Staffing.QuotaChanges(r.Context(), credentials(r), r.PathValue("id"), q)
		respond(w, 200, out, staffingError(e))
	})
	mux.HandleFunc("GET "+base+"/departments/{departmentId}", func(w http.ResponseWriter, r *http.Request) {
		q, e := queryValues(r, "businessAt", "sourceVersion", "selectedRecordId")
		if e != nil {
			writeError(w, staffingError(e))
			return
		}
		id := r.PathValue("departmentId")
		if !foundation.ValidID(id) {
			writeError(w, foundation.Fail(404, "M03_NOT_FOUND", "未找到该部门"))
			return
		}
		out, e := a.Runtime.Staffing.Quotas(r.Context(), credentials(r), s.QuotaQuery{DepartmentDetail: true, Statistics: true, DepartmentID: id, Day: q.Get("businessAt"), Source: q.Get("sourceVersion"), RecordID: q.Get("selectedRecordId")})
		respond(w, 200, out, staffingError(e))
	})
	mux.HandleFunc("POST "+base+"/departments/{departmentId}/allocations", func(w http.ResponseWriter, r *http.Request) {
		var input s.AllocationInput
		raw, e := decode(r, &input)
		if e != nil {
			writeError(w, staffingError(e))
			return
		}
		cmd := s.QuotaCommand{Action: "allocate", DepartmentID: r.PathValue("departmentId"), Allocation: &input, SubmissionID: r.Header.Get("Idempotency-Key"), Digest: digest(r, raw)}
		out, e := a.Runtime.Staffing.ChangeStaffingQuota(r.Context(), credentials(r), cmd)
		if e != nil {
			writeError(w, staffingError(e))
			return
		}
		if out.Failure != nil {
			writeError(w, out.Failure)
			return
		}
		writeJSON(w, 200, out)
	})
	for _, route := range []struct {
		path, action     string
		fields, required []string
	}{
		{"", "create", []string{"positionId", "quota", "effectiveDate", "reason"}, []string{"positionId", "quota", "effectiveDate", "reason"}},
		{"/{quotaId}/adjust", "adjust", []string{"quota", "expectedSourceVersion", "effectiveDate", "reason"}, []string{"quota", "expectedSourceVersion", "effectiveDate", "reason"}},
		{"/{quotaId}/withdraw", "withdraw", []string{"expectedSourceVersion", "effectiveDate", "reason"}, []string{"expectedSourceVersion", "effectiveDate", "reason"}},
		{"/{quotaId}/records/{recordId}/correct", "correct", []string{"expectedSourceVersion", "expectedRecordRevision", "quota", "reason"}, []string{"expectedSourceVersion", "expectedRecordRevision"}},
		{"/{quotaId}/records/{recordId}/redate", "redate", []string{"expectedSourceVersion", "expectedRecordRevision", "newEffectiveDate"}, []string{"expectedSourceVersion", "expectedRecordRevision", "newEffectiveDate"}},
		{"/{quotaId}/records/{recordId}/delete", "delete", []string{"expectedSourceVersion", "expectedRecordRevision"}, []string{"expectedSourceVersion", "expectedRecordRevision"}},
	} {
		handler := func(preview bool) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				var input s.QuotaInput
				raw, e := decode(r, &input)
				if e != nil {
					writeError(w, staffingError(e))
					return
				}
				var fields map[string]json.RawMessage
				_ = json.Unmarshal(raw, &fields)
				_, input.ReasonProvided = fields["reason"]
				for f, v := range fields {
					if !slices.Contains(route.fields, f) || string(v) == "null" {
						writeError(w, foundation.Fail(400, "M03_INVALID_INPUT", "此操作不接受该字段或空值"))
						return
					}
				}
				for _, f := range route.required {
					if _, ok := fields[f]; !ok {
						writeError(w, foundation.Fail(400, "M03_INVALID_INPUT", "核编资料不完整"))
						return
					}
				}
				cmd := s.QuotaCommand{Action: route.action, QuotaID: r.PathValue("quotaId"), RecordID: r.PathValue("recordId"), SubmissionID: r.Header.Get("Idempotency-Key"), Digest: digest(r, raw), Input: input}
				if preview {
					out, e := a.Runtime.Staffing.PreviewQuota(r.Context(), credentials(r), cmd)
					respond(w, 200, out, staffingError(e))
					return
				}
				out, e := a.Runtime.Staffing.ChangeStaffingQuota(r.Context(), credentials(r), cmd)
				if e != nil {
					writeError(w, staffingError(e))
					return
				}
				if out.Failure != nil {
					writeError(w, out.Failure)
					return
				}
				writeJSON(w, 200, out)
			}
		}
		mux.HandleFunc("POST "+base+route.path, handler(false))
		mux.HandleFunc("POST "+base+route.path+"/preview", handler(true))
	}
	mux.HandleFunc("POST "+base+"/submissions/{id}/resolve", func(w http.ResponseWriter, r *http.Request) {
		out, e := a.Runtime.Staffing.ResolveQuota(r.Context(), credentials(r), r.PathValue("id"))
		respond(w, 200, out, staffingError(e))
	})
}

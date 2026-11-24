package bootstrap

import (
	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	f "chnhcm.local/chnhcm/internal/foundation"
	"net/http"
	"strconv"
)

func (a *API) employmentRegisterContext(w http.ResponseWriter, r *http.Request) {
	if _, e := queryValues(r); e != nil {
		writeError(w, e)
		return
	}
	ok, e := a.Runtime.Hires.CanListRegister(r.Context(), credentials(r))
	respond(w, 200, map[string]bool{"canList": ok}, e)
}
func (a *API) listEmploymentRegister(w http.ResponseWriter, r *http.Request) {
	v, err := queryValues(r, "ref", "kind", "status", "businessFrom", "businessTo", "recordedFrom", "recordedTo", "department", "fromDepartment", "toDepartment", "operator", "sort", "page", "pageSize")
	if err != nil {
		writeError(w, err)
		return
	}
	q := h.RegisterQuery{Ref: v.Get("ref"), Kind: v.Get("kind"), Status: v.Get("status"), BusinessFrom: v.Get("businessFrom"), BusinessTo: v.Get("businessTo"),
		RecordedFrom: v.Get("recordedFrom"), RecordedTo: v.Get("recordedTo"), Department: v.Get("department"), FromDepartment: v.Get("fromDepartment"), ToDepartment: v.Get("toDepartment"),
		Operator: v.Get("operator"), Sort: v.Get("sort"), Page: 1, PageSize: 20}
	for name, field := range map[string]*int{"page": &q.Page, "pageSize": &q.PageSize} {
		if v.Has(name) {
			*field, err = strconv.Atoi(v.Get(name))
			if err != nil {
				writeError(w, f.Fail(400, "REGISTER_QUERY", "请核对页码和每页条数"))
				return
			}
		}
	}
	result, err := a.Runtime.Hires.ListRegister(r.Context(), credentials(r), q)
	respond(w, 200, result, err)
}

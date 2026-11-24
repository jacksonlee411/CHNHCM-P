package bootstrap

import (
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"net/http"
)

func (a *API) changes(w http.ResponseWriter, r *http.Request) {
	q, err := queryValues(r, "page", "pageSize", "contextSourceVersion", "throughSourceVersion", "filterField", "filterOperator", "filterValue")
	if err != nil {
		writeError(w, err)
		return
	}
	for _, key := range []string{"page", "pageSize", "contextSourceVersion", "throughSourceVersion"} {
		if q.Has(key) && q.Get(key) == "" {
			writeError(w, logParameterError(key, foundation.Fail(400, "M02_INVALID_INPUT", "查询条件不能为空")))
			return
		}
	}
	page, err := number(q.Get("page"), 1)
	if err != nil {
		writeError(w, logParameterError("page", err))
		return
	}
	size, err := number(q.Get("pageSize"), 20)
	if err != nil {
		writeError(w, logParameterError("pageSize", err))
		return
	}
	in := org.ChangeQuery{ID: r.PathValue("id"), Page: page, PageSize: size, ContextSourceVersion: q.Get("contextSourceVersion"), ThroughSourceVersion: q.Get("throughSourceVersion")}
	if q.Has("filterField") || q.Has("filterOperator") || q.Has("filterValue") {
		in.Filter = &org.ColumnFilter{Field: q.Get("filterField"), Operator: q.Get("filterOperator"), Value: q.Get("filterValue")}
	}
	value, err := a.Runtime.Organization.Changes(r.Context(), credentials(r), in)
	respond(w, 200, value, err)
}

func (a *API) departmentChanges(w http.ResponseWriter, r *http.Request) {
	q, err := queryValues(r, "page", "pageSize", "contextSourceVersion", "throughSourceVersion", "filterField", "filterOperator", "filterValue")
	if err != nil {
		writeError(w, err)
		return
	}
	for _, key := range []string{"page", "pageSize", "contextSourceVersion", "throughSourceVersion"} {
		if q.Has(key) && q.Get(key) == "" {
			writeError(w, logParameterError(key, foundation.Fail(400, "M02_INVALID_INPUT", "查询条件不能为空")))
			return
		}
	}
	page, err := number(q.Get("page"), 1)
	if err != nil {
		writeError(w, logParameterError("page", err))
		return
	}
	size, err := number(q.Get("pageSize"), 20)
	if err != nil {
		writeError(w, logParameterError("pageSize", err))
		return
	}
	in := org.DepartmentChangeQuery{DepartmentID: r.PathValue("id"), Page: page, PageSize: size, ContextSourceVersion: q.Get("contextSourceVersion"), ThroughSourceVersion: q.Get("throughSourceVersion")}
	if q.Has("filterField") || q.Has("filterOperator") || q.Has("filterValue") {
		in.Filter = &org.ColumnFilter{Field: q.Get("filterField"), Operator: q.Get("filterOperator"), Value: q.Get("filterValue")}
	}
	value, err := a.Runtime.Department.Changes(r.Context(), credentials(r), in)
	var pg *pgconn.PgError
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) || errors.As(err, &pg) && (pg.Code == "57014" || pg.Code == "55P03") {
		err = foundation.Fail(503, "M02_EXECUTION_LIMIT", "变更日志暂时无法加载，请重试")
	}
	respond(w, 200, value, err)
}

// Describe the rejected transport parameter without changing its validation.
func logParameterError(field string, err error) error {
	var fault *foundation.Fault
	if errors.As(err, &fault) && fault.Status == 400 {
		copy := *fault
		copy.Field = field
		return &copy
	}
	return err
}

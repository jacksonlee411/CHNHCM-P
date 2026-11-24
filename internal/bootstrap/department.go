package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
	"strconv"

	"chnhcm.local/chnhcm/internal/contexts/organization/domain/dated"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5/pgconn"
)

func (a *API) departmentRoutes(mux *http.ServeMux) {
	a.departmentTreeRoutes(mux)
	mux.HandleFunc("GET /api/v1/departments/context", func(w http.ResponseWriter, r *http.Request) {
		if _, e := queryValues(r); e != nil {
			writeError(w, e)
			return
		}
		v, e := a.Runtime.Department.Context(r.Context(), credentials(r))
		respond(w, 200, v, e)
	})
	mux.HandleFunc("GET /api/v1/departments/operation-reasons", func(w http.ResponseWriter, r *http.Request) {
		q, e := queryValues(r, "operationType")
		if e != nil {
			writeError(w, e)
			return
		}
		if _, e = a.Runtime.Department.Context(r.Context(), credentials(r)); e != nil {
			writeError(w, e)
			return
		}
		choices := dated.OperationReasons(dated.Department, q.Get("operationType"))
		if len(choices) == 0 {
			writeError(w, foundation.Fail(501, "M02_CAPABILITY_UNSUPPORTED", "未开放该操作"))
			return
		}
		codes := []string{}
		for k := range choices {
			codes = append(codes, k)
		}
		slices.Sort(codes)
		out := []org.Reason{}
		for _, k := range codes {
			out = append(out, org.Reason{Code: k, Name: choices[k], Default: len(choices) == 1})
		}
		writeJSON(w, 200, out)
	})
	mux.HandleFunc("GET /api/v1/departments", a.departmentList)
	mux.HandleFunc("POST /api/v1/departments/{id}/preview", func(w http.ResponseWriter, r *http.Request) {
		if _, e := queryValues(r); e != nil {
			writeError(w, e)
			return
		}
		var in org.DepartmentPreviewInput
		raw, e := decode(r, &in)
		if e == nil {
			e = departmentPreviewShape(raw, in)
		}
		if e != nil {
			writeError(w, e)
			return
		}
		v, e := a.Runtime.Department.Preview(r.Context(), credentials(r), r.PathValue("id"), in)
		respond(w, 200, v, previewExecutionError(e))
	})
	for _, route := range []struct{ path, kind string }{{"business-units", "bu"}} {
		mux.HandleFunc("POST /api/v1/departments/references/"+route.path, func(w http.ResponseWriter, r *http.Request) {
			if _, e := queryValues(r); e != nil {
				writeError(w, e)
				return
			}
			var in org.DepartmentReferenceQuery
			raw, e := decode(r, &in)
			if e == nil {
				e = departmentReferenceShape(raw, in)
			}
			if e != nil {
				writeError(w, e)
				return
			}
			v, e := a.Runtime.Department.References(r.Context(), credentials(r), route.kind, in)
			respond(w, 200, v, e)
		})
	}

	mux.HandleFunc("POST /api/v1/departments/references/parents", func(w http.ResponseWriter, r *http.Request) {
		if _, e := queryValues(r); e != nil {
			writeError(w, e)
			return
		}
		var in org.DepartmentParentQuery
		raw, e := decode(r, &in)
		if e == nil {
			e = departmentReferenceShape(raw, org.DepartmentReferenceQuery{Context: in.Context, SelectedID: in.SelectedID})
		}
		if e != nil {
			writeError(w, e)
			return
		}
		v, e := a.Runtime.Department.Parents(r.Context(), credentials(r), in)
		respond(w, 200, v, e)
	})

	mux.HandleFunc("POST /api/v1/departments/navigation", func(w http.ResponseWriter, r *http.Request) {
		if _, e := queryValues(r); e != nil {
			writeError(w, e)
			return
		}
		var in org.DepartmentNavigationInput
		raw, e := decode(r, &in)
		if e != nil {
			writeError(w, e)
			return
		}
		var body map[string]json.RawMessage
		_ = json.Unmarshal(raw, &body)
		if e = departmentPredicateShape(body["query"]); e != nil {
			writeError(w, e)
			return
		}

		v, e := a.Runtime.Department.Navigate(r.Context(), credentials(r), in)
		respond(w, 200, v, e)
	})
	mux.HandleFunc("POST /api/v1/departments/selection/start", func(w http.ResponseWriter, r *http.Request) {
		var in org.DepartmentSelectionStartInput
		if !decodeSelection(w, r, &in) {
			return
		}
		v, e := a.Runtime.Department.StartSelection(r.Context(), credentials(r), in)
		if e != nil {
			writeSelectionError(w, e)
			return
		}
		writeJSON(w, 200, v)
	})
	mux.HandleFunc("POST /api/v1/departments/selection/resolve", func(w http.ResponseWriter, r *http.Request) {
		var in org.DepartmentSelectionInput
		if !decodeIdentityDraft(w, r, &in, true) {
			return
		}
		v, e := a.Runtime.Department.ResolveSelection(r.Context(), credentials(r), in)
		if e != nil {
			writeSelectionError(w, e)
			return
		}
		writeJSON(w, 200, v)
	})
	mux.HandleFunc("POST /api/v1/departments/selection/cancel", func(w http.ResponseWriter, r *http.Request) {
		var in org.DepartmentDraftInput
		if !decodeIdentityDraft(w, r, &in, false) {
			return
		}
		if err := a.Runtime.Department.CancelSelection(r.Context(), credentials(r), in); err != nil {
			writeSelectionError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /api/v1/departments/export", func(w http.ResponseWriter, r *http.Request) {
		var in org.DepartmentExportInput
		if !decodeIdentityDraft(w, r, &in, true) {
			return
		}
		v, e := a.Runtime.Department.ExportSelection(r.Context(), credentials(r), in)
		if e != nil {
			writeSelectionError(w, e)
			return
		}
		defer v.File.Close()
		if r.Context().Err() != nil {
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="chnhcm-departments-`+v.BusinessAt+`.csv"`)
		w.Header().Set("X-Selected-Count", strconv.Itoa(v.Count))
		w.WriteHeader(200)
		_, _ = io.Copy(w, v.File)
	})

	mux.HandleFunc("GET /api/v1/departments/{id}", a.departmentDetail)
	mux.HandleFunc("GET /api/v1/departments/{id}/changes", a.departmentChanges)
	for _, route := range []struct{ path, action string }{
		{"/departments", "CREATE"}, {"/departments/{id}/records", "ADD"},
		{"/departments/{id}/records/{recordId}/correct", "CORRECT"}, {"/departments/{id}/records/{recordId}/delete", "DELETE"},
		{"/departments/{id}/records/{recordId}/redate", "REDATE"}, {"/departments/{id}/deactivate", "DEACTIVATE"},
		{"/departments/{id}/reactivate", "REACTIVATE"}, {"/departments/{id}/lifecycle-pairs", "INSERT_LIFECYCLE_PAIR"},
		{"/departments/{id}/lifecycle-pairs/delete", "DELETE_LIFECYCLE_PAIR"},
	} {
		mux.HandleFunc("POST /api/v1"+route.path, func(w http.ResponseWriter, r *http.Request) { a.departmentExecute(w, r, route.action) })
	}
	mux.HandleFunc("POST /api/v1/departments/submissions/{id}/resolve", func(w http.ResponseWriter, r *http.Request) {
		if _, e := queryValues(r); e != nil {
			writeError(w, e)
			return
		}
		v, e := a.Runtime.Department.Resolve(r.Context(), credentials(r), r.PathValue("id"))
		var pg *pgconn.PgError
		if errors.Is(e, context.DeadlineExceeded) || errors.Is(e, context.Canceled) || (errors.As(e, &pg) && (pg.Code == "55P03" || pg.Code == "57014")) {
			writeJSON(w, 202, org.DepartmentResult{SubmissionID: r.PathValue("id"), Outcome: "pending"})
			return
		}
		respond(w, 200, v, e)
	})
}
func (a *API) departmentExecute(w http.ResponseWriter, r *http.Request, action string) {
	if _, e := queryValues(r); e != nil {
		writeError(w, e)
		return
	}
	cmd := org.DepartmentCommand{Action: action, DepartmentID: r.PathValue("id"), RecordID: r.PathValue("recordId"), SubmissionID: r.Header.Get("Idempotency-Key")}
	var input any
	var stop org.DepartmentDeactivateInput
	switch action {
	case "CREATE":
		input = &cmd.Create
	case "ADD", "REACTIVATE":
		input = &cmd.Add
	case "DEACTIVATE":
		input = &stop
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
	}
	raw, e := decode(r, input)
	if e != nil {
		writeError(w, e)
		return
	}
	// Optional classification is absent or a complete object, never JSON null.
	if action == "CORRECT" {
		var obj map[string]json.RawMessage
		_ = json.Unmarshal(raw, &obj)
		if v, ok := obj["classification"]; ok && string(v) == "null" {
			writeError(w, foundation.Fail(400, "M02_INVALID_INPUT", "业务分类必须完整"))
			return
		}
	}
	if action == "DEACTIVATE" {
		cmd.Add = org.DepartmentAddInput{ExpectedSourceVersion: stop.ExpectedSourceVersion, AnchorRecordID: stop.AnchorRecordID, EffectiveDate: stop.EffectiveDate, OperationType: stop.OperationType, OperationReason: stop.OperationReason}
	}
	cmd.Fields = fields(raw)
	cmd.Digest = digest(r, raw)
	v, e := a.Runtime.Department.Execute(r.Context(), credentials(r), cmd)
	status := 200
	if slices.Contains([]string{"CREATE", "ADD", "DEACTIVATE", "REACTIVATE"}, action) {
		status = 201
	}
	departmentCommandResponse(w, status, v, e)
}
func departmentCommandResponse(w http.ResponseWriter, status int, v org.DepartmentResult, e error) {
	if v.Outcome == "rejected" {
		code := "M02_INVALID_INPUT"
		if v.ErrorCode != nil {
			code = *v.ErrorCode
		}
		status = 409
		if code == "M02_INVALID_INPUT" || code == "M02_FORBIDDEN_INPUT" {
			status = 400
		}
		message := "原请求已拒绝，请核对输入和当前资料后重新办理"
		var f *foundation.Fault
		if errors.As(e, &f) {
			status = f.Status
			message = f.Message
		}
		// Locations were projected under current read rights. Do not append dependency payloads.
		writeJSON(w, status, map[string]any{"code": code, "message": message, "submissionId": v.SubmissionID, "outcome": v.Outcome, "failureDetails": v.FailureDetails})
		return
	}
	if v.Outcome == "no_change" {
		status = 200
	}
	respond(w, status, v, e)
}
func (a *API) departmentDetail(w http.ResponseWriter, r *http.Request) {
	q, e := queryValues(r, "selectedRecordId", "sourceVersion")
	if e != nil {
		writeError(w, e)
		return
	}
	for _, key := range []string{"selectedRecordId", "sourceVersion"} {
		if q.Has(key) && !foundation.ValidID(q.Get(key)) {
			writeError(w, foundation.Fail(400, "M02_INVALID_INPUT", "所选历史资料或记录无效"))
			return
		}
	}
	v, e := a.Runtime.Department.Detail(r.Context(), credentials(r), org.DepartmentDetailInput{DepartmentID: r.PathValue("id"), SelectedRecordID: q.Get("selectedRecordId"), SourceVersion: q.Get("sourceVersion")})
	respond(w, 200, v, e)
}
func (a *API) departmentList(w http.ResponseWriter, r *http.Request) {
	q, e := queryValues(r, "q", "status", "sort", "page", "pageSize", "filterField", "filterOperator", "filterValue", "cursor", "scopeDepartmentId", "includeDescendants")
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
	var filter *org.ColumnFilter
	if q.Has("filterField") || q.Has("filterOperator") || q.Has("filterValue") {
		filter = &org.ColumnFilter{Field: q.Get("filterField"), Operator: q.Get("filterOperator"), Value: q.Get("filterValue")}
	}
	var descendants *bool
	if q.Has("includeDescendants") {
		v := q.Get("includeDescendants")
		if v != "true" && v != "false" {
			writeError(w, foundation.Fail(400, "M02_INVALID_INPUT", "包含下级参数无效"))
			return
		}
		yes := v == "true"
		descendants = &yes
	}
	if q.Has("scopeDepartmentId") && !foundation.ValidID(q.Get("scopeDepartmentId")) {
		writeError(w, foundation.Fail(400, "M02_INVALID_INPUT", "部门范围无效，请重新选择"))
		return
	}
	v, e := a.Runtime.Department.List(r.Context(), credentials(r), org.DepartmentListInput{ScopeDepartmentID: q.Get("scopeDepartmentId"), IncludeDescendants: descendants, Cursor: q.Get("cursor"), Q: q.Get("q"), Status: q.Get("status"), Sort: q.Get("sort"), Page: page, PageSize: size, Filter: filter})
	respond(w, 200, v, e)
}

func commandShape(raw []byte, allowed, required []string) error {
	var obj map[string]json.RawMessage
	_ = json.Unmarshal(raw, &obj)
	for key, v := range obj {
		if key == "classification" && string(v) == "null" {
			return foundation.Fail(400, "M02_INVALID_INPUT", "业务分类必须完整")
		}
		if !slices.Contains(allowed, key) {
			return foundation.Fail(400, "M02_FORBIDDEN_INPUT", "此动作不接受该字段或空分类")
		}
	}
	for _, key := range required {
		if _, ok := obj[key]; !ok {
			return foundation.Fail(400, "M02_INVALID_INPUT", "操作内容不完整，请重新打开表单")
		}
	}
	if v, ok := obj["changes"]; ok {
		var patch map[string]any
		_ = json.Unmarshal(v, &patch)
		if len(patch) == 0 {
			return foundation.Fail(400, "M02_INVALID_INPUT", "没有修改内容")
		}
	}
	return nil
}
func departmentPreviewShape(raw []byte, in org.DepartmentPreviewInput) error {
	allowed := []string{"action", "expectedSourceVersion"}
	required := slices.Clone(allowed)
	switch in.Action {
	case "ADD", "REACTIVATE", "DEACTIVATE":
		allowed = append(allowed, "anchorRecordId", "effectiveDate", "operationType", "operationReason")
		required = slices.Clone(allowed)
		if in.Action != "DEACTIVATE" {
			allowed = append(allowed, "changes")
		}
		if in.Action == "ADD" {
			required = append(required, "changes")
		}
	case "CORRECT", "DELETE", "REDATE":
		allowed = append(allowed, "recordId", "expectedRecordRevision")
		required = slices.Clone(allowed)
		if in.Action == "CORRECT" {
			allowed = append(allowed, "classification", "changes", "newEffectiveDate")
		}
		if in.Action == "REDATE" {
			allowed = append(allowed, "newEffectiveDate")
			required = append(required, "newEffectiveDate")
		}
	default:
		return foundation.Fail(400, "M02_INVALID_INPUT", "预览动作无效")
	}
	return commandShape(raw, allowed, required)
}
func departmentReferenceShape(raw []byte, in org.DepartmentReferenceQuery) error {
	var body map[string]json.RawMessage
	_ = json.Unmarshal(raw, &body)
	if _, present := body["selectedId"]; present && !foundation.ValidID(in.SelectedID) {
		return foundation.Fail(400, "M02_INVALID_INPUT", "所选资料无效，请重新选择")
	}
	raw = body["context"]
	allowed := []string{"mode"}
	required := []string{"mode"}
	if in.Context.Mode == "filter" {
		// A list-filter lookup has no maintenance date, candidate or write intent.
	} else if in.Context.Mode == "create" {
		allowed = append(allowed, "effectiveDate", "managementBuId")
		required = append(required, "effectiveDate")
	} else if in.Context.Mode == "maintain" {
		allowed = append(allowed, "departmentId", "expectedSourceVersion", "action")
		required = slices.Clone(allowed)
		switch in.Context.Action {
		case "ADD":
			allowed = append(allowed, "anchorRecordId", "effectiveDate", "operationType", "operationReason", "changes")
			required = append(required, "anchorRecordId", "effectiveDate", "operationType")
			if in.Context.OperationType != "CHANGE" {
				return foundation.Fail(400, "M02_INVALID_INPUT", "位置新增类型无效")
			}
		case "REACTIVATE":
			allowed = append(allowed, "anchorRecordId", "effectiveDate", "operationReason", "changes")
			required = append(required, "anchorRecordId", "effectiveDate")
		case "CORRECT":
			allowed = append(allowed, "recordId", "expectedRecordRevision", "newEffectiveDate", "classification", "changes")
			required = append(required, "recordId", "expectedRecordRevision")
		default:
			return foundation.Fail(400, "M02_INVALID_INPUT", "引用查询动作无效")
		}
	} else {
		return foundation.Fail(400, "M02_INVALID_INPUT", "无法读取候选资料，请重新打开表单")
	}
	if in.Context.Classification != nil && in.Context.Classification.OperationType == "" {
		return foundation.Fail(400, "M02_INVALID_INPUT", "拟分类类型必填")
	}
	return commandShape(raw, allowed, required)
}
